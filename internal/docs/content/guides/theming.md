# Theming the login screen

Each project carries a small design system — colors, typography, spacing,
corner radius, logo, legal links — defined once in the admin console and
rendered everywhere its end users look: the SDK's `MothLoginScreen`, the
hosted verify/reset/confirm-email pages, and the transactional emails.

The promise is "looks like your brand with zero design work", so the
theme space is deliberately constrained: a curated token set where every
combination produces an acceptable screen, not a free-form CSS editor.

## The tokens

- **Colors** — `primary`/`onPrimary`, `background`/`onBackground`,
  `surface`/`onSurface`, `error`/`onError`, as a light palette plus
  optional dark overrides (anything omitted is derived server-side, so
  clients always receive two complete palettes).
- **Typography** — a font family from the server's curated embedded set,
  plus a global text-size multiplier. (Curated rather than arbitrary
  uploads: keeps mobile rendering predictable and the binary
  self-contained.)
- **Spacing** — the base spacing unit in logical pixels.
- **Shape** — the component corner radius.
- **Logo** — light/dark PNG or SVG, uploaded in the editor. Images are
  re-encoded server-side (decoding and re-encoding strips embedded
  payloads — an SVG with a script inside comes out inert) and served
  from `/assets/{project}/…` with cache headers.
- **Legal links** — optional terms-of-service and privacy-policy URLs,
  rendered in the login footer and on hosted pages, where app review
  expects them.

## The editor

The project's **Design** tab: color pickers, font selector, sliders, logo
upload — next to a live phone-frame preview of the login screen with a
light/dark toggle. The preview is an HTML/CSS replica driven by the same
tokens the SDK consumes, kept honest by a golden-test suite comparing it
against real Flutter renders.

Two guardrails:

- **Contrast validation** — the server enforces WCAG AA (≥ 4.5:1)
  between every color and its `on` counterpart; an illegible palette is
  rejected with a clear message, not saved.
- **Revisions** — every save is a new revision; recent ones are kept for
  one-click undo, and "reset to defaults" is always available.

## How it reaches the app

The theme travels in `GetProjectConfig` (the same publishable-key RPC
the SDK already calls) with a `revision_id`. The SDK caches the theme on
disk keyed by that revision and renders it immediately on next launch —
stale-while-revalidate, so there's no flash of unthemed UI — while the
config call echoes `known_theme_revision` and the server omits the theme
body entirely when nothing changed. Binary assets (logo, font files)
stay plain-HTTP downloads with cache headers; fonts are registered via
`FontLoader` and cached.

Net effect: **change the primary color in the admin and every install
picks it up on next launch, without an app release.** Hosted pages and
emails reflect it immediately. The [React SDK](../../react/) consumes the
same tokens the same way, rendered as CSS custom properties scoped to the
moth screens — one theme drives mobile, web, pages, and emails.

## Escape hatches in the SDK

When the built-in screen isn't enough, the pieces are exposed — see the
[SDK reference](../../sdk/#theming-hooks):

- `MothLoginScreen(theme: …)` / `MothApp(theme: …)` — override the
  server theme with a local `MothTheme`.
- Build your own screen from the themed parts: `MothEmailForm`,
  `MothProviderButtons`, `MothLogo`.
- Error-state colors stay fixed under any theme — legibility of failure
  is not themable.

## Defaults

Projects that never touch the Design tab get moth's neutral default
theme (near-monochrome, platform font) — fine for development, but do
give your users a logo and a primary color before shipping. The
[analytics](../analytics/) won't measure it, but they'll notice.
