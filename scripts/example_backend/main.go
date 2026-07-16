// Command example_backend is the "your own API" half of the moth loop: a
// tiny HTTP server that authenticates requests by verifying the moth access
// token (an ES256 JWT) against the project's public JWKS — exactly what any
// real backend does, in ~200 lines of standard library.
//
//	go run ./scripts/example_backend --issuer http://localhost:8080/p/<slug>
//
// The SDK example app's "Call my backend" button hits GET /api/hello here
// with `Authorization: Bearer <moth access token>`.
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

func main() {
	addr := flag.String("addr", ":8081", "listen address")
	issuer := flag.String("issuer", "",
		"expected token issuer: <moth base URL>/p/<project slug> (required)")
	flag.Parse()
	if *issuer == "" {
		log.Fatal("--issuer is required, e.g. --issuer http://localhost:8080/p/my-app")
	}

	verifier := &verifier{
		issuer:  strings.TrimSuffix(*issuer, "/"),
		jwksURL: strings.TrimSuffix(*issuer, "/") + "/.well-known/jwks.json",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/hello", func(w http.ResponseWriter, r *http.Request) {
		claims, err := verifier.authenticate(r)
		if err != nil {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"message":        fmt.Sprintf("Hello %s, your JWT checks out.", claims.Email),
			"user_id":        claims.Subject,
			"email":          claims.Email,
			"email_verified": claims.EmailVerified,
			"claims":         claims.Custom,
		})
	})

	log.Printf("example backend on %s (verifying tokens from %s)", *addr, verifier.jwksURL)
	log.Fatal(http.ListenAndServe(*addr, cors(mux)))
}

// cors makes the demo callable from Flutter Web dev builds; native apps
// don't need it.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// claims is the part of moth's access-token claim set this backend uses.
// Custom carries the project-assigned claims (roles, permissions, ...).
type claims struct {
	Issuer        string         `json:"iss"`
	Subject       string         `json:"sub"`
	ExpiresAt     int64          `json:"exp"`
	Email         string         `json:"email"`
	EmailVerified bool           `json:"email_verified"`
	Custom        map[string]any `json:"claims"`
}

// verifier checks moth access tokens against the project JWKS. Keys are
// cached; an unknown kid triggers a refetch (rotation) at most once per
// minute.
type verifier struct {
	issuer  string
	jwksURL string

	mu        sync.Mutex
	keys      map[string]*ecdsa.PublicKey
	lastFetch time.Time
}

// authenticate extracts and verifies the request's Bearer token.
func (v *verifier) authenticate(r *http.Request) (claims, error) {
	auth := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(auth, "Bearer ")
	if !ok {
		return claims{}, errors.New("missing bearer token")
	}
	return v.verify(token, time.Now())
}

// verify checks an ES256 compact JWS: signature against the JWKS key named
// by the kid header, then expiry and issuer.
func (v *verifier) verify(token string, now time.Time) (claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return claims{}, errors.New("malformed token")
	}

	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := decodeSegment(parts[0], &header); err != nil {
		return claims{}, errors.New("malformed token header")
	}
	if header.Alg != "ES256" {
		return claims{}, fmt.Errorf("unexpected alg %q", header.Alg)
	}
	key, err := v.key(header.Kid)
	if err != nil {
		return claims{}, err
	}

	// JWS ES256 signature: r and s as fixed 32-byte big-endian values over
	// the SHA-256 of "<header>.<payload>".
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(sig) != 64 {
		return claims{}, errors.New("malformed signature")
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])
	if !ecdsa.Verify(key, digest[:], r, s) {
		return claims{}, errors.New("invalid signature")
	}

	var c claims
	if err := decodeSegment(parts[1], &c); err != nil {
		return claims{}, errors.New("malformed claims")
	}
	if !now.Before(time.Unix(c.ExpiresAt, 0)) {
		return claims{}, errors.New("token expired")
	}
	if c.Issuer != v.issuer {
		return claims{}, fmt.Errorf("unexpected issuer %q", c.Issuer)
	}
	return c, nil
}

// key returns the public key for kid, refetching the JWKS when the kid is
// unknown (key rotation) but at most once per minute.
func (v *verifier) key(kid string) (*ecdsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if key, ok := v.keys[kid]; ok {
		return key, nil
	}
	if time.Since(v.lastFetch) < time.Minute {
		return nil, fmt.Errorf("unknown key %q", kid)
	}
	if err := v.fetchLocked(); err != nil {
		return nil, err
	}
	if key, ok := v.keys[kid]; ok {
		return key, nil
	}
	return nil, fmt.Errorf("unknown key %q", kid)
}

func (v *verifier) fetchLocked() error {
	v.lastFetch = time.Now()
	resp, err := http.Get(v.jwksURL)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch jwks: %s", resp.Status)
	}
	var doc struct {
		Keys []struct {
			Kty string `json:"kty"`
			Crv string `json:"crv"`
			X   string `json:"x"`
			Y   string `json:"y"`
			Kid string `json:"kid"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}
	keys := make(map[string]*ecdsa.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Kty != "EC" || k.Crv != "P-256" {
			continue
		}
		x, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			continue
		}
		y, err := base64.RawURLEncoding.DecodeString(k.Y)
		if err != nil {
			continue
		}
		keys[k.Kid] = &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(x),
			Y:     new(big.Int).SetBytes(y),
		}
	}
	v.keys = keys
	return nil
}

func decodeSegment(segment string, into any) error {
	raw, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, into)
}
