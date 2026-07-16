package server

import (
	"context"
	"html"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/store"
)

func (e *testEnv) auditSvc() adminv1connect.AuditServiceClient {
	return adminv1connect.NewAuditServiceClient(e.client, e.url)
}

// bearerTransport authenticates every request with a personal access token
// (no cookie jar), mimicking the CLI.
type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r)
}

// mintPAT creates a personal access token through the cookie-authed admin API.
func (e *testEnv) mintPAT(t *testing.T, name string) string {
	t.Helper()
	resp, err := e.adminAccounts().CreatePersonalAccessToken(context.Background(),
		connect.NewRequest(&adminv1.CreatePersonalAccessTokenRequest{Name: name}))
	if err != nil {
		t.Fatal(err)
	}
	return resp.Msg.Token
}

// TestAuditActorAttribution: every admin mutation is recorded, attributed to
// the exact credential — the browser session (cookie) and a personal access
// token produce distinct actor types.
func TestAuditActorAttribution(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	ctx := context.Background()

	// A cookie-authed project create.
	cookieProject, _ := e.createProject(t, "Cookie App")

	// A PAT-authed project create over a separate, cookieless client.
	pat := e.mintPAT(t, "ci-token")
	patClient := &http.Client{Transport: bearerTransport{pat}}
	patProjects := adminv1connect.NewProjectServiceClient(patClient, e.url)
	patCreated, err := patProjects.CreateProject(ctx,
		connect.NewRequest(&adminv1.CreateProjectRequest{Name: "PAT App"}))
	if err != nil {
		t.Fatal(err)
	}

	entries, err := e.store.ListAudit(ctx, store.AuditFilter{Action: adminActionProjectCreate})
	if err != nil {
		t.Fatal(err)
	}
	byProject := map[string]store.AuditEntry{}
	for _, en := range entries {
		byProject[en.TargetID] = en
	}
	c := byProject[cookieProject.Id]
	if c.ActorType != store.AuditActorCookie {
		t.Fatalf("cookie action actor_type = %q, want %q", c.ActorType, store.AuditActorCookie)
	}
	if c.ActorLabel != "ops@example.com" || c.ActorID == "" {
		t.Fatalf("cookie action actor not attributed: %+v", c)
	}
	p := byProject[patCreated.Msg.Project.Id]
	if p.ActorType != store.AuditActorPAT {
		t.Fatalf("pat action actor_type = %q, want %q", p.ActorType, store.AuditActorPAT)
	}
	if p.ActorID == "" || p.ProjectID != patCreated.Msg.Project.Id {
		t.Fatalf("pat action not attributed: %+v", p)
	}
	// The coarse client IP is recorded, never the full loopback address.
	if c.IP == "" || strings.Contains(c.IP, ":") {
		t.Fatalf("expected a coarse IPv4 prefix, got %q", c.IP)
	}

	// The PAT create itself was audited too (pat.create).
	pats, err := e.store.ListAudit(ctx, store.AuditFilter{Action: adminActionPATCreate})
	if err != nil {
		t.Fatal(err)
	}
	if len(pats) != 1 || pats[0].ActorType != store.AuditActorCookie {
		t.Fatalf("pat.create audit: %+v", pats)
	}
}

// mirror the machine action names asserted here (kept local so the test does
// not reach into the adminrpc package internals).
const (
	adminActionProjectCreate = "project.create"
	adminActionPATCreate     = "pat.create"
	adminActionUserDisable   = "user.disable"
)

// TestAuditListAndPaging drives the AuditService: filters narrow the result
// and page_size / page_token walk it newest-first.
func TestAuditListAndPaging(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Audited App")

	// Generate a handful of user actions to page through.
	users := e.adminUsers()
	var ids []string
	for _, name := range []string{"a", "b", "c"} {
		u, err := users.CreateUser(ctx, connect.NewRequest(&adminv1.CreateUserRequest{
			ProjectId: p.Id, Email: name + "@example.com", Password: "password-1"}))
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, u.Msg.User.Id)
	}
	if _, err := users.DisableUser(ctx, connect.NewRequest(&adminv1.DisableUserRequest{
		ProjectId: p.Id, UserId: ids[0]})); err != nil {
		t.Fatal(err)
	}

	svc := e.auditSvc()

	// Filter by action: exactly one user.disable entry.
	disables, err := svc.ListAuditLog(ctx, connect.NewRequest(&adminv1.ListAuditLogRequest{
		ProjectId: p.Id, Action: adminActionUserDisable}))
	if err != nil {
		t.Fatal(err)
	}
	if len(disables.Msg.Entries) != 1 {
		t.Fatalf("want 1 disable entry, got %d", len(disables.Msg.Entries))
	}

	// Page the project's log one entry at a time; ids strictly descend.
	seen := 0
	var lastID string
	token := ""
	for {
		page, err := svc.ListAuditLog(ctx, connect.NewRequest(&adminv1.ListAuditLogRequest{
			ProjectId: p.Id, PageSize: 1, PageToken: token}))
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Msg.Entries) == 0 {
			break
		}
		if len(page.Msg.Entries) != 1 {
			t.Fatalf("page_size=1 returned %d rows", len(page.Msg.Entries))
		}
		id := page.Msg.Entries[0].Id
		if lastID != "" && id >= lastID {
			t.Fatalf("ids not strictly newest-first: %s then %s", lastID, id)
		}
		lastID = id
		seen++
		if page.Msg.NextPageToken == "" {
			break
		}
		token = page.Msg.NextPageToken
		if seen > 50 {
			t.Fatal("paging did not terminate")
		}
	}
	// project.create is project-scoped too: 1 create + 3 user.create +
	// 1 user.disable = 5 entries under this project.
	if seen != 5 {
		t.Fatalf("paged %d project entries, want 5", seen)
	}
}

// TestCSVSafe covers every dangerous lead character csvSafe must neutralize,
// so a regression narrowing the switch (e.g. handling only '=') is caught.
func TestCSVSafe(t *testing.T) {
	dangerous := []struct {
		name string
		in   string
	}{
		{"equals", "=HYPERLINK(\"http://evil\")"},
		{"plus", "+1+2"},
		{"minus", "-2+3+cmd"},
		{"at", "@SUM(A1:A9)"},
		{"tab", "\tSUM(1)"},
		{"carriage-return", "\rSUM(1)"},
	}
	for _, tc := range dangerous {
		t.Run(tc.name, func(t *testing.T) {
			got := csvSafe(tc.in)
			if got != "'"+tc.in {
				t.Fatalf("csvSafe(%q) = %q, want it single-quote prefixed", tc.in, got)
			}
		})
	}
	// Safe values pass through untouched.
	for _, safe := range []string{"", "user@example.com is fine mid-string", "Project Alpha", "123"} {
		if got := csvSafe(safe); got != safe {
			t.Fatalf("csvSafe(%q) = %q, want unchanged", safe, got)
		}
	}
}

// TestAuditCSVExport: the endpoint needs an admin credential, accepts a PAT,
// and defuses spreadsheet formula injection in every cell.
func TestAuditCSVExport(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	ctx := context.Background()
	e.createProject(t, "Export App")

	// A dangerous summary reaches the log through a direct store write (an
	// admin display name could carry one in practice).
	if err := e.store.AppendAudit(ctx, store.AuditEntry{
		ID: "0193c000-0000-7000-8000-000000000001", ActorType: store.AuditActorSystem,
		Action: "test.injection", TargetType: "test", TargetID: "x",
		Summary: "=HYPERLINK(\"http://evil\")", CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	// No credential: rejected.
	noAuth := &http.Client{}
	resp, err := noAuth.Get(e.url + "/admin/export/audit.csv")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated export: want 401, got %d", resp.StatusCode)
	}

	// Cookie-authed: succeeds and the injection cell is neutralized.
	body := e.getCSV(t, e.client, e.url+"/admin/export/audit.csv")
	if !strings.Contains(body, "'=HYPERLINK") {
		t.Fatalf("formula cell not escaped in CSV:\n%s", body)
	}
	if strings.Contains(body, "\n=HYPERLINK") || strings.Contains(body, ",=HYPERLINK") {
		t.Fatalf("raw formula leaked into CSV:\n%s", body)
	}

	// PAT-authed: also succeeds.
	pat := e.mintPAT(t, "export-token")
	patClient := &http.Client{Transport: bearerTransport{pat}}
	if b := e.getCSV(t, patClient, e.url+"/admin/export/audit.csv"); !strings.Contains(b, "id,create_time,actor_type") {
		t.Fatalf("PAT export missing header:\n%s", b)
	}
}

func (e *testEnv) getCSV(t *testing.T, c *http.Client, url string) string {
	t.Helper()
	resp, err := c.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("csv export: status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Fatalf("csv export content-type = %q", ct)
	}
	raw, _ := io.ReadAll(resp.Body)
	return string(raw)
}

// TestSecurityHeaders: the admin surface carries a strict no-inline CSP and
// the hosted pages carry a nonce policy their inline <style> matches; the SPA
// still loads under that policy.
func TestSecurityHeaders(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")

	// Admin SPA: strict CSP, no unsafe-inline, standard hardening headers.
	resp, err := e.client.Get(e.url + "/admin")
	if err != nil {
		t.Fatal(err)
	}
	spa, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(spa), `id="root"`) {
		t.Fatalf("admin SPA did not load: status %d", resp.StatusCode)
	}
	csp := resp.Header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'self'") {
		t.Fatalf("admin CSP missing script-src 'self': %q", csp)
	}
	if strings.Contains(csp, "unsafe-inline") {
		t.Fatalf("admin CSP must not use unsafe-inline: %q", csp)
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" ||
		resp.Header.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("admin hardening headers missing: %+v", resp.Header)
	}
	if resp.Header.Get("Strict-Transport-Security") != "" {
		t.Fatal("HSTS must not be set on a plain-http instance")
	}

	// Hosted page: nonce policy, no scripts, and the inline <style> carries a
	// nonce that the CSP admits.
	p, _ := e.createProject(t, "Themed App")
	proj, err := e.store.GetProject(context.Background(), p.Id)
	if err != nil {
		t.Fatal(err)
	}
	presp, err := e.client.Get(e.url + "/p/" + proj.Slug + "/reset?token=x")
	if err != nil {
		t.Fatal(err)
	}
	pageBytes, _ := io.ReadAll(presp.Body)
	presp.Body.Close()
	page := string(pageBytes)
	hcsp := presp.Header.Get("Content-Security-Policy")
	if !strings.Contains(hcsp, "script-src 'none'") {
		t.Fatalf("hosted CSP should forbid scripts: %q", hcsp)
	}
	if strings.Contains(hcsp, "unsafe-inline") {
		t.Fatalf("hosted CSP must not use unsafe-inline: %q", hcsp)
	}
	nonce := cspNonce(hcsp)
	if nonce == "" {
		t.Fatalf("hosted CSP missing style nonce: %q", hcsp)
	}
	// The template stamps the same nonce onto the inline <style>. html/template
	// entity-encodes base64 specials (+, /) in the attribute value — the
	// browser decodes them back, so compare after unescaping.
	pageNonce := html.UnescapeString(styleNonce(page))
	if pageNonce != nonce {
		t.Fatalf("hosted <style> nonce %q does not match CSP nonce %q", pageNonce, nonce)
	}
}

// styleNonce returns the nonce attribute value of the page's inline <style>.
func styleNonce(page string) string {
	const marker = `<style nonce="`
	i := strings.Index(page, marker)
	if i < 0 {
		return ""
	}
	rest := page[i+len(marker):]
	j := strings.IndexByte(rest, '"')
	if j < 0 {
		return ""
	}
	return rest[:j]
}

// cspNonce extracts the base64 nonce from a "style-src 'nonce-XXX'" directive.
func cspNonce(csp string) string {
	const marker = "'nonce-"
	i := strings.Index(csp, marker)
	if i < 0 {
		return ""
	}
	rest := csp[i+len(marker):]
	j := strings.IndexByte(rest, '\'')
	if j < 0 {
		return ""
	}
	return rest[:j]
}

// TestSecurityHeadersHSTS: on an https-configured instance HSTS is asserted,
// but only for requests that actually arrived over TLS.
func TestSecurityHeadersHSTS(t *testing.T) {
	e := newTestEnv(t, "tok", func(o *Options) {
		o.Config.BaseURL = "https://auth.example.com"
	})
	e.setup(t, "tok")

	req, _ := http.NewRequest(http.MethodGet, e.url+"/admin", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if hsts := resp.Header.Get("Strict-Transport-Security"); !strings.Contains(hsts, "max-age=") {
		t.Fatalf("https instance behind a TLS proxy should assert HSTS, got %q", hsts)
	}
}

// TestSMTPPasswordEncryptedAtRest: the relay password is stored as ciphertext
// under the master key, never as plaintext in the settings JSON, and it
// round-trips back through the settings read path that feeds the mailer.
func TestSMTPPasswordEncryptedAtRest(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	ctx := context.Background()

	const secret = "sup3r-s3cret-relay-pw"
	if _, err := e.adminSettings().UpdateSmtpSettings(ctx,
		connect.NewRequest(&adminv1.UpdateSmtpSettingsRequest{
			Smtp: &adminv1.SmtpSettings{
				Host: "smtp.example.com", Port: 587, Username: "mailer",
				Password: secret, From: "noreply@example.com",
			}})); err != nil {
		t.Fatal(err)
	}

	// The settings JSON row must not carry the plaintext password.
	raw, err := e.store.GetInstanceSetting(ctx, store.InstanceSettingSMTP)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, secret) {
		t.Fatalf("plaintext SMTP password leaked into settings JSON: %s", raw)
	}

	// The encrypted blob exists, is not the plaintext, and decrypts back.
	enc, err := e.store.GetInstanceSecret(ctx, store.InstanceSecretSMTPPassword)
	if err != nil {
		t.Fatalf("smtp password secret not stored: %v", err)
	}
	if strings.Contains(string(enc), secret) {
		t.Fatal("stored SMTP secret is not encrypted")
	}
	plain, err := e.master.Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != secret {
		t.Fatalf("decrypted SMTP password = %q, want %q", plain, secret)
	}

	// The admin API reports the password is set without echoing it.
	got, err := e.adminSettings().GetInstanceSettings(ctx,
		connect.NewRequest(&adminv1.GetInstanceSettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Msg.SmtpHasPassword || got.Msg.Smtp.Password != "" {
		t.Fatalf("settings should report has-password with no echo: %+v", got.Msg)
	}

	// A later edit leaving the password blank keeps the stored one (the
	// encrypted secret is unchanged and still decrypts).
	if _, err := e.adminSettings().UpdateSmtpSettings(ctx,
		connect.NewRequest(&adminv1.UpdateSmtpSettingsRequest{
			Smtp: &adminv1.SmtpSettings{
				Host: "smtp.example.com", Port: 2525, Username: "mailer",
				From: "noreply@example.com",
			}})); err != nil {
		t.Fatal(err)
	}
	enc2, err := e.store.GetInstanceSecret(ctx, store.InstanceSecretSMTPPassword)
	if err != nil {
		t.Fatal(err)
	}
	plain2, err := e.master.Decrypt(enc2)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain2) != secret {
		t.Fatalf("blank-password edit dropped the stored password: got %q", plain2)
	}
}
