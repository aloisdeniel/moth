---
title: Changelog
description: What each moth milestone delivered, and what v1.0 still needs.
---

moth is pre-1.0 and built in ordered milestones, each ending in a
demoable state. Versioned release notes begin at **v1.0**; until then this
page tracks the milestones as they land. The authoritative history is the
[git log](https://github.com/aloisdeniel/moth/commits/main).

## Unreleased — v1.0

**v1.0 is the first public release.** It bundles everything below —
milestones 01 through 22 — and cuts the release itself: signed,
checksummed release binaries for darwin/linux/windows (amd64/arm64), a
Homebrew tap, cosign-signed scratch Docker images, and the final docs
pass. The hardening milestone's substance (persistent rate limiting, an
append-only audit log, `moth backup`/`restore` plus scheduled snapshots, a
`/metrics` endpoint, structured JSON logs, and built-in ACME) is already
in the binary; the remaining work is publishing the artifacts.

## Milestones delivered

### 22 · Guided project creation
An adaptive [project-creation wizard](../cli/) in the console and CLI that
walks a new project through providers, theme, and keys, adjusting the
steps to what you're building.

### 21 · Push registration in the SDKs
Device registration wired into the Flutter and React SDKs — the app
registers its APNs/FCM/Web Push credential and moth keeps the
[push registry](../guides/push/) current across launches.

### 20 · Push device registry (server core)
A project-scoped [push-device registry](../guides/push/): the server
stores each user's active push credentials and exposes them to your sender
over `moth.server.v1.PushService` (the only surface that returns tokens).

### 19 · Native billing (first-party store plugin)
First-party in-app purchase handling for the Flutter SDK — StoreKit and
Google Play Billing — feeding the same
[entitlement model](../guides/monetization/) as the other stores.

### 18 · React SDK & npm serving
The [`@moth/react`](../react/) package, served from each instance's own
npm registry at `/npm`: hooks and components mirroring the Flutter SDK for
web apps.

### 17 · Stripe billing (web)
[Stripe subscriptions](../guides/monetization/) as a first-class store
alongside Apple and Google — checkout, webhooks, and revenue that sits in
the same analytics and entitlement model as native purchases.

### 16 · Localized Flutter SDK
The SDK login screen, paywall, and error messages render in the user's
locale, driven by the project's [customizable copy](../guides/theming/).

### 15 · Localization & customizable copy
Per-project, per-locale [copy and localization](../guides/theming/) for
the login screen, paywall, hosted pages, and emails — edited in the
console, delivered without an app release.

### 14 · Subscription analytics
Revenue [dashboards](../guides/analytics/) — MRR per currency, active
subscribers with trend, new vs churned, trial-to-paid conversion, and
per-tier / per-store breakdowns.

### 13 · Flutter purchasing & themed paywall
A batteries-included, themed [paywall](../guides/monetization/) in the
Flutter SDK, wired to the store catalog and the entitlement model.

### 12 · Store catalog provisioning
Admin [store-catalog](../guides/monetization/) management: products, tiers,
and entitlements defined once and mapped to each store's identifiers.

### 11 · Subscriptions & entitlements (server core)
The server-side [subscription and entitlement](../guides/monetization/)
model — receipts verified server-side, entitlements exposed to the SDK and
your backend over `moth.server.v1`.

### 10 · Hardening & release
The v1.0 hardening pass: persistent rate limiting, an append-only admin
audit log, first-class backups (`moth backup` / `moth restore` and
scheduled snapshots), a Prometheus `/metrics` endpoint, structured JSON
logs, built-in ACME, and signed release packaging.

### 09 · Public website
This site: a static landing page and single-sourced documentation
(Astro + Starlight), built on the [`DESIGN.md`](https://github.com/aloisdeniel/moth/blob/main/DESIGN.md)
visual system, with no analytics or trackers (the display font is served
from Fontshare's CDN with a system-font fallback). The docs tree also
embeds into the binary to serve version-matched docs at `/docs`
(finalized in v1.0).

### 08 · Admin CLI & one-command provider setup
The `moth` binary doubles as a remote client with named contexts and
personal access tokens: scriptable [project and user management](../cli/),
declarative `moth project apply`, `--json` everywhere,
[`moth setup google|apple`](../guides/google/) for one-command provider
configuration, `moth doctor` for diagnosis, and
[`moth skill export`](../agents/) for coding agents.

### 07 · Analytics
Per-project [event capture and dashboards](../guides/analytics/) — signups,
logins, DAU, success rate, provider/platform breakdowns — with nightly
rollups and a privacy-respecting model that stores no IPs or device IDs.

### 06 · Design system & themed login
A per-project [theme editor](../guides/theming/) — colors, typography,
spacing, radius, logo, legal links — rendered across the SDK login screen,
hosted pages, and emails, with contrast validation and no-app-release
delivery.

### 05 · Flutter SDK & pub serving
The [`moth_auth`](../sdk/) package, served from each instance's own
[pub repository](../sdk/) at `/pub`: `MothApp` wraps the app, `MothScope`
exposes auth state, `MothLoginScreen` is batteries-included, and
`MothClient` auto-refreshes the token you attach to your backend calls.

### 04 · Sign in with Google & Apple
Per-project [social sign-in](../guides/google/) with server-side token
verification (provider JWKS, nonce, `aud`), account linking on
provider-verified email, and a web-redirect fallback for Android/web.

### 03 · Admin web app
The embedded console at `/admin`: projects, API keys, user management, the
per-project Setup instructions, and instance/SMTP settings.

### 02 · Email/password authentication
The full auth lifecycle over gRPC — signup, sign-in, refresh with rotation
and reuse detection, verification, password reset, email change, account
deletion — plus the [`moth.server.v1`](../api/#mothserverv1) backend API
and [token verification](../api/#verifying-tokens-on-your-backend).

### 01 · Foundations
The runnable binary: CLI, config resolution, SQLite with embedded
migrations, the project model, per-project ES256 signing keys and JWKS,
admin bootstrap, and CI.
