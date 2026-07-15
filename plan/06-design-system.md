# Milestone 06 — Design System & Themed Login

## Goal

Each project defines a small design system — colors, typography, spacing, corner radii, logo — in the admin console, and every surface that end users see renders with it: the SDK's `MothLoginScreen`, and the hosted web pages (email verification, password reset, OAuth fallback). One theme, defined once, applied everywhere.

## Deliverables

### Theme model

- `theme` JSON on the project row, versioned schema:
  ```json
  {
    "version": 1,
    "colors": {
      "primary": "#6750A4", "onPrimary": "#FFFFFF",
      "background": "#FFFBFE", "onBackground": "#1C1B1F",
      "surface": "#FFFBFE", "error": "#B3261E"
    },
    "darkColors": { "...": "optional overrides; omitted = derived" },
    "typography": { "fontFamily": "Inter", "scale": 1.0 },
    "spacing": { "unit": 8 },
    "shape": { "cornerRadius": 12 },
    "logo": { "light": "/assets/{project}/logo-light.png", "dark": "..." }
  }
  ```
- Deliberately small: a curated token set that always looks decent, not a free-form CSS editor.
- Logo upload: stored under `data/uploads/`, served at `/assets/{project}/...` with cache headers; PNG/SVG, size-validated, re-encoded server-side (decode + re-encode strips payloads).
- Fonts: curated embedded set (Inter, plus a handful of open-license faces) rather than arbitrary uploads — keeps mobile rendering predictable and the binary self-contained.
- Optional legal links (terms of service, privacy policy URLs) rendered in the login screen footer and on hosted pages — app review expects them near signup.

### Admin theme editor

- New "Design" tab per project: color pickers with contrast-ratio warnings (WCAG AA check on primary/onPrimary etc.), font selector, spacing/radius sliders, logo upload with light/dark preview.
- **Live preview** pane rendering a faithful HTML/CSS replica of the mobile login screen (portrait phone frame, light/dark toggle). Same tokens → close-enough fidelity without embedding Flutter web.
- Save = new theme revision (keep last N for undo); "reset to defaults".

### Theme delivery & consumption

- Theme included in the `GetProjectConfig` response (from 05) as a proto message with a `revision_id`, so the SDK can cache and skip re-parsing when unchanged (`GetProjectConfig` accepts `known_theme_revision` and omits the theme body on match).
- Logo images and font files remain plain-HTTP downloads (`/assets/...`) with cache headers — binary assets don't belong in RPC responses.
- **Flutter SDK**: `MothTheme` maps the theme message into a `ThemeData`-compatible object; `MothLoginScreen` (and its sub-widgets) consume it exclusively — no hardcoded styles remain. Fonts fetched from the server and registered via `FontLoader`, cached on disk. Theme refreshes on app start (stale-while-revalidate: render cached theme immediately).
- Developer escape hatches: `MothLoginScreen(theme: ...)` override, and exposed building blocks (`MothEmailForm`, `MothProviderButtons`, `MothLogo`) so a custom login screen can still use themed parts.
- **Hosted pages** (verify/reset/OAuth pages from 02/04) restyled from the same tokens via a generated CSS custom-properties block.
- **Email templates** pick up logo + primary color.

## Key design points

- **Constrain the theme space** — every combination of the offered tokens must produce an acceptable screen; validation (contrast, logo dimensions) rejects the rest. moth's promise is "looks like your brand with zero design work".
- **Tokens over screens** — the SDK maps tokens to widgets; new SDK screens automatically inherit the system.
- **Preview honesty** — the HTML preview and the Flutter rendering share a golden-test suite: for a fixed set of themes, Flutter golden images are compared against preview screenshots at review time to keep drift visible.

## Acceptance criteria

- Change primary color + logo in admin → hosted reset page reflects it immediately; example app reflects it on next launch without an app update.
- Dark mode: device in dark mode renders `darkColors` (or derived palette) in SDK and hosted pages.
- Contrast validation blocks an illegible palette with a clear message.
- Flutter golden tests for `MothLoginScreen` across 3 reference themes × light/dark.
- Uploaded SVG with embedded script is neutralized (test).

## Out of scope

Arbitrary font uploads, full login-flow layout customization (custom field order, extra fields), per-screen overrides — all post-v1 if demanded.
