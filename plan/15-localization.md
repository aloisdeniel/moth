# Milestone 15 — Localization & Customizable Copy (server + admin)

## Goal

Make every string an end user sees adapt to their language, and let the operator
override any of it per project. The app (and any browser) sends its language as an
HTTP header; moth negotiates the best available locale and returns the adapted copy —
for the SDK screens (via the public config), the hosted pages (verify/reset/OAuth/
paywall-fallback), and the emails. On top of the bundled translations, an operator can
customize the copy for a project's sign-in, sign-up, and paywall flows, edited and
previewed per language in the admin. This milestone delivers the server-side negotiation
+ copy model + admin editor; the Flutter SDK consuming it is milestone 16.

The model mirrors the design system (milestone 06/13): a small, curated, versioned set
of tokens with sensible bundled defaults that always look right, project overrides on
top, and a live admin preview — but keyed by **screen × locale** instead of by color.

## Locale negotiation

- Language arrives as an HTTP header: standard `Accept-Language` for browsers/hosted
  pages, and an explicit `x-moth-language` metadata header the SDK sends (milestone 16),
  which wins when present. Parse the quality-ordered list, match against the project's
  **available locales** (its default plus any it has customized/translated), and fall
  back deterministically: exact tag → base language (`fr-CA` → `fr`) → project default
  locale → bundled English. The negotiated locale is echoed back so the client can cache
  correctly (an `x-moth-locale` response header / a field in the config).
- Curated locale set, not arbitrary: moth bundles translations for a handful of common
  languages (English default, plus e.g. French, German, Spanish, Portuguese, Italian,
  Japanese — final set TBD), keeping the binary self-contained and the copy space
  reviewable. A project can override any bundled locale's strings and can add copy for a
  locale moth doesn't bundle (the override becomes the full source for that locale).

## Copy model

- A **message catalog**: a curated, versioned set of keys per surface —
  `sign_in.*`, `sign_up.*`, `password_reset.*`, `verify_email.*`, `paywall.*`, and the
  hosted-page/email equivalents (title, field labels, button labels, helper text, error
  copy, legal/footer text). Small and closed, like the theme token set — every key has a
  bundled default in every bundled locale, so a project that customizes nothing is still
  fully localized.
- Per-project **copy overrides**: a JSON structure keyed by locale then key, stored with
  a `revision_id` for client caching (exactly like the theme/paywall config), merged
  bundled-default → project-default-locale-override → project-locale-override at read
  time. Interpolation placeholders (app name, expiry duration, etc.) are part of the key
  contract and validated. Migration adds the storage (a `copy` config + revisions table,
  following how `theme`/`paywalls` are stored).
- Deliberately constrained: a fixed key set (no free-form key creation), length caps,
  required-placeholder validation, and a "reset to default" per key/locale — so any
  combination still renders a sensible screen, the same promise the design system makes.

## Deliverables

### Server

- Locale negotiation middleware/helper usable from both the gRPC config path and the
  plain-HTTP hosted pages; the negotiated locale threaded through to copy resolution.
- `GetProjectConfig` (milestone 05/06) and `GetPaywall` (milestone 13) return the copy
  for the negotiated locale alongside the theme, revision-cached (a `known_copy_revision`
  omission like the theme, so the SDK skips re-parsing unchanged copy).
- Hosted pages (verify/reset/confirm-email/OAuth success/paywall fallback) render in the
  negotiated locale from the same catalog; `lang`/`dir` attributes set; RTL-safe layout
  where a bundled RTL locale exists (or documented as out of scope for the initial set).
- Email templates (verification, password reset, email change) localized from the same
  catalog, negotiated from the requesting context's locale where known, else the
  project default.
- Admin RPCs (`moth.admin.v1`, extending the design/paywall services): get/update the
  per-locale copy overrides (new revision), list/restore revisions, reset a key or a
  whole locale to the bundled default, and list the project's available + bundled
  locales. Audit-logged like the theme edits.

### Admin — Design tab restructured

The Design tab becomes a set of **sub-tabs**, each a screen the SDK renders, so an
operator can edit and preview all of them for a chosen language:

- **Theme** (first) — the existing color/typography/spacing/logo editor (milestone 06),
  unchanged in function, now the first sub-tab.
- **Sign in**, **Sign up**, **Paywall** — one sub-tab each: a **language selector**
  (the project's available locales + bundled locales, with an "add language" action) and
  the per-key copy editor for that screen and locale, with reset-to-default per key and a
  bundled-value hint.
- A **live preview** phone frame on every sub-tab renders that screen for the selected
  language and light/dark, using the project theme + the (unsaved) copy — reusing the
  milestone-06 login-preview and milestone-13 paywall-preview replicas, now driven by the
  copy catalog and split into the sign-in vs sign-up modes. Switching language re-renders
  the preview so the operator sees every screen in every language before shipping.
- Save creates a new copy revision (kept last N, restorable), consistent with the theme
  editor.

## Key design points

- **Negotiate, never trust a body** — the language comes from a header and is negotiated
  against what the project actually has; the client never dictates raw copy.
- **Bundled defaults always work** — a project (or a locale) with zero customization is
  fully localized from the bundled catalog; overrides are additive, exactly like the
  design system's token defaults.
- **One catalog, every surface** — SDK screens, hosted pages, and emails all resolve from
  the same key set, so a translation or an override lands everywhere at once.
- **Tokens/keys over screens** — a fixed, closed key set with validation keeps the copy
  space small and always-renderable; adding a new SDK screen adds keys, not a new system.

## Acceptance criteria

- A request with `Accept-Language: fr` (and the SDK's `x-moth-language: de`) gets the
  hosted reset page and `GetProjectConfig` copy in that language; an unknown/unsupported
  tag falls back through base-language → project default → English deterministically
  (table-driven negotiation tests, including quality values and `fr-CA` → `fr`).
- Editing the sign-up copy for `fr` in the admin reflects immediately in that sub-tab's
  live preview and, after save, in `GetProjectConfig` for a `fr` request — while `en`
  and unedited keys keep the bundled default.
- The Design tab shows Theme / Sign in / Sign up / Paywall sub-tabs; each previews its
  screen for a selected language in light and dark.
- `GetProjectConfig`/`GetPaywall` omit the copy body when `known_copy_revision` matches.
- A localized email (password reset) renders in the negotiated locale; a project with no
  overrides still sends fully-localized bundled copy.
- Copy validation rejects a missing required placeholder / over-length string with a
  clear message.

## Out of scope

The Flutter SDK consuming the localized copy, device-locale detection, and the SDK's own
bundled fallback strings — all milestone 16. Machine translation, translator/reviewer
workflows, per-user language preference storage, pluralization/ICU-message complexity
beyond simple placeholders, and RTL beyond the bundled set — post-milestone-16 if demanded.
