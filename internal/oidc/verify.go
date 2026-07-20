package oidc

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Leeway absorbs small clock skew between moth and the provider when
// checking exp and iat.
const defaultLeeway = time.Minute

var b64 = base64.RawURLEncoding

// Verifier checks provider ID tokens for one provider against its JWKS.
type Verifier struct {
	provider Provider
	keys     *KeySet
	now      func() time.Time
	leeway   time.Duration
}

// NewVerifier returns a verifier for p. httpc and now default to a
// timeout-bounded client and time.Now when nil.
func NewVerifier(p Provider, httpc Doer, now func() time.Time) *Verifier {
	if now == nil {
		now = time.Now
	}
	return &Verifier{
		provider: p,
		keys:     NewKeySet(p.JWKSURL, httpc, now),
		now:      now,
		leeway:   defaultLeeway,
	}
}

// idTokenClaims is the provider claim superset moth reads. aud may be a
// string or an array; Apple sends email_verified as a bool or the string
// "true", hence the custom types.
type idTokenClaims struct {
	Issuer        string    `json:"iss"`
	Subject       string    `json:"sub"`
	Audience      audience  `json:"aud"`
	ExpiresAt     int64     `json:"exp"`
	IssuedAt      int64     `json:"iat"`
	Nonce         string    `json:"nonce"`
	Email         string    `json:"email"`
	EmailVerified looseBool `json:"email_verified"`
	Name          string    `json:"name"`
	GivenName     string    `json:"given_name"`
	FamilyName    string    `json:"family_name"`
}

type audience []string

func (a *audience) UnmarshalJSON(data []byte) error {
	var one string
	if err := json.Unmarshal(data, &one); err == nil {
		*a = audience{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*a = audience(many)
	return nil
}

type looseBool bool

func (b *looseBool) UnmarshalJSON(data []byte) error {
	var v bool
	if err := json.Unmarshal(data, &v); err == nil {
		*b = looseBool(v)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("email_verified: %s", data)
	}
	*b = s == "true"
	return nil
}

// Verify checks an ID token's signature (RS256 only — any other alg,
// including "none" and HS256, is rejected before key material is touched),
// issuer, audience membership, exp/iat and nonce, and returns the
// normalized identity. rawNonce is the per-attempt nonce the client sent in
// the RPC; the comparison against the token's nonce claim follows the
// provider's NonceMode. An empty rawNonce skips the nonce check — only the
// web-redirect flow, where the signed state parameter binds the attempt,
// may pass it empty.
func (v *Verifier) Verify(ctx context.Context, idToken string, audiences []string, rawNonce string) (Identity, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return Identity{}, ErrMalformed
	}
	headRaw, err := b64.DecodeString(parts[0])
	if err != nil {
		return Identity{}, ErrMalformed
	}
	var head struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headRaw, &head); err != nil {
		return Identity{}, ErrMalformed
	}
	if head.Alg != "RS256" {
		return Identity{}, fmt.Errorf("%w: unexpected alg %q", ErrMalformed, head.Alg)
	}
	sig, err := b64.DecodeString(parts[2])
	if err != nil {
		return Identity{}, ErrMalformed
	}

	pub, err := v.keys.Key(ctx, head.Kid)
	if err != nil {
		return Identity{}, err
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig); err != nil {
		return Identity{}, ErrInvalidSignature
	}

	body, err := b64.DecodeString(parts[1])
	if err != nil {
		return Identity{}, ErrMalformed
	}
	var claims idTokenClaims
	if err := json.Unmarshal(body, &claims); err != nil {
		return Identity{}, ErrMalformed
	}

	if !slices.Contains(v.provider.Issuers, claims.Issuer) {
		return Identity{}, fmt.Errorf("%w: %q", ErrIssuerMismatch, claims.Issuer)
	}
	if !slices.ContainsFunc(claims.Audience, func(aud string) bool {
		return slices.Contains(audiences, aud)
	}) {
		return Identity{}, fmt.Errorf("%w: %v", ErrAudienceMismatch, []string(claims.Audience))
	}
	now := v.now()
	if claims.ExpiresAt == 0 || now.After(time.Unix(claims.ExpiresAt, 0).Add(v.leeway)) {
		return Identity{}, ErrExpired
	}
	if claims.IssuedAt != 0 && time.Unix(claims.IssuedAt, 0).After(now.Add(v.leeway)) {
		return Identity{}, fmt.Errorf("%w: issued in the future", ErrExpired)
	}
	if rawNonce != "" {
		expected := rawNonce
		if v.provider.NonceMode == NonceSHA256Hex {
			sum := sha256.Sum256([]byte(rawNonce))
			expected = hex.EncodeToString(sum[:])
		}
		if subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(expected)) != 1 {
			return Identity{}, ErrNonceMismatch
		}
	}
	if claims.Subject == "" {
		return Identity{}, fmt.Errorf("%w: missing sub", ErrMalformed)
	}

	return Identity{
		Issuer:        claims.Issuer,
		Subject:       claims.Subject,
		Email:         claims.Email,
		EmailVerified: bool(claims.EmailVerified),
		Name:          claims.Name,
		GivenName:     claims.GivenName,
		FamilyName:    claims.FamilyName,
		ExpiresAt:     time.Unix(claims.ExpiresAt, 0),
	}, nil
}
