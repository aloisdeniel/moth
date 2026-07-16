package serverapi

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"

	serverv1 "github.com/aloisdeniel/moth/gen/moth/server/v1"
	"github.com/aloisdeniel/moth/internal/keys"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

func TestGetUserEntitlementsMatchesDerivedSet(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	master, err := keys.LoadOrCreateMasterKey(t.TempDir(), func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	project := store.Project{ID: authrpc.NewID(), Name: "Demo", Slug: "demo",
		PublishableKey: "pk_" + authrpc.NewID(), SecretKeyHash: authrpc.NewID(),
		Settings: store.DefaultProjectSettings(), CreatedAt: now, UpdatedAt: now}
	sk, err := keys.GenerateSigningKey(master)
	if err != nil {
		t.Fatal(err)
	}
	pk := store.ProjectKey{ID: authrpc.NewID(), ProjectID: project.ID, Kid: sk.Kid, Algorithm: sk.Algorithm,
		PublicKeyPEM: sk.PublicKeyPEM, PrivateKeyEnc: sk.PrivateKeyEnc,
		Status: store.ProjectKeyStatusActive, CreatedAt: now}
	if err := st.CreateProject(ctx, project, pk); err != nil {
		t.Fatal(err)
	}
	user := store.User{ID: authrpc.NewID(), ProjectID: project.ID, Email: "u@demo.test",
		CustomClaims: "{}", CreatedAt: now, UpdatedAt: now}
	if err := st.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	ent := store.Entitlement{ID: authrpc.NewID(), ProjectID: project.ID, Identifier: "pro", CreatedAt: now, UpdatedAt: now}
	if err := st.CreateEntitlement(ctx, ent); err != nil {
		t.Fatal(err)
	}
	expire := now.Add(24 * time.Hour)
	if err := st.CreateSubscriptionGrant(ctx, store.SubscriptionGrant{ID: authrpc.NewID(),
		ProjectID: project.ID, UserID: user.ID, EntitlementID: ent.ID, ExpiresAt: &expire,
		GrantedBy: "admin@demo.test", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}

	h := NewEntitlementHandler(st, func() time.Time { return now })
	reqCtx := authrpc.WithProject(ctx, project)

	resp, err := h.GetUserEntitlements(reqCtx, connect.NewRequest(&serverv1.GetUserEntitlementsRequest{UserId: user.ID}))
	if err != nil {
		t.Fatalf("GetUserEntitlements: %v", err)
	}
	if len(resp.Msg.Entitlements) != 1 {
		t.Fatalf("want 1 entitlement, got %d", len(resp.Msg.Entitlements))
	}
	e := resp.Msg.Entitlements[0]
	if e.Identifier != "pro" || e.Source != "grant" {
		t.Fatalf("unexpected entitlement: %+v", e)
	}
	if e.ExpireTime == nil || !e.ExpireTime.AsTime().Equal(expire) {
		t.Fatalf("expire time mismatch: %v want %v", e.ExpireTime, expire)
	}

	// A never-paid user returns the empty (none) set with no error.
	free := store.User{ID: authrpc.NewID(), ProjectID: project.ID, Email: "free@demo.test",
		CustomClaims: "{}", CreatedAt: now, UpdatedAt: now}
	if err := st.CreateUser(ctx, free); err != nil {
		t.Fatal(err)
	}
	got, err := h.GetUserEntitlements(reqCtx, connect.NewRequest(&serverv1.GetUserEntitlementsRequest{UserId: free.ID}))
	if err != nil || len(got.Msg.Entitlements) != 0 {
		t.Fatalf("free user should be none with no error: %v %+v", err, got.Msg.Entitlements)
	}

	// An unknown user id is NotFound.
	_, err = h.GetUserEntitlements(reqCtx, connect.NewRequest(&serverv1.GetUserEntitlementsRequest{UserId: "nope"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("unknown user: want NotFound, got %v", err)
	}

	// A store subscription in the sandbox environment surfaces is_sandbox=true so
	// a secret-key backend can tell a tester grant from a real paid one.
	prod := store.Product{ID: authrpc.NewID(), ProjectID: project.ID, Identifier: "monthly",
		AppleProductID: "com.demo.monthly", BillingPeriod: "monthly",
		EntitlementIDs: []string{ent.ID}, CreatedAt: now, UpdatedAt: now}
	if err := st.CreateProduct(ctx, prod); err != nil {
		t.Fatal(err)
	}
	tester := store.User{ID: authrpc.NewID(), ProjectID: project.ID, Email: "tester@demo.test",
		CustomClaims: "{}", CreatedAt: now, UpdatedAt: now}
	if err := st.CreateUser(ctx, tester); err != nil {
		t.Fatal(err)
	}
	end := now.Add(24 * time.Hour)
	if _, err := st.UpsertSubscription(ctx, store.Subscription{ID: authrpc.NewID(),
		ProjectID: project.ID, UserID: tester.ID, Store: store.SubscriptionStoreApple, ProductID: prod.ID,
		StoreTransactionID: "sandbox-otx", Status: store.SubscriptionStatusActive, CurrentPeriodEnd: &end,
		Environment: store.SubscriptionEnvironmentSandbox, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	sresp, err := h.GetUserEntitlements(reqCtx, connect.NewRequest(&serverv1.GetUserEntitlementsRequest{UserId: tester.ID}))
	if err != nil || len(sresp.Msg.Entitlements) != 1 {
		t.Fatalf("sandbox subscription entitlement: %v %+v", err, sresp.Msg.Entitlements)
	}
	if se := sresp.Msg.Entitlements[0]; se.Source != "store" || !se.IsSandbox {
		t.Fatalf("sandbox entitlement must report is_sandbox=true from a store source: %+v", se)
	}
}
