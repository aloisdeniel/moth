# React SDK reference

`@moth/react` is a framework-free client core plus a thin React layer. It
is served from your own instance's npm registry, so the SDK version always
tracks the server version and nothing is published to npmjs. One line in
`.npmrc` routes only `@moth`-scoped packages to your instance; everything
else stays on npmjs:

```ini title=".npmrc"
@moth:registry=https://auth.example.com/npm
```

```sh
npm install @moth/react
```

The package is **project-agnostic** — endpoint and publishable key are
passed at runtime. The project's **Setup** tab in the admin console
renders every snippet below with your real values already filled in.

There are two ways to use it: the **React layer** (`MothProvider` / hooks
/ `MothLoginScreen`) for a batteries-included flow, and the **client
core** (`MothClient`) for full control or non-React code. The React layer
is built on the client core; you can mix them.

## React layer

### MothProvider

The top-level wrapper. It owns a `MothClient`, restores any persisted
session on page load, and gates `children` behind authentication:

```tsx title="src/main.tsx"
import { MothProvider, MothLoginScreen } from '@moth/react'

createRoot(document.getElementById('root')!).render(
  <MothProvider
    config={{ endpoint: 'https://auth.example.com', publishableKey: 'pk_...' }}
    signedOut={<MothLoginScreen />}
  >
    <App />
  </MothProvider>,
)
```

- `config` — endpoint + publishable key; `http://` URLs work for local
  development. Optional fields: `locale` (overrides the browser
  language), `projectSlug` (required for the Google/Apple buttons — the
  web-redirect OAuth flow needs it, along with the app's origin — e.g.
  `https://app.example.com` — registered in the admin under Providers →
  "Redirect origins (web)"; exact origin match, `http://localhost` is
  accepted for development), and [`storage`](#sessions--tokens).
- `signedOut` — rendered while signed out; `<MothLoginScreen />` is the
  built-in flow, or pass your own.

While the session is being restored the provider shows a neutral loading
state; it never flashes the login screen for a user who is actually
signed in.

### Hooks

- `useMoth()` — the auth state (`loading | signedOut | signedIn`), the
  underlying [`MothClient`](#client-core), and actions (`signOut()`,
  `refreshUser()`, `deleteAccount()`). Components re-render on state
  change.
- `useMothUser()` — the current `MothUser`, or `null` when not signed in.
  `MothUser` carries `id`, `email`, `emailVerified`, `displayName`, and
  `claims` — the project-assigned custom claims, readable for client-side
  gating (the server remains the authority).
- `useMothCustomerInfo()` / `useMothEntitlement('pro')` — subscription
  state; see [entitlements & paywall](#entitlements--paywall-stripe).

### MothLoginScreen

The default sign-in surface: email/password sign-in and sign-up with
validation, forgot-password, and — when the project enables them —
Google/Apple buttons (Google via Google Identity Services, Apple via the
web-redirect flow). It reads the project's public config to show only the
providers that are turned on, and renders the project's
[theme](../guides/theming/) and localized copy — see
[theming & copy](#theming--copy).

## Entitlements & paywall (Stripe)

The web counterpart of the [subscriptions & paywall
guide](../guides/monetization/), with Stripe Checkout playing the role the
native stores play on mobile. Everything is optional: a project with no
Stripe credentials or no products runs the whole auth story with gates
never blocking.

```tsx
// Gate a page behind an entitlement; free users see the paywall.
<MothGate entitlement="pro" fallback={<MothPaywallScreen />}>
  <ProPage />
</MothGate>

// Or check imperatively:
const { active } = useMothEntitlement('pro')
```

`MothPaywallScreen` renders the offering from the same admin-configured
paywall as the Flutter one — headline, benefits, tier cards, highlighted
tier — themed and localized by the project config, so **Design → Paywall**
governs web and mobile alike. Tiers without a Stripe price render as
unavailable on the web.

The purchase flow is a redirect, never a card field:

- `purchase(product)` creates a Stripe Checkout session and navigates to
  it. On return the SDK re-reads the entitlements (polling briefly to
  absorb webhook latency) and gates unlock without a manual refresh. The
  result is a typed value, never an exception: `redirect`, `purchased`,
  `pending`, `alreadyOwned`, `cancelled`, or `error`.
- `manageBilling()` redirects to the Stripe Billing Portal — cancel,
  payment methods, invoices.

Entitlement state is cached locally, so gating is instant on load and
refreshes in the background; the server-derived state is always the
authority.

## Client core

`MothClient` is an ergonomic wrapper over the connect-web clients
generated from `moth.auth.v1` and `moth.billing.v1` — the same protos the
Flutter SDK and the admin console are generated from. Use it directly in
non-React code, tests, or a custom UI.

```ts
import { MothClient } from '@moth/react'

const moth = new MothClient({
  endpoint: 'https://auth.example.com',
  publishableKey: 'pk_...',
})

await moth.restore() // resume a persisted session, if any

const { user } = await moth.signIn({ email: 'jane@example.com', password: '…' })
```

Methods map one-to-one to the [auth API](../api/#mothauthv1):

| Area | Methods |
|---|---|
| Session | `restore()`, `signIn({email, password})`, `signUp({email, password, displayName?})`, `signOut({allDevices?})` |
| Current user | `getMe()` / `refresh()`, `updateMe({displayName})`, `changePassword({current, next})` |
| Email | `requestEmailVerification(email)`, `requestEmailChange(newEmail)` |
| Password reset | `requestPasswordReset(email)` |
| Social | `signInWithOAuth(...)`, `exchangeOAuthCode(...)`, `unlinkIdentity(provider)` |
| Account | `deleteAccount({password})` (fresh re-auth required) |
| Config | `getProjectConfig()` |
| Billing | `getCustomerInfo()`, `getOfferings()`, `purchase(product)`, `manageBilling()`, `createCheckoutSession(...)`, `createBillingPortalSession(...)` |
| Tokens | `accessToken()` |

The confirmation half of email verification, password reset, and email
change is completed from the [hosted pages](../api/#hosted-pages) moth
emails the user — the app requests them, the link finishes them.

### Auth state outside React

`onAuthStateChanged(listener)` and `onEntitlementsChanged(listener)`
subscribe non-React code to the same state the hooks expose; the current
value is replayed to every new listener, and both return an unsubscribe
function. `currentState` reads it synchronously.

### Errors

Every failure is a typed subclass of `MothError`, mapped from the
server's gRPC status and stable `ErrorInfo` reason
([error model](../api/#errors)):

```ts
try {
  await moth.signIn({ email, password })
} catch (e) {
  if (e instanceof MothInvalidCredentialsError) {
    // wrong email or password (uniform — never reveals which)
  } else if (e instanceof MothEmailNotVerifiedError) {
    // project requires verification before sign-in
  } else if (e instanceof MothRateLimitedError) {
    // too many attempts; back off
  } else if (e instanceof MothNetworkError) {
    // transport failure, not an auth decision
  }
}
```

Others include `MothWeakPasswordError`, `MothEmailAlreadyExistsError`,
`MothSignUpClosedError`, and `MothBillingNotConfiguredError`.

## Calling your own backend

The reason auth exists: your API trusts the app's requests.
`moth.accessToken()` always returns a valid, auto-refreshed JWT, and
`createMothFetch` is a drop-in `fetch` that attaches it:

```ts
const apiFetch = createMothFetch(moth)
const resp = await apiFetch('https://api.example.com/todos')
```

Your backend verifies that token offline against the project JWKS — see
[verifying tokens on your backend](../api/#verifying-tokens-on-your-backend).

## Sessions & tokens

- **Persistence** — the access token lives in memory; the rotating
  refresh token persists via `config.storage`: `'local'` (localStorage,
  the default — survives restarts), `'session'` (per-tab),
  `'memory'` (nothing survives a reload), or a custom `TokenStore`.
- **The XSS trade-off is real** — any script running on your origin can
  read web storage; there is no way around this for a pure SPA, and moth
  documents it rather than papering over it. Server-side rotation-reuse
  detection limits the blast radius of a stolen refresh token — a
  replayed token revokes the session — but a strict Content-Security-
  Policy is your first line of defense.
- **Automatic refresh** — access tokens refresh proactively before
  expiry, implemented as a connect-web interceptor with single-flight
  de-duplication, so concurrent callers share one refresh RPC. A refresh
  rejected as revoked or reused clears the stored session and emits the
  signed-out state.
- **Version coupling** — the SDK version matches the server version by
  construction: the tarball is built into the binary that serves it.

## Theming & copy

Both `MothLoginScreen` and `MothPaywallScreen` are driven entirely by the
project config — no hardcoded styles or strings:

- The [theme](../guides/theming/) renders as CSS custom properties
  (`--moth-*`) scoped under the moth components, with light and dark
  resolved per `prefers-color-scheme`.
- Copy is negotiated per the browser language (override with
  `config.locale`), with the SDK's bundled locales as offline /
  first-paint fallback.
- Both are cached locally and revalidated in the background, keyed by the
  server-side revision — an admin edit reaches running apps without a
  deploy.
