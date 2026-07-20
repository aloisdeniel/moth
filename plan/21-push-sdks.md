# Milestone 21 — Push Registration in the SDKs

## Goal

Make the milestone-20 registry self-populating: a Flutter app opts in with one flag and
one first-party plugin, a React app with one hook — and every signed-in device shows up
in `ListUserPushDevices` with a live token and an honest permission state, then
disappears on sign-out. This is the client half of push: **`moth_push`**, a native
plugin (APNs on iOS, FCM on Android) served from `/pub` like `moth_billing` (19), the
registration lifecycle wired into `MothScope`, and Web Push subscription in
`@moth/react` (18) using the project's VAPID public key (20).

## Deliverables

### `moth_push` package (`sdk/flutter_push/`)

The milestone-19 plugin model applied to push: a small method-channel plugin producing
exactly the credential the registry stores, mapped per platform to the native push
service.

- **Dart layer** — `MothNativePush implements MothPushAdapter` (the adapter interface
  lands in `moth_auth`, mirroring `MothBillingAdapter`): `requestPermission()`,
  `permissionStatus()`, `getToken()` → `(target, token)`, and an `onTokenRefresh`
  stream. `moth_auth` itself stays pure Dart; the adapter is the escape hatch for apps
  with their own push stack (e.g. an existing `firebase_messaging` setup).
- **iOS (Swift, APNs)** — native `UNUserNotificationCenter` permission request
  (including provisional), `registerForRemoteNotifications`, the APNs device token
  surfaced as `target: apns`; token forwarded on every `didRegisterForRemoteNotifications`
  (APNs tokens can rotate on restore/OS update).
- **Android (Kotlin, FCM)** — FCM registration-token retrieval and rotation callback
  surfaced as `target: fcm`, plus the runtime `POST_NOTIFICATIONS` permission flow
  (API 33+). FCM requires the app's own Firebase config (`google-services.json`) —
  documented plainly as the one piece of setup moth cannot absorb; `moth doctor`-style
  actionable errors when it's missing.
- Delivery/display is **not** the plugin's business: no notification handlers, no
  foreground banners — the app keeps its own tap/display handling. The plugin produces
  credentials; the registry stores them; your server sends.

### Registration lifecycle in `moth_auth`

- `MothApp(push: MothNativePush())` (or any `MothPushAdapter`) turns the flow on;
  no adapter, no push — nothing else changes.
- `MothScope` orchestrates against `moth.push.v1` (20): after sign-in (and on every
  launch while signed in) it obtains the token and calls `RegisterDevice` with a
  persisted stable `device_id`, the current permission state, and device metadata;
  `onTokenRefresh` and permission changes re-register (upsert semantics make this
  carefree); sign-out calls `UnregisterDevice` **before** dropping the session, then
  clears local push state.
- Permission UX stays in the app's hands: `MothScope.of(context).requestPushPermission()`
  is explicit — the SDK never prompts on its own (permission prompts are a product
  decision, not an SDK side effect). `pushStatus` (registered / permission state) is
  exposed on the scope for settings screens.
- Registration failures are non-fatal by design: auth never blocks on push; failures
  retry on next launch (the registry's idempotence is the retry policy).

### Web Push in `@moth/react` (18)

- `useMothPush()` — `{ status, permission, subscribe(), unsubscribe() }`:
  `subscribe()` requests browser permission, subscribes the app's service worker's
  `PushManager` with the project's **VAPID public key** (read from the public project
  config, 20), and registers the serialized subscription as `target: webpush`;
  `unsubscribe()` unsubscribes and calls `UnregisterDevice`. The app supplies its own
  service worker (display/click handling is app code, same rule as Flutter); the SDK
  documents a minimal `sw.js`.
- The provider re-registers on subscription change and unregisters on `signOut()`;
  a project with no VAPID key renders `status: unavailable` and `subscribe()` is a
  typed no-op — optional by construction, like billing in 18.
- Browser support is feature-detected (`PushManager` in `window`), not user-agent
  sniffed; unsupported browsers get `unavailable`, never an exception.

### Examples, docs & tests

- Flutter example app: a settings row with a push toggle + permission state; README
  walk-through from `moth_push` dependency to a device row in the admin panel, plus
  the Firebase-config caveat. React example: the same toggle with the sample service
  worker.
- Dart contract tests for `MothNativePush` against a fake method-channel host (grant/
  deny/provisional, token rotation) and `MothScope` lifecycle tests (sign-in registers,
  rotation re-registers, sign-out unregisters — asserted against an in-process fake
  push service). Vitest/component tests for `useMothPush` with a stubbed `PushManager`.
- `/pub` serves `moth_push` (multi-package serving from 19); setup tab and
  `moth skill export` gain the push section.

## Key design points

- **One flag to opt in, zero prompts by surprise.** Wiring the adapter enables the
  machinery; showing the OS permission dialog stays an explicit app call. The SDK's
  job is that registration state is always faithful to what the OS reports.
- **The registry's idempotence is the client's simplicity.** Register-on-every-launch
  with a stable `device_id` replaces client-side bookkeeping (no "am I registered?"
  cache to corrupt) — the same lean-on-the-server discipline as the theme/copy caches.
- **Credentials, not notifications.** Neither plugin touches message display,
  routing, or analytics; moth's SDKs end where the push services' payloads begin.
- **Same plugin economics as 19.** Served from `/pub`, version-locked to the server,
  native surface small enough to audit; `moth_auth` gains an interface, never a
  dependency.

## Acceptance criteria

- Flutter example on a device: sign in → grant permission → the device appears in the
  admin Devices panel and `ListUserPushDevices` with `target` `apns`/`fcm`; sign out →
  it's revoked (`signed_out`). Token rotation (simulated via the fake host in tests)
  re-registers without duplicating.
- Denying permission still registers with `permission: denied`; granting later
  updates it — asserted in lifecycle tests.
- React example in a browser: `subscribe()` yields a `webpush` registration whose
  subscription JSON round-trips to a real `web-push` send in the docs walk-through;
  sign-out unregisters; a project without a VAPID key shows `unavailable` and never
  throws.
- Auth flows are unaffected when no adapter is wired (widget tests) and when
  registration fails (fault-injected fake — sign-in still completes).
- `dart pub get` resolves all three packages (`moth_auth`, `moth_billing`,
  `moth_push`) from a running instance.

## Out of scope

Sending pushes and sender credentials (still the developer's backend, per 20),
notification display/tap handling and deep links, rich media, background data-message
processing, topics/segmentation client-side, a first-party Firebase bootstrap (the
app owns its `google-services.json`), and Safari legacy web push (only standard Web
Push API browsers).
