import 'dart:async';
import 'dart:developer' as developer;

import 'package:flutter/foundation.dart';

import 'auth_state.dart';
import 'client.dart';
import 'customer_info.dart';
import 'entitlement_cache.dart';

/// Owns the signed-in user's subscription state for a widget tree: a
/// [ValueListenable] that starts at the free `none` tier, flips to the
/// disk-cached [MothCustomerInfo] as soon as it loads (per user), then to the
/// server's current state once a background `GetCustomerInfo` confirms it —
/// stale-while-revalidate, the same shape as [MothThemeController] for the
/// theme, so gating is instant on launch and picks up store changes in the
/// background.
///
/// [MothApp] creates one automatically; instantiate one only when reading
/// entitlements outside a [MothApp].
class MothSubscriptionController extends ValueNotifier<MothCustomerInfo> {
  MothSubscriptionController({
    required MothClient client,
    MothEntitlementCache? cache,
  }) : _client = client,
       _cache = cache ?? defaultEntitlementCache(client.config.publishableKey),
       super(const MothCustomerInfo.free());

  final MothClient _client;
  final MothEntitlementCache _cache;

  StreamSubscription<MothAuthState>? _authSub;
  StreamSubscription<MothCustomerInfo>? _infoSub;

  /// The user id the current [value] belongs to; null while signed out.
  String? _userId;
  bool _started = false;
  bool _disposed = false;

  /// Begins tracking: every billing RPC result (via the client) publishes and
  /// caches here, and each transition to a new signed-in user loads that
  /// user's cached snapshot immediately, then refreshes from the server.
  /// Idempotent; failures are swallowed — the current value simply stays.
  Future<void> start() async {
    if (_started) return;
    _started = true;
    // customerInfoChanges must be attached before authStateChanges so a fresh
    // GetCustomerInfo triggered by the sign-in transition is not missed.
    _infoSub = _client.customerInfoChanges.listen(_onInfo);
    _authSub = _client.authStateChanges.listen(_onAuth);
  }

  void _onAuth(MothAuthState state) {
    switch (state) {
      case MothSignedIn(:final user):
        if (_userId == user.id) return; // same session (e.g. token refresh)
        _userId = user.id;
        unawaited(_loadAndRefresh(user.id));
      case MothSignedOut():
      case MothAuthLoading():
        _userId = null;
        _publish(const MothCustomerInfo.free());
    }
  }

  void _onInfo(MothCustomerInfo info) {
    _publish(info);
    final userId = _userId;
    if (userId == null) return;
    // Persist the latest server truth for instant gating next launch.
    unawaited(_save(userId, info));
  }

  Future<void> _loadAndRefresh(String userId) async {
    try {
      final cached = await _cache.load(userId);
      if (cached != null && _userId == userId) {
        _publish(cached);
        // Mirror the cached snapshot into the client so non-widget subscribers
        // (MothScope.entitlementsChanged) and currentCustomerInfo agree with
        // this cached widget state until the background refresh confirms it.
        _client.primeCustomerInfo(cached);
      }
    } on Object catch (err) {
      _log('entitlement cache load failed: $err');
    }
    // Background refresh; the result arrives via customerInfoChanges → _onInfo.
    // Failures (offline) keep the cached (or free) snapshot.
    try {
      await _client.getCustomerInfo();
    } on Object {
      // Best effort.
    }
  }

  Future<void> _save(String userId, MothCustomerInfo info) async {
    try {
      await _cache.save(userId, info);
    } on Object catch (err) {
      _log('entitlement cache save failed: $err');
    }
  }

  void _publish(MothCustomerInfo info) {
    if (_disposed) return;
    value = info;
  }

  void _log(String message) {
    assert(() {
      developer.log('moth: $message', name: 'moth', level: 900 /* warning */);
      return true;
    }());
  }

  @override
  void dispose() {
    _disposed = true;
    _authSub?.cancel();
    _infoSub?.cancel();
    super.dispose();
  }
}
