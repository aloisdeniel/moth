package oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"
)

func TestKeySetCachesUntilExpiry(t *testing.T) {
	priv := testRSAKey(t)
	jwks := newJWKSDouble(t)
	jwks.setKey("kid-1", &priv.PublicKey)
	jwks.cacheControl = "public, max-age=120"

	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	ks := NewKeySet(jwks.srv.URL, jwks.srv.Client(), func() time.Time { return now })

	for range 3 {
		if _, err := ks.Key(t.Context(), "kid-1"); err != nil {
			t.Fatalf("Key() error = %v", err)
		}
	}
	if got := jwks.fetchCount(); got != 1 {
		t.Fatalf("fetches within TTL = %d, want 1", got)
	}

	now = now.Add(119 * time.Second)
	if _, err := ks.Key(t.Context(), "kid-1"); err != nil {
		t.Fatalf("Key() error = %v", err)
	}
	if got := jwks.fetchCount(); got != 1 {
		t.Fatalf("fetches just before expiry = %d, want 1", got)
	}

	now = now.Add(2 * time.Second)
	if _, err := ks.Key(t.Context(), "kid-1"); err != nil {
		t.Fatalf("Key() error = %v", err)
	}
	if got := jwks.fetchCount(); got != 2 {
		t.Fatalf("fetches after expiry = %d, want 2", got)
	}
}

func TestKeySetTTLClamped(t *testing.T) {
	priv := testRSAKey(t)
	jwks := newJWKSDouble(t)
	jwks.setKey("kid-1", &priv.PublicKey)
	jwks.cacheControl = "max-age=1" // below the floor; must be clamped up

	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	ks := NewKeySet(jwks.srv.URL, jwks.srv.Client(), func() time.Time { return now })

	if _, err := ks.Key(t.Context(), "kid-1"); err != nil {
		t.Fatalf("Key() error = %v", err)
	}
	now = now.Add(30 * time.Second)
	if _, err := ks.Key(t.Context(), "kid-1"); err != nil {
		t.Fatalf("Key() error = %v", err)
	}
	if got := jwks.fetchCount(); got != 1 {
		t.Fatalf("fetches = %d, want 1 (max-age clamped to floor)", got)
	}
}

func TestKeySetUnknownKidRefetchBounded(t *testing.T) {
	priv := testRSAKey(t)
	rotated, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwks := newJWKSDouble(t)
	jwks.setKey("kid-1", &priv.PublicKey)

	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	ks := NewKeySet(jwks.srv.URL, jwks.srv.Client(), func() time.Time { return now })

	if _, err := ks.Key(t.Context(), "kid-1"); err != nil {
		t.Fatalf("Key(kid-1) error = %v", err)
	}

	// Provider rotates in a new key; the unknown kid triggers exactly one
	// refetch and then resolves.
	jwks.setKey("kid-2", &rotated.PublicKey)
	now = now.Add(unknownKidRefresh) // past the refetch rate limit
	pub, err := ks.Key(t.Context(), "kid-2")
	if err != nil {
		t.Fatalf("Key(kid-2) error = %v", err)
	}
	if pub.N.Cmp(rotated.N) != 0 {
		t.Fatal("Key(kid-2) returned the wrong key")
	}
	if got := jwks.fetchCount(); got != 2 {
		t.Fatalf("fetches after rotation = %d, want 2", got)
	}

	// A garbage kid right after must NOT hit the provider again.
	if _, err := ks.Key(t.Context(), "kid-garbage"); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("Key(kid-garbage) error = %v, want ErrUnknownKey", err)
	}
	if got := jwks.fetchCount(); got != 2 {
		t.Fatalf("fetches after garbage kid = %d, want 2 (rate-limited)", got)
	}

	// After the rate-limit window it may probe once more.
	now = now.Add(unknownKidRefresh)
	if _, err := ks.Key(t.Context(), "kid-garbage"); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("Key(kid-garbage) error = %v, want ErrUnknownKey", err)
	}
	if got := jwks.fetchCount(); got != 3 {
		t.Fatalf("fetches after window = %d, want 3", got)
	}
}

func TestParseRSAJWKSSkipsNonRSAAndSmallKeys(t *testing.T) {
	priv := testRSAKey(t)
	small, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	jwks := newJWKSDouble(t)
	jwks.setKey("kid-ok", &priv.PublicKey)
	jwks.setKey("kid-small", &small.PublicKey)

	ks := NewKeySet(jwks.srv.URL, jwks.srv.Client(), nil)
	if _, err := ks.Key(t.Context(), "kid-ok"); err != nil {
		t.Fatalf("Key(kid-ok) error = %v", err)
	}
	if _, err := ks.Key(t.Context(), "kid-small"); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("Key(kid-small) error = %v, want ErrUnknownKey (1024-bit key skipped)", err)
	}
}
