# Push notifications

moth gives every project a **push-device registry**: each signed-in device
registers its push credential (an APNs device token, an FCM registration
token, or a Web Push subscription) together with an honest OS permission
state, and your backend reads the current set over `moth.server.v1` to
deliver through the push services' own APIs. The same division of labor as
auth and billing: **moth registers; your server sends.** Delivery needs
sender credentials, retry logic, and payload semantics that belong to your
backend — moth never holds an APNs key, an FCM service account, or a VAPID
private key, and it never sends a push.

The registry is deliberately self-healing: registration is an **upsert**
keyed by a stable per-installation `device_id` (and unique per
`(target, token)`), so the SDKs simply re-register on every launch, token
rotation, and permission change — no client-side "am I registered?"
bookkeeping to corrupt. Sign-out revokes the registration, your sender's
feedback revokes dead credentials, and a staleness sweep decays abandoned
installs (90 days by default).

## 1 · Enable push for the project

In the project's **Push** tab, under **Push notifications**, switch on
push registration. For Web Push, also paste your **VAPID public key** (see
[step 3](#3--generate-a-vapid-keypair-web-push)); mobile-only projects
leave it empty. Both
land in the public project config, so the SDKs pick them up without a
release — a project with push disabled reports `unavailable` in the SDKs
and refuses new registrations.

## 2 · Register devices from Flutter

Add `moth_push` next to `moth_auth` — moth's first-party push plugin, APNs
(`UNUserNotificationCenter` + `registerForRemoteNotifications`) on iOS and
Firebase Cloud Messaging on Android, served from your instance's own
`/pub` at the server's version, so the credentials it produces are exactly
what the registry stores:

```yaml
dependencies:
  moth_push:
    hosted: https://auth.example.com/pub
    version: ^1.0.0
```

```dart
import 'package:moth_push/moth_push.dart';

MothApp(
  config: MothConfig(endpoint: ..., publishableKey: 'pk_...'),
  pushAdapter: MothNativePush(),
  child: const MyApp(),
);
```

Wiring the adapter is the whole opt-in. While a user is signed in, the SDK
registers on every launch, re-registers on token rotation and permission
changes, and revokes the registration on sign-out — all silently, and all
non-fatal: auth never blocks on push, and a failed registration simply
retries on the next launch.

The one thing the SDK never does on its own is **show the OS permission
prompt** — that's a product decision, so it stays an explicit app call:

```dart
final permission = await MothScope.of(context).requestPushPermission();
```

`MothScope.of(context).pushStatus` exposes the state for a settings row
(available, permission, registered — see the example app). A device that
**denies** permission still registers, flagged `denied` — data pushes may
still work, and granting later simply updates the registration.

Platform setup:

- **iOS** — add the Push Notifications capability in Xcode (and a matching
  APNs key in your sender). `MothNativePush(provisional: true)` requests
  [provisional authorization](https://developer.apple.com/documentation/usernotifications/asking-permission-to-use-notifications)
  — quiet delivery without a prompt.
- **Android** — FCM requires your app's own Firebase configuration
  (`google-services.json` plus the `com.google.gms.google-services` Gradle
  plugin). This is the one piece of setup moth cannot absorb: the Firebase
  project is your sender identity. When it's missing, the plugin fails
  registration with an actionable `firebase-not-initialized` error —
  visible in debug logs, never blocking auth.

Delivery and display are **not** the plugin's business: it produces
credentials and reports permission faithfully, nothing more. Notification
handlers, foreground banners, and tap routing stay your app's code — an
app with its own push stack (an existing `firebase_messaging` setup, say)
implements `moth_auth`'s `MothPushAdapter` interface instead of using
`moth_push`, and moth still gets faithful registrations.

## 3 · Generate a VAPID keypair (Web Push)

Web Push authenticates senders with a [VAPID](https://datatracker.ietf.org/doc/html/rfc8292)
keypair instead of a per-service credential. Generate one once:

```sh
npx web-push generate-vapid-keys
```

Paste the **public** key into the project's push settings — browsers need
it to subscribe, and it is plain config, delivered via the public project
config. The **private** key stays in your sender and never touches moth.

## 4 · Web Push (React)

`useMothPush()` turns the registry into a settings-screen toggle:

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

`subscribe()` prompts for browser permission, subscribes your service
worker's `PushManager` with the project's VAPID public key, and registers
the serialized subscription as `target: webpush`. Environment problems are
states, never exceptions: a project without push (or without a VAPID key)
reports `unavailable`, a browser without the Push API reports
`unsupported` (feature-detected, never user-agent sniffed), and
`subscribe()` is a typed no-op in both.

The app owns its service worker — display and click handling are app code,
same rule as Flutter. A minimal `sw.js` (served from your origin and
registered once at startup with `navigator.serviceWorker.register('/sw.js')`):

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

## 5 · Send from your backend

Your sender reads the registry with the project **secret key** over
`moth.server.v1.PushService` — the only surface that ever returns tokens
(the client and admin APIs see metadata only):

```sh
# One user's active devices — everything a sender needs to fan out:
grpcurl -H "x-moth-key: sk_..." -d '{"user_id":"..."}' \
  your-moth:8080 moth.server.v1.PushService/ListUserPushDevices

# Or page through the whole project for broadcast sends, filterable by
# target: ListPushDevices(page_size, page_token, target).
```

> **grpcurl against a release build**
>
> Release builds don't advertise their schema over reflection. Either start
> the server with `--reflection`, or point grpcurl at the `.proto` sources
> the instance serves at `/protos/` (`grpcurl -import-path … -proto …`).

Each `PushDevice` carries the `target` (which API to call), the `token`
(the credential to hand to that API), the last-reported `permission`
(skip alert pushes for `denied` devices — data pushes may still work), and
display metadata (`platform`, `model`, `locale`, …) for targeting. Then
deliver with the services' own APIs or any library: APNs over HTTP/2 with
your `.p8` key, FCM via the HTTP v1 API with your service-account JSON,
Web Push with e.g. the [`web-push`](https://github.com/web-push-libs/web-push)
package and the same VAPID keypair — the `webpush` token is the serialized
subscription JSON, ready to pass straight to `webpush.sendNotification`.

Tokens are credentials: scope them to your sender, and never log or
re-expose them.

## 6 · Close the feedback loop

Push services tell senders about dead credentials — report them back and
moth revokes the registration (`reported_invalid`) so it is never served
again:

- APNs: `410 Unregistered`
- FCM: `UNREGISTERED` / `NOT_FOUND`
- Web Push: `404` or `410` from the subscription endpoint

```sh
grpcurl -H "x-moth-key: sk_..." -d '{"token":"..."}' \
  your-moth:8080 moth.server.v1.PushService/RevokePushDevice
```

Idempotent by design — unknown or already-revoked credentials succeed, so
your sender can report without bookkeeping. The rest of the lifecycle is
automatic: sign-out revokes (`signed_out`, called by the SDKs while the
session is still valid), re-registration supersedes (`replaced`),
registrations unseen for the staleness window decay (`stale`), and an
operator can revoke from the admin (`admin`, audit-logged).

## Watching the registry

The project's **Push** tab lists every registered device — owning user,
target, platform/model, permission, app version, last seen — newest first,
with per-target totals, a target filter, and a revoke action. Each user's
detail drawer keeps a per-user **Devices** panel that also shows recently
revoked registrations and why. Neither surface ever shows tokens — those
stay on the secret-key API your sender uses.

## Out of scope

Sending pushes and sender credentials (your backend's job, by design),
notification display/tap handling and deep links, rich media, topics and
client-side segmentation, and a first-party Firebase bootstrap — the app
owns its `google-services.json`.
