package oidc

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"
)

// Apple accepts client secrets up to six months old; a short lifetime keeps
// a leaked secret's window small while the cache avoids re-signing per call.
const defaultAppleSecretLifetime = 6 * time.Hour

// ParseP8 parses an Apple developer private key (.p8 file: PEM-wrapped
// PKCS#8 EC P-256).
func ParseP8(data []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid .p8 PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse .p8: %w", err)
	}
	ec, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New(".p8 key is not ECDSA")
	}
	return ec, nil
}

// AppleSecretConfig identifies the Apple developer key that signs client
// secrets and the client they authenticate.
type AppleSecretConfig struct {
	TeamID   string // iss
	KeyID    string // JWS kid header (Apple Key ID, from the developer portal)
	ClientID string // sub: the Services ID (web) or bundle ID (native)
	Key      *ecdsa.PrivateKey
	// Audience defaults to AppleBaseURL; overridable for tests.
	Audience string
	// Lifetime defaults to defaultAppleSecretLifetime.
	Lifetime time.Duration
}

// AppleSecrets generates Apple client secrets (ES256 JWTs signed with the
// developer .p8) and caches each one until ~80% of its lifetime has elapsed.
type AppleSecrets struct {
	cfg AppleSecretConfig
	now func() time.Time

	mu      sync.Mutex
	secret  string
	renewAt time.Time
}

// NewAppleSecrets returns a cached client-secret generator. now defaults to
// time.Now when nil.
func NewAppleSecrets(cfg AppleSecretConfig, now func() time.Time) *AppleSecrets {
	if cfg.Audience == "" {
		cfg.Audience = AppleBaseURL
	}
	if cfg.Lifetime <= 0 {
		cfg.Lifetime = defaultAppleSecretLifetime
	}
	if now == nil {
		now = time.Now
	}
	return &AppleSecrets{cfg: cfg, now: now}
}

// ClientSecret returns a currently valid client secret, minting a fresh one
// when the cached secret is past 80% of its lifetime.
func (s *AppleSecrets) ClientSecret() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if s.secret != "" && now.Before(s.renewAt) {
		return s.secret, nil
	}
	claims := map[string]any{
		"iss": s.cfg.TeamID,
		"sub": s.cfg.ClientID,
		"aud": s.cfg.Audience,
		"iat": now.Unix(),
		"exp": now.Add(s.cfg.Lifetime).Unix(),
	}
	secret, err := signES256(s.cfg.Key, s.cfg.KeyID, claims)
	if err != nil {
		return "", err
	}
	s.secret = secret
	s.renewAt = now.Add(s.cfg.Lifetime * 8 / 10)
	return secret, nil
}

// signES256 produces a compact ES256 JWS with only the claims Apple wants —
// internal/jwt.Claims is moth-shaped (always emits email_verified), so the
// client secret gets its own minimal signer.
func signES256(priv *ecdsa.PrivateKey, kid string, claims map[string]any) (string, error) {
	head, err := json.Marshal(map[string]string{"alg": "ES256", "kid": kid})
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := b64.EncodeToString(head) + "." + b64.EncodeToString(body)
	digest := sha256.Sum256([]byte(signingInput))
	r, sv, err := ecdsa.Sign(rand.Reader, priv, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign client secret: %w", err)
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	sv.FillBytes(sig[32:])
	return signingInput + "." + b64.EncodeToString(sig), nil
}

// AppleClient talks to Apple's OAuth token endpoints.
type AppleClient struct {
	baseURL  string
	clientID string
	secrets  *AppleSecrets
	httpc    Doer
}

// NewAppleClient returns a client for Apple's token endpoints under baseURL
// (defaults to AppleBaseURL when empty; overridable for tests). httpc
// defaults to a timeout-bounded client when nil.
func NewAppleClient(baseURL, clientID string, secrets *AppleSecrets, httpc Doer) *AppleClient {
	if baseURL == "" {
		baseURL = AppleBaseURL
	}
	if httpc == nil {
		httpc = defaultDoer()
	}
	return &AppleClient{baseURL: baseURL, clientID: clientID, secrets: secrets, httpc: httpc}
}

// ExchangeCode trades an authorization code for tokens at {base}/auth/token.
// redirectURI must match the one used to obtain the code; native-app codes
// have none, so it may be empty.
func (c *AppleClient) ExchangeCode(ctx context.Context, code, redirectURI string) (TokenResponse, error) {
	secret, err := c.secrets.ClientSecret()
	if err != nil {
		return TokenResponse{}, err
	}
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {c.clientID},
		"client_secret": {secret},
	}
	if redirectURI != "" {
		form.Set("redirect_uri", redirectURI)
	}
	return postTokenForm(ctx, c.httpc, c.baseURL+"/auth/token", form)
}

// Revoke invalidates a refresh token at {base}/auth/token/revoke, as App
// Store review requires on account deletion.
func (c *AppleClient) Revoke(ctx context.Context, refreshToken string) error {
	secret, err := c.secrets.ClientSecret()
	if err != nil {
		return err
	}
	form := url.Values{
		"client_id":       {c.clientID},
		"client_secret":   {secret},
		"token":           {refreshToken},
		"token_type_hint": {"refresh_token"},
	}
	_, err = postForm(ctx, c.httpc, c.baseURL+"/auth/token/revoke", form)
	return err
}
