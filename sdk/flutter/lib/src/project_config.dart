/// A project's public, non-secret configuration
/// (`moth.auth.v1.ConfigService.GetProjectConfig`), so login UI can render
/// exactly the sign-in methods the project enables without an app release.
class MothProjectConfig {
  const MothProjectConfig({
    required this.google,
    required this.apple,
    required this.passwordMinLength,
    required this.signUpOpen,
  });

  final MothGoogleConfig google;
  final MothAppleConfig apple;

  /// Minimum accepted password length.
  final int passwordMinLength;

  /// Whether the public sign-up RPC is open.
  final bool signUpOpen;
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
