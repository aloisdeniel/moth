package ratelimit

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/aloisdeniel/moth/internal/netutil"
	"github.com/aloisdeniel/moth/internal/store"
)

func openStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "moth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return st
}

func fixedNow(base time.Time) func() time.Time { return func() time.Time { return base } }

// A limit of 3/window admits three hits and blocks the fourth.
func TestLimiterBucketsPerTier(t *testing.T) {
	st := openStore(t)
	base := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	cfg := Config{IP: Tier{Limit: 3, Window: time.Minute}}
	l := New(st, cfg, nil, fixedNow(base))
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		d, err := l.IP(ctx, "203.0.113.7")
		if err != nil {
			t.Fatal(err)
		}
		if !d.Allowed {
			t.Fatalf("hit %d within limit denied", i)
		}
	}
	d, err := l.IP(ctx, "203.0.113.7")
	if err != nil {
		t.Fatal(err)
	}
	if d.Allowed {
		t.Fatal("fourth hit over limit allowed")
	}
	if d.RetryAfter <= 0 {
		t.Fatal("blocked decision must carry a positive RetryAfter")
	}
}

// Two different IPs, accounts and the project tier keep independent buckets.
func TestLimiterTierIsolation(t *testing.T) {
	st := openStore(t)
	base := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	cfg := Config{
		IP:      Tier{Limit: 1, Window: time.Minute},
		Account: Tier{Limit: 1, Window: time.Minute},
		Project: Tier{Limit: 1, Window: time.Minute},
	}
	l := New(st, cfg, nil, fixedNow(base))
	ctx := context.Background()

	// Distinct IPs do not share a bucket.
	if d, _ := l.IP(ctx, "a"); !d.Allowed {
		t.Fatal("first IP denied")
	}
	if d, _ := l.IP(ctx, "a"); d.Allowed {
		t.Fatal("first IP not limited")
	}
	if d, _ := l.IP(ctx, "b"); !d.Allowed {
		t.Fatal("second IP affected by first")
	}

	// Same email in different projects is isolated.
	if d, _ := l.Account(ctx, "p1", "u@x"); !d.Allowed {
		t.Fatal("account p1 denied")
	}
	if d, _ := l.Account(ctx, "p1", "u@x"); d.Allowed {
		t.Fatal("account p1 not limited")
	}
	if d, _ := l.Account(ctx, "p2", "u@x"); !d.Allowed {
		t.Fatal("account p2 affected by p1")
	}

	// The project tier is its own bucket, unaffected by the IP hits above.
	if d, _ := l.Project(ctx, "p1"); !d.Allowed {
		t.Fatal("project p1 denied")
	}
	if d, _ := l.Project(ctx, "p1"); d.Allowed {
		t.Fatal("project p1 not limited")
	}
}

// A fresh Limiter over the same database still sees an exhausted bucket:
// state is persistent, so a restart does not reset the limit.
func TestLimiterPersistsAcrossRestart(t *testing.T) {
	st := openStore(t)
	base := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	cfg := Config{IP: Tier{Limit: 2, Window: time.Minute}}
	ctx := context.Background()

	before := New(st, cfg, nil, fixedNow(base))
	for i := 0; i < 2; i++ {
		if d, _ := before.IP(ctx, "203.0.113.9"); !d.Allowed {
			t.Fatalf("hit %d should pass before restart", i)
		}
	}

	// Simulate a restart: a brand-new Limiter instance, same store, same
	// window instant.
	after := New(st, cfg, nil, fixedNow(base))
	if d, _ := after.IP(ctx, "203.0.113.9"); d.Allowed {
		t.Fatal("bucket must stay exhausted across a restart")
	}
}

// A spoofed X-Forwarded-For from an untrusted peer is ignored, so the limit
// tracks the real peer; a trusted proxy's XFF is honoured.
func TestLimiterTrustedProxyIP(t *testing.T) {
	st := openStore(t)
	base := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	proxies, err := netutil.ParseTrustedProxies([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatal(err)
	}
	l := New(st, Config{IP: Tier{Limit: 1, Window: time.Minute}}, proxies, fixedNow(base))
	ctx := context.Background()

	// Untrusted peer 203.0.113.1 spoofs XFF: ignored, peer is the bucket key.
	if got := l.ClientIP("203.0.113.1:5555", "9.9.9.9"); got != "203.0.113.1" {
		t.Fatalf("spoofed XFF honoured: got %q", got)
	}
	// Trusted proxy 10.1.2.3 forwards a real client.
	if got := l.ClientIP("10.1.2.3:80", "198.51.100.7"); got != "198.51.100.7" {
		t.Fatalf("trusted XFF not honoured: got %q", got)
	}

	// End to end: two different real clients behind the trusted proxy do not
	// share a per-IP bucket even though the peer is identical.
	ipA := l.ClientIP("10.1.2.3:80", "198.51.100.7")
	ipB := l.ClientIP("10.1.2.3:80", "198.51.100.8")
	if d, _ := l.IP(ctx, ipA); !d.Allowed {
		t.Fatal("client A first hit denied")
	}
	if d, _ := l.IP(ctx, ipA); d.Allowed {
		t.Fatal("client A not limited")
	}
	if d, _ := l.IP(ctx, ipB); !d.Allowed {
		t.Fatal("client B throttled by client A behind same proxy")
	}
}
