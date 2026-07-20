package setup

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/oidc"
)

// Shape validation for Apple identifiers.
var (
	appleTeamIDRE = regexp.MustCompile(`^[A-Z0-9]{10}$`)
	appleKeyIDRE  = regexp.MustCompile(`^[A-Z0-9]{10}$`)
	ascIssuerRE   = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// ValidateAppleTeamID checks a 10-character Apple Developer Team ID.
func ValidateAppleTeamID(s string) (string, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if !appleTeamIDRE.MatchString(s) {
		return "", fmt.Errorf("%q does not look like an Apple Team ID (10 characters, e.g. AB12CD34EF)", s)
	}
	return s, nil
}

// ValidateAppleKeyID checks a 10-character Apple key ID.
func ValidateAppleKeyID(s string) (string, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if !appleKeyIDRE.MatchString(s) {
		return "", fmt.Errorf("%q does not look like an Apple key ID (10 characters)", s)
	}
	return s, nil
}

// ValidateASCIssuerID checks an App Store Connect API issuer ID (a UUID).
func ValidateASCIssuerID(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if !ascIssuerRE.MatchString(s) {
		return "", fmt.Errorf("%q does not look like an App Store Connect issuer ID (a UUID)", s)
	}
	return s, nil
}

func validateServicesID(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !appleBundleIDRE.MatchString(s) {
		return "", fmt.Errorf("%q does not look like a Services ID (reverse-DNS, e.g. com.example.app.signin)", s)
	}
	return s, nil
}

// AppleSetup drives `moth setup apple` for one project.
type AppleSetup struct {
	Projects adminv1connect.ProjectServiceClient
	Prompt   *Prompter
	Out      io.Writer
	// ASC is the App Store Connect client, authenticated with the
	// operator's API key (used in-process only, never stored).
	ASC   *ASC
	HTTPC oidc.Doer
	// AppleTokenBase is Apple's OAuth base URL (test override; the dry-run
	// verification posts to {base}/auth/token).
	AppleTokenBase string
	// BaseURL is the moth instance base URL (return-URL construction).
	BaseURL string

	// Inputs; empty ones are prompted for.
	Slug       string
	BundleID   string
	TeamID     string
	ServicesID string
	// RotateKey forces creating a fresh Sign in with Apple key even when
	// the project already stores one.
	RotateKey bool
	// UseUnofficialAPI is a documented stub: the spike evaluated driving
	// the developer portal's unofficial API (fastlane/spaceship precedent)
	// for Services ID + return-URL registration and deliberately did not
	// ship it — it is unversioned, ToS-gray and breaks silently. The flag
	// exists so scripts written against a future implementation fail
	// loudly today instead of half-running.
	UseUnofficialAPI bool
}

// ErrUnofficialAPINotImplemented is returned for --unofficial-api.
var ErrUnofficialAPINotImplemented = errors.New(
	"--unofficial-api is not implemented: Services ID registration has no official API and the unofficial portal API was evaluated and deliberately not shipped; use the guided flow")

func (s *AppleSetup) returnURL() string {
	return strings.TrimSuffix(s.BaseURL, "/") + "/oauth/apple/callback"
}

// Run executes the flow and returns the verification checklist.
func (s *AppleSetup) Run(ctx context.Context) (*Report, error) {
	if s.UseUnofficialAPI {
		return nil, ErrUnofficialAPINotImplemented
	}
	if s.Out == nil {
		s.Out = io.Discard
	}
	if s.HTTPC == nil {
		s.HTTPC = &http.Client{}
	}
	if s.AppleTokenBase == "" {
		s.AppleTokenBase = oidc.AppleBaseURL
	}
	rep := &Report{}

	project, err := findProjectBySlug(ctx, s.Projects, s.Slug)
	if err != nil {
		return nil, err
	}
	settings := project.Settings
	if settings == nil {
		settings = &adminv1.ProjectSettings{}
	}
	current := settings.Apple
	if current == nil {
		current = &adminv1.AppleProviderConfig{}
	}

	if s.BundleID == "" {
		if s.BundleID, err = s.Prompt.Ask("App bundle ID", func(v string) (string, error) {
			v, err := validateBundleID(v)
			if err == nil && v == "" {
				return "", errors.New("bundle ID is required")
			}
			return v, err
		}); err != nil {
			return nil, err
		}
	} else if s.BundleID, err = validateBundleID(s.BundleID); err != nil {
		return nil, err
	}
	switch {
	case s.TeamID != "":
		if s.TeamID, err = ValidateAppleTeamID(s.TeamID); err != nil {
			return nil, err
		}
	case current.TeamId != "":
		s.TeamID = current.TeamId
	default:
		if s.TeamID, err = s.Prompt.Ask("Apple Team ID", ValidateAppleTeamID); err != nil {
			return nil, err
		}
	}

	// Bundle ID + capability: fully automated, official ASC API.
	bundle, err := s.ensureBundleID(ctx, rep, project.Name)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCapability(ctx, rep, bundle); err != nil {
		return nil, err
	}

	// Sign in with Apple key. Apple serves the .p8 exactly once; when a
	// key is created here it is uploaded into moth's encrypted provider
	// config in the very next RPC.
	keyID, keyP8, siwaKey, err := s.ensureKey(ctx, rep, current, bundle)
	if err != nil {
		return nil, err
	}

	// Services ID: guided only (no official API; see the capability
	// spike). An already-configured value is kept untouched.
	if err := s.resolveServicesID(current); err != nil {
		return nil, err
	}

	// Diff against moth's stored config; update only when something moved.
	bundleIDs := current.BundleIds
	if !slices.Contains(bundleIDs, s.BundleID) {
		bundleIDs = append(slices.Clone(bundleIDs), s.BundleID)
	}
	changed := !current.Enabled ||
		current.TeamId != s.TeamID ||
		current.KeyId != keyID ||
		current.ServicesId != s.ServicesID ||
		!slices.Equal(current.BundleIds, bundleIDs) ||
		len(keyP8) > 0
	if changed {
		desired := settings
		desired.Apple = &adminv1.AppleProviderConfig{
			Enabled:      true,
			ServicesId:   s.ServicesID,
			TeamId:       s.TeamID,
			KeyId:        keyID,
			PrivateKeyP8: string(keyP8), // "" keeps the stored key
			BundleIds:    bundleIDs,
		}
		_, err := s.Projects.UpdateProject(ctx, connect.NewRequest(&adminv1.UpdateProjectRequest{
			Id:         project.Id,
			Settings:   desired,
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"settings"}},
		}))
		if err != nil {
			return nil, fmt.Errorf("update project %q: %w", s.Slug, err)
		}
		detail := "project " + s.Slug
		if len(keyP8) > 0 {
			detail += " (private key stored encrypted)"
		}
		rep.Pass("moth: Apple provider config updated", detail)
	} else {
		rep.Pass("moth: Apple provider config", "already up to date — no changes")
	}

	// Verification.
	s.verifyDryRun(ctx, rep, keyID, siwaKey)
	rep.Warn("Apple: Services ID return URL", "cannot be verified without portal access",
		fmt.Sprintf("confirm %s has Sign in with Apple enabled with return URL %s", s.ServicesID, s.returnURL()))
	return rep, nil
}

func (s *AppleSetup) ensureBundleID(ctx context.Context, rep *Report, projectName string) (*ASCBundleID, error) {
	bundle, err := s.ASC.FindBundleID(ctx, s.BundleID)
	if err != nil {
		return nil, fmt.Errorf("look up bundle ID %s: %w", s.BundleID, err)
	}
	if bundle != nil {
		rep.Pass("Apple: bundle ID registered", s.BundleID)
		return bundle, nil
	}
	name := projectName
	if name == "" {
		name = s.Slug
	}
	bundle, err = s.ASC.CreateBundleID(ctx, s.BundleID, name)
	if err != nil {
		return nil, fmt.Errorf("create bundle ID %s: %w", s.BundleID, err)
	}
	rep.Pass("Apple: bundle ID registered", s.BundleID+" (created)")
	return bundle, nil
}

func (s *AppleSetup) ensureCapability(ctx context.Context, rep *Report, bundle *ASCBundleID) error {
	has, err := s.ASC.HasSignInWithApple(ctx, bundle.ResourceID)
	if err != nil {
		return fmt.Errorf("list capabilities of %s: %w", s.BundleID, err)
	}
	if has {
		rep.Pass("Apple: Sign in with Apple capability", "already enabled on "+s.BundleID)
		return nil
	}
	if err := s.ASC.EnableSignInWithApple(ctx, bundle.ResourceID); err != nil {
		return fmt.Errorf("enable Sign in with Apple on %s: %w", s.BundleID, err)
	}
	rep.Pass("Apple: Sign in with Apple capability", "enabled on "+s.BundleID)
	return nil
}

// ensureKey returns the Sign in with Apple key to configure: the stored
// one (idempotent re-run), a freshly created one (with its .p8 to upload),
// or one pasted through the guided fallback. siwaKey is the parsed private
// key when the .p8 is in hand this run — that is what makes the dry-run
// verification possible.
func (s *AppleSetup) ensureKey(ctx context.Context, rep *Report, current *adminv1.AppleProviderConfig, bundle *ASCBundleID) (keyID string, p8 []byte, siwaKey *ecdsa.PrivateKey, err error) {
	if current.KeyId != "" && current.HasPrivateKey && !s.RotateKey {
		rep.Pass("Apple: Sign in with Apple key", "key "+current.KeyId+" already stored (use --rotate-key to replace)")
		return current.KeyId, nil, nil, nil
	}
	name := "moth " + s.Slug + " sign in with apple"
	keyID, p8, err = s.ASC.CreateSignInWithAppleKey(ctx, name, bundle.ResourceID)
	switch {
	case err == nil:
		siwaKey, err = oidc.ParseP8(p8)
		if err != nil {
			return "", nil, nil, fmt.Errorf("app store connect returned an unusable key: %w", err)
		}
		if keyID, err = ValidateAppleKeyID(keyID); err != nil {
			return "", nil, nil, fmt.Errorf("app store connect returned an unusable key ID: %w", err)
		}
		rep.Pass("Apple: Sign in with Apple key", "created key "+keyID+" via the App Store Connect API")
		return keyID, p8, siwaKey, nil
	case isASCNotFound(err):
		// Endpoint not available (capability spike): guided fallback.
		s.Prompt.Say("")
		s.Prompt.Say("Key creation is not available through this App Store Connect API surface;")
		s.Prompt.Say("create the key manually (Apple lets you download the .p8 exactly once):")
		s.Prompt.Say("  Open  https://developer.apple.com/account/resources/authkeys/add")
		s.Prompt.Say("  1. Name it e.g. %q.", name)
		s.Prompt.Say("  2. Check \"Sign in with Apple\", configure it with primary App ID %s.", s.BundleID)
		s.Prompt.Say("  3. Register, download the .p8 and note the Key ID.")
		keyID, err = s.Prompt.Ask("Key ID", ValidateAppleKeyID)
		if err != nil {
			return "", nil, nil, err
		}
		path, err := s.Prompt.Ask("Path to the downloaded .p8", func(v string) (string, error) {
			if strings.TrimSpace(v) == "" {
				return "", errors.New("a path is required")
			}
			return strings.TrimSpace(v), nil
		})
		if err != nil {
			return "", nil, nil, err
		}
		p8, err = os.ReadFile(path)
		if err != nil {
			return "", nil, nil, err
		}
		if siwaKey, err = oidc.ParseP8(p8); err != nil {
			return "", nil, nil, fmt.Errorf("%s: %w", path, err)
		}
		rep.Pass("Apple: Sign in with Apple key", "key "+keyID+" registered (guided)")
		return keyID, p8, siwaKey, nil
	default:
		return "", nil, nil, fmt.Errorf("create Sign in with Apple key: %w", err)
	}
}

func (s *AppleSetup) resolveServicesID(current *adminv1.AppleProviderConfig) error {
	var err error
	if s.ServicesID != "" {
		s.ServicesID, err = validateServicesID(s.ServicesID)
		return err
	}
	if current.ServicesId != "" {
		s.ServicesID = current.ServicesId
		return nil
	}
	host := s.BaseURL
	if u, err := url.Parse(s.BaseURL); err == nil && u.Host != "" {
		host = u.Host
	}
	s.Prompt.Say("")
	s.Prompt.Say("Services IDs have no official API — one portal visit (guided):")
	s.Prompt.Say("  Open  https://developer.apple.com/account/resources/identifiers/list/serviceId")
	s.Prompt.Say("  1. Register a new Services ID, e.g. %q.", s.BundleID+".signin")
	s.Prompt.Say("  2. Enable \"Sign in with Apple\" and click Configure:")
	s.Prompt.Say("       Primary App ID: %s", s.BundleID)
	s.Prompt.Say("       Domain:         %s", host)
	s.Prompt.Say("       Return URL:     %s", s.returnURL())
	s.Prompt.Say("  3. Save, then paste the Services ID below.")
	s.ServicesID, err = s.Prompt.Ask("Services ID", validateServicesID)
	return err
}

// verifyDryRun mints a client secret from the key and asks Apple's token
// endpoint to reject a garbage code: "invalid_grant" proves the
// key/team/client triple was accepted, "invalid_client" proves it was not.
// Only possible when the .p8 passed through this run.
func (s *AppleSetup) verifyDryRun(ctx context.Context, rep *Report, keyID string, siwaKey *ecdsa.PrivateKey) {
	const name = "Apple: client secret accepted (token endpoint dry-run)"
	if siwaKey == nil {
		rep.Warn(name, "the private key only passes through when it is (re)created, so this run cannot mint a client secret",
			"run `moth doctor --project "+s.Slug+" --apple-key <siwa .p8>` to verify the stored configuration")
		return
	}
	rep.add(appleTokenDryRun(ctx, name, s.HTTPC, s.AppleTokenBase, s.ServicesID, s.TeamID, keyID, siwaKey))
}

// appleTokenDryRun is shared with `moth doctor`.
func appleTokenDryRun(ctx context.Context, name string, httpc oidc.Doer, base, clientID, teamID, keyID string, key *ecdsa.PrivateKey) Check {
	secrets := oidc.NewAppleSecrets(oidc.AppleSecretConfig{
		TeamID:   teamID,
		KeyID:    keyID,
		ClientID: clientID,
		Key:      key,
	}, nil)
	client := oidc.NewAppleClient(base, clientID, secrets, httpc)
	_, err := client.ExchangeCode(ctx, "moth-setup-dry-run", "")
	var tokErr *oidc.TokenError
	switch {
	case err == nil:
		return Check{Name: name, Status: StatusWarn,
			Detail: "Apple accepted a garbage authorization code — unexpected, treat with suspicion"}
	case errors.As(err, &tokErr) && tokErr.Code == "invalid_grant":
		return Check{Name: name, Status: StatusPass,
			Detail: fmt.Sprintf("Apple accepted the secret for %s (key %s, team %s)", clientID, keyID, teamID)}
	case errors.As(err, &tokErr) && tokErr.Code == "invalid_client":
		return Check{Name: name, Status: StatusFail,
			Detail:      "Apple rejected the client secret (invalid_client)",
			Remediation: "the key may be revoked, or the Services ID / Team ID / Key ID do not belong together; re-run `moth setup apple`"}
	default:
		return Check{Name: name, Status: StatusWarn,
			Detail:      fmt.Sprintf("dry-run inconclusive: %v", err),
			Remediation: "re-run `moth doctor` later"}
	}
}
