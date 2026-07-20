package setup

// Store-catalog provisioning (milestone 12).
//
// This file and its catalog_apple.go / catalog_google.go siblings push moth's
// subscription tier definitions into App Store Connect and Google Play — the
// monetization counterpart to `moth setup google|apple`. Same honest-automation
// contract as milestone 08: automate every step the official APIs expose, fall
// back to a precise guided checklist (exact paste values) where they don't, and
// keep every run idempotent (read store state, diff, change only deltas).
//
// # Capability spike — what each store's API actually allows
//
// Apple — App Store Connect API (JWT-authed with the ASC .p8, reusing the
// milestone-08 ASC client):
//   - Subscription GROUPS: create/list under an app
//     (/v1/apps/{id}/subscriptionGroups, POST /v1/subscriptionGroups). AUTOMATED.
//   - Auto-renewable SUBSCRIPTIONS: create/list in a group with productId, a
//     reference name, subscriptionPeriod and groupLevel
//     (/v1/subscriptionGroups/{id}/subscriptions, POST /v1/subscriptions).
//     AUTOMATED (create + name/level update).
//   - PRICES: Apple does NOT accept an arbitrary amount — every price is a fixed
//     "price point" off a per-territory ladder. We resolve the closest price
//     point in the base territory (/v1/subscriptions/{id}/pricePoints?filter…)
//     and attach it (POST /v1/subscriptionPrices). AUTOMATED for the base
//     territory; the full per-territory price SCHEDULE, preserve-current-price
//     nuances and promotional offers stay GUIDED.
//   - LOCALIZATIONS: name + description per locale
//     (/v1/subscriptionLocalizations). AUTOMATED (create/update).
//   - AVAILABILITY and "submit for review" of a new subscription: NO usable API
//     surface for the review submission of the first build of a subscription.
//     GUIDED, with the exact console page and values (this is the canonical
//     "we can't automate this, here's precisely what to click").
//   - App Store Server Notification V2 URL: there is no stable public ASC API to
//     set the production/sandbox server-notification URL. We ATTEMPT it (behind
//     an injectable endpoint, so it lights up if Apple ships one) and degrade to
//     a GUIDED step on 404 — mirroring CreateSignInWithAppleKey's 404 fallback.
//
// Google — Android Publisher API (service-account OAuth2, reusing the
// milestone-11 billing.GoogleTokenSource):
//   - SUBSCRIPTIONS with BASE PLANS and REGIONAL PRICES: create/patch
//     (/androidpublisher/v3/applications/{pkg}/subscriptions). AUTOMATED,
//     including base-plan activation (…/basePlans/{id}:activate).
//   - Price OFFERS (intro/free-trial) are a separate resource
//     (…/basePlans/{id}/offers). moth models an optional intro offer on a tier
//     (IntroOffer); creating the offer resource is a documented DEFERRAL — the
//     current sync provisions the base plan + base price only.
//   - Play Console AVAILABILITY / countries beyond the base region: GUIDED.
//   - RTDN: the Cloud Pub/Sub topic + push subscription are created via the
//     Pub/Sub Admin API (behind an injectable endpoint) — but that needs a
//     PUBSUB-scoped credential, distinct from the androidpublisher-scoped
//     billing SA, and POINTING Play Console at the topic has NO Android
//     Publisher API. So topic/push creation is AUTOMATED when a pubsub-scoped
//     token is supplied; wiring Play → topic is always GUIDED with exact values.
//
// Every external call sits behind an injectable HTTP Doer / endpoint and every
// prompt behind io streams, so the whole thing is testable against httptest
// doubles with no network, exactly like billing and the milestone-08 setup.

// BillingPeriod is a subscription renewal cadence, declared once and mapped to
// each store's own vocabulary (see applePeriod / googlePeriod).
type BillingPeriod string

// Supported billing periods. Kept to the cadences both stores share.
const (
	PeriodWeekly    BillingPeriod = "weekly"
	PeriodMonthly   BillingPeriod = "monthly"
	PeriodTwoMonth  BillingPeriod = "two_month"
	PeriodQuarterly BillingPeriod = "quarterly"
	PeriodHalfYear  BillingPeriod = "half_year"
	PeriodYearly    BillingPeriod = "yearly"
)

// Money is a price in micro-units of an ISO 4217 currency (9.99 USD ->
// Micros 9_990_000, Currency "USD"). Micros is the lossless representation both
// stores accept — Google natively (units+nanos), Apple after matching to the
// nearest price point.
type Money struct {
	Currency string
	Micros   int64
}

// IntroOffer is an optional introductory price / free trial for a tier. Free
// trials set FreeTrial and leave Price zero; paid intros set Price.
type IntroOffer struct {
	// Period is the intro duration (one billing period of Period length).
	Period BillingPeriod
	// FreeTrial makes the intro a free trial (Price ignored).
	FreeTrial bool
	// Price is the introductory amount when not a free trial.
	Price Money
}

// DesiredTier is one subscription product in moth's catalog, in a
// store-agnostic form. The same ProductID is used in both stores by moth
// convention (billing keys on it).
type DesiredTier struct {
	// ProductID is the store product identifier, identical in App Store
	// Connect and Google Play.
	ProductID string
	// Reference is the internal reference name (Apple subscription "name",
	// never shown to customers).
	Reference string
	// DisplayName is the customer-facing localized name.
	DisplayName string
	// Description is the customer-facing localized description.
	Description string
	// Period is the renewal cadence.
	Period BillingPeriod
	// Price is the base (base-territory / base-region) price.
	Price Money
	// Locale is the BCP-47 localization locale, e.g. "en-US".
	Locale string
	// GroupLevel is the Apple ranking within the subscription group (1 = top
	// tier). Ignored by Google.
	GroupLevel int
	// Intro is an optional introductory offer / free trial.
	Intro *IntroOffer
	// StripePriceID / StripeProductID are the Stripe resources moth currently
	// records for this tier ("" when never provisioned). Unlike the Apple and
	// Google SKUs, Stripe ids are generated by provisioning rather than
	// authored, so the Stripe sync diffs against these and writes fresh ids
	// back through ProductResult. Ignored by Apple and Google.
	StripePriceID   string
	StripeProductID string
}

// DesiredCatalog is moth's whole subscription catalog for one project's app —
// the desired state each store is reconciled into. Deliberately small: one
// subscription group, a few tiers (plan/12).
type DesiredCatalog struct {
	// GroupReference is the App Store Connect subscription group reference
	// name; all tiers share one group. Ignored by Google (no group concept).
	GroupReference string
	Tiers          []DesiredTier
}

// Action is what a sync did to one product or notification hook.
type Action string

// Sync actions.
const (
	ActionCreated   Action = "created"
	ActionUpdated   Action = "updated"
	ActionUnchanged Action = "unchanged"
	// ActionManual means the store API cannot do it; see the ManualSteps.
	ActionManual Action = "manual"
	ActionFailed Action = "failed"
)

// ProductResult is the per-tier outcome of a catalog sync.
type ProductResult struct {
	// ProductID is moth's / the store product id (for Stripe, the moth tier
	// identifier — Stripe ids are generated, not authored).
	ProductID string
	// StoreID is the store's own resource id to map back into moth's product
	// (Apple subscription resource id; Google == ProductID; Stripe recurring
	// Price "price_...").
	StoreID string
	// StoreParentID is the parent resource StoreID hangs off, when the store
	// has a two-level catalog identity (Stripe: the Product "prod_..." owning
	// the Price in StoreID). Empty for Apple and Google.
	StoreParentID string
	Action        Action
	Detail        string
}

// NotificationResult is the outcome of wiring one store notification hook.
type NotificationResult struct {
	// Kind is a stable slug, e.g. "apple_server_notification_url" or
	// "google_rtdn_topic".
	Kind     string
	Action   Action
	Endpoint string
	Detail   string
}

// ManualStep is a guided-fallback instruction for something the API cannot do.
// Returned as structured data (never printed here) so the CLI and admin handler
// render it their own way, with the exact values to enter.
type ManualStep struct {
	Title string
	// Reason says why the API cannot perform it.
	Reason string
	// URL is the console page to open, if any.
	URL string
	// Instructions are the exact lines/values to enter.
	Instructions []string
}

// SyncResult is the store-agnostic output of one catalog push, consumed by the
// CLI checklist and the admin monetization screen.
type SyncResult struct {
	// Store is billing.StoreApple or billing.StoreGoogle.
	Store         string
	Products      []ProductResult
	Notifications []NotificationResult
	// ManualSteps are the guided-fallback steps the API could not perform.
	ManualSteps []ManualStep
}

func (r *SyncResult) addProduct(p ProductResult) { r.Products = append(r.Products, p) }
func (r *SyncResult) addNotification(n NotificationResult) {
	r.Notifications = append(r.Notifications, n)
}
func (r *SyncResult) addManual(m ManualStep) { r.ManualSteps = append(r.ManualSteps, m) }

// Changed reports whether the sync altered store state (any product or
// notification created/updated). Idempotent re-runs return false — the
// "second run reports zero changes" acceptance criterion.
func (r *SyncResult) Changed() bool {
	for _, p := range r.Products {
		if p.Action == ActionCreated || p.Action == ActionUpdated {
			return true
		}
	}
	for _, n := range r.Notifications {
		if n.Action == ActionCreated || n.Action == ActionUpdated {
			return true
		}
	}
	return false
}

// applePeriod maps a BillingPeriod to Apple's subscriptionPeriod enum.
func applePeriod(p BillingPeriod) (string, bool) {
	switch p {
	case PeriodWeekly:
		return "ONE_WEEK", true
	case PeriodMonthly:
		return "ONE_MONTH", true
	case PeriodTwoMonth:
		return "TWO_MONTHS", true
	case PeriodQuarterly:
		return "THREE_MONTHS", true
	case PeriodHalfYear:
		return "SIX_MONTHS", true
	case PeriodYearly:
		return "ONE_YEAR", true
	}
	return "", false
}

// googlePeriod maps a BillingPeriod to an ISO-8601 billingPeriodDuration.
func googlePeriod(p BillingPeriod) (string, bool) {
	switch p {
	case PeriodWeekly:
		return "P1W", true
	case PeriodMonthly:
		return "P1M", true
	case PeriodTwoMonth:
		return "P2M", true
	case PeriodQuarterly:
		return "P3M", true
	case PeriodHalfYear:
		return "P6M", true
	case PeriodYearly:
		return "P1Y", true
	}
	return "", false
}

// moneyToUnitsNanos splits Micros into the Google Money units/nanos pair.
func moneyToUnitsNanos(m Money) (units int64, nanos int32) {
	units = m.Micros / 1_000_000
	nanos = int32((m.Micros % 1_000_000) * 1000)
	return units, nanos
}
