# moth_auth

Flutter SDK for [moth](https://github.com/aloisdeniel/moth), the one-binary
authentication server. Served by your own moth instance at `/pub` — the SDK
version tracks your server version.

```yaml
dependencies:
  moth_auth:
    hosted: https://auth.example.com/pub
    version: ^1.0.0
```

## Quick start

```dart
import 'package:moth_auth/moth_auth.dart';

final moth = MothClient(MothConfig(
  endpoint: Uri.parse('https://auth.example.com'), // http:// works for local dev
  publishableKey: 'pk_...',
));

await moth.restore(); // resumes a persisted session, if any

final user = await moth.signIn(email: 'jane@example.com', password: '...');
```

Auth state for non-widget code (the widget layer ships separately):

```dart
moth.authStateChanges.listen((state) {
  switch (state) {
    case MothAuthLoading():
    case MothSignedOut():
    case MothSignedIn(:final user): // ...
  }
});
```

Errors are typed — catch `MothInvalidCredentials`, `MothEmailNotVerified`,
`MothWeakPassword`, `MothRateLimited`, ... (all extend `MothException`;
transport failures are `MothNetworkError`).

## Calling your own backend

`moth.accessToken()` always returns a valid, auto-refreshed JWT. For
`package:http` there is a drop-in client:

```dart
final api = authenticatedClient(moth);
final resp = await api.get(Uri.parse('https://api.example.com/todos'));
```

Your backend verifies the token against the project JWKS
(`https://auth.example.com/p/<project-slug>/.well-known/jwks.json` — the
project's setup page in the moth admin shows the exact URL).

Using dio? The SDK does not depend on it — add the equivalent yourself:

```dart
dio.interceptors.add(InterceptorsWrapper(
  onRequest: (options, handler) async {
    options.headers['authorization'] = 'Bearer ${await moth.accessToken()}';
    handler.next(options);
  },
));
```

## Sessions & tokens

- Sessions persist in the platform keystore (`flutter_secure_storage`) and
  survive restarts; pass a custom `TokenStore` to `MothClient` to override
  (e.g. `InMemoryTokenStore` in tests).
- Access tokens refresh automatically — proactively before expiry, with
  concurrent callers sharing a single refresh RPC.
- A refresh rejected by the server (revoked or reused refresh token) clears
  the stored session and emits `MothSignedOut`.

## Social sign-in

Run the native Google/Apple flow yourself (e.g. `google_sign_in`,
`sign_in_with_apple`), then exchange the ID token:

```dart
await moth.signInWithOAuth(
  provider: MothOAuthProvider.google,
  idToken: googleAuth.idToken!,
  rawNonce: nonce,
);
```

`getProjectConfig()` tells you which providers the project enables and the
Google client IDs to initialize them with.
