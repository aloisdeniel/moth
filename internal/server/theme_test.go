package server

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	projectconfigv1 "github.com/aloisdeniel/moth/gen/moth/projectconfig/v1"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/theme"
)

func (e *testEnv) themes() adminv1connect.ThemeServiceClient {
	return adminv1connect.NewThemeServiceClient(e.client, e.url)
}

// getTheme fetches a project's current admin theme.
func (e *testEnv) getTheme(t *testing.T, projectID string) *adminv1.GetThemeResponse {
	t.Helper()
	got, err := e.themes().GetTheme(context.Background(),
		connect.NewRequest(&adminv1.GetThemeRequest{ProjectId: projectID}))
	if err != nil {
		t.Fatal(err)
	}
	return got.Msg
}

// updateTheme fetches, mutates and saves the project theme, returning the
// new revision id.
func (e *testEnv) updateTheme(t *testing.T, projectID string, mutate func(*adminv1.Theme)) string {
	t.Helper()
	cur := e.getTheme(t, projectID).Theme
	mutate(cur)
	got, err := e.themes().UpdateTheme(context.Background(),
		connect.NewRequest(&adminv1.UpdateThemeRequest{ProjectId: projectID, Theme: cur}))
	if err != nil {
		t.Fatal(err)
	}
	return got.Msg.RevisionId
}

func testLogoPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 48, 48))
	for i := 0; i < 48; i++ {
		img.Set(i, i, color.RGBA{R: 200, A: 255})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestThemeLifecycle(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Themed App")

	// A fresh project renders the built-in defaults.
	got := e.getTheme(t, p.Id)
	if !got.IsDefault || got.RevisionId != "" {
		t.Fatalf("fresh project: %+v", got)
	}
	if got.Theme.Colors.Primary != "#6750A4" || got.Theme.Typography.FontFamily != "Inter" {
		t.Fatalf("default theme: %+v", got.Theme)
	}

	// Save a custom primary; logo paths sent by the client are ignored.
	rev1 := e.updateTheme(t, p.Id, func(th *adminv1.Theme) {
		th.Colors.Primary = "#0B57D0"
		th.Logo = &adminv1.ThemeLogo{LightPath: "/assets/evil/nope.png"}
	})
	got = e.getTheme(t, p.Id)
	if got.IsDefault || got.RevisionId != rev1 {
		t.Fatalf("after update: %+v", got)
	}
	if got.Theme.Colors.Primary != "#0B57D0" {
		t.Fatalf("primary = %q", got.Theme.Colors.Primary)
	}
	if got.Theme.Logo.LightPath != "" {
		t.Fatalf("logo path must be output-only, got %q", got.Theme.Logo.LightPath)
	}

	// An illegible palette is rejected with a clear message.
	_, err := e.themes().UpdateTheme(ctx, connect.NewRequest(&adminv1.UpdateThemeRequest{
		ProjectId: p.Id,
		Theme: func() *adminv1.Theme {
			th := e.getTheme(t, p.Id).Theme
			th.Colors.OnPrimary = "#0B57D0" // same as primary: contrast 1:1
			return th
		}(),
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument ||
		!strings.Contains(err.Error(), "contrast") {
		t.Fatalf("contrast violation: %v", err)
	}

	// Unknown fonts and out-of-range tokens are rejected too.
	_, err = e.themes().UpdateTheme(ctx, connect.NewRequest(&adminv1.UpdateThemeRequest{
		ProjectId: p.Id,
		Theme: func() *adminv1.Theme {
			th := e.getTheme(t, p.Id).Theme
			th.Typography.FontFamily = "Comic Sans MS"
			return th
		}(),
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unknown font: %v", err)
	}

	// Every save is a revision; the history is pruned to the newest 10.
	for i := 0; i < 12; i++ {
		e.updateTheme(t, p.Id, func(th *adminv1.Theme) {
			th.Shape.CornerRadius = int32(i + 1)
		})
	}
	revs, err := e.themes().ListThemeRevisions(ctx,
		connect.NewRequest(&adminv1.ListThemeRevisionsRequest{ProjectId: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if len(revs.Msg.Revisions) != store.ThemeRevisionKeep {
		t.Fatalf("revisions kept = %d, want %d", len(revs.Msg.Revisions), store.ThemeRevisionKeep)
	}
	newest := revs.Msg.Revisions[0]
	if newest.Theme.Shape.CornerRadius != 12 {
		t.Fatalf("newest revision radius = %d", newest.Theme.Shape.CornerRadius)
	}

	// Restore an older revision: its tokens come back as a NEW revision.
	older := revs.Msg.Revisions[3] // radius 9
	restored, err := e.themes().RestoreThemeRevision(ctx,
		connect.NewRequest(&adminv1.RestoreThemeRevisionRequest{
			ProjectId: p.Id, RevisionId: older.RevisionId,
		}))
	if err != nil {
		t.Fatal(err)
	}
	if restored.Msg.RevisionId == older.RevisionId {
		t.Fatal("restore must mint a new revision id")
	}
	got = e.getTheme(t, p.Id)
	if got.Theme.Shape.CornerRadius != older.Theme.Shape.CornerRadius {
		t.Fatalf("restored radius = %d, want %d",
			got.Theme.Shape.CornerRadius, older.Theme.Shape.CornerRadius)
	}

	// Restoring another project's revision is NotFound.
	p2, _ := e.createProject(t, "Other App")
	_, err = e.themes().RestoreThemeRevision(ctx,
		connect.NewRequest(&adminv1.RestoreThemeRevisionRequest{
			ProjectId: p2.Id, RevisionId: older.RevisionId,
		}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("cross-project restore: %v", err)
	}

	// Reset returns to the defaults but keeps the history restorable.
	if _, err := e.themes().ResetTheme(ctx,
		connect.NewRequest(&adminv1.ResetThemeRequest{ProjectId: p.Id})); err != nil {
		t.Fatal(err)
	}
	got = e.getTheme(t, p.Id)
	if !got.IsDefault || got.Theme.Colors.Primary != "#6750A4" {
		t.Fatalf("after reset: %+v", got)
	}
	revs, err = e.themes().ListThemeRevisions(ctx,
		connect.NewRequest(&adminv1.ListThemeRevisionsRequest{ProjectId: p.Id}))
	if err != nil || len(revs.Msg.Revisions) == 0 {
		t.Fatalf("history after reset: %v %d", err, len(revs.Msg.Revisions))
	}

	// Theme RPCs require an admin session.
	anon := adminv1connect.NewThemeServiceClient(http.DefaultClient, e.url)
	_, err = anon.GetTheme(ctx, connect.NewRequest(&adminv1.GetThemeRequest{ProjectId: p.Id}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("anonymous GetTheme: %v", err)
	}
}

func TestLogoUploadAndAssetServing(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Logo App")

	// Oversized uploads are rejected before any decoding.
	_, err := e.themes().UploadLogo(ctx, connect.NewRequest(&adminv1.UploadLogoRequest{
		ProjectId: p.Id, Variant: adminv1.LogoVariant_LOGO_VARIANT_LIGHT,
		Data: make([]byte, 512*1024+1), ContentType: "image/png",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("oversized upload: %v", err)
	}

	// Bodies past the global read cap are rejected while being read,
	// before connect buffers the whole message for the handler.
	_, err = e.themes().UploadLogo(ctx, connect.NewRequest(&adminv1.UploadLogoRequest{
		ProjectId: p.Id, Variant: adminv1.LogoVariant_LOGO_VARIANT_LIGHT,
		Data: make([]byte, 2<<20), ContentType: "image/png",
	}))
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("over-read-cap upload: %v", err)
	}

	// A hostile SVG (html smuggling) is rejected outright.
	_, err = e.themes().UploadLogo(ctx, connect.NewRequest(&adminv1.UploadLogoRequest{
		ProjectId: p.Id, Variant: adminv1.LogoVariant_LOGO_VARIANT_LIGHT,
		Data: []byte("<html><script>evil()</script></html>"), ContentType: "image/svg+xml",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("hostile svg: %v", err)
	}

	// PNG upload: stored bytes are a re-encode, not the original (the
	// trailing junk stands in for a payload hidden in the upload).
	raw := append(testLogoPNG(t), []byte("EVILPAYLOAD")...)
	up, err := e.themes().UploadLogo(ctx, connect.NewRequest(&adminv1.UploadLogoRequest{
		ProjectId: p.Id, Variant: adminv1.LogoVariant_LOGO_VARIANT_LIGHT,
		Data: raw, ContentType: "image/png",
	}))
	if err != nil {
		t.Fatal(err)
	}
	wantPath := "/assets/" + p.Id + "/logo-light.png"
	if up.Msg.Path != wantPath || up.Msg.Theme.Logo.LightPath != wantPath {
		t.Fatalf("upload path = %q / %q, want %q", up.Msg.Path, up.Msg.Theme.Logo.LightPath, wantPath)
	}

	resp, err := e.client.Get(e.url + wantPath + "?v=" + up.Msg.RevisionId)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: %d", wantPath, resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "image/png" {
		t.Errorf("content type = %q", got)
	}
	if cc := resp.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("cache control = %q", cc)
	}
	etag := resp.Header.Get("ETag")
	if etag == "" {
		t.Error("missing ETag")
	}
	if bytes.Equal(body, raw) || bytes.Contains(body, []byte("EVILPAYLOAD")) {
		t.Error("served logo must be a re-encode without the smuggled payload")
	}
	if _, err := png.Decode(bytes.NewReader(body)); err != nil {
		t.Errorf("served logo does not decode: %v", err)
	}

	// Conditional requests hit the ETag — including the RFC 9110 forms a
	// proxy or shared cache may send: a weakened validator, a list of
	// validators, and "*".
	for _, header := range []string{
		etag,
		"W/" + etag,
		`"another", ` + etag,
		`"another", W/` + etag,
		"*",
	} {
		req, _ := http.NewRequest(http.MethodGet, e.url+wantPath, nil)
		req.Header.Set("If-None-Match", header)
		resp2, err := e.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp2.Body.Close()
		if resp2.StatusCode != http.StatusNotModified {
			t.Errorf("If-None-Match %q: %d, want 304", header, resp2.StatusCode)
		}
	}
	// A non-matching validator still gets the full response.
	req, _ := http.NewRequest(http.MethodGet, e.url+wantPath, nil)
	req.Header.Set("If-None-Match", `"0000000000000000"`)
	resp2, err := e.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("non-matching If-None-Match: %d, want 200", resp2.StatusCode)
	}

	// SVG with a script survives upload sanitized: the script is gone.
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">` +
		`<script>alert(document.cookie)</script><circle r="4" onload="evil()"/></svg>`)
	up2, err := e.themes().UploadLogo(ctx, connect.NewRequest(&adminv1.UploadLogoRequest{
		ProjectId: p.Id, Variant: adminv1.LogoVariant_LOGO_VARIANT_DARK,
		Data: svg, ContentType: "image/svg+xml",
	}))
	if err != nil {
		t.Fatal(err)
	}
	svgPath := "/assets/" + p.Id + "/logo-dark.svg"
	if up2.Msg.Path != svgPath {
		t.Fatalf("svg path = %q", up2.Msg.Path)
	}
	resp3, err := e.client.Get(e.url + svgPath)
	if err != nil {
		t.Fatal(err)
	}
	stored, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK || resp3.Header.Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("GET svg: %d %q", resp3.StatusCode, resp3.Header.Get("Content-Type"))
	}
	lower := strings.ToLower(string(stored))
	if strings.Contains(lower, "script") || strings.Contains(lower, "onload") {
		t.Errorf("stored SVG still contains script/onload:\n%s", stored)
	}
	if !strings.Contains(lower, "circle") {
		t.Errorf("stored SVG lost its shape:\n%s", stored)
	}

	// Unknown filenames, foreign directories and non-UUID segments 404.
	for _, path := range []string{
		"/assets/" + p.Id + "/other.png",
		"/assets/" + p.Id + "/logo-light.svg", // replaced by nothing: only .png exists
		"/assets/not-a-uuid/logo-light.png",
		"/assets/../moth.db",
		"/assets/" + p.Id,
	} {
		resp, err := e.client.Get(e.url + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s: %d, want 404", path, resp.StatusCode)
		}
	}

	// The embedded fonts are served under the same prefix.
	respFont, err := e.client.Get(e.url + "/assets/fonts/inter/Inter.ttf")
	if err != nil {
		t.Fatal(err)
	}
	respFont.Body.Close()
	if respFont.StatusCode != http.StatusOK || respFont.Header.Get("Content-Type") != "font/ttf" {
		t.Fatalf("font: %d %q", respFont.StatusCode, respFont.Header.Get("Content-Type"))
	}

	// DeleteLogo clears the theme pointer and the file.
	if _, err := e.themes().DeleteLogo(ctx, connect.NewRequest(&adminv1.DeleteLogoRequest{
		ProjectId: p.Id, Variant: adminv1.LogoVariant_LOGO_VARIANT_LIGHT,
	})); err != nil {
		t.Fatal(err)
	}
	if got := e.getTheme(t, p.Id); got.Theme.Logo.LightPath != "" {
		t.Fatalf("logo path after delete = %q", got.Theme.Logo.LightPath)
	}
	resp4, err := e.client.Get(e.url + wantPath)
	if err != nil {
		t.Fatal(err)
	}
	resp4.Body.Close()
	if resp4.StatusCode != http.StatusNotFound {
		t.Errorf("deleted logo still served: %d", resp4.StatusCode)
	}
}

func TestRestoreThemeRevisionLogoFollowsDisk(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Restore Logo App")

	// rev0 predates any logo upload.
	rev0 := e.updateTheme(t, p.Id, func(th *adminv1.Theme) {
		th.Shape.CornerRadius = 5
	})

	// Upload an SVG light logo, then save a token edit: rev2 records the
	// .svg pointer.
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><circle r="4"/></svg>`)
	if _, err := e.themes().UploadLogo(ctx, connect.NewRequest(&adminv1.UploadLogoRequest{
		ProjectId: p.Id, Variant: adminv1.LogoVariant_LOGO_VARIANT_LIGHT,
		Data: svg, ContentType: "image/svg+xml",
	})); err != nil {
		t.Fatal(err)
	}
	rev2 := e.updateTheme(t, p.Id, func(th *adminv1.Theme) {
		th.Shape.CornerRadius = 20
	})

	// Replace the logo with a PNG: logo-light.svg is deleted from disk.
	if _, err := e.themes().UploadLogo(ctx, connect.NewRequest(&adminv1.UploadLogoRequest{
		ProjectId: p.Id, Variant: adminv1.LogoVariant_LOGO_VARIANT_LIGHT,
		Data: testLogoPNG(t), ContentType: "image/png",
	})); err != nil {
		t.Fatal(err)
	}
	pngPath := "/assets/" + p.Id + "/logo-light.png"

	// Restoring rev2 must re-point at the logo file as it exists today
	// (.png), not drop it because the recorded .svg filename is gone.
	restored, err := e.themes().RestoreThemeRevision(ctx,
		connect.NewRequest(&adminv1.RestoreThemeRevisionRequest{
			ProjectId: p.Id, RevisionId: rev2,
		}))
	if err != nil {
		t.Fatal(err)
	}
	if got := restored.Msg.Theme.Logo.LightPath; got != pngPath {
		t.Errorf("restored logo path = %q, want %q", got, pngPath)
	}

	// A revision that predates any upload also re-attaches today's file.
	restored, err = e.themes().RestoreThemeRevision(ctx,
		connect.NewRequest(&adminv1.RestoreThemeRevisionRequest{
			ProjectId: p.Id, RevisionId: rev0,
		}))
	if err != nil {
		t.Fatal(err)
	}
	if got := restored.Msg.Theme.Logo.LightPath; got != pngPath {
		t.Errorf("restored pre-upload logo path = %q, want %q", got, pngPath)
	}
	if got := e.getTheme(t, p.Id); got.Theme.Logo.LightPath != pngPath ||
		got.Theme.Shape.CornerRadius != 5 {
		t.Errorf("current theme after restore: %+v", got.Theme)
	}
}

func TestUnparseableStoredTheme(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Future Theme App")

	// Two good revisions to keep restorable.
	e.updateTheme(t, p.Id, func(th *adminv1.Theme) { th.Shape.CornerRadius = 5 })
	goodRev := e.updateTheme(t, p.Id, func(th *adminv1.Theme) { th.Colors.Primary = "#0B57D0" })

	// Simulate a database last written by a future moth: a schema-v2
	// document this binary cannot parse, installed as the current theme.
	future, err := proto.Marshal(&projectconfigv1.StoredTheme{
		Version: 2,
		Colors:  &projectconfigv1.ThemeColors{Primary: "#123456"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.store.SetProjectTheme(ctx, store.ThemeRevision{
		ID: "rev-future", ProjectID: p.Id, Theme: future, CreatedAt: time.Now(),
	}, goodRev); err != nil {
		t.Fatal(err)
	}

	// Read path: defensively renders the defaults.
	if got := e.getTheme(t, p.Id); !got.IsDefault {
		t.Errorf("unparseable current theme must read as default: %+v", got)
	}

	// Write paths must refuse instead of silently replacing the newer
	// document with a default.
	_, err = e.themes().UploadLogo(ctx, connect.NewRequest(&adminv1.UploadLogoRequest{
		ProjectId: p.Id, Variant: adminv1.LogoVariant_LOGO_VARIANT_LIGHT,
		Data: testLogoPNG(t), ContentType: "image/png",
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("UploadLogo on unparseable theme: %v", err)
	}
	_, err = e.themes().UpdateTheme(ctx, connect.NewRequest(&adminv1.UpdateThemeRequest{
		ProjectId: p.Id, Theme: e.getTheme(t, p.Id).Theme,
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("UpdateTheme on unparseable theme: %v", err)
	}
	proj, err := e.store.GetProject(ctx, p.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(proj.Theme, future) {
		t.Errorf("stored document was replaced: %q", proj.Theme)
	}

	// The revision list skips the unparseable entry instead of failing
	// wholesale, so the good revisions stay discoverable...
	revs, err := e.themes().ListThemeRevisions(ctx,
		connect.NewRequest(&adminv1.ListThemeRevisionsRequest{ProjectId: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if len(revs.Msg.Revisions) != 2 {
		t.Fatalf("revisions listed = %d, want 2 (unparseable one skipped)", len(revs.Msg.Revisions))
	}
	for _, rev := range revs.Msg.Revisions {
		if rev.RevisionId == "rev-future" {
			t.Errorf("unparseable revision listed: %+v", rev)
		}
	}

	// ...and restoring one of them recovers the project.
	restored, err := e.themes().RestoreThemeRevision(ctx,
		connect.NewRequest(&adminv1.RestoreThemeRevisionRequest{
			ProjectId: p.Id, RevisionId: goodRev,
		}))
	if err != nil {
		t.Fatal(err)
	}
	if restored.Msg.Theme.Colors.Primary != "#0B57D0" {
		t.Errorf("restored primary = %q", restored.Msg.Theme.Colors.Primary)
	}
	if got := e.getTheme(t, p.Id); got.IsDefault || got.Theme.Colors.Primary != "#0B57D0" {
		t.Errorf("theme after recovery: %+v", got)
	}
}

func TestGetProjectConfigTheme(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Config Theme App")
	cfg := e.configClient(p.PublishableKey)

	// First call: the default theme is present under the sentinel revision.
	got, err := cfg.GetProjectConfig(ctx, connect.NewRequest(&authv1.GetProjectConfigRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	th := got.Msg.Theme
	if th == nil {
		t.Fatal("first call must include the theme")
	}
	if th.RevisionId != authrpc.DefaultThemeRevision {
		t.Fatalf("default revision = %q", th.RevisionId)
	}
	if th.Colors.Primary != "#6750A4" || th.DarkColors.Primary == "" ||
		th.DarkColors.Primary == th.Colors.Primary {
		t.Fatalf("palettes: %+v", th)
	}
	if th.FontFamily != "Inter" || !strings.HasSuffix(th.FontUrl, "/assets/fonts/inter/Inter.ttf") ||
		!strings.HasPrefix(th.FontUrl, "http") {
		t.Fatalf("font: %q %q", th.FontFamily, th.FontUrl)
	}

	// Echoing the revision omits the body; anything else re-sends it.
	got, err = cfg.GetProjectConfig(ctx, connect.NewRequest(
		&authv1.GetProjectConfigRequest{KnownThemeRevision: th.RevisionId}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.Theme != nil {
		t.Fatal("matching known_theme_revision must omit the theme")
	}

	// An admin edit reflects immediately with a new revision.
	rev := e.updateTheme(t, p.Id, func(th *adminv1.Theme) {
		th.Colors.Primary = "#0B57D0"
		th.Legal = &adminv1.ThemeLegal{TermsUrl: "https://example.com/terms"}
	})
	got, err = cfg.GetProjectConfig(ctx, connect.NewRequest(
		&authv1.GetProjectConfigRequest{KnownThemeRevision: authrpc.DefaultThemeRevision}))
	if err != nil {
		t.Fatal(err)
	}
	th = got.Msg.Theme
	if th == nil || th.RevisionId != rev || th.Colors.Primary != "#0B57D0" ||
		th.TermsUrl != "https://example.com/terms" {
		t.Fatalf("after edit: %+v", th)
	}

	// Logo URLs are absolute and revision-keyed.
	up, err := e.themes().UploadLogo(ctx, connect.NewRequest(&adminv1.UploadLogoRequest{
		ProjectId: p.Id, Variant: adminv1.LogoVariant_LOGO_VARIANT_LIGHT,
		Data: testLogoPNG(t), ContentType: "image/png",
	}))
	if err != nil {
		t.Fatal(err)
	}
	got, err = cfg.GetProjectConfig(ctx, connect.NewRequest(&authv1.GetProjectConfigRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	wantLogo := fmt.Sprintf("http://localhost:8080/assets/%s/logo-light.png?v=%s",
		p.Id, up.Msg.RevisionId)
	if got.Msg.Theme.LogoLightUrl != wantLogo {
		t.Fatalf("logo url = %q, want %q", got.Msg.Theme.LogoLightUrl, wantLogo)
	}
}

func TestHostedPagesThemed(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Page Theme App")

	// Default theme: the CSS custom properties and font-face are inlined.
	page := e.getPage(t, e.url+"/p/"+p.Slug+"/reset?token=x")
	for _, want := range []string{
		"--primary: #6750A4;",
		"--radius: 12px;",
		"@font-face",
		"/assets/fonts/inter/Inter.ttf",
		"prefers-color-scheme: dark",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("default page missing %q", want)
		}
	}
	// The dark block must carry the *derived* dark palette, not a copy of
	// the light one.
	darkPrimary := theme.Default().EffectiveDark().Primary
	if darkPrimary == "#6750A4" {
		t.Fatal("derived dark primary must differ from the light primary")
	}
	assertDarkCSS(t, page, darkPrimary)
	if strings.Contains(page, "class=\"logo") {
		t.Error("page must not render a logo before one is uploaded")
	}

	// Change the theme + upload a logo: the very next render reflects it.
	e.updateTheme(t, p.Id, func(th *adminv1.Theme) {
		th.Colors.Primary = "#0B57D0"
		th.Legal = &adminv1.ThemeLegal{
			TermsUrl:   "https://example.com/terms",
			PrivacyUrl: "https://example.com/privacy",
		}
	})
	up, err := e.themes().UploadLogo(ctx, connect.NewRequest(&adminv1.UploadLogoRequest{
		ProjectId: p.Id, Variant: adminv1.LogoVariant_LOGO_VARIANT_LIGHT,
		Data: testLogoPNG(t), ContentType: "image/png",
	}))
	if err != nil {
		t.Fatal(err)
	}
	page = e.getPage(t, e.url+"/p/"+p.Slug+"/reset?token=x")
	for _, want := range []string{
		"--primary: #0B57D0;",
		"/assets/" + p.Id + "/logo-light.png?v=" + up.Msg.RevisionId,
		`href="https://example.com/terms"`,
		`href="https://example.com/privacy"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("themed page missing %q", want)
		}
	}
	edited := theme.Default()
	edited.Colors.Primary = "#0B57D0"
	assertDarkCSS(t, page, edited.EffectiveDark().Primary)
}

// assertDarkCSS asserts that the page carries a prefers-color-scheme block
// whose --primary is the given dark value.
func assertDarkCSS(t *testing.T, page, darkPrimary string) {
	t.Helper()
	idx := strings.Index(page, "prefers-color-scheme: dark")
	if idx < 0 {
		t.Error("page has no dark media query")
		return
	}
	if !strings.Contains(page[idx:], "--primary: "+darkPrimary+";") {
		t.Errorf("dark block missing --primary: %s", darkPrimary)
	}
}
