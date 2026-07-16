// Package analytics runs the milestone-07 aggregate-and-prune job: it
// rolls the raw event stream up into per-project daily_stats rows (bucketed
// in each project's rollup timezone) and prunes events older than the
// project's retention window. The same job runs on an in-process schedule
// and on demand through moth.admin.v1.AnalyticsService/RunRollup.
package analytics

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/aloisdeniel/moth/internal/store"
)

// dateFormat is the calendar-day key of daily_stats rows.
const dateFormat = "2006-01-02"

// maxBackfillDays bounds how far back a project's *first* rollup
// re-aggregates: at most its retention window (older days may already have
// had their raw events pruned, so re-rolling them would overwrite good rows
// with zeros), capped at the largest accepted retention. Once a project has
// rolled up at least one day the job always resumes from that day instead,
// however far back it is — its raw events are still intact because pruning
// never outruns aggregation (see rollupProject).
const maxBackfillDays = 366

// Store is everything the rollup job needs from persistence.
type Store interface {
	GetProject(ctx context.Context, id string) (store.Project, error)
	ListProjects(ctx context.Context) ([]store.Project, error)
	DeleteEventsBefore(ctx context.Context, projectID string, cutoff time.Time) (int64, error)
	store.StatsStore
}

// Rollup is the aggregate-and-prune job.
type Rollup struct {
	store Store
	log   *slog.Logger
	now   func() time.Time
}

// NewRollup builds the job. now is injectable for tests; nil means
// time.Now.
func NewRollup(st Store, log *slog.Logger, now func() time.Time) *Rollup {
	if log == nil {
		log = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	return &Rollup{store: st, log: log, now: now}
}

// Run rolls up one project (or every project when projectID is empty) and
// records the run in rollup_runs. Only completed local days are processed:
// for each project it re-aggregates every day from the newest already
// rolled-up day (re-rolling it is idempotent and catches events flushed
// across midnight by the async writer) through yesterday in the project's
// rollup timezone — a first rollup backfills at most the retention window.
// It then prunes raw events older than the project's retention window,
// never reaching into a day a later run still re-rolls. Failures on one
// project do not stop the others; the combined error is recorded on the
// run and returned.
func (r *Rollup) Run(ctx context.Context, projectID string) (store.RollupRun, error) {
	run := store.RollupRun{ID: newID(), StartedAt: r.now()}

	var projects []store.Project
	if projectID != "" {
		p, err := r.store.GetProject(ctx, projectID)
		if err != nil {
			return run, fmt.Errorf("rollup: get project: %w", err)
		}
		projects = []store.Project{p}
	} else {
		var err error
		if projects, err = r.store.ListProjects(ctx); err != nil {
			return run, fmt.Errorf("rollup: list projects: %w", err)
		}
	}

	var errs []string
	for _, p := range projects {
		days, pruned, err := r.rollupProject(ctx, p)
		run.DaysProcessed += days
		run.EventsPruned += pruned
		if err != nil {
			errs = append(errs, fmt.Sprintf("project %s: %v", p.Slug, err))
			r.log.ErrorContext(ctx, "analytics rollup failed for project",
				"project_id", p.ID, "error", err.Error())
		}
		if ctx.Err() != nil {
			errs = append(errs, ctx.Err().Error())
			break
		}
	}
	run.FinishedAt = r.now()
	run.Error = strings.Join(errs, "; ")

	if err := r.store.InsertRollupRun(ctx, run); err != nil {
		return run, fmt.Errorf("rollup: record run: %w", err)
	}
	if run.Error != "" {
		return run, fmt.Errorf("rollup: %s", run.Error)
	}
	return run, nil
}

// rollupProject aggregates one project's pending days and prunes its
// expired events. It returns how many day windows were (re-)aggregated and
// how many events were pruned.
func (r *Rollup) rollupProject(ctx context.Context, p store.Project) (int, int64, error) {
	loc := p.Settings.RollupLocation()
	localNow := r.now().In(loc)

	// Work on date-only values; the actual UTC instants of each day come
	// from DayWindow, which owns the timezone/DST arithmetic.
	latest := time.Date(localNow.Year(), localNow.Month(), localNow.Day()-1, 0, 0, 0, 0, time.UTC)
	backfill := p.Settings.AnalyticsRetentionDays
	if backfill <= 0 {
		backfill = store.DefaultProjectSettings().AnalyticsRetentionDays
	}
	if backfill > maxBackfillDays {
		backfill = maxBackfillDays
	}
	start := latest.AddDate(0, 0, -(backfill - 1))
	last, err := r.store.LatestDailyStatsDate(ctx, p.ID)
	if err != nil {
		return 0, 0, err
	}
	if last != "" {
		// Resume from the newest rolled-up day, wherever it is: later than
		// the backfill window on a routine hourly run, but possibly earlier
		// after an outage longer than the retention window — those gap days
		// still have their raw events (pruning below never outruns the
		// newest rolled-up day) and must be aggregated before they expire.
		if d, err := time.Parse(dateFormat, last); err == nil {
			start = d
		}
	}

	days := 0
	for day := start; !day.After(latest); day = day.AddDate(0, 0, 1) {
		if ctx.Err() != nil {
			return days, 0, ctx.Err()
		}
		date := day.Format(dateFormat)
		from, to, err := store.DayWindow(date, loc)
		if err != nil {
			return days, 0, err
		}
		ds, err := r.store.AggregateDailyStats(ctx, p.ID, date, from, to)
		if err != nil {
			return days, 0, err
		}
		if err := r.store.UpsertDailyStats(ctx, ds); err != nil {
			return days, 0, err
		}
		days++
	}

	retention := p.Settings.AnalyticsRetentionDays
	if retention <= 0 {
		retention = store.DefaultProjectSettings().AnalyticsRetentionDays
	}
	cutoff := r.now().AddDate(0, 0, -retention)
	// Never prune into the newest rolled-up day: the next run resumes from
	// it (re-rolling it to catch cross-midnight flushes), and a rolling
	// cutoff with a short retention — or a 25-hour DST day — would otherwise
	// eat that day's raw events run by run, decaying its daily_stats row.
	if from, _, err := store.DayWindow(latest.Format(dateFormat), loc); err == nil && cutoff.After(from) {
		cutoff = from
	}
	pruned, err := r.store.DeleteEventsBefore(ctx, p.ID, cutoff)
	if err != nil {
		return days, 0, err
	}
	return days, pruned, nil
}

// How often the scheduled job fires, and the random extra delay spread on
// top of it.
const (
	rollupInterval  = time.Hour
	rollupJitterMax = 5 * time.Minute
)

// RunPeriodically runs the whole-instance rollup on a jittered hourly
// ticker until ctx is done. Nightly-at-00:30-project-local scheduling would
// need one timer per timezone; instead the job simply runs every hour and
// only ever processes completed local days, so each project's finished day
// is rolled up within about an hour of its local midnight and intermediate
// runs just re-roll the newest day (idempotent, one small query). The first
// run happens after the initial jitter so a restarted instance catches up
// quickly. Failures are logged, never fatal.
func (r *Rollup) RunPeriodically(ctx context.Context) {
	delay := jitter()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		if _, err := r.Run(ctx, ""); err != nil && ctx.Err() == nil {
			r.log.ErrorContext(ctx, "scheduled analytics rollup", "error", err.Error())
		}
		delay = rollupInterval + jitter()
	}
}

func jitter() time.Duration {
	return time.Duration(rand.Int64N(int64(rollupJitterMax)))
}

// newID returns a UUIDv7 string (time-sortable primary keys).
func newID() string {
	id, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Sprintf("uuidv7: %v", err))
	}
	return id.String()
}
