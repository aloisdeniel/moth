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

/// PaywallLayout is the rendering variant the paywall screen uses; the token
/// space (colors/spacing/radius) always comes from the theme.
class PaywallLayout extends $pb.ProtobufEnum {
  static const PaywallLayout PAYWALL_LAYOUT_UNSPECIFIED =
      PaywallLayout._(0, _omitEnumNames ? '' : 'PAYWALL_LAYOUT_UNSPECIFIED');

  /// One card per tier, side by side (the default).
  static const PaywallLayout PAYWALL_LAYOUT_TILES =
      PaywallLayout._(1, _omitEnumNames ? '' : 'PAYWALL_LAYOUT_TILES');

  /// Tiers stacked as full-width rows.
  static const PaywallLayout PAYWALL_LAYOUT_LIST =
      PaywallLayout._(2, _omitEnumNames ? '' : 'PAYWALL_LAYOUT_LIST');

  /// A single selected tier with a period toggle.
  static const PaywallLayout PAYWALL_LAYOUT_COMPACT =
      PaywallLayout._(3, _omitEnumNames ? '' : 'PAYWALL_LAYOUT_COMPACT');

  static const $core.List<PaywallLayout> values = <PaywallLayout>[
    PAYWALL_LAYOUT_UNSPECIFIED,
    PAYWALL_LAYOUT_TILES,
    PAYWALL_LAYOUT_LIST,
    PAYWALL_LAYOUT_COMPACT,
  ];

  static final $core.List<PaywallLayout?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static PaywallLayout? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const PaywallLayout._(super.value, super.name);
}

/// Store identifies which app store a purchase or subscription belongs to.
class Store extends $pb.ProtobufEnum {
  static const Store STORE_UNSPECIFIED =
      Store._(0, _omitEnumNames ? '' : 'STORE_UNSPECIFIED');
  static const Store STORE_APPLE =
      Store._(1, _omitEnumNames ? '' : 'STORE_APPLE');
  static const Store STORE_GOOGLE =
      Store._(2, _omitEnumNames ? '' : 'STORE_GOOGLE');
  static const Store STORE_STRIPE =
      Store._(3, _omitEnumNames ? '' : 'STORE_STRIPE');

  static const $core.List<Store> values = <Store>[
    STORE_UNSPECIFIED,
    STORE_APPLE,
    STORE_GOOGLE,
    STORE_STRIPE,
  ];

  static final $core.List<Store?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static Store? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const Store._(super.value, super.name);
}

/// SubscriptionStatus mirrors the store's renewal state, mapped to a small set
/// common to Apple and Google. active/trialing/in_grace_period/in_billing_retry
/// all keep access; paused/expired/revoked do not.
class SubscriptionStatus extends $pb.ProtobufEnum {
  static const SubscriptionStatus SUBSCRIPTION_STATUS_UNSPECIFIED =
      SubscriptionStatus._(
          0, _omitEnumNames ? '' : 'SUBSCRIPTION_STATUS_UNSPECIFIED');
  static const SubscriptionStatus SUBSCRIPTION_STATUS_ACTIVE =
      SubscriptionStatus._(
          1, _omitEnumNames ? '' : 'SUBSCRIPTION_STATUS_ACTIVE');
  static const SubscriptionStatus SUBSCRIPTION_STATUS_TRIALING =
      SubscriptionStatus._(
          2, _omitEnumNames ? '' : 'SUBSCRIPTION_STATUS_TRIALING');
  static const SubscriptionStatus SUBSCRIPTION_STATUS_IN_GRACE_PERIOD =
      SubscriptionStatus._(
          3, _omitEnumNames ? '' : 'SUBSCRIPTION_STATUS_IN_GRACE_PERIOD');

  /// Google "on hold": the renewal is being retried after a payment failure.
  static const SubscriptionStatus SUBSCRIPTION_STATUS_IN_BILLING_RETRY =
      SubscriptionStatus._(
          4, _omitEnumNames ? '' : 'SUBSCRIPTION_STATUS_IN_BILLING_RETRY');
  static const SubscriptionStatus SUBSCRIPTION_STATUS_PAUSED =
      SubscriptionStatus._(
          5, _omitEnumNames ? '' : 'SUBSCRIPTION_STATUS_PAUSED');
  static const SubscriptionStatus SUBSCRIPTION_STATUS_EXPIRED =
      SubscriptionStatus._(
          6, _omitEnumNames ? '' : 'SUBSCRIPTION_STATUS_EXPIRED');
  static const SubscriptionStatus SUBSCRIPTION_STATUS_REVOKED =
      SubscriptionStatus._(
          7, _omitEnumNames ? '' : 'SUBSCRIPTION_STATUS_REVOKED');

  static const $core.List<SubscriptionStatus> values = <SubscriptionStatus>[
    SUBSCRIPTION_STATUS_UNSPECIFIED,
    SUBSCRIPTION_STATUS_ACTIVE,
    SUBSCRIPTION_STATUS_TRIALING,
    SUBSCRIPTION_STATUS_IN_GRACE_PERIOD,
    SUBSCRIPTION_STATUS_IN_BILLING_RETRY,
    SUBSCRIPTION_STATUS_PAUSED,
    SUBSCRIPTION_STATUS_EXPIRED,
    SUBSCRIPTION_STATUS_REVOKED,
  ];

  static final $core.List<SubscriptionStatus?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 7);
  static SubscriptionStatus? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const SubscriptionStatus._(super.value, super.name);
}

/// EntitlementSource explains why an entitlement is active.
class EntitlementSource extends $pb.ProtobufEnum {
  static const EntitlementSource ENTITLEMENT_SOURCE_UNSPECIFIED =
      EntitlementSource._(
          0, _omitEnumNames ? '' : 'ENTITLEMENT_SOURCE_UNSPECIFIED');

  /// Granted by an active store subscription.
  static const EntitlementSource ENTITLEMENT_SOURCE_STORE =
      EntitlementSource._(1, _omitEnumNames ? '' : 'ENTITLEMENT_SOURCE_STORE');

  /// Granted by an operator (promo/comp), independent of store state.
  static const EntitlementSource ENTITLEMENT_SOURCE_GRANT =
      EntitlementSource._(2, _omitEnumNames ? '' : 'ENTITLEMENT_SOURCE_GRANT');

  /// The built-in free tier (no active subscription or grant). Reserved; the
  /// free tier is normally conveyed by an empty active_entitlements list.
  static const EntitlementSource ENTITLEMENT_SOURCE_NONE =
      EntitlementSource._(3, _omitEnumNames ? '' : 'ENTITLEMENT_SOURCE_NONE');

  static const $core.List<EntitlementSource> values = <EntitlementSource>[
    ENTITLEMENT_SOURCE_UNSPECIFIED,
    ENTITLEMENT_SOURCE_STORE,
    ENTITLEMENT_SOURCE_GRANT,
    ENTITLEMENT_SOURCE_NONE,
  ];

  static final $core.List<EntitlementSource?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static EntitlementSource? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const EntitlementSource._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
