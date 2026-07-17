package billing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

// stripeDouble is an httptest stand-in for the Stripe REST API, serving every
// endpoint the client calls on one handler. It records the last request shape
// and the ParseForm-parsed body so tests can assert Stripe's bracketed
// form-encoding exactly.
type stripeDouble struct {
	srv *httptest.Server

	subBody     string // GET /v1/subscriptions/{id}; sentinels __404__ / __err__
	priceBody   string // GET /v1/prices/{id}
	productBody string // GET /v1/products/{id}; sentinel __404__

	lastAuth        string
	lastPath        string
	lastMethod      string
	lastContentType string
	lastForm        url.Values
}

func newStripeDouble(t *testing.T) *stripeDouble {
	t.Helper()
	d := &stripeDouble{}
	d.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d.lastAuth = r.Header.Get("Authorization")
		d.lastPath = r.URL.Path
		d.lastMethod = r.Method
		d.lastContentType = r.Header.Get("Content-Type")
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "bad form", http.StatusBadRequest)
				return
			}
			d.lastForm = r.PostForm
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/customers":
			fmt.Fprintf(w, `{"id":"cus_123","object":"customer","email":%q}`, r.PostForm.Get("email"))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/checkout/sessions":
			_, _ = io.WriteString(w, `{"id":"cs_test_1","object":"checkout.session","url":"https://checkout.stripe.com/c/pay/cs_test_1"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/billing_portal/sessions":
			_, _ = io.WriteString(w, `{"id":"bps_1","object":"billing_portal.session","url":"https://billing.stripe.com/p/session/bps_1"}`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/subscriptions/"):
			switch d.subBody {
			case "__404__":
				w.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(w, `{"error":{"message":"No such subscription: 'sub_x'","type":"invalid_request_error","code":"resource_missing"}}`)
			case "__err__":
				w.WriteHeader(http.StatusPaymentRequired)
				_, _ = io.WriteString(w, `{"error":{"message":"Your card was declined.","type":"card_error","code":"card_declined"}}`)
			default:
				_, _ = io.WriteString(w, d.subBody)
			}
		case r.Method == http.MethodPost && r.URL.Path == "/v1/products":
			fmt.Fprintf(w, `{"id":"prod_1","object":"product","name":%q}`, r.PostForm.Get("name"))
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/products/"):
			fmt.Fprintf(w, `{"id":%q,"object":"product","name":%q}`,
				strings.TrimPrefix(r.URL.Path, "/v1/products/"), r.PostForm.Get("name"))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/products/"):
			if d.productBody == "__404__" {
				w.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(w, `{"error":{"message":"No such product: 'prod_x'","type":"invalid_request_error","code":"resource_missing"}}`)
				return
			}
			_, _ = io.WriteString(w, d.productBody)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/prices":
			count := r.PostForm.Get("recurring[interval_count]")
			if count == "" {
				count = "1"
			}
			fmt.Fprintf(w, `{"id":"price_new","object":"price","product":%q,"unit_amount":%s,"currency":%q,"active":true,"recurring":{"interval":%q,"interval_count":%s}}`,
				r.PostForm.Get("product"), r.PostForm.Get("unit_amount"), r.PostForm.Get("currency"), r.PostForm.Get("recurring[interval]"), count)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/prices/"):
			_, _ = io.WriteString(w, d.priceBody)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/webhook_endpoints":
			_, _ = io.WriteString(w, `{"object":"list","data":[{"id":"we_1","url":"https://moth.example/billing/stripe/webhook/demo","status":"enabled","enabled_events":["checkout.session.completed"]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/webhook_endpoints":
			events, _ := json.Marshal(r.PostForm["enabled_events[]"])
			fmt.Fprintf(w, `{"id":"we_2","url":%q,"status":"enabled","enabled_events":%s,"secret":"whsec_test_secret"}`,
				r.PostForm.Get("url"), events)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/webhook_endpoints/"):
			// Update: the signing secret is deliberately NOT in the response —
			// Stripe reveals it on create only.
			events, _ := json.Marshal(r.PostForm["enabled_events[]"])
			fmt.Fprintf(w, `{"id":%q,"url":"https://moth.example/billing/stripe/webhook/demo","status":"enabled","enabled_events":%s}`,
				strings.TrimPrefix(r.URL.Path, "/v1/webhook_endpoints/"), events)
		default:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"error":{"message":"unexpected call"}}`)
		}
	}))
	t.Cleanup(d.srv.Close)
	return d
}

func (d *stripeDouble) client() *StripeClient {
	return &StripeClient{
		BaseURL:   d.srv.URL,
		SecretKey: "sk_test_123",
		HTTPC:     http.DefaultClient,
		Now:       func() time.Time { return testNow },
	}
}

func TestStripeCreateCustomer(t *testing.T) {
	d := newStripeDouble(t)
	c := d.client()
	got, err := c.CreateCustomer(t.Context(), "user@example.com", map[string]string{"project_id": "p1", "user_id": "u1"})
	if err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	if got.ID != "cus_123" || got.Email != "user@example.com" {
		t.Fatalf("customer = %+v", got)
	}
	if d.lastMethod != http.MethodPost || d.lastPath != "/v1/customers" {
		t.Fatalf("call = %s %s", d.lastMethod, d.lastPath)
	}
	if d.lastAuth != "Bearer sk_test_123" {
		t.Fatalf("auth = %q", d.lastAuth)
	}
	if d.lastContentType != "application/x-www-form-urlencoded" {
		t.Fatalf("content-type = %q", d.lastContentType)
	}
	if d.lastForm.Get("email") != "user@example.com" ||
		d.lastForm.Get("metadata[project_id]") != "p1" ||
		d.lastForm.Get("metadata[user_id]") != "u1" {
		t.Fatalf("form = %v", d.lastForm)
	}
}

func TestStripeCreateCheckoutSessionFormEncoding(t *testing.T) {
	d := newStripeDouble(t)
	c := d.client()
	got, err := c.CreateCheckoutSession(t.Context(), StripeCheckoutParams{
		PriceID:           "price_123",
		CustomerID:        "cus_123",
		SuccessURL:        "https://app.example/done",
		CancelURL:         "https://app.example/cancel",
		ClientReferenceID: "user-42",
		Metadata:          map[string]string{"project_id": "p1", "user_id": "user-42"},
		TrialPeriodDays:   7,
	})
	if err != nil {
		t.Fatalf("CreateCheckoutSession: %v", err)
	}
	if got.ID != "cs_test_1" || got.URL != "https://checkout.stripe.com/c/pay/cs_test_1" {
		t.Fatalf("session = %+v", got)
	}
	want := map[string]string{
		"mode":                    "subscription",
		"line_items[0][price]":    "price_123",
		"line_items[0][quantity]": "1",
		"customer":                "cus_123",
		"success_url":             "https://app.example/done",
		"cancel_url":              "https://app.example/cancel",
		"client_reference_id":     "user-42",
		"metadata[project_id]":    "p1",
		"metadata[user_id]":       "user-42",
		"subscription_data[metadata][project_id]": "p1",
		"subscription_data[metadata][user_id]":    "user-42",
		"subscription_data[trial_period_days]":    "7",
	}
	for k, v := range want {
		if got := d.lastForm.Get(k); got != v {
			t.Errorf("form[%q] = %q, want %q", k, got, v)
		}
	}
}

func TestStripeCheckoutSessionNoTrialOmitsField(t *testing.T) {
	d := newStripeDouble(t)
	c := d.client()
	if _, err := c.CreateCheckoutSession(t.Context(), StripeCheckoutParams{
		PriceID: "price_123", SuccessURL: "https://a/s", CancelURL: "https://a/c",
	}); err != nil {
		t.Fatalf("CreateCheckoutSession: %v", err)
	}
	if _, ok := d.lastForm["subscription_data[trial_period_days]"]; ok {
		t.Fatalf("trial_period_days sent for no-trial session: %v", d.lastForm)
	}
	if _, ok := d.lastForm["customer"]; ok {
		t.Fatalf("empty customer sent: %v", d.lastForm)
	}
}

func TestStripeCreateBillingPortalSession(t *testing.T) {
	d := newStripeDouble(t)
	c := d.client()
	got, err := c.CreateBillingPortalSession(t.Context(), "cus_123", "https://app.example/account")
	if err != nil {
		t.Fatalf("CreateBillingPortalSession: %v", err)
	}
	if got.ID != "bps_1" || got.URL != "https://billing.stripe.com/p/session/bps_1" {
		t.Fatalf("portal session = %+v", got)
	}
	if d.lastPath != "/v1/billing_portal/sessions" ||
		d.lastForm.Get("customer") != "cus_123" ||
		d.lastForm.Get("return_url") != "https://app.example/account" {
		t.Fatalf("call = %s form = %v", d.lastPath, d.lastForm)
	}
}

// stripeSubJSON builds a canned subscription response. periodEndOn selects
// where current_period_end lives: "sub" (older API versions), "item" (current
// versions), "both", or "" for neither.
func stripeSubJSON(t *testing.T, status string, paused, livemode, cancelAtPeriodEnd bool, periodEndOn string, end int64) string {
	t.Helper()
	item := map[string]any{
		"id": "si_1",
		"price": map[string]any{
			"id":          "price_123",
			"product":     "prod_1",
			"unit_amount": 999,
			"currency":    "usd",
			"recurring":   map[string]any{"interval": "month", "interval_count": 1},
		},
	}
	body := map[string]any{
		"id":                   "sub_123",
		"object":               "subscription",
		"status":               status,
		"customer":             "cus_123",
		"livemode":             livemode,
		"cancel_at_period_end": cancelAtPeriodEnd,
		"metadata":             map[string]any{"project_id": "p1", "user_id": "u1"},
		"items":                map[string]any{"object": "list", "data": []any{item}},
	}
	if paused {
		body["pause_collection"] = map[string]any{"behavior": "void"}
	}
	if periodEndOn == "sub" || periodEndOn == "both" {
		body["current_period_end"] = end
	}
	if periodEndOn == "item" || periodEndOn == "both" {
		item["current_period_end"] = end
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal sub body: %v", err)
	}
	return string(raw)
}

func TestStripeGetSubscriptionNormalization(t *testing.T) {
	end := testNow.Add(30 * 24 * time.Hour).Unix()
	cases := []struct {
		name        string
		status      string
		paused      bool
		livemode    bool
		cancel      bool
		periodEndOn string
		wantStatus  string
		wantEnv     string
	}{
		{name: "active", status: "active", periodEndOn: "sub", wantStatus: StatusActive, wantEnv: EnvSandbox},
		{name: "active pause_collection", status: "active", paused: true, periodEndOn: "sub", wantStatus: StatusPaused, wantEnv: EnvSandbox},
		{name: "trialing", status: "trialing", periodEndOn: "item", wantStatus: StatusTrialing, wantEnv: EnvSandbox},
		{name: "past_due", status: "past_due", periodEndOn: "item", wantStatus: StatusInBillingRetry, wantEnv: EnvSandbox},
		{name: "paused", status: "paused", periodEndOn: "sub", wantStatus: StatusPaused, wantEnv: EnvSandbox},
		{name: "canceled", status: "canceled", periodEndOn: "sub", wantStatus: StatusExpired, wantEnv: EnvSandbox},
		{name: "unpaid", status: "unpaid", periodEndOn: "sub", wantStatus: StatusExpired, wantEnv: EnvSandbox},
		{name: "incomplete", status: "incomplete", periodEndOn: "sub", wantStatus: StatusExpired, wantEnv: EnvSandbox},
		{name: "incomplete_expired", status: "incomplete_expired", periodEndOn: "sub", wantStatus: StatusExpired, wantEnv: EnvSandbox},
		{name: "livemode production", status: "active", livemode: true, periodEndOn: "both", wantStatus: StatusActive, wantEnv: EnvProduction},
		{name: "cancel_at_period_end", status: "active", cancel: true, periodEndOn: "item", wantStatus: StatusActive, wantEnv: EnvSandbox},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := newStripeDouble(t)
			d.subBody = stripeSubJSON(t, tc.status, tc.paused, tc.livemode, tc.cancel, tc.periodEndOn, end)
			c := d.client()

			norm, raw, err := c.GetSubscription(t.Context(), "sub_123")
			if err != nil {
				t.Fatalf("GetSubscription: %v", err)
			}
			if d.lastMethod != http.MethodGet || d.lastPath != "/v1/subscriptions/sub_123" {
				t.Fatalf("call = %s %s", d.lastMethod, d.lastPath)
			}
			if d.lastAuth != "Bearer sk_test_123" {
				t.Fatalf("auth = %q", d.lastAuth)
			}
			if norm.Status != tc.wantStatus {
				t.Fatalf("status %s -> %q, want %q", tc.status, norm.Status, tc.wantStatus)
			}
			if norm.Environment != tc.wantEnv {
				t.Fatalf("environment = %q, want %q", norm.Environment, tc.wantEnv)
			}
			if norm.Store != StoreStripe || norm.StoreTransactionID != "sub_123" || norm.SubscriptionID != "cus_123" {
				t.Fatalf("identity = %+v", norm)
			}
			if norm.ProductID != "price_123" {
				t.Fatalf("product id = %q, want price id", norm.ProductID)
			}
			if norm.PriceAmountMicros != 9_990_000 {
				t.Fatalf("micros = %d, want 9990000 (999 cents)", norm.PriceAmountMicros)
			}
			if norm.Currency != "USD" {
				t.Fatalf("currency = %q, want USD", norm.Currency)
			}
			if norm.AutoRenew != !tc.cancel {
				t.Fatalf("auto renew = %v with cancel_at_period_end = %v", norm.AutoRenew, tc.cancel)
			}
			if want := time.Unix(end, 0).UTC(); !norm.CurrentPeriodEnd.Equal(want) {
				t.Fatalf("period end (%s) = %v, want %v", tc.periodEndOn, norm.CurrentPeriodEnd, want)
			}
			if string(norm.RawState) != d.subBody {
				t.Fatalf("raw state not verbatim response body:\n%s", norm.RawState)
			}
			if raw.ID != "sub_123" || raw.Status != tc.status || raw.Metadata["project_id"] != "p1" {
				t.Fatalf("raw = %+v", raw)
			}
			if len(raw.Items.Data) != 1 || raw.Items.Data[0].Price.Recurring.Interval != "month" {
				t.Fatalf("raw items = %+v", raw.Items)
			}
		})
	}
}

func TestStripeGetSubscriptionNotFound(t *testing.T) {
	d := newStripeDouble(t)
	d.subBody = "__404__"
	c := d.client()
	_, _, err := c.GetSubscription(t.Context(), "sub_missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
	if !strings.Contains(err.Error(), "No such subscription") {
		t.Fatalf("error lacks stripe message: %v", err)
	}
}

func TestStripeErrorMessageExtraction(t *testing.T) {
	d := newStripeDouble(t)
	d.subBody = "__err__"
	c := d.client()
	_, _, err := c.GetSubscription(t.Context(), "sub_123")
	if err == nil {
		t.Fatal("expected error on 402")
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("402 mapped to ErrNotFound: %v", err)
	}
	for _, want := range []string{"status 402", "Your card was declined.", "card_error", "card_declined"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err.Error(), want)
		}
	}
}

func TestStripeCreateProductAndPrice(t *testing.T) {
	d := newStripeDouble(t)
	c := d.client()

	prod, err := c.CreateProduct(t.Context(), "Pro Monthly", map[string]string{"moth_product": "pro.monthly"})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	if prod.ID != "prod_1" || prod.Name != "Pro Monthly" {
		t.Fatalf("product = %+v", prod)
	}
	if d.lastForm.Get("name") != "Pro Monthly" || d.lastForm.Get("metadata[moth_product]") != "pro.monthly" {
		t.Fatalf("product form = %v", d.lastForm)
	}

	price, err := c.CreatePrice(t.Context(), StripePriceParams{
		ProductID:     "prod_1",
		Currency:      "USD",
		UnitAmount:    499,
		Interval:      "month",
		IntervalCount: 3,
		Metadata:      map[string]string{"moth_product": "pro.monthly"},
	})
	if err != nil {
		t.Fatalf("CreatePrice: %v", err)
	}
	if d.lastPath != "/v1/prices" ||
		d.lastForm.Get("product") != "prod_1" ||
		d.lastForm.Get("currency") != "usd" || // Stripe wants lowercase on the wire
		d.lastForm.Get("unit_amount") != "499" ||
		d.lastForm.Get("recurring[interval]") != "month" ||
		d.lastForm.Get("recurring[interval_count]") != "3" ||
		d.lastForm.Get("metadata[moth_product]") != "pro.monthly" {
		t.Fatalf("price form = %v", d.lastForm)
	}
	want := StripePrice{ID: "price_new", ProductID: "prod_1", UnitAmount: 499, Currency: "USD", Interval: "month", IntervalCount: 3, Active: true}
	if price != want {
		t.Fatalf("price = %+v, want %+v", price, want)
	}
}

func TestStripeGetPrice(t *testing.T) {
	d := newStripeDouble(t)
	d.priceBody = `{"id":"price_123","object":"price","product":"prod_1","unit_amount":999,"currency":"usd","active":true,"recurring":{"interval":"year","interval_count":1}}`
	c := d.client()
	price, err := c.GetPrice(t.Context(), "price_123")
	if err != nil {
		t.Fatalf("GetPrice: %v", err)
	}
	if d.lastMethod != http.MethodGet || d.lastPath != "/v1/prices/price_123" {
		t.Fatalf("call = %s %s", d.lastMethod, d.lastPath)
	}
	want := StripePrice{ID: "price_123", ProductID: "prod_1", UnitAmount: 999, Currency: "USD", Interval: "year", IntervalCount: 1, Active: true}
	if price != want {
		t.Fatalf("price = %+v, want %+v", price, want)
	}
}

func TestStripeGetProduct(t *testing.T) {
	d := newStripeDouble(t)
	d.productBody = `{"id":"prod_1","object":"product","name":"Pro Monthly"}`
	c := d.client()
	prod, err := c.GetProduct(t.Context(), "prod_1")
	if err != nil {
		t.Fatalf("GetProduct: %v", err)
	}
	if d.lastMethod != http.MethodGet || d.lastPath != "/v1/products/prod_1" {
		t.Fatalf("call = %s %s", d.lastMethod, d.lastPath)
	}
	if d.lastAuth != "Bearer sk_test_123" {
		t.Fatalf("auth = %q", d.lastAuth)
	}
	if prod.ID != "prod_1" || prod.Name != "Pro Monthly" {
		t.Fatalf("product = %+v", prod)
	}
}

func TestStripeGetProductNotFound(t *testing.T) {
	d := newStripeDouble(t)
	d.productBody = "__404__"
	c := d.client()
	_, err := c.GetProduct(t.Context(), "prod_x")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
	if !strings.Contains(err.Error(), "No such product") {
		t.Fatalf("error lacks stripe message: %v", err)
	}
}

func TestStripeUpdateProduct(t *testing.T) {
	d := newStripeDouble(t)
	c := d.client()
	prod, err := c.UpdateProduct(t.Context(), "prod_1", "Pro Annual")
	if err != nil {
		t.Fatalf("UpdateProduct: %v", err)
	}
	if d.lastMethod != http.MethodPost || d.lastPath != "/v1/products/prod_1" {
		t.Fatalf("call = %s %s", d.lastMethod, d.lastPath)
	}
	if d.lastContentType != "application/x-www-form-urlencoded" {
		t.Fatalf("content-type = %q", d.lastContentType)
	}
	if d.lastForm.Get("name") != "Pro Annual" || len(d.lastForm) != 1 {
		t.Fatalf("form = %v", d.lastForm)
	}
	if prod.ID != "prod_1" || prod.Name != "Pro Annual" {
		t.Fatalf("product = %+v", prod)
	}
}

func TestStripeWebhookEndpoints(t *testing.T) {
	d := newStripeDouble(t)
	c := d.client()

	eps, err := c.ListWebhookEndpoints(t.Context())
	if err != nil {
		t.Fatalf("ListWebhookEndpoints: %v", err)
	}
	if len(eps) != 1 || eps[0].ID != "we_1" || eps[0].Status != "enabled" ||
		eps[0].URL != "https://moth.example/billing/stripe/webhook/demo" ||
		len(eps[0].EnabledEvents) != 1 || eps[0].Secret != "" {
		t.Fatalf("endpoints = %+v", eps)
	}

	events := []string{"checkout.session.completed", "customer.subscription.updated", "customer.subscription.deleted"}
	ep, err := c.CreateWebhookEndpoint(t.Context(), "https://moth.example/billing/stripe/webhook/demo", events)
	if err != nil {
		t.Fatalf("CreateWebhookEndpoint: %v", err)
	}
	if ep.ID != "we_2" || ep.Secret != "whsec_test_secret" {
		t.Fatalf("created endpoint = %+v (secret must be exposed on create)", ep)
	}
	if got := d.lastForm["enabled_events[]"]; len(got) != 3 || got[0] != events[0] || got[2] != events[2] {
		t.Fatalf("enabled_events[] = %v", got)
	}
	if d.lastForm.Get("url") != "https://moth.example/billing/stripe/webhook/demo" {
		t.Fatalf("url form = %v", d.lastForm)
	}
}

func TestStripeUpdateWebhookEndpoint(t *testing.T) {
	d := newStripeDouble(t)
	c := d.client()
	events := []string{"checkout.session.completed", "customer.subscription.created"}
	ep, err := c.UpdateWebhookEndpoint(t.Context(), "we_1", events)
	if err != nil {
		t.Fatalf("UpdateWebhookEndpoint: %v", err)
	}
	if d.lastMethod != http.MethodPost || d.lastPath != "/v1/webhook_endpoints/we_1" {
		t.Fatalf("call = %s %s", d.lastMethod, d.lastPath)
	}
	// The update re-enables a Stripe-disabled endpoint and re-points the events.
	if d.lastForm.Get("disabled") != "false" {
		t.Fatalf("form = %v, want disabled=false", d.lastForm)
	}
	if got := d.lastForm["enabled_events[]"]; len(got) != 2 || got[0] != events[0] || got[1] != events[1] {
		t.Fatalf("enabled_events[] = %v", got)
	}
	if ep.ID != "we_1" || len(ep.EnabledEvents) != 2 {
		t.Fatalf("endpoint = %+v", ep)
	}
	if ep.Secret != "" {
		t.Fatalf("secret = %q; Stripe reveals the signing secret on create only", ep.Secret)
	}
}

func TestStripeListWebhookEndpointsPaginates(t *testing.T) {
	// A portfolio instance easily exceeds Stripe's list page: the client must
	// follow has_more/starting_after to exhaustion.
	var queries []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/webhook_endpoints" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		queries = append(queries, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("starting_after") == "" {
			_, _ = io.WriteString(w, `{"object":"list","data":[{"id":"we_1"},{"id":"we_2"}],"has_more":true}`)
			return
		}
		_, _ = io.WriteString(w, `{"object":"list","data":[{"id":"we_3"}],"has_more":false}`)
	}))
	t.Cleanup(srv.Close)
	c := &StripeClient{BaseURL: srv.URL, SecretKey: "sk_test_123", HTTPC: http.DefaultClient}

	eps, err := c.ListWebhookEndpoints(t.Context())
	if err != nil {
		t.Fatalf("ListWebhookEndpoints: %v", err)
	}
	if len(eps) != 3 || eps[0].ID != "we_1" || eps[2].ID != "we_3" {
		t.Fatalf("endpoints = %+v, want we_1..we_3", eps)
	}
	if len(queries) != 2 {
		t.Fatalf("requests = %d (%v), want 2 pages", len(queries), queries)
	}
	if !strings.Contains(queries[0], "limit=100") {
		t.Fatalf("first page query = %q, want limit=100", queries[0])
	}
	if !strings.Contains(queries[1], "starting_after=we_2") {
		t.Fatalf("second page query = %q, want starting_after=we_2 (cursor from last row)", queries[1])
	}
}

func TestStripeAPIErrorFields(t *testing.T) {
	d := newStripeDouble(t)
	d.subBody = "__err__"
	c := d.client()
	_, _, err := c.GetSubscription(t.Context(), "sub_123")
	var apiErr *StripeAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error %v is not a *StripeAPIError", err)
	}
	if apiErr.Status != http.StatusPaymentRequired || apiErr.Type != "card_error" ||
		apiErr.Code != "card_declined" || apiErr.Message != "Your card was declined." {
		t.Fatalf("api error = %+v", apiErr)
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("non-404 API error must not be ErrNotFound: %v", err)
	}
}

func TestStripeAPIErrorResourceMissing(t *testing.T) {
	// A stale stored id (deleted customer, swapped test/live keys) surfaces on
	// POST bodies as a 400 — not a 404 — carrying Stripe's stable
	// "resource_missing" code, the value callers branch on via errors.As.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"message":"No such customer: 'cus_gone'","type":"invalid_request_error","code":"resource_missing"}}`)
	}))
	t.Cleanup(srv.Close)
	c := &StripeClient{BaseURL: srv.URL, SecretKey: "sk_test_123", HTTPC: http.DefaultClient}

	_, err := c.CreateBillingPortalSession(t.Context(), "cus_gone", "https://app.example/account")
	var apiErr *StripeAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error %v is not a *StripeAPIError", err)
	}
	if apiErr.Status != http.StatusBadRequest || apiErr.Code != "resource_missing" {
		t.Fatalf("api error = %+v, want 400 resource_missing", apiErr)
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("400 resource_missing must not be ErrNotFound: %v", err)
	}
}

func TestStripeUnitAmountConversions(t *testing.T) {
	cases := []struct {
		micros   int64
		currency string
		unit     int64
		wantErr  bool
	}{
		{4_990_000, "USD", 499, false},   // two-decimal: unit_amount is cents
		{4_990_000, "usd", 499, false},   // case-insensitive
		{500_000_000, "JPY", 500, false}, // zero-decimal: unit_amount is whole yen
		{1_234_000, "BHD", 1234, false},  // three-decimal
		{4_995_000, "USD", 0, true},      // half a cent: not representable
		{1_000_500_000, "JPY", 0, true},  // fractional yen: not representable
		{1_234_500, "BHD", 0, true},      // half a fils: not representable
	}
	for _, c := range cases {
		unit, err := StripeUnitAmountForMicros(c.micros, c.currency)
		if c.wantErr {
			if err == nil {
				t.Errorf("StripeUnitAmountForMicros(%d, %q) = %d, want error", c.micros, c.currency, unit)
			}
			continue
		}
		if err != nil || unit != c.unit {
			t.Errorf("StripeUnitAmountForMicros(%d, %q) = %d, %v; want %d", c.micros, c.currency, unit, err, c.unit)
			continue
		}
		// Round trip back to micros.
		if back := StripeMicrosForUnitAmount(unit, c.currency); back != c.micros {
			t.Errorf("StripeMicrosForUnitAmount(%d, %q) = %d, want %d", unit, c.currency, back, c.micros)
		}
	}
}

func TestStripeNormalizationZeroDecimalCurrency(t *testing.T) {
	// normalizeStripeSubscription must convert through the currency's minor
	// unit: ¥500 is unit_amount 500 -> 500_000_000 micros, not 5_000_000.
	d := newStripeDouble(t)
	d.subBody = `{"id":"sub_jpy","status":"active","cancel_at_period_end":false,"current_period_end":1893456000,"customer":"cus_1","livemode":true,"items":{"data":[{"price":{"id":"price_jpy","product":"prod_1","unit_amount":500,"currency":"jpy","recurring":{"interval":"month","interval_count":1}}}]}}`
	c := d.client()
	norm, _, err := c.GetSubscription(t.Context(), "sub_jpy")
	if err != nil {
		t.Fatalf("GetSubscription: %v", err)
	}
	if norm.Currency != "JPY" || norm.PriceAmountMicros != 500_000_000 {
		t.Fatalf("normalized price = %d %s, want 500000000 JPY", norm.PriceAmountMicros, norm.Currency)
	}
}

// stripeSigHeader builds a real Stripe-Signature header over payload.
func stripeSigHeader(ts int64, secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	return fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func TestVerifyStripeSignature(t *testing.T) {
	payload := []byte(`{"id":"evt_1","type":"checkout.session.completed"}`)
	const secret = "whsec_test_secret"
	tol := 5 * time.Minute
	now := testNow

	t.Run("valid", func(t *testing.T) {
		h := stripeSigHeader(now.Unix(), secret, payload)
		if err := VerifyStripeSignature(payload, h, secret, now, tol); err != nil {
			t.Fatalf("valid signature rejected: %v", err)
		}
	})
	t.Run("wrong secret", func(t *testing.T) {
		h := stripeSigHeader(now.Unix(), "whsec_other", payload)
		if err := VerifyStripeSignature(payload, h, secret, now, tol); !errors.Is(err, ErrInvalidSignature) {
			t.Fatalf("error = %v, want ErrInvalidSignature", err)
		}
	})
	t.Run("tampered payload", func(t *testing.T) {
		h := stripeSigHeader(now.Unix(), secret, payload)
		if err := VerifyStripeSignature([]byte(`{"id":"evt_evil"}`), h, secret, now, tol); !errors.Is(err, ErrInvalidSignature) {
			t.Fatalf("error = %v, want ErrInvalidSignature", err)
		}
	})
	t.Run("expired timestamp", func(t *testing.T) {
		ts := now.Add(-tol - time.Second).Unix()
		h := stripeSigHeader(ts, secret, payload)
		if err := VerifyStripeSignature(payload, h, secret, now, tol); !errors.Is(err, ErrInvalidSignature) {
			t.Fatalf("error = %v, want ErrInvalidSignature", err)
		}
	})
	t.Run("future timestamp", func(t *testing.T) {
		ts := now.Add(tol + time.Second).Unix()
		h := stripeSigHeader(ts, secret, payload)
		if err := VerifyStripeSignature(payload, h, secret, now, tol); !errors.Is(err, ErrInvalidSignature) {
			t.Fatalf("error = %v, want ErrInvalidSignature", err)
		}
	})
	t.Run("within tolerance", func(t *testing.T) {
		ts := now.Add(-tol + time.Second).Unix()
		h := stripeSigHeader(ts, secret, payload)
		if err := VerifyStripeSignature(payload, h, secret, now, tol); err != nil {
			t.Fatalf("in-window signature rejected: %v", err)
		}
	})
	t.Run("malformed header", func(t *testing.T) {
		for _, h := range []string{"", "garbage", "v1=deadbeef", "t=notanumber,v1=deadbeef", "t=123"} {
			if err := VerifyStripeSignature(payload, h, secret, now, tol); !errors.Is(err, ErrMalformed) {
				t.Fatalf("header %q: error = %v, want ErrMalformed", h, err)
			}
		}
	})
	t.Run("multiple v1 second matches", func(t *testing.T) {
		valid := stripeSigHeader(now.Unix(), secret, payload)
		v1 := strings.TrimPrefix(strings.SplitN(valid, ",", 2)[1], "v1=")
		h := fmt.Sprintf("t=%d,v1=%s,v1=%s", now.Unix(), strings.Repeat("ab", 32), v1)
		if err := VerifyStripeSignature(payload, h, secret, now, tol); err != nil {
			t.Fatalf("second v1 not accepted: %v", err)
		}
	})
	t.Run("empty secret", func(t *testing.T) {
		h := stripeSigHeader(now.Unix(), secret, payload)
		if err := VerifyStripeSignature(payload, h, "", now, tol); !errors.Is(err, ErrInvalidSignature) {
			t.Fatalf("error = %v, want ErrInvalidSignature", err)
		}
	})
	t.Run("zero tolerance skips window", func(t *testing.T) {
		ts := now.Add(-24 * time.Hour).Unix()
		h := stripeSigHeader(ts, secret, payload)
		if err := VerifyStripeSignature(payload, h, secret, now, 0); err != nil {
			t.Fatalf("tolerance 0 should skip the window check: %v", err)
		}
	})
}

func TestParseStripeEvent(t *testing.T) {
	sessionObj := `{"id":"cs_test_1","object":"checkout.session","subscription":"sub_123","customer":"cus_123","client_reference_id":"user-42","metadata":{"project_id":"p1","user_id":"user-42"}}`
	body := fmt.Sprintf(`{"id":"evt_1","object":"event","type":"checkout.session.completed","livemode":false,"data":{"object":%s}}`, sessionObj)

	ev, err := ParseStripeEvent([]byte(body))
	if err != nil {
		t.Fatalf("ParseStripeEvent: %v", err)
	}
	if ev.ID != "evt_1" || ev.Type != "checkout.session.completed" || ev.LiveMode {
		t.Fatalf("event = %+v", ev)
	}
	cs, err := ev.CheckoutSession()
	if err != nil {
		t.Fatalf("CheckoutSession: %v", err)
	}
	if cs.ID != "cs_test_1" || cs.Subscription != "sub_123" || cs.Customer != "cus_123" ||
		cs.ClientReferenceID != "user-42" || cs.Metadata["project_id"] != "p1" {
		t.Fatalf("checkout session = %+v", cs)
	}
}

func TestParseStripeEventSubscriptionObject(t *testing.T) {
	subObj := stripeSubJSON(t, "active", false, true, false, "item", testNow.Add(24*time.Hour).Unix())
	body := fmt.Sprintf(`{"id":"evt_2","type":"customer.subscription.updated","livemode":true,"data":{"object":%s}}`, subObj)
	ev, err := ParseStripeEvent([]byte(body))
	if err != nil {
		t.Fatalf("ParseStripeEvent: %v", err)
	}
	sub, err := ev.SubscriptionObject()
	if err != nil {
		t.Fatalf("SubscriptionObject: %v", err)
	}
	if sub.ID != "sub_123" || sub.Status != "active" || !sub.LiveMode {
		t.Fatalf("subscription = %+v", sub)
	}
	norm := normalizeStripeSubscription(sub)
	if norm.Environment != EnvProduction || norm.ProductID != "price_123" {
		t.Fatalf("normalized = %+v", norm)
	}
	if string(norm.RawState) != subObj {
		t.Fatalf("raw state not the verbatim data.object:\n%s", norm.RawState)
	}
}

func TestParseStripeEventMalformed(t *testing.T) {
	cases := []string{
		`not json`,
		`{"type":"checkout.session.completed"}`,
		`{"id":"evt_1"}`,
	}
	for _, body := range cases {
		if _, err := ParseStripeEvent([]byte(body)); !errors.Is(err, ErrMalformed) {
			t.Fatalf("body %q: error = %v, want ErrMalformed", body, err)
		}
	}
	ev, err := ParseStripeEvent([]byte(`{"id":"evt_1","type":"x","data":{"object":"not an object"}}`))
	if err != nil {
		t.Fatalf("ParseStripeEvent: %v", err)
	}
	if _, err := ev.CheckoutSession(); !errors.Is(err, ErrMalformed) {
		t.Fatalf("CheckoutSession error = %v, want ErrMalformed", err)
	}
	if _, err := ev.SubscriptionObject(); !errors.Is(err, ErrMalformed) {
		t.Fatalf("SubscriptionObject error = %v, want ErrMalformed", err)
	}
}

func TestStripeRecurringForPeriod(t *testing.T) {
	cases := []struct {
		in       string
		interval string
		count    int
	}{
		{"weekly", "week", 1},
		{"week", "week", 1},
		{"P1W", "week", 1},
		{"monthly", "month", 1},
		{"month", "month", 1},
		{"p1m", "month", 1},
		{"two_month", "month", 2},
		{"P2M", "month", 2},
		{"quarterly", "month", 3},
		{"3month", "month", 3},
		{"P3M", "month", 3},
		{"half_year", "month", 6},
		{"P6M", "month", 6},
		{"yearly", "year", 1},
		{"annual", "year", 1},
		{"P1Y", "year", 1},
		{" Monthly ", "month", 1},
	}
	for _, tc := range cases {
		interval, count, err := StripeRecurringForPeriod(tc.in)
		if err != nil {
			t.Fatalf("StripeRecurringForPeriod(%q): %v", tc.in, err)
		}
		if interval != tc.interval || count != tc.count {
			t.Fatalf("StripeRecurringForPeriod(%q) = %q/%d, want %q/%d", tc.in, interval, count, tc.interval, tc.count)
		}
	}
	for _, bad := range []string{"", "daily", "P5X", "fortnightly"} {
		if _, _, err := StripeRecurringForPeriod(bad); err == nil {
			t.Fatalf("StripeRecurringForPeriod(%q) accepted", bad)
		}
	}
}

func TestStripeTrialDays(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"", 0},
		{"P3D", 3},
		{"p7d", 7},
		{"P1W", 7},
		{"P2W", 14},
		{"weekly", 7},
		{"week", 7},
		{"P1M", 30},
		{"monthly", 30},
		{"two_month", 60},
		{"quarterly", 90},
		{"half_year", 180},
		{"yearly", 365},
		{"P1Y", 365},
	}
	for _, tc := range cases {
		got, err := StripeTrialDays(tc.in)
		if err != nil {
			t.Fatalf("StripeTrialDays(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("StripeTrialDays(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
	for _, bad := range []string{"forever", "P0D", "PXW", "3 days"} {
		if _, err := StripeTrialDays(bad); err == nil {
			t.Fatalf("StripeTrialDays(%q) accepted", bad)
		}
	}
}
