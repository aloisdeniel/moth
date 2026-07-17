package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/billing"
	"github.com/aloisdeniel/moth/internal/oidc"
	"github.com/aloisdeniel/moth/internal/store"
)

// Doctor runs the `moth doctor` health checks against one instance and,
// when Slug is set, one project. It never mutates anything (the optional
// SMTP test send excepted, which the operator requests explicitly).
type Doctor struct {
	// BaseURL is the instance URL the checks reach it at (the context URL).
	BaseURL      string
	HTTPC        oidc.Doer
	Session      adminv1connect.SessionServiceClient
	Settings     adminv1connect.InstanceSettingsServiceClient
	Projects     adminv1connect.ProjectServiceClient
	BillingCreds adminv1connect.BillingCredentialsServiceClient
	Products     adminv1connect.ProductServiceClient

	// Slug selects a project for the provider checks; empty runs the
	// instance-level checks only.
	Slug string
	// SMTPTestTo, when set, sends a real test email to that address.
	SMTPTestTo string
	// AppleKeyPath optionally points at the project's Sign in with Apple
	// .p8 so the Apple dry-run can happen: moth stores the key encrypted
	// and never returns it, so a remote doctor cannot mint a client secret
	// without it.
	AppleKeyPath string
	// GoogleAuthURL and AppleTokenBase are test overrides.
	GoogleAuthURL  string
	AppleTokenBase string

	// AppleIAPKeyPath optionally points at the project's App Store Server API
	// In-App-Purchase .p8 so the billing check can probe the App Store Server
	// API (moth stores the key encrypted and never returns it).
	AppleIAPKeyPath string
	// GoogleServiceAccountPath optionally points at the Play Developer API
	// service-account JSON so the billing check can probe the Play API.
	GoogleServiceAccountPath string
	// StripeSecretKey optionally supplies the project's Stripe secret key so
	// the billing check can probe the Stripe API (moth stores the key
	// encrypted and never returns it, like the other store secrets).
	StripeSecretKey string
	// AppleServerAPIBase, GoogleAPIBase, GoogleTokenURL and StripeAPIBase are
	// test overrides for the billing store probes.
	AppleServerAPIBase string
	GoogleAPIBase      string
	GoogleTokenURL     string
	StripeAPIBase      string
}

func (d *Doctor) defaults() {
	if d.HTTPC == nil {
		d.HTTPC = &http.Client{}
	}
	if d.GoogleAuthURL == "" {
		d.GoogleAuthURL = googleAuthEndpoint
	}
	if d.AppleTokenBase == "" {
		d.AppleTokenBase = oidc.AppleBaseURL
	}
	d.BaseURL = strings.TrimSuffix(d.BaseURL, "/")
}

// Run produces the health report. It returns an error only when the run
// itself could not proceed (no connection at all); individual problems are
// FAIL checks.
func (d *Doctor) Run(ctx context.Context) (*Report, error) {
	d.defaults()
	rep := &Report{}

	// Context/auth: one RPC validates the credential and reports what the
	// CLI is talking to.
	me, err := d.Session.GetCurrentAdmin(ctx, connect.NewRequest(&adminv1.GetCurrentAdminRequest{}))
	if err != nil {
		rep.Fail("instance: admin API reachable and authenticated", err.Error(),
			"check the server URL and that the personal access token is valid (moth admin token, or the admin SPA)")
		return rep, nil
	}
	rep.Pass("instance: admin API reachable and authenticated",
		fmt.Sprintf("%s (server %s)", me.Msg.Admin.Email, me.Msg.ServerVersion))

	d.checkBaseURL(ctx, rep)
	d.checkHTTP(ctx, rep, "instance: health endpoint", "/healthz", nil)
	d.checkPub(ctx, rep)
	d.checkSMTP(ctx, rep)

	if d.Slug == "" {
		return rep, nil
	}
	project, err := findProjectBySlug(ctx, d.Projects, d.Slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			rep.Fail("project: exists", "no project with slug "+d.Slug,
				"list projects with `moth project list`")
			return rep, nil
		}
		return nil, err
	}
	rep.Pass("project: exists", project.Name+" ("+d.Slug+")")
	d.checkJWKS(ctx, rep)
	d.checkGoogle(ctx, rep, project.Settings.GetGoogle())
	d.checkApple(ctx, rep, project.Settings.GetApple())
	d.checkBilling(ctx, rep, project.Id)
	return rep, nil
}

// checkBaseURL compares the URL the CLI reached the server at with the
// base URL the server believes it has — a mismatch breaks every absolute
// URL moth hands out (JWKS, email links, OAuth redirects) — and flags
// plain-http instances.
func (d *Doctor) checkBaseURL(ctx context.Context, rep *Report) {
	const name = "instance: base URL sanity"
	settings, err := d.Settings.GetInstanceSettings(ctx, connect.NewRequest(&adminv1.GetInstanceSettingsRequest{}))
	if err != nil {
		rep.Fail(name, "GetInstanceSettings: "+err.Error(), "")
		return
	}
	configured := strings.TrimSuffix(settings.Msg.BaseUrl, "/")
	u, err := url.Parse(configured)
	if err != nil || u.Scheme == "" || u.Host == "" {
		rep.Fail(name, fmt.Sprintf("configured base URL %q does not parse as an absolute URL", configured),
			"fix base_url in moth.toml (or MOTH_BASE_URL / --base-url)")
		return
	}
	if configured != d.BaseURL {
		rep.Warn(name, fmt.Sprintf("reached at %s but the server believes it is %s", d.BaseURL, configured),
			"JWKS/email/OAuth URLs are built from the server value; align base_url with the public address")
		return
	}
	if u.Scheme != "https" && !isLoopbackHost(u.Host) {
		rep.Warn(name, configured+" is plain http on a non-local host",
			"put moth behind TLS; OAuth providers and mobile ATS require https")
		return
	}
	rep.Pass(name, configured)
}

func isLoopbackHost(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// checkHTTP GETs a plain-HTTP surface of the instance and expects 200;
// inspect, when set, examines the body.
func (d *Doctor) checkHTTP(ctx context.Context, rep *Report, name, path string, inspect func([]byte) error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.BaseURL+path, nil)
	if err != nil {
		rep.Fail(name, err.Error(), "")
		return
	}
	resp, err := d.HTTPC.Do(req)
	if err != nil {
		rep.Fail(name, fmt.Sprintf("GET %s: %v", path, err), "is the server reachable from here?")
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		rep.Fail(name, fmt.Sprintf("GET %s: read: %v", path, err), "")
		return
	}
	if resp.StatusCode != http.StatusOK {
		rep.Fail(name, fmt.Sprintf("GET %s: status %d", path, resp.StatusCode), "")
		return
	}
	if inspect != nil {
		if err := inspect(body); err != nil {
			rep.Fail(name, fmt.Sprintf("GET %s: %v", path, err), "")
			return
		}
	}
	rep.Pass(name, "GET "+path+" → 200")
}

func (d *Doctor) checkPub(ctx context.Context, rep *Report) {
	d.checkHTTP(ctx, rep, "instance: pub endpoint serves the SDK", "/pub/api/packages/moth_auth", func(body []byte) error {
		var listing struct {
			Versions []json.RawMessage `json:"versions"`
		}
		if err := json.Unmarshal(body, &listing); err != nil {
			return fmt.Errorf("not a pub listing: %w", err)
		}
		if len(listing.Versions) == 0 {
			return errors.New("pub listing has no versions")
		}
		return nil
	})
}

func (d *Doctor) checkSMTP(ctx context.Context, rep *Report) {
	const name = "instance: outgoing email (SMTP)"
	settings, err := d.Settings.GetInstanceSettings(ctx, connect.NewRequest(&adminv1.GetInstanceSettingsRequest{}))
	if err != nil {
		rep.Fail(name, "GetInstanceSettings: "+err.Error(), "")
		return
	}
	switch settings.Msg.SmtpSource {
	case adminv1.SmtpSource_SMTP_SOURCE_NONE, adminv1.SmtpSource_SMTP_SOURCE_UNSPECIFIED:
		rep.Warn(name, "no SMTP configured — emails only appear in the server console",
			"configure SMTP in the admin SPA (Settings) or moth.toml [smtp]")
		return
	case adminv1.SmtpSource_SMTP_SOURCE_CONFIG, adminv1.SmtpSource_SMTP_SOURCE_DATABASE:
	}
	if d.SMTPTestTo == "" {
		rep.Pass(name, fmt.Sprintf("configured via %s (%s); pass --smtp-to <you> for a real test send",
			smtpSourceLabel(settings.Msg.SmtpSource), settings.Msg.Smtp.GetHost()))
		return
	}
	_, err = d.Settings.SendTestEmail(ctx, connect.NewRequest(&adminv1.SendTestEmailRequest{To: d.SMTPTestTo}))
	if err != nil {
		rep.Fail(name, "test send to "+d.SMTPTestTo+" failed: "+err.Error(),
			"check host/port/credentials in the SMTP settings")
		return
	}
	rep.Pass(name, "test email sent to "+d.SMTPTestTo)
}

func smtpSourceLabel(s adminv1.SmtpSource) string {
	switch s {
	case adminv1.SmtpSource_SMTP_SOURCE_CONFIG:
		return "config file"
	case adminv1.SmtpSource_SMTP_SOURCE_DATABASE:
		return "database"
	case adminv1.SmtpSource_SMTP_SOURCE_NONE, adminv1.SmtpSource_SMTP_SOURCE_UNSPECIFIED:
	}
	return "unknown"
}

func (d *Doctor) checkJWKS(ctx context.Context, rep *Report) {
	path := "/p/" + d.Slug + "/.well-known/jwks.json"
	d.checkHTTP(ctx, rep, "project: JWKS reachable", path, func(body []byte) error {
		var jwks struct {
			Keys []json.RawMessage `json:"keys"`
		}
		if err := json.Unmarshal(body, &jwks); err != nil {
			return fmt.Errorf("not a JWKS document: %w", err)
		}
		if len(jwks.Keys) == 0 {
			return errors.New("JWKS has no keys")
		}
		return nil
	})
}

func (d *Doctor) checkGoogle(ctx context.Context, rep *Report, g *adminv1.GoogleProviderConfig) {
	const name = "project: Google sign-in"
	if g == nil || !g.Enabled {
		rep.Skip(name, "provider disabled")
		return
	}
	ids := []struct{ platform, id, redirect string }{
		{"web", g.WebClientId, d.BaseURL + "/oauth/google/callback"},
		{"iOS", g.IosClientId, reversedClientScheme(g.IosClientId) + ":/oauth2redirect"},
		{"Android", g.AndroidClientId, reversedClientScheme(g.AndroidClientId) + ":/oauth2redirect"},
	}
	configuredAny := false
	for _, c := range ids {
		checkName := fmt.Sprintf("project: Google %s client ID resolves", c.platform)
		if c.id == "" {
			continue
		}
		configuredAny = true
		if _, err := ValidateGoogleClientID(c.id); err != nil {
			rep.Fail(checkName, err.Error(), "fix the client ID in the project settings")
			continue
		}
		result, err := probeGoogleClient(ctx, d.HTTPC, d.GoogleAuthURL, c.id, c.redirect)
		switch result {
		case probeOK:
			rep.Pass(checkName, c.id)
		case probeClientNotFound:
			rep.Fail(checkName, "Google does not know "+c.id,
				"the client was probably deleted; re-run `moth setup google`")
		case probeRedirectMismatch:
			if c.platform == "web" {
				rep.Fail("project: Google web redirect URI registered",
					"client exists but "+c.redirect+" is not registered",
					"add it under the client's \"Authorized redirect URIs\" in the Google console")
			} else {
				rep.Pass(checkName, c.id)
			}
		default:
			rep.Warn(checkName, fmt.Sprintf("probe inconclusive: %v", err), "")
		}
	}
	if !configuredAny {
		rep.Fail(name, "enabled but no client ID configured", "run `moth setup google --project "+d.Slug+"`")
		return
	}
	if g.WebClientId != "" && !g.HasWebClientSecret {
		rep.Warn("project: Google web client secret", "no secret stored — the web-redirect fallback will fail",
			"re-run `moth setup google` with --web-client-secret")
	}
}

func (d *Doctor) checkApple(ctx context.Context, rep *Report, a *adminv1.AppleProviderConfig) {
	const name = "project: Apple sign-in"
	if a == nil || !a.Enabled {
		rep.Skip(name, "provider disabled")
		return
	}
	var missing []string
	if a.TeamId == "" {
		missing = append(missing, "team ID")
	}
	if a.KeyId == "" {
		missing = append(missing, "key ID")
	}
	if !a.HasPrivateKey {
		missing = append(missing, "private key")
	}
	if a.ServicesId == "" && len(a.BundleIds) == 0 {
		missing = append(missing, "services ID or bundle ID")
	}
	if len(missing) > 0 {
		rep.Fail(name, "enabled but incomplete: missing "+strings.Join(missing, ", "),
			"run `moth setup apple --project "+d.Slug+"`")
		return
	}
	rep.Pass(name, fmt.Sprintf("configured (team %s, key %s)", a.TeamId, a.KeyId))

	const dryRunName = "project: Apple key accepted (token endpoint dry-run)"
	if d.AppleKeyPath == "" {
		rep.Warn(dryRunName, "moth stores the key encrypted and never returns it, so a remote doctor cannot mint a client secret",
			"re-run with --apple-key <path to the Sign in with Apple .p8> to verify key validity")
		return
	}
	raw, err := os.ReadFile(d.AppleKeyPath)
	if err != nil {
		rep.Fail(dryRunName, err.Error(), "")
		return
	}
	key, err := oidc.ParseP8(raw)
	if err != nil {
		rep.Fail(dryRunName, fmt.Sprintf("%s: %v", d.AppleKeyPath, err), "")
		return
	}
	clientID := a.ServicesId
	if clientID == "" {
		clientID = a.BundleIds[0]
	}
	rep.add(appleTokenDryRun(ctx, dryRunName, d.HTTPC, d.AppleTokenBase, clientID, a.TeamId, a.KeyId, key))
}

// checkBilling reports on the project's store billing configuration: whether the
// store API credentials and notification plumbing are present, whether moth's
// webhook endpoints are reachable, a catalog mapping (drift) summary, and — when
// the operator passes the store keys (moth stores them encrypted and never
// returns them) — a live reachability probe against each store API.
func (d *Doctor) checkBilling(ctx context.Context, rep *Report, projectID string) {
	const name = "project: store billing credentials"
	if d.BillingCreds == nil {
		return
	}
	resp, err := d.BillingCreds.GetBillingCredentials(ctx, connect.NewRequest(&adminv1.GetBillingCredentialsRequest{ProjectId: projectID}))
	if err != nil {
		rep.Fail(name, "GetBillingCredentials: "+err.Error(), "")
		return
	}
	a, g, sc := resp.Msg.Apple, resp.Msg.Google, resp.Msg.Stripe
	appleConfigured := a.GetBundleId() != "" || a.GetHasIapKey()
	googleConfigured := g.GetPackageName() != "" || g.GetHasServiceAccount()
	stripeConfigured := sc.GetHasSecretKey() || d.StripeSecretKey != ""
	if !appleConfigured && !googleConfigured && !stripeConfigured {
		rep.Skip(name, "no store billing credentials configured (subscriptions not set up)")
		return
	}
	if appleConfigured {
		d.checkAppleBilling(ctx, rep, a)
	}
	if googleConfigured {
		d.checkGoogleBilling(ctx, rep, projectID, g)
	}
	if stripeConfigured {
		d.checkStripeBilling(ctx, rep, sc)
	}
	d.checkCatalogMapping(ctx, rep, projectID, appleConfigured, googleConfigured)
}

func (d *Doctor) checkAppleBilling(ctx context.Context, rep *Report, a *adminv1.AppleBillingConfig) {
	const name = "project: Apple billing credentials"
	var missing []string
	if !a.GetHasIapKey() {
		missing = append(missing, "In-App-Purchase .p8")
	}
	if a.GetIapKeyId() == "" {
		missing = append(missing, "key id")
	}
	if a.GetIapIssuerId() == "" {
		missing = append(missing, "issuer id")
	}
	if a.GetBundleId() == "" {
		missing = append(missing, "bundle id")
	}
	if len(missing) > 0 {
		rep.Fail(name, "incomplete: missing "+strings.Join(missing, ", "),
			"run `moth setup billing --project "+d.Slug+"`")
	} else {
		rep.Pass(name, "configured (bundle "+a.GetBundleId()+", key "+a.GetIapKeyId()+")")
	}
	if !a.GetHasNotificationSecret() {
		rep.Warn("project: Apple notification secret", "no App Store Server Notifications secret stored",
			"re-run `moth setup billing` with --apple-notification-secret to authenticate the webhook")
	} else {
		rep.Pass("project: Apple notification secret", "stored")
	}
	d.checkWebhookReachable(ctx, rep, "project: Apple notification endpoint", "/billing/apple/notifications/"+d.Slug)

	// Live probe only when the operator supplies the key material.
	const probeName = "project: Apple App Store Server API reachable"
	if d.AppleIAPKeyPath == "" {
		rep.Warn(probeName, "moth stores the In-App-Purchase key encrypted and never returns it, so a remote doctor cannot probe the store API",
			"re-run with --apple-iap-p8 <path to the App Store Server API .p8> to verify the key")
		return
	}
	if len(missing) > 0 {
		return
	}
	raw, err := os.ReadFile(d.AppleIAPKeyPath)
	if err != nil {
		rep.Fail(probeName, err.Error(), "")
		return
	}
	key, err := billing.ParseP8(raw)
	if err != nil {
		rep.Fail(probeName, fmt.Sprintf("%s: %v", d.AppleIAPKeyPath, err), "")
		return
	}
	rep.add(appleServerAPIProbe(ctx, probeName, d.HTTPC, d.AppleServerAPIBase,
		a.GetIapIssuerId(), a.GetIapKeyId(), a.GetBundleId(), key))
}

func (d *Doctor) checkGoogleBilling(ctx context.Context, rep *Report, projectID string, g *adminv1.GoogleBillingConfig) {
	const name = "project: Google billing credentials"
	var missing []string
	if !g.GetHasServiceAccount() {
		missing = append(missing, "service-account JSON")
	}
	if g.GetPackageName() == "" {
		missing = append(missing, "package name")
	}
	if len(missing) > 0 {
		rep.Fail(name, "incomplete: missing "+strings.Join(missing, ", "),
			"run `moth setup billing --project "+d.Slug+"`")
	} else {
		rep.Pass(name, "configured (package "+g.GetPackageName()+")")
	}
	if g.GetPubsubTopic() == "" {
		rep.Warn("project: Google RTDN topic", "no Pub/Sub topic configured — renewal notifications will not arrive",
			"re-run `moth setup billing` with --google-pubsub-topic")
	} else if !g.GetHasRtdnSecret() {
		rep.Warn("project: Google RTDN topic", "topic "+g.GetPubsubTopic()+" set but no RTDN push secret stored",
			"re-run `moth setup billing` with --google-rtdn-secret to authenticate the webhook")
	} else {
		rep.Pass("project: Google RTDN topic", g.GetPubsubTopic())
	}
	d.checkWebhookReachable(ctx, rep, "project: Google RTDN endpoint", "/billing/google/rtdn/"+d.Slug)

	const probeName = "project: Google Play Developer API reachable"
	if d.GoogleServiceAccountPath == "" {
		rep.Warn(probeName, "moth stores the service account encrypted and never returns it, so a remote doctor cannot probe the store API",
			"re-run with --google-service-account <path to the SA JSON> to verify access")
		return
	}
	if len(missing) > 0 {
		return
	}
	raw, err := os.ReadFile(d.GoogleServiceAccountPath)
	if err != nil {
		rep.Fail(probeName, err.Error(), "")
		return
	}
	sa, err := billing.ParseServiceAccount(raw)
	if err != nil {
		rep.Fail(probeName, fmt.Sprintf("%s: %v", d.GoogleServiceAccountPath, err), "")
		return
	}
	probe := googlePlayAPIProbe(ctx, probeName, d.HTTPC, d.GoogleAPIBase, d.GoogleTokenURL, g.GetPackageName(), sa)
	rep.add(probe)
	// Only when the SA can actually reach the Play API is it meaningful to read
	// back each mapped SKU to confirm it still exists store-side.
	if probe.Status == StatusPass {
		d.checkGoogleCatalogLive(ctx, rep, projectID, g, sa)
	}
}

// checkGoogleCatalogLive reads each mapped Google SKU back from the Play
// Developer API to flag a product that was deleted store-side — the acceptance
// criterion "moth doctor flags a deleted store product". A 404 for a SKU moth
// still maps means the subscription was removed in Play Console (or never
// created). This is the Google half; Apple's subscription catalog is not
// readable with the App Store Server API key moth holds (only the ASC catalog
// key, which lives in `moth setup billing` in-process), so an Apple deleted
// product surfaces on the next `moth setup billing` push, not here.
func (d *Doctor) checkGoogleCatalogLive(ctx context.Context, rep *Report, projectID string, g *adminv1.GoogleBillingConfig, sa *billing.GoogleServiceAccount) {
	const name = "project: Google catalog products exist"
	if d.Products == nil {
		return
	}
	resp, err := d.Products.ListProducts(ctx, connect.NewRequest(&adminv1.ListProductsRequest{ProjectId: projectID}))
	if err != nil {
		rep.Fail(name, "ListProducts: "+err.Error(), "")
		return
	}
	tokens := billing.NewGoogleTokenSource(sa, d.GoogleTokenURL, d.HTTPC, nil)
	client := &billing.GoogleClient{BaseURL: d.GoogleAPIBase, PackageName: g.GetPackageName(), Tokens: tokens, HTTPC: d.HTTPC}
	var missing []string
	checked := 0
	for _, p := range resp.Msg.Products {
		sku := p.GetGoogleProductId()
		if sku == "" {
			continue
		}
		checked++
		if _, _, err := client.GetSubscriptionV2(ctx, sku); errors.Is(err, billing.ErrNotFound) {
			missing = append(missing, p.GetIdentifier()+" ("+sku+")")
		}
	}
	switch {
	case len(missing) > 0:
		rep.Fail(name, fmt.Sprintf("%d mapped product(s) no longer exist in Google Play: %s", len(missing), strings.Join(missing, ", ")),
			"recreate them in Play Console or run `moth setup billing --project "+d.Slug+"` to re-push")
	case checked == 0:
		rep.Skip(name, "no products carry a Google SKU")
	default:
		rep.Pass(name, fmt.Sprintf("%d product(s) present in Google Play", checked))
	}
}

// checkStripeBilling reports on the project's Stripe configuration — key and
// webhook-secret presence, moth's webhook route — and, when the operator
// supplies the secret key in hand (moth never returns the stored one), a live
// reachability probe against the Stripe API. Mirrors checkAppleBilling's
// in-hand-credential pattern.
func (d *Doctor) checkStripeBilling(ctx context.Context, rep *Report, sc *adminv1.StripeBillingConfig) {
	const name = "project: Stripe billing credentials"
	if !sc.GetHasSecretKey() {
		rep.Fail(name, "no secret key stored",
			"run `moth setup billing --project "+d.Slug+" --stripe-secret-key <key>`")
	} else {
		rep.Pass(name, "configured (secret key stored)")
	}
	if !sc.GetHasWebhookSecret() {
		rep.Warn("project: Stripe webhook secret", "no webhook signing secret stored — Stripe events cannot be verified",
			"re-run `moth setup billing` with --stripe-secret-key to create the endpoint and store the secret")
	} else {
		detail := "stored"
		if sc.GetWebhookEndpointId() != "" {
			detail = "stored (endpoint " + sc.GetWebhookEndpointId() + ")"
		}
		rep.Pass("project: Stripe webhook secret", detail)
	}
	d.checkWebhookReachable(ctx, rep, "project: Stripe webhook endpoint", "/billing/stripe/webhook/"+d.Slug)

	const probeName = "project: Stripe API reachable"
	if d.StripeSecretKey == "" {
		rep.Warn(probeName, "moth stores the secret key encrypted and never returns it, so a remote doctor cannot probe the Stripe API",
			"re-run with --stripe-secret-key <key> to verify the key")
		return
	}
	rep.add(stripeAPIProbe(ctx, probeName, d.HTTPC, d.StripeAPIBase, d.StripeSecretKey))
}

// stripeAPIProbe reads a bogus price id against the Stripe API: a 404
// (billing.ErrNotFound) proves the key authenticated and the API is reachable,
// while an auth error proves the key is wrong. Shared semantics with
// appleServerAPIProbe / googlePlayAPIProbe.
func stripeAPIProbe(ctx context.Context, name string, httpc billing.Doer, base, secretKey string) Check {
	client := &billing.StripeClient{BaseURL: base, SecretKey: secretKey, HTTPC: httpc}
	_, err := client.GetPrice(ctx, "price_moth_doctor_probe")
	switch {
	case err == nil, errors.Is(err, billing.ErrNotFound):
		return Check{Name: name, Status: StatusPass, Detail: "authenticated; price read reached the API"}
	default:
		return Check{Name: name, Status: StatusFail,
			Detail:      "Stripe rejected the request: " + err.Error(),
			Remediation: "check the secret key (sk_/rk_) is valid and has read access to Prices"}
	}
}

// checkCatalogMapping summarizes catalog drift the CLI can see without the
// per-store sync records: products missing a store SKU for a configured store.
func (d *Doctor) checkCatalogMapping(ctx context.Context, rep *Report, projectID string, apple, google bool) {
	const name = "project: catalog store mapping"
	if d.Products == nil {
		return
	}
	resp, err := d.Products.ListProducts(ctx, connect.NewRequest(&adminv1.ListProductsRequest{ProjectId: projectID}))
	if err != nil {
		rep.Fail(name, "ListProducts: "+err.Error(), "")
		return
	}
	products := resp.Msg.Products
	if len(products) == 0 {
		rep.Warn(name, "no products defined — nothing to sell",
			"define products in the admin Monetization screen or `moth project apply`")
		return
	}
	var unmapped []string
	for _, p := range products {
		if apple && p.GetAppleProductId() == "" {
			unmapped = append(unmapped, p.GetIdentifier()+" (Apple)")
		}
		if google && p.GetGoogleProductId() == "" {
			unmapped = append(unmapped, p.GetIdentifier()+" (Google)")
		}
	}
	if len(unmapped) > 0 {
		rep.Warn(name, fmt.Sprintf("%d product(s) have no store SKU: %s", len(unmapped), strings.Join(unmapped, ", ")),
			"set the store product ids and run `moth setup billing --project "+d.Slug+"` to push them")
		return
	}
	rep.Pass(name, fmt.Sprintf("%d product(s) mapped to every configured store", len(products)))
}

// checkWebhookReachable confirms moth's own notification route exists. It POSTs
// an empty body: a wired endpoint rejects it before doing any work (Apple 400
// "invalid notification body", Google 401/unauthorized), while an unregistered
// route falls through to moth's catch-all handler and 404s. A GET cannot be used
// — moth serves every unmatched GET from the SPA/root handler, so a GET to a
// live POST-only webhook also 404s (the catch-all shadows the method-mismatch),
// which would misreport a wired route as missing.
func (d *Doctor) checkWebhookReachable(ctx context.Context, rep *Report, name, path string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.BaseURL+path, http.NoBody)
	if err != nil {
		rep.Fail(name, err.Error(), "")
		return
	}
	resp, err := d.HTTPC.Do(req)
	if err != nil {
		rep.Fail(name, fmt.Sprintf("POST %s: %v", path, err), "is the server reachable from here?")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		rep.Fail(name, "POST "+path+" → 404 (route not registered)", "the server build is missing the billing webhooks")
		return
	}
	rep.Pass(name, "reachable ("+d.BaseURL+path+")")
}

// googlePlayAPIProbe mints an access token from the service account and reads a
// bogus purchase token against the Play Developer API: a 404 (billing.ErrNotFound)
// proves the SA authenticated and has access; an auth error proves it does not.
// Shared between `moth setup billing` and `moth doctor`.
func googlePlayAPIProbe(ctx context.Context, name string, httpc billing.Doer, base, tokenURL, packageName string, sa *billing.GoogleServiceAccount) Check {
	tokens := billing.NewGoogleTokenSource(sa, tokenURL, httpc, nil)
	if _, err := tokens.Token(ctx); err != nil {
		return Check{Name: name, Status: StatusFail,
			Detail:      "service account could not obtain an access token: " + err.Error(),
			Remediation: "check the service-account JSON is valid and unexpired"}
	}
	client := &billing.GoogleClient{BaseURL: base, PackageName: packageName, Tokens: tokens, HTTPC: httpc}
	_, _, err := client.GetSubscriptionV2(ctx, "moth-doctor-probe")
	switch {
	case err == nil, errors.Is(err, billing.ErrNotFound):
		return Check{Name: name, Status: StatusPass, Detail: "authenticated for " + packageName}
	default:
		return Check{Name: name, Status: StatusFail,
			Detail:      "Play Developer API rejected the request: " + err.Error(),
			Remediation: "grant the service account Play Developer API access to " + packageName}
	}
}
