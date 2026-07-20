package adminrpc

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/audit"
	"github.com/aloisdeniel/moth/internal/mail"
	"github.com/aloisdeniel/moth/internal/password"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
	"github.com/aloisdeniel/moth/internal/version"
)

// newPATTestServer mounts the session and account services behind the real
// auth interceptor, exactly as server.New wires them.
func newPATTestServer(t *testing.T) (*store.Store, *httptest.Server) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}

	interceptors := connect.WithInterceptors(NewAuthInterceptor(st))
	mux := http.NewServeMux()
	sessPath, sessHandler := adminv1connect.NewSessionServiceHandler(
		NewSessionHandler(st, false), interceptors)
	mux.Handle(sessPath, sessHandler)
	acctPath, acctHandler := adminv1connect.NewAdminAccountServiceHandler(
		NewAccountHandler(st, mail.Console{Log: slog.Default()}, "http://moth.test",
			false, func() bool { return false },
			NewAuditor(audit.New(st, slog.Default(), nil))), interceptors)
	mux.Handle(acctPath, acctHandler)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return st, srv
}

func createPATTestAdmin(t *testing.T, st *store.Store, email, pass string) store.Admin {
	t.Helper()
	hash, err := password.Hash(pass)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	admin := store.Admin{ID: NewID(), Email: email, PasswordHash: hash, CreatedAt: now, UpdatedAt: now}
	if err := st.CreateAdmin(context.Background(), admin); err != nil {
		t.Fatal(err)
	}
	return admin
}

// A PAT minted over a cookie session must authenticate existing admin RPCs
// as Bearer metadata, die on revocation, and leave the cookie path intact.
func TestPATAuthenticatesAdminRPCs(t *testing.T) {
	st, srv := newPATTestServer(t)
	ctx := context.Background()
	admin := createPATTestAdmin(t, st, "ops@example.com", "correct horse")

	sessions := adminv1connect.NewSessionServiceClient(srv.Client(), srv.URL)
	account := adminv1connect.NewAdminAccountServiceClient(srv.Client(), srv.URL)

	login, err := sessions.Login(ctx, connect.NewRequest(&adminv1.LoginRequest{
		Email: admin.Email, Password: "correct horse",
	}))
	if err != nil {
		t.Fatal(err)
	}
	cookie, _, _ := strings.Cut(login.Header().Get("Set-Cookie"), ";")

	// Mint a PAT over the cookie session.
	createReq := connect.NewRequest(&adminv1.CreatePersonalAccessTokenRequest{
		Name: "ci", ExpiresInDays: 30,
	})
	createReq.Header().Set("Cookie", cookie)
	created, err := account.CreatePersonalAccessToken(ctx, createReq)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(created.Msg.Token, token.PATPrefix) {
		t.Fatalf("token %q should carry the %s prefix", created.Msg.Token, token.PATPrefix)
	}
	if created.Msg.Metadata.GetExpireTime() == nil {
		t.Fatal("expires_in_days should set expire_time")
	}
	bearer := "Bearer " + created.Msg.Token

	// An existing admin RPC over the PAT alone: same identity as the cookie.
	whoReq := connect.NewRequest(&adminv1.GetCurrentAdminRequest{})
	whoReq.Header().Set("Authorization", bearer)
	who, err := sessions.GetCurrentAdmin(ctx, whoReq)
	if err != nil {
		t.Fatalf("GetCurrentAdmin over PAT: %v", err)
	}
	if who.Msg.Admin.Id != admin.ID {
		t.Fatalf("PAT authenticated as %q, want %q", who.Msg.Admin.Id, admin.ID)
	}
	if who.Msg.ServerVersion != version.Version {
		t.Fatalf("server_version = %q, want %q", who.Msg.ServerVersion, version.Version)
	}

	// The PAT manages PATs too, and its use was just recorded.
	listReq := connect.NewRequest(&adminv1.ListPersonalAccessTokensRequest{})
	listReq.Header().Set("Authorization", bearer)
	list, err := account.ListPersonalAccessTokens(ctx, listReq)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Msg.Tokens) != 1 || list.Msg.Tokens[0].Id != created.Msg.Metadata.Id {
		t.Fatalf("list mismatch: %+v", list.Msg.Tokens)
	}
	if list.Msg.Tokens[0].GetLastUsedTime() == nil {
		t.Fatal("PAT authentication should bump last_used_time")
	}

	// An unknown token fails closed.
	badReq := connect.NewRequest(&adminv1.GetCurrentAdminRequest{})
	badReq.Header().Set("Authorization", "Bearer "+token.New(token.PATPrefix))
	if _, err := sessions.GetCurrentAdmin(ctx, badReq); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unknown PAT: want unauthenticated, got %v", err)
	}

	// Revoking over the cookie kills the PAT's next call immediately.
	revokeReq := connect.NewRequest(&adminv1.RevokePersonalAccessTokenRequest{
		Id: created.Msg.Metadata.Id,
	})
	revokeReq.Header().Set("Cookie", cookie)
	if _, err := account.RevokePersonalAccessToken(ctx, revokeReq); err != nil {
		t.Fatal(err)
	}
	deadReq := connect.NewRequest(&adminv1.GetCurrentAdminRequest{})
	deadReq.Header().Set("Authorization", bearer)
	if _, err := sessions.GetCurrentAdmin(ctx, deadReq); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("revoked PAT: want unauthenticated, got %v", err)
	}

	// The cookie path is untouched.
	cookieReq := connect.NewRequest(&adminv1.GetCurrentAdminRequest{})
	cookieReq.Header().Set("Cookie", cookie)
	if _, err := sessions.GetCurrentAdmin(ctx, cookieReq); err != nil {
		t.Fatalf("cookie session broken: %v", err)
	}
}

func TestExpiredPATRejected(t *testing.T) {
	st, srv := newPATTestServer(t)
	ctx := context.Background()
	admin := createPATTestAdmin(t, st, "ops@example.com", "correct horse")

	plain := token.New(token.PATPrefix)
	past := time.Now().Add(-time.Hour)
	if err := st.CreatePAT(ctx, store.PersonalAccessToken{
		ID: NewID(), AdminID: admin.ID, Name: "old", TokenHash: token.Hash(plain),
		CreatedAt: past.Add(-time.Hour), ExpiresAt: &past,
	}); err != nil {
		t.Fatal(err)
	}

	sessions := adminv1connect.NewSessionServiceClient(srv.Client(), srv.URL)
	req := connect.NewRequest(&adminv1.GetCurrentAdminRequest{})
	req.Header().Set("Authorization", "Bearer "+plain)
	if _, err := sessions.GetCurrentAdmin(ctx, req); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expired PAT: want unauthenticated, got %v", err)
	}
}

// ChangePassword over a PAT has no cookie session to spare, so it must end
// every browser session — the CLI password rotation is the "my browser
// session was stolen" recovery path.
func TestChangePasswordOverPATEndsAllSessions(t *testing.T) {
	st, srv := newPATTestServer(t)
	ctx := context.Background()
	admin := createPATTestAdmin(t, st, "ops@example.com", "correct horse")

	sessions := adminv1connect.NewSessionServiceClient(srv.Client(), srv.URL)
	account := adminv1connect.NewAdminAccountServiceClient(srv.Client(), srv.URL)

	// The (possibly stolen) browser session.
	login, err := sessions.Login(ctx, connect.NewRequest(&adminv1.LoginRequest{
		Email: admin.Email, Password: "correct horse",
	}))
	if err != nil {
		t.Fatal(err)
	}
	cookie, _, _ := strings.Cut(login.Header().Get("Set-Cookie"), ";")

	// A PAT minted directly in the store (the `moth admin token` path).
	plain := token.New(token.PATPrefix)
	if err := st.CreatePAT(ctx, store.PersonalAccessToken{
		ID: NewID(), AdminID: admin.ID, Name: "cli", TokenHash: token.Hash(plain),
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	changeReq := connect.NewRequest(&adminv1.ChangePasswordRequest{
		CurrentPassword: "correct horse", NewPassword: "battery staple",
	})
	changeReq.Header().Set("Authorization", "Bearer "+plain)
	if _, err := account.ChangePassword(ctx, changeReq); err != nil {
		t.Fatalf("ChangePassword over PAT: %v", err)
	}

	// The stolen browser session is dead immediately, not at natural expiry.
	cookieReq := connect.NewRequest(&adminv1.GetCurrentAdminRequest{})
	cookieReq.Header().Set("Cookie", cookie)
	if _, err := sessions.GetCurrentAdmin(ctx, cookieReq); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("browser session should be ended, got %v", err)
	}
	// The PAT itself still works (PATs are managed separately).
	whoReq := connect.NewRequest(&adminv1.GetCurrentAdminRequest{})
	whoReq.Header().Set("Authorization", "Bearer "+plain)
	if _, err := sessions.GetCurrentAdmin(ctx, whoReq); err != nil {
		t.Fatalf("PAT should survive the rotation: %v", err)
	}
}

// A token minted over a PAT must never outlive its creator: a leaked
// 7-day CI token must not be laundered into a non-expiring one.
func TestPATMintedTokenCappedAtCreatorExpiry(t *testing.T) {
	st, srv := newPATTestServer(t)
	ctx := context.Background()
	admin := createPATTestAdmin(t, st, "ops@example.com", "correct horse")
	account := adminv1connect.NewAdminAccountServiceClient(srv.Client(), srv.URL)

	creatorExpiry := time.Now().Add(7 * 24 * time.Hour)
	plain := token.New(token.PATPrefix)
	if err := st.CreatePAT(ctx, store.PersonalAccessToken{
		ID: NewID(), AdminID: admin.ID, Name: "ci", TokenHash: token.Hash(plain),
		CreatedAt: time.Now(), ExpiresAt: &creatorExpiry,
	}); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name          string
		expiresInDays int32
	}{
		{"never-expiring request is capped", 0},
		{"longer-lived request is capped", 365},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := connect.NewRequest(&adminv1.CreatePersonalAccessTokenRequest{
				Name: "laundered", ExpiresInDays: tc.expiresInDays,
			})
			req.Header().Set("Authorization", "Bearer "+plain)
			created, err := account.CreatePersonalAccessToken(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			expire := created.Msg.Metadata.GetExpireTime()
			if expire == nil {
				t.Fatal("PAT-minted token must not be non-expiring")
			}
			if expire.AsTime().After(creatorExpiry) {
				t.Fatalf("minted token expires %v, after its creator %v", expire.AsTime(), creatorExpiry)
			}
		})
	}

	// A shorter-lived request keeps its own expiry.
	req := connect.NewRequest(&adminv1.CreatePersonalAccessTokenRequest{
		Name: "short", ExpiresInDays: 1,
	})
	req.Header().Set("Authorization", "Bearer "+plain)
	created, err := account.CreatePersonalAccessToken(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	expire := created.Msg.Metadata.GetExpireTime()
	if expire == nil || expire.AsTime().After(time.Now().Add(48*time.Hour)) {
		t.Fatalf("short-lived request should keep its own expiry, got %v", expire)
	}
}

func TestCreatePersonalAccessTokenValidation(t *testing.T) {
	st, srv := newPATTestServer(t)
	ctx := context.Background()
	admin := createPATTestAdmin(t, st, "ops@example.com", "correct horse")

	sessions := adminv1connect.NewSessionServiceClient(srv.Client(), srv.URL)
	login, err := sessions.Login(ctx, connect.NewRequest(&adminv1.LoginRequest{
		Email: admin.Email, Password: "correct horse",
	}))
	if err != nil {
		t.Fatal(err)
	}
	cookie, _, _ := strings.Cut(login.Header().Get("Set-Cookie"), ";")
	account := adminv1connect.NewAdminAccountServiceClient(srv.Client(), srv.URL)

	cases := []struct {
		name string
		msg  *adminv1.CreatePersonalAccessTokenRequest
	}{
		{"empty name", &adminv1.CreatePersonalAccessTokenRequest{Name: "  "}},
		{"name too long", &adminv1.CreatePersonalAccessTokenRequest{Name: strings.Repeat("x", maxPATNameLen+1)}},
		{"negative expiry", &adminv1.CreatePersonalAccessTokenRequest{Name: "ok", ExpiresInDays: -1}},
	}
	for _, tc := range cases {
		req := connect.NewRequest(tc.msg)
		req.Header().Set("Cookie", cookie)
		if _, err := account.CreatePersonalAccessToken(ctx, req); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("%s: want invalid argument, got %v", tc.name, err)
		}
	}
}
