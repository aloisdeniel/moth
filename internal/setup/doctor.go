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
	"github.com/aloisdeniel/moth/internal/oidc"
	"github.com/aloisdeniel/moth/internal/store"
)

// Doctor runs the `moth doctor` health checks against one instance and,
// when Slug is set, one project. It never mutates anything (the optional
// SMTP test send excepted, which the operator requests explicitly).
type Doctor struct {
	// BaseURL is the instance URL the checks reach it at (the context URL).
	BaseURL  string
	HTTPC    oidc.Doer
	Session  adminv1connect.SessionServiceClient
	Settings adminv1connect.InstanceSettingsServiceClient
	Projects adminv1connect.ProjectServiceClient

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
