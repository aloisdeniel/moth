package cli

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/setup"
)

// script joins scripted prompt answers into the stdin the wizard reads.
func script(answers ...string) string {
	return strings.Join(answers, "\n") + "\n"
}

func runWizard(t *testing.T, stdin string) (*InitAnswers, string, error) {
	t.Helper()
	var out strings.Builder
	a, err := RunInitWizard(setup.NewPrompter(strings.NewReader(stdin), &out))
	return a, out.String(), err
}

// TestInitWizardMinimalWebOnly is the two-step acceptance flow: basics plus
// sign-in, every optional feature declined with its default answer.
func TestInitWizardMinimalWebOnly(t *testing.T) {
	a, out, err := runWizard(t, script(
		"Web App", // name
		"",        // slug: derived
		"web",     // platforms
		"",        // allow public sign-up: default yes
		"",        // require email verification: default yes
		"",        // min password length: default 8
		"",        // google: default no
		"",        // apple: default no
		"",        // subscriptions: default no
		"",        // pushes: default no
		"",        // create now: default yes
	))
	if err != nil {
		t.Fatalf("wizard: %v\n%s", err, out)
	}
	if a.Spec.Name != "Web App" || a.Spec.Slug != "" {
		t.Fatalf("basics = %q/%q", a.Spec.Name, a.Spec.Slug)
	}
	if got := a.Profile.Platforms; len(got) != 1 || got[0] != adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB {
		t.Fatalf("platforms = %v", got)
	}
	s := a.Spec.Settings
	if !s.AllowPublicSignup || !s.RequireEmailVerification || s.PasswordMinLength != 8 {
		t.Fatalf("settings = %+v", s)
	}
	if s.Google != nil || s.Apple != nil {
		t.Fatalf("no provider was chosen, got google=%v apple=%v", s.Google, s.Apple)
	}
	if a.Profile.GoogleSignIn || a.Profile.AppleSignIn || a.Profile.SellsSubscriptions || a.Profile.SendsPushes {
		t.Fatalf("profile flags should all be off: %+v", a.Profile)
	}
	if a.Spec.Monetization != nil || a.Push != nil || len(a.Deferred) != 0 {
		t.Fatalf("nothing optional was chosen: mon=%v push=%v deferred=%v",
			a.Spec.Monetization, a.Push, a.Deferred)
	}
	// The wizard never rendered store, push or native-provider prompts.
	for _, absent := range []string{"VAPID", "client ID", "bundle ID", "entitlement"} {
		if strings.Contains(out, absent) {
			t.Errorf("web-only free flow should not prompt for %q:\n%s", absent, out)
		}
	}
}

// TestInitWizardFullDeferredFlutter is the Flutter acceptance flow: iOS +
// Android, Apple sign-in, subscriptions and push — every credential
// honestly deferred.
func TestInitWizardFullDeferredFlutter(t *testing.T) {
	a, out, err := runWizard(t, script(
		"Flutter App",  // name
		"flutter-app",  // slug
		"ios, android", // platforms
		"",             // allow public sign-up
		"",             // require email verification
		"10",           // min password length
		"n",            // google
		"y",            // apple
		"n",            // enter apple credentials now? -> defer
		"y",            // sells subscriptions
		"pro",          // entitlement identifier
		"Pro",          // entitlement display name
		"monthly",      // tier identifier
		"Monthly",      // tier display name
		"",             // billing period: default monthly
		"9.99",         // price
		"",             // currency: default USD
		"pro.monthly",  // App Store product id
		"pro_monthly",  // Google Play product id
		"",             // add another tier: default no
		"y",            // sends pushes
		"",             // create now
	))
	if err != nil {
		t.Fatalf("wizard: %v\n%s", err, out)
	}
	if a.Spec.Slug != "flutter-app" || a.Spec.Settings.PasswordMinLength != 10 {
		t.Fatalf("basics = %+v", a.Spec)
	}
	wantPlatforms := []adminv1.ProfilePlatform{
		adminv1.ProfilePlatform_PROFILE_PLATFORM_IOS,
		adminv1.ProfilePlatform_PROFILE_PLATFORM_ANDROID,
	}
	if got := a.Profile.Platforms; len(got) != 2 || got[0] != wantPlatforms[0] || got[1] != wantPlatforms[1] {
		t.Fatalf("platforms = %v", got)
	}
	if !a.Profile.AppleSignIn || a.Profile.GoogleSignIn || !a.Profile.SellsSubscriptions || !a.Profile.SendsPushes {
		t.Fatalf("profile flags = %+v", a.Profile)
	}
	// Deferred Apple credentials never touch the settings.
	if a.Spec.Settings.Apple != nil || a.Spec.Settings.Google != nil {
		t.Fatalf("deferred providers must stay unconfigured: %+v", a.Spec.Settings)
	}
	mon := a.Spec.Monetization
	if mon == nil || len(mon.Entitlements) != 1 || len(mon.Products) != 1 {
		t.Fatalf("monetization = %+v", mon)
	}
	if e := mon.Entitlements[0]; e.Identifier != "pro" || e.DisplayName != "Pro" {
		t.Fatalf("entitlement = %+v", e)
	}
	p := mon.Products[0]
	if p.Identifier != "monthly" || p.DisplayName != "Monthly" || p.BillingPeriod != "monthly" ||
		p.PriceAmountMicros != 9_990_000 || p.Currency != "USD" ||
		p.AppleProductId != "pro.monthly" || p.GoogleProductId != "pro_monthly" ||
		len(p.Entitlements) != 1 || p.Entitlements[0] != "pro" {
		t.Fatalf("tier = %+v", p)
	}
	if a.Push == nil || !a.Push.Enabled || a.Push.WebpushVapidPublicKey != "" {
		t.Fatalf("push = %+v", a.Push)
	}
	// A Flutter-only app never sees VAPID keys.
	if strings.Contains(out, "VAPID") {
		t.Errorf("native-only flow should not prompt for a VAPID key:\n%s", out)
	}
	if !strings.Contains(out, "Firebase") {
		t.Errorf("Android push should surface the Firebase caveat:\n%s", out)
	}
	joined := strings.Join(a.Deferred, "\n")
	for _, want := range []string{"moth setup apple", "moth setup billing"} {
		if !strings.Contains(joined, want) {
			t.Errorf("deferred should point at %q: %v", want, a.Deferred)
		}
	}
}

// TestInitWizardGoogleCredentialsNow enters the Google web credentials
// in-flow: the provider lands enabled in the settings with nothing
// deferred, and native client-ID prompts never render for a web-only app.
func TestInitWizardGoogleCredentialsNow(t *testing.T) {
	a, out, err := runWizard(t, script(
		"Web Two",
		"web-two",
		"web",
		"", "", "", // email/password defaults
		"y",                                 // google
		"y",                                 // enter credentials now
		"web-id.apps.googleusercontent.com", // web client id
		"shhh",                              // web client secret
		"n",                                 // apple
		"n",                                 // subscriptions
		"n",                                 // pushes
		"",                                  // create now
	))
	if err != nil {
		t.Fatalf("wizard: %v\n%s", err, out)
	}
	g := a.Spec.Settings.Google
	if g == nil || !g.Enabled || g.WebClientId != "web-id.apps.googleusercontent.com" || g.WebClientSecret != "shhh" {
		t.Fatalf("google config = %+v", g)
	}
	if g.IosClientId != "" || g.AndroidClientId != "" {
		t.Fatalf("web-only flow must not collect native client IDs: %+v", g)
	}
	if strings.Contains(out, "iOS OAuth client ID") || strings.Contains(out, "Android OAuth client ID") {
		t.Errorf("web-only flow should not prompt for native client IDs:\n%s", out)
	}
	if len(a.Deferred) != 0 {
		t.Fatalf("nothing was deferred, got %v", a.Deferred)
	}
}

// TestInitWizardGoogleAndroidOnly enters Google credentials on an
// Android-only project: Android ID tokens carry the *web* client ID as
// audience, so the wizard must collect the web client too (with the one-line
// explanation), exactly like the SPA wizard — otherwise verification would
// reject every Android token.
func TestInitWizardGoogleAndroidOnly(t *testing.T) {
	a, out, err := runWizard(t, script(
		"Droid App",
		"",
		"android",
		"", "", "", // email/password defaults
		"y",                                   // google
		"y",                                   // enter credentials now
		"web-id.apps.googleusercontent.com",   // web client id
		"",                                    // web client secret: none yet
		"droid-id.apps.googleusercontent.com", // android client id
		"n",                                   // apple
		"n",                                   // subscriptions
		"n",                                   // pushes
		"",                                    // create now
	))
	if err != nil {
		t.Fatalf("wizard: %v\n%s", err, out)
	}
	g := a.Spec.Settings.Google
	if g == nil || !g.Enabled ||
		g.WebClientId != "web-id.apps.googleusercontent.com" ||
		g.AndroidClientId != "droid-id.apps.googleusercontent.com" {
		t.Fatalf("google config = %+v", g)
	}
	if g.IosClientId != "" {
		t.Fatalf("android-only flow must not collect the iOS client ID: %+v", g)
	}
	if !strings.Contains(out, "web client ID as audience") {
		t.Errorf("android-only flow should explain why the web client is needed:\n%s", out)
	}
	if strings.Contains(out, "iOS OAuth client ID") {
		t.Errorf("android-only flow should not prompt for the iOS client ID:\n%s", out)
	}
}

// TestInitWizardVapidPromptReasks validates the VAPID public key at the
// prompt with the server's own shape rule (internal/push): a typo re-asks
// instead of failing the UpdatePushSettings write after the project exists.
func TestInitWizardVapidPromptReasks(t *testing.T) {
	raw := make([]byte, 65)
	raw[0] = 0x04
	validKey := base64.RawURLEncoding.EncodeToString(raw)
	a, out, err := runWizard(t, script(
		"Pushy", "", "web",
		"", "", "", // email/password defaults
		"", "", "", // google, apple, subscriptions: no
		"y",         // pushes
		"not-a-key", // invalid VAPID key -> re-ask
		validKey,    // valid key
		"",          // create now
	))
	if err != nil {
		t.Fatalf("wizard: %v\n%s", err, out)
	}
	if !strings.Contains(out, "try again") {
		t.Errorf("an invalid VAPID key should re-ask:\n%s", out)
	}
	if a.Push == nil || a.Push.WebpushVapidPublicKey != validKey {
		t.Fatalf("push = %+v, want the valid key", a.Push)
	}
	if len(a.Deferred) != 0 {
		t.Fatalf("nothing was deferred, got %v", a.Deferred)
	}
}

// TestInitWizardAbandonAtConfirm declines the review confirmation: the
// wizard returns ErrInitAborted and hands back no answers to execute.
func TestInitWizardAbandonAtConfirm(t *testing.T) {
	a, out, err := runWizard(t, script(
		"Doomed", "", "web",
		"", "", "", // email/password defaults
		"", "", "", "", // decline google/apple/subscriptions/pushes
		"n", // create now? no
	))
	if !errors.Is(err, ErrInitAborted) {
		t.Fatalf("err = %v, want ErrInitAborted\n%s", err, out)
	}
	if a != nil {
		t.Fatalf("aborted wizard must return no answers, got %+v", a)
	}
}

// TestInitWizardRetriesInvalidAnswers re-asks on validation failures
// instead of dying: a bogus platform and a bogus price both get a second
// chance.
func TestInitWizardRetriesInvalidAnswers(t *testing.T) {
	a, out, err := runWizard(t, script(
		"Retry", "",
		"desktop", // invalid platform -> re-ask
		"web",
		"", "", "", // email/password defaults
		"", "", // google, apple
		"y",    // subscriptions
		"Pro!", // invalid identifier -> re-ask
		"pro",
		"",        // entitlement display name: default pro
		"monthly", // tier identifier
		"",        // display name default
		"",        // period default
		"free",    // invalid price -> re-ask
		"0.99",
		"",  // currency default
		"",  // add another tier: no
		"n", // pushes
		"",  // create now
	))
	if err != nil {
		t.Fatalf("wizard: %v\n%s", err, err)
	}
	if !strings.Contains(out, "try again") {
		t.Errorf("invalid answers should re-ask:\n%s", out)
	}
	if a.Spec.Monetization.Products[0].PriceAmountMicros != 990_000 {
		t.Fatalf("price = %d", a.Spec.Monetization.Products[0].PriceAmountMicros)
	}
	if a.Spec.Monetization.Entitlements[0].DisplayName != "pro" {
		t.Fatalf("default display name = %q", a.Spec.Monetization.Entitlements[0].DisplayName)
	}
}
