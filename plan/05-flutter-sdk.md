# Milestone 05 — Flutter SDK & Package Serving

## Goal

The headline developer experience: add one dependency served by your moth instance, wrap your app in one widget, read auth state from an `InheritedWidget`. This milestone delivers the `moth_auth` package and the pub-repository endpoint that serves it.

## Deliverables

### Package serving (`/pub`)

- Implement the minimal [pub hosted repository API](https://github.com/dart-lang/pub/blob/master/doc/repository-spec-v2.md) needed by the `pub` client:
  - `GET /pub/api/packages/moth_auth` — version listing with archive URLs.
  - `GET /pub/packages/moth_auth/versions/{v}.tar.gz` — package archive.
- The package tarball is **built at moth release time** from `sdk/flutter/` and embedded in the binary; its version tracks the moth version. Developer pubspec:
  ```yaml
  dependencies:
    moth_auth:
      hosted: https://auth.example.com/pub
      version: ^1.0.0
  ```
- The package itself is **project-agnostic**: configuration (base URL + publishable key) is passed at runtime. "Preconfigured" is achieved by the setup-instructions page (03) rendering the exact snippet with the project's values — no per-project package builds to cache-bust.

### `moth_auth` package (`sdk/flutter/`)

Pure-Dart core + Flutter layer, no Firebase-style native config files.

- **Client core** (`MothClient`): ergonomic wrapper around the Dart gRPC stubs generated from `moth/auth/v1` (`grpc` + `protobuf` packages; generated code shipped inside the package so consumers never run `protoc`). Covers signup, sign-in, refresh, me, sign-out, verification, password reset, email change, account deletion, oauth exchange. Maps gRPC status codes + `ErrorInfo` reasons from milestone 02 to typed exceptions (`MothInvalidCredentials`, `MothEmailNotVerified`, ...); exposes `Stream<MothAuthState> authStateChanges` for non-widget code; native gRPC on iOS/Android, gRPC-Web channel on Flutter Web.
- **Calling the developer's own backend** — the reason auth exists: `client.accessToken()` returns a valid (auto-refreshed) JWT, and a drop-in `http`/`dio` interceptor attaches `Authorization: Bearer ...` to the app's own API calls; the backend verifies per milestone 02. Custom claims are readable on `MothUser.claims` for client-side gating (server remains the authority).
- **Token management**: secure persistence via `flutter_secure_storage` (Keychain/Keystore); automatic access-token refresh implemented as a gRPC client interceptor with single-flight de-duplication; session survives app restarts; refresh-failure ⇒ signed-out state. The same interceptor attaches `x-moth-key` and `authorization` metadata to every call.
- **Widget layer**:
  ```dart
  void main() {
    runApp(MothApp(
      config: MothConfig(
        endpoint: Uri.parse('https://auth.example.com'),
        publishableKey: 'pk_...',
      ),
      // shown while signed out — default flow provided by the SDK:
      signedOut: const MothLoginScreen(),
      child: const MyApp(),
    ));
  }
  ```
  - `MothApp` — top-level wrapper: owns the client, restores the session, gates `child` behind authentication (configurable: `requireAuth: false` to render `child` always).
  - `MothScope` — the `InheritedWidget`: `MothScope.of(context)` → auth state (`MothAuthState.loading | signedOut | signedIn(MothUser)`), the client, and actions (`signOut()`, `refreshUser()`, `deleteAccount()` — with re-auth prompt, per the App Store requirement). Rebuilds dependents on state change.
  - `MothLoginScreen` — batteries-included login/signup flow: email/password forms with validation, forgot-password, and provider buttons (wired to milestone-04 endpoints via `google_sign_in` and `sign_in_with_apple` when enabled in the project's public config). Restyled by the design system in 06; plain Material defaults for now.
- **Public project config** (`moth.auth.v1.ConfigService.GetProjectConfig`, publishable-key authed): enabled providers, Google client IDs, password policy, (later) theme — so the login screen adapts without app releases.

### Example & tests

- `sdk/flutter/example/` — runnable app against a local moth, including a call to a tiny sample backend route that verifies the JWT via the project JWKS — demonstrating the full loop (app → moth → app → developer API).
- Dart unit tests (client, token refresh single-flight, error mapping) + widget tests (`MothScope` rebuilds, login form validation) with an in-process fake gRPC server; integration test that runs against a real moth binary spawned by the test harness.
- CI job with Flutter SDK: `dart analyze`, `flutter test`, stale-codegen check for the Dart stubs, package `dry-run` publish validation, tarball build reproducibility.

## Key design points

- **Time-to-first-login is the metric** — from "project created in admin" to "logged in on simulator" should be under 10 minutes; the acceptance test below walks it literally.
- **No codegen, no native setup for email/password** — Google/Apple need their usual platform setup (URL schemes, entitlements); the setup-instructions page documents exactly those steps per project.
- **Version coupling** — SDK major version matches server major version; `buf breaking` in CI guarantees wire compatibility within a major. The server sends `x-moth-version` response metadata and the SDK warns on mismatch in debug builds.
- **Pub serving stays HTTP** — the `dart pub` client dictates the repository protocol; `/pub/*` is the one developer-facing surface that can't be gRPC.

## Acceptance criteria

- `flutter pub get` against a running moth instance resolves and downloads `moth_auth` from `/pub`.
- Example app: signup → email verification → login → hot restart keeps session → sign out; Google login works on Android emulator, Apple login on iOS simulator (with milestone-04 test credentials).
- Access-token expiry mid-session refreshes transparently (integration test with 5-second TTL project).
- `MothScope.of(context)` state transitions covered by widget tests.

## Out of scope

Themed login UI (06 — this milestone ships default Material styling), analytics client events (07), offline support beyond persisted session.
