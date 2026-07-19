/// Per-project configuration baked into the package by the moth server.
///
/// This file ships with **empty placeholders** in the canonical SDK source.
/// When a project pulls its own preconfigured build from that instance's
/// per-project pub repository (`/p/<slug>/pub`), the server rewrites these
/// constants — the same stamping mechanism used for [mothSdkVersion] in
/// `version.dart` — so the package arrives with the endpoint, publishable key,
/// and full public config already inlined.
///
/// A non-empty [mothEndpoint] marks a generated build: [MothConfig.generated]
/// reads these values so `MothApp(child: ...)` runs with zero configuration
/// and zero initial network request. When empty (the canonical package pulled
/// from the generic `/pub` repository, or a source checkout), the app must
/// supply its own [MothConfig] as before.
library;

/// Base URL of the moth instance, e.g. `https://auth.example.com`. Empty in
/// the canonical package; stamped per project by the server.
const String mothEndpoint = '';

/// The project's publishable key (`pk_...`). Safe to embed; authorizes the
/// SDK's public RPCs and identifies the project. Empty in the canonical
/// package.
const String mothPublishableKey = '';

/// The project's public config — a base64-encoded `moth.auth.v1`
/// `GetProjectConfigResponse` wire message (providers, password policy,
/// sign-up state, theme, and the default-locale copy). Decoded once at
/// startup to seed the login screen and the theme/copy caches with no network
/// round-trip. Empty in the canonical package.
const String mothConfigB64 = '';

/// The project's public paywall — a base64-encoded `moth.billing.v1` `Paywall`
/// wire message. Seeds the paywall cache offline. Empty when the project has
/// no paywall or in the canonical package.
const String mothPaywallB64 = '';

/// Whether this is a server-generated, preconfigured build (a non-empty
/// [mothEndpoint]). When false the app must pass its own [MothConfig].
bool get mothIsGenerated => mothEndpoint.isNotEmpty;
