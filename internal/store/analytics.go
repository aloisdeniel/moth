package store

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// DailyStats is one project's pre-aggregated counters for one calendar day
// — date is "YYYY-MM-DD" in the project's rollup timezone. The provider
// and platform counters break down that day's logins.
type DailyStats struct {
	ProjectID string
	Date      string
	Signups   int
	Logins    int
	// DAU counts distinct users with a login or token-refresh event that
	// day (documented approximation of "active").
	DAU      int
	Failures int
	// Logins by identity provider.
	LoginsPassword int
	LoginsGoogle   int
	LoginsApple    int
	// Logins by SDK-reported platform; PlatformOther collects everything
	// that is not ios/android/web, including logins with no platform.
	PlatformIOS     int
	PlatformAndroid int
	PlatformWeb     int
	PlatformOther   int
}

// RollupRun is one completed run of the aggregate-and-prune job, recorded
// for observability.
type RollupRun struct {
	ID            string
	StartedAt     time.Time
	FinishedAt    time.Time
	DaysProcessed int
	EventsPruned  int64
	// Error is empty when the run succeeded.
	Error string
}

// DayWindow returns the UTC instants [from, to) covering the calendar day
// date ("YYYY-MM-DD") in loc. Timezone conversion happens here, in Go —
// the aggregation queries only ever see UTC instants. time.Date normalizes
// local times that do not exist in loc (DST spring-forward), so windows on
// transition days are 23 or 25 hours long and consecutive days stay
// contiguous and non-overlapping.
func DayWindow(date string, loc *time.Location) (from, to time.Time, err error) {
	day, err := time.Parse("2006-01-02", date)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid date %q: %w", date, err)
	}
	y, m, d := day.Date()
	from = time.Date(y, m, d, 0, 0, 0, 0, loc).UTC()
	to = time.Date(y, m, d+1, 0, 0, 0, 0, loc).UTC()
	return from, to, nil
}

// timeBound formats a UTC instant for lexicographic comparison against
// stored RFC3339Nano timestamps. formatTime trims trailing fractional
// zeros but keeps the trailing "Z", so "…00:00:00.5Z" would compare
// *before* a whole-second boundary "…00:00:00Z" ('.' < 'Z') and a
// boundary-second event would land in the wrong day. Dropping the "Z"
// from the bound makes every half-open [from, to) comparison correct:
// timestamps in the second at `from` compare >= the trimmed bound, and
// timestamps at or after `to` compare >= its trimmed bound.
func timeBound(t time.Time) string {
	return strings.TrimSuffix(formatTime(t), "Z")
}

// AggregateDailyStats computes the daily_stats row for one (project,
// local-day) window from the raw events between the UTC instants
// [from, to) — callers derive them with DayWindow in the project's rollup
// timezone. The result is not persisted; pair with UpsertDailyStats.
func (s *Store) AggregateDailyStats(ctx context.Context, projectID, date string, from, to time.Time) (DailyStats, error) {
	ds := DailyStats{ProjectID: projectID, Date: date}

	rows, err := s.db.QueryContext(ctx,
		`SELECT type, provider, platform, COUNT(*)
		   FROM events
		  WHERE project_id = ? AND created_at >= ? AND created_at < ?
		  GROUP BY type, provider, platform`,
		projectID, timeBound(from), timeBound(to))
	if err != nil {
		return DailyStats{}, fmt.Errorf("aggregate daily stats: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var typ, provider, platform string
		var n int
		if err := rows.Scan(&typ, &provider, &platform, &n); err != nil {
			return DailyStats{}, fmt.Errorf("scan daily stats: %w", err)
		}
		switch typ {
		case EventUserSignup:
			ds.Signups += n
		case EventUserLoginFailed:
			ds.Failures += n
		case EventUserLogin:
			ds.Logins += n
			switch provider {
			case IdentityProviderPassword:
				ds.LoginsPassword += n
			case IdentityProviderGoogle:
				ds.LoginsGoogle += n
			case IdentityProviderApple:
				ds.LoginsApple += n
			}
			switch platform {
			case PlatformIOS:
				ds.PlatformIOS += n
			case PlatformAndroid:
				ds.PlatformAndroid += n
			case PlatformWeb:
				ds.PlatformWeb += n
			default:
				ds.PlatformOther += n
			}
		}
	}
	if err := rows.Err(); err != nil {
		return DailyStats{}, fmt.Errorf("aggregate daily stats: %w", err)
	}

	// DAU: distinct users seen logging in or refreshing a token.
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT user_id)
		   FROM events
		  WHERE project_id = ? AND created_at >= ? AND created_at < ?
		    AND type IN (?, ?) AND user_id IS NOT NULL`,
		projectID, timeBound(from), timeBound(to),
		EventUserLogin, EventTokenRefresh,
	).Scan(&ds.DAU); err != nil {
		return DailyStats{}, fmt.Errorf("aggregate dau: %w", err)
	}
	return ds, nil
}

// UpsertDailyStats installs the row for (project, date), replacing any
// previous rollup of the same day — re-running a day is idempotent.
func (s *Store) UpsertDailyStats(ctx context.Context, ds DailyStats) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO daily_stats (project_id, date, signups, logins, dau, failures,
		                          logins_password, logins_google, logins_apple,
		                          platform_ios, platform_android, platform_web, platform_other)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (project_id, date) DO UPDATE SET
		   signups = excluded.signups,
		   logins = excluded.logins,
		   dau = excluded.dau,
		   failures = excluded.failures,
		   logins_password = excluded.logins_password,
		   logins_google = excluded.logins_google,
		   logins_apple = excluded.logins_apple,
		   platform_ios = excluded.platform_ios,
		   platform_android = excluded.platform_android,
		   platform_web = excluded.platform_web,
		   platform_other = excluded.platform_other`,
		ds.ProjectID, ds.Date, ds.Signups, ds.Logins, ds.DAU, ds.Failures,
		ds.LoginsPassword, ds.LoginsGoogle, ds.LoginsApple,
		ds.PlatformIOS, ds.PlatformAndroid, ds.PlatformWeb, ds.PlatformOther)
	if err != nil {
		return fmt.Errorf("upsert daily stats: %w", err)
	}
	return nil
}

// GetDailyStats returns the project's rolled-up days in [fromDate, toDate]
// (inclusive, "YYYY-MM-DD"), oldest first. Days without a row are absent —
// callers zero-fill for chart rendering.
func (s *Store) GetDailyStats(ctx context.Context, projectID, fromDate, toDate string) ([]DailyStats, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT project_id, date, signups, logins, dau, failures,
		        logins_password, logins_google, logins_apple,
		        platform_ios, platform_android, platform_web, platform_other
		   FROM daily_stats
		  WHERE project_id = ? AND date >= ? AND date <= ?
		  ORDER BY date`,
		projectID, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("get daily stats: %w", err)
	}
	defer rows.Close()
	var stats []DailyStats
	for rows.Next() {
		var ds DailyStats
		if err := rows.Scan(&ds.ProjectID, &ds.Date, &ds.Signups, &ds.Logins,
			&ds.DAU, &ds.Failures,
			&ds.LoginsPassword, &ds.LoginsGoogle, &ds.LoginsApple,
			&ds.PlatformIOS, &ds.PlatformAndroid, &ds.PlatformWeb,
			&ds.PlatformOther); err != nil {
			return nil, fmt.Errorf("scan daily stats: %w", err)
		}
		stats = append(stats, ds)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get daily stats: %w", err)
	}
	return stats, nil
}

// LatestDailyStatsDate returns the newest rolled-up date ("YYYY-MM-DD") of
// a project, or "" when no day was rolled up yet. The rollup job resumes
// from it (re-rolling that day, which is idempotent) instead of re-scanning
// its whole backfill window.
func (s *Store) LatestDailyStatsDate(ctx context.Context, projectID string) (string, error) {
	var date string
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(date), '') FROM daily_stats WHERE project_id = ?`,
		projectID).Scan(&date)
	if err != nil {
		return "", fmt.Errorf("latest daily stats date: %w", err)
	}
	return date, nil
}

// InsertRollupRun records one completed rollup job run.
func (s *Store) InsertRollupRun(ctx context.Context, r RollupRun) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO rollup_runs (id, started_at, finished_at, days_processed, events_pruned, error)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, formatTime(r.StartedAt), formatTime(r.FinishedAt),
		r.DaysProcessed, r.EventsPruned, nullString(r.Error))
	if err != nil {
		return fmt.Errorf("insert rollup run: %w", err)
	}
	return nil
}

// ListRollupRuns returns the newest rollup runs, newest first.
func (s *Store) ListRollupRuns(ctx context.Context, limit int) ([]RollupRun, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, started_at, finished_at, days_processed, events_pruned, COALESCE(error, '')
		   FROM rollup_runs
		  ORDER BY started_at DESC, id DESC
		  LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list rollup runs: %w", err)
	}
	defer rows.Close()
	var runs []RollupRun
	for rows.Next() {
		var r RollupRun
		var startedAt, finishedAt string
		if err := rows.Scan(&r.ID, &startedAt, &finishedAt, &r.DaysProcessed,
			&r.EventsPruned, &r.Error); err != nil {
			return nil, fmt.Errorf("scan rollup run: %w", err)
		}
		if r.StartedAt, err = parseTime(startedAt); err != nil {
			return nil, fmt.Errorf("parse rollup run started_at: %w", err)
		}
		if r.FinishedAt, err = parseTime(finishedAt); err != nil {
			return nil, fmt.Errorf("parse rollup run finished_at: %w", err)
		}
		runs = append(runs, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list rollup runs: %w", err)
	}
	return runs, nil
}
