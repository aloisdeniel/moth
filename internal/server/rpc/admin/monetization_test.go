package adminrpc

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/setup"
	"github.com/aloisdeniel/moth/internal/store"
)

// fakeSyncer is a no-network storeSyncer double.
type fakeSyncer struct {
	result *setup.SyncResult
	err    error
	called bool
}

func (f *fakeSyncer) Sync(_ context.Context, _, _, _ string, _ store.BillingCredentials, _ setup.DesiredCatalog) (*setup.SyncResult, error) {
	f.called = true
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func newMonetizationTestHandler(t *testing.T, syncer storeSyncer) (*MonetizationHandler, *store.Store, keys.MasterKey, store.Project) {
	t.Helper()
	// Reuse the billing harness for store/project/master setup.
	_, st, master, project := newBillingTestHandler(t)
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	h := NewMonetizationHandler(st, master, "https://moth.example", nil, syncer, func() time.Time { return now })
	return h, st, master, project
}

func makeProduct(t *testing.T, st *store.Store, projectID, id, offering string, sort int, googleSKU string) store.Product {
	t.Helper()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	p := store.Product{
		ID: NewID(), ProjectID: projectID, Identifier: id, DisplayName: id,
		GoogleProductID: googleSKU, BillingPeriod: "monthly", PriceAmountMicros: 9_990_000,
		Currency: "USD", Offering: offering, SortOrder: sort, CreatedAt: now, UpdatedAt: now,
	}
	if err := st.CreateProduct(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestGetOfferingOrdersBySortOrder(t *testing.T) {
	h, st, _, project := newMonetizationTestHandler(t, nil)
	ctx := context.Background()
	p1 := makeProduct(t, st, project.ID, "monthly", "", 1, "sku.monthly")
	p0 := makeProduct(t, st, project.ID, "yearly", "", 0, "sku.yearly")
	makeProduct(t, st, project.ID, "special", "promo", 0, "sku.special") // other offering

	resp, err := h.GetOffering(ctx, connect.NewRequest(&adminv1.GetOfferingRequest{ProjectId: project.ID}))
	if err != nil {
		t.Fatal(err)
	}
	off := resp.Msg.Offering
	if !off.IsDefault || off.Identifier != store.DefaultOffering {
		t.Fatalf("expected default offering, got %q default=%v", off.Identifier, off.IsDefault)
	}
	if len(off.ProductIds) != 2 {
		t.Fatalf("expected 2 members, got %d", len(off.ProductIds))
	}
	if off.ProductIds[0] != p0.ID || off.ProductIds[1] != p1.ID {
		t.Fatalf("wrong order: %v (want %s,%s)", off.ProductIds, p0.ID, p1.ID)
	}
}

func TestReorderOffering(t *testing.T) {
	h, st, _, project := newMonetizationTestHandler(t, nil)
	ctx := context.Background()
	p0 := makeProduct(t, st, project.ID, "monthly", "", 0, "sku.monthly")
	p1 := makeProduct(t, st, project.ID, "yearly", "", 1, "sku.yearly")

	// Reorder: yearly first.
	resp, err := h.ReorderOffering(ctx, connect.NewRequest(&adminv1.ReorderOfferingRequest{
		ProjectId: project.ID, ProductIds: []string{p1.ID, p0.ID}}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.Offering.ProductIds[0] != p1.ID {
		t.Fatalf("reorder did not stick: %v", resp.Msg.Offering.ProductIds)
	}

	// A stranger id is rejected.
	_, err = h.ReorderOffering(ctx, connect.NewRequest(&adminv1.ReorderOfferingRequest{
		ProjectId: project.ID, ProductIds: []string{p1.ID}}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument for incomplete set, got %v", err)
	}
}

func TestGetStoreCatalogStatusNoCredentials(t *testing.T) {
	h, st, _, project := newMonetizationTestHandler(t, nil)
	ctx := context.Background()
	makeProduct(t, st, project.ID, "monthly", "", 0, "sku.monthly") // google-mapped
	makeProduct(t, st, project.ID, "apponly", "", 1, "")            // unmapped everywhere

	resp, err := h.GetStoreCatalogStatus(ctx, connect.NewRequest(&adminv1.GetStoreCatalogStatusRequest{ProjectId: project.ID}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.Stores) != 3 {
		t.Fatalf("expected 3 stores (apple, google, stripe), got %d", len(resp.Msg.Stores))
	}
	for _, s := range resp.Msg.Stores {
		if s.CredentialsPresent || s.NotificationsWired {
			t.Fatalf("store %v should report no creds/notifications", s.Store)
		}
		if s.ProductsTotal != 2 {
			t.Fatalf("expected total 2, got %d", s.ProductsTotal)
		}
	}
}

func TestSyncStoreCatalogDryRunNoCredentials(t *testing.T) {
	h, st, _, project := newMonetizationTestHandler(t, nil)
	ctx := context.Background()
	makeProduct(t, st, project.ID, "monthly", "", 0, "sku.monthly")

	resp, err := h.SyncStoreCatalog(ctx, connect.NewRequest(&adminv1.SyncStoreCatalogRequest{
		ProjectId: project.ID, Store: adminv1.Store_STORE_GOOGLE, DryRun: true}))
	if err != nil {
		t.Fatalf("dry-run must not error: %v", err)
	}
	if len(resp.Msg.Items) != 1 || resp.Msg.Items[0].Action != adminv1.SyncAction_SYNC_ACTION_CREATE {
		t.Fatalf("expected one CREATE item, got %+v", resp.Msg.Items)
	}
	if len(resp.Msg.GuidedSteps) == 0 {
		t.Fatal("expected guided steps for an unconfigured store")
	}

	// Unspecified store is rejected.
	_, err = h.SyncStoreCatalog(ctx, connect.NewRequest(&adminv1.SyncStoreCatalogRequest{
		ProjectId: project.ID, Store: adminv1.Store_STORE_UNSPECIFIED, DryRun: true}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument for unspecified store, got %v", err)
	}
}

func TestSyncStoreCatalogApplyPersistsAndVerifies(t *testing.T) {
	syncer := &fakeSyncer{result: &setup.SyncResult{
		Store:    store.SubscriptionStoreGoogle,
		Products: []setup.ProductResult{{ProductID: "sku.monthly", StoreID: "sku.monthly", Action: setup.ActionCreated}},
	}}
	h, st, master, project := newMonetizationTestHandler(t, syncer)
	ctx := context.Background()
	makeProduct(t, st, project.ID, "monthly", "", 0, "sku.monthly")

	// Configure Google credentials so credentialsPresent() is true.
	saEnc, err := master.Encrypt([]byte("{}"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertBillingCredentials(ctx, store.BillingCredentials{
		ProjectID: project.ID, GoogleServiceAccountEnc: saEnc, GooglePackageName: "com.demo",
		GooglePubsubTopic: "projects/p/topics/moth", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	resp, err := h.SyncStoreCatalog(ctx, connect.NewRequest(&adminv1.SyncStoreCatalogRequest{
		ProjectId: project.ID, Store: adminv1.Store_STORE_GOOGLE, DryRun: false}))
	if err != nil {
		t.Fatal(err)
	}
	if !syncer.called {
		t.Fatal("expected live syncer to be called on apply with creds present")
	}
	if len(resp.Msg.Items) != 1 || resp.Msg.Items[0].Status != adminv1.ProductSyncStatus_PRODUCT_SYNC_STATUS_IN_SYNC {
		t.Fatalf("expected one in-sync item, got %+v", resp.Msg.Items)
	}

	// Status now reports the product in sync.
	stat, err := h.GetStoreCatalogStatus(ctx, connect.NewRequest(&adminv1.GetStoreCatalogStatusRequest{ProjectId: project.ID}))
	if err != nil {
		t.Fatal(err)
	}
	var google *adminv1.StoreCatalogStatus
	for _, s := range stat.Msg.Stores {
		if s.Store == adminv1.Store_STORE_GOOGLE {
			google = s
		}
	}
	if google == nil || google.ProductsInSync != 1 {
		t.Fatalf("expected 1 in-sync google product, got %+v", google)
	}
	if !google.CredentialsPresent || !google.NotificationsWired {
		t.Fatal("expected google creds + notifications wired")
	}
}

// TestSyncStoreCatalogAppleGuidedNotInSync guards the honest-reporting fix: the
// Apple live path is guided-only (moth never holds the ASC catalog key), so the
// syncer returns zero product confirmations and only ManualSteps. Reporting
// InSync here would falsely claim the catalog was pushed/verified.
func TestSyncStoreCatalogAppleGuidedNotInSync(t *testing.T) {
	syncer := &fakeSyncer{result: &setup.SyncResult{
		Store: store.SubscriptionStoreApple,
		ManualSteps: []setup.ManualStep{{
			Title: "Push the Apple catalog with the CLI", Reason: "moth never holds the ASC key",
		}},
	}}
	h, st, master, project := newMonetizationTestHandler(t, syncer)
	ctx := context.Background()
	// An Apple-mapped product.
	p := makeProduct(t, st, project.ID, "monthly", "", 0, "")
	p.AppleProductID = "pro_monthly"
	if err := st.UpdateProduct(ctx, p); err != nil {
		t.Fatal(err)
	}
	// Apple credentials present so the live syncer is invoked.
	keyEnc, err := master.Encrypt([]byte("p8"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertBillingCredentials(ctx, store.BillingCredentials{
		ProjectID: project.ID, AppleIAPKeyEnc: keyEnc, AppleBundleID: "com.demo",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	resp, err := h.SyncStoreCatalog(ctx, connect.NewRequest(&adminv1.SyncStoreCatalogRequest{
		ProjectId: project.ID, Store: adminv1.Store_STORE_APPLE, DryRun: false}))
	if err != nil {
		t.Fatal(err)
	}
	if !syncer.called {
		t.Fatal("expected the live syncer to run with Apple creds present")
	}
	if resp.Msg.InSync {
		t.Fatal("Apple guided-only apply must NOT report in_sync (nothing was pushed/verified)")
	}
	if len(resp.Msg.Items) != 0 {
		t.Fatalf("expected no per-product confirmations, got %+v", resp.Msg.Items)
	}
	if len(resp.Msg.GuidedSteps) == 0 {
		t.Fatal("expected guided steps from the Apple manual path")
	}
}

// TestStoreCatalogStatusDetectsMothSideDrift proves the status/plan surfaces show
// a live moth-side diff: after a product is pushed (in_sync), editing its price
// in moth surfaces as DRIFT (and an UPDATE plan), not stale parity.
func TestStoreCatalogStatusDetectsMothSideDrift(t *testing.T) {
	syncer := &fakeSyncer{result: &setup.SyncResult{
		Store:    store.SubscriptionStoreGoogle,
		Products: []setup.ProductResult{{ProductID: "sku.monthly", StoreID: "sku.monthly", Action: setup.ActionCreated}},
	}}
	h, st, master, project := newMonetizationTestHandler(t, syncer)
	ctx := context.Background()
	p := makeProduct(t, st, project.ID, "monthly", "", 0, "sku.monthly")

	saEnc, err := master.Encrypt([]byte("{}"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertBillingCredentials(ctx, store.BillingCredentials{
		ProjectID: project.ID, GoogleServiceAccountEnc: saEnc, GooglePackageName: "com.demo",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	// Push: records the sync row in_sync with a revision.
	if _, err := h.SyncStoreCatalog(ctx, connect.NewRequest(&adminv1.SyncStoreCatalogRequest{
		ProjectId: project.ID, Store: adminv1.Store_STORE_GOOGLE, DryRun: false})); err != nil {
		t.Fatal(err)
	}

	// Change the price in moth (no re-push).
	p.PriceAmountMicros = 4_990_000
	if err := st.UpdateProduct(ctx, p); err != nil {
		t.Fatal(err)
	}

	// Status now reports drift, not in_sync.
	stat, err := h.GetStoreCatalogStatus(ctx, connect.NewRequest(&adminv1.GetStoreCatalogStatusRequest{ProjectId: project.ID}))
	if err != nil {
		t.Fatal(err)
	}
	var google *adminv1.StoreCatalogStatus
	for _, s := range stat.Msg.Stores {
		if s.Store == adminv1.Store_STORE_GOOGLE {
			google = s
		}
	}
	if google == nil || google.ProductsDrift != 1 || google.ProductsInSync != 0 {
		t.Fatalf("expected 1 drift / 0 in-sync after a moth-side price change, got %+v", google)
	}
	if len(google.Products) != 1 || google.Products[0].Status != adminv1.ProductSyncStatus_PRODUCT_SYNC_STATUS_DRIFT {
		t.Fatalf("expected product status DRIFT, got %+v", google.Products)
	}

	// Dry-run plan reflects the drift as an UPDATE and is not in sync.
	plan, err := h.SyncStoreCatalog(ctx, connect.NewRequest(&adminv1.SyncStoreCatalogRequest{
		ProjectId: project.ID, Store: adminv1.Store_STORE_GOOGLE, DryRun: true}))
	if err != nil {
		t.Fatal(err)
	}
	if plan.Msg.InSync {
		t.Fatal("plan must report drift (not in sync) after a moth-side price change")
	}
	if len(plan.Msg.Items) != 1 || plan.Msg.Items[0].Action != adminv1.SyncAction_SYNC_ACTION_UPDATE {
		t.Fatalf("expected one UPDATE item, got %+v", plan.Msg.Items)
	}
}
