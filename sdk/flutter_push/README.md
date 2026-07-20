# moth_push

First-party native push registration for
[moth](https://github.com/aloisdeniel/moth): a Flutter plugin implementing
`moth_auth`'s `MothPushAdapter` with **APNs** on iOS and **Firebase Cloud
Messaging** on Android. Served by your own moth instance at `/pub` — the
plugin version tracks your server version, so the credentials it produces
are exactly what the server's `RegisterDevice` stores.

```yaml
dependencies:
  moth_auth:
    hosted: https://auth.example.com/pub
    version: ^1.0.0
  moth_push:
    hosted: https://auth.example.com/pub
    version: ^1.0.0
```

## Usage

Pass `MothNativePush()` to `MothApp` — that flag is the whole opt-in:

```dart
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_push/moth_push.dart';

MothApp(
  config: MothConfig(...),
  pushAdapter: MothNativePush(),
  child: ...,
)
```

While a user is signed in the SDK keeps the server's device registry
current: it registers on every launch with a stable device id, re-registers
on token rotation, and unregisters on sign-out. The SDK never shows the OS
permission prompt on its own — call
`MothScope.of(context).requestPushPermission()` from your own UI (a settings
row, an onboarding step), and read `MothScope.of(context).pushStatus` for
the current permission/registration state.

## Platform setup

- **iOS** — enable the *Push Notifications* capability (and *remote
  notifications* background mode if you send background pushes) on your app
  target. No AppDelegate code: the plugin hooks the registration callbacks
  through Flutter's plugin lifecycle. Tokens are registered as
  `target: apns`. `MothNativePush(provisional: true)` requests provisional
  (quiet, prompt-less) authorization instead of the full dialog.
- **Android** — FCM needs your app's own Firebase project, the one piece of
  setup moth cannot absorb: download `google-services.json` into
  `android/app/` and apply the `com.google.gms.google-services` Gradle
  plugin. Without it, token retrieval fails with an actionable
  `firebase-not-initialized` error (auth is never blocked — the SDK treats
  push failures as non-fatal). The plugin declares `POST_NOTIFICATIONS` and
  handles the API 33+ runtime permission flow; tokens are registered as
  `target: fcm`.

## Scope: credentials, not notifications

This plugin ends where the push services' payloads begin. It produces the
`(target, token)` credential and an honest permission state — **delivery and
display are deliberately out of scope**: no notification handlers, no
foreground banners, no tap routing, no deep links. The only native receiver
it registers is an Android `FirebaseMessagingService` that forwards *token
rotation* (`onNewToken`) and touches no messages. Keep your existing
display/tap handling (or add your own service — it takes over message
delivery without breaking moth's registration, which refreshes the token on
every launch anyway). Your server sends; moth just knows where.

Apps with their own push stack (e.g. an existing `firebase_messaging`
setup) implement `MothPushAdapter` themselves — `moth_auth` stays pure Dart
and never depends on this plugin.
