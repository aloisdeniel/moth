// This is a generated file - do not edit.
//
// Generated from moth/auth/v1/config.proto.

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

import 'config.pb.dart' as $0;

export 'config.pb.dart';

/// ConfigService exposes a project's public, non-secret configuration to the
/// mobile SDK, so the login screen can render exactly the sign-in methods
/// the project enables. Authenticated like AuthService: every call carries
/// the project's publishable key in `x-moth-key: pk_...` request metadata.
///
/// Later milestones extend GetProjectConfigResponse: SDK bootstrap values in
/// 05, login-screen branding/theme in 06. Fields are only ever added.
@$pb.GrpcServiceName('moth.auth.v1.ConfigService')
class ConfigServiceClient extends $grpc.Client {
  /// The hostname for this service.
  static const $core.String defaultHost = '';

  /// OAuth scopes needed for the client.
  static const $core.List<$core.String> oauthScopes = [
    '',
  ];

  ConfigServiceClient(super.channel, {super.options, super.interceptors});

  /// GetProjectConfig returns the project configuration a client may see.
  /// Never includes secrets; only values that are safe to embed in an app.
  $grpc.ResponseFuture<$0.GetProjectConfigResponse> getProjectConfig(
    $0.GetProjectConfigRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$getProjectConfig, request, options: options);
  }

  // method descriptors

  static final _$getProjectConfig = $grpc.ClientMethod<
          $0.GetProjectConfigRequest, $0.GetProjectConfigResponse>(
      '/moth.auth.v1.ConfigService/GetProjectConfig',
      ($0.GetProjectConfigRequest value) => value.writeToBuffer(),
      $0.GetProjectConfigResponse.fromBuffer);
}

@$pb.GrpcServiceName('moth.auth.v1.ConfigService')
abstract class ConfigServiceBase extends $grpc.Service {
  $core.String get $name => 'moth.auth.v1.ConfigService';

  ConfigServiceBase() {
    $addMethod($grpc.ServiceMethod<$0.GetProjectConfigRequest,
            $0.GetProjectConfigResponse>(
        'GetProjectConfig',
        getProjectConfig_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.GetProjectConfigRequest.fromBuffer(value),
        ($0.GetProjectConfigResponse value) => value.writeToBuffer()));
  }

  $async.Future<$0.GetProjectConfigResponse> getProjectConfig_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.GetProjectConfigRequest> $request) async {
    return getProjectConfig($call, await $request);
  }

  $async.Future<$0.GetProjectConfigResponse> getProjectConfig(
      $grpc.ServiceCall call, $0.GetProjectConfigRequest request);
}
