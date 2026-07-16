import 'dart:convert';

/// Decodes a JWT payload WITHOUT verifying the signature. The SDK only reads
/// claims for client-side gating; the developer's backend verifies tokens
/// against the project JWKS and remains the authority. Returns an empty map
/// for anything that does not decode as a JWT.
Map<String, Object?> decodeJwtPayload(String token) {
  final parts = token.split('.');
  if (parts.length != 3) return const {};
  try {
    final payload = utf8.decode(
      base64Url.decode(base64Url.normalize(parts[1])),
    );
    final decoded = jsonDecode(payload);
    return decoded is Map<String, Object?> ? decoded : const {};
  } on FormatException {
    return const {};
  }
}

/// The custom-claims object moth embeds under the `claims` claim, or empty.
Map<String, Object?> customClaimsOf(String accessToken) =>
    switch (decodeJwtPayload(accessToken)['claims']) {
      final Map<String, Object?> m => m,
      _ => const {},
    };
