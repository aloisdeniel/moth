package server

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	billingv1 "github.com/aloisdeniel/moth/gen/moth/billing/v1"
	"github.com/aloisdeniel/moth/gen/moth/billing/v1/billingv1connect"
	"github.com/aloisdeniel/moth/internal/store"
)

func (e *testEnv) copySvc() adminv1connect.CopyServiceClient {
	return adminv1connect.NewCopyServiceClient(e.client, e.url)
}

func (e *testEnv) billingClient(publishableKey string) billingv1connect.BillingServiceClient {
	hc := &http.Client{Transport: keyTransport{publishableKey}}
	return billingv1connect.NewBillingServiceClient(hc, e.url)
}

// configWithLang fetches GetProjectConfig setting the given Accept-Language and
// x-moth-language metadata (either may be "").
func (e *testEnv) configWithLang(t *testing.T, pk, acceptLang, mothLang string) *authv1.GetProjectConfigResponse {
	t.Helper()
	req := connect.NewRequest(&authv1.GetProjectConfigRequest{})
	if acceptLang != "" {
		req.Header().Set("Accept-Language", acceptLang)
	}
	if mothLang != "" {
		req.Header().Set("x-moth-language", mothLang)
	}
	got, err := e.configClient(pk).GetProjectConfig(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	return got.Msg
}

func TestGetProjectConfigCopyNegotiation(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Copy App")
	pk := p.PublishableKey

	cases := []struct {
		name       string
		acceptLang string
		mothLang   string
		wantLocale string
		wantTitle  string // sign_in.title in the negotiated locale
	}{
		{"accept-language fr", "fr", "", "fr", "Connexion"},
		{"x-moth-language de wins", "fr", "de", "de", "Anmelden"},
		{"fr-CA falls back to fr", "fr-CA", "", "fr", "Connexion"},
		{"unknown falls back to english", "xx", "", "en", "Sign in"},
		{"quality ordered picks de", "de;q=0.9, ja;q=0.2", "", "de", "Anmelden"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := e.configWithLang(t, pk, tc.acceptLang, tc.mothLang)
			if msg.Copy == nil {
				t.Fatal("copy must be present")
			}
			if msg.Copy.Locale != tc.wantLocale {
				t.Fatalf("locale = %q, want %q", msg.Copy.Locale, tc.wantLocale)
			}
			if got := msg.Copy.Messages["sign_in.title"]; got != tc.wantTitle {
				t.Fatalf("sign_in.title = %q, want %q", got, tc.wantTitle)
			}
			// {app} is interpolated with the project name server-side.
			if got := msg.Copy.Messages["sign_in.subtitle"]; !strings.Contains(got, "Copy App") {
				t.Fatalf("sign_in.subtitle not interpolated: %q", got)
			}
		})
	}
}

func TestGetProjectConfigCopyRevisionCaching(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Cache App")
	pk := p.PublishableKey

	// First call: copy carries the messages + a token.
	first := e.configWithLang(t, pk, "fr", "")
	if first.Copy == nil || len(first.Copy.Messages) == 0 {
		t.Fatal("first call must carry copy messages")
	}
	token := first.Copy.CopyRevision
	if token == "" {
		t.Fatal("copy_revision token must be set")
	}

	// Echoing the token omits the messages but keeps the locale + token.
	req := connect.NewRequest(&authv1.GetProjectConfigRequest{KnownCopyRevision: token})
	req.Header().Set("Accept-Language", "fr")
	got, err := e.configClient(pk).GetProjectConfig(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.Copy == nil || got.Msg.Copy.Locale != "fr" || got.Msg.Copy.CopyRevision != token {
		t.Fatalf("matching token must keep locale+token: %+v", got.Msg.Copy)
	}
	if len(got.Msg.Copy.Messages) != 0 {
		t.Fatal("matching known_copy_revision must omit the messages")
	}

	// A different locale re-sends even at the same store revision (the token
	// is keyed by locale too).
	de := e.configWithLang(t, pk, "de", "")
	if de.Copy.CopyRevision == token {
		t.Fatal("a different locale must yield a different copy token")
	}
	if len(de.Copy.Messages) == 0 {
		t.Fatal("a different locale must re-send the messages")
	}
}

func TestGetPaywallCopyNegotiation(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Paywall Copy App")
	pk := p.PublishableKey

	req := connect.NewRequest(&billingv1.GetPaywallRequest{})
	req.Header().Set("Accept-Language", "fr")
	got, err := e.billingClient(pk).GetPaywall(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	cp := got.Msg.Copy
	if cp == nil || cp.Locale != "fr" {
		t.Fatalf("paywall copy locale: %+v", cp)
	}
	if title := cp.Messages["paywall.title"]; !strings.Contains(title, "Débloquez") ||
		!strings.Contains(title, "Paywall Copy App") {
		t.Fatalf("paywall.title = %q", title)
	}
	// Echoing the token omits the body.
	req2 := connect.NewRequest(&billingv1.GetPaywallRequest{KnownCopyRevision: cp.CopyRevision})
	req2.Header().Set("Accept-Language", "fr")
	got2, err := e.billingClient(pk).GetPaywall(context.Background(), req2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got2.Msg.Copy.Messages) != 0 {
		t.Fatal("matching known_copy_revision must omit paywall messages")
	}
}

func TestHostedResetPageLocalized(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Hosted Copy App")

	req, _ := http.NewRequest(http.MethodGet, e.url+"/p/"+p.Slug+"/reset?token=x", nil)
	req.Header.Set("Accept-Language", "fr")
	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	body := string(raw)
	if got := resp.Header.Get("X-Moth-Locale"); got != "fr" {
		t.Fatalf("x-moth-locale = %q", got)
	}
	for _, want := range []string{
		`lang="fr"`,
		`dir="ltr"`,
		"Réinitialisez votre mot de passe", // hosted.reset_title
		"Nouveau mot de passe",             // hosted.reset_password_label
	} {
		if !strings.Contains(body, want) {
			t.Errorf("french reset page missing %q", want)
		}
	}

	// Default (no Accept-Language) renders bundled English.
	en := e.getPage(t, e.url+"/p/"+p.Slug+"/reset?token=x")
	if !strings.Contains(en, `lang="en"`) || !strings.Contains(en, "Reset your password") {
		t.Errorf("english reset page: %s", en)
	}
}

func TestLocalizedPasswordResetEmail(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Mail Copy App")
	auth := e.authClient(p.PublishableKey)
	ctx := context.Background()

	if _, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "jean@example.com", Password: "password-1",
	})); err != nil {
		t.Fatal(err)
	}
	// The reset request carries the SDK language metadata.
	req := connect.NewRequest(&authv1.RequestPasswordResetRequest{Email: "jean@example.com"})
	req.Header().Set("x-moth-language", "fr")
	if _, err := auth.RequestPasswordReset(ctx, req); err != nil {
		t.Fatal(err)
	}
	msg := e.mails.lastTo(t, "jean@example.com")
	if !strings.Contains(msg.Subject, "Réinitialisez votre mot de passe") ||
		!strings.Contains(msg.Subject, "Mail Copy App") {
		t.Fatalf("french reset subject = %q", msg.Subject)
	}
	// The whole body is localized too, not just the subject: the reset-body
	// paragraph (interpolated with the app name), the ignore notice and the
	// button label all resolve from the fr catalog. Apostrophe-free substrings
	// so the assertion holds against both the plain-text and the
	// (apostrophe-escaped) HTML rendering.
	for _, want := range []string{
		"Une réinitialisation du mot de passe a été demandée pour votre compte Mail Copy App.", // email.reset_body
		"vous pouvez ignorer cet e-mail", // email.ignore_notice
		"Réinitialiser le mot de passe",  // email.reset_button
	} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("french reset text body missing %q", want)
		}
		if !strings.Contains(msg.HTML, want) {
			t.Errorf("french reset html body missing %q", want)
		}
	}
}

// TestLocalizedEmailChangeEmail covers the plan/15 "email change" deliverable:
// the confirm-your-new-email mail renders in the requester's negotiated locale.
func TestLocalizedEmailChangeEmail(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Change Copy App")
	auth := e.authClient(p.PublishableKey)
	ctx := context.Background()

	su, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "old@example.com", Password: "password-1",
	}))
	if err != nil {
		t.Fatal(err)
	}
	req := bearer(&authv1.RequestEmailChangeRequest{NewEmail: "new@example.com"}, su.Msg.Tokens.AccessToken)
	req.Header().Set("x-moth-language", "fr")
	if _, err := auth.RequestEmailChange(ctx, req); err != nil {
		t.Fatal(err)
	}
	msg := e.mails.lastTo(t, "new@example.com")
	if !strings.Contains(msg.Subject, "Confirmez votre nouvel e-mail") ||
		!strings.Contains(msg.Subject, "Change Copy App") {
		t.Fatalf("french email-change subject = %q", msg.Subject)
	}
	if !strings.Contains(msg.Text, "Confirmez que vous souhaitez utiliser cette adresse") {
		t.Fatalf("french email-change body not localized: %q", msg.Text)
	}
}

func TestAdminCopyOverrideReflectsInConfig(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Editable App")
	pk := p.PublishableKey
	ctx := context.Background()
	copies := e.copySvc()

	// The editor surface lists every editable key with the bundled default.
	got, err := copies.GetProjectCopy(ctx, connect.NewRequest(&adminv1.GetProjectCopyRequest{
		ProjectId: p.Id, Locale: "fr",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Msg.IsDefault || got.Msg.Locale != "fr" {
		t.Fatalf("fresh copy: %+v", got.Msg)
	}
	var sawSignInTitle bool
	for _, k := range got.Msg.Keys {
		if k.Key == "sign_in.title" {
			sawSignInTitle = true
			if k.DefaultValue != "Connexion" || k.OverrideValue != "" {
				t.Fatalf("sign_in.title key: %+v", k)
			}
		}
	}
	if !sawSignInTitle {
		t.Fatal("editor must list sign_in.title")
	}

	// Save a French override for one key.
	upd, err := copies.UpdateProjectCopy(ctx, connect.NewRequest(&adminv1.UpdateProjectCopyRequest{
		ProjectId: p.Id, Locale: "fr",
		Values: map[string]string{"sign_in.title": "Bonjour"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if upd.Msg.RevisionId == "" {
		t.Fatal("update must mint a revision")
	}

	// A fr request reflects the override; en keeps the bundled default.
	fr := e.configWithLang(t, pk, "fr", "")
	if fr.Copy.Messages["sign_in.title"] != "Bonjour" {
		t.Fatalf("fr override not applied: %q", fr.Copy.Messages["sign_in.title"])
	}
	en := e.configWithLang(t, pk, "en", "")
	if en.Copy.Messages["sign_in.title"] != "Sign in" {
		t.Fatalf("en must keep bundled default: %q", en.Copy.Messages["sign_in.title"])
	}

	// fr now appears customized in the locale list.
	locs, err := copies.ListLocales(ctx, connect.NewRequest(&adminv1.ListLocalesRequest{ProjectId: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	var frCustom bool
	for _, l := range locs.Msg.Locales {
		if l.Tag == "fr" {
			frCustom = l.Customized
		}
	}
	if !frCustom {
		t.Fatal("fr must be flagged customized after an override")
	}

	// An audit entry records the copy update.
	entries, err := e.store.ListAudit(ctx, store.AuditFilter{ProjectID: p.Id, Action: "copy.update"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("copy update must be audited")
	}

	// Reset the key restores the bundled default.
	if _, err := copies.ResetCopy(ctx, connect.NewRequest(&adminv1.ResetCopyRequest{
		ProjectId: p.Id, Locale: "fr", Key: "sign_in.title",
	})); err != nil {
		t.Fatal(err)
	}
	fr = e.configWithLang(t, pk, "fr", "")
	if fr.Copy.Messages["sign_in.title"] != "Connexion" {
		t.Fatalf("after reset fr = %q", fr.Copy.Messages["sign_in.title"])
	}
}

// TestAdminCopyDefaultLocaleLayerInEditor pins that the editor/preview see the
// same three-layer resolution the SDK delivery does: a default-locale (en)
// override bleeds into every untranslated locale as its effective default, so
// the fr editor tab shows "Log in" (the en override) and it matches what
// GetProjectConfig ships to fr devices.
func TestAdminCopyDefaultLocaleLayerInEditor(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Layer App")
	ctx := context.Background()
	copies := e.copySvc()

	if _, err := copies.UpdateProjectCopy(ctx, connect.NewRequest(&adminv1.UpdateProjectCopyRequest{
		ProjectId: p.Id, Locale: "en", Values: map[string]string{"sign_in.submit": "Log in"},
	})); err != nil {
		t.Fatal(err)
	}

	// fr editor tab (no fr override): the effective default is the en override.
	fr, err := copies.GetProjectCopy(ctx, connect.NewRequest(&adminv1.GetProjectCopyRequest{
		ProjectId: p.Id, Locale: "fr", Screen: adminv1.CopyScreen_COPY_SCREEN_SIGN_IN,
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range fr.Msg.Keys {
		if k.Key == "sign_in.submit" && k.DefaultValue != "Log in" {
			t.Fatalf("fr editor default = %q, want the en override 'Log in'", k.DefaultValue)
		}
	}
	// Delivery agrees with the preview base.
	got := e.configWithLang(t, p.PublishableKey, "fr", "")
	if got.Copy.Messages["sign_in.submit"] != "Log in" {
		t.Fatalf("fr delivery = %q, want 'Log in'", got.Copy.Messages["sign_in.submit"])
	}
	// The default (en) tab still hints the BUNDLED default, and its own override
	// is surfaced separately as the override value.
	en, err := copies.GetProjectCopy(ctx, connect.NewRequest(&adminv1.GetProjectCopyRequest{
		ProjectId: p.Id, Locale: "en", Screen: adminv1.CopyScreen_COPY_SCREEN_SIGN_IN,
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range en.Msg.Keys {
		if k.Key == "sign_in.submit" && (k.DefaultValue != "Sign in" || k.OverrideValue != "Log in") {
			t.Fatalf("en editor key = %+v, want default 'Sign in' / override 'Log in'", k)
		}
	}
}

// TestCopyUpdatePreservesOtherScreens is the regression guard for the
// whole-locale replace bug: saving one screen's copy for a locale must not wipe
// another screen's overrides for the same locale.
func TestCopyUpdatePreservesOtherScreens(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Merge App")
	ctx := context.Background()
	copies := e.copySvc()

	if _, err := copies.UpdateProjectCopy(ctx, connect.NewRequest(&adminv1.UpdateProjectCopyRequest{
		ProjectId: p.Id, Locale: "fr", Values: map[string]string{"sign_in.title": "Bonjour"},
	})); err != nil {
		t.Fatal(err)
	}
	// A separate save for a DIFFERENT screen, same locale (the per-screen editor
	// flow). This used to replace the whole fr document and delete sign_in.title.
	if _, err := copies.UpdateProjectCopy(ctx, connect.NewRequest(&adminv1.UpdateProjectCopyRequest{
		ProjectId: p.Id, Locale: "fr", Values: map[string]string{"sign_up.title": "Salut"},
	})); err != nil {
		t.Fatal(err)
	}
	got := e.configWithLang(t, p.PublishableKey, "fr", "")
	if got.Copy.Messages["sign_in.title"] != "Bonjour" {
		t.Fatalf("sign_in.title wiped by a later sign_up save: %q", got.Copy.Messages["sign_in.title"])
	}
	if got.Copy.Messages["sign_up.title"] != "Salut" {
		t.Fatalf("sign_up.title not saved: %q", got.Copy.Messages["sign_up.title"])
	}
}

// TestHostedPageEscapesOperatorCopy proves operator-controlled copy is
// HTML-escaped on the hosted pages (a stored-XSS guard). hosted.* keys are not
// editable in the admin UI but UpdateProjectCopy accepts any valid catalog key,
// so this markup is genuinely reachable.
func TestHostedPageEscapesOperatorCopy(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "XSS App")
	ctx := context.Background()

	const payload = `<script>alert(1)</script>`
	if _, err := e.copySvc().UpdateProjectCopy(ctx, connect.NewRequest(&adminv1.UpdateProjectCopyRequest{
		ProjectId: p.Id, Locale: "en", Values: map[string]string{"hosted.reset_title": payload},
	})); err != nil {
		t.Fatal(err)
	}
	body := e.getPage(t, e.url+"/p/"+p.Slug+"/reset?token=x")
	if strings.Contains(body, payload) {
		t.Fatalf("operator markup rendered live (stored XSS):\n%s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Fatalf("operator markup not HTML-escaped:\n%s", body)
	}
}

// TestOAuthCallbackPageLocalized covers the OAuth callback hosted pages in the
// negotiated locale — both the success body and the error copy (which used to
// be hardcoded English behind a localized lang attribute).
func TestOAuthCallbackPageLocalized(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "OAuth Loc App")

	// A callback carrying an explicit provider error renders the localized
	// "not completed" page.
	req, _ := http.NewRequest(http.MethodGet,
		e.url+"/oauth/google/callback?state="+p.Slug+".x&error=access_denied", nil)
	req.Header.Set("Accept-Language", "fr")
	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	body := string(raw)
	if got := resp.Header.Get("X-Moth-Locale"); got != "fr" {
		t.Fatalf("x-moth-locale = %q", got)
	}
	for _, want := range []string{
		`lang="fr"`,
		"Connexion non terminée", // hosted.oauth_incomplete_title
		"terminé la connexion",   // hosted.oauth_incomplete (apostrophe-free chunk)
	} {
		if !strings.Contains(body, want) {
			t.Errorf("french oauth error page missing %q", want)
		}
	}
}

func TestAdminCopyValidationRejections(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Validate App")
	ctx := context.Background()
	copies := e.copySvc()

	cases := []struct {
		name   string
		values map[string]string
	}{
		{"unknown key", map[string]string{"sign_in.nope": "x"}},
		{"missing placeholder", map[string]string{"sign_in.subtitle": "No placeholder here"}},
		{"over length", map[string]string{"sign_in.title": strings.Repeat("x", 401)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := copies.UpdateProjectCopy(ctx, connect.NewRequest(&adminv1.UpdateProjectCopyRequest{
				ProjectId: p.Id, Locale: "fr", Values: tc.values,
			}))
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("want InvalidArgument, got %v", err)
			}
		})
	}

	// CopyService requires an admin session.
	anon := adminv1connect.NewCopyServiceClient(http.DefaultClient, e.url)
	_, err := anon.GetProjectCopy(ctx, connect.NewRequest(&adminv1.GetProjectCopyRequest{ProjectId: p.Id}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("anonymous GetProjectCopy: %v", err)
	}
}
