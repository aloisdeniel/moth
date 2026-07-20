---
title: Security & threat model
description: How moth stores secrets, signs tokens, and isolates projects — and what v1 does and does not defend.
---

An auth product earns trust by being exact about what it does. This page
states how moth protects credentials and tokens, and — just as important —
what is out of scope for v1.

## Project isolation

The core invariant: **every user, key, credential, and token hangs off a
project, and nothing is shared across projects.** Every store query is
`project_id`-scoped, with a store-layer test asserting cross-project reads
are impossible. Two apps on one instance with the same user email are two
unrelated accounts.

Isolation is enforced cryptographically as well as in queries: each
project has its **own ES256 signing keypair**, so an access token minted
for one project can never validate for another — it is signed by the wrong
key *and* carries the wrong `iss`/`aud`. Compromising or rotating one
project's key touches only that project.

## Secrets at rest

| Secret | Storage |
|---|---|
| User passwords | **argon2id** (OWASP baseline `m=19456 KiB, t=2, p=1`), parameters encoded in the hash for future migration |
| Refresh tokens | **SHA-256 hash only** — the plaintext exists only on the client |
| Secret keys (`sk_`) & personal access tokens (`moth_pat_`) | **SHA-256 hash only** — shown once at creation, never again |
| Project signing private keys | **AES-GCM encrypted** under the instance master key |
| Provider secrets (Apple `.p8`, Google web secret, stored Apple refresh tokens) | **AES-GCM encrypted** under the instance master key |

The **master key** lives at `data/keys/master.key`, or is injected via
`MOTH_MASTER_KEY` (KMS-style) so no key file need touch disk. It encrypts
secrets at rest and nothing else. Losing it means losing every project's
signing keys and provider secrets — back it up independently of the
database (see [Backups](../guides/backups/)).

Publishable keys (`pk_…`) are **not** secret — they identify a project to
the SDK and are meant to ship in the app. Authorization comes from the JWT
the user holds, never from the `pk_`.

## Tokens & sessions

- **Access tokens** are short-lived (15 min) ES256 JWTs. Your backend
  verifies them offline against the project JWKS — see
  [verifying tokens](../api/#verifying-tokens-on-your-backend).
- **Refresh tokens rotate on every use.** Presenting an already-rotated
  token is treated as theft and **revokes the entire token family**, so a
  stolen-and-replayed refresh token logs the attacker *and* the victim out
  rather than granting silent access.
- **Revocation.** `ResetSigningKey` invalidates every token a project ever
  issued at once; `RevokeUserSessions` / disabling a user kills their
  sessions. Offline verifiers see key resets immediately (the `kid`
  vanishes from the JWKS); for instant per-user revocation without waiting
  out the 15-minute access-token TTL, use online
  [introspection](../api/#mothserverv1).

## Social sign-in

Provider tokens are verified server-side against the provider's JWKS —
signature, `iss`, `exp`, and `aud` against the project's configured client
/ bundle IDs. A per-attempt **nonce** is required so a replayed ID token is
rejected. Email, name, and subject are read **only from the verified
token**, never from client-supplied request fields (Apple's first-launch
name is the one documented, clearly-flagged exception). Accounts are
auto-linked across providers only when the provider asserts the email is
verified. No provider SDKs run server-side — plain OIDC verification keeps
the binary small and auditable.

## Abuse resistance

- Per-IP, per-account, and per-project **rate limiting** on the
  credential-facing RPCs — `SignIn`, `SignUp`, `RequestPasswordReset`.
  Limits are persisted in the database (fixed windows), so they survive a
  restart and can't be reset by bouncing the process; the tiers are
  configurable (`MOTH_RATELIMIT_*` / `[ratelimit]`) and behind a proxy the
  real client IP comes from `X-Forwarded-For` only for
  [trusted proxies](../installation/#configuration).
- **Account-enumeration-safe** by design: `SignIn` returns a uniform
  `INVALID_CREDENTIALS` regardless of whether the email exists;
  `RequestPasswordReset` always returns `OK`; projects can enable an
  enumeration-safe `SignUp` that emails the existing owner instead of
  erroring.
- **Open-redirect protection** on the OAuth web fallback: callbacks only
  ever redirect to custom URL schemes on the project's allow-list, and the
  `state` parameter is signed and short-lived.

## Privacy stance

moth is an auth server, not a product-analytics tool, and the
[analytics](../guides/analytics/) reflect that:

- **No IP addresses stored** in the event stream, and **no device IDs.**
- No PII beyond what authentication already requires.
- Raw events have a capped, configurable retention (90 days by default),
  pruned automatically; dashboards read pre-aggregated rollups.

The [website you're reading](https://github.com/aloisdeniel/moth) practices
the same stance: no trackers, no analytics, no cookies. Its only external
request is the Satoshi display font from Fontshare's CDN, loaded
non-blocking with a system-font fallback. The end-user hosted pages the
binary serves (verify / reset / confirm-email) make **zero** external
requests — their fonts are self-hosted and served from the instance itself.

## In scope for v1

Email/password and Google/Apple sign-in; per-project ES256 keys and JWKS;
argon2id passwords; rotating refresh tokens with reuse detection;
encrypted secrets at rest; persistent rate limiting; account-enumeration
safety; self-service account deletion (App Store guideline 5.1.1)
including Apple token revocation; an append-only **admin audit log**; a
Prometheus **`/metrics`** endpoint and structured JSON logs; first-class
**backups** (`moth backup` / `moth restore`, plus scheduled snapshots);
and built-in **ACME**/Let's Encrypt TLS. Subscriptions & entitlements
(Apple, Google, Stripe), push-device registry, localization, and the
React SDK ship alongside the Flutter SDK.

## Out of scope for v1

Stated plainly, because credibility matters more than a feature checkbox:

- **MFA / TOTP / passkeys** — not in v1. The `identities` model leaves
  room; it's a post-v1 addition.
- **Other providers** — GitHub, Facebook, phone/SMS, magic links.
- **Managed scale / HA** — moth is single-process SQLite by design: one
  instance serves one team's portfolio, not a horizontally-scaled fleet.
- **A hosted moth service** — moth is self-hosted; there is no SaaS to
  trust with your users.

## Reporting a vulnerability

Found a security issue? Please report it **privately** — do not open a
public issue. Use GitHub's
[private vulnerability reporting](https://github.com/aloisdeniel/moth/security/advisories/new)
or follow the [security policy](https://github.com/aloisdeniel/moth/blob/main/SECURITY.md).
See [`SECURITY.md`](https://github.com/aloisdeniel/moth/blob/main/SECURITY.md)
for scope and the disclosure timeline.
