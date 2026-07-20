---
title: Subscriptions & paywall
description: Sell subscriptions from your apps and on the web — store receipts and Stripe validated server-side, entitlements, a themed paywall, and revenue analytics, with no billing SaaS.
---

moth gives every project its own subscription engine: App Store and Google
Play receipts are validated **server-side**, Stripe sells the same tiers
**on the web**, subscription state is distilled into **entitlements** your
app gates on, a **themed paywall** renders your tiers, and revenue lands on
the analytics tab. There is no billing SaaS in the loop — the stores own
the money and the renewal truth; moth mirrors and re-validates it.

Declaring paid tiers is optional. Every user always has a valid subscription
state — the built-in free tier — so `getCustomerInfo()` never errors for a
user who has never paid, and a project that never configures a product keeps
working unchanged.

## Concepts

- **Entitlement** — a named capability (`pro`, `premium`, …) your app checks
  with `hasEntitlement('pro')`. Apps never check product identifiers, so which
  tier grants an entitlement can change without an app release.
- **Product / tier** — an auto-renewing subscription mapped to an App Store
  product id, a Play product id, and/or a Stripe price — any subset of the
  three stores — granting one or more entitlements while active. Price and
  period metadata drive the paywall display and revenue analytics.
- **Offering** — the ordered set of tiers the paywall presents, with one
  highlighted "most popular" tier. One offering per project keeps it simple.
- **Grants** — operator-issued entitlements (comps, promos, grace extensions)
  attached to a user from the admin, independent of store state.

Access-granting statuses follow store policy: `active`, `trialing`,
`in_grace_period`, and `in_billing_retry` keep the entitlement (a card hiccup
never locks out a paying user); `expired`, `revoked`, and `paused` drop it.

## 1 · Connect the stores

In the project's **Monetization** tab, store credentials are write-only and
encrypted at rest under the instance master key:

- **Apple** — an App Store Connect **In-App Purchase key** (`.p8`, key id,
  issuer id), the app's bundle id and Apple app id. Register the App Store
  Server Notification URL shown in the tab
  (`/billing/apple/notifications/{slug}`) in App Store Connect.
- **Google** — a service account JSON with Play Developer API access, the
  package name, and a Cloud Pub/Sub topic for Real-time Developer
  Notifications pushed to `/billing/google/rtdn/{slug}`.
- **Stripe** — a restricted secret key (`rk_…`, or a full `sk_…`) and the
  webhook signing secret. Register the webhook endpoint shown in the tab
  (`/billing/stripe/webhook/{slug}`) for `checkout.session.completed` and
  the `customer.subscription.*` events. Stripe **test mode** plays the
  sandbox role on the web.

Or do all of it from the terminal in one verified pass:

```sh
moth setup billing --project bird-spotter
```

For Stripe, `moth setup billing` does all three steps itself — stores the
credentials, and creates the webhook endpoint via the Stripe API, capturing
its signing secret. `moth doctor --project bird-spotter` checks the
credentials, catalog sync, and notification endpoints any time.

## 2 · Define tiers and push them to the stores

Create the entitlement(s) and tiers in the Monetization tab (or declaratively
via the `monetization:` block of `moth project apply`). **Push to stores**
shows a dry-run plan first, then creates/updates the subscription products in
App Store Connect and Google Play — automated where the store APIs allow,
with exact guided steps where they don't (Apple's review submission, for
example). The push is idempotent: re-running after a price change updates
only that price.

For Stripe the automation is total: the push creates a Product and a
recurring Price per tier from the existing price metadata and writes the
price id back. Prices are immutable in Stripe, so a later price change
creates a **new** Stripe Price and re-points the tier — existing
subscribers keep the price they signed up at, and the admin says so
instead of pretending to edit in place.

## 3 · Sell from the app

Add `moth_billing` next to `moth_auth` — moth's first-party billing plugin,
StoreKit 2 on iOS and the Play Billing Library on Android, served from your
instance's own `/pub` at the server's version so the receipts it produces
are exactly what the server validates. No third-party billing plugin, no
adapter code to write:

```yaml
dependencies:
  moth_billing:
    hosted: https://auth.example.com/pub
    version: ^1.0.0
```

```dart
import 'package:moth_billing/moth_billing.dart';

// Gate a screen behind an entitlement; free users see the paywall.
MothApp(
  config: MothConfig(endpoint: ..., publishableKey: 'pk_...'),
  billingAdapter: MothStoreBilling(),
  requiresEntitlement: 'pro',
  paywall: const MothPaywallScreen(),
  child: const MyApp(),
);

// Or check and purchase imperatively:
final scope = MothScope.of(context);
if (!scope.hasEntitlement('pro')) { /* show MothPaywallScreen */ }
await scope.purchase(product);        // typed result: purchased/pending/…
await scope.restorePurchases();       // new device / reinstall
```

moth handles everything after the native purchase: the StoreKit 2 signed
transaction (Apple) or Play Billing purchase token (Google) goes to the
server, is validated against the store, and comes back as entitlements.
`moth_auth` itself stays pure Dart and never depends on the plugin — apps
with exotic store needs implement its `MothBillingAdapter` interface
themselves instead of using `moth_billing`.

`MothPaywallScreen` renders the offering with the project theme; its
headline, benefit bullets, layout, and highlighted tier are edited — per
language — under **Design → Paywall**, with a live preview. Entitlement state
is cached on device, so gating is instant on launch and refreshes in the
background.

## 4 · Sell on the web (Stripe)

With Stripe connected, the same tiers sell in the browser through the
[React SDK](../../react/). Checkout is a **redirect to Stripe-hosted
Checkout** — moth and the SDK never render a card field, so there is no
PCI surface and no Stripe.js dependency:

```tsx
// Gate a page behind an entitlement; free users see the paywall.
<MothGate entitlement="pro" fallback={<MothPaywallScreen />}>
  <ProPage />
</MothGate>
```

`MothPaywallScreen` renders the same admin-configured paywall — theme,
copy, tiers — as the Flutter one; a tier without a Stripe price shows as
unavailable on the web rather than disappearing silently. Its purchase
button calls `purchase(product)`, which creates a Checkout session and
redirects; on return the SDK re-reads the entitlements and the gate
unlocks. `manageBilling()` redirects to the Stripe **Billing Portal** for
cancellation, payment methods, and invoices — the web analogue of
deep-linking to the stores' subscription management.

Under the hood these are two RPCs on `moth.billing.v1` (publishable key +
user Bearer token): `CreateCheckoutSession(product_identifier,
success_url, cancel_url)` and `CreateBillingPortalSession(return_url)`,
each returning a URL — usable from any client, not just React.

State changes arrive on the signature-verified webhook, and the same
trust rule as the mobile stores applies: the event is a nudge, moth
re-reads the subscription from the Stripe API before changing anything.
Statuses map into the same matrix — `active` and `trialing` grant,
`past_due` becomes `in_billing_retry` (still granted — a card hiccup
never locks out a paying user), `canceled`/`unpaid`/`incomplete_expired`
become `expired`, and paused collection is `paused`.

## 5 · Trust it from your backend

Your own API should never trust the client's entitlement claim. Ask moth
directly with the project's secret key:

```sh
grpcurl -H "x-moth-key: sk_..." -d '{"user_id":"..."}' \
  your-moth:8080 moth.server.v1.EntitlementService/GetUserEntitlements
```

:::note[grpcurl against a release build]
Release builds don't advertise their schema over reflection. Either start
the server with `--reflection`, or point grpcurl at the `.proto` sources
the instance serves at `/protos/` (`grpcurl -import-path … -proto …`).
:::

## 6 · Watch the money

The project's analytics tab gains revenue per month (store-reported, per
currency — no invented FX), active subscribers with trend, new vs churned,
trial-to-paid conversion, and per-tier / per-store breakdowns — **web**
sits alongside Apple and Google in the store dimension, with Stripe's
actual charged amounts, not list-price estimates — plus a CSV export. An
elevated-churn banner surfaces billing trouble early.

## Testing with sandbox accounts

Sandbox purchases (TestFlight / license testers / Stripe test mode with a
test card) validate exactly like production ones but are flagged by
environment: they grant entitlements for testing yet are excluded from
production revenue analytics. Run the example app against your instance
with a sandbox account to walk the whole loop before release.

## Out of scope

Consumables and one-time purchases, promotional-offer campaigns,
coupons/promotion codes, Stripe Tax and Stripe Connect, and proration UX
beyond what the stores handle natively. The
[security model](../../security/) covers how receipts are verified (Apple
x5c chain pinning, Play Developer API re-reads) and how store credentials
are encrypted.
