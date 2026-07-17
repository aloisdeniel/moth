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

## Localization

The built-in screens (`MothLoginScreen` sign-in and sign-up, `MothPaywallScreen`)
speak the user's language. The SDK resolves the device locale
(`PlatformDispatcher.locale`), sends it as the `x-moth-language` header on every
call, and renders the copy the server negotiates back for that locale — the
project's admin-customized wording when the instance has it, else moth's bundled
translations. Pin a fixed language (ignoring the device) with
`MothConfig(locale: Locale('fr'))`.

Copy resolves **server override → bundled → English**: the SDK bundles the same
curated locale set as the server (`en`, `fr`, `de`, `es`, `pt`, `it`, `ja`), so
the screens render fully localized **before** the config arrives and **offline** —
the bundle is the floor, the project's copy the ceiling. A locale neither side has
falls back to English. The delivered copy is cached on device with
stale-while-revalidate keyed by `(locale, revision)` (the same discipline as the
theme and paywall caches), so a launch shows the right language instantly and an
operator's copy edit in the admin lands on the next background refresh — no app
release. Changing the device language refetches automatically.

Bundled copy interpolates an `{app}` placeholder from `MothConfig(appName: ...)`
(the server fills its own project name into the copy it delivers):

```dart
MothApp(
  config: MothConfig(
    endpoint: Uri.parse('https://auth.example.com'),
    publishableKey: 'pk_...',
    appName: 'Aurora',        // fills {app} in the bundled fallback strings
    // locale: Locale('fr'),  // optional: pin a language, else follow device
  ),
  child: const MyApp(),
);
```

`MothApp` installs the moth localization delegates on the shell it wraps its own
screens in, so a drop-in `MothApp` gets correct `MaterialLocalizations` for every
bundled language with no extra setup. An app with its own `MaterialApp` that wants
the same coverage spreads them in alongside its own:

```dart
MaterialApp(
  localizationsDelegates: const [
    ...mothLocalizationsDelegates,
    // your app's own delegates…
  ],
  supportedLocales: mothSupportedLocales,
);
```

The composable pieces read the resolved copy from an ambient `MothCopyScope`
(inserted by `MothApp` and by each screen); wrap a custom screen built from
`MothEmailForm` / `MothPurchaseButton` in one to localize them too. Read a single
string yourself with `MothCopyScope.of(context).value('sign_in.title')`.
