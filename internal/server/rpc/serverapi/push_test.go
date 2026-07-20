package serverapi

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"

	serverv1 "github.com/aloisdeniel/moth/gen/moth/server/v1"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// pushFixture is two projects with one user each, seeded through the store,
// and a push handler under a controlled clock — enough to prove the sk_
// surface serves tokens, pages, revokes and never crosses projects.
type pushFixture struct {
	t     *testing.T
	h     *PushHandler
	st    *store.Store
	now   time.Time
	projA store.Project
	projB store.Project
	userA store.User
	userB store.User
}

func newPushFixture(t *testing.T) *pushFixture {
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

	f := &pushFixture{t: t, st: st, now: now}
	f.projA, f.userA = f.seedProject("a")
	f.projB, f.userB = f.seedProject("b")
	f.h = NewPushHandler(st, func() time.Time { return f.now })
	return f
}

func (f *pushFixture) seedProject(slug string) (store.Project, store.User) {
	f.t.Helper()
	ctx := context.Background()
	p := store.Project{
		ID: authrpc.NewID(), Name: slug, Slug: slug,
		PublishableKey: "pk_" + authrpc.NewID(), SecretKeyHash: "hash-" + slug,
		Settings: store.DefaultProjectSettings(), CreatedAt: f.now, UpdatedAt: f.now,
	}
	k := store.ProjectKey{
		ID: authrpc.NewID(), ProjectID: p.ID, Kid: "kid-" + slug, Algorithm: "ES256",
		PublicKeyPEM: "PEM", PrivateKeyEnc: []byte{1}, Status: store.ProjectKeyStatusActive,
		CreatedAt: f.now,
	}
	if err := f.st.CreateProject(ctx, p, k); err != nil {
		f.t.Fatal(err)
	}
	u := store.User{ID: authrpc.NewID(), ProjectID: p.ID, Email: slug + "@demo.test",
		CustomClaims: "{}", CreatedAt: f.now, UpdatedAt: f.now}
	if err := f.st.CreateUser(ctx, u); err != nil {
		f.t.Fatal(err)
	}
	return p, u
}

// seedDevice registers one device row directly through the store.
func (f *pushFixture) seedDevice(p store.Project, u store.User, deviceID, token string, lastSeen time.Time) store.PushDevice {
	f.t.Helper()
	d, err := f.st.UpsertPushDevice(context.Background(), store.PushDevice{
		ID: authrpc.NewID(), ProjectID: p.ID, UserID: u.ID,
		Target: store.PushTargetAPNs, Token: token, DeviceID: deviceID,
		Permission: store.PushPermissionGranted, Platform: "ios", Locale: "en-US",
		CreatedAt: lastSeen, UpdatedAt: lastSeen, LastSeenAt: lastSeen,
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return d
}

func (f *pushFixture) ctxA() context.Context {
	return authrpc.WithProject(context.Background(), f.projA)
}

func (f *pushFixture) ctxB() context.Context {
	return authrpc.WithProject(context.Background(), f.projB)
}

func TestServerListUserPushDevices(t *testing.T) {
	f := newPushFixture(t)
	// The most recently *seen* row was registered first (its UUIDv7 id is the
	// smallest), so a creation-order list would get this backwards.
	newer := f.seedDevice(f.projA, f.userA, "device-2", "tok-2", f.now)
	older := f.seedDevice(f.projA, f.userA, "device-1", "tok-1", f.now.Add(-time.Hour))
	// A revoked registration never appears on the sender surface.
	revoked := f.seedDevice(f.projA, f.userA, "device-3", "tok-3", f.now)
	if err := f.st.RevokePushDevice(context.Background(), f.projA.ID, revoked.ID,
		store.PushRevokeReasonReportedInvalid, f.now); err != nil {
		t.Fatal(err)
	}

	resp, err := f.h.ListUserPushDevices(f.ctxA(), connect.NewRequest(
		&serverv1.ListUserPushDevicesRequest{UserId: f.userA.ID}))
	if err != nil {
		t.Fatalf("ListUserPushDevices: %v", err)
	}
	if len(resp.Msg.Devices) != 2 {
		t.Fatalf("want 2 devices, got %d", len(resp.Msg.Devices))
	}
	// Most recently seen first, and this surface carries the credential.
	if resp.Msg.Devices[0].Id != newer.ID || resp.Msg.Devices[1].Id != older.ID {
		t.Fatalf("order: %s, %s", resp.Msg.Devices[0].Id, resp.Msg.Devices[1].Id)
	}
	d := resp.Msg.Devices[0]
	if d.Token != "tok-2" || d.UserId != f.userA.ID ||
		d.Target != serverv1.PushTarget_PUSH_TARGET_APNS ||
		d.Permission != serverv1.PushPermission_PUSH_PERMISSION_GRANTED {
		t.Fatalf("device: %+v", d)
	}
	if d.Metadata.GetPlatform() != "ios" || d.Metadata.GetLocale() != "en-US" {
		t.Fatalf("metadata: %+v", d.Metadata)
	}

	// A bad user id is NotFound, not a silent empty set.
	_, err = f.h.ListUserPushDevices(f.ctxA(), connect.NewRequest(
		&serverv1.ListUserPushDevicesRequest{UserId: "nope"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("want NotFound, got %v", err)
	}
}

func TestServerListPushDevicesPagination(t *testing.T) {
	f := newPushFixture(t)
	for i := 0; i < 3; i++ {
		f.seedDevice(f.projA, f.userA, fmt.Sprintf("device-%d", i), fmt.Sprintf("tok-%d", i), f.now)
	}

	first, err := f.h.ListPushDevices(f.ctxA(), connect.NewRequest(
		&serverv1.ListPushDevicesRequest{PageSize: 2}))
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first.Msg.Devices) != 2 || first.Msg.NextPageToken == "" {
		t.Fatalf("first page: %d devices, token %q", len(first.Msg.Devices), first.Msg.NextPageToken)
	}
	second, err := f.h.ListPushDevices(f.ctxA(), connect.NewRequest(
		&serverv1.ListPushDevicesRequest{PageSize: 2, PageToken: first.Msg.NextPageToken}))
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(second.Msg.Devices) != 1 || second.Msg.NextPageToken != "" {
		t.Fatalf("second page: %d devices, token %q", len(second.Msg.Devices), second.Msg.NextPageToken)
	}
	// No overlap across pages.
	seen := map[string]bool{}
	for _, d := range append(first.Msg.Devices, second.Msg.Devices...) {
		if seen[d.Id] {
			t.Fatalf("device %s appeared twice", d.Id)
		}
		seen[d.Id] = true
	}

	// Target filter: no webpush registrations exist.
	none, err := f.h.ListPushDevices(f.ctxA(), connect.NewRequest(
		&serverv1.ListPushDevicesRequest{Target: serverv1.PushTarget_PUSH_TARGET_WEBPUSH}))
	if err != nil {
		t.Fatal(err)
	}
	if len(none.Msg.Devices) != 0 {
		t.Fatalf("webpush filter: %+v", none.Msg.Devices)
	}

	_, err = f.h.ListPushDevices(f.ctxA(), connect.NewRequest(
		&serverv1.ListPushDevicesRequest{PageSize: -1}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("negative page size: want InvalidArgument, got %v", err)
	}
}

func TestServerRevokePushDeviceFeedback(t *testing.T) {
	f := newPushFixture(t)
	d := f.seedDevice(f.projA, f.userA, "device-1", "tok-1", f.now)

	byToken := &serverv1.RevokePushDeviceRequest{
		Selector: &serverv1.RevokePushDeviceRequest_Token{Token: "tok-1"}}
	if _, err := f.h.RevokePushDevice(f.ctxA(), connect.NewRequest(byToken)); err != nil {
		t.Fatalf("RevokePushDevice: %v", err)
	}
	row, err := f.st.GetPushDevice(context.Background(), f.projA.ID, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if row.RevokedAt == nil || row.RevokeReason != store.PushRevokeReasonReportedInvalid {
		t.Fatalf("not revoked reported_invalid: %+v", row)
	}

	// Replays and unknown credentials succeed (the sender reports lazily).
	if _, err := f.h.RevokePushDevice(f.ctxA(), connect.NewRequest(byToken)); err != nil {
		t.Fatalf("replay: %v", err)
	}
	byDevice := &serverv1.RevokePushDeviceRequest{
		Selector: &serverv1.RevokePushDeviceRequest_DeviceId{DeviceId: "never-seen"}}
	if _, err := f.h.RevokePushDevice(f.ctxA(), connect.NewRequest(byDevice)); err != nil {
		t.Fatalf("unknown device: %v", err)
	}
	// The replay never overwrites the original reason.
	if row, err = f.st.GetPushDevice(context.Background(), f.projA.ID, d.ID); err != nil {
		t.Fatal(err)
	}
	if row.RevokeReason != store.PushRevokeReasonReportedInvalid {
		t.Fatalf("reason overwritten: %q", row.RevokeReason)
	}

	// An empty selector is a request bug, not a silent no-op.
	_, err = f.h.RevokePushDevice(f.ctxA(), connect.NewRequest(&serverv1.RevokePushDeviceRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("empty selector: want InvalidArgument, got %v", err)
	}
}

// TestServerPushCrossProjectIsolation proves the sk_ scoping: project B's key
// can neither see nor revoke project A's registrations.
func TestServerPushCrossProjectIsolation(t *testing.T) {
	f := newPushFixture(t)
	d := f.seedDevice(f.projA, f.userA, "device-1", "tok-1", f.now)

	// B's broadcast list is empty.
	list, err := f.h.ListPushDevices(f.ctxB(), connect.NewRequest(&serverv1.ListPushDevicesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Msg.Devices) != 0 {
		t.Fatalf("project B sees A's devices: %+v", list.Msg.Devices)
	}

	// A's user id does not resolve inside B.
	_, err = f.h.ListUserPushDevices(f.ctxB(), connect.NewRequest(
		&serverv1.ListUserPushDevicesRequest{UserId: f.userA.ID}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("want NotFound, got %v", err)
	}

	// B revoking A's token succeeds (idempotent surface) but touches nothing.
	if _, err := f.h.RevokePushDevice(f.ctxB(), connect.NewRequest(&serverv1.RevokePushDeviceRequest{
		Selector: &serverv1.RevokePushDeviceRequest_Token{Token: "tok-1"}})); err != nil {
		t.Fatal(err)
	}
	row, err := f.st.GetPushDevice(context.Background(), f.projA.ID, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if row.RevokedAt != nil {
		t.Fatalf("project B revoked A's registration: %+v", row)
	}
}
