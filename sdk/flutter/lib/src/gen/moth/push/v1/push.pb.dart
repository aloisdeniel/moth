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
import 'package:protobuf/well_known_types/google/protobuf/timestamp.pb.dart'
    as $1;

import 'push.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'push.pbenum.dart';

/// PushDeviceMetadata is display metadata about the installation, for the
/// admin device panel and sender-side locale targeting. All fields optional.
class PushDeviceMetadata extends $pb.GeneratedMessage {
  factory PushDeviceMetadata({
    $core.String? platform,
    $core.String? model,
    $core.String? osVersion,
    $core.String? appVersion,
    $core.String? locale,
  }) {
    final result = create();
    if (platform != null) result.platform = platform;
    if (model != null) result.model = model;
    if (osVersion != null) result.osVersion = osVersion;
    if (appVersion != null) result.appVersion = appVersion;
    if (locale != null) result.locale = locale;
    return result;
  }

  PushDeviceMetadata._();

  factory PushDeviceMetadata.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PushDeviceMetadata.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PushDeviceMetadata',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.push.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'platform')
    ..aOS(2, _omitFieldNames ? '' : 'model')
    ..aOS(3, _omitFieldNames ? '' : 'osVersion')
    ..aOS(4, _omitFieldNames ? '' : 'appVersion')
    ..aOS(5, _omitFieldNames ? '' : 'locale')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PushDeviceMetadata clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PushDeviceMetadata copyWith(void Function(PushDeviceMetadata) updates) =>
      super.copyWith((message) => updates(message as PushDeviceMetadata))
          as PushDeviceMetadata;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PushDeviceMetadata create() => PushDeviceMetadata._();
  @$core.override
  PushDeviceMetadata createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PushDeviceMetadata getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PushDeviceMetadata>(create);
  static PushDeviceMetadata? _defaultInstance;

  /// OS family ("ios", "android", "web", "macos", ...). Display only — the
  /// API to call lives in PushDevice.target.
  @$pb.TagNumber(1)
  $core.String get platform => $_getSZ(0);
  @$pb.TagNumber(1)
  set platform($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlatform() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlatform() => $_clearField(1);

  /// Device model (e.g. "iPhone16,1", "Pixel 9").
  @$pb.TagNumber(2)
  $core.String get model => $_getSZ(1);
  @$pb.TagNumber(2)
  set model($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasModel() => $_has(1);
  @$pb.TagNumber(2)
  void clearModel() => $_clearField(2);

  /// OS version (e.g. "18.2").
  @$pb.TagNumber(3)
  $core.String get osVersion => $_getSZ(2);
  @$pb.TagNumber(3)
  set osVersion($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasOsVersion() => $_has(2);
  @$pb.TagNumber(3)
  void clearOsVersion() => $_clearField(3);

  /// App version the registration came from (e.g. "2.4.1+87").
  @$pb.TagNumber(4)
  $core.String get appVersion => $_getSZ(3);
  @$pb.TagNumber(4)
  set appVersion($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasAppVersion() => $_has(3);
  @$pb.TagNumber(4)
  void clearAppVersion() => $_clearField(4);

  /// BCP-47 locale of the device (e.g. "fr-FR"), for locale targeting.
  @$pb.TagNumber(5)
  $core.String get locale => $_getSZ(4);
  @$pb.TagNumber(5)
  set locale($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasLocale() => $_has(4);
  @$pb.TagNumber(5)
  void clearLocale() => $_clearField(5);
}

/// PushDevice is one stored registration as the client sees it. The push
/// credential itself is NOT echoed back: tokens are returned only over the
/// secret-key surface (moth.server.v1), and the client already holds its own.
class PushDevice extends $pb.GeneratedMessage {
  factory PushDevice({
    $core.String? id,
    PushTarget? target,
    $core.String? deviceId,
    PushPermission? permission,
    PushDeviceMetadata? metadata,
    $1.Timestamp? createTime,
    $1.Timestamp? updateTime,
    $1.Timestamp? lastSeenTime,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (target != null) result.target = target;
    if (deviceId != null) result.deviceId = deviceId;
    if (permission != null) result.permission = permission;
    if (metadata != null) result.metadata = metadata;
    if (createTime != null) result.createTime = createTime;
    if (updateTime != null) result.updateTime = updateTime;
    if (lastSeenTime != null) result.lastSeenTime = lastSeenTime;
    return result;
  }

  PushDevice._();

  factory PushDevice.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PushDevice.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PushDevice',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.push.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aE<PushTarget>(2, _omitFieldNames ? '' : 'target',
        enumValues: PushTarget.values)
    ..aOS(3, _omitFieldNames ? '' : 'deviceId')
    ..aE<PushPermission>(4, _omitFieldNames ? '' : 'permission',
        enumValues: PushPermission.values)
    ..aOM<PushDeviceMetadata>(5, _omitFieldNames ? '' : 'metadata',
        subBuilder: PushDeviceMetadata.create)
    ..aOM<$1.Timestamp>(6, _omitFieldNames ? '' : 'createTime',
        subBuilder: $1.Timestamp.create)
    ..aOM<$1.Timestamp>(7, _omitFieldNames ? '' : 'updateTime',
        subBuilder: $1.Timestamp.create)
    ..aOM<$1.Timestamp>(8, _omitFieldNames ? '' : 'lastSeenTime',
        subBuilder: $1.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PushDevice clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PushDevice copyWith(void Function(PushDevice) updates) =>
      super.copyWith((message) => updates(message as PushDevice)) as PushDevice;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PushDevice create() => PushDevice._();
  @$core.override
  PushDevice createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PushDevice getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PushDevice>(create);
  static PushDevice? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  PushTarget get target => $_getN(1);
  @$pb.TagNumber(2)
  set target(PushTarget value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTarget() => $_has(1);
  @$pb.TagNumber(2)
  void clearTarget() => $_clearField(2);

  /// The client-generated stable installation id it registered under.
  @$pb.TagNumber(3)
  $core.String get deviceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set deviceId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDeviceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearDeviceId() => $_clearField(3);

  @$pb.TagNumber(4)
  PushPermission get permission => $_getN(3);
  @$pb.TagNumber(4)
  set permission(PushPermission value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasPermission() => $_has(3);
  @$pb.TagNumber(4)
  void clearPermission() => $_clearField(4);

  @$pb.TagNumber(5)
  PushDeviceMetadata get metadata => $_getN(4);
  @$pb.TagNumber(5)
  set metadata(PushDeviceMetadata value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasMetadata() => $_has(4);
  @$pb.TagNumber(5)
  void clearMetadata() => $_clearField(5);
  @$pb.TagNumber(5)
  PushDeviceMetadata ensureMetadata() => $_ensure(4);

  @$pb.TagNumber(6)
  $1.Timestamp get createTime => $_getN(5);
  @$pb.TagNumber(6)
  set createTime($1.Timestamp value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasCreateTime() => $_has(5);
  @$pb.TagNumber(6)
  void clearCreateTime() => $_clearField(6);
  @$pb.TagNumber(6)
  $1.Timestamp ensureCreateTime() => $_ensure(5);

  @$pb.TagNumber(7)
  $1.Timestamp get updateTime => $_getN(6);
  @$pb.TagNumber(7)
  set updateTime($1.Timestamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasUpdateTime() => $_has(6);
  @$pb.TagNumber(7)
  void clearUpdateTime() => $_clearField(7);
  @$pb.TagNumber(7)
  $1.Timestamp ensureUpdateTime() => $_ensure(6);

  /// Refreshed on every RegisterDevice call.
  @$pb.TagNumber(8)
  $1.Timestamp get lastSeenTime => $_getN(7);
  @$pb.TagNumber(8)
  set lastSeenTime($1.Timestamp value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasLastSeenTime() => $_has(7);
  @$pb.TagNumber(8)
  void clearLastSeenTime() => $_clearField(8);
  @$pb.TagNumber(8)
  $1.Timestamp ensureLastSeenTime() => $_ensure(7);
}

class RegisterDeviceRequest extends $pb.GeneratedMessage {
  factory RegisterDeviceRequest({
    PushTarget? target,
    $core.String? token,
    $core.String? deviceId,
    PushPermission? permission,
    PushDeviceMetadata? metadata,
  }) {
    final result = create();
    if (target != null) result.target = target;
    if (token != null) result.token = token;
    if (deviceId != null) result.deviceId = deviceId;
    if (permission != null) result.permission = permission;
    if (metadata != null) result.metadata = metadata;
    return result;
  }

  RegisterDeviceRequest._();

  factory RegisterDeviceRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RegisterDeviceRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RegisterDeviceRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.push.v1'),
      createEmptyInstance: create)
    ..aE<PushTarget>(1, _omitFieldNames ? '' : 'target',
        enumValues: PushTarget.values)
    ..aOS(2, _omitFieldNames ? '' : 'token')
    ..aOS(3, _omitFieldNames ? '' : 'deviceId')
    ..aE<PushPermission>(4, _omitFieldNames ? '' : 'permission',
        enumValues: PushPermission.values)
    ..aOM<PushDeviceMetadata>(5, _omitFieldNames ? '' : 'metadata',
        subBuilder: PushDeviceMetadata.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegisterDeviceRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegisterDeviceRequest copyWith(
          void Function(RegisterDeviceRequest) updates) =>
      super.copyWith((message) => updates(message as RegisterDeviceRequest))
          as RegisterDeviceRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RegisterDeviceRequest create() => RegisterDeviceRequest._();
  @$core.override
  RegisterDeviceRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RegisterDeviceRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RegisterDeviceRequest>(create);
  static RegisterDeviceRequest? _defaultInstance;

  /// Which push service `token` belongs to. Required.
  @$pb.TagNumber(1)
  PushTarget get target => $_getN(0);
  @$pb.TagNumber(1)
  set target(PushTarget value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasTarget() => $_has(0);
  @$pb.TagNumber(1)
  void clearTarget() => $_clearField(1);

  /// The push credential: APNs device token, FCM registration token, or the
  /// serialized Web Push subscription (endpoint + keys). Required.
  @$pb.TagNumber(2)
  $core.String get token => $_getSZ(1);
  @$pb.TagNumber(2)
  set token($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasToken() => $_has(1);
  @$pb.TagNumber(2)
  void clearToken() => $_clearField(2);

  /// Client-generated stable installation id, so one physical device
  /// re-registering replaces its own row instead of accumulating. Required.
  @$pb.TagNumber(3)
  $core.String get deviceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set deviceId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDeviceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearDeviceId() => $_clearField(3);

  /// The OS-level permission state the client observed.
  @$pb.TagNumber(4)
  PushPermission get permission => $_getN(3);
  @$pb.TagNumber(4)
  set permission(PushPermission value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasPermission() => $_has(3);
  @$pb.TagNumber(4)
  void clearPermission() => $_clearField(4);

  @$pb.TagNumber(5)
  PushDeviceMetadata get metadata => $_getN(4);
  @$pb.TagNumber(5)
  set metadata(PushDeviceMetadata value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasMetadata() => $_has(4);
  @$pb.TagNumber(5)
  void clearMetadata() => $_clearField(5);
  @$pb.TagNumber(5)
  PushDeviceMetadata ensureMetadata() => $_ensure(4);
}

class RegisterDeviceResponse extends $pb.GeneratedMessage {
  factory RegisterDeviceResponse({
    PushDevice? device,
  }) {
    final result = create();
    if (device != null) result.device = device;
    return result;
  }

  RegisterDeviceResponse._();

  factory RegisterDeviceResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RegisterDeviceResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RegisterDeviceResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.push.v1'),
      createEmptyInstance: create)
    ..aOM<PushDevice>(1, _omitFieldNames ? '' : 'device',
        subBuilder: PushDevice.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegisterDeviceResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegisterDeviceResponse copyWith(
          void Function(RegisterDeviceResponse) updates) =>
      super.copyWith((message) => updates(message as RegisterDeviceResponse))
          as RegisterDeviceResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RegisterDeviceResponse create() => RegisterDeviceResponse._();
  @$core.override
  RegisterDeviceResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RegisterDeviceResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RegisterDeviceResponse>(create);
  static RegisterDeviceResponse? _defaultInstance;

  /// The stored registration (created or upserted).
  @$pb.TagNumber(1)
  PushDevice get device => $_getN(0);
  @$pb.TagNumber(1)
  set device(PushDevice value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasDevice() => $_has(0);
  @$pb.TagNumber(1)
  void clearDevice() => $_clearField(1);
  @$pb.TagNumber(1)
  PushDevice ensureDevice() => $_ensure(0);
}

class UnregisterDeviceRequest extends $pb.GeneratedMessage {
  factory UnregisterDeviceRequest({
    $core.String? deviceId,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    return result;
  }

  UnregisterDeviceRequest._();

  factory UnregisterDeviceRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UnregisterDeviceRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UnregisterDeviceRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.push.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UnregisterDeviceRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UnregisterDeviceRequest copyWith(
          void Function(UnregisterDeviceRequest) updates) =>
      super.copyWith((message) => updates(message as UnregisterDeviceRequest))
          as UnregisterDeviceRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UnregisterDeviceRequest create() => UnregisterDeviceRequest._();
  @$core.override
  UnregisterDeviceRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UnregisterDeviceRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UnregisterDeviceRequest>(create);
  static UnregisterDeviceRequest? _defaultInstance;

  /// The installation id passed to RegisterDevice.
  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);
}

class UnregisterDeviceResponse extends $pb.GeneratedMessage {
  factory UnregisterDeviceResponse() => create();

  UnregisterDeviceResponse._();

  factory UnregisterDeviceResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UnregisterDeviceResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UnregisterDeviceResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.push.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UnregisterDeviceResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UnregisterDeviceResponse copyWith(
          void Function(UnregisterDeviceResponse) updates) =>
      super.copyWith((message) => updates(message as UnregisterDeviceResponse))
          as UnregisterDeviceResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UnregisterDeviceResponse create() => UnregisterDeviceResponse._();
  @$core.override
  UnregisterDeviceResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UnregisterDeviceResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UnregisterDeviceResponse>(create);
  static UnregisterDeviceResponse? _defaultInstance;
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
