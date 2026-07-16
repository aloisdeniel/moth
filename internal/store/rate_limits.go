package store

import (
	"context"
	"fmt"
	"time"
)

// Rate-limit tier prefixes. A bucket key is the tier joined to the
// identifier it limits (e.g. "ip:203.0.113.7", "account:proj/user@x",
// "project:01J..."), keeping the three tiers in one table without
// collisions.
const (
	// RateTierIP limits by client IP address.
	RateTierIP = "ip"
	// RateTierAccount limits by end-user account (credential-facing RPCs).
	RateTierAccount = "account"
	// RateTierProject limits by project.
	RateTierProject = "project"
)

// RateLimitKey builds a bucket key from a tier and identifier.
func RateLimitKey(tier, identifier string) string {
	return tier + ":" + identifier
}

// RateLimitResult reports the outcome of a TakeRateLimit call.
type RateLimitResult struct {
	// Allowed is true when the request stays within the limit.
	Allowed bool
	// Remaining is how many hits are left in the current window (>= 0).
	Remaining int
	// RetryAfter is how long until the window resets; only meaningful when
	// Allowed is false.
	RetryAfter time.Duration
}

// TakeRateLimit atomically records n hits against a fixed window and reports
// whether the bucket stays within limit. Windows are aligned to the epoch
// (now truncated to window), so a single UPSERT decides, in the database,
// whether this hit falls in the same window as the stored count (increment)
// or a fresh one (reset) — no read-modify-write race even under concurrent
// callers, because SQLite serializes the statement. It survives restarts
// because the state is a row, not an in-memory bucket.
func (s *Store) TakeRateLimit(ctx context.Context, key string, n, limit int, window time.Duration, now time.Time) (RateLimitResult, error) {
	if window <= 0 {
		return RateLimitResult{}, fmt.Errorf("rate limit window must be positive")
	}
	bucket := now.UTC().Truncate(window)
	bucketStr := formatTime(bucket)

	var count int
	var storedStart string
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO rate_limits (key, count, window_start)
		 VALUES (?, ?, ?)
		 ON CONFLICT (key) DO UPDATE SET
		     count = CASE WHEN rate_limits.window_start = excluded.window_start
		                  THEN rate_limits.count + excluded.count
		                  ELSE excluded.count END,
		     window_start = excluded.window_start
		 RETURNING count, window_start`,
		key, n, bucketStr).Scan(&count, &storedStart)
	if err != nil {
		return RateLimitResult{}, fmt.Errorf("take rate limit: %w", err)
	}

	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}
	res := RateLimitResult{Allowed: count <= limit, Remaining: remaining}
	if !res.Allowed {
		reset := bucket.Add(window)
		if ra := reset.Sub(now); ra > 0 {
			res.RetryAfter = ra
		}
	}
	return res, nil
}

// DeleteStaleRateLimits removes buckets whose window ended before cutoff,
// keeping the table from growing without bound. Callers pass a cutoff a few
// windows in the past so live buckets are never dropped.
func (s *Store) DeleteStaleRateLimits(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM rate_limits WHERE window_start < ?`, formatTime(cutoff.UTC()))
	if err != nil {
		return 0, fmt.Errorf("delete stale rate limits: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete stale rate limits: %w", err)
	}
	return n, nil
}
