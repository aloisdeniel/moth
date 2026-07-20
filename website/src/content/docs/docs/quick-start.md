---
title: Quick start
description: From an empty server to a logged-in app in ten minutes.
---

The ten-minute path: run the binary, create a project in the admin, add
`moth_auth` to your Flutter app, wrap it in one widget, sign in. (Building
for the web instead? The same three steps with npm and `@moth/react` — the
snippets below have direct equivalents in the
[React SDK reference](../react/), and the Setup tab renders both.)

Every snippet below is also rendered — with your project's real values
already filled in — on the project's **Setup** tab in the admin console.
This page and that tab are kept deliberately identical; when in doubt, the
Setup tab is the copy-paste source of truth for *your* instance.

## 1 · Get the binary

Prebuilt binaries, Homebrew, and an official Docker image ship with v1.0.
Until then, build from source (Go 1.25+):

```sh
git clone https://github.com/aloisdeniel/moth.git
cd moth
make build   # → bin/moth
```

## 2 · Start the server

```sh
./bin/moth serve --data-dir ./data
```

That's the whole deployment for local development: moth creates the data
directory (SQLite database, key material, uploads) on first start and
listens on `:8080`. Outgoing email defaults to a **console transport** —
verification and reset emails are printed to the server log, so the full
flow works with zero SMTP setup.

Open [http://localhost:8080/admin](http://localhost:8080/admin). The
first-run screen asks you to create the first admin account, then signs
you in. (Prefer the terminal? `moth admin create --email you@example.com`
does the same on the server host.)

## 3 · Create a project

In the admin console, **Create project** and give it a name — say
`Bird Spotter`. A project is one mobile app: it gets its own users, its
own ES256 signing keypair, a publishable key (`pk_…`, safe to embed in the
app) and a secret key (`sk_…`, for [your backend](#6--verify-tokens-on-your-backend)
— shown exactly once).

Open the project's **Setup** tab. Everything from here on is printed there
with your real values.

## 4 · Add the SDK to your Flutter app

Your moth instance serves the `moth_auth` package from its own pub
repository at `/pub` — the SDK version always matches the server version,
and nothing is fetched from pub.dev.

```yaml title="pubspec.yaml"
dependencies:
  moth_auth:
    hosted: http://localhost:8080/pub
    version: ^1.0.0
```

:::note
Running a dev build (built from source, unversioned)? It serves a
pre-release version like `0.0.0-dev.1`, and Dart version ranges never
match pre-releases — pin it exactly (`version: 0.0.0-dev.1`). The Setup
tab prints the exact constraint your instance serves.
:::

Then:

```sh
flutter pub get
```

## 5 · Wrap your app

`MothApp` owns the client, restores any persisted session, and gates your
app behind authentication: signed out, it shows the SDK's built-in
`MothLoginScreen`; signed in, it shows `child`. Replace the publishable
key with yours from the Setup tab.

```dart title="lib/main.dart"
import 'package:flutter/material.dart';
import 'package:moth_auth/moth_auth.dart';

void main() {
  runApp(
    MothApp(
      config: MothConfig(
        endpoint: Uri.parse('http://localhost:8080'),
        publishableKey: 'pk_YOUR_PUBLISHABLE_KEY',
      ),
      // Signed out -> the SDK's built-in MothLoginScreen; signed in -> child.
      child: const MyApp(),
    ),
  );
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    final user = MothScope.of(context).user;
    return MaterialApp(
      home: Scaffold(
        body: Center(child: Text('Signed in as ${user?.email}')),
      ),
    );
  }
}
```

Run it:

```sh
flutter run
```

You'll see the login screen. **Create an account** with any email and
password — by default new projects allow open signup and don't require
email verification before sign-in (both are project settings). You're in:
`MothScope.of(context).user` is your account, and the session survives
restarts (it's stored in the platform keystore).

:::caution[Talking to localhost from a device]
- **Android emulator**: the host machine is `10.0.2.2`, not `localhost` —
  use `Uri.parse('http://10.0.2.2:8080')`. Android also blocks cleartext
  HTTP by default; for local development add
  `android:usesCleartextTraffic="true"` to the `<application>` element of
  your debug manifest.
- **iOS simulator**: `localhost` works as-is.
- A real device needs your machine's LAN address — and a real deployment
  needs HTTPS ([installation & deployment](../installation/)).
:::

## 6 · Verify tokens on your backend

The point of auth: your own API trusts the app's requests. On the client,
the SDK attaches an auto-refreshed JWT for you:

```dart
final api = authenticatedClient(moth); // package:http drop-in
final resp = await api.get(Uri.parse('https://api.example.com/todos'));
```

On the server, verify the token offline against your project's JWKS with
any standard JWT library — no call to moth per request:

```
JWKS  http://localhost:8080/p/<project-slug>/.well-known/jwks.json
iss   http://localhost:8080/p/<project-slug>
aud   <project-slug>
alg   ES256
```

The Setup tab prints ready-made verifier snippets for Node, Go, and Dart
with these exact values, and the repository ships a complete ~200-line
example backend (`scripts/example_backend/`) demonstrating the loop. See
the [API reference](../api/#verifying-tokens-on-your-backend) for the claims
contract and online introspection.

## Where next

- [Sign in with Google](../guides/google/) and
  [Sign in with Apple](../guides/apple/) — one command each with the CLI.
- [Theme the login screen](../guides/theming/) to match your app's brand.
- [Deploy it for real](../installation/) — systemd, reverse proxy, TLS.
- [Flutter SDK reference](../sdk/) — everything beyond the built-in flow.
- [React SDK reference](../react/) — the same developer experience for the
  web, served from your instance's npm registry.
- [Sell subscriptions](../guides/monetization/) — App Store, Google Play,
  and Stripe on the web, one entitlement model.
