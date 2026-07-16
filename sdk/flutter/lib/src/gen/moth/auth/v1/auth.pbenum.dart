// This is a generated file - do not edit.
//
// Generated from moth/auth/v1/auth.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

/// OAuthProvider identifies a supported social sign-in provider.
/// (buf splits "OAuth" as "O_Auth"; the natural OAUTH_ prefix is kept.)
class OAuthProvider extends $pb.ProtobufEnum {
  /// buf:lint:ignore ENUM_VALUE_PREFIX
  static const OAuthProvider OAUTH_PROVIDER_UNSPECIFIED =
      OAuthProvider._(0, _omitEnumNames ? '' : 'OAUTH_PROVIDER_UNSPECIFIED');

  /// buf:lint:ignore ENUM_VALUE_PREFIX
  static const OAuthProvider OAUTH_PROVIDER_GOOGLE =
      OAuthProvider._(1, _omitEnumNames ? '' : 'OAUTH_PROVIDER_GOOGLE');

  /// buf:lint:ignore ENUM_VALUE_PREFIX
  static const OAuthProvider OAUTH_PROVIDER_APPLE =
      OAuthProvider._(2, _omitEnumNames ? '' : 'OAUTH_PROVIDER_APPLE');

  static const $core.List<OAuthProvider> values = <OAuthProvider>[
    OAUTH_PROVIDER_UNSPECIFIED,
    OAUTH_PROVIDER_GOOGLE,
    OAUTH_PROVIDER_APPLE,
  ];

  static final $core.List<OAuthProvider?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static OAuthProvider? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const OAuthProvider._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
