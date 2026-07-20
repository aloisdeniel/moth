# Milestone 16 — Localized Flutter SDK

## Goal

Make the SDK's screens speak the user's language. The device locale drives what moth
returns and what the widgets render: the SDK sends its language as a header, consumes the
server-negotiated, project-customized copy from milestone 15, and falls back to bundled
translations when offline or before the config loads — so `MothLoginScreen` (sign-in and
sign-up), `MothPaywallScreen`, and the hosted-page-equivalent flows appear fully
localized with the project's own wording. This is the on-device half of internationalization;
it consumes the milestone-15 negotiation, copy model, and admin editor.

## Deliverables

### Language on the wire

- The SDK resolves the device locale (Flutter `PlatformDispatcher.locale`, with an app
  override hook `MothConfig(locale: ...)`) and sends it as the `x-moth-language` metadata
  header on every call, via the existing milestone-05 client interceptor (alongside
  `x-moth-key`, `x-moth-platform`, `x-moth-sdk-version`). The server (milestone 15)
  negotiates against the project's available locales and returns the matched copy +
  the negotiated locale.

### Copy consumption

- `GetProjectConfig`/`GetPaywall` now carry localized copy + a `copy_revision`; the SDK
  caches it on device with stale-while-revalidate keyed by `(locale, revision)` — the same
  pattern as the milestone-06 theme cache and milestone-13 paywall cache — so screens show
  the right language instantly on launch and refresh in the background. Changing device
  language refetches; an operator's copy edit lands on next launch without an app release.
- `MothLoginScreen` and its building blocks (`MothEmailForm`, `MothProviderButtons`,
  legal footer), the sign-up mode, `MothPaywallScreen` and its blocks
  (`MothPaywallHeader`, `MothTierCard`, `MothPurchaseButton`), and the SDK's error/toast
  copy consume the resolved copy **exclusively** — no hardcoded user-facing English
  strings remain (audit for them), mirroring how milestone 06 removed hardcoded styles.

### Bundled fallback strings

- The SDK bundles the same curated locale set as the server (Flutter localization
  delegates / an embedded ARB-or-map catalog) so it can render fully localized copy
  **before** the config arrives and **offline** — the server-delivered project copy
  overrides these when present, the bundle is the floor. Locales the SDK doesn't bundle
  fall back to English in the widget layer even when the server has custom copy cached.
- Framework strings the app itself needs (e.g. `MaterialLocalizations`) are wired so a
  developer dropping in `MothApp` gets correct locale plumbing without extra setup, and
  the SDK documents how to add the app's own `localizationsDelegates`.

### Example, tests, docs

- The example app demonstrates switching device language and seeing the login + paywall
  screens re-localize (and the project's custom French/German copy from a seeded instance).
- Dart tests: locale resolution + header attachment; copy cache stale-while-revalidate by
  `(locale, revision)`; widget tests asserting `MothLoginScreen` (sign-in and sign-up) and
  `MothPaywallScreen` render the server copy for a locale and the bundled fallback when the
  config is absent; a "no hardcoded English" guard where practical. GOLDEN tests for the
  login and paywall screens in two languages (English + one other) × light/dark, extending
  the milestone-06/13 golden suites.
- SDK reference docs updated: `MothConfig(locale:)`, how copy resolution + fallback works,
  and how customized copy from the admin flows to the device.

## Key design points

- **Device locale in, negotiated copy out** — the SDK asserts a language; the server
  decides what's available and returns it; the SDK renders it, never inventing copy.
- **Bundled floor, server ceiling** — bundled translations guarantee a localized screen
  with zero network; the project's admin-customized copy refines it when reachable.
- **Same cache discipline as theme/paywall** — one revision-keyed stale-while-revalidate
  cache mechanism, now parameterized by locale, so the three config-delivered concerns
  (theme, paywall, copy) behave identically.
- **No new user-facing English** — every SDK screen and message resolves from the catalog;
  a new screen adds keys to the shared catalog, inheriting localization for free.

## Acceptance criteria

- Setting the device (or `MothConfig`) locale to `fr` shows the login and paywall screens
  in French: the project's custom French copy when the instance has it, the bundled French
  strings otherwise, English for a locale neither side has.
- Airplane mode / first launch before the config loads still renders localized (bundled)
  copy, not English placeholders or empty strings.
- An admin copy edit for `de` appears in the example app on next launch (cache refresh),
  without an app update.
- `MothLoginScreen` sign-in and sign-up and `MothPaywallScreen` have no remaining
  hardcoded English (verified by test/audit) and pass golden tests in English + one other
  language × light/dark.
- The copy cache is keyed by `(locale, revision)` and covered by a stale-while-revalidate
  test.

## Out of scope

Per-user server-stored language preference, machine translation, RTL beyond the bundled
set, ICU plural/gender message complexity, and localizing the admin console UI itself
(operator-facing) — all post-v1.1 if demanded. This milestone closes the Internationalization
phase; it ships coupled to the milestone-15 server support.
