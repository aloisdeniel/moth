# moth_billing

First-party native billing for [moth](https://github.com/aloisdeniel/moth):
a Flutter plugin implementing `moth_auth`'s `MothBillingAdapter` with
**StoreKit 2** on iOS and the **Play Billing Library** on Android. Served by
your own moth instance at `/pub` — the plugin version tracks your server
version, so the receipts it produces are exactly what the server validates.

```yaml
dependencies:
  moth_auth:
    hosted: https://auth.example.com/pub
    version: ^1.0.0
  moth_billing:
    hosted: https://auth.example.com/pub
    version: ^1.0.0
```

## Usage

Pass `MothStoreBilling()` to `MothApp` (or `MothPaywallScreen`) — no adapter
code to write:

```dart
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_billing/moth_billing.dart';

MothApp(
  config: MothConfig(...),
  billingAdapter: MothStoreBilling(),
  child: ...,
)
```

The milestone-13 paywall then sells: `productsFor` reads localized store
prices, `purchase` runs the native flow and returns the StoreKit 2 signed
transaction (Apple) or Play Billing purchase token (Google) for
`SubmitPurchase`, and `restore` feeds `RestorePurchases`.

## Scope

Auto-renewing subscriptions only, matching the server's validation scope. No
consumables, offer codes, or price-change flows. The server stays the
authority on entitlements:

- **iOS** — StoreKit 2 only (iOS 15+). Unverified transactions are rejected
  on-device; transactions are finished only after the receipt is handed to
  Dart. Ask-to-buy deferrals surface as a `pending` result and complete
  through the transaction-updates listener: `MothApp` subscribes to
  `MothStoreBilling.transactionUpdates` and submits each receipt for
  validation, so a deferred purchase flips entitlements with no app code.
- **Android** — Play Billing Library. Purchases are **never acknowledged
  on-device**: the moth server acknowledges after validating the purchase
  token, so an unvalidated purchase auto-refunds instead of being silently
  kept. `PENDING` purchases surface as the `pending` result.

Apps with needs beyond this implement `MothBillingAdapter` themselves —
`moth_auth` stays pure Dart and never depends on this plugin.
