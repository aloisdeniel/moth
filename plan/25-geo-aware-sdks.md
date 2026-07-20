# Milestone 25 — Geo-Aware SDKs

## Goal

Make the cluster invisible to app developers. The Flutter SDK (05) and React SDK
(18) learn three things: **discover** the instance directory, **pick** the nearest
instance for first contact, and **stick** to the home instance the JWT names for
the life of the account. The developer still configures exactly one thing — a moth
URL and a publishable key; `MothScope` / `MothProvider` do the rest. An app built
against a standalone moth is byte-for-byte unaffected.

## Model

- **Instance directory** in the public project config (the same unauthenticated
  config RPC that already delivers theme and copy revisions): the list of active
  instances — id, region label, `base_url` — plus the directory's own revision for
  caching. Served identically by every instance (it propagates over the 23
  fabric), so any configured URL bootstraps the full map.
- **Client-side persistence** (secure storage on Flutter, localStorage on web),
  alongside the existing session: the chosen `home_url` + instance id, and the
  last-seen directory. The `mih` claim in every refreshed JWT re-asserts the home,
  so the stored value self-heals.

## Deliverables

### Nearest-instance selection (first contact only)

- On first launch with no stored session and no stored home: fetch the directory
  from the configured URL, then run a **latency probe** — a parallel, timeboxed
  (~1s) call to each instance's existing `/healthz` — and pick the fastest
  responder. Probe results are cached; ties and failures fall back to the
  configured URL. The probe is honest geolocation: it measures the network that
  actually matters instead of guessing from IP or locale.
- The selection happens **before signup**, because signup is what homes the
  account (24). The login/signup screen renders immediately from the configured
  instance's config; the probe completes in the background well before the user
  finishes typing.
- Directory of size one (standalone or single-instance cluster): the probe is
  skipped entirely — zero added latency, zero behavior change.

### Sticky home

- After any authentication, the SDK reads `mih` from the token and pins it: all
  subsequent `moth.auth.v1`, `moth.billing.v1`, and `moth.push.v1` channels target
  the home URL. The pin survives restarts via persistence and is refreshed from
  every new token, so an operator changing an instance's `base_url` (23) heals on
  the next refresh.
- `WRONG_INSTANCE` (24) is handled inside the client transport: re-target to the
  `home_url` in the error metadata, replay the call once, update the pin. The app
  and the developer never observe it — it surfaces only in SDK debug logging.
- Sign-out clears the pin but keeps the probed nearest choice, so the next signup
  from this device still lands nearby.

### Cross-cutting client behavior

- Config/theme/copy revision caching (05/15/16) keys its cache by instance, and
  reads config from the pinned home once signed in — the home is authoritative
  for everything user-visible after auth.
- The pub (`/pub`) and npm (`/npm`) package endpoints, hosted-page URLs shown in
  setup instructions, and OAuth redirect URIs (04) are documented to use the
  **main's** URL — SDK delivery and console configuration are developer-time
  concerns, not user-latency concerns, and a single stable origin keeps provider
  console setup sane.
- `moth doctor` (08) gains cluster probes: directory coherence across instances,
  per-instance reachability and JWKS agreement, and a `WRONG_INSTANCE` round-trip
  check with a fabricated foreign identity.

## Key design points

- **One URL in, cluster out.** The configured URL is only a bootstrap; the
  directory makes every instance equivalent as an entry point. This keeps the
  pubspec/npm story identical to standalone moth and means cluster topology can
  change without an app release.
- **Probe once, pin forever.** Latency measurement happens only in the
  no-session state. After homing, geography is settled — re-probing could only
  discover an instance the user's data is *not* on. This is the SDK-side mirror
  of 24's "sticky by birth" rule.
- **Transparent retry, bounded.** The `WRONG_INSTANCE` replay happens exactly
  once per call; a second miss surfaces as an error. Self-correction must never
  become a retry storm against a misconfigured cluster.
- **No new developer API.** `MothScope`/`useMoth()` shapes are unchanged; the
  only additions are optional debug hooks (current instance, probe results) for
  support tooling. If a developer never learns the cluster exists, the milestone
  succeeded.

## Acceptance criteria

- Flutter + React integration tests against a two-instance dev cluster (the
  Playwright/e2e harness gains a second `moth serve` fixture): fresh install
  probes, signs up on the nearest (fastest-responding) instance, and the JWT's
  `mih` matches it; all post-auth traffic targets it exclusively (asserted by
  per-instance request logs).
- Same account restored on a device configured with the *other* instance's URL:
  first call self-corrects via `WRONG_INSTANCE`, the pin updates, exactly one
  replay happens, the user never sees an error.
- Kill the non-home instance: the signed-in app is fully functional. Kill the
  home instance: the app fails with a clear unavailable state and recovers when
  the instance returns (no silent re-homing).
- Standalone server: no probe request is ever issued, no `mih` handling runs, and
  the existing 05/16/18 SDK test suites pass unchanged.
- Sign out, sign up as a new user: the new account homes at the probed-nearest
  instance, not the previous account's home.

## Out of scope

Re-homing or migrating accounts, background re-probing of latency, per-call
failover between instances, exposing region choice in the SDK's public API
(a `region` override for tests/debugging is internal), and admin/analytics
surfaces (26).
