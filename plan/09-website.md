# Milestone 09 — Public Website (Landing Page + Docs)

## Goal

A fast static website that sells moth in one scroll and documents it completely: what it is, why self-host auth this way, how to go from zero to a logged-in Flutter app. It is the project's front door — separate from any moth instance, deployable to static hosting, and the single source of truth for documentation.

## Deliverables

### Site framework

- `website/` in the repo: **Astro** with **Starlight** for the docs section — static output, excellent docs ergonomics (sidebar, search, dark mode) without inventing a docs system; fits the existing Node toolchain from the admin SPA.
- Deployed to **GitHub Pages** via a GitHub Actions workflow (build on push to `main`, versioned deploy on release tags); custom domain ready (CNAME configurable, exact domain TBD).
- Brand foundation: visuals are based on [`DESIGN.md`](../DESIGN.md) at the repo root — its near-monochrome palette, Satoshi/Cascadia Code typography, type scale, and per-surface notes for the website are the source of truth, materialized here as a design token file shared with the admin SPA and default SDK theme so product and site feel related. The moth logo/wordmark is designed in this milestone and folded back into `DESIGN.md`.
- No trackers, no external requests (fonts self-hosted) — the site practices the privacy stance the product claims. Optional privacy-respecting page counts (e.g. self-hosted Plausible) documented but off by default.

### Landing page

One scroll, in order:

1. **Hero** — one-sentence pitch ("Authentication for **all** your mobile apps. One small binary."), subline making the portfolio promise explicit ("Run one server. Every app you ship gets its own users, keys, and branded login — just add a project."), install one-liner (`brew install moth` / `curl`), and an animated terminal (asciinema-style, prerendered) showing `moth serve` → project created → Flutter login.
2. **How it works** — three-step diagram: run the binary → create a project in the admin → add `moth_auth` to pubspec and wrap your app. Real code snippets, copy buttons.
3. **Feature grid** — leading tile: **one server, all your apps** (admin projects list screenshot showing several apps side by side, each with its own branding); per-app isolated users; email/password + Google + Apple; admin console; design system (themed login screenshots, light/dark); analytics (dashboard screenshot); Flutter SDK; admin CLI with one-command provider setup; server API + JWKS for your backend.
4. **Why moth** — honest comparison table vs Firebase Auth / Auth0 / Supabase Auth on the axes moth wins: self-hosted, single binary, your data, unlimited apps on one instance (vs per-project/per-tenant setup and per-MAU pricing elsewhere), per-project keys; candid about what it doesn't do (managed scale, MFA in v1). A cost sidebar makes the portfolio math concrete: five apps × N users on one $5 VPS vs the same on managed pricing.
5. **Install & quick start** — platform tabs (brew, curl+binary, Docker, systemd), linking into the docs quick start.
6. **Footer** — GitHub, license, docs, changelog.

Screenshots are generated from a seeded demo instance by a Playwright script (reusing the milestone-03 test infra) so they stay current instead of rotting.

### Documentation (single-sourced)

- All docs authored as markdown in `website/docs/` and **single-sourced two ways**: rendered by Starlight for the public site, and embedded into the binary at build time to serve version-matched docs at `/docs` (finalized in milestone 10). One content tree, two outputs — no drift.
- Structure: Quick start (the 10-minute path) · Installation & deployment (systemd, Docker, reverse proxy, ACME) · Guides (provider setup for Google/Apple, theming, analytics, backups, migration import/export) · Flutter SDK reference · CLI reference (generated, milestone 08) · Agents & automation (`moth skill export`, `--json` contracts, driving moth from a coding agent) · API reference (generated from protos) · Security & threat model · Changelog.
- Generated references (CLI, proto API) are produced by CI into the content tree, never hand-edited.
- Docs pages carry a version banner; versioned docs snapshots start at v1.0 (post-v1 concern otherwise).

### Quality gates

- Link checking (lychee) and docs code-snippet compilation checks in CI — Dart snippets in docs are extracted and analyzed against the current SDK so examples can't silently break.
- Lighthouse budget enforced in CI: performance/accessibility/SEO ≥ 95 on landing and a docs page.
- OpenGraph/social cards, sitemap, favicon set.

## Key design points

- **Docs are part of the product loop** — the milestone-03 setup-instructions page links to these docs; the docs quick start must stay byte-compatible with what that page renders. One review checklist ties them together.
- **Static or bust** — no backend, no signups, no cookies; the site must survive on any static host and cost nothing to run.
- **Honesty as positioning** — the comparison section states limitations plainly; for a security product, credibility is the growth channel.
- **`DESIGN.md` is the visual contract** — landing and docs (via Starlight theming) stay inside its tokens and voice rules; deviations are made by amending `DESIGN.md` first, not ad hoc in the site.

## Acceptance criteria

- `npm run build` in `website/` yields a fully static site; deployed automatically to GitHub Pages from `main` with the custom-domain configuration in place.
- A newcomer following only the website (hero → install → quick start) reaches a logged-in example app without touching any other resource — dry-run tested by someone who didn't write the docs.
- The same markdown tree renders in Starlight and embeds cleanly for the binary's `/docs` (pipeline proven with a subset; completed in 10).
- Screenshot generation script runs in CI and produces current images from a seeded instance.
- Lighthouse ≥ 95 across the board; zero broken links; all Dart snippets compile.

## Out of scope

Blog (add post-v1 if there's something to say), hosted demo instance (parking lot — needs abuse controls first), docs i18n, versioned docs archive before v1.0.
