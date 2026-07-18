package adminrpc

import (
	"context"
	"encoding/base64"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/audit"
	"github.com/aloisdeniel/moth/internal/store"
)

// pushAdminFixture is one project with one user and a push admin handler
// whose audit sink writes to the same store.
type pushAdminFixture struct {
	t       *testing.T
	h       *PushHandler
	st      *store.Store
	now     time.Time
	project store.Project
	user    store.User
}

func newPushAdminFixture(t *testing.T) *pushAdminFixture {
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
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	p := store.Project{
		ID: NewID(), Name: "Demo", Slug: "demo",
		PublishableKey: "pk_" + NewID(), SecretKeyHash: "hash",
		Settings: store.DefaultProjectSettings(), CreatedAt: now, UpdatedAt: now,
	}
	k := store.ProjectKey{
		ID: NewID(), ProjectID: p.ID, Kid: "kid", Algorithm: "ES256",
		PublicKeyPEM: "PEM", PrivateKeyEnc: []byte{1},
		Status: store.ProjectKeyStatusActive, CreatedAt: now,
	}
	if err := st.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	u := store.User{ID: NewID(), ProjectID: p.ID, Email: "u@demo.test",
		CustomClaims: "{}", CreatedAt: now, UpdatedAt: now}
	if err := st.CreateUser(ctx, u); err != nil {
		t.Fatal(err)
	}

	f := &pushAdminFixture{t: t, st: st, now: now, project: p, user: u}
	auditor := NewAuditor(audit.New(st, slog.New(slog.DiscardHandler), func() time.Time { return f.now }))
	f.h = NewPushHandler(st, auditor, func() time.Time { return f.now })
	return f
}

func (f *pushAdminFixture) seedDevice(deviceID, token string, lastSeen time.Time) store.PushDevice {
	f.t.Helper()
	d, err := f.st.UpsertPushDevice(context.Background(), store.PushDevice{
		ID: NewID(), ProjectID: f.project.ID, UserID: f.user.ID,
		Target: store.PushTargetFCM, Token: token, DeviceID: deviceID,
		Permission: store.PushPermissionGranted, Platform: "android", Model: "Pixel 9",
		CreatedAt: lastSeen, UpdatedAt: lastSeen, LastSeenAt: lastSeen,
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return d
}

func TestPushSettingsRoundTrip(t *testing.T) {
	f := newPushAdminFixture(t)
	ctx := context.Background()

	// Defaults: a project that never configured push is disabled with no key.
	got, err := f.h.GetPushSettings(ctx, connect.NewRequest(
		&adminv1.GetPushSettingsRequest{ProjectId: f.project.ID}))
	if err != nil {
		t.Fatalf("GetPushSettings: %v", err)
	}
	if got.Msg.Settings.Enabled || got.Msg.Settings.WebpushVapidPublicKey != "" {
		t.Fatalf("defaults: %+v", got.Msg.Settings)
	}

	// Full replacement with a shape-valid VAPID public key (0x04 || 64 bytes,
	// base64url).
	vapid := validVAPIDKey(t)
	updated, err := f.h.UpdatePushSettings(ctx, connect.NewRequest(&adminv1.UpdatePushSettingsRequest{
		ProjectId: f.project.ID,
		Settings:  &adminv1.PushSettings{Enabled: true, WebpushVapidPublicKey: vapid},
	}))
	if err != nil {
		t.Fatalf("UpdatePushSettings: %v", err)
	}
	if !updated.Msg.Settings.Enabled || updated.Msg.Settings.WebpushVapidPublicKey != vapid {
		t.Fatalf("updated: %+v", updated.Msg.Settings)
	}
	got, err = f.h.GetPushSettings(ctx, connect.NewRequest(
		&adminv1.GetPushSettingsRequest{ProjectId: f.project.ID}))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Msg.Settings.Enabled || got.Msg.Settings.WebpushVapidPublicKey != vapid {
		t.Fatalf("read back: %+v", got.Msg.Settings)
	}

	// The update is audit-logged.
	entries, err := f.st.ListAudit(ctx, store.AuditFilter{Action: ActionPushSettingsUpdate, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ProjectID != f.project.ID {
		t.Fatalf("audit entries: %+v", entries)
	}

	// A malformed VAPID key is rejected before anything is stored.
	_, err = f.h.UpdatePushSettings(ctx, connect.NewRequest(&adminv1.UpdatePushSettingsRequest{
		ProjectId: f.project.ID,
		Settings:  &adminv1.PushSettings{Enabled: true, WebpushVapidPublicKey: "not-a-key"},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("bad key: want InvalidArgument, got %v", err)
	}
	_, err = f.h.UpdatePushSettings(ctx, connect.NewRequest(&adminv1.UpdatePushSettingsRequest{
		ProjectId: f.project.ID,
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("nil settings: want InvalidArgument, got %v", err)
	}

	// Unknown projects are NotFound on both RPCs.
	_, err = f.h.GetPushSettings(ctx, connect.NewRequest(
		&adminv1.GetPushSettingsRequest{ProjectId: "nope"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("get unknown project: want NotFound, got %v", err)
	}
	_, err = f.h.UpdatePushSettings(ctx, connect.NewRequest(&adminv1.UpdatePushSettingsRequest{
		ProjectId: "nope", Settings: &adminv1.PushSettings{Enabled: true},
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("update unknown project: want NotFound, got %v", err)
	}
}

// validVAPIDKey builds a base64url string that decodes to a 65-byte
// uncompressed P-256 point marker (0x04 prefix).
func validVAPIDKey(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 65)
	raw[0] = 0x04
	for i := 1; i < len(raw); i++ {
		raw[i] = byte(i)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func TestAdminListUserPushDevicesShowsRevoked(t *testing.T) {
	f := newPushAdminFixture(t)
	ctx := context.Background()
	active := f.seedDevice("device-1", "tok-1", f.now)
	revoked := f.seedDevice("device-2", "tok-2", f.now.Add(-time.Hour))
	if err := f.st.RevokePushDevice(ctx, f.project.ID, revoked.ID,
		store.PushRevokeReasonSignedOut, f.now); err != nil {
		t.Fatal(err)
	}

	resp, err := f.h.ListUserPushDevices(ctx, connect.NewRequest(&adminv1.ListUserPushDevicesRequest{
		ProjectId: f.project.ID, UserId: f.user.ID}))
	if err != nil {
		t.Fatalf("ListUserPushDevices: %v", err)
	}
	if len(resp.Msg.Devices) != 2 {
		t.Fatalf("want active+revoked, got %d", len(resp.Msg.Devices))
	}
	// Most recently seen first; the revoked row keeps its reason and time.
	first, second := resp.Msg.Devices[0], resp.Msg.Devices[1]
	if first.Id != active.ID || second.Id != revoked.ID {
		t.Fatalf("order: %s, %s", first.Id, second.Id)
	}
	if first.RevokeTime != nil || first.RevokeReason != adminv1.PushRevokeReason_PUSH_REVOKE_REASON_UNSPECIFIED {
		t.Fatalf("active row carries revocation: %+v", first)
	}
	if second.RevokeTime == nil || !second.RevokeTime.AsTime().Equal(f.now) ||
		second.RevokeReason != adminv1.PushRevokeReason_PUSH_REVOKE_REASON_SIGNED_OUT {
		t.Fatalf("revoked row: %+v", second)
	}
	// The panel gets metadata, never the credential — adminv1.PushDevice has
	// no token field by design, so this stays a compile-time guarantee; spot
	// check the metadata made it through.
	if first.Metadata.GetModel() != "Pixel 9" ||
		first.Target != adminv1.PushTarget_PUSH_TARGET_FCM {
		t.Fatalf("metadata: %+v", first)
	}

	_, err = f.h.ListUserPushDevices(ctx, connect.NewRequest(&adminv1.ListUserPushDevicesRequest{
		ProjectId: f.project.ID, UserId: "nope"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("unknown user: want NotFound, got %v", err)
	}
}

func TestAdminRevokePushDevice(t *testing.T) {
	f := newPushAdminFixture(t)
	ctx := context.Background()
	d := f.seedDevice("device-1", "tok-1", f.now)

	resp, err := f.h.RevokePushDevice(ctx, connect.NewRequest(&adminv1.RevokePushDeviceRequest{
		ProjectId: f.project.ID, PushDeviceId: d.ID}))
	if err != nil {
		t.Fatalf("RevokePushDevice: %v", err)
	}
	if resp.Msg.Device.RevokeReason != adminv1.PushRevokeReason_PUSH_REVOKE_REASON_ADMIN ||
		resp.Msg.Device.RevokeTime == nil {
		t.Fatalf("revoked device: %+v", resp.Msg.Device)
	}
	entries, err := f.st.ListAudit(ctx, store.AuditFilter{Action: ActionPushDeviceRevoke, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].TargetID != d.ID {
		t.Fatalf("audit entries: %+v", entries)
	}

	// Replay: still success, reason stays `admin`, no second audit line.
	if resp, err = f.h.RevokePushDevice(ctx, connect.NewRequest(&adminv1.RevokePushDeviceRequest{
		ProjectId: f.project.ID, PushDeviceId: d.ID})); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if resp.Msg.Device.RevokeReason != adminv1.PushRevokeReason_PUSH_REVOKE_REASON_ADMIN {
		t.Fatalf("replay reason: %v", resp.Msg.Device.RevokeReason)
	}
	if entries, err = f.st.ListAudit(ctx, store.AuditFilter{Action: ActionPushDeviceRevoke, Limit: 10}); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("replay re-audited: %d entries", len(entries))
	}

	// A registration already revoked for another reason keeps that reason.
	d2 := f.seedDevice("device-2", "tok-2", f.now)
	if err := f.st.RevokePushDevice(ctx, f.project.ID, d2.ID,
		store.PushRevokeReasonReportedInvalid, f.now); err != nil {
		t.Fatal(err)
	}
	if resp, err = f.h.RevokePushDevice(ctx, connect.NewRequest(&adminv1.RevokePushDeviceRequest{
		ProjectId: f.project.ID, PushDeviceId: d2.ID})); err != nil {
		t.Fatal(err)
	}
	if resp.Msg.Device.RevokeReason != adminv1.PushRevokeReason_PUSH_REVOKE_REASON_REPORTED_INVALID {
		t.Fatalf("original reason lost: %v", resp.Msg.Device.RevokeReason)
	}

	// Unknown row ids are NotFound (the id is the registration's row id, not
	// the installation device_id).
	_, err = f.h.RevokePushDevice(ctx, connect.NewRequest(&adminv1.RevokePushDeviceRequest{
		ProjectId: f.project.ID, PushDeviceId: "device-1"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("device_id as row id: want NotFound, got %v", err)
	}
}

// TestAdminListPushDevices covers the Push tab's project-wide listing:
// active-only, newest first, keyset pagination, target filter, user emails
// resolved, project-wide counts, and no token anywhere in the response type.
func TestAdminListPushDevices(t *testing.T) {
	f := newPushAdminFixture(t)
	ctx := context.Background()

	// A second user shows the email resolution is per row, not per page.
	u2 := store.User{ID: NewID(), ProjectID: f.project.ID, Email: "second@demo.test",
		CustomClaims: "{}", CreatedAt: f.now, UpdatedAt: f.now}
	if err := f.st.CreateUser(ctx, u2); err != nil {
		t.Fatal(err)
	}
	d1 := f.seedDevice("dev-1", "tok-1", f.now.Add(-2*time.Hour))
	if _, err := f.st.UpsertPushDevice(ctx, store.PushDevice{
		ID: NewID(), ProjectID: f.project.ID, UserID: u2.ID,
		Target: store.PushTargetAPNs, Token: "tok-2", DeviceID: "dev-2",
		Permission: store.PushPermissionDenied, Platform: "ios", Model: "iPhone16,1",
		CreatedAt: f.now.Add(-time.Hour), UpdatedAt: f.now.Add(-time.Hour), LastSeenAt: f.now.Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	// A revoked row must never surface in the project listing.
	revoked := f.seedDevice("dev-3", "tok-3", f.now)
	if err := f.st.RevokePushDevice(ctx, f.project.ID, revoked.ID, store.PushRevokeReasonAdmin, f.now); err != nil {
		t.Fatal(err)
	}

	resp, err := f.h.ListPushDevices(ctx, connect.NewRequest(&adminv1.ListPushDevicesRequest{
		ProjectId: f.project.ID,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.Devices) != 2 || resp.Msg.NextPageToken != "" {
		t.Fatalf("want 2 active devices and no next page, got %d %q", len(resp.Msg.Devices), resp.Msg.NextPageToken)
	}
	// Newest first by row id (UUIDv7): dev-2 was inserted after dev-1.
	first, second := resp.Msg.Devices[0], resp.Msg.Devices[1]
	if first.Device.DeviceId != "dev-2" || second.Device.DeviceId != "dev-1" {
		t.Fatalf("want newest first (dev-2, dev-1), got (%s, %s)", first.Device.DeviceId, second.Device.DeviceId)
	}
	if first.UserEmail != "second@demo.test" || second.UserEmail != "u@demo.test" {
		t.Fatalf("emails not resolved: %q %q", first.UserEmail, second.UserEmail)
	}
	if first.UserId != u2.ID || second.UserId != f.user.ID {
		t.Fatal("user ids not carried")
	}
	if resp.Msg.ApnsCount != 1 || resp.Msg.FcmCount != 1 || resp.Msg.WebpushCount != 0 {
		t.Fatalf("counts wrong: apns=%d fcm=%d webpush=%d", resp.Msg.ApnsCount, resp.Msg.FcmCount, resp.Msg.WebpushCount)
	}

	// Target filter narrows the page but never the project-wide counts.
	apns, err := f.h.ListPushDevices(ctx, connect.NewRequest(&adminv1.ListPushDevicesRequest{
		ProjectId: f.project.ID, Target: adminv1.PushTarget_PUSH_TARGET_APNS,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(apns.Msg.Devices) != 1 || apns.Msg.Devices[0].Device.DeviceId != "dev-2" {
		t.Fatalf("target filter wrong: %v", apns.Msg.Devices)
	}
	if apns.Msg.FcmCount != 1 {
		t.Fatal("counts must stay project-wide under a target filter")
	}

	// Keyset pagination: page size 1 walks dev-2 then dev-1.
	page1, err := f.h.ListPushDevices(ctx, connect.NewRequest(&adminv1.ListPushDevicesRequest{
		ProjectId: f.project.ID, PageSize: 1,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(page1.Msg.Devices) != 1 || page1.Msg.NextPageToken == "" {
		t.Fatal("want a full first page with a next token")
	}
	page2, err := f.h.ListPushDevices(ctx, connect.NewRequest(&adminv1.ListPushDevicesRequest{
		ProjectId: f.project.ID, PageSize: 1, PageToken: page1.Msg.NextPageToken,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Msg.Devices) != 1 || page2.Msg.Devices[0].Device.DeviceId != "dev-1" ||
		page2.Msg.NextPageToken != "" {
		t.Fatalf("second page wrong: %v next=%q", page2.Msg.Devices, page2.Msg.NextPageToken)
	}
	_ = d1

	// Unknown project → NotFound, like every project-scoped admin RPC.
	if _, err := f.h.ListPushDevices(ctx, connect.NewRequest(&adminv1.ListPushDevicesRequest{
		ProjectId: NewID(),
	})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("want NotFound for unknown project, got %v", err)
	}
}

// TestAdminListPushDevicesCrossProject asserts a project's listing never
// leaks another project's registrations or user emails.
func TestAdminListPushDevicesCrossProject(t *testing.T) {
	f := newPushAdminFixture(t)
	ctx := context.Background()
	f.seedDevice("dev-1", "tok-1", f.now)

	other := store.Project{
		ID: NewID(), Name: "Other", Slug: "other",
		PublishableKey: "pk_" + NewID(), SecretKeyHash: "hash2",
		Settings: store.DefaultProjectSettings(), CreatedAt: f.now, UpdatedAt: f.now,
	}
	k := store.ProjectKey{
		ID: NewID(), ProjectID: other.ID, Kid: "kid2", Algorithm: "ES256",
		PublicKeyPEM: "PEM", PrivateKeyEnc: []byte{1},
		Status: store.ProjectKeyStatusActive, CreatedAt: f.now,
	}
	if err := f.st.CreateProject(ctx, other, k); err != nil {
		t.Fatal(err)
	}
	resp, err := f.h.ListPushDevices(ctx, connect.NewRequest(&adminv1.ListPushDevicesRequest{
		ProjectId: other.ID,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.Devices) != 0 || resp.Msg.FcmCount != 0 {
		t.Fatalf("other project must see nothing, got %d devices fcm=%d", len(resp.Msg.Devices), resp.Msg.FcmCount)
	}
}
