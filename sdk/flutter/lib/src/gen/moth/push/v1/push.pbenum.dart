// This is a generated file - do not edit.
//
// Generated from moth/push/v1/push.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

/// PushTarget says which push service the credential belongs to — i.e. which
/// API the developer's backend must call to reach the device. Deliberately not
/// the platform: an iOS app using FCM registers as PUSH_TARGET_FCM; the OS
/// lives in PushDeviceMetadata.platform as display metadata.
class PushTarget extends $pb.ProtobufEnum {
  static const PushTarget PUSH_TARGET_UNSPECIFIED =
      PushTarget._(0, _omitEnumNames ? '' : 'PUSH_TARGET_UNSPECIFIED');

  /// Apple Push Notification service; token is the APNs device token.
  static const PushTarget PUSH_TARGET_APNS =
      PushTarget._(1, _omitEnumNames ? '' : 'PUSH_TARGET_APNS');

  /// Firebase Cloud Messaging; token is the FCM registration token.
  static const PushTarget PUSH_TARGET_FCM =
      PushTarget._(2, _omitEnumNames ? '' : 'PUSH_TARGET_FCM');

  /// Web Push; token is the serialized subscription (endpoint + keys).
  static const PushTarget PUSH_TARGET_WEBPUSH =
      PushTarget._(3, _omitEnumNames ? '' : 'PUSH_TARGET_WEBPUSH');

  static const $core.List<PushTarget> values = <PushTarget>[
    PUSH_TARGET_UNSPECIFIED,
    PUSH_TARGET_APNS,
    PUSH_TARGET_FCM,
    PUSH_TARGET_WEBPUSH,
  ];

  static final $core.List<PushTarget?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static PushTarget? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const PushTarget._(super.value, super.name);
}

/// PushPermission is the OS-level notification-permission state the client
/// last reported. A registration with a token but DENIED permission is kept
/// (data pushes may still work) but flagged, so senders can skip alert pushes.
class PushPermission extends $pb.ProtobufEnum {
  static const PushPermission PUSH_PERMISSION_UNSPECIFIED =
      PushPermission._(0, _omitEnumNames ? '' : 'PUSH_PERMISSION_UNSPECIFIED');
  static const PushPermission PUSH_PERMISSION_GRANTED =
      PushPermission._(1, _omitEnumNames ? '' : 'PUSH_PERMISSION_GRANTED');

  /// iOS provisional authorization (quiet notifications).
  static const PushPermission PUSH_PERMISSION_PROVISIONAL =
      PushPermission._(2, _omitEnumNames ? '' : 'PUSH_PERMISSION_PROVISIONAL');
  static const PushPermission PUSH_PERMISSION_DENIED =
      PushPermission._(3, _omitEnumNames ? '' : 'PUSH_PERMISSION_DENIED');

  /// The client could not determine the permission state.
  static const PushPermission PUSH_PERMISSION_UNKNOWN =
      PushPermission._(4, _omitEnumNames ? '' : 'PUSH_PERMISSION_UNKNOWN');

  static const $core.List<PushPermission> values = <PushPermission>[
    PUSH_PERMISSION_UNSPECIFIED,
    PUSH_PERMISSION_GRANTED,
    PUSH_PERMISSION_PROVISIONAL,
    PUSH_PERMISSION_DENIED,
    PUSH_PERMISSION_UNKNOWN,
  ];

  static final $core.List<PushPermission?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 4);
  static PushPermission? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const PushPermission._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
