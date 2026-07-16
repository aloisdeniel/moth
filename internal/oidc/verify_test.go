package oidc

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	testGoogleAud = "12345-abc.apps.googleusercontent.com"
	testAppleAud  = "com.example.app"
	testNonce     = "raw-nonce-42"
)

func hashedNonce(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func TestVerify(t *testing.T) {
	priv := testRSAKey(t)
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	nowFn := func() time.Time { return now }

	jwks := newJWKSDouble(t)
	jwks.setKey("kid-1", &priv.PublicKey)

	googleClaims := func(mutate func(map[string]any)) map[string]any {
		c := map[string]any{
			"iss":            "https://accounts.google.com",
			"sub":            "google-sub-1",
			"aud":            testGoogleAud,
			"iat":            now.Add(-time.Minute).Unix(),
			"exp":            now.Add(time.Hour).Unix(),
			"nonce":          testNonce,
			"email":          "alice@example.com",
			"email_verified": true,
			"name":           "Alice Doe",
			"given_name":     "Alice",
			"family_name":    "Doe",
		}
		if mutate != nil {
			mutate(c)
		}
		return c
	}
	appleClaims := func(mutate func(map[string]any)) map[string]any {
		c := map[string]any{
			"iss":            "https://appleid.apple.com",
			"sub":            "apple-sub-1",
			"aud":            testAppleAud,
			"iat":            now.Add(-time.Minute).Unix(),
			"exp":            now.Add(time.Hour).Unix(),
			"nonce":          hashedNonce(testNonce),
			"email":          "bob@privaterelay.appleid.com",
			"email_verified": "true", // Apple's string variant
		}
		if mutate != nil {
			mutate(c)
		}
		return c
	}

	google := Google()
	google.JWKSURL = jwks.srv.URL
	apple := Apple()
	apple.JWKSURL = jwks.srv.URL

	tests := []struct {
		name      string
		provider  Provider
		token     func(t *testing.T) string
		audiences []string
		nonce     string
		wantErr   error
		check     func(t *testing.T, id Identity)
	}{
		{
			name:      "valid google",
			provider:  google,
			token:     func(t *testing.T) string { return signRS256(t, priv, "kid-1", googleClaims(nil)) },
			audiences: []string{testGoogleAud},
			nonce:     testNonce,
			check: func(t *testing.T, id Identity) {
				want := Identity{
					Issuer: "https://accounts.google.com", Subject: "google-sub-1",
					Email: "alice@example.com", EmailVerified: true,
					Name: "Alice Doe", GivenName: "Alice", FamilyName: "Doe",
					ExpiresAt: time.Unix(now.Add(time.Hour).Unix(), 0),
				}
				if id != want {
					t.Errorf("identity = %+v, want %+v", id, want)
				}
			},
		},
		{
			name:     "valid google schemeless issuer",
			provider: google,
			token: func(t *testing.T) string {
				return signRS256(t, priv, "kid-1", googleClaims(func(c map[string]any) {
					c["iss"] = "accounts.google.com"
				}))
			},
			audiences: []string{testGoogleAud},
			nonce:     testNonce,
		},
		{
			name:     "valid apple hashed nonce and string email_verified",
			provider: apple,
			token:    func(t *testing.T) string { return signRS256(t, priv, "kid-1", appleClaims(nil)) },
			audiences: []string{
				"com.example.services", testAppleAud, // aud set membership
			},
			nonce: testNonce,
			check: func(t *testing.T, id Identity) {
				if id.Subject != "apple-sub-1" || !id.EmailVerified {
					t.Errorf("identity = %+v, want verified apple-sub-1", id)
				}
			},
		},
		{
			name:     "apple email_verified bool variant",
			provider: apple,
			token: func(t *testing.T) string {
				return signRS256(t, priv, "kid-1", appleClaims(func(c map[string]any) {
					c["email_verified"] = true
				}))
			},
			audiences: []string{testAppleAud},
			nonce:     testNonce,
			check: func(t *testing.T, id Identity) {
				if !id.EmailVerified {
					t.Error("EmailVerified = false, want true")
				}
			},
		},
		{
			name:     "email_verified string false stays unverified",
			provider: apple,
			token: func(t *testing.T) string {
				return signRS256(t, priv, "kid-1", appleClaims(func(c map[string]any) {
					c["email_verified"] = "false"
				}))
			},
			audiences: []string{testAppleAud},
			nonce:     testNonce,
			check: func(t *testing.T, id Identity) {
				if id.EmailVerified {
					t.Error("EmailVerified = true, want false")
				}
			},
		},
		{
			name:     "aud array accepted",
			provider: google,
			token: func(t *testing.T) string {
				return signRS256(t, priv, "kid-1", googleClaims(func(c map[string]any) {
					c["aud"] = []string{"other-client", testGoogleAud}
				}))
			},
			audiences: []string{testGoogleAud},
			nonce:     testNonce,
		},
		{
			name:     "wrong aud",
			provider: google,
			token: func(t *testing.T) string {
				return signRS256(t, priv, "kid-1", googleClaims(func(c map[string]any) {
					c["aud"] = "someone-elses-client"
				}))
			},
			audiences: []string{testGoogleAud},
			nonce:     testNonce,
			wantErr:   ErrAudienceMismatch,
		},
		{
			name:     "wrong issuer",
			provider: google,
			token: func(t *testing.T) string {
				return signRS256(t, priv, "kid-1", googleClaims(func(c map[string]any) {
					c["iss"] = "https://evil.example.com"
				}))
			},
			audiences: []string{testGoogleAud},
			nonce:     testNonce,
			wantErr:   ErrIssuerMismatch,
		},
		{
			name:     "expired",
			provider: google,
			token: func(t *testing.T) string {
				return signRS256(t, priv, "kid-1", googleClaims(func(c map[string]any) {
					c["exp"] = now.Add(-2 * time.Minute).Unix()
				}))
			},
			audiences: []string{testGoogleAud},
			nonce:     testNonce,
			wantErr:   ErrExpired,
		},
		{
			name:     "issued in the future",
			provider: google,
			token: func(t *testing.T) string {
				return signRS256(t, priv, "kid-1", googleClaims(func(c map[string]any) {
					c["iat"] = now.Add(10 * time.Minute).Unix()
				}))
			},
			audiences: []string{testGoogleAud},
			nonce:     testNonce,
			wantErr:   ErrExpired,
		},
		{
			name:     "tampered signature",
			provider: google,
			token: func(t *testing.T) string {
				tok := signRS256(t, priv, "kid-1", googleClaims(nil))
				forged := googleClaims(func(c map[string]any) { c["email"] = "mallory@example.com" })
				parts := strings.Split(tok, ".")
				return parts[0] + "." + encodeJWTPart(t, forged) + "." + parts[2]
			},
			audiences: []string{testGoogleAud},
			nonce:     testNonce,
			wantErr:   ErrInvalidSignature,
		},
		{
			name:     "wrong nonce",
			provider: google,
			token: func(t *testing.T) string {
				return signRS256(t, priv, "kid-1", googleClaims(func(c map[string]any) {
					c["nonce"] = "replayed-other-nonce"
				}))
			},
			audiences: []string{testGoogleAud},
			nonce:     testNonce,
			wantErr:   ErrNonceMismatch,
		},
		{
			name:     "missing nonce claim",
			provider: google,
			token: func(t *testing.T) string {
				return signRS256(t, priv, "kid-1", googleClaims(func(c map[string]any) {
					delete(c, "nonce")
				}))
			},
			audiences: []string{testGoogleAud},
			nonce:     testNonce,
			wantErr:   ErrNonceMismatch,
		},
		{
			name:     "apple raw nonce where hash expected",
			provider: apple,
			token: func(t *testing.T) string {
				return signRS256(t, priv, "kid-1", appleClaims(func(c map[string]any) {
					c["nonce"] = testNonce // client bypassed Apple's hashing scheme
				}))
			},
			audiences: []string{testAppleAud},
			nonce:     testNonce,
			wantErr:   ErrNonceMismatch,
		},
		{
			name:     "alg none rejected",
			provider: google,
			token: func(t *testing.T) string {
				head := encodeJWTPart(t, map[string]string{"alg": "none", "kid": "kid-1"})
				return head + "." + encodeJWTPart(t, googleClaims(nil)) + "."
			},
			audiences: []string{testGoogleAud},
			nonce:     testNonce,
			wantErr:   ErrMalformed,
		},
		{
			// Key-confusion attack: alg=HS256 with the RSA public key's PEM
			// as the HMAC secret must die on the alg check, never reach a
			// symmetric verify.
			name:     "HS256 with public key as secret rejected",
			provider: google,
			token: func(t *testing.T) string {
				pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
				if err != nil {
					t.Fatal(err)
				}
				pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
				signingInput := encodeJWTPart(t, map[string]string{"alg": "HS256", "typ": "JWT", "kid": "kid-1"}) +
					"." + encodeJWTPart(t, googleClaims(nil))
				mac := hmac.New(sha256.New, pubPEM)
				mac.Write([]byte(signingInput))
				return signingInput + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
			},
			audiences: []string{testGoogleAud},
			nonce:     testNonce,
			wantErr:   ErrMalformed,
		},
		{
			name:      "garbage token",
			provider:  google,
			token:     func(t *testing.T) string { return "not-a-jwt" },
			audiences: []string{testGoogleAud},
			nonce:     testNonce,
			wantErr:   ErrMalformed,
		},
		{
			name:     "missing sub",
			provider: google,
			token: func(t *testing.T) string {
				return signRS256(t, priv, "kid-1", googleClaims(func(c map[string]any) {
					delete(c, "sub")
				}))
			},
			audiences: []string{testGoogleAud},
			nonce:     testNonce,
			wantErr:   ErrMalformed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewVerifier(tt.provider, jwks.srv.Client(), nowFn)
			id, err := v.Verify(t.Context(), tt.token(t), tt.audiences, tt.nonce)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Verify() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Verify() error = %v", err)
			}
			if tt.check != nil {
				tt.check(t, id)
			}
		})
	}
}
