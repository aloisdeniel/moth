# API reference

moth is protobuf-first: everything except a handful of browser- and
tool-facing HTTP endpoints is gRPC, served with
[Connect](https://connectrpc.com/) so the same port speaks native gRPC
(the Flutter SDK), gRPC-Web (the admin console), and Connect's HTTP/JSON.

There are **three API surfaces, each with its own credential**. The
credential is the security boundary — it decides which surface a caller
can reach:

| Surface | Credential (request metadata) | Consumer |
|---|---|---|
| [`moth.auth.v1`](#mothauthv1) | publishable key — `x-moth-key: pk_…` | the mobile app, via the SDK |
| [`moth.server.v1`](#mothserverv1) | secret key — `x-moth-secret: sk_…` | your own backend |
| [`moth.admin.v1`](#mothadminv1) | admin session cookie or `authorization: Bearer moth_pat_…` | admin console, `moth` CLI |

Everything is project-scoped: `pk_`/`sk_` name a project directly, and a
token minted for one project can never be used against another.

Two further end-user services ride the publishable key plus the user's
Bearer token: `moth.billing.v1` (subscriptions — see the
[monetization guide](../guides/monetization/)) and `moth.push.v1`
(push-device registration — see the
[push guide](../guides/push/)).

## moth.auth.v1

The end-user surface the SDK calls, authorized by the publishable key
(safe to embed in the app). The project is resolved from the `x-moth-key`
metadata by an interceptor; per-IP and per-account rate limits guard the
credential-facing RPCs.

- **`AuthService`** — the lifecycle: `SignUp`, `SignIn`, `RefreshToken`,
  `SignOut`, `GetMe`, `UpdateMe`, `ChangePassword`, the email-verification
  / password-reset / email-change request+confirm pairs,
  `SignInWithOAuth` and `ExchangeOAuthCode` for social sign-in,
  `UnlinkIdentity`, and `DeleteAccount`.
- **`ConfigService.GetProjectConfig`** — the project's public config:
  which providers are enabled, Google client IDs, password policy, and the
  [theme](../guides/theming/). The SDK reads it so the login screen adapts
  without an app release.

`SignIn`, `SignInWithOAuth`, and `RefreshToken` all return the same
`{ access_token, refresh_token, expires_in, user }` — one token pipeline
regardless of how the user authenticated.

The Flutter SDK wraps all of this; you rarely call it directly. See the
[SDK reference](../sdk/#client-core).

### Tokens

- **Access token** — an ES256 JWT signed with the **project's own**
  signing key, 15-minute TTL, the `kid` header naming the key. Verify it
  as below.
- **Refresh token** — a 256-bit opaque random string, stored server-side
  as a SHA-256 hash only, on a sliding window (30 days by default, per
  project). It **rotates on every use**; presenting an already-rotated
  token is treated as theft and revokes the whole token family.

The access-token claim set:

| Claim | Value |
|---|---|
| `iss` | `<base URL>/p/<project-slug>` |
| `sub` | user id (UUIDv7) |
| `aud` | `<project-slug>` |
| `iat` / `exp` | issued-at / expiry (15 min) |
| `email` | user email |
| `email_verified` | boolean |
| `claims` | the user's [custom claims](#custom-claims) |

### Custom claims

`claims` carries per-user roles/permissions — moth's equivalent of
Firebase custom claims. They are settable **only** by an admin (console or
CLI) or your backend via `moth.server.v1` — never by the app — and appear
in the next refreshed access token. To force existing sessions to pick up
a change immediately, revoke the user's sessions so their next refresh
re-mints the token. Readable client-side on `MothUser.claims` for UI
gating, but the server is always the authority.

## moth.server.v1

The surface your **own backend** calls, authorized by the secret key
(`sk_…`, never shipped in the app). It is the moth counterpart of a
server-side admin SDK.

- **`TokenService.IntrospectToken`** — online verification: returns
  validity, the claims, and revocation / disabled-user status that offline
  verification can't see. Use it when you need instant revocation over
  raw latency.
- **`UserService`** — programmatic user management: `GetUser`,
  `ListUsers`, `CreateUser`, `UpdateUser` (including `custom_claims`),
  `DisableUser` / `EnableUser`, `DeleteUser`, `RevokeUserSessions`.
- **`PushService`** — the read side of the
  [push-device registry](../guides/push/): `ListUserPushDevices` /
  `ListPushDevices` return the live push credentials your sender needs
  (this is the only surface that ever returns tokens), and
  `RevokePushDevice` is the feedback loop for credentials APNs/FCM/Web
  Push report dead.

A secret key names exactly one project; using another project's `sk_`
gets `PERMISSION_DENIED`.

### Verifying tokens on your backend

Two ways to validate the JWT the app sends your API:

**Offline (recommended).** Verify the token against the project's JWKS
with any standard JOSE library — no round-trip to moth per request, keys
cached by `kid`. Because keys are per-project, a token minted for another
app on the same instance can never pass: wrong key *and* wrong
`iss`/`aud`.

```
JWKS  https://auth.example.com/p/<project-slug>/.well-known/jwks.json
iss   https://auth.example.com/p/<project-slug>
aud   <project-slug>
alg   ES256
```

Check, at minimum: `alg` is `ES256` and the header `kid` resolves to a
JWKS key; the signature is valid; `exp` has not passed; `iss` is exactly
your instance's `<base URL>/p/<slug>`. On an unknown `kid`, refetch the
JWKS (key rotation) — bounded, e.g. once a minute.

The repository ships a complete, ~200-line standard-library
[example backend](https://github.com/aloisdeniel/moth/tree/main/scripts/example_backend)
doing exactly this in Go; the project's Setup tab prints ready-made
verifier snippets for Node (`jose`), Go (`lestrrat-go/jwx`), and Dart with
your real values.

**Online.** Call `TokenService.IntrospectToken` with your `sk_`. Slower
(a network hop per check) but sees revocations and disabled users
immediately — a token that still looks valid offline introspects as
inactive once the user is disabled or their sessions are revoked.

## moth.admin.v1

The surface behind the admin console and the `moth` CLI, authorized by an
admin session cookie or a personal access token. It is the widest surface
and never reachable with a `pk_` or `sk_`.

- **`SessionService`** — admin login/logout, current admin.
- **`ProjectService`** — project CRUD, `RegenerateSecretKey`,
  `GetSigningKey`, `ResetSigningKey` (new keypair, old key dropped from the
  JWKS, all refresh tokens revoked — atomic).
- **`UserService`** — the cookie-authed façade over the same user
  management as `moth.server.v1`, plus `SendPasswordReset`.
- **`AnalyticsService`** — `GetStats`, `ListRecentEvents`, `RunRollup`
  (see [Analytics](../guides/analytics/)).
- **`ThemeService`** — theme get/update, revisions, logo upload (see
  [Theming](../guides/theming/)).
- **`AdminAccountService`** — admin invites, password change, and
  **personal access token** management (`CreatePersonalAccessToken`,
  `List…`, `Revoke…`) — the credentials the CLI and agents use.
- **`InstanceSettingsService`** — SMTP config and test send.

The [CLI](../cli/) is a thin generated client over these services, which
is why it can't lag the console in capability.

## Plain-HTTP endpoints

Some consumers can't speak gRPC — browsers, the `dart pub` client,
third-party JWT libraries — so a few surfaces stay plain HTTP:

- `GET /p/{slug}/.well-known/jwks.json` — the project's active public
  signing keys, for offline verification.
- `GET /healthz` — liveness (alongside the standard gRPC health service).
- `/pub/*` — the [pub repository](../sdk/) serving `moth_auth`.
- `/oauth/{provider}/…` — the web-redirect fallback for social sign-in.
- `/assets/{project}/…` — project logo/font assets.

### Hosted pages

`/p/{slug}/verify`, `/p/{slug}/reset`, and `/p/{slug}/confirm-email` are
server-rendered pages the transactional emails link to. The app *requests*
verification / reset / email-change over gRPC; the user *completes* it by
following the emailed link to these pages, which invoke the confirm RPCs
in-process. They render with the project [theme](../guides/theming/) and
ship with zero external asset requests.

### The protos

Every instance serves its `.proto` sources at **`/protos/`**. Point `buf`
or `protoc` at them to generate a typed client in any language and call
the surfaces above directly — the same protos the SDK, console, and CLI
are generated from. `buf breaking` in CI guarantees wire compatibility
within a major version.

## Errors

Every RPC returns a standard gRPC status code plus a
`google.rpc.ErrorInfo` detail carrying a **stable, machine-readable
`reason`** — `INVALID_CREDENTIALS`, `EMAIL_NOT_VERIFIED`,
`RATE_LIMITED`, `WEAK_PASSWORD`, … The SDK maps these to
[typed exceptions](../sdk/#errors); build your own client the same way.
Authentication failures are deliberately uniform (`INVALID_CREDENTIALS`
whether or not the email exists) and account-enumeration-safe RPCs
(`RequestPasswordReset`, enumeration-safe `SignUp`) always return `OK`.
