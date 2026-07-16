import 'dart:convert';
import 'dart:math';

import 'package:crypto/crypto.dart';

/// A single-use OAuth nonce pair for Sign in with Apple.
///
/// Apple's flow takes the SHA-256 digest ([hashed]) in the authorization
/// request and embeds it in the returned ID token; moth then verifies the
/// [raw] value sent with `SignInWithOAuth` against that digest, binding the
/// token to this sign-in attempt. [MothLoginScreen]'s Apple button does this
/// automatically; custom UIs generate one per attempt:
///
/// ```dart
/// final nonce = MothNonce.generate();
/// // pass nonce.hashed to SignInWithApple, nonce.raw to signInWithOAuth
/// ```
class MothNonce {
  MothNonce._(this.raw, this.hashed);

  /// Generates a cryptographically random nonce of [length] characters.
  factory MothNonce.generate([int length = 32]) {
    const charset =
        'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._';
    final random = Random.secure();
    final raw = String.fromCharCodes(
      Iterable.generate(
        length,
        (_) => charset.codeUnitAt(random.nextInt(charset.length)),
      ),
    );
    return MothNonce._(raw, sha256.convert(utf8.encode(raw)).toString());
  }

  /// The value to send to moth as `rawNonce`.
  final String raw;

  /// Hex-encoded SHA-256 of [raw] — the value to hand to Apple.
  final String hashed;
}
