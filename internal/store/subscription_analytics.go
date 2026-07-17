package store

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// SubscriptionStats is one project's pre-aggregated subscription counters for
// one month and one currency — period is "YYYY-MM" in the project's rollup
// timezone, currency is the store-reported ISO-4217 code ("" when the event
// carried none). Revenue is store-reported gross (net of refunds), never
// blended across currencies. A month can hold several rows, one per currency.
type SubscriptionStats struct {
	ProjectID string
	Period    string
	Currency  string
	// RevenueMicros is net gross: SUM(purchased+renewed) - SUM(refunded).
	RevenueMicros int64
	// ActiveSubscribers counts distinct users with an active-subscription
	// event this month (purchased, renewed or trial_started) — the "seen
	// active in the window" approximation, not a point-in-time stock.
	ActiveSubscribers int
	NewSubscribers    int // count(subscription.purchased)
	Renewals          int // count(subscription.renewed)
	Churned           int // count(expired + revoked)
	TrialsStarted     int // count(subscription.trial_started)
	TrialsConverted   int // count(subscription.converted)
	// Same net revenue split by store, for the per-store breakdown.
	StoreAppleRevenueMicros  int64
	StoreGoogleRevenueMicros int64
	StoreStripeRevenueMicros int64
}

// SubscriptionTierStats is the per-product slice of one month/currency rollup —
// ProductID is "" for events with no mapped moth product. ActiveSubscribers
// here is the per-currency distinct-user count and is NOT additive across
// currencies; the currency-agnostic per-tier count lives in
// SubscriptionPeriodActive.
type SubscriptionTierStats struct {
	ProjectID         string
	Period            string
	Currency          string
	ProductID         string
	RevenueMicros     int64
	NewSubscribers    int
	ActiveSubscribers int
}

// SubscriptionPeriodActive is a currency-agnostic distinct active-subscriber
// count for one (project, month, product). ProductID "" is the all-products
// month total (COUNT(DISTINCT user_id) over every active-type event that
// month, regardless of currency or tier) — the correct, non-additive figure
// for the headline tile and the monthly series. A non-empty ProductID is that
// tier's distinct count for the per-tier breakdown. Summing the per-currency
// SubscriptionStats.ActiveSubscribers would double-count a user who transacts
// in more than one currency; these rows are computed with a single
// COUNT(DISTINCT) across all currencies, so they never do.
type SubscriptionPeriodActive struct {
	ProjectID         string
	Period            string
	ProductID         string
	ActiveSubscribers int
}

// monthFormat is the period key of subscription rollup rows.
const monthFormat = "2006-01"

// subscriptionActiveTypes are the event types that mark a user as an active
// subscriber in a window (drives ActiveSubscribers and the "seen active"
// count). Kept in sync with the milestone-11 emitter in internal/store/billing.go.
var subscriptionActiveTypes = []string{
	SubscriptionEventPurchased,
	SubscriptionEventRenewed,
	SubscriptionEventTrialStarted,
}

// MonthWindow returns the UTC instants [from, to) covering the calendar month
// period ("YYYY-MM") in loc — the month analogue of DayWindow. Timezone
// conversion happens here, in Go, so aggregation only ever sees UTC instants.
func MonthWindow(period string, loc *time.Location) (from, to time.Time, err error) {
	m, err := time.Parse(monthFormat, period)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid period %q: %w", period, err)
	}
	y, mo, _ := m.Date()
	from = time.Date(y, mo, 1, 0, 0, 0, 0, loc).UTC()
	to = time.Date(y, mo+1, 1, 0, 0, 0, 0, loc).UTC()
	return from, to, nil
}

// AggregateSubscription computes the subscription rollup rows for one (project,
// month) window from the raw subscription_events between the UTC instants
// [from, to) — callers derive them with MonthWindow in the project's rollup
// timezone. It returns one SubscriptionStats per currency seen and the per-tier
// slices, both deterministically ordered. Sandbox events are excluded unless
// includeSandbox is set (production dashboards pass false). The result is not
// persisted; pair with UpsertSubscriptionStats.
func (s *Store) AggregateSubscription(ctx context.Context, projectID, period string, from, to time.Time, includeSandbox bool) ([]SubscriptionStats, []SubscriptionTierStats, []SubscriptionPeriodActive, error) {
	where := "project_id = ? AND created_at >= ? AND created_at < ?"
	args := []any{projectID, timeBound(from), timeBound(to)}
	if !includeSandbox {
		// Exclude sandbox; keep events with an unknown ('') environment so
		// production dashboards do not silently drop legacy rows.
		where += " AND environment != ?"
		args = append(args, SubscriptionEnvironmentSandbox)
	}

	byCurrency := map[string]*SubscriptionStats{}
	byTier := map[[2]string]*SubscriptionTierStats{}
	stat := func(cur string) *SubscriptionStats {
		st, ok := byCurrency[cur]
		if !ok {
			st = &SubscriptionStats{ProjectID: projectID, Period: period, Currency: cur}
			byCurrency[cur] = st
		}
		return st
	}
	tier := func(cur, product string) *SubscriptionTierStats {
		key := [2]string{cur, product}
		t, ok := byTier[key]
		if !ok {
			t = &SubscriptionTierStats{ProjectID: projectID, Period: period, Currency: cur, ProductID: product}
			byTier[key] = t
		}
		return t
	}

	// Counts + revenue, grouped by every dimension the rollup slices on.
	rows, err := s.db.QueryContext(ctx,
		`SELECT currency, store, COALESCE(product_id, ''), type,
		        COUNT(*), COALESCE(SUM(price_amount_micros), 0)
		   FROM subscription_events
		  WHERE `+where+`
		  GROUP BY currency, store, product_id, type`, args...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("aggregate subscription stats: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cur, storeName, product, typ string
		var n int
		var sum int64
		if err := rows.Scan(&cur, &storeName, &product, &typ, &n, &sum); err != nil {
			return nil, nil, nil, fmt.Errorf("scan subscription stats: %w", err)
		}
		st := stat(cur)
		tr := tier(cur, product)
		switch typ {
		case SubscriptionEventPurchased:
			st.NewSubscribers += n
			st.RevenueMicros += sum
			tr.NewSubscribers += n
			tr.RevenueMicros += sum
			addStoreRevenue(st, storeName, sum)
		case SubscriptionEventRenewed:
			st.Renewals += n
			st.RevenueMicros += sum
			tr.RevenueMicros += sum
			addStoreRevenue(st, storeName, sum)
		case SubscriptionEventRefunded:
			st.RevenueMicros -= sum
			tr.RevenueMicros -= sum
			addStoreRevenue(st, storeName, -sum)
		case SubscriptionEventExpired, SubscriptionEventRevoked:
			st.Churned += n
		case SubscriptionEventTrialStarted:
			st.TrialsStarted += n
		case SubscriptionEventConverted:
			st.TrialsConverted += n
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("aggregate subscription stats: %w", err)
	}

	// Distinct active subscribers per currency, then per currency+tier — the
	// COUNT(DISTINCT) cannot ride the grouped query above.
	activeArgs := append(append([]any{}, args...), toAny(subscriptionActiveTypes)...)
	inList := placeholders(len(subscriptionActiveTypes))
	curRows, err := s.db.QueryContext(ctx,
		`SELECT currency, COUNT(DISTINCT user_id)
		   FROM subscription_events
		  WHERE `+where+` AND user_id IS NOT NULL AND type IN (`+inList+`)
		  GROUP BY currency`, activeArgs...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("aggregate subscription active: %w", err)
	}
	defer curRows.Close()
	for curRows.Next() {
		var cur string
		var n int
		if err := curRows.Scan(&cur, &n); err != nil {
			return nil, nil, nil, fmt.Errorf("scan subscription active: %w", err)
		}
		stat(cur).ActiveSubscribers = n
	}
	if err := curRows.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("aggregate subscription active: %w", err)
	}

	tierRows, err := s.db.QueryContext(ctx,
		`SELECT currency, COALESCE(product_id, ''), COUNT(DISTINCT user_id)
		   FROM subscription_events
		  WHERE `+where+` AND user_id IS NOT NULL AND type IN (`+inList+`)
		  GROUP BY currency, product_id`, activeArgs...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("aggregate subscription tier active: %w", err)
	}
	defer tierRows.Close()
	for tierRows.Next() {
		var cur, product string
		var n int
		if err := tierRows.Scan(&cur, &product, &n); err != nil {
			return nil, nil, nil, fmt.Errorf("scan subscription tier active: %w", err)
		}
		tier(cur, product).ActiveSubscribers = n
	}
	if err := tierRows.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("aggregate subscription tier active: %w", err)
	}

	// Currency-agnostic distinct active subscribers: the all-products month
	// total (product_id '') plus one row per product. Each is a single
	// COUNT(DISTINCT user_id) across every currency, so a user transacting in
	// more than one currency is counted once — unlike summing the per-currency
	// SubscriptionStats.ActiveSubscribers rows, which double-counts them.
	byPeriodProduct := map[string]int{}
	total, err := s.countDistinctActive(ctx, where+` AND user_id IS NOT NULL AND type IN (`+inList+`)`, activeArgs)
	if err != nil {
		return nil, nil, nil, err
	}
	byPeriodProduct[""] = total
	prodRows, err := s.db.QueryContext(ctx,
		`SELECT COALESCE(product_id, ''), COUNT(DISTINCT user_id)
		   FROM subscription_events
		  WHERE `+where+` AND user_id IS NOT NULL AND type IN (`+inList+`)
		  GROUP BY product_id`, activeArgs...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("aggregate subscription period active: %w", err)
	}
	defer prodRows.Close()
	for prodRows.Next() {
		var product string
		var n int
		if err := prodRows.Scan(&product, &n); err != nil {
			return nil, nil, nil, fmt.Errorf("scan subscription period active: %w", err)
		}
		byPeriodProduct[product] = n
	}
	if err := prodRows.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("aggregate subscription period active: %w", err)
	}
	periodActive := make([]SubscriptionPeriodActive, 0, len(byPeriodProduct))
	for product, n := range byPeriodProduct {
		periodActive = append(periodActive, SubscriptionPeriodActive{
			ProjectID: projectID, Period: period, ProductID: product, ActiveSubscribers: n,
		})
	}
	sort.Slice(periodActive, func(i, j int) bool { return periodActive[i].ProductID < periodActive[j].ProductID })

	return sortStats(byCurrency), sortTiers(byTier), periodActive, nil
}

// countDistinctActive returns COUNT(DISTINCT user_id) for the given where
// clause + args (the active-subscriber query, all currencies folded together).
func (s *Store) countDistinctActive(ctx context.Context, where string, args []any) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM subscription_events WHERE `+where, args...).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("aggregate subscription period active: %w", err)
	}
	return n, nil
}

func addStoreRevenue(st *SubscriptionStats, storeName string, delta int64) {
	switch storeName {
	case SubscriptionStoreApple:
		st.StoreAppleRevenueMicros += delta
	case SubscriptionStoreGoogle:
		st.StoreGoogleRevenueMicros += delta
	case SubscriptionStoreStripe:
		st.StoreStripeRevenueMicros += delta
	}
}

func sortStats(m map[string]*SubscriptionStats) []SubscriptionStats {
	out := make([]SubscriptionStats, 0, len(m))
	for _, st := range m {
		out = append(out, *st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Currency < out[j].Currency })
	return out
}

func sortTiers(m map[[2]string]*SubscriptionTierStats) []SubscriptionTierStats {
	out := make([]SubscriptionTierStats, 0, len(m))
	for _, t := range m {
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Currency != out[j].Currency {
			return out[i].Currency < out[j].Currency
		}
		return out[i].ProductID < out[j].ProductID
	})
	return out
}

// UpsertSubscriptionStats replaces every rollup row for (project, period) with
// the given per-currency stats, per-tier slices and currency-agnostic
// period-active counts in one transaction — re-running a month is idempotent,
// and a currency, tier or product that dropped out of the fresh aggregate is
// removed rather than left stale.
func (s *Store) UpsertSubscriptionStats(ctx context.Context, projectID, period string, stats []SubscriptionStats, tiers []SubscriptionTierStats, periodActive []SubscriptionPeriodActive) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("upsert subscription stats: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM subscription_monthly_stats WHERE project_id = ? AND period = ?`,
		projectID, period); err != nil {
		return fmt.Errorf("clear subscription monthly stats: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM subscription_tier_stats WHERE project_id = ? AND period = ?`,
		projectID, period); err != nil {
		return fmt.Errorf("clear subscription tier stats: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM subscription_period_active WHERE project_id = ? AND period = ?`,
		projectID, period); err != nil {
		return fmt.Errorf("clear subscription period active: %w", err)
	}
	for _, st := range stats {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO subscription_monthly_stats (project_id, period, currency, revenue_micros,
			        active_subscribers, new_subscribers, renewals, churned, trials_started,
			        trials_converted, store_apple_revenue_micros, store_google_revenue_micros,
			        store_stripe_revenue_micros)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			projectID, period, st.Currency, st.RevenueMicros, st.ActiveSubscribers,
			st.NewSubscribers, st.Renewals, st.Churned, st.TrialsStarted, st.TrialsConverted,
			st.StoreAppleRevenueMicros, st.StoreGoogleRevenueMicros, st.StoreStripeRevenueMicros); err != nil {
			return fmt.Errorf("insert subscription monthly stats: %w", err)
		}
	}
	for _, t := range tiers {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO subscription_tier_stats (project_id, period, currency, product_id,
			        revenue_micros, new_subscribers, active_subscribers)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			projectID, period, t.Currency, t.ProductID, t.RevenueMicros,
			t.NewSubscribers, t.ActiveSubscribers); err != nil {
			return fmt.Errorf("insert subscription tier stats: %w", err)
		}
	}
	for _, a := range periodActive {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO subscription_period_active (project_id, period, product_id, active_subscribers)
			 VALUES (?, ?, ?, ?)`,
			projectID, period, a.ProductID, a.ActiveSubscribers); err != nil {
			return fmt.Errorf("insert subscription period active: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("upsert subscription stats: %w", err)
	}
	return nil
}

// GetSubscriptionPeriodActive returns the currency-agnostic distinct
// active-subscriber counts for the project's months in [fromPeriod, toPeriod]
// (inclusive), oldest first then by product_id. product_id "" is the
// all-products month total; a non-empty product_id is that tier's count. These
// are the correct figures for the headline tile, the monthly series and the
// per-tier breakdown — never the sum of the per-currency rows.
func (s *Store) GetSubscriptionPeriodActive(ctx context.Context, projectID, fromPeriod, toPeriod string) ([]SubscriptionPeriodActive, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT project_id, period, product_id, active_subscribers
		   FROM subscription_period_active
		  WHERE project_id = ? AND period >= ? AND period <= ?
		  ORDER BY period, product_id`, projectID, fromPeriod, toPeriod)
	if err != nil {
		return nil, fmt.Errorf("get subscription period active: %w", err)
	}
	defer rows.Close()
	var out []SubscriptionPeriodActive
	for rows.Next() {
		var a SubscriptionPeriodActive
		if err := rows.Scan(&a.ProjectID, &a.Period, &a.ProductID, &a.ActiveSubscribers); err != nil {
			return nil, fmt.Errorf("scan subscription period active: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get subscription period active: %w", err)
	}
	return out, nil
}

// GetSubscriptionStats returns the project's rolled-up months in
// [fromPeriod, toPeriod] (inclusive, "YYYY-MM"), oldest first then by currency.
// Months without a row are absent — callers zero-fill for chart rendering.
func (s *Store) GetSubscriptionStats(ctx context.Context, projectID, fromPeriod, toPeriod string) ([]SubscriptionStats, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT project_id, period, currency, revenue_micros, active_subscribers, new_subscribers,
		        renewals, churned, trials_started, trials_converted,
		        store_apple_revenue_micros, store_google_revenue_micros, store_stripe_revenue_micros
		   FROM subscription_monthly_stats
		  WHERE project_id = ? AND period >= ? AND period <= ?
		  ORDER BY period, currency`, projectID, fromPeriod, toPeriod)
	if err != nil {
		return nil, fmt.Errorf("get subscription stats: %w", err)
	}
	defer rows.Close()
	var out []SubscriptionStats
	for rows.Next() {
		var st SubscriptionStats
		if err := rows.Scan(&st.ProjectID, &st.Period, &st.Currency, &st.RevenueMicros,
			&st.ActiveSubscribers, &st.NewSubscribers, &st.Renewals, &st.Churned,
			&st.TrialsStarted, &st.TrialsConverted, &st.StoreAppleRevenueMicros,
			&st.StoreGoogleRevenueMicros, &st.StoreStripeRevenueMicros); err != nil {
			return nil, fmt.Errorf("scan subscription stats: %w", err)
		}
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get subscription stats: %w", err)
	}
	return out, nil
}

// GetSubscriptionTierStats returns the per-tier rollup rows for the project's
// months in [fromPeriod, toPeriod] (inclusive), oldest first then by
// (currency, product_id).
func (s *Store) GetSubscriptionTierStats(ctx context.Context, projectID, fromPeriod, toPeriod string) ([]SubscriptionTierStats, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT project_id, period, currency, product_id, revenue_micros,
		        new_subscribers, active_subscribers
		   FROM subscription_tier_stats
		  WHERE project_id = ? AND period >= ? AND period <= ?
		  ORDER BY period, currency, product_id`, projectID, fromPeriod, toPeriod)
	if err != nil {
		return nil, fmt.Errorf("get subscription tier stats: %w", err)
	}
	defer rows.Close()
	var out []SubscriptionTierStats
	for rows.Next() {
		var t SubscriptionTierStats
		if err := rows.Scan(&t.ProjectID, &t.Period, &t.Currency, &t.ProductID,
			&t.RevenueMicros, &t.NewSubscribers, &t.ActiveSubscribers); err != nil {
			return nil, fmt.Errorf("scan subscription tier stats: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get subscription tier stats: %w", err)
	}
	return out, nil
}

// LatestSubscriptionStatsPeriod returns the newest rolled-up month ("YYYY-MM")
// of a project, or "" when none. The rollup job resumes from it (re-rolling
// that month, which is idempotent) instead of re-scanning its whole backfill.
func (s *Store) LatestSubscriptionStatsPeriod(ctx context.Context, projectID string) (string, error) {
	var period string
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(period), '') FROM subscription_monthly_stats WHERE project_id = ?`,
		projectID).Scan(&period)
	if err != nil {
		return "", fmt.Errorf("latest subscription stats period: %w", err)
	}
	return period, nil
}

// DeleteSubscriptionEventsBefore prunes raw subscription events older than
// cutoff for one project (the retention sweep, mirroring DeleteEventsBefore)
// and reports how many were removed.
func (s *Store) DeleteSubscriptionEventsBefore(ctx context.Context, projectID string, cutoff time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM subscription_events WHERE project_id = ? AND created_at < ?`,
		projectID, formatTime(cutoff))
	if err != nil {
		return 0, fmt.Errorf("delete subscription events: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete subscription events: %w", err)
	}
	return n, nil
}

// placeholders returns "?, ?, …" with n placeholders for an IN clause.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, 0, n*3)
	for i := 0; i < n; i++ {
		if i > 0 {
			b = append(b, ',', ' ')
		}
		b = append(b, '?')
	}
	return string(b)
}

func toAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}
