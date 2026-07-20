package pushrpc

import (
	"context"
	"crypto/ecdsa"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	_ "modernc.org/sqlite"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	pushv1 "github.com/aloisdeniel/moth/gen/moth/push/v1"
	"github.com/aloisdeniel/moth/gen/moth/push/v1/pushv1connect"
	"github.com/aloisdeniel/moth/internal/jwt"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/mail"
	"github.com/aloisdeniel/moth/internal/netutil"
	"github.com/aloisdeniel/moth/internal/push"
	"github.com/aloisdeniel/moth/internal/ratelimit"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// fixture is a push-enabled project with one signed-in user and a push
// handler under a controlled clock.
type fixture struct {
	t       *testing.T
	h       *Handler
	st      *store.Store
	project store.Project
	user    store.User
	access  string
	now     time.Time
	priv    *ecdsa.PrivateKey
	kid     string
	authH   *authrpc.Handler
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "moth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	master, err := keys.LoadOrCreateMasterKey(t.TempDir(), os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)

	project := store.Project{
		ID: authrpc.NewID(), Name: "Demo", Slug: "demo",
		PublishableKey: "pk_" + authrpc.NewID(), SecretKeyHash: authrpc.NewID(),
		Settings: store.DefaultProjectSettings(), CreatedAt: now, UpdatedAt: now,
	}
	sk, err := keys.GenerateSigningKey(master)
	if err != nil {
		t.Fatal(err)
	}
	pk := store.ProjectKey{
		ID: authrpc.NewID(), ProjectID: project.ID, Kid: sk.Kid, Algorithm: sk.Algorithm,
		PublicKeyPEM: sk.PublicKeyPEM, PrivateKeyEnc: sk.PrivateKeyEnc,
		Status: store.ProjectKeyStatusActive, CreatedAt: now,
	}
	if err := st.CreateProject(ctx, project, pk); err != nil {
		t.Fatal(err)
	}
	// Enable push for the project (the default is disabled).
	raw, err := push.Encode(push.Config{Version: push.SchemaVersion, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetProjectPush(ctx, project.ID, raw, now); err != nil {
		t.Fatal(err)
	}
	if project, err = st.GetProject(ctx, project.ID); err != nil {
		t.Fatal(err)
	}

	f := &fixture{t: t, st: st, project: project, now: now, kid: sk.Kid}
	if f.priv, err = keys.DecryptPrivateKey(master, sk.PrivateKeyEnc); err != nil {
		t.Fatal(err)
	}
	f.user = f.createUser("u@demo.test")
	f.access = f.signAccess(f.user.ID)

	f.authH = authrpc.New(authrpc.Options{Store: st, Master: master, Mailer: mail.Console{},
		BaseURL: "http://localhost", Now: func() time.Time { return f.now }})
	f.h = New(Options{Store: st, Auth: f.authH, Now: func() time.Time { return f.now }})
	return f
}

func (f *fixture) createUser(email string) store.User {
	f.t.Helper()
	u := store.User{ID: authrpc.NewID(), ProjectID: f.project.ID, Email: email,
		CustomClaims: "{}", CreatedAt: f.now, UpdatedAt: f.now}
	verified := f.now
	u.EmailVerifiedAt = &verified
	if err := f.st.CreateUser(context.Background(), u); err != nil {
		f.t.Fatal(err)
	}
	return u
}

func (f *fixture) signAccess(userID string) string {
	f.t.Helper()
	access, err := jwt.Sign(f.priv, f.kid, jwt.Claims{
		Subject: userID, Audience: f.project.Slug,
		IssuedAt: f.now.Unix(), ExpiresAt: f.now.Add(time.Hour).Unix(),
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return access
}

// ctx returns a context scoped to the fixture project, as the pk_ interceptor
// would set it.
func (f *fixture) ctx() context.Context {
	return authrpc.WithProject(context.Background(), f.project)
}

func authReq[T any](access string, msg *T) *connect.Request[T] {
	req := connect.NewRequest(msg)
	req.Header().Set("Authorization", "Bearer "+access)
	return req
}

func registerMsg(deviceID, token string) *pushv1.RegisterDeviceRequest {
	return &pushv1.RegisterDeviceRequest{
		Target:     pushv1.PushTarget_PUSH_TARGET_APNS,
		Token:      token,
		DeviceId:   deviceID,
		Permission: pushv1.PushPermission_PUSH_PERMISSION_GRANTED,
		Metadata: &pushv1.PushDeviceMetadata{
			Platform: "ios", Model: "iPhone16,1", OsVersion: "18.2",
			AppVersion: "1.0.0", Locale: "fr-FR",
		},
	}
}

func (f *fixture) register(access string, msg *pushv1.RegisterDeviceRequest) (*pushv1.PushDevice, error) {
	resp, err := f.h.RegisterDevice(f.ctx(), authReq(access, msg))
	if err != nil {
		return nil, err
	}
	return resp.Msg.Device, nil
}

func (f *fixture) activeDevices(userID string) []store.PushDevice {
	f.t.Helper()
	devices, err := f.st.ListActivePushDevicesByUser(context.Background(), f.project.ID, userID)
	if err != nil {
		f.t.Fatal(err)
	}
	return devices
}

func TestRegisterDeviceRoundTrip(t *testing.T) {
	f := newFixture(t)

	d, err := f.register(f.access, registerMsg("device-1", "tok-1"))
	if err != nil {
		t.Fatalf("RegisterDevice: %v", err)
	}
	if d.Id == "" || d.DeviceId != "device-1" {
		t.Fatalf("unexpected device: %+v", d)
	}
	if d.Target != pushv1.PushTarget_PUSH_TARGET_APNS ||
		d.Permission != pushv1.PushPermission_PUSH_PERMISSION_GRANTED {
		t.Fatalf("target/permission mismatch: %+v", d)
	}
	if d.Metadata.GetLocale() != "fr-FR" || d.Metadata.GetPlatform() != "ios" {
		t.Fatalf("metadata mismatch: %+v", d.Metadata)
	}
	if !d.LastSeenTime.AsTime().Equal(f.now) {
		t.Fatalf("last seen: got %v want %v", d.LastSeenTime.AsTime(), f.now)
	}

	// The registration hangs off the authenticated user (the request carries
	// no user id), with the credential stored server-side only.
	rows := f.activeDevices(f.user.ID)
	if len(rows) != 1 || rows[0].UserID != f.user.ID || rows[0].Token != "tok-1" {
		t.Fatalf("stored rows: %+v", rows)
	}

	// Re-registering the identical credential refreshes in place: same row id,
	// bumped last_seen. (The clock stays inside the access token's lifetime.)
	f.now = f.now.Add(30 * time.Minute)
	d2, err := f.register(f.access, registerMsg("device-1", "tok-1"))
	if err != nil {
		t.Fatalf("re-register: %v", err)
	}
	if d2.Id != d.Id {
		t.Fatalf("refresh must keep the row: got %s want %s", d2.Id, d.Id)
	}
	if !d2.LastSeenTime.AsTime().Equal(f.now) {
		t.Fatalf("last seen not refreshed: %v", d2.LastSeenTime.AsTime())
	}
	if got := len(f.activeDevices(f.user.ID)); got != 1 {
		t.Fatalf("want 1 active row, got %d", got)
	}
}

func TestRegisterDeviceRotationSupersedes(t *testing.T) {
	f := newFixture(t)

	d1, err := f.register(f.access, registerMsg("device-1", "tok-old"))
	if err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(time.Minute)
	d2, err := f.register(f.access, registerMsg("device-1", "tok-new"))
	if err != nil {
		t.Fatal(err)
	}
	if d2.Id == d1.Id {
		t.Fatal("rotation must create a fresh row")
	}
	active := f.activeDevices(f.user.ID)
	if len(active) != 1 || active[0].Token != "tok-new" {
		t.Fatalf("active rows after rotation: %+v", active)
	}
	// The displaced row is revoked `replaced`, not deleted or duplicated.
	old, err := f.st.GetPushDevice(context.Background(), f.project.ID, d1.Id)
	if err != nil {
		t.Fatal(err)
	}
	if old.RevokedAt == nil || old.RevokeReason != store.PushRevokeReasonReplaced {
		t.Fatalf("old row not superseded: %+v", old)
	}
}

func TestRegisterDeviceUserTakeover(t *testing.T) {
	f := newFixture(t)
	if _, err := f.register(f.access, registerMsg("device-1", "tok-1")); err != nil {
		t.Fatal(err)
	}

	// A second user signs in on the same physical device: the newest owner
	// wins and the first user's list no longer shows it.
	second := f.createUser("second@demo.test")
	f.now = f.now.Add(time.Minute)
	if _, err := f.register(f.signAccess(second.ID), registerMsg("device-1", "tok-1")); err != nil {
		t.Fatal(err)
	}
	if got := f.activeDevices(f.user.ID); len(got) != 0 {
		t.Fatalf("first user still owns the device: %+v", got)
	}
	got := f.activeDevices(second.ID)
	if len(got) != 1 || got[0].DeviceID != "device-1" {
		t.Fatalf("second user's devices: %+v", got)
	}
}

func TestRegisterDevicePermissionTransition(t *testing.T) {
	f := newFixture(t)
	d1, err := f.register(f.access, registerMsg("device-1", "tok-1"))
	if err != nil {
		t.Fatal(err)
	}

	msg := registerMsg("device-1", "tok-1")
	msg.Permission = pushv1.PushPermission_PUSH_PERMISSION_DENIED
	d2, err := f.register(f.access, msg)
	if err != nil {
		t.Fatal(err)
	}
	if d2.Id != d1.Id {
		t.Fatal("permission change must not replace the row")
	}
	if d2.Permission != pushv1.PushPermission_PUSH_PERMISSION_DENIED {
		t.Fatalf("permission: %v", d2.Permission)
	}
	// A denied registration is kept active (data pushes may still work).
	rows := f.activeDevices(f.user.ID)
	if len(rows) != 1 || rows[0].Permission != store.PushPermissionDenied {
		t.Fatalf("stored permission: %+v", rows)
	}
}

func TestUnregisterDeviceIdempotent(t *testing.T) {
	f := newFixture(t)
	d, err := f.register(f.access, registerMsg("device-1", "tok-1"))
	if err != nil {
		t.Fatal(err)
	}

	unreg := &pushv1.UnregisterDeviceRequest{DeviceId: "device-1"}
	if _, err := f.h.UnregisterDevice(f.ctx(), authReq(f.access, unreg)); err != nil {
		t.Fatalf("UnregisterDevice: %v", err)
	}
	row, err := f.st.GetPushDevice(context.Background(), f.project.ID, d.Id)
	if err != nil {
		t.Fatal(err)
	}
	if row.RevokedAt == nil || row.RevokeReason != store.PushRevokeReasonSignedOut {
		t.Fatalf("not revoked signed_out: %+v", row)
	}

	// Replays and unknown installation ids succeed.
	if _, err := f.h.UnregisterDevice(f.ctx(), authReq(f.access, unreg)); err != nil {
		t.Fatalf("replay: %v", err)
	}
	unknown := &pushv1.UnregisterDeviceRequest{DeviceId: "never-registered"}
	if _, err := f.h.UnregisterDevice(f.ctx(), authReq(f.access, unknown)); err != nil {
		t.Fatalf("unknown id: %v", err)
	}
	// The replay kept the original reason.
	if row, err = f.st.GetPushDevice(context.Background(), f.project.ID, d.Id); err != nil {
		t.Fatal(err)
	}
	if row.RevokeReason != store.PushRevokeReasonSignedOut {
		t.Fatalf("reason overwritten: %q", row.RevokeReason)
	}
}

func TestRegisterDeviceDisabledProject(t *testing.T) {
	f := newFixture(t)
	// Turn push back off (the fixture enables it).
	raw, err := push.Encode(push.Config{Version: push.SchemaVersion, Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.st.SetProjectPush(context.Background(), f.project.ID, raw, f.now); err != nil {
		t.Fatal(err)
	}
	if f.project, err = f.st.GetProject(context.Background(), f.project.ID); err != nil {
		t.Fatal(err)
	}

	_, err = f.register(f.access, registerMsg("device-1", "tok-1"))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("want FailedPrecondition, got %v", err)
	}
	// Unregister stays allowed while disabled: revoking is always safe.
	unreg := &pushv1.UnregisterDeviceRequest{DeviceId: "device-1"}
	if _, err := f.h.UnregisterDevice(f.ctx(), authReq(f.access, unreg)); err != nil {
		t.Fatalf("UnregisterDevice while disabled: %v", err)
	}
}

func TestRegisterDeviceValidation(t *testing.T) {
	f := newFixture(t)
	cases := map[string]*pushv1.RegisterDeviceRequest{
		"missing target": {Token: "tok", DeviceId: "d"},
		"missing token":  {Target: pushv1.PushTarget_PUSH_TARGET_FCM, DeviceId: "d"},
		"missing device": {Target: pushv1.PushTarget_PUSH_TARGET_FCM, Token: "tok"},
		"oversized token": {Target: pushv1.PushTarget_PUSH_TARGET_FCM,
			Token: string(make([]byte, maxTokenLen+1)), DeviceId: "d"},
	}
	for name, msg := range cases {
		if _, err := f.register(f.access, msg); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("%s: want InvalidArgument, got %v", name, err)
		}
	}
	// No credentials at all is Unauthenticated, not a validation error.
	_, err := f.h.RegisterDevice(f.ctx(), connect.NewRequest(registerMsg("d", "tok")))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

// TestGetProjectConfigCarriesPush covers the public delivery path: the VAPID
// public key and enabled flag flow into moth.auth.v1.GetProjectConfig.
func TestGetProjectConfigCarriesPush(t *testing.T) {
	f := newFixture(t)
	vapid := "BPzsDKrifYqlYqPh6474mZZjIF0oJVdcz2iZ6ZCwCUvhHpJhqRZLnLWkOro9MPKC8i8Bik9jRWFj-DBK7ZzX6Tk"
	raw, err := push.Encode(push.Config{Version: push.SchemaVersion, Enabled: true, WebPushVAPIDPublicKey: vapid})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.st.SetProjectPush(context.Background(), f.project.ID, raw, f.now); err != nil {
		t.Fatal(err)
	}
	if f.project, err = f.st.GetProject(context.Background(), f.project.ID); err != nil {
		t.Fatal(err)
	}

	resp, err := f.authH.GetProjectConfig(f.ctx(), connect.NewRequest(&authv1.GetProjectConfigRequest{}))
	if err != nil {
		t.Fatalf("GetProjectConfig: %v", err)
	}
	pc := resp.Msg.Push
	if pc == nil || !pc.Enabled || pc.WebpushVapidPublicKey != vapid {
		t.Fatalf("push config: %+v", pc)
	}
}

// TestRegisterDeviceRateLimited drives the mounted service through the real
// publishable-key + rate-limit interceptor chain (as server.New wires it) and
// asserts RegisterDevice is throttled per IP like the other credential-facing
// RPCs.
func TestRegisterDeviceRateLimited(t *testing.T) {
	f := newFixture(t)
	proxies, err := netutil.ParseTrustedProxies(nil)
	if err != nil {
		t.Fatal(err)
	}
	limiter := ratelimit.New(f.st, ratelimit.Config{
		IP: ratelimit.Tier{Limit: 2, Window: time.Minute},
	}, proxies, func() time.Time { return f.now })

	mux := http.NewServeMux()
	path, handler := pushv1connect.NewPushServiceHandler(f.h, connect.WithInterceptors(
		authrpc.NewProjectInterceptor(f.st),
		authrpc.NewRateLimitInterceptor(limiter, nil)))
	mux.Handle(path, handler)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	client := pushv1connect.NewPushServiceClient(ts.Client(), ts.URL)

	var last error
	for i := 0; i < 5; i++ {
		req := authReq(f.access, registerMsg("device-1", "tok-1"))
		req.Header().Set("X-Moth-Key", f.project.PublishableKey)
		_, last = client.RegisterDevice(context.Background(), req)
		if connect.CodeOf(last) == connect.CodeResourceExhausted {
			break
		}
		if last != nil {
			t.Fatalf("call %d: %v", i, last)
		}
	}
	if connect.CodeOf(last) != connect.CodeResourceExhausted {
		t.Fatalf("want ResourceExhausted, got %v", last)
	}
}
