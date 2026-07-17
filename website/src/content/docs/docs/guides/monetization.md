---
title: Subscriptions & paywall
description: Sell subscriptions from your apps — store receipts validated server-side, entitlements, a themed paywall, and revenue analytics, with no billing SaaS.
---

moth gives every project its own subscription engine: App Store and Google
Play receipts are validated **server-side**, subscription state is distilled
into **entitlements** your app gates on, a **themed paywall** renders your
tiers, and revenue lands on the analytics tab. There is no billing SaaS in
the loop — the stores own the money and the renewal truth; moth mirrors and
re-validates it.

Declaring paid tiers is optional. Every user always has a valid subscription
state — the built-in free tier — so `getCustomerInfo()` never errors for a
user who has never paid, and a project that never configures a product keeps
working unchanged.

## Concepts

- **Entitlement** — a named capability (`pro`, `premium`, …) your app checks
  with `hasEntitlement('pro')`. Apps never check product identifiers, so which
  tier grants an entitlement can change without an app release.
- **Product / tier** — an auto-renewing subscription mapped to an App Store
  and/or Play product id, granting one or more entitlements while active.
  Price and period metadata drive the paywall display and revenue analytics.
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

Or do all of it from the terminal in one verified pass:

```sh
moth setup billing --project bird-spotter
```

`moth doctor --project bird-spotter` checks the credentials, catalog sync,
and notification endpoints any time.

## 2 · Define tiers and push them to the stores

Create the entitlement(s) and tiers in the Monetization tab (or declaratively
via the `monetization:` block of `moth project apply`). **Push to stores**
shows a dry-run plan first, then creates/updates the subscription products in
App Store Connect and Google Play — automated where the store APIs allow,
with exact guided steps where they don't (Apple's review submission, for
example). The push is idempotent: re-running after a price change updates
only that price.

## 3 · Sell from the app

Native billing stays out of the SDK's dependencies: the app supplies a
`MothBillingAdapter` (the example app ships one built on `in_app_purchase`),
and moth handles everything after the native purchase — the receipt goes to
the server, is validated against the store, and comes back as entitlements.

```dart
// Gate a screen behind an entitlement; free users see the paywall.
MothApp(
  config: MothConfig(endpoint: ..., publishableKey: 'pk_...'),
  billingAdapter: MyBillingAdapter(),
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

`MothPaywallScreen` renders the offering with the project theme; its
headline, benefit bullets, layout, and highlighted tier are edited — per
language — under **Design → Paywall**, with a live preview. Entitlement state
is cached on device, so gating is instant on launch and refreshes in the
background.

## 4 · Trust it from your backend

Your own API should never trust the client's entitlement claim. Ask moth
directly with the project's secret key:

```sh
grpcurl -H "x-moth-key: sk_..." -d '{"user_id":"..."}' \
  your-moth:8080 moth.server.v1.EntitlementService/GetUserEntitlements
```

## 5 · Watch the money

The project's analytics tab gains revenue per month (store-reported, per
currency — no invented FX), active subscribers with trend, new vs churned,
trial-to-paid conversion, and per-tier / per-store breakdowns, plus a CSV
export. An elevated-churn banner surfaces billing trouble early.

## Testing with sandbox accounts

Sandbox purchases (TestFlight / license testers) validate exactly like
production ones but are flagged by environment: they grant entitlements for
testing yet are excluded from production revenue analytics. Run the example
app against your instance with a sandbox account to walk the whole loop
before release.

## Out of scope

Consumables and one-time purchases, web/Stripe billing, promotional-offer
campaigns, and proration UX beyond what the stores handle natively. The
[security model](../../security/) covers how receipts are verified (Apple
x5c chain pinning, Play Developer API re-reads) and how store credentials
are encrypted.
