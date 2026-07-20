package adminrpc

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/audit"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/store"
)

func newBillingTestHandler(t *testing.T) (*BillingHandler, *store.Store, keys.MasterKey, store.Project) {
	t.Helper()
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
	project := store.Project{ID: NewID(), Name: "Demo", Slug: "demo",
		PublishableKey: "pk_" + NewID(), SecretKeyHash: NewID(),
		Settings: store.DefaultProjectSettings(), CreatedAt: now, UpdatedAt: now}
	sk, err := keys.GenerateSigningKey(master)
	if err != nil {
		t.Fatal(err)
	}
	pk := store.ProjectKey{ID: NewID(), ProjectID: project.ID, Kid: sk.Kid, Algorithm: sk.Algorithm,
		PublicKeyPEM: sk.PublicKeyPEM, PrivateKeyEnc: sk.PrivateKeyEnc,
		Status: store.ProjectKeyStatusActive, CreatedAt: now}
	if err := st.CreateProject(ctx, project, pk); err != nil {
		t.Fatal(err)
	}
	h := NewBillingHandler(st, master, nil, func() time.Time { return now })
	return h, st, master, project
}

func testP8(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

func testSA(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	raw, _ := json.Marshal(map[string]string{
		"type": "service_account", "client_email": "moth@proj.iam.gserviceaccount.com",
		"private_key_id": "kid-1", "private_key": pemStr, "token_uri": "https://oauth2.googleapis.com/token"})
	return string(raw)
}

func TestUpdateBillingCredentialsEncryptsSecretsAtRest(t *testing.T) {
	h, st, master, project := newBillingTestHandler(t)
	ctx := context.Background()
	p8 := testP8(t)
	sa := testSA(t)

	resp, err := h.UpdateBillingCredentials(ctx, connect.NewRequest(&adminv1.UpdateBillingCredentialsRequest{
		ProjectId: project.ID,
		Apple: &adminv1.AppleBillingConfig{IapKeyId: "K1", IapIssuerId: "I1", IapKeyP8: p8,
			BundleId: "com.demo.app", NotificationSecret: "asn-secret"},
		Google: &adminv1.GoogleBillingConfig{ServiceAccountJson: sa, PackageName: "com.demo.app",
			RtdnSecret: "rtdn-secret"},
	}))
	if err != nil {
		t.Fatalf("UpdateBillingCredentials: %v", err)
	}
	// The response echoes no secrets, only has_* indicators.
	if !resp.Msg.Apple.HasIapKey || !resp.Msg.Apple.HasNotificationSecret ||
		!resp.Msg.Google.HasServiceAccount || !resp.Msg.Google.HasRtdnSecret {
		t.Fatalf("has_* indicators not all set: %+v %+v", resp.Msg.Apple, resp.Msg.Google)
	}

	// At rest, the stored blobs are ciphertext (never the plaintext), and they
	// decrypt back to what was submitted.
	cred, err := st.GetBillingCredentials(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(cred.AppleIAPKeyEnc, []byte("PRIVATE KEY")) {
		t.Fatalf("apple .p8 stored in cleartext")
	}
	if bytes.Contains(cred.GoogleServiceAccountEnc, []byte("service_account")) {
		t.Fatalf("google service account stored in cleartext")
	}
	dec, err := master.Decrypt(cred.AppleIAPKeyEnc)
	if err != nil || string(dec) != p8 {
		t.Fatalf("apple key does not round-trip: %v", err)
	}
	decSA, err := master.Decrypt(cred.GoogleServiceAccountEnc)
	if err != nil || string(decSA) != sa {
		t.Fatalf("google sa does not round-trip: %v", err)
	}

	// An empty secret on a later update keeps the stored one (write-only).
	if _, err := h.UpdateBillingCredentials(ctx, connect.NewRequest(&adminv1.UpdateBillingCredentialsRequest{
		ProjectId: project.ID,
		Apple:     &adminv1.AppleBillingConfig{IapKeyId: "K2", BundleId: "com.demo.app"},
	})); err != nil {
		t.Fatal(err)
	}
	cred2, _ := st.GetBillingCredentials(ctx, project.ID)
	if !bytes.Equal(cred2.AppleIAPKeyEnc, cred.AppleIAPKeyEnc) {
		t.Fatalf("empty secret should keep stored key")
	}
	if cred2.AppleIAPKeyID != "K2" {
		t.Fatalf("non-secret field should update: %q", cred2.AppleIAPKeyID)
	}
}

func TestUpdateBillingCredentialsRejectsBadP8(t *testing.T) {
	h, _, _, project := newBillingTestHandler(t)
	_, err := h.UpdateBillingCredentials(context.Background(), connect.NewRequest(&adminv1.UpdateBillingCredentialsRequest{
		ProjectId: project.ID,
		Apple:     &adminv1.AppleBillingConfig{IapKeyP8: "not a key", BundleId: "com.demo.app"},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument for bad .p8, got %v", err)
	}
}

func TestGrantEntitlementAndDerivation(t *testing.T) {
	h, st, _, project := newBillingTestHandler(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	user := store.User{ID: NewID(), ProjectID: project.ID, Email: "u@demo.test",
		CustomClaims: "{}", CreatedAt: now, UpdatedAt: now}
	if err := st.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	ent := store.Entitlement{ID: NewID(), ProjectID: project.ID, Identifier: "pro", CreatedAt: now, UpdatedAt: now}
	if err := st.CreateEntitlement(ctx, ent); err != nil {
		t.Fatal(err)
	}
	// Comp a dated entitlement that expires in 24h.
	expire := now.Add(24 * time.Hour)
	resp, err := h.GrantEntitlement(ctx, connect.NewRequest(&adminv1.GrantEntitlementRequest{
		ProjectId: project.ID, UserId: user.ID, EntitlementId: ent.ID, Reason: "comp reviewer",
		ExpireTime: timestamppb.New(expire),
	}))
	if err != nil {
		t.Fatalf("GrantEntitlement: %v", err)
	}
	if resp.Msg.Grant.EntitlementId != ent.ID {
		t.Fatalf("grant not returned: %+v", resp.Msg.Grant)
	}
	if resp.Msg.Grant.ExpireTime == nil || !resp.Msg.Grant.ExpireTime.AsTime().Equal(expire) {
		t.Fatalf("grant expiry not plumbed through: %+v", resp.Msg.Grant.ExpireTime)
	}
	// Active grant appears in the active set now.
	active, err := st.ListActiveUserGrants(ctx, project.ID, user.ID, now)
	if err != nil || len(active) != 1 {
		t.Fatalf("active grant not visible: %d (%v)", len(active), err)
	}
	// It expires on schedule: once now passes ExpireTime it drops out of the
	// active (derived) set.
	afterExpiry := expire.Add(time.Minute)
	active, err = st.ListActiveUserGrants(ctx, project.ID, user.ID, afterExpiry)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 0 {
		t.Fatalf("dated grant should have lapsed after its expiry, got %d active", len(active))
	}
	// Revoke removes it immediately.
	if _, err := h.RevokeGrant(ctx, connect.NewRequest(&adminv1.RevokeGrantRequest{
		ProjectId: project.ID, GrantId: resp.Msg.Grant.Id,
	})); err != nil {
		t.Fatalf("RevokeGrant: %v", err)
	}
	active, _ = st.ListActiveUserGrants(ctx, project.ID, user.ID, now)
	if len(active) != 0 {
		t.Fatalf("revoked grant still active: %d", len(active))
	}
}

// newAuditedBillingTestHandler mirrors newBillingTestHandler but wires a real
// audit sink so tests can assert the audit-log side effects the plan requires.
func newAuditedBillingTestHandler(t *testing.T) (*BillingHandler, *store.Store, store.Project) {
	t.Helper()
	_, st, master, project := newBillingTestHandler(t)
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	auditor := NewAuditor(audit.New(st, slog.New(slog.DiscardHandler), func() time.Time { return now }))
	h := NewBillingHandler(st, master, auditor, func() time.Time { return now })
	return h, st, project
}

// adminCtx attaches an authenticated admin (cookie session) to ctx so audit
// entries are attributed, exactly as the auth interceptor would.
func adminCtx(ctx context.Context, admin store.Admin) context.Context {
	ctx = context.WithValue(ctx, adminCtxKey{}, admin)
	return context.WithValue(ctx, credentialCtxKey{}, Credential{Type: CredentialSession})
}

func TestGrantEntitlementIsAudited(t *testing.T) {
	h, st, project := newAuditedBillingTestHandler(t)
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	admin := store.Admin{ID: NewID(), Email: "ops@moth.test"}
	ctx := adminCtx(context.Background(), admin)

	user := store.User{ID: NewID(), ProjectID: project.ID, Email: "u@demo.test",
		CustomClaims: "{}", CreatedAt: now, UpdatedAt: now}
	if err := st.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	ent := store.Entitlement{ID: NewID(), ProjectID: project.ID, Identifier: "pro", CreatedAt: now, UpdatedAt: now}
	if err := st.CreateEntitlement(ctx, ent); err != nil {
		t.Fatal(err)
	}
	if _, err := h.GrantEntitlement(ctx, connect.NewRequest(&adminv1.GrantEntitlementRequest{
		ProjectId: project.ID, UserId: user.ID, EntitlementId: ent.ID, Reason: "comp reviewer",
	})); err != nil {
		t.Fatalf("GrantEntitlement: %v", err)
	}

	entries, err := st.ListAudit(ctx, store.AuditFilter{ProjectID: project.ID, Action: ActionGrantCreate, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("want one %s audit entry, got %d", ActionGrantCreate, len(entries))
	}
	if entries[0].ActorLabel != admin.Email || entries[0].TargetID != user.ID {
		t.Fatalf("audit entry not attributed to the granting admin/user: %+v", entries[0])
	}
}

func TestUpdateBillingCredentialsIsAudited(t *testing.T) {
	h, st, project := newAuditedBillingTestHandler(t)
	admin := store.Admin{ID: NewID(), Email: "ops@moth.test"}
	ctx := adminCtx(context.Background(), admin)

	if _, err := h.UpdateBillingCredentials(ctx, connect.NewRequest(&adminv1.UpdateBillingCredentialsRequest{
		ProjectId: project.ID,
		Apple:     &adminv1.AppleBillingConfig{IapKeyId: "K1", BundleId: "com.demo.app", IapKeyP8: testP8(t)},
	})); err != nil {
		t.Fatalf("UpdateBillingCredentials: %v", err)
	}
	entries, err := st.ListAudit(ctx, store.AuditFilter{ProjectID: project.ID, Action: ActionBillingCredsUpdate, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ActorLabel != admin.Email {
		t.Fatalf("want one attributed %s audit entry, got %+v", ActionBillingCredsUpdate, entries)
	}
}
