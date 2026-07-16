package oidc

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// JWKS cache tuning: the TTL honors the response's Cache-Control max-age,
// clamped to [minJWKSTTL, maxJWKSTTL], defaulting when absent. An unknown
// kid forces a refetch (key rotation) at most once per unknownKidRefresh.
const (
	defaultJWKSTTL    = time.Hour
	minJWKSTTL        = time.Minute
	maxJWKSTTL        = 24 * time.Hour
	unknownKidRefresh = 15 * time.Second
)

// KeySet fetches and caches a provider's RSA JWKS, resolving kids to public
// keys. Refreshes are single-flight: concurrent lookups share one fetch.
type KeySet struct {
	url   string
	httpc Doer
	now   func() time.Time

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
	fetchedAt time.Time
	inflight  chan struct{}
}

// NewKeySet returns a cache over the JWKS at url. httpc and now default to a
// timeout-bounded client and time.Now when nil.
func NewKeySet(url string, httpc Doer, now func() time.Time) *KeySet {
	if httpc == nil {
		httpc = defaultDoer()
	}
	if now == nil {
		now = time.Now
	}
	return &KeySet{url: url, httpc: httpc, now: now}
}

// Key resolves kid to an RSA public key, refetching the JWKS when the cache
// has expired or the kid is unknown (rate-limited so garbage kids cannot
// hammer the provider).
func (k *KeySet) Key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	for {
		k.mu.Lock()
		now := k.now()
		fresh := now.Before(k.expiresAt)
		if fresh {
			if key, ok := k.keys[kid]; ok {
				k.mu.Unlock()
				return key, nil
			}
			if now.Sub(k.fetchedAt) < unknownKidRefresh {
				k.mu.Unlock()
				return nil, fmt.Errorf("%w: %q", ErrUnknownKey, kid)
			}
		}
		if ch := k.inflight; ch != nil {
			k.mu.Unlock()
			select {
			case <-ch:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue // re-check the refreshed cache
		}
		ch := make(chan struct{})
		k.inflight = ch
		k.mu.Unlock()

		keys, ttl, err := k.fetch(ctx)

		k.mu.Lock()
		k.inflight = nil
		k.fetchedAt = k.now()
		if err == nil {
			k.keys = keys
			k.expiresAt = k.fetchedAt.Add(ttl)
		}
		k.mu.Unlock()
		close(ch)

		if err != nil {
			return nil, err
		}
		if key, ok := keys[kid]; ok {
			return key, nil
		}
		return nil, fmt.Errorf("%w: %q", ErrUnknownKey, kid)
	}
}

func (k *KeySet) fetch(ctx context.Context) (map[string]*rsa.PublicKey, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, k.url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("jwks request: %w", err)
	}
	resp, err := k.httpc.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("fetch jwks: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, 0, fmt.Errorf("read jwks: %w", err)
	}
	keys, err := parseRSAJWKS(body)
	if err != nil {
		return nil, 0, err
	}
	return keys, jwksTTL(resp.Header.Get("Cache-Control")), nil
}

// jwksTTL derives the cache lifetime from a Cache-Control header, clamped.
func jwksTTL(cacheControl string) time.Duration {
	ttl := defaultJWKSTTL
	for _, directive := range strings.Split(cacheControl, ",") {
		directive = strings.TrimSpace(directive)
		if v, ok := strings.CutPrefix(directive, "max-age="); ok {
			if secs, err := strconv.ParseInt(v, 10, 64); err == nil {
				ttl = time.Duration(secs) * time.Second
			}
		}
	}
	return min(max(ttl, minJWKSTTL), maxJWKSTTL)
}

// parseRSAJWKS extracts the RSA signing keys of a JWKS document, keyed by
// kid. Non-RSA entries and keys below 2048 bits are skipped, not fatal, so a
// provider adding an EC key cannot break verification.
func parseRSAJWKS(data []byte) (map[string]*rsa.PublicKey, error) {
	var doc struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			Use string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse jwks: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, jwk := range doc.Keys {
		if jwk.Kty != "RSA" || jwk.Kid == "" || (jwk.Use != "" && jwk.Use != "sig") {
			continue
		}
		nb, err := base64.RawURLEncoding.DecodeString(jwk.N)
		if err != nil {
			return nil, fmt.Errorf("parse jwks: bad modulus for kid %q", jwk.Kid)
		}
		eb, err := base64.RawURLEncoding.DecodeString(jwk.E)
		if err != nil || len(eb) == 0 || len(eb) > 4 {
			return nil, fmt.Errorf("parse jwks: bad exponent for kid %q", jwk.Kid)
		}
		e := 0
		for _, b := range eb {
			e = e<<8 | int(b)
		}
		if e < 3 {
			return nil, fmt.Errorf("parse jwks: bad exponent for kid %q", jwk.Kid)
		}
		n := new(big.Int).SetBytes(nb)
		if n.BitLen() < 2048 {
			continue
		}
		keys[jwk.Kid] = &rsa.PublicKey{N: n, E: e}
	}
	return keys, nil
}
