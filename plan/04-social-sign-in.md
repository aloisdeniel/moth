# Milestone 04 — Sign in with Google & Apple

## Goal

Per-project social sign-in with account linking. Native mobile flows are the primary path (the Flutter SDK will use them in 05); a web-redirect fallback covers testing and non-mobile clients.

## Deliverables

### Provider configuration (admin)

- Project settings gain provider sections, editable in the admin SPA:
  - **Google**: OAuth client IDs (iOS, Android, web) — used as allowed `aud` values when verifying ID tokens.
  - **Apple**: Services ID, Team ID, Key ID, private key (`.p8`, stored encrypted at rest with an instance key), bundle IDs.
- Enable/disable toggle per provider per project; the SDK's login screen (05/06) reflects this via the public `GetProjectConfig` RPC.
- Admin setup screens include inline step-by-step guides (what to create in Google Cloud Console / Apple Developer portal, exact redirect URI to paste) — configuring Apple auth is notoriously painful; hand-holding here is a differentiator.

### Native token flow (primary)

- `AuthService.SignInWithOAuth` — request: provider enum (`GOOGLE` | `APPLE`), provider ID token (+ Apple's `authorization_code` and first-launch full name, since Apple only sends the name once).
- Verification server-side, no outbound dependency beyond provider JWKS (cached with expiry):
  - Google: verify ID token signature against Google JWKS, `iss`, `aud` ∈ configured client IDs, `exp`.
  - Apple: verify against Apple JWKS, `iss`, `aud` ∈ configured bundle/Services IDs; exchange `authorization_code` for refresh token validation when configured.
  - Both: **nonce check** — the SDK generates a per-attempt nonce (sent SHA-256-hashed to Apple per their scheme), passes the raw value in the RPC, and the server requires the token's `nonce` claim to match; replayed ID tokens are rejected.
- Identity resolution against the milestone-02 `identities` table:
  1. `(provider, subject)` exists → login that user.
  2. Else if verified email matches an existing user **and** the incoming token's email is verified → link new identity to that user (project setting: `auto_link_verified_email`, default on).
  3. Else → create user + identity (`email_verified` from provider claim).
- Response identical to `SignIn` (access + refresh token) — one token pipeline.

### Web redirect flow (fallback)

- The redirect legs are plain HTTP by necessity — OAuth consent screens are browser round-trips, not RPCs: `GET /oauth/{provider}/start?...` → provider consent → `.../callback` → validates `state` (signed, short-lived), performs code exchange, redirects to the app's registered custom scheme (`{slug}://auth?code=...`) or shows a hosted success page.
- The app then calls `AuthService.ExchangeOAuthCode` (gRPC) to trade the one-time code for tokens. Registered redirect schemes are a project setting (open-redirect protection).
- This flow is what "Sign in with Apple" on Android uses.

### Supporting work

- `users` without passwords fully supported in admin UI (provider badges from 03 now populated; "force password reset" hidden for social-only users).
- `AuthService.UnlinkIdentity` RPC — refused if it would leave zero login methods.
- Apple client secret generation (signed ES256 JWT from the `.p8`) with caching and rotation before the 6-month expiry.
- **Apple token revocation on account deletion**: `DeleteAccount` (02) for a user with an Apple identity calls Apple's `/auth/token/revoke` with the stored refresh token — an App Store review requirement paired with the in-app deletion one. Stored Apple refresh tokens live encrypted under the master key.

## Key design points

- **Verify locally, trust nothing from the client** — email/name/subject only ever read from the *verified* token, never from request fields (Apple name is the one documented exception, flagged as client-asserted).
- **Linking rules are security-sensitive** — only auto-link when the provider asserts the email is verified; otherwise create a separate account. Document the matrix in code comments and tests.
- **No provider SDKs server-side** — plain OIDC verification keeps the binary small and auditable.

## Acceptance criteria

- With real test credentials: Google and Apple native-token login create users, repeat login reuses them, and password-then-Google on the same verified email links (one user, two identities).
- Token with wrong `aud`, expired token, tampered signature, and wrong/missing nonce all rejected (table-driven tests against recorded fixtures + a JWKS test double).
- Deleting an Apple-linked account issues the revoke call (asserted against a test double).
- Web fallback round-trips against real Google in a manual checklist; `state` tampering rejected.
- Admin can disable a provider; `GetProjectConfig` and sign-in attempts reflect it immediately.

## Out of scope

The Flutter-side integration of these flows (05), login UI branding (06), additional providers (post-v1).
