// This is a generated file - do not edit.
//
// Generated from moth/push/v1/push.proto.

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

@$core.Deprecated('Use pushTargetDescriptor instead')
const PushTarget$json = {
  '1': 'PushTarget',
  '2': [
    {'1': 'PUSH_TARGET_UNSPECIFIED', '2': 0},
    {'1': 'PUSH_TARGET_APNS', '2': 1},
    {'1': 'PUSH_TARGET_FCM', '2': 2},
    {'1': 'PUSH_TARGET_WEBPUSH', '2': 3},
  ],
};

/// Descriptor for `PushTarget`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List pushTargetDescriptor = $convert.base64Decode(
    'CgpQdXNoVGFyZ2V0EhsKF1BVU0hfVEFSR0VUX1VOU1BFQ0lGSUVEEAASFAoQUFVTSF9UQVJHRV'
    'RfQVBOUxABEhMKD1BVU0hfVEFSR0VUX0ZDTRACEhcKE1BVU0hfVEFSR0VUX1dFQlBVU0gQAw==');

@$core.Deprecated('Use pushPermissionDescriptor instead')
const PushPermission$json = {
  '1': 'PushPermission',
  '2': [
    {'1': 'PUSH_PERMISSION_UNSPECIFIED', '2': 0},
    {'1': 'PUSH_PERMISSION_GRANTED', '2': 1},
    {'1': 'PUSH_PERMISSION_PROVISIONAL', '2': 2},
    {'1': 'PUSH_PERMISSION_DENIED', '2': 3},
    {'1': 'PUSH_PERMISSION_UNKNOWN', '2': 4},
  ],
};

/// Descriptor for `PushPermission`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List pushPermissionDescriptor = $convert.base64Decode(
    'Cg5QdXNoUGVybWlzc2lvbhIfChtQVVNIX1BFUk1JU1NJT05fVU5TUEVDSUZJRUQQABIbChdQVV'
    'NIX1BFUk1JU1NJT05fR1JBTlRFRBABEh8KG1BVU0hfUEVSTUlTU0lPTl9QUk9WSVNJT05BTBAC'
    'EhoKFlBVU0hfUEVSTUlTU0lPTl9ERU5JRUQQAxIbChdQVVNIX1BFUk1JU1NJT05fVU5LTk9XTh'
    'AE');

@$core.Deprecated('Use pushDeviceMetadataDescriptor instead')
const PushDeviceMetadata$json = {
  '1': 'PushDeviceMetadata',
  '2': [
    {'1': 'platform', '3': 1, '4': 1, '5': 9, '10': 'platform'},
    {'1': 'model', '3': 2, '4': 1, '5': 9, '10': 'model'},
    {'1': 'os_version', '3': 3, '4': 1, '5': 9, '10': 'osVersion'},
    {'1': 'app_version', '3': 4, '4': 1, '5': 9, '10': 'appVersion'},
    {'1': 'locale', '3': 5, '4': 1, '5': 9, '10': 'locale'},
  ],
};

/// Descriptor for `PushDeviceMetadata`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pushDeviceMetadataDescriptor = $convert.base64Decode(
    'ChJQdXNoRGV2aWNlTWV0YWRhdGESGgoIcGxhdGZvcm0YASABKAlSCHBsYXRmb3JtEhQKBW1vZG'
    'VsGAIgASgJUgVtb2RlbBIdCgpvc192ZXJzaW9uGAMgASgJUglvc1ZlcnNpb24SHwoLYXBwX3Zl'
    'cnNpb24YBCABKAlSCmFwcFZlcnNpb24SFgoGbG9jYWxlGAUgASgJUgZsb2NhbGU=');

@$core.Deprecated('Use pushDeviceDescriptor instead')
const PushDevice$json = {
  '1': 'PushDevice',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {
      '1': 'target',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.moth.push.v1.PushTarget',
      '10': 'target'
    },
    {'1': 'device_id', '3': 3, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'permission',
      '3': 4,
      '4': 1,
      '5': 14,
      '6': '.moth.push.v1.PushPermission',
      '10': 'permission'
    },
    {
      '1': 'metadata',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.moth.push.v1.PushDeviceMetadata',
      '10': 'metadata'
    },
    {
      '1': 'create_time',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'createTime'
    },
    {
      '1': 'update_time',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'updateTime'
    },
    {
      '1': 'last_seen_time',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'lastSeenTime'
    },
  ],
};

/// Descriptor for `PushDevice`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pushDeviceDescriptor = $convert.base64Decode(
    'CgpQdXNoRGV2aWNlEg4KAmlkGAEgASgJUgJpZBIwCgZ0YXJnZXQYAiABKA4yGC5tb3RoLnB1c2'
    'gudjEuUHVzaFRhcmdldFIGdGFyZ2V0EhsKCWRldmljZV9pZBgDIAEoCVIIZGV2aWNlSWQSPAoK'
    'cGVybWlzc2lvbhgEIAEoDjIcLm1vdGgucHVzaC52MS5QdXNoUGVybWlzc2lvblIKcGVybWlzc2'
    'lvbhI8CghtZXRhZGF0YRgFIAEoCzIgLm1vdGgucHVzaC52MS5QdXNoRGV2aWNlTWV0YWRhdGFS'
    'CG1ldGFkYXRhEjsKC2NyZWF0ZV90aW1lGAYgASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdG'
    'FtcFIKY3JlYXRlVGltZRI7Cgt1cGRhdGVfdGltZRgHIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5U'
    'aW1lc3RhbXBSCnVwZGF0ZVRpbWUSQAoObGFzdF9zZWVuX3RpbWUYCCABKAsyGi5nb29nbGUucH'
    'JvdG9idWYuVGltZXN0YW1wUgxsYXN0U2VlblRpbWU=');

@$core.Deprecated('Use registerDeviceRequestDescriptor instead')
const RegisterDeviceRequest$json = {
  '1': 'RegisterDeviceRequest',
  '2': [
    {
      '1': 'target',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.moth.push.v1.PushTarget',
      '10': 'target'
    },
    {'1': 'token', '3': 2, '4': 1, '5': 9, '10': 'token'},
    {'1': 'device_id', '3': 3, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'permission',
      '3': 4,
      '4': 1,
      '5': 14,
      '6': '.moth.push.v1.PushPermission',
      '10': 'permission'
    },
    {
      '1': 'metadata',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.moth.push.v1.PushDeviceMetadata',
      '10': 'metadata'
    },
  ],
};

/// Descriptor for `RegisterDeviceRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List registerDeviceRequestDescriptor = $convert.base64Decode(
    'ChVSZWdpc3RlckRldmljZVJlcXVlc3QSMAoGdGFyZ2V0GAEgASgOMhgubW90aC5wdXNoLnYxLl'
    'B1c2hUYXJnZXRSBnRhcmdldBIUCgV0b2tlbhgCIAEoCVIFdG9rZW4SGwoJZGV2aWNlX2lkGAMg'
    'ASgJUghkZXZpY2VJZBI8CgpwZXJtaXNzaW9uGAQgASgOMhwubW90aC5wdXNoLnYxLlB1c2hQZX'
    'JtaXNzaW9uUgpwZXJtaXNzaW9uEjwKCG1ldGFkYXRhGAUgASgLMiAubW90aC5wdXNoLnYxLlB1'
    'c2hEZXZpY2VNZXRhZGF0YVIIbWV0YWRhdGE=');

@$core.Deprecated('Use registerDeviceResponseDescriptor instead')
const RegisterDeviceResponse$json = {
  '1': 'RegisterDeviceResponse',
  '2': [
    {
      '1': 'device',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.moth.push.v1.PushDevice',
      '10': 'device'
    },
  ],
};

/// Descriptor for `RegisterDeviceResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List registerDeviceResponseDescriptor =
    $convert.base64Decode(
        'ChZSZWdpc3RlckRldmljZVJlc3BvbnNlEjAKBmRldmljZRgBIAEoCzIYLm1vdGgucHVzaC52MS'
        '5QdXNoRGV2aWNlUgZkZXZpY2U=');

@$core.Deprecated('Use unregisterDeviceRequestDescriptor instead')
const UnregisterDeviceRequest$json = {
  '1': 'UnregisterDeviceRequest',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
  ],
};

/// Descriptor for `UnregisterDeviceRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List unregisterDeviceRequestDescriptor =
    $convert.base64Decode(
        'ChdVbnJlZ2lzdGVyRGV2aWNlUmVxdWVzdBIbCglkZXZpY2VfaWQYASABKAlSCGRldmljZUlk');

@$core.Deprecated('Use unregisterDeviceResponseDescriptor instead')
const UnregisterDeviceResponse$json = {
  '1': 'UnregisterDeviceResponse',
};

/// Descriptor for `UnregisterDeviceResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List unregisterDeviceResponseDescriptor =
    $convert.base64Decode('ChhVbnJlZ2lzdGVyRGV2aWNlUmVzcG9uc2U=');
