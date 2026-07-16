import '../exceptions.dart';

/// End-user copy for a [MothException] (or any error), used by the SDK's
/// built-in widgets. Exposed so custom login UIs can reuse the same
/// wording; the strings are plain English for now (localization arrives
/// with the design system milestone).
String friendlyMothErrorMessage(Object error) => switch (error) {
  MothInvalidCredentials() => 'Incorrect email or password.',
  MothEmailNotVerified() =>
    'Please verify your email address first — check your inbox.',
  MothEmailAlreadyExists() =>
    'An account with this email already exists. Try signing in instead.',
  MothSignUpClosed() => 'Sign-up is currently closed for this app.',
  // The server message spells out the policy that was not met.
  MothWeakPassword(:final message) => message,
  MothInvalidEmail() => 'That email address does not look right.',
  MothInvalidAccessToken() => 'Your session has expired — sign in again.',
  MothUserDisabled() => 'This account has been disabled.',
  MothRateLimited() => 'Too many attempts — wait a moment and try again.',
  MothProviderDisabled() => 'This sign-in method is not enabled for this app.',
  MothInvalidProviderToken() =>
    'Sign-in with the provider failed. Please try again.',
  MothLastLoginMethod() =>
    'This is your only way to sign in, so it cannot be removed.',
  MothNetworkError() =>
    'Cannot reach the server. Check your connection and try again.',
  MothException(:final message) => message,
  _ => 'Something went wrong. Please try again.',
};
