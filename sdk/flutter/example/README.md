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

## Localization

Tap the **language pill** in the top-right corner to cycle the moth screens
through Device → English → Français → Deutsch → 日本語. The login screen (and
the paywall behind "Premium feature") re-render in the chosen language on the
fly: the project's admin-customized copy for that locale when the running
instance has it, the SDK's bundled translations otherwise — including offline
and before the config loads. Under the hood the switcher sets
`MothConfig(locale:)`; leaving it on **Device** follows the OS language. Edit a
project's copy for, say, `de` in the admin (milestone 15) and it appears here on
the next launch — no app rebuild. `appName: 'moth example'` fills the `{app}`
placeholder in the bundled fallback strings.

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

Native billing comes from `moth_billing`, moth's first-party plugin
(StoreKit 2 on iOS, the Play Billing Library on Android; a dependency of
this example only, not of `moth_auth`) — `main.dart` passes
`MothStoreBilling()` to `MothApp` and there is no adapter code to write.
It runs the native purchase; the SDK forwards the resulting receipt to
moth's `SubmitPurchase` for server-side validation. Apps with exotic store
needs can still implement `MothBillingAdapter` themselves, mirroring the
OAuth adapter in `lib/oauth_adapter.dart`.

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
never blocks. Store purchases need the usual platform setup (a StoreKit
configuration or sandbox products on iOS, a Play Billing test track on
Android) before real purchases work; without it the purchase surfaces a
typed error rather than crashing.

## Push notifications

The "Push notifications" card on the home screen is the milestone-21 loop
end to end: `main.dart` passes `MothNativePush()` (from `moth_push`, moth's
first-party APNs/FCM plugin — a dependency of this example only, not of
`moth_auth`) to `MothApp`, and the card drives the toggle off
`MothScope.of(context).pushStatus` and `requestPushPermission()` — the only
way the SDK ever shows the OS prompt.

To see a device row appear in the admin:

1. Enable **push registration** in the project's Push tab (the card
   shows "Unavailable" until you do).
2. Run the app on a device or simulator, sign in, and flip the toggle to
   grant permission.
3. Open the user in the admin's Users tab — the **Devices** panel shows the
   registration (target `apns`/`fcm`, model, permission, last seen).
   Signing out in the app revokes it (`signed_out`).

Platform caveats, same as any real app:

- **Android** — FCM needs this app's own Firebase config: create a Firebase
  project, add an Android app with the example's application id, and drop
  `google-services.json` into `android/app/`. This is the one piece of
  setup moth cannot absorb (the Firebase project is your sender identity);
  without it registration fails with an actionable
  `firebase-not-initialized` debug log while auth keeps working.
- **iOS** — add the Push Notifications capability; the iOS **simulator**
  has no real APNs, so expect a registration on device only.

Even a device that denies permission registers (flagged `denied`); sending
stays your backend's job — read the registry via
`moth.server.v1.PushService` with the project secret key.
