import 'dart:async';
import 'dart:developer' as developer;

import 'package:flutter/foundation.dart';

import 'auth_state.dart';
import 'client.dart';
import 'locale.dart';
import 'platform/platform_stub.dart'
    if (dart.library.io) 'platform/platform_io.dart'
    if (dart.library.js_interop) 'platform/platform_web.dart';
import 'push.dart';
import 'push_device_id.dart';
import 'widgets/push_adapter.dart';

/// Orchestrates push-device registration against `moth.push.v1` for a
/// [MothPushAdapter]: while a user is signed in it obtains the credential
/// from the adapter and calls `RegisterDevice` with this installation's
/// persisted stable `device_id`, the current permission state and device
/// metadata — on sign-in, on every launch while signed in, on every
/// [MothPushAdapter.onTokenRefresh] event and after [requestPermission].
/// The server upserts, so repeating ourselves is the whole client-side
/// bookkeeping (no "am I registered?" cache to corrupt).
///
/// Registration failures are **non-fatal by design**: auth flows never block
/// on push and never surface push errors — a failed registration simply
/// retries on the next launch/sign-in/rotation. Sign-out revocation runs
/// through [unregisterForSignOut], which [MothScope.signOut] calls **before**
/// dropping the session (the RPC needs the still-live Bearer token); the
/// device id survives sign-out — it identifies the installation, not the
/// user, and the server's newest-owner-wins upsert makes reuse across
/// accounts safe.
///
/// [MothApp] creates one automatically when given a `pushAdapter`; the
/// current state is exposed as a [ValueListenable] of [MothPushStatus]
/// (surfaced as `MothScope.pushStatus`). The SDK never shows the OS
/// permission prompt on its own — [requestPermission] runs only on an
/// explicit app call.
class MothPushController extends ValueNotifier<MothPushStatus> {
  MothPushController({
    required MothClient client,
    required this.adapter,
    MothPushDeviceIdStore? deviceIdStore,
  }) : _client = client,
       _deviceIds =
           deviceIdStore ??
           defaultPushDeviceIdStore(client.config.publishableKey),
       super(MothPushStatus.unavailable);

  /// The adapter wired into [MothApp], producing the platform's credential.
  final MothPushAdapter adapter;

  final MothClient _client;
  final MothPushDeviceIdStore _deviceIds;

  StreamSubscription<MothAuthState>? _authSub;
  StreamSubscription<MothPushToken>? _tokenSub;

  /// The signed-in user id registrations currently belong to; null while
  /// signed out.
  String? _userId;
  String? _deviceIdCache;
  bool _started = false;
  bool _disposed = false;

  /// Bumped per registration attempt so a slow one that loses the race
  /// (token rotated, user switched, sign-out) never publishes stale state.
  int _sequence = 0;

  /// Begins tracking: each transition to a signed-in user registers, and
  /// every adapter token rotation re-registers. Idempotent; failures are
  /// swallowed (retried on the next trigger).
  Future<void> start() async {
    if (_started) return;
    _started = true;
    _tokenSub = adapter.onTokenRefresh.listen((_) => unawaited(refresh()));
    _authSub = _client.authStateChanges.listen(_onAuth);
  }

  void _onAuth(MothAuthState state) {
    switch (state) {
      case MothSignedIn(:final user):
        if (_userId == user.id) return; // same session (e.g. token refresh)
        _userId = user.id;
        unawaited(_register());
      case MothSignedOut():
      case MothAuthLoading():
        _userId = null;
        _sequence++; // cancel any in-flight registration's publish
        _publish(value.copyWith(registered: false));
    }
  }

  /// Shows the OS permission prompt via the adapter and — while signed in —
  /// re-registers so the server sees the new permission state. Explicit by
  /// contract: the SDK never calls this on its own.
  Future<MothPushPermission> requestPermission() async {
    final permission = await adapter.requestPermission();
    if (_disposed) return permission;
    _publish(value.copyWith(permission: permission));
    await refresh();
    return permission;
  }

  /// Re-runs registration now (no-op while signed out). Useful after a
  /// permission change made outside the SDK, e.g. returning from the OS
  /// settings screen.
  Future<void> refresh() async {
    if (_userId == null) return;
    await _register();
  }

  /// Revokes this installation's registration (`signed_out`) — called by
  /// [MothScope.signOut] / [MothScope.deleteAccount] **before** the session
  /// drops, while the Bearer token can still authenticate the RPC. Best
  /// effort and idempotent: failures are swallowed (the server's takeover
  /// and staleness sweeps cover a missed revocation). Local push state is
  /// cleared, but the device id is kept — it identifies the installation,
  /// not the user.
  Future<void> unregisterForSignOut() async {
    if (_userId == null) return;
    _sequence++; // a racing registration must not resurrect `registered`
    _publish(value.copyWith(registered: false));
    var deviceId = _deviceIdCache;
    if (deviceId == null) {
      try {
        deviceId = await _deviceIds.load();
      } on Object catch (err) {
        _log('push device id load failed: $err');
      }
    }
    // Never registered on this installation: nothing to revoke.
    if (deviceId == null) return;
    try {
      await _client.unregisterPushDevice(deviceId: deviceId);
    } on Object catch (err) {
      _log('push unregister failed: $err');
    }
  }

  /// One registration attempt. Never throws: push is strictly best effort —
  /// a failure leaves [value] un-registered and the next trigger retries.
  Future<void> _register() async {
    final sequence = ++_sequence;
    final userId = _userId;
    if (userId == null) return;
    try {
      final config = await _client.getPushConfig();
      if (!config.enabled) {
        if (sequence == _sequence) _publish(MothPushStatus.unavailable);
        return;
      }
      final permission = await adapter.permissionStatus();
      final token = await adapter.getToken();
      if (_disposed || sequence != _sequence) return;
      if (token == null) {
        // No credential yet (native registration pending): report the
        // permission so settings screens are honest; retry on rotation.
        _publish(MothPushStatus(available: true, permission: permission));
        return;
      }
      final deviceId = await _deviceId();
      final extra = await adapter.deviceMetadata();
      await _client.registerPushDevice(
        target: token.target,
        token: token.token,
        deviceId: deviceId,
        permission: permission,
        metadata: MothPushDeviceMetadata(
          platform: extra.platform.isEmpty ? currentPlatform() : extra.platform,
          model: extra.model,
          osVersion: extra.osVersion.isEmpty
              ? currentOsVersion()
              : extra.osVersion,
          appVersion: extra.appVersion,
          locale: extra.locale.isEmpty
              ? mothLanguageTag(_client.currentLocale)
              : extra.locale,
        ),
      );
      if (_disposed || sequence != _sequence) return;
      _publish(
        MothPushStatus(
          available: true,
          permission: permission,
          registered: true,
        ),
      );
    } on Object catch (err) {
      _log('push registration failed: $err');
    }
  }

  /// The stable installation id: persisted on first use, then reused for
  /// every registration (including across sign-outs) so one physical device
  /// always replaces its own row.
  Future<String> _deviceId() async {
    final cached = _deviceIdCache;
    if (cached != null) return cached;
    String? stored;
    try {
      stored = await _deviceIds.load();
    } on Object catch (err) {
      _log('push device id load failed: $err');
    }
    final deviceId = stored ?? generatePushDeviceId();
    _deviceIdCache = deviceId;
    if (stored == null) {
      try {
        await _deviceIds.save(deviceId);
      } on Object catch (err) {
        // A fresh id next launch just supersedes this row (same token).
        _log('push device id save failed: $err');
      }
    }
    return deviceId;
  }

  void _publish(MothPushStatus status) {
    if (_disposed) return;
    value = status;
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
    _tokenSub?.cancel();
    super.dispose();
  }
}
