package server

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/gen/moth/auth/v1/authv1connect"
	adminrpc "github.com/aloisdeniel/moth/internal/server/rpc/admin"
	"github.com/aloisdeniel/moth/internal/store"
)

// sdkTransport mimics the Flutter SDK: publishable key plus the ambient
// analytics metadata on every call.
type sdkTransport struct {
	key, platform, version string
}

func (st sdkTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("x-moth-key", st.key)
	r.Header.Set("x-moth-platform", st.platform)
	r.Header.Set("x-moth-sdk-version", st.version)
	return http.DefaultTransport.RoundTrip(r)
}

func (e *testEnv) analytics() adminv1connect.AnalyticsServiceClient {
	return adminv1connect.NewAnalyticsServiceClient(e.client, e.url)
}

// Auth RPCs emit analytics events through the async writer, carrying the
// SDK metadata, and login failures carry a bucketed reason but no user id.
func TestAnalyticsEventEmission(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Events App")
	ctx := context.Background()

	hc := &http.Client{Transport: sdkTransport{p.PublishableKey, "iOS", "1.2.3"}}
	auth := authv1connect.NewAuthServiceClient(hc, e.url)

	if _, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "jane@example.com", Password: "password-1"})); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
		Email: "jane@example.com", Password: "wrong-password"})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("wrong password: %v", err)
	}
	if _, err := auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
		Email: "jane@example.com", Password: "password-1"})); err != nil {
		t.Fatal(err)
	}

	// Drain the buffered writer so the events are queryable.
	drainCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := e.srv.Close(drainCtx); err != nil {
		t.Fatalf("drain events: %v", err)
	}

	events, err := e.store.ListRecentEvents(ctx, p.Id, 50)
	if err != nil {
		t.Fatal(err)
	}
	byType := map[string][]store.Event{}
	for _, ev := range events {
		byType[ev.Type] = append(byType[ev.Type], ev)
	}
	signups := byType[store.EventUserSignup]
	if len(signups) != 1 || signups[0].Provider != store.IdentityProviderPassword ||
		signups[0].UserID == "" {
		t.Fatalf("signup event: %+v", signups)
	}
	logins := byType[store.EventUserLogin]
	if len(logins) != 1 || logins[0].Provider != store.IdentityProviderPassword ||
		logins[0].UserID != signups[0].UserID {
		t.Fatalf("login event: %+v", logins)
	}
	failures := byType[store.EventUserLoginFailed]
	if len(failures) != 1 {
		t.Fatalf("failure events: %+v", failures)
	}
	if failures[0].UserID != "" {
		t.Fatalf("login_failed must carry no user id: %+v", failures[0])
	}
	if failures[0].Metadata != `{"reason":"invalid_credentials"}` {
		t.Fatalf("failure reason not bucketed: %q", failures[0].Metadata)
	}
	// The SDK metadata flowed from the request headers into every event
	// (platform normalized to lower case).
	for _, ev := range events {
		if ev.Platform != "ios" || ev.SDKVersion != "1.2.3" {
			t.Fatalf("client info missing on %s: %+v", ev.Type, ev)
		}
	}
}

// insertDayEvents writes hand-crafted raw events for one UTC day.
func (e *testEnv) insertDayEvents(t *testing.T, projectID string, day time.Time, events []store.Event) {
	t.Helper()
	at := time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, time.UTC)
	for i := range events {
		events[i].ID = adminrpc.NewID()
		events[i].ProjectID = projectID
		events[i].CreatedAt = at.Add(time.Duration(i) * time.Minute)
	}
	if err := e.store.InsertEvents(context.Background(), events); err != nil {
		t.Fatal(err)
	}
}

// RunRollup + GetStats + the CSV export agree with hand-computed numbers,
// and the export enforces the admin session.
func TestAnalyticsStatsAndExport(t *testing.T) {
	// The server's clock is pinned so the test cannot flake when the wall
	// clock crosses UTC midnight between inserting events and rolling up
	// (the rollup and GetStats would move on to a new "yesterday").
	fixed := time.Date(2026, 5, 20, 15, 0, 0, 0, time.UTC)
	e := newTestEnv(t, "tok", func(o *Options) {
		o.Now = func() time.Time { return fixed }
	})
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Stats App")
	ctx := context.Background()

	// Two completed UTC days of traffic (the project's rollup timezone
	// defaults to UTC).
	yesterday := fixed.AddDate(0, 0, -1)
	dayBefore := yesterday.AddDate(0, 0, -1)
	d1 := dayBefore.Format("2006-01-02")
	d2 := yesterday.Format("2006-01-02")
	e.insertDayEvents(t, p.Id, dayBefore, []store.Event{
		{UserID: "u1", Type: store.EventUserSignup, Provider: store.IdentityProviderPassword, Platform: store.PlatformIOS},
		{UserID: "u1", Type: store.EventUserLogin, Provider: store.IdentityProviderPassword, Platform: store.PlatformIOS},
		{UserID: "u2", Type: store.EventUserLogin, Provider: store.IdentityProviderGoogle, Platform: store.PlatformAndroid},
		{Type: store.EventUserLoginFailed, Provider: store.IdentityProviderPassword, Metadata: `{"reason":"invalid_credentials"}`},
	})
	e.insertDayEvents(t, p.Id, yesterday, []store.Event{
		{UserID: "u3", Type: store.EventUserLogin, Provider: store.IdentityProviderApple, Platform: store.PlatformWeb},
		{UserID: "u4", Type: store.EventTokenRefresh},
	})
	// Signups pinning down the two 7-day tile windows around their edges:
	// current = [yesterday-6, yesterday], previous = [yesterday-13,
	// yesterday-7]. One signup on each boundary, one just outside.
	e.insertDayEvents(t, p.Id, yesterday.AddDate(0, 0, -6), []store.Event{
		{UserID: "u5", Type: store.EventUserSignup, Provider: store.IdentityProviderPassword},
	})
	e.insertDayEvents(t, p.Id, yesterday.AddDate(0, 0, -7), []store.Event{
		{UserID: "u6", Type: store.EventUserSignup, Provider: store.IdentityProviderPassword},
	})
	e.insertDayEvents(t, p.Id, yesterday.AddDate(0, 0, -13), []store.Event{
		{UserID: "u7", Type: store.EventUserSignup, Provider: store.IdentityProviderPassword},
	})
	e.insertDayEvents(t, p.Id, yesterday.AddDate(0, 0, -14), []store.Event{
		{UserID: "u8", Type: store.EventUserSignup, Provider: store.IdentityProviderPassword},
	})

	analytics := e.analytics()
	run, err := analytics.RunRollup(ctx, connect.NewRequest(&adminv1.RunRollupRequest{ProjectId: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if run.Msg.DaysProcessed == 0 || run.Msg.RunId == "" {
		t.Fatalf("rollup response: %+v", run.Msg)
	}

	stats, err := analytics.GetStats(ctx, connect.NewRequest(&adminv1.GetStatsRequest{
		ProjectId: p.Id, FromDate: d1, ToDate: d2}))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(stats.Msg.Series); n != 2 {
		t.Fatalf("series length %d, want 2", n)
	}
	s1, s2 := stats.Msg.Series[0], stats.Msg.Series[1]
	if s1.Date != d1 || s1.Signups != 1 || s1.Logins != 2 || s1.Dau != 2 || s1.Failures != 1 {
		t.Fatalf("day 1: %+v", s1)
	}
	if s2.Date != d2 || s2.Signups != 0 || s2.Logins != 1 || s2.Dau != 2 || s2.Failures != 0 {
		t.Fatalf("day 2 (refresh counts toward DAU only): %+v", s2)
	}
	if pr := stats.Msg.Providers; pr.Password != 1 || pr.Google != 1 || pr.Apple != 1 {
		t.Fatalf("providers: %+v", pr)
	}
	if pl := stats.Msg.Platforms; pl.Ios != 1 || pl.Android != 1 || pl.Web != 1 || pl.Other != 0 {
		t.Fatalf("platforms: %+v", pl)
	}
	tiles := stats.Msg.Tiles
	if tiles.TotalUsers != 0 { // no real users rows behind the synthetic events
		t.Fatalf("total users: %d", tiles.TotalUsers)
	}
	// Current window: u1 (d1) + u5 (yesterday-6). Previous window: u6
	// (yesterday-7) + u7 (yesterday-13); u8 (yesterday-14) is outside both.
	if tiles.NewUsers_7D != 2 || tiles.Logins_7D != 3 || tiles.LoginFailures_7D != 1 {
		t.Fatalf("7d tiles: %+v", tiles)
	}
	if tiles.NewUsersPrevious_7D != 2 {
		t.Fatalf("previous-7d new users: %d, want 2 (%+v)", tiles.NewUsersPrevious_7D, tiles)
	}
	if tiles.LoginSuccessRate_7D != 0.75 {
		t.Fatalf("success rate: %v", tiles.LoginSuccessRate_7D)
	}
	if tiles.LatestDau != 2 || tiles.LatestDauDate != d2 {
		t.Fatalf("latest dau: %+v", tiles)
	}

	// The activity feed returns the newest raw events, capped.
	feed, err := analytics.ListRecentEvents(ctx, connect.NewRequest(&adminv1.ListRecentEventsRequest{
		ProjectId: p.Id, Limit: 2}))
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Msg.Events) != 2 || feed.Msg.Events[0].Type != store.EventTokenRefresh {
		t.Fatalf("feed: %+v", feed.Msg.Events)
	}

	// CSV download: session-cookie authed, zero-filled, correct numbers.
	// The range deliberately extends to "today", which the rollup never
	// materializes, so the export's zero-fill of days without a
	// daily_stats row is actually exercised.
	today := fixed.Format("2006-01-02")
	url := fmt.Sprintf("%s/admin/export/stats.csv?project_id=%s&from=%s&to=%s", e.url, p.Id, d1, today)
	resp, err := e.client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("csv: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/csv; charset=utf-8" {
		t.Fatalf("csv content type: %s", ct)
	}
	records, err := csv.NewReader(resp.Body).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 4 { // header + 2 rolled days + zero-filled today
		t.Fatalf("csv rows: %d (%v)", len(records), records)
	}
	if records[0][0] != "date" || records[0][4] != "failures" {
		t.Fatalf("csv header: %v", records[0])
	}
	want1 := []string{d1, "1", "2", "2", "1", "1", "1", "0", "1", "1", "0", "0"}
	for i, cell := range want1 {
		if records[1][i] != cell {
			t.Fatalf("csv day 1 column %d = %q, want %q (%v)", i, records[1][i], cell, records[1])
		}
	}
	wantGap := []string{today, "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0"}
	for i, cell := range wantGap {
		if records[3][i] != cell {
			t.Fatalf("csv gap day column %d = %q, want %q (%v)", i, records[3][i], cell, records[3])
		}
	}

	// No session cookie → 401; unknown project → 404; bad range → 400.
	anon, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	anon.Body.Close()
	if anon.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anon csv: %d, want 401", anon.StatusCode)
	}
	notFound, err := e.client.Get(e.url + "/admin/export/stats.csv?project_id=nope")
	if err != nil {
		t.Fatal(err)
	}
	notFound.Body.Close()
	if notFound.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown project csv: %d, want 404", notFound.StatusCode)
	}
	bad, err := e.client.Get(fmt.Sprintf("%s/admin/export/stats.csv?project_id=%s&from=%s&to=%s", e.url, p.Id, d2, d1))
	if err != nil {
		t.Fatal(err)
	}
	bad.Body.Close()
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("inverted range csv: %d, want 400", bad.StatusCode)
	}

	// The admin RPCs themselves require a session too.
	anonRPC := adminv1connect.NewAnalyticsServiceClient(http.DefaultClient, e.url)
	if _, err := anonRPC.GetStats(ctx, connect.NewRequest(&adminv1.GetStatsRequest{ProjectId: p.Id})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("anonymous GetStats: %v", err)
	}
}

// analytics_retention_days is validated like the rollup timezone: zero
// means "default", anything else must stay within 1..366 — an unbounded
// value would keep raw per-user events forever.
func TestAnalyticsRetentionValidation(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Retention App")
	ctx := context.Background()

	update := func(days int32) error {
		s := p.Settings
		s.AnalyticsRetentionDays = days
		_, err := e.projects.UpdateProject(ctx, connect.NewRequest(&adminv1.UpdateProjectRequest{
			Id: p.Id, Name: p.Name, Settings: s}))
		return err
	}
	if err := update(30); err != nil {
		t.Fatalf("30 days rejected: %v", err)
	}
	for _, bad := range []int32{-1, 367, 2147483647} {
		if err := update(bad); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("retention %d: %v, want invalid argument", bad, err)
		}
	}
	// Zero follows the proto3 zero-means-unset convention (defaults to 90
	// on the next load).
	if err := update(0); err != nil {
		t.Fatalf("zero retention rejected: %v", err)
	}
}

// A project with zero traffic gets a zero-filled default range, not an
// error — the dashboard renders an empty state from it.
func TestAnalyticsZeroTraffic(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Quiet App")
	ctx := context.Background()

	stats, err := e.analytics().GetStats(ctx, connect.NewRequest(&adminv1.GetStatsRequest{ProjectId: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(stats.Msg.Series); n != 30 {
		t.Fatalf("default series length %d, want 30", n)
	}
	for _, day := range stats.Msg.Series {
		if day.Signups != 0 || day.Logins != 0 || day.Dau != 0 || day.Failures != 0 {
			t.Fatalf("zero-traffic day not zero: %+v", day)
		}
	}
	tiles := stats.Msg.Tiles
	if tiles.TotalUsers != 0 || tiles.Logins_7D != 0 || tiles.LoginSuccessRate_7D != 0 ||
		tiles.LatestDauDate != "" {
		t.Fatalf("zero-traffic tiles: %+v", tiles)
	}

	// Unknown project and malformed dates are proper errors.
	if _, err := e.analytics().GetStats(ctx, connect.NewRequest(&adminv1.GetStatsRequest{ProjectId: "nope"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("unknown project: %v", err)
	}
	if _, err := e.analytics().GetStats(ctx, connect.NewRequest(&adminv1.GetStatsRequest{
		ProjectId: p.Id, FromDate: "01/02/2026"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("malformed date: %v", err)
	}
	if _, err := e.analytics().RunRollup(ctx, connect.NewRequest(&adminv1.RunRollupRequest{ProjectId: "nope"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("rollup unknown project: %v", err)
	}
}
