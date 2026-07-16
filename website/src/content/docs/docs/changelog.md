---
title: Changelog
description: What each moth milestone delivered, and what v1.0 still needs.
---

moth is pre-1.0 and built in ordered milestones, each ending in a
demoable state. Versioned release notes begin at **v1.0**; until then this
page tracks the milestones as they land. The authoritative history is the
[git log](https://github.com/aloisdeniel/moth/commits/main).

## Unreleased

Working toward **v1.0** — the hardening & release milestone: security
hardening, an append-only audit log, first-class backups, packaging
(signed binaries, Homebrew tap, Docker image), built-in ACME, and the
final docs pass. Items marked **coming in v1.0** across these docs land
here. Nothing below requires v1.0 to run today.

## Milestones delivered

### 09 · Public website
This site: a static landing page and single-sourced documentation
(Astro + Starlight), built on the [`DESIGN.md`](https://github.com/aloisdeniel/moth/blob/main/DESIGN.md)
visual system with self-hosted fonts and zero external requests. The docs
tree also embeds into the binary to serve version-matched docs at `/docs`
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
