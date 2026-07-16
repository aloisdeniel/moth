---
title: Flutter SDK reference
description: The moth_auth package — MothApp, MothScope, MothLoginScreen, and the full MothClient API.
---

`moth_auth` is a pure-Dart core plus a thin Flutter layer. It is served
from your own instance's pub repository, so the SDK version always tracks
the server version and nothing is fetched from pub.dev:

```yaml title="pubspec.yaml"
dependencies:
  moth_auth:
    hosted: https://auth.example.com/pub
    version: ^1.0.0
```

The package is **project-agnostic** — endpoint and publishable key are
passed at runtime. The project's **Setup** tab in the admin console
renders every snippet below with your real values already filled in.

There are two ways to use it: the **widget layer** (`MothApp` /
`MothScope` / `MothLoginScreen`) for a batteries-included flow, and the
**client core** (`MothClient`) for full control or non-widget code. The
widget layer is built on the client core; you can mix them.

## Widget layer

### MothApp

The top-level wrapper. It owns a `MothClient`, restores any persisted
session on startup, and gates `child` behind authentication:

```dart title="lib/main.dart"
void main() {
  runApp(
    MothApp(
      config: MothConfig(
        endpoint: Uri.parse('https://auth.example.com'),
        publishableKey: 'pk_...',
      ),
      child: const MyApp(),
    ),
  );
}
```

- `config` — a `MothConfig(endpoint:, publishableKey:)`. `http://` URLs
  work for local development.
- `child` — shown once signed in.
- `signedOut` — the widget shown while signed out. Defaults to the
  built-in [`MothLoginScreen`](#mothloginscreen); pass your own to replace
  it.
- `requireAuth` — set `false` to always render `child` and let the app
  decide when to present sign-in (e.g. guest-first apps). Defaults to
  `true`.
- `theme` — a local [`MothTheme`](#theming-hooks) overriding the theme the
  server sends.

While the session is being restored `MothApp` shows a neutral loading
state; it never flashes the login screen for a user who is actually
signed in.

### MothScope

The `InheritedWidget` that exposes auth state to the tree.
`MothScope.of(context)` gives you:

- `state` — a [`MothAuthState`](#auth-state): `loading`, `signedOut`, or
  `signedIn(MothUser)`.
- `user` — the current `MothUser`, or `null` when not signed in
  (shorthand for the `signedIn` case).
- `client` — the underlying [`MothClient`](#client-core) for any call not
  surfaced as an action.
- Actions: `signOut()`, `refreshUser()`, and `deleteAccount()` (which runs
  the re-authentication prompt the App Store requires for account
  deletion).

Dependents rebuild when the auth state changes:

```dart
final user = MothScope.of(context).user;
return Text(user == null ? 'Signed out' : 'Signed in as ${user.email}');
```

### MothLoginScreen

The default sign-in surface: email/password sign-in and sign-up with
validation, forgot-password, and — when the project enables them —
Google/Apple buttons. It reads the project's public config
(`getProjectConfig`) to show only the providers that are turned on, and
renders the project's [theme](../guides/theming/). No wiring required
beyond `MothApp`; it is what `signedOut` defaults to.

To use it outside `MothApp`, construct it directly. To restyle it, pass a
`theme:`; to rebuild it from parts, see [theming hooks](#theming-hooks).

## Client core

`MothClient` is an ergonomic wrapper over the generated `moth.auth.v1`
gRPC stubs (native gRPC on iOS/Android, gRPC-Web on Flutter Web). Use it
directly in non-widget code, tests, or a custom UI.

```dart
final moth = MothClient(MothConfig(
  endpoint: Uri.parse('https://auth.example.com'),
  publishableKey: 'pk_...',
));

await moth.restore(); // resume a persisted session, if any

final user = await moth.signIn(email: 'jane@example.com', password: '…');
```

Methods map one-to-one to the [auth API](../api/#mothauthv1):

| Area | Methods |
|---|---|
| Session | `restore()`, `signIn(email:, password:)`, `signUp(email:, password:, displayName:)`, `signOut({allDevices})` |
| Current user | `getMe()` / `refresh()`, `updateMe(displayName:)`, `changePassword(current:, next:)` |
| Email | `requestEmailVerification()`, `requestEmailChange(newEmail:)` |
| Password reset | `requestPasswordReset(email:)` |
| Social | `signInWithOAuth(...)`, `unlinkIdentity(provider:)` |
| Account | `deleteAccount(...)` (fresh re-auth required) |
| Config | `getProjectConfig()` |
| Tokens | `accessToken()` |

The confirmation half of email verification, password reset, and email
change is completed from the [hosted pages](../api/#hosted-pages) moth
emails the user — the app requests them, the link finishes them.

### Auth state

`authStateChanges` is a broadcast `Stream<MothAuthState>` for code that
lives outside the widget tree. The state is a sealed type:

```dart
moth.authStateChanges.listen((state) {
  switch (state) {
    case MothAuthLoading():   // restoring or refreshing
    case MothSignedOut():     // no valid session
    case MothSignedIn(:final user): // user is a MothUser
  }
});
```

`MothUser` carries `id`, `email`, `emailVerified`, `displayName`, and
`claims` — the project-assigned custom claims, readable for client-side
gating (the server remains the authority; see
[custom claims](../api/#custom-claims)).

### Errors

Every failure is a typed subclass of `MothException`, mapped from the
server's gRPC status and stable `ErrorInfo` reason
([error model](../api/#errors)):

```dart
try {
  await moth.signIn(email: e, password: p);
} on MothInvalidCredentials {
  // wrong email or password (uniform — never reveals which)
} on MothEmailNotVerified {
  // project requires verification before sign-in
} on MothRateLimited {
  // too many attempts; back off
} on MothNetworkError {
  // transport failure, not an auth decision
}
```

Others include `MothWeakPassword`, `MothEmailAlreadyExists`, and
`MothSignUpClosed`. Catch `MothException` for the catch-all.

## Calling your own backend

The reason auth exists: your API trusts the app's requests.
`moth.accessToken()` always returns a valid, auto-refreshed JWT. For
`package:http` there is a drop-in client that attaches it:

```dart
final api = authenticatedClient(moth);
final resp = await api.get(Uri.parse('https://api.example.com/todos'));
```

Your backend verifies that token offline against the project JWKS — see
[verifying tokens on your backend](../api/#verifying-tokens-on-your-backend).

The SDK does not depend on `dio`; add the equivalent interceptor yourself:

```dart
dio.interceptors.add(InterceptorsWrapper(
  onRequest: (options, handler) async {
    options.headers['authorization'] = 'Bearer ${await moth.accessToken()}';
    handler.next(options);
  },
));
```

## Sessions & tokens

- **Persistence** — sessions are stored in the platform keystore
  (`flutter_secure_storage`, i.e. Keychain / Keystore) and survive app
  restarts. Pass a custom `TokenStore` to `MothClient` to override — e.g.
  `InMemoryTokenStore` in tests.
- **Automatic refresh** — access tokens refresh proactively before expiry,
  implemented as a gRPC interceptor with single-flight de-duplication, so
  concurrent callers share one refresh RPC.
- **Rotation & theft detection** — refresh tokens rotate on every use; a
  token rejected as revoked or reused clears the stored session and emits
  `MothSignedOut`.
- **Version coupling** — the SDK major version matches the server major
  version. The server sends `x-moth-version` response metadata; the SDK
  warns on mismatch in debug builds.

## Social sign-in

`moth_auth` deliberately does **not** depend on `google_sign_in` or
`sign_in_with_apple` — you run the native provider flow yourself and hand
moth the resulting ID token. This keeps the package small and lets you
pick the plugin versions your app already uses.

```dart
// After the native Google flow yields an ID token and you generated a nonce:
await moth.signInWithOAuth(
  provider: MothOAuthProvider.google,
  idToken: googleAuth.idToken!,
  rawNonce: nonce,
);
```

`getProjectConfig()` tells you which providers the project enables and the
Google client IDs to initialize them with, so the app never hardcodes
them. Apple additionally passes its one-time `authorizationCode` and, on
first authorization, the user's name. Provider-specific setup lives in the
[Google](../guides/google/) and [Apple](../guides/apple/) guides.

## Theming hooks

`MothLoginScreen` consumes the project's [theme](../guides/theming/)
exclusively — no hardcoded styles. When the built-in screen isn't enough:

- **Override the server theme** — `MothLoginScreen(theme: myTheme)` or
  `MothApp(theme: myTheme)` with a local `MothTheme`.
- **Build your own screen from themed parts** — `MothEmailForm`,
  `MothProviderButtons`, and `MothLogo` are exported and pick up the same
  theme, so a custom layout still matches the brand.
- **Error-state colors are fixed** under any theme — the legibility of a
  failure message is not themable.

## Example app

`sdk/flutter/example/` in the repository is a runnable app against a local
moth instance, including a "Call my backend" button that hits the
[example backend](../api/#verifying-tokens-on-your-backend) with an
auto-refreshed token — the full loop, app → moth → app → your API.
