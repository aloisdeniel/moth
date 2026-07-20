import '../push.dart';

/// Bridges moth's push-device registry to a native push service.
///
/// `moth_auth` deliberately does **not** depend on any push plugin — apps
/// that don't send notifications stay light and skip the native setup, the
/// same optional-native-adapter pattern as [MothOAuthAdapter] and
/// [MothBillingAdapter]. Wiring an adapter into [MothApp] is the whole
/// opt-in: while a user is signed in the SDK obtains the credential from the
/// adapter and keeps the server's registry current ([RegisterDevice] on
/// every launch, on token rotation and on permission changes;
/// [UnregisterDevice] on sign-out). No adapter, no push — nothing else
/// changes.
///
/// The first-party `moth_push` package implements this interface natively
/// (APNs on iOS, FCM on Android). It is also the escape hatch: an app with
/// its own push stack — e.g. an existing `firebase_messaging` setup — wraps
/// it in an adapter instead, and moth still gets faithful registrations
/// without owning the plugin choice.
///
/// The adapter produces **credentials, not notifications**: message display,
/// tap handling and routing stay entirely in the app's hands, and the SDK
/// never shows the OS permission prompt on its own — [requestPermission] runs
/// only when the app explicitly calls
/// `MothScope.of(context).requestPushPermission()`.
abstract class MothPushAdapter {
  /// Shows the OS notification-permission prompt (when not already decided)
  /// and returns the resulting state. Called only from the app's explicit
  /// `requestPushPermission()` — never by the SDK on its own.
  Future<MothPushPermission> requestPermission();

  /// The current OS permission state, without prompting.
  Future<MothPushPermission> permissionStatus();

  /// The current push credential, or null when none is available yet (e.g.
  /// the native registration has not completed). May throw when the native
  /// setup is broken (e.g. missing Firebase config) — the SDK treats that as
  /// a non-fatal registration failure.
  Future<MothPushToken?> getToken();

  /// Fires whenever the platform rotates the credential (APNs tokens rotate
  /// on restore/OS update, FCM refreshes registration tokens); the SDK
  /// re-registers on every event. Adapters without rotation events keep the
  /// default empty stream.
  Stream<MothPushToken> get onTokenRefresh => const Stream.empty();

  /// Display metadata the native side can provide (device model, app
  /// version, ...). Fields left empty are filled by the SDK where it can
  /// (platform, OS version, locale). The default provides nothing.
  Future<MothPushDeviceMetadata> deviceMetadata() async =>
      const MothPushDeviceMetadata();
}
