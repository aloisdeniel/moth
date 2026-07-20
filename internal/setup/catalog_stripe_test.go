package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// stripeDouble stands in for the Stripe REST API surfaces the catalog sync
// touches (Products, Prices, Webhook Endpoints), holding in-memory state so a
// second Sync run sees the first run's writes (idempotency). Like Stripe, the
// webhook endpoint signing secret is returned only on create, never on list.
type stripeDouble struct {
	srv *httptest.Server

	mu        sync.Mutex
	products  map[string]map[string]string // id -> {name, metadata...}
	prices    map[string]stripeDoublePrice
	endpoints []map[string]any
	seq       int
	lastAuth  string
	posts     []string // method+path log
	// listPageSize caps the webhook-endpoint list page (0 = everything in one
	// page), so tests can prove the client follows has_more/starting_after.
	listPageSize int
	// failPriceCreate makes POST /v1/prices answer 500, simulating a
	// price-stage provisioning failure after the Product was created.
	failPriceCreate bool
}

type stripeDoublePrice struct {
	Product       string
	UnitAmount    int64
	Currency      string
	Interval      string
	IntervalCount int
	Active        bool
	Metadata      map[string]string
}

func (d *stripeDouble) nextID(prefix string) string {
	d.seq++
	return fmt.Sprintf("%s_%d", prefix, d.seq)
}

func (d *stripeDouble) priceJSON(id string, p stripeDoublePrice) map[string]any {
	return map[string]any{
		"id": id, "product": p.Product, "unit_amount": p.UnitAmount,
		"currency": p.Currency, "active": p.Active,
		"recurring": map[string]any{"interval": p.Interval, "interval_count": p.IntervalCount},
	}
}

func newStripeDouble(t *testing.T) *stripeDouble {
	t.Helper()
	d := &stripeDouble{products: map[string]map[string]string{}, prices: map[string]stripeDoublePrice{}}
	mux := http.NewServeMux()
	writeJSON := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
	notFound := func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"No such resource","type":"invalid_request_error","code":"resource_missing"}}`))
	}
	record := func(r *http.Request) {
		d.lastAuth = r.Header.Get("Authorization")
		d.posts = append(d.posts, r.Method+" "+r.URL.Path)
	}

	mux.HandleFunc("POST /v1/products", func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		defer d.mu.Unlock()
		record(r)
		_ = r.ParseForm()
		id := d.nextID("prod")
		attrs := map[string]string{"name": r.PostForm.Get("name")}
		for k, vs := range r.PostForm {
			if strings.HasPrefix(k, "metadata[") && len(vs) > 0 {
				attrs[k] = vs[0]
			}
		}
		d.products[id] = attrs
		writeJSON(w, map[string]any{"id": id, "name": attrs["name"]})
	})
	mux.HandleFunc("GET /v1/products/{id}", func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		defer d.mu.Unlock()
		record(r)
		id := r.PathValue("id")
		attrs, ok := d.products[id]
		if !ok {
			notFound(w)
			return
		}
		writeJSON(w, map[string]any{"id": id, "name": attrs["name"]})
	})
	mux.HandleFunc("POST /v1/products/{id}", func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		defer d.mu.Unlock()
		record(r)
		_ = r.ParseForm()
		id := r.PathValue("id")
		attrs, ok := d.products[id]
		if !ok {
			notFound(w)
			return
		}
		if name := r.PostForm.Get("name"); name != "" {
			attrs["name"] = name
		}
		writeJSON(w, map[string]any{"id": id, "name": attrs["name"]})
	})
	mux.HandleFunc("POST /v1/prices", func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		defer d.mu.Unlock()
		record(r)
		if d.failPriceCreate {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"message":"price creation exploded","type":"api_error"}}`))
			return
		}
		_ = r.ParseForm()
		amount, _ := strconv.ParseInt(r.PostForm.Get("unit_amount"), 10, 64)
		count, _ := strconv.Atoi(r.PostForm.Get("recurring[interval_count]"))
		if count == 0 {
			count = 1
		}
		id := d.nextID("price")
		p := stripeDoublePrice{
			Product:       r.PostForm.Get("product"),
			UnitAmount:    amount,
			Currency:      r.PostForm.Get("currency"),
			Interval:      r.PostForm.Get("recurring[interval]"),
			IntervalCount: count,
			Active:        true,
		}
		d.prices[id] = p
		writeJSON(w, d.priceJSON(id, p))
	})
	mux.HandleFunc("GET /v1/prices/{id}", func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		defer d.mu.Unlock()
		record(r)
		id := r.PathValue("id")
		p, ok := d.prices[id]
		if !ok {
			notFound(w)
			return
		}
		writeJSON(w, d.priceJSON(id, p))
	})
	mux.HandleFunc("GET /v1/webhook_endpoints", func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		defer d.mu.Unlock()
		record(r)
		// Stripe never returns the signing secret on list — only on create.
		list := make([]map[string]any, 0, len(d.endpoints))
		for _, ep := range d.endpoints {
			listed := map[string]any{}
			for k, v := range ep {
				if k != "secret" {
					listed[k] = v
				}
			}
			list = append(list, listed)
		}
		// Page like Stripe: start after the ?starting_after= id, cap the page,
		// and report has_more so the client must follow the cursor.
		start := 0
		if after := r.URL.Query().Get("starting_after"); after != "" {
			for i, ep := range list {
				if ep["id"] == after {
					start = i + 1
					break
				}
			}
		}
		list = list[start:]
		hasMore := false
		if d.listPageSize > 0 && len(list) > d.listPageSize {
			list = list[:d.listPageSize]
			hasMore = true
		}
		writeJSON(w, map[string]any{"data": list, "has_more": hasMore})
	})
	mux.HandleFunc("POST /v1/webhook_endpoints/{id}", func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		defer d.mu.Unlock()
		record(r)
		_ = r.ParseForm()
		for _, ep := range d.endpoints {
			if ep["id"] != r.PathValue("id") {
				continue
			}
			if r.PostForm.Get("disabled") == "false" {
				ep["status"] = "enabled"
			}
			if events := r.PostForm["enabled_events[]"]; len(events) > 0 {
				ep["enabled_events"] = events
			}
			// Like Stripe, an update never returns the signing secret.
			updated := map[string]any{}
			for k, v := range ep {
				if k != "secret" {
					updated[k] = v
				}
			}
			writeJSON(w, updated)
			return
		}
		notFound(w)
	})
	mux.HandleFunc("POST /v1/webhook_endpoints", func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		defer d.mu.Unlock()
		record(r)
		_ = r.ParseForm()
		id := d.nextID("we")
		ep := map[string]any{
			"id": id, "url": r.PostForm.Get("url"), "status": "enabled",
			"enabled_events": r.PostForm["enabled_events[]"],
			"secret":         "whsec_" + id,
		}
		d.endpoints = append(d.endpoints, ep)
		writeJSON(w, ep)
	})
	d.srv = httptest.NewServer(mux)
	t.Cleanup(d.srv.Close)
	return d
}

func newStripeCatalog(d *stripeDouble) *StripeCatalog {
	return &StripeCatalog{BaseURL: d.srv.URL, SecretKey: "sk_test_moth", HTTPC: http.DefaultClient}
}

// stripeTestCatalog is testCatalog's Stripe face: the tier keys on moth's
// identifier (Stripe ids are generated, not authored).
func stripeTestCatalog() DesiredCatalog {
	return DesiredCatalog{
		GroupReference: "moth-demo",
		Tiers: []DesiredTier{{
			ProductID: "monthly", Reference: "monthly", DisplayName: "Pro",
			Description: "All features", Period: PeriodMonthly,
			Price: Money{Currency: "USD", Micros: 9_990_000}, Locale: "en-US", GroupLevel: 1,
		}},
	}
}

func TestStripeCatalogSyncCreatesAndIsIdempotent(t *testing.T) {
	d := newStripeDouble(t)
	c := newStripeCatalog(d)
	cat := stripeTestCatalog()

	res, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if d.lastAuth != "Bearer sk_test_moth" {
		t.Fatalf("auth = %q", d.lastAuth)
	}
	if len(res.Products) != 1 || res.Products[0].Action != ActionCreated {
		t.Fatalf("products = %+v", res.Products)
	}
	pr := res.Products[0]
	if pr.ProductID != "monthly" || !strings.HasPrefix(pr.StoreID, "price_") || !strings.HasPrefix(pr.StoreParentID, "prod_") {
		t.Fatalf("ids not returned for write-back: %+v", pr)
	}
	// The created price carries the exact cent conversion and cadence.
	price := d.prices[pr.StoreID]
	if price.UnitAmount != 999 || price.Currency != "usd" || price.Interval != "month" || price.IntervalCount != 1 {
		t.Fatalf("price = %+v", price)
	}
	if !price.Active {
		t.Fatal("created price not active")
	}
	// The product carries the moth identity metadata.
	prod := d.products[pr.StoreParentID]
	if prod["metadata[moth_product]"] != "monthly" || prod["metadata[moth_project]"] != "demo" {
		t.Fatalf("product metadata = %+v", prod)
	}

	// Second run with the ids written back: unchanged, nothing created.
	cat.Tiers[0].StripePriceID = pr.StoreID
	cat.Tiers[0].StripeProductID = pr.StoreParentID
	before := len(d.prices)
	postsBefore := len(d.posts)
	res2, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync 2: %v", err)
	}
	if res2.Products[0].Action != ActionUnchanged || res2.Changed() {
		t.Fatalf("second run = %+v changed=%v", res2.Products, res2.Changed())
	}
	if len(d.prices) != before {
		t.Fatalf("second run created prices: %d -> %d", before, len(d.prices))
	}
	// An unchanged tier is read-only: no UpdateProduct (or any POST) issued.
	for _, call := range d.posts[postsBefore:] {
		if strings.HasPrefix(call, "POST ") {
			t.Fatalf("unchanged run issued a write: %s", call)
		}
	}
}

func TestStripeCatalogPriceDriftCreatesNewPriceAndRepoints(t *testing.T) {
	d := newStripeDouble(t)
	c := newStripeCatalog(d)
	cat := stripeTestCatalog()
	res, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	oldPrice := res.Products[0].StoreID
	cat.Tiers[0].StripePriceID = oldPrice
	cat.Tiers[0].StripeProductID = res.Products[0].StoreParentID

	// Price edit in moth: Stripe prices are immutable, so the sync must create
	// a NEW price on the same product and re-point the tier.
	cat.Tiers[0].Price.Micros = 4_990_000
	res2, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync 2: %v", err)
	}
	pr := res2.Products[0]
	if pr.Action != ActionUpdated {
		t.Fatalf("action = %q, want updated", pr.Action)
	}
	if pr.StoreID == oldPrice || !strings.HasPrefix(pr.StoreID, "price_") {
		t.Fatalf("expected a new price id, got %q (old %q)", pr.StoreID, oldPrice)
	}
	if d.prices[pr.StoreID].UnitAmount != 499 {
		t.Fatalf("new price amount = %d", d.prices[pr.StoreID].UnitAmount)
	}
	// The old price still exists (existing subscribers keep it) and the detail
	// says so — the plan's honesty requirement.
	if _, ok := d.prices[oldPrice]; !ok {
		t.Fatal("old price was removed")
	}
	if !strings.Contains(pr.Detail, "existing subscribers keep "+oldPrice) {
		t.Fatalf("detail does not surface the immutability semantics: %q", pr.Detail)
	}
	// Same product, no second Stripe Product created.
	if len(d.products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(d.products))
	}
}

func TestStripeCatalogNameDriftRenamesProductInPlace(t *testing.T) {
	d := newStripeDouble(t)
	c := newStripeCatalog(d)
	cat := stripeTestCatalog()
	res, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	cat.Tiers[0].StripePriceID = res.Products[0].StoreID
	cat.Tiers[0].StripeProductID = res.Products[0].StoreParentID
	productID := res.Products[0].StoreParentID

	// Rename in moth only: product names are mutable in Stripe, so the sync
	// renames the Product in place — no new Price, no new Product.
	cat.Tiers[0].DisplayName = "Pro Plus"
	pricesBefore := len(d.prices)
	res2, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync 2: %v", err)
	}
	pr := res2.Products[0]
	if pr.Action != ActionUpdated {
		t.Fatalf("action = %q, want updated", pr.Action)
	}
	if pr.StoreID != cat.Tiers[0].StripePriceID {
		t.Fatalf("name-only drift re-pointed the price: %q -> %q", cat.Tiers[0].StripePriceID, pr.StoreID)
	}
	if len(d.prices) != pricesBefore {
		t.Fatalf("name-only drift created prices: %d -> %d", pricesBefore, len(d.prices))
	}
	if got := d.products[productID]["name"]; got != "Pro Plus" {
		t.Fatalf("stripe product name = %q, want renamed", got)
	}
	if !strings.Contains(pr.Detail, "renamed product "+productID) || !strings.Contains(pr.Detail, `"Pro Plus"`) {
		t.Fatalf("detail does not surface the rename: %q", pr.Detail)
	}
	if strings.Contains(pr.Detail, "created new Price") {
		t.Fatalf("name-only drift reported a price change: %q", pr.Detail)
	}

	// Third run: the rename converged, nothing further to do.
	res3, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync 3: %v", err)
	}
	if res3.Products[0].Action != ActionUnchanged {
		t.Fatalf("post-rename run = %+v", res3.Products[0])
	}
}

func TestStripeCatalogNameAndPriceDriftReportsBoth(t *testing.T) {
	d := newStripeDouble(t)
	c := newStripeCatalog(d)
	cat := stripeTestCatalog()
	res, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	oldPrice := res.Products[0].StoreID
	productID := res.Products[0].StoreParentID
	cat.Tiers[0].StripePriceID = oldPrice
	cat.Tiers[0].StripeProductID = productID

	cat.Tiers[0].DisplayName = "Pro Plus"
	cat.Tiers[0].Price.Micros = 4_990_000
	res2, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync 2: %v", err)
	}
	pr := res2.Products[0]
	if pr.Action != ActionUpdated {
		t.Fatalf("action = %q, want updated", pr.Action)
	}
	// Rename in place AND a new immutable Price, both on the same Product.
	if got := d.products[productID]["name"]; got != "Pro Plus" {
		t.Fatalf("stripe product name = %q, want renamed", got)
	}
	if pr.StoreID == oldPrice || d.prices[pr.StoreID].UnitAmount != 499 {
		t.Fatalf("price not re-pointed: %+v (old %q)", pr, oldPrice)
	}
	if pr.StoreParentID != productID || len(d.products) != 1 {
		t.Fatalf("product identity changed: %+v products=%d", pr, len(d.products))
	}
	if !strings.Contains(pr.Detail, "renamed product "+productID) ||
		!strings.Contains(pr.Detail, "existing subscribers keep "+oldPrice) {
		t.Fatalf("detail does not report both changes: %q", pr.Detail)
	}
}

func TestStripeCatalogRecreatesMissingProduct(t *testing.T) {
	d := newStripeDouble(t)
	c := newStripeCatalog(d)
	cat := stripeTestCatalog()
	res, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	oldProduct := res.Products[0].StoreParentID
	cat.Tiers[0].StripePriceID = res.Products[0].StoreID
	cat.Tiers[0].StripeProductID = oldProduct

	// The product vanished from Stripe while its recorded price id still
	// resolves: recreate the tier on a fresh Product.
	d.mu.Lock()
	delete(d.products, oldProduct)
	d.mu.Unlock()
	res2, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync 2: %v", err)
	}
	pr := res2.Products[0]
	if pr.Action != ActionCreated || pr.StoreParentID == oldProduct {
		t.Fatalf("expected recreation on a fresh product, got %+v", pr)
	}
	if !strings.Contains(pr.Detail, oldProduct+" no longer exists") {
		t.Fatalf("detail = %q", pr.Detail)
	}
}

func TestStripeCatalogRecreatesMissingPrice(t *testing.T) {
	d := newStripeDouble(t)
	c := newStripeCatalog(d)
	cat := stripeTestCatalog()
	// Recorded price id no longer exists; recorded product does not either.
	cat.Tiers[0].StripePriceID = "price_gone"
	res, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	pr := res.Products[0]
	if pr.Action != ActionCreated || pr.StoreID == "price_gone" {
		t.Fatalf("expected recreation, got %+v", pr)
	}
	if !strings.Contains(pr.Detail, "price_gone no longer exists") {
		t.Fatalf("detail = %q", pr.Detail)
	}

	// With a recorded product id in hand, only the price is recreated.
	d2 := newStripeDouble(t)
	c2 := newStripeCatalog(d2)
	cat2 := stripeTestCatalog()
	cat2.Tiers[0].StripePriceID = "price_gone"
	cat2.Tiers[0].StripeProductID = "prod_keep"
	res2, err := c2.Sync(context.Background(), cat2)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if res2.Products[0].StoreParentID != "prod_keep" {
		t.Fatalf("expected the recorded product to be reused, got %+v", res2.Products[0])
	}
	if len(d2.products) != 0 {
		t.Fatalf("a new Stripe Product was created despite a recorded one: %+v", d2.products)
	}
}

func TestStripeCatalogUnrepresentablePriceFailsTierOnly(t *testing.T) {
	d := newStripeDouble(t)
	c := newStripeCatalog(d)
	cat := stripeTestCatalog()
	cat.Tiers = append(cat.Tiers, DesiredTier{
		ProductID: "odd", Reference: "odd", DisplayName: "Odd", Period: PeriodMonthly,
		// 9.9955 USD is not cent-representable: must fail loudly, not round.
		Price: Money{Currency: "USD", Micros: 9_995_500},
	})
	res, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if res.Products[0].Action != ActionCreated {
		t.Fatalf("good tier should still sync: %+v", res.Products[0])
	}
	bad := res.Products[1]
	if bad.Action != ActionFailed || !strings.Contains(bad.Detail, "minor unit") {
		t.Fatalf("bad tier = %+v", bad)
	}
}

// TestStripeCatalogCurrencyAwareUnitAmounts proves the micros → unit_amount
// conversion respects the currency's minor unit: JPY (zero-decimal) sells
// ¥500 as unit_amount 500 — a hard-coded /10_000 would provision ¥50,000, a
// 100x overcharge — and KWD (three-decimal) uses thousandths.
func TestStripeCatalogCurrencyAwareUnitAmounts(t *testing.T) {
	d := newStripeDouble(t)
	c := newStripeCatalog(d)
	cat := DesiredCatalog{
		GroupReference: "moth-demo",
		Tiers: []DesiredTier{
			{ProductID: "jpy", Reference: "jpy", DisplayName: "Pro JP", Period: PeriodMonthly,
				Price: Money{Currency: "JPY", Micros: 500_000_000}}, // ¥500
			{ProductID: "kwd", Reference: "kwd", DisplayName: "Pro KW", Period: PeriodMonthly,
				Price: Money{Currency: "KWD", Micros: 1_234_000}}, // 1.234 KWD
		},
	}
	res, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	byID := map[string]ProductResult{}
	for _, pr := range res.Products {
		if pr.Action != ActionCreated {
			t.Fatalf("tier %s = %+v, want created", pr.ProductID, pr)
		}
		byID[pr.ProductID] = pr
	}
	if got := d.prices[byID["jpy"].StoreID].UnitAmount; got != 500 {
		t.Fatalf("JPY unit_amount = %d, want 500 (whole yen)", got)
	}
	if got := d.prices[byID["kwd"].StoreID].UnitAmount; got != 1234 {
		t.Fatalf("KWD unit_amount = %d, want 1234 (thousandths)", got)
	}

	// Drift check: with the ids written back, the JPY tier reads back as
	// unchanged — the compare uses the same per-currency conversion, so an
	// in-sync price never re-provisions (no false 100x drift).
	cat.Tiers[0].StripePriceID = byID["jpy"].StoreID
	cat.Tiers[0].StripeProductID = byID["jpy"].StoreParentID
	cat.Tiers[1].StripePriceID = byID["kwd"].StoreID
	cat.Tiers[1].StripeProductID = byID["kwd"].StoreParentID
	pricesBefore := len(d.prices)
	res2, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync 2: %v", err)
	}
	for _, pr := range res2.Products {
		if pr.Action != ActionUnchanged {
			t.Fatalf("second run tier %s = %+v, want unchanged", pr.ProductID, pr)
		}
	}
	if len(d.prices) != pricesBefore {
		t.Fatalf("stable currency drift check created prices: %d -> %d", pricesBefore, len(d.prices))
	}
}

// TestStripeCatalogUnrepresentableJPYFraction: ¥500.50 has no minor unit in
// zero-decimal JPY — the tier must fail loudly, never round.
func TestStripeCatalogUnrepresentableJPYFraction(t *testing.T) {
	d := newStripeDouble(t)
	c := newStripeCatalog(d)
	cat := DesiredCatalog{
		GroupReference: "moth-demo",
		Tiers: []DesiredTier{{ProductID: "jpy", Reference: "jpy", DisplayName: "Pro JP",
			Period: PeriodMonthly, Price: Money{Currency: "JPY", Micros: 500_500_000}}},
	}
	res, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	pr := res.Products[0]
	if pr.Action != ActionFailed || !strings.Contains(pr.Detail, "JPY") || !strings.Contains(pr.Detail, "minor unit") {
		t.Fatalf("tier = %+v, want failed with a JPY minor-unit detail", pr)
	}
	if len(d.products) != 0 || len(d.prices) != 0 {
		t.Fatalf("failed tier still provisioned: products=%d prices=%d", len(d.products), len(d.prices))
	}
}

// TestStripeCatalogPriceCreateFailureRecordsProduct: when the Price create
// fails after the Product create, the tier fails with the Product id carried
// in StoreParentID (write-back records it), and a re-run reuses that Product
// instead of provisioning a duplicate.
func TestStripeCatalogPriceCreateFailureRecordsProduct(t *testing.T) {
	d := newStripeDouble(t)
	c := newStripeCatalog(d)
	cat := stripeTestCatalog()
	d.failPriceCreate = true

	res, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync must fail per-tier, not hard: %v", err)
	}
	pr := res.Products[0]
	if pr.Action != ActionFailed || pr.StoreID != "" {
		t.Fatalf("tier = %+v, want failed with no price id", pr)
	}
	if !strings.HasPrefix(pr.StoreParentID, "prod_") {
		t.Fatalf("created Product id not carried for write-back: %+v", pr)
	}
	if !strings.Contains(pr.Detail, pr.StoreParentID) || !strings.Contains(pr.Detail, "re-run reuses it") {
		t.Fatalf("detail = %q", pr.Detail)
	}

	// Re-run with the product id recorded (as write-back would): the existing
	// Product is reused — no duplicate — and the Price is created.
	d.failPriceCreate = false
	cat.Tiers[0].StripeProductID = pr.StoreParentID
	res2, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync 2: %v", err)
	}
	pr2 := res2.Products[0]
	if pr2.Action != ActionCreated || pr2.StoreParentID != pr.StoreParentID {
		t.Fatalf("re-run = %+v, want created on product %s", pr2, pr.StoreParentID)
	}
	if len(d.products) != 1 {
		t.Fatalf("re-run provisioned a duplicate Product: %d", len(d.products))
	}
	if d.prices[pr2.StoreID].Product != pr.StoreParentID {
		t.Fatalf("price hangs off %q, want %q", d.prices[pr2.StoreID].Product, pr.StoreParentID)
	}
}

func TestStripeEnsureWebhookEndpointIdempotent(t *testing.T) {
	d := newStripeDouble(t)
	c := newStripeCatalog(d)
	url := "https://moth.example.com/billing/stripe/webhook/demo"

	ep, created, repaired, err := c.EnsureWebhookEndpoint(context.Background(), url)
	if err != nil {
		t.Fatalf("EnsureWebhookEndpoint: %v", err)
	}
	if !created || repaired || ep.Secret == "" || ep.URL != url {
		t.Fatalf("create = %+v created=%v repaired=%v", ep, created, repaired)
	}
	// The endpoint subscribes exactly the moth event set.
	if len(ep.EnabledEvents) != len(StripeWebhookEvents) {
		t.Fatalf("events = %v", ep.EnabledEvents)
	}

	// Second call: found by exact URL (a real read+diff), no create, and —
	// like Stripe — no secret on a listed endpoint. An intact endpoint
	// (enabled, full moth event set) is left untouched: no POST issued.
	postsBefore := len(d.posts)
	ep2, created2, repaired2, err := c.EnsureWebhookEndpoint(context.Background(), url)
	if err != nil {
		t.Fatalf("EnsureWebhookEndpoint 2: %v", err)
	}
	if created2 || repaired2 || ep2.ID != ep.ID || ep2.Secret != "" {
		t.Fatalf("second call = %+v created=%v repaired=%v", ep2, created2, repaired2)
	}
	if len(d.endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(d.endpoints))
	}
	for _, call := range d.posts[postsBefore:] {
		if strings.HasPrefix(call, "POST ") {
			t.Fatalf("intact endpoint was written to: %s", call)
		}
	}
}

// TestStripeEnsureWebhookEndpointRepairsDisabled: an endpoint Stripe disabled
// (too many failed deliveries) must not pass as registered — it is re-enabled
// in place and reported as repaired.
func TestStripeEnsureWebhookEndpointRepairsDisabled(t *testing.T) {
	d := newStripeDouble(t)
	c := newStripeCatalog(d)
	url := "https://moth.example.com/billing/stripe/webhook/demo"
	d.endpoints = append(d.endpoints, map[string]any{
		"id": "we_disabled", "url": url, "status": "disabled",
		"enabled_events": StripeWebhookEvents,
	})

	ep, created, repaired, err := c.EnsureWebhookEndpoint(context.Background(), url)
	if err != nil {
		t.Fatalf("EnsureWebhookEndpoint: %v", err)
	}
	if created || !repaired || ep.ID != "we_disabled" || ep.Secret != "" {
		t.Fatalf("repair = %+v created=%v repaired=%v", ep, created, repaired)
	}
	if ep.Status != "enabled" || d.endpoints[0]["status"] != "enabled" {
		t.Fatalf("endpoint not re-enabled: %+v / %+v", ep, d.endpoints[0])
	}
	if len(d.endpoints) != 1 {
		t.Fatalf("repair created an endpoint: %d", len(d.endpoints))
	}
}

// TestStripeEnsureWebhookEndpointRepairsMissingEvents: an endpoint missing any
// of moth's subscription events would silently drop deliveries — its event set
// is updated in place (extra events beyond moth's set are fine and untouched).
func TestStripeEnsureWebhookEndpointRepairsMissingEvents(t *testing.T) {
	d := newStripeDouble(t)
	c := newStripeCatalog(d)
	url := "https://moth.example.com/billing/stripe/webhook/demo"
	d.endpoints = append(d.endpoints, map[string]any{
		"id": "we_partial", "url": url, "status": "enabled",
		// customer.subscription.deleted missing: expirations would never arrive.
		"enabled_events": []string{"checkout.session.completed", "customer.subscription.created", "customer.subscription.updated"},
	})

	ep, created, repaired, err := c.EnsureWebhookEndpoint(context.Background(), url)
	if err != nil {
		t.Fatalf("EnsureWebhookEndpoint: %v", err)
	}
	if created || !repaired || ep.ID != "we_partial" {
		t.Fatalf("repair = %+v created=%v repaired=%v", ep, created, repaired)
	}
	if !hasAllEvents(ep.EnabledEvents, StripeWebhookEvents) {
		t.Fatalf("events not repaired: %v", ep.EnabledEvents)
	}

	// An endpoint with EXTRA events on top of moth's set is fine: untouched.
	d2 := newStripeDouble(t)
	c2 := newStripeCatalog(d2)
	d2.endpoints = append(d2.endpoints, map[string]any{
		"id": "we_extra", "url": url, "status": "enabled",
		"enabled_events": append([]string{"invoice.paid"}, StripeWebhookEvents...),
	})
	postsBefore := len(d2.posts)
	_, created2, repaired2, err := c2.EnsureWebhookEndpoint(context.Background(), url)
	if err != nil {
		t.Fatalf("EnsureWebhookEndpoint extra: %v", err)
	}
	if created2 || repaired2 {
		t.Fatalf("superset endpoint should be untouched: created=%v repaired=%v", created2, repaired2)
	}
	for _, call := range d2.posts[postsBefore:] {
		if strings.HasPrefix(call, "POST ") {
			t.Fatalf("superset endpoint was written to: %s", call)
		}
	}
}

// TestStripeEnsureWebhookEndpointPaginatedList: Stripe pages the endpoint list;
// an instance with more endpoints than one page must still find moth's endpoint
// on a later page instead of creating a duplicate (and rotating the secret).
func TestStripeEnsureWebhookEndpointPaginatedList(t *testing.T) {
	d := newStripeDouble(t)
	c := newStripeCatalog(d)
	url := "https://moth.example.com/billing/stripe/webhook/demo"
	d.listPageSize = 5
	for i := 0; i < 11; i++ {
		d.endpoints = append(d.endpoints, map[string]any{
			"id": fmt.Sprintf("we_other_%d", i), "url": fmt.Sprintf("https://other.example.com/hook/%d", i),
			"status": "enabled", "enabled_events": StripeWebhookEvents,
		})
	}
	// moth's endpoint is the 12th — beyond the first two pages.
	d.endpoints = append(d.endpoints, map[string]any{
		"id": "we_moth", "url": url, "status": "enabled", "enabled_events": StripeWebhookEvents,
	})

	ep, created, repaired, err := c.EnsureWebhookEndpoint(context.Background(), url)
	if err != nil {
		t.Fatalf("EnsureWebhookEndpoint: %v", err)
	}
	if created || repaired || ep.ID != "we_moth" {
		t.Fatalf("endpoint on a later page missed: %+v created=%v repaired=%v", ep, created, repaired)
	}
	if len(d.endpoints) != 12 {
		t.Fatalf("a duplicate endpoint was created: %d", len(d.endpoints))
	}
}
