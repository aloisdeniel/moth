package setup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/oidc"
)

// Shape validation for values pasted back from the Google console. Catching
// a mispaste here beats a "login stopped working" report later.
var (
	googleClientIDRE   = regexp.MustCompile(`^[0-9]+-[a-z0-9]+\.apps\.googleusercontent\.com$`)
	gcpProjectIDRE     = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)
	androidPackageRE   = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)+$`)
	appleBundleIDRE    = regexp.MustCompile(`^[a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)+$`)
	googleAuthEndpoint = "https://accounts.google.com/o/oauth2/v2/auth"
)

// ValidateGoogleClientID checks the pasted value looks like an OAuth
// client ID and returns it trimmed.
func ValidateGoogleClientID(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !googleClientIDRE.MatchString(s) {
		return "", fmt.Errorf("%q does not look like a Google OAuth client ID (…apps.googleusercontent.com)", s)
	}
	return s, nil
}

func validateGCPProjectID(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !gcpProjectIDRE.MatchString(s) {
		return "", fmt.Errorf("%q does not look like a GCP project ID (6-30 lowercase letters, digits, dashes)", s)
	}
	return s, nil
}

func validateAndroidPackage(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s != "" && !androidPackageRE.MatchString(s) {
		return "", fmt.Errorf("%q does not look like an Android application ID (e.g. com.example.app)", s)
	}
	return s, nil
}

func validateBundleID(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s != "" && !appleBundleIDRE.MatchString(s) {
		return "", fmt.Errorf("%q does not look like a bundle ID (e.g. com.example.app)", s)
	}
	return s, nil
}

// GoogleSetup drives `moth setup google` for one project. Every external
// dependency is a field so tests inject doubles.
type GoogleSetup struct {
	Projects adminv1connect.ProjectServiceClient
	Prompt   *Prompter
	Out      io.Writer
	Runner   Runner    // gcloud + keytool; ExecRunner in the CLI
	HTTPC    oidc.Doer // verification probes
	// AuthURL is Google's OAuth authorization endpoint (test override).
	AuthURL string
	// BaseURL is the moth instance base URL (redirect URI construction).
	BaseURL string

	// Inputs; empty ones are prompted for.
	Slug           string
	GCPProject     string
	IOSBundleID    string
	AndroidPackage string
	AndroidSHA1    string
	AndroidSHA256  string
	Keystore       string // compute fingerprints from this keystore
	KeystorePass   string
	// Pre-supplied client IDs skip the guided console visit.
	WebClientID     string
	IOSClientID     string
	AndroidClientID string
	WebClientSecret string
}

func (s *GoogleSetup) defaults() {
	if s.AuthURL == "" {
		s.AuthURL = googleAuthEndpoint
	}
	if s.Runner == nil {
		s.Runner = ExecRunner{}
	}
	if s.HTTPC == nil {
		s.HTTPC = &http.Client{}
	}
	if s.Out == nil {
		s.Out = io.Discard
	}
}

func (s *GoogleSetup) redirectURI() string {
	return strings.TrimSuffix(s.BaseURL, "/") + "/oauth/google/callback"
}

// Run executes the flow and returns the verification checklist. A non-nil
// error aborts before verification (bad input, RPC failure); console-side
// problems surface as FAIL checks instead.
func (s *GoogleSetup) Run(ctx context.Context) (*Report, error) {
	s.defaults()
	rep := &Report{}

	project, err := findProjectBySlug(ctx, s.Projects, s.Slug)
	if err != nil {
		return nil, err
	}
	settings := project.Settings
	if settings == nil {
		settings = &adminv1.ProjectSettings{}
	}
	current := settings.Google
	if current == nil {
		current = &adminv1.GoogleProviderConfig{}
	}

	// GCP project — required; gcloud (when present) confirms it exists
	// before the user is sent clicking through its console.
	if s.GCPProject == "" {
		if s.GCPProject, err = s.Prompt.Ask("GCP project ID", validateGCPProjectID); err != nil {
			return nil, err
		}
	} else if s.GCPProject, err = validateGCPProjectID(s.GCPProject); err != nil {
		return nil, err
	}
	s.checkGCPProject(ctx, rep)

	// Platform inputs. Blank answers skip a platform honestly instead of
	// half-configuring it.
	if s.IOSBundleID == "" && s.IOSClientID == "" && current.IosClientId == "" {
		if s.IOSBundleID, err = s.Prompt.Ask("iOS bundle ID (blank to skip iOS)", validateBundleID); err != nil {
			return nil, err
		}
	}
	if s.AndroidPackage == "" && s.AndroidClientID == "" && current.AndroidClientId == "" {
		if s.AndroidPackage, err = s.Prompt.Ask("Android package name (blank to skip Android)", validateAndroidPackage); err != nil {
			return nil, err
		}
	}
	if s.AndroidPackage != "" {
		if err := s.resolveAndroidFingerprints(ctx); err != nil {
			return nil, err
		}
	}

	// Client IDs: keep configured values (idempotent), accept flags, and
	// only then fall back to the guided console visit.
	web, err := s.resolveClientID("web", s.WebClientID, current.WebClientId, s.webInstructions)
	if err != nil {
		return nil, err
	}
	ios, err := s.resolveClientID("iOS", s.IOSClientID, current.IosClientId, s.iosInstructions)
	if err != nil {
		return nil, err
	}
	android, err := s.resolveClientID("Android", s.AndroidClientID, current.AndroidClientId, s.androidInstructions)
	if err != nil {
		return nil, err
	}
	if web == "" && ios == "" && android == "" {
		return nil, errors.New("nothing to configure: no client ID was provided for any platform")
	}
	if web != "" && s.WebClientSecret == "" && !current.HasWebClientSecret {
		s.Prompt.Say("The web client's secret is needed for the web-redirect code exchange")
		s.Prompt.Say("(shown next to the client ID in the console; moth stores it encrypted).")
		// AskSecret: no echo on a terminal — the secret must not end up in
		// scrollback or session recordings.
		if s.WebClientSecret, err = s.Prompt.AskSecret("Web client secret (blank to skip)"); err != nil {
			return nil, err
		}
	}

	// Diff against moth's stored config; update only when something moved.
	changed := !current.Enabled ||
		current.WebClientId != web ||
		current.IosClientId != ios ||
		current.AndroidClientId != android ||
		s.WebClientSecret != ""
	if changed {
		desired := settings // reads always populate every settings field
		desired.Google = &adminv1.GoogleProviderConfig{
			Enabled:         true,
			WebClientId:     web,
			IosClientId:     ios,
			AndroidClientId: android,
			WebClientSecret: s.WebClientSecret, // "" keeps the stored secret
		}
		_, err := s.Projects.UpdateProject(ctx, connect.NewRequest(&adminv1.UpdateProjectRequest{
			Id:         project.Id,
			Settings:   desired,
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"settings"}},
		}))
		if err != nil {
			return nil, fmt.Errorf("update project %q: %w", s.Slug, err)
		}
		rep.Pass("moth: Google provider config updated", "project "+s.Slug)
	} else {
		rep.Pass("moth: Google provider config", "already up to date — no changes")
	}

	// Verification: unauthenticated probes against Google's authorization
	// endpoint (see the capability spike in the package doc).
	s.verifyClientID(ctx, rep, "web", web, s.redirectURI(), true)
	s.verifyClientID(ctx, rep, "iOS", ios, reversedClientScheme(ios)+":/oauth2redirect", false)
	s.verifyClientID(ctx, rep, "Android", android, reversedClientScheme(android)+":/oauth2redirect", false)
	if web != "" && s.WebClientSecret == "" && !current.HasWebClientSecret {
		rep.Warn("Google: web client secret", "no secret stored",
			"the web-redirect fallback flow will fail; re-run with --web-client-secret")
	}
	rep.Warn("Google: OAuth consent screen", "cannot be verified without console access",
		"confirm the consent screen is configured and published for "+s.GCPProject)
	return rep, nil
}

func (s *GoogleSetup) checkGCPProject(ctx context.Context, rep *Report) {
	if _, err := s.Runner.LookPath("gcloud"); err != nil {
		rep.Skip("gcloud: GCP project exists", "gcloud not found — guided mode, nothing verified Google-side")
		return
	}
	_, err := s.Runner.Output(ctx, nil, "gcloud", "projects", "describe", s.GCPProject, "--format=value(projectId)")
	if err != nil {
		rep.Fail("gcloud: GCP project exists", fmt.Sprintf("cannot describe %q: %v", s.GCPProject, err),
			"check the project ID and `gcloud auth login` / `gcloud auth application-default login`")
		return
	}
	rep.Pass("gcloud: GCP project exists", s.GCPProject)
}

// resolveAndroidFingerprints fills AndroidSHA1/AndroidSHA256 from the
// keystore (via keytool) or by prompting for pasted values.
func (s *GoogleSetup) resolveAndroidFingerprints(ctx context.Context) error {
	var err error
	if s.AndroidSHA1 != "" {
		if s.AndroidSHA1, err = NormalizeFingerprint(s.AndroidSHA1, 20); err != nil {
			return fmt.Errorf("--android-sha1: %w", err)
		}
	}
	if s.AndroidSHA256 != "" {
		if s.AndroidSHA256, err = NormalizeFingerprint(s.AndroidSHA256, 32); err != nil {
			return fmt.Errorf("--android-sha256: %w", err)
		}
	}
	if s.AndroidSHA1 != "" && s.AndroidSHA256 != "" {
		return nil
	}
	if s.Keystore != "" {
		if s.KeystorePass == "" {
			// keytool cannot prompt itself (the runner pipes its stdin and
			// stdout), so ask here — without echo. Blank works for JKS
			// keystores, which list fingerprints password-less.
			if s.KeystorePass, err = s.Prompt.AskSecret("Keystore password (blank for none)"); err != nil {
				return err
			}
		}
		sha1, sha256, err := KeystoreFingerprints(ctx, s.Runner, s.Keystore, s.KeystorePass)
		if err != nil {
			return fmt.Errorf("keystore %s: %w", s.Keystore, err)
		}
		s.AndroidSHA1, s.AndroidSHA256 = sha1, sha256
		s.Prompt.Say("Computed signing fingerprints from %s:", s.Keystore)
		s.Prompt.Say("  SHA-1:   %s", s.AndroidSHA1)
		s.Prompt.Say("  SHA-256: %s", s.AndroidSHA256)
		return nil
	}
	s.Prompt.Say("Android needs the signing certificate fingerprints")
	s.Prompt.Say("(run: keytool -list -v -keystore <your keystore>).")
	if s.AndroidSHA1 == "" {
		if s.AndroidSHA1, err = s.Prompt.Ask("SHA-1 fingerprint", func(v string) (string, error) {
			return NormalizeFingerprint(v, 20)
		}); err != nil {
			return err
		}
	}
	if s.AndroidSHA256 == "" {
		if s.AndroidSHA256, err = s.Prompt.Ask("SHA-256 fingerprint", func(v string) (string, error) {
			return NormalizeFingerprint(v, 32)
		}); err != nil {
			return err
		}
	}
	return nil
}

// resolveClientID returns the client ID for one platform: the flag value,
// else the already-configured one, else — when the platform is wanted —
// the guided console flow. Empty means the platform is skipped.
func (s *GoogleSetup) resolveClientID(platform, flag, existing string, instructions func()) (string, error) {
	if flag != "" {
		return ValidateGoogleClientID(flag)
	}
	if existing != "" {
		return existing, nil
	}
	switch platform {
	case "iOS":
		if s.IOSBundleID == "" {
			return "", nil
		}
	case "Android":
		if s.AndroidPackage == "" {
			return "", nil
		}
	}
	instructions()
	return s.Prompt.Ask(fmt.Sprintf("Paste the %s client ID (blank to skip)", platform), func(v string) (string, error) {
		if strings.TrimSpace(v) == "" {
			return "", nil
		}
		return ValidateGoogleClientID(v)
	})
}

func (s *GoogleSetup) consoleCreateURL() string {
	return "https://console.cloud.google.com/auth/clients/create?project=" + url.QueryEscape(s.GCPProject)
}

func (s *GoogleSetup) webInstructions() {
	s.Prompt.Say("")
	s.Prompt.Say("Create the WEB OAuth client (no API exists for this — one console visit):")
	s.Prompt.Say("  Open  %s", s.consoleCreateURL())
	s.Prompt.Say("  1. Application type: \"Web application\"; name it e.g. \"moth %s web\".", s.Slug)
	s.Prompt.Say("  2. Under \"Authorized redirect URIs\" add exactly:")
	s.Prompt.Say("       %s", s.redirectURI())
	s.Prompt.Say("  3. Create, then copy the client ID (…apps.googleusercontent.com).")
}

func (s *GoogleSetup) iosInstructions() {
	s.Prompt.Say("")
	s.Prompt.Say("Create the iOS OAuth client:")
	s.Prompt.Say("  Open  %s", s.consoleCreateURL())
	s.Prompt.Say("  1. Application type: \"iOS\".")
	s.Prompt.Say("  2. Bundle ID: %s", s.IOSBundleID)
	s.Prompt.Say("  3. Create, then copy the client ID.")
}

func (s *GoogleSetup) androidInstructions() {
	s.Prompt.Say("")
	s.Prompt.Say("Create the ANDROID OAuth client:")
	s.Prompt.Say("  Open  %s", s.consoleCreateURL())
	s.Prompt.Say("  1. Application type: \"Android\".")
	s.Prompt.Say("  2. Package name: %s", s.AndroidPackage)
	s.Prompt.Say("  3. SHA-1 certificate fingerprint: %s", s.AndroidSHA1)
	s.Prompt.Say("     (SHA-256, if asked: %s)", s.AndroidSHA256)
	s.Prompt.Say("  4. Create, then copy the client ID.")
}

// probeResult classifies Google's answer to an authorization-request probe.
type probeResult int

const (
	probeOK probeResult = iota
	probeClientNotFound
	probeRedirectMismatch
	probeIndeterminate
)

// probeGoogleClient asks Google's authorization endpoint about a client ID
// without any credentials: 2xx/3xx means client and redirect URI check
// out; error pages distinguish an unknown client from an unregistered
// redirect URI.
func probeGoogleClient(ctx context.Context, httpc oidc.Doer, authURL, clientID, redirectURI string) (probeResult, error) {
	q := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {"openid"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authURL+"?"+q.Encode(), nil)
	if err != nil {
		return probeIndeterminate, err
	}
	resp, err := httpc.Do(req)
	if err != nil {
		return probeIndeterminate, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return probeIndeterminate, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return probeOK, nil
	}
	page := string(body)
	switch {
	case strings.Contains(page, "invalid_client") || strings.Contains(page, "The OAuth client was not found"):
		return probeClientNotFound, nil
	case strings.Contains(page, "redirect_uri_mismatch"):
		return probeRedirectMismatch, nil
	}
	return probeIndeterminate, fmt.Errorf("unexpected status %d", resp.StatusCode)
}

func (s *GoogleSetup) verifyClientID(ctx context.Context, rep *Report, platform, clientID, redirectURI string, redirectMatters bool) {
	name := "Google: " + platform + " client ID resolves"
	if clientID == "" {
		rep.Skip(name, "platform not configured")
		return
	}
	result, err := probeGoogleClient(ctx, s.HTTPC, s.AuthURL, clientID, redirectURI)
	switch result {
	case probeOK:
		detail := clientID
		if redirectMatters {
			detail += " (redirect URI registered)"
		}
		rep.Pass(name, detail)
	case probeClientNotFound:
		rep.Fail(name, "Google does not know "+clientID,
			"the client may have been deleted or mispasted; recreate it at "+s.consoleCreateURL())
	case probeRedirectMismatch:
		if redirectMatters {
			rep.Fail("Google: web redirect URI registered", "client exists but "+s.redirectURI()+" is not registered",
				"add it under the client's \"Authorized redirect URIs\" in the console")
		} else {
			// The client resolved; the probe redirect is moot for
			// installed-app clients.
			rep.Pass(name, clientID)
		}
	default:
		rep.Warn(name, fmt.Sprintf("probe inconclusive: %v", err), "re-run `moth doctor` later")
	}
}

// reversedClientScheme is the custom URL scheme Google registers for
// installed-app clients: the client ID with its dotted host reversed
// ("123-abc.apps.googleusercontent.com" → "com.googleusercontent.apps.123-abc").
func reversedClientScheme(clientID string) string {
	if clientID == "" {
		return ""
	}
	parts := strings.Split(clientID, ".")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ".")
}
