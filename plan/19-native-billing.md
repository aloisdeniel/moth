# Milestone 19 — Native Billing (first-party store plugin)

## Goal

Close the last hand-wired gap in the purchase story. Milestone 13 deliberately kept
native billing out of `moth_auth` behind the `MothBillingAdapter` interface, with every
app copying an `in_app_purchase` implementation into its own tree. This milestone ships
moth's **own native billing**: `moth_billing`, a Flutter plugin package served from the
binary at `/pub` alongside `moth_auth`, implementing the adapter with **StoreKit 2**
(Swift) on iOS and the **Play Billing Library** (Kotlin) on Android — no third-party
billing plugin, no adapter to write. `dependencies: moth_billing` + one constructor
argument, and the milestone-13 paywall sells.

The same reasoning that made moth validate store state itself (11) instead of renting a
billing SaaS applies on-device: the StoreKit 2 `jwsRepresentation` and the Play Billing
`purchaseToken` are exactly what the server verifies, so moth should produce them
first-party, in lockstep with the server version, rather than round-tripping through a
general-purpose plugin's abstractions.

## Deliverables

### Multi-package pub serving

- The `/pub` repository grows from one hardcoded package to a **package set**:
  `GET /pub/api/packages/{name}` and the tarball route serve `moth_auth` **and**
  `moth_billing` (and future companions, e.g. milestone 21's `moth_push`) from the
  embedded `sdk/` FS; unknown names still 404. Both packages are stamped with the same
  server version at serve time — the SDK-lockstep discipline is unchanged.
- `sdk/embed.go` embeds the publishable subset of `sdk/flutter_billing/` (lib, native
  `ios/` + `android/` sources, pubspec, plugin manifest, README/CHANGELOG/LICENSE).
- The setup-instructions tab (03) and docs gain the `moth_billing` dependency line in
  the monetization section; `moth skill export` (08) teaches it.

### `moth_billing` package (`sdk/flutter_billing/`)

A thin, honest plugin: one method-channel surface, no Dart-side store modeling beyond
what `MothBillingAdapter` already defines in `moth_auth`.

- **Dart layer** — `MothStoreBilling implements MothBillingAdapter`:
  `productsFor(offering)` resolves the offering's store product ids to native store
  products (localized display price, currency, intro/trial phase), `purchase(productId)`
  runs the native flow and returns the `MothPurchaseReceipt` (`apple_jws` |
  `google_purchase_token`) that `SubmitPurchase` (11) expects, `restore()` returns the
  current store receipts for `RestorePurchases`. All store failures map to the existing
  `MothPurchaseException` kinds (`cancelled`, `pending`, `alreadyOwned`, `unavailable`,
  `storeError`) — the milestone-13 typed-result contract is the contract.
- **iOS (Swift, StoreKit 2)** — `Product.products(for:)` for catalog lookup,
  `product.purchase()` for the flow, verified `Transaction.jwsRepresentation` forwarded
  as the receipt, `Transaction.currentEntitlements` for restore, a `Transaction.updates`
  listener surfacing ask-to-buy/deferred completions and renewals as adapter events,
  `transaction.finish()` only after the receipt is handed to Dart. Unverified
  transactions are rejected on-device (the server re-verifies regardless).
- **Android (Kotlin, Play Billing Library)** — `BillingClient` connection management
  with retry, `queryProductDetailsAsync` for catalog lookup, `launchBillingFlow` for the
  flow, the resulting `purchaseToken` forwarded as the receipt, `queryPurchasesAsync`
  for restore, `PENDING` purchases surfaced as the `pending` result. Acknowledgement
  stays **server-side** (milestone 11 already acknowledges on validation) — the plugin
  never acknowledges, so an unvalidated purchase auto-refunds instead of being silently
  kept.
- **`moth_auth` stays pure Dart** — the adapter interface is unchanged and remains the
  escape hatch; `moth_billing` is one implementation of it, not a new coupling. Apps
  with exotic needs keep writing their own adapter.

### Example & tests

- The example app deletes its hand-written `billing_adapter.dart` and passes
  `MothStoreBilling()` to `MothApp` — the diff *is* the demo.
- Dart-side contract tests drive `MothStoreBilling` against a fake method-channel host
  (purchase happy path, cancel, pending, already-owned, restore, product-not-found),
  asserting the exact receipt payloads handed to `SubmitPurchase`.
- Go tests cover the multi-package pub listing/tarball routes (correct pubspecs, both
  packages version-stamped, unknown package 404).
- Native sources are exercised by building the example app for both platforms in the
  release checklist (documented manual pass with sandbox purchases, per milestone 13's
  acceptance path); CI asserts the embedded FS contains the plugin's native sources.

## Key design points

- **The receipt is the interface.** StoreKit 2's JWS and Play's purchase token are what
  milestone 11 verifies; the plugin's only job is to produce them faithfully and map
  store errors to the typed results. No client-side entitlement inference — the server
  stays the authority (13's rule, unchanged).
- **Version lockstep, now for native code too.** The plugin ships from `/pub` at the
  server's version, so the store-facing client code and the server's validation logic
  can never drift apart — the core moth delivery model extended to Swift/Kotlin.
- **Optional by construction.** `moth_auth` gains no plugin dependency; projects that
  never sell keep a pure-Dart SDK, and the adapter interface still admits custom
  implementations.
- **Minimal native surface.** Auto-renewing subscriptions only (11's scope): no
  consumables, no offer codes, no price-change confirmation flows — small enough to
  audit in one sitting.

## Acceptance criteria

- A fresh app with `moth_auth` + `moth_billing` from a running instance's `/pub` and
  zero adapter code completes a sandbox purchase on an iOS simulator and an Android
  emulator; the receipt validates server-side and flips `hasEntitlement('pro')`
  (milestone-13 acceptance, minus the hand-wired adapter).
- `restore()` on a fresh install re-links via `RestorePurchases`; an ask-to-buy
  deferral surfaces as `pending` and completes through the update listener.
- Contract tests cover every `MothPurchaseException` kind against the fake host;
  `flutter test` passes for both packages.
- `/pub` serves both packages with the same stamped version; `dart pub get` resolves
  `moth_billing` against a running instance.
- The example app builds for iOS and Android with the plugin wired.

## Out of scope

Consumables and one-time purchases, promotional offers / offer codes / win-back UI,
price-change confirmation sheets, Amazon/alternative stores, a Dart fallback
implementation over `in_app_purchase` (the adapter interface already covers custom
needs), and any React/web billing change (Stripe Checkout, 17/18, is already the web
path).
