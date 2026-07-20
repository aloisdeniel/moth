package store

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestTakeRateLimit(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	key := RateLimitKey(RateTierIP, "203.0.113.7")

	cases := []struct {
		name          string
		at            time.Time
		wantAllowed   bool
		wantRemaining int
	}{
		{"first hit", base, true, 2},
		{"second hit", base.Add(1 * time.Second), true, 1},
		{"third hit fills window", base.Add(2 * time.Second), true, 0},
		{"fourth hit over limit", base.Add(3 * time.Second), false, 0},
		{"still over inside window", base.Add(59 * time.Second), false, 0},
		{"new window resets", base.Add(60 * time.Second), true, 2},
		{"second in new window", base.Add(61 * time.Second), true, 1},
	}
	for _, tc := range cases {
		got, err := s.TakeRateLimit(ctx, key, 1, 3, time.Minute, tc.at)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got.Allowed != tc.wantAllowed {
			t.Errorf("%s: allowed = %v, want %v", tc.name, got.Allowed, tc.wantAllowed)
		}
		if got.Remaining != tc.wantRemaining {
			t.Errorf("%s: remaining = %d, want %d", tc.name, got.Remaining, tc.wantRemaining)
		}
		if !got.Allowed && got.RetryAfter <= 0 {
			t.Errorf("%s: blocked hit should report a positive RetryAfter, got %v", tc.name, got.RetryAfter)
		}
		if got.Allowed && got.RetryAfter != 0 {
			t.Errorf("%s: allowed hit should not report RetryAfter, got %v", tc.name, got.RetryAfter)
		}
	}
}

func TestTakeRateLimitRetryAfter(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	// now is 20s into an aligned minute window: reset is 40s away.
	now := time.Date(2026, 7, 16, 12, 0, 20, 0, time.UTC)
	key := RateLimitKey(RateTierAccount, "p1/user@example.com")

	if _, err := s.TakeRateLimit(ctx, key, 1, 1, time.Minute, now); err != nil {
		t.Fatal(err)
	}
	got, err := s.TakeRateLimit(ctx, key, 1, 1, time.Minute, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Allowed {
		t.Fatal("second hit should be blocked")
	}
	if got.RetryAfter != 40*time.Second {
		t.Fatalf("RetryAfter = %v, want 40s", got.RetryAfter)
	}
}

func TestTakeRateLimitN(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	key := RateLimitKey(RateTierProject, "p1")

	got, err := s.TakeRateLimit(ctx, key, 5, 10, time.Minute, now)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Allowed || got.Remaining != 5 {
		t.Fatalf("bulk take: allowed=%v remaining=%d, want true/5", got.Allowed, got.Remaining)
	}
	got, err = s.TakeRateLimit(ctx, key, 6, 10, time.Minute, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if got.Allowed {
		t.Fatalf("11 > 10 should be blocked, got %+v", got)
	}
}

func TestTakeRateLimitPersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/moth.db"
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	key := RateLimitKey(RateTierIP, "198.51.100.5")

	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := s.TakeRateLimit(ctx, key, 1, 3, time.Minute, now); err != nil {
			t.Fatal(err)
		}
	}
	s.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	// A restart must not reset the bucket: the 4th hit in the same window is
	// still blocked.
	got, err := s2.TakeRateLimit(ctx, key, 1, 3, time.Minute, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if got.Allowed {
		t.Fatal("rate-limit state should survive restart")
	}
}

func TestTakeRateLimitConcurrent(t *testing.T) {
	// Run with -race: TakeRateLimit must be atomic under concurrent callers.
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	key := RateLimitKey(RateTierIP, "192.0.2.1")
	const limit = 50
	const goroutines = 100

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := s.TakeRateLimit(ctx, key, 1, limit, time.Minute, now)
			if err != nil {
				t.Errorf("take: %v", err)
				return
			}
			if res.Allowed {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if allowed != limit {
		t.Fatalf("exactly %d of %d hits should be allowed, got %d", limit, goroutines, allowed)
	}
}

func TestDeleteStaleRateLimits(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)

	old := RateLimitKey(RateTierIP, "old")
	fresh := RateLimitKey(RateTierIP, "fresh")
	if _, err := s.TakeRateLimit(ctx, old, 1, 10, time.Minute, base); err != nil {
		t.Fatal(err)
	}
	if _, err := s.TakeRateLimit(ctx, fresh, 1, 10, time.Minute, base.Add(10*time.Minute)); err != nil {
		t.Fatal(err)
	}

	n, err := s.DeleteStaleRateLimits(ctx, base.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want 1 stale row deleted, got %d", n)
	}
	// The fresh bucket survives: a follow-up hit still counts from its state.
	got, err := s.TakeRateLimit(ctx, fresh, 1, 10, time.Minute, base.Add(10*time.Minute).Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if got.Remaining != 8 {
		t.Fatalf("fresh bucket should retain its count, remaining=%d want 8", got.Remaining)
	}
}
