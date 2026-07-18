# moth — Mobile + Auth

## Vision

moth makes authentication for a new mobile app trivial: run **one small binary**, create a project in the admin web app, add the served Flutter package to your `pubspec.yaml`, wrap your app in a login widget — done.

The defining idea: **one moth server carries your entire portfolio of independent apps**. An indie developer, studio, or agency runs a single instance; every app they ship is just another project on it — with its own users, signing keys, provider credentials, branding, and analytics, fully isolated from the others. Adding app #10 costs exactly what app #1 did: one project created in the admin, zero new infrastructure.

## Core capabilities

- **Single binary server** — no external dependencies, embedded database, embedded admin UI, embedded Flutter package. `moth serve` and you're running.
- **One server, many independent apps** — unlimited projects per instance, each a sealed tenant: its own user base, its own signing keypair, its own Google/Apple credentials, its own login branding. Nothing is shared across projects unless the operator is looking at the admin console.
- **Per-project authentication** — email/password, Sign in with Google, Sign in with Apple. Users belong to exactly one project; the same email in two apps is two unrelated accounts.
- **Subscriptions & entitlements** (post-v1.0) — per-user App Store / Google Play subscriptions, validated server-side, with entitlements (`pro`, …) derived from subscription state and grants (promos, grace periods). A built-in `none` (free) tier is always present, so paid subscriptions are optional. moth centralizes subscription state, reflects tiers into the stores, drives a themed Flutter paywall, and reports revenue per month. Post-v1.2, Stripe joins as a third store for the web: same tiers, same entitlements, hosted Checkout.
- **Internationalization** (post-v1.1) — the app sends its language as an HTTP header; moth negotiates the best available locale and returns adapted copy for the SDK screens, hosted pages, and emails, from a bundled curated locale set. Operators customize the sign-in, sign-up, and paywall copy per language in the admin and preview every SDK screen for any language.
- **Admin web application** — create/configure projects, manage users, view analytics, copy setup instructions, customize the login design system (colors, font, spacing, logo) and per-language copy.
- **Admin CLI** — the same binary drives any moth instance from the terminal (personal access tokens): scriptable project/user management, declarative `moth project apply`, and one-command Google/Apple console configuration (`moth setup google|apple`).
- **Agent-ready** — `moth skill export` emits an Agent Skills package (optionally interpolated with a project's real config) that teaches coding agents both how to integrate moth into an app and how to administer an instance through the CLI.
- **Served Flutter package** — the server exposes a pub-compatible repository so developers reference the SDK directly from their moth instance in `pubspec.yaml`. The SDK provides a wrapper widget for the whole app and exposes auth state through an `InheritedWidget`.
- **Served React SDK** (post-v1.2) — the same model for the web: an npm-compatible registry serves `@moth/react` from the binary; a provider component gates the app, hooks expose auth and entitlement state, and a themed paywall sells web subscriptions through Stripe Checkout — with the project's theme and localized copy applied.
- **Native billing plugin** (post-v1.3) — `moth_billing`, a first-party Flutter plugin served from `/pub` (StoreKit 2 on iOS, Play Billing Library on Android) implementing the milestone-13 billing adapter, so a native purchase needs one dependency and zero adapter code.
- **Push device registry** (post-v1.3) — moth registers every signed-in device's push identity (APNs / FCM token, Web Push subscription) with permission state and invalidation; the developer's backend reads the live set via `moth.server.v1` and sends through the push services itself. `moth_push` (Flutter) and `useMothPush()` (React) populate the registry automatically.

## Architecture decisions

| Decision | Choice | Rationale |
|---|---|---|
| Server language | **Go** (1.23+) | Best-in-class for small static cross-platform binaries; `go:embed` bundles all assets; trivial deployment. |
| Database | **SQLite** via `modernc.org/sqlite` | Embedded, zero-config, pure-Go driver keeps CGO-free cross-compilation. Single data directory to back up. |
| API protocol | **gRPC**, protobuf-first, served with `connect-go` | Typed, codegen'd clients for Dart and TypeScript from one set of `.proto` files. connect-go speaks native gRPC (Flutter SDK) *and* gRPC-Web (browser SPA) on the same port, multiplexed with the plain-HTTP surfaces. `buf` for codegen/lint/breaking-change checks. |
| Web surfaces | stdlib `net/http` alongside gRPC | Pub repository API, hosted verify/reset/OAuth-redirect pages, asset serving, JWKS, healthz — their consumers (pub client, browsers, third-party JWT libs) require plain HTTP. |
| Access tokens | **JWT, ES256**, **one keypair per project**, per-project JWKS endpoint | The developer's own backend verifies its app's tokens offline against the project's public key (or online via an introspection RPC). Per-project keys mean a token minted for one app can never validate for another, and one project's key can be rotated or revoked without touching the rest. |
| Sessions | Opaque **rotating refresh tokens** stored server-side | Revocable, detects token theft via rotation reuse. |
| Password hashing | **argon2id** | Current best practice. |
| Admin UI | **React + Vite + TypeScript SPA**, embedded via `go:embed` | Rich interactive admin (theme editor, charts) without server-side templating complexity. |
| Flutter SDK delivery | Server implements the **pub hosted repository API** | `hosted: https://your-moth/pub` in pubspec — no publishing to pub.dev needed, version always matches the server. |
| Email delivery | SMTP (configurable), console/log transport in dev | Keeps the binary dependency-free. |
| Subscriptions | **Server-side validation** against the App Store Server API + Google Play Developer API; no billing SaaS | Stores own the money and renewal truth; moth mirrors and re-validates via store notifications. Entitlements are decoupled from products (RevenueCat-style) so tiers can change without an app release. Keeps the binary dependency-free and the model minimal (a few tiers per app). |
| Localization | **Header-negotiated locale + a closed, bundled message catalog** with per-project overrides | The client asserts its language via a header; moth negotiates against the project's available locales and returns adapted copy from a curated bundled set (English default), refined by per-project per-locale overrides. One catalog serves SDK screens, hosted pages, and emails; bundled defaults always render, mirroring the design-system token model. |

## System shape

```
┌────────────────────────── moth (single binary) ──────────────────────────┐
│                                                                          │
│  /admin           → embedded React SPA (admin console)                   │
│  moth.admin.v1.*  → admin gRPC services (gRPC-Web from the SPA,           │
│                     session-cookie auth)                                  │
│  moth.auth.v1.*   → project auth gRPC services (publishable key + JWT)    │
│  moth.server.v1.* → developer-backend gRPC services (secret key):         │
│                     token introspection, user management, entitlements    │
│  moth.billing.v1.*→ subscription gRPC services (publishable key + JWT):    │
│                     customer info, submit/restore purchase (post-v1.0)     │
│  moth.push.v1.*   → push device registration (publishable key + JWT,       │
│                     post-v1.3); read side lives in moth.server.v1          │
│  /pub/*           → pub repository API (HTTP) serving moth_auth           │
│  /npm/*           → npm registry API (HTTP) serving @moth/react (post-v1.2)│
│  /p/*, /assets/*  → hosted pages & project assets (HTTP)                  │
│  /billing/*       → App Store / Play / Stripe notifications (HTTP, post-v1.0)│
│  /p/{slug}/.well-known/jwks.json → per-project public keys (HTTP)         │
│                                                                          │
│  SQLite (data/moth.db) · file store (data/uploads/) · keys (data/keys/)  │
└──────────────────────────────────────────────────────────────────────────┘
        ▲                                ▲
        │ admin browser                  │ Flutter app (moth_auth SDK)
        │                                │ web app (@moth/react SDK)
```

## Data model (sketch)

- `admins` — operators of the moth instance.
- `projects` — one per mobile app: name, slug, publishable key, secret key, provider config (Google/Apple credentials), theme, settings. Config blobs (theme, paywall, copy) are stored as serialized protos (`moth.projectconfig.v1`) — the same schema language as the wire, no parallel JSON.
- `project_keys` — per-project ES256 signing keypairs (private key encrypted at rest, public part served via the project JWKS); multiple rows per project to support rotation.
- `users` — scoped by `project_id`; email, password hash (nullable for social-only), verification state, custom claims (roles/permissions embedded in the JWT).
- `identities` — links a user to a provider (`password`, `google`, `apple`) with provider subject ID; enables account linking.
- `refresh_tokens` — hashed, rotating, per device session.
- `events` — analytics event stream (signup, login, provider, platform, timestamp).
- `email_tokens` — verification / password-reset tokens.

Subscriptions (post-v1.0) add:

- `entitlements` — named capabilities per project (`pro`, `premium`, …); apps gate on these.
- `products` — subscription tiers per project, mapped to App Store / Play product ids, granting entitlements, with price/period metadata; grouped into an `offering` for the paywall.
- `subscriptions` — per user per store: product, store transaction/purchase-token ids, status, current-period end, environment (sandbox/production).
- `subscription_grants` — manual promo/comp grants and grace overrides an operator attaches to a user.
- `store_notifications` — raw App Store Server Notifications / Play RTDN payloads (idempotency + audit).
- `subscription_events` — revenue/analytics stream (start, renew, cancel, refund, amount, currency).
- `billing_credentials` — per-project store API credentials, encrypted under the master key.
- `paywalls` — per-project paywall config (copy, offering, layout, legal links), an extension of the theme.

Web billing (post-v1.2) adds:

- `stripe_customers` — per-project per-user mapping to the Stripe customer id; `products` gain a `stripe_price_id` and `subscriptions`/`billing_credentials` accept `stripe` as a third store.

Push registration (post-v1.3) adds:

- `push_devices` — per-project per-user device registrations: target (`apns` | `fcm` | `webpush`), push credential, stable device id, permission state, device metadata, last-seen, and revocation (reason-coded, never hard-deleted). Project config gains a push section (Web Push VAPID public key).

Localization (post-v1.1) adds:

- `copy` — per-project per-locale overrides on the bundled message catalog (sign-in, sign-up, password-reset, verify-email, paywall, hosted-page and email keys), revisioned for client caching, merged over the bundled defaults.

## Milestones

| # | Milestone | Outcome |
|---|---|---|
| [01](01-foundations.md) | Foundations | Runnable binary: CLI, config, SQLite + migrations, project model, admin bootstrap, CI. |
| [02](02-email-password-auth.md) | Email/password auth | Full signup/login lifecycle per project: tokens, refresh, verification, password reset. |
| [03](03-admin-web-app.md) | Admin web app v1 | Embedded SPA: projects, API keys, user management, setup instructions. |
| [04](04-social-sign-in.md) | Social sign-in | Sign in with Google & Apple, native flows, account linking. |
| [05](05-flutter-sdk.md) | Flutter SDK + pub serving | `moth_auth` package served from the binary; wrapper widget + inherited auth state. |
| [06](06-design-system.md) | Design system & themed login | Admin theme editor; SDK login screens render project branding. |
| [07](07-analytics.md) | Analytics | Event capture and admin dashboards. |
| [08](08-admin-cli.md) | Admin CLI | Terminal project management via personal access tokens; one-command Google/Apple console setup; `moth doctor`. |
| [09](09-website.md) | Public website | Static landing page + single-sourced docs (Astro/Starlight, GitHub Pages), brand foundation. |
| [10](10-hardening-release.md) | Hardening & release | Security hardening, audit log, backups, packaging, final docs, v1.0. |

Milestones are strictly ordered by dependency: each ends with a demoable state. 01–02 make the auth engine real, 03 makes it operable, 04–05 make the headline developer experience real, 06–08 make it lovable and automatable (08 needs only 03+04 and can run in parallel with 05–07), 09 gives it a front door (parallelizable; goes live with the release), 10 makes it shippable.

### Phase 2 — Monetization (post-v1.0, ships as v1.1)

| # | Milestone | Outcome |
|---|---|---|
| [11](11-subscriptions-entitlements.md) | Subscriptions & entitlements | Server engine: per-user App Store / Play subscriptions, store validation + notifications, entitlement derivation, admin management, developer-backend entitlement API; a built-in `none` tier so paid subscriptions stay optional. |
| [12](12-store-catalog.md) | Store catalog provisioning | Admin monetization screen + `moth setup billing`: define tiers once, reflect them into App Store Connect & Google Play (honest automation, guided fallback), wire store notifications. |
| [13](13-flutter-paywall.md) | Flutter purchasing & themed paywall | SDK entitlement state in `MothScope`, native purchase flow, and a themed, admin-configurable paywall screen (design system, like the login screen). |
| [14](14-subscription-analytics.md) | Subscription analytics | Revenue-per-month and subscriber dashboards on the project analytics tab; closes the phase as a coordinated v1.1 release. |

Phase 2 is dependency-ordered too: 11 is the engine, 12 gets tiers into the stores, 13 is the on-device purchase + paywall (can proceed against manually-created store products if 12's automation lags), 14 reports the money. It builds on the whole v1.0 stack — entitlements ride the milestone-04 store-credential and milestone-08 store-API infrastructure, the paywall extends the milestone-06 design system, and the analytics extend the milestone-07 pipeline.

### Phase 3 — Internationalization (post-v1.1, ships as v1.2)

| # | Milestone | Outcome |
|---|---|---|
| [15](15-localization.md) | Localization & customizable copy | Header-negotiated locale; a bundled message catalog with per-project overrides for the sign-in, sign-up, and paywall copy; localized hosted pages + emails; the Design tab restructured into Theme / Sign in / Sign up / Paywall sub-tabs with a per-language live preview of every SDK screen. |
| [16](16-localized-sdk.md) | Localized Flutter SDK | The SDK sends the device language, consumes the negotiated project copy (revision-cached), and falls back to bundled translations offline — `MothLoginScreen` (sign-in/sign-up) and `MothPaywallScreen` render fully localized with the project's own wording. |

Phase 3 is dependency-ordered: 15 delivers the server negotiation, the copy model, and the admin editor + preview; 16 makes the Flutter SDK send its language and render the negotiated, project-customized copy with a bundled offline fallback. It extends the milestone-06 design system (the Design tab and its live preview) and the milestone-05/13 config-delivery + client-cache mechanism (theme, paywall, now copy).

### Phase 4 — Web (post-v1.2, ships as v1.3)

| # | Milestone | Outcome |
|---|---|---|
| [17](17-stripe-billing.md) | Stripe billing (web) | Stripe as a third store in the milestone-11 engine: tiers gain a Stripe price, hosted Checkout + Billing Portal sessions, signature-verified webhooks with validating re-reads, full API-driven catalog provisioning, exact web revenue in the analytics. |
| [18](18-react-sdk.md) | React SDK + npm serving | `@moth/react` served from the binary via an embedded npm registry; `MothProvider` + hooks + a batteries-included login screen, themed (06) and localized (15) from day one; entitlement gating and a themed web paywall selling through Stripe Checkout (17) — the milestone-05 + 13 developer experience for the web. |

Phase 4 is dependency-ordered like the others: 17 extends the milestone-11 billing engine server-side (same status enum, derivation matrix, notifications table, and analytics stream — Stripe is a new `store` value, not a parallel system), and 18 packages the whole stack for the browser the way 05 did for Flutter — the auth protos already speak gRPC-Web (the admin SPA proves it), milestone 04 built the web OAuth flows, theme + copy delivery exist, and the milestone-13 paywall config now drives a web paywall too. The React SDK's auth story stands alone, so 18's login-facing work can proceed in parallel with 17; its billing surface lands once 17 does.

### Phase 5 — Native platform services (post-v1.3, ships as v1.4)

| # | Milestone | Outcome |
|---|---|---|
| [19](19-native-billing.md) | Native billing plugin | `moth_billing` served from `/pub`: first-party StoreKit 2 (Swift) + Play Billing Library (Kotlin) implementation of the milestone-13 `MothBillingAdapter` — one dependency, zero adapter code, receipts in lockstep with the server's validation; `/pub` learns to serve a package set. |
| [20](20-push-registration.md) | Push device registry (server) | `moth.push.v1` + `push_devices`: signed-in devices register their APNs / FCM / Web Push credential with permission state; upsert/rotation/staleness/feedback invalidation; `moth.server.v1` hands the live set to the developer's backend, which sends the pushes itself; admin Devices panel + per-project VAPID public key. |
| [21](21-push-sdks.md) | Push in the SDKs | `moth_push` plugin (native APNs + FCM token acquisition) and the `MothScope` lifecycle — register on sign-in, re-register on rotation, unregister on sign-out; `useMothPush()` Web Push subscription in `@moth/react`; permission prompts stay explicit app calls. |

Phase 5 is dependency-ordered: 19 stands alone on the milestone-13 adapter interface and teaches `/pub` multi-package serving; 20 is the server registry; 21 rides both — the 19 plugin-delivery model for its native halves and the 20 RPCs for its lifecycle. 19 and 20 are independent and can proceed in parallel; 21 closes the phase.

## Non-goals (v1)

- Other providers (GitHub, Facebook, phone/SMS, magic links) — architecture leaves room via the `identities` table.
- Multi-node / horizontal scaling — SQLite + single process is the point; a moth instance serves one team's portfolio.
- User-facing account portal (self-service profile page) — SDK covers in-app needs.
- iOS/Android native SDKs (Swift/Kotlin) — Flutter first; the published protos make generating native gRPC clients straightforward when the time comes.
