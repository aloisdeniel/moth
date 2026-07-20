package oidc

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// rsaTestKey is generated once: 2048-bit keygen is slow enough to matter
// across the table-driven cases.
var (
	rsaKeyOnce sync.Once
	rsaKey     *rsa.PrivateKey
)

func testRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	rsaKeyOnce.Do(func() {
		var err error
		rsaKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("generate RSA key: %v", err)
		}
	})
	return rsaKey
}

// jwksDouble is an httptest JWKS endpoint with a mutable key set and a
// fetch counter.
type jwksDouble struct {
	srv *httptest.Server

	mu           sync.Mutex
	keys         map[string]*rsa.PublicKey
	cacheControl string
	fetches      int
}

func newJWKSDouble(t *testing.T) *jwksDouble {
	t.Helper()
	d := &jwksDouble{keys: map[string]*rsa.PublicKey{}}
	d.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.fetches++
		if d.cacheControl != "" {
			w.Header().Set("Cache-Control", d.cacheControl)
		}
		type jwk struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			Use string `json:"use"`
			Alg string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
		}
		doc := struct {
			Keys []jwk `json:"keys"`
		}{Keys: []jwk{}}
		for kid, pub := range d.keys {
			doc.Keys = append(doc.Keys, jwk{
				Kty: "RSA", Kid: kid, Use: "sig", Alg: "RS256",
				N: base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
				E: base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(doc)
	}))
	t.Cleanup(d.srv.Close)
	return d
}

func (d *jwksDouble) setKey(kid string, pub *rsa.PublicKey) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.keys[kid] = pub
}

func (d *jwksDouble) fetchCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.fetches
}

func encodeJWTPart(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal JWT part: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

// signRS256 builds a compact RS256 JWS over arbitrary claims.
func signRS256(t *testing.T, priv *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	signingInput := encodeJWTPart(t, map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid}) +
		"." + encodeJWTPart(t, claims)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign RS256: %v", err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}
