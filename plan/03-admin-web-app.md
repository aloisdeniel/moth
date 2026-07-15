# Milestone 03 — Admin Web Application v1

## Goal

A polished embedded admin console at `/admin`: log in, manage projects, manage each project's users, and copy working setup instructions. This is where a new user forms their opinion of moth — it must feel instant and obvious.

## Deliverables

### App shell

- React + Vite + TypeScript in `web/admin/`; `vite build` output embedded via `go:embed` and served at `/admin` (SPA fallback to `index.html`).
- Client stack kept lean: `react-router`, a TypeScript client generated from the `moth.admin.v1` protos (`connect-web`, speaking gRPC-Web) with `connect-query` for TanStack Query integration, a small component layer (Radix primitives + hand-rolled styles or Tailwind). No heavyweight UI kit — the admin's look should foreshadow moth's design-system feature.
- Session handling against the milestone-01 cookie auth: login page, logout, `UNAUTHENTICATED` status → redirect to login.
- First-run setup screen (create initial admin) replacing the milestone-01 placeholder.

### Screens

1. **Projects list** — cards with name, user count, created date; create-project dialog (name → slug preview).
2. **Project overview** — publishable key with copy button, secret key regeneration (old one invalidated, new one shown once), signing key card (JWKS URL, public key PEM download). Danger zone: **Reset signing key** — generates a new keypair, removes the old key from the JWKS immediately, and revokes all refresh tokens, so every issued token is invalid and all users must sign in again; typed confirmation spelling out exactly that. Also: delete project with typed confirmation.
3. **Project settings** — the milestone-02 settings JSON as a form: email verification required, password policy, token TTLs, signup mode (open / invite-only), enumeration-safe signup toggle; email sender config status.
4. **Users** — paginated, searchable table: email, name, providers (badges), verified, last login, created. **Add user** button (admin-created account: set a password or send an invite/set-password email — the counterpart of invite-only mode). Row actions: view detail, disable/enable, force password-reset email, revoke all sessions, delete. User detail drawer shows identities, active sessions, and a validated JSON editor for `custom_claims`.
5. **Setup instructions** — per-project, copy-paste ready and rendered with the project's real values (base URL, publishable key):
   - `pubspec.yaml` snippet pointing at this instance's pub repository (final URL shape lands in 05; render it now behind a "SDK available in vX" note if 05 isn't shipped).
   - Minimal Dart `main.dart` example.
   - **Backend verification section**: the project's JWKS URL, expected `iss`/`aud` values, and copy-paste token-verification snippets for common stacks (Node `jose`, Go `lestrrat-go/jwx`, Dart) — plus the IntrospectToken alternative with a `grpcurl` example using the secret key, and the downloadable `moth.auth.v1` / `moth.server.v1` proto files.
6. **Instance settings** — admin account management (invite additional admins, change password), SMTP configuration with "send test email" button.

### API work

- Flesh out the `moth.admin.v1` services backing all of the above: `UserService` (list/search/detail/create/invite/disable/delete, custom claims, revoke sessions — thin cookie-authed façade over the same domain layer as `moth.server.v1`), `ProjectService.ResetSigningKey` (new keypair + old key dropped + all refresh tokens revoked, atomically), `AdminAccountService` (invites, password change), `InstanceSettingsService` (SMTP settings + test send).
- Shared proto conventions established here: `page_token`/`page_size` pagination, `google.rpc.ErrorInfo` reasons, `FieldMask` for partial updates — reused by every later service.

### Dev workflow

- `make dev` runs Go server + Vite dev server with a gRPC-Web-aware proxy so frontend iteration doesn't require rebuilding the binary; `make proto` regenerates the connect-web client alongside the Go code; production build embeds `dist/`.
- CI builds the SPA, fails if `dist/` embedding or generated TS client is stale; Playwright smoke test (login → create project → see keys).

## Key design points

- **Instructions are the product** — the setup page must produce a copy-paste block that actually works against this instance, not generic docs. Values interpolated server-side via the admin API.
- **Secrets shown once** — publishable key always visible; secret key only at creation/regeneration. Matches user expectations from Stripe et al.
- **SPA stays dumb** — all invariants enforced by the services; the SPA is a pure generated-client consumer, so future CLI/automation gets the same power by generating a client from the same protos.

## Acceptance criteria

- Fresh instance → browser-only flow: first-run admin creation → create project → copy publishable key → configure SMTP → send test email → disable a user (created via `grpcurl` from milestone 02) → that user's RefreshToken call fails.
- Reset signing key → previously issued JWT no longer validates against the JWKS, IntrospectToken reports it invalid, and all refresh tokens are dead (test).
- Binary size impact of the embedded SPA < ~2 MB gzipped; `/admin` loads with no external network requests (fonts/assets embedded).
- Playwright smoke test green in CI.

## Out of scope

Design-system/theme editor (06), analytics dashboard (07), personal access token management UI (08), audit log UI (10).
