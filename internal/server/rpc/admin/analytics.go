package adminrpc

import (
	"context"
	"errors"
	"fmt"
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
