
# moth — Design System

This file is the single source of truth for the visual design of every moth surface: the public website and docs (milestone 09), the embedded admin web app (milestone 03), the hosted auth pages (`/p/{slug}/verify`, `/reset`, `/confirm-email`) and transactional emails, and the default chrome of the `moth_auth` Flutter SDK (milestone 05). AI tools and humans alike should read this file before building or modifying any UI.

**Scope note:** milestone 06 adds a per-project theme editor so developers can brand their app's login screens (colors, font, spacing, logo). That per-project theme is the *developer's* design system and overrides ours inside their app. Everything below governs moth's **own** surfaces and the SDK's **default, unthemed** appearance — the canvas the project theme paints over.

## Identity

moth is a developer tool that guards sign-ins for a whole portfolio of apps. The design should feel like precision instrumentation with the calm of a well-run vault: digital, simple, elegant, clean, trustworthy. It is deliberately **not colourful** — a near-monochrome system in the spirit of Vercel/Geist, where hierarchy comes from typography, spacing, and subtle borders rather than colour. Colour is reserved exclusively for state (status, errors, focus) and never for decoration. An auth product earns trust by looking exact, not exciting.

### Principles

1. **Monochrome first.** Black, white, and a disciplined gray scale carry 95% of the UI. If a design needs colour to work, redesign it.
2. **Borders over shadows.** Structure is expressed with 1px hairline borders and background steps, not drop shadows. Shadows exist only at the two highest elevation levels.
3. **Typography is the interface.** Weight and size changes replace boxes, dividers, and colour blocking wherever possible.
4. **Dense but breathable.** Developer-tool density: compact controls, generous whitespace between groups. Never cramped, never airy for its own sake.
5. **Dark and light are equals.** Every token has a value in both schemes; neither is an afterthought. Surfaces invert, semantics stay put.
6. **Motion is functional.** Transitions confirm actions (120–200 ms ease-out). No bounces, no delight animations.
7. **Boring where it counts.** Auth flows (hosted pages, login screens) are the most conservative surfaces of all: one column, one action, zero surprises.

## Colors

Tokens are named by role. Hex values are given for **light** and **dark** schemes.

### Neutrals (the core palette)

| Token | Light | Dark | Usage |
|---|---|---|---|
| `background` | `#FFFFFF` | `#0A0A0A` | Page/app background |
| `background-subtle` | `#FAFAFA` | `#111111` | Alternating sections, sidebars, wells |
| `surface` | `#FFFFFF` | `#161616` | Cards, panels, popovers |
| `surface-hover` | `#F5F5F5` | `#1F1F1F` | Hover state of interactive surfaces |
| `border` | `#EBEBEB` | `#262626` | Default hairline borders, dividers |
| `border-strong` | `#D4D4D4` | `#3D3D3D` | Input borders, emphasized dividers |
| `text-primary` | `#0A0A0A` | `#EDEDED` | Headings, body, labels |
| `text-secondary` | `#666666` | `#9C9C9C` | Descriptions, captions, placeholders |
| `text-tertiary` | `#999999` | `#6E6E6E` | Disabled text, faint metadata |
| `inverse` | `#0A0A0A` | `#EDEDED` | Fill of primary buttons |
| `inverse-text` | `#FFFFFF` | `#0A0A0A` | Text on `inverse` |

### Functional colors (state only — never decorative)

Muted, slightly desaturated so they sit quietly in a monochrome UI.

| Token | Light | Dark | Usage |
|---|---|---|---|
| `accent` | `#0068D6` | `#52A8FF` | Links, focus rings, active/selected indicators |
| `success` | `#0F7B3E` | `#3ECF6E` | Verified, enabled, healthy |
| `warning` | `#A35200` | `#F5A623` | Pending states, cautions |
| `danger` | `#CC0000` | `#FF6166` | Errors, destructive actions, revocations |
| `info` | `#666666` | `#9C9C9C` | Neutral/idle status (reuses `text-secondary`) |

Status mapping (admin app + SDK): Email verified / user enabled / key active / SMTP configured → `success` · Verification pending / email-change revert window open / console mailer in production → `warning` · User disabled / session revoked / signing key retired / error → `danger` · Never signed in, empty states → `info` · Selected project, current session → `accent`.

Rules:
- Functional colors appear only as: status dots, thin indicator bars, focus rings, link text, error text, and destructive button fills. Never as backgrounds of large areas, decorative gradients, or illustration fills.
- Tinted backgrounds for banners/chips use the functional color at 10% opacity over `surface`, with full-strength text/icon.

## Typography

| Role | Family | Fallback stack |
|---|---|---|
| Text (website + admin SPA) | **Satoshi** | `Satoshi, -apple-system, 'Segoe UI', Roboto, sans-serif` |
| Mono (website + admin SPA) | **Cascadia Code** | `'Cascadia Code', 'SF Mono', Menlo, Consolas, monospace` |
| Hosted pages + emails | **system stack** | `-apple-system, 'Segoe UI', Roboto, sans-serif` — these render inside the binary and email clients; they ship no font assets and must look native everywhere. |
| SDK (all roles) | **platform default** | Flutter default (Roboto/SF) — the SDK ships no font assets to stay dependency-free and lightweight; a project theme (milestone 06) may supply its own font. Mono in the SDK uses `FontFeature.tabularFigures()` on the default font, not a mono family. |

**Satoshi** (display/text) is loaded from Fontshare's CDN — its ITF license
permits CDN hosting but not redistributing the font files, so it is not
bundled. The `<link>` is non-render-blocking (`media="print"` + `onload`
swap) with the system-font fallback stack above, so both the website and the
admin console stay usable if the CDN is unreachable. **Cascadia Code** (mono)
is self-hosted woff2 under `src/fonts/` (SIL OFL 1.1, freely
redistributable). No `google_fonts` package; no other third-party CDNs. The
**hosted pages, emails, and SDK** ship or reference no external fonts at all
(system/platform stacks and the project's own OFL theme font), so the
binary's end-user surfaces make zero external font requests.

### Scale

A single scale shared by all surfaces. Sizes in px (web) / logical px (Flutter).

| Token | Size / Line height | Weight | Usage |
|---|---|---|---|
| `display` | 56 / 1.1 | 700 | Landing hero only |
| `title-1` | 32 / 1.2 | 700 | Page titles |
| `title-2` | 24 / 1.3 | 700 | Section headings |
| `title-3` | 18 / 1.4 | 500 | Card titles, dialog titles |
| `body` | 15 / 1.6 | 400 | Default body text |
| `body-strong` | 15 / 1.6 | 500 | Emphasis, labels |
| `caption` | 13 / 1.5 | 400 | Secondary info, help text |
| `micro` | 11 / 1.4 | 500 | Overlines, badges — uppercase, letter-spacing 0.08em |
| `code` | 13 / 1.6 | 400 | Mono: keys, kids, slugs, JWTs, timestamps, code |

Rules:
- Headings use tight letter-spacing (−0.01em to −0.02em at ≥24px).
- Only three weights exist: 400 (regular), 500 (medium), 700 (bold). Satoshi ships no 600; never use weights above 700 or below 400.
- Numbers that update live (user counts, analytics charts, rate meters) always use `code` or tabular figures — they must not jitter horizontally.
- API keys (`pk_…`, `sk_…`), key IDs, project slugs, emails-as-data, and tokens always render in `code`, single-line, with an adjacent copy affordance where relevant. Secrets shown once (a fresh `sk_`) get `body`-sized mono and a copy button — never truncate them.

## Spacing

Base unit **4px**. Allowed steps: `4, 8, 12, 16, 24, 32, 48, 64, 96`. Nothing in between.

- Control padding: 8×12 (compact), 10×16 (default).
- Card/panel padding: 16 (compact), 24 (default).
- Gap between related items: 8–12. Between groups: 24–32. Between page sections: 64–96 (landing), 32–48 (app).
- Landing page content max-width: **1080px**, gutter 24px. Hosted auth pages max-width: **380px**, single column.

## Radius

| Token | Value | Usage |
|---|---|---|
| `radius-sm` | 6px | Buttons, inputs, chips, badges |
| `radius-md` | 8px | Cards, panels, popovers |
| `radius-lg` | 12px | Modals, hosted-page card, SDK login card |
| `radius-full` | 9999px | Status dots, avatars, pills |

## Elevation

| Level | Recipe | Usage |
|---|---|---|
| 0 | none | Default. Structure comes from `border` |
| 1 | 1px `border` + bg step (`surface` on `background-subtle`) | Cards, panels, hosted-page card |
| 2 | border + `0 4px 12px rgba(0,0,0,0.08)` (dark: `0.32`) | Popovers, dropdowns |
| 3 | border + `0 8px 30px rgba(0,0,0,0.12)` (dark: `0.48`) | Modals, command palettes |

## Components

### Buttons

Height 40px (default) / 32px (compact), `radius-sm`, `body-strong` label, padding 0 16px.

| Variant | Fill | Text | Border | Hover |
|---|---|---|---|---|
| Primary | `inverse` | `inverse-text` | none | 90% opacity fill |
| Secondary | transparent | `text-primary` | 1px `border-strong` | bg `surface-hover` |
| Ghost | transparent | `text-secondary` | none | bg `surface-hover`, text `text-primary` |
| Danger | `danger` | white | none | 90% opacity fill |

Focus: 2px `accent` ring, 2px offset. Disabled: 40% opacity, no pointer events. Never use `accent` as a button fill. Destructive admin actions (delete project, delete user, revoke sessions) use Danger and always confirm in a modal that names the target.

### Inputs

Height 40px, `radius-sm`, bg `background`, 1px `border-strong`, text `body`, placeholder `text-tertiary`. Focus: border becomes `accent` + 1px ring. Error: border `danger`, help text `danger` at `caption`. Labels: `caption` weight 500, `text-secondary`, 6px above the field. Password fields get a show/hide ghost toggle; email fields never autocapitalize.

### Cards / Panels

Bg `surface`, 1px `border`, `radius-md`, elevation 0–1. Title: `title-3`. Hoverable cards (project list): border → `border-strong` on hover, no lift.

### Tabs (admin app)

Underline style: transparent bg, `caption`-weight-500 labels; inactive `text-secondary`, active `text-primary` with 2px `text-primary` underline (not `accent`). Container has a 1px `border` bottom. Project sections (Users, Keys, Settings, Analytics, Setup) are tabs, not sidebars.

### Status indicator

8px `radius-full` dot in the functional color + `caption` label in `text-secondary`. Dots are static — an auth console must read as settled, not busy. (No ambient animation anywhere in the system.)

### Badges / Chips

`micro` uppercase text, 4×8 padding, `radius-sm`, bg = functional color at 10% (or `surface-hover` for neutral), text at full functional strength. Used for: `VERIFIED`, `PENDING`, `DISABLED`, provider tags (`PASSWORD`, `GOOGLE`, `APPLE`), key status (`ACTIVE`, `RETIRED`).

### Key / secret display

A `code` value in an elevation-0 well (`background-subtle`, 1px `border`, `radius-sm`, 8×12 padding) with a trailing copy button. Publishable keys show in full. Secret keys show as `sk_••••…` after creation — the plaintext exists on screen exactly once, in the creation dialog, with a copy button and the sentence "You won't see this key again."

### Code blocks (landing, docs, setup instructions)

`code` type on `background-subtle` (dark: `#111111`), 1px `border`, `radius-md`, 16px padding. Inline code: 2×6 padding, `radius-sm`, same bg. Setup snippets (pubspec entry, JWKS URL, grpcurl examples) always carry a copy button.

### Hosted auth pages

One elevation-1 card (`radius-lg`, max-width 380px) centered on `background-subtle`: `title-3` heading, `body` explanation, at most one form and one primary button. Footer: project name in `caption` `text-tertiary`. Errors render as `danger` text inside the card, never as a bare error page. These pages are served from the binary with inline CSS — no external assets, no JS.

### Transactional emails

Same skeleton as hosted pages, tables-free minimal HTML: heading, 1–3 short paragraphs, one dark button (`inverse` fill), the raw link in `caption` mono below it, project name as sign-off. Plain-text part always included and readable on its own.

## Iconography

Outlined style only, 1.5px stroke feel, 16/20/24px sizes, colored `text-secondary` (or `text-primary` when active). In Flutter use the default `Icons` set (Material Symbols, outlined variants where they exist). No filled or two-tone icons, no emoji in the UI. Provider logos (Google, Apple) follow the providers' own brand rules inside sign-in buttons and appear monochrome everywhere else.

## Motion

- Durations: 120 ms (hover/press), 200 ms (panel/tab transitions), 300 ms (modals).
- Easing: `cubic-bezier(0.25, 0.1, 0.25, 1)` (standard ease) out; ease-in for exits.
- Respect `prefers-reduced-motion`: disable non-essential transitions.

## Voice & content

- Sentence case everywhere — buttons, titles, labels ("Create project", not "Create Project").
- Terse and technical; no exclamation marks, no marketing superlatives inside the product. The landing page may be aspirational but stays factual.
- Security-relevant copy is explicit and calm: say exactly what happened and what to do ("This link has expired. Request a new one from the app."), never blame the user, never say "oops".
- Keys, slugs, kids, timestamps, and counts render in mono.
- The product name is always lowercase **moth**, even at sentence start.

## Per-surface notes

### Public website + docs (milestone 09)
Astro/Starlight on GitHub Pages, Satoshi from Fontshare's CDN (non-blocking, system fallback) + self-hosted Cascadia Code (woff2), minimal JS. Dark-scheme aware via `prefers-color-scheme`. Hero (`display` type) → one-line value prop ("One binary. Every app you ship.") → primary button "Get started" into the docs → feature grid of elevation-1 cards → mono code snippet showing `moth serve` + pubspec setup → footer. Docs inherit the same tokens through Starlight theming.

### Admin web app (`/admin`, milestone 03)
React + Vite + TypeScript SPA embedded via `go:embed`. Tokens live in a single CSS-variables file (both schemes, follow system). Cascadia Code bundled as an asset; Satoshi loaded from Fontshare's CDN (non-blocking, with the system-font fallback stack). Layout: top bar (instance) → project switcher → tabbed project view. Tables are the workhorse: `caption` headers, `body` cells, mono for emails-as-data, keys, and dates; row hover `surface-hover`; no zebra stripes.

### Hosted auth pages (`/p/{slug}/…`, embedded)
Server-rendered from `internal/server/web/page.html.tmpl` with inline CSS and the system font stack — they must work with zero asset requests inside any webview or browser an email client opens. Keep them on the shared tokens but never add webfonts, JS, or images. When milestone 06 lands, these pages may absorb the project's theme colors; until then they stay neutral.

### Flutter SDK (`moth_auth`, milestone 05)
Default (unthemed) login UI uses these tokens on the **platform default font** — no font assets, no new dependencies. Tokens live in a single private `theme.dart` exposing light/dark `ThemeData` keyed on platform brightness. The login screen is one `radius-lg` card: app logo slot, email + password inputs, primary button, provider buttons below a hairline divider. Everything here is a *default* that the per-project theme (milestone 06) overrides; functional colors for error states remain fixed even under project themes.
