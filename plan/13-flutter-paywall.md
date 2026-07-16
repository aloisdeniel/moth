# Milestone 13 — Flutter Purchasing & Themed Paywall (SDK)

## Goal

Make subscriptions a first-class part of the `moth_auth` SDK: read entitlement state
from `MothScope`, trigger a native store purchase in a few lines, and drop in a
batteries-included **paywall screen** that renders the project's tiers with the project's
branding — themed by the design system (milestone 06) and configured from the admin like
the login screen. This is the on-device half of the monetization phase; it consumes the
milestone-11 engine and the milestone-12 catalog.

## Deliverables

### Entitlement state in the SDK

- `MothScope.of(context)` gains subscription state alongside auth state:
  `customerInfo` (active entitlements with expiry + source, active subscriptions),
  `hasEntitlement('pro')` for gating, and `entitlementsChanged` for non-widget code.
  Free/`none` users get a valid `CustomerInfo` — gating code never special-cases "never
  paid".
- Fetched via `moth.billing.v1.GetCustomerInfo`, cached on device (stale-while-revalidate,
  same pattern as the theme cache in milestone 06) so gating is instant on launch and
  refreshes in the background; refreshed on store-notification-driven pushes at next call
  and on `restorePurchases()`.

### Purchase flow (native, thin, optional-plugin)

- The SDK stays light: native billing lives behind a `MothBillingAdapter` interface
  (`purchase(productId)`, `restore()`, `productsFor(offering)`), so `in_app_purchase`
  (StoreKit / Play Billing) is a dependency of the **app**, not the SDK — the same
  optional-native-adapter pattern used for Google/Apple sign-in in milestone 04/05. The
  example app wires a real adapter.
- Flow: the app (or the paywall) calls `MothScope.of(context).purchase(product)` → the
  adapter runs the native purchase → the SDK hands the resulting signed transaction / 
  purchase token to `SubmitPurchase` → the server validates and re-derives entitlements →
  `MothScope` rebuilds. `restorePurchases()` re-links store purchases to the current user.
- Robust states surfaced as typed results: `purchased`, `pending` (deferred/ask-to-buy),
  `alreadyOwned`, `cancelled`, `error` — mapped from the milestone-11 `ErrorInfo` reasons
  and the adapter's store errors.

### Paywall screen (themed + admin-configurable)

- `MothPaywallScreen` — a batteries-included Material screen rendering the project's default
  **offering**: header (logo + headline + subtitle), a feature/benefit list, one card per
  tier (name, price from the store, trial/intro badge, "most popular" highlight), a primary
  purchase button, restore, and terms/privacy + "manage subscription" links (deep-linking to
  the store's management UI). Shown by the app wherever it gates a feature, or as
  `MothApp(requiresEntitlement: 'pro', paywall: const MothPaywallScreen())`.
- **Consumes `MothTheme` exclusively** (milestone 06): colors, typography, radius, spacing,
  logo, fonts — no hardcoded styling, so the paywall matches the login screen and the brand.
- **Admin-configurable, like the login screen**: a paywall config (headline, subtitle,
  benefit bullets, which offering, highlighted tier, layout variant, legal links) delivered
  in the public project config and cached client-side by revision. Building blocks exported
  (`MothPaywallHeader`, `MothTierCard`, `MothPurchaseButton`) plus a
  `MothPaywallScreen(config:)` / full-custom escape hatch, mirroring the login-screen
  building blocks.

### Admin paywall editor

- A "Paywall" section in the project **Design** tab (milestone 06): edit the paywall copy,
  benefit bullets, offering/tier highlighting, layout variant, and legal links, with a
  **live preview** in the phone frame — the same faithful HTML/CSS replica approach the login
  editor uses, sharing the design tokens. Colors/typography inherit from the project theme;
  the paywall never introduces its own token space.

## Key design points

- **Time-to-first-purchase mirrors time-to-first-login** — from "tier defined in admin" to
  "sandbox purchase on a simulator" should be a short, documented path; the acceptance test
  walks it.
- **Tokens over screens, again** — the paywall maps the same design-system tokens to widgets,
  so it inherits the brand for free and stays consistent with the login flow; a golden-test
  suite (reference themes × light/dark) keeps it honest, like the login screen.
- **The SDK never handles money** — it triggers the native purchase and forwards the receipt;
  validation and entitlement truth stay server-side (milestone 11). The client's entitlement
  cache is a convenience, never the authority.
- **Optional by construction** — a project with no products still runs; `MothPaywallScreen`
  renders a graceful empty/"nothing to purchase" state, and gating on an undefined entitlement
  simply never blocks.

## Acceptance criteria

- Example app: with milestone-11/12 sandbox credentials, a sandbox purchase on an Android
  emulator and an iOS simulator completes, validates server-side, and flips
  `hasEntitlement('pro')` to true; a hot restart keeps the entitlement (cached +
  revalidated); `restorePurchases()` re-links on a fresh install.
- `MothScope.of(context)` entitlement transitions (none → active → expired) are covered by
  widget tests with an in-process fake billing server.
- Changing the paywall copy + highlighted tier in the admin reflects in the example app on
  next launch without an app update, and in the admin live preview immediately.
- `MothPaywallScreen` golden tests across the reference themes × light/dark.
- Gating a screen behind `requiresEntitlement` shows the paywall to a free user and the
  content to an entitled user.

## Out of scope

Revenue analytics (14). Web/desktop purchase flows, promotional-offer redemption UI,
paywall A/B experiments, in-app upgrade/downgrade/proration UX beyond what the stores handle
natively, and consumables/one-time products — post-v1.1.
