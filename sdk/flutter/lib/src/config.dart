import 'dart:ui';

/// Connection settings for a moth project.
///
/// The values to paste here are shown on the project's setup-instructions
/// page in the moth admin.
class MothConfig {
  const MothConfig({
    required this.endpoint,
    required this.publishableKey,
    this.locale,
    this.appName,
  });

  /// Base URL of the moth server, e.g. `https://auth.example.com`.
  ///
  /// TLS follows the scheme: `https` uses secure transport, plain `http` is
  /// supported for local development (e.g. `http://localhost:8080`).
  final Uri endpoint;

  /// The project's publishable key (`pk_...`), attached to every call as
  /// `x-moth-key` metadata. Safe to embed in the app.
  final String publishableKey;

  /// Overrides the device locale for language negotiation and localized copy.
  /// Leave null (the default) to follow the device language
  /// ([PlatformDispatcher.locale]); set it to pin the moth screens to a fixed
  /// language regardless of the device setting. Sent as the `x-moth-language`
  /// header (a BCP-47 tag) on every call.
  final Locale? locale;

  /// The app's display name, substituted for the `{app}` placeholder in the
  /// SDK's bundled fallback copy (e.g. `sign_in.subtitle` → "Welcome back to
  /// {app}."). Only used offline / before the first `GetProjectConfig`: the
  /// server already interpolates its own project name into the copy it
  /// delivers, so this is the localization floor's stand-in. Leave null to
  /// render the placeholder empty.
  final String? appName;
}
