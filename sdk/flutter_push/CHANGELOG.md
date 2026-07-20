# Changelog

Versions of this package track the moth server version; the changelog below
is per SDK milestone until the first stamped release.

## 0.1.0

- Initial release: `MothNativePush` implements `MothPushAdapter` over a
  first-party method channel — APNs (Swift) on iOS, Firebase Cloud Messaging
  (Kotlin) on Android. OS permission request/status (including iOS
  provisional authorization and the Android API 33+ `POST_NOTIFICATIONS`
  flow), `(target, token)` credential retrieval for `RegisterDevice`, an
  `onTokenRefresh` stream fed by APNs re-registration callbacks and FCM
  `onNewToken`, and native device metadata (model, app version). Missing
  Firebase config surfaces as an actionable `firebase-not-initialized`
  error. Credentials only — no notification display or tap handling.
