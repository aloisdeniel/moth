import 'gen/moth/push/v1/push.pbenum.dart' as pbe;

/// Which push service a device credential belongs to — i.e. which API a
/// sender must call to reach the device. An iOS app using FCM registers as
/// [fcm]; the OS is display metadata, not the target.
enum MothPushTarget {
  apns,
  fcm,
  webpush;

  pbe.PushTarget get proto => switch (this) {
    MothPushTarget.apns => pbe.PushTarget.PUSH_TARGET_APNS,
    MothPushTarget.fcm => pbe.PushTarget.PUSH_TARGET_FCM,
    MothPushTarget.webpush => pbe.PushTarget.PUSH_TARGET_WEBPUSH,
  };

  static MothPushTarget? fromProto(pbe.PushTarget target) => switch (target) {
    pbe.PushTarget.PUSH_TARGET_APNS => MothPushTarget.apns,
    pbe.PushTarget.PUSH_TARGET_FCM => MothPushTarget.fcm,
    pbe.PushTarget.PUSH_TARGET_WEBPUSH => MothPushTarget.webpush,
    _ => null,
  };
}

/// The OS-level notification-permission state, as last reported by the
/// platform. A device with a token but [denied] permission stays registered
/// (data pushes may still work) — senders decide what to skip.
enum MothPushPermission {
  granted,

  /// iOS provisional authorization (quiet notifications, no prompt shown).
  provisional,
  denied,

  /// Not determined yet, or the platform cannot report it.
  unknown;

  pbe.PushPermission get proto => switch (this) {
    MothPushPermission.granted => pbe.PushPermission.PUSH_PERMISSION_GRANTED,
    MothPushPermission.provisional =>
      pbe.PushPermission.PUSH_PERMISSION_PROVISIONAL,
    MothPushPermission.denied => pbe.PushPermission.PUSH_PERMISSION_DENIED,
    MothPushPermission.unknown => pbe.PushPermission.PUSH_PERMISSION_UNKNOWN,
  };

  static MothPushPermission fromProto(pbe.PushPermission permission) =>
      switch (permission) {
        pbe.PushPermission.PUSH_PERMISSION_GRANTED =>
          MothPushPermission.granted,
        pbe.PushPermission.PUSH_PERMISSION_PROVISIONAL =>
          MothPushPermission.provisional,
        pbe.PushPermission.PUSH_PERMISSION_DENIED => MothPushPermission.denied,
        _ => MothPushPermission.unknown,
      };
}

/// One push credential as produced by a [MothPushAdapter]: the [target]
/// service it belongs to and the raw [token] (APNs device token, FCM
/// registration token, or a serialized Web Push subscription).
class MothPushToken {
  const MothPushToken({required this.target, required this.token});

  final MothPushTarget target;
  final String token;

  @override
  bool operator ==(Object other) =>
      other is MothPushToken && other.target == target && other.token == token;

  @override
  int get hashCode => Object.hash(target, token);

  @override
  String toString() => 'MothPushToken($target)';
}

/// Display metadata sent along with a registration, shown in the admin
/// Devices panel and available for sender-side locale targeting. All fields
/// are optional; empty strings are simply omitted server-side.
class MothPushDeviceMetadata {
  const MothPushDeviceMetadata({
    this.platform = '',
    this.model = '',
    this.osVersion = '',
    this.appVersion = '',
    this.locale = '',
  });

  /// OS family (`ios`, `android`, `web`, ...). Display only — the API to
  /// call lives in the registration's target.
  final String platform;

  /// Device model (e.g. `iPhone16,1`, `Pixel 9`).
  final String model;

  /// OS version string.
  final String osVersion;

  /// App version the registration came from (e.g. `2.4.1+87`).
  final String appVersion;

  /// BCP-47 locale of the device (e.g. `fr-FR`).
  final String locale;
}

/// The push machinery's state for settings screens, exposed as
/// `MothScope.of(context).pushStatus`.
///
/// [available] is false when no [MothPushAdapter] is wired into [MothApp] or
/// the project has push disabled — the other fields are then meaningless.
/// [registered] flips true once this installation's registration reached the
/// server in the current session.
class MothPushStatus {
  const MothPushStatus({
    this.available = false,
    this.permission = MothPushPermission.unknown,
    this.registered = false,
  });

  /// No adapter wired, or the project has push disabled.
  static const MothPushStatus unavailable = MothPushStatus();

  final bool available;
  final MothPushPermission permission;
  final bool registered;

  MothPushStatus copyWith({
    bool? available,
    MothPushPermission? permission,
    bool? registered,
  }) => MothPushStatus(
    available: available ?? this.available,
    permission: permission ?? this.permission,
    registered: registered ?? this.registered,
  );

  @override
  bool operator ==(Object other) =>
      other is MothPushStatus &&
      other.available == available &&
      other.permission == permission &&
      other.registered == registered;

  @override
  int get hashCode => Object.hash(available, permission, registered);

  @override
  String toString() =>
      'MothPushStatus(available: $available, permission: $permission, '
      'registered: $registered)';
}
