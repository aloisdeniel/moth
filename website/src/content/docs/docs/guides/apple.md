---
title: Sign in with Apple
description: Configure Apple as a sign-in provider for a project — the painful parts, automated.
---

Sign in with Apple is the most error-prone provider to configure: four
identifiers from three different screens of the Apple Developer portal, a
private key you can download exactly once, and a client secret that has
to be re-minted as a signed JWT. moth automates and verifies as much of
it as Apple's APIs allow.

- **`moth setup apple`** — drives the official App Store Connect API end
  to end and dry-runs the result against Apple's token endpoint.
  Recommended.
- **Manually** — the admin console's **Providers** tab has the same
  fields with step-by-step portal instructions.

## What Apple requires

| Value | Where it comes from | Used for |
|---|---|---|
| **Team ID** | Apple Developer membership page | client secret `iss` |
| **Bundle ID** (App ID) | Certificates, Identifiers & Profiles → Identifiers, with the *Sign in with Apple* capability enabled | native flow audience |
| **Sign in with Apple key** (`.p8` + Key ID) | Certificates, Identifiers & Profiles → Keys | minting the client secret moth needs for code exchange and token revocation |
| **Services ID** | Certificates, Identifiers & Profiles → Identifiers → Services IDs | web-redirect audience — required for **Apple sign-in on Android** and web |

The native flow on iOS needs the bundle ID, key, and Team ID. Android and
web have no native Apple flow: they go through moth's
[web-redirect fallback](#the-android--web-path), which additionally needs
the Services ID.

:::caution[The .p8 downloads exactly once]
Apple serves the Sign in with Apple private key a single time, at
creation. moth stores it encrypted (under the instance master key) the
moment it gets it — via `moth setup apple` or the Providers tab — and
never displays it again.
:::

## With the CLI

Authenticate with an App Store Connect API key (used in-process only —
moth never stores it):

```sh
moth setup apple --project bird-spotter \
  --bundle-id com.example.birdspotter \
  --team-id ABCDE12345 \
  --issuer-id 69a6de70-… \
  --key-id 2X9R4HXF34 \
  --p8 ~/keys/AuthKey_2X9R4HXF34.p8
```

Via the official API, the command verifies or creates the bundle ID,
enables the Sign in with Apple capability, and creates the Sign in with
Apple key — immediately storing the one-time `.p8` in the project's
encrypted provider config. The **Services ID** registration has no
official API (moth doesn't drive Apple's unofficial portal API), so that
one step is guided: the CLI states exactly what to create and which
return URL to paste, then validates what you enter.

It finishes by minting a real client secret from the stored key and
dry-running it against Apple's token endpoint — so "setup completed"
means *verified working*, not "probably fine". Idempotent: re-run any
time (e.g. `--rotate-key` after an expiry). Full flags in the
[command reference](../../cli/reference/#moth-setup-apple).

## Manually

In [developer.apple.com](https://developer.apple.com/account/resources/identifiers/list),
Certificates, Identifiers & Profiles:

1. **App ID** — register your bundle ID (or edit the existing one) and
   check the **Sign in with Apple** capability.
2. **Key** — create a key with *Sign in with Apple* enabled, choose your
   App ID as primary, download the `.p8` (once!), note the **Key ID**.
3. **Services ID** (for Android/web) — register a Services ID (the
   convention is `<bundle id>.signin`), enable Sign in with Apple,
   configure it with your App ID, and under **Return URLs** add exactly:

   ```
   https://auth.example.com/oauth/apple/callback
   ```

4. In the moth admin, project **Providers** tab: upload the `.p8`, fill
   Team ID, Key ID, bundle ID(s), Services ID, and enable Apple.

moth generates the ES256 client secret JWT from the stored key on demand,
caches it, and rotates it before Apple's 6-month expiry — you never
handle client secrets yourself.

## In the app

As with Google, the SDK doesn't bundle `sign_in_with_apple`; you provide
it through the [OAuth adapter](../../sdk/#social-sign-in). Two Apple
quirks the stack handles for you, worth knowing:

- **Nonce** — the SDK generates a per-attempt nonce, sends Apple its
  SHA-256 (per Apple's scheme), and moth requires the ID token's `nonce`
  claim to match. Replayed tokens are rejected.
- **The name arrives once** — Apple exposes the user's name only to the
  app and only on first authorization. The adapter forwards it; moth uses
  it solely to seed the display name (it's client-asserted and never used
  for identity).

Also handled: when a user with an Apple identity deletes their account,
moth revokes the stored Apple refresh token against Apple's
`/auth/token/revoke` — an App Store review requirement.

## The Android / web path

"Sign in with Apple" on Android uses moth's web-redirect fallback:
`/oauth/apple/start` → Apple's consent page → `/oauth/apple/callback`
(the return URL you registered on the Services ID) → redirect back into
the app via a registered custom scheme → the app exchanges the one-time
code for tokens. Register the scheme under **Linking & redirects** in the
Providers tab.

## Verify

```sh
moth doctor --project bird-spotter --apple-key ~/keys/AuthKey_….p8
```

checks the stored configuration and — with the key — performs the token
endpoint dry-run. Run it first when Apple sign-in breaks in production;
an expired key or a portal-side change shows up here with the fix
spelled out.
