import 'dart:async';

import 'package:flutter/material.dart';

import '../auth_state.dart';
import '../client.dart';
import '../config.dart';
import '../copy.dart';
import '../copy_cache.dart';
import '../copy_controller.dart';
import '../customer_info.dart';
import '../entitlement_cache.dart';
import '../i18n/localizations.dart';
import '../purchase.dart';
import '../push.dart';
import '../push_controller.dart';
import '../push_device_id.dart';
import '../subscription_controller.dart';
import '../theme.dart';
import '../theme_cache.dart';
import '../theme_controller.dart';
import '../token_store.dart';
import 'billing_adapter.dart';
import 'moth_copy_scope.dart';
import 'moth_login_screen.dart';
import 'moth_paywall_screen.dart';
import 'moth_scope.dart';
import 'moth_theme_scope.dart';
import 'oauth_adapter.dart';
import 'purchase_flow.dart';
import 'push_adapter.dart';

/// Top-level widget that owns a [MothClient] and gates [child] behind
/// authentication:
///
/// ```dart
/// void main() {
///   runApp(MothApp(
///     config: MothConfig(
///       endpoint: Uri.parse('https://auth.example.com'),
///       publishableKey: 'pk_...',
///     ),
///     child: const MyApp(),
///   ));
/// }
/// ```
///
/// On mount it restores the persisted session, then renders per state:
/// [MothAuthLoading] → [loading] (default: a centered progress indicator),
/// [MothSignedOut] → [signedOut] (default: [MothLoginScreen]),
/// [MothSignedIn] → [child]. With `requireAuth: false` [child] always
/// renders and reads the state itself via [MothScope.of], which is
/// available below this widget either way.
///
/// The screens MothApp owns (loading and signed-out) render with the
/// project's [MothTheme] as configured in the moth admin, refreshed
/// stale-while-revalidate: the last cached theme shows immediately, a
/// background fetch picks up admin edits. [child] — the app itself — keeps
/// the app's own theme untouched; the moth theme only ever applies to moth
/// screens. Pass [theme] to pin a hand-built theme instead (no fetch, no
/// cache), or [themeCache] to change where the delivered theme persists.
///
/// Pass either [config] (the widget creates and disposes the client) or an
/// existing [client] (the caller keeps ownership and disposes it); both
/// are fixed for the lifetime of the widget. When `MothApp` sits above
/// [MaterialApp] — the usual layout — the loading/signed-out screens are
/// wrapped in a minimal `MaterialApp` shell of their own, themed from the
/// project theme.
class MothApp extends StatefulWidget {
  const MothApp({
    super.key,
    this.config,
    this.client,
    this.tokenStore,
    this.entitlementCache,
    this.oauthAdapter,
    this.billingAdapter,
    this.pushAdapter,
    this.pushDeviceIdStore,
    this.theme,
    this.themeCache,
    this.copyCache,
    this.loading,
    this.signedOut,
    this.requiresEntitlement,
    this.paywall,
    this.requireAuth = true,
    required this.child,
  }) : assert(
         (config == null) != (client == null),
         'Provide exactly one of config or client.',
       ),
       assert(
         client == null || tokenStore == null,
         'tokenStore only applies when MothApp creates the client.',
       ),
       assert(
         client == null || entitlementCache == null,
         'entitlementCache only applies when MothApp creates the client.',
       ),
       assert(
         theme == null || themeCache == null,
         'themeCache only applies when the server theme is used.',
       ),
       assert(
         pushAdapter != null || pushDeviceIdStore == null,
         'pushDeviceIdStore only applies when a pushAdapter is wired.',
       ),
       assert(
         requiresEntitlement == null || requireAuth,
         'requiresEntitlement gates the signed-in child, so requireAuth must '
         'stay true.',
       );

  /// Connection settings; the widget creates (and disposes) the client.
  final MothConfig? config;

  /// An externally owned client, e.g. one also used outside the widget
  /// tree. The caller disposes it.
  final MothClient? client;

  /// Session persistence override for the client created from [config]
  /// (defaults to secure storage).
  final TokenStore? tokenStore;

  /// Entitlement-cache override for the client created from [config]
  /// (defaults to a device file cache; useful for tests).
  final MothEntitlementCache? entitlementCache;

  /// Bridges the login screen's Google/Apple buttons to the native
  /// sign-in SDKs; exposed to descendants via [MothScope.oauthAdapter].
  final MothOAuthAdapter? oauthAdapter;

  /// Runs native store purchases for [MothScope.purchase] / the paywall;
  /// exposed to descendants via [MothScope.billingAdapter]. The widget also
  /// listens to the adapter's [MothBillingAdapter.transactionUpdates] and
  /// submits every out-of-band receipt (Ask to Buy approval, pending payment
  /// confirming, renewal) for validation, so deferred purchases complete
  /// without app code.
  final MothBillingAdapter? billingAdapter;

  /// Turns on push-device registration: while a user is signed in the SDK
  /// obtains the push credential from this adapter and keeps the project's
  /// device registry current (register on launch/sign-in/token rotation,
  /// unregister on sign-out) — see [MothPushController]. No adapter, no
  /// push; nothing else changes. The OS permission prompt stays an explicit
  /// app call ([MothScope.requestPushPermission]) — wiring the adapter
  /// never prompts. Fixed for the lifetime of the widget.
  final MothPushAdapter? pushAdapter;

  /// Persistence override for the stable push installation id (defaults to
  /// a device file store; useful for tests).
  final MothPushDeviceIdStore? pushDeviceIdStore;

  /// Fixed theme for the moth screens; wins over the server-configured
  /// project theme (which is then neither fetched nor cached).
  final MothTheme? theme;

  /// Persistence override for the server-delivered theme (defaults to a
  /// file cache; useful for tests).
  final MothThemeCache? themeCache;

  /// Persistence override for the server-delivered localized copy (defaults to
  /// a file cache; useful for tests).
  final MothCopyCache? copyCache;

  /// Shown while the session restore is in flight.
  final Widget? loading;

  /// Shown while signed out; defaults to [MothLoginScreen].
  final Widget? signedOut;

  /// When set, the signed-in [child] is gated behind this entitlement (e.g.
  /// `pro`): a user who holds it sees [child]; a user who doesn't sees
  /// [paywall]. Gating on an entitlement no product grants never blocks
  /// (nothing to sell), keeping a project with no products runnable.
  final String? requiresEntitlement;

  /// Shown to a signed-in user who lacks [requiresEntitlement]; defaults to
  /// [MothPaywallScreen].
  final Widget? paywall;

  /// When false, [child] renders regardless of auth state.
  final bool requireAuth;

  /// The app itself, rendered once signed in.
  final Widget child;

  @override
  State<MothApp> createState() => _MothAppState();
}

class _MothAppState extends State<MothApp> with WidgetsBindingObserver {
  late final MothClient _client;
  late final bool _ownsClient;
  late MothAuthState _state;
  late MothCustomerInfo _customerInfo;
  StreamSubscription<MothAuthState>? _subscription;
  StreamSubscription<MothPurchaseReceipt>? _billingUpdates;
  MothSubscriptionController? _subs;
  MothPushController? _push;
  MothPushStatus _pushStatus = MothPushStatus.unavailable;
  MothThemeController? _theme;
  MothCopyController? _copy;

  @override
  void initState() {
    super.initState();
    _ownsClient = widget.client == null;
    _client =
        widget.client ??
        MothClient(widget.config!, tokenStore: widget.tokenStore);
    _state = _client.currentState;
    _subscription = _client.authStateChanges.listen((state) {
      if (!mounted) return;
      setState(() => _state = state);
    });
    // Subscription state: stale-while-revalidate per user, the same shape as
    // the theme controller. MothScope reads the controller's value, so gating
    // is instant on launch and refreshes in the background.
    final subs = MothSubscriptionController(
      client: _client,
      cache: widget.entitlementCache,
    );
    subs.addListener(_onCustomerInfoChanged);
    _subs = subs;
    _customerInfo = subs.value;
    unawaited(subs.start());
    _listenForBillingUpdates();
    final pushAdapter = widget.pushAdapter;
    if (pushAdapter != null) {
      // Push registration: register while signed in (every launch — the
      // server upserts), re-register on token rotation, unregister through
      // the scope's sign-out. Never prompts; requestPushPermission is the
      // app's explicit call.
      final push = MothPushController(
        client: _client,
        adapter: pushAdapter,
        deviceIdStore: widget.pushDeviceIdStore,
      );
      push.addListener(_onPushChanged);
      _push = push;
      _pushStatus = push.value;
      unawaited(push.start());
    }
    if (_state is MothAuthLoading) {
      // Failures surface through the state stream (restore keeps or clears
      // the session itself); nothing to await here.
      unawaited(_client.restore());
    }
    if (widget.theme == null && widget.requireAuth) {
      // Stale-while-revalidate: cached theme first, background refresh
      // after. Started even when the restore will land on signedIn, so the
      // cache is warm for the next sign-out.
      final controller = MothThemeController(
        client: _client,
        cache: widget.themeCache,
      );
      controller.addListener(_onThemeChanged);
      _theme = controller;
      unawaited(controller.start());
    }
    if (widget.requireAuth) {
      // Localized copy for the moth-owned screens: same stale-while-revalidate
      // discipline as the theme, keyed by (locale, revision). A device-language
      // change triggers a refetch via didChangeLocales below.
      final copy = MothCopyController(client: _client, cache: widget.copyCache);
      copy.addListener(_onCopyChanged);
      _copy = copy;
      WidgetsBinding.instance.addObserver(this);
      unawaited(copy.start());
    }
  }

  /// Forwards receipts that complete outside a purchase call (Ask to Buy
  /// approvals, pending payments confirming, store renewals) to
  /// `SubmitPurchase`. This is what completes a deferred purchase with no app
  /// code: validation re-derives entitlements and the subscription controller
  /// picks them up through the client's customer-info stream.
  /// submitMothReceipt never throws — a failure (offline, signed out, receipt
  /// rejected) leaves the store transaction unfinished/unacknowledged, so the
  /// store redelivers it or a restore recovers it.
  void _listenForBillingUpdates() {
    _billingUpdates = widget.billingAdapter?.transactionUpdates.listen(
      (receipt) => unawaited(submitMothReceipt(_client, receipt)),
    );
  }

  @override
  void didUpdateWidget(MothApp oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.billingAdapter != oldWidget.billingAdapter) {
      _billingUpdates?.cancel();
      _listenForBillingUpdates();
    }
  }

  void _onThemeChanged() {
    if (mounted) setState(() {});
  }

  void _onCopyChanged() {
    if (mounted) setState(() {});
  }

  @override
  void didChangeLocales(List<Locale>? locales) {
    // The device language changed: reload the new locale's cached floor and
    // refetch its copy. MothCopyController diffs the locale itself.
    unawaited(_copy?.refresh());
  }

  void _onCustomerInfoChanged() {
    if (mounted) setState(() => _customerInfo = _subs!.value);
  }

  void _onPushChanged() {
    if (mounted) setState(() => _pushStatus = _push!.value);
  }

  @override
  void dispose() {
    if (_copy != null) WidgetsBinding.instance.removeObserver(this);
    _subscription?.cancel();
    _billingUpdates?.cancel();
    _subs?.dispose();
    _push?.dispose();
    _theme?.dispose();
    _copy?.dispose();
    if (_ownsClient) unawaited(_client.dispose());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    Widget body;
    var ownSurface = false;
    if (widget.requireAuth) {
      switch (_state) {
        case MothAuthLoading():
          body = widget.loading ?? const _MothSplash();
          ownSurface = true;
        case MothSignedOut():
          body = widget.signedOut ?? const MothLoginScreen();
          ownSurface = true;
        case MothSignedIn():
          final gate = widget.requiresEntitlement;
          if (gate == null || _customerInfo.hasEntitlement(gate)) {
            body = widget.child;
          } else {
            // Not (yet) entitled: hand off to a gate that shows the paywall,
            // or falls through to the child when no product grants the
            // entitlement (nothing to sell — never block). It is a moth-owned
            // surface, so it gets the themed shell like the login screen.
            body = _MothEntitlementGate(
              entitlement: gate,
              paywall: widget.paywall ?? const MothPaywallScreen(),
              child: widget.child,
            );
            ownSurface = true;
          }
      }
    } else {
      body = widget.child;
    }
    if (ownSurface) {
      // moth-owned screens render with the project theme and the negotiated
      // localized copy; the app's own subtree (child) is deliberately left
      // alone.
      final mothTheme = widget.theme ?? _theme?.value ?? MothTheme.fallback();
      final mothCopy = _copy?.value ?? MothCopy.bundled(_client.currentLocale);
      body = MothCopyScope(
        copy: mothCopy,
        child: _MothThemedSurface(theme: mothTheme, child: body),
      );
      // When MothApp is the root of the tree (above the app's
      // MaterialApp), its own surfaces need an app shell for
      // Directionality, Material theming, overlays, and the localization
      // delegates so MaterialLocalizations resolve in the device language.
      if (Directionality.maybeOf(context) == null) {
        body = MaterialApp(
          debugShowCheckedModeBanner: false,
          theme: mothTheme.toThemeData(Brightness.light),
          darkTheme: mothTheme.toThemeData(Brightness.dark),
          locale: _client.config.locale,
          localizationsDelegates: mothLocalizationsDelegates,
          supportedLocales: mothSupportedLocales,
          home: body,
        );
      }
    }
    if (widget.requireAuth) {
      // Distinct keys per side of the gate: a flip must fully remount the
      // subtree, never update it in place — otherwise (both sides usually
      // being MaterialApps) the app's navigator state, open routes and
      // dialogs would survive a sign-out underneath the login screen.
      body = KeyedSubtree(key: ValueKey<bool>(ownSurface), child: body);
    }
    return MothScope(
      client: _client,
      state: _state,
      customerInfo: _customerInfo,
      oauthAdapter: widget.oauthAdapter,
      billingAdapter: widget.billingAdapter,
      pushController: _push,
      pushStatus: _pushStatus,
      child: body,
    );
  }
}

/// Gates its [child] behind an entitlement once the user is signed in but
/// lacks it. Fetches the paywall's offering to decide whether there is
/// anything to sell: when a product grants the entitlement, shows [paywall];
/// when the offering is empty or no product grants it, falls through to
/// [child] (an undefined entitlement never blocks). Rebuilds — and hands the
/// user through — the moment [MothScope] reports the entitlement as held.
class _MothEntitlementGate extends StatefulWidget {
  const _MothEntitlementGate({
    required this.entitlement,
    required this.paywall,
    required this.child,
  });

  final String entitlement;
  final Widget paywall;
  final Widget child;

  @override
  State<_MothEntitlementGate> createState() => _MothEntitlementGateState();
}

class _MothEntitlementGateState extends State<_MothEntitlementGate> {
  bool _resolving = true;
  bool _blocks = true;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    // The offering never changes within a session; resolve it once.
    if (_resolving && !_resolved) {
      _resolved = true;
      unawaited(_resolveOffering());
    }
  }

  bool _resolved = false;

  Future<void> _resolveOffering() async {
    final client = MothScope.of(context).client;
    bool blocks;
    try {
      // Resolve the SAME offering the paywall will present, not the default
      // one: the paywall config can point at a non-default offering, and the
      // gated entitlement may be granted only by products there. Checking the
      // default offering would wrongly conclude "nothing to sell" and hand a
      // free user the gated content.
      final paywall = await client.getPaywall();
      final offering = await client.getOfferings(
        offering: paywall?.offering ?? '',
      );
      blocks = offering.grants(widget.entitlement);
    } on Object {
      // Couldn't load the catalog: default to showing the paywall, which
      // renders its own retry/empty state.
      blocks = true;
    }
    if (!mounted) return;
    setState(() {
      _resolving = false;
      _blocks = blocks;
    });
  }

  @override
  Widget build(BuildContext context) {
    // Rebuilds when the entitlement flips; MothApp then remounts to the child
    // directly, so this branch is only a transient shortcut.
    if (MothScope.of(context).hasEntitlement(widget.entitlement)) {
      return widget.child;
    }
    if (_resolving) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    return _blocks ? widget.paywall : widget.child;
  }
}

/// Applies the moth theme to a moth-owned screen: publishes it via
/// [MothThemeScope] and installs the matching Material [Theme] for the
/// ambient brightness.
class _MothThemedSurface extends StatelessWidget {
  const _MothThemedSurface({required this.theme, required this.child});

  final MothTheme theme;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final brightness =
        MediaQuery.maybePlatformBrightnessOf(context) ?? Brightness.light;
    return MothThemeScope(
      theme: theme,
      child: Theme(data: theme.toThemeData(brightness), child: child),
    );
  }
}

class _MothSplash extends StatelessWidget {
  const _MothSplash();

  @override
  Widget build(BuildContext context) =>
      const Scaffold(body: Center(child: CircularProgressIndicator()));
}
