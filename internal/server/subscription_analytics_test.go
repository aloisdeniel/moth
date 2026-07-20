package server

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	adminrpc "github.com/aloisdeniel/moth/internal/server/rpc/admin"
	"github.com/aloisdeniel/moth/internal/store"
)

// insertSubEvents writes hand-crafted subscription events at noon on the given
// UTC day.
func (e *testEnv) insertSubEvents(t *testing.T, projectID string, day time.Time, evs []store.SubscriptionEvent) {
	t.Helper()
	at := time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, time.UTC)
	for i := range evs {
		evs[i].ID = adminrpc.NewID()
		evs[i].ProjectID = projectID
		evs[i].CreatedAt = at.Add(time.Duration(i) * time.Minute)
	}
	if err := e.store.InsertSubscriptionEvents(context.Background(), evs); err != nil {
		t.Fatal(err)
	}
}

// RunRollup + GetSubscriptionStats + the CSV export agree with hand-computed
// per-currency numbers; sandbox is excluded, the export is admin-authed and
// formula-injection-safe.
func TestSubscriptionStatsAndExport(t *testing.T) {
	fixed := time.Date(2026, 5, 20, 15, 0, 0, 0, time.UTC)
	e := newTestEnv(t, "tok", func(o *Options) {
		o.Now = func() time.Time { return fixed }
	})
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Revenue App")
	ctx := context.Background()

	const prod = store.SubscriptionEnvironmentProduction
	const sand = store.SubscriptionEnvironmentSandbox
	apple, google, stripe := store.SubscriptionStoreApple, store.SubscriptionStoreGoogle, store.SubscriptionStoreStripe

	april := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	may := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	e.insertSubEvents(t, p.Id, april, []store.SubscriptionEvent{
		{Type: store.SubscriptionEventPurchased, UserID: "u1", ProductID: "monthly", Store: apple, Currency: "USD", Environment: prod, PriceAmountMicros: 9_990_000},
		{Type: store.SubscriptionEventRenewed, UserID: "u2", ProductID: "monthly", Store: google, Currency: "USD", Environment: prod, PriceAmountMicros: 9_990_000},
		{Type: store.SubscriptionEventRefunded, UserID: "u3", ProductID: "monthly", Store: apple, Currency: "USD", Environment: prod, PriceAmountMicros: 9_990_000},
	})
	e.insertSubEvents(t, p.Id, may, []store.SubscriptionEvent{
		{Type: store.SubscriptionEventPurchased, UserID: "u4", ProductID: "monthly-eur", Store: google, Currency: "EUR", Environment: prod, PriceAmountMicros: 5_000_000},
		{Type: store.SubscriptionEventTrialStarted, UserID: "u5", ProductID: "monthly", Store: apple, Currency: "USD", Environment: prod},
		// Sandbox purchase — excluded from the production rollup.
		{Type: store.SubscriptionEventPurchased, UserID: "u6", ProductID: "monthly", Store: apple, Currency: "USD", Environment: sand, PriceAmountMicros: 1_000_000},
		// Hostile store-reported currency — must be neutralized in the CSV.
		{Type: store.SubscriptionEventPurchased, UserID: "u7", ProductID: "monthly", Store: apple, Currency: "=CMD()", Environment: prod, PriceAmountMicros: 2_000_000},
		// Stripe (web) purchase — the third store leg of the breakdown.
		{Type: store.SubscriptionEventPurchased, UserID: "u8", ProductID: "monthly", Store: stripe, Currency: "USD", Environment: prod, PriceAmountMicros: 4_000_000},
	})

	analytics := e.analytics()
	if _, err := analytics.RunRollup(ctx, connect.NewRequest(&adminv1.RunRollupRequest{ProjectId: p.Id})); err != nil {
		t.Fatal(err)
	}

	resp, err := analytics.GetSubscriptionStats(ctx, connect.NewRequest(&adminv1.GetSubscriptionStatsRequest{
		ProjectId: p.Id, FromPeriod: "2026-04", ToPeriod: "2026-05"}))
	if err != nil {
		t.Fatal(err)
	}
	msg := resp.Msg
	if len(msg.Series) != 2 {
		t.Fatalf("series length %d, want 2", len(msg.Series))
	}
	apr, mayS := msg.Series[0], msg.Series[1]
	// April USD net = 9.99 + 9.99 - 9.99 = 9.99.
	if apr.Period != "2026-04" || len(apr.Revenue) != 1 ||
		apr.Revenue[0].Currency != "USD" || apr.Revenue[0].AmountMicros != 9_990_000 {
		t.Fatalf("April series: %+v", apr)
	}
	if apr.NewSubscribers != 1 || apr.Renewals != 1 {
		t.Fatalf("April counts: %+v", apr)
	}
	// May: EUR 5.00 and the hostile-currency 2.00 (never blended); sandbox out.
	if mayS.Period != "2026-05" || mayS.TrialsStarted != 1 {
		t.Fatalf("May series: %+v", mayS)
	}
	mayByCur := map[string]int64{}
	for _, a := range mayS.Revenue {
		mayByCur[a.Currency] = a.AmountMicros
	}
	if mayByCur["EUR"] != 5_000_000 || mayByCur["USD"] != 4_000_000 ||
		mayByCur["=CMD()"] != 2_000_000 || len(mayS.Revenue) != 3 {
		t.Fatalf("May per-currency revenue (sandbox must be excluded): %+v", mayS.Revenue)
	}
	// Per-store breakdown over the range. Apple USD nets to zero (purchase minus
	// refund) so only the hostile currency survives; google spans two
	// currencies; stripe is the new third leg.
	if s := msg.Stores.Apple; len(s) != 1 || s[0].Currency != "=CMD()" || s[0].AmountMicros != 2_000_000 {
		t.Fatalf("apple store breakdown: %+v", s)
	}
	if s := msg.Stores.Google; len(s) != 2 ||
		s[0].Currency != "EUR" || s[0].AmountMicros != 5_000_000 ||
		s[1].Currency != "USD" || s[1].AmountMicros != 9_990_000 {
		t.Fatalf("google store breakdown: %+v", s)
	}
	if s := msg.Stores.Stripe; len(s) != 1 || s[0].Currency != "USD" || s[0].AmountMicros != 4_000_000 {
		t.Fatalf("stripe store breakdown: %+v", s)
	}
	// Tiles headline the latest rolled month (May, the current month).
	if msg.Tiles.LatestPeriod != "2026-05" {
		t.Fatalf("tiles latest period: %+v", msg.Tiles)
	}

	// CSV export: cookie-authed, per-currency rows, injection-safe.
	url := fmt.Sprintf("%s/admin/export/subscriptions.csv?project_id=%s&from=2026-04&to=2026-05", e.url, p.Id)
	csvResp, err := e.client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer csvResp.Body.Close()
	if csvResp.StatusCode != http.StatusOK {
		t.Fatalf("csv status: %d", csvResp.StatusCode)
	}
	records, err := csv.NewReader(csvResp.Body).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if records[0][0] != "period" || records[0][1] != "currency" || records[0][2] != "revenue_micros" ||
		records[0][len(records[0])-1] != "store_stripe_revenue_micros" {
		t.Fatalf("csv header: %v", records[0])
	}
	var sawAprilUSD, sawMayUSD, sawHostile, sawSandbox bool
	for _, row := range records[1:] {
		if row[0] == "2026-04" && row[1] == "USD" {
			sawAprilUSD = true
			if row[2] != "9990000" {
				t.Fatalf("April USD revenue cell = %q, want 9990000", row[2])
			}
		}
		// May USD revenue is stripe-only — the per-store cells must agree.
		if row[0] == "2026-05" && row[1] == "USD" {
			sawMayUSD = true
			if row[2] != "4000000" || row[9] != "0" || row[10] != "0" || row[11] != "4000000" {
				t.Fatalf("May USD row (stripe revenue): %v", row)
			}
		}
		if strings.Contains(row[1], "CMD") {
			sawHostile = true
			if !strings.HasPrefix(row[1], "'") {
				t.Fatalf("hostile currency not neutralized: %q", row[1])
			}
		}
		if row[2] == "1000000" {
			sawSandbox = true
		}
	}
	if !sawAprilUSD {
		t.Fatal("April USD row missing from CSV")
	}
	if !sawMayUSD {
		t.Fatal("May USD (stripe) row missing from CSV")
	}
	if !sawHostile {
		t.Fatal("hostile currency row missing from CSV")
	}
	if sawSandbox {
		t.Fatal("sandbox revenue leaked into CSV")
	}

	// No session cookie → 401; unknown project → 404; inverted range → 400.
	anon, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	anon.Body.Close()
	if anon.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anon csv: %d, want 401", anon.StatusCode)
	}
	notFound, err := e.client.Get(e.url + "/admin/export/subscriptions.csv?project_id=nope")
	if err != nil {
		t.Fatal(err)
	}
	notFound.Body.Close()
	if notFound.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown project csv: %d, want 404", notFound.StatusCode)
	}
	bad, err := e.client.Get(fmt.Sprintf("%s/admin/export/subscriptions.csv?project_id=%s&from=2026-05&to=2026-04", e.url, p.Id))
	if err != nil {
		t.Fatal(err)
	}
	bad.Body.Close()
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("inverted range csv: %d, want 400", bad.StatusCode)
	}
}

// Active subscribers must be a currency-agnostic distinct-user count, never the
// sum of per-currency rows (which double-counts a cross-currency user) nor a
// sum across months (which multiplies a recurring subscriber). Guards findings
// 1, 2, 6, 12.
func TestSubscriptionActiveSubscribersNotDoubleCounted(t *testing.T) {
	fixed := time.Date(2026, 5, 20, 15, 0, 0, 0, time.UTC)
	e := newTestEnv(t, "tok", func(o *Options) { o.Now = func() time.Time { return fixed } })
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Revenue App")
	ctx := context.Background()

	const prod = store.SubscriptionEnvironmentProduction
	apple := store.SubscriptionStoreApple

	april := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	may := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	e.insertSubEvents(t, p.Id, april, []store.SubscriptionEvent{
		{Type: store.SubscriptionEventPurchased, UserID: "u1", ProductID: "monthly", Store: apple, Currency: "USD", Environment: prod, PriceAmountMicros: 9_990_000},
	})
	e.insertSubEvents(t, p.Id, may, []store.SubscriptionEvent{
		// u1 transacts in BOTH USD and EUR the same month — one distinct human.
		{Type: store.SubscriptionEventRenewed, UserID: "u1", ProductID: "monthly", Store: apple, Currency: "USD", Environment: prod, PriceAmountMicros: 9_990_000},
		{Type: store.SubscriptionEventRenewed, UserID: "u1", ProductID: "monthly", Store: apple, Currency: "EUR", Environment: prod, PriceAmountMicros: 8_990_000},
		{Type: store.SubscriptionEventPurchased, UserID: "u2", ProductID: "monthly", Store: apple, Currency: "USD", Environment: prod, PriceAmountMicros: 9_990_000},
	})

	analytics := e.analytics()
	if _, err := analytics.RunRollup(ctx, connect.NewRequest(&adminv1.RunRollupRequest{ProjectId: p.Id})); err != nil {
		t.Fatal(err)
	}
	resp, err := analytics.GetSubscriptionStats(ctx, connect.NewRequest(&adminv1.GetSubscriptionStatsRequest{
		ProjectId: p.Id, FromPeriod: "2026-04", ToPeriod: "2026-05"}))
	if err != nil {
		t.Fatal(err)
	}
	msg := resp.Msg

	// May all-products active = {u1, u2} = 2, NOT 3 (summing USD's {u1,u2}=2 and
	// EUR's {u1}=1 would double-count u1).
	mayS := msg.Series[1]
	if mayS.Period != "2026-05" || mayS.ActiveSubscribers != 2 {
		t.Fatalf("May active = %d, want 2 (cross-currency user counted once): %+v", mayS.ActiveSubscribers, mayS)
	}
	// Headline tile (May) = 2, previous (April) = 1.
	if msg.Tiles.ActiveSubscribers != 2 || msg.Tiles.ActiveSubscribersPrevious != 1 {
		t.Fatalf("tiles active = %d, previous = %d, want 2 and 1", msg.Tiles.ActiveSubscribers, msg.Tiles.ActiveSubscribersPrevious)
	}
	// Per-tier active is the LATEST month's distinct count (May {u1,u2} = 2), not
	// summed across April+May (which would over-count the recurring u1).
	var monthlyActive int64 = -1
	for _, tr := range msg.Tiers {
		if tr.ProductId == "monthly" {
			monthlyActive = tr.ActiveSubscribers
		}
	}
	if monthlyActive != 2 {
		t.Fatalf("tier 'monthly' active = %d, want 2 (latest month, not summed across months)", monthlyActive)
	}
}

// A project with no subscription events renders a friendly empty state: zeroed
// tiles and a zero-filled series, not an error or broken charts.
func TestSubscriptionZeroTraffic(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Quiet Revenue App")
	ctx := context.Background()

	resp, err := e.analytics().GetSubscriptionStats(ctx, connect.NewRequest(&adminv1.GetSubscriptionStatsRequest{ProjectId: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	msg := resp.Msg
	if len(msg.Series) != 12 {
		t.Fatalf("default series length %d, want 12", len(msg.Series))
	}
	for _, m := range msg.Series {
		if len(m.Revenue) != 0 || m.ActiveSubscribers != 0 || m.NewSubscribers != 0 {
			t.Fatalf("zero-traffic month not empty: %+v", m)
		}
	}
	if msg.Tiles.LatestPeriod != "" || len(msg.Tiles.RevenueThisMonth) != 0 {
		t.Fatalf("zero-traffic tiles: %+v", msg.Tiles)
	}

	if _, err := e.analytics().GetSubscriptionStats(ctx, connect.NewRequest(&adminv1.GetSubscriptionStatsRequest{ProjectId: "nope"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("unknown project: %v", err)
	}
}
