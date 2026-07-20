package billingrpc

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	billingv1 "github.com/aloisdeniel/moth/gen/moth/billing/v1"
	"github.com/aloisdeniel/moth/internal/billing"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

const (
	testStripeWebhookSecret = "whsec_test_secret"
	testStripePriceID       = "price_test1"
	testStripeCustomerID    = "cus_test1"
	testStripeSubID         = "sub_test1"
)

// stripeDouble is an httptest Stripe API with parameterized subscription state:
// customers + checkout sessions + billing portal sessions + subscriptions get.
type stripeDouble struct {
	srv *httptest.Server

	mu                sync.Mutex
	subStatus         string
	cancelAtPeriodEnd bool
	periodEnd         time.Time
	livemode          bool
	subFail           bool   // 500 on subscription GET (transient blip)
	subMissing        bool   // 404 on subscription GET (dropped from retention)
	staleCustomer     string // checkout POST naming this customer 400s resource_missing
	portalMissing     bool   // portal POST 400s resource_missing (stale customer)
	checkoutDeclined  bool   // checkout POST 402s with a detailed card error
	unitAmount        int64
	currency          string
	customerCreates   int
	checkoutForms     []url.Values
}

// stripeResourceMissing writes Stripe's stable stale-id error envelope.
func stripeResourceMissing(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{
		"message": msg, "type": "invalid_request_error", "code": "resource_missing",
	}})
}

func newStripeDouble(f *fixture) *stripeDouble {
	d := &stripeDouble{
		subStatus:  "active",
		periodEnd:  f.now.Add(30 * 24 * time.Hour),
		unitAmount: 499, // cents -> 4_990_000 micros
		currency:   "usd",
	}
	d.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		defer d.mu.Unlock()
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/customers":
			d.customerCreates++
			_ = json.NewEncoder(w).Encode(map[string]any{"id": testStripeCustomerID, "email": "u@demo.test"})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/checkout/sessions":
			_ = r.ParseForm()
			d.checkoutForms = append(d.checkoutForms, r.PostForm)
			if d.checkoutDeclined {
				w.WriteHeader(http.StatusPaymentRequired)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{
					"message": "Your card was declined; customer cus_secret_42 (sk_live mode).",
					"type":    "card_error", "code": "card_declined",
				}})
				return
			}
			if d.staleCustomer != "" && r.PostForm.Get("customer") == d.staleCustomer {
				stripeResourceMissing(w, "No such customer: '"+d.staleCustomer+"'")
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "cs_1", "url": "https://checkout.stripe.test/cs_1"})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/billing_portal/sessions":
			if d.portalMissing {
				stripeResourceMissing(w, "No such customer: 'cus_stale'")
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "bps_1", "url": "https://portal.stripe.test/bps_1"})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/subscriptions/"):
			if d.subFail {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if d.subMissing {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "no such subscription"}})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": testStripeSubID, "status": d.subStatus,
				"cancel_at_period_end": d.cancelAtPeriodEnd,
				"current_period_end":   d.periodEnd.Unix(),
				"customer":             testStripeCustomerID,
				"livemode":             d.livemode,
				"items": map[string]any{"data": []map[string]any{{
					"current_period_end": d.periodEnd.Unix(),
					"price": map[string]any{
						"id": testStripePriceID, "product": "prod_1",
						"unit_amount": d.unitAmount, "currency": d.currency,
						"recurring": map[string]any{"interval": "month", "interval_count": 1},
					},
				}}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	f.t.Cleanup(d.srv.Close)
	return d
}

// set mutates double state under the lock between deliveries.
func (d *stripeDouble) set(fn func(d *stripeDouble)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	fn(d)
}

// setStripeCreds stores encrypted Stripe billing credentials pointing the
// client at the double; webhookSecret == "" leaves the webhook secret unset.
func (f *fixture) setStripeCreds(baseURL, webhookSecret string) {
	f.t.Helper()
	skEnc, err := f.master.Encrypt([]byte("sk_test_moth"))
	if err != nil {
		f.t.Fatal(err)
	}
	cred := store.BillingCredentials{ProjectID: f.project.ID, StripeSecretKeyEnc: skEnc,
		CreatedAt: f.now, UpdatedAt: f.now}
	if webhookSecret != "" {
		whEnc, err := f.master.Encrypt([]byte(webhookSecret))
		if err != nil {
			f.t.Fatal(err)
		}
		cred.StripeWebhookSecretEnc = whEnc
	}
	if err := f.st.UpsertBillingCredentials(f.ctx(), cred); err != nil {
		f.t.Fatal(err)
	}
	f.h.stripeBaseURL = baseURL
}

// setStripePriceID points the fixture "monthly" tier at a Stripe price.
func (f *fixture) setStripePriceID(priceID string) {
	f.t.Helper()
	ctx := context.Background()
	p, err := f.st.GetProduct(ctx, f.project.ID, f.prodID)
	if err != nil {
		f.t.Fatal(err)
	}
	p.StripePriceID = priceID
	p.StripeProductID = "prod_1"
	p.UpdatedAt = f.now
	if err := f.st.UpdateProduct(ctx, p); err != nil {
		f.t.Fatal(err)
	}
}

// stripeSign builds a real Stripe-Signature header: HMAC-SHA256 over
// "<t>.<body>" with the endpoint secret.
func stripeSign(secret string, at time.Time, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.%s", at.Unix(), body)
	return fmt.Sprintf("t=%d,v1=%s", at.Unix(), hex.EncodeToString(mac.Sum(nil)))
}

// deliverStripe signs and delivers a webhook body with the fixture secret.
func (f *fixture) deliverStripe(body []byte) error {
	return f.h.ProcessStripeWebhook(f.ctx(), f.project, body,
		stripeSign(testStripeWebhookSecret, f.now, body))
}

// stripeCheckoutEvent builds a checkout.session.completed webhook body.
func stripeCheckoutEvent(evtID, subID, custID, userID, projectID string) []byte {
	obj := map[string]any{
		"id": "cs_1", "subscription": subID, "customer": custID,
		"client_reference_id": userID,
		"metadata":            map[string]string{"project_id": projectID, "user_id": userID, "product": "monthly"},
	}
	raw, _ := json.Marshal(map[string]any{
		"id": evtID, "type": "checkout.session.completed", "livemode": false,
		"data": map[string]any{"object": obj},
	})
	return raw
}

// stripeSubEvent builds a customer.subscription.* webhook body. The embedded
// object is deliberately stale/minimal: moth must re-read, never trust it.
func stripeSubEvent(evtID, evtType, subID, custID string) []byte {
	obj := map[string]any{"id": subID, "customer": custID, "status": "active"}
	raw, _ := json.Marshal(map[string]any{
		"id": evtID, "type": evtType, "livemode": false,
		"data": map[string]any{"object": obj},
	})
	return raw
}

// seedStripeSubscription drives a full checkout.session.completed through the
// webhook so later tests start from an attributed, active Stripe subscription.
func seedStripeSubscription(t *testing.T, f *fixture, d *stripeDouble) {
	t.Helper()
	f.setStripeCreds(d.srv.URL, testStripeWebhookSecret)
	f.setStripePriceID(testStripePriceID)
	body := stripeCheckoutEvent("evt_seed", testStripeSubID, testStripeCustomerID, f.user.ID, f.project.ID)
	if err := f.deliverStripe(body); err != nil {
		t.Fatalf("seed checkout webhook: %v", err)
	}
}

// --- checkout RPC ----------------------------------------------------------

func TestCreateCheckoutSessionHappyPathReusesCustomer(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, "")
	f.setStripePriceID(testStripePriceID)

	resp, err := f.h.CreateCheckoutSession(f.ctx(), authReq(f, &billingv1.CreateCheckoutSessionRequest{
		ProductIdentifier: "monthly",
		SuccessUrl:        "https://app.demo.test/ok",
		CancelUrl:         "https://app.demo.test/cancel",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.Url != "https://checkout.stripe.test/cs_1" {
		t.Fatalf("url = %q", resp.Msg.Url)
	}
	// The lazy customer mapping was created.
	sc, err := f.st.GetStripeCustomer(context.Background(), f.project.ID, f.user.ID)
	if err != nil {
		t.Fatalf("stripe customer row: %v", err)
	}
	if sc.StripeCustomerID != testStripeCustomerID {
		t.Fatalf("customer id = %q", sc.StripeCustomerID)
	}
	// The session was bound to the customer, price and moth identity.
	if n := len(d.checkoutForms); n != 1 {
		t.Fatalf("checkout creates = %d", n)
	}
	form := d.checkoutForms[0]
	if form.Get("line_items[0][price]") != testStripePriceID ||
		form.Get("customer") != testStripeCustomerID ||
		form.Get("client_reference_id") != f.user.ID ||
		form.Get("metadata[project_id]") != f.project.ID ||
		form.Get("metadata[user_id]") != f.user.ID {
		t.Fatalf("checkout form missing bindings: %v", form)
	}

	// A second checkout reuses the mapping: exactly one customer was created.
	if _, err := f.h.CreateCheckoutSession(f.ctx(), authReq(f, &billingv1.CreateCheckoutSessionRequest{
		ProductIdentifier: "monthly",
		SuccessUrl:        "https://app.demo.test/ok",
		CancelUrl:         "https://app.demo.test/cancel",
	})); err != nil {
		t.Fatal(err)
	}
	if d.customerCreates != 1 {
		t.Fatalf("customer creates = %d, want 1 (mapping not reused)", d.customerCreates)
	}
}

func TestCreateCheckoutSessionNotConfigured(t *testing.T) {
	f := newFixture(t)
	f.setStripePriceID(testStripePriceID)
	req := func() *connect.Request[billingv1.CreateCheckoutSessionRequest] {
		return authReq(f, &billingv1.CreateCheckoutSessionRequest{
			ProductIdentifier: "monthly",
			SuccessUrl:        "https://app.demo.test/ok",
			CancelUrl:         "https://app.demo.test/cancel",
		})
	}
	// No credentials row at all.
	_, err := f.h.CreateCheckoutSession(f.ctx(), req())
	assertReason(t, err, authrpc.ReasonBillingNotConfigured)

	// A credentials row with only Apple fields: Stripe stays not-configured
	// (field presence, not row presence).
	f.setAppleCreds("http://unused.test")
	_, err = f.h.CreateCheckoutSession(f.ctx(), req())
	assertReason(t, err, authrpc.ReasonBillingNotConfigured)
}

func TestCreateCheckoutSessionProductNotOnStripe(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, "")
	// The fixture product has no stripe_price_id.
	_, err := f.h.CreateCheckoutSession(f.ctx(), authReq(f, &billingv1.CreateCheckoutSessionRequest{
		ProductIdentifier: "monthly",
		SuccessUrl:        "https://app.demo.test/ok",
		CancelUrl:         "https://app.demo.test/cancel",
	}))
	assertReason(t, err, authrpc.ReasonProductNotOnStore)
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want FailedPrecondition", connect.CodeOf(err))
	}
}

func TestCreateCheckoutSessionRejectsBadURLs(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, "")
	f.setStripePriceID(testStripePriceID)
	cases := []struct{ success, cancel string }{
		{"notaurl", "https://app.demo.test/cancel"},
		{"/relative/path", "https://app.demo.test/cancel"},
		{"ftp://app.demo.test/ok", "https://app.demo.test/cancel"},
		{"https://app.demo.test/ok", ""},
		{"https://app.demo.test/ok", "javascript:alert(1)"},
	}
	for _, c := range cases {
		_, err := f.h.CreateCheckoutSession(f.ctx(), authReq(f, &billingv1.CreateCheckoutSessionRequest{
			ProductIdentifier: "monthly", SuccessUrl: c.success, CancelUrl: c.cancel,
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("success=%q cancel=%q: code = %v, want InvalidArgument", c.success, c.cancel, connect.CodeOf(err))
		}
	}
}

func TestCreateCheckoutSessionUnknownProduct(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, "")
	_, err := f.h.CreateCheckoutSession(f.ctx(), authReq(f, &billingv1.CreateCheckoutSessionRequest{
		ProductIdentifier: "nope",
		SuccessUrl:        "https://app.demo.test/ok",
		CancelUrl:         "https://app.demo.test/cancel",
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code = %v, want NotFound", connect.CodeOf(err))
	}
}

func TestCreateCheckoutSessionUnauthenticated(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, "")
	f.setStripePriceID(testStripePriceID)
	req := connect.NewRequest(&billingv1.CreateCheckoutSessionRequest{
		ProductIdentifier: "monthly",
		SuccessUrl:        "https://app.demo.test/ok",
		CancelUrl:         "https://app.demo.test/cancel",
	}) // no Bearer
	if _, err := f.h.CreateCheckoutSession(f.ctx(), req); err == nil {
		t.Fatal("unauthenticated checkout must fail")
	}
}

// --- billing portal RPC ----------------------------------------------------

func TestCreateBillingPortalSession(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, "")
	if err := f.st.CreateStripeCustomer(context.Background(), store.StripeCustomer{
		ProjectID: f.project.ID, UserID: f.user.ID,
		StripeCustomerID: testStripeCustomerID, CreatedAt: f.now,
	}); err != nil {
		t.Fatal(err)
	}
	resp, err := f.h.CreateBillingPortalSession(f.ctx(), authReq(f, &billingv1.CreateBillingPortalSessionRequest{
		ReturnUrl: "https://app.demo.test/account",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.Url != "https://portal.stripe.test/bps_1" {
		t.Fatalf("url = %q", resp.Msg.Url)
	}
}

func TestCreateBillingPortalSessionNoCustomer(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, "")
	_, err := f.h.CreateBillingPortalSession(f.ctx(), authReq(f, &billingv1.CreateBillingPortalSessionRequest{
		ReturnUrl: "https://app.demo.test/account",
	}))
	assertReason(t, err, authrpc.ReasonNoBillingHistory)
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want FailedPrecondition", connect.CodeOf(err))
	}
}

// --- webhook: checkout completion (acquisition) ----------------------------

func TestStripeWebhookCheckoutCompletedGrantsEntitlement(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	seedStripeSubscription(t, f, d)

	sub, err := f.st.GetSubscriptionByStoreID(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, testStripeSubID)
	if err != nil {
		t.Fatalf("subscription row: %v", err)
	}
	if sub.UserID != f.user.ID || sub.Status != store.SubscriptionStatusActive ||
		sub.Environment != store.SubscriptionEnvironmentSandbox ||
		sub.SubscriptionID != testStripeCustomerID || sub.ProductID != f.prodID {
		t.Fatalf("subscription = %+v", sub)
	}
	// The customer mapping was reconciled from the session (none existed: this
	// checkout was provisioned without the RPC).
	if _, err := f.st.GetStripeCustomer(context.Background(), f.project.ID, f.user.ID); err != nil {
		t.Fatalf("stripe customer not reconciled: %v", err)
	}
	// GetCustomerInfo: entitlement from the store, subscription flagged stripe +
	// sandbox (livemode=false).
	got, err := f.h.GetCustomerInfo(f.ctx(), authReq(f, &billingv1.GetCustomerInfoRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	ent := hasEntitlement(got.Msg.CustomerInfo, "pro")
	if ent == nil || ent.Source != billingv1.EntitlementSource_ENTITLEMENT_SOURCE_STORE {
		t.Fatalf("entitlement = %+v", ent)
	}
	if len(got.Msg.CustomerInfo.Subscriptions) != 1 ||
		got.Msg.CustomerInfo.Subscriptions[0].Store != billingv1.Store_STORE_STRIPE ||
		!got.Msg.CustomerInfo.Subscriptions[0].IsSandbox {
		t.Fatalf("subscriptions = %+v", got.Msg.CustomerInfo.Subscriptions)
	}
	// Exactly one purchased event with Stripe's REAL amount: 499 cents ->
	// 4_990_000 micros USD.
	rows, err := storeRawEventRows(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Type != store.SubscriptionEventPurchased ||
		rows[0].PriceAmountMicros != 4_990_000 || rows[0].Currency != "USD" {
		t.Fatalf("events = %+v", rows)
	}

	// Replaying the same event id is a no-op: no double-emit, no re-apply.
	body := stripeCheckoutEvent("evt_seed", testStripeSubID, testStripeCustomerID, f.user.ID, f.project.ID)
	if err := f.deliverStripe(body); err != nil {
		t.Fatalf("replay: %v", err)
	}
	assertEventCount(t, f, store.SubscriptionEventPurchased, 1)
}

func TestStripeWebhookCheckoutCompletedTrial(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	d.set(func(d *stripeDouble) { d.subStatus = "trialing" })
	seedStripeSubscription(t, f, d)

	sub, err := f.st.GetSubscriptionByStoreID(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, testStripeSubID)
	if err != nil {
		t.Fatal(err)
	}
	if sub.Status != store.SubscriptionStatusTrialing {
		t.Fatalf("status = %q, want trialing", sub.Status)
	}
	assertEventCount(t, f, store.SubscriptionEventTrialStarted, 1)
	assertEventCount(t, f, store.SubscriptionEventPurchased, 0)
}

func TestStripeWebhookBadSignatureRejected(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, testStripeWebhookSecret)
	body := stripeCheckoutEvent("evt_bad", testStripeSubID, testStripeCustomerID, f.user.ID, f.project.ID)

	// Signed with the wrong secret.
	err := f.h.ProcessStripeWebhook(f.ctx(), f.project, body, stripeSign("whsec_wrong", f.now, body))
	if !errors.Is(err, billing.ErrInvalidSignature) {
		t.Fatalf("wrong-secret err = %v, want ErrInvalidSignature", err)
	}
	// Timestamp outside the tolerance window.
	err = f.h.ProcessStripeWebhook(f.ctx(), f.project, body,
		stripeSign(testStripeWebhookSecret, f.now.Add(-time.Hour), body))
	if !errors.Is(err, billing.ErrInvalidSignature) {
		t.Fatalf("stale-timestamp err = %v, want ErrInvalidSignature", err)
	}
	// Nothing was recorded or applied.
	if _, err := f.st.GetStoreNotification(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, "evt_bad"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unverified event must not be recorded: %v", err)
	}
}

func TestStripeWebhookProjectMismatchAckedNotApplied(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, testStripeWebhookSecret)
	f.setStripePriceID(testStripePriceID)
	// A session provisioned for another project (metadata names a different
	// project id): on a shared Stripe account every project's endpoint receives
	// every event, so a validly signed cross-tenant event is acknowledged (nil
	// -> 200, no Stripe retry storm) but never applied.
	body := stripeCheckoutEvent("evt_xproj", testStripeSubID, testStripeCustomerID, f.user.ID, "another-project")
	if err := f.deliverStripe(body); err != nil {
		t.Fatalf("cross-project event must be acknowledged, got %v", err)
	}
	// Recorded and sealed as processed (a redelivery would be a pure replay)...
	n, err := f.st.GetStoreNotification(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, "evt_xproj")
	if err != nil {
		t.Fatalf("cross-project event not recorded: %v", err)
	}
	if n.ProcessedAt == nil {
		t.Fatal("cross-project event must be marked processed")
	}
	// ...with zero state change: no subscription, no revenue events.
	if _, err := f.st.GetSubscriptionByStoreID(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, testStripeSubID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-project event must not create a subscription: %v", err)
	}
	if rows, err := storeRawEventRows(f); err != nil || len(rows) != 0 {
		t.Fatalf("cross-project event must not emit revenue events: %+v (%v)", rows, err)
	}
}

// --- webhook: subscription lifecycle ---------------------------------------

func TestStripeWebhookUpdatedRenewalEmitsRenewed(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	seedStripeSubscription(t, f, d)

	// Stripe renewed the subscription: the authoritative period end advanced.
	newEnd := f.now.Add(60 * 24 * time.Hour)
	d.set(func(d *stripeDouble) { d.periodEnd = newEnd })
	body := stripeSubEvent("evt_renew", "customer.subscription.updated", testStripeSubID, testStripeCustomerID)
	if err := f.deliverStripe(body); err != nil {
		t.Fatal(err)
	}
	sub, _ := f.st.GetSubscriptionByStoreID(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, testStripeSubID)
	if sub.CurrentPeriodEnd == nil || !sub.CurrentPeriodEnd.Equal(newEnd.Truncate(time.Second)) {
		t.Fatalf("period end = %v, want %v", sub.CurrentPeriodEnd, newEnd)
	}
	// The renewed event carries the store-reported price.
	rows, err := storeRawEventRows(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[1].Type != store.SubscriptionEventRenewed ||
		rows[1].PriceAmountMicros != 4_990_000 || rows[1].Currency != "USD" {
		t.Fatalf("events = %+v", rows)
	}
}

func TestStripeWebhookUpdatedCancelFlipEmitsCanceled(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	seedStripeSubscription(t, f, d)

	// The user canceled from the portal: still active until period end, but no
	// longer auto-renewing.
	d.set(func(d *stripeDouble) { d.cancelAtPeriodEnd = true })
	body := stripeSubEvent("evt_cancel", "customer.subscription.updated", testStripeSubID, testStripeCustomerID)
	if err := f.deliverStripe(body); err != nil {
		t.Fatal(err)
	}
	sub, _ := f.st.GetSubscriptionByStoreID(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, testStripeSubID)
	if sub.AutoRenew || sub.Status != store.SubscriptionStatusActive {
		t.Fatalf("subscription = %+v, want active non-renewing", sub)
	}
	assertEventCount(t, f, store.SubscriptionEventCanceled, 1)
	// Entitlement survives until period end.
	got, _ := f.h.GetCustomerInfo(f.ctx(), authReq(f, &billingv1.GetCustomerInfoRequest{}))
	if hasEntitlement(got.Msg.CustomerInfo, "pro") == nil {
		t.Fatal("entitlement must survive a cancel until period end")
	}
}

func TestStripeWebhookDeletedExpiresEntitlement(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	seedStripeSubscription(t, f, d)

	// On deletion Stripe's API still returns the subscription, status=canceled.
	d.set(func(d *stripeDouble) { d.subStatus = "canceled" })
	body := stripeSubEvent("evt_del", "customer.subscription.deleted", testStripeSubID, testStripeCustomerID)
	if err := f.deliverStripe(body); err != nil {
		t.Fatal(err)
	}
	sub, _ := f.st.GetSubscriptionByStoreID(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, testStripeSubID)
	if sub.Status != store.SubscriptionStatusExpired {
		t.Fatalf("status = %q, want expired", sub.Status)
	}
	assertEventCount(t, f, store.SubscriptionEventExpired, 1)
	got, _ := f.h.GetCustomerInfo(f.ctx(), authReq(f, &billingv1.GetCustomerInfoRequest{}))
	if hasEntitlement(got.Msg.CustomerInfo, "pro") != nil {
		t.Fatal("entitlement must be gone after deletion")
	}
}

func TestStripeWebhookDeletedSubGoneFromAPI(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	seedStripeSubscription(t, f, d)

	// The re-read 404s (subscription dropped from Stripe's retention): the
	// deleted event still lands as expired, synthesized from the stored row.
	d.set(func(d *stripeDouble) { d.subMissing = true })
	body := stripeSubEvent("evt_del404", "customer.subscription.deleted", testStripeSubID, testStripeCustomerID)
	if err := f.deliverStripe(body); err != nil {
		t.Fatal(err)
	}
	sub, _ := f.st.GetSubscriptionByStoreID(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, testStripeSubID)
	if sub.Status != store.SubscriptionStatusExpired || sub.AutoRenew {
		t.Fatalf("subscription = %+v, want expired non-renewing", sub)
	}
	// The synthesized state keeps the product mapping.
	if sub.ProductID != f.prodID {
		t.Fatalf("product id = %q, want %q", sub.ProductID, f.prodID)
	}
	assertEventCount(t, f, store.SubscriptionEventExpired, 1)
}

func TestStripeWebhookPastDueKeepsEntitlement(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	seedStripeSubscription(t, f, d)

	// past_due maps to in_billing_retry — granted (never lock out a paying user
	// over a card hiccup).
	d.set(func(d *stripeDouble) { d.subStatus = "past_due" })
	body := stripeSubEvent("evt_pastdue", "customer.subscription.updated", testStripeSubID, testStripeCustomerID)
	if err := f.deliverStripe(body); err != nil {
		t.Fatal(err)
	}
	sub, _ := f.st.GetSubscriptionByStoreID(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, testStripeSubID)
	if sub.Status != store.SubscriptionStatusInBillingRetry {
		t.Fatalf("status = %q, want in_billing_retry", sub.Status)
	}
	got, _ := f.h.GetCustomerInfo(f.ctx(), authReq(f, &billingv1.GetCustomerInfoRequest{}))
	if hasEntitlement(got.Msg.CustomerInfo, "pro") == nil {
		t.Fatal("entitlement must survive billing retry")
	}
}

func TestStripeWebhookUnknownSubscriptionRecordedNotApplied(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, testStripeWebhookSecret)
	f.setStripePriceID(testStripePriceID)

	// Neither the subscription nor the customer is known to moth: recorded and
	// acknowledged, never applied.
	body := stripeSubEvent("evt_unknown", "customer.subscription.updated", "sub_foreign", "cus_foreign")
	if err := f.deliverStripe(body); err != nil {
		t.Fatal(err)
	}
	if _, err := f.st.GetSubscriptionByStoreID(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, "sub_foreign"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown subscription must not be created: %v", err)
	}
	n, err := f.st.GetStoreNotification(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, "evt_unknown")
	if err != nil {
		t.Fatal(err)
	}
	if n.ProcessedAt == nil {
		t.Fatal("record-only event must be marked processed")
	}
}

func TestStripeWebhookMissedCheckoutAttributedViaCustomer(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, testStripeWebhookSecret)
	f.setStripePriceID(testStripePriceID)
	// CreateCheckoutSession had created the customer mapping, but the
	// checkout.session.completed webhook was lost. subscription.created can
	// still attribute via the customer id.
	if err := f.st.CreateStripeCustomer(context.Background(), store.StripeCustomer{
		ProjectID: f.project.ID, UserID: f.user.ID,
		StripeCustomerID: testStripeCustomerID, CreatedAt: f.now,
	}); err != nil {
		t.Fatal(err)
	}
	body := stripeSubEvent("evt_created", "customer.subscription.created", testStripeSubID, testStripeCustomerID)
	if err := f.deliverStripe(body); err != nil {
		t.Fatal(err)
	}
	sub, err := f.st.GetSubscriptionByStoreID(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, testStripeSubID)
	if err != nil {
		t.Fatalf("attributed subscription not created: %v", err)
	}
	if sub.UserID != f.user.ID || sub.Status != store.SubscriptionStatusActive {
		t.Fatalf("subscription = %+v", sub)
	}
	// The acquisition is recovered exactly once.
	assertEventCount(t, f, store.SubscriptionEventPurchased, 1)

	// If the checkout event arrives late, it must not double-emit.
	late := stripeCheckoutEvent("evt_late", testStripeSubID, testStripeCustomerID, f.user.ID, f.project.ID)
	if err := f.deliverStripe(late); err != nil {
		t.Fatal(err)
	}
	assertEventCount(t, f, store.SubscriptionEventPurchased, 1)
}

// TestStripeWebhookTransientFailureRedelivers mirrors the Google RTDN test: a
// transient Stripe API failure leaves the notification unprocessed and the
// error propagates (503 -> Stripe redelivers); the redelivery applies the flip.
func TestStripeWebhookTransientFailureRedelivers(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	seedStripeSubscription(t, f, d)

	d.set(func(d *stripeDouble) { d.subFail = true; d.subStatus = "canceled" })
	body := stripeSubEvent("evt_flaky", "customer.subscription.updated", testStripeSubID, testStripeCustomerID)
	if err := f.deliverStripe(body); err == nil {
		t.Fatal("transient re-read failure must propagate so the webhook can signal a retry")
	}
	// Unchanged subscription, unprocessed notification row.
	sub, _ := f.st.GetSubscriptionByStoreID(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, testStripeSubID)
	if sub.Status != store.SubscriptionStatusActive {
		t.Fatalf("subscription must be unchanged on transient failure: %+v", sub)
	}
	n, err := f.st.GetStoreNotification(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, "evt_flaky")
	if err != nil {
		t.Fatalf("notification not recorded: %v", err)
	}
	if n.ProcessedAt != nil {
		t.Fatal("notification wrongly marked processed after a failed apply")
	}

	// Stripe redelivers the SAME event id; the re-read now succeeds.
	d.set(func(d *stripeDouble) { d.subFail = false })
	if err := f.deliverStripe(body); err != nil {
		t.Fatalf("redelivery: %v", err)
	}
	sub, _ = f.st.GetSubscriptionByStoreID(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, testStripeSubID)
	if sub.Status != store.SubscriptionStatusExpired {
		t.Fatalf("redelivery did not apply the state change: %+v", sub)
	}
}

func TestStripeWebhookOtherEventTypesAreAuditOnly(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, testStripeWebhookSecret)
	raw, _ := json.Marshal(map[string]any{
		"id": "evt_other", "type": "invoice.paid", "livemode": false,
		"data": map[string]any{"object": map[string]any{"id": "in_1"}},
	})
	if err := f.deliverStripe(raw); err != nil {
		t.Fatal(err)
	}
	n, err := f.st.GetStoreNotification(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, "evt_other")
	if err != nil {
		t.Fatal(err)
	}
	if n.ProcessedAt == nil {
		t.Fatal("audit-only event must be acknowledged")
	}
}

// --- reconcile & receipt-path guards ---------------------------------------

func TestStripeReconcileRereadsLapsedSubscription(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, "")
	f.setStripePriceID(testStripePriceID)
	// An active Stripe subscription whose period already lapsed (missed webhook).
	end := f.now.Add(-time.Hour)
	if _, _, err := f.st.UpsertSubscription(context.Background(), store.Subscription{
		ID: authrpc.NewID(), ProjectID: f.project.ID, UserID: f.user.ID,
		Store: store.SubscriptionStoreStripe, ProductID: f.prodID,
		StoreTransactionID: testStripeSubID, SubscriptionID: testStripeCustomerID,
		Status: store.SubscriptionStatusActive, CurrentPeriodEnd: &end,
		Environment: store.SubscriptionEnvironmentSandbox, CreatedAt: f.now, UpdatedAt: f.now,
	}); err != nil {
		t.Fatal(err)
	}
	d.set(func(d *stripeDouble) { d.subStatus = "canceled"; d.periodEnd = end })
	if err := f.h.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	sub, _ := f.st.GetSubscriptionByStoreID(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, testStripeSubID)
	if sub.Status != store.SubscriptionStatusExpired {
		t.Fatalf("reconcile did not flip lapsed stripe subscription: %+v", sub)
	}
}

func TestSubmitPurchaseStripeStoreRejected(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, "")
	// Web purchases only go through checkout; a stripe "receipt" is invalid.
	_, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
		Store: billingv1.Store_STORE_STRIPE,
	}))
	assertReason(t, err, authrpc.ReasonInvalidReceipt)
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", connect.CodeOf(err))
	}
	_, err = f.h.RestorePurchases(f.ctx(), authReq(f, &billingv1.RestorePurchasesRequest{
		Store: billingv1.Store_STORE_STRIPE, Receipts: []string{"anything"},
	}))
	assertReason(t, err, authrpc.ReasonInvalidReceipt)
}

// --- duplicate-subscription guard -------------------------------------------

func TestCreateCheckoutSessionAlreadySubscribed(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	seedStripeSubscription(t, f, d) // active stripe sub on "monthly"

	checkout := func(product string) error {
		_, err := f.h.CreateCheckoutSession(f.ctx(), authReq(f, &billingv1.CreateCheckoutSessionRequest{
			ProductIdentifier: product,
			SuccessUrl:        "https://app.demo.test/ok",
			CancelUrl:         "https://app.demo.test/cancel",
		}))
		return err
	}
	// A second checkout for the same tier would double-bill: rejected.
	err := checkout("monthly")
	assertReason(t, err, authrpc.ReasonAlreadySubscribed)
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want FailedPrecondition", connect.CodeOf(err))
	}

	// A different tier is allowed (upgrade/cross-grade stays possible).
	yearly := store.Product{ID: authrpc.NewID(), ProjectID: f.project.ID, Identifier: "yearly",
		DisplayName: "Yearly", BillingPeriod: "yearly", PriceAmountMicros: 49_900_000, Currency: "USD",
		StripePriceID: "price_yearly", StripeProductID: "prod_yearly",
		EntitlementIDs: []string{f.entPro.ID}, CreatedAt: f.now, UpdatedAt: f.now}
	if err := f.st.CreateProduct(context.Background(), yearly); err != nil {
		t.Fatal(err)
	}
	if err := checkout("yearly"); err != nil {
		t.Fatalf("different product must be allowed: %v", err)
	}

	// Once the existing subscription no longer grants access, re-subscribing to
	// the same tier is allowed again.
	if _, _, err := f.st.UpsertSubscription(context.Background(), store.Subscription{
		ID: authrpc.NewID(), ProjectID: f.project.ID, UserID: f.user.ID,
		Store: store.SubscriptionStoreStripe, ProductID: f.prodID,
		StoreTransactionID: testStripeSubID, SubscriptionID: testStripeCustomerID,
		Status: store.SubscriptionStatusExpired, Environment: store.SubscriptionEnvironmentSandbox,
		CreatedAt: f.now, UpdatedAt: f.now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := checkout("monthly"); err != nil {
		t.Fatalf("expired subscription must not block a new checkout: %v", err)
	}
}

// --- trial guard -------------------------------------------------------------

func TestCreateCheckoutSessionUnsupportedTrialRejected(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, "")
	f.setStripePriceID(testStripePriceID)
	ctx := context.Background()
	p, err := f.st.GetProduct(ctx, f.project.ID, f.prodID)
	if err != nil {
		t.Fatal(err)
	}
	p.TrialPeriod = "lifetime" // no Stripe trial_period_days mapping
	p.UpdatedAt = f.now
	if err := f.st.UpdateProduct(ctx, p); err != nil {
		t.Fatal(err)
	}
	// The advertised trial cannot be expressed: the checkout fails instead of
	// silently charging without the trial.
	_, err = f.h.CreateCheckoutSession(f.ctx(), authReq(f, &billingv1.CreateCheckoutSessionRequest{
		ProductIdentifier: "monthly",
		SuccessUrl:        "https://app.demo.test/ok",
		CancelUrl:         "https://app.demo.test/cancel",
	}))
	assertReason(t, err, authrpc.ReasonTrialNotSupported)
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want FailedPrecondition", connect.CodeOf(err))
	}
	if len(d.checkoutForms) != 0 {
		t.Fatalf("no checkout session must be created, got %d", len(d.checkoutForms))
	}
}

// --- stale customer self-heal ------------------------------------------------

func TestCreateCheckoutSessionHealsStaleCustomerMapping(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, "")
	f.setStripePriceID(testStripePriceID)
	// The stored mapping names a customer the Stripe account no longer knows
	// (swapped test/live keys or dashboard deletion).
	if err := f.st.CreateStripeCustomer(context.Background(), store.StripeCustomer{
		ProjectID: f.project.ID, UserID: f.user.ID,
		StripeCustomerID: "cus_stale", CreatedAt: f.now,
	}); err != nil {
		t.Fatal(err)
	}
	d.set(func(d *stripeDouble) { d.staleCustomer = "cus_stale" })

	resp, err := f.h.CreateCheckoutSession(f.ctx(), authReq(f, &billingv1.CreateCheckoutSessionRequest{
		ProductIdentifier: "monthly",
		SuccessUrl:        "https://app.demo.test/ok",
		CancelUrl:         "https://app.demo.test/cancel",
	}))
	if err != nil {
		t.Fatalf("stale mapping must self-heal, got %v", err)
	}
	if resp.Msg.Url != "https://checkout.stripe.test/cs_1" {
		t.Fatalf("url = %q", resp.Msg.Url)
	}
	// A fresh customer was minted once and the mapping overwritten.
	if d.customerCreates != 1 {
		t.Fatalf("customer creates = %d, want 1", d.customerCreates)
	}
	sc, err := f.st.GetStripeCustomer(context.Background(), f.project.ID, f.user.ID)
	if err != nil || sc.StripeCustomerID != testStripeCustomerID {
		t.Fatalf("mapping not refreshed: %+v (%v)", sc, err)
	}
	// Exactly one retry: the failed attempt with the stale id, then the fresh one.
	if n := len(d.checkoutForms); n != 2 {
		t.Fatalf("checkout creates = %d, want 2", n)
	}
	if d.checkoutForms[0].Get("customer") != "cus_stale" ||
		d.checkoutForms[1].Get("customer") != testStripeCustomerID {
		t.Fatalf("checkout customers = %q, %q",
			d.checkoutForms[0].Get("customer"), d.checkoutForms[1].Get("customer"))
	}
}

func TestCreateBillingPortalSessionStaleCustomer(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, "")
	if err := f.st.CreateStripeCustomer(context.Background(), store.StripeCustomer{
		ProjectID: f.project.ID, UserID: f.user.ID,
		StripeCustomerID: "cus_stale", CreatedAt: f.now,
	}); err != nil {
		t.Fatal(err)
	}
	d.set(func(d *stripeDouble) { d.portalMissing = true })
	// A stale mapping means there is no usable billing history: a permanent
	// FailedPrecondition, never a retryable STORE_UNAVAILABLE.
	_, err := f.h.CreateBillingPortalSession(f.ctx(), authReq(f, &billingv1.CreateBillingPortalSessionRequest{
		ReturnUrl: "https://app.demo.test/account",
	}))
	assertReason(t, err, authrpc.ReasonNoBillingHistory)
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want FailedPrecondition", connect.CodeOf(err))
	}
}

// --- generic client errors ---------------------------------------------------

func TestStripeCallErrorHidesDetails(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, "")
	f.setStripePriceID(testStripePriceID)
	d.set(func(d *stripeDouble) { d.checkoutDeclined = true })

	_, err := f.h.CreateCheckoutSession(f.ctx(), authReq(f, &billingv1.CreateCheckoutSessionRequest{
		ProductIdentifier: "monthly",
		SuccessUrl:        "https://app.demo.test/ok",
		CancelUrl:         "https://app.demo.test/cancel",
	}))
	assertReason(t, err, authrpc.ReasonStoreUnavailable)
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("code = %v, want Unavailable", connect.CodeOf(err))
	}
	var cerr *connect.Error
	if !errors.As(err, &cerr) {
		t.Fatalf("err = %v, want *connect.Error", err)
	}
	// Stripe's raw error (customer ids, key mode) must never reach the client.
	if cerr.Message() != "the store could not be reached" {
		t.Fatalf("message = %q, want generic message", cerr.Message())
	}
	for _, leak := range []string{"cus_secret_42", "sk_live", "declined"} {
		if strings.Contains(err.Error(), leak) {
			t.Fatalf("client error leaks %q: %v", leak, err)
		}
	}
}

// --- price re-point safety ---------------------------------------------------

func TestStripeWebhookRenewalAfterPriceRepointKeepsProduct(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	seedStripeSubscription(t, f, d)

	// A catalog price change re-points the tier at a brand-new Stripe price
	// (and, worst case, a new Stripe product): the existing subscriber's
	// webhooks keep reporting the OLD price id, which now matches nothing.
	ctx := context.Background()
	p, err := f.st.GetProduct(ctx, f.project.ID, f.prodID)
	if err != nil {
		t.Fatal(err)
	}
	p.StripePriceID = "price_new2"
	p.StripeProductID = "prod_other"
	p.UpdatedAt = f.now
	if err := f.st.UpdateProduct(ctx, p); err != nil {
		t.Fatal(err)
	}

	newEnd := f.now.Add(60 * 24 * time.Hour)
	d.set(func(d *stripeDouble) { d.periodEnd = newEnd })
	body := stripeSubEvent("evt_renew_old_price", "customer.subscription.updated", testStripeSubID, testStripeCustomerID)
	if err := f.deliverStripe(body); err != nil {
		t.Fatal(err)
	}
	// The product link (and with it the entitlement) survives the mismatch.
	sub, _ := f.st.GetSubscriptionByStoreID(ctx, f.project.ID,
		store.SubscriptionStoreStripe, testStripeSubID)
	if sub.ProductID != f.prodID {
		t.Fatalf("product id = %q, want %q (link stripped by price re-point)", sub.ProductID, f.prodID)
	}
	got, _ := f.h.GetCustomerInfo(f.ctx(), authReq(f, &billingv1.GetCustomerInfoRequest{}))
	if hasEntitlement(got.Msg.CustomerInfo, "pro") == nil {
		t.Fatal("entitlement must survive a catalog price re-point")
	}
	// The renewed event carries the store-reported (old) price.
	rows, err := storeRawEventRows(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[1].Type != store.SubscriptionEventRenewed ||
		rows[1].PriceAmountMicros != 4_990_000 || rows[1].Currency != "USD" {
		t.Fatalf("events = %+v", rows)
	}
}

func TestStripeWebhookMatchesProductByStripeProductID(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	f.setStripeCreds(d.srv.URL, testStripeWebhookSecret)
	// The tier already sells under a re-pointed price id, but its Stripe
	// product id still matches what the incoming subscription reports: a brand
	// new subscription on the old price resolves the tier via the product id.
	f.setStripePriceID("price_repointed") // StripeProductID stays "prod_1"
	body := stripeCheckoutEvent("evt_prodmatch", testStripeSubID, testStripeCustomerID, f.user.ID, f.project.ID)
	if err := f.deliverStripe(body); err != nil {
		t.Fatal(err)
	}
	sub, err := f.st.GetSubscriptionByStoreID(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, testStripeSubID)
	if err != nil {
		t.Fatal(err)
	}
	if sub.ProductID != f.prodID {
		t.Fatalf("product id = %q, want %q (stripe product id fallback)", sub.ProductID, f.prodID)
	}
	assertEventCount(t, f, store.SubscriptionEventPurchased, 1)
}

// --- failed-charge renewal suppression --------------------------------------

func TestStripeWebhookPastDueRenewalSuppressedUntilRecovery(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	seedStripeSubscription(t, f, d)

	// Stripe advances current_period_end even when the renewal charge FAILS
	// (status past_due): no revenue is booked, but the status flips and the
	// entitlement survives the billing retry.
	newEnd := f.now.Add(60 * 24 * time.Hour)
	d.set(func(d *stripeDouble) { d.subStatus = "past_due"; d.periodEnd = newEnd })
	if err := f.deliverStripe(stripeSubEvent("evt_pd_advance", "customer.subscription.updated",
		testStripeSubID, testStripeCustomerID)); err != nil {
		t.Fatal(err)
	}
	sub, _ := f.st.GetSubscriptionByStoreID(context.Background(), f.project.ID,
		store.SubscriptionStoreStripe, testStripeSubID)
	if sub.Status != store.SubscriptionStatusInBillingRetry {
		t.Fatalf("status = %q, want in_billing_retry", sub.Status)
	}
	if rows, err := storeRawEventRows(f); err != nil || len(rows) != 1 {
		t.Fatalf("failed charge must book nothing beyond the seed purchase: %+v (%v)", rows, err)
	}
	got, _ := f.h.GetCustomerInfo(f.ctx(), authReq(f, &billingv1.GetCustomerInfoRequest{}))
	if hasEntitlement(got.Msg.CustomerInfo, "pro") == nil {
		t.Fatal("entitlement must survive the billing retry")
	}

	// The retried charge succeeds later: past_due -> active with the period
	// end unchanged. THAT is when the deferred renewal books, exactly once.
	d.set(func(d *stripeDouble) { d.subStatus = "active" })
	if err := f.deliverStripe(stripeSubEvent("evt_pd_recovered", "customer.subscription.updated",
		testStripeSubID, testStripeCustomerID)); err != nil {
		t.Fatal(err)
	}
	rows, err := storeRawEventRows(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[1].Type != store.SubscriptionEventRenewed ||
		rows[1].PriceAmountMicros != 4_990_000 || rows[1].Currency != "USD" {
		t.Fatalf("recovery must book exactly one renewal with price: %+v", rows)
	}
}

func TestStripeWebhookCancelFlipWithPeriodAdvanceEmitsBoth(t *testing.T) {
	f := newFixture(t)
	d := newStripeDouble(f)
	seedStripeSubscription(t, f, d)

	// One event carries two transitions: the charge renewed the period AND the
	// user flipped off auto-renew. Both must book — a first-match chain would
	// swallow one.
	newEnd := f.now.Add(60 * 24 * time.Hour)
	d.set(func(d *stripeDouble) { d.cancelAtPeriodEnd = true; d.periodEnd = newEnd })
	if err := f.deliverStripe(stripeSubEvent("evt_cancel_and_renew", "customer.subscription.updated",
		testStripeSubID, testStripeCustomerID)); err != nil {
		t.Fatal(err)
	}
	assertEventCount(t, f, store.SubscriptionEventCanceled, 1)
	assertEventCount(t, f, store.SubscriptionEventRenewed, 1)
}

func TestGetOfferingsCarriesStripePriceID(t *testing.T) {
	f := newFixture(t)
	f.setStripePriceID(testStripePriceID)
	resp, err := f.h.GetOfferings(f.ctx(), connect.NewRequest(&billingv1.GetOfferingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.Offering.Products) != 1 ||
		resp.Msg.Offering.Products[0].StripePriceId != testStripePriceID {
		t.Fatalf("offering products = %+v", resp.Msg.Offering.Products)
	}
}
