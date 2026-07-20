package cli

import (
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/push"
	"github.com/aloisdeniel/moth/internal/setup"
)

// ErrInitAborted is returned when the operator abandons the wizard at the
// final confirmation. Nothing has been created at that point: the wizard
// holds every answer client-side and only writes after the confirm.
var ErrInitAborted = errors.New("aborted — nothing was created")

// InitAnswers is everything `moth project init` collected: the desired
// state (the same ProjectSpec `moth project apply` consumes, plus the push
// settings and profile the spec cannot carry) and the work the operator
// explicitly deferred.
type InitAnswers struct {
	// Spec carries name, slug (may be empty: derived server-side), settings
	// including provider credentials entered in-flow, and the monetization
	// catalog. It is the document the finishing `moth project apply` spec is
	// rendered from.
	Spec *adminv1.ProjectSpec
	// Profile records the answers themselves — platforms and feature intent
	// — for UpdateProfile; the derived checklist keys off it.
	Profile *adminv1.Profile
	// Push is the milestone-20 settings to install, nil when the backend
	// will not send pushes.
	Push *adminv1.PushSettings
	// Deferred lists what the wizard honestly did not finish, one human
	// line each ("Google sign-in credentials — run 'moth setup google'").
	Deferred []string
}

// hasPlatform reports whether the answers include the platform.
func (a *InitAnswers) hasPlatform(p adminv1.ProfilePlatform) bool {
	return slices.Contains(a.Profile.GetPlatforms(), p)
}

var (
	slugRE  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	identRE = regexp.MustCompile(`^[a-z0-9._-]{1,64}$`)
)

// validateProjectName requires a non-empty display name.
func validateProjectName(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("the project name cannot be empty")
	}
	return s, nil
}

// validateSlug accepts an empty slug (derived from the name server-side) or
// the server's slug shape: lowercase letters, digits and single dashes.
func validateSlug(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if !slugRE.MatchString(s) {
		return "", fmt.Errorf("invalid slug %q: lowercase letters, digits and single dashes only", s)
	}
	return s, nil
}

// parsePlatforms parses the comma-separated multi-select ("ios, web") into
// the profile enum values, normalized to ios/android/web order.
func parsePlatforms(s string) ([]adminv1.ProfilePlatform, error) {
	seen := map[adminv1.ProfilePlatform]bool{}
	for _, part := range strings.Split(s, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		var p adminv1.ProfilePlatform
		switch part {
		case "ios":
			p = adminv1.ProfilePlatform_PROFILE_PLATFORM_IOS
		case "android":
			p = adminv1.ProfilePlatform_PROFILE_PLATFORM_ANDROID
		case "web":
			p = adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB
		default:
			return nil, fmt.Errorf("unknown platform %q (choose from: ios, android, web)", part)
		}
		seen[p] = true
	}
	if len(seen) == 0 {
		return nil, errors.New("choose at least one platform (ios, android, web)")
	}
	var out []adminv1.ProfilePlatform
	for _, p := range []adminv1.ProfilePlatform{
		adminv1.ProfilePlatform_PROFILE_PLATFORM_IOS,
		adminv1.ProfilePlatform_PROFILE_PLATFORM_ANDROID,
		adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB,
	} {
		if seen[p] {
			out = append(out, p)
		}
	}
	return out, nil
}

// platformNames renders the enum values as the words the prompts use.
func platformNames(ps []adminv1.ProfilePlatform) []string {
	var out []string
	for _, p := range ps {
		switch p {
		case adminv1.ProfilePlatform_PROFILE_PLATFORM_IOS:
			out = append(out, "ios")
		case adminv1.ProfilePlatform_PROFILE_PLATFORM_ANDROID:
			out = append(out, "android")
		case adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB:
			out = append(out, "web")
		case adminv1.ProfilePlatform_PROFILE_PLATFORM_UNSPECIFIED:
		}
	}
	return out
}

// validateIdentifier mirrors the server's catalog identifier rule.
func validateIdentifier(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !identRE.MatchString(s) {
		return "", errors.New("identifier must be 1-64 chars of lowercase letters, digits, '.', '_' or '-'")
	}
	return s, nil
}

// nonEmpty trims and requires a value.
func nonEmpty(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("a value is required")
	}
	return s, nil
}

// askDefault asks with a visible default; empty input means the default,
// anything else must pass validate.
func askDefault(p *setup.Prompter, label, def string, validate func(string) (string, error)) (string, error) {
	return p.Ask(fmt.Sprintf("%s [%s]", label, def), func(s string) (string, error) {
		if strings.TrimSpace(s) == "" {
			return def, nil
		}
		if validate == nil {
			return strings.TrimSpace(s), nil
		}
		return validate(s)
	})
}

// askInt asks for a positive integer with a default.
func askInt(p *setup.Prompter, label string, def int32) (int32, error) {
	s, err := askDefault(p, label, strconv.Itoa(int(def)), func(s string) (string, error) {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || n <= 0 {
			return "", errors.New("enter a positive whole number")
		}
		return strconv.Itoa(n), nil
	})
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return int32(n), nil
}

// parsePriceMicros converts a decimal price ("9.99") to micros.
func parsePriceMicros(s string) (int64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v < 0 || v > 1e6 {
		return 0, errors.New(`enter the price as a decimal number, e.g. "9.99"`)
	}
	return int64(math.Round(v * 1e6)), nil
}

// RunInitWizard runs the interactive ask-configure-defer flow of
// `moth project init` over the prompter and returns the collected answers.
// It performs no RPC and writes nothing: creation is atomic after the final
// confirmation, so abandoning at any prompt (including answering "n" at the
// review step, which returns ErrInitAborted) leaves no project behind.
func RunInitWizard(p *setup.Prompter) (*InitAnswers, error) {
	a := &InitAnswers{
		Spec:    &adminv1.ProjectSpec{Settings: &adminv1.ProjectSettings{}},
		Profile: &adminv1.Profile{},
	}

	// 1. Basics — name, slug, platforms. Platforms drive every later branch.
	p.Say("moth project init — guided project creation. Every feature can be")
	p.Say("deferred; whatever is left lands on the setup checklist.")
	p.Say("")
	name, err := p.Ask("Project name", validateProjectName)
	if err != nil {
		return nil, err
	}
	a.Spec.Name = name
	slug, err := p.Ask("Slug (empty = derived from the name)", validateSlug)
	if err != nil {
		return nil, err
	}
	a.Spec.Slug = slug
	platformsRaw, err := p.Ask("Platforms (comma-separated: ios, android, web)", func(s string) (string, error) {
		if _, err := parsePlatforms(s); err != nil {
			return "", err
		}
		return s, nil
	})
	if err != nil {
		return nil, err
	}
	a.Profile.Platforms, _ = parsePlatforms(platformsRaw)
	web := a.hasPlatform(adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB)
	ios := a.hasPlatform(adminv1.ProfilePlatform_PROFILE_PLATFORM_IOS)
	android := a.hasPlatform(adminv1.ProfilePlatform_PROFILE_PLATFORM_ANDROID)

	// 2. Sign-in — email/password defaults, then the social providers.
	p.Say("")
	p.Say("Sign-in — email/password is always on.")
	if a.Spec.Settings.AllowPublicSignup, err = p.Confirm("Allow public sign-up", true); err != nil {
		return nil, err
	}
	if a.Spec.Settings.RequireEmailVerification, err = p.Confirm("Require email verification", true); err != nil {
		return nil, err
	}
	if a.Spec.Settings.PasswordMinLength, err = askInt(p, "Minimum password length", 8); err != nil {
		return nil, err
	}
	if a.Profile.GoogleSignIn, err = p.Confirm("Enable Google sign-in", false); err != nil {
		return nil, err
	}
	if a.Profile.GoogleSignIn {
		if err := askGoogle(p, a, web, ios, android); err != nil {
			return nil, err
		}
	}
	if a.Profile.AppleSignIn, err = p.Confirm("Enable Sign in with Apple", false); err != nil {
		return nil, err
	}
	if a.Profile.AppleSignIn {
		if err := askApple(p, a, web, ios, android); err != nil {
			return nil, err
		}
	}

	// 3. Monetization — no means the built-in `none` tier and nothing else.
	p.Say("")
	if a.Profile.SellsSubscriptions, err = p.Confirm("Does this app sell subscriptions?", false); err != nil {
		return nil, err
	}
	if a.Profile.SellsSubscriptions {
		if err := askMonetization(p, a, web, ios, android); err != nil {
			return nil, err
		}
	}

	// 4. Push notifications.
	p.Say("")
	if a.Profile.SendsPushes, err = p.Confirm("Will your backend send push notifications?", false); err != nil {
		return nil, err
	}
	if a.Profile.SendsPushes {
		if err := askPush(p, a, web, ios, android); err != nil {
			return nil, err
		}
	}

	// 5. Review & confirm — the only gate before anything is written.
	sayReview(p, a)
	ok, err := p.Confirm("Create the project now?", true)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInitAborted
	}
	return a, nil
}

// askGoogle collects the Google OAuth credentials in-flow or defers them to
// `moth setup google`. Only the fields the chosen platforms need are asked:
// web-only projects never see the native client IDs and native-only
// projects never see the web client fields.
func askGoogle(p *setup.Prompter, a *InitAnswers, web, ios, android bool) error {
	now, err := p.Confirm("Enter the Google OAuth client IDs now (otherwise defer to 'moth setup google')", false)
	if err != nil {
		return err
	}
	if !now {
		a.Deferred = append(a.Deferred, "Google sign-in credentials — run 'moth setup google'")
		return nil
	}
	g := &adminv1.GoogleProviderConfig{Enabled: true}
	// The web client is asked for Android too, not just web: Google's
	// Android sign-in issues ID tokens with the *web* client ID as audience,
	// so verification would reject every Android token without it (the SPA
	// wizard collects the same fields with the same caption).
	if web || android {
		if android && !web {
			p.Say("Google's Android sign-in issues ID tokens with the web client ID as audience, so the web client is needed even for Android-only apps.")
		}
		if g.WebClientId, err = p.Ask("Google web OAuth client ID", nonEmpty); err != nil {
			return err
		}
		if g.WebClientSecret, err = p.AskSecret("Google web OAuth client secret (empty = none yet)"); err != nil {
			return err
		}
	}
	if ios {
		if g.IosClientId, err = p.Ask("Google iOS OAuth client ID", nonEmpty); err != nil {
			return err
		}
	}
	if android {
		if g.AndroidClientId, err = p.Ask("Google Android OAuth client ID", nonEmpty); err != nil {
			return err
		}
	}
	a.Spec.Settings.Google = g
	return nil
}

// askApple collects the Sign in with Apple configuration in-flow or defers
// it to `moth setup apple`. iOS asks the bundle IDs; web (and the Android
// web-redirect flow) asks the Services ID; the .p8 key may itself be
// deferred (Apple serves it exactly once — it may simply not exist yet).
func askApple(p *setup.Prompter, a *InitAnswers, web, ios, android bool) error {
	now, err := p.Confirm("Enter the Sign in with Apple credentials now (otherwise defer to 'moth setup apple')", false)
	if err != nil {
		return err
	}
	if !now {
		a.Deferred = append(a.Deferred, "Sign in with Apple credentials — run 'moth setup apple'")
		return nil
	}
	ap := &adminv1.AppleProviderConfig{Enabled: true}
	if ios {
		raw, err := p.Ask("App bundle IDs (comma-separated)", nonEmpty)
		if err != nil {
			return err
		}
		for _, b := range strings.Split(raw, ",") {
			if b = strings.TrimSpace(b); b != "" {
				ap.BundleIds = append(ap.BundleIds, b)
			}
		}
	}
	if web || android {
		if ap.ServicesId, err = p.Ask("Apple Services ID (web-redirect flow, e.g. com.example.app.signin)", nonEmpty); err != nil {
			return err
		}
	}
	if ap.TeamId, err = p.Ask("Apple Developer Team ID", nonEmpty); err != nil {
		return err
	}
	if ap.KeyId, err = p.Ask("Sign in with Apple key ID", setup.ValidateAppleKeyID); err != nil {
		return err
	}
	// The validator reads the file so a typoed path re-asks instead of
	// failing the whole wizard.
	keyPEM, err := p.Ask("Path to the Sign in with Apple .p8 key (empty = defer the key)", func(s string) (string, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return "", nil
		}
		raw, err := os.ReadFile(s)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	})
	if err != nil {
		return err
	}
	ap.PrivateKeyP8 = keyPEM
	if keyPEM == "" {
		a.Deferred = append(a.Deferred, "Sign in with Apple private key (.p8) — run 'moth setup apple'")
	}
	a.Spec.Settings.Apple = ap
	return nil
}

// askMonetization defines the first entitlement and its tiers — the same
// catalog fields the apply spec carries. Store credentials and the catalog
// sync are explicitly deferred: collecting a .p8 mid-wizard helps nobody.
func askMonetization(p *setup.Prompter, a *InitAnswers, web, ios, android bool) error {
	var stores []string
	if ios {
		stores = append(stores, "App Store (ios)")
	}
	if android {
		stores = append(stores, "Google Play (android)")
	}
	if web {
		stores = append(stores, "Stripe (web)")
	}
	p.Say("Stores implied by your platforms: %s.", strings.Join(stores, ", "))
	entIdent, err := p.Ask("First entitlement identifier (e.g. pro)", validateIdentifier)
	if err != nil {
		return err
	}
	entName, err := askDefault(p, "Entitlement display name", entIdent, nil)
	if err != nil {
		return err
	}
	mon := &adminv1.MonetizationSpec{
		Entitlements: []*adminv1.EntitlementSpec{{Identifier: entIdent, DisplayName: entName}},
	}
	for {
		t := &adminv1.ProductSpec{Entitlements: []string{entIdent}}
		if t.Identifier, err = p.Ask("Tier identifier (e.g. monthly)", validateIdentifier); err != nil {
			return err
		}
		if t.DisplayName, err = askDefault(p, "Tier display name", t.Identifier, nil); err != nil {
			return err
		}
		if t.BillingPeriod, err = askDefault(p, "Billing period (e.g. monthly, yearly)", "monthly", func(s string) (string, error) {
			return strings.ToLower(strings.TrimSpace(s)), nil
		}); err != nil {
			return err
		}
		priceRaw, err := p.Ask(`Price per period (e.g. "9.99")`, func(s string) (string, error) {
			if _, err := parsePriceMicros(s); err != nil {
				return "", err
			}
			return strings.TrimSpace(s), nil
		})
		if err != nil {
			return err
		}
		t.PriceAmountMicros, _ = parsePriceMicros(priceRaw)
		if t.Currency, err = askDefault(p, "Currency", "USD", func(s string) (string, error) {
			s = strings.ToUpper(strings.TrimSpace(s))
			if len(s) != 3 {
				return "", errors.New("enter a 3-letter currency code, e.g. USD")
			}
			return s, nil
		}); err != nil {
			return err
		}
		if ios {
			if t.AppleProductId, err = askDefault(p, "App Store product ID", t.Identifier, nil); err != nil {
				return err
			}
		}
		if android {
			if t.GoogleProductId, err = askDefault(p, "Google Play product ID", t.Identifier, nil); err != nil {
				return err
			}
		}
		mon.Products = append(mon.Products, t)
		more, err := p.Confirm("Add another tier?", false)
		if err != nil {
			return err
		}
		if !more {
			break
		}
	}
	a.Spec.Monetization = mon
	a.Deferred = append(a.Deferred, "Store billing credentials and catalog sync — run 'moth setup billing'")
	return nil
}

// askPush enables the milestone-20 push settings: web asks for (or defers)
// the VAPID public key; Android and iOS only get their honest caveats.
func askPush(p *setup.Prompter, a *InitAnswers, web, ios, android bool) error {
	a.Push = &adminv1.PushSettings{Enabled: true}
	if web {
		// Validated at the prompt with the server's own shape rule
		// (internal/push), so a typo re-asks here instead of failing the
		// UpdatePushSettings write after the project exists.
		key, err := p.Ask("Web Push VAPID public key (empty = defer)", func(s string) (string, error) {
			s = strings.TrimSpace(s)
			if s == "" {
				return "", nil
			}
			if err := push.ValidateVAPIDPublicKey(s); err != nil {
				return "", err
			}
			return s, nil
		})
		if err != nil {
			return err
		}
		a.Push.WebpushVapidPublicKey = strings.TrimSpace(key)
		if a.Push.WebpushVapidPublicKey == "" {
			a.Deferred = append(a.Deferred,
				"Web Push VAPID public key — generate the pair with 'npx web-push generate-vapid-keys', then add the public key in Settings")
		}
	}
	if android {
		p.Say("Note: Android delivery goes through your own Firebase project — moth stores the registrations; your backend sends via FCM.")
	}
	if ios {
		p.Say("Note: iOS only needs the Push Notifications capability on your App ID; nothing to configure in moth.")
	}
	return nil
}

// sayReview prints the one-screen summary the final confirmation gates.
func sayReview(p *setup.Prompter, a *InitAnswers) {
	p.Say("")
	p.Say("Review:")
	p.Say("  name:       %s", a.Spec.Name)
	slug := a.Spec.Slug
	if slug == "" {
		slug = "(derived from the name)"
	}
	p.Say("  slug:       %s", slug)
	p.Say("  platforms:  %s", strings.Join(platformNames(a.Profile.Platforms), ", "))
	signIn := []string{fmt.Sprintf("email/password (public sign-up: %s, email verification: %s, min password length: %d)",
		yesNo(a.Spec.Settings.AllowPublicSignup), yesNo(a.Spec.Settings.RequireEmailVerification), a.Spec.Settings.PasswordMinLength)}
	if a.Profile.GoogleSignIn {
		signIn = append(signIn, "google")
	}
	if a.Profile.AppleSignIn {
		signIn = append(signIn, "apple")
	}
	p.Say("  sign-in:    %s", strings.Join(signIn, ", "))
	if mon := a.Spec.Monetization; mon != nil {
		var tiers []string
		for _, t := range mon.Products {
			tiers = append(tiers, t.Identifier)
		}
		p.Say("  tiers:      %s (entitlement %s)", strings.Join(tiers, ", "), mon.Entitlements[0].Identifier)
	} else {
		p.Say("  tiers:      none (free app)")
	}
	if a.Push != nil {
		detail := "enabled"
		if a.Push.WebpushVapidPublicKey != "" {
			detail += ", VAPID key set"
		}
		p.Say("  push:       %s", detail)
	} else {
		p.Say("  push:       no")
	}
	if len(a.Deferred) > 0 {
		p.Say("  deferred:")
		for _, d := range a.Deferred {
			p.Say("    - %s", d)
		}
	}
}

// yesNo renders a boolean for the review summary.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
