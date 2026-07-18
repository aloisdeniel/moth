# Changelog

All notable changes to moth are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/) and
[Semantic Versioning](https://semver.org/). From v1.0.0 onward this file is
regenerated from conventional-commit history by `git cliff` (see
`cliff.toml`); the pre-1.0 milestone entries below are the seed.

## [Unreleased]

### Native billing

- `moth_billing`: a first-party Flutter plugin implementing `moth_auth`'s
  `MothBillingAdapter` with StoreKit 2 (Swift) on iOS and the Play Billing
  Library (Kotlin) on Android — no third-party billing plugin, no adapter
  code to write.
- `/pub` now serves a package set: `moth_billing` ships alongside
  `moth_auth`, both stamped with the server version, with `moth_billing`'s
  `moth_auth` dependency pinned to that exact version at serve time.
- The example app, setup-instructions tab, docs and `moth skill export`
  present `moth_billing` as the default purchase path; a custom
  `MothBillingAdapter` remains the escape hatch.

### Push notifications

- Per-project push-device registry: signed-in devices register their APNs /
  FCM / Web Push credentials with an honest permission state via
  `moth.push.v1`; the developer's backend reads them (and reports dead
  credentials back) over `moth.server.v1.PushService` — moth registers,
  your server sends. Sign-out, feedback, staleness and admin revocation
  keep the registry self-healing; the admin gains a per-user Devices panel
  and per-project push settings (enable switch + Web Push VAPID public
  key).
- `moth_push`: a first-party Flutter plugin implementing `moth_auth`'s new
  `MothPushAdapter` with APNs (Swift) on iOS and Firebase Cloud Messaging
  (Kotlin) on Android, served from `/pub` alongside `moth_auth` and
  `moth_billing`. Wiring `MothApp(pushAdapter: MothNativePush())` is the
  whole opt-in: register on launch/rotation, unregister on sign-out, all
  non-fatal; the OS permission prompt stays behind an explicit
  `MothScope.requestPushPermission()`.
- `@moth/react` gains `useMothPush()`: Web Push subscription against the
  project's VAPID public key with feature detection and typed
  `unavailable`/`unsupported` states; the app supplies its own service
  worker (a minimal `sw.js` is documented).
- Examples, setup tab, docs (a new push guide) and `moth skill export`
  cover the full loop from registration to a backend send.

### Release engineering

- GoReleaser pipeline producing CGO-free binaries for darwin/linux/windows ×
  amd64/arm64, `.tar.gz`/`.zip` archives, a SHA-256 checksums file signed
  with keyless cosign, a Homebrew tap formula, and scratch-based multi-arch
  Docker images (~15 MB) published to GHCR and cosign-signed.
- `release.yml` workflow triggered on `v*` tags; the SDK version served at
  `/pub` and the embedded `/docs` are stamped to the release version via
  ldflags, so binary, SDK and docs land together.
- `govulncheck` and `gosec` (fail on high) added to CI.
- Embedded, version-matched documentation served at `/docs`, single-sourced
  from the public website content.
- `ghz` load-test harness under `scripts/loadtest/` with an honest,
  record-your-own-numbers baseline.

## [0.9.0] — Milestone 09: Public website

- Astro + Starlight marketing site and documentation tree, deployed to
  GitHub Pages, with a single-sourced CLI reference and seeded screenshots.

## [0.8.0] — Milestone 08: Admin CLI & provider setup

- `moth` cobra CLI (admin, project, user, token, instance, doctor, stats)
  and one-command Google/Apple provider setup.

## [0.7.0] — Milestone 07: Analytics

- Async event pipeline, daily rollups, and the admin analytics dashboard.

## [0.6.0] — Milestone 06: Design system & themed login

- Per-project design tokens, themed hosted login/verify/reset pages, and the
  Flutter SDK's themed login widget with golden tests.

## [0.5.0] — Milestone 05: Flutter SDK & package serving

- `moth_auth` Flutter SDK served from the instance's own pub repository at
  `/pub`, its version pinned to the binary.

## [0.4.0] — Milestone 04: Sign in with Google & Apple

- OAuth/OIDC social sign-in with per-project provider credentials and a
  web-redirect fallback.

## [0.3.0] — Milestone 03: Admin web application

- React admin SPA (embedded) for projects, users, keys and settings.

## [0.2.0] — Milestone 02: Email/password authentication

- Sign-up, sign-in, email verification, password reset, refresh-token
  rotation and reuse detection.

## [0.1.0] — Milestone 01: Foundations

- One-binary server: config resolution, SQLite store with embedded
  migrations, master key, per-project ES256 keys, JWKS, and the connect RPC
  scaffolding.
