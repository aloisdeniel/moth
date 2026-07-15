package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"
)

func testKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return priv
}

func lookupFor(priv *ecdsa.PrivateKey, kid string) func(string) (*ecdsa.PublicKey, error) {
	return func(got string) (*ecdsa.PublicKey, error) {
		if got != kid {
			return nil, errors.New("unknown kid")
		}
		return &priv.PublicKey, nil
	}
}

func TestSignVerifyRoundTrip(t *testing.T) {
	priv := testKey(t)
	now := time.Now()
	in := Claims{
		Issuer: "https://moth.example/p/app", Subject: "user-1", Audience: "app",
		IssuedAt: now.Unix(), ExpiresAt: now.Add(15 * time.Minute).Unix(),
		Email: "u@example.com", EmailVerified: true,
		Custom: map[string]any{"role": "admin"},
	}
	tok, err := Sign(priv, "kid-1", in)
	if err != nil {
		t.Fatal(err)
	}

	kid, err := Kid(tok)
	if err != nil || kid != "kid-1" {
		t.Fatalf("Kid = %q, %v", kid, err)
	}
	out, err := Verify(tok, lookupFor(priv, "kid-1"), now)
	if err != nil {
		t.Fatal(err)
	}
	if out.Subject != in.Subject || out.Email != in.Email || !out.EmailVerified ||
		out.Audience != in.Audience || out.Custom["role"] != "admin" {
		t.Fatalf("claims did not round-trip: %+v", out)
	}
}

func TestVerifyRejectsTampering(t *testing.T) {
	priv := testKey(t)
	now := time.Now()
	tok, err := Sign(priv, "kid-1", Claims{Subject: "user-1", ExpiresAt: now.Add(time.Hour).Unix()})
	if err != nil {
		t.Fatal(err)
	}

	// Payload swap must invalidate the signature.
	forged, err := Sign(priv, "kid-1", Claims{Subject: "user-2", ExpiresAt: now.Add(time.Hour).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(tok, ".")
	forgedParts := strings.Split(forged, ".")
	tampered := parts[0] + "." + forgedParts[1] + "." + parts[2]
	if _, err := Verify(tampered, lookupFor(priv, "kid-1"), now); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("tampered payload: got %v, want ErrInvalidSignature", err)
	}

	// A different key must not verify.
	other := testKey(t)
	if _, err := Verify(tok, lookupFor(other, "kid-1"), now); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("wrong key: got %v, want ErrInvalidSignature", err)
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	priv := testKey(t)
	now := time.Now()
	tok, err := Sign(priv, "kid-1", Claims{Subject: "u", ExpiresAt: now.Add(-time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(tok, lookupFor(priv, "kid-1"), now); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired: got %v, want ErrExpired", err)
	}
}

func TestVerifyRejectsGarbage(t *testing.T) {
	priv := testKey(t)
	for _, tok := range []string{"", "abc", "a.b", "a.b.c.d", "!!!.???.###"} {
		if _, err := Verify(tok, lookupFor(priv, "kid-1"), time.Now()); !errors.Is(err, ErrMalformed) {
			t.Errorf("Verify(%q) = %v, want ErrMalformed", tok, err)
		}
	}
}
