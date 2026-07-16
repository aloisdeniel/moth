/// Base class for every error surfaced by the moth SDK.
///
/// The server attaches a stable machine-readable reason to its errors as a
/// `google.rpc.ErrorInfo` detail (domain `moth.dev`); the SDK maps each known
/// reason to a subclass below so callers catch types instead of matching
/// strings. Unknown reasons and non-moth failures surface as the base class,
/// with [reason] still populated when one was present.
class MothException implements Exception {
  MothException(this.message, {this.reason});

  /// Human-readable server message.
  final String message;

  /// The `ErrorInfo` reason, e.g. `INVALID_CREDENTIALS`, or null.
  final String? reason;

  @override
  String toString() => reason == null
      ? '$runtimeType: $message'
      : '$runtimeType($reason): $message';
}

/// Wrong email/password combination (`INVALID_CREDENTIALS`).
class MothInvalidCredentials extends MothException {
  MothInvalidCredentials(super.message) : super(reason: 'INVALID_CREDENTIALS');
}

/// The project requires a verified email before sign-in
/// (`EMAIL_NOT_VERIFIED`).
class MothEmailNotVerified extends MothException {
  MothEmailNotVerified(super.message) : super(reason: 'EMAIL_NOT_VERIFIED');
}

/// The email is already registered (`EMAIL_ALREADY_EXISTS`).
class MothEmailAlreadyExists extends MothException {
  MothEmailAlreadyExists(super.message) : super(reason: 'EMAIL_ALREADY_EXISTS');
}

/// Public sign-up is closed for this project (`SIGNUP_CLOSED`).
class MothSignUpClosed extends MothException {
  MothSignUpClosed(super.message) : super(reason: 'SIGNUP_CLOSED');
}

/// The password does not meet the project's policy (`WEAK_PASSWORD`).
class MothWeakPassword extends MothException {
  MothWeakPassword(super.message) : super(reason: 'WEAK_PASSWORD');
}

/// The email address is malformed (`INVALID_EMAIL`).
class MothInvalidEmail extends MothException {
  MothInvalidEmail(super.message) : super(reason: 'INVALID_EMAIL');
}

/// An email verification / password reset / email change token is invalid or
/// expired (`INVALID_TOKEN`).
class MothInvalidToken extends MothException {
  MothInvalidToken(super.message) : super(reason: 'INVALID_TOKEN');
}

/// The refresh token is unknown, revoked or expired
/// (`INVALID_REFRESH_TOKEN`). The SDK clears the session when this happens.
class MothInvalidRefreshToken extends MothException {
  MothInvalidRefreshToken(super.message)
    : super(reason: 'INVALID_REFRESH_TOKEN');
}

/// An already-rotated refresh token was presented; the server treats it as
/// theft and revokes the whole token family (`REFRESH_TOKEN_REUSED`). The
/// SDK clears the session when this happens.
class MothRefreshTokenReused extends MothException {
  MothRefreshTokenReused(super.message) : super(reason: 'REFRESH_TOKEN_REUSED');
}

/// The access token is missing, malformed or expired
/// (`INVALID_ACCESS_TOKEN`).
class MothInvalidAccessToken extends MothException {
  MothInvalidAccessToken(super.message) : super(reason: 'INVALID_ACCESS_TOKEN');
}

/// The account is disabled (`USER_DISABLED`).
class MothUserDisabled extends MothException {
  MothUserDisabled(super.message) : super(reason: 'USER_DISABLED');
}

/// Too many attempts; retry later (`RATE_LIMITED`).
class MothRateLimited extends MothException {
  MothRateLimited(super.message) : super(reason: 'RATE_LIMITED');
}

/// The social provider is not enabled for this project
/// (`PROVIDER_DISABLED`).
class MothProviderDisabled extends MothException {
  MothProviderDisabled(super.message) : super(reason: 'PROVIDER_DISABLED');
}

/// The provider ID token failed verification (`INVALID_PROVIDER_TOKEN`).
class MothInvalidProviderToken extends MothException {
  MothInvalidProviderToken(super.message)
    : super(reason: 'INVALID_PROVIDER_TOKEN');
}

/// The one-time OAuth exchange code is invalid, expired or already used
/// (`INVALID_OAUTH_CODE`).
class MothInvalidOAuthCode extends MothException {
  MothInvalidOAuthCode(super.message) : super(reason: 'INVALID_OAUTH_CODE');
}

/// The OAuth redirect URI is not allowed (`INVALID_REDIRECT`).
class MothInvalidRedirect extends MothException {
  MothInvalidRedirect(super.message) : super(reason: 'INVALID_REDIRECT');
}

/// Unlinking this identity would leave the account without any way to sign
/// in (`LAST_LOGIN_METHOD`).
class MothLastLoginMethod extends MothException {
  MothLastLoginMethod(super.message) : super(reason: 'LAST_LOGIN_METHOD');
}

/// The project has no store credentials configured, so purchases cannot be
/// validated (`BILLING_NOT_CONFIGURED`).
class MothBillingNotConfigured extends MothException {
  MothBillingNotConfigured(super.message)
    : super(reason: 'BILLING_NOT_CONFIGURED');
}

/// The purchase receipt / signed transaction was rejected by the store or is
/// malformed (`INVALID_RECEIPT`).
class MothInvalidReceipt extends MothException {
  MothInvalidReceipt(super.message) : super(reason: 'INVALID_RECEIPT');
}

/// The store could not be reached to validate the purchase; retrying may
/// succeed (`STORE_UNAVAILABLE`).
class MothStoreUnavailable extends MothException {
  MothStoreUnavailable(super.message) : super(reason: 'STORE_UNAVAILABLE');
}

/// The server could not be reached (connection, timeout, transport failure).
/// The session is kept — retrying may succeed.
class MothNetworkError extends MothException {
  MothNetworkError(super.message);
}
