# Milestone 22 — Guided Project Creation (adaptive wizard)

## Goal

Turn project creation from a name-and-slug dialog into a **guided, adaptive
flow** that asks what the app actually needs and configures — or honestly
defers — everything it heard. Today the dialog mints keys and drops the
operator into seven tabs (Providers, Design, Monetization, Paywall, Copy,
Settings, Setup) to discover on their own; the wizard walks the same ground
in order, **branching on the answers**: a web-only free app never sees store
credentials, a Flutter-only app never sees VAPID keys, and a "just email
sign-in" project is created in two steps flat. What can't be finished
in-flow (an Apple key that doesn't exist yet, a Play service account pending
review) becomes a **derived checklist** on the project overview instead of a
silent gap.

The organizing idea: moth already knows, per feature, what "configured"
means (`moth doctor` proves it). The wizard is that knowledge run forward —
ask, configure, defer — and the checklist is the same knowledge run
backward over the live config. Nothing is stored about wizard progress;
completeness is always **derived from actual state**, so the checklist is
also right for projects created before this milestone or via the CLI.

## Deliverables

### The wizard (admin SPA, replaces the create dialog)

A full-screen stepped flow; every step after the first is skippable
("decide later" is always an answer) and the step list itself adapts to
earlier answers. Steps, in order:

1. **Basics** — name, slug (derived, editable), and **platforms**: iOS /
   Android / Web (multi-select). Platforms drive every later branch and the
   setup tab forever after.
2. **Sign-in** — email/password toggles (sign-up open, verification
   required, minimum length) prefilled with the defaults; Google and Apple
   toggles. Enabling a provider inlines the milestone-04 credential fields
   with the same validation the Providers tab runs — or defers with a
   pointer to `moth setup google|apple` (08), which lands on the checklist.
   Web-only projects don't see the native-app fields (reversed-client-id,
   Android SHA fingerprints); native-only projects don't see web origins.
3. **Monetization** *(optional)* — "does this app sell subscriptions?" No →
   step disappears entirely (the built-in `none` tier means nothing else is
   required). Yes → define the first entitlement + tiers inline (the
   milestone-11 catalog editor's core fields), scoped to stores the
   platforms imply: App Store / Play for iOS / Android, Stripe for Web.
   Store credentials and catalog sync (12) are explicitly **deferred to the
   checklist** — collecting a `.p8` mid-wizard helps nobody.
4. **Push notifications** *(optional)* — "will your backend send pushes?"
   Yes on Web → paste the VAPID public key (or defer, with the
   `npx web-push generate-vapid-keys` snippet); yes on Android → the
   Firebase caveat surfaces here instead of in a README later; iOS needs
   nothing but the capability note. Enables the milestone-20 settings.
5. **Branding** *(optional)* — logo upload, brand color, light/dark
   preference seeded into the milestone-06 theme; the live phone-frame
   preview from the Design tab renders beside it. Languages beyond English
   (15) are offered as a checkbox list seeding the copy locales.
6. **Review & create** — everything chosen on one screen, then a single
   `CreateProject` + the batched config writes. The project is created
   **only here**: abandoning the wizard mid-way creates nothing, and every
   write after `CreateProject` reuses the existing per-domain admin RPCs
   (providers, catalog, push settings, theme, copy) — the wizard is
   composition, not a new write path. The keys screen (pk_/sk_, shown once)
   stays the finale, followed by the tailored setup tab.

### The setup profile (`moth.projectconfig.v1.StoredProfile`)

The one new piece of stored state: the answers themselves — platforms,
sells subscriptions, sends pushes — as a small config blob (stored proto,
like theme/paywall/push). It exists so the product can keep adapting after
creation:

- The **setup tab** (03/18/21) filters to the chosen platforms and
  features: a web-only project shows npm + React snippets only; no
  monetization → section 6 disappears; no push → section 7 disappears.
- The **checklist** (below) knows which features were *intended*, so a
  deferred Apple key is "to do" on a project that chose Apple sign-in and
  simply absent on one that didn't.
- Editable later from Settings (a project can grow a web platform); absent
  on old projects, where surfaces behave exactly as today.

### The setup checklist (derived, on the overview)

- A `GetProjectSetupStatus` admin RPC computing, from live config + the
  profile, the outstanding items: provider credentials missing, billing
  credentials / catalog sync pending (12's sync state), VAPID key missing,
  theme still default, SMTP unconfigured at the instance level. Each item
  links to the tab (or CLI command) that finishes it; each disappears the
  moment the underlying config exists — the same probes `moth doctor` (08)
  runs, factored to be shared, not duplicated.
- Rendered as a dismissable card on the project overview until empty.
  "Dismiss" hides the card (a profile flag), never fakes completeness.

### CLI parity

- `moth project create` keeps its exact current behavior (scripts depend
  on it). A new **`moth project init`** runs the same ask-configure-defer
  flow as terminal prompts — platforms, sign-in, monetization, push —
  emitting the same profile and finishing with the checklist as text plus
  a ready-to-commit `moth project apply` spec (08) of what it just built.
  Non-interactive terminals get a plain error pointing at `create`/`apply`.

## Key design points

- **Branch on answers, never on assumptions.** Every conditional surface
  keys off an explicit answer stored in the profile — not off inference
  from config state, which can't distinguish "doesn't want Apple sign-in"
  from "hasn't configured it yet". That distinction is the whole
  difference between a checklist that nags and one that helps.
- **Defer is a first-class outcome.** Credentials that take a console
  round-trip (Apple keys, Play service accounts, VAPID pairs) are never
  blocking steps; the wizard's job is to leave a truthful trail of what
  remains, where the existing tabs and `moth setup` commands finish it.
- **No draft projects.** Creation is atomic at the review step; the wizard
  holds state client-side only. No half-configured tenants, no cleanup
  sweep, nothing new to migrate.
- **One write path.** The wizard composes the same admin RPCs the tabs
  call; a project built by the wizard, the tabs, or `project apply` is
  indistinguishable. The profile is data *about* intent, never a second
  source of config truth.
- **Derived completeness.** The checklist recomputes from live state on
  every view — finishing a step via the CLI, the tabs, or a teammate's
  session updates it with no bookkeeping to go stale.

## Acceptance criteria

- A web-only, free, no-push project is created in two steps (basics +
  sign-in); the wizard never rendered store, push, or native-provider
  fields, and its setup tab shows only the npm/React path.
- A Flutter app with Apple sign-in + subscriptions + push, with all
  credentials deferred: creation succeeds, and the overview checklist
  lists exactly the Apple credential, billing credentials + catalog sync,
  and the Firebase note — each linking to the right tab or CLI command;
  completing one (e.g. via `moth setup apple`) removes it on next view.
- Abandoning the wizard at any step leaves no project behind
  (`ListProjects` unchanged, asserted in an e2e test).
- Entitlement + tiers defined in the wizard land identically to ones
  created in the Monetization tab (same store rows, same catalog-sync
  eligibility); the seeded theme renders in the Design tab preview.
- A pre-milestone project (no profile) shows today's full setup tab and no
  checklist card; setting a profile from Settings adapts both.
- `moth project init` on a TTY produces a project whose profile,
  checklist, and `project apply` spec match the same answers given in the
  SPA wizard; piped stdin fails with the pointer to `create`/`apply`.
- Playwright covers the two flows above end-to-end against the real
  binary, including the keys-shown-once finale.

## Out of scope

Instance-level onboarding (first-run SMTP/base-URL setup is milestone 03's
flow), collecting store credentials or running catalog sync inside the
wizard (12 owns that lifecycle), in-wizard email/SMTP testing, project
templates or duplication, multi-admin collaborative drafts, and any change
to `moth project create` / `apply` semantics beyond the additive profile.
