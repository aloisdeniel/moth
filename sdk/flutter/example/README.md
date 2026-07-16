# moth_auth example

A complete moth-authenticated app: `MothApp` gates the home screen behind
the SDK's built-in login flow, `MothScope` exposes the signed-in user, and
"Call my backend" hits a sample API that verifies the moth JWT against the
project JWKS — the full loop (app → moth → app → your API). The home screen
also reads the subscription state and gates a "premium feature" behind the
`pro` entitlement, showing the batteries-included `MothPaywallScreen`.

## Run it against a local moth

1. **Start moth** (from the repository root):

   ```sh
   make run
   ```

2. **Create a project.** Open the printed `/admin?setup=...` link (first
   run) or `http://localhost:8080/admin`, create a project, and copy its
   publishable key (`pk_...`) from the setup page.

3. **Run the app** (iOS simulator, Android emulator or desktop):

   ```sh
   cd sdk/flutter/example
   flutter run --dart-define=MOTH_PUBLISHABLE_KEY=pk_...
   ```

   The moth endpoint defaults to `http://localhost:8080`; override it with
   `--dart-define=MOTH_ENDPOINT=...`. On the Android emulator the app
   automatically swaps `localhost` for `10.0.2.2` (the host machine).

4. **Sign up** in the app. With the dev config, verification and reset
   emails are printed to the moth console. Toggle "require email
   verification", password policy and providers in the project settings —
   the login screen adapts via the project's public config.

## The sample backend

"Call my backend" expects the tiny JWT-verifying API from
`scripts/example_backend`:

```sh
go run ./scripts/example_backend --issuer http://localhost:8080/p/<slug>
```

`<slug>` is the project slug shown in the admin. The backend fetches the
project JWKS once, verifies the `Authorization: Bearer` token of each
request (signature, expiry, issuer) and echoes the verified identity.
Override the app's backend URL with `--dart-define=API_BASE=...`.

## Google / Apple sign-in

The provider buttons appear as soon as the provider is enabled in the
project settings. This app wires them to the native SDKs in
`lib/oauth_adapter.dart` (an implementation of `MothOAuthAdapter` you can
copy) — `google_sign_in` and `sign_in_with_apple` are dependencies of this
example only, not of `moth_auth`.

Both need their usual platform setup (the project's setup page in the moth
admin lists the steps): Google needs the iOS URL scheme / Android SHA-1
registration for the client IDs configured in the project; Apple needs the
"Sign in with Apple" capability. Until then the buttons explain what is
missing instead of crashing.

## Subscriptions & the paywall

The subscription card on the home screen shows the current entitlement
(`MothScope.of(context).hasEntitlement('pro')`) and opens a screen gated
behind `pro`: a free user sees the themed `MothPaywallScreen` (the project's
default offering + admin-configured copy), and the moment a purchase flips
the entitlement the gate hands them through to the unlocked content — no app
restart. A hot restart keeps the entitlement (cached and revalidated in the
background), and "Restore purchases" re-links a prior purchase on a fresh
install.

Native billing lives in `lib/billing_adapter.dart`, an implementation of
`MothBillingAdapter` backed by `in_app_purchase` (a dependency of this
example only, not of `moth_auth`, mirroring the OAuth adapter). It runs the
StoreKit / Play Billing purchase; the SDK forwards the resulting receipt to
moth's `SubmitPurchase` for server-side validation.

To try a sandbox purchase you need milestone-11/12 store credentials
configured for the project and the tier(s) defined in the admin catalog:

1. Define a product in the moth admin and map it to a store SKU (App Store
   Connect / Play Console) that exists as a sandbox subscription.
2. Configure the store server credentials in the project's billing settings
   so moth can validate receipts.
3. Sign in on a sandbox account (iOS Sandbox tester / Play internal test
   track), open "Premium feature" and complete the purchase.

A project with **no** products still runs: the paywall renders a graceful
"nothing to purchase" state and gating on an entitlement no product grants
never blocks. `in_app_purchase` needs its usual platform setup (StoreKit
configuration / Play Billing) before real purchases work; without it the
purchase surfaces a typed error rather than crashing.

> Caveat: `in_app_purchase` surfaces StoreKit 1 receipts on iOS, while moth's
> `SubmitPurchase` expects a StoreKit 2 signed transaction (JWS). The wiring
> is complete on Android; an iOS production app should forward a StoreKit 2
> transaction. See the note in `lib/billing_adapter.dart`.
