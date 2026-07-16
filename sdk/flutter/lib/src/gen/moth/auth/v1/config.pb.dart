// This is a generated file - do not edit.
//
// Generated from moth/auth/v1/config.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

/// GoogleConfig is the public part of a project's Sign in with Google
/// configuration.
class GoogleConfig extends $pb.GeneratedMessage {
  factory GoogleConfig({
    $core.bool? enabled,
    $core.String? webClientId,
    $core.String? iosClientId,
    $core.String? androidClientId,
  }) {
    final result = create();
    if (enabled != null) result.enabled = enabled;
    if (webClientId != null) result.webClientId = webClientId;
    if (iosClientId != null) result.iosClientId = iosClientId;
    if (androidClientId != null) result.androidClientId = androidClientId;
    return result;
  }

  GoogleConfig._();

  factory GoogleConfig.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GoogleConfig.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GoogleConfig',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.auth.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'enabled')
    ..aOS(2, _omitFieldNames ? '' : 'webClientId')
    ..aOS(3, _omitFieldNames ? '' : 'iosClientId')
    ..aOS(4, _omitFieldNames ? '' : 'androidClientId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GoogleConfig clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GoogleConfig copyWith(void Function(GoogleConfig) updates) =>
      super.copyWith((message) => updates(message as GoogleConfig))
          as GoogleConfig;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GoogleConfig create() => GoogleConfig._();
  @$core.override
  GoogleConfig createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GoogleConfig getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GoogleConfig>(create);
  static GoogleConfig? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get enabled => $_getBF(0);
  @$pb.TagNumber(1)
  set enabled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEnabled() => $_has(0);
  @$pb.TagNumber(1)
  void clearEnabled() => $_clearField(1);

  /// OAuth client IDs the native flows initialize with. Client IDs are
  /// public values (the secret never leaves the server).
  @$pb.TagNumber(2)
  $core.String get webClientId => $_getSZ(1);
  @$pb.TagNumber(2)
  set webClientId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasWebClientId() => $_has(1);
  @$pb.TagNumber(2)
  void clearWebClientId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get iosClientId => $_getSZ(2);
  @$pb.TagNumber(3)
  set iosClientId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasIosClientId() => $_has(2);
  @$pb.TagNumber(3)
  void clearIosClientId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get androidClientId => $_getSZ(3);
  @$pb.TagNumber(4)
  set androidClientId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasAndroidClientId() => $_has(3);
  @$pb.TagNumber(4)
  void clearAndroidClientId() => $_clearField(4);
}

/// AppleConfig is the public part of a project's Sign in with Apple
/// configuration.
class AppleConfig extends $pb.GeneratedMessage {
  factory AppleConfig({
    $core.bool? enabled,
  }) {
    final result = create();
    if (enabled != null) result.enabled = enabled;
    return result;
  }

  AppleConfig._();

  factory AppleConfig.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AppleConfig.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AppleConfig',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.auth.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'enabled')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AppleConfig clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AppleConfig copyWith(void Function(AppleConfig) updates) =>
      super.copyWith((message) => updates(message as AppleConfig))
          as AppleConfig;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AppleConfig create() => AppleConfig._();
  @$core.override
  AppleConfig createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AppleConfig getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AppleConfig>(create);
  static AppleConfig? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get enabled => $_getBF(0);
  @$pb.TagNumber(1)
  set enabled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEnabled() => $_has(0);
  @$pb.TagNumber(1)
  void clearEnabled() => $_clearField(1);
}

class GetProjectConfigRequest extends $pb.GeneratedMessage {
  factory GetProjectConfigRequest() => create();

  GetProjectConfigRequest._();

  factory GetProjectConfigRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetProjectConfigRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetProjectConfigRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.auth.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProjectConfigRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProjectConfigRequest copyWith(
          void Function(GetProjectConfigRequest) updates) =>
      super.copyWith((message) => updates(message as GetProjectConfigRequest))
          as GetProjectConfigRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetProjectConfigRequest create() => GetProjectConfigRequest._();
  @$core.override
  GetProjectConfigRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetProjectConfigRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetProjectConfigRequest>(create);
  static GetProjectConfigRequest? _defaultInstance;
}

class GetProjectConfigResponse extends $pb.GeneratedMessage {
  factory GetProjectConfigResponse({
    GoogleConfig? google,
    AppleConfig? apple,
    $core.int? passwordMinLength,
    $core.bool? signUpOpen,
  }) {
    final result = create();
    if (google != null) result.google = google;
    if (apple != null) result.apple = apple;
    if (passwordMinLength != null) result.passwordMinLength = passwordMinLength;
    if (signUpOpen != null) result.signUpOpen = signUpOpen;
    return result;
  }

  GetProjectConfigResponse._();

  factory GetProjectConfigResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetProjectConfigResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetProjectConfigResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.auth.v1'),
      createEmptyInstance: create)
    ..aOM<GoogleConfig>(1, _omitFieldNames ? '' : 'google',
        subBuilder: GoogleConfig.create)
    ..aOM<AppleConfig>(2, _omitFieldNames ? '' : 'apple',
        subBuilder: AppleConfig.create)
    ..aI(3, _omitFieldNames ? '' : 'passwordMinLength')
    ..aOB(4, _omitFieldNames ? '' : 'signUpOpen')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProjectConfigResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProjectConfigResponse copyWith(
          void Function(GetProjectConfigResponse) updates) =>
      super.copyWith((message) => updates(message as GetProjectConfigResponse))
          as GetProjectConfigResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetProjectConfigResponse create() => GetProjectConfigResponse._();
  @$core.override
  GetProjectConfigResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetProjectConfigResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetProjectConfigResponse>(create);
  static GetProjectConfigResponse? _defaultInstance;

  @$pb.TagNumber(1)
  GoogleConfig get google => $_getN(0);
  @$pb.TagNumber(1)
  set google(GoogleConfig value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasGoogle() => $_has(0);
  @$pb.TagNumber(1)
  void clearGoogle() => $_clearField(1);
  @$pb.TagNumber(1)
  GoogleConfig ensureGoogle() => $_ensure(0);

  @$pb.TagNumber(2)
  AppleConfig get apple => $_getN(1);
  @$pb.TagNumber(2)
  set apple(AppleConfig value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasApple() => $_has(1);
  @$pb.TagNumber(2)
  void clearApple() => $_clearField(2);
  @$pb.TagNumber(2)
  AppleConfig ensureApple() => $_ensure(1);

  /// Minimum accepted password length.
  @$pb.TagNumber(3)
  $core.int get passwordMinLength => $_getIZ(2);
  @$pb.TagNumber(3)
  set passwordMinLength($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPasswordMinLength() => $_has(2);
  @$pb.TagNumber(3)
  void clearPasswordMinLength() => $_clearField(3);

  /// Whether the public SignUp RPC is open.
  @$pb.TagNumber(4)
  $core.bool get signUpOpen => $_getBF(3);
  @$pb.TagNumber(4)
  set signUpOpen($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSignUpOpen() => $_has(3);
  @$pb.TagNumber(4)
  void clearSignUpOpen() => $_clearField(4);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
