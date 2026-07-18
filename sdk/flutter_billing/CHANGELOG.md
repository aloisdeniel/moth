# Changelog

Versions of this package track the moth server version; the changelog below
is per SDK milestone until the first stamped release.

## 0.1.0

- Initial release: `MothStoreBilling` implements `MothBillingAdapter` over a
  first-party method channel — StoreKit 2 (Swift) on iOS, Play Billing
  Library (Kotlin) on Android. Localized product lookup, native purchase flow
  returning the signed transaction / purchase token for `SubmitPurchase`,
  restore for `RestorePurchases`, typed `pending` / `alreadyOwned` / `error`
  failures, and an out-of-band `transactionUpdates` stream for ask-to-buy and
  pending-payment completions (consumed by `MothApp`, which submits each
  receipt for validation). Auto-renewing subscriptions only; Android
  purchases are acknowledged server-side, never on-device.
