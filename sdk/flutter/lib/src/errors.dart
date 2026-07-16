import 'package:protobuf/protobuf.dart';

import 'exceptions.dart';
import 'transport/grpc.dart';

/// The `google.rpc.ErrorInfo` domain moth uses
/// (internal/server/rpc/auth/errors.go).
const String mothErrorDomain = 'moth.dev';

/// Maps a transport error to the typed [MothException] hierarchy using the
/// `google.rpc.ErrorInfo` reason the server attaches in
/// `grpc-status-details-bin`. Mirrors internal/server/rpc/auth/errors.go.
MothException mapGrpcError(GrpcError error) {
  final reason = _mothReason(error);
  final message = error.message ?? 'request failed';
  switch (reason) {
    case 'INVALID_CREDENTIALS':
      return MothInvalidCredentials(message);
    case 'EMAIL_NOT_VERIFIED':
      return MothEmailNotVerified(message);
    case 'EMAIL_ALREADY_EXISTS':
      return MothEmailAlreadyExists(message);
    case 'SIGNUP_CLOSED':
      return MothSignUpClosed(message);
    case 'WEAK_PASSWORD':
      return MothWeakPassword(message);
    case 'INVALID_EMAIL':
      return MothInvalidEmail(message);
    case 'INVALID_TOKEN':
      return MothInvalidToken(message);
    case 'INVALID_REFRESH_TOKEN':
      return MothInvalidRefreshToken(message);
    case 'REFRESH_TOKEN_REUSED':
      return MothRefreshTokenReused(message);
    case 'INVALID_ACCESS_TOKEN':
      return MothInvalidAccessToken(message);
    case 'USER_DISABLED':
      return MothUserDisabled(message);
    case 'RATE_LIMITED':
      return MothRateLimited(message);
    case 'PROVIDER_DISABLED':
      return MothProviderDisabled(message);
    case 'INVALID_PROVIDER_TOKEN':
      return MothInvalidProviderToken(message);
    case 'INVALID_OAUTH_CODE':
      return MothInvalidOAuthCode(message);
    case 'INVALID_REDIRECT':
      return MothInvalidRedirect(message);
    case 'LAST_LOGIN_METHOD':
      return MothLastLoginMethod(message);
  }
  // No moth reason: transport-level failures become MothNetworkError so
  // callers can retry; everything else keeps the raw reason (if any).
  switch (error.code) {
    case StatusCode.unavailable:
    case StatusCode.deadlineExceeded:
    case StatusCode.aborted:
      return MothNetworkError(message);
  }
  return MothException(message, reason: reason);
}

/// Extracts the moth `ErrorInfo` reason from the parsed status details.
/// Matched structurally (message name + field tags 1=reason, 2=domain) so
/// this works with whatever generated `ErrorInfo` class the transport used
/// to decode the detail.
String? _mothReason(GrpcError error) {
  for (final detail in error.details ?? const <GeneratedMessage>[]) {
    if (detail.info_.qualifiedMessageName != 'google.rpc.ErrorInfo') continue;
    final reason = detail.getField(1);
    final domain = detail.getField(2);
    if (domain == mothErrorDomain && reason is String && reason.isNotEmpty) {
      return reason;
    }
  }
  return null;
}
