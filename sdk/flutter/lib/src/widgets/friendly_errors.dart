import '../copy.dart';
import '../exceptions.dart';

/// End-user copy for a [MothException] (or any error), used by the SDK's
/// built-in widgets. Exposed so custom login UIs can reuse the same wording.
///
/// When a [copy] is supplied (the SDK screens always pass their resolved
/// [MothCopy]), every mapped error resolves from the localized catalog —
/// wrong credentials (`sign_in.error_invalid`) and an already-registered email
/// (`sign_up.error_email_taken`) share the login-form keys, the rest use the
/// shared `error.*` group. Without a [copy] each falls back to English. The two
/// cases that echo a server-supplied `message` (a weak-password policy and the
/// generic `MothException`) are already localized by the server.
String friendlyMothErrorMessage(Object error, {MothCopy? copy}) {
  String c(String key, String fallback) =>
      copy == null ? fallback : copy.value(key);
  return switch (error) {
    MothInvalidCredentials() => c(
      'sign_in.error_invalid',
      'Incorrect email or password.',
    ),
    MothEmailNotVerified() => c(
      'error.email_not_verified',
      'Please verify your email address first — check your inbox.',
    ),
    MothEmailAlreadyExists() => c(
      'sign_up.error_email_taken',
      'An account with this email already exists. Try signing in instead.',
    ),
    MothSignUpClosed() => c(
      'error.signup_closed',
      'Sign-up is currently closed for this app.',
    ),
    // The server message spells out the policy that was not met.
    MothWeakPassword(:final message) => message,
    MothInvalidEmail() => c(
      'error.invalid_email',
      'That email address does not look right.',
    ),
    MothInvalidAccessToken() => c(
      'error.session_expired',
      'Your session has expired — sign in again.',
    ),
    MothUserDisabled() => c(
      'error.user_disabled',
      'This account has been disabled.',
    ),
    MothRateLimited() => c(
      'error.rate_limited',
      'Too many attempts — wait a moment and try again.',
    ),
    MothProviderDisabled() => c(
      'error.provider_disabled',
      'This sign-in method is not enabled for this app.',
    ),
    MothInvalidProviderToken() => c(
      'error.provider_failed',
      'Sign-in with the provider failed. Please try again.',
    ),
    MothLastLoginMethod() => c(
      'error.last_login_method',
      'This is your only way to sign in, so it cannot be removed.',
    ),
    MothNetworkError() => c(
      'error.network',
      'Cannot reach the server. Check your connection and try again.',
    ),
    MothException(:final message) => message,
    _ => c('error.generic', 'Something went wrong. Please try again.'),
  };
}
