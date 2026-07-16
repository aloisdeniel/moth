// This is a generated file - do not edit.
//
// Generated from moth/billing/v1/billing.proto.

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

import 'billing.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'billing.pbenum.dart';

/// CustomerInfo is the complete subscription picture for one user. Apps gate
/// features on active_entitlements (never on a product id): check whether the
/// stable entitlement identifier (e.g. "pro") is present.
class CustomerInfo extends $pb.GeneratedMessage {
  factory CustomerInfo({
    $core.Iterable<Entitlement>? activeEntitlements,
    $core.Iterable<ActiveSubscription>? subscriptions,
  }) {
    final result = create();
    if (activeEntitlements != null)
      result.activeEntitlements.addAll(activeEntitlements);
    if (subscriptions != null) result.subscriptions.addAll(subscriptions);
    return result;
  }

  CustomerInfo._();

  factory CustomerInfo.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CustomerInfo.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CustomerInfo',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..pPM<Entitlement>(1, _omitFieldNames ? '' : 'activeEntitlements',
        subBuilder: Entitlement.create)
    ..pPM<ActiveSubscription>(2, _omitFieldNames ? '' : 'subscriptions',
        subBuilder: ActiveSubscription.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CustomerInfo clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CustomerInfo copyWith(void Function(CustomerInfo) updates) =>
      super.copyWith((message) => updates(message as CustomerInfo))
          as CustomerInfo;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CustomerInfo create() => CustomerInfo._();
  @$core.override
  CustomerInfo createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CustomerInfo getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CustomerInfo>(create);
  static CustomerInfo? _defaultInstance;

  /// The entitlements the user currently holds. Empty means the free `none`
  /// tier — a valid, expected state, not an error.
  @$pb.TagNumber(1)
  $pb.PbList<Entitlement> get activeEntitlements => $_getList(0);

  /// The user's known subscriptions across stores (may include inactive ones,
  /// for history/paywall display).
  @$pb.TagNumber(2)
  $pb.PbList<ActiveSubscription> get subscriptions => $_getList(1);
}

/// Entitlement is one active capability the user holds.
class Entitlement extends $pb.GeneratedMessage {
  factory Entitlement({
    $core.String? identifier,
    $1.Timestamp? expireTime,
    EntitlementSource? source,
    $core.String? productIdentifier,
  }) {
    final result = create();
    if (identifier != null) result.identifier = identifier;
    if (expireTime != null) result.expireTime = expireTime;
    if (source != null) result.source = source;
    if (productIdentifier != null) result.productIdentifier = productIdentifier;
    return result;
  }

  Entitlement._();

  factory Entitlement.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Entitlement.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Entitlement',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'identifier')
    ..aOM<$1.Timestamp>(2, _omitFieldNames ? '' : 'expireTime',
        subBuilder: $1.Timestamp.create)
    ..aE<EntitlementSource>(3, _omitFieldNames ? '' : 'source',
        enumValues: EntitlementSource.values)
    ..aOS(4, _omitFieldNames ? '' : 'productIdentifier')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Entitlement clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Entitlement copyWith(void Function(Entitlement) updates) =>
      super.copyWith((message) => updates(message as Entitlement))
          as Entitlement;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Entitlement create() => Entitlement._();
  @$core.override
  Entitlement createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Entitlement getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<Entitlement>(create);
  static Entitlement? _defaultInstance;

  /// Stable identifier the app checks (e.g. "pro"). Never changes across app
  /// releases even when the granting product does.
  @$pb.TagNumber(1)
  $core.String get identifier => $_getSZ(0);
  @$pb.TagNumber(1)
  set identifier($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasIdentifier() => $_has(0);
  @$pb.TagNumber(1)
  void clearIdentifier() => $_clearField(1);

  /// When the entitlement lapses; unset for a non-expiring grant.
  @$pb.TagNumber(2)
  $1.Timestamp get expireTime => $_getN(1);
  @$pb.TagNumber(2)
  set expireTime($1.Timestamp value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasExpireTime() => $_has(1);
  @$pb.TagNumber(2)
  void clearExpireTime() => $_clearField(2);
  @$pb.TagNumber(2)
  $1.Timestamp ensureExpireTime() => $_ensure(1);

  /// Why it is active (store subscription vs operator grant).
  @$pb.TagNumber(3)
  EntitlementSource get source => $_getN(2);
  @$pb.TagNumber(3)
  set source(EntitlementSource value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasSource() => $_has(2);
  @$pb.TagNumber(3)
  void clearSource() => $_clearField(3);

  /// The moth product identifier that granted it, when source is STORE; empty
  /// for grants.
  @$pb.TagNumber(4)
  $core.String get productIdentifier => $_getSZ(3);
  @$pb.TagNumber(4)
  set productIdentifier($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasProductIdentifier() => $_has(3);
  @$pb.TagNumber(4)
  void clearProductIdentifier() => $_clearField(4);
}

/// ActiveSubscription is one of the user's store subscriptions.
class ActiveSubscription extends $pb.GeneratedMessage {
  factory ActiveSubscription({
    $core.String? productIdentifier,
    Store? store,
    SubscriptionStatus? status,
    $1.Timestamp? currentPeriodEnd,
    $core.bool? autoRenew,
    $core.bool? isSandbox,
  }) {
    final result = create();
    if (productIdentifier != null) result.productIdentifier = productIdentifier;
    if (store != null) result.store = store;
    if (status != null) result.status = status;
    if (currentPeriodEnd != null) result.currentPeriodEnd = currentPeriodEnd;
    if (autoRenew != null) result.autoRenew = autoRenew;
    if (isSandbox != null) result.isSandbox = isSandbox;
    return result;
  }

  ActiveSubscription._();

  factory ActiveSubscription.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ActiveSubscription.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ActiveSubscription',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'productIdentifier')
    ..aE<Store>(2, _omitFieldNames ? '' : 'store', enumValues: Store.values)
    ..aE<SubscriptionStatus>(3, _omitFieldNames ? '' : 'status',
        enumValues: SubscriptionStatus.values)
    ..aOM<$1.Timestamp>(4, _omitFieldNames ? '' : 'currentPeriodEnd',
        subBuilder: $1.Timestamp.create)
    ..aOB(5, _omitFieldNames ? '' : 'autoRenew')
    ..aOB(6, _omitFieldNames ? '' : 'isSandbox')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ActiveSubscription clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ActiveSubscription copyWith(void Function(ActiveSubscription) updates) =>
      super.copyWith((message) => updates(message as ActiveSubscription))
          as ActiveSubscription;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ActiveSubscription create() => ActiveSubscription._();
  @$core.override
  ActiveSubscription createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ActiveSubscription getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ActiveSubscription>(create);
  static ActiveSubscription? _defaultInstance;

  /// The moth product identifier, when the store SKU is mapped; empty otherwise.
  @$pb.TagNumber(1)
  $core.String get productIdentifier => $_getSZ(0);
  @$pb.TagNumber(1)
  set productIdentifier($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProductIdentifier() => $_has(0);
  @$pb.TagNumber(1)
  void clearProductIdentifier() => $_clearField(1);

  @$pb.TagNumber(2)
  Store get store => $_getN(1);
  @$pb.TagNumber(2)
  set store(Store value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasStore() => $_has(1);
  @$pb.TagNumber(2)
  void clearStore() => $_clearField(2);

  @$pb.TagNumber(3)
  SubscriptionStatus get status => $_getN(2);
  @$pb.TagNumber(3)
  set status(SubscriptionStatus value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasStatus() => $_has(2);
  @$pb.TagNumber(3)
  void clearStatus() => $_clearField(3);

  /// End of the current paid (or trial) period; the renewal date when
  /// auto_renew is true.
  @$pb.TagNumber(4)
  $1.Timestamp get currentPeriodEnd => $_getN(3);
  @$pb.TagNumber(4)
  set currentPeriodEnd($1.Timestamp value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasCurrentPeriodEnd() => $_has(3);
  @$pb.TagNumber(4)
  void clearCurrentPeriodEnd() => $_clearField(4);
  @$pb.TagNumber(4)
  $1.Timestamp ensureCurrentPeriodEnd() => $_ensure(3);

  @$pb.TagNumber(5)
  $core.bool get autoRenew => $_getBF(4);
  @$pb.TagNumber(5)
  set autoRenew($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasAutoRenew() => $_has(4);
  @$pb.TagNumber(5)
  void clearAutoRenew() => $_clearField(5);

  /// Whether this subscription is a sandbox/test purchase.
  @$pb.TagNumber(6)
  $core.bool get isSandbox => $_getBF(5);
  @$pb.TagNumber(6)
  set isSandbox($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasIsSandbox() => $_has(5);
  @$pb.TagNumber(6)
  void clearIsSandbox() => $_clearField(6);
}

class GetCustomerInfoRequest extends $pb.GeneratedMessage {
  factory GetCustomerInfoRequest() => create();

  GetCustomerInfoRequest._();

  factory GetCustomerInfoRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetCustomerInfoRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetCustomerInfoRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCustomerInfoRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCustomerInfoRequest copyWith(
          void Function(GetCustomerInfoRequest) updates) =>
      super.copyWith((message) => updates(message as GetCustomerInfoRequest))
          as GetCustomerInfoRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetCustomerInfoRequest create() => GetCustomerInfoRequest._();
  @$core.override
  GetCustomerInfoRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetCustomerInfoRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetCustomerInfoRequest>(create);
  static GetCustomerInfoRequest? _defaultInstance;
}

class GetCustomerInfoResponse extends $pb.GeneratedMessage {
  factory GetCustomerInfoResponse({
    CustomerInfo? customerInfo,
  }) {
    final result = create();
    if (customerInfo != null) result.customerInfo = customerInfo;
    return result;
  }

  GetCustomerInfoResponse._();

  factory GetCustomerInfoResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetCustomerInfoResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetCustomerInfoResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOM<CustomerInfo>(1, _omitFieldNames ? '' : 'customerInfo',
        subBuilder: CustomerInfo.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCustomerInfoResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCustomerInfoResponse copyWith(
          void Function(GetCustomerInfoResponse) updates) =>
      super.copyWith((message) => updates(message as GetCustomerInfoResponse))
          as GetCustomerInfoResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetCustomerInfoResponse create() => GetCustomerInfoResponse._();
  @$core.override
  GetCustomerInfoResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetCustomerInfoResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetCustomerInfoResponse>(create);
  static GetCustomerInfoResponse? _defaultInstance;

  @$pb.TagNumber(1)
  CustomerInfo get customerInfo => $_getN(0);
  @$pb.TagNumber(1)
  set customerInfo(CustomerInfo value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasCustomerInfo() => $_has(0);
  @$pb.TagNumber(1)
  void clearCustomerInfo() => $_clearField(1);
  @$pb.TagNumber(1)
  CustomerInfo ensureCustomerInfo() => $_ensure(0);
}

enum SubmitPurchaseRequest_Receipt {
  appleJwsTransaction,
  googlePurchaseToken,
  notSet
}

class SubmitPurchaseRequest extends $pb.GeneratedMessage {
  factory SubmitPurchaseRequest({
    Store? store,
    $core.String? productIdentifier,
    $core.String? appleJwsTransaction,
    $core.String? googlePurchaseToken,
    $core.String? googleSubscriptionId,
  }) {
    final result = create();
    if (store != null) result.store = store;
    if (productIdentifier != null) result.productIdentifier = productIdentifier;
    if (appleJwsTransaction != null)
      result.appleJwsTransaction = appleJwsTransaction;
    if (googlePurchaseToken != null)
      result.googlePurchaseToken = googlePurchaseToken;
    if (googleSubscriptionId != null)
      result.googleSubscriptionId = googleSubscriptionId;
    return result;
  }

  SubmitPurchaseRequest._();

  factory SubmitPurchaseRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SubmitPurchaseRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, SubmitPurchaseRequest_Receipt>
      _SubmitPurchaseRequest_ReceiptByTag = {
    3: SubmitPurchaseRequest_Receipt.appleJwsTransaction,
    4: SubmitPurchaseRequest_Receipt.googlePurchaseToken,
    0: SubmitPurchaseRequest_Receipt.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SubmitPurchaseRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..oo(0, [3, 4])
    ..aE<Store>(1, _omitFieldNames ? '' : 'store', enumValues: Store.values)
    ..aOS(2, _omitFieldNames ? '' : 'productIdentifier')
    ..aOS(3, _omitFieldNames ? '' : 'appleJwsTransaction')
    ..aOS(4, _omitFieldNames ? '' : 'googlePurchaseToken')
    ..aOS(5, _omitFieldNames ? '' : 'googleSubscriptionId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SubmitPurchaseRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SubmitPurchaseRequest copyWith(
          void Function(SubmitPurchaseRequest) updates) =>
      super.copyWith((message) => updates(message as SubmitPurchaseRequest))
          as SubmitPurchaseRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SubmitPurchaseRequest create() => SubmitPurchaseRequest._();
  @$core.override
  SubmitPurchaseRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SubmitPurchaseRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SubmitPurchaseRequest>(create);
  static SubmitPurchaseRequest? _defaultInstance;

  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  SubmitPurchaseRequest_Receipt whichReceipt() =>
      _SubmitPurchaseRequest_ReceiptByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  void clearReceipt() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  Store get store => $_getN(0);
  @$pb.TagNumber(1)
  set store(Store value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasStore() => $_has(0);
  @$pb.TagNumber(1)
  void clearStore() => $_clearField(1);

  /// The moth product identifier the app is purchasing (its own catalog id, not
  /// the store SKU). moth maps it to the store product for validation.
  @$pb.TagNumber(2)
  $core.String get productIdentifier => $_getSZ(1);
  @$pb.TagNumber(2)
  set productIdentifier($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasProductIdentifier() => $_has(1);
  @$pb.TagNumber(2)
  void clearProductIdentifier() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get appleJwsTransaction => $_getSZ(2);
  @$pb.TagNumber(3)
  set appleJwsTransaction($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAppleJwsTransaction() => $_has(2);
  @$pb.TagNumber(3)
  void clearAppleJwsTransaction() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get googlePurchaseToken => $_getSZ(3);
  @$pb.TagNumber(4)
  set googlePurchaseToken($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasGooglePurchaseToken() => $_has(3);
  @$pb.TagNumber(4)
  void clearGooglePurchaseToken() => $_clearField(4);

  /// Google Play subscription id (the store product id); required alongside
  /// google_purchase_token, ignored for Apple.
  @$pb.TagNumber(5)
  $core.String get googleSubscriptionId => $_getSZ(4);
  @$pb.TagNumber(5)
  set googleSubscriptionId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasGoogleSubscriptionId() => $_has(4);
  @$pb.TagNumber(5)
  void clearGoogleSubscriptionId() => $_clearField(5);
}

class SubmitPurchaseResponse extends $pb.GeneratedMessage {
  factory SubmitPurchaseResponse({
    CustomerInfo? customerInfo,
  }) {
    final result = create();
    if (customerInfo != null) result.customerInfo = customerInfo;
    return result;
  }

  SubmitPurchaseResponse._();

  factory SubmitPurchaseResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SubmitPurchaseResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SubmitPurchaseResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOM<CustomerInfo>(1, _omitFieldNames ? '' : 'customerInfo',
        subBuilder: CustomerInfo.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SubmitPurchaseResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SubmitPurchaseResponse copyWith(
          void Function(SubmitPurchaseResponse) updates) =>
      super.copyWith((message) => updates(message as SubmitPurchaseResponse))
          as SubmitPurchaseResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SubmitPurchaseResponse create() => SubmitPurchaseResponse._();
  @$core.override
  SubmitPurchaseResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SubmitPurchaseResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SubmitPurchaseResponse>(create);
  static SubmitPurchaseResponse? _defaultInstance;

  @$pb.TagNumber(1)
  CustomerInfo get customerInfo => $_getN(0);
  @$pb.TagNumber(1)
  set customerInfo(CustomerInfo value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasCustomerInfo() => $_has(0);
  @$pb.TagNumber(1)
  void clearCustomerInfo() => $_clearField(1);
  @$pb.TagNumber(1)
  CustomerInfo ensureCustomerInfo() => $_ensure(0);
}

class RestorePurchasesRequest extends $pb.GeneratedMessage {
  factory RestorePurchasesRequest({
    Store? store,
    $core.Iterable<$core.String>? receipts,
  }) {
    final result = create();
    if (store != null) result.store = store;
    if (receipts != null) result.receipts.addAll(receipts);
    return result;
  }

  RestorePurchasesRequest._();

  factory RestorePurchasesRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RestorePurchasesRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RestorePurchasesRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aE<Store>(1, _omitFieldNames ? '' : 'store', enumValues: Store.values)
    ..pPS(2, _omitFieldNames ? '' : 'receipts')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RestorePurchasesRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RestorePurchasesRequest copyWith(
          void Function(RestorePurchasesRequest) updates) =>
      super.copyWith((message) => updates(message as RestorePurchasesRequest))
          as RestorePurchasesRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RestorePurchasesRequest create() => RestorePurchasesRequest._();
  @$core.override
  RestorePurchasesRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RestorePurchasesRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RestorePurchasesRequest>(create);
  static RestorePurchasesRequest? _defaultInstance;

  @$pb.TagNumber(1)
  Store get store => $_getN(0);
  @$pb.TagNumber(1)
  set store(Store value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasStore() => $_has(0);
  @$pb.TagNumber(1)
  void clearStore() => $_clearField(1);

  /// The receipts to re-link. For Apple, StoreKit 2 signed transactions (JWS);
  /// for Google, Play Billing purchase tokens.
  @$pb.TagNumber(2)
  $pb.PbList<$core.String> get receipts => $_getList(1);
}

class RestorePurchasesResponse extends $pb.GeneratedMessage {
  factory RestorePurchasesResponse({
    CustomerInfo? customerInfo,
  }) {
    final result = create();
    if (customerInfo != null) result.customerInfo = customerInfo;
    return result;
  }

  RestorePurchasesResponse._();

  factory RestorePurchasesResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RestorePurchasesResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RestorePurchasesResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOM<CustomerInfo>(1, _omitFieldNames ? '' : 'customerInfo',
        subBuilder: CustomerInfo.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RestorePurchasesResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RestorePurchasesResponse copyWith(
          void Function(RestorePurchasesResponse) updates) =>
      super.copyWith((message) => updates(message as RestorePurchasesResponse))
          as RestorePurchasesResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RestorePurchasesResponse create() => RestorePurchasesResponse._();
  @$core.override
  RestorePurchasesResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RestorePurchasesResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RestorePurchasesResponse>(create);
  static RestorePurchasesResponse? _defaultInstance;

  @$pb.TagNumber(1)
  CustomerInfo get customerInfo => $_getN(0);
  @$pb.TagNumber(1)
  set customerInfo(CustomerInfo value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasCustomerInfo() => $_has(0);
  @$pb.TagNumber(1)
  void clearCustomerInfo() => $_clearField(1);
  @$pb.TagNumber(1)
  CustomerInfo ensureCustomerInfo() => $_ensure(0);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
