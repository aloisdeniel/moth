package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

// catalogFixture creates a project with two products and returns their ids.
func catalogFixture(t *testing.T, s *Store) (projectID string, monthlyID, yearlyID string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	pid, _ := billingFixture(t, s)
	monthly := Product{ID: "pm", ProjectID: pid, Identifier: "monthly", Offering: DefaultOffering,
		SortOrder: 0, CreatedAt: now, UpdatedAt: now}
	yearly := Product{ID: "py", ProjectID: pid, Identifier: "yearly", Offering: DefaultOffering,
		SortOrder: 1, CreatedAt: now, UpdatedAt: now}
	if err := s.CreateProduct(ctx, monthly); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateProduct(ctx, yearly); err != nil {
		t.Fatal(err)
	}
	return pid, monthly.ID, yearly.ID
}

func TestProductStoreSyncUpsert(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	pid, monthlyID, _ := catalogFixture(t, s)

	if _, err := s.GetProductStoreSync(ctx, monthlyID, SubscriptionStoreApple); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing sync: want ErrNotFound, got %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	rec := ProductStoreSync{
		ProductID: monthlyID, Store: SubscriptionStoreApple, Status: ProductSyncInSync,
		StoreProductID: "app.monthly", Revision: "r1", SyncedAt: &now,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.UpsertProductStoreSync(ctx, rec); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetProductStoreSync(ctx, monthlyID, SubscriptionStoreApple)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ProductSyncInSync || got.StoreProductID != "app.monthly" || got.Revision != "r1" {
		t.Fatalf("upsert mismatch: %+v", got)
	}
	if got.SyncedAt == nil || !got.SyncedAt.Equal(now) {
		t.Fatalf("synced_at mismatch: %v want %v", got.SyncedAt, now)
	}

	// Second upsert overwrites mutable fields, preserves created_at.
	later := now.Add(time.Hour)
	rec.Status = ProductSyncError
	rec.Error = "boom"
	rec.SyncedAt = nil
	rec.Revision = "r2"
	rec.UpdatedAt = later
	rec.CreatedAt = later // ignored on conflict
	if err := s.UpsertProductStoreSync(ctx, rec); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetProductStoreSync(ctx, monthlyID, SubscriptionStoreApple)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ProductSyncError || got.Error != "boom" || got.Revision != "r2" {
		t.Fatalf("second upsert mismatch: %+v", got)
	}
	if got.SyncedAt != nil {
		t.Fatalf("synced_at should clear to nil, got %v", got.SyncedAt)
	}
	if !got.CreatedAt.Equal(now) {
		t.Fatalf("created_at should be preserved, got %v want %v", got.CreatedAt, now)
	}
	_ = pid
}

func TestListProductStoreSyncsFiltersByStore(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	pid, monthlyID, yearlyID := catalogFixture(t, s)
	now := time.Now().UTC()

	must := func(r ProductStoreSync) {
		t.Helper()
		if err := s.UpsertProductStoreSync(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	must(ProductStoreSync{ProductID: monthlyID, Store: SubscriptionStoreApple, Status: ProductSyncInSync, CreatedAt: now, UpdatedAt: now})
	must(ProductStoreSync{ProductID: monthlyID, Store: SubscriptionStoreGoogle, Status: ProductSyncDrift, CreatedAt: now, UpdatedAt: now})
	must(ProductStoreSync{ProductID: yearlyID, Store: SubscriptionStoreApple, Status: ProductSyncPending, CreatedAt: now, UpdatedAt: now})

	all, err := s.ListProductStoreSyncs(ctx, pid, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("all stores: want 3, got %d", len(all))
	}
	apple, err := s.ListProductStoreSyncs(ctx, pid, SubscriptionStoreApple)
	if err != nil {
		t.Fatal(err)
	}
	if len(apple) != 2 {
		t.Fatalf("apple: want 2, got %d", len(apple))
	}
	for _, r := range apple {
		if r.Store != SubscriptionStoreApple {
			t.Fatalf("filter leaked store %q", r.Store)
		}
	}
}

func TestListProductStoreSyncsScopedToProject(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	pid, monthlyID, _ := catalogFixture(t, s)
	now := time.Now().UTC()

	// A second project must not see the first's sync rows.
	p2, k2 := testProject("bp2", "other-app")
	if err := s.CreateProject(ctx, p2, k2); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertProductStoreSync(ctx, ProductStoreSync{
		ProductID: monthlyID, Store: SubscriptionStoreApple, Status: ProductSyncInSync,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListProductStoreSyncs(ctx, p2.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("cross-project leak: got %d rows", len(got))
	}
	_ = pid
}

func TestSetProductSortOrders(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	pid, monthlyID, yearlyID := catalogFixture(t, s)

	// Reorder: yearly first, monthly second.
	if err := s.SetProductSortOrders(ctx, pid, map[string]int{yearlyID: 0, monthlyID: 1}); err != nil {
		t.Fatal(err)
	}
	products, err := s.ListProducts(ctx, pid)
	if err != nil {
		t.Fatal(err)
	}
	// ListProducts orders by (sort_order, id).
	if products[0].ID != yearlyID || products[1].ID != monthlyID {
		t.Fatalf("reorder failed: %s then %s", products[0].Identifier, products[1].Identifier)
	}

	// An unknown id fails the whole call with ErrNotFound and rolls back.
	err = s.SetProductSortOrders(ctx, pid, map[string]int{monthlyID: 5, "nope": 6})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown id: want ErrNotFound, got %v", err)
	}
	products, err = s.ListProducts(ctx, pid)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range products {
		if p.ID == monthlyID && p.SortOrder == 5 {
			t.Fatal("rollback failed: monthly sort_order was committed despite the error")
		}
	}
}

func TestProductStoreSyncCascadesWithProduct(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	pid, monthlyID, _ := catalogFixture(t, s)
	now := time.Now().UTC()
	if err := s.UpsertProductStoreSync(ctx, ProductStoreSync{
		ProductID: monthlyID, Store: SubscriptionStoreApple, Status: ProductSyncInSync,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteProduct(ctx, pid, monthlyID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetProductStoreSync(ctx, monthlyID, SubscriptionStoreApple); !errors.Is(err, ErrNotFound) {
		t.Fatalf("sync row should cascade-delete with the product, got %v", err)
	}
}
