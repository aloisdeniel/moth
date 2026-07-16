import 'package:flutter/widgets.dart';

import '../auth_state.dart';
import '../client.dart';
import '../customer_info.dart';
import '../offering.dart';
import '../purchase.dart';
import '../user.dart';
import 'billing_adapter.dart';
import 'oauth_adapter.dart';
import 'purchase_flow.dart';

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
    this.customerInfo = const MothCustomerInfo.free(),
    this.oauthAdapter,
    this.billingAdapter,
    required super.child,
  });

  /// The client, for calls beyond the convenience actions below.
  final MothClient client;

  /// The auth state at the time of the last change.
  final MothAuthState state;

  /// The signed-in user's subscription state at the time of the last change.
  /// Always valid — an empty [MothCustomerInfo] (the free `none` tier) for
  /// free / signed-out users; never null, so gating code never special-cases
  /// "never paid".
  final MothCustomerInfo customerInfo;

  /// The adapter wired into [MothApp], consumed by [MothLoginScreen]'s
  /// provider buttons.
  final MothOAuthAdapter? oauthAdapter;

  /// The adapter wired into [MothApp] that runs native store purchases,
  /// consumed by [purchase] / [restorePurchases] and [MothPaywallScreen].
  final MothBillingAdapter? billingAdapter;

  /// The signed-in user, or null while loading / signed out.
  MothUser? get user => switch (state) {
    MothSignedIn(:final user) => user,
    _ => null,
  };

  /// Whether the signed-in user currently holds [entitlement] (e.g. `pro`).
  /// The single question to ask when gating a feature.
  bool hasEntitlement(String entitlement) =>
      customerInfo.hasEntitlement(entitlement);

  /// Subscription-state changes, for non-widget code (widgets rebuild via
  /// [of]). Replays the current value, then every subsequent change.
  Stream<MothCustomerInfo> get entitlementsChanged =>
      client.customerInfoChanges;

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

  /// Re-fetches the subscription state from the server; dependents rebuild
  /// with the fresh entitlements. Throws when signed out.
  Future<MothCustomerInfo> refreshCustomerInfo() => client.getCustomerInfo();

  /// Buys [product]: runs the native store purchase through the
  /// [billingAdapter], forwards the receipt to moth for validation, and — on
  /// success — updates the subscription state so dependents rebuild. Returns a
  /// typed [MothPurchaseResult]; never throws for the expected outcomes
  /// (cancel, pending, already-owned, store/validation error).
  Future<MothPurchaseResult> purchase(MothOfferingProduct product) async {
    final adapter = billingAdapter;
    if (adapter == null) {
      return const MothPurchaseError(
        'No MothBillingAdapter is configured: pass one to MothApp or '
        'MothPaywallScreen.',
      );
    }
    return runMothPurchase(client, adapter, product);
  }

  /// Re-links the store's existing purchases on this device to the current
  /// user (new device, reinstall, account change) via the [billingAdapter],
  /// then updates the subscription state. Throws [StateError] when no adapter
  /// is configured.
  Future<MothCustomerInfo> restorePurchases() async {
    final adapter = billingAdapter;
    if (adapter == null) {
      throw StateError(
        'moth: no MothBillingAdapter is configured for restorePurchases().',
      );
    }
    return runMothRestore(client, adapter);
  }

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
      customerInfo != oldWidget.customerInfo ||
      client != oldWidget.client ||
      oauthAdapter != oldWidget.oauthAdapter ||
      billingAdapter != oldWidget.billingAdapter;
}
