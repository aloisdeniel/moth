// Package oidc verifies Google and Apple ID tokens for social sign-in and
// talks to the providers' OAuth token endpoints. It is deliberately not a
// general OIDC client: no discovery, no provider SDKs — just RS256 JWS
// verification against a cached remote JWKS plus the two token-endpoint
// calls moth needs (Apple code exchange/revoke, Google web code exchange).
//
// moth's own tokens are ES256 and live in internal/jwt; this package only
// ever verifies third-party provider tokens, which are RS256.
package oidc

import (
	"errors"
	"net/http"
	"time"
)

// Verification errors.
var (
	ErrMalformed        = errors.New("malformed token")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrExpired          = errors.New("token expired")
	ErrIssuerMismatch   = errors.New("issuer mismatch")
	ErrAudienceMismatch = errors.New("audience mismatch")
	ErrNonceMismatch    = errors.New("nonce mismatch")
	ErrUnknownKey       = errors.New("unknown key id")
)

// Doer is the subset of *http.Client the package needs; injectable so tests
// and the server package can point it at doubles.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

func defaultDoer() Doer { return &http.Client{Timeout: 10 * time.Second} }

// NonceMode selects how a token's nonce claim relates to the raw per-attempt
// nonce the client sent in the RPC.
type NonceMode int

const (
	// NonceRaw: the token's nonce claim is the raw nonce itself. Google
	// echoes the nonce request parameter verbatim.
	NonceRaw NonceMode = iota
	// NonceSHA256Hex: the token's nonce claim is hex(SHA-256(raw)). Apple
	// echoes whatever the client sent it, and the SDKs send Apple the
	// SHA-256 hex digest (per Apple's scheme) while passing the raw value
	// in the RPC, so the server hashes before comparing.
	NonceSHA256Hex
)

// Provider describes one identity provider: which issuer values its tokens
// carry, where its JWKS lives, and its nonce convention. Fields are plain so
// tests and the server package can point them at test doubles.
type Provider struct {
	Name      string
	Issuers   []string // exact-match set for the iss claim
	JWKSURL   string
	NonceMode NonceMode
}

// Default endpoint locations, overridable on the clients for tests.
const (
	GoogleIssuer   = "https://accounts.google.com"
	GoogleJWKSURL  = "https://www.googleapis.com/oauth2/v3/certs"
	GoogleTokenURL = "https://oauth2.googleapis.com/token"
	AppleBaseURL   = "https://appleid.apple.com"
)

// Google returns the Google preset. Both issuer spellings are accepted:
// Google historically emitted "accounts.google.com" without a scheme.
func Google() Provider {
	return Provider{
		Name:      "google",
		Issuers:   []string{GoogleIssuer, "accounts.google.com"},
		JWKSURL:   GoogleJWKSURL,
		NonceMode: NonceRaw,
	}
}

// Apple returns the Apple preset.
func Apple() Provider {
	return Provider{
		Name:      "apple",
		Issuers:   []string{AppleBaseURL},
		JWKSURL:   AppleBaseURL + "/auth/keys",
		NonceMode: NonceSHA256Hex,
	}
}

// Identity is the normalized result of a verified ID token. All fields come
// from the verified token, never from client-asserted request fields.
type Identity struct {
	Issuer  string
	Subject string
	Email   string
	// EmailVerified is true only when the provider asserts it in the token
	// (Google sends a bool, Apple a bool or the string "true").
	EmailVerified bool
	Name          string
	GivenName     string
	FamilyName    string
	// ExpiresAt is the token's exp claim; callers use it to bound how long
	// a consumed token's hash must be remembered for replay rejection.
	ExpiresAt time.Time
}
