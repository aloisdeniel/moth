package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	gojwt "github.com/golang-jwt/jwt/v5"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/types/known/structpb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/gen/moth/auth/v1/authv1connect"
	serverv1 "github.com/aloisdeniel/moth/gen/moth/server/v1"
	"github.com/aloisdeniel/moth/gen/moth/server/v1/serverv1connect"
	"github.com/aloisdeniel/moth/internal/mail"
	"github.com/aloisdeniel/moth/internal/ratelimit"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
)

// captureMailer records every email instead of sending it.
type captureMailer struct {
	mu   sync.Mutex
	msgs []mail.Message
}

func (m *captureMailer) Send(_ context.Context, msg mail.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
	return nil
}

// lastTo returns the most recent email sent to addr.
func (m *captureMailer) lastTo(t *testing.T, addr string) mail.Message {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.msgs) - 1; i >= 0; i-- {
		if m.msgs[i].To == addr {
			return m.msgs[i]
		}
	}
	t.Fatalf("no email was sent to %s", addr)
	return mail.Message{}
}

func (m *captureMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.msgs)
}

var linkRE = regexp.MustCompile(`https?://\S+`)

// lastLinkTo extracts the action link from the most recent email to addr,
// rewritten to hit the test server instead of the configured base URL.
func (e *testEnv) lastLinkTo(t *testing.T, addr string) string {
	t.Helper()
	msg := e.mails.lastTo(t, addr)
	link := linkRE.FindString(msg.Text)
	if link == "" {
		t.Fatalf("email to %s carries no link:\n%s", addr, msg.Text)
	}
	return strings.Replace(link, "http://localhost:8080", e.url, 1)
}

// keyTransport adds the project API key to every request.
type keyTransport struct{ key string }

func (kt keyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("x-moth-key", kt.key)
	return http.DefaultTransport.RoundTrip(r)
}

func (e *testEnv) authClient(publishableKey string) authv1connect.AuthServiceClient {
	hc := &http.Client{Transport: keyTransport{publishableKey}}
	return authv1connect.NewAuthServiceClient(hc, e.url)
}

func (e *testEnv) tokenClient(secretKey string) serverv1connect.TokenServiceClient {
	hc := &http.Client{Transport: keyTransport{secretKey}}
	return serverv1connect.NewTokenServiceClient(hc, e.url)
}

func (e *testEnv) userClient(secretKey string) serverv1connect.UserServiceClient {
	hc := &http.Client{Transport: keyTransport{secretKey}}
	return serverv1connect.NewUserServiceClient(hc, e.url)
}

func (e *testEnv) adminUsers() adminv1connect.UserServiceClient {
	return adminv1connect.NewUserServiceClient(e.client, e.url)
}

// createProject provisions a project through the admin API and returns it
// with its secret key.
func (e *testEnv) createProject(t *testing.T, name string) (*adminv1.Project, string) {
	t.Helper()
	created, err := e.projects.CreateProject(context.Background(),
		connect.NewRequest(&adminv1.CreateProjectRequest{Name: name}))
	if err != nil {
		t.Fatal(err)
	}
	return created.Msg.Project, created.Msg.SecretKey
}

// updateSettings mutates a project's settings through the admin API.
func (e *testEnv) updateSettings(t *testing.T, p *adminv1.Project, mutate func(*adminv1.ProjectSettings)) {
	t.Helper()
	settings := p.Settings
	mutate(settings)
	_, err := e.projects.UpdateProject(context.Background(),
		connect.NewRequest(&adminv1.UpdateProjectRequest{Id: p.Id, Name: p.Name, Settings: settings}))
	if err != nil {
		t.Fatal(err)
	}
}

// bearer builds a request carrying an access token.
func bearer[T any](msg *T, accessToken string) *connect.Request[T] {
	r := connect.NewRequest(msg)
	r.Header().Set("Authorization", "Bearer "+accessToken)
	return r
}

func wantReason(t *testing.T, err error, code connect.Code, reason string) {
	t.Helper()
	if connect.CodeOf(err) != code {
		t.Fatalf("code = %v (err %v), want %v", connect.CodeOf(err), err, code)
	}
	if got := authrpc.ErrorReason(err); got != reason {
		t.Fatalf("reason = %q (err %v), want %q", got, err, reason)
	}
}

// getPage fetches a hosted page and returns its body.
func (e *testEnv) getPage(t *testing.T, link string) string {
	t.Helper()
	resp, err := e.client.Get(link)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: %d\n%s", link, resp.StatusCode, body)
	}
	return string(body)
}

// TestAuthLifecycleE2E walks the acceptance scenario end to end over real
// HTTP: SignUp → verify via emailed link → SignIn → GetMe → RefreshToken →
// ChangePassword → old refresh rejected → reset flow → SignIn with the new
// password.
func TestAuthLifecycleE2E(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Lifecycle App")
	auth := e.authClient(p.PublishableKey)
	ctx := context.Background()

	// SignUp normalizes the email and (default policy) signs the user in.
	su, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "  Jane@Example.COM ", Password: "password-1", DisplayName: "Jane", DeviceInfo: "test device",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if su.Msg.User.Email != "jane@example.com" {
		t.Fatalf("email not normalized: %q", su.Msg.User.Email)
	}
	if su.Msg.Tokens == nil || su.Msg.Tokens.AccessToken == "" || su.Msg.Tokens.RefreshToken == "" {
		t.Fatal("default policy should sign the user in on signup")
	}
	if su.Msg.User.EmailVerified {
		t.Fatal("email must start unverified")
	}

	// Verify through the emailed link (hosted page).
	body := e.getPage(t, e.lastLinkTo(t, "jane@example.com"))
	if !strings.Contains(body, "verified") {
		t.Fatalf("verify page: %s", body)
	}

	// SignIn.
	si, err := auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
		Email: "jane@example.com", Password: "password-1",
	}))
	if err != nil {
		t.Fatal(err)
	}

	// GetMe via bearer token reflects the verification.
	me, err := auth.GetMe(ctx, bearer(&authv1.GetMeRequest{}, si.Msg.Tokens.AccessToken))
	if err != nil {
		t.Fatal(err)
	}
	if !me.Msg.User.EmailVerified || me.Msg.User.DisplayName != "Jane" {
		t.Fatalf("GetMe: %+v", me.Msg.User)
	}

	// UpdateMe.
	name := "Jane D."
	upd, err := auth.UpdateMe(ctx, bearer(&authv1.UpdateMeRequest{DisplayName: &name}, si.Msg.Tokens.AccessToken))
	if err != nil || upd.Msg.User.DisplayName != "Jane D." {
		t.Fatalf("UpdateMe: %v %+v", err, upd)
	}

	// RefreshToken rotates.
	rt, err := auth.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{
		RefreshToken: si.Msg.Tokens.RefreshToken,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if rt.Msg.Tokens.RefreshToken == si.Msg.Tokens.RefreshToken {
		t.Fatal("refresh token must rotate")
	}

	// ChangePassword revokes other sessions and returns a fresh pair.
	cp, err := auth.ChangePassword(ctx, bearer(&authv1.ChangePasswordRequest{
		CurrentPassword: "password-1", NewPassword: "password-2",
	}, rt.Msg.Tokens.AccessToken))
	if err != nil {
		t.Fatal(err)
	}
	// The pre-change refresh token is dead.
	_, err = auth.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{
		RefreshToken: rt.Msg.Tokens.RefreshToken,
	}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("old refresh after password change: %v", err)
	}
	// The fresh one lives.
	if _, err := auth.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{
		RefreshToken: cp.Msg.Tokens.RefreshToken,
	})); err != nil {
		t.Fatalf("fresh refresh after password change: %v", err)
	}

	// Password reset flow through the hosted form.
	if _, err := auth.RequestPasswordReset(ctx, connect.NewRequest(&authv1.RequestPasswordResetRequest{
		Email: "jane@example.com",
	})); err != nil {
		t.Fatal(err)
	}
	link := e.lastLinkTo(t, "jane@example.com")
	if !strings.Contains(e.getPage(t, link), "form") {
		t.Fatal("reset page should show the password form")
	}
	u, _ := url.Parse(link)
	resp, err := e.client.PostForm(strings.TrimSuffix(link, "?token="+u.Query().Get("token")),
		url.Values{"token": {u.Query().Get("token")}, "password": {"password-3"}})
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(page), "has been reset") {
		t.Fatalf("reset submit page:\n%s", page)
	}

	// Old password out, new password in; reset revoked all sessions.
	_, err = auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
		Email: "jane@example.com", Password: "password-2"}))
	wantReason(t, err, connect.CodeUnauthenticated, authrpc.ReasonInvalidCredentials)
	_, err = auth.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{
		RefreshToken: cp.Msg.Tokens.RefreshToken}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("refresh after reset: %v", err)
	}
	if _, err := auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
		Email: "jane@example.com", Password: "password-3"})); err != nil {
		t.Fatalf("sign in with reset password: %v", err)
	}
}

func TestRefreshTokenReuseRevokesFamily(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "App")
	auth := e.authClient(p.PublishableKey)
	ctx := context.Background()

	su, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "u@example.com", Password: "password-1"}))
	if err != nil {
		t.Fatal(err)
	}
	first := su.Msg.Tokens.RefreshToken

	rt, err := auth.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{RefreshToken: first}))
	if err != nil {
		t.Fatal(err)
	}
	second := rt.Msg.Tokens.RefreshToken

	// Replaying the rotated token is theft evidence.
	_, err = auth.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{RefreshToken: first}))
	wantReason(t, err, connect.CodeUnauthenticated, authrpc.ReasonRefreshTokenReused)

	// The whole family died with it, including the live successor.
	_, err = auth.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{RefreshToken: second}))
	wantReason(t, err, connect.CodeUnauthenticated, authrpc.ReasonInvalidRefreshToken)

	// A separate sign-in (new family) is unaffected by the revocation.
	si, err := auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
		Email: "u@example.com", Password: "password-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{
		RefreshToken: si.Msg.Tokens.RefreshToken})); err != nil {
		t.Fatalf("independent family: %v", err)
	}
}

func TestProjectsAreIndependentTenants(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p1, _ := e.createProject(t, "App One")
	p2, _ := e.createProject(t, "App Two")
	auth1 := e.authClient(p1.PublishableKey)
	auth2 := e.authClient(p2.PublishableKey)
	ctx := context.Background()

	// The same email signs up independently in both projects.
	su1, err := auth1.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "same@example.com", Password: "password-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth2.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "same@example.com", Password: "different-2"})); err != nil {
		t.Fatalf("same email in project 2: %v", err)
	}

	// Project 1's password does not open project 2's account.
	_, err = auth2.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
		Email: "same@example.com", Password: "password-1"}))
	wantReason(t, err, connect.CodeUnauthenticated, authrpc.ReasonInvalidCredentials)

	// Project 1's refresh token means nothing to project 2.
	_, err = auth2.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{
		RefreshToken: su1.Msg.Tokens.RefreshToken}))
	wantReason(t, err, connect.CodeUnauthenticated, authrpc.ReasonInvalidRefreshToken)

	// Project 1's access token is rejected by project 2.
	_, err = auth2.GetMe(ctx, bearer(&authv1.GetMeRequest{}, su1.Msg.Tokens.AccessToken))
	wantReason(t, err, connect.CodeUnauthenticated, authrpc.ReasonInvalidAccessToken)
}

// jwksKeys fetches a project's JWKS and returns kid → public key.
func (e *testEnv) jwksKeys(t *testing.T, slug string) map[string]*ecdsa.PublicKey {
	t.Helper()
	resp, err := e.client.Get(e.url + "/p/" + slug + "/.well-known/jwks.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var jwks struct {
		Keys []struct{ Kty, Crv, X, Y, Kid string }
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		t.Fatal(err)
	}
	out := make(map[string]*ecdsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		xb, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			t.Fatal(err)
		}
		yb, err := base64.RawURLEncoding.DecodeString(k.Y)
		if err != nil {
			t.Fatal(err)
		}
		out[k.Kid] = &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(xb),
			Y:     new(big.Int).SetBytes(yb),
		}
	}
	return out
}

// TestJWTValidatesAgainstJWKSWithThirdPartyLibrary verifies moth's tokens
// with golang-jwt (an independent JOSE implementation) against the served
// JWKS — and that another project's JWKS rejects them.
func TestJWTValidatesAgainstJWKSWithThirdPartyLibrary(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p1, _ := e.createProject(t, "App One")
	p2, _ := e.createProject(t, "App Two")
	ctx := context.Background()

	su, err := e.authClient(p1.PublishableKey).SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "u@example.com", Password: "password-1"}))
	if err != nil {
		t.Fatal(err)
	}
	access := su.Msg.Tokens.AccessToken

	keyfunc := func(keys map[string]*ecdsa.PublicKey) gojwt.Keyfunc {
		return func(tok *gojwt.Token) (any, error) {
			kid, _ := tok.Header["kid"].(string)
			key, ok := keys[kid]
			if !ok {
				return nil, gojwt.ErrTokenUnverifiable
			}
			return key, nil
		}
	}

	parsed, err := gojwt.Parse(access, keyfunc(e.jwksKeys(t, p1.Slug)),
		gojwt.WithValidMethods([]string{"ES256"}),
		gojwt.WithAudience(p1.Slug),
		gojwt.WithIssuer("http://localhost:8080/p/"+p1.Slug),
		gojwt.WithExpirationRequired(),
	)
	if err != nil || !parsed.Valid {
		t.Fatalf("third-party validation failed: %v", err)
	}
	claims := parsed.Claims.(gojwt.MapClaims)
	if claims["sub"] != su.Msg.User.Id || claims["email"] != "u@example.com" {
		t.Fatalf("claims: %+v", claims)
	}

	// The same token must fail against the other project's JWKS.
	if _, err := gojwt.Parse(access, keyfunc(e.jwksKeys(t, p2.Slug)),
		gojwt.WithValidMethods([]string{"ES256"})); err == nil {
		t.Fatal("token validated against another project's JWKS")
	}
}

func TestIntrospectToken(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p1, sk1 := e.createProject(t, "App One")
	_, sk2 := e.createProject(t, "App Two")
	ctx := context.Background()

	su, err := e.authClient(p1.PublishableKey).SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "u@example.com", Password: "password-1"}))
	if err != nil {
		t.Fatal(err)
	}
	access := su.Msg.Tokens.AccessToken

	// Valid sk of the right project → active with claims.
	intro, err := e.tokenClient(sk1).IntrospectToken(ctx, connect.NewRequest(
		&serverv1.IntrospectTokenRequest{AccessToken: access}))
	if err != nil {
		t.Fatal(err)
	}
	if !intro.Msg.Active || intro.Msg.UserId != su.Msg.User.Id || intro.Msg.Email != "u@example.com" {
		t.Fatalf("introspection: %+v", intro.Msg)
	}

	// Wrong project's secret key → PERMISSION_DENIED.
	_, err = e.tokenClient(sk2).IntrospectToken(ctx, connect.NewRequest(
		&serverv1.IntrospectTokenRequest{AccessToken: access}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("wrong sk: %v, want permission_denied", err)
	}

	// Unknown secret key → unauthenticated.
	_, err = e.tokenClient("sk_bogus").IntrospectToken(ctx, connect.NewRequest(
		&serverv1.IntrospectTokenRequest{AccessToken: access}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("bogus sk: %v, want unauthenticated", err)
	}

	// Garbage token → inactive, not an error.
	garbage, err := e.tokenClient(sk1).IntrospectToken(ctx, connect.NewRequest(
		&serverv1.IntrospectTokenRequest{AccessToken: "not-a-jwt"}))
	if err != nil || garbage.Msg.Active {
		t.Fatalf("garbage token: %v %+v", err, garbage)
	}

	// A disabled user's still-unexpired JWT introspects as inactive.
	if _, err := e.userClient(sk1).DisableUser(ctx, connect.NewRequest(
		&serverv1.DisableUserRequest{UserId: su.Msg.User.Id})); err != nil {
		t.Fatal(err)
	}
	intro2, err := e.tokenClient(sk1).IntrospectToken(ctx, connect.NewRequest(
		&serverv1.IntrospectTokenRequest{AccessToken: access}))
	if err != nil {
		t.Fatal(err)
	}
	if intro2.Msg.Active || intro2.Msg.InactiveReason != "USER_DISABLED" {
		t.Fatalf("disabled user introspection: %+v", intro2.Msg)
	}
}

func TestCustomClaimsFlowIntoRefreshedJWT(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, sk := e.createProject(t, "App")
	auth := e.authClient(p.PublishableKey)
	ctx := context.Background()

	su, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "u@example.com", Password: "password-1"}))
	if err != nil {
		t.Fatal(err)
	}

	claims, _ := structpb.NewStruct(map[string]any{"role": "admin", "tier": "gold"})
	if _, err := e.userClient(sk).UpdateUser(ctx, connect.NewRequest(&serverv1.UpdateUserRequest{
		UserId: su.Msg.User.Id, CustomClaims: claims})); err != nil {
		t.Fatal(err)
	}

	// The pre-update access token has no custom claims; the refreshed one
	// does.
	rt, err := auth.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{
		RefreshToken: su.Msg.Tokens.RefreshToken}))
	if err != nil {
		t.Fatal(err)
	}
	parsed, _, err := gojwt.NewParser().ParseUnverified(rt.Msg.Tokens.AccessToken, gojwt.MapClaims{})
	if err != nil {
		t.Fatal(err)
	}
	custom, _ := parsed.Claims.(gojwt.MapClaims)["claims"].(map[string]any)
	if custom["role"] != "admin" || custom["tier"] != "gold" {
		t.Fatalf("custom claims in refreshed JWT: %+v", custom)
	}

	// Introspection reports them too.
	intro, err := e.tokenClient(sk).IntrospectToken(ctx, connect.NewRequest(
		&serverv1.IntrospectTokenRequest{AccessToken: rt.Msg.Tokens.AccessToken}))
	if err != nil || !intro.Msg.Active {
		t.Fatalf("introspect: %v %+v", err, intro)
	}
	if intro.Msg.CustomClaims.Fields["role"].GetStringValue() != "admin" {
		t.Fatalf("introspected claims: %+v", intro.Msg.CustomClaims)
	}
}

func TestEmailChangeRoundTripWithRevert(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "App")
	auth := e.authClient(p.PublishableKey)
	ctx := context.Background()

	su, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "old@example.com", Password: "password-1"}))
	if err != nil {
		t.Fatal(err)
	}
	access := su.Msg.Tokens.AccessToken

	// Request the change; the confirmation goes to the NEW address.
	if _, err := auth.RequestEmailChange(ctx, bearer(&authv1.RequestEmailChangeRequest{
		NewEmail: "new@example.com"}, access)); err != nil {
		t.Fatal(err)
	}
	e.getPage(t, e.lastLinkTo(t, "new@example.com"))

	me, err := auth.GetMe(ctx, bearer(&authv1.GetMeRequest{}, access))
	if err != nil {
		t.Fatal(err)
	}
	if me.Msg.User.Email != "new@example.com" || !me.Msg.User.EmailVerified {
		t.Fatalf("after change: %+v", me.Msg.User)
	}

	// Tokens reflect the new email on the next refresh.
	rt, err := auth.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{
		RefreshToken: su.Msg.Tokens.RefreshToken}))
	if err != nil {
		t.Fatal(err)
	}
	parsed, _, err := gojwt.NewParser().ParseUnverified(rt.Msg.Tokens.AccessToken, gojwt.MapClaims{})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Claims.(gojwt.MapClaims)["email"] != "new@example.com" {
		t.Fatal("refreshed JWT should carry the new email")
	}

	// The old address got a notice with a revert link; use it.
	notice := e.mails.lastTo(t, "old@example.com")
	if !strings.Contains(notice.Text, "new@example.com") {
		t.Fatalf("notice does not name the new address:\n%s", notice.Text)
	}
	e.getPage(t, e.lastLinkTo(t, "old@example.com"))

	me2, err := auth.GetMe(ctx, bearer(&authv1.GetMeRequest{}, access))
	if err != nil {
		t.Fatal(err)
	}
	if me2.Msg.User.Email != "old@example.com" {
		t.Fatalf("revert did not restore the email: %+v", me2.Msg.User)
	}
	// The revert assumed compromise: every session is gone.
	_, err = auth.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{
		RefreshToken: rt.Msg.Tokens.RefreshToken}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("refresh after revert: %v", err)
	}

	// Changing to an address that is already an account is refused.
	if _, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "taken@example.com", Password: "password-1"})); err != nil {
		t.Fatal(err)
	}
	_, err = auth.RequestEmailChange(ctx, bearer(&authv1.RequestEmailChangeRequest{
		NewEmail: "taken@example.com"}, access))
	wantReason(t, err, connect.CodeAlreadyExists, authrpc.ReasonEmailAlreadyExists)
}

func TestDeleteAccountRequiresFreshReauth(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, sk := e.createProject(t, "App")
	auth := e.authClient(p.PublishableKey)
	ctx := context.Background()

	su, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "u@example.com", Password: "password-1"}))
	if err != nil {
		t.Fatal(err)
	}
	access := su.Msg.Tokens.AccessToken

	// Wrong password → rejected, account intact.
	_, err = auth.DeleteAccount(ctx, bearer(&authv1.DeleteAccountRequest{Password: "wrong"}, access))
	wantReason(t, err, connect.CodeUnauthenticated, authrpc.ReasonInvalidCredentials)

	// Correct password → user and sessions are gone.
	if _, err := auth.DeleteAccount(ctx, bearer(&authv1.DeleteAccountRequest{Password: "password-1"}, access)); err != nil {
		t.Fatal(err)
	}
	_, err = e.userClient(sk).GetUser(ctx, connect.NewRequest(&serverv1.GetUserRequest{UserId: su.Msg.User.Id}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("user after deletion: %v", err)
	}
	_, err = auth.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{
		RefreshToken: su.Msg.Tokens.RefreshToken}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("refresh after deletion: %v", err)
	}
	_, err = auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
		Email: "u@example.com", Password: "password-1"}))
	wantReason(t, err, connect.CodeUnauthenticated, authrpc.ReasonInvalidCredentials)
}

func TestAdminUserServiceListAndDisable(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "App")
	auth := e.authClient(p.PublishableKey)
	admins := e.adminUsers()
	ctx := context.Background()

	su, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "u@example.com", Password: "password-1"}))
	if err != nil {
		t.Fatal(err)
	}

	list, err := admins.ListUsers(ctx, connect.NewRequest(&adminv1.ListUsersRequest{ProjectId: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Msg.Users) != 1 || list.Msg.Users[0].Email != "u@example.com" {
		t.Fatalf("admin list: %+v", list.Msg.Users)
	}

	// Disable blocks sign-in and refresh.
	dis, err := admins.DisableUser(ctx, connect.NewRequest(&adminv1.DisableUserRequest{
		ProjectId: p.Id, UserId: su.Msg.User.Id}))
	if err != nil || !dis.Msg.User.Disabled {
		t.Fatalf("disable: %v %+v", err, dis)
	}
	_, err = auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
		Email: "u@example.com", Password: "password-1"}))
	wantReason(t, err, connect.CodePermissionDenied, authrpc.ReasonUserDisabled)
	_, err = auth.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{
		RefreshToken: su.Msg.Tokens.RefreshToken}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("refresh while disabled: %v", err)
	}

	// Enable lets the user back in.
	if _, err := admins.EnableUser(ctx, connect.NewRequest(&adminv1.EnableUserRequest{
		ProjectId: p.Id, UserId: su.Msg.User.Id})); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
		Email: "u@example.com", Password: "password-1"})); err != nil {
		t.Fatalf("sign in after enable: %v", err)
	}

	// Admin user RPCs require a session.
	anon := adminv1connect.NewUserServiceClient(http.DefaultClient, e.url)
	_, err = anon.ListUsers(ctx, connect.NewRequest(&adminv1.ListUsersRequest{ProjectId: p.Id}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("anonymous admin list: %v", err)
	}
}

func TestSignupPolicies(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	ctx := context.Background()

	t.Run("closed signup", func(t *testing.T) {
		p, _ := e.createProject(t, "Invite Only")
		e.updateSettings(t, p, func(s *adminv1.ProjectSettings) { s.AllowPublicSignup = false })
		_, err := e.authClient(p.PublishableKey).SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
			Email: "u@example.com", Password: "password-1"}))
		wantReason(t, err, connect.CodePermissionDenied, authrpc.ReasonSignupClosed)
	})

	t.Run("weak password", func(t *testing.T) {
		p, _ := e.createProject(t, "Strict PW")
		e.updateSettings(t, p, func(s *adminv1.ProjectSettings) { s.PasswordMinLength = 12 })
		auth := e.authClient(p.PublishableKey)
		_, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
			Email: "u@example.com", Password: "short-pw"}))
		wantReason(t, err, connect.CodeInvalidArgument, authrpc.ReasonWeakPassword)
		if _, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
			Email: "u@example.com", Password: "long-enough-pw"})); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		p, _ := e.createProject(t, "Dupes")
		auth := e.authClient(p.PublishableKey)
		if _, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
			Email: "u@example.com", Password: "password-1"})); err != nil {
			t.Fatal(err)
		}
		_, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
			Email: "u@example.com", Password: "password-2"}))
		wantReason(t, err, connect.CodeAlreadyExists, authrpc.ReasonEmailAlreadyExists)
	})

	t.Run("verification required", func(t *testing.T) {
		p, _ := e.createProject(t, "Verified Only")
		e.updateSettings(t, p, func(s *adminv1.ProjectSettings) { s.RequireEmailVerification = true })
		auth := e.authClient(p.PublishableKey)

		su, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
			Email: "u@example.com", Password: "password-1"}))
		if err != nil {
			t.Fatal(err)
		}
		if su.Msg.Tokens != nil {
			t.Fatal("no tokens before verification")
		}
		_, err = auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
			Email: "u@example.com", Password: "password-1"}))
		wantReason(t, err, connect.CodeFailedPrecondition, authrpc.ReasonEmailNotVerified)

		e.getPage(t, e.lastLinkTo(t, "u@example.com"))
		if _, err := auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
			Email: "u@example.com", Password: "password-1"})); err != nil {
			t.Fatalf("sign in after verification: %v", err)
		}
	})

	t.Run("enumeration-safe signup", func(t *testing.T) {
		p, _ := e.createProject(t, "Enum Safe")
		e.updateSettings(t, p, func(s *adminv1.ProjectSettings) { s.EnumerationSafeSignup = true })
		auth := e.authClient(p.PublishableKey)

		fresh, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
			Email: "u@example.com", Password: "password-1"}))
		if err != nil {
			t.Fatal(err)
		}
		again, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
			Email: "u@example.com", Password: "other-password"}))
		if err != nil {
			t.Fatal(err)
		}
		// Both responses are identical: nothing to enumerate.
		if fresh.Msg.User != nil || fresh.Msg.Tokens != nil || again.Msg.User != nil || again.Msg.Tokens != nil {
			t.Fatalf("enumeration-safe responses must be empty: %+v / %+v", fresh.Msg, again.Msg)
		}
		// The owner is told someone tried to sign up.
		note := e.mails.lastTo(t, "u@example.com")
		if !strings.Contains(note.Subject, "already have") {
			t.Fatalf("expected account-exists note, got %q", note.Subject)
		}
	})

	t.Run("uniform invalid credentials", func(t *testing.T) {
		p, _ := e.createProject(t, "Uniform")
		auth := e.authClient(p.PublishableKey)
		if _, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
			Email: "known@example.com", Password: "password-1"})); err != nil {
			t.Fatal(err)
		}
		unknownErr := func() error {
			_, err := auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
				Email: "unknown@example.com", Password: "password-1"}))
			return err
		}()
		wrongPwErr := func() error {
			_, err := auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
				Email: "known@example.com", Password: "wrong"}))
			return err
		}()
		wantReason(t, unknownErr, connect.CodeUnauthenticated, authrpc.ReasonInvalidCredentials)
		wantReason(t, wrongPwErr, connect.CodeUnauthenticated, authrpc.ReasonInvalidCredentials)
		var e1, e2 *connect.Error
		if !asConnectErr(unknownErr, &e1) || !asConnectErr(wrongPwErr, &e2) || e1.Message() != e2.Message() {
			t.Fatalf("error messages differ: %q vs %q", unknownErr, wrongPwErr)
		}
	})

	t.Run("password reset is enumeration safe", func(t *testing.T) {
		p, _ := e.createProject(t, "Reset Safe")
		auth := e.authClient(p.PublishableKey)
		before := e.mails.count()
		if _, err := auth.RequestPasswordReset(ctx, connect.NewRequest(&authv1.RequestPasswordResetRequest{
			Email: "ghost@example.com"})); err != nil {
			t.Fatalf("reset for unknown account must return OK: %v", err)
		}
		if e.mails.count() != before {
			t.Fatal("no email must go out for an unknown account")
		}
	})

	t.Run("unknown publishable key", func(t *testing.T) {
		_, err := e.authClient("pk_bogus").SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
			Email: "u@example.com", Password: "password-1"}))
		if connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Fatalf("bogus pk: %v", err)
		}
	})
}

func asConnectErr(err error, target **connect.Error) bool {
	if cerr, ok := err.(*connect.Error); ok {
		*target = cerr
		return true
	}
	return false
}

func TestSignInRateLimited(t *testing.T) {
	e := newTestEnv(t, "tok", func(o *Options) {
		o.RateLimit = &ratelimit.Config{
			IP:      ratelimit.Tier{Limit: 3, Window: time.Minute},
			Account: ratelimit.Tier{Limit: 100, Window: time.Minute},
		}
	})
	e.setup(t, "tok")
	p, _ := e.createProject(t, "App")
	auth := e.authClient(p.PublishableKey)
	ctx := context.Background()

	var last error
	for i := 0; i < 5; i++ {
		_, last = auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
			Email: "u@example.com", Password: "wrong"}))
		if connect.CodeOf(last) == connect.CodeResourceExhausted {
			break
		}
	}
	wantReason(t, last, connect.CodeResourceExhausted, authrpc.ReasonRateLimited)
	// The RESOURCE_EXHAUSTED error must carry a google.rpc.RetryInfo detail so
	// clients can back off for the advertised delay.
	var cerr *connect.Error
	if !asConnectErr(last, &cerr) {
		t.Fatalf("expected a connect error, got %v", last)
	}
	foundRetry := false
	for _, d := range cerr.Details() {
		if msg, err := d.Value(); err == nil {
			if ri, ok := msg.(*errdetails.RetryInfo); ok {
				foundRetry = true
				if ri.GetRetryDelay().AsDuration() <= 0 {
					t.Fatal("RetryInfo must advertise a positive delay")
				}
			}
		}
	}
	if !foundRetry {
		t.Fatal("rate-limit error missing RetryInfo detail")
	}
}

// TestSignupEmailDomainLists: an allowlist admits only its domains; a
// blocklist rejects its domains. The lists are enforced on password SignUp.
func TestSignupEmailDomainLists(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Gated App")
	ctx := context.Background()

	// Restrict signup to example.com, and block one subdomain of it.
	proj, err := e.store.GetProject(ctx, p.Id)
	if err != nil {
		t.Fatal(err)
	}
	proj.Settings.SignupEmailAllowlist = []string{"example.com"}
	proj.Settings.SignupEmailBlocklist = []string{"blocked.example.com"}
	if err := e.store.UpdateProject(ctx, proj); err != nil {
		t.Fatal(err)
	}
	auth := e.authClient(p.PublishableKey)

	// Allowed domain succeeds.
	if _, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "jane@example.com", Password: "password-1"})); err != nil {
		t.Fatalf("allowed domain rejected: %v", err)
	}
	// Domain outside the allowlist is refused.
	_, err = auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "bob@other.com", Password: "password-1"}))
	wantReason(t, err, connect.CodePermissionDenied, authrpc.ReasonEmailDomainNotAllowed)
	// A blocklisted subdomain of the allowed domain is also refused.
	_, err = auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "eve@blocked.example.com", Password: "password-1"}))
	wantReason(t, err, connect.CodePermissionDenied, authrpc.ReasonEmailDomainNotAllowed)
}
