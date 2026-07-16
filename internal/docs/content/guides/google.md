# Sign in with Google

Sign in with Google needs OAuth client IDs created in the Google Cloud
console — one per platform — and those IDs written into the moth project.
Two ways to get there:

- **`moth setup google`** — the CLI orchestrates the whole thing and
  verifies the result. Recommended.
- **Manually** — the admin console's **Providers** tab has the same
  fields, with inline instructions and the exact values to paste.

Either way, the moth side is per-project: another app on the same
instance configures Google independently, with its own client IDs.

## How it works

The Flutter app runs Google's native sign-in flow itself (via an
[OAuth adapter](../../sdk/#social-sign-in), e.g. `google_sign_in`) and
hands the resulting **ID token** to moth. moth verifies it server-side —
signature against Google's JWKS, issuer, expiry, a per-attempt nonce, and
`aud` against the client IDs you configure here — then signs the user in,
links the identity to an existing account with the same provider-verified
email (project setting `auto_link_verified_email`, on by default), or
creates a new user. Nothing from the client is trusted except the
verified token.

Client IDs are public values; the SDK fetches them at runtime from
`GetProjectConfig`, so apps never hardcode them. The one secret — the web
client secret, used only by the web-redirect fallback — is stored
encrypted and never shown again.

## With the CLI

```sh
moth login https://auth.example.com      # once; asks for a personal access token
moth setup google --project bird-spotter \
  --gcp-project bird-spotter-prod \
  --ios-bundle-id com.example.birdspotter \
  --android-package com.example.birdspotter \
  --keystore ~/keys/upload.jks
```

The command automates what Google's APIs expose (verifying the GCP
project via `gcloud` when installed, computing Android signing
fingerprints from the keystore with `keytool`) and **guides** what they
don't: OAuth client creation has no public API, so the CLI opens the
exact console page, says precisely what to click and paste back, and
validates each pasted value's shape. It then writes the client IDs into
the moth project and verifies each one against Google's endpoints,
ending with a checklist.

Re-running is safe: it diffs current state and only changes what's
needed. Already have client IDs? Pass `--web-client-id`,
`--ios-client-id`, `--android-client-id` to skip the guided steps — see
the [command reference](../../cli/reference/#moth-setup-google).

## Manually

In the [Google Cloud console](https://console.cloud.google.com/), with an
OAuth consent screen configured for your GCP project, create OAuth client
IDs under **APIs & Services → Credentials**:

1. **Web application** — under **Authorized redirect URIs**, add exactly:

   ```
   https://auth.example.com/oauth/google/callback
   ```

   The web client ID doubles as the server-side audience for the redirect
   fallback flow. Copy the client ID *and* client secret (the secret is
   needed only for that fallback).

2. **iOS** — enter your app's bundle ID. Copy the client ID.

3. **Android** — enter the application ID and the SHA-1 fingerprint of
   your signing certificate (`keytool -list -v -keystore …`; for Play App
   Signing, the fingerprint shown in the Play console). Copy the client
   ID.

Then, in the moth admin, open the project's **Providers** tab, paste the
client IDs (and the web client secret), and enable Google. The change is
live immediately: `GetProjectConfig` advertises Google, and the SDK's
login screen shows the button on next launch — no app release.

## In the app

`moth_auth` deliberately doesn't depend on `google_sign_in`; you plug it
in through a small adapter passed to `MothApp`. See
[Flutter SDK reference — social sign-in](../../sdk/#social-sign-in) for
the interface and a complete example. Android and iOS also need
`google_sign_in`'s usual platform setup (URL scheme on iOS); the
project's Setup tab lists the exact steps with your values.

## Verify

```sh
moth doctor --project bird-spotter
```

checks the stored client IDs against Google's live endpoints and the
redirect URI registration, and tells you exactly what's wrong when
sign-in stops working (deleted client, changed fingerprint, …).

## Web-redirect fallback

Platforms without a native flow (and "Sign in with Apple on Android",
see [the Apple guide](../apple/)) use a browser round-trip:
`/oauth/google/start` → consent → `/oauth/google/callback` → redirect
back into the app via a custom URL scheme, which the app exchanges for
tokens. Register the scheme(s) under **Linking & redirects** in the
Providers tab — callbacks only ever redirect to schemes on that list.
