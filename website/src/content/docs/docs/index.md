---
title: moth documentation
description: Self-hosted authentication for all your mobile apps, in one binary.
---

moth is a self-hosted authentication server for mobile apps. One instance
hosts your whole portfolio: every app you ship is a **project** — a sealed
tenant with its own users, its own ES256 signing keypair, its own
Google/Apple credentials, its own login branding, its own analytics.
Adding app #10 costs exactly what app #1 did: one project created in the
admin, zero new infrastructure.

Everything ships inside a single Go binary: the SQLite database, the admin
web console, the hosted email pages, the Flutter SDK packages (`moth_auth`
plus the `moth_billing` and `moth_push` native companions, served from the
instance's own pub repository), the `@moth/react` React SDK (served from
the instance's own npm registry), and the admin CLI. `moth serve` and
you're running.

## Where to start

- **[Quick start](quick-start/)** — from an empty server to a logged-in
  Flutter app in ten minutes. Start here.
- **[Installation & deployment](installation/)** — configuration, systemd,
  Docker, reverse proxies, TLS.
- **[Sign in with Google](guides/google/)** and
  **[Sign in with Apple](guides/apple/)** — provider setup, automated by
  `moth setup google` / `moth setup apple` where the provider consoles
  allow it.
- **[Theming](guides/theming/)** — brand each project's login screens from
  the admin, no app release needed.
- **[Subscriptions & paywall](guides/monetization/)** — App Store, Google
  Play, and Stripe subscriptions validated server-side, distilled into
  entitlements, sold through a themed paywall — with `moth_billing`
  running the native purchase first-party.
- **[Push notifications](guides/push/)** — every signed-in device
  registers its APNs / FCM / Web Push credential with moth; your backend
  reads the registry and sends. moth registers, your server sends.
- **[Flutter SDK reference](sdk/)** — `MothApp`, `MothScope`,
  `MothLoginScreen`, and the full `MothClient` API.
- **[React SDK reference](react/)** — `MothProvider`, hooks, entitlement
  gates, and the web paywall backed by Stripe Checkout.
- **[CLI reference](cli/)** — every command, generated from the binary.
- **[API reference](api/)** — the gRPC surfaces, token claims, JWKS, and
  how your backend verifies moth tokens.
- **[Security & threat model](security/)** — how secrets are stored, how
  tokens are signed, and what v1 deliberately does not do.

## The shape of the system

```
┌────────────────────────── moth (single binary) ─────────────────────────┐
│                                                                         │
│  /admin           → embedded admin console (React SPA)                  │
│  moth.admin.v1.*  → admin gRPC services (session cookie or              │
│                     personal access token)                              │
│  moth.auth.v1.*   → end-user auth gRPC services (publishable key, pk_)  │
│  moth.server.v1.* → your backend's gRPC services (secret key, sk_):     │
│                     token introspection, user management, entitlements, │
│                     push-device reads                                   │
│  moth.billing.v1.*→ subscription gRPC services (publishable key, pk_)   │
│  moth.push.v1.*   → push-device registration (publishable key, pk_)     │
│  /pub/*           → pub repository serving moth_auth, moth_billing      │
│                     and moth_push                                       │
│  /npm/*           → npm registry serving the @moth/react React SDK      │
│  /p/{slug}/*      → hosted verify/reset/confirm-email pages             │
│  /p/{slug}/.well-known/jwks.json → per-project public signing keys      │
│  /oauth/*         → web-redirect fallback for social sign-in            │
│  /protos/         → the .proto sources, for generating your own clients │
│                                                                         │
│  SQLite (data/moth.db) · uploads (data/uploads/) · keys (data/keys/)    │
└─────────────────────────────────────────────────────────────────────────┘
```

Three API surfaces, three credentials — see the [API reference](api/) for
the full contract:

| Surface | Credential | Consumer |
|---|---|---|
| `moth.auth.v1` | publishable key (`pk_…`), safe to embed in the app | the mobile app, via the SDK |
| `moth.server.v1` | secret key (`sk_…`), never leaves your servers | your own backend |
| `moth.admin.v1` | admin session or personal access token | admin console, `moth` CLI |
