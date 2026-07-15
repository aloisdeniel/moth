// Package ratelimit is an in-memory token-bucket limiter for the
// credential-facing auth RPCs. Full hardening (persistence, lockouts)
// arrives in milestone 10.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter tracks one token bucket per key (an IP, an account, ...).
type Limiter struct {
	rate  float64 // tokens per second
	burst float64

	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

// maxBuckets caps memory; when exceeded, replenished buckets are swept.
const maxBuckets = 100_000

// New returns a limiter allowing perMinute sustained requests per key with
// the given burst.
func New(perMinute float64, burst int) *Limiter {
	return &Limiter{
		rate:    perMinute / 60,
		burst:   float64(burst),
		buckets: make(map[string]*bucket),
	}
}

// Allow reports whether one request for key may proceed now.
func (l *Limiter) Allow(key string) bool {
	return l.allow(key, time.Now())
}

func (l *Limiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= maxBuckets {
			l.sweep(now)
		}
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}

	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens = min(l.burst, b.tokens+elapsed*l.rate)
		b.last = now
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// sweep drops buckets that have fully replenished; callers hold l.mu.
func (l *Limiter) sweep(now time.Time) {
	for key, b := range l.buckets {
		if b.tokens+now.Sub(b.last).Seconds()*l.rate >= l.burst {
			delete(l.buckets, key)
		}
	}
}
