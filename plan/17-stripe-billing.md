# Milestone 17 — Stripe Billing (web)

## Goal

Give the milestone-11 subscription engine a third store: **Stripe**, so the same
per-project tiers and entitlements that sell through the App Store and Google Play can
sell on the web. A tier gains a Stripe price alongside its Apple/Google product ids; a
user who subscribes through Stripe Checkout lands in the same `subscriptions` table,
flows through the same entitlement derivation, and shows up in the same admin views and
revenue analytics — the stores stay the source of truth for mobile money, Stripe becomes
the source of truth for web money, and moth remains the validating mirror for all three.
This is the server half; the React SDK (18) puts a paywall in front of it.

## Model

Stripe is a new `store` value, not a parallel system — everything reuses milestone 11:

- `products` gain a `stripe_price_id` (nullable, like the other two store ids; a tier may
  ship on any subset of the three stores) and moth records the linked Stripe product id
  from provisioning.
- `subscriptions` accept `store = 'stripe'` with the Stripe identity
  (`subscription_id`, `customer_id`), the same `subscription_status` enum, and
  `environment` mapping Stripe's test/live modes onto `sandbox`/`production`.
- `stripe_customers` — per `(project_id, user_id)` mapping to the Stripe customer id,
  created lazily on first checkout so every moth user has at most one Stripe customer
  per project.
- `billing_credentials` extend with per-project Stripe credentials: restricted secret
  key + webhook signing secret, encrypted under the master key, write-only in the admin
  with `has_*` indicators — exactly like the Apple key and Google service account.
- **Status mapping** into the milestone-11 matrix (documented and table-tested like the
  rest): `active`, `trialing` → granted; `past_due` → `in_billing_retry` (granted —
  same "never lock out a paying user over a card hiccup" policy); `canceled`,
  `unpaid`, `incomplete_expired` → `expired` (not granted); Stripe pause collection →
  `paused` (not granted).

## Deliverables

### Stripe API client (no SDK dependency)

- Call the Stripe REST API directly (Checkout Sessions, Billing Portal Sessions,
  Subscriptions, Products/Prices, Customers) with the project's secret key, behind the
  same injectable-HTTP-client + overridable-endpoint pattern as the App Store Server
  API and Play Developer API clients — the whole engine stays testable against recorded
  fixtures, and the binary stays dependency-light.

### Checkout & management (`moth.billing.v1`, publishable-key + Bearer)

- `CreateCheckoutSession(product_id, success_url, cancel_url)` → a Stripe Checkout URL
  for a subscription to the tier's `stripe_price_id`, bound to the user's Stripe
  customer (created on demand) with `(project_id, user_id)` in the session metadata.
  Hosted Checkout deliberately: no card fields in moth or the SDK, no PCI surface.
- `CreateBillingPortalSession(return_url)` → a Stripe Billing Portal URL — cancel,
  payment-method and invoice management stay Stripe-hosted, the web analogue of
  deep-linking to the stores' subscription-management UI.
- `GetCustomerInfo` is unchanged by design: Stripe-sourced entitlements come back
  through the exact same shape, `source: store`. No new client-side concepts.

### Webhook (authoritative state changes)

- `POST /billing/stripe/webhook/{slug}` — plain-HTTP, project-scoped, alongside the
  Apple/Google endpoints. Verify the `Stripe-Signature` header against the project's
  webhook secret; handle `checkout.session.completed` and the
  `customer.subscription.created|updated|deleted` family; dedupe on event id; persist
  the raw payload into `store_notifications`.
- Same trust rule as milestone 11: **the webhook is a nudge, not a payload to trust** —
  on every event moth re-reads the subscription from the Stripe API before updating
  state, re-derives entitlements, and emits a `subscription_event` (with Stripe's
  actual amount + currency, so milestone-14 revenue for web is exact, not list-price
  estimated).
- Reconciliation: Stripe subscriptions join the milestone-10/11 periodic sweep that
  catches missed webhooks.

### Catalog provisioning (the milestone-12 story, but honest automation is total)

- Unlike App Store Connect, the Stripe API can do everything: the admin monetization
  screen gains a Stripe column and a "create in Stripe" action that provisions a
  Product + recurring Price per tier from the existing price metadata and writes back
  the ids; `moth setup billing` gains the Stripe leg, including creating the webhook
  endpoint itself via the API and storing the returned signing secret.
- Price immutability handled the Stripe way: a price change creates a new Price and
  re-points the tier; existing subscribers keep their old price (Stripe's model), and
  moth surfaces that clearly instead of pretending to edit in place.

### Admin & analytics touch-ups

- Per-user subscription view (11) and the monetization screen render `stripe` rows with
  the same status vocabulary; the operator grant/comp flow is untouched (grants are
  store-independent by construction).
- Milestone-14 dashboards gain `web` alongside `apple`/`google` in the store dimension —
  no new pipeline, just a new value flowing through `subscription_events`.

## Key design points

- **Third store, same engine.** No parallel Stripe subsystem: one status enum, one
  derivation matrix, one notifications table, one analytics stream. The React SDK (18)
  and the developer's backend see entitlements, never "which processor".
- **Stripe-hosted money surfaces.** Checkout and the Billing Portal are redirects to
  Stripe; moth never renders a card field and inherits none of the PCI burden — the
  web mirror of "the SDK never handles money" from milestone 13.
- **Validating mirror, still.** Signature-verified webhooks trigger API re-reads;
  entitlement state never comes from a client or an unverified event body.
- **Optional, like everything billing.** No Stripe credentials → the RPCs return a
  clear precondition error, mobile-store billing and the free `none` tier are
  unaffected; a web-only project can equally skip the mobile stores.

## Acceptance criteria

- With test-mode credentials: `CreateCheckoutSession` → completing the hosted Checkout
  (Stripe test card) → webhook → `GetCustomerInfo` shows the entitlement with
  `source: store`; the subscription row carries `store = stripe`,
  `environment = sandbox`.
- Cancelling from the Billing Portal flips the subscription to non-renewing and, after
  period end (simulated via test clocks or fixture), the entitlement is revoked.
- Webhook fixtures for created/updated/deleted and `past_due` flip status per the
  mapping table; replayed events are idempotent; a bad `Stripe-Signature` is rejected;
  a webhook for another project's subscription is rejected.
- The derivation-matrix tests extend to the Stripe statuses, every cell covered.
- "Create in Stripe" provisions Product + Price for a tier and writes back ids; a
  subsequent price edit creates a new Price and re-points the tier.
- A `subscription_event` from a test purchase carries Stripe's real amount/currency and
  appears in the milestone-14 revenue dashboard under `web`.

## Out of scope

The web paywall UI and checkout redirect UX (18). One-time payments, metered/usage
billing, per-seat quantities, coupons/promotion codes, Stripe Tax, invoicing beyond
what the Billing Portal shows, Stripe Connect/multi-account, and proration UX beyond
Stripe's defaults. Cross-store switching flows (a user moving mobile ↔ web keeps both
subscriptions' store rules; moth just derives entitlements from the union).
