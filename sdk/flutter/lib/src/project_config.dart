import 'copy.dart';
import 'theme.dart';

/// A project's public, non-secret configuration
/// (`moth.auth.v1.ConfigService.GetProjectConfig`), so login UI can render
/// exactly the sign-in methods the project enables without an app release.
class MothProjectConfig {
  const MothProjectConfig({
    required this.google,
    required this.apple,
    required this.passwordMinLength,
    required this.signUpOpen,
    this.theme,
    this.copy,
  });

  final MothGoogleConfig google;
  final MothAppleConfig apple;

  /// Minimum accepted password length.
  final int passwordMinLength;

  /// Whether the public sign-up RPC is open.
  final bool signUpOpen;

  /// The project's design system, or null when the server confirmed the
  /// `knownThemeRevision` passed to `getProjectConfig` is still current
  /// (keep the cached theme) — or when the server predates themes.
  final MothTheme? theme;

  /// The localized copy for the negotiated locale (locale + revision always
  /// present, `messages` present only when the `knownCopyRevision` differed),
  /// or null when the server predates localized copy. Consumed by
  /// [MothCopyController].
  final MothCopyUpdate? copy;
}

/// Public part of the project's Sign in with Google configuration. Client
/// IDs are public values; the secret never leaves the server.
class MothGoogleConfig {
  const MothGoogleConfig({
    required this.enabled,
    this.webClientId,
    this.iosClientId,
    this.androidClientId,
  });

  final bool enabled;
  final String? webClientId;
  final String? iosClientId;
  final String? androidClientId;
}

/// Public part of the project's Sign in with Apple configuration.
class MothAppleConfig {
  const MothAppleConfig({required this.enabled});

  final bool enabled;
}
