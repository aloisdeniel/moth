// Package ratelimit is moth's shared, SQLite-backed rate limiter for the
// credential-facing surfaces. One Limiter is used from both the gRPC
// interceptor (SignIn, SignUp, ...) and the plain-HTTP middleware (OAuth
// redirects, hosted pages, the pub repository), so a request is throttled
// the same way whichever door it comes through.
//
// State lives in the rate_limits table (fixed epoch-aligned windows, one
// atomic UPSERT per hit), so limits survive restarts and are shared across
// every goroutine and — for a single-binary deployment — the whole process.
// The client IP is derived through an optional trusted-proxy set so a
// spoofed X-Forwarded-For can never dodge a per-IP bucket.
package ratelimit

import (
	"context"
	"time"

	"github.com/aloisdeniel/moth/internal/netutil"
	"github.com/aloisdeniel/moth/internal/store"
)

// Store is the persistence the limiter needs; *store.Store satisfies it.
type Store interface {
	TakeRateLimit(ctx context.Context, key string, n, limit int, window time.Duration, now time.Time) (store.RateLimitResult, error)
}

// Tier is one bucket policy: at most Limit hits per Window. A non-positive
// Limit or Window disables the tier (every request is allowed), which is how
// tests and the "off" configuration express "no limit".
type Tier struct {
	Limit  int
	Window time.Duration
}

func (t Tier) enabled() bool { return t.Limit > 0 && t.Window > 0 }

// Config groups the three tiers a credential-facing request is checked
// against: the client IP, the targeted account, and the whole project.
type Config struct {
	IP      Tier
	Account Tier
	Project Tier
}

// DefaultConfig returns the built-in limits (per-minute windows). The server
// resolves the effective config from config.Config, defaulting to these.
func DefaultConfig() Config {
	return Config{
		IP:      Tier{Limit: 60, Window: time.Minute},
		Account: Tier{Limit: 10, Window: time.Minute},
		Project: Tier{Limit: 600, Window: time.Minute},
	}
}

// Decision reports the outcome of a limiter check.
type Decision struct {
	// Allowed is false when the bucket is over its limit.
	Allowed bool
	// RetryAfter is how long until the window resets; meaningful only when
	// Allowed is false. It feeds the RPC RetryInfo detail and the HTTP
	// Retry-After header.
	RetryAfter time.Duration
}

// Limiter is safe for concurrent use; all state lives in the store.
type Limiter struct {
	store   Store
	cfg     Config
	proxies *netutil.TrustedProxies
	now     func() time.Time
}

// New builds a Limiter. proxies may be nil (trust no proxy — the safe
// default). now may be nil (defaults to time.Now).
func New(st Store, cfg Config, proxies *netutil.TrustedProxies, now func() time.Time) *Limiter {
	if now == nil {
		now = time.Now
	}
	return &Limiter{store: st, cfg: cfg, proxies: proxies, now: now}
}

// ClientIP resolves the caller's real IP from the connection peer
// (remoteAddr, "host:port" or bare host) and the X-Forwarded-For header,
// honouring xff only when the peer is a trusted proxy.
func (l *Limiter) ClientIP(remoteAddr, xff string) string {
	return l.proxies.ClientIP(remoteAddr, xff)
}

// IP throttles by client IP.
func (l *Limiter) IP(ctx context.Context, ip string) (Decision, error) {
	return l.take(ctx, store.RateTierIP, ip, l.cfg.IP)
}

// Account throttles by end-user account (project-scoped so the same email in
// two projects gets independent buckets).
func (l *Limiter) Account(ctx context.Context, projectID, email string) (Decision, error) {
	id := email
	if projectID != "" {
		id = projectID + "/" + email
	}
	return l.take(ctx, store.RateTierAccount, id, l.cfg.Account)
}

// Project throttles by project (a coarse ceiling protecting one tenant's
// whole credential surface).
func (l *Limiter) Project(ctx context.Context, projectID string) (Decision, error) {
	return l.take(ctx, store.RateTierProject, projectID, l.cfg.Project)
}

func (l *Limiter) take(ctx context.Context, tier, identifier string, t Tier) (Decision, error) {
	if !t.enabled() || identifier == "" {
		return Decision{Allowed: true}, nil
	}
	res, err := l.store.TakeRateLimit(ctx, store.RateLimitKey(tier, identifier), 1, t.Limit, t.Window, l.now())
	if err != nil {
		return Decision{}, err
	}
	return Decision{Allowed: res.Allowed, RetryAfter: res.RetryAfter}, nil
}
