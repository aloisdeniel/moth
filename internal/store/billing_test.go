package store

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

// billingFixture creates a project and one user, returning their ids.
func billingFixture(t *testing.T, s *Store) (projectID, userID string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	p, k := testProject("bp1", "billing-app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	u := User{ID: "u1", ProjectID: p.ID, Email: "user@example.com", CreatedAt: now, UpdatedAt: now}
	if err := s.CreateUser(ctx, u); err != nil {
		t.Fatal(err)
	}
	return p.ID, u.ID
}

func TestEntitlementCRUD(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	pid, _ := billingFixture(t, s)
	now := time.Now()

	if _, err := s.GetEntitlement(ctx, pid, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing entitlement: want ErrNotFound, got %v", err)
	}

	e := Entitlement{ID: "e1", ProjectID: pid, Identifier: "pro", DisplayName: "Pro", CreatedAt: now, UpdatedAt: now}
	if err := s.CreateEntitlement(ctx, e); err != nil {
		t.Fatal(err)
	}
	// Unique (project_id, identifier) surfaces as ErrConflict so the admin
	// handler can map it to CodeAlreadyExists.
	dup := Entitlement{ID: "e2", ProjectID: pid, Identifier: "pro", CreatedAt: now, UpdatedAt: now}
	if err := s.CreateEntitlement(ctx, dup); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate identifier: want ErrConflict, got %v", err)
	}

	got, err := s.GetEntitlement(ctx, pid, "e1")
	if err != nil || got.Identifier != "pro" || got.DisplayName != "Pro" {
		t.Fatalf("get mismatch: %+v (%v)", got, err)
	}
	byID, err := s.GetEntitlementByIdentifier(ctx, pid, "pro")
	if err != nil || byID.ID != "e1" {
		t.Fatalf("get by identifier mismatch: %+v (%v)", byID, err)
	}

	got.DisplayName = "Pro Plan"
	got.UpdatedAt = now.Add(time.Second)
	if err := s.UpdateEntitlement(ctx, got); err != nil {
		t.Fatal(err)
	}
	if e2, _ := s.GetEntitlement(ctx, pid, "e1"); e2.DisplayName != "Pro Plan" {
		t.Fatalf("update not applied: %+v", e2)
	}

	list, err := s.ListEntitlements(ctx, pid)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: want 1, got %d (%v)", len(list), err)
	}

	if err := s.DeleteEntitlement(ctx, pid, "e1"); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteEntitlement(ctx, pid, "e1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("double delete: want ErrNotFound, got %v", err)
	}
}

func TestProductCRUDWithEntitlements(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	pid, _ := billingFixture(t, s)
	now := time.Now()

	for _, id := range []string{"pro", "premium"} {
		if err := s.CreateEntitlement(ctx, Entitlement{ID: "ent-" + id, ProjectID: pid, Identifier: id, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}

	p := Product{
		ID: "prod1", ProjectID: pid, Identifier: "monthly", DisplayName: "Monthly",
		AppleProductID: "com.app.monthly", GoogleProductID: "",
		BillingPeriod: "monthly", PriceAmountMicros: 4990000, Currency: "USD",
		TrialPeriod: "P1W", Offering: "default", SortOrder: 1,
		EntitlementIDs: []string{"ent-pro"}, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.CreateProduct(ctx, p); err != nil {
		t.Fatal(err)
	}
	// Duplicate (project_id, identifier) surfaces as ErrConflict.
	dupProd := p
	dupProd.ID = "prod-dup"
	dupProd.AppleProductID = "com.app.monthly.other"
	if err := s.CreateProduct(ctx, dupProd); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate product identifier: want ErrConflict, got %v", err)
	}

	got, err := s.GetProduct(ctx, pid, "prod1")
	if err != nil {
		t.Fatal(err)
	}
	if got.AppleProductID != "com.app.monthly" || got.GoogleProductID != "" {
		t.Fatalf("nullable store ids not round-tripped: %+v", got)
	}
	if got.PriceAmountMicros != 4990000 || got.Currency != "USD" {
		t.Fatalf("price metadata mismatch: %+v", got)
	}
	if len(got.EntitlementIDs) != 1 || got.EntitlementIDs[0] != "ent-pro" {
		t.Fatalf("entitlement grants mismatch: %+v", got.EntitlementIDs)
	}

	// Update replaces the grant set.
	got.EntitlementIDs = []string{"ent-pro", "ent-premium"}
	got.DisplayName = "Monthly Pro"
	got.UpdatedAt = now.Add(time.Second)
	if err := s.UpdateProduct(ctx, got); err != nil {
		t.Fatal(err)
	}
	got2, _ := s.GetProduct(ctx, pid, "prod1")
	if got2.DisplayName != "Monthly Pro" || len(got2.EntitlementIDs) != 2 {
		t.Fatalf("update not applied: %+v", got2)
	}

	// Deleting an entitlement cascades the join row.
	if err := s.DeleteEntitlement(ctx, pid, "ent-premium"); err != nil {
		t.Fatal(err)
	}
	got3, _ := s.GetProduct(ctx, pid, "prod1")
	if len(got3.EntitlementIDs) != 1 || got3.EntitlementIDs[0] != "ent-pro" {
		t.Fatalf("entitlement cascade not reflected: %+v", got3.EntitlementIDs)
	}

	list, err := s.ListProducts(ctx, pid)
	if err != nil || len(list) != 1 {
		t.Fatalf("list products: want 1, got %d (%v)", len(list), err)
	}

	if err := s.DeleteProduct(ctx, pid, "prod1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetProduct(ctx, pid, "prod1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted product: want ErrNotFound, got %v", err)
	}
}

func TestSubscriptionUpsertByStoreIdentity(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	pid, uid := billingFixture(t, s)
	now := time.Now()
	end := now.Add(30 * 24 * time.Hour)

	sub := Subscription{
		ID: "sub1", ProjectID: pid, UserID: uid, Store: SubscriptionStoreApple,
		StoreTransactionID: "otx-123", Status: SubscriptionStatusActive,
		CurrentPeriodEnd: &end, AutoRenew: true, Environment: SubscriptionEnvironmentProduction,
		RawState: `{"a":1}`, CreatedAt: now, UpdatedAt: now,
	}
	stored, err := s.UpsertSubscription(ctx, sub)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ID != "sub1" || stored.CurrentPeriodEnd == nil || !stored.AutoRenew {
		t.Fatalf("insert mismatch: %+v", stored)
	}

	// Second upsert on the same store identity keeps id/created_at, updates
	// the mutable fields.
	sub.ID = "ignored-id"
	sub.Status = SubscriptionStatusExpired
	sub.AutoRenew = false
	sub.UpdatedAt = now.Add(time.Hour)
	stored2, err := s.UpsertSubscription(ctx, sub)
	if err != nil {
		t.Fatal(err)
	}
	if stored2.ID != "sub1" {
		t.Fatalf("upsert must preserve id, got %q", stored2.ID)
	}
	if stored2.Status != SubscriptionStatusExpired || stored2.AutoRenew {
		t.Fatalf("upsert did not update mutable fields: %+v", stored2)
	}

	byStore, err := s.GetSubscriptionByStoreID(ctx, pid, SubscriptionStoreApple, "otx-123")
	if err != nil || byStore.ID != "sub1" {
		t.Fatalf("get by store id mismatch: %+v (%v)", byStore, err)
	}

	list, err := s.ListUserSubscriptions(ctx, pid, uid)
	if err != nil || len(list) != 1 {
		t.Fatalf("list user subs: want 1, got %d (%v)", len(list), err)
	}
	if _, err := s.GetSubscriptionByStoreID(ctx, pid, SubscriptionStoreGoogle, "otx-123"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong store: want ErrNotFound, got %v", err)
	}
}

func TestListSubscriptionsForReconciliation(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	pid, uid := billingFixture(t, s)
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	mk := func(id, txn, status string, end *time.Time) {
		if _, err := s.UpsertSubscription(ctx, Subscription{
			ID: id, ProjectID: pid, UserID: uid, Store: SubscriptionStoreApple,
			StoreTransactionID: txn, Status: status, CurrentPeriodEnd: end,
			Environment: SubscriptionEnvironmentProduction, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	mk("s-lapsed", "t1", SubscriptionStatusActive, &past)   // due: active, period ended
	mk("s-future", "t2", SubscriptionStatusActive, &future) // not due: period not ended
	mk("s-expired", "t3", SubscriptionStatusExpired, &past) // not due: already expired
	mk("s-nilend", "t4", SubscriptionStatusActive, nil)     // not due: no period end

	got, err := s.ListSubscriptionsForReconciliation(ctx, now, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "s-lapsed" {
		t.Fatalf("want only s-lapsed due for reconciliation, got %+v", got)
	}
}

func TestSubscriptionGrants(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	pid, uid := billingFixture(t, s)
	now := time.Now()
	if err := s.CreateEntitlement(ctx, Entitlement{ID: "e1", ProjectID: pid, Identifier: "pro", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}

	future := now.Add(24 * time.Hour)
	past := now.Add(-time.Hour)
	grants := []SubscriptionGrant{
		{ID: "g-active", ProjectID: pid, UserID: uid, EntitlementID: "e1", ExpiresAt: &future, Reason: "comp", GrantedBy: "ops@x", CreatedAt: now},
		{ID: "g-forever", ProjectID: pid, UserID: uid, EntitlementID: "e1", Reason: "vip", CreatedAt: now},
		{ID: "g-expired", ProjectID: pid, UserID: uid, EntitlementID: "e1", ExpiresAt: &past, CreatedAt: now},
	}
	for _, g := range grants {
		if err := s.CreateSubscriptionGrant(ctx, g); err != nil {
			t.Fatal(err)
		}
	}

	all, err := s.ListUserGrants(ctx, pid, uid)
	if err != nil || len(all) != 3 {
		t.Fatalf("list all grants: want 3, got %d (%v)", len(all), err)
	}
	active, err := s.ListActiveUserGrants(ctx, pid, uid, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 2 {
		t.Fatalf("active grants: want 2 (active+forever), got %d", len(active))
	}

	// Revoking removes it from the active set.
	if err := s.RevokeSubscriptionGrant(ctx, pid, "g-active", now); err != nil {
		t.Fatal(err)
	}
	active, _ = s.ListActiveUserGrants(ctx, pid, uid, now)
	if len(active) != 1 || active[0].ID != "g-forever" {
		t.Fatalf("after revoke: want only g-forever, got %+v", active)
	}
	// Double revoke is ErrNotFound.
	if err := s.RevokeSubscriptionGrant(ctx, pid, "g-active", now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("double revoke: want ErrNotFound, got %v", err)
	}
	if g, _ := s.GetSubscriptionGrant(ctx, pid, "g-active"); g.RevokedAt == nil {
		t.Fatalf("revoked_at not set: %+v", g)
	}
}

func TestStoreNotificationIdempotency(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	pid, _ := billingFixture(t, s)
	now := time.Now()

	n := StoreNotification{
		ID: "n1", ProjectID: pid, Store: SubscriptionStoreApple, NotificationID: "apple-uuid-1",
		Type: "DID_RENEW", Subtype: "", RawPayload: `{"x":1}`, CreatedAt: now,
	}
	isNew, err := s.InsertStoreNotificationIfNew(ctx, n)
	if err != nil || !isNew {
		t.Fatalf("first insert should be new: %v (%v)", isNew, err)
	}
	// Replay with a different row id but same store notification id → deduped.
	n.ID = "n2"
	isNew, err = s.InsertStoreNotificationIfNew(ctx, n)
	if err != nil {
		t.Fatal(err)
	}
	if isNew {
		t.Fatal("replay should be deduped (not new)")
	}
	if err := s.MarkStoreNotificationProcessed(ctx, pid, "n1", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkStoreNotificationProcessed(ctx, pid, "missing", now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("mark missing: want ErrNotFound, got %v", err)
	}
}

func TestBillingCredentialsWriteOnlySecrets(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	pid, _ := billingFixture(t, s)
	now := time.Now()

	if _, err := s.GetBillingCredentials(ctx, pid); !errors.Is(err, ErrNotFound) {
		t.Fatalf("no creds: want ErrNotFound, got %v", err)
	}

	p8 := []byte{0x01, 0x02, 0x03}
	c := BillingCredentials{
		ProjectID: pid, AppleIAPKeyID: "KID", AppleIAPIssuerID: "ISS", AppleIAPKeyEnc: p8,
		AppleBundleID: "com.app", GooglePackageName: "com.app", CreatedAt: now, UpdatedAt: now,
	}
	if err := s.UpsertBillingCredentials(ctx, c); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetBillingCredentials(ctx, pid)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.AppleIAPKeyEnc, p8) || got.AppleIAPKeyID != "KID" {
		t.Fatalf("apple creds mismatch: %+v", got)
	}

	// Upsert with nil secret keeps the stored ciphertext; non-secret fields
	// still update.
	c.AppleIAPKeyEnc = nil
	c.AppleBundleID = "com.app.new"
	c.UpdatedAt = now.Add(time.Second)
	if err := s.UpsertBillingCredentials(ctx, c); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetBillingCredentials(ctx, pid)
	if !bytes.Equal(got.AppleIAPKeyEnc, p8) {
		t.Fatalf("nil secret should keep stored value, got %v", got.AppleIAPKeyEnc)
	}
	if got.AppleBundleID != "com.app.new" {
		t.Fatalf("non-secret field should update: %+v", got)
	}

	// Empty (non-nil) slice clears the secret.
	c.AppleIAPKeyEnc = []byte{}
	c.UpdatedAt = now.Add(2 * time.Second)
	if err := s.UpsertBillingCredentials(ctx, c); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetBillingCredentials(ctx, pid)
	if len(got.AppleIAPKeyEnc) != 0 {
		t.Fatalf("empty slice should clear secret, got %v", got.AppleIAPKeyEnc)
	}
}

func TestSubscriptionEventsBatch(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	pid, uid := billingFixture(t, s)
	now := time.Now()

	if err := s.InsertSubscriptionEvents(ctx, nil); err != nil {
		t.Fatalf("empty batch should be a no-op: %v", err)
	}
	events := []SubscriptionEvent{
		{ID: "se1", ProjectID: pid, Type: SubscriptionEventPurchased, UserID: uid, ProductID: "prod1", Store: SubscriptionStoreApple, PriceAmountMicros: 4990000, Currency: "USD", Environment: SubscriptionEnvironmentProduction, CreatedAt: now},
		{ID: "se2", ProjectID: pid, Type: SubscriptionEventRenewed, UserID: uid, Store: SubscriptionStoreApple, CreatedAt: now.Add(time.Second)},
	}
	if err := s.InsertSubscriptionEvents(ctx, events); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM subscription_events WHERE project_id = ?`, pid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("want 2 events, got %d", n)
	}
}
