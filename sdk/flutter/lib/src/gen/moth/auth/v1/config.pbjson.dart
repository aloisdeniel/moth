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

@$core.Deprecated('Use pushConfigDescriptor instead')
const PushConfig$json = {
  '1': 'PushConfig',
  '2': [
    {'1': 'enabled', '3': 1, '4': 1, '5': 8, '10': 'enabled'},
    {
      '1': 'webpush_vapid_public_key',
      '3': 2,
      '4': 1,
      '5': 9,
      '10': 'webpushVapidPublicKey'
    },
  ],
};

/// Descriptor for `PushConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pushConfigDescriptor = $convert.base64Decode(
    'CgpQdXNoQ29uZmlnEhgKB2VuYWJsZWQYASABKAhSB2VuYWJsZWQSNwoYd2VicHVzaF92YXBpZF'
    '9wdWJsaWNfa2V5GAIgASgJUhV3ZWJwdXNoVmFwaWRQdWJsaWNLZXk=');

@$core.Deprecated('Use themeDescriptor instead')
const Theme$json = {
  '1': 'Theme',
  '2': [
    {'1': 'revision_id', '3': 1, '4': 1, '5': 9, '10': 'revisionId'},
    {
      '1': 'colors',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.ThemeColors',
      '10': 'colors'
    },
    {
      '1': 'dark_colors',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.ThemeColors',
      '10': 'darkColors'
    },
    {'1': 'font_family', '3': 4, '4': 1, '5': 9, '10': 'fontFamily'},
    {'1': 'font_url', '3': 5, '4': 1, '5': 9, '10': 'fontUrl'},
    {'1': 'font_scale', '3': 6, '4': 1, '5': 1, '10': 'fontScale'},
    {'1': 'spacing_unit', '3': 7, '4': 1, '5': 5, '10': 'spacingUnit'},
    {'1': 'corner_radius', '3': 8, '4': 1, '5': 5, '10': 'cornerRadius'},
    {'1': 'logo_light_url', '3': 9, '4': 1, '5': 9, '10': 'logoLightUrl'},
    {'1': 'logo_dark_url', '3': 10, '4': 1, '5': 9, '10': 'logoDarkUrl'},
    {'1': 'terms_url', '3': 11, '4': 1, '5': 9, '10': 'termsUrl'},
    {'1': 'privacy_url', '3': 12, '4': 1, '5': 9, '10': 'privacyUrl'},
  ],
};

/// Descriptor for `Theme`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List themeDescriptor = $convert.base64Decode(
    'CgVUaGVtZRIfCgtyZXZpc2lvbl9pZBgBIAEoCVIKcmV2aXNpb25JZBIxCgZjb2xvcnMYAiABKA'
    'syGS5tb3RoLmF1dGgudjEuVGhlbWVDb2xvcnNSBmNvbG9ycxI6CgtkYXJrX2NvbG9ycxgDIAEo'
    'CzIZLm1vdGguYXV0aC52MS5UaGVtZUNvbG9yc1IKZGFya0NvbG9ycxIfCgtmb250X2ZhbWlseR'
    'gEIAEoCVIKZm9udEZhbWlseRIZCghmb250X3VybBgFIAEoCVIHZm9udFVybBIdCgpmb250X3Nj'
    'YWxlGAYgASgBUglmb250U2NhbGUSIQoMc3BhY2luZ191bml0GAcgASgFUgtzcGFjaW5nVW5pdB'
    'IjCg1jb3JuZXJfcmFkaXVzGAggASgFUgxjb3JuZXJSYWRpdXMSJAoObG9nb19saWdodF91cmwY'
    'CSABKAlSDGxvZ29MaWdodFVybBIiCg1sb2dvX2RhcmtfdXJsGAogASgJUgtsb2dvRGFya1VybB'
    'IbCgl0ZXJtc191cmwYCyABKAlSCHRlcm1zVXJsEh8KC3ByaXZhY3lfdXJsGAwgASgJUgpwcml2'
    'YWN5VXJs');

@$core.Deprecated('Use themeColorsDescriptor instead')
const ThemeColors$json = {
  '1': 'ThemeColors',
  '2': [
    {'1': 'primary', '3': 1, '4': 1, '5': 9, '10': 'primary'},
    {'1': 'on_primary', '3': 2, '4': 1, '5': 9, '10': 'onPrimary'},
    {'1': 'background', '3': 3, '4': 1, '5': 9, '10': 'background'},
    {'1': 'on_background', '3': 4, '4': 1, '5': 9, '10': 'onBackground'},
    {'1': 'surface', '3': 5, '4': 1, '5': 9, '10': 'surface'},
    {'1': 'on_surface', '3': 6, '4': 1, '5': 9, '10': 'onSurface'},
    {'1': 'error', '3': 7, '4': 1, '5': 9, '10': 'error'},
    {'1': 'on_error', '3': 8, '4': 1, '5': 9, '10': 'onError'},
  ],
};

/// Descriptor for `ThemeColors`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List themeColorsDescriptor = $convert.base64Decode(
    'CgtUaGVtZUNvbG9ycxIYCgdwcmltYXJ5GAEgASgJUgdwcmltYXJ5Eh0KCm9uX3ByaW1hcnkYAi'
    'ABKAlSCW9uUHJpbWFyeRIeCgpiYWNrZ3JvdW5kGAMgASgJUgpiYWNrZ3JvdW5kEiMKDW9uX2Jh'
    'Y2tncm91bmQYBCABKAlSDG9uQmFja2dyb3VuZBIYCgdzdXJmYWNlGAUgASgJUgdzdXJmYWNlEh'
    '0KCm9uX3N1cmZhY2UYBiABKAlSCW9uU3VyZmFjZRIUCgVlcnJvchgHIAEoCVIFZXJyb3ISGQoI'
    'b25fZXJyb3IYCCABKAlSB29uRXJyb3I=');

@$core.Deprecated('Use copyDescriptor instead')
const Copy$json = {
  '1': 'Copy',
  '2': [
    {'1': 'copy_revision', '3': 1, '4': 1, '5': 9, '10': 'copyRevision'},
    {'1': 'locale', '3': 2, '4': 1, '5': 9, '10': 'locale'},
    {
      '1': 'messages',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.moth.auth.v1.Copy.MessagesEntry',
      '10': 'messages'
    },
  ],
  '3': [Copy_MessagesEntry$json],
};

@$core.Deprecated('Use copyDescriptor instead')
const Copy_MessagesEntry$json = {
  '1': 'MessagesEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `Copy`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List copyDescriptor = $convert.base64Decode(
    'CgRDb3B5EiMKDWNvcHlfcmV2aXNpb24YASABKAlSDGNvcHlSZXZpc2lvbhIWCgZsb2NhbGUYAi'
    'ABKAlSBmxvY2FsZRI8CghtZXNzYWdlcxgDIAMoCzIgLm1vdGguYXV0aC52MS5Db3B5Lk1lc3Nh'
    'Z2VzRW50cnlSCG1lc3NhZ2VzGjsKDU1lc3NhZ2VzRW50cnkSEAoDa2V5GAEgASgJUgNrZXkSFA'
    'oFdmFsdWUYAiABKAlSBXZhbHVlOgI4AQ==');

@$core.Deprecated('Use getProjectConfigRequestDescriptor instead')
const GetProjectConfigRequest$json = {
  '1': 'GetProjectConfigRequest',
  '2': [
    {
      '1': 'known_theme_revision',
      '3': 1,
      '4': 1,
      '5': 9,
      '10': 'knownThemeRevision'
    },
    {
      '1': 'known_copy_revision',
      '3': 2,
      '4': 1,
      '5': 9,
      '10': 'knownCopyRevision'
    },
  ],
};

/// Descriptor for `GetProjectConfigRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getProjectConfigRequestDescriptor = $convert.base64Decode(
    'ChdHZXRQcm9qZWN0Q29uZmlnUmVxdWVzdBIwChRrbm93bl90aGVtZV9yZXZpc2lvbhgBIAEoCV'
    'ISa25vd25UaGVtZVJldmlzaW9uEi4KE2tub3duX2NvcHlfcmV2aXNpb24YAiABKAlSEWtub3du'
    'Q29weVJldmlzaW9u');

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
    {
      '1': 'theme',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.Theme',
      '10': 'theme'
    },
    {
      '1': 'copy',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.Copy',
      '10': 'copy'
    },
    {
      '1': 'push',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.PushConfig',
      '10': 'push'
    },
  ],
};

/// Descriptor for `GetProjectConfigResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getProjectConfigResponseDescriptor = $convert.base64Decode(
    'ChhHZXRQcm9qZWN0Q29uZmlnUmVzcG9uc2USMgoGZ29vZ2xlGAEgASgLMhoubW90aC5hdXRoLn'
    'YxLkdvb2dsZUNvbmZpZ1IGZ29vZ2xlEi8KBWFwcGxlGAIgASgLMhkubW90aC5hdXRoLnYxLkFw'
    'cGxlQ29uZmlnUgVhcHBsZRIuChNwYXNzd29yZF9taW5fbGVuZ3RoGAMgASgFUhFwYXNzd29yZE'
    '1pbkxlbmd0aBIgCgxzaWduX3VwX29wZW4YBCABKAhSCnNpZ25VcE9wZW4SKQoFdGhlbWUYBSAB'
    'KAsyEy5tb3RoLmF1dGgudjEuVGhlbWVSBXRoZW1lEiYKBGNvcHkYBiABKAsyEi5tb3RoLmF1dG'
    'gudjEuQ29weVIEY29weRIsCgRwdXNoGAcgASgLMhgubW90aC5hdXRoLnYxLlB1c2hDb25maWdS'
    'BHB1c2g=');
