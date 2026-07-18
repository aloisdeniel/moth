# @moth/react

React SDK for [moth](https://github.com/aloisdeniel/moth) — authentication,
entitlements, and a themed login screen + paywall for apps backed by a moth
instance. Served directly from your moth server's embedded npm registry; the
package version always matches the server.

## Install

Route the `@moth` scope to your instance (everything else stays on npmjs):

```ini
# .npmrc
@moth:registry=https://auth.example.com/npm
```

```sh
npm install @moth/react
```

## Use

```tsx
import { createRoot } from 'react-dom/client'
import { MothProvider } from '@moth/react'

createRoot(document.getElementById('root')!).render(
  <MothProvider
    config={{ endpoint: 'https://auth.example.com', publishableKey: 'pk_...' }}
  >
    <App />
  </MothProvider>,
)
```

`MothProvider` restores the persisted session on mount and gates its
children: a loading splash while restoring, the built-in themed
`MothLoginScreen` while signed out (override with `signedOut={...}` /
`loadingFallback={...}`), your app once signed in. Pass
`requireAuth={false}` to always render children and read the state from the
hooks yourself. The project's admin-configured theme and localized copy
drive every moth-owned surface — cached download-once (protobuf
`CacheEnvelope` blobs, one-hour TTL by default) and revalidated by revision,
so branding changes apply without a release. Moth styling is scoped under a
`.moth-root` wrapper around moth-owned surfaces only; your app's subtree is
never touched.

## Hooks

- `useMoth()` — `{ client, state, user, customerInfo, signOut, refreshUser, deleteAccount, refreshCustomerInfo }`.
- `useMothUser()` — the signed-in `MothUser` (id, email, emailVerified,
  displayName, avatarUrl, createTime, `claims` decoded from the JWT), or null.
- `useMothCustomerInfo()` — the `MothCustomerInfo` snapshot; always valid,
  the free tier while signed out. Refreshed on window focus and after
  checkout returns.
- `useMothEntitlement('pro')` — `{ active, entitlement }`; re-renders when
  the entitlement flips, including at its expiry.
- `useMothPush()` — `{ status, permission, subscribe, unsubscribe }` for
  Web Push registration; see below.

## Components

- `MothProvider` — owns the client, session restore, theme/copy/entitlement
  state.
- `MothLoginScreen` — email/password sign-in / sign-up (validation from the
  project's password policy and sign-up switch), forgot-password flow, and
  Google/Apple buttons via the web-redirect OAuth flow (needs
  `config.projectSlug`, plus the app's origin — e.g.
  `https://app.example.com` — registered in the admin under Providers →
  "Redirect origins (web)"; the browser returns to the same page with a
  one-time `?code=` the SDK exchanges automatically). The redirect URL is
  the current URL with its fragment stripped — the server refuses redirect
  URIs containing `#` — so hash-routed apps (e.g. `HashRouter`) come back
  to the fragment-less URL after Google/Apple sign-in and should restore
  their route themselves.
- `MothGate` — `<MothGate entitlement="pro" fallback={...}>` shows the
  paywall (default) until the entitlement arrives; when no product in the
  paywall's offering grants the entitlement it falls through to children —
  a project with no billing never blocks.
- `MothPaywallScreen` — the admin-configured paywall: header, benefits,
  tier cards (price from the catalog, trial + most-popular badges),
  Stripe-hosted Checkout purchase button, manage-billing link, legal links,
  empty and error states. Tiers without a `stripe_price_id` render as
  unavailable-on-web. Building blocks exported: `MothPaywallHeader`,
  `MothTierCard`, `MothPurchaseButton`.

## Core client (framework-free)

`MothClient` works without React (and is the base a future Vue/Svelte layer
would reuse): `restore`, `accessToken`, `refresh`, `signUp`, `signIn`,
`signOut`, `changePassword`, `signInWithOAuth`, `exchangeOAuthCode`,
`oauthStartUrl`, `signInWithRedirect` (navigates to the web-redirect OAuth
flow; the default redirect is the current URL without its fragment),
`unlinkIdentity`, `getMe`, `updateMe`, `deleteAccount`,
`requestEmailVerification`/`confirmEmailVerification`,
`requestPasswordReset`/`confirmPasswordReset`,
`requestEmailChange`/`confirmEmailChange`, `getProjectConfig`,
`getCustomerInfo`, `getOfferings`, `getPaywall`, `createCheckoutSession`,
`createBillingPortalSession`, `purchase`, `manageBilling`,
`handleCheckoutReturn`, `registerPushDevice`, `unregisterPushDevice`, plus
`onAuthStateChanged` / `onEntitlementsChanged` subscriptions that replay
the current value to every new subscriber and `onBeforeSignOut` hooks that
run while the session is still valid (how push unregistration rides along
with `signOut`).

Errors are typed by the server's stable `ErrorInfo` reasons
(`MothInvalidCredentialsError`, `MothEmailNotVerifiedError`,
`MothRateLimitedError`, ...); transport failures surface as
`MothNetworkError`; unknown reasons fall back to `MothError` with `reason`
set.

## Calling your own backend

```ts
import { createMothFetch } from '@moth/react'

const apiFetch = createMothFetch(client)
const resp = await apiFetch('https://api.example.com/todos')
```

`createMothFetch` attaches `Authorization: Bearer <access token>` — kept
fresh automatically (single-flight refresh, 30s proactive skew, one
reactive retry on a server-side rejection). Your backend verifies the JWT
against the project JWKS and remains the authority; `MothUser.claims` is
for client-side gating only.

## Purchases (Stripe Checkout)

`client.purchase(product)` creates a Checkout session and redirects to
Stripe's hosted page — no card fields, no Stripe.js in your bundle. The
return URL carries a `moth_checkout` marker; the provider consumes it on
load and briefly polls `GetCustomerInfo` to absorb webhook latency, so the
gated page unlocks without a manual refresh. `client.manageBilling()`
opens the Stripe Billing Portal the same way.

## Web Push

`useMothPush()` turns the project's Web Push configuration (a VAPID public
key, set in the admin's Push tab) into a settings-screen toggle:

```tsx
import { useMothPush } from '@moth/react'

function PushToggle() {
  const { status, subscribe, unsubscribe } = useMothPush()
  if (status === 'unavailable' || status === 'unsupported') return null
  if (status === 'denied') return <p>Notifications are blocked in the browser.</p>
  return status === 'subscribed' ? (
    <button onClick={() => void unsubscribe()}>Disable notifications</button>
  ) : (
    <button onClick={() => void subscribe()}>Enable notifications</button>
  )
}
```

`subscribe()` requests the browser notification permission, subscribes the
app's service worker's `PushManager` with the project's VAPID public key
(read from the public project config) and registers the serialized
subscription with the moth device registry (`target: webpush`, a stable
per-installation id persisted in `localStorage`). While signed in, an
existing subscription is re-registered on every launch — the registry's
upsert semantics are the retry policy — and sign-out revokes the
registration before the session drops. Environment problems are states,
never exceptions: a project without a VAPID key reports
`status: 'unavailable'`, a browser without the Push API (feature-detected)
reports `'unsupported'`, and `subscribe()` is a typed no-op in both.

The app owns its service worker — display and click handling are app code,
moth only manages the subscription and the registry row. A minimal `sw.js`
(served from your app's origin, e.g. `public/sw.js`, and registered once at
startup with `navigator.serviceWorker.register('/sw.js')`):

```js
// sw.js — payload shape is yours; this expects { title, body, url }.
self.addEventListener('push', (event) => {
  const data = event.data?.json() ?? {}
  event.waitUntil(
    self.registration.showNotification(data.title ?? 'Notification', {
      body: data.body,
      data,
    }),
  )
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  event.waitUntil(self.clients.openWindow(event.notification.data?.url ?? '/'))
})
```

Sending stays your backend's job: fetch the registered devices over
`moth.server.v1` and deliver with any Web Push library (e.g. `web-push`
with the same VAPID key pair). moth registers; your server sends.

## Session storage trade-off

The rotating refresh token persists in `localStorage` by default
(`storage: 'local'`), namespaced per publishable key. Web storage is
readable by any script on your origin — the XSS trade-off of SPA token
storage is real, not papered over. moth's rotation-reuse detection revokes
the whole token family when a stolen refresh token is replayed, limiting
the blast radius, but a strict Content-Security-Policy is your first line
of defense. Alternatives: `storage: 'session'` (per-tab), `'memory'`
(nothing survives a reload), or your own `TokenStore` implementation.

## Example

`example/` in the repo is a small Vite app wired to a local moth: login,
session persistence across reloads, an authenticated call to the sample
backend, and a pro page behind `MothGate` with the full checkout loop.

See your moth admin's Setup tab for snippets pre-filled with your project's
values, and the hosted docs under `/docs` on your instance.
