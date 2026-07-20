package store

import (
	"context"
	"testing"
	"time"

	// The DST/UTC+13 cases need IANA zones even on hosts without a system
	// zoneinfo directory.
	_ "time/tzdata"
)

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestDayWindow(t *testing.T) {
	tests := []struct {
		name     string
		date     string
		tz       string
		from, to string
	}{
		{
			name: "utc",
			date: "2026-07-01", tz: "UTC",
			from: "2026-07-01T00:00:00Z", to: "2026-07-02T00:00:00Z",
		},
		{
			// Paris springs forward on 2026-03-29 (02:00 CET → 03:00
			// CEST): a 23-hour day.
			name: "dst spring forward",
			date: "2026-03-29", tz: "Europe/Paris",
			from: "2026-03-28T23:00:00Z", to: "2026-03-29T22:00:00Z",
		},
		{
			// Paris falls back on 2026-10-25 (03:00 CEST → 02:00 CET): a
			// 25-hour day.
			name: "dst fall back",
			date: "2026-10-25", tz: "Europe/Paris",
			from: "2026-10-24T22:00:00Z", to: "2026-10-25T23:00:00Z",
		},
		{
			// UTC+13: the local day starts on the previous UTC calendar
			// day (and here, the previous UTC year).
			name: "utc+13",
			date: "2026-01-01", tz: "Pacific/Fakaofo",
			from: "2025-12-31T11:00:00Z", to: "2026-01-01T11:00:00Z",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			from, to, err := DayWindow(tc.date, mustLoc(t, tc.tz))
			if err != nil {
				t.Fatal(err)
			}
			if !from.Equal(mustTime(t, tc.from)) || !to.Equal(mustTime(t, tc.to)) {
				t.Fatalf("window [%v, %v), want [%s, %s)", from, to, tc.from, tc.to)
			}
		})
	}

	if _, _, err := DayWindow("01/02/2026", time.UTC); err == nil {
		t.Fatal("malformed date should error")
	}
}

func TestInsertEventsAndListRecent(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p, k := testProject("p1", "app-one")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	p2, k2 := testProject("p2", "app-two")
	if err := s.CreateProject(ctx, p2, k2); err != nil {
		t.Fatal(err)
	}

	events := []Event{
		{
			ID: "e1", ProjectID: "p1", UserID: "u1", Type: EventUserSignup,
			Provider: IdentityProviderPassword, Platform: PlatformIOS,
			SDKVersion: "1.2.0", CreatedAt: mustTime(t, "2026-07-01T10:00:00Z"),
		},
		{
			ID: "e2", ProjectID: "p1", Type: EventUserLoginFailed,
			Metadata:  `{"reason":"wrong_password"}`,
			CreatedAt: mustTime(t, "2026-07-01T11:00:00Z"),
		},
		{
			ID: "e3", ProjectID: "p1", UserID: "u1", Type: EventUserLogin,
			Provider: IdentityProviderPassword, Platform: PlatformIOS,
			CreatedAt: mustTime(t, "2026-07-01T12:00:00Z"),
		},
	}
	if err := s.InsertEvents(ctx, events); err != nil {
		t.Fatal(err)
	}
	// Empty batch is a no-op, and another project's feed stays separate.
	if err := s.InsertEvents(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertEvent(ctx, Event{
		ID: "e4", ProjectID: "p2", UserID: "ux", Type: EventUserLogin,
		CreatedAt: mustTime(t, "2026-07-01T13:00:00Z"),
	}); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListRecentEvents(ctx, "p1", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 events, got %d", len(got))
	}
	if got[0].ID != "e3" || got[1].ID != "e2" || got[2].ID != "e1" {
		t.Fatalf("want newest first (e3,e2,e1), got %s,%s,%s", got[0].ID, got[1].ID, got[2].ID)
	}
	if got[1].UserID != "" || got[1].Metadata != `{"reason":"wrong_password"}` {
		t.Fatalf("nullable fields not round-tripped: %+v", got[1])
	}
	if got[2].SDKVersion != "1.2.0" || got[2].Provider != IdentityProviderPassword ||
		got[2].Platform != PlatformIOS || !got[2].CreatedAt.Equal(events[0].CreatedAt) {
		t.Fatalf("event fields not round-tripped: %+v", got[2])
	}

	limited, err := s.ListRecentEvents(ctx, "p1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(limited) != 1 || limited[0].ID != "e3" {
		t.Fatalf("limit not applied: %+v", limited)
	}
}

func TestAggregateDailyStats(t *testing.T) {
	// Compact event literal: only what aggregation looks at.
	type ev struct {
		id, user, typ, provider, platform string
		at                                string // RFC3339Nano
	}
	tests := []struct {
		name   string
		tz     string
		date   string
		events []ev
		want   DailyStats
	}{
		{
			name: "utc counts and boundaries",
			tz:   "UTC", date: "2026-07-01",
			events: []ev{
				{"e1", "u1", EventUserSignup, IdentityProviderPassword, PlatformIOS, "2026-07-01T01:00:00Z"},
				{"e2", "u1", EventUserLogin, IdentityProviderPassword, PlatformIOS, "2026-07-01T02:00:00Z"},
				{"e3", "u2", EventUserLogin, IdentityProviderGoogle, PlatformAndroid, "2026-07-01T03:00:00Z"},
				{"e4", "u3", EventUserLogin, IdentityProviderApple, PlatformWeb, "2026-07-01T04:00:00Z"},
				// No platform reported → bucketed as "other"; u3 is still
				// one distinct active user.
				{"e5", "u3", EventUserLogin, IdentityProviderApple, "", "2026-07-01T05:00:00Z"},
				// Refresh counts toward DAU, not logins.
				{"e6", "u4", EventTokenRefresh, "", PlatformIOS, "2026-07-01T06:00:00Z"},
				// Failure has no subject; must not disturb DAU.
				{"e7", "", EventUserLoginFailed, IdentityProviderPassword, PlatformIOS, "2026-07-01T07:00:00Z"},
				// Non-counter event type is ignored.
				{"e8", "u1", EventEmailVerified, "", "", "2026-07-01T08:00:00Z"},
				// RFC3339Nano ordering edge: a fractional timestamp in the
				// very first second of the day belongs to this day even
				// though "…00.5Z" sorts before "…00Z" lexicographically.
				{"e9", "u5", EventUserLogin, IdentityProviderPassword, PlatformIOS, "2026-07-01T00:00:00.5Z"},
				// Outside the window on both sides, including the first
				// (whole and fractional) second of the next day.
				{"x1", "u9", EventUserLogin, IdentityProviderPassword, PlatformIOS, "2026-06-30T23:59:59.999Z"},
				{"x2", "u9", EventUserLogin, IdentityProviderPassword, PlatformIOS, "2026-07-02T00:00:00Z"},
				{"x3", "u9", EventUserLogin, IdentityProviderPassword, PlatformIOS, "2026-07-02T00:00:00.5Z"},
			},
			want: DailyStats{
				Date: "2026-07-01", Signups: 1, Logins: 5, DAU: 5, Failures: 1,
				LoginsPassword: 2, LoginsGoogle: 1, LoginsApple: 2,
				PlatformIOS: 2, PlatformAndroid: 1, PlatformWeb: 1, PlatformOther: 1,
			},
		},
		{
			// 23-hour day: Paris springs forward on 2026-03-29, so the
			// local day is [2026-03-28T23:00Z, 2026-03-29T22:00Z).
			name: "dst spring forward day",
			tz:   "Europe/Paris", date: "2026-03-29",
			events: []ev{
				{"e1", "u1", EventUserLogin, IdentityProviderPassword, PlatformIOS, "2026-03-28T23:00:00Z"},
				{"e2", "u2", EventUserLogin, IdentityProviderPassword, PlatformIOS, "2026-03-29T21:59:59Z"},
				{"x1", "u9", EventUserLogin, IdentityProviderPassword, PlatformIOS, "2026-03-28T22:59:59Z"},
				{"x2", "u9", EventUserLogin, IdentityProviderPassword, PlatformIOS, "2026-03-29T22:00:00Z"},
			},
			want: DailyStats{
				Date: "2026-03-29", Logins: 2, DAU: 2,
				LoginsPassword: 2, PlatformIOS: 2,
			},
		},
		{
			// UTC+13: local Jan 1 spans two UTC days (and two UTC years).
			name: "utc+13 day",
			tz:   "Pacific/Fakaofo", date: "2026-01-01",
			events: []ev{
				{"e1", "u1", EventUserLogin, IdentityProviderGoogle, PlatformAndroid, "2025-12-31T11:00:00Z"},
				{"e2", "u2", EventUserLogin, IdentityProviderGoogle, PlatformAndroid, "2026-01-01T10:59:59Z"},
				{"x1", "u9", EventUserLogin, IdentityProviderGoogle, PlatformAndroid, "2025-12-31T10:59:59Z"},
				{"x2", "u9", EventUserLogin, IdentityProviderGoogle, PlatformAndroid, "2026-01-01T11:00:00Z"},
			},
			want: DailyStats{
				Date: "2026-01-01", Logins: 2, DAU: 2,
				LoginsGoogle: 2, PlatformAndroid: 2,
			},
		},
		{
			name:   "zero traffic",
			tz:     "UTC",
			date:   "2026-07-01",
			events: nil,
			want:   DailyStats{Date: "2026-07-01"},
		},
	}
	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := openTestStore(t)
			ctx := context.Background()
			p, k := testProject("p1", "my-app")
			if err := s.CreateProject(ctx, p, k); err != nil {
				t.Fatal(err)
			}
			var batch []Event
			for _, e := range tc.events {
				batch = append(batch, Event{
					ID: e.id, ProjectID: "p1", UserID: e.user, Type: e.typ,
					Provider: e.provider, Platform: e.platform,
					CreatedAt: mustTime(t, e.at),
				})
			}
			if err := s.InsertEvents(ctx, batch); err != nil {
				t.Fatal(err)
			}
			from, to, err := DayWindow(tc.date, mustLoc(t, tc.tz))
			if err != nil {
				t.Fatal(err)
			}
			got, err := s.AggregateDailyStats(ctx, "p1", tc.date, from, to)
			if err != nil {
				t.Fatal(err)
			}
			tc.want.ProjectID = "p1"
			if got != tc.want {
				t.Fatalf("case %d: aggregate mismatch:\n got %+v\nwant %+v", i, got, tc.want)
			}
		})
	}
}

func TestUpsertDailyStatsIdempotentAndGet(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p, k := testProject("p1", "my-app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}

	day1 := DailyStats{
		ProjectID: "p1", Date: "2026-07-01", Signups: 3, Logins: 10, DAU: 7,
		Failures: 1, LoginsPassword: 6, LoginsGoogle: 3, LoginsApple: 1,
		PlatformIOS: 5, PlatformAndroid: 3, PlatformWeb: 1, PlatformOther: 1,
	}
	day2 := DailyStats{ProjectID: "p1", Date: "2026-07-02", Logins: 4, DAU: 4, LoginsPassword: 4, PlatformOther: 4}
	for _, ds := range []DailyStats{day1, day2} {
		if err := s.UpsertDailyStats(ctx, ds); err != nil {
			t.Fatal(err)
		}
	}
	// Re-rolling the same day must replace, not duplicate or accumulate.
	if err := s.UpsertDailyStats(ctx, day1); err != nil {
		t.Fatal(err)
	}
	day1.Logins = 11
	if err := s.UpsertDailyStats(ctx, day1); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetDailyStats(ctx, "p1", "2026-07-01", "2026-07-02")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 rows, got %d: %+v", len(got), got)
	}
	if got[0] != day1 || got[1] != day2 {
		t.Fatalf("rows mismatch:\n got %+v\nwant %+v", got, []DailyStats{day1, day2})
	}

	// Range bounds are inclusive; other ranges and projects stay empty.
	one, err := s.GetDailyStats(ctx, "p1", "2026-07-02", "2026-07-02")
	if err != nil || len(one) != 1 || one[0].Date != "2026-07-02" {
		t.Fatalf("inclusive range: %+v (%v)", one, err)
	}
	none, err := s.GetDailyStats(ctx, "p1", "2026-07-03", "2026-07-31")
	if err != nil || len(none) != 0 {
		t.Fatalf("empty range: %+v (%v)", none, err)
	}
	other, err := s.GetDailyStats(ctx, "p2", "2026-07-01", "2026-07-02")
	if err != nil || len(other) != 0 {
		t.Fatalf("project scoping: %+v (%v)", other, err)
	}
}

func TestDeleteEventsBefore(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	for _, id := range []string{"p1", "p2"} {
		p, k := testProject(id, "app-"+id)
		if err := s.CreateProject(ctx, p, k); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.InsertEvents(ctx, []Event{
		{ID: "old1", ProjectID: "p1", Type: EventUserLogin, CreatedAt: mustTime(t, "2026-03-01T00:00:00Z")},
		{ID: "old2", ProjectID: "p1", Type: EventUserLogin, CreatedAt: mustTime(t, "2026-03-31T23:59:59.9Z")},
		{ID: "keep1", ProjectID: "p1", Type: EventUserLogin, CreatedAt: mustTime(t, "2026-04-01T00:00:00Z")},
		{ID: "keep2", ProjectID: "p1", Type: EventUserLogin, CreatedAt: mustTime(t, "2026-04-01T00:00:00.5Z")},
		{ID: "other", ProjectID: "p2", Type: EventUserLogin, CreatedAt: mustTime(t, "2026-03-01T00:00:00Z")},
	}); err != nil {
		t.Fatal(err)
	}

	n, err := s.DeleteEventsBefore(ctx, "p1", mustTime(t, "2026-04-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("want 2 pruned, got %d", n)
	}
	left, err := s.ListRecentEvents(ctx, "p1", 50)
	if err != nil {
		t.Fatal(err)
	}
	// Same-second events have no guaranteed relative order in the feed
	// (see ListRecentEvents), so assert the surviving set, not its order.
	survived := map[string]bool{}
	for _, e := range left {
		survived[e.ID] = true
	}
	if len(left) != 2 || !survived["keep1"] || !survived["keep2"] {
		t.Fatalf("cutoff wrong (events at/after cutoff must survive): %+v", left)
	}
	// The other project's events are untouched.
	otherLeft, err := s.ListRecentEvents(ctx, "p2", 50)
	if err != nil || len(otherLeft) != 1 {
		t.Fatalf("other project pruned: %+v (%v)", otherLeft, err)
	}
}

func TestRollupRuns(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	r1 := RollupRun{
		ID:        "r1",
		StartedAt: mustTime(t, "2026-07-01T03:00:00Z"), FinishedAt: mustTime(t, "2026-07-01T03:00:02Z"),
		DaysProcessed: 4, EventsPruned: 120,
	}
	r2 := RollupRun{
		ID:        "r2",
		StartedAt: mustTime(t, "2026-07-02T03:00:00Z"), FinishedAt: mustTime(t, "2026-07-02T03:00:01Z"),
		Error: "aggregate p1: disk I/O error",
	}
	for _, r := range []RollupRun{r1, r2} {
		if err := s.InsertRollupRun(ctx, r); err != nil {
			t.Fatal(err)
		}
	}

	runs, err := s.ListRollupRuns(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 || runs[0].ID != "r2" || runs[1].ID != "r1" {
		t.Fatalf("want newest first (r2,r1), got %+v", runs)
	}
	if runs[0].Error != r2.Error || runs[1].Error != "" {
		t.Fatalf("error column not round-tripped: %+v", runs)
	}
	if runs[1].DaysProcessed != 4 || runs[1].EventsPruned != 120 ||
		!runs[1].StartedAt.Equal(r1.StartedAt) || !runs[1].FinishedAt.Equal(r1.FinishedAt) {
		t.Fatalf("run fields not round-tripped: %+v", runs[1])
	}

	limited, err := s.ListRollupRuns(ctx, 1)
	if err != nil || len(limited) != 1 || limited[0].ID != "r2" {
		t.Fatalf("limit not applied: %+v (%v)", limited, err)
	}
}

func TestProjectSettingsAnalyticsDefaults(t *testing.T) {
	// A stored JSON predating milestone 07 gains the defaults.
	ps, err := parseProjectSettings(`{"password_min_length":10}`)
	if err != nil {
		t.Fatal(err)
	}
	if ps.AnalyticsRetentionDays != 90 || ps.RollupTimezone != "UTC" {
		t.Fatalf("defaults not applied: %+v", ps)
	}

	// Explicit values survive the round trip.
	ps.AnalyticsRetentionDays = 30
	ps.RollupTimezone = "Europe/Paris"
	raw, err := encodeProjectSettings(ps)
	if err != nil {
		t.Fatal(err)
	}
	back, err := parseProjectSettings(raw)
	if err != nil {
		t.Fatal(err)
	}
	if back.AnalyticsRetentionDays != 30 || back.RollupTimezone != "Europe/Paris" {
		t.Fatalf("values not round-tripped: %+v", back)
	}
	if back.RollupLocation().String() != "Europe/Paris" {
		t.Fatalf("RollupLocation = %v", back.RollupLocation())
	}

	// A timezone that stopped resolving must degrade to UTC, not crash the
	// rollup.
	broken := ProjectSettings{RollupTimezone: "Mars/Olympus_Mons"}
	if broken.RollupLocation() != time.UTC {
		t.Fatalf("broken timezone should fall back to UTC, got %v", broken.RollupLocation())
	}
}
