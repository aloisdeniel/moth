// Wires moth's login screen to the real platform sign-in SDKs. The
// moth_auth package deliberately has no dependency on google_sign_in or
// sign_in_with_apple — this file is the glue an app copies and adapts.
import 'package:flutter/foundation.dart';
import 'package:google_sign_in/google_sign_in.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:sign_in_with_apple/sign_in_with_apple.dart';

/// [MothOAuthAdapter] backed by `google_sign_in` and `sign_in_with_apple`.
///
/// Both flows need their usual native setup (the project's setup page in
/// the moth admin lists the exact steps); until that is done the flows
/// throw and the login screen surfaces the message instead of crashing.
class ExampleOAuthAdapter implements MothOAuthAdapter {
  bool _googleInitialized = false;

  @override
  Future<MothGoogleCredential?> getGoogleIdToken(
    MothGoogleConfig config,
  ) async {
    final signIn = GoogleSignIn.instance;
    if (!_googleInitialized) {
      // The client IDs come from the project's public config, so the app
      // never hardcodes them: the iOS client identifies this app, the web
      // client is the audience moth accepts ID tokens for.
      await signIn.initialize(
        clientId: defaultTargetPlatform == TargetPlatform.iOS
            ? config.iosClientId
            : null,
        serverClientId: config.webClientId,
      );
      _googleInitialized = true;
    }
    try {
      final account = await signIn.authenticate();
      final idToken = account.authentication.idToken;
      if (idToken == null) {
        throw MothInvalidProviderToken(
          'Google returned no ID token — check the client IDs in the '
          'project settings and the platform setup.',
        );
      }
      return MothGoogleCredential(idToken: idToken);
    } on GoogleSignInException catch (err) {
      if (err.code == GoogleSignInExceptionCode.canceled) return null;
      throw MothInvalidProviderToken('Google sign-in failed: ${err.code}');
    }
  }

  @override
  Future<MothAppleCredential?> getAppleCredential({
    required String hashedNonce,
  }) async {
    try {
      final credential = await SignInWithApple.getAppleIDCredential(
        scopes: [
          AppleIDAuthorizationScopes.email,
          AppleIDAuthorizationScopes.fullName,
        ],
        // moth verifies that the ID token's nonce matches the raw nonce the
        // SDK generated; Apple embeds this (hashed) value in the token.
        nonce: hashedNonce,
      );
      final idToken = credential.identityToken;
      if (idToken == null) {
        throw MothInvalidProviderToken('Apple returned no identity token.');
      }
      return MothAppleCredential(
        idToken: idToken,
        authorizationCode: credential.authorizationCode,
        // Only present on the very first authorization; moth uses them to
        // seed the display name.
        givenName: credential.givenName,
        familyName: credential.familyName,
      );
    } on SignInWithAppleAuthorizationException catch (err) {
      if (err.code == AuthorizationErrorCode.canceled) return null;
      throw MothInvalidProviderToken('Apple sign-in failed: ${err.message}');
    }
  }
}
