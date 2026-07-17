package adminrpc

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/analytics"
	"github.com/aloisdeniel/moth/internal/store"
)

const (
	// statsDateFormat is the calendar-day key of the analytics API.
	statsDateFormat = "2006-01-02"
	// statsMonthFormat is the month key of the subscription analytics API.
	statsMonthFormat = "2006-01"
	// defaultStatsRangeMonths is the subscription series range when the
	// request leaves from_period/to_period empty.
	defaultStatsRangeMonths = 12
	// maxStatsRangeMonths bounds one GetSubscriptionStats range.
	maxStatsRangeMonths = 60
	// defaultStatsRangeDays is the series range when the request leaves
	// from_date/to_date empty.
	defaultStatsRangeDays = 30
	// maxStatsRangeDays bounds one GetStats/export range.
	maxStatsRangeDays = 366
	// Recent-events feed limits.
	defaultRecentEvents = 50
	maxRecentEvents     = 100
)

// AnalyticsHandler implements moth.admin.v1.AnalyticsService. All numbers
// except the live user count and the activity feed come from the
// pre-aggregated daily_stats rows — GetStats never scans raw events.
var _ adminv1connect.AnalyticsServiceHandler = (*AnalyticsHandler)(nil)

type AnalyticsHandler struct {
	store  Store
	rollup *analytics.Rollup
	now    func() time.Time
}

// NewAnalyticsHandler builds the analytics service. now is injectable for
// tests; nil means time.Now.
func NewAnalyticsHandler(st Store, rollup *analytics.Rollup, now func() time.Time) *AnalyticsHandler {
	if now == nil {
		now = time.Now
	}
	return &AnalyticsHandler{store: st, rollup: rollup, now: now}
}

func (h *AnalyticsHandler) GetStats(ctx context.Context, req *connect.Request[adminv1.GetStatsRequest]) (*connect.Response[adminv1.GetStatsResponse], error) {
	project, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	if g := req.Msg.Granularity; g != adminv1.Granularity_GRANULARITY_UNSPECIFIED &&
		g != adminv1.Granularity_GRANULARITY_DAY {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unsupported granularity %v", g))
	}

	// The newest completed local day; the default and 7-day windows anchor
	// on it.
	loc := project.Settings.RollupLocation()
	localNow := h.now().In(loc)
	latest := time.Date(localNow.Year(), localNow.Month(), localNow.Day()-1, 0, 0, 0, 0, time.UTC)

	from, to, err := statsRange(req.Msg.FromDate, req.Msg.ToDate, latest)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	rows, err := h.store.GetDailyStats(ctx, project.ID,
		from.Format(statsDateFormat), to.Format(statsDateFormat))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	byDate := make(map[string]store.DailyStats, len(rows))
	for _, r := range rows {
		byDate[r.Date] = r
	}

	resp := &adminv1.GetStatsResponse{
		Providers: &adminv1.ProviderBreakdown{},
		Platforms: &adminv1.PlatformBreakdown{},
	}
	// Contiguous, zero-filled series so a zero-traffic project renders an
	// empty chart, not a broken one.
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		r := byDate[day.Format(statsDateFormat)]
		resp.Series = append(resp.Series, &adminv1.DailyStat{
			Date:     day.Format(statsDateFormat),
			Signups:  int64(r.Signups),
			Logins:   int64(r.Logins),
			Dau:      int64(r.DAU),
			Failures: int64(r.Failures),
		})
		resp.Providers.Password += int64(r.LoginsPassword)
		resp.Providers.Google += int64(r.LoginsGoogle)
		resp.Providers.Apple += int64(r.LoginsApple)
		resp.Platforms.Ios += int64(r.PlatformIOS)
		resp.Platforms.Android += int64(r.PlatformAndroid)
		resp.Platforms.Web += int64(r.PlatformWeb)
		resp.Platforms.Other += int64(r.PlatformOther)
	}

	if resp.Tiles, err = h.tiles(ctx, project, latest); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// tiles computes the headline block. The 7-day windows cover the last 7
// completed local days ending at latest, regardless of the requested series
// range.
func (h *AnalyticsHandler) tiles(ctx context.Context, project store.Project, latest time.Time) (*adminv1.StatTiles, error) {
	tiles := &adminv1.StatTiles{}

	total, err := h.store.CountUsers(ctx, project.ID, "")
	if err != nil {
		return nil, err
	}
	tiles.TotalUsers = int64(total)

	// One query covers both 7-day windows.
	windowStart := latest.AddDate(0, 0, -13)
	currentStart := latest.AddDate(0, 0, -6).Format(statsDateFormat)
	rows, err := h.store.GetDailyStats(ctx, project.ID,
		windowStart.Format(statsDateFormat), latest.Format(statsDateFormat))
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.Date >= currentStart {
			tiles.NewUsers_7D += int64(r.Signups)
			tiles.Logins_7D += int64(r.Logins)
			tiles.LoginFailures_7D += int64(r.Failures)
		} else {
			tiles.NewUsersPrevious_7D += int64(r.Signups)
		}
	}
	if attempts := tiles.Logins_7D + tiles.LoginFailures_7D; attempts > 0 {
		tiles.LoginSuccessRate_7D = float64(tiles.Logins_7D) / float64(attempts)
	}

	// The DAU tile shows the most recent rolled-up day, whenever that was.
	date, err := h.store.LatestDailyStatsDate(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	if date != "" {
		day, err := h.store.GetDailyStats(ctx, project.ID, date, date)
		if err != nil {
			return nil, err
		}
		if len(day) == 1 {
			tiles.LatestDau = int64(day[0].DAU)
			tiles.LatestDauDate = date
		}
	}
	return tiles, nil
}

// statsRange resolves and validates the [from, to] day range, defaulting to
// the defaultStatsRangeDays days ending at latest.
func statsRange(fromDate, toDate string, latest time.Time) (from, to time.Time, err error) {
	to = latest
	if toDate != "" {
		if to, err = time.Parse(statsDateFormat, toDate); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to_date %q", toDate)
		}
	}
	from = to.AddDate(0, 0, -(defaultStatsRangeDays - 1))
	if fromDate != "" {
		if from, err = time.Parse(statsDateFormat, fromDate); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from_date %q", fromDate)
		}
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, errors.New("from_date is after to_date")
	}
	if to.Sub(from) > maxStatsRangeDays*24*time.Hour {
		return time.Time{}, time.Time{}, fmt.Errorf("range longer than %d days", maxStatsRangeDays)
	}
	return from, to, nil
}

func (h *AnalyticsHandler) ListRecentEvents(ctx context.Context, req *connect.Request[adminv1.ListRecentEventsRequest]) (*connect.Response[adminv1.ListRecentEventsResponse], error) {
	project, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = defaultRecentEvents
	}
	if limit > maxRecentEvents {
		limit = maxRecentEvents
	}
	rows, err := h.store.ListRecentEvents(ctx, project.ID, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &adminv1.ListRecentEventsResponse{}
	for _, e := range rows {
		// PII-minimal by construction: no metadata payload, no email.
		resp.Events = append(resp.Events, &adminv1.Event{
			Id:         e.ID,
			Type:       e.Type,
			UserId:     e.UserID,
			Provider:   e.Provider,
			Platform:   e.Platform,
			CreateTime: timestamppb.New(e.CreatedAt),
		})
	}
	return connect.NewResponse(resp), nil
}

// GetSubscriptionStats serves the subscription revenue dashboards from the
// pre-aggregated monthly rollup (subscription_monthly_stats /
// subscription_tier_stats) — it never scans raw subscription_events. Money is
// per currency, never blended.
func (h *AnalyticsHandler) GetSubscriptionStats(ctx context.Context, req *connect.Request[adminv1.GetSubscriptionStatsRequest]) (*connect.Response[adminv1.GetSubscriptionStatsResponse], error) {
	project, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	loc := project.Settings.RollupLocation()
	latestMonth := h.now().In(loc).Format(statsMonthFormat)

	from, to, err := statsMonthRange(req.Msg.FromPeriod, req.Msg.ToPeriod, latestMonth)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	rows, err := h.store.GetSubscriptionStats(ctx, project.ID, from, to)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	tierRows, err := h.store.GetSubscriptionTierStats(ctx, project.ID, from, to)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	activeRows, err := h.store.GetSubscriptionPeriodActive(ctx, project.ID, from, to)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	byPeriod := make(map[string][]store.SubscriptionStats)
	for _, r := range rows {
		byPeriod[r.Period] = append(byPeriod[r.Period], r)
	}

	// Currency-agnostic active-subscriber counts, keyed period -> product ("" =
	// month total). Active is a COUNT(DISTINCT user_id) and is NOT additive
	// across currencies, so it comes from these dedicated rows, never from
	// summing the per-currency SubscriptionStats.ActiveSubscribers. latestActive
	// is the newest month in range with data — the per-tier "active" card shows
	// that month's counts, matching the headline tile rather than summing every
	// month in the window.
	activeByPeriod := make(map[string]map[string]int)
	latestActive := ""
	for _, a := range activeRows {
		if activeByPeriod[a.Period] == nil {
			activeByPeriod[a.Period] = map[string]int{}
		}
		activeByPeriod[a.Period][a.ProductID] = a.ActiveSubscribers
		if a.Period > latestActive {
			latestActive = a.Period
		}
	}

	resp := &adminv1.GetSubscriptionStatsResponse{
		Stores: &adminv1.SubscriptionStoreBreakdown{},
	}
	// Contiguous, zero-filled monthly series.
	appleRev, googleRev, stripeRev := map[string]int64{}, map[string]int64{}, map[string]int64{}
	for m := from; m <= to; m = nextMonth(m) {
		stat := &adminv1.SubscriptionMonthlyStat{Period: m}
		rev := map[string]int64{}
		for _, r := range byPeriod[m] {
			rev[r.Currency] += r.RevenueMicros
			// NewSubscribers/Renewals/Churned/Trials are event counts, additive
			// across currencies. ActiveSubscribers is NOT — set below from the
			// currency-agnostic distinct count.
			stat.NewSubscribers += int64(r.NewSubscribers)
			stat.Renewals += int64(r.Renewals)
			stat.Churned += int64(r.Churned)
			stat.TrialsStarted += int64(r.TrialsStarted)
			stat.TrialsConverted += int64(r.TrialsConverted)
			appleRev[r.Currency] += r.StoreAppleRevenueMicros
			googleRev[r.Currency] += r.StoreGoogleRevenueMicros
			stripeRev[r.Currency] += r.StoreStripeRevenueMicros
		}
		stat.ActiveSubscribers = int64(activeByPeriod[m][""])
		stat.Revenue = currencyAmounts(rev)
		resp.Series = append(resp.Series, stat)
	}
	resp.Stores.Apple = currencyAmounts(appleRev)
	resp.Stores.Google = currencyAmounts(googleRev)
	resp.Stores.Stripe = currencyAmounts(stripeRev)

	// Per-tier breakdown over the whole range. Revenue and new-subscriber counts
	// are additive, so they sum across every month/currency in the range. Active
	// subscribers is a non-additive distinct count, so it is taken from the
	// latest month's currency-agnostic per-tier row (activeByPeriod[latest]) —
	// this both avoids inflating a distinct count by summing it across N months
	// (a user active all N months would otherwise count N times) and matches the
	// headline "Active subscribers" tile.
	type tierAgg struct {
		revenue        map[string]int64
		newSubscribers int64
	}
	tiers := map[string]*tierAgg{}
	var tierOrder []string
	for _, t := range tierRows {
		a, ok := tiers[t.ProductID]
		if !ok {
			a = &tierAgg{revenue: map[string]int64{}}
			tiers[t.ProductID] = a
			tierOrder = append(tierOrder, t.ProductID)
		}
		a.revenue[t.Currency] += t.RevenueMicros
		a.newSubscribers += int64(t.NewSubscribers)
	}
	for _, id := range tierOrder {
		a := tiers[id]
		resp.Tiers = append(resp.Tiers, &adminv1.SubscriptionTierBreakdown{
			ProductId:         id,
			Revenue:           currencyAmounts(a.revenue),
			NewSubscribers:    a.newSubscribers,
			ActiveSubscribers: int64(activeByPeriod[latestActive][id]),
		})
	}

	if resp.Tiles, err = h.subscriptionTiles(ctx, project.ID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// subscriptionTiles computes the headline block: the latest rolled-up month
// against the one before it.
func (h *AnalyticsHandler) subscriptionTiles(ctx context.Context, projectID string) (*adminv1.SubscriptionTiles, error) {
	tiles := &adminv1.SubscriptionTiles{}
	latest, err := h.store.LatestSubscriptionStatsPeriod(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if latest == "" {
		return tiles, nil
	}
	previous := prevMonth(latest)
	rows, err := h.store.GetSubscriptionStats(ctx, projectID, previous, latest)
	if err != nil {
		return nil, err
	}
	active, err := h.store.GetSubscriptionPeriodActive(ctx, projectID, previous, latest)
	if err != nil {
		return nil, err
	}
	tiles.LatestPeriod = latest
	thisRev, prevRev := map[string]int64{}, map[string]int64{}
	for _, r := range rows {
		switch r.Period {
		case latest:
			thisRev[r.Currency] += r.RevenueMicros
			// Event counts are additive across currencies; active is not (set
			// below from the currency-agnostic distinct count).
			tiles.NewSubscribers += int64(r.NewSubscribers)
			tiles.Churned += int64(r.Churned)
			tiles.TrialsStarted += int64(r.TrialsStarted)
			tiles.TrialsConverted += int64(r.TrialsConverted)
		case previous:
			prevRev[r.Currency] += r.RevenueMicros
		}
	}
	for _, a := range active {
		if a.ProductID != "" {
			continue // "" is the all-products month total
		}
		switch a.Period {
		case latest:
			tiles.ActiveSubscribers = int64(a.ActiveSubscribers)
		case previous:
			tiles.ActiveSubscribersPrevious = int64(a.ActiveSubscribers)
		}
	}
	tiles.RevenueThisMonth = currencyAmounts(thisRev)
	tiles.RevenuePreviousMonth = currencyAmounts(prevRev)
	if tiles.TrialsStarted > 0 {
		// Clamp to 1.0: trials_started and trials_converted are independent
		// single-month event counts (a trial started in month M-1 can convert in
		// M), so converted can exceed started in a low-volume or seasonally
		// shifting month. Report at most 100% rather than a nonsensical ">100%".
		rate := float64(tiles.TrialsConverted) / float64(tiles.TrialsStarted)
		if rate > 1 {
			rate = 1
		}
		tiles.TrialConversionRate = rate
	}
	return tiles, nil
}

// statsMonthRange resolves and validates the [from, to] month range,
// defaulting to defaultStatsRangeMonths months ending at latestMonth.
func statsMonthRange(fromPeriod, toPeriod, latestMonth string) (from, to string, err error) {
	to = latestMonth
	if toPeriod != "" {
		if _, err = time.Parse(statsMonthFormat, toPeriod); err != nil {
			return "", "", fmt.Errorf("invalid to_period %q", toPeriod)
		}
		to = toPeriod
	}
	toT, _ := time.Parse(statsMonthFormat, to)
	from = toT.AddDate(0, -(defaultStatsRangeMonths - 1), 0).Format(statsMonthFormat)
	if fromPeriod != "" {
		if _, err = time.Parse(statsMonthFormat, fromPeriod); err != nil {
			return "", "", fmt.Errorf("invalid from_period %q", fromPeriod)
		}
		from = fromPeriod
	}
	if from > to {
		return "", "", errors.New("from_period is after to_period")
	}
	fromT, _ := time.Parse(statsMonthFormat, from)
	if months := monthsBetween(fromT, toT); months >= maxStatsRangeMonths {
		return "", "", fmt.Errorf("range longer than %d months", maxStatsRangeMonths)
	}
	return from, to, nil
}

// currencyAmounts turns a currency→micros map into a stable, currency-sorted
// slice, dropping zero entries so a refunded-to-zero currency does not linger.
func currencyAmounts(m map[string]int64) []*adminv1.CurrencyAmount {
	out := make([]*adminv1.CurrencyAmount, 0, len(m))
	for cur, amt := range m {
		if amt == 0 {
			continue
		}
		out = append(out, &adminv1.CurrencyAmount{Currency: cur, AmountMicros: amt})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Currency < out[j].Currency })
	return out
}

func nextMonth(period string) string {
	t, _ := time.Parse(statsMonthFormat, period)
	return t.AddDate(0, 1, 0).Format(statsMonthFormat)
}

func prevMonth(period string) string {
	t, _ := time.Parse(statsMonthFormat, period)
	return t.AddDate(0, -1, 0).Format(statsMonthFormat)
}

func monthsBetween(from, to time.Time) int {
	return int(to.Year()-from.Year())*12 + int(to.Month()) - int(from.Month())
}

func (h *AnalyticsHandler) RunRollup(ctx context.Context, req *connect.Request[adminv1.RunRollupRequest]) (*connect.Response[adminv1.RunRollupResponse], error) {
	if req.Msg.ProjectId != "" {
		if _, err := h.store.GetProject(ctx, req.Msg.ProjectId); err != nil {
			return nil, projectErr(err)
		}
	}
	run, err := h.rollup.Run(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.RunRollupResponse{
		RunId:         run.ID,
		StartTime:     timestamppb.New(run.StartedAt),
		FinishTime:    timestamppb.New(run.FinishedAt),
		DaysProcessed: int32(run.DaysProcessed), //nolint:gosec // bounded by backfill window
		EventsPruned:  run.EventsPruned,
	}), nil
}
