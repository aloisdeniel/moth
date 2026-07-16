// This is a generated file - do not edit.
//
// Generated from moth/auth/v1/auth.proto.

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

@$core.Deprecated('Use oAuthProviderDescriptor instead')
const OAuthProvider$json = {
  '1': 'OAuthProvider',
  '2': [
    {'1': 'OAUTH_PROVIDER_UNSPECIFIED', '2': 0},
    {'1': 'OAUTH_PROVIDER_GOOGLE', '2': 1},
    {'1': 'OAUTH_PROVIDER_APPLE', '2': 2},
  ],
};

/// Descriptor for `OAuthProvider`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List oAuthProviderDescriptor = $convert.base64Decode(
    'Cg1PQXV0aFByb3ZpZGVyEh4KGk9BVVRIX1BST1ZJREVSX1VOU1BFQ0lGSUVEEAASGQoVT0FVVE'
    'hfUFJPVklERVJfR09PR0xFEAESGAoUT0FVVEhfUFJPVklERVJfQVBQTEUQAg==');

@$core.Deprecated('Use userDescriptor instead')
const User$json = {
  '1': 'User',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'email', '3': 2, '4': 1, '5': 9, '10': 'email'},
    {'1': 'email_verified', '3': 3, '4': 1, '5': 8, '10': 'emailVerified'},
    {'1': 'display_name', '3': 4, '4': 1, '5': 9, '10': 'displayName'},
    {'1': 'avatar_url', '3': 5, '4': 1, '5': 9, '10': 'avatarUrl'},
    {
      '1': 'create_time',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'createTime'
    },
  ],
};

/// Descriptor for `User`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List userDescriptor = $convert.base64Decode(
    'CgRVc2VyEg4KAmlkGAEgASgJUgJpZBIUCgVlbWFpbBgCIAEoCVIFZW1haWwSJQoOZW1haWxfdm'
    'VyaWZpZWQYAyABKAhSDWVtYWlsVmVyaWZpZWQSIQoMZGlzcGxheV9uYW1lGAQgASgJUgtkaXNw'
    'bGF5TmFtZRIdCgphdmF0YXJfdXJsGAUgASgJUglhdmF0YXJVcmwSOwoLY3JlYXRlX3RpbWUYBi'
    'ABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUgpjcmVhdGVUaW1l');

@$core.Deprecated('Use tokenPairDescriptor instead')
const TokenPair$json = {
  '1': 'TokenPair',
  '2': [
    {'1': 'access_token', '3': 1, '4': 1, '5': 9, '10': 'accessToken'},
    {'1': 'refresh_token', '3': 2, '4': 1, '5': 9, '10': 'refreshToken'},
    {'1': 'expires_in', '3': 3, '4': 1, '5': 3, '10': 'expiresIn'},
  ],
};

/// Descriptor for `TokenPair`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List tokenPairDescriptor = $convert.base64Decode(
    'CglUb2tlblBhaXISIQoMYWNjZXNzX3Rva2VuGAEgASgJUgthY2Nlc3NUb2tlbhIjCg1yZWZyZX'
    'NoX3Rva2VuGAIgASgJUgxyZWZyZXNoVG9rZW4SHQoKZXhwaXJlc19pbhgDIAEoA1IJZXhwaXJl'
    'c0lu');

@$core.Deprecated('Use signUpRequestDescriptor instead')
const SignUpRequest$json = {
  '1': 'SignUpRequest',
  '2': [
    {'1': 'email', '3': 1, '4': 1, '5': 9, '10': 'email'},
    {'1': 'password', '3': 2, '4': 1, '5': 9, '10': 'password'},
    {'1': 'display_name', '3': 3, '4': 1, '5': 9, '10': 'displayName'},
    {'1': 'device_info', '3': 4, '4': 1, '5': 9, '10': 'deviceInfo'},
  ],
};

/// Descriptor for `SignUpRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List signUpRequestDescriptor = $convert.base64Decode(
    'Cg1TaWduVXBSZXF1ZXN0EhQKBWVtYWlsGAEgASgJUgVlbWFpbBIaCghwYXNzd29yZBgCIAEoCV'
    'IIcGFzc3dvcmQSIQoMZGlzcGxheV9uYW1lGAMgASgJUgtkaXNwbGF5TmFtZRIfCgtkZXZpY2Vf'
    'aW5mbxgEIAEoCVIKZGV2aWNlSW5mbw==');

@$core.Deprecated('Use signUpResponseDescriptor instead')
const SignUpResponse$json = {
  '1': 'SignUpResponse',
  '2': [
    {
      '1': 'user',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.User',
      '10': 'user'
    },
    {
      '1': 'tokens',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.TokenPair',
      '10': 'tokens'
    },
  ],
};

/// Descriptor for `SignUpResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List signUpResponseDescriptor = $convert.base64Decode(
    'Cg5TaWduVXBSZXNwb25zZRImCgR1c2VyGAEgASgLMhIubW90aC5hdXRoLnYxLlVzZXJSBHVzZX'
    'ISLwoGdG9rZW5zGAIgASgLMhcubW90aC5hdXRoLnYxLlRva2VuUGFpclIGdG9rZW5z');

@$core.Deprecated('Use signInRequestDescriptor instead')
const SignInRequest$json = {
  '1': 'SignInRequest',
  '2': [
    {'1': 'email', '3': 1, '4': 1, '5': 9, '10': 'email'},
    {'1': 'password', '3': 2, '4': 1, '5': 9, '10': 'password'},
    {'1': 'device_info', '3': 3, '4': 1, '5': 9, '10': 'deviceInfo'},
  ],
};

/// Descriptor for `SignInRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List signInRequestDescriptor = $convert.base64Decode(
    'Cg1TaWduSW5SZXF1ZXN0EhQKBWVtYWlsGAEgASgJUgVlbWFpbBIaCghwYXNzd29yZBgCIAEoCV'
    'IIcGFzc3dvcmQSHwoLZGV2aWNlX2luZm8YAyABKAlSCmRldmljZUluZm8=');

@$core.Deprecated('Use signInResponseDescriptor instead')
const SignInResponse$json = {
  '1': 'SignInResponse',
  '2': [
    {
      '1': 'user',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.User',
      '10': 'user'
    },
    {
      '1': 'tokens',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.TokenPair',
      '10': 'tokens'
    },
  ],
};

/// Descriptor for `SignInResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List signInResponseDescriptor = $convert.base64Decode(
    'Cg5TaWduSW5SZXNwb25zZRImCgR1c2VyGAEgASgLMhIubW90aC5hdXRoLnYxLlVzZXJSBHVzZX'
    'ISLwoGdG9rZW5zGAIgASgLMhcubW90aC5hdXRoLnYxLlRva2VuUGFpclIGdG9rZW5z');

@$core.Deprecated('Use refreshTokenRequestDescriptor instead')
const RefreshTokenRequest$json = {
  '1': 'RefreshTokenRequest',
  '2': [
    {'1': 'refresh_token', '3': 1, '4': 1, '5': 9, '10': 'refreshToken'},
  ],
};

/// Descriptor for `RefreshTokenRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List refreshTokenRequestDescriptor = $convert.base64Decode(
    'ChNSZWZyZXNoVG9rZW5SZXF1ZXN0EiMKDXJlZnJlc2hfdG9rZW4YASABKAlSDHJlZnJlc2hUb2'
    'tlbg==');

@$core.Deprecated('Use refreshTokenResponseDescriptor instead')
const RefreshTokenResponse$json = {
  '1': 'RefreshTokenResponse',
  '2': [
    {
      '1': 'user',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.User',
      '10': 'user'
    },
    {
      '1': 'tokens',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.TokenPair',
      '10': 'tokens'
    },
  ],
};

/// Descriptor for `RefreshTokenResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List refreshTokenResponseDescriptor = $convert.base64Decode(
    'ChRSZWZyZXNoVG9rZW5SZXNwb25zZRImCgR1c2VyGAEgASgLMhIubW90aC5hdXRoLnYxLlVzZX'
    'JSBHVzZXISLwoGdG9rZW5zGAIgASgLMhcubW90aC5hdXRoLnYxLlRva2VuUGFpclIGdG9rZW5z');

@$core.Deprecated('Use signOutRequestDescriptor instead')
const SignOutRequest$json = {
  '1': 'SignOutRequest',
  '2': [
    {'1': 'refresh_token', '3': 1, '4': 1, '5': 9, '10': 'refreshToken'},
    {'1': 'all_devices', '3': 2, '4': 1, '5': 8, '10': 'allDevices'},
  ],
};

/// Descriptor for `SignOutRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List signOutRequestDescriptor = $convert.base64Decode(
    'Cg5TaWduT3V0UmVxdWVzdBIjCg1yZWZyZXNoX3Rva2VuGAEgASgJUgxyZWZyZXNoVG9rZW4SHw'
    'oLYWxsX2RldmljZXMYAiABKAhSCmFsbERldmljZXM=');

@$core.Deprecated('Use signOutResponseDescriptor instead')
const SignOutResponse$json = {
  '1': 'SignOutResponse',
};

/// Descriptor for `SignOutResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List signOutResponseDescriptor =
    $convert.base64Decode('Cg9TaWduT3V0UmVzcG9uc2U=');

@$core.Deprecated('Use getMeRequestDescriptor instead')
const GetMeRequest$json = {
  '1': 'GetMeRequest',
};

/// Descriptor for `GetMeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getMeRequestDescriptor =
    $convert.base64Decode('CgxHZXRNZVJlcXVlc3Q=');

@$core.Deprecated('Use getMeResponseDescriptor instead')
const GetMeResponse$json = {
  '1': 'GetMeResponse',
  '2': [
    {
      '1': 'user',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.User',
      '10': 'user'
    },
  ],
};

/// Descriptor for `GetMeResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getMeResponseDescriptor = $convert.base64Decode(
    'Cg1HZXRNZVJlc3BvbnNlEiYKBHVzZXIYASABKAsyEi5tb3RoLmF1dGgudjEuVXNlclIEdXNlcg'
    '==');

@$core.Deprecated('Use updateMeRequestDescriptor instead')
const UpdateMeRequest$json = {
  '1': 'UpdateMeRequest',
  '2': [
    {
      '1': 'display_name',
      '3': 1,
      '4': 1,
      '5': 9,
      '9': 0,
      '10': 'displayName',
      '17': true
    },
    {
      '1': 'avatar_url',
      '3': 2,
      '4': 1,
      '5': 9,
      '9': 1,
      '10': 'avatarUrl',
      '17': true
    },
  ],
  '8': [
    {'1': '_display_name'},
    {'1': '_avatar_url'},
  ],
};

/// Descriptor for `UpdateMeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List updateMeRequestDescriptor = $convert.base64Decode(
    'Cg9VcGRhdGVNZVJlcXVlc3QSJgoMZGlzcGxheV9uYW1lGAEgASgJSABSC2Rpc3BsYXlOYW1liA'
    'EBEiIKCmF2YXRhcl91cmwYAiABKAlIAVIJYXZhdGFyVXJsiAEBQg8KDV9kaXNwbGF5X25hbWVC'
    'DQoLX2F2YXRhcl91cmw=');

@$core.Deprecated('Use updateMeResponseDescriptor instead')
const UpdateMeResponse$json = {
  '1': 'UpdateMeResponse',
  '2': [
    {
      '1': 'user',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.User',
      '10': 'user'
    },
  ],
};

/// Descriptor for `UpdateMeResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List updateMeResponseDescriptor = $convert.base64Decode(
    'ChBVcGRhdGVNZVJlc3BvbnNlEiYKBHVzZXIYASABKAsyEi5tb3RoLmF1dGgudjEuVXNlclIEdX'
    'Nlcg==');

@$core.Deprecated('Use changePasswordRequestDescriptor instead')
const ChangePasswordRequest$json = {
  '1': 'ChangePasswordRequest',
  '2': [
    {'1': 'current_password', '3': 1, '4': 1, '5': 9, '10': 'currentPassword'},
    {'1': 'new_password', '3': 2, '4': 1, '5': 9, '10': 'newPassword'},
  ],
};

/// Descriptor for `ChangePasswordRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List changePasswordRequestDescriptor = $convert.base64Decode(
    'ChVDaGFuZ2VQYXNzd29yZFJlcXVlc3QSKQoQY3VycmVudF9wYXNzd29yZBgBIAEoCVIPY3Vycm'
    'VudFBhc3N3b3JkEiEKDG5ld19wYXNzd29yZBgCIAEoCVILbmV3UGFzc3dvcmQ=');

@$core.Deprecated('Use changePasswordResponseDescriptor instead')
const ChangePasswordResponse$json = {
  '1': 'ChangePasswordResponse',
  '2': [
    {
      '1': 'tokens',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.TokenPair',
      '10': 'tokens'
    },
  ],
};

/// Descriptor for `ChangePasswordResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List changePasswordResponseDescriptor =
    $convert.base64Decode(
        'ChZDaGFuZ2VQYXNzd29yZFJlc3BvbnNlEi8KBnRva2VucxgBIAEoCzIXLm1vdGguYXV0aC52MS'
        '5Ub2tlblBhaXJSBnRva2Vucw==');

@$core.Deprecated('Use requestEmailVerificationRequestDescriptor instead')
const RequestEmailVerificationRequest$json = {
  '1': 'RequestEmailVerificationRequest',
  '2': [
    {'1': 'email', '3': 1, '4': 1, '5': 9, '10': 'email'},
  ],
};

/// Descriptor for `RequestEmailVerificationRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List requestEmailVerificationRequestDescriptor =
    $convert.base64Decode(
        'Ch9SZXF1ZXN0RW1haWxWZXJpZmljYXRpb25SZXF1ZXN0EhQKBWVtYWlsGAEgASgJUgVlbWFpbA'
        '==');

@$core.Deprecated('Use requestEmailVerificationResponseDescriptor instead')
const RequestEmailVerificationResponse$json = {
  '1': 'RequestEmailVerificationResponse',
};

/// Descriptor for `RequestEmailVerificationResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List requestEmailVerificationResponseDescriptor =
    $convert.base64Decode('CiBSZXF1ZXN0RW1haWxWZXJpZmljYXRpb25SZXNwb25zZQ==');

@$core.Deprecated('Use confirmEmailVerificationRequestDescriptor instead')
const ConfirmEmailVerificationRequest$json = {
  '1': 'ConfirmEmailVerificationRequest',
  '2': [
    {'1': 'token', '3': 1, '4': 1, '5': 9, '10': 'token'},
  ],
};

/// Descriptor for `ConfirmEmailVerificationRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List confirmEmailVerificationRequestDescriptor =
    $convert.base64Decode(
        'Ch9Db25maXJtRW1haWxWZXJpZmljYXRpb25SZXF1ZXN0EhQKBXRva2VuGAEgASgJUgV0b2tlbg'
        '==');

@$core.Deprecated('Use confirmEmailVerificationResponseDescriptor instead')
const ConfirmEmailVerificationResponse$json = {
  '1': 'ConfirmEmailVerificationResponse',
};

/// Descriptor for `ConfirmEmailVerificationResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List confirmEmailVerificationResponseDescriptor =
    $convert.base64Decode('CiBDb25maXJtRW1haWxWZXJpZmljYXRpb25SZXNwb25zZQ==');

@$core.Deprecated('Use requestPasswordResetRequestDescriptor instead')
const RequestPasswordResetRequest$json = {
  '1': 'RequestPasswordResetRequest',
  '2': [
    {'1': 'email', '3': 1, '4': 1, '5': 9, '10': 'email'},
  ],
};

/// Descriptor for `RequestPasswordResetRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List requestPasswordResetRequestDescriptor =
    $convert.base64Decode(
        'ChtSZXF1ZXN0UGFzc3dvcmRSZXNldFJlcXVlc3QSFAoFZW1haWwYASABKAlSBWVtYWls');

@$core.Deprecated('Use requestPasswordResetResponseDescriptor instead')
const RequestPasswordResetResponse$json = {
  '1': 'RequestPasswordResetResponse',
};

/// Descriptor for `RequestPasswordResetResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List requestPasswordResetResponseDescriptor =
    $convert.base64Decode('ChxSZXF1ZXN0UGFzc3dvcmRSZXNldFJlc3BvbnNl');

@$core.Deprecated('Use confirmPasswordResetRequestDescriptor instead')
const ConfirmPasswordResetRequest$json = {
  '1': 'ConfirmPasswordResetRequest',
  '2': [
    {'1': 'token', '3': 1, '4': 1, '5': 9, '10': 'token'},
    {'1': 'new_password', '3': 2, '4': 1, '5': 9, '10': 'newPassword'},
  ],
};

/// Descriptor for `ConfirmPasswordResetRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List confirmPasswordResetRequestDescriptor =
    $convert.base64Decode(
        'ChtDb25maXJtUGFzc3dvcmRSZXNldFJlcXVlc3QSFAoFdG9rZW4YASABKAlSBXRva2VuEiEKDG'
        '5ld19wYXNzd29yZBgCIAEoCVILbmV3UGFzc3dvcmQ=');

@$core.Deprecated('Use confirmPasswordResetResponseDescriptor instead')
const ConfirmPasswordResetResponse$json = {
  '1': 'ConfirmPasswordResetResponse',
};

/// Descriptor for `ConfirmPasswordResetResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List confirmPasswordResetResponseDescriptor =
    $convert.base64Decode('ChxDb25maXJtUGFzc3dvcmRSZXNldFJlc3BvbnNl');

@$core.Deprecated('Use requestEmailChangeRequestDescriptor instead')
const RequestEmailChangeRequest$json = {
  '1': 'RequestEmailChangeRequest',
  '2': [
    {'1': 'new_email', '3': 1, '4': 1, '5': 9, '10': 'newEmail'},
  ],
};

/// Descriptor for `RequestEmailChangeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List requestEmailChangeRequestDescriptor =
    $convert.base64Decode(
        'ChlSZXF1ZXN0RW1haWxDaGFuZ2VSZXF1ZXN0EhsKCW5ld19lbWFpbBgBIAEoCVIIbmV3RW1haW'
        'w=');

@$core.Deprecated('Use requestEmailChangeResponseDescriptor instead')
const RequestEmailChangeResponse$json = {
  '1': 'RequestEmailChangeResponse',
};

/// Descriptor for `RequestEmailChangeResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List requestEmailChangeResponseDescriptor =
    $convert.base64Decode('ChpSZXF1ZXN0RW1haWxDaGFuZ2VSZXNwb25zZQ==');

@$core.Deprecated('Use confirmEmailChangeRequestDescriptor instead')
const ConfirmEmailChangeRequest$json = {
  '1': 'ConfirmEmailChangeRequest',
  '2': [
    {'1': 'token', '3': 1, '4': 1, '5': 9, '10': 'token'},
  ],
};

/// Descriptor for `ConfirmEmailChangeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List confirmEmailChangeRequestDescriptor =
    $convert.base64Decode(
        'ChlDb25maXJtRW1haWxDaGFuZ2VSZXF1ZXN0EhQKBXRva2VuGAEgASgJUgV0b2tlbg==');

@$core.Deprecated('Use confirmEmailChangeResponseDescriptor instead')
const ConfirmEmailChangeResponse$json = {
  '1': 'ConfirmEmailChangeResponse',
};

/// Descriptor for `ConfirmEmailChangeResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List confirmEmailChangeResponseDescriptor =
    $convert.base64Decode('ChpDb25maXJtRW1haWxDaGFuZ2VSZXNwb25zZQ==');

@$core.Deprecated('Use signInWithOAuthRequestDescriptor instead')
const SignInWithOAuthRequest$json = {
  '1': 'SignInWithOAuthRequest',
  '2': [
    {
      '1': 'provider',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.moth.auth.v1.OAuthProvider',
      '10': 'provider'
    },
    {'1': 'id_token', '3': 2, '4': 1, '5': 9, '10': 'idToken'},
    {'1': 'nonce', '3': 3, '4': 1, '5': 9, '10': 'nonce'},
    {
      '1': 'authorization_code',
      '3': 4,
      '4': 1,
      '5': 9,
      '10': 'authorizationCode'
    },
    {'1': 'given_name', '3': 5, '4': 1, '5': 9, '10': 'givenName'},
    {'1': 'family_name', '3': 6, '4': 1, '5': 9, '10': 'familyName'},
    {'1': 'device_info', '3': 7, '4': 1, '5': 9, '10': 'deviceInfo'},
  ],
};

/// Descriptor for `SignInWithOAuthRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List signInWithOAuthRequestDescriptor = $convert.base64Decode(
    'ChZTaWduSW5XaXRoT0F1dGhSZXF1ZXN0EjcKCHByb3ZpZGVyGAEgASgOMhsubW90aC5hdXRoLn'
    'YxLk9BdXRoUHJvdmlkZXJSCHByb3ZpZGVyEhkKCGlkX3Rva2VuGAIgASgJUgdpZFRva2VuEhQK'
    'BW5vbmNlGAMgASgJUgVub25jZRItChJhdXRob3JpemF0aW9uX2NvZGUYBCABKAlSEWF1dGhvcm'
    'l6YXRpb25Db2RlEh0KCmdpdmVuX25hbWUYBSABKAlSCWdpdmVuTmFtZRIfCgtmYW1pbHlfbmFt'
    'ZRgGIAEoCVIKZmFtaWx5TmFtZRIfCgtkZXZpY2VfaW5mbxgHIAEoCVIKZGV2aWNlSW5mbw==');

@$core.Deprecated('Use signInWithOAuthResponseDescriptor instead')
const SignInWithOAuthResponse$json = {
  '1': 'SignInWithOAuthResponse',
  '2': [
    {
      '1': 'user',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.User',
      '10': 'user'
    },
    {
      '1': 'tokens',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.TokenPair',
      '10': 'tokens'
    },
  ],
};

/// Descriptor for `SignInWithOAuthResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List signInWithOAuthResponseDescriptor = $convert.base64Decode(
    'ChdTaWduSW5XaXRoT0F1dGhSZXNwb25zZRImCgR1c2VyGAEgASgLMhIubW90aC5hdXRoLnYxLl'
    'VzZXJSBHVzZXISLwoGdG9rZW5zGAIgASgLMhcubW90aC5hdXRoLnYxLlRva2VuUGFpclIGdG9r'
    'ZW5z');

@$core.Deprecated('Use exchangeOAuthCodeRequestDescriptor instead')
const ExchangeOAuthCodeRequest$json = {
  '1': 'ExchangeOAuthCodeRequest',
  '2': [
    {'1': 'code', '3': 1, '4': 1, '5': 9, '10': 'code'},
    {'1': 'device_info', '3': 2, '4': 1, '5': 9, '10': 'deviceInfo'},
  ],
};

/// Descriptor for `ExchangeOAuthCodeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List exchangeOAuthCodeRequestDescriptor =
    $convert.base64Decode(
        'ChhFeGNoYW5nZU9BdXRoQ29kZVJlcXVlc3QSEgoEY29kZRgBIAEoCVIEY29kZRIfCgtkZXZpY2'
        'VfaW5mbxgCIAEoCVIKZGV2aWNlSW5mbw==');

@$core.Deprecated('Use exchangeOAuthCodeResponseDescriptor instead')
const ExchangeOAuthCodeResponse$json = {
  '1': 'ExchangeOAuthCodeResponse',
  '2': [
    {
      '1': 'user',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.User',
      '10': 'user'
    },
    {
      '1': 'tokens',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.moth.auth.v1.TokenPair',
      '10': 'tokens'
    },
  ],
};

/// Descriptor for `ExchangeOAuthCodeResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List exchangeOAuthCodeResponseDescriptor = $convert.base64Decode(
    'ChlFeGNoYW5nZU9BdXRoQ29kZVJlc3BvbnNlEiYKBHVzZXIYASABKAsyEi5tb3RoLmF1dGgudj'
    'EuVXNlclIEdXNlchIvCgZ0b2tlbnMYAiABKAsyFy5tb3RoLmF1dGgudjEuVG9rZW5QYWlyUgZ0'
    'b2tlbnM=');

@$core.Deprecated('Use unlinkIdentityRequestDescriptor instead')
const UnlinkIdentityRequest$json = {
  '1': 'UnlinkIdentityRequest',
  '2': [
    {
      '1': 'provider',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.moth.auth.v1.OAuthProvider',
      '10': 'provider'
    },
  ],
};

/// Descriptor for `UnlinkIdentityRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List unlinkIdentityRequestDescriptor = $convert.base64Decode(
    'ChVVbmxpbmtJZGVudGl0eVJlcXVlc3QSNwoIcHJvdmlkZXIYASABKA4yGy5tb3RoLmF1dGgudj'
    'EuT0F1dGhQcm92aWRlclIIcHJvdmlkZXI=');

@$core.Deprecated('Use unlinkIdentityResponseDescriptor instead')
const UnlinkIdentityResponse$json = {
  '1': 'UnlinkIdentityResponse',
};

/// Descriptor for `UnlinkIdentityResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List unlinkIdentityResponseDescriptor =
    $convert.base64Decode('ChZVbmxpbmtJZGVudGl0eVJlc3BvbnNl');

@$core.Deprecated('Use deleteAccountRequestDescriptor instead')
const DeleteAccountRequest$json = {
  '1': 'DeleteAccountRequest',
  '2': [
    {'1': 'password', '3': 1, '4': 1, '5': 9, '10': 'password'},
  ],
};

/// Descriptor for `DeleteAccountRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deleteAccountRequestDescriptor =
    $convert.base64Decode(
        'ChREZWxldGVBY2NvdW50UmVxdWVzdBIaCghwYXNzd29yZBgBIAEoCVIIcGFzc3dvcmQ=');

@$core.Deprecated('Use deleteAccountResponseDescriptor instead')
const DeleteAccountResponse$json = {
  '1': 'DeleteAccountResponse',
};

/// Descriptor for `DeleteAccountResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deleteAccountResponseDescriptor =
    $convert.base64Decode('ChVEZWxldGVBY2NvdW50UmVzcG9uc2U=');
