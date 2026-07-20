import '../project_config.dart';

/// Bridges moth's login UI to the platform sign-in SDKs.
///
/// `moth_auth` deliberately does **not** depend on `google_sign_in` or
/// `sign_in_with_apple`, so apps that only use email/password stay light and
/// skip those packages' native setup. Apps that enable social sign-in
/// implement this interface with the package of their choice and pass it to
/// [MothApp] (or [MothLoginScreen]) — the SDK's example app ships a
/// ready-made implementation to copy. Without an adapter the provider
/// buttons still render (per project config) but explain what is missing
/// when tapped.
abstract class MothOAuthAdapter {
  /// Runs the native Google sign-in flow and returns the resulting ID
  /// token, or null when the user cancelled. [config] carries the project's
  /// public Google client IDs (from `GetProjectConfig`) so the app never
  /// hardcodes them.
  Future<MothGoogleCredential?> getGoogleIdToken(MothGoogleConfig config);

  /// Runs the native Sign in with Apple flow. Pass [hashedNonce] as the
  /// request's nonce — it is the SHA-256 of a raw nonce the SDK generated
  /// and later sends to moth for verification. Returns null when the user
  /// cancelled.
  Future<MothAppleCredential?> getAppleCredential({
    required String hashedNonce,
  });
}

/// The outcome of a native Google sign-in flow.
class MothGoogleCredential {
  const MothGoogleCredential({required this.idToken});

  final String idToken;
}

/// The outcome of a native Sign in with Apple flow. [givenName] and
/// [familyName] are only provided by Apple on the very first authorization;
/// forward them when present so moth can seed the display name.
class MothAppleCredential {
  const MothAppleCredential({
    required this.idToken,
    this.authorizationCode,
    this.givenName,
    this.familyName,
  });

  final String idToken;
  final String? authorizationCode;
  final String? givenName;
  final String? familyName;
}
