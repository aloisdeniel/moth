package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/cli"
)

// runCLIStdin runs the in-process command tree with a scripted stdin — the
// wizard reads it through the injected command streams, so no TTY is
// involved (and no TTY refusal either: the refusal only triggers on a real
// non-terminal *os.File, i.e. actual piped stdin).
func runCLIStdin(t *testing.T, stdin string, args ...string) (stdout string, err error) {
	t.Helper()
	var out, errb bytes.Buffer
	root := newRootCmd()
	root.SetIn(strings.NewReader(stdin))
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs(args)
	err = root.Execute()
	return out.String() + errb.String(), err
}

// initScript joins scripted wizard answers into stdin.
func initScript(answers ...string) string {
	return strings.Join(answers, "\n") + "\n"
}

// TestProjectInitMinimalWebOnly drives the two-step acceptance flow against
// a real in-process server: web-only, free, no push — created with the
// defaults, keys printed once, checklist and apply spec appended.
func TestProjectInitMinimalWebOnly(t *testing.T) {
	ts, pat := newSkillTestServer(t)
	configureContext(t, ts.URL, pat)
	ctx := context.Background()

	out, err := runCLIStdin(t, initScript(
		"Web App",  // name
		"",         // slug: derived
		"web",      // platforms
		"", "", "", // email/password defaults
		"", "", "", "", // decline google/apple/subscriptions/pushes
		"", // create now: default yes
	), "--context", "test", "project", "init")
	if err != nil {
		t.Fatalf("project init: %v\n%s", err, out)
	}

	// Keys are printed exactly once, secret included (the wizard finale).
	if !strings.Contains(out, "publishable key: pk_") {
		t.Errorf("missing publishable key:\n%s", out)
	}
	if !strings.Contains(out, "secret key (shown exactly once — store it now): sk_") {
		t.Errorf("missing one-time secret key:\n%s", out)
	}

	client := cli.New(ts.URL, pat)
	list, err := client.Projects.ListProjects(ctx, connect.NewRequest(&adminv1.ListProjectsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Msg.Projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(list.Msg.Projects))
	}
	p := list.Msg.Projects[0]
	if p.Name != "Web App" || p.Slug != "web-app" {
		t.Fatalf("project = %q/%q", p.Name, p.Slug)
	}
	s := p.Settings
	if !s.GetAllowPublicSignup() || !s.GetRequireEmailVerification() || s.GetPasswordMinLength() != 8 {
		t.Fatalf("settings = %+v", s)
	}
	if s.GetGoogle().GetEnabled() || s.GetApple().GetEnabled() {
		t.Fatalf("providers must stay disabled: %+v", s)
	}
	prof, err := client.Profiles.GetProfile(ctx,
		connect.NewRequest(&adminv1.GetProfileRequest{ProjectId: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if !prof.Msg.HasProfile {
		t.Fatal("init must store the setup profile")
	}
	if got := prof.Msg.Profile.Platforms; len(got) != 1 || got[0] != adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB {
		t.Fatalf("profile platforms = %v", got)
	}
	if prof.Msg.Profile.SellsSubscriptions || prof.Msg.Profile.SendsPushes {
		t.Fatalf("profile flags = %+v", prof.Msg.Profile)
	}
	// The emitted spec is printed and parses as an apply document, with the
	// honest scope note: the profile and push settings are not part of it.
	if !strings.Contains(out, "slug: web-app") {
		t.Errorf("printed spec should carry the derived slug:\n%s", out)
	}
	if !strings.Contains(out, "the setup profile and push settings are not part of apply specs") {
		t.Errorf("spec output should carry the scope note:\n%s", out)
	}
}

// TestProjectInitFullDeferredFlutter drives the Flutter acceptance flow —
// iOS + Android, Apple sign-in deferred, subscriptions, push — and asserts
// the created project's profile, provider config, catalog and push
// settings, the checklist text, and that the emitted apply spec
// round-trips through `moth project apply` as a no-op.
func TestProjectInitFullDeferredFlutter(t *testing.T) {
	ts, pat := newSkillTestServer(t)
	configureContext(t, ts.URL, pat)
	ctx := context.Background()
	specPath := filepath.Join(t.TempDir(), "flutter-app.yaml")

	out, err := runCLIStdin(t, initScript(
		"Flutter App",  // name
		"flutter-app",  // slug
		"ios, android", // platforms
		"", "", "",     // email/password defaults
		"n",           // google
		"y",           // apple
		"n",           // credentials now? -> defer
		"y",           // sells subscriptions
		"pro",         // entitlement identifier
		"Pro",         // entitlement display name
		"monthly",     // tier identifier
		"Monthly",     // tier display name
		"",            // billing period default
		"9.99",        // price
		"",            // currency default USD
		"pro.monthly", // App Store product id
		"pro_monthly", // Google Play product id
		"",            // no more tiers
		"y",           // sends pushes
		"",            // create now
	), "--context", "test", "project", "init", "--spec-out", specPath)
	if err != nil {
		t.Fatalf("project init: %v\n%s", err, out)
	}

	client := cli.New(ts.URL, pat)
	list, err := client.Projects.ListProjects(ctx, connect.NewRequest(&adminv1.ListProjectsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Msg.Projects) != 1 || list.Msg.Projects[0].Slug != "flutter-app" {
		t.Fatalf("projects = %+v", list.Msg.Projects)
	}
	p := list.Msg.Projects[0]

	// Profile: the answers themselves.
	prof, err := client.Profiles.GetProfile(ctx,
		connect.NewRequest(&adminv1.GetProfileRequest{ProjectId: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	pr := prof.Msg.Profile
	if len(pr.GetPlatforms()) != 2 || !pr.GetAppleSignIn() || pr.GetGoogleSignIn() ||
		!pr.GetSellsSubscriptions() || !pr.GetSendsPushes() {
		t.Fatalf("profile = %+v", pr)
	}

	// Provider config: deferring left Apple unconfigured (checklist item,
	// not a silent half-config).
	if p.Settings.GetApple().GetEnabled() || p.Settings.GetGoogle().GetEnabled() {
		t.Fatalf("deferred providers must stay disabled: %+v", p.Settings)
	}

	// Catalog: the entitlement and tier landed as defined.
	ents, err := client.Entitlements.ListEntitlements(ctx,
		connect.NewRequest(&adminv1.ListEntitlementsRequest{ProjectId: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if len(ents.Msg.Entitlements) != 1 || ents.Msg.Entitlements[0].Identifier != "pro" {
		t.Fatalf("entitlements = %+v", ents.Msg.Entitlements)
	}
	prods, err := client.Products.ListProducts(ctx,
		connect.NewRequest(&adminv1.ListProductsRequest{ProjectId: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if len(prods.Msg.Products) != 1 {
		t.Fatalf("products = %+v", prods.Msg.Products)
	}
	tier := prods.Msg.Products[0]
	if tier.Identifier != "monthly" || tier.BillingPeriod != "monthly" ||
		tier.PriceAmountMicros != 9_990_000 || tier.Currency != "USD" ||
		tier.AppleProductId != "pro.monthly" || tier.GoogleProductId != "pro_monthly" ||
		len(tier.EntitlementIds) != 1 || tier.EntitlementIds[0] != ents.Msg.Entitlements[0].Id {
		t.Fatalf("tier = %+v", tier)
	}

	// Push settings: enabled, no VAPID key (native-only).
	push, err := client.Push.GetPushSettings(ctx,
		connect.NewRequest(&adminv1.GetPushSettingsRequest{ProjectId: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if !push.Msg.Settings.GetEnabled() || push.Msg.Settings.GetWebpushVapidPublicKey() != "" {
		t.Fatalf("push settings = %+v", push.Msg.Settings)
	}

	// The checklist as text: the deferred Apple credential and the billing
	// credentials + catalog sync, each with the CLI command finishing it.
	for _, want := range []string{
		"Outstanding setup:",
		"Finish Sign in with Apple",
		"moth setup apple",
		"Add store billing credentials",
		"moth setup billing",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("checklist should mention %q:\n%s", want, out)
		}
	}

	// The spec's scope note prints alongside the written-file pointer.
	if !strings.Contains(out, "the setup profile and push settings are not part of apply specs") {
		t.Errorf("spec output should carry the scope note:\n%s", out)
	}

	// The emitted spec round-trips through the apply parser and diffs to
	// zero against the freshly built project.
	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := cli.SpecFromYAML(raw)
	if err != nil {
		t.Fatalf("emitted spec does not parse: %v\n%s", err, raw)
	}
	if spec.Slug != "flutter-app" || spec.Name != "Flutter App" || spec.Monetization == nil {
		t.Fatalf("emitted spec = %+v", spec)
	}
	applyOut, err := runCLIStdin(t, "", "--context", "test", "project", "apply", "-f", specPath, "--yes", "--json")
	if err != nil {
		t.Fatalf("apply emitted spec: %v\n%s", err, applyOut)
	}
	var applied struct {
		Plan    cli.ApplyPlan        `json:"plan"`
		Mon     cli.MonetizationPlan `json:"monetization"`
		Applied bool                 `json:"applied"`
	}
	if err := json.Unmarshal([]byte(applyOut), &applied); err != nil {
		t.Fatalf("apply --json: %v\n%s", err, applyOut)
	}
	if applied.Applied || !applied.Plan.Empty() || !applied.Mon.Empty() {
		t.Fatalf("re-applying the emitted spec must be a no-op: %+v\n%s", applied, applyOut)
	}
}

// TestProjectInitFollowUpFailurePrintsKeysFirst injects a follow-up-write
// failure (a duplicate tier identifier the server rejects on the second
// CreateProduct) and asserts the milestone-22 hardening: the pk_/sk_ keys
// were printed before the failing write, the failure is reported with the
// tab/CLI pointer, the checklist and the apply spec still print, and the
// command exits non-zero.
func TestProjectInitFollowUpFailurePrintsKeysFirst(t *testing.T) {
	ts, pat := newSkillTestServer(t)
	configureContext(t, ts.URL, pat)
	ctx := context.Background()

	out, err := runCLIStdin(t, initScript(
		"Dup App",  // name
		"",         // slug: derived
		"web",      // platforms
		"", "", "", // email/password defaults
		"", "", // google, apple: no
		"y",       // sells subscriptions
		"pro",     // entitlement identifier
		"",        // entitlement display name default
		"monthly", // tier identifier
		"",        // tier display name default
		"",        // billing period default
		"9.99",    // price
		"",        // currency default
		"y",       // add another tier
		"monthly", // duplicate identifier — the server rejects this create
		"",        // display name default
		"",        // period default
		"4.99",    // price
		"",        // currency default
		"n",       // no more tiers
		"n",       // pushes
		"",        // create now
	), "--context", "test", "project", "init")
	if err == nil {
		t.Fatalf("a follow-up failure must exit non-zero:\n%s", out)
	}
	if !strings.Contains(err.Error(), "setup step(s) failed") {
		t.Fatalf("err = %v", err)
	}

	// The keys were printed BEFORE the failing write — the sk_ is never lost.
	if !strings.Contains(out, "publishable key: pk_") ||
		!strings.Contains(out, "secret key (shown exactly once — store it now): sk_") {
		t.Fatalf("keys must print before follow-up writes:\n%s", out)
	}
	// The failure is reported with the pointer that finishes it, and the
	// checklist and spec still print.
	if !strings.Contains(out, "could not configure the monetization catalog") ||
		!strings.Contains(out, "Monetization tab or 'moth setup billing'") {
		t.Errorf("missing the follow-up failure report:\n%s", out)
	}
	if !strings.Contains(out, "Outstanding setup:") {
		t.Errorf("the checklist must still print after a failure:\n%s", out)
	}
	if !strings.Contains(out, "slug: dup-app") ||
		!strings.Contains(out, "the setup profile and push settings are not part of apply specs") {
		t.Errorf("the apply spec must still print after a failure:\n%s", out)
	}

	// The project exists with everything that landed before the failure: the
	// entitlement and the first tier.
	client := cli.New(ts.URL, pat)
	list, err := client.Projects.ListProjects(ctx, connect.NewRequest(&adminv1.ListProjectsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Msg.Projects) != 1 {
		t.Fatalf("projects = %+v", list.Msg.Projects)
	}
	prods, err := client.Products.ListProducts(ctx,
		connect.NewRequest(&adminv1.ListProductsRequest{ProjectId: list.Msg.Projects[0].Id}))
	if err != nil {
		t.Fatal(err)
	}
	if len(prods.Msg.Products) != 1 || prods.Msg.Products[0].Identifier != "monthly" {
		t.Fatalf("products = %+v", prods.Msg.Products)
	}
	// The profile write runs after the failed catalog write — collected, not
	// aborted, so the intent still lands and the checklist keys off it.
	prof, err := client.Profiles.GetProfile(ctx,
		connect.NewRequest(&adminv1.GetProfileRequest{ProjectId: list.Msg.Projects[0].Id}))
	if err != nil {
		t.Fatal(err)
	}
	if !prof.Msg.HasProfile || !prof.Msg.Profile.GetSellsSubscriptions() {
		t.Fatalf("profile must still be written after an earlier failure: %+v", prof.Msg)
	}
}

// TestProjectInitAbandonCreatesNothing declines the final confirmation:
// the command fails and no project exists afterwards.
func TestProjectInitAbandonCreatesNothing(t *testing.T) {
	ts, pat := newSkillTestServer(t)
	configureContext(t, ts.URL, pat)

	out, err := runCLIStdin(t, initScript(
		"Doomed", "", "web",
		"", "", "", // email/password defaults
		"", "", "", "", // decline everything optional
		"n", // create now? no
	), "--context", "test", "project", "init")
	if err == nil {
		t.Fatalf("declining the confirmation must fail the command:\n%s", out)
	}
	if !strings.Contains(err.Error(), "nothing was created") {
		t.Fatalf("err = %v", err)
	}

	client := cli.New(ts.URL, pat)
	list, err := client.Projects.ListProjects(context.Background(),
		connect.NewRequest(&adminv1.ListProjectsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Msg.Projects) != 0 {
		t.Fatalf("abandoning the wizard must create nothing, got %+v", list.Msg.Projects)
	}
}

// TestProjectInitRefusesNonInteractiveStdin runs init on the process
// stdin (under `go test` that is a real non-terminal file): it must refuse
// with the pointer at the scriptable commands, before dialing anything.
func TestProjectInitRefusesNonInteractiveStdin(t *testing.T) {
	var out, errb bytes.Buffer
	root := newRootCmd()
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs([]string{"project", "init"})
	err := root.Execute()
	if err == nil {
		t.Fatalf("init on piped stdin must fail:\n%s", out.String())
	}
	for _, want := range []string{"moth project create", "moth project apply"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should point at %q: %v", want, err)
		}
	}
}
