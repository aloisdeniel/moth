import 'package:flutter/widgets.dart';

import '../auth_state.dart';
import '../client.dart';
import '../customer_info.dart';
import '../offering.dart';
import '../purchase.dart';
import '../push.dart';
import '../push_controller.dart';
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
    this.pushController,
    this.pushStatus = MothPushStatus.unavailable,
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

  /// The push registration machinery [MothApp] creates when given a
  /// `pushAdapter`; null when no adapter is wired (push is then off).
  /// Consumed by [requestPushPermission] and the sign-out flow.
  final MothPushController? pushController;

  /// The push machinery's state at the time of the last change, for
  /// settings screens: unavailable (no adapter wired, or the project has
  /// push disabled), the OS permission, and whether this installation's
  /// registration reached the server.
  final MothPushStatus pushStatus;

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
  ///
  /// When a push adapter is wired, this installation's push registration is
  /// revoked **before** the session drops (the RPC needs the still-live
  /// Bearer token) — best effort, sign-out never blocks on push. Calling
  /// `client.signOut` directly skips that revocation; the server's takeover
  /// and staleness sweeps then reclaim the registration lazily.
  Future<void> signOut({bool allDevices = false}) async {
    await pushController?.unregisterForSignOut();
    await client.signOut(allDevices: allDevices);
  }

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
  Future<void> deleteAccount({String password = ''}) async {
    // As in signOut: revoke the push registration while the session can
    // still authenticate the call (best effort, never blocking).
    await pushController?.unregisterForSignOut();
    await client.deleteAccount(password: password);
  }

  /// Re-fetches the subscription state from the server; dependents rebuild
  /// with the fresh entitlements. Throws when signed out.
  Future<MothCustomerInfo> refreshCustomerInfo() => client.getCustomerInfo();

  /// Shows the OS notification-permission prompt and returns the resulting
  /// state; while signed in the SDK then re-registers so the server sees the
  /// new permission. This is the **only** way the SDK ever prompts —
  /// permission UX is a product decision, so it stays an explicit app call
  /// (a settings toggle, an onboarding step), never an SDK side effect.
  /// Returns [MothPushPermission.unknown] when no push adapter is wired.
  Future<MothPushPermission> requestPushPermission() async {
    final controller = pushController;
    if (controller == null) return MothPushPermission.unknown;
    return controller.requestPermission();
  }

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
      billingAdapter != oldWidget.billingAdapter ||
      pushController != oldWidget.pushController ||
      pushStatus != oldWidget.pushStatus;
}
