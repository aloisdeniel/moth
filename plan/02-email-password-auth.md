# Milestone 02 — Email/Password Authentication

## Goal

The complete end-user auth lifecycle for a project, over gRPC: signup, login, token refresh, logout, email verification, password reset. After this milestone moth is a working auth server, usable with `grpcurl` before any SDK exists.

## Deliverables

### Schema

- `users` — `id`, `project_id`, `email` (unique per project, stored lowercased), `email_verified_at`, `password_hash` (nullable), `display_name`, `avatar_url`, `custom_claims` (JSON, settable only by admin/server API — roles etc., embedded in the JWT), `disabled_at`, timestamps.
- `identities` — `id`, `user_id`, `provider` (`password` | `google` | `apple`), `provider_subject`, unique `(project_id, provider, provider_subject)`. Email/password creates a `password` identity now; social providers slot in at milestone 04 without schema change.
- `refresh_tokens` — `id`, `user_id`, `token_hash` (SHA-256), `family_id`, `device_info`, `expires_at`, `rotated_at`, `revoked_at`.
- `email_tokens` — `id`, `user_id`, `purpose` (`verify` | `reset` | `email_change`), `token_hash`, `payload` (e.g. the pending new email), `expires_at`, `consumed_at`.

### Public auth service (`moth.auth.v1.AuthService`, project resolved from `x-moth-key: pk_...` request metadata by an interceptor)

- `SignUp` — email + password (+ optional display name). Policy from project settings: password min length, whether email verification is required before login, whether public signup is open at all (invite-only projects reject SignUp), and an optional enumeration-safe mode (signup with an already-registered email returns OK and emails the owner a "you already have an account" note instead of erroring).
- `SignIn` — returns `{ access_token, refresh_token, expires_in, user }`. Uniform `INVALID_CREDENTIALS` error whether email exists or not.
- `RefreshToken` — rotate refresh token, new access token. **Reuse detection**: presenting an already-rotated token revokes the whole family.
- `SignOut` — revoke the presented refresh token (or family with `all_devices: true`).
- `GetMe` — current user from access token (`authorization: Bearer ...` metadata).
- `UpdateMe` — update display name; `ChangePassword` — requires current password, revokes other sessions.
- `RequestEmailVerification` + `ConfirmEmailVerification` — email verification.
- `RequestPasswordReset` + `ConfirmPasswordReset` — the request RPC always returns OK (no account enumeration); a completed reset revokes all refresh tokens.
- `RequestEmailChange` + `ConfirmEmailChange` — the new address is verified before the switch, the old address gets a notification (with a revert link valid 72 h), tokens reflect the new email on next refresh.
- `DeleteAccount` — self-service deletion, **required by App Store review (guideline 5.1.1) for any app with account creation**. Demands fresh re-authentication (current password, or a recent provider sign-in for social-only users); cascades identities, sessions, and email tokens; emits `user.deleted`; triggers Apple token revocation once 04 lands.

### Tokens

- Access token: ES256 JWT signed with the **project's own key** (milestone 01), `kid` header identifying it. 15 min TTL. Claims: `iss` (base URL + project slug), `sub` (user id), `aud` (project slug), `iat`, `exp`, `email`, `email_verified`, and `claims` (the user's `custom_claims` — how developers put roles/permissions in the token, à la Firebase custom claims / Auth0 app_metadata).
- Refresh token: 256-bit random, opaque, 30-day sliding window (configurable per project), stored hashed only.

### Server-side API (the developer's backend)

Once a mobile app has a moth JWT, the developer's own API needs to validate it — and usually manage users programmatically too. All of this lives in `moth.server.v1`, authenticated with the project **secret key** (`sk_...`) in metadata (never shipped in the app):

- **Offline verification (recommended)**: standard JWT verification against the project JWKS (`/p/{slug}/.well-known/jwks.json`) with any JOSE library — no moth round-trip per request, keys cached per `kid`. Because keys are per-project, a token minted for another project on the same moth instance can never pass: wrong key *and* wrong `iss`/`aud`.
- **Online verification**: `TokenService.IntrospectToken` — returns validity, claims, and revocation/disabled-user status that offline verification can't see. For backends that want instant revocation over verification latency.
- **User management** — the moth counterpart of the Firebase Admin SDK: `UserService.GetUser` / `ListUsers` / `CreateUser` / `UpdateUser` (including `custom_claims` — the only way to set them besides the admin UI) / `DisableUser` / `DeleteUser` / `RevokeUserSessions`. Claim changes take effect on the next token refresh; pair with `RevokeUserSessions` to force it.

### Email delivery

- `Mailer` interface with two transports: **SMTP** (host/port/user/pass/from via config) and **console** (logs the full email; dev default so the flow works with zero setup).
- Embedded plain-text + minimal HTML templates: verification, password reset. Links point at hosted confirmation pages (`/p/{slug}/verify`, `/p/{slug}/reset`) — simple server-rendered plain-HTTP pages (email clients open a browser, so these can't be gRPC) that invoke the confirm RPCs in-process, so the flow works before deep links exist.

### Safety baseline (full hardening in 10)

- argon2id (64MB, t=3, p=2) with parameters encoded in the hash for future migration.
- Per-IP and per-account throttling on the SignIn/SignUp/RequestPasswordReset RPCs via an interceptor (in-memory token bucket).
- Uniform error model: standard gRPC status codes plus a `google.rpc.ErrorInfo` detail carrying a stable machine-readable `reason` (`INVALID_CREDENTIALS`, `EMAIL_NOT_VERIFIED`, ...) the SDK will map to typed errors.

## Key design points

- **Project scoping is absolute** — every query includes `project_id`; a store-layer test asserts cross-project reads are impossible. Getting this wrong is the one unrecoverable class of bug for this product.
- **Identities from day one** — modeling `password` as just another identity means milestone 04 (social) and account linking are additive.
- **Settings on the project row** — JSON settings column (verification required?, password policy, token TTLs) editable via the existing admin projects API.

## Acceptance criteria

- Scripted end-to-end pass with `grpcurl` against a dev instance: SignUp → (verify via console-logged link) → SignIn → GetMe → RefreshToken → ChangePassword → old refresh rejected → reset flow → SignIn with new password.
- Refresh-token reuse triggers family revocation (test).
- Two projects with the same user email are fully independent (test).
- JWT from SignIn validates against the project's JWKS with a third-party JOSE library; the same token fails against another project's JWKS (tests).
- IntrospectToken with a valid `sk_` returns claims; wrong project's secret key gets `PERMISSION_DENIED`; disabled user's still-unexpired JWT introspects as inactive (tests).
- `custom_claims` set via `UserService.UpdateUser` appear under `claims` in the next refreshed JWT (test).
- Email change round-trip: new address verified, old address receives revert link, revert restores it (test).
- `DeleteAccount` without fresh re-auth is rejected; with it, the user and all sessions are gone (test).
- Admin `UserService` can list/disable a project's users (minimal RPCs; UI in 03). Disabled users can't sign in or refresh.

## Out of scope

Social providers (04), themed login UI (06), analytics events beyond writing a stub `events` insert on signup/login (07).
