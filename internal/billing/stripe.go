package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// StripeAPIBaseURL is the production Stripe REST API host, overridable for
// tests via StripeClient.BaseURL. Stripe has no separate sandbox host — test
// mode is selected by the secret key (sk_test_...) and reported back as
// livemode=false.
const StripeAPIBaseURL = "https://api.stripe.com"

// StripeClient calls the Stripe REST API directly (no Stripe SDK) with a
// project's restricted/secret key. Like the Apple and Google clients it is a
// struct literal with zero values meaning production defaults, so the whole
// engine stays testable against httptest doubles.
type StripeClient struct {
	// BaseURL defaults to StripeAPIBaseURL; tests point it at a double.
	BaseURL string
	// SecretKey is the project's sk_/rk_ secret key, sent as a Bearer token.
	SecretKey string
	HTTPC     Doer
	Now       func() time.Time
}

func (c *StripeClient) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return StripeAPIBaseURL
}

func (c *StripeClient) httpc() Doer {
	if c.HTTPC != nil {
		return c.HTTPC
	}
	return defaultDoer()
}

// do is the single request funnel: Bearer auth, form-encoded POST bodies
// (Stripe's wire format, bracketed keys included), JSON responses, 1MiB body
// cap, 404 wrapped as ErrNotFound. Mirrors GoogleClient.do.
func (c *StripeClient) do(ctx context.Context, method, path string, form url.Values, out any) error {
	var reader io.Reader
	if form != nil {
		reader = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL()+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.SecretKey)
	req.Header.Set("Accept", "application/json")
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := c.httpc().Do(req)
	if err != nil {
		return fmt.Errorf("stripe: %w", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("stripe: read response: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: %s", ErrNotFound, stripeErrMessage(payload))
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return stripeAPIError(resp.StatusCode, payload)
	}
	if out != nil {
		if err := json.Unmarshal(payload, out); err != nil {
			return fmt.Errorf("stripe: decode response: %w", err)
		}
	}
	return nil
}

// StripeAPIError is a non-2xx (non-404) Stripe API response. Callers use
// errors.As to branch on the stable Code (e.g. "resource_missing" when a
// stored customer or price id no longer exists in the account/mode) instead
// of string-matching the message.
type StripeAPIError struct {
	Status  int
	Code    string
	Type    string
	Message string
}

func (e *StripeAPIError) Error() string {
	msg := e.Message
	if e.Type != "" || e.Code != "" {
		msg = fmt.Sprintf("%s (type=%s, code=%s)", msg, e.Type, e.Code)
	}
	return fmt.Sprintf("stripe: status %d: %s", e.Status, msg)
}

func stripeAPIError(status int, body []byte) error {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &parsed)
	return &StripeAPIError{
		Status:  status,
		Code:    parsed.Error.Code,
		Type:    parsed.Error.Type,
		Message: parsed.Error.Message,
	}
}

// stripeErrMessage extracts Stripe's error envelope (.error.message/.type/
// .code) from a non-2xx body for the returned error.
func stripeErrMessage(body []byte) string {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &parsed)
	msg := parsed.Error.Message
	if parsed.Error.Type != "" || parsed.Error.Code != "" {
		msg = fmt.Sprintf("%s (type=%s, code=%s)", msg, parsed.Error.Type, parsed.Error.Code)
	}
	return msg
}

// setStripeMetadata form-encodes a metadata map under prefix, e.g.
// metadata[project_id]=... or subscription_data[metadata][user_id]=...
func setStripeMetadata(form url.Values, prefix string, metadata map[string]string) {
	for k, v := range metadata {
		form.Set(prefix+"["+k+"]", v)
	}
}

// ---- Customers ----

// StripeCustomer is the subset of a Stripe customer moth reads.
type StripeCustomer struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// CreateCustomer creates a Stripe customer (POST /v1/customers). moth creates
// at most one per (project, user), lazily on first checkout, carrying the moth
// ids in metadata.
func (c *StripeClient) CreateCustomer(ctx context.Context, email string, metadata map[string]string) (StripeCustomer, error) {
	form := url.Values{}
	if email != "" {
		form.Set("email", email)
	}
	setStripeMetadata(form, "metadata", metadata)
	var out StripeCustomer
	if err := c.do(ctx, http.MethodPost, "/v1/customers", form, &out); err != nil {
		return StripeCustomer{}, err
	}
	return out, nil
}

// ---- Checkout Sessions ----

// StripeCheckoutParams shapes a subscription-mode Checkout Session.
type StripeCheckoutParams struct {
	// PriceID is the recurring price the session subscribes to.
	PriceID string
	// CustomerID binds the session to an existing Stripe customer.
	CustomerID string
	SuccessURL string
	CancelURL  string
	// ClientReferenceID carries the moth user id back on
	// checkout.session.completed.
	ClientReferenceID string
	// Metadata is attached to BOTH the session and (via subscription_data)
	// the resulting subscription, so the subscription itself carries the moth
	// project/user ids.
	Metadata map[string]string
	// TrialPeriodDays > 0 starts the subscription with a free trial.
	TrialPeriodDays int64
}

// StripeCheckoutSession is the subset of a Checkout Session moth reads: on
// create the hosted URL to redirect to; on checkout.session.completed the
// created subscription/customer and the moth identity echoes.
type StripeCheckoutSession struct {
	ID                string            `json:"id"`
	URL               string            `json:"url"`
	Subscription      string            `json:"subscription"`
	Customer          string            `json:"customer"`
	ClientReferenceID string            `json:"client_reference_id"`
	Metadata          map[string]string `json:"metadata"`
}

// CreateCheckoutSession creates a subscription-mode hosted Checkout Session
// (POST /v1/checkout/sessions). Checkout stays Stripe-hosted: moth never
// renders a card field.
func (c *StripeClient) CreateCheckoutSession(ctx context.Context, params StripeCheckoutParams) (StripeCheckoutSession, error) {
	form := url.Values{
		"mode":                    {"subscription"},
		"line_items[0][price]":    {params.PriceID},
		"line_items[0][quantity]": {"1"},
		"success_url":             {params.SuccessURL},
		"cancel_url":              {params.CancelURL},
	}
	if params.CustomerID != "" {
		form.Set("customer", params.CustomerID)
	}
	if params.ClientReferenceID != "" {
		form.Set("client_reference_id", params.ClientReferenceID)
	}
	setStripeMetadata(form, "metadata", params.Metadata)
	setStripeMetadata(form, "subscription_data[metadata]", params.Metadata)
	if params.TrialPeriodDays > 0 {
		form.Set("subscription_data[trial_period_days]", strconv.FormatInt(params.TrialPeriodDays, 10))
	}
	var out StripeCheckoutSession
	if err := c.do(ctx, http.MethodPost, "/v1/checkout/sessions", form, &out); err != nil {
		return StripeCheckoutSession{}, err
	}
	return out, nil
}

// ---- Billing Portal ----

// StripePortalSession is a Billing Portal session: the URL is where the user
// manages payment methods, invoices and cancellation, Stripe-hosted.
type StripePortalSession struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// CreateBillingPortalSession creates a Billing Portal session (POST
// /v1/billing_portal/sessions) for a customer.
func (c *StripeClient) CreateBillingPortalSession(ctx context.Context, customerID, returnURL string) (StripePortalSession, error) {
	form := url.Values{
		"customer":   {customerID},
		"return_url": {returnURL},
	}
	var out StripePortalSession
	if err := c.do(ctx, http.MethodPost, "/v1/billing_portal/sessions", form, &out); err != nil {
		return StripePortalSession{}, err
	}
	return out, nil
}

// ---- Subscriptions ----

// StripeSubscription is the subset of a Stripe subscription moth reads.
type StripeSubscription struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
	// CurrentPeriodEnd (unix seconds) lives on the subscription in older
	// Stripe API versions; current versions expose it per item instead —
	// normalization reads both and prefers whichever is set.
	CurrentPeriodEnd int64  `json:"current_period_end"`
	Customer         string `json:"customer"`
	LiveMode         bool   `json:"livemode"`
	TrialEnd         int64  `json:"trial_end"`
	// PauseCollection is non-nil while payment collection is paused.
	PauseCollection *struct {
		Behavior string `json:"behavior"`
	} `json:"pause_collection"`
	Metadata map[string]string `json:"metadata"`
	Items    struct {
		Data []StripeSubscriptionItem `json:"data"`
	} `json:"items"`

	// Raw is the verbatim JSON this struct was decoded from (the API response
	// body or a webhook data.object), carried into RawState for audit.
	Raw json.RawMessage `json:"-"`
}

// StripeSubscriptionItem is the subset of a subscription item moth reads.
type StripeSubscriptionItem struct {
	// CurrentPeriodEnd (unix seconds) is where current Stripe API versions
	// report the period end.
	CurrentPeriodEnd int64 `json:"current_period_end"`
	Price            struct {
		ID         string `json:"id"`
		Product    string `json:"product"`
		UnitAmount int64  `json:"unit_amount"`
		Currency   string `json:"currency"`
		Recurring  struct {
			Interval      string `json:"interval"`
			IntervalCount int    `json:"interval_count"`
		} `json:"recurring"`
	} `json:"price"`
}

// GetSubscription resolves a Stripe subscription id (sub_...) to authoritative
// state (GET /v1/subscriptions/{id}) and returns it normalized alongside the
// partial raw decode. Webhooks are nudges: every event triggers this re-read.
// An unknown id surfaces as ErrNotFound.
func (c *StripeClient) GetSubscription(ctx context.Context, id string) (NormalizedSubscription, StripeSubscription, error) {
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodGet, "/v1/subscriptions/"+url.PathEscape(id), nil, &raw); err != nil {
		return NormalizedSubscription{}, StripeSubscription{}, err
	}
	var sub StripeSubscription
	if err := json.Unmarshal(raw, &sub); err != nil {
		return NormalizedSubscription{}, StripeSubscription{}, fmt.Errorf("stripe: decode subscription: %w", err)
	}
	sub.Raw = raw
	return normalizeStripeSubscription(sub), sub, nil
}

// normalizeStripeSubscription maps a Stripe subscription into the normalized
// model. Status mapping (plan/17 §Model):
//
//	active                        -> active (paused when pause_collection set)
//	trialing                      -> trialing
//	past_due                      -> in_billing_retry (granted — same "never
//	                                 lock out a paying user over a card
//	                                 hiccup" policy as the mobile stores)
//	paused                        -> paused
//	canceled / unpaid /
//	incomplete_expired /
//	incomplete / other            -> expired ("incomplete" means the first
//	                                 payment never succeeded — the user was
//	                                 never entitled, so it maps to the
//	                                 not-granted bucket rather than a retry
//	                                 state)
func normalizeStripeSubscription(sub StripeSubscription) NormalizedSubscription {
	norm := NormalizedSubscription{
		Store:              StoreStripe,
		StoreTransactionID: sub.ID,
		// SubscriptionID carries the Stripe customer id (cus_...): the
		// subscription id is already the store transaction identity, and the
		// customer is the secondary identity moth needs to open the Billing
		// Portal and correlate checkout events.
		SubscriptionID: sub.Customer,
		Status:         stripeStatus(sub),
		AutoRenew:      !sub.CancelAtPeriodEnd,
		Environment:    EnvSandbox,
	}
	if sub.LiveMode {
		norm.Environment = EnvProduction
	}
	end := sub.CurrentPeriodEnd
	if len(sub.Items.Data) > 0 {
		item := sub.Items.Data[0]
		// ProductID is the Stripe price id — the value moth stores as the
		// tier's stripe_price_id, so product matching compares against it.
		norm.ProductID = item.Price.ID
		// unit_amount is in the currency's minor units; moth stores micros
		// of whole currency units. The factor depends on the currency's
		// decimal count (JPY has none, BHD has three).
		norm.Currency = strings.ToUpper(item.Price.Currency)
		norm.PriceAmountMicros = StripeMicrosForUnitAmount(item.Price.UnitAmount, norm.Currency)
		if end == 0 {
			end = item.CurrentPeriodEnd
		}
	}
	if end != 0 {
		norm.CurrentPeriodEnd = time.Unix(end, 0).UTC()
	}
	if len(sub.Raw) > 0 {
		norm.RawState = sub.Raw
	} else {
		norm.RawState, _ = json.Marshal(sub)
	}
	return norm
}

// Stripe denominates unit_amount in a currency's MINOR units, and the size
// of the minor unit varies: most currencies have two decimals (unit_amount
// is cents), the currencies below have zero (unit_amount is whole units) or
// three. moth stores micros of whole currency units everywhere, so every
// conversion must go through the per-currency factor — a hard-coded /10_000
// overcharges JPY customers 100x. Sets follow Stripe's documented
// "zero-decimal" and "three-decimal" currency lists.
var stripeZeroDecimalCurrencies = map[string]bool{
	"BIF": true, "CLP": true, "DJF": true, "GNF": true, "JPY": true,
	"KMF": true, "KRW": true, "MGA": true, "PYG": true, "RWF": true,
	"UGX": true, "VND": true, "VUV": true, "XAF": true, "XOF": true,
	"XPF": true,
}

var stripeThreeDecimalCurrencies = map[string]bool{
	"BHD": true, "JOD": true, "KWD": true, "OMR": true, "TND": true,
}

// stripeMicrosPerUnitAmount returns how many moth micros one Stripe
// unit_amount step represents for the currency.
func stripeMicrosPerUnitAmount(currency string) int64 {
	switch cur := strings.ToUpper(currency); {
	case stripeZeroDecimalCurrencies[cur]:
		return 1_000_000
	case stripeThreeDecimalCurrencies[cur]:
		return 1_000
	default:
		return 10_000
	}
}

// StripeUnitAmountForMicros converts a moth price (micros of whole currency
// units) to Stripe's unit_amount for the currency. Amounts that are not
// representable in the currency's minor unit are rejected rather than
// silently rounded.
func StripeUnitAmountForMicros(micros int64, currency string) (int64, error) {
	factor := stripeMicrosPerUnitAmount(currency)
	if micros%factor != 0 {
		return 0, fmt.Errorf("price %d micros is not representable in %s's minor unit", micros, strings.ToUpper(currency))
	}
	return micros / factor, nil
}

// StripeMicrosForUnitAmount converts Stripe's unit_amount back to moth
// micros for the currency.
func StripeMicrosForUnitAmount(unitAmount int64, currency string) int64 {
	return unitAmount * stripeMicrosPerUnitAmount(currency)
}

func stripeStatus(sub StripeSubscription) string {
	switch sub.Status {
	case "active":
		if sub.PauseCollection != nil && sub.PauseCollection.Behavior != "" {
			return StatusPaused
		}
		return StatusActive
	case "trialing":
		return StatusTrialing
	case "past_due":
		return StatusInBillingRetry
	case "paused":
		return StatusPaused
	default: // canceled, unpaid, incomplete, incomplete_expired
		return StatusExpired
	}
}

// ---- Products & Prices (catalog provisioning) ----

// StripeProduct is the subset of a Stripe product moth reads.
type StripeProduct struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CreateProduct creates a Stripe product (POST /v1/products) during catalog
// provisioning, carrying the moth tier identity in metadata.
func (c *StripeClient) CreateProduct(ctx context.Context, name string, metadata map[string]string) (StripeProduct, error) {
	form := url.Values{"name": {name}}
	setStripeMetadata(form, "metadata", metadata)
	var out StripeProduct
	if err := c.do(ctx, http.MethodPost, "/v1/products", form, &out); err != nil {
		return StripeProduct{}, err
	}
	return out, nil
}

// GetProduct fetches a product by id (GET /v1/products/{id}) so provisioning
// can detect display-name drift between the moth catalog and the live Stripe
// product. An unknown id surfaces as ErrNotFound.
func (c *StripeClient) GetProduct(ctx context.Context, id string) (StripeProduct, error) {
	var out StripeProduct
	if err := c.do(ctx, http.MethodGet, "/v1/products/"+url.PathEscape(id), nil, &out); err != nil {
		return StripeProduct{}, err
	}
	return out, nil
}

// UpdateProduct renames a product in place (POST /v1/products/{id}). Unlike
// prices, Stripe product names are mutable, so a moth display-name change is a
// real update rather than a recreate.
func (c *StripeClient) UpdateProduct(ctx context.Context, id, name string) (StripeProduct, error) {
	form := url.Values{"name": {name}}
	var out StripeProduct
	if err := c.do(ctx, http.MethodPost, "/v1/products/"+url.PathEscape(id), form, &out); err != nil {
		return StripeProduct{}, err
	}
	return out, nil
}

// StripePriceParams shapes a recurring price for a provisioned product.
type StripePriceParams struct {
	ProductID string
	// Currency is ISO-4217 in any case; Stripe is sent the lowercase form.
	Currency string
	// UnitAmount is in the currency's minor units (cents).
	UnitAmount int64
	// Interval is day|week|month|year (see StripeRecurringForPeriod).
	Interval      string
	IntervalCount int
	Metadata      map[string]string
}

// StripePrice is the subset of a Stripe price moth reads, flattened. Currency
// is upper-cased ISO-4217 to compare directly against the moth catalog.
type StripePrice struct {
	ID            string
	ProductID     string
	UnitAmount    int64
	Currency      string
	Interval      string
	IntervalCount int
	Active        bool
}

// stripePriceResponse is the wire shape of a Stripe price.
type stripePriceResponse struct {
	ID         string `json:"id"`
	Product    string `json:"product"`
	UnitAmount int64  `json:"unit_amount"`
	Currency   string `json:"currency"`
	Active     bool   `json:"active"`
	Recurring  *struct {
		Interval      string `json:"interval"`
		IntervalCount int    `json:"interval_count"`
	} `json:"recurring"`
}

func (r stripePriceResponse) flatten() StripePrice {
	p := StripePrice{
		ID:         r.ID,
		ProductID:  r.Product,
		UnitAmount: r.UnitAmount,
		Currency:   strings.ToUpper(r.Currency),
		Active:     r.Active,
	}
	if r.Recurring != nil {
		p.Interval = r.Recurring.Interval
		p.IntervalCount = r.Recurring.IntervalCount
	}
	return p
}

// CreatePrice creates a recurring price (POST /v1/prices). Stripe prices are
// immutable: a price change creates a new price and re-points the tier.
func (c *StripeClient) CreatePrice(ctx context.Context, params StripePriceParams) (StripePrice, error) {
	form := url.Values{
		"product":             {params.ProductID},
		"currency":            {strings.ToLower(params.Currency)},
		"unit_amount":         {strconv.FormatInt(params.UnitAmount, 10)},
		"recurring[interval]": {params.Interval},
	}
	if params.IntervalCount > 0 {
		form.Set("recurring[interval_count]", strconv.Itoa(params.IntervalCount))
	}
	setStripeMetadata(form, "metadata", params.Metadata)
	var out stripePriceResponse
	if err := c.do(ctx, http.MethodPost, "/v1/prices", form, &out); err != nil {
		return StripePrice{}, err
	}
	return out.flatten(), nil
}

// GetPrice fetches a price by id (GET /v1/prices/{id}) so provisioning can
// detect drift between the moth catalog and the live Stripe price. An unknown
// id surfaces as ErrNotFound.
func (c *StripeClient) GetPrice(ctx context.Context, id string) (StripePrice, error) {
	var out stripePriceResponse
	if err := c.do(ctx, http.MethodGet, "/v1/prices/"+url.PathEscape(id), nil, &out); err != nil {
		return StripePrice{}, err
	}
	return out.flatten(), nil
}

// ---- Webhook endpoints (setup) ----

// StripeWebhookEndpoint is the subset of a Stripe webhook endpoint moth reads.
// Secret (whsec_...) is returned by Stripe only on create.
type StripeWebhookEndpoint struct {
	ID            string   `json:"id"`
	URL           string   `json:"url"`
	Status        string   `json:"status"`
	EnabledEvents []string `json:"enabled_events"`
	Secret        string   `json:"secret"`
}

// ListWebhookEndpoints lists ALL of the account's webhook endpoints (GET
// /v1/webhook_endpoints) so setup can detect an already-provisioned
// endpoint. Stripe pages the list (default 10) — a moth instance hosting a
// portfolio of projects easily exceeds that, and a missed endpoint would
// make setup create duplicates and rotate the stored signing secret — so
// this follows has_more/starting_after to exhaustion.
func (c *StripeClient) ListWebhookEndpoints(ctx context.Context) ([]StripeWebhookEndpoint, error) {
	var all []StripeWebhookEndpoint
	startingAfter := ""
	for {
		path := "/v1/webhook_endpoints?limit=100"
		if startingAfter != "" {
			path += "&starting_after=" + url.QueryEscape(startingAfter)
		}
		var out struct {
			Data    []StripeWebhookEndpoint `json:"data"`
			HasMore bool                    `json:"has_more"`
		}
		if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
			return nil, err
		}
		all = append(all, out.Data...)
		if !out.HasMore || len(out.Data) == 0 {
			return all, nil
		}
		startingAfter = out.Data[len(out.Data)-1].ID
	}
}

// UpdateWebhookEndpoint re-points an existing endpoint's event subscription
// and re-enables it if Stripe disabled it (POST /v1/webhook_endpoints/{id}).
// The signing secret is NOT returned on update — only create reveals it.
func (c *StripeClient) UpdateWebhookEndpoint(ctx context.Context, id string, events []string) (StripeWebhookEndpoint, error) {
	form := url.Values{"disabled": {"false"}}
	for _, e := range events {
		form.Add("enabled_events[]", e)
	}
	var out StripeWebhookEndpoint
	if err := c.do(ctx, http.MethodPost, "/v1/webhook_endpoints/"+url.PathEscape(id), form, &out); err != nil {
		return StripeWebhookEndpoint{}, err
	}
	return out, nil
}

// CreateWebhookEndpoint creates a webhook endpoint (POST
// /v1/webhook_endpoints) subscribed to events. The response carries the
// signing Secret exactly once — the caller stores it encrypted.
func (c *StripeClient) CreateWebhookEndpoint(ctx context.Context, endpointURL string, events []string) (StripeWebhookEndpoint, error) {
	form := url.Values{"url": {endpointURL}}
	for _, e := range events {
		form.Add("enabled_events[]", e)
	}
	var out StripeWebhookEndpoint
	if err := c.do(ctx, http.MethodPost, "/v1/webhook_endpoints", form, &out); err != nil {
		return StripeWebhookEndpoint{}, err
	}
	return out, nil
}

// ---- Webhook signature verification ----

// VerifyStripeSignature verifies a Stripe-Signature header
// (t=<unix>,v1=<hex>[,v1=<hex>...]) against the endpoint's signing secret:
// HMAC-SHA256 over "<t>.<payload>", constant-time compare against every v1
// (Stripe sends several during secret rolls), and a ±tolerance window on t
// (skipped when tolerance <= 0). A malformed header wraps ErrMalformed; every
// other failure — empty secret included — wraps ErrInvalidSignature.
func VerifyStripeSignature(payload []byte, header, secret string, now time.Time, tolerance time.Duration) error {
	if secret == "" {
		return fmt.Errorf("%w: empty stripe webhook secret", ErrInvalidSignature)
	}
	if header == "" {
		return fmt.Errorf("%w: missing Stripe-Signature header", ErrMalformed)
	}
	var (
		ts     int64
		tsSeen bool
		sigs   [][]byte
	)
	for _, part := range strings.Split(header, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch k {
		case "t":
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return fmt.Errorf("%w: stripe signature timestamp: %v", ErrMalformed, err)
			}
			ts, tsSeen = n, true
		case "v1":
			sig, err := hex.DecodeString(v)
			if err != nil {
				continue // an undecodable v1 simply never matches
			}
			sigs = append(sigs, sig)
		}
	}
	if !tsSeen {
		return fmt.Errorf("%w: stripe signature missing timestamp", ErrMalformed)
	}
	if len(sigs) == 0 {
		return fmt.Errorf("%w: stripe signature missing v1", ErrMalformed)
	}
	if tolerance > 0 {
		if d := now.Sub(time.Unix(ts, 0)); d > tolerance || d < -tolerance {
			return fmt.Errorf("%w: stripe signature timestamp outside tolerance", ErrInvalidSignature)
		}
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	want := mac.Sum(nil)
	for _, sig := range sigs {
		if hmac.Equal(sig, want) {
			return nil
		}
	}
	return fmt.Errorf("%w: no matching stripe v1 signature", ErrInvalidSignature)
}

// ---- Event envelope ----

// StripeEvent is the webhook event envelope. Data.Object is the raw embedded
// object, decoded on demand via CheckoutSession/SubscriptionObject — and only
// ever treated as a nudge: state is re-read from the API before applying.
type StripeEvent struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	LiveMode bool   `json:"livemode"`
	Data     struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

// ParseStripeEvent parses a verified webhook body into the event envelope. The
// event id (evt_...) is the dedupe key. Malformed JSON or a missing id/type
// wraps ErrMalformed.
func ParseStripeEvent(payload []byte) (StripeEvent, error) {
	var ev StripeEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return StripeEvent{}, fmt.Errorf("%w: stripe event: %v", ErrMalformed, err)
	}
	if ev.ID == "" || ev.Type == "" {
		return StripeEvent{}, fmt.Errorf("%w: stripe event missing id or type", ErrMalformed)
	}
	return ev, nil
}

// CheckoutSession decodes Data.Object as a Checkout Session
// (checkout.session.completed events).
func (e StripeEvent) CheckoutSession() (StripeCheckoutSession, error) {
	var s StripeCheckoutSession
	if err := json.Unmarshal(e.Data.Object, &s); err != nil {
		return StripeCheckoutSession{}, fmt.Errorf("%w: stripe checkout session object: %v", ErrMalformed, err)
	}
	return s, nil
}

// SubscriptionObject decodes Data.Object as a subscription
// (customer.subscription.* events), keeping the verbatim object as Raw.
func (e StripeEvent) SubscriptionObject() (StripeSubscription, error) {
	var s StripeSubscription
	if err := json.Unmarshal(e.Data.Object, &s); err != nil {
		return StripeSubscription{}, fmt.Errorf("%w: stripe subscription object: %v", ErrMalformed, err)
	}
	s.Raw = append(json.RawMessage(nil), e.Data.Object...)
	return s, nil
}

// ---- Catalog vocabulary mapping (provisioning) ----

// StripeRecurringForPeriod maps a moth product billing_period onto Stripe's
// recurring[interval]/[interval_count]. It accepts the same free-form
// vocabulary as the Apple/Google catalog sync (setup.parseBillingPeriod):
// words ("weekly", "monthly", "two_month", "quarterly", "half_year",
// "yearly" and their synonyms) and the ISO-8601 forms P1W/P1M/P2M/P3M/P6M/P1Y.
func StripeRecurringForPeriod(period string) (interval string, count int, err error) {
	switch strings.ToLower(strings.TrimSpace(period)) {
	case "weekly", "week", "p1w":
		return "week", 1, nil
	case "monthly", "month", "p1m":
		return "month", 1, nil
	case "two_month", "twomonth", "2month", "p2m":
		return "month", 2, nil
	case "quarterly", "quarter", "3month", "p3m":
		return "month", 3, nil
	case "half_year", "halfyear", "6month", "p6m":
		return "month", 6, nil
	case "yearly", "annual", "year", "p1y":
		return "year", 1, nil
	}
	return "", 0, fmt.Errorf("billing: unsupported billing period %q", period)
}

// StripeTrialDays maps a moth product trial_period onto Stripe's
// subscription_data[trial_period_days]. It accepts the same vocabulary as
// StripeRecurringForPeriod plus arbitrary ISO-8601 durations of a single unit
// (P3D, P2W, ...); calendar units use the usual 30-day month / 365-day year
// approximation since Stripe trials are day-denominated. Empty means no trial
// (0, nil).
func StripeTrialDays(trial string) (int64, error) {
	s := strings.ToLower(strings.TrimSpace(trial))
	if s == "" {
		return 0, nil
	}
	switch s {
	case "weekly", "week":
		return 7, nil
	case "monthly", "month":
		return 30, nil
	case "two_month", "twomonth", "2month":
		return 60, nil
	case "quarterly", "quarter", "3month":
		return 90, nil
	case "half_year", "halfyear", "6month":
		return 180, nil
	case "yearly", "annual", "year":
		return 365, nil
	}
	if len(s) >= 3 && s[0] == 'p' {
		if n, err := strconv.ParseInt(s[1:len(s)-1], 10, 64); err == nil && n > 0 {
			switch s[len(s)-1] {
			case 'd':
				return n, nil
			case 'w':
				return n * 7, nil
			case 'm':
				return n * 30, nil
			case 'y':
				return n * 365, nil
			}
		}
	}
	return 0, fmt.Errorf("billing: unsupported trial period %q", trial)
}
