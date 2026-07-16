# moth_auth example

A complete moth-authenticated app: `MothApp` gates the home screen behind
the SDK's built-in login flow, `MothScope` exposes the signed-in user, and
"Call my backend" hits a sample API that verifies the moth JWT against the
project JWKS — the full loop (app → moth → app → your API).

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
