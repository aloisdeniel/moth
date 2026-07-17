# moth — Mobile + Auth

## Vision

moth makes authentication for a new mobile app trivial: run **one small binary**, create a project in the admin web app, add the served Flutter package to your `pubspec.yaml`, wrap your app in a login widget — done.

The defining idea: **one moth server carries your entire portfolio of independent apps**. An indie developer, studio, or agency runs a single instance; every app they ship is just another project on it — with its own users, signing keys, provider credentials, branding, and analytics, fully isolated from the others. Adding app #10 costs exactly what app #1 did: one project created in the admin, zero new infrastructure.

## Core capabilities

- **Single binary server** — no external dependencies, embedded database, embedded admin UI, embedded Flutter package. `moth serve` and you're running.
- **One server, many independent apps** — unlimited projects per instance, each a sealed tenant: its own user base, its own signing keypair, its own Google/Apple credentials, its own login branding. Nothing is shared across projects unless the operator is looking at the admin console.
- **Per-project authentication** — email/password, Sign in with Google, Sign in with Apple. Users belong to exactly one project; the same email in two apps is two unrelated accounts.
- **Subscriptions & entitlements** (post-v1.0) — per-user App Store / Google Play subscriptions, validated server-side, with entitlements (`pro`, …) derived from subscription state and grants (promos, grace periods). A built-in `none` (free) tier is always present, so paid subscriptions are optional. moth centralizes subscription state, reflects tiers into the stores, drives a themed Flutter paywall, and reports revenue per month.
- **Internationalization** (post-v1.1) — the app sends its language as an HTTP header; moth negotiates the best available locale and returns adapted copy for the SDK screens, hosted pages, and emails, from a bundled curated locale set. Operators customize the sign-in, sign-up, and paywall copy per language in the admin and preview every SDK screen for any language.
- **Admin web application** — create/configure projects, manage users, view analytics, copy setup instructions, customize the login design system (colors, font, spacing, logo) and per-language copy.
- **Admin CLI** — the same binary drives any moth instance from the terminal (personal access tokens): scriptable project/user management, declarative `moth project apply`, and one-command Google/Apple console configuration (`moth setup google|apple`).
- **Agent-ready** — `moth skill export` emits an Agent Skills package (optionally interpolated with a project's real config) that teaches coding agents both how to integrate moth into an app and how to administer an instance through the CLI.
- **Served Flutter package** — the server exposes a pub-compatible repository so developers reference the SDK directly from their moth instance in `pubspec.yaml`. The SDK provides a wrapper widget for the whole app and exposes auth state through an `InheritedWidget`.

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
│  /pub/*           → pub repository API (HTTP) serving moth_auth           │
│  /p/*, /assets/*  → hosted pages & project assets (HTTP)                  │
│  /billing/*       → App Store / Play store notifications (HTTP, post-v1.0) │
│  /p/{slug}/.well-known/jwks.json → per-project public keys (HTTP)         │
│                                                                          │
│  SQLite (data/moth.db) · file store (data/uploads/) · keys (data/keys/)  │
└──────────────────────────────────────────────────────────────────────────┘
        ▲                                ▲
        │ admin browser                  │ Flutter app (moth_auth SDK)
```

## Data model (sketch)

- `admins` — operators of the moth instance.
- `projects` — one per mobile app: name, slug, publishable key, secret key, provider config (Google/Apple credentials), theme JSON, settings.
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

## Non-goals (v1)

- Other providers (GitHub, Facebook, phone/SMS, magic links) — architecture leaves room via the `identities` table.
- Multi-node / horizontal scaling — SQLite + single process is the point; a moth instance serves one team's portfolio.
- User-facing account portal (self-service profile page) — SDK covers in-app needs.
- iOS/Android native SDKs (Swift/Kotlin) — Flutter first; the published protos make generating native gRPC clients straightforward when the time comes.
