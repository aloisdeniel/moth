package analytics

import (
	"context"
	"database/sql"
	"testing"

	_ "time/tzdata"

	"github.com/aloisdeniel/moth/internal/store"
)

// rawSubMonth reads one subscription_monthly_stats row with the independent
// connection (shares none of the store's aggregation code).
type rawSubMonth struct {
	revenue                          int64
	active, newSubs, renewals        int
	churned, trialsStarted, trialsOK int
	appleRev, googleRev              int64
	found                            bool
}

func (e *testEnv) rawSubMonth(t *testing.T, projectID, period, currency string) rawSubMonth {
	t.Helper()
	var r rawSubMonth
	err := e.raw.QueryRow(
		`SELECT revenue_micros, active_subscribers, new_subscribers, renewals,
		        churned, trials_started, trials_converted,
		        store_apple_revenue_micros, store_google_revenue_micros
		   FROM subscription_monthly_stats
		  WHERE project_id = ? AND period = ? AND currency = ?`,
		projectID, period, currency,
	).Scan(&r.revenue, &r.active, &r.newSubs, &r.renewals,
		&r.churned, &r.trialsStarted, &r.trialsOK, &r.appleRev, &r.googleRev)
	if err == sql.ErrNoRows {
		return r
	}
	if err != nil {
		t.Fatal(err)
	}
	r.found = true
	return r
}

func subEvent(t *testing.T, projectID, typ, user, product, storeName, currency, env, at string, micros int64) store.SubscriptionEvent {
	t.Helper()
	return store.SubscriptionEvent{
		ID: newID(), ProjectID: projectID, Type: typ, UserID: user,
		ProductID: product, Store: storeName, PriceAmountMicros: micros,
		Currency: currency, Environment: env, CreatedAt: mustTime(t, at),
	}
}

// The subscription rollup buckets events into the project's local months
// (including a far-from-UTC boundary), sums revenue per currency, subtracts
// refunds from the month they land in, splits by store, and excludes sandbox.
func TestSubscriptionRollupCorrectness(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	// UTC+13: month January = [2025-12-31T11:00Z, 2026-01-31T11:00Z).
	p := e.createProject(t, "p1", "Pacific/Fakaofo", 0)
	const prod = store.SubscriptionEnvironmentProduction
	const sand = store.SubscriptionEnvironmentSandbox
	usd, apple, google := "USD", store.SubscriptionStoreApple, store.SubscriptionStoreGoogle

	evs := []store.SubscriptionEvent{
		// --- January (local) ---
		subEvent(t, p.ID, store.SubscriptionEventPurchased, "u1", "monthly", apple, usd, prod, "2026-01-05T00:00:00Z", 9_990_000),
		subEvent(t, p.ID, store.SubscriptionEventRenewed, "u2", "yearly", google, usd, prod, "2026-01-06T00:00:00Z", 79_990_000),
		subEvent(t, p.ID, store.SubscriptionEventRefunded, "u3", "monthly", apple, usd, prod, "2026-01-07T00:00:00Z", 9_990_000),
		subEvent(t, p.ID, store.SubscriptionEventTrialStarted, "u4", "monthly", apple, usd, prod, "2026-01-08T00:00:00Z", 0),
		subEvent(t, p.ID, store.SubscriptionEventConverted, "u4", "monthly", apple, usd, prod, "2026-01-09T00:00:00Z", 0),
		subEvent(t, p.ID, store.SubscriptionEventExpired, "u5", "monthly", apple, usd, prod, "2026-01-10T00:00:00Z", 0),
		// EUR row in the same month.
		subEvent(t, p.ID, store.SubscriptionEventPurchased, "u6", "monthly", google, "EUR", prod, "2026-01-11T00:00:00Z", 5_000_000),
		// Sandbox purchase — must be excluded from the production rollup.
		subEvent(t, p.ID, store.SubscriptionEventPurchased, "u7", "monthly", apple, usd, sand, "2026-01-12T00:00:00Z", 1_000_000),
		// Boundary: 2026-01-31T10:00Z is 2026-01-31 23:00 local → January.
		subEvent(t, p.ID, store.SubscriptionEventRenewed, "u8", "monthly", apple, usd, prod, "2026-01-31T10:00:00Z", 9_990_000),
		// Boundary: 2026-01-31T11:00Z is 2026-02-01 00:00 local → February.
		subEvent(t, p.ID, store.SubscriptionEventRenewed, "u9", "monthly", apple, usd, prod, "2026-01-31T11:00:00Z", 9_990_000),
	}
	if err := e.store.InsertSubscriptionEvents(ctx, evs); err != nil {
		t.Fatal(err)
	}

	if _, err := NewRollup(e.store, testLogger(), fixedNow(t, "2026-03-15T00:00:00Z")).Run(ctx, p.ID); err != nil {
		t.Fatal(err)
	}

	// January USD: purchased 9.99 + renewed(u2) 79.99 - refund 9.99 + renewed(u8) 9.99.
	jan := e.rawSubMonth(t, p.ID, "2026-01", usd)
	want := rawSubMonth{
		found:         true,
		revenue:       9_990_000 + 79_990_000 - 9_990_000 + 9_990_000,
		active:        4, // u1(purchased), u2(renewed), u4(trial), u8(renewed) — distinct
		newSubs:       1, // u1
		renewals:      2, // u2, u8
		churned:       1, // u5 expired
		trialsStarted: 1,
		trialsOK:      1,
		appleRev:      9_990_000,  // purchase 9.99 - refund 9.99 + u8 renew 9.99
		googleRev:     79_990_000, // u2 renew
	}
	if jan != want {
		t.Fatalf("Jan USD:\n got %+v\nwant %+v", jan, want)
	}

	// January EUR: a separate per-currency row, never blended into USD.
	janEUR := e.rawSubMonth(t, p.ID, "2026-01", "EUR")
	if !janEUR.found || janEUR.revenue != 5_000_000 || janEUR.newSubs != 1 || janEUR.googleRev != 5_000_000 {
		t.Fatalf("Jan EUR: got %+v", janEUR)
	}

	// February gets only the boundary renewal.
	feb := e.rawSubMonth(t, p.ID, "2026-02", usd)
	if !feb.found || feb.revenue != 9_990_000 || feb.renewals != 1 {
		t.Fatalf("Feb USD: got %+v", feb)
	}
}

// Re-running the subscription rollup replaces rows with identical values.
func TestSubscriptionRollupIdempotent(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	p := e.createProject(t, "p1", "UTC", 0)
	if err := e.store.InsertSubscriptionEvents(ctx, []store.SubscriptionEvent{
		subEvent(t, p.ID, store.SubscriptionEventPurchased, "u1", "monthly", store.SubscriptionStoreApple, "USD", store.SubscriptionEnvironmentProduction, "2026-06-05T10:00:00Z", 9_990_000),
		subEvent(t, p.ID, store.SubscriptionEventRenewed, "u1", "monthly", store.SubscriptionStoreApple, "USD", store.SubscriptionEnvironmentProduction, "2026-07-05T10:00:00Z", 9_990_000),
	}); err != nil {
		t.Fatal(err)
	}
	r := NewRollup(e.store, testLogger(), fixedNow(t, "2026-07-20T00:00:00Z"))
	if _, err := r.Run(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	dump := func() []store.SubscriptionStats {
		s, err := e.store.GetSubscriptionStats(ctx, p.ID, "2020-01", "2030-01")
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	before := dump()
	if _, err := r.Run(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	after := dump()
	if len(before) != len(after) {
		t.Fatalf("row count changed on re-run: %d -> %d", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("row %d changed on re-run:\n was %+v\n now %+v", i, before[i], after[i])
		}
	}
}

// The subscription rollup prunes raw subscription_events older than the
// project's retention window and keeps newer ones; the rolled-up rows persist.
func TestSubscriptionRollupRetention(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	p := e.createProject(t, "p1", "UTC", 30)
	const prod = store.SubscriptionEnvironmentProduction
	if err := e.store.InsertSubscriptionEvents(ctx, []store.SubscriptionEvent{
		// Older than 30 days before 2026-07-20 → pruned.
		subEvent(t, p.ID, store.SubscriptionEventPurchased, "u1", "monthly", store.SubscriptionStoreApple, "USD", prod, "2026-05-10T10:00:00Z", 9_990_000),
		// Within retention → kept.
		subEvent(t, p.ID, store.SubscriptionEventRenewed, "u1", "monthly", store.SubscriptionStoreApple, "USD", prod, "2026-07-10T10:00:00Z", 9_990_000),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewRollup(e.store, testLogger(), fixedNow(t, "2026-07-20T00:00:00Z")).Run(ctx, p.ID); err != nil {
		t.Fatal(err)
	}

	var remaining int
	if err := e.raw.QueryRow(`SELECT COUNT(*) FROM subscription_events WHERE project_id = ?`, p.ID).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 1 {
		t.Fatalf("retention: %d raw events left, want 1", remaining)
	}
	// May's rolled-up row survives even though its raw event was pruned.
	if may := e.rawSubMonth(t, p.ID, "2026-05", "USD"); !may.found || may.revenue != 9_990_000 {
		t.Fatalf("May row lost after pruning: %+v", may)
	}
}

// A project with no subscription events rolls up to no rows and no error.
func TestSubscriptionRollupEmptyProject(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	p := e.createProject(t, "p1", "UTC", 0)
	if _, err := NewRollup(e.store, testLogger(), fixedNow(t, "2026-07-20T00:00:00Z")).Run(ctx, ""); err != nil {
		t.Fatal(err)
	}
	rows, err := e.store.GetSubscriptionStats(ctx, p.ID, "2020-01", "2030-01")
	if err != nil {
		t.Fatalf("empty project errored: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("empty project produced %d rows, want 0", len(rows))
	}
}

// Seeded subscription data rolls up into per-currency revenue that matches
// independent SQL over the raw events, and the generator is deterministic.
func TestSeedSubscriptionsAndRollup(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	// Retention wider than the seeded window so the post-rollup raw cross-check
	// sees every event (pruning would otherwise remove already-aggregated ones).
	p := e.createProject(t, "p1", "UTC", 400)
	nowFn := fixedNow(t, "2026-07-01T12:00:00Z")

	n, err := SeedSubscriptions(ctx, e.store, p, SeedOptions{Days: 90, Seed: 7, Now: nowFn})
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("seed generated no subscription events")
	}
	if _, err := NewRollup(e.store, testLogger(), nowFn).Run(ctx, p.ID); err != nil {
		t.Fatal(err)
	}

	// Per (month,currency) net revenue from the rollup must equal independent
	// SQL over the production raw events (UTC project, so strftime months line
	// up exactly with the rollup windows).
	type key struct{ period, currency string }
	wantRev := map[key]int64{}
	rows, err := e.raw.Query(
		`SELECT strftime('%Y-%m', created_at), currency,
		        COALESCE(SUM(CASE
		          WHEN type IN ('subscription.purchased','subscription.renewed') THEN price_amount_micros
		          WHEN type = 'subscription.refunded' THEN -price_amount_micros
		          ELSE 0 END), 0)
		   FROM subscription_events
		  WHERE project_id = ? AND environment != 'sandbox'
		  GROUP BY 1, 2`, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var k key
		var v int64
		if err := rows.Scan(&k.period, &k.currency, &v); err != nil {
			t.Fatal(err)
		}
		wantRev[k] = v
	}
	rows.Close()

	got, err := e.store.GetSubscriptionStats(ctx, p.ID, "2020-01", "2030-01")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[key]bool{}
	for _, r := range got {
		k := key{r.Period, r.Currency}
		seen[k] = true
		if wantRev[k] != r.RevenueMicros {
			t.Fatalf("rollup revenue %s/%s = %d, raw events = %d", r.Period, r.Currency, r.RevenueMicros, wantRev[k])
		}
	}
	for k, v := range wantRev {
		if v != 0 && !seen[k] {
			t.Fatalf("rollup missing %s/%s (raw events = %d)", k.period, k.currency, v)
		}
	}
	if len(seen) == 0 {
		t.Fatal("seed produced no rolled-up revenue rows")
	}

	// Determinism: same seed, same event count and same digest.
	e2 := newEnv(t)
	p2 := e2.createProject(t, "p1", "UTC", 0)
	n2, err := SeedSubscriptions(ctx, e2.store, p2, SeedOptions{Days: 90, Seed: 7, Now: nowFn})
	if err != nil {
		t.Fatal(err)
	}
	if n2 != n {
		t.Fatalf("same seed produced %d then %d events", n, n2)
	}
	if d1, d2 := e.subSeedDigest(t, p.ID), e2.subSeedDigest(t, p2.ID); d1 != d2 {
		t.Fatalf("same seed produced different content:\n%s\n%s", d1, d2)
	}
}

func (e *testEnv) subSeedDigest(t *testing.T, projectID string) string {
	t.Helper()
	var digest string
	err := e.raw.QueryRow(
		`SELECT COALESCE(GROUP_CONCAT(row, '|'), '') FROM (
		   SELECT type || '/' || COALESCE(product_id,'') || '/' || store || '/' ||
		          currency || '/' || environment || '/' || COUNT(*) || '/' ||
		          COALESCE(SUM(price_amount_micros),0) AS row
		     FROM subscription_events WHERE project_id = ?
		    GROUP BY type, product_id, store, currency, environment
		    ORDER BY 1)`, projectID).Scan(&digest)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
