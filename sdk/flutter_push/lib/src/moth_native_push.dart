import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:moth_auth/moth_auth.dart';

/// [MothPushAdapter] backed by moth's own native push bridge: APNs
/// (`UNUserNotificationCenter` + `registerForRemoteNotifications`) on iOS,
/// Firebase Cloud Messaging on Android. Ships from the moth binary at `/pub`
/// in lockstep with the server, so the credentials it produces are exactly
/// what `RegisterDevice` stores.
///
/// Credentials, not notifications: the plugin never displays messages,
/// handles taps or touches delivery — it produces the `(target, token)` pair
/// and reports the OS permission state faithfully, nothing more. And it
/// never prompts on its own: [requestPermission] runs only when the app
/// explicitly calls `MothScope.of(context).requestPushPermission()`.
///
/// On Android FCM needs the app's own Firebase config (google-services.json)
/// — the one piece of setup moth cannot absorb. When it is missing,
/// [getToken] throws a [PlatformException] with code
/// `firebase-not-initialized` and an actionable message; the SDK treats that
/// as a non-fatal registration failure, so auth is never blocked by push.
class MothNativePush implements MothPushAdapter {
  /// [provisional] (iOS only) requests provisional authorization: quiet
  /// notifications delivered without showing the permission prompt. Android
  /// ignores it.
  MothNativePush({this.provisional = false}) {
    channel.setMethodCallHandler(_onNativeCall);
  }

  /// Ask for iOS provisional authorization instead of the full prompt.
  final bool provisional;

  /// The method channel shared with the native sides.
  @visibleForTesting
  static const MethodChannel channel = MethodChannel('moth_push');

  final _refreshes = StreamController<MothPushToken>.broadcast();

  /// Credential rotations pushed by the platform: APNs re-registration
  /// callbacks (tokens rotate on restore/OS update) and FCM `onNewToken`.
  /// `MothApp` re-registers on every event; a rotation nobody listens to is
  /// harmless — registration on next launch picks the fresh token up.
  @override
  Stream<MothPushToken> get onTokenRefresh => _refreshes.stream;

  /// Call when the owning widget disposes.
  void dispose() {
    _refreshes.close();
  }

  @override
  Future<MothPushPermission> requestPermission() async {
    final raw = await channel.invokeMethod<String>('requestPermission', {
      'provisional': provisional,
    });
    return _permission(raw);
  }

  @override
  Future<MothPushPermission> permissionStatus() async =>
      _permission(await channel.invokeMethod<String>('permissionStatus'));

  @override
  Future<MothPushToken?> getToken() async {
    final raw = await channel.invokeMapMethod<Object?, Object?>('getToken');
    return _token(raw);
  }

  @override
  Future<MothPushDeviceMetadata> deviceMetadata() async {
    final raw = await channel.invokeMapMethod<Object?, Object?>(
      'deviceMetadata',
    );
    // Only what the native side knows better than Dart: the SDK fills
    // platform, OS version and locale itself.
    return MothPushDeviceMetadata(
      model: raw?['model'] as String? ?? '',
      appVersion: raw?['appVersion'] as String? ?? '',
    );
  }

  MothPushPermission _permission(String? raw) => switch (raw) {
    'granted' => MothPushPermission.granted,
    'provisional' => MothPushPermission.provisional,
    'denied' => MothPushPermission.denied,
    _ => MothPushPermission.unknown,
  };

  /// The credential for one native payload: `{target: apns|fcm, token}`.
  /// Null (no payload, or a target this version does not know) means no
  /// credential yet — the SDK retries on the next trigger.
  MothPushToken? _token(Map<Object?, Object?>? raw) {
    final target = switch (raw?['target']) {
      'apns' => MothPushTarget.apns,
      'fcm' => MothPushTarget.fcm,
      _ => null,
    };
    final token = raw?['token'] as String? ?? '';
    if (target == null || token.isEmpty) return null;
    return MothPushToken(target: target, token: token);
  }

  Future<Object?> _onNativeCall(MethodCall call) async {
    if (call.method == 'onTokenRefresh' && call.arguments is Map) {
      final token = _token((call.arguments as Map).cast<Object?, Object?>());
      if (token != null && !_refreshes.isClosed) {
        _refreshes.add(token);
      }
    }
    return null;
  }
}
