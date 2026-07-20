# Milestone 18 — React SDK & npm Serving

## Goal

Bring the milestone-05 developer experience to the web: add one dependency served by
your moth instance, wrap your app in one provider, read auth state from a hook — and,
with milestone 17 in place, gate features on entitlements and sell subscriptions
through a themed paywall backed by Stripe Checkout. This milestone delivers the
`@moth/react` package and the npm-registry endpoint that serves it — the same "no
external publishing, version always matches the server" model as the Flutter SDK,
packaged for the browser. It reuses nearly the whole stack: the milestone-02 auth
protos over the gRPC-Web transport the admin SPA already exercises, the milestone-04
web OAuth flows, the milestone-06 theme, the milestone-15 negotiated copy, and the
milestone-11/17 billing engine.

## Deliverables

### Package serving (`/npm`)

- Implement the minimal npm registry API needed by the npm/pnpm/yarn/bun clients:
  - `GET /npm/@moth/react` — packument: version listing with dist-tags, tarball URLs,
    and integrity hashes.
  - `GET /npm/@moth/react/-/react-{v}.tgz` — package tarball.
- The tarball is **built at moth release time** from `sdk/react/` and embedded in the
  binary; its version tracks the moth version (same discipline as `/pub`). The package
  is **scoped** so a one-line `.npmrc` routes only moth packages to the instance and
  everything else stays on npmjs:
  ```ini
  @moth:registry=https://auth.example.com/npm
  ```
  ```sh
  npm install @moth/react
  ```
- The package is **project-agnostic**: configuration (base URL + publishable key) is
  passed at runtime. The setup-instructions tab (03) gains a React section rendering
  the exact `.npmrc`, install command, and provider snippet with the project's values.

### `@moth/react` package (`sdk/react/`)

Framework-free client core + React layer, structured like the Flutter SDK's
pure-Dart core + widget layer so other frameworks can be added later without
reimplementing auth.

- **Client core** (`MothClient`): ergonomic wrapper around the connect-web client
  generated from `moth/auth/v1` and `moth/billing/v1` — the third codegen consumer of
  the same protos after the Dart SDK and the admin SPA; `make proto` gains an
  `sdk/react/src/gen` output and CI gets a stale-codegen check. Covers signup, sign-in,
  refresh, me, sign-out, verification, password reset, email change, account deletion,
  oauth exchange, customer info, checkout/portal session creation. Maps gRPC status
  codes + `ErrorInfo` reasons to typed errors (`MothInvalidCredentialsError`,
  `MothEmailNotVerifiedError`, ...); exposes `onAuthStateChanged` and
  `onEntitlementsChanged` subscriptions for non-React code.
- **Calling the developer's own backend**: `client.accessToken()` returns a valid
  (auto-refreshed) JWT, and a drop-in `fetch` wrapper attaches
  `Authorization: Bearer ...` to the app's own API calls; the backend verifies per
  milestone 02. Custom claims are readable on `MothUser.claims` for client-side gating
  (server remains the authority).
- **Token management**: access token in memory; rotating refresh token persisted via a
  pluggable storage interface (default `localStorage`, with `sessionStorage` and
  in-memory options) — the XSS trade-off of SPA token storage is documented honestly
  rather than papered over, with rotation-reuse detection (02) limiting the blast
  radius. Automatic refresh as a connect-web interceptor with single-flight
  de-duplication; session restores on page load; refresh-failure ⇒ signed-out state.
  The same interceptor attaches `x-moth-key`, `x-moth-platform: web`,
  `x-moth-sdk-version`, and `x-moth-language` (15) to every call.
- **React layer**:
  ```tsx
  createRoot(document.getElementById('root')!).render(
    <MothProvider
      config={{ endpoint: 'https://auth.example.com', publishableKey: 'pk_...' }}
      signedOut={<MothLoginScreen />}
    >
      <App />
    </MothProvider>,
  )
  ```
  - `MothProvider` — owns the client, restores the session, gates `children` behind
    authentication (configurable: `requireAuth={false}` renders `children` always).
  - `useMoth()` / `useMothUser()` — hooks exposing auth state
    (`loading | signedOut | signedIn(MothUser)`), the client, and actions (`signOut()`,
    `refreshUser()`, `deleteAccount()` with re-auth prompt). Re-render on state change.
  - `MothLoginScreen` — batteries-included login/signup flow: email/password forms with
    validation, forgot-password, and provider buttons. Google via Google Identity
    Services, Apple via the milestone-04 web redirect flow and hosted redirect pages.
- **Theme & copy from day one** (unlike Flutter, which grew them over 06/15/16):
  `GetProjectConfig` drives everything — milestone-06 theme tokens rendered as CSS
  custom properties scoped under the moth components, milestone-15 negotiated copy with
  the bundled locale set as offline/first-paint fallback, both cached
  stale-while-revalidate keyed by revision (and locale for copy), mirroring the SDK
  cache discipline.

### Billing: entitlements & web paywall (requires 17)

The web counterpart of milestone 13, with Stripe Checkout playing the role the native
stores play on mobile — and simpler for it, because checkout is a redirect, not a
platform billing plugin.

- **Entitlement state in the hooks**: `useMothCustomerInfo()` and
  `useMothEntitlement('pro')` expose the same `CustomerInfo` shape as `MothScope` does
  in Flutter — active entitlements with expiry + source, `none` always valid — fetched
  via `GetCustomerInfo`, cached stale-while-revalidate like theme/copy, refreshed on
  focus and after checkout returns. Gating:
  `<MothGate entitlement="pro" fallback={<MothPaywallScreen />}>...</MothGate>` or the
  hook directly.
- **Purchase flow**: `purchase(product)` calls `CreateCheckoutSession` (17) and
  redirects to Stripe's hosted Checkout; on return to the success URL the SDK
  re-fetches `CustomerInfo` (polling briefly to absorb webhook latency) and re-renders.
  `manageBilling()` opens the Billing Portal session the same way. No card fields, no
  Stripe.js dependency in the SDK — money surfaces stay Stripe-hosted (17's design
  point).
- **`MothPaywallScreen` for the web**: renders the project's offering from the same
  admin-configured paywall config as milestone 13 — header, benefit bullets, tier
  cards (price from the tier's price metadata, trial badge, highlighted tier), purchase
  button, manage-billing and legal links — themed by the milestone-06 tokens and
  localized by the milestone-15 copy, so the admin's paywall editor and live preview
  govern web and mobile alike. Building blocks exported (`MothPaywallHeader`,
  `MothTierCard`, `MothPurchaseButton`) with a full-custom escape hatch, mirroring 13.
  Tiers without a `stripe_price_id` render as unavailable-on-web rather than
  disappearing silently.
- **Optional by construction**: no Stripe credentials or no products → `useMothEntitlement`
  gates never block, the paywall renders its graceful empty state, and the auth-only
  story from the sections above stands alone.

### Example & tests

- `sdk/react/example/` — Vite app against a local moth, including a call to a tiny
  sample backend route that verifies the JWT via the project JWKS — the full loop
  (app → moth → app → developer API) — and a gated "pro" page demonstrating paywall →
  test-mode Checkout → entitlement unlock.
- Vitest unit tests (client, refresh single-flight, error mapping, storage fallback,
  customer-info cache) + component tests (Testing Library) for provider state
  transitions, login form validation, and gate/paywall rendering (none → active →
  expired) against an in-process fake transport; Playwright integration test against a
  real moth binary spawned by the harness (reusing the `web/admin` e2e machinery), with
  the billing leg driven against the milestone-17 Stripe test double.
- CI job: typecheck, `vitest`, stale-codegen check for the TS stubs, tarball build
  reproducibility, `npm pack` validation.

## Key design points

- **Time-to-first-login is still the metric** — `.npmrc` + install + provider snippet
  from the setup tab to "logged in in the browser" in under 10 minutes; time-to-first-
  purchase piggybacks on it the way milestone 13's did on 05.
- **One proto source, three clients** — Dart SDK, admin SPA, React SDK all generate
  from `proto/moth/`; connect-go already serves gRPC-Web on the same port, so the
  server needs no new RPC surface, only `/npm/*`.
- **Core/react split** — auth + billing logic lives in the framework-free core; the
  React layer is thin bindings. A future Vue/Svelte layer is a packaging exercise, not
  a rewrite.
- **One paywall config, every platform** — the admin's paywall editor (13) drives the
  Flutter screen and the web screen from the same stored config, theme tokens, and
  copy; the web adds nothing the operator has to configure twice.
- **The SDK never handles money** — same rule as mobile: it redirects to Stripe-hosted
  Checkout and reads back derived entitlements; validation and entitlement truth stay
  server-side (11/17), and the client cache is a convenience, never the authority.

## Acceptance criteria

- `npm install @moth/react` with the rendered `.npmrc` against a running moth instance
  resolves and downloads the package from `/npm`.
- Example app: signup → email verification → login → full page reload keeps session →
  sign out; Google login works in the browser against milestone-04 test credentials.
- Access-token expiry mid-session refreshes transparently (integration test with a
  5-second TTL project, same as milestone 05).
- A seeded project's theme and custom French copy render in `MothLoginScreen` **and**
  `MothPaywallScreen` without any code change in the example app.
- Billing loop: a free user hits the gated page, sees the paywall, completes a
  test-mode Stripe Checkout, returns, and the page unlocks without a manual refresh;
  `manageBilling()` reaches the Billing Portal; a project with no Stripe config still
  runs the whole auth story with gates never blocking.
- Provider state transitions (`loading → signedOut → signedIn`) and entitlement
  transitions (none → active → expired) covered by component tests.

## Out of scope

SSR/framework server helpers (Next.js/Remix — the SDK works client-side inside them;
server-component integration is future work), a cookie/BFF session mode for same-origin
backends, other framework bindings (Vue, Svelte — enabled by the core split), and any
in-SDK payment UI beyond the redirect (card fields, Stripe.js Elements, one-time
purchases — the Stripe scope is fixed by milestone 17).
