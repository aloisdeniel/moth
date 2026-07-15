// Package jwt signs and verifies moth access tokens: compact JWS with
// ES256, a kid header, and the fixed claim set moth mints. It is
// deliberately not a general JOSE library — third-party libraries verify
// these tokens against the project JWKS; this package only has to produce
// and check moth's own.
package jwt

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Verification errors.
var (
	ErrMalformed        = errors.New("malformed token")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrExpired          = errors.New("token expired")
)

// Claims is the moth access-token claim set.
type Claims struct {
	Issuer        string `json:"iss,omitempty"`
	Subject       string `json:"sub,omitempty"`
	Audience      string `json:"aud,omitempty"`
	IssuedAt      int64  `json:"iat,omitempty"`
	ExpiresAt     int64  `json:"exp,omitempty"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified"`
	// Custom is the user's custom_claims object (roles, permissions, ...),
	// embedded verbatim under the "claims" claim.
	Custom map[string]any `json:"claims,omitempty"`
}

type header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

var b64 = base64.RawURLEncoding

// Sign returns the compact ES256 JWS of claims, with kid in the header.
func Sign(priv *ecdsa.PrivateKey, kid string, claims Claims) (string, error) {
	head, err := json.Marshal(header{Alg: "ES256", Typ: "JWT", Kid: kid})
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := b64.EncodeToString(head) + "." + b64.EncodeToString(body)
	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, priv, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}
	// JWS ES256 signature: r and s as fixed 32-byte big-endian values.
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	return signingInput + "." + b64.EncodeToString(sig), nil
}

// Kid returns the kid header of a token without verifying it, so callers
// can pick the right key.
func Kid(token string) (string, error) {
	head, err := decodeHeader(token)
	if err != nil {
		return "", err
	}
	return head.Kid, nil
}

// ParseUnverified decodes the claims WITHOUT checking the signature or
// expiry. Introspection uses it to identify which project a token claims
// to belong to before verification; never trust its output for
// authentication.
func ParseUnverified(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrMalformed
	}
	body, err := b64.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrMalformed
	}
	var claims Claims
	if err := json.Unmarshal(body, &claims); err != nil {
		return Claims{}, ErrMalformed
	}
	return claims, nil
}

// Verify checks the signature and expiry of a compact ES256 JWS and
// returns its claims. lookup resolves the header kid to a public key;
// issuer/audience checks are the caller's job.
func Verify(token string, lookup func(kid string) (*ecdsa.PublicKey, error), now time.Time) (Claims, error) {
	head, err := decodeHeader(token)
	if err != nil {
		return Claims{}, err
	}
	if head.Alg != "ES256" {
		return Claims{}, fmt.Errorf("%w: unexpected alg %q", ErrMalformed, head.Alg)
	}
	pub, err := lookup(head.Kid)
	if err != nil {
		return Claims{}, err
	}

	parts := strings.Split(token, ".")
	sig, err := b64.DecodeString(parts[2])
	if err != nil || len(sig) != 64 {
		return Claims{}, ErrMalformed
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])
	if !ecdsa.Verify(pub, digest[:], r, s) {
		return Claims{}, ErrInvalidSignature
	}

	body, err := b64.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrMalformed
	}
	var claims Claims
	if err := json.Unmarshal(body, &claims); err != nil {
		return Claims{}, ErrMalformed
	}
	if claims.ExpiresAt != 0 && !now.Before(time.Unix(claims.ExpiresAt, 0)) {
		return claims, ErrExpired
	}
	return claims, nil
}

func decodeHeader(token string) (header, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return header{}, ErrMalformed
	}
	raw, err := b64.DecodeString(parts[0])
	if err != nil {
		return header{}, ErrMalformed
	}
	var head header
	if err := json.Unmarshal(raw, &head); err != nil {
		return header{}, ErrMalformed
	}
	return head, nil
}
