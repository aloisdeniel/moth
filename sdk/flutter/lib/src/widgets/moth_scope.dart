import 'package:flutter/widgets.dart';

import '../auth_state.dart';
import '../client.dart';
import '../user.dart';
import 'oauth_adapter.dart';

/// Makes the moth auth state and client available to the widget tree.
///
/// [MothApp] inserts one automatically; `MothScope.of(context)` anywhere
/// below it returns the scope, and the calling widget rebuilds on every
/// auth-state change:
///
/// ```dart
/// final scope = MothScope.of(context);
/// switch (scope.state) {
///   MothAuthLoading() => ...,
///   MothSignedOut() => ...,
///   MothSignedIn(:final user) => Text(user.email),
/// }
/// ```
class MothScope extends InheritedWidget {
  const MothScope({
    super.key,
    required this.client,
    required this.state,
    this.oauthAdapter,
    required super.child,
  });

  /// The client, for calls beyond the convenience actions below.
  final MothClient client;

  /// The auth state at the time of the last change.
  final MothAuthState state;

  /// The adapter wired into [MothApp], consumed by [MothLoginScreen]'s
  /// provider buttons.
  final MothOAuthAdapter? oauthAdapter;

  /// The signed-in user, or null while loading / signed out.
  MothUser? get user => switch (state) {
    MothSignedIn(:final user) => user,
    _ => null,
  };

  /// Signs out (server-side revocation is best effort; the local session
  /// always ends). With [allDevices] every session of the user is revoked.
  Future<void> signOut({bool allDevices = false}) =>
      client.signOut(allDevices: allDevices);

  /// Re-fetches the profile from the server; dependents rebuild with the
  /// fresh user.
  Future<MothUser> refreshUser() => client.getMe();

  /// Permanently deletes the signed-in account, then ends the session.
  ///
  /// Deletion requires re-authentication (the App Store rule moth
  /// enforces server-side): password users pass their [password]; users
  /// who only sign in with Google/Apple leave it empty and may get a typed
  /// error asking for a recent sign-in. [showMothDeleteAccountDialog]
  /// wraps this in a ready-made Material prompt.
  Future<void> deleteAccount({String password = ''}) =>
      client.deleteAccount(password: password);

  /// The nearest scope, or null when there is none (no [MothApp] above).
  static MothScope? maybeOf(BuildContext context) =>
      context.dependOnInheritedWidgetOfExactType<MothScope>();

  /// The nearest scope. Throws when the widget tree has no [MothApp] (or
  /// hand-inserted [MothScope]) above [context].
  static MothScope of(BuildContext context) {
    final scope = maybeOf(context);
    if (scope == null) {
      throw FlutterError(
        'MothScope.of() called with a context that has no MothScope.\n'
        'Wrap your app in MothApp (or provide a MothScope) above this '
        'widget.',
      );
    }
    return scope;
  }

  @override
  bool updateShouldNotify(MothScope oldWidget) =>
      state != oldWidget.state ||
      client != oldWidget.client ||
      oauthAdapter != oldWidget.oauthAdapter;
}
