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
    this.configCacheTtl = const Duration(hours: 1),
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

  /// How long the on-device config caches (theme, paywall, localized copy)
  /// are served without any network revalidation — download once, then
  /// launch quietly until the TTL expires.
  ///
  /// Every cached config payload records when it was fetched or last
  /// revalidated. While that moment is younger than this TTL, a launch
  /// renders straight from the cache and performs **zero** config RPCs.
  /// Once it expires, the SDK revalidates cheaply on the next launch —
  /// echoing the cached revision so the server omits an unchanged body —
  /// and the window restarts (an omitted-body match also refreshes it).
  ///
  /// Defaults to one hour. Use [Duration.zero] to revalidate on every
  /// launch. Explicit refresh calls (e.g. [MothThemeController.refresh],
  /// [MothCopyController.refresh]) always hit the server regardless.
  final Duration configCacheTtl;
}
