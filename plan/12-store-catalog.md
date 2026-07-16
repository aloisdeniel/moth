# Milestone 12 — Store Catalog Provisioning (admin)

## Goal

Let an operator define subscription tiers **once, in the moth admin**, and have moth
push them into App Store Connect and Google Play — configuring the products, prices,
and the notification plumbing so the milestone-11 engine actually receives renewal
events. This is the monetization counterpart to milestone 08's one-command Google/Apple
sign-in setup, and it inherits 08's contract of **honest automation**: automate every
step the official APIs expose, fall back to a precise guided flow where they don't, and
verify the result — never silently degrade.

## Deliverables

### Admin monetization screen

- A "Monetization" area per project: define the entitlements and tiers from milestone 11
  in a form (display name, billing period, price + currency, trial/intro descriptor,
  which entitlement each tier grants, store product ids), plus an **offering** — the
  ordered set of tiers a paywall presents, with a default offering per project. Kept
  deliberately small: one offering, a few tiers.
- Store-connection status panel: which store credentials are configured, whether the
  App Store Server Notification URL and the Play Pub/Sub topic are wired, and whether the
  catalog in each store matches moth (a live diff).

### Automatic catalog reflection (honest automation)

First task is a **capability spike** documenting what each store's API actually allows,
committed as code comments and docs — the plan commits to the UX (define once, verified
in both stores), not to over-promised coverage.

- **Apple — App Store Connect API** (reusing the ASC JWT auth from milestone 08): create
  the subscription group, auto-renewable subscription products, base price points, and
  localizations from moth's tier definitions; map the created product ids back into the
  moth products. Steps the API can't do (price-schedule nuances, availability, and
  submitting for review) become a guided checklist with exact values to set.
- **Google — Android Publisher API** (service-account authed): create/update subscription
  products with base plans and price offers from the same tier definitions; map ids back.
  Guided fallback for anything the API doesn't cover (e.g. Play Console availability).
- **Notification wiring**: register moth's App Store Server Notification URL for the app,
  and create/point the Play RTDN Cloud Pub/Sub topic + push subscription at moth's
  `/billing/google/rtdn/{slug}` endpoint — the plumbing milestone 11 depends on.

### `moth setup billing --project <slug>` (CLI)

- Mirrors `moth setup google|apple`: configure the store credentials (Apple IAP `.p8` +
  key/issuer ids; Google service-account JSON) into moth's encrypted billing config, push
  the catalog, wire the notification endpoints, and **verify** — a sandbox subscription
  status read against each store, the notification URL/topic registered, `GetCustomerInfo`
  reachable. Idempotent and re-runnable (diff current store/moth state, change only what's
  needed — safe after a price change or a new tier). Colored checklist, `--json`.
- `moth doctor` (milestone 08) gains billing checks: store credentials valid and unexpired,
  catalog in sync, notification endpoints reachable.

### Shared behavior

- **Idempotent and re-runnable**, like the sign-in setup commands: safe to run after adding
  a tier, changing a price, or rotating a store key.
- **Store nothing platform-side**: service-account and ASC credentials used in-process for
  the setup call and persisted only as moth's own encrypted billing config, never leaked.

## Key design points

- **One catalog, three faces** — the admin screen, `moth setup billing`, and
  `moth project apply` (milestone 08 declarative mode gains a `monetization:` block) all
  drive the same tier/offering definitions and the same store-sync logic; no divergence.
- **The stores stay authoritative for money and renewals**; moth's catalog is the *desired
  state* it reconciles into each store. A diff view makes drift visible rather than
  pretending the push always fully succeeds.
- **Honest automation** — the command states what it automated, what it needs done by hand
  (with exact values), and verifies the outcome, exactly like milestone 08. Apple's review
  step for new subscriptions is the canonical "we can't automate this, here's precisely
  what to click".

## Acceptance criteria

- From the admin (and equivalently `moth setup billing`): define a tier granting `pro`,
  push it, and see the product created/updated in both App Store Connect and Google Play
  with the right id, price, and period — any remaining manual step (e.g. Apple review
  submission) enumerated by moth itself, none discovered by surprise.
- Running the push twice reports zero changes on the second run (idempotency); changing a
  price and re-running updates only that price.
- The App Store Server Notification URL and Play RTDN topic are registered and verified to
  reach the milestone-11 webhook endpoints (a test notification round-trips).
- `moth doctor` flags a deleted store product, an expired store key, and a broken
  notification topic with clear remediation.
- `moth project apply -f` with a `monetization:` block is idempotent (second run: no
  changes).

## Out of scope

The Flutter purchase flow + paywall (13) and revenue analytics (14). Price experiments,
regional price overrides beyond the store defaults, promotional-offer/win-back
configuration, staged rollouts, and store-review orchestration beyond a guided checklist —
post-v1.1. moth configures subscription *products*; it does not manage the app's store
listing, screenshots, or release.
