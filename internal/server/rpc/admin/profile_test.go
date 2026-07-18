package adminrpc

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/audit"
	"github.com/aloisdeniel/moth/internal/push"
	"github.com/aloisdeniel/moth/internal/store"
)

// profileFixture is one project and a profile handler whose audit sink
// writes to the same store; smtp toggles the instance SMTP probe.
type profileFixture struct {
	t       *testing.T
	h       *ProfileHandler
	st      *store.Store
	now     time.Time
	project store.Project
	smtp    bool
}

func newProfileFixture(t *testing.T) *profileFixture {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	f := &profileFixture{t: t, st: st, now: now}
	f.project = f.seedProject("demo")
	auditor := NewAuditor(audit.New(st, slog.New(slog.DiscardHandler), func() time.Time { return f.now }))
	f.h = NewProfileHandler(st, auditor, func() bool { return f.smtp }, func() time.Time { return f.now })
	return f
}

func (f *profileFixture) seedProject(slug string) store.Project {
	f.t.Helper()
	p := store.Project{
		ID: NewID(), Name: slug, Slug: slug,
		PublishableKey: "pk_" + NewID(), SecretKeyHash: "hash-" + slug,
		Settings: store.DefaultProjectSettings(), CreatedAt: f.now, UpdatedAt: f.now,
	}
	k := store.ProjectKey{
		ID: NewID(), ProjectID: p.ID, Kid: "kid-" + slug, Algorithm: "ES256",
		PublicKeyPEM: "PEM", PrivateKeyEnc: []byte{1},
		Status: store.ProjectKeyStatusActive, CreatedAt: f.now,
	}
	if err := f.st.CreateProject(context.Background(), p, k); err != nil {
		f.t.Fatal(err)
	}
	return p
}

// setProfile installs a profile through the RPC, failing the test on error.
func (f *profileFixture) setProfile(projectID string, p *adminv1.Profile) {
	f.t.Helper()
	_, err := f.h.UpdateProfile(context.Background(), connect.NewRequest(&adminv1.UpdateProfileRequest{
		ProjectId: projectID, Profile: p,
	}))
	if err != nil {
		f.t.Fatalf("UpdateProfile: %v", err)
	}
}

// itemIDs returns the checklist item ids in order.
func (f *profileFixture) itemIDs(projectID string) []string {
	f.t.Helper()
	resp, err := f.h.GetProjectSetupStatus(context.Background(), connect.NewRequest(
		&adminv1.GetProjectSetupStatusRequest{ProjectId: projectID}))
	if err != nil {
		f.t.Fatalf("GetProjectSetupStatus: %v", err)
	}
	if !resp.Msg.HasProfile {
		f.t.Fatalf("expected a profile on %s", projectID)
	}
	ids := make([]string, 0, len(resp.Msg.Items))
	for _, item := range resp.Msg.Items {
		ids = append(ids, item.Id)
	}
	return ids
}

func equalIDs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestProfileRoundTrip(t *testing.T) {
	f := newProfileFixture(t)
	ctx := context.Background()

	// A fresh project has no profile.
	got, err := f.h.GetProfile(ctx, connect.NewRequest(&adminv1.GetProfileRequest{ProjectId: f.project.ID}))
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if got.Msg.HasProfile || got.Msg.Profile != nil {
		t.Fatalf("fresh project has a profile: %+v", got.Msg)
	}

	in := &adminv1.Profile{
		Platforms: []adminv1.ProfilePlatform{
			adminv1.ProfilePlatform_PROFILE_PLATFORM_IOS,
			adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB,
		},
		GoogleSignIn: true, AppleSignIn: true,
		SellsSubscriptions: true, SendsPushes: true, ChecklistDismissed: true,
	}
	updated, err := f.h.UpdateProfile(ctx, connect.NewRequest(&adminv1.UpdateProfileRequest{
		ProjectId: f.project.ID, Profile: in,
	}))
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if len(updated.Msg.Profile.Platforms) != 2 || !updated.Msg.Profile.ChecklistDismissed {
		t.Fatalf("updated: %+v", updated.Msg.Profile)
	}

	got, err = f.h.GetProfile(ctx, connect.NewRequest(&adminv1.GetProfileRequest{ProjectId: f.project.ID}))
	if err != nil {
		t.Fatal(err)
	}
	p := got.Msg.Profile
	if !got.Msg.HasProfile || p == nil {
		t.Fatalf("read back: %+v", got.Msg)
	}
	if len(p.Platforms) != 2 ||
		p.Platforms[0] != adminv1.ProfilePlatform_PROFILE_PLATFORM_IOS ||
		p.Platforms[1] != adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB ||
		!p.GoogleSignIn || !p.AppleSignIn || !p.SellsSubscriptions ||
		!p.SendsPushes || !p.ChecklistDismissed {
		t.Fatalf("read back: %+v", p)
	}

	// The update is audit-logged.
	entries, err := f.st.ListAudit(ctx, store.AuditFilter{Action: ActionProfileUpdate, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ProjectID != f.project.ID {
		t.Fatalf("audit entries: %+v", entries)
	}

	// Validation: nil profile and empty platforms are rejected before storage.
	_, err = f.h.UpdateProfile(ctx, connect.NewRequest(&adminv1.UpdateProfileRequest{ProjectId: f.project.ID}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("nil profile: want InvalidArgument, got %v", err)
	}
	_, err = f.h.UpdateProfile(ctx, connect.NewRequest(&adminv1.UpdateProfileRequest{
		ProjectId: f.project.ID, Profile: &adminv1.Profile{},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("no platforms: want InvalidArgument, got %v", err)
	}

	// Unknown projects are NotFound on all three RPCs.
	if _, err = f.h.GetProfile(ctx, connect.NewRequest(&adminv1.GetProfileRequest{ProjectId: "nope"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("get unknown project: want NotFound, got %v", err)
	}
	_, err = f.h.UpdateProfile(ctx, connect.NewRequest(&adminv1.UpdateProfileRequest{
		ProjectId: "nope",
		Profile:   &adminv1.Profile{Platforms: []adminv1.ProfilePlatform{adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB}},
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("update unknown project: want NotFound, got %v", err)
	}
	if _, err = f.h.GetProjectSetupStatus(ctx, connect.NewRequest(&adminv1.GetProjectSetupStatusRequest{ProjectId: "nope"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("status unknown project: want NotFound, got %v", err)
	}
}

func TestSetupStatusNoProfile(t *testing.T) {
	f := newProfileFixture(t)
	resp, err := f.h.GetProjectSetupStatus(context.Background(), connect.NewRequest(
		&adminv1.GetProjectSetupStatusRequest{ProjectId: f.project.ID}))
	if err != nil {
		t.Fatalf("GetProjectSetupStatus: %v", err)
	}
	// Pre-wizard projects get no checklist at all — not even theme/SMTP items.
	if resp.Msg.HasProfile || len(resp.Msg.Items) != 0 || resp.Msg.ChecklistDismissed {
		t.Fatalf("no-profile status: %+v", resp.Msg)
	}
}

// TestSetupStatusChecklist walks a fully-intending project (all platforms,
// all features chosen, nothing configured) from the full checklist to an
// empty one, asserting each item disappears the moment — and only the
// moment — its underlying configuration exists.
func TestSetupStatusChecklist(t *testing.T) {
	f := newProfileFixture(t)
	ctx := context.Background()
	f.setProfile(f.project.ID, &adminv1.Profile{
		Platforms: []adminv1.ProfilePlatform{
			adminv1.ProfilePlatform_PROFILE_PLATFORM_IOS,
			adminv1.ProfilePlatform_PROFILE_PLATFORM_ANDROID,
			adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB,
		},
		GoogleSignIn: true, AppleSignIn: true,
		SellsSubscriptions: true, SendsPushes: true,
	})

	all := []string{"google_credentials", "apple_credentials", "billing_credentials",
		"catalog_sync", "push_vapid", "theme_default", "smtp"}
	if got := f.itemIDs(f.project.ID); !equalIDs(got, all) {
		t.Fatalf("initial checklist: %v", got)
	}

	steps := []struct {
		name      string
		configure func()
		remove    string
	}{
		{"google provider", func() {
			p, err := f.st.GetProject(ctx, f.project.ID)
			if err != nil {
				t.Fatal(err)
			}
			p.Settings.Google = store.GoogleProviderSettings{Enabled: true, WebClientID: "web.apps.googleusercontent.com"}
			p.UpdatedAt = f.now
			if err := f.st.UpdateProject(ctx, p); err != nil {
				t.Fatal(err)
			}
		}, "google_credentials"},
		{"apple provider", func() {
			p, err := f.st.GetProject(ctx, f.project.ID)
			if err != nil {
				t.Fatal(err)
			}
			p.Settings.Apple = store.AppleProviderSettings{
				Enabled: true, ServicesID: "com.demo.web", TeamID: "TEAM123456", KeyID: "KEY1234567",
			}
			p.UpdatedAt = f.now
			if err := f.st.UpdateProject(ctx, p); err != nil {
				t.Fatal(err)
			}
			if err := f.st.SetProviderSecret(ctx, f.project.ID,
				store.ProviderSecretApplePrivateKey, []byte("enc"), f.now); err != nil {
				t.Fatal(err)
			}
		}, "apple_credentials"},
		{"billing credentials", func() {
			if err := f.st.UpsertBillingCredentials(ctx, store.BillingCredentials{
				ProjectID:     f.project.ID,
				AppleIAPKeyID: "IAPKEY1234", AppleIAPIssuerID: "issuer-uuid",
				AppleIAPKeyEnc: []byte("enc"), AppleBundleID: "com.demo.app",
				GoogleServiceAccountEnc: []byte("enc"), GooglePackageName: "com.demo.app",
				StripeSecretKeyEnc: []byte("enc"),
				CreatedAt:          f.now, UpdatedAt: f.now,
			}); err != nil {
				t.Fatal(err)
			}
		}, "billing_credentials"},
		{"catalog sync", func() {
			prod := store.Product{
				ID: NewID(), ProjectID: f.project.ID, Identifier: "pro_monthly",
				DisplayName: "Pro monthly", AppleProductID: "com.demo.pro.monthly",
				GoogleProductID: "pro-monthly", StripePriceID: "price_123",
				BillingPeriod: "monthly", PriceAmountMicros: 9_990_000, Currency: "USD",
				CreatedAt: f.now, UpdatedAt: f.now,
			}
			if err := f.st.CreateProduct(ctx, prod); err != nil {
				t.Fatal(err)
			}
			for _, storeName := range []string{store.SubscriptionStoreApple,
				store.SubscriptionStoreGoogle, store.SubscriptionStoreStripe} {
				if err := f.st.UpsertProductStoreSync(ctx, store.ProductStoreSync{
					ProductID: prod.ID, Store: storeName, Status: store.ProductSyncInSync,
					CreatedAt: f.now, UpdatedAt: f.now,
				}); err != nil {
					t.Fatal(err)
				}
			}
		}, "catalog_sync"},
		{"push settings", func() {
			raw, err := push.Encode(push.Config{Enabled: true, WebPushVAPIDPublicKey: validVAPIDKey(t)})
			if err != nil {
				t.Fatal(err)
			}
			if err := f.st.SetProjectPush(ctx, f.project.ID, raw, f.now); err != nil {
				t.Fatal(err)
			}
		}, "push_vapid"},
		{"theme", func() {
			if err := f.st.SetProjectTheme(ctx, store.ThemeRevision{
				ID: NewID(), ProjectID: f.project.ID, Theme: []byte{1}, CreatedAt: f.now,
			}, ""); err != nil {
				t.Fatal(err)
			}
		}, "theme_default"},
		{"smtp", func() { f.smtp = true }, "smtp"},
	}
	remaining := all
	for _, step := range steps {
		step.configure()
		filtered := remaining[:0:0]
		for _, id := range remaining {
			if id != step.remove {
				filtered = append(filtered, id)
			}
		}
		remaining = filtered
		if got := f.itemIDs(f.project.ID); !equalIDs(got, remaining) {
			t.Fatalf("after %s: got %v, want %v", step.name, got, remaining)
		}
	}
	if len(remaining) != 0 {
		t.Fatalf("checklist not exhausted: %v", remaining)
	}
}

// TestSetupStatusRespectsIntent verifies that features the profile did not
// choose never produce items, and that platform choice narrows the
// billing/push branches.
func TestSetupStatusRespectsIntent(t *testing.T) {
	f := newProfileFixture(t)
	f.smtp = true // silence the instance-level item

	// Minimal intent: web-only, nothing chosen — only the theme nags.
	f.setProfile(f.project.ID, &adminv1.Profile{
		Platforms: []adminv1.ProfilePlatform{adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB},
	})
	if got := f.itemIDs(f.project.ID); !equalIDs(got, []string{"theme_default"}) {
		t.Fatalf("minimal intent: %v", got)
	}

	// Pushes on a native-only app: no VAPID key is ever demanded, but a
	// disabled push registry (a failed or aborted wizard write) still
	// surfaces as push_enable — sends_pushes intent must never go silent.
	f.setProfile(f.project.ID, &adminv1.Profile{
		Platforms:   []adminv1.ProfilePlatform{adminv1.ProfilePlatform_PROFILE_PLATFORM_IOS},
		SendsPushes: true,
	})
	if got := f.itemIDs(f.project.ID); !equalIDs(got, []string{"push_enable", "theme_default"}) {
		t.Fatalf("native-only pushes, registry disabled: %v", got)
	}
	raw, err := push.Encode(push.Config{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.st.SetProjectPush(context.Background(), f.project.ID, raw, f.now); err != nil {
		t.Fatal(err)
	}
	if got := f.itemIDs(f.project.ID); !equalIDs(got, []string{"theme_default"}) {
		t.Fatalf("native-only pushes, registry enabled: %v", got)
	}

	// Subscriptions on web-only: billing narrows to Stripe, no store items
	// for App Store / Google Play.
	f.setProfile(f.project.ID, &adminv1.Profile{
		Platforms:          []adminv1.ProfilePlatform{adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB},
		SellsSubscriptions: true,
	})
	resp, err := f.h.GetProjectSetupStatus(context.Background(), connect.NewRequest(
		&adminv1.GetProjectSetupStatusRequest{ProjectId: f.project.ID}))
	if err != nil {
		t.Fatal(err)
	}
	var billing *adminv1.SetupItem
	for _, item := range resp.Msg.Items {
		if item.Id == "billing_credentials" {
			billing = item
		}
	}
	if billing == nil {
		t.Fatalf("web-only subscriptions: no billing item in %+v", resp.Msg.Items)
	}
	if billing.Detail != "Store credentials are missing for: Stripe." {
		t.Fatalf("billing detail not narrowed to Stripe: %q", billing.Detail)
	}
}

// hasItemID reports whether the checklist ids include id.
func hasItemID(ids []string, id string) bool {
	for _, got := range ids {
		if got == id {
			return true
		}
	}
	return false
}

// TestSetupStatusApplePlatformAware verifies the Apple probe keys off the
// profile's platforms: a native-only (iOS) project is complete with a bundle
// ID alone — the native flow verifies ID tokens against bundle-ID audiences
// and never needs the team/key/.p8 trio — while a web or Android platform
// brings the web-redirect pieces back as required.
func TestSetupStatusApplePlatformAware(t *testing.T) {
	ios := adminv1.ProfilePlatform_PROFILE_PLATFORM_IOS
	android := adminv1.ProfilePlatform_PROFILE_PLATFORM_ANDROID
	web := adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB
	cases := []struct {
		name       string
		platforms  []adminv1.ProfilePlatform
		bundleIDs  []string
		wantItem   bool
		wantDetail string
	}{
		{
			name: "ios-only with a bundle ID is configured", platforms: []adminv1.ProfilePlatform{ios},
			bundleIDs: []string{"com.demo.app"}, wantItem: false,
		},
		{
			name: "ios-only without a bundle ID still nags", platforms: []adminv1.ProfilePlatform{ios},
			wantItem: true, wantDetail: "Sign in with Apple is missing: bundle ID.",
		},
		{
			name: "a web platform demands the web-redirect pieces", platforms: []adminv1.ProfilePlatform{ios, web},
			bundleIDs: []string{"com.demo.app"}, wantItem: true,
			wantDetail: "Sign in with Apple is missing: team ID, key ID, private key.",
		},
		{
			name: "an android platform demands the web-redirect pieces", platforms: []adminv1.ProfilePlatform{ios, android},
			bundleIDs: []string{"com.demo.app"}, wantItem: true,
			wantDetail: "Sign in with Apple is missing: team ID, key ID, private key.",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newProfileFixture(t)
			ctx := context.Background()
			p, err := f.st.GetProject(ctx, f.project.ID)
			if err != nil {
				t.Fatal(err)
			}
			p.Settings.Apple = store.AppleProviderSettings{Enabled: true, BundleIDs: tc.bundleIDs}
			p.UpdatedAt = f.now
			if err := f.st.UpdateProject(ctx, p); err != nil {
				t.Fatal(err)
			}
			f.setProfile(f.project.ID, &adminv1.Profile{Platforms: tc.platforms, AppleSignIn: true})

			resp, err := f.h.GetProjectSetupStatus(ctx, connect.NewRequest(
				&adminv1.GetProjectSetupStatusRequest{ProjectId: f.project.ID}))
			if err != nil {
				t.Fatal(err)
			}
			var item *adminv1.SetupItem
			for _, it := range resp.Msg.Items {
				if it.Id == "apple_credentials" {
					item = it
				}
			}
			if !tc.wantItem {
				if item != nil {
					t.Fatalf("unexpected apple item: %+v", item)
				}
				return
			}
			if item == nil {
				t.Fatalf("missing apple item in %+v", resp.Msg.Items)
			}
			if item.Detail != tc.wantDetail {
				t.Fatalf("detail = %q, want %q", item.Detail, tc.wantDetail)
			}
		})
	}
}

// TestSetupStatusCatalogSyncDrift verifies the catalog_sync probe applies
// the same revision-drift demotion as the monetization status/plan surfaces:
// an in_sync row is only parity while the product still matches the recorded
// revision — a moth-side price edit brings the item back.
func TestSetupStatusCatalogSyncDrift(t *testing.T) {
	f := newProfileFixture(t)
	ctx := context.Background()
	f.smtp = true
	f.setProfile(f.project.ID, &adminv1.Profile{
		Platforms:          []adminv1.ProfilePlatform{adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB},
		SellsSubscriptions: true,
	})

	prod := store.Product{
		ID: NewID(), ProjectID: f.project.ID, Identifier: "pro-monthly",
		DisplayName: "Pro monthly", StripePriceID: "price_123",
		BillingPeriod: "monthly", PriceAmountMicros: 9_990_000, Currency: "USD",
		CreatedAt: f.now, UpdatedAt: f.now,
	}
	if err := f.st.CreateProduct(ctx, prod); err != nil {
		t.Fatal(err)
	}
	if err := f.st.UpsertProductStoreSync(ctx, store.ProductStoreSync{
		ProductID: prod.ID, Store: store.SubscriptionStoreStripe,
		Status:    store.ProductSyncInSync,
		Revision:  productRevision(prod, store.SubscriptionStoreStripe),
		CreatedAt: f.now, UpdatedAt: f.now,
	}); err != nil {
		t.Fatal(err)
	}

	// Synced with a matching revision: no catalog_sync item.
	if ids := f.itemIDs(f.project.ID); hasItemID(ids, "catalog_sync") {
		t.Fatalf("synced catalog must not produce catalog_sync: %v", ids)
	}

	// A price edit in moth makes the recorded revision stale: the item is
	// back even though the row still says in_sync.
	prod.PriceAmountMicros = 14_990_000
	prod.UpdatedAt = f.now
	if err := f.st.UpdateProduct(ctx, prod); err != nil {
		t.Fatal(err)
	}
	if ids := f.itemIDs(f.project.ID); !hasItemID(ids, "catalog_sync") {
		t.Fatalf("drifted catalog must produce catalog_sync: %v", ids)
	}
}

func TestSetupStatusChecklistDismissed(t *testing.T) {
	f := newProfileFixture(t)
	f.smtp = true
	f.setProfile(f.project.ID, &adminv1.Profile{
		Platforms:          []adminv1.ProfilePlatform{adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB},
		ChecklistDismissed: true,
	})
	resp, err := f.h.GetProjectSetupStatus(context.Background(), connect.NewRequest(
		&adminv1.GetProjectSetupStatusRequest{ProjectId: f.project.ID}))
	if err != nil {
		t.Fatal(err)
	}
	// Dismissal hides the card client-side but never fakes completeness: the
	// outstanding items are still computed and returned.
	if !resp.Msg.ChecklistDismissed || len(resp.Msg.Items) == 0 {
		t.Fatalf("dismissed status: %+v", resp.Msg)
	}
}

// TestSetupStatusCrossProjectIsolation verifies one project's profile and
// configuration never leak into another's checklist.
func TestSetupStatusCrossProjectIsolation(t *testing.T) {
	f := newProfileFixture(t)
	ctx := context.Background()
	other := f.seedProject("other")

	f.setProfile(f.project.ID, &adminv1.Profile{
		Platforms:    []adminv1.ProfilePlatform{adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB},
		GoogleSignIn: true,
	})

	// The sibling project still has no profile — and no checklist.
	resp, err := f.h.GetProjectSetupStatus(ctx, connect.NewRequest(
		&adminv1.GetProjectSetupStatusRequest{ProjectId: other.ID}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.HasProfile || len(resp.Msg.Items) != 0 {
		t.Fatalf("profile leaked to sibling: %+v", resp.Msg)
	}

	// Configuring Google on the sibling does not satisfy the first project's
	// google_credentials item.
	p, err := f.st.GetProject(ctx, other.ID)
	if err != nil {
		t.Fatal(err)
	}
	p.Settings.Google = store.GoogleProviderSettings{Enabled: true, WebClientID: "web.apps.googleusercontent.com"}
	p.UpdatedAt = f.now
	if err := f.st.UpdateProject(ctx, p); err != nil {
		t.Fatal(err)
	}
	ids := f.itemIDs(f.project.ID)
	found := false
	for _, id := range ids {
		if id == "google_credentials" {
			found = true
		}
	}
	if !found {
		t.Fatalf("sibling config satisfied this project's item: %v", ids)
	}
}
