package setup

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
)

// newStripeBillingSetup builds a Stripe-only BillingSetup against the stripe
// double (no Apple/Google inputs — a web-only project, plan/17).
func newStripeBillingSetup(t *testing.T, d *stripeDouble, out *bytes.Buffer) (*BillingSetup, *fakeBillingCreds, *fakeProducts) {
	t.Helper()
	creds := &fakeBillingCreds{}
	products := &fakeProducts{products: testProducts()}
	s := &BillingSetup{
		Projects:     &fakeProjects{projects: []*adminv1.Project{testProject("demo")}},
		Products:     products,
		BillingCreds: creds,
		Prompt:       NewPrompter(strings.NewReader(""), out),
		Out:          out,
		BaseURL:      "https://moth.example.com",
		Slug:         "demo",
		Yes:          true,
		HTTPC:        http.DefaultClient,

		StripeSecretKey: "sk_test_moth",
		StripeBaseURL:   d.srv.URL,
	}
	return s, creds, products
}

func TestBillingSetupStripeLeg(t *testing.T) {
	d := newStripeDouble(t)
	var out bytes.Buffer
	s, creds, products := newStripeBillingSetup(t, d, &out)

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Credentials stored (write-only key), then the webhook creation persisted
	// the signing secret + endpoint id in a second write.
	if creds.stripe == nil || !creds.stripe.HasSecretKey {
		t.Fatalf("stripe key not stored: %+v", creds.stripe)
	}
	if !creds.stripe.HasWebhookSecret || !strings.HasPrefix(creds.stripe.WebhookEndpointId, "we_") {
		t.Fatalf("webhook secret/endpoint not persisted: %+v", creds.stripe)
	}
	assertCheck(t, rep, "Stripe: catalog pushed", StatusPass)
	assertCheck(t, rep, "Stripe: webhook endpoint", StatusPass)
	assertCheck(t, rep, "Stripe: API reachable", StatusPass)
	if rep.Failed() {
		t.Fatalf("report has failures:\n%s", reportString(rep))
	}
	// The webhook endpoint targets moth's project-scoped route with the moth
	// event set.
	if len(d.endpoints) != 1 || d.endpoints[0]["url"] != "https://moth.example.com/billing/stripe/webhook/demo" {
		t.Fatalf("endpoints = %+v", d.endpoints)
	}
	// Provisioned ids were written back onto the moth product.
	if len(products.updates) != 1 {
		t.Fatalf("expected 1 product update (id write-back), got %d", len(products.updates))
	}
	up := products.updates[0].Product
	if !strings.HasPrefix(up.GetStripePriceId(), "price_") || !strings.HasPrefix(up.GetStripeProductId(), "prod_") {
		t.Fatalf("write-back = price %q product %q", up.GetStripePriceId(), up.GetStripeProductId())
	}

	// Second run: catalog in sync, webhook already registered, no re-writes.
	rep2, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	assertCheck(t, rep2, "Stripe: catalog in sync", StatusPass)
	assertCheck(t, rep2, "Stripe: webhook endpoint", StatusPass)
	if hasCheck(rep2, "Stripe: catalog pushed") {
		t.Fatalf("second run should report no changes:\n%s", reportString(rep2))
	}
	if len(products.updates) != 1 {
		t.Fatalf("second run re-wrote product ids: %d updates", len(products.updates))
	}
	if len(d.endpoints) != 1 {
		t.Fatalf("second run created another endpoint: %d", len(d.endpoints))
	}
}

func TestBillingSetupStripeMissingKeyWarns(t *testing.T) {
	d := newStripeDouble(t)
	var out bytes.Buffer
	s, creds, _ := newStripeBillingSetup(t, d, &out)
	// Stripe requested but no key in hand this run (moth never returns the
	// stored one): credentials kept, push degraded to a warning — never a
	// hard error.
	s.StripeSecretKey = ""
	s.Stripe = true

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	assertCheck(t, rep, "Stripe: catalog push", StatusWarn)
	if rep.Failed() {
		t.Fatalf("missing key must not fail:\n%s", reportString(rep))
	}
	// The credential write still ran (empty key keeps the stored one).
	if creds.updates != 1 {
		t.Fatalf("expected 1 credential write, got %d", creds.updates)
	}
	// No live Stripe call was attempted.
	if len(d.posts) != 0 {
		t.Fatalf("no key in hand but Stripe was called: %v", d.posts)
	}
}

// TestBillingSetupStripeOnlyKeepsAppleGoogleCredentials: a Stripe-only run
// against a project already configured for Apple + Google must leave the
// Apple/Google configuration untouched — the credential update carries only a
// Stripe section, and the server merges per store instead of writing a blank
// full row (the wipe would silently break Apple/Google webhook processing).
func TestBillingSetupStripeOnlyKeepsAppleGoogleCredentials(t *testing.T) {
	d := newStripeDouble(t)
	var out bytes.Buffer
	s, creds, _ := newStripeBillingSetup(t, d, &out)
	// The project was previously configured for Apple and Google.
	creds.apple = &adminv1.AppleBillingConfig{
		IapKeyId: "IAPKEY0001", IapIssuerId: "iss-1", BundleId: "com.example.demo",
		AppAppleId: "1234567890", HasIapKey: true, HasNotificationSecret: true,
		NotificationUrl: "https://moth.example.com/billing/apple/notifications/demo",
	}
	creds.google = &adminv1.GoogleBillingConfig{
		PackageName: "com.example.app", PubsubTopic: "projects/demo-proj/topics/moth-rtdn",
		HasServiceAccount: true, HasRtdnSecret: true,
	}

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Failed() {
		t.Fatalf("report has failures:\n%s", reportString(rep))
	}
	// Stripe got configured...
	if creds.stripe == nil || !creds.stripe.HasSecretKey || !creds.stripe.HasWebhookSecret {
		t.Fatalf("stripe creds = %+v", creds.stripe)
	}
	// ...and every Apple/Google field survived the Stripe-only writes
	// (storeCredentials AND the webhook-secret persist).
	a := creds.apple
	if a.IapKeyId != "IAPKEY0001" || a.IapIssuerId != "iss-1" || a.BundleId != "com.example.demo" ||
		a.AppAppleId != "1234567890" || !a.HasIapKey || !a.HasNotificationSecret ||
		a.NotificationUrl != "https://moth.example.com/billing/apple/notifications/demo" {
		t.Fatalf("apple creds wiped by a stripe-only run: %+v", a)
	}
	g := creds.google
	if g.PackageName != "com.example.app" || g.PubsubTopic != "projects/demo-proj/topics/moth-rtdn" ||
		!g.HasServiceAccount || !g.HasRtdnSecret {
		t.Fatalf("google creds wiped by a stripe-only run: %+v", g)
	}
}

// TestBillingSetupStripeUnmappablePeriodFails: a product whose billing period
// cannot map onto a Stripe cadence must surface as an explicit per-product
// failure — never silently dropped while the run claims the catalog is in sync.
func TestBillingSetupStripeUnmappablePeriodFails(t *testing.T) {
	d := newStripeDouble(t)
	var out bytes.Buffer
	s, _, products := newStripeBillingSetup(t, d, &out)
	products.products = append(products.products, &adminv1.Product{
		Id: "prod2", Identifier: "typo", DisplayName: "Typo",
		BillingPeriod:     "montly", // sic — the operator's typo must not vanish
		PriceAmountMicros: 4_990_000, Currency: "USD",
	})

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	check := findCheck(t, rep, "Stripe: product typo")
	if check.Status != StatusFail || !strings.Contains(check.Detail, "montly") {
		t.Fatalf("unmappable period check = %+v, want fail naming the period", check)
	}
	if !rep.Failed() {
		t.Fatal("report must fail overall — the product will not sync")
	}
	// The good product still synced; nothing was provisioned for the bad one.
	assertCheck(t, rep, "Stripe: catalog pushed", StatusPass)
	if len(d.products) != 1 {
		t.Fatalf("expected only the good product provisioned, got %d", len(d.products))
	}
}

// TestBillingSetupStripeWebhookRepairSurfaced: a Stripe-disabled endpoint (or
// one missing moth events) is repaired in place and the checklist says so.
func TestBillingSetupStripeWebhookRepairSurfaced(t *testing.T) {
	d := newStripeDouble(t)
	var out bytes.Buffer
	s, creds, _ := newStripeBillingSetup(t, d, &out)
	// moth created this endpoint in a previous run (secret stored), and Stripe
	// has since disabled it after too many failed deliveries.
	creds.stripe = &adminv1.StripeBillingConfig{HasWebhookSecret: true, WebhookEndpointId: "we_disabled"}
	d.endpoints = append(d.endpoints, map[string]any{
		"id": "we_disabled", "url": "https://moth.example.com/billing/stripe/webhook/demo",
		"status": "disabled", "enabled_events": StripeWebhookEvents,
	})

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	check := findCheck(t, rep, "Stripe: webhook endpoint")
	if check.Status != StatusPass || !strings.Contains(check.Detail, "re-enabled / events updated") {
		t.Fatalf("webhook check = %+v, want pass with the repair surfaced", check)
	}
	if d.endpoints[0]["status"] != "enabled" {
		t.Fatalf("endpoint left disabled: %+v", d.endpoints[0])
	}
	if len(d.endpoints) != 1 {
		t.Fatalf("repair created a duplicate endpoint: %d", len(d.endpoints))
	}
}

// TestBillingSetupStripePriceFailureRecordsProductID: when provisioning fails
// at the price stage, the created Product id is still written back onto moth's
// product so the next run reuses it instead of creating a duplicate.
func TestBillingSetupStripePriceFailureRecordsProductID(t *testing.T) {
	d := newStripeDouble(t)
	var out bytes.Buffer
	s, _, products := newStripeBillingSetup(t, d, &out)
	d.failPriceCreate = true

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !rep.Failed() {
		t.Fatalf("price-stage failure must fail the report:\n%s", reportString(rep))
	}
	// The Product id was written back despite the missing Price.
	p := products.products[0]
	if !strings.HasPrefix(p.GetStripeProductId(), "prod_") || p.GetStripePriceId() != "" {
		t.Fatalf("write-back after price failure = product %q price %q", p.GetStripeProductId(), p.GetStripePriceId())
	}
	recordedProduct := p.GetStripeProductId()

	// The next run reuses the recorded Product — no duplicate — and completes.
	d.failPriceCreate = false
	rep2, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	assertCheck(t, rep2, "Stripe: catalog pushed", StatusPass)
	if len(d.products) != 1 {
		t.Fatalf("re-run provisioned a duplicate Product: %d", len(d.products))
	}
	if products.products[0].GetStripeProductId() != recordedProduct {
		t.Fatalf("product id changed across runs: %q -> %q", recordedProduct, products.products[0].GetStripeProductId())
	}
	if !strings.HasPrefix(products.products[0].GetStripePriceId(), "price_") {
		t.Fatalf("price id not recorded on the re-run: %+v", products.products[0])
	}
}

// TestBillingSetupStripeWebhookSecretUnknownWarns covers the honest remediation
// when the endpoint already exists but moth holds no signing secret: Stripe
// reveals the secret only at creation, so the run cannot recover it and must
// say so instead of passing.
func TestBillingSetupStripeWebhookSecretUnknownWarns(t *testing.T) {
	d := newStripeDouble(t)
	var out bytes.Buffer
	s, creds, _ := newStripeBillingSetup(t, d, &out)
	// The endpoint pre-exists (created out-of-band; secret never seen by moth).
	d.endpoints = append(d.endpoints, map[string]any{
		"id": "we_preexisting", "url": "https://moth.example.com/billing/stripe/webhook/demo",
		"status": "enabled", "secret": "whsec_never_revealed_again",
	})

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	check := findCheck(t, rep, "Stripe: webhook endpoint")
	if check.Status != StatusWarn {
		t.Fatalf("webhook check = %+v, want warn", check)
	}
	if !strings.Contains(check.Remediation, "delete the endpoint") {
		t.Fatalf("remediation does not explain the delete+rerun path: %q", check.Remediation)
	}
	if creds.stripe.HasWebhookSecret {
		t.Fatal("no secret should have been stored")
	}
	if len(d.endpoints) != 1 {
		t.Fatalf("a duplicate endpoint was created: %d", len(d.endpoints))
	}
}
