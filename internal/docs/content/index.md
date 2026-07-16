# moth documentation

moth is a self-hosted authentication server for mobile apps. One instance
hosts your whole portfolio: every app you ship is a **project** — a sealed
tenant with its own users, its own ES256 signing keypair, its own
Google/Apple credentials, its own login branding, its own analytics.
Adding app #10 costs exactly what app #1 did: one project created in the
admin, zero new infrastructure.

Everything ships inside a single Go binary: the SQLite database, the admin
web console, the hosted email pages, the `moth_auth` Flutter SDK (served
from the instance's own pub repository), and the admin CLI. `moth serve`
and you're running.

## Where to start

- **[Quick start](/docs/quick-start)** — from an empty server to a logged-in
  Flutter app in ten minutes. Start here.
- **[Installation & deployment](/docs/installation)** — configuration, systemd,
  Docker, reverse proxies, TLS.
- **[Sign in with Google](/docs/guides/google)** and
  **[Sign in with Apple](/docs/guides/apple)** — provider setup, automated by
  `moth setup google` / `moth setup apple` where the provider consoles
  allow it.
- **[Theming](/docs/guides/theming)** — brand each project's login screens from
  the admin, no app release needed.
- **[Flutter SDK reference](/docs/sdk)** — `MothApp`, `MothScope`,
  `MothLoginScreen`, and the full `MothClient` API.
- **[CLI reference](/docs/cli)** — every command, generated from the binary.
- **[API reference](/docs/api)** — the gRPC surfaces, token claims, JWKS, and
  how your backend verifies moth tokens.
- **[Security & threat model](/docs/security)** — how secrets are stored, how
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
│                     token introspection, user management                │
│  /pub/*           → pub repository serving the moth_auth Flutter SDK    │
│  /p/{slug}/*      → hosted verify/reset/confirm-email pages             │
│  /p/{slug}/.well-known/jwks.json → per-project public signing keys      │
│  /oauth/*         → web-redirect fallback for social sign-in            │
│  /protos/         → the .proto sources, for generating your own clients │
│                                                                         │
│  SQLite (data/moth.db) · uploads (data/uploads/) · keys (data/keys/)    │
└─────────────────────────────────────────────────────────────────────────┘
```

Three API surfaces, three credentials — see the [API reference](/docs/api) for
the full contract:

| Surface | Credential | Consumer |
|---|---|---|
| `moth.auth.v1` | publishable key (`pk_…`), safe to embed in the app | the mobile app, via the SDK |
| `moth.server.v1` | secret key (`sk_…`), never leaves your servers | your own backend |
| `moth.admin.v1` | admin session or personal access token | admin console, `moth` CLI |
