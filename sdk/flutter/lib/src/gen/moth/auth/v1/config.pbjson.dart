// This is a generated file - do not edit.
//
// Generated from moth/auth/v1/config.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports
// ignore_for_file: unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use googleConfigDescriptor instead')
const GoogleConfig$json = {
  '1': 'GoogleConfig',
  '2': [
    {'1': 'enabled', '3': 1, '4': 1, '5': 8, '10': 'enabled'},
    {'1': 'web_client_id', '3': 2, '4': 1, '5': 9, '10': 'webClientId'},
    {'1': 'ios_client_id', '3': 3, '4': 1, '5': 9, '10': 'iosClientId'},
    {'1': 'android_client_id', '3': 4, '4': 1, '5': 9, '10': 'androidClientId'},
  ],
};

/// Descriptor for `GoogleConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List googleConfigDescriptor = $convert.base64Decode(
    'CgxHb29nbGVDb25maWcSGAoHZW5hYmxlZBgBIAEoCFIHZW5hYmxlZBIiCg13ZWJfY2xpZW50X2'
    'lkGAIgASgJUgt3ZWJDbGllbnRJZBIiCg1pb3NfY2xpZW50X2lkGAMgASgJUgtpb3NDbGllbnRJ'
    'ZBIqChFhbmRyb2lkX2NsaWVudF9pZBgEIAEoCVIPYW5kcm9pZENsaWVudElk');

@$core.Deprecated('Use appleConfigDescriptor instead')
const AppleConfig$json = {
  '1': 'AppleConfig',
  '2': [
    {'1': 'enabled', '3': 1, '4': 1, '5': 8, '10': 'enabled'},
  ],
};

/// Descriptor for `AppleConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List appleConfigDescriptor = $convert
    .base64Decode('CgtBcHBsZUNvbmZpZxIYCgdlbmFibGVkGAEgASgIUgdlbmFibGVk');

@$core.Deprecated('Use getProjectConfigRequestDescriptor instead')
const GetProjectConfigRequest$json = {
  '1': 'GetProjectConfigRequest',
};

/// Descriptor for `GetProjectConfigRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getProjectConfigRequestDescriptor =
    $convert.base64Decode('ChdHZXRQcm9qZWN0Q29uZmlnUmVxdWVzdA==');

@$core.Deprecated('Use getProjectConfigResponseDescriptor instead')
const GetProjectConfigResponse$json = {
  '1': 'GetProjectConfigResponse',
  '2': [
    {
      '1': 'google',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.GoogleConfig',
      '10': 'google'
    },
    {
      '1': 'apple',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.AppleConfig',
      '10': 'apple'
    },
    {
      '1': 'password_min_length',
      '3': 3,
      '4': 1,
      '5': 5,
      '10': 'passwordMinLength'
    },
    {'1': 'sign_up_open', '3': 4, '4': 1, '5': 8, '10': 'signUpOpen'},
  ],
};

/// Descriptor for `GetProjectConfigResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getProjectConfigResponseDescriptor = $convert.base64Decode(
    'ChhHZXRQcm9qZWN0Q29uZmlnUmVzcG9uc2USMgoGZ29vZ2xlGAEgASgLMhoubW90aC5hdXRoLn'
    'YxLkdvb2dsZUNvbmZpZ1IGZ29vZ2xlEi8KBWFwcGxlGAIgASgLMhkubW90aC5hdXRoLnYxLkFw'
    'cGxlQ29uZmlnUgVhcHBsZRIuChNwYXNzd29yZF9taW5fbGVuZ3RoGAMgASgFUhFwYXNzd29yZE'
    '1pbkxlbmd0aBIgCgxzaWduX3VwX29wZW4YBCABKAhSCnNpZ25VcE9wZW4=');
