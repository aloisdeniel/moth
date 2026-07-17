package store

import (
	"context"
	"testing"
	"time"

	_ "time/tzdata"
)

func TestMonthWindow(t *testing.T) {
	tests := []struct {
		name     string
		period   string
		tz       string
		from, to string
	}{
		{
			name: "utc", period: "2026-07", tz: "UTC",
			from: "2026-07-01T00:00:00Z", to: "2026-08-01T00:00:00Z",
		},
		{
			// Year boundary in UTC+13: the local month starts on the previous
			// UTC calendar day (and year).
			name: "utc+13 year boundary", period: "2026-01", tz: "Pacific/Fakaofo",
			from: "2025-12-31T11:00:00Z", to: "2026-01-31T11:00:00Z",
		},
		{
			// Paris: December spans a standard-time month; January starts at
			// 23:00 UTC the previous day.
			name: "paris winter", period: "2026-01", tz: "Europe/Paris",
			from: "2025-12-31T23:00:00Z", to: "2026-01-31T23:00:00Z",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			from, to, err := MonthWindow(tc.period, mustLoc(t, tc.tz))
			if err != nil {
				t.Fatal(err)
			}
			if !from.Equal(mustTime(t, tc.from)) || !to.Equal(mustTime(t, tc.to)) {
				t.Fatalf("window [%v, %v), want [%s, %s)", from, to, tc.from, tc.to)
			}
		})
	}
	if _, _, err := MonthWindow("2026-13", time.UTC); err == nil {
		t.Fatal("malformed period should error")
	}
}

// subEvent is a compact literal for the aggregation tests.
type subEvent struct {
	id, user, typ, product, storeName, currency, env string
	priceMicros                                      int64
	at                                               string // RFC3339Nano
}

func insertSubEvents(t *testing.T, s *Store, projectID string, evs []subEvent) {
	t.Helper()
	ctx := context.Background()
	for _, e := range evs {
		env := e.env
		if env == "" {
			env = SubscriptionEnvironmentProduction
		}
		if err := s.InsertSubscriptionEvent(ctx, SubscriptionEvent{
			ID: e.id, ProjectID: projectID, Type: e.typ, UserID: e.user, ProductID: e.product,
			Store: e.storeName, PriceAmountMicros: e.priceMicros, Currency: e.currency,
			Environment: env, CreatedAt: mustTime(t, e.at),
		}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAggregateSubscription(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p, k := testProject("p1", "app-one")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}

	const usd, eur = "USD", "EUR"
	events := []subEvent{
		// July, Paris timezone. A trial start (no revenue), a paid purchase, a
		// renewal, and a partial refund — all USD, apple.
		{"e1", "u1", SubscriptionEventTrialStarted, "prod_pro", SubscriptionStoreApple, usd, "", 0, "2026-07-02T10:00:00Z"},
		{"e2", "u1", SubscriptionEventPurchased, "prod_pro", SubscriptionStoreApple, usd, "", 9_990_000, "2026-07-05T10:00:00Z"},
		{"e3", "u2", SubscriptionEventRenewed, "prod_pro", SubscriptionStoreApple, usd, "", 9_990_000, "2026-07-10T10:00:00Z"},
		{"e4", "u3", SubscriptionEventRefunded, "prod_pro", SubscriptionStoreApple, usd, "", 9_990_000, "2026-07-12T10:00:00Z"},
		// A EUR google purchase for a different tier, plus a churn (expired).
		{"e5", "u4", SubscriptionEventPurchased, "prod_lite", SubscriptionStoreGoogle, eur, "", 4_990_000, "2026-07-15T10:00:00Z"},
		{"e6", "u5", SubscriptionEventExpired, "prod_lite", SubscriptionStoreGoogle, eur, "", 0, "2026-07-20T10:00:00Z"},
		// A sandbox purchase that must be excluded from production aggregates.
		{"e7", "u9", SubscriptionEventPurchased, "prod_pro", SubscriptionStoreApple, usd, SubscriptionEnvironmentSandbox, 9_990_000, "2026-07-06T10:00:00Z"},
		// Month-boundary: Paris July starts at 2026-06-30T22:00:00Z. This event
		// at 21:59Z is still June — must be excluded.
		{"e8", "u8", SubscriptionEventPurchased, "prod_pro", SubscriptionStoreApple, usd, "", 9_990_000, "2026-06-30T21:59:00Z"},
	}
	insertSubEvents(t, s, "p1", events)

	loc := mustLoc(t, "Europe/Paris")
	from, to, err := MonthWindow("2026-07", loc)
	if err != nil {
		t.Fatal(err)
	}
	stats, tiers, periodActive, err := s.AggregateSubscription(ctx, "p1", "2026-07", from, to, false)
	if err != nil {
		t.Fatal(err)
	}

	// Two currencies, sorted: EUR then USD.
	if len(stats) != 2 {
		t.Fatalf("want 2 currency rows, got %d: %+v", len(stats), stats)
	}
	eurRow, usdRow := stats[0], stats[1]
	if eurRow.Currency != eur || usdRow.Currency != usd {
		t.Fatalf("currency order: %q, %q", eurRow.Currency, usdRow.Currency)
	}
	// USD revenue = purchase + renewal - refund (sandbox excluded).
	if want := int64(9_990_000); usdRow.RevenueMicros != want {
		t.Errorf("USD revenue = %d, want %d", usdRow.RevenueMicros, want)
	}
	if usdRow.StoreAppleRevenueMicros != 9_990_000 || usdRow.StoreGoogleRevenueMicros != 0 {
		t.Errorf("USD store split: apple=%d google=%d", usdRow.StoreAppleRevenueMicros, usdRow.StoreGoogleRevenueMicros)
	}
	if usdRow.NewSubscribers != 1 || usdRow.Renewals != 1 || usdRow.TrialsStarted != 1 {
		t.Errorf("USD counts: new=%d renewals=%d trials=%d", usdRow.NewSubscribers, usdRow.Renewals, usdRow.TrialsStarted)
	}
	// u1 (trial+purchase) and u2 (renew) are the active USD users; sandbox u9 excluded.
	if usdRow.ActiveSubscribers != 2 {
		t.Errorf("USD active = %d, want 2", usdRow.ActiveSubscribers)
	}
	// EUR: one purchase, one expiry (churn).
	if eurRow.RevenueMicros != 4_990_000 || eurRow.NewSubscribers != 1 || eurRow.Churned != 1 {
		t.Errorf("EUR row: %+v", eurRow)
	}
	if eurRow.StoreGoogleRevenueMicros != 4_990_000 {
		t.Errorf("EUR google revenue = %d", eurRow.StoreGoogleRevenueMicros)
	}
	if eurRow.ActiveSubscribers != 1 {
		t.Errorf("EUR active = %d, want 1", eurRow.ActiveSubscribers)
	}

	// Per-tier: (EUR,prod_lite), (USD,prod_pro).
	if len(tiers) != 2 {
		t.Fatalf("want 2 tier rows, got %d: %+v", len(tiers), tiers)
	}
	if tiers[0].Currency != eur || tiers[0].ProductID != "prod_lite" || tiers[0].RevenueMicros != 4_990_000 {
		t.Errorf("tier[0] = %+v", tiers[0])
	}
	if tiers[1].Currency != usd || tiers[1].ProductID != "prod_pro" || tiers[1].RevenueMicros != 9_990_000 {
		t.Errorf("tier[1] = %+v", tiers[1])
	}
	if tiers[1].NewSubscribers != 1 || tiers[1].ActiveSubscribers != 2 {
		t.Errorf("tier[1] counts: new=%d active=%d", tiers[1].NewSubscribers, tiers[1].ActiveSubscribers)
	}

	// Currency-agnostic period-active: the all-products total "" is the distinct
	// count across every currency (u1, u2 in USD + u4 in EUR = 3 distinct;
	// sandbox u9 excluded), and each tier carries its own distinct count.
	active := map[string]int{}
	for _, a := range periodActive {
		active[a.ProductID] = a.ActiveSubscribers
	}
	if active[""] != 3 {
		t.Errorf("period active total = %d, want 3", active[""])
	}
	if active["prod_pro"] != 2 || active["prod_lite"] != 1 {
		t.Errorf("per-tier active: pro=%d lite=%d", active["prod_pro"], active["prod_lite"])
	}

	// includeSandbox=true folds the sandbox purchase back into USD.
	withSandbox, _, _, err := s.AggregateSubscription(ctx, "p1", "2026-07", from, to, true)
	if err != nil {
		t.Fatal(err)
	}
	if withSandbox[1].RevenueMicros != 19_980_000 {
		t.Errorf("with sandbox USD revenue = %d, want 19980000", withSandbox[1].RevenueMicros)
	}
}

// TestAggregateSubscriptionMultiCurrencyUser guards the non-additive
// active-subscriber fix: a single user transacting in two currencies in the
// same month is one distinct active subscriber in the currency-agnostic total,
// even though the per-currency rows each count them once.
func TestAggregateSubscriptionMultiCurrencyUser(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p, k := testProject("p1", "app-one")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	insertSubEvents(t, s, "p1", []subEvent{
		// One user, same tier, two currencies (a storefront/region change).
		{"a", "u1", SubscriptionEventPurchased, "prod_pro", SubscriptionStoreApple, "USD", "", 9_990_000, "2026-07-05T10:00:00Z"},
		{"b", "u1", SubscriptionEventRenewed, "prod_pro", SubscriptionStoreApple, "EUR", "", 8_990_000, "2026-07-20T10:00:00Z"},
	})
	from, to, _ := MonthWindow("2026-07", time.UTC)
	stats, _, periodActive, err := s.AggregateSubscription(ctx, "p1", "2026-07", from, to, false)
	if err != nil {
		t.Fatal(err)
	}
	// Two per-currency rows, each with active=1 (summing them would give 2).
	if len(stats) != 2 || stats[0].ActiveSubscribers != 1 || stats[1].ActiveSubscribers != 1 {
		t.Fatalf("per-currency active rows: %+v", stats)
	}
	total := 0
	for _, a := range periodActive {
		if a.ProductID == "" {
			total = a.ActiveSubscribers
		}
	}
	if total != 1 {
		t.Errorf("currency-agnostic active total = %d, want 1 (one distinct user)", total)
	}
}

func TestSubscriptionStatsRoundTripAndIdempotent(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p, k := testProject("p1", "app-one")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}

	stats := []SubscriptionStats{
		{ProjectID: "p1", Period: "2026-07", Currency: "USD", RevenueMicros: 9_990_000,
			ActiveSubscribers: 2, NewSubscribers: 1, Renewals: 1, TrialsStarted: 1,
			StoreAppleRevenueMicros: 9_990_000},
		{ProjectID: "p1", Period: "2026-07", Currency: "EUR", RevenueMicros: 4_990_000,
			ActiveSubscribers: 1, NewSubscribers: 1, StoreGoogleRevenueMicros: 4_990_000},
	}
	tiers := []SubscriptionTierStats{
		{ProjectID: "p1", Period: "2026-07", Currency: "USD", ProductID: "prod_pro",
			RevenueMicros: 9_990_000, NewSubscribers: 1, ActiveSubscribers: 2},
	}
	periodActive := []SubscriptionPeriodActive{
		{ProjectID: "p1", Period: "2026-07", ProductID: "", ActiveSubscribers: 3},
		{ProjectID: "p1", Period: "2026-07", ProductID: "prod_pro", ActiveSubscribers: 2},
	}
	if err := s.UpsertSubscriptionStats(ctx, "p1", "2026-07", stats, tiers, periodActive); err != nil {
		t.Fatal(err)
	}
	// Re-running the same month must be a no-op (idempotent): same row count.
	if err := s.UpsertSubscriptionStats(ctx, "p1", "2026-07", stats, tiers, periodActive); err != nil {
		t.Fatal(err)
	}
	gotActive, err := s.GetSubscriptionPeriodActive(ctx, "p1", "2026-01", "2026-12")
	if err != nil {
		t.Fatal(err)
	}
	if len(gotActive) != 2 || gotActive[0].ProductID != "" || gotActive[0].ActiveSubscribers != 3 {
		t.Fatalf("period-active round-trip: %+v", gotActive)
	}

	got, err := s.GetSubscriptionStats(ctx, "p1", "2026-01", "2026-12")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 rows after idempotent re-run, got %d", len(got))
	}
	// Ordered by period, currency: EUR before USD.
	if got[0].Currency != "EUR" || got[1].Currency != "USD" || got[1].RevenueMicros != 9_990_000 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}

	gotTiers, err := s.GetSubscriptionTierStats(ctx, "p1", "2026-01", "2026-12")
	if err != nil {
		t.Fatal(err)
	}
	if len(gotTiers) != 1 || gotTiers[0].ProductID != "prod_pro" || gotTiers[0].ActiveSubscribers != 2 {
		t.Fatalf("tier round-trip mismatch: %+v", gotTiers)
	}

	latest, err := s.LatestSubscriptionStatsPeriod(ctx, "p1")
	if err != nil || latest != "2026-07" {
		t.Fatalf("latest period = %q (%v)", latest, err)
	}

	// A currency dropping out of the fresh aggregate is removed, not left stale.
	if err := s.UpsertSubscriptionStats(ctx, "p1", "2026-07",
		stats[:1], nil, nil); err != nil { // only USD now
		t.Fatal(err)
	}
	got, _ = s.GetSubscriptionStats(ctx, "p1", "2026-01", "2026-12")
	if len(got) != 1 || got[0].Currency != "USD" {
		t.Fatalf("stale currency not cleared: %+v", got)
	}
	if gt, _ := s.GetSubscriptionTierStats(ctx, "p1", "2026-01", "2026-12"); len(gt) != 0 {
		t.Fatalf("tier rows not cleared: %+v", gt)
	}

	// Empty for a project with no rollup, and "" latest period.
	if latest, _ := s.LatestSubscriptionStatsPeriod(ctx, "missing"); latest != "" {
		t.Fatalf("missing project latest = %q", latest)
	}
}

func TestDeleteSubscriptionEventsBefore(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p, k := testProject("p1", "app-one")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	insertSubEvents(t, s, "p1", []subEvent{
		{"old", "u1", SubscriptionEventPurchased, "prod", SubscriptionStoreApple, "USD", "", 1, "2026-01-01T00:00:00Z"},
		{"new", "u2", SubscriptionEventPurchased, "prod", SubscriptionStoreApple, "USD", "", 1, "2026-07-01T00:00:00Z"},
	})
	n, err := s.DeleteSubscriptionEventsBefore(ctx, "p1", mustTime(t, "2026-06-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("pruned %d, want 1", n)
	}
	from, to, _ := MonthWindow("2026-07", time.UTC)
	stats, _, _, err := s.AggregateSubscription(ctx, "p1", "2026-07", from, to, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 1 || stats[0].NewSubscribers != 1 {
		t.Fatalf("survivor row wrong: %+v", stats)
	}
}
