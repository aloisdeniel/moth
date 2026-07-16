package server

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/gen/moth/auth/v1/authv1connect"
	"github.com/aloisdeniel/moth/internal/ratelimit"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// ---- provider doubles -------------------------------------------------

// oauthRSAKey signs the fake provider ID tokens (2048-bit keygen is slow;
// share one key across tests).
var (
	oauthRSAKeyOnce sync.Once
	oauthRSAKey     *rsa.PrivateKey
)

const oauthTestKid = "test-kid"

func testProviderKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	oauthRSAKeyOnce.Do(func() {
		var err error
		oauthRSAKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("generate rsa key: %v", err)
		}
	})
	return oauthRSAKey
}

var rawB64 = base64.RawURLEncoding

// signRS256 mints a provider-style ID token.
func signRS256(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	head, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": kid, "typ": "JWT"})
	body, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	input := rawB64.EncodeToString(head) + "." + rawB64.EncodeToString(body)
	digest := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + rawB64.EncodeToString(sig)
}

// providerDoubles fakes Google's and Apple's JWKS and token endpoints.
type providerDoubles struct {
	key         *rsa.PrivateKey
	googleJWKS  *httptest.Server
	googleToken *httptest.Server
	apple       *httptest.Server

	mu           sync.Mutex
	idToken      string // returned by the next code exchange
	refreshToken string
	exchanges    []url.Values // recorded token-endpoint forms
	revoked      []url.Values // recorded revocation forms
}

func newProviderDoubles(t *testing.T) *providerDoubles {
	t.Helper()
	pd := &providerDoubles{key: testProviderKey(t)}
	jwksHandler := func(w http.ResponseWriter, _ *http.Request) {
		pub := &pd.key.PublicKey
		writeJSON(w, http.StatusOK, map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "kid": oauthTestKid, "use": "sig", "alg": "RS256",
			"n": rawB64.EncodeToString(pub.N.Bytes()),
			"e": rawB64.EncodeToString([]byte{1, 0, 1}),
		}}})
	}
	tokenHandler := func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		pd.mu.Lock()
		pd.exchanges = append(pd.exchanges, r.PostForm)
		resp := map[string]any{
			"access_token": "provider-access", "token_type": "Bearer", "expires_in": 3600,
			"id_token": pd.idToken, "refresh_token": pd.refreshToken,
		}
		pd.mu.Unlock()
		writeJSON(w, http.StatusOK, resp)
	}
	pd.googleJWKS = httptest.NewServer(http.HandlerFunc(jwksHandler))
	t.Cleanup(pd.googleJWKS.Close)
	pd.googleToken = httptest.NewServer(http.HandlerFunc(tokenHandler))
	t.Cleanup(pd.googleToken.Close)

	appleMux := http.NewServeMux()
	appleMux.HandleFunc("GET /auth/keys", jwksHandler)
	appleMux.HandleFunc("POST /auth/token", tokenHandler)
	appleMux.HandleFunc("POST /auth/token/revoke", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		pd.mu.Lock()
		pd.revoked = append(pd.revoked, r.PostForm)
		pd.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	pd.apple = httptest.NewServer(appleMux)
	t.Cleanup(pd.apple.Close)
	return pd
}

func (pd *providerDoubles) setExchange(idToken, refreshToken string) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	pd.idToken = idToken
	pd.refreshToken = refreshToken
}

func (pd *providerDoubles) lastExchange(t *testing.T) url.Values {
	t.Helper()
	pd.mu.Lock()
	defer pd.mu.Unlock()
	if len(pd.exchanges) == 0 {
		t.Fatal("no code exchange happened")
	}
	return pd.exchanges[len(pd.exchanges)-1]
}

func (pd *providerDoubles) exchangeForms() []url.Values {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	return append([]url.Values(nil), pd.exchanges...)
}

func (pd *providerDoubles) revokedTokens() []string {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	var out []string
	for _, f := range pd.revoked {
		out = append(out, f.Get("token"))
	}
	return out
}

// idToken builds provider claims with sane defaults, overridable per test.
func (pd *providerDoubles) idTokenFor(t *testing.T, provider, sub, email, aud, nonce string, extra map[string]any) string {
	t.Helper()
	iss := "https://accounts.google.com"
	if provider == "apple" {
		iss = "https://appleid.apple.com"
	}
	claims := map[string]any{
		"iss": iss, "sub": sub, "aud": aud,
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
		"email": email, "email_verified": true,
	}
	if nonce != "" {
		claims["nonce"] = nonce
	}
	for k, v := range extra {
		claims[k] = v
	}
	return signRS256(t, pd.key, oauthTestKid, claims)
}

// hashedNonce is Apple's nonce convention: the token carries
// hex(SHA-256(raw)).
func hashedNonce(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// testP8 generates a valid Apple-style .p8 (PEM PKCS#8 EC P-256) key.
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

// ---- env helpers ------------------------------------------------------

// fakeClock is a mutable clock injected as Options.Now.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock { return &fakeClock{t: time.Now()} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// withProviderDoubles points the server's OAuth endpoints at the doubles
// and lifts the rate limits out of the way.
func withProviderDoubles(pd *providerDoubles) func(*Options) {
	return func(o *Options) {
		o.AuthEndpoints = authrpc.ProviderEndpoints{
			GoogleJWKSURL:  pd.googleJWKS.URL,
			GoogleTokenURL: pd.googleToken.URL,
			GoogleAuthURL:  "https://google-consent.test/auth",
			AppleBaseURL:   pd.apple.URL,
		}
		// Disabled tiers: OAuth flow tests fire many requests and must not be
		// throttled (the throttle itself is covered by TestOAuthHTTPRateLimited).
		o.RateLimit = &ratelimit.Config{}
	}
}

func (e *testEnv) configClient(publishableKey string) authv1connect.ConfigServiceClient {
	hc := &http.Client{Transport: keyTransport{publishableKey}}
	return authv1connect.NewConfigServiceClient(hc, e.url)
}

const (
	googleIOSClient = "ios-client.apps.googleusercontent.com"
	googleWebClient = "web-client.apps.googleusercontent.com"
	appleBundleID   = "com.example.app"
	appleServicesID = "com.example.app.signin"
)

// enableGoogle switches Google sign-in on with the test client IDs.
func (e *testEnv) enableGoogle(t *testing.T, p *adminv1.Project) {
	t.Helper()
	e.updateSettings(t, p, func(s *adminv1.ProjectSettings) {
		s.Google.Enabled = true
		s.Google.IosClientId = googleIOSClient
		s.Google.WebClientId = googleWebClient
	})
}

// enableApple switches Apple sign-in on, with full key material when p8 is
// non-empty.
func (e *testEnv) enableApple(t *testing.T, p *adminv1.Project, p8 string) {
	t.Helper()
	e.updateSettings(t, p, func(s *adminv1.ProjectSettings) {
		s.Apple.Enabled = true
		s.Apple.BundleIds = []string{appleBundleID}
		s.Apple.ServicesId = appleServicesID
		if p8 != "" {
			s.Apple.TeamId = "TEAM123456"
			s.Apple.KeyId = "KEY1234567"
			s.Apple.PrivateKeyP8 = p8
		}
	})
}

func oauthSignIn(t *testing.T, auth authv1connect.AuthServiceClient, provider authv1.OAuthProvider, idToken, nonce string) (*authv1.SignInWithOAuthResponse, error) {
	t.Helper()
	resp, err := auth.SignInWithOAuth(context.Background(), connect.NewRequest(&authv1.SignInWithOAuthRequest{
		Provider: provider, IdToken: idToken, Nonce: nonce,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// ---- tests ------------------------------------------------------------

// TestSignInWithOAuthMatrix drives the identity-resolution matrix for both
// providers: (a) repeat sign-in reuses the identity, (b) a verified email
// links to the existing password user, (c) unknown identities create a
// user.
func TestSignInWithOAuthMatrix(t *testing.T) {
	pd := newProviderDoubles(t)
	e := newTestEnv(t, "tok", withProviderDoubles(pd))
	e.setup(t, "tok")
	ctx := context.Background()

	cases := []struct {
		name     string
		provider authv1.OAuthProvider
		store    string
		aud      string
		nonce    func(raw string) string // token nonce claim from the raw nonce
	}{
		{"google", authv1.OAuthProvider_OAUTH_PROVIDER_GOOGLE, store.IdentityProviderGoogle,
			googleIOSClient, func(raw string) string { return raw }},
		{"apple", authv1.OAuthProvider_OAUTH_PROVIDER_APPLE, store.IdentityProviderApple,
			appleBundleID, hashedNonce},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, _ := e.createProject(t, "Matrix "+tc.name)
			e.enableGoogle(t, p)
			e.enableApple(t, p, "")
			auth := e.authClient(p.PublishableKey)

			// (c) unknown identity → user created, claims from the token.
			tok := pd.idTokenFor(t, tc.name, "sub-new", "sue@example.com", tc.aud,
				tc.nonce("nonce-1"), map[string]any{"name": "Sue Storm"})
			created, err := oauthSignIn(t, auth, tc.provider, tok, "nonce-1")
			if err != nil {
				t.Fatal(err)
			}
			if created.User.Email != "sue@example.com" || !created.User.EmailVerified ||
				created.User.DisplayName != "Sue Storm" {
				t.Fatalf("created user: %+v", created.User)
			}
			if created.Tokens.GetAccessToken() == "" || created.Tokens.GetRefreshToken() == "" {
				t.Fatal("social sign-in must issue a full token pair")
			}
			me, err := auth.GetMe(ctx, bearer(&authv1.GetMeRequest{}, created.Tokens.AccessToken))
			if err != nil || me.Msg.User.Id != created.User.Id {
				t.Fatalf("GetMe with social tokens: %v", err)
			}

			// (a) same (provider, subject) → same user, even with a changed
			// email.
			tok2 := pd.idTokenFor(t, tc.name, "sub-new", "renamed@example.com", tc.aud,
				tc.nonce("nonce-2"), nil)
			again, err := oauthSignIn(t, auth, tc.provider, tok2, "nonce-2")
			if err != nil {
				t.Fatal(err)
			}
			if again.User.Id != created.User.Id {
				t.Fatalf("repeat sign-in created a new user: %s vs %s", again.User.Id, created.User.Id)
			}
			// The identity keeps the provider-asserted email current.
			identity, err := e.store.GetIdentity(ctx, p.Id, tc.store, "sub-new")
			if err != nil {
				t.Fatal(err)
			}
			if identity.ProviderEmail != "renamed@example.com" {
				t.Fatalf("provider email not synced on repeat sign-in: %q", identity.ProviderEmail)
			}

			// (b) verified password account with the same verified email →
			// linked, one user with two identities.
			su, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
				Email: "bob@example.com", Password: "password-1"}))
			if err != nil {
				t.Fatal(err)
			}
			// Auto-link requires the existing account itself to be verified
			// (pre-hijacking protection), so confirm the address first.
			e.getPage(t, e.lastLinkTo(t, "bob@example.com"))
			tok3 := pd.idTokenFor(t, tc.name, "sub-bob", "bob@example.com", tc.aud,
				tc.nonce("nonce-3"), nil)
			linked, err := oauthSignIn(t, auth, tc.provider, tok3, "nonce-3")
			if err != nil {
				t.Fatal(err)
			}
			if linked.User.Id != su.Msg.User.Id {
				t.Fatalf("link created a separate user: %s vs %s", linked.User.Id, su.Msg.User.Id)
			}
			got, err := e.adminUsers().GetUser(ctx, connect.NewRequest(&adminv1.GetUserRequest{
				ProjectId: p.Id, UserId: su.Msg.User.Id}))
			if err != nil {
				t.Fatal(err)
			}
			providers := map[string]bool{}
			for _, id := range got.Msg.Identities {
				providers[id.Provider] = true
			}
			if len(got.Msg.Identities) != 2 || !providers["password"] || !providers[tc.store] {
				t.Fatalf("identities after link: %+v", got.Msg.Identities)
			}
			if !got.Msg.User.EmailVerified {
				t.Fatal("linked account should stay verified")
			}
		})
	}
}

func TestSignInWithOAuthRejections(t *testing.T) {
	pd := newProviderDoubles(t)
	e := newTestEnv(t, "tok", withProviderDoubles(pd))
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Rejections")
	e.enableGoogle(t, p)
	auth := e.authClient(p.PublishableKey)
	google := authv1.OAuthProvider_OAUTH_PROVIDER_GOOGLE

	valid := func(nonceClaim string) string {
		return pd.idTokenFor(t, "google", "sub-1", "u@example.com", googleIOSClient, nonceClaim, nil)
	}

	t.Run("wrong audience", func(t *testing.T) {
		tok := pd.idTokenFor(t, "google", "sub-1", "u@example.com", "someone-else", "n", nil)
		_, err := oauthSignIn(t, auth, google, tok, "n")
		wantReason(t, err, connect.CodeUnauthenticated, authrpc.ReasonInvalidProviderToken)
	})
	t.Run("expired", func(t *testing.T) {
		tok := pd.idTokenFor(t, "google", "sub-1", "u@example.com", googleIOSClient, "n",
			map[string]any{"exp": time.Now().Add(-time.Hour).Unix()})
		_, err := oauthSignIn(t, auth, google, tok, "n")
		wantReason(t, err, connect.CodeUnauthenticated, authrpc.ReasonInvalidProviderToken)
	})
	t.Run("tampered signature", func(t *testing.T) {
		parts := strings.Split(valid("n"), ".")
		// Re-encode the payload with an extra claim, keeping the old
		// signature.
		body, _ := rawB64.DecodeString(parts[1])
		var claims map[string]any
		json.Unmarshal(body, &claims)
		claims["email"] = "attacker@example.com"
		forged, _ := json.Marshal(claims)
		tok := parts[0] + "." + rawB64.EncodeToString(forged) + "." + parts[2]
		_, err := oauthSignIn(t, auth, google, tok, "n")
		wantReason(t, err, connect.CodeUnauthenticated, authrpc.ReasonInvalidProviderToken)
	})
	t.Run("wrong nonce", func(t *testing.T) {
		_, err := oauthSignIn(t, auth, google, valid("other-nonce"), "n")
		wantReason(t, err, connect.CodeUnauthenticated, authrpc.ReasonInvalidProviderToken)
	})
	t.Run("token without nonce claim", func(t *testing.T) {
		_, err := oauthSignIn(t, auth, google, valid(""), "n")
		wantReason(t, err, connect.CodeUnauthenticated, authrpc.ReasonInvalidProviderToken)
	})
	t.Run("missing request nonce", func(t *testing.T) {
		_, err := oauthSignIn(t, auth, google, valid("n"), "")
		wantReason(t, err, connect.CodeInvalidArgument, authrpc.ReasonInvalidProviderToken)
	})
	t.Run("provider disabled", func(t *testing.T) {
		_, err := oauthSignIn(t, auth, authv1.OAuthProvider_OAUTH_PROVIDER_APPLE,
			valid("n"), "n")
		wantReason(t, err, connect.CodeFailedPrecondition, authrpc.ReasonProviderDisabled)
	})
	t.Run("unverified email never links", func(t *testing.T) {
		if _, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
			Email: "victim@example.com", Password: "password-1"})); err != nil {
			t.Fatal(err)
		}
		tok := pd.idTokenFor(t, "google", "sub-unverified", "victim@example.com", googleIOSClient, "n",
			map[string]any{"email_verified": false})
		_, err := oauthSignIn(t, auth, google, tok, "n")
		wantReason(t, err, connect.CodeAlreadyExists, authrpc.ReasonEmailAlreadyExists)
	})
	t.Run("existing unverified account never links (pre-hijacking)", func(t *testing.T) {
		// victim@example.com was pre-registered above with a password and
		// never verified; a provider-verified token for the same address
		// must not attach to (and verify) that account, or whoever
		// registered it first would capture the victim's provider identity.
		tok := pd.idTokenFor(t, "google", "sub-prehijack", "victim@example.com", googleIOSClient, "n", nil)
		_, err := oauthSignIn(t, auth, google, tok, "n")
		wantReason(t, err, connect.CodeAlreadyExists, authrpc.ReasonEmailAlreadyExists)
		if _, err := e.store.GetIdentity(ctx, p.Id, store.IdentityProviderGoogle, "sub-prehijack"); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("identity after refused link: %v, want ErrNotFound", err)
		}
	})
	t.Run("replayed id token", func(t *testing.T) {
		tok := pd.idTokenFor(t, "google", "sub-replay", "replay@example.com", googleIOSClient, "nr", nil)
		if _, err := oauthSignIn(t, auth, google, tok, "nr"); err != nil {
			t.Fatal(err)
		}
		// The same token (with its correct nonce) must not mint a second
		// session: the payload of a captured token is readable, so the
		// nonce alone cannot stop a holder from replaying it.
		_, err := oauthSignIn(t, auth, google, tok, "nr")
		wantReason(t, err, connect.CodeUnauthenticated, authrpc.ReasonInvalidProviderToken)
	})
	t.Run("auto-link disabled", func(t *testing.T) {
		e.updateSettings(t, p, func(s *adminv1.ProjectSettings) {
			off := false
			s.AutoLinkVerifiedEmail = &off
		})
		// A verified existing account, so only the auto-link toggle can
		// refuse the link here.
		if _, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
			Email: "nolink@example.com", Password: "password-1"})); err != nil {
			t.Fatal(err)
		}
		e.getPage(t, e.lastLinkTo(t, "nolink@example.com"))
		tok := pd.idTokenFor(t, "google", "sub-nolink", "nolink@example.com", googleIOSClient, "n", nil)
		_, err := oauthSignIn(t, auth, google, tok, "n")
		wantReason(t, err, connect.CodeAlreadyExists, authrpc.ReasonEmailAlreadyExists)
	})
	t.Run("signup closed blocks creation", func(t *testing.T) {
		e.updateSettings(t, p, func(s *adminv1.ProjectSettings) { s.AllowPublicSignup = false })
		tok := pd.idTokenFor(t, "google", "sub-closed", "new@example.com", googleIOSClient, "n", nil)
		_, err := oauthSignIn(t, auth, google, tok, "n")
		wantReason(t, err, connect.CodePermissionDenied, authrpc.ReasonSignupClosed)
	})
}

// TestAppleRefreshTokenLifecycle exercises the native Apple extras: the
// authorization code is exchanged and the refresh token stored encrypted;
// deleting the account (social-only, fresh token) revokes it at Apple.
func TestAppleRefreshTokenLifecycle(t *testing.T) {
	pd := newProviderDoubles(t)
	e := newTestEnv(t, "tok", withProviderDoubles(pd))
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Apple App")
	e.enableApple(t, p, testP8(t))
	auth := e.authClient(p.PublishableKey)

	pd.setExchange("", "apple-refresh-1")
	tok := pd.idTokenFor(t, "apple", "sub-apple", "ana@example.com", appleBundleID,
		hashedNonce("nonce-1"), nil)
	si, err := auth.SignInWithOAuth(ctx, connect.NewRequest(&authv1.SignInWithOAuthRequest{
		Provider:          authv1.OAuthProvider_OAUTH_PROVIDER_APPLE,
		IdToken:           tok,
		Nonce:             "nonce-1",
		AuthorizationCode: "auth-code-1",
		GivenName:         "Ana",
		FamilyName:        "Apple",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if si.Msg.User.DisplayName != "Ana Apple" {
		t.Fatalf("first-launch name: %q", si.Msg.User.DisplayName)
	}

	// The exchange authenticated as the token's bundle-ID audience and
	// carried the code.
	form := pd.lastExchange(t)
	if form.Get("client_id") != appleBundleID || form.Get("code") != "auth-code-1" {
		t.Fatalf("apple exchange form: %v", form)
	}

	// A second sign-in reuses the cached client secret (ES256 signatures
	// are randomized, so a re-minted secret would differ).
	pd.setExchange("", "apple-refresh-2")
	tok2 := pd.idTokenFor(t, "apple", "sub-apple", "ana@example.com", appleBundleID,
		hashedNonce("nonce-2"), nil)
	si2, err := auth.SignInWithOAuth(ctx, connect.NewRequest(&authv1.SignInWithOAuthRequest{
		Provider:          authv1.OAuthProvider_OAUTH_PROVIDER_APPLE,
		IdToken:           tok2,
		Nonce:             "nonce-2",
		AuthorizationCode: "auth-code-2",
	}))
	if err != nil {
		t.Fatal(err)
	}
	si = si2
	forms := pd.exchangeForms()
	if len(forms) != 2 {
		t.Fatalf("%d exchanges, want 2", len(forms))
	}
	if s1, s2 := forms[0].Get("client_secret"), forms[1].Get("client_secret"); s1 == "" || s1 != s2 {
		t.Fatalf("apple client secret not cached across exchanges: %q vs %q", s1, s2)
	}
	// The refresh token is stored encrypted on the identity.
	identity, err := e.store.GetIdentity(ctx, p.Id, store.IdentityProviderApple, "sub-apple")
	if err != nil {
		t.Fatal(err)
	}
	if len(identity.AppleRefreshTokenEnc) == 0 {
		t.Fatal("apple refresh token was not stored")
	}
	if strings.Contains(string(identity.AppleRefreshTokenEnc), "apple-refresh-1") {
		t.Fatal("apple refresh token stored in plaintext")
	}

	// Social-only + fresh sign-in → deletion allowed, revocation issued
	// for the currently stored refresh token.
	if _, err := auth.DeleteAccount(ctx, bearer(&authv1.DeleteAccountRequest{}, si.Msg.Tokens.AccessToken)); err != nil {
		t.Fatal(err)
	}
	revoked := pd.revokedTokens()
	if len(revoked) != 1 || revoked[0] != "apple-refresh-2" {
		t.Fatalf("revoked tokens: %v", revoked)
	}
	if _, err := e.store.GetUser(ctx, p.Id, si.Msg.User.Id); err == nil {
		t.Fatal("user should be deleted")
	}
}

// TestDeleteAccountSocialOnlyFreshness enforces the 5-minute iat window for
// social-only accounts (password accounts keep the password check).
func TestDeleteAccountSocialOnlyFreshness(t *testing.T) {
	pd := newProviderDoubles(t)
	clock := newFakeClock()
	e := newTestEnv(t, "tok", withProviderDoubles(pd), func(o *Options) { o.Now = clock.Now })
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Fresh App")
	e.enableGoogle(t, p)
	auth := e.authClient(p.PublishableKey)

	tok := pd.idTokenFor(t, "google", "sub-1", "u@example.com", googleIOSClient, "n", nil)
	si, err := oauthSignIn(t, auth, authv1.OAuthProvider_OAUTH_PROVIDER_GOOGLE, tok, "n")
	if err != nil {
		t.Fatal(err)
	}

	// Six minutes later the access token is still valid but too stale to
	// authorize deletion.
	clock.Advance(6 * time.Minute)
	_, err = auth.DeleteAccount(ctx, bearer(&authv1.DeleteAccountRequest{}, si.Tokens.AccessToken))
	wantReason(t, err, connect.CodeFailedPrecondition, authrpc.ReasonInvalidCredentials)

	// A refreshed access token is not a provider re-authentication:
	// deletion stays blocked even though the new token is seconds old.
	rt, err := auth.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{
		RefreshToken: si.Tokens.RefreshToken}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = auth.DeleteAccount(ctx, bearer(&authv1.DeleteAccountRequest{}, rt.Msg.Tokens.AccessToken))
	wantReason(t, err, connect.CodeFailedPrecondition, authrpc.ReasonInvalidCredentials)

	// A fresh provider sign-in unlocks it.
	tok2 := pd.idTokenFor(t, "google", "sub-1", "u@example.com", googleIOSClient, "n2", nil)
	si2, err := oauthSignIn(t, auth, authv1.OAuthProvider_OAUTH_PROVIDER_GOOGLE, tok2, "n2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.DeleteAccount(ctx, bearer(&authv1.DeleteAccountRequest{}, si2.Tokens.AccessToken)); err != nil {
		t.Fatal(err)
	}
}

func TestUnlinkIdentity(t *testing.T) {
	pd := newProviderDoubles(t)
	e := newTestEnv(t, "tok", withProviderDoubles(pd))
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Unlink App")
	e.enableGoogle(t, p)
	e.enableApple(t, p, testP8(t))
	auth := e.authClient(p.PublishableKey)
	google := authv1.OAuthProvider_OAUTH_PROVIDER_GOOGLE
	apple := authv1.OAuthProvider_OAUTH_PROVIDER_APPLE

	// Social-only user with a single identity cannot unlink it.
	tok := pd.idTokenFor(t, "google", "sub-solo", "solo@example.com", googleIOSClient, "n", nil)
	solo, err := oauthSignIn(t, auth, google, tok, "n")
	if err != nil {
		t.Fatal(err)
	}
	_, err = auth.UnlinkIdentity(ctx, bearer(&authv1.UnlinkIdentityRequest{Provider: google}, solo.Tokens.AccessToken))
	wantReason(t, err, connect.CodeFailedPrecondition, authrpc.ReasonLastLoginMethod)

	// A second (Apple) identity makes the Google one removable — and
	// unlinking the Apple one revokes its stored refresh token.
	pd.setExchange("", "unlink-refresh")
	atok := pd.idTokenFor(t, "apple", "sub-solo-apple", "solo@example.com", appleBundleID,
		hashedNonce("n2"), nil)
	linked, err := auth.SignInWithOAuth(ctx, connect.NewRequest(&authv1.SignInWithOAuthRequest{
		Provider: apple, IdToken: atok, Nonce: "n2", AuthorizationCode: "code-x"}))
	if err != nil {
		t.Fatal(err)
	}
	if linked.Msg.User.Id != solo.User.Id {
		t.Fatal("apple identity should have linked to the same user")
	}
	access := linked.Msg.Tokens.AccessToken
	if _, err := auth.UnlinkIdentity(ctx, bearer(&authv1.UnlinkIdentityRequest{Provider: apple}, access)); err != nil {
		t.Fatal(err)
	}
	if revoked := pd.revokedTokens(); len(revoked) != 1 || revoked[0] != "unlink-refresh" {
		t.Fatalf("revoked on unlink: %v", revoked)
	}
	// The Apple identity is gone; unlinking it again is NotFound, and the
	// remaining Google identity is now the last method again.
	_, err = auth.UnlinkIdentity(ctx, bearer(&authv1.UnlinkIdentityRequest{Provider: apple}, access))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("second unlink: %v", err)
	}
	_, err = auth.UnlinkIdentity(ctx, bearer(&authv1.UnlinkIdentityRequest{Provider: google}, access))
	wantReason(t, err, connect.CodeFailedPrecondition, authrpc.ReasonLastLoginMethod)

	// A password user can drop its only social identity.
	su, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "pw@example.com", Password: "password-1"}))
	if err != nil {
		t.Fatal(err)
	}
	// Auto-link requires the existing account to be verified.
	e.getPage(t, e.lastLinkTo(t, "pw@example.com"))
	gtok := pd.idTokenFor(t, "google", "sub-pw", "pw@example.com", googleIOSClient, "n3", nil)
	if _, err := oauthSignIn(t, auth, google, gtok, "n3"); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.UnlinkIdentity(ctx, bearer(&authv1.UnlinkIdentityRequest{Provider: google}, su.Msg.Tokens.AccessToken)); err != nil {
		t.Fatalf("password user unlinking google: %v", err)
	}
}

// noRedirectClient never follows redirects, so tests can inspect Location.
func noRedirectClient() *http.Client {
	return &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
}

func TestOAuthRedirectFlowGoogle(t *testing.T) {
	pd := newProviderDoubles(t)
	e := newTestEnv(t, "tok", withProviderDoubles(pd))
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Web App")
	e.updateSettings(t, p, func(s *adminv1.ProjectSettings) {
		s.Google.Enabled = true
		s.Google.WebClientId = googleWebClient
		s.Google.WebClientSecret = "web-secret-1"
		s.RedirectSchemes = []string{"myapp"}
	})
	auth := e.authClient(p.PublishableKey)
	hc := noRedirectClient()
	slug := p.Slug

	start := func(redirect string) *http.Response {
		t.Helper()
		resp, err := hc.Get(e.url + "/oauth/google/start?project=" + slug +
			"&redirect=" + url.QueryEscape(redirect))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp
	}

	// Unregistered scheme → open-redirect protection.
	if resp := start("https://evil.example/phish"); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unregistered redirect: want 400, got %d", resp.StatusCode)
	}

	resp := start("myapp://auth")
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("start: want 302, got %d", resp.StatusCode)
	}
	consent, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	q := consent.Query()
	if consent.Host != "google-consent.test" || q.Get("client_id") != googleWebClient ||
		q.Get("response_type") != "code" || !strings.Contains(q.Get("scope"), "openid") {
		t.Fatalf("consent url: %s", consent)
	}
	state, nonce := q.Get("state"), q.Get("nonce")
	if state == "" || nonce == "" {
		t.Fatalf("consent url misses state/nonce: %s", consent)
	}
	if !strings.HasPrefix(state, slug+".") {
		t.Fatalf("state should embed the project slug: %q", state)
	}

	// Tampered state → 4xx, and the real state stays usable.
	tampered, err := hc.Get(e.url + "/oauth/google/callback?code=abc&state=" +
		url.QueryEscape(state+"x"))
	if err != nil {
		t.Fatal(err)
	}
	tampered.Body.Close()
	if tampered.StatusCode != http.StatusBadRequest {
		t.Fatalf("tampered state: want 400, got %d", tampered.StatusCode)
	}

	// Provider consent happened; the callback exchanges the code and
	// redirects into the app with a one-time code.
	pd.setExchange(pd.idTokenFor(t, "google", "sub-web", "web@example.com", googleWebClient, nonce, nil), "")
	cb, err := hc.Get(e.url + "/oauth/google/callback?code=provider-code&state=" + url.QueryEscape(state))
	if err != nil {
		t.Fatal(err)
	}
	cb.Body.Close()
	if cb.StatusCode != http.StatusFound {
		t.Fatalf("callback: want 302, got %d", cb.StatusCode)
	}
	loc, err := url.Parse(cb.Header.Get("Location"))
	if err != nil || loc.Scheme != "myapp" {
		t.Fatalf("callback location: %q (%v)", cb.Header.Get("Location"), err)
	}
	appCode := loc.Query().Get("code")
	if appCode == "" {
		t.Fatalf("callback location misses code: %s", loc)
	}
	// The server-side exchange used the stored web client secret.
	form := pd.lastExchange(t)
	if form.Get("client_id") != googleWebClient || form.Get("client_secret") != "web-secret-1" ||
		form.Get("code") != "provider-code" {
		t.Fatalf("google exchange form: %v", form)
	}

	// State replay → 4xx (single use).
	replay, err := hc.Get(e.url + "/oauth/google/callback?code=other&state=" + url.QueryEscape(state))
	if err != nil {
		t.Fatal(err)
	}
	replay.Body.Close()
	if replay.StatusCode != http.StatusBadRequest {
		t.Fatalf("state replay: want 400, got %d", replay.StatusCode)
	}

	// A token minted without the state's nonce fails the callback: the
	// exchanged ID token must be bound to this exact attempt.
	badNonce := start("myapp://auth")
	badConsent, err := url.Parse(badNonce.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	pd.setExchange(pd.idTokenFor(t, "google", "sub-web", "web@example.com", googleWebClient,
		"other-nonce", nil), "")
	badCb, err := hc.Get(e.url + "/oauth/google/callback?code=provider-code&state=" +
		url.QueryEscape(badConsent.Query().Get("state")))
	if err != nil {
		t.Fatal(err)
	}
	badCb.Body.Close()
	if badCb.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong-nonce callback: want 401, got %d", badCb.StatusCode)
	}

	// A Google state cannot be redeemed at the Apple callback.
	crossStart := start("myapp://auth")
	crossConsent, err := url.Parse(crossStart.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	cross, err := hc.Get(e.url + "/oauth/apple/callback?code=abc&state=" +
		url.QueryEscape(crossConsent.Query().Get("state")))
	if err != nil {
		t.Fatal(err)
	}
	cross.Body.Close()
	if cross.StatusCode != http.StatusBadRequest {
		t.Fatalf("cross-provider state: want 400, got %d", cross.StatusCode)
	}

	// The app trades the one-time code for tokens — exactly once.
	ex, err := auth.ExchangeOAuthCode(ctx, connect.NewRequest(&authv1.ExchangeOAuthCodeRequest{Code: appCode}))
	if err != nil {
		t.Fatal(err)
	}
	if ex.Msg.User.Email != "web@example.com" || ex.Msg.Tokens.GetAccessToken() == "" {
		t.Fatalf("exchange: %+v", ex.Msg)
	}
	if _, err := auth.GetMe(ctx, bearer(&authv1.GetMeRequest{}, ex.Msg.Tokens.AccessToken)); err != nil {
		t.Fatal(err)
	}
	_, err = auth.ExchangeOAuthCode(ctx, connect.NewRequest(&authv1.ExchangeOAuthCodeRequest{Code: appCode}))
	wantReason(t, err, connect.CodeUnauthenticated, authrpc.ReasonInvalidOAuthCode)

	// Event semantics match the native flow: the brand-new web-flow user
	// produced one user.signup and no user.login (the code carried the
	// signup marker into the exchange).
	drainCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := e.srv.Close(drainCtx); err != nil {
		t.Fatalf("drain events: %v", err)
	}
	events, err := e.store.ListRecentEvents(ctx, p.Id, 50)
	if err != nil {
		t.Fatal(err)
	}
	var signups, logins int
	for _, ev := range events {
		switch ev.Type {
		case store.EventUserSignup:
			signups++
		case store.EventUserLogin:
			logins++
		}
	}
	if signups != 1 || logins != 0 {
		t.Fatalf("web-flow first sign-in events: %d signups, %d logins, want 1/0 (%+v)",
			signups, logins, events)
	}
}

func TestOAuthRedirectFlowApple(t *testing.T) {
	pd := newProviderDoubles(t)
	e := newTestEnv(t, "tok", withProviderDoubles(pd))
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Apple Web App")
	e.enableApple(t, p, testP8(t))
	auth := e.authClient(p.PublishableKey)
	hc := noRedirectClient()

	// No redirect parameter → the callback shows the hosted success page.
	resp, err := hc.Get(e.url + "/oauth/apple/start?project=" + p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("start: want 302, got %d", resp.StatusCode)
	}
	consent, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	q := consent.Query()
	if q.Get("response_mode") != "form_post" || q.Get("client_id") != appleServicesID {
		t.Fatalf("apple consent url: %s", consent)
	}
	state, hashed := q.Get("state"), q.Get("nonce")

	// Apple form_posts the callback, with the first-launch user JSON.
	pd.setExchange(pd.idTokenFor(t, "apple", "sub-web-apple", "jane@example.com",
		appleServicesID, hashed, nil), "apple-web-refresh")
	cb, err := hc.PostForm(e.url+"/oauth/apple/callback", url.Values{
		"state": {state},
		"code":  {"apple-code"},
		"user":  {`{"name":{"firstName":"Jane","lastName":"Doe"}}`},
	})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(cb.Body)
	cb.Body.Close()
	if cb.StatusCode != http.StatusOK {
		t.Fatalf("apple callback: %d\n%s", cb.StatusCode, body)
	}
	m := regexp.MustCompile(`code: ([a-z0-9]+)`).FindStringSubmatch(string(body))
	if m == nil {
		t.Fatalf("success page misses the code:\n%s", body)
	}

	ex, err := auth.ExchangeOAuthCode(ctx, connect.NewRequest(&authv1.ExchangeOAuthCodeRequest{Code: m[1]}))
	if err != nil {
		t.Fatal(err)
	}
	if ex.Msg.User.Email != "jane@example.com" || ex.Msg.User.DisplayName != "Jane Doe" {
		t.Fatalf("apple web user: %+v", ex.Msg.User)
	}
	// The refresh token from the web exchange is stored on the identity.
	identity, err := e.store.GetIdentity(ctx, p.Id, store.IdentityProviderApple, "sub-web-apple")
	if err != nil {
		t.Fatal(err)
	}
	if len(identity.AppleRefreshTokenEnc) == 0 {
		t.Fatal("apple web refresh token was not stored")
	}
}

func TestGetProjectConfig(t *testing.T) {
	pd := newProviderDoubles(t)
	e := newTestEnv(t, "tok", withProviderDoubles(pd))
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Config App")
	cfg := e.configClient(p.PublishableKey)

	got, err := cfg.GetProjectConfig(ctx, connect.NewRequest(&authv1.GetProjectConfigRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.Google.Enabled || got.Msg.Apple.Enabled || !got.Msg.SignUpOpen ||
		got.Msg.PasswordMinLength != 8 {
		t.Fatalf("default config: %+v", got.Msg)
	}

	// Admin toggles reflect immediately, and secrets never leak.
	e.updateSettings(t, p, func(s *adminv1.ProjectSettings) {
		s.Google.Enabled = true
		s.Google.IosClientId = googleIOSClient
		s.Google.WebClientId = googleWebClient
		s.Google.WebClientSecret = "super-secret"
		s.AllowPublicSignup = false
	})
	got, err = cfg.GetProjectConfig(ctx, connect.NewRequest(&authv1.GetProjectConfigRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Msg.Google.Enabled || got.Msg.Google.IosClientId != googleIOSClient || got.Msg.SignUpOpen {
		t.Fatalf("updated config: %+v", got.Msg)
	}
	if strings.Contains(got.Msg.String(), "super-secret") {
		t.Fatal("config response leaks a secret")
	}

	// Disabling flips it back — and sign-in attempts follow suit.
	e.updateSettings(t, p, func(s *adminv1.ProjectSettings) { s.Google.Enabled = false })
	got, err = cfg.GetProjectConfig(ctx, connect.NewRequest(&authv1.GetProjectConfigRequest{}))
	if err != nil || got.Msg.Google.Enabled {
		t.Fatalf("disabled config: %v %+v", err, got.Msg)
	}
	tok := pd.idTokenFor(t, "google", "sub", "u@example.com", googleIOSClient, "n", nil)
	_, err = oauthSignIn(t, e.authClient(p.PublishableKey),
		authv1.OAuthProvider_OAUTH_PROVIDER_GOOGLE, tok, "n")
	wantReason(t, err, connect.CodeFailedPrecondition, authrpc.ReasonProviderDisabled)

	// The config RPC needs a publishable key like every auth RPC.
	_, err = e.configClient("pk_bogus").GetProjectConfig(ctx,
		connect.NewRequest(&authv1.GetProjectConfigRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("bogus pk: %v", err)
	}
}

// TestAdminProviderConfig covers the write-only secret convention and the
// social-only guardrails of the admin API.
func TestAdminProviderConfig(t *testing.T) {
	pd := newProviderDoubles(t)
	e := newTestEnv(t, "tok", withProviderDoubles(pd))
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Admin App")

	// Secrets are write-only: reads report presence, never the value.
	e.updateSettings(t, p, func(s *adminv1.ProjectSettings) {
		s.Google.Enabled = true
		s.Google.WebClientId = googleWebClient
		s.Google.WebClientSecret = "secret-1"
		s.Apple.Enabled = true
		s.Apple.BundleIds = []string{appleBundleID}
		s.Apple.TeamId = "TEAM123456"
		s.Apple.KeyId = "KEY1234567"
		s.Apple.PrivateKeyP8 = testP8(t)
	})
	got, err := e.projects.GetProject(ctx, connect.NewRequest(&adminv1.GetProjectRequest{Id: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	s := got.Msg.Project.Settings
	if !s.Google.HasWebClientSecret || !s.Apple.HasPrivateKey {
		t.Fatalf("has flags: %+v / %+v", s.Google, s.Apple)
	}
	if s.Google.WebClientSecret != "" || s.Apple.PrivateKeyP8 != "" {
		t.Fatal("secrets must never be returned")
	}

	// An update with empty secret fields keeps the stored values.
	e.updateSettings(t, got.Msg.Project, func(s *adminv1.ProjectSettings) {
		s.Google.WebClientId = "changed-" + googleWebClient
	})
	got, err = e.projects.GetProject(ctx, connect.NewRequest(&adminv1.GetProjectRequest{Id: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	s = got.Msg.Project.Settings
	if !s.Google.HasWebClientSecret || !s.Apple.HasPrivateKey ||
		s.Google.WebClientId != "changed-"+googleWebClient {
		t.Fatalf("after keep-stored update: %+v / %+v", s.Google, s.Apple)
	}

	// A malformed .p8 is rejected at write time.
	settings := got.Msg.Project.Settings
	settings.Apple.PrivateKeyP8 = "not a key"
	_, err = e.projects.UpdateProject(ctx, connect.NewRequest(&adminv1.UpdateProjectRequest{
		Id: p.Id, Name: p.Name, Settings: settings}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("bad p8: %v", err)
	}

	// Enabling a provider without any usable audience is rejected.
	settings = got.Msg.Project.Settings
	settings.Google = &adminv1.GoogleProviderConfig{Enabled: true}
	_, err = e.projects.UpdateProject(ctx, connect.NewRequest(&adminv1.UpdateProjectRequest{
		Id: p.Id, Name: p.Name, Settings: settings}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("google without client ids: %v", err)
	}

	// http(s) redirect schemes are rejected: the redirect check matches
	// the scheme only, so registering them would be an open redirect.
	got, err = e.projects.GetProject(ctx, connect.NewRequest(&adminv1.GetProjectRequest{Id: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	for _, scheme := range []string{"http", "https"} {
		settings = got.Msg.Project.Settings
		settings.RedirectSchemes = []string{scheme}
		_, err = e.projects.UpdateProject(ctx, connect.NewRequest(&adminv1.UpdateProjectRequest{
			Id: p.Id, Name: p.Name, Settings: settings}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("%s redirect scheme: %v", scheme, err)
		}
	}

	// SendPasswordReset refuses social-only users.
	auth := e.authClient(p.PublishableKey)
	tok := pd.idTokenFor(t, "google", "sub-social", "social@example.com", "changed-"+googleWebClient, "n", nil)
	si, err := oauthSignIn(t, auth, authv1.OAuthProvider_OAUTH_PROVIDER_GOOGLE, tok, "n")
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.adminUsers().SendPasswordReset(ctx, connect.NewRequest(&adminv1.SendPasswordResetRequest{
		ProjectId: p.Id, UserId: si.User.Id}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("reset for social-only user: %v", err)
	}
}

// TestDeleteAccountPasswordUserRevokesApple covers the password re-auth
// branch of DeleteAccount: a password user with a linked Apple identity
// still gets its stored refresh token revoked at Apple.
func TestDeleteAccountPasswordUserRevokesApple(t *testing.T) {
	pd := newProviderDoubles(t)
	e := newTestEnv(t, "tok", withProviderDoubles(pd))
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Apple PW App")
	e.enableApple(t, p, testP8(t))
	auth := e.authClient(p.PublishableKey)

	su, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "pwapple@example.com", Password: "password-1"}))
	if err != nil {
		t.Fatal(err)
	}
	// Auto-link requires the existing account to be verified.
	e.getPage(t, e.lastLinkTo(t, "pwapple@example.com"))

	pd.setExchange("", "pw-apple-refresh")
	atok := pd.idTokenFor(t, "apple", "sub-pw-apple", "pwapple@example.com", appleBundleID,
		hashedNonce("n"), nil)
	linked, err := auth.SignInWithOAuth(ctx, connect.NewRequest(&authv1.SignInWithOAuthRequest{
		Provider: authv1.OAuthProvider_OAUTH_PROVIDER_APPLE, IdToken: atok, Nonce: "n",
		AuthorizationCode: "code-pw"}))
	if err != nil {
		t.Fatal(err)
	}
	if linked.Msg.User.Id != su.Msg.User.Id {
		t.Fatal("apple identity should have linked to the password user")
	}

	// The password branch authorizes the deletion; the Apple identity's
	// refresh token must still be revoked.
	if _, err := auth.DeleteAccount(ctx, bearer(&authv1.DeleteAccountRequest{
		Password: "password-1"}, su.Msg.Tokens.AccessToken)); err != nil {
		t.Fatal(err)
	}
	if revoked := pd.revokedTokens(); len(revoked) != 1 || revoked[0] != "pw-apple-refresh" {
		t.Fatalf("revoked tokens: %v", revoked)
	}
}

// TestOAuthDisabledUser locks admin-disabled users out of every OAuth
// entry point: repeat native sign-in, auto-link, and a pending one-time
// web code.
func TestOAuthDisabledUser(t *testing.T) {
	pd := newProviderDoubles(t)
	e := newTestEnv(t, "tok", withProviderDoubles(pd))
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Disabled App")
	e.updateSettings(t, p, func(s *adminv1.ProjectSettings) {
		s.Google.Enabled = true
		s.Google.IosClientId = googleIOSClient
		s.Google.WebClientId = googleWebClient
		s.Google.WebClientSecret = "web-secret-1"
		s.RedirectSchemes = []string{"myapp"}
	})
	auth := e.authClient(p.PublishableKey)
	google := authv1.OAuthProvider_OAUTH_PROVIDER_GOOGLE
	hc := noRedirectClient()

	tok := pd.idTokenFor(t, "google", "sub-dis", "dis@example.com", googleIOSClient, "n1", nil)
	si, err := oauthSignIn(t, auth, google, tok, "n1")
	if err != nil {
		t.Fatal(err)
	}

	// Run the web flow up to the one-time app code while still enabled.
	startResp, err := hc.Get(e.url + "/oauth/google/start?project=" + p.Slug +
		"&redirect=" + url.QueryEscape("myapp://auth"))
	if err != nil {
		t.Fatal(err)
	}
	startResp.Body.Close()
	consent, err := url.Parse(startResp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	state, nonce := consent.Query().Get("state"), consent.Query().Get("nonce")
	pd.setExchange(pd.idTokenFor(t, "google", "sub-dis", "dis@example.com", googleWebClient, nonce, nil), "")
	cb, err := hc.Get(e.url + "/oauth/google/callback?code=provider-code&state=" + url.QueryEscape(state))
	if err != nil {
		t.Fatal(err)
	}
	cb.Body.Close()
	loc, err := url.Parse(cb.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	appCode := loc.Query().Get("code")
	if appCode == "" {
		t.Fatalf("callback location misses code: %q", cb.Header.Get("Location"))
	}

	if _, err := e.adminUsers().DisableUser(ctx, connect.NewRequest(&adminv1.DisableUserRequest{
		ProjectId: p.Id, UserId: si.User.Id})); err != nil {
		t.Fatal(err)
	}

	// Repeat native sign-in on the existing identity is blocked.
	tok2 := pd.idTokenFor(t, "google", "sub-dis", "dis@example.com", googleIOSClient, "n2", nil)
	_, err = oauthSignIn(t, auth, google, tok2, "n2")
	wantReason(t, err, connect.CodePermissionDenied, authrpc.ReasonUserDisabled)

	// The pending one-time code no longer signs the user in either.
	_, err = auth.ExchangeOAuthCode(ctx, connect.NewRequest(&authv1.ExchangeOAuthCodeRequest{Code: appCode}))
	wantReason(t, err, connect.CodePermissionDenied, authrpc.ReasonUserDisabled)

	// Auto-linking into a disabled (verified) account is blocked too.
	su, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "dis2@example.com", Password: "password-1"}))
	if err != nil {
		t.Fatal(err)
	}
	e.getPage(t, e.lastLinkTo(t, "dis2@example.com"))
	if _, err := e.adminUsers().DisableUser(ctx, connect.NewRequest(&adminv1.DisableUserRequest{
		ProjectId: p.Id, UserId: su.Msg.User.Id})); err != nil {
		t.Fatal(err)
	}
	tok3 := pd.idTokenFor(t, "google", "sub-dis2", "dis2@example.com", googleIOSClient, "n3", nil)
	_, err = oauthSignIn(t, auth, google, tok3, "n3")
	wantReason(t, err, connect.CodePermissionDenied, authrpc.ReasonUserDisabled)
}

// TestExchangeOAuthCodeProviderDisabled: disabling a provider immediately
// invalidates codes minted while it was enabled.
func TestExchangeOAuthCodeProviderDisabled(t *testing.T) {
	pd := newProviderDoubles(t)
	e := newTestEnv(t, "tok", withProviderDoubles(pd))
	e.setup(t, "tok")
	ctx := context.Background()
	p, _ := e.createProject(t, "Toggle App")
	e.updateSettings(t, p, func(s *adminv1.ProjectSettings) {
		s.Google.Enabled = true
		s.Google.WebClientId = googleWebClient
		s.Google.WebClientSecret = "web-secret-1"
		s.RedirectSchemes = []string{"myapp"}
	})
	auth := e.authClient(p.PublishableKey)
	hc := noRedirectClient()

	startResp, err := hc.Get(e.url + "/oauth/google/start?project=" + p.Slug +
		"&redirect=" + url.QueryEscape("myapp://auth"))
	if err != nil {
		t.Fatal(err)
	}
	startResp.Body.Close()
	consent, err := url.Parse(startResp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	state, nonce := consent.Query().Get("state"), consent.Query().Get("nonce")
	pd.setExchange(pd.idTokenFor(t, "google", "sub-toggle", "toggle@example.com", googleWebClient, nonce, nil), "")
	cb, err := hc.Get(e.url + "/oauth/google/callback?code=provider-code&state=" + url.QueryEscape(state))
	if err != nil {
		t.Fatal(err)
	}
	cb.Body.Close()
	loc, err := url.Parse(cb.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	appCode := loc.Query().Get("code")
	if appCode == "" {
		t.Fatalf("callback location misses code: %q", cb.Header.Get("Location"))
	}

	e.updateSettings(t, p, func(s *adminv1.ProjectSettings) { s.Google.Enabled = false })
	_, err = auth.ExchangeOAuthCode(ctx, connect.NewRequest(&authv1.ExchangeOAuthCodeRequest{Code: appCode}))
	wantReason(t, err, connect.CodeFailedPrecondition, authrpc.ReasonProviderDisabled)
}

// TestOAuthStartRejections covers the start leg's misconfiguration
// rejections: unknown providers and providers that are off or not fully
// configured for the web flow.
func TestOAuthStartRejections(t *testing.T) {
	pd := newProviderDoubles(t)
	e := newTestEnv(t, "tok", withProviderDoubles(pd))
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Start Rejections")
	hc := noRedirectClient()

	start := func(provider string) (int, string) {
		t.Helper()
		resp, err := hc.Get(e.url + "/oauth/" + provider + "/start?project=" + p.Slug)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp.StatusCode, string(body)
	}

	// Unknown provider path value.
	if code, _ := start("github"); code != http.StatusBadRequest {
		t.Fatalf("unknown provider: want 400, got %d", code)
	}
	// Provider entirely disabled.
	if code, body := start("google"); code != http.StatusBadRequest || !strings.Contains(body, "not available") {
		t.Fatalf("disabled google: want 400 provider-disabled page, got %d\n%s", code, body)
	}
	// Enabled for native sign-in only: no web client ID.
	e.updateSettings(t, p, func(s *adminv1.ProjectSettings) {
		s.Google.Enabled = true
		s.Google.IosClientId = googleIOSClient
	})
	if code, _ := start("google"); code != http.StatusBadRequest {
		t.Fatalf("google without web client id: want 400, got %d", code)
	}
	// Web client ID without a stored web client secret.
	e.updateSettings(t, p, func(s *adminv1.ProjectSettings) {
		s.Google.WebClientId = googleWebClient
	})
	if code, _ := start("google"); code != http.StatusBadRequest {
		t.Fatalf("google without stored secret: want 400, got %d", code)
	}
	// Apple with a Services ID but incomplete key material.
	e.updateSettings(t, p, func(s *adminv1.ProjectSettings) {
		s.Apple.Enabled = true
		s.Apple.ServicesId = appleServicesID
	})
	if code, _ := start("apple"); code != http.StatusBadRequest {
		t.Fatalf("apple without key material: want 400, got %d", code)
	}
}

// TestOAuthHTTPRateLimited: the plain-HTTP OAuth endpoints sit outside the
// connect interceptors, so they carry their own per-IP throttle.
func TestOAuthHTTPRateLimited(t *testing.T) {
	pd := newProviderDoubles(t)
	e := newTestEnv(t, "tok", withProviderDoubles(pd), func(o *Options) {
		o.RateLimit = &ratelimit.Config{
			IP: ratelimit.Tier{Limit: 2, Window: time.Minute},
		}
	})
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Throttled App")
	e.updateSettings(t, p, func(s *adminv1.ProjectSettings) {
		s.Google.Enabled = true
		s.Google.WebClientId = googleWebClient
		s.Google.WebClientSecret = "web-secret-1"
		s.RedirectSchemes = []string{"myapp"}
	})
	hc := noRedirectClient()

	codes := make([]int, 0, 3)
	var retryAfter string
	for range 3 {
		resp, err := hc.Get(e.url + "/oauth/google/start?project=" + p.Slug +
			"&redirect=" + url.QueryEscape("myapp://auth"))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		codes = append(codes, resp.StatusCode)
		retryAfter = resp.Header.Get("Retry-After")
	}
	if codes[0] != http.StatusFound || codes[1] != http.StatusFound {
		t.Fatalf("first two starts should pass: %v", codes)
	}
	if codes[2] != http.StatusTooManyRequests {
		t.Fatalf("third start should be throttled: %v", codes)
	}
	// The 429 must advertise a Retry-After so browsers/clients can back off.
	if retryAfter == "" {
		t.Fatal("throttled response missing Retry-After header")
	}
}
