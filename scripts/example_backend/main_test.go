package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aloisdeniel/moth/internal/jwt"
	"github.com/aloisdeniel/moth/internal/keys"
)

// TestVerifyMothToken proves the developer-backend half of the loop: a
// token minted exactly like moth mints them (internal/jwt + a JWKS built by
// internal/keys) verifies against this backend's stdlib-only verifier.
func TestVerifyMothToken(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
	kid, err := keys.Thumbprint(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	jwks, err := keys.BuildJWKS(map[string]string{kid: pubPEM})
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/p/my-app/.well-known/jwks.json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwks)
	}))
	defer srv.Close()

	issuer := srv.URL + "/p/my-app"
	v := &verifier{issuer: issuer, jwksURL: issuer + "/.well-known/jwks.json"}
	now := time.Now()

	mint := func(c jwt.Claims) string {
		t.Helper()
		signed, err := jwt.Sign(priv, kid, c)
		if err != nil {
			t.Fatal(err)
		}
		return signed
	}
	valid := jwt.Claims{
		Issuer:        issuer,
		Subject:       "user-1",
		Audience:      "my-app",
		IssuedAt:      now.Unix(),
		ExpiresAt:     now.Add(time.Minute).Unix(),
		Email:         "jane@example.com",
		EmailVerified: true,
		Custom:        map[string]any{"role": "admin"},
	}

	got, err := v.verify(mint(valid), now)
	if err != nil {
		t.Fatalf("verify valid token: %v", err)
	}
	if got.Subject != "user-1" || got.Email != "jane@example.com" || !got.EmailVerified {
		t.Errorf("unexpected claims: %+v", got)
	}
	if got.Custom["role"] != "admin" {
		t.Errorf("custom claims not surfaced: %+v", got.Custom)
	}

	expired := valid
	expired.ExpiresAt = now.Add(-time.Minute).Unix()
	if _, err := v.verify(mint(expired), now); err == nil {
		t.Error("expired token verified")
	}

	wrongIssuer := valid
	wrongIssuer.Issuer = "https://evil.example.com/p/my-app"
	if _, err := v.verify(mint(wrongIssuer), now); err == nil {
		t.Error("token with wrong issuer verified")
	}

	// Signature by a different key (same kid claimed) must fail.
	otherPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	forged, err := jwt.Sign(otherPriv, kid, valid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.verify(forged, now); err == nil {
		t.Error("forged signature verified")
	}

	if _, err := v.verify("not.a.jwt", now); err == nil {
		t.Error("malformed token verified")
	}
}
