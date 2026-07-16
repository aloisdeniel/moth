package entitlements

import (
	"testing"
	"time"

	"github.com/aloisdeniel/moth/internal/store"
)

var now = time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)

func ptr(t time.Time) *time.Time { return &t }

// catalog: entitlement "pro" (id e-pro), product "monthly" (id p-monthly)
// granting it.
func catalog() ([]store.Entitlement, []store.Product) {
	ents := []store.Entitlement{
		{ID: "e-pro", ProjectID: "prj", Identifier: "pro"},
		{ID: "e-plus", ProjectID: "prj", Identifier: "plus"},
	}
	products := []store.Product{
		{ID: "p-monthly", ProjectID: "prj", Identifier: "monthly", EntitlementIDs: []string{"e-pro"}},
		{ID: "p-bundle", ProjectID: "prj", Identifier: "bundle", EntitlementIDs: []string{"e-pro", "e-plus"}},
	}
	return ents, products
}

func sub(status string, end *time.Time) store.Subscription {
	return store.Subscription{ID: "s1", ProjectID: "prj", UserID: "u1", Store: store.SubscriptionStoreApple,
		ProductID: "p-monthly", Status: status, CurrentPeriodEnd: end}
}

func TestStatusGrantsAccessMatrix(t *testing.T) {
	granted := []string{
		store.SubscriptionStatusActive,
		store.SubscriptionStatusTrialing,
		store.SubscriptionStatusInGracePeriod,
		store.SubscriptionStatusInBillingRetry,
	}
	notGranted := []string{
		store.SubscriptionStatusPaused,
		store.SubscriptionStatusExpired,
		store.SubscriptionStatusRevoked,
		"", "garbage",
	}
	for _, s := range granted {
		if !StatusGrantsAccess(s) {
			t.Errorf("status %q should grant access", s)
		}
	}
	for _, s := range notGranted {
		if StatusGrantsAccess(s) {
			t.Errorf("status %q should NOT grant access", s)
		}
	}
}

func TestDeriveSubscriptionStatusCells(t *testing.T) {
	ents, products := catalog()
	end := now.Add(24 * time.Hour)
	cases := []struct {
		status  string
		granted bool
	}{
		{store.SubscriptionStatusActive, true},
		{store.SubscriptionStatusTrialing, true},
		{store.SubscriptionStatusInGracePeriod, true},
		{store.SubscriptionStatusInBillingRetry, true},
		{store.SubscriptionStatusPaused, false},
		{store.SubscriptionStatusExpired, false},
		{store.SubscriptionStatusRevoked, false},
	}
	for _, c := range cases {
		got := Derive(now, ents, products, []store.Subscription{sub(c.status, &end)}, nil)
		has := len(got) == 1 && got[0].Identifier == "pro"
		if has != c.granted {
			t.Errorf("status %q: granted=%v, want %v (got %+v)", c.status, has, c.granted, got)
		}
		if c.granted {
			if got[0].Source != SourceStore {
				t.Errorf("status %q: source=%q want store", c.status, got[0].Source)
			}
			if got[0].ProductIdentifier != "monthly" {
				t.Errorf("status %q: product=%q want monthly", c.status, got[0].ProductIdentifier)
			}
			if !got[0].ExpireTime.Equal(end) {
				t.Errorf("status %q: expire=%v want %v", c.status, got[0].ExpireTime, end)
			}
		}
	}
}

func TestDeriveNoneIsEmpty(t *testing.T) {
	ents, products := catalog()
	if got := Derive(now, ents, products, nil, nil); len(got) != 0 {
		t.Errorf("never-paid user should be none (empty), got %+v", got)
	}
}

func TestDeriveUnmappedSubscriptionGrantsNothing(t *testing.T) {
	ents, products := catalog()
	s := sub(store.SubscriptionStatusActive, ptr(now.Add(time.Hour)))
	s.ProductID = "" // unmapped SKU
	if got := Derive(now, ents, products, []store.Subscription{s}, nil); len(got) != 0 {
		t.Errorf("unmapped active subscription should grant nothing, got %+v", got)
	}
}

func TestDeriveMarksSandboxEntitlement(t *testing.T) {
	ents, products := catalog()
	end := now.Add(24 * time.Hour)

	prod := sub(store.SubscriptionStatusActive, &end)
	prod.Environment = store.SubscriptionEnvironmentProduction
	got := Derive(now, ents, products, []store.Subscription{prod}, nil)
	if len(got) != 1 || got[0].IsSandbox {
		t.Fatalf("production subscription must not be flagged sandbox: %+v", got)
	}

	sand := sub(store.SubscriptionStatusActive, &end)
	sand.Environment = store.SubscriptionEnvironmentSandbox
	got = Derive(now, ents, products, []store.Subscription{sand}, nil)
	if len(got) != 1 || !got[0].IsSandbox {
		t.Fatalf("sandbox subscription must be flagged sandbox: %+v", got)
	}

	// Operator grants are never sandbox.
	future := now.Add(48 * time.Hour)
	got = Derive(now, ents, products, nil, []store.SubscriptionGrant{{EntitlementID: "e-pro", ExpiresAt: &future}})
	if len(got) != 1 || got[0].IsSandbox {
		t.Fatalf("grant must not be flagged sandbox: %+v", got)
	}
}

func TestDeriveGrantCells(t *testing.T) {
	ents, products := catalog()
	future := now.Add(48 * time.Hour)
	past := now.Add(-time.Hour)
	cases := []struct {
		name    string
		grant   store.SubscriptionGrant
		granted bool
	}{
		{"active dated", store.SubscriptionGrant{EntitlementID: "e-pro", ExpiresAt: &future}, true},
		{"active unbounded", store.SubscriptionGrant{EntitlementID: "e-pro"}, true},
		{"expired", store.SubscriptionGrant{EntitlementID: "e-pro", ExpiresAt: &past}, false},
		{"expiry now (inclusive lapse)", store.SubscriptionGrant{EntitlementID: "e-pro", ExpiresAt: &now}, false},
		{"revoked", store.SubscriptionGrant{EntitlementID: "e-pro", ExpiresAt: &future, RevokedAt: &past}, false},
	}
	for _, c := range cases {
		got := Derive(now, ents, products, nil, []store.SubscriptionGrant{c.grant})
		has := len(got) == 1 && got[0].Identifier == "pro" && got[0].Source == SourceGrant
		if has != c.granted {
			t.Errorf("%s: granted=%v want %v (got %+v)", c.name, has, c.granted, got)
		}
	}
}

func TestDeriveGrantAndSubscriptionMergeKeepsMostGenerous(t *testing.T) {
	ents, products := catalog()
	subEnd := now.Add(24 * time.Hour)
	grantEnd := now.Add(72 * time.Hour)
	// Store subscription grants "pro" until +24h; an operator grant extends it
	// to +72h. The union keeps the later expiry.
	got := Derive(now, ents, products,
		[]store.Subscription{sub(store.SubscriptionStatusActive, &subEnd)},
		[]store.SubscriptionGrant{{EntitlementID: "e-pro", ExpiresAt: &grantEnd}})
	if len(got) != 1 {
		t.Fatalf("want single merged entitlement, got %+v", got)
	}
	if !got[0].ExpireTime.Equal(grantEnd) {
		t.Errorf("merge kept expiry %v, want the later %v", got[0].ExpireTime, grantEnd)
	}
}

func TestDeriveUnboundedGrantBeatsDatedSubscription(t *testing.T) {
	ents, products := catalog()
	subEnd := now.Add(24 * time.Hour)
	got := Derive(now, ents, products,
		[]store.Subscription{sub(store.SubscriptionStatusActive, &subEnd)},
		[]store.SubscriptionGrant{{EntitlementID: "e-pro"}}) // unbounded
	if len(got) != 1 || !got[0].ExpireTime.IsZero() {
		t.Errorf("unbounded grant should win, got %+v", got)
	}
}

func TestDeriveMultipleEntitlementsFromBundle(t *testing.T) {
	ents, products := catalog()
	end := now.Add(24 * time.Hour)
	s := sub(store.SubscriptionStatusActive, &end)
	s.ProductID = "p-bundle"
	got := Derive(now, ents, products, []store.Subscription{s}, nil)
	if len(got) != 2 || got[0].Identifier != "plus" || got[1].Identifier != "pro" {
		t.Errorf("bundle should grant plus+pro sorted, got %+v", got)
	}
}
