import 'user.dart';

/// The authentication state of a [MothClient], for `switch`-based handling:
///
/// ```dart
/// switch (state) {
///   MothAuthLoading() => ...,   // session restore in progress
///   MothSignedOut() => ...,
///   MothSignedIn(:final user) => ...,
/// }
/// ```
sealed class MothAuthState {
  const MothAuthState();
}

/// The persisted session has not been restored yet (before `restore()`
/// completes).
final class MothAuthLoading extends MothAuthState {
  const MothAuthLoading();
}

/// No user is signed in.
final class MothSignedOut extends MothAuthState {
  const MothSignedOut();
}

/// [user] is signed in. Re-emitted when the session refreshes or the
/// profile changes.
final class MothSignedIn extends MothAuthState {
  const MothSignedIn(this.user);

  final MothUser user;
}
