package oidc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func testP8(t *testing.T) ([]byte, *ecdsa.PrivateKey) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), priv
}

func newTestSecrets(t *testing.T, priv *ecdsa.PrivateKey, now func() time.Time) *AppleSecrets {
	t.Helper()
	return NewAppleSecrets(AppleSecretConfig{
		TeamID:   "TEAM123456",
		KeyID:    "KEY1234567",
		ClientID: "com.example.services",
		Key:      priv,
	}, now)
}

// verifyES256 decodes an Apple client secret, checks its signature with the
// developer public key, and returns header and claims.
func verifyES256(t *testing.T, token string, pub *ecdsa.PublicKey) (header, claims map[string]any) {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("client secret has %d parts, want 3", len(parts))
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(sig) != 64 {
		t.Fatalf("bad signature encoding: %v", err)
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])
	if !ecdsa.Verify(pub, digest[:], r, s) {
		t.Fatal("client secret signature does not verify")
	}
	for i, dst := range []*map[string]any{&header, &claims} {
		raw, err := base64.RawURLEncoding.DecodeString(parts[i])
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, dst); err != nil {
			t.Fatal(err)
		}
	}
	return header, claims
}

func TestAppleClientSecretRoundTrip(t *testing.T) {
	p8, want := testP8(t)
	priv, err := ParseP8(p8)
	if err != nil {
		t.Fatalf("ParseP8() error = %v", err)
	}
	if !priv.Equal(want) {
		t.Fatal("ParseP8() returned a different key")
	}

	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	secrets := newTestSecrets(t, priv, func() time.Time { return now })
	secret, err := secrets.ClientSecret()
	if err != nil {
		t.Fatalf("ClientSecret() error = %v", err)
	}

	header, claims := verifyES256(t, secret, &priv.PublicKey)
	if header["alg"] != "ES256" || header["kid"] != "KEY1234567" {
		t.Errorf("header = %v, want alg ES256 kid KEY1234567", header)
	}
	if claims["iss"] != "TEAM123456" || claims["sub"] != "com.example.services" ||
		claims["aud"] != "https://appleid.apple.com" {
		t.Errorf("claims = %v", claims)
	}
	iat, exp := int64(claims["iat"].(float64)), int64(claims["exp"].(float64))
	if iat != now.Unix() {
		t.Errorf("iat = %d, want %d", iat, now.Unix())
	}
	if got := time.Duration(exp-iat) * time.Second; got != defaultAppleSecretLifetime {
		t.Errorf("lifetime = %v, want %v", got, defaultAppleSecretLifetime)
	}
}

func TestAppleClientSecretCached(t *testing.T) {
	_, priv := testP8(t)
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	secrets := newTestSecrets(t, priv, func() time.Time { return now })

	first, err := secrets.ClientSecret()
	if err != nil {
		t.Fatal(err)
	}

	// Before 80% of the lifetime: the exact same cached string.
	now = now.Add(defaultAppleSecretLifetime * 7 / 10)
	again, err := secrets.ClientSecret()
	if err != nil {
		t.Fatal(err)
	}
	if again != first {
		t.Error("secret regenerated before 80% of lifetime")
	}

	// Past 80%: a fresh secret with a new iat.
	now = now.Add(defaultAppleSecretLifetime * 2 / 10)
	renewed, err := secrets.ClientSecret()
	if err != nil {
		t.Fatal(err)
	}
	if renewed == first {
		t.Error("secret not regenerated after 80% of lifetime")
	}
	_, claims := verifyES256(t, renewed, &priv.PublicKey)
	if got := int64(claims["iat"].(float64)); got != now.Unix() {
		t.Errorf("renewed iat = %d, want %d", got, now.Unix())
	}
}

// tokenEndpointDouble records the last form post per path and serves canned
// responses.
type tokenEndpointDouble struct {
	srv   *httptest.Server
	forms map[string]url.Values
}

func newTokenEndpointDouble(t *testing.T, status int, body string) *tokenEndpointDouble {
	t.Helper()
	d := &tokenEndpointDouble{forms: map[string]url.Values{}}
	d.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		d.forms[r.URL.Path] = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(d.srv.Close)
	return d
}

func TestAppleExchangeCodeAndRevoke(t *testing.T) {
	_, priv := testP8(t)
	secrets := newTestSecrets(t, priv, nil)
	endpoint := newTokenEndpointDouble(t, http.StatusOK,
		`{"access_token":"at","token_type":"Bearer","expires_in":3600,"refresh_token":"rt-1","id_token":"idt"}`)
	client := NewAppleClient(endpoint.srv.URL, "com.example.services", secrets, endpoint.srv.Client())

	tokens, err := client.ExchangeCode(t.Context(), "auth-code-1", "")
	if err != nil {
		t.Fatalf("ExchangeCode() error = %v", err)
	}
	if tokens.RefreshToken != "rt-1" || tokens.IDToken != "idt" || tokens.ExpiresIn != 3600 {
		t.Errorf("tokens = %+v", tokens)
	}
	form := endpoint.forms["/auth/token"]
	if form == nil {
		t.Fatal("no POST to /auth/token")
	}
	if form.Get("grant_type") != "authorization_code" || form.Get("code") != "auth-code-1" ||
		form.Get("client_id") != "com.example.services" {
		t.Errorf("exchange form = %v", form)
	}
	if form.Has("redirect_uri") {
		t.Error("redirect_uri sent for a native-app exchange")
	}
	secret, err := secrets.ClientSecret()
	if err != nil {
		t.Fatal(err)
	}
	if form.Get("client_secret") != secret {
		t.Error("client_secret does not match the cached generator output")
	}

	if err := client.Revoke(t.Context(), "rt-1"); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	form = endpoint.forms["/auth/token/revoke"]
	if form == nil {
		t.Fatal("no POST to /auth/token/revoke")
	}
	if form.Get("token") != "rt-1" || form.Get("token_type_hint") != "refresh_token" ||
		form.Get("client_id") != "com.example.services" || form.Get("client_secret") != secret {
		t.Errorf("revoke form = %v", form)
	}
}

func TestAppleExchangeCodeOAuthError(t *testing.T) {
	_, priv := testP8(t)
	secrets := newTestSecrets(t, priv, nil)
	endpoint := newTokenEndpointDouble(t, http.StatusBadRequest, `{"error":"invalid_grant"}`)
	client := NewAppleClient(endpoint.srv.URL, "com.example.services", secrets, endpoint.srv.Client())

	_, err := client.ExchangeCode(t.Context(), "bad-code", "")
	var tokErr *TokenError
	if !errors.As(err, &tokErr) {
		t.Fatalf("error = %v, want *TokenError", err)
	}
	if tokErr.StatusCode != http.StatusBadRequest || tokErr.Code != "invalid_grant" {
		t.Errorf("TokenError = %+v", tokErr)
	}
}
