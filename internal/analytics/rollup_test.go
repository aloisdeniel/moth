package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	// The DST/UTC+13 cases need IANA zones even on hosts without a system
	// zoneinfo directory.
	_ "time/tzdata"

	_ "modernc.org/sqlite" // raw cross-check connection

	"github.com/aloisdeniel/moth/internal/store"
)

// testEnv is one store plus a second, independent SQL connection to the
// same database file, used to cross-check rollup results with raw queries
// that share none of the store's code.
type testEnv struct {
	store *store.Store
	raw   *sql.DB
}

func newEnv(t *testing.T) *testEnv {
	t.Helper()
	path := filepath.Join(t.TempDir(), "moth.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { raw.Close() })
	return &testEnv{store: st, raw: raw}
}

func (e *testEnv) createProject(t *testing.T, id, tz string, retentionDays int) store.Project {
	t.Helper()
	now := time.Now()
	settings := store.DefaultProjectSettings()
	settings.RollupTimezone = tz
	if retentionDays > 0 {
		settings.AnalyticsRetentionDays = retentionDays
	}
	p := store.Project{
		ID: id, Name: "App " + id, Slug: "app-" + id,
		PublishableKey: "pk_" + id, SecretKeyHash: "hash-" + id,
		Settings:  settings,
		CreatedAt: now, UpdatedAt: now,
	}
	k := store.ProjectKey{
		ID: "key-" + id, ProjectID: id, Kid: "kid-" + id, Algorithm: "ES256",
		PublicKeyPEM: "PEM", PrivateKeyEnc: []byte{1},
		Status: store.ProjectKeyStatusActive, CreatedAt: now,
	}
	if err := e.store.CreateProject(context.Background(), p, k); err != nil {
		t.Fatal(err)
	}
	return p
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

func fixedNow(t *testing.T, s string) func() time.Time {
	now := mustTime(t, s)
	return func() time.Time { return now }
}

// rawStats reads one daily_stats row with the independent connection.
type rawStats struct {
	signups, logins, dau, failures int
	password, google, apple        int
	ios, android, web, other       int
	found                          bool
}

func (e *testEnv) rawDailyStats(t *testing.T, projectID, date string) rawStats {
	t.Helper()
	var r rawStats
	err := e.raw.QueryRow(
		`SELECT signups, logins, dau, failures,
		        logins_password, logins_google, logins_apple,
		        platform_ios, platform_android, platform_web, platform_other
		   FROM daily_stats WHERE project_id = ? AND date = ?`, projectID, date,
	).Scan(&r.signups, &r.logins, &r.dau, &r.failures,
		&r.password, &r.google, &r.apple,
		&r.ios, &r.android, &r.web, &r.other)
	if err == sql.ErrNoRows {
		return r
	}
	if err != nil {
		t.Fatal(err)
	}
	r.found = true
	return r
}

func testLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

// The rollup buckets events into the project's local days, including on
// DST-transition and far-from-UTC days, and completed days only.
func TestRollupCorrectnessAcrossTimezones(t *testing.T) {
	type ev struct {
		user, typ, provider, platform, at string
	}
	tests := []struct {
		name string
		tz   string
		now  string // instant Run executes at
		evs  []ev
		date string
		want rawStats
	}{
		{
			name: "utc",
			tz:   "UTC",
			now:  "2026-07-03T02:00:00Z",
			evs: []ev{
				{"u1", store.EventUserSignup, store.IdentityProviderPassword, store.PlatformIOS, "2026-07-01T00:00:00Z"},
				{"u1", store.EventUserLogin, store.IdentityProviderPassword, store.PlatformIOS, "2026-07-01T10:00:00Z"},
				{"u2", store.EventUserLogin, store.IdentityProviderGoogle, store.PlatformAndroid, "2026-07-01T23:59:59Z"},
				{"u2", store.EventTokenRefresh, "", store.PlatformAndroid, "2026-07-01T12:00:00Z"},
				{"u3", store.EventTokenRefresh, "", "", "2026-07-01T13:00:00Z"},
				{"", store.EventUserLoginFailed, store.IdentityProviderPassword, "", "2026-07-01T14:00:00Z"},
				{"u9", store.EventUserLogin, store.IdentityProviderApple, store.PlatformWeb, "2026-07-02T00:00:00Z"}, // next day
			},
			date: "2026-07-01",
			want: rawStats{found: true, signups: 1, logins: 2, dau: 3, failures: 1,
				password: 1, google: 1, ios: 1, android: 1},
		},
		{
			// Los Angeles springs forward on 2026-03-08: the local day is
			// [2026-03-08T08:00Z, 2026-03-09T07:00Z), 23 hours long.
			name: "dst spring forward",
			tz:   "America/Los_Angeles",
			now:  "2026-03-10T12:00:00Z",
			evs: []ev{
				{"u1", store.EventUserLogin, store.IdentityProviderPassword, store.PlatformIOS, "2026-03-08T08:00:00Z"},
				{"u2", store.EventUserLogin, store.IdentityProviderApple, store.PlatformWeb, "2026-03-09T06:59:59Z"},
				{"u8", store.EventUserLogin, store.IdentityProviderPassword, store.PlatformIOS, "2026-03-08T07:59:59Z"}, // 03-07
				{"u9", store.EventUserLogin, store.IdentityProviderPassword, store.PlatformIOS, "2026-03-09T07:00:00Z"}, // 03-09
			},
			date: "2026-03-08",
			want: rawStats{found: true, logins: 2, dau: 2, password: 1, apple: 1, ios: 1, web: 1},
		},
		{
			// UTC+13: the local day starts on the previous UTC calendar day.
			name: "utc+13",
			tz:   "Pacific/Fakaofo",
			now:  "2026-01-03T00:00:00Z",
			evs: []ev{
				{"u1", store.EventUserLogin, store.IdentityProviderGoogle, store.PlatformAndroid, "2025-12-31T11:00:00Z"},
				{"u2", store.EventUserLogin, store.IdentityProviderGoogle, "tvos", "2026-01-01T10:59:59Z"},
				{"u9", store.EventUserLogin, store.IdentityProviderGoogle, store.PlatformAndroid, "2025-12-31T10:59:59Z"},
				{"u9", store.EventUserLogin, store.IdentityProviderGoogle, store.PlatformAndroid, "2026-01-01T11:00:00Z"},
			},
			date: "2026-01-01",
			want: rawStats{found: true, logins: 2, dau: 2, google: 2, android: 1, other: 1},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t)
			ctx := context.Background()
			p := e.createProject(t, "p1", tc.tz, 0)
			var batch []store.Event
			for _, ev := range tc.evs {
				batch = append(batch, store.Event{
					ID: newID(), ProjectID: p.ID, UserID: ev.user, Type: ev.typ,
					Provider: ev.provider, Platform: ev.platform,
					CreatedAt: mustTime(t, ev.at),
				})
			}
			if err := e.store.InsertEvents(ctx, batch); err != nil {
				t.Fatal(err)
			}

			r := NewRollup(e.store, testLogger(), fixedNow(t, tc.now))
			run, err := r.Run(ctx, p.ID)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if run.DaysProcessed == 0 {
				t.Fatal("no days processed")
			}
			got := e.rawDailyStats(t, p.ID, tc.date)
			if got != tc.want {
				t.Fatalf("daily_stats[%s]:\n got %+v\nwant %+v", tc.date, got, tc.want)
			}
		})
	}
}

// Re-running the rollup replaces rows with identical values, and resumes
// from the newest rolled-up day instead of re-scanning the whole window.
func TestRollupIdempotentAndIncremental(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	p := e.createProject(t, "p1", "UTC", 0)
	if err := e.store.InsertEvents(ctx, []store.Event{
		{ID: newID(), ProjectID: p.ID, UserID: "u1", Type: store.EventUserLogin,
			Provider: store.IdentityProviderPassword, Platform: store.PlatformIOS,
			CreatedAt: mustTime(t, "2026-07-01T10:00:00Z")},
		{ID: newID(), ProjectID: p.ID, UserID: "u2", Type: store.EventUserSignup,
			Provider: store.IdentityProviderGoogle, CreatedAt: mustTime(t, "2026-07-02T10:00:00Z")},
	}); err != nil {
		t.Fatal(err)
	}

	r := NewRollup(e.store, testLogger(), fixedNow(t, "2026-07-03T01:00:00Z"))
	run1, err := r.Run(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}

	dump := func() []store.DailyStats {
		rows, err := e.store.GetDailyStats(ctx, p.ID, "2020-01-01", "2030-01-01")
		if err != nil {
			t.Fatal(err)
		}
		return rows
	}
	before := dump()
	run2, err := r.Run(ctx, p.ID)
	if err != nil {
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
	// The second run only re-rolls the newest already-rolled day.
	if run2.DaysProcessed != 1 {
		t.Fatalf("incremental run processed %d days, want 1", run2.DaysProcessed)
	}
	if run1.DaysProcessed <= run2.DaysProcessed {
		t.Fatalf("first run should have backfilled more than %d days", run1.DaysProcessed)
	}

	// The point of re-rolling the newest day: an event the async writer
	// flushed after the day was first rolled up (e.g. across midnight) is
	// picked up by the next incremental run.
	if err := e.store.InsertEvents(ctx, []store.Event{
		{ID: newID(), ProjectID: p.ID, UserID: "u3", Type: store.EventUserLogin,
			Provider: store.IdentityProviderPassword, Platform: store.PlatformIOS,
			CreatedAt: mustTime(t, "2026-07-02T23:59:30Z")},
	}); err != nil {
		t.Fatal(err)
	}
	run3, err := r.Run(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if run3.DaysProcessed != 1 {
		t.Fatalf("late-event run processed %d days, want 1", run3.DaysProcessed)
	}
	got := e.rawDailyStats(t, p.ID, "2026-07-02")
	want := rawStats{found: true, signups: 1, logins: 1, dau: 1, password: 1, ios: 1}
	if got != want {
		t.Fatalf("late event not re-rolled into 2026-07-02:\n got %+v\nwant %+v", got, want)
	}
}

// The rollup prunes raw events older than the project's retention window —
// and only those.
func TestRollupPrunesExpiredEvents(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	p := e.createProject(t, "p1", "UTC", 30)
	other := e.createProject(t, "p2", "UTC", 30)

	now := "2026-07-01T00:00:00Z"
	cutoff := "2026-06-01T00:00:00Z" // now - 30d
	if err := e.store.InsertEvents(ctx, []store.Event{
		{ID: "prune-1", ProjectID: p.ID, Type: store.EventUserLogin, CreatedAt: mustTime(t, "2026-05-01T00:00:00Z")},
		{ID: "prune-2", ProjectID: p.ID, Type: store.EventUserLogin, CreatedAt: mustTime(t, "2026-05-31T23:59:59.999Z")},
		{ID: "keep-1", ProjectID: p.ID, Type: store.EventUserLogin, CreatedAt: mustTime(t, cutoff)},
		{ID: "keep-2", ProjectID: p.ID, Type: store.EventUserLogin, CreatedAt: mustTime(t, "2026-06-30T00:00:00Z")},
		{ID: "other-1", ProjectID: other.ID, Type: store.EventUserLogin, CreatedAt: mustTime(t, "2026-05-01T00:00:00Z")},
	}); err != nil {
		t.Fatal(err)
	}

	run, err := NewRollup(e.store, testLogger(), fixedNow(t, now)).Run(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if run.EventsPruned != 2 {
		t.Fatalf("pruned %d events, want 2", run.EventsPruned)
	}
	left, err := e.store.ListRecentEvents(ctx, p.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 2 || left[0].ID != "keep-2" || left[1].ID != "keep-1" {
		t.Fatalf("wrong events survived: %+v", left)
	}
	// Scoped run: the other project's expired event is untouched.
	otherLeft, err := e.store.ListRecentEvents(ctx, other.ID, 100)
	if err != nil || len(otherLeft) != 1 {
		t.Fatalf("other project pruned: %+v (%v)", otherLeft, err)
	}

	// The run is on record.
	runs, err := e.store.ListRollupRuns(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].ID != run.ID || runs[0].EventsPruned != 2 || runs[0].Error != "" {
		t.Fatalf("rollup run not recorded: %+v", runs)
	}
}

// Pruning never cuts into the newest rolled-up day: with a 1-day retention
// every hourly run re-rolls yesterday, so yesterday's raw events must stay
// intact until the day is final (a rolling cutoff used to eat them run by
// run, decaying the day's row to zero).
func TestRollupShortRetentionKeepsRerolledDay(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	p := e.createProject(t, "p1", "UTC", 1)
	if err := e.store.InsertEvents(ctx, []store.Event{
		{ID: newID(), ProjectID: p.ID, UserID: "u1", Type: store.EventUserLogin,
			Provider: store.IdentityProviderPassword, CreatedAt: mustTime(t, "2026-07-02T00:30:00Z")},
		{ID: newID(), ProjectID: p.ID, UserID: "u2", Type: store.EventUserLogin,
			Provider: store.IdentityProviderPassword, CreatedAt: mustTime(t, "2026-07-02T10:00:00Z")},
	}); err != nil {
		t.Fatal(err)
	}

	// 02:00 run: a naive cutoff of now-1d (= 02:00 yesterday) would prune
	// the 00:30 event the 03:00 re-roll still needs.
	if _, err := NewRollup(e.store, testLogger(), fixedNow(t, "2026-07-03T02:00:00Z")).Run(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := NewRollup(e.store, testLogger(), fixedNow(t, "2026-07-03T03:00:00Z")).Run(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	got := e.rawDailyStats(t, p.ID, "2026-07-02")
	want := rawStats{found: true, logins: 2, dau: 2, password: 2, other: 2}
	if got != want {
		t.Fatalf("yesterday decayed across hourly runs:\n got %+v\nwant %+v", got, want)
	}
	left, err := e.store.ListRecentEvents(ctx, p.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 2 {
		t.Fatalf("yesterday's events pruned while still re-rolled: %+v", left)
	}
}

// Days that fell behind the backfill window during an outage are still
// aggregated on recovery — and only then pruned — instead of being deleted
// unrolled, which would leave a permanent gap in daily_stats.
func TestRollupBackfillsGapAfterOutage(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	p := e.createProject(t, "p1", "UTC", 7)

	// Normal operation: 2026-07-01 is rolled up.
	if err := e.store.InsertEvents(ctx, []store.Event{
		{ID: newID(), ProjectID: p.ID, UserID: "u1", Type: store.EventUserLogin,
			Provider: store.IdentityProviderPassword, CreatedAt: mustTime(t, "2026-07-01T10:00:00Z")},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewRollup(e.store, testLogger(), fixedNow(t, "2026-07-02T01:00:00Z")).Run(ctx, p.ID); err != nil {
		t.Fatal(err)
	}

	// Rollups then fail for 10 days (longer than the 7-day retention)
	// while traffic continues.
	if err := e.store.InsertEvents(ctx, []store.Event{
		{ID: newID(), ProjectID: p.ID, UserID: "u2", Type: store.EventUserLogin,
			Provider: store.IdentityProviderPassword, CreatedAt: mustTime(t, "2026-07-03T10:00:00Z")},
		{ID: newID(), ProjectID: p.ID, UserID: "u3", Type: store.EventUserLogin,
			Provider: store.IdentityProviderPassword, CreatedAt: mustTime(t, "2026-07-10T10:00:00Z")},
	}); err != nil {
		t.Fatal(err)
	}

	// Recovery: 2026-07-03 is beyond the retention-sized backfill window,
	// but must be aggregated before its events expire.
	run, err := NewRollup(e.store, testLogger(), fixedNow(t, "2026-07-12T01:00:00Z")).Run(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Resumed from 2026-07-01 (the newest rolled day) through 2026-07-11.
	if run.DaysProcessed != 11 {
		t.Fatalf("recovery processed %d days, want 11", run.DaysProcessed)
	}
	got := e.rawDailyStats(t, p.ID, "2026-07-03")
	want := rawStats{found: true, logins: 1, dau: 1, password: 1, other: 1}
	if got != want {
		t.Fatalf("gap day not aggregated on recovery:\n got %+v\nwant %+v", got, want)
	}
	// Retention still holds afterwards: the recovered days' expired raw
	// events are pruned once aggregated.
	left, err := e.store.ListRecentEvents(ctx, p.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 1 || left[0].UserID != "u3" {
		t.Fatalf("expired events should be pruned after aggregation: %+v", left)
	}
}

// A retention longer than the old flat 90-day backfill cap: the first
// rollup covers the whole retention window instead of leaving permanent
// zero rows beyond day 90.
func TestRollupBackfillCoversLongRetention(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	p := e.createProject(t, "p1", "UTC", 120)
	if err := e.store.InsertEvents(ctx, []store.Event{
		{ID: newID(), ProjectID: p.ID, UserID: "u1", Type: store.EventUserLogin,
			Provider: store.IdentityProviderPassword, CreatedAt: mustTime(t, "2026-03-23T12:00:00Z")},
	}); err != nil {
		t.Fatal(err)
	}

	// 100 days later, still within the 120-day retention.
	if _, err := NewRollup(e.store, testLogger(), fixedNow(t, "2026-07-01T01:00:00Z")).Run(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	got := e.rawDailyStats(t, p.ID, "2026-03-23")
	want := rawStats{found: true, logins: 1, dau: 1, password: 1, other: 1}
	if got != want {
		t.Fatalf("day beyond 90-day backfill not aggregated:\n got %+v\nwant %+v", got, want)
	}
	// Its raw event is retained too (retention > age).
	left, err := e.store.ListRecentEvents(ctx, p.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 1 {
		t.Fatalf("event within retention pruned: %+v", left)
	}
}

// A project with zero traffic rolls up cleanly to all-zero rows.
func TestRollupEmptyProject(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	p := e.createProject(t, "p1", "UTC", 0)

	run, err := NewRollup(e.store, testLogger(), fixedNow(t, "2026-07-03T00:00:00Z")).Run(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if run.DaysProcessed == 0 || run.EventsPruned != 0 {
		t.Fatalf("unexpected run: %+v", run)
	}
	got := e.rawDailyStats(t, p.ID, "2026-07-02")
	if !got.found || got != (rawStats{found: true}) {
		t.Fatalf("want an all-zero row, got %+v", got)
	}
}

// Seeded data rolls up into numbers that match independent SQL over the raw
// events, and the generator is deterministic in its seed.
func TestSeedAndRollupSmoke(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	p := e.createProject(t, "p1", "UTC", 0)

	nowFn := fixedNow(t, "2026-07-01T12:00:00Z")
	n, err := Seed(ctx, e.store, p, SeedOptions{Days: 30, Seed: 42, Now: nowFn})
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("seed generated no events")
	}
	var stored int
	if err := e.raw.QueryRow(`SELECT COUNT(*) FROM events WHERE project_id = ?`, p.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != n {
		t.Fatalf("seed reported %d events, stored %d", n, stored)
	}

	if _, err := NewRollup(e.store, testLogger(), nowFn).Run(ctx, p.ID); err != nil {
		t.Fatal(err)
	}

	// Cross-check the rolled-up totals against independent SQL on the raw
	// events (all seeded days are within the UTC backfill window).
	var wantSignups, wantLogins, wantFailures int
	for typ, dst := range map[string]*int{
		store.EventUserSignup:      &wantSignups,
		store.EventUserLogin:       &wantLogins,
		store.EventUserLoginFailed: &wantFailures,
	} {
		if err := e.raw.QueryRow(
			`SELECT COUNT(*) FROM events WHERE project_id = ? AND type = ?`,
			p.ID, typ).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	var gotSignups, gotLogins, gotFailures int
	if err := e.raw.QueryRow(
		`SELECT COALESCE(SUM(signups),0), COALESCE(SUM(logins),0), COALESCE(SUM(failures),0)
		   FROM daily_stats WHERE project_id = ?`, p.ID,
	).Scan(&gotSignups, &gotLogins, &gotFailures); err != nil {
		t.Fatal(err)
	}
	if gotSignups != wantSignups || gotLogins != wantLogins || gotFailures != wantFailures {
		t.Fatalf("rollup totals (s=%d l=%d f=%d) != raw events (s=%d l=%d f=%d)",
			gotSignups, gotLogins, gotFailures, wantSignups, wantLogins, wantFailures)
	}
	if gotSignups == 0 || gotLogins == 0 || gotFailures == 0 {
		t.Fatal("seed should produce signups, logins and failures")
	}

	// Determinism: the same seed produces the same stream — content, not
	// just count (a nondeterministic timestamp or provider pick would keep
	// totals equal while changing every dashboard between runs).
	e2 := newEnv(t)
	p2 := e2.createProject(t, "p1", "UTC", 0)
	n2, err := Seed(ctx, e2.store, p2, SeedOptions{Days: 30, Seed: 42, Now: nowFn})
	if err != nil {
		t.Fatal(err)
	}
	if n2 != n {
		t.Fatalf("same seed produced %d then %d events", n, n2)
	}
	d1, d2 := e.seedDigest(t, p.ID), e2.seedDigest(t, p2.ID)
	if d1 != d2 {
		t.Fatalf("same seed produced different event content:\n run 1: %s\n run 2: %s", d1, d2)
	}
}

// seedDigest summarizes a project's raw event content with independent SQL
// (per-group counts, distinct users and timestamp extremes) so run-to-run
// determinism can be asserted on content, not only on the event count.
func (e *testEnv) seedDigest(t *testing.T, projectID string) string {
	t.Helper()
	rows, err := e.raw.Query(
		`SELECT type, provider, platform, COUNT(*), COUNT(DISTINCT user_id),
		        MIN(created_at), MAX(created_at)
		   FROM events
		  WHERE project_id = ?
		  GROUP BY type, provider, platform
		  ORDER BY type, provider, platform`, projectID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var b strings.Builder
	for rows.Next() {
		var typ, provider, platform, minAt, maxAt string
		var count, users int
		if err := rows.Scan(&typ, &provider, &platform, &count, &users, &minAt, &maxAt); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&b, "%s/%s/%s n=%d u=%d %s..%s; ",
			typ, provider, platform, count, users, minAt, maxAt)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return b.String()
}
