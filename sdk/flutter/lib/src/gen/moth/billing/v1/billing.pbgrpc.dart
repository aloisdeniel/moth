// This is a generated file - do not edit.
//
// Generated from moth/billing/v1/billing.proto.

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

import 'billing.pb.dart' as $0;

export 'billing.pb.dart';

/// BillingService is the client-facing subscription API, consumed by the moth
/// Flutter SDK. Authenticated exactly like AuthService: every call carries the
/// project publishable key in `x-moth-key: pk_...` request metadata AND a user
/// access token in the `Authorization: Bearer <jwt>` header — every RPC is
/// scoped to the signed-in user.
///
/// The core contract: **a user always has a valid subscription state.** A
/// never-paid user, a free-tier user, and a user in a project that has declared
/// no products all get a well-formed CustomerInfo with an empty
/// active_entitlements list (the built-in `none` tier) — never an error. moth is
/// a validating mirror of the store: it never marks a subscription active on the
/// client's say-so, only after verifying a signed transaction or reading the
/// store's authoritative state.
@$pb.GrpcServiceName('moth.billing.v1.BillingService')
class BillingServiceClient extends $grpc.Client {
  /// The hostname for this service.
  static const $core.String defaultHost = '';

  /// OAuth scopes needed for the client.
  static const $core.List<$core.String> oauthScopes = [
    '',
  ];

  BillingServiceClient(super.channel, {super.options, super.interceptors});

  /// GetCustomerInfo returns the signed-in user's active entitlements and
  /// subscriptions. Always succeeds with a valid object; `none` (empty
  /// entitlements) for free users. Cheap and safe to call on every app launch.
  $grpc.ResponseFuture<$0.GetCustomerInfoResponse> getCustomerInfo(
    $0.GetCustomerInfoRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$getCustomerInfo, request, options: options);
  }

  /// SubmitPurchase hands moth the receipt of a purchase the app just completed
  /// natively. moth validates it against the store, links the subscription to
  /// the current user, derives entitlements, and returns the fresh CustomerInfo.
  $grpc.ResponseFuture<$0.SubmitPurchaseResponse> submitPurchase(
    $0.SubmitPurchaseRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$submitPurchase, request, options: options);
  }

  /// RestorePurchases re-links a user's existing store purchases to the current
  /// account (new device, reinstall, account change), applying the store's own
  /// transfer rules, then returns the fresh CustomerInfo.
  $grpc.ResponseFuture<$0.RestorePurchasesResponse> restorePurchases(
    $0.RestorePurchasesRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$restorePurchases, request, options: options);
  }

  /// GetOfferings returns an offering's products for the paywall to display:
  /// per product the catalog identifier, display name, store SKUs (so the SDK
  /// can match the native store products), price/period metadata, trial/intro
  /// descriptor, the entitlements it grants, sort order and the "most popular"
  /// highlight flag. Unlike the three RPCs above this is publishable-key only
  /// (no Bearer): a paywall renders before the user signs in.
  $grpc.ResponseFuture<$0.GetOfferingsResponse> getOfferings(
    $0.GetOfferingsRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$getOfferings, request, options: options);
  }

  /// GetPaywall returns the project's public paywall configuration (copy,
  /// benefit bullets, offering ref, layout, highlighted tier, legal links) with
  /// a revision id, for the SDK's batteries-included paywall screen. Colors and
  /// typography are NOT here — the paywall inherits them from the theme
  /// (GetProjectConfig, milestone 06).
  ///
  /// Caching contract (identical to GetProjectConfig + theme): the client caches
  /// the paywall keyed by revision_id and echoes it as
  /// GetPaywallRequest.known_paywall_revision. When it still matches, the
  /// response omits `paywall` and the client keeps rendering its cache
  /// (stale-while-revalidate); when it differs (or was empty on first call),
  /// `paywall` is present and the client replaces its cache. Publishable-key
  /// only, like GetOfferings.
  $grpc.ResponseFuture<$0.GetPaywallResponse> getPaywall(
    $0.GetPaywallRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$getPaywall, request, options: options);
  }

  // method descriptors

  static final _$getCustomerInfo =
      $grpc.ClientMethod<$0.GetCustomerInfoRequest, $0.GetCustomerInfoResponse>(
          '/moth.billing.v1.BillingService/GetCustomerInfo',
          ($0.GetCustomerInfoRequest value) => value.writeToBuffer(),
          $0.GetCustomerInfoResponse.fromBuffer);
  static final _$submitPurchase =
      $grpc.ClientMethod<$0.SubmitPurchaseRequest, $0.SubmitPurchaseResponse>(
          '/moth.billing.v1.BillingService/SubmitPurchase',
          ($0.SubmitPurchaseRequest value) => value.writeToBuffer(),
          $0.SubmitPurchaseResponse.fromBuffer);
  static final _$restorePurchases = $grpc.ClientMethod<
          $0.RestorePurchasesRequest, $0.RestorePurchasesResponse>(
      '/moth.billing.v1.BillingService/RestorePurchases',
      ($0.RestorePurchasesRequest value) => value.writeToBuffer(),
      $0.RestorePurchasesResponse.fromBuffer);
  static final _$getOfferings =
      $grpc.ClientMethod<$0.GetOfferingsRequest, $0.GetOfferingsResponse>(
          '/moth.billing.v1.BillingService/GetOfferings',
          ($0.GetOfferingsRequest value) => value.writeToBuffer(),
          $0.GetOfferingsResponse.fromBuffer);
  static final _$getPaywall =
      $grpc.ClientMethod<$0.GetPaywallRequest, $0.GetPaywallResponse>(
          '/moth.billing.v1.BillingService/GetPaywall',
          ($0.GetPaywallRequest value) => value.writeToBuffer(),
          $0.GetPaywallResponse.fromBuffer);
}

@$pb.GrpcServiceName('moth.billing.v1.BillingService')
abstract class BillingServiceBase extends $grpc.Service {
  $core.String get $name => 'moth.billing.v1.BillingService';

  BillingServiceBase() {
    $addMethod($grpc.ServiceMethod<$0.GetCustomerInfoRequest,
            $0.GetCustomerInfoResponse>(
        'GetCustomerInfo',
        getCustomerInfo_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.GetCustomerInfoRequest.fromBuffer(value),
        ($0.GetCustomerInfoResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.SubmitPurchaseRequest,
            $0.SubmitPurchaseResponse>(
        'SubmitPurchase',
        submitPurchase_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.SubmitPurchaseRequest.fromBuffer(value),
        ($0.SubmitPurchaseResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.RestorePurchasesRequest,
            $0.RestorePurchasesResponse>(
        'RestorePurchases',
        restorePurchases_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.RestorePurchasesRequest.fromBuffer(value),
        ($0.RestorePurchasesResponse value) => value.writeToBuffer()));
    $addMethod(
        $grpc.ServiceMethod<$0.GetOfferingsRequest, $0.GetOfferingsResponse>(
            'GetOfferings',
            getOfferings_Pre,
            false,
            false,
            ($core.List<$core.int> value) =>
                $0.GetOfferingsRequest.fromBuffer(value),
            ($0.GetOfferingsResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.GetPaywallRequest, $0.GetPaywallResponse>(
        'GetPaywall',
        getPaywall_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.GetPaywallRequest.fromBuffer(value),
        ($0.GetPaywallResponse value) => value.writeToBuffer()));
  }

  $async.Future<$0.GetCustomerInfoResponse> getCustomerInfo_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.GetCustomerInfoRequest> $request) async {
    return getCustomerInfo($call, await $request);
  }

  $async.Future<$0.GetCustomerInfoResponse> getCustomerInfo(
      $grpc.ServiceCall call, $0.GetCustomerInfoRequest request);

  $async.Future<$0.SubmitPurchaseResponse> submitPurchase_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.SubmitPurchaseRequest> $request) async {
    return submitPurchase($call, await $request);
  }

  $async.Future<$0.SubmitPurchaseResponse> submitPurchase(
      $grpc.ServiceCall call, $0.SubmitPurchaseRequest request);

  $async.Future<$0.RestorePurchasesResponse> restorePurchases_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.RestorePurchasesRequest> $request) async {
    return restorePurchases($call, await $request);
  }

  $async.Future<$0.RestorePurchasesResponse> restorePurchases(
      $grpc.ServiceCall call, $0.RestorePurchasesRequest request);

  $async.Future<$0.GetOfferingsResponse> getOfferings_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.GetOfferingsRequest> $request) async {
    return getOfferings($call, await $request);
  }

  $async.Future<$0.GetOfferingsResponse> getOfferings(
      $grpc.ServiceCall call, $0.GetOfferingsRequest request);

  $async.Future<$0.GetPaywallResponse> getPaywall_Pre($grpc.ServiceCall $call,
      $async.Future<$0.GetPaywallRequest> $request) async {
    return getPaywall($call, await $request);
  }

  $async.Future<$0.GetPaywallResponse> getPaywall(
      $grpc.ServiceCall call, $0.GetPaywallRequest request);
}
