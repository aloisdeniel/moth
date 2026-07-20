# Milestone 11 — Subscriptions & Entitlements (server core)

## Goal

Give every project a per-user subscription and **entitlement** engine, validated
server-side against the App Store and Google Play — no third-party billing SaaS.
moth becomes the single place where a portfolio's subscriptions live: the stores
own the money and the source-of-truth renewal state; moth mirrors that state,
derives entitlements from it, and hands both to the app and to the developer's
own backend. This milestone is the engine; the store-catalog automation (12), the
Flutter purchase flow + paywall (13), and revenue analytics (14) build on it.

Guiding principle from the request: **declaring paid subscriptions is optional**.
Every user always has a valid subscription state — a built-in `none` (free) tier —
so a project that never configures a product still works, and `GetCustomerInfo`
never returns an error for a user who has never paid.

## Model

Entitlements are **decoupled from products** (the RevenueCat lesson): the app gates
features on a stable entitlement identifier, and which product grants it can change
without an app release.

- `entitlements` — named capabilities per project (`pro`, `premium`, …): identifier,
  display name. A project may define zero (then everyone is just `none`).
- `products` — subscription **tiers** per project: internal id, display name,
  `apple_product_id`, `google_product_id` (either may be null if a tier ships on one
  store only), the entitlement(s) it grants while active, billing period, and price
  metadata (list price + currency, intro/trial descriptor) used for display and
  analytics. **Kept minimal**: a handful of tiers per project.
- `subscriptions` — per `(project_id, user_id, store)`: product, store
  (`apple` | `google`), store identity (`original_transaction_id` for Apple,
  `purchase_token` + `subscription_id` for Google), `status`, `current_period_end`,
  `auto_renew`, `environment` (`sandbox` | `production`), and the last raw store
  state JSON. Absence of a row = the user is on `none`.
- `subscription_grants` — manual/promotional grants and grace overrides an operator
  can attach to a user (comp a reviewer, extend a grace period, grant a promo): the
  entitlement, an expiry, a reason, and the granting credential (for the audit log).
- `store_notifications` — raw App Store Server Notifications / Play RTDN payloads,
  stored for idempotency (dedupe by store notification id) and audit.
- `billing_credentials` — per-project store API credentials, **encrypted under the
  master key** (milestone-01 key, already used for Apple `.p8` and SMTP): Apple
  In-App-Purchase key (`.p8`, key id, issuer id) + bundle id + app Apple id; Google
  service-account JSON + package name + Pub/Sub topic. Write-only in the admin, like
  every other secret.

`subscription_status` enum (minimal, mapped from both stores): `active`, `trialing`,
`in_grace_period`, `in_billing_retry` (Google "on hold"), `paused`, `expired`,
`revoked`. The **entitlement derivation** — the one piece of business logic worth
getting right and testing exhaustively — maps status → granted:

- `active`, `trialing`, `in_grace_period`, `in_billing_retry` → **granted** (grace and
  billing-retry deliberately keep access, matching store policy so paying users are
  never locked out during a card hiccup).
- `expired`, `revoked`, `paused` → **not granted**.
- Any active `subscription_grant` → granted until its expiry, independent of store
  state (promos, comps, grace extensions).
- No subscription and no grant → `none`: the free default, always valid.

Document this matrix in code comments and cover every cell in tests.

## Deliverables

### Store validation (no store SDK server-side)

- **Apple** — verify StoreKit 2 signed transactions (JWS) against Apple's root
  certificate chain locally, and call the **App Store Server API** (JWT-authed with
  the project's In-App-Purchase key) `Get All Subscription Statuses` /
  `Get Transaction Info` to fetch authoritative renewal state. `verifyReceipt` is
  deprecated and not used.
- **Google** — call the **Google Play Developer API** `purchases.subscriptionsv2.get`
  (service-account authed) to resolve a `purchase_token` to a `SubscriptionPurchaseV2`
  (state, line items, expiry, auto-renew), and acknowledge the purchase when required.
- Both behind an injectable HTTP client + overridable endpoints so the whole engine is
  testable against recorded fixtures and a JWKS/cert test double, exactly like the
  milestone-04 OIDC verifier.

### Store notifications (authoritative state changes)

- Plain-HTTP, project-scoped webhook endpoints (renewals are store round-trips, not
  RPCs): `POST /billing/apple/notifications/{slug}` (App Store Server Notifications V2:
  verify the `signedPayload` JWS chain, map `notificationType`/`subtype`) and
  `POST /billing/google/rtdn/{slug}` (Play Real-time Developer Notifications delivered
  via a Cloud Pub/Sub **push** subscription: authenticate the push, then call the
  Developer API to fetch current state — the notification is a nudge, not a payload to
  trust).
- Idempotent: dedupe on the store notification id, persist the raw payload, update the
  subscription + re-derive entitlements, emit a `subscription_event` (for 14). Never
  trust a notification body for entitlement state without a validating store read.

### Client-facing RPCs (`moth.billing.v1`, publishable-key + Bearer)

- `GetCustomerInfo` → the user's active entitlements (identifier, expiry, source:
  `store` | `grant` | `none`) and active subscriptions. **Always returns a valid
  object**, `none` for free users.
- `SubmitPurchase(store, product_id, apple_jws | google_purchase_token)` — the app
  hands moth the receipt after a native purchase; moth validates against the store,
  links the subscription to the current user, derives entitlements, returns the fresh
  `CustomerInfo`.
- `RestorePurchases(store, receipts…)` — re-link store purchases to the current user
  (new device, reinstall, account change), applying the store's own transfer rules.

### Developer-backend RPC (`moth.server.v1`, secret-key)

- `GetUserEntitlements(user_id)` → the same derived entitlement set, so the app's own
  backend can gate server-side features without trusting the client. (A push webhook
  to the developer's backend on entitlement change is out of scope — parking lot.)

### Admin management (`moth.admin.v1`)

- CRUD for entitlement definitions and products/tiers (identifiers, store product ids,
  entitlement grants, price metadata) — the data the store-catalog automation (12) and
  paywall (13) consume.
- Per-user subscription view: current subscription, status, period end, store, history;
  and operator actions — grant/revoke a promo or comp entitlement, extend a grace
  period (writes a `subscription_grant`, audit-logged per milestone 10).
- Billing credentials editor (write-only secrets, `has_*` indicators).

## Key design points

- **The store is the source of truth; moth is a validating mirror.** moth never marks a
  subscription active on a client's say-so — only after a store read or a verified
  signed transaction. Notifications trigger re-reads; a periodic reconciliation sweep
  (hook into the milestone-10 background sweep) catches missed webhooks.
- **Entitlements over products.** Apps check `hasEntitlement("pro")`, never a product id;
  swapping which tier grants `pro` is an admin change, not an app release.
- **`none` is a first-class state, not an error.** Free users, misconfigured projects,
  and never-paid accounts all get a valid `CustomerInfo`. Subscriptions are additive.
- **Secrets at rest.** Store API keys/service accounts join Apple `.p8` and SMTP under
  the master key; plaintext returned to the caller exactly once.
- **Minimal by construction.** Several tiers, one subscription group's worth of products,
  a handful of entitlements — auto-renewing subscriptions only.

## Acceptance criteria

- With sandbox credentials: a StoreKit 2 signed transaction and a Play `purchase_token`
  each validate, create a subscription, and grant the mapped entitlement; `GetCustomerInfo`
  reflects it; a never-paid user returns `none` with no error.
- Table-driven entitlement-derivation tests cover every status × grant cell, including
  grace/billing-retry granting access and expired/revoked/paused revoking it.
- App Store Server Notification (renewal, refund, revoke) and Play RTDN (recovered,
  canceled, on-hold) fixtures each flip subscription + entitlement state; replayed
  notifications are idempotent (asserted against a store test double).
- Tampered Apple JWS, wrong-bundle transaction, and a `purchase_token` for another
  project are all rejected.
- An operator can comp an entitlement to a user and see it expire on schedule; the action
  appears in the audit log.
- `moth.server.v1.GetUserEntitlements` returns the same set the client sees.

## Out of scope

Store-catalog auto-provisioning (12), the Flutter purchase flow + paywall (13), revenue
analytics (14). Consumables and non-renewing/one-time purchases, web/Stripe billing,
promotional-offer and win-back campaigns, refund automation, cross-store entitlement
merging beyond `RestorePurchases`, tax/VAT accounting, family sharing — all post-v1.1.
