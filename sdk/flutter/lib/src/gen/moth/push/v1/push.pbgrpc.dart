// This is a generated file - do not edit.
//
// Generated from moth/push/v1/push.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:grpc/service_api.dart' as $grpc;
import 'package:protobuf/protobuf.dart' as $pb;

import 'push.pb.dart' as $0;

export 'push.pb.dart';

/// PushService is the client-facing push-device registry, consumed by the moth
/// SDKs (milestone 21). Authenticated exactly like BillingService: every call
/// carries the project publishable key in `x-moth-key: pk_...` request metadata
/// AND a user access token in the `Authorization: Bearer <jwt>` header — a
/// registration always hangs off (project, signed-in user); there are no
/// anonymous device registrations.
///
/// moth registers; the developer's backend sends. moth never talks to
/// APNs/FCM/Web Push — it tracks each registration's target, permission state
/// and liveness, and hands the current set to the developer's backend through
/// moth.server.v1.PushService, which delivers. Rate-limited like the other
/// credential-facing RPCs (milestone 02).
@$pb.GrpcServiceName('moth.push.v1.PushService')
class PushServiceClient extends $grpc.Client {
  /// The hostname for this service.
  static const $core.String defaultHost = '';

  /// OAuth scopes needed for the client.
  static const $core.List<$core.String> oauthScopes = [
    '',
  ];

  PushServiceClient(super.channel, {super.options, super.interceptors});

  /// RegisterDevice upserts the calling user's registration and returns the
  /// stored row. Idempotent by design — the SDK calls it on every app launch,
  /// token rotation and permission change, without bookkeeping:
  ///   - same device_id → the device's existing row is replaced (a rotated
  ///     token supersedes the old one, revoked `replaced`);
  ///   - same (target, token) under a new user → the newest owner wins (the
  ///     previous user's row is revoked `replaced`), which handles a device
  ///     changing accounts on sign-in;
  ///   - otherwise → a new registration is created.
  /// Every call refreshes last_seen_time, so it doubles as the liveness
  /// heartbeat feeding the staleness sweep.
  $grpc.ResponseFuture<$0.RegisterDeviceResponse> registerDevice(
    $0.RegisterDeviceRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$registerDevice, request, options: options);
  }

  /// UnregisterDevice revokes the calling user's registration for one
  /// installation (`signed_out`); the SDKs call it on sign-out. Idempotent:
  /// unknown or already-revoked device ids succeed.
  $grpc.ResponseFuture<$0.UnregisterDeviceResponse> unregisterDevice(
    $0.UnregisterDeviceRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$unregisterDevice, request, options: options);
  }

  // method descriptors

  static final _$registerDevice =
      $grpc.ClientMethod<$0.RegisterDeviceRequest, $0.RegisterDeviceResponse>(
          '/moth.push.v1.PushService/RegisterDevice',
          ($0.RegisterDeviceRequest value) => value.writeToBuffer(),
          $0.RegisterDeviceResponse.fromBuffer);
  static final _$unregisterDevice = $grpc.ClientMethod<
          $0.UnregisterDeviceRequest, $0.UnregisterDeviceResponse>(
      '/moth.push.v1.PushService/UnregisterDevice',
      ($0.UnregisterDeviceRequest value) => value.writeToBuffer(),
      $0.UnregisterDeviceResponse.fromBuffer);
}

@$pb.GrpcServiceName('moth.push.v1.PushService')
abstract class PushServiceBase extends $grpc.Service {
  $core.String get $name => 'moth.push.v1.PushService';

  PushServiceBase() {
    $addMethod($grpc.ServiceMethod<$0.RegisterDeviceRequest,
            $0.RegisterDeviceResponse>(
        'RegisterDevice',
        registerDevice_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.RegisterDeviceRequest.fromBuffer(value),
        ($0.RegisterDeviceResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.UnregisterDeviceRequest,
            $0.UnregisterDeviceResponse>(
        'UnregisterDevice',
        unregisterDevice_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.UnregisterDeviceRequest.fromBuffer(value),
        ($0.UnregisterDeviceResponse value) => value.writeToBuffer()));
  }

  $async.Future<$0.RegisterDeviceResponse> registerDevice_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.RegisterDeviceRequest> $request) async {
    return registerDevice($call, await $request);
  }

  $async.Future<$0.RegisterDeviceResponse> registerDevice(
      $grpc.ServiceCall call, $0.RegisterDeviceRequest request);

  $async.Future<$0.UnregisterDeviceResponse> unregisterDevice_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.UnregisterDeviceRequest> $request) async {
    return unregisterDevice($call, await $request);
  }

  $async.Future<$0.UnregisterDeviceResponse> unregisterDevice(
      $grpc.ServiceCall call, $0.UnregisterDeviceRequest request);
}
