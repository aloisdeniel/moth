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

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;
import 'package:protobuf/well_known_types/google/protobuf/timestamp.pb.dart'
    as $1;

import 'billing.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'billing.pbenum.dart';

/// Offering is the ordered set of products a paywall presents — the products
/// sharing an `offering` tag, in sort order. Every project has a default
/// offering ("default").
class Offering extends $pb.GeneratedMessage {
  factory Offering({
    $core.String? identifier,
    $core.bool? isDefault,
    $core.Iterable<OfferingProduct>? products,
  }) {
    final result = create();
    if (identifier != null) result.identifier = identifier;
    if (isDefault != null) result.isDefault = isDefault;
    if (products != null) result.products.addAll(products);
    return result;
  }

  Offering._();

  factory Offering.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Offering.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Offering',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'identifier')
    ..aOB(2, _omitFieldNames ? '' : 'isDefault')
    ..pPM<OfferingProduct>(3, _omitFieldNames ? '' : 'products',
        subBuilder: OfferingProduct.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Offering clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Offering copyWith(void Function(Offering) updates) =>
      super.copyWith((message) => updates(message as Offering)) as Offering;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Offering create() => Offering._();
  @$core.override
  Offering createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Offering getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Offering>(create);
  static Offering? _defaultInstance;

  /// Offering tag; "default" for the project's default offering.
  @$pb.TagNumber(1)
  $core.String get identifier => $_getSZ(0);
  @$pb.TagNumber(1)
  set identifier($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasIdentifier() => $_has(0);
  @$pb.TagNumber(1)
  void clearIdentifier() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get isDefault => $_getBF(1);
  @$pb.TagNumber(2)
  set isDefault($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasIsDefault() => $_has(1);
  @$pb.TagNumber(2)
  void clearIsDefault() => $_clearField(2);

  /// The products to display, in paywall order.
  @$pb.TagNumber(3)
  $pb.PbList<OfferingProduct> get products => $_getList(2);
}

/// OfferingProduct is one purchasable tier as the paywall needs it: enough to
/// render a card and match the native store product. Price/period are display
/// + analytics metadata; the native store read stays authoritative for the
/// localized price actually charged.
class OfferingProduct extends $pb.GeneratedMessage {
  factory OfferingProduct({
    $core.String? identifier,
    $core.String? displayName,
    $core.String? appleProductId,
    $core.String? googleProductId,
    $core.String? billingPeriod,
    $fixnum.Int64? priceAmountMicros,
    $core.String? currency,
    $core.String? trialPeriod,
    $fixnum.Int64? introPriceAmountMicros,
    $core.String? introPeriod,
    $core.Iterable<$core.String>? entitlements,
    $core.int? sortOrder,
    $core.bool? highlighted,
    $core.String? stripePriceId,
  }) {
    final result = create();
    if (identifier != null) result.identifier = identifier;
    if (displayName != null) result.displayName = displayName;
    if (appleProductId != null) result.appleProductId = appleProductId;
    if (googleProductId != null) result.googleProductId = googleProductId;
    if (billingPeriod != null) result.billingPeriod = billingPeriod;
    if (priceAmountMicros != null) result.priceAmountMicros = priceAmountMicros;
    if (currency != null) result.currency = currency;
    if (trialPeriod != null) result.trialPeriod = trialPeriod;
    if (introPriceAmountMicros != null)
      result.introPriceAmountMicros = introPriceAmountMicros;
    if (introPeriod != null) result.introPeriod = introPeriod;
    if (entitlements != null) result.entitlements.addAll(entitlements);
    if (sortOrder != null) result.sortOrder = sortOrder;
    if (highlighted != null) result.highlighted = highlighted;
    if (stripePriceId != null) result.stripePriceId = stripePriceId;
    return result;
  }

  OfferingProduct._();

  factory OfferingProduct.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory OfferingProduct.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'OfferingProduct',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'identifier')
    ..aOS(2, _omitFieldNames ? '' : 'displayName')
    ..aOS(3, _omitFieldNames ? '' : 'appleProductId')
    ..aOS(4, _omitFieldNames ? '' : 'googleProductId')
    ..aOS(5, _omitFieldNames ? '' : 'billingPeriod')
    ..aInt64(6, _omitFieldNames ? '' : 'priceAmountMicros')
    ..aOS(7, _omitFieldNames ? '' : 'currency')
    ..aOS(8, _omitFieldNames ? '' : 'trialPeriod')
    ..aInt64(9, _omitFieldNames ? '' : 'introPriceAmountMicros')
    ..aOS(10, _omitFieldNames ? '' : 'introPeriod')
    ..pPS(11, _omitFieldNames ? '' : 'entitlements')
    ..aI(12, _omitFieldNames ? '' : 'sortOrder')
    ..aOB(13, _omitFieldNames ? '' : 'highlighted')
    ..aOS(14, _omitFieldNames ? '' : 'stripePriceId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OfferingProduct clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OfferingProduct copyWith(void Function(OfferingProduct) updates) =>
      super.copyWith((message) => updates(message as OfferingProduct))
          as OfferingProduct;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static OfferingProduct create() => OfferingProduct._();
  @$core.override
  OfferingProduct createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static OfferingProduct getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<OfferingProduct>(create);
  static OfferingProduct? _defaultInstance;

  /// Stable moth catalog identifier (e.g. "monthly"); the app never gates on
  /// this — it gates on entitlements — but the SDK uses it to drive purchases.
  @$pb.TagNumber(1)
  $core.String get identifier => $_getSZ(0);
  @$pb.TagNumber(1)
  set identifier($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasIdentifier() => $_has(0);
  @$pb.TagNumber(1)
  void clearIdentifier() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get displayName => $_getSZ(1);
  @$pb.TagNumber(2)
  set displayName($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDisplayName() => $_has(1);
  @$pb.TagNumber(2)
  void clearDisplayName() => $_clearField(2);

  /// Store SKUs so the SDK can pair this tier with the native store product;
  /// either may be empty when the tier ships on one store only.
  @$pb.TagNumber(3)
  $core.String get appleProductId => $_getSZ(2);
  @$pb.TagNumber(3)
  set appleProductId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAppleProductId() => $_has(2);
  @$pb.TagNumber(3)
  void clearAppleProductId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get googleProductId => $_getSZ(3);
  @$pb.TagNumber(4)
  set googleProductId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasGoogleProductId() => $_has(3);
  @$pb.TagNumber(4)
  void clearGoogleProductId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get billingPeriod => $_getSZ(4);
  @$pb.TagNumber(5)
  set billingPeriod($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasBillingPeriod() => $_has(4);
  @$pb.TagNumber(5)
  void clearBillingPeriod() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get priceAmountMicros => $_getI64(5);
  @$pb.TagNumber(6)
  set priceAmountMicros($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasPriceAmountMicros() => $_has(5);
  @$pb.TagNumber(6)
  void clearPriceAmountMicros() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get currency => $_getSZ(6);
  @$pb.TagNumber(7)
  set currency($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasCurrency() => $_has(6);
  @$pb.TagNumber(7)
  void clearCurrency() => $_clearField(7);

  /// Trial/intro descriptor (display + analytics only).
  @$pb.TagNumber(8)
  $core.String get trialPeriod => $_getSZ(7);
  @$pb.TagNumber(8)
  set trialPeriod($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasTrialPeriod() => $_has(7);
  @$pb.TagNumber(8)
  void clearTrialPeriod() => $_clearField(8);

  @$pb.TagNumber(9)
  $fixnum.Int64 get introPriceAmountMicros => $_getI64(8);
  @$pb.TagNumber(9)
  set introPriceAmountMicros($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasIntroPriceAmountMicros() => $_has(8);
  @$pb.TagNumber(9)
  void clearIntroPriceAmountMicros() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.String get introPeriod => $_getSZ(9);
  @$pb.TagNumber(10)
  set introPeriod($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasIntroPeriod() => $_has(9);
  @$pb.TagNumber(10)
  void clearIntroPeriod() => $_clearField(10);

  /// The stable entitlement identifiers this product grants while active (e.g.
  /// "pro"), so the paywall can label what a tier unlocks.
  @$pb.TagNumber(11)
  $pb.PbList<$core.String> get entitlements => $_getList(10);

  @$pb.TagNumber(12)
  $core.int get sortOrder => $_getIZ(11);
  @$pb.TagNumber(12)
  set sortOrder($core.int value) => $_setSignedInt32(11, value);
  @$pb.TagNumber(12)
  $core.bool hasSortOrder() => $_has(11);
  @$pb.TagNumber(12)
  void clearSortOrder() => $_clearField(12);

  /// Whether this tier is the paywall's highlighted "most popular" tier (from
  /// the paywall config's highlighted_product_identifier).
  @$pb.TagNumber(13)
  $core.bool get highlighted => $_getBF(12);
  @$pb.TagNumber(13)
  set highlighted($core.bool value) => $_setBool(12, value);
  @$pb.TagNumber(13)
  $core.bool hasHighlighted() => $_has(12);
  @$pb.TagNumber(13)
  void clearHighlighted() => $_clearField(13);

  /// Stripe recurring Price id ("price_..."); empty when the tier does not sell
  /// on the web (the React paywall marks such tiers unavailable).
  @$pb.TagNumber(14)
  $core.String get stripePriceId => $_getSZ(13);
  @$pb.TagNumber(14)
  set stripePriceId($core.String value) => $_setString(13, value);
  @$pb.TagNumber(14)
  $core.bool hasStripePriceId() => $_has(13);
  @$pb.TagNumber(14)
  void clearStripePriceId() => $_clearField(14);
}

/// Paywall is the public, render-ready paywall configuration. Copy and layout
/// only — colors/typography inherit from the theme.
class Paywall extends $pb.GeneratedMessage {
  factory Paywall({
    $core.String? revisionId,
    $core.String? headline,
    $core.String? subtitle,
    $core.Iterable<$core.String>? benefits,
    $core.String? offering,
    $core.String? highlightedProductIdentifier,
    PaywallLayout? layout,
    $core.String? termsUrl,
    $core.String? privacyUrl,
  }) {
    final result = create();
    if (revisionId != null) result.revisionId = revisionId;
    if (headline != null) result.headline = headline;
    if (subtitle != null) result.subtitle = subtitle;
    if (benefits != null) result.benefits.addAll(benefits);
    if (offering != null) result.offering = offering;
    if (highlightedProductIdentifier != null)
      result.highlightedProductIdentifier = highlightedProductIdentifier;
    if (layout != null) result.layout = layout;
    if (termsUrl != null) result.termsUrl = termsUrl;
    if (privacyUrl != null) result.privacyUrl = privacyUrl;
    return result;
  }

  Paywall._();

  factory Paywall.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Paywall.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Paywall',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'revisionId')
    ..aOS(2, _omitFieldNames ? '' : 'headline')
    ..aOS(3, _omitFieldNames ? '' : 'subtitle')
    ..pPS(4, _omitFieldNames ? '' : 'benefits')
    ..aOS(5, _omitFieldNames ? '' : 'offering')
    ..aOS(6, _omitFieldNames ? '' : 'highlightedProductIdentifier')
    ..aE<PaywallLayout>(7, _omitFieldNames ? '' : 'layout',
        enumValues: PaywallLayout.values)
    ..aOS(8, _omitFieldNames ? '' : 'termsUrl')
    ..aOS(9, _omitFieldNames ? '' : 'privacyUrl')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Paywall clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Paywall copyWith(void Function(Paywall) updates) =>
      super.copyWith((message) => updates(message as Paywall)) as Paywall;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Paywall create() => Paywall._();
  @$core.override
  Paywall createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Paywall getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Paywall>(create);
  static Paywall? _defaultInstance;

  /// Identifies this version of the paywall config; changes on every admin
  /// edit. Cache the paywall keyed by this value and echo it as
  /// GetPaywallRequest.known_paywall_revision.
  @$pb.TagNumber(1)
  $core.String get revisionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set revisionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRevisionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRevisionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get headline => $_getSZ(1);
  @$pb.TagNumber(2)
  set headline($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasHeadline() => $_has(1);
  @$pb.TagNumber(2)
  void clearHeadline() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get subtitle => $_getSZ(2);
  @$pb.TagNumber(3)
  set subtitle($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSubtitle() => $_has(2);
  @$pb.TagNumber(3)
  void clearSubtitle() => $_clearField(3);

  /// Feature/benefit bullets, in display order.
  @$pb.TagNumber(4)
  $pb.PbList<$core.String> get benefits => $_getList(3);

  /// The offering tag whose products this paywall lists; pass it to
  /// GetOfferings.offering. Empty selects the default offering.
  @$pb.TagNumber(5)
  $core.String get offering => $_getSZ(4);
  @$pb.TagNumber(5)
  set offering($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasOffering() => $_has(4);
  @$pb.TagNumber(5)
  void clearOffering() => $_clearField(5);

  /// The product identifier to render as "most popular"; empty for none.
  @$pb.TagNumber(6)
  $core.String get highlightedProductIdentifier => $_getSZ(5);
  @$pb.TagNumber(6)
  set highlightedProductIdentifier($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasHighlightedProductIdentifier() => $_has(5);
  @$pb.TagNumber(6)
  void clearHighlightedProductIdentifier() => $_clearField(6);

  @$pb.TagNumber(7)
  PaywallLayout get layout => $_getN(6);
  @$pb.TagNumber(7)
  set layout(PaywallLayout value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasLayout() => $_has(6);
  @$pb.TagNumber(7)
  void clearLayout() => $_clearField(7);

  /// Optional legal links rendered in the paywall footer.
  @$pb.TagNumber(8)
  $core.String get termsUrl => $_getSZ(7);
  @$pb.TagNumber(8)
  set termsUrl($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasTermsUrl() => $_has(7);
  @$pb.TagNumber(8)
  void clearTermsUrl() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.String get privacyUrl => $_getSZ(8);
  @$pb.TagNumber(9)
  set privacyUrl($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasPrivacyUrl() => $_has(8);
  @$pb.TagNumber(9)
  void clearPrivacyUrl() => $_clearField(9);
}

class GetOfferingsRequest extends $pb.GeneratedMessage {
  factory GetOfferingsRequest({
    $core.String? offering,
  }) {
    final result = create();
    if (offering != null) result.offering = offering;
    return result;
  }

  GetOfferingsRequest._();

  factory GetOfferingsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetOfferingsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetOfferingsRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'offering')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetOfferingsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetOfferingsRequest copyWith(void Function(GetOfferingsRequest) updates) =>
      super.copyWith((message) => updates(message as GetOfferingsRequest))
          as GetOfferingsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetOfferingsRequest create() => GetOfferingsRequest._();
  @$core.override
  GetOfferingsRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetOfferingsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetOfferingsRequest>(create);
  static GetOfferingsRequest? _defaultInstance;

  /// Offering tag; empty selects the project's default offering.
  @$pb.TagNumber(1)
  $core.String get offering => $_getSZ(0);
  @$pb.TagNumber(1)
  set offering($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOffering() => $_has(0);
  @$pb.TagNumber(1)
  void clearOffering() => $_clearField(1);
}

class GetOfferingsResponse extends $pb.GeneratedMessage {
  factory GetOfferingsResponse({
    Offering? offering,
  }) {
    final result = create();
    if (offering != null) result.offering = offering;
    return result;
  }

  GetOfferingsResponse._();

  factory GetOfferingsResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetOfferingsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetOfferingsResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOM<Offering>(1, _omitFieldNames ? '' : 'offering',
        subBuilder: Offering.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetOfferingsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetOfferingsResponse copyWith(void Function(GetOfferingsResponse) updates) =>
      super.copyWith((message) => updates(message as GetOfferingsResponse))
          as GetOfferingsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetOfferingsResponse create() => GetOfferingsResponse._();
  @$core.override
  GetOfferingsResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetOfferingsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetOfferingsResponse>(create);
  static GetOfferingsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  Offering get offering => $_getN(0);
  @$pb.TagNumber(1)
  set offering(Offering value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasOffering() => $_has(0);
  @$pb.TagNumber(1)
  void clearOffering() => $_clearField(1);
  @$pb.TagNumber(1)
  Offering ensureOffering() => $_ensure(0);
}

/// Copy is the resolved, localized paywall copy for the negotiated locale: the
/// paywall.* message key → localized-string map (headline, subtitle, benefit
/// bullets, CTA, legal labels), merged bundled-default → project-override. The
/// paywall copy keys are part of the same catalog as the auth-screen copy
/// (moth.auth.v1). The locale is negotiated server-side from the request's
/// Accept-Language / x-moth-language metadata; the client never dictates raw
/// copy. The structural Paywall message above stays authoritative for
/// layout/offering/tier selection.
class Copy extends $pb.GeneratedMessage {
  factory Copy({
    $core.String? copyRevision,
    $core.String? locale,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? messages,
  }) {
    final result = create();
    if (copyRevision != null) result.copyRevision = copyRevision;
    if (locale != null) result.locale = locale;
    if (messages != null) result.messages.addEntries(messages);
    return result;
  }

  Copy._();

  factory Copy.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Copy.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Copy',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'copyRevision')
    ..aOS(2, _omitFieldNames ? '' : 'locale')
    ..m<$core.String, $core.String>(3, _omitFieldNames ? '' : 'messages',
        entryClassName: 'Copy.MessagesEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('moth.billing.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Copy clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Copy copyWith(void Function(Copy) updates) =>
      super.copyWith((message) => updates(message as Copy)) as Copy;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Copy create() => Copy._();
  @$core.override
  Copy createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Copy getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Copy>(create);
  static Copy? _defaultInstance;

  /// Opaque cache token identifying this (locale, override-revision) pair.
  /// Cache `messages` keyed by it and echo it as
  /// GetPaywallRequest.known_copy_revision; the response omits `messages` when
  /// it still matches.
  @$pb.TagNumber(1)
  $core.String get copyRevision => $_getSZ(0);
  @$pb.TagNumber(1)
  set copyRevision($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCopyRevision() => $_has(0);
  @$pb.TagNumber(1)
  void clearCopyRevision() => $_clearField(1);

  /// The negotiated BCP-47 locale this copy is for (e.g. "fr").
  @$pb.TagNumber(2)
  $core.String get locale => $_getSZ(1);
  @$pb.TagNumber(2)
  set locale($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLocale() => $_has(1);
  @$pb.TagNumber(2)
  void clearLocale() => $_clearField(2);

  /// Resolved paywall.* message key → localized string for the negotiated
  /// locale.
  @$pb.TagNumber(3)
  $pb.PbMap<$core.String, $core.String> get messages => $_getMap(2);
}

class GetPaywallRequest extends $pb.GeneratedMessage {
  factory GetPaywallRequest({
    $core.String? knownPaywallRevision,
    $core.String? knownCopyRevision,
  }) {
    final result = create();
    if (knownPaywallRevision != null)
      result.knownPaywallRevision = knownPaywallRevision;
    if (knownCopyRevision != null) result.knownCopyRevision = knownCopyRevision;
    return result;
  }

  GetPaywallRequest._();

  factory GetPaywallRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetPaywallRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetPaywallRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'knownPaywallRevision')
    ..aOS(2, _omitFieldNames ? '' : 'knownCopyRevision')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetPaywallRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetPaywallRequest copyWith(void Function(GetPaywallRequest) updates) =>
      super.copyWith((message) => updates(message as GetPaywallRequest))
          as GetPaywallRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetPaywallRequest create() => GetPaywallRequest._();
  @$core.override
  GetPaywallRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetPaywallRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetPaywallRequest>(create);
  static GetPaywallRequest? _defaultInstance;

  /// The revision_id of the paywall the client has cached (empty on first
  /// call). When it still matches the current revision, the response omits
  /// `paywall`; see the caching contract on GetPaywall.
  @$pb.TagNumber(1)
  $core.String get knownPaywallRevision => $_getSZ(0);
  @$pb.TagNumber(1)
  set knownPaywallRevision($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasKnownPaywallRevision() => $_has(0);
  @$pb.TagNumber(1)
  void clearKnownPaywallRevision() => $_clearField(1);

  /// The copy_revision the client has cached for the locale it is about to
  /// render (empty on first call). When it still matches the token the server
  /// computes for the negotiated locale, the response's `copy` carries the
  /// locale + copy_revision but omits `messages`; when it differs (or was
  /// empty), `messages` is present. The negotiated locale comes from
  /// Accept-Language / x-moth-language metadata, never from this body.
  @$pb.TagNumber(2)
  $core.String get knownCopyRevision => $_getSZ(1);
  @$pb.TagNumber(2)
  set knownCopyRevision($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasKnownCopyRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearKnownCopyRevision() => $_clearField(2);
}

class GetPaywallResponse extends $pb.GeneratedMessage {
  factory GetPaywallResponse({
    Paywall? paywall,
    Copy? copy,
  }) {
    final result = create();
    if (paywall != null) result.paywall = paywall;
    if (copy != null) result.copy = copy;
    return result;
  }

  GetPaywallResponse._();

  factory GetPaywallResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetPaywallResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetPaywallResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOM<Paywall>(1, _omitFieldNames ? '' : 'paywall',
        subBuilder: Paywall.create)
    ..aOM<Copy>(2, _omitFieldNames ? '' : 'copy', subBuilder: Copy.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetPaywallResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetPaywallResponse copyWith(void Function(GetPaywallResponse) updates) =>
      super.copyWith((message) => updates(message as GetPaywallResponse))
          as GetPaywallResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetPaywallResponse create() => GetPaywallResponse._();
  @$core.override
  GetPaywallResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetPaywallResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetPaywallResponse>(create);
  static GetPaywallResponse? _defaultInstance;

  /// Omitted when GetPaywallRequest.known_paywall_revision matches the current
  /// revision; present otherwise (including for projects on the built-in
  /// default paywall config).
  @$pb.TagNumber(1)
  Paywall get paywall => $_getN(0);
  @$pb.TagNumber(1)
  set paywall(Paywall value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasPaywall() => $_has(0);
  @$pb.TagNumber(1)
  void clearPaywall() => $_clearField(1);
  @$pb.TagNumber(1)
  Paywall ensurePaywall() => $_ensure(0);

  /// The localized paywall copy for the negotiated locale. Always present (it
  /// carries the negotiated locale + copy_revision); its `messages` map is
  /// omitted when GetPaywallRequest.known_copy_revision matches, present
  /// otherwise — including for projects with no copy overrides.
  @$pb.TagNumber(2)
  Copy get copy => $_getN(1);
  @$pb.TagNumber(2)
  set copy(Copy value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasCopy() => $_has(1);
  @$pb.TagNumber(2)
  void clearCopy() => $_clearField(2);
  @$pb.TagNumber(2)
  Copy ensureCopy() => $_ensure(1);
}

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

class CreateCheckoutSessionRequest extends $pb.GeneratedMessage {
  factory CreateCheckoutSessionRequest({
    $core.String? productIdentifier,
    $core.String? successUrl,
    $core.String? cancelUrl,
  }) {
    final result = create();
    if (productIdentifier != null) result.productIdentifier = productIdentifier;
    if (successUrl != null) result.successUrl = successUrl;
    if (cancelUrl != null) result.cancelUrl = cancelUrl;
    return result;
  }

  CreateCheckoutSessionRequest._();

  factory CreateCheckoutSessionRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CreateCheckoutSessionRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateCheckoutSessionRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'productIdentifier')
    ..aOS(2, _omitFieldNames ? '' : 'successUrl')
    ..aOS(3, _omitFieldNames ? '' : 'cancelUrl')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateCheckoutSessionRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateCheckoutSessionRequest copyWith(
          void Function(CreateCheckoutSessionRequest) updates) =>
      super.copyWith(
              (message) => updates(message as CreateCheckoutSessionRequest))
          as CreateCheckoutSessionRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateCheckoutSessionRequest create() =>
      CreateCheckoutSessionRequest._();
  @$core.override
  CreateCheckoutSessionRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CreateCheckoutSessionRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateCheckoutSessionRequest>(create);
  static CreateCheckoutSessionRequest? _defaultInstance;

  /// The moth product identifier to subscribe to (its own catalog id, not the
  /// Stripe price id). The tier must carry a stripe_price_id.
  @$pb.TagNumber(1)
  $core.String get productIdentifier => $_getSZ(0);
  @$pb.TagNumber(1)
  set productIdentifier($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProductIdentifier() => $_has(0);
  @$pb.TagNumber(1)
  void clearProductIdentifier() => $_clearField(1);

  /// Where Stripe redirects the browser after a completed checkout.
  @$pb.TagNumber(2)
  $core.String get successUrl => $_getSZ(1);
  @$pb.TagNumber(2)
  set successUrl($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSuccessUrl() => $_has(1);
  @$pb.TagNumber(2)
  void clearSuccessUrl() => $_clearField(2);

  /// Where Stripe redirects the browser when the user backs out.
  @$pb.TagNumber(3)
  $core.String get cancelUrl => $_getSZ(2);
  @$pb.TagNumber(3)
  set cancelUrl($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCancelUrl() => $_has(2);
  @$pb.TagNumber(3)
  void clearCancelUrl() => $_clearField(3);
}

class CreateCheckoutSessionResponse extends $pb.GeneratedMessage {
  factory CreateCheckoutSessionResponse({
    $core.String? url,
  }) {
    final result = create();
    if (url != null) result.url = url;
    return result;
  }

  CreateCheckoutSessionResponse._();

  factory CreateCheckoutSessionResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CreateCheckoutSessionResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateCheckoutSessionResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'url')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateCheckoutSessionResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateCheckoutSessionResponse copyWith(
          void Function(CreateCheckoutSessionResponse) updates) =>
      super.copyWith(
              (message) => updates(message as CreateCheckoutSessionResponse))
          as CreateCheckoutSessionResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateCheckoutSessionResponse create() =>
      CreateCheckoutSessionResponse._();
  @$core.override
  CreateCheckoutSessionResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CreateCheckoutSessionResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateCheckoutSessionResponse>(create);
  static CreateCheckoutSessionResponse? _defaultInstance;

  /// The Stripe-hosted Checkout URL to redirect the browser to.
  @$pb.TagNumber(1)
  $core.String get url => $_getSZ(0);
  @$pb.TagNumber(1)
  set url($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasUrl() => $_has(0);
  @$pb.TagNumber(1)
  void clearUrl() => $_clearField(1);
}

class CreateBillingPortalSessionRequest extends $pb.GeneratedMessage {
  factory CreateBillingPortalSessionRequest({
    $core.String? returnUrl,
  }) {
    final result = create();
    if (returnUrl != null) result.returnUrl = returnUrl;
    return result;
  }

  CreateBillingPortalSessionRequest._();

  factory CreateBillingPortalSessionRequest.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CreateBillingPortalSessionRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateBillingPortalSessionRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'returnUrl')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateBillingPortalSessionRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateBillingPortalSessionRequest copyWith(
          void Function(CreateBillingPortalSessionRequest) updates) =>
      super.copyWith((message) =>
              updates(message as CreateBillingPortalSessionRequest))
          as CreateBillingPortalSessionRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateBillingPortalSessionRequest create() =>
      CreateBillingPortalSessionRequest._();
  @$core.override
  CreateBillingPortalSessionRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CreateBillingPortalSessionRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateBillingPortalSessionRequest>(
          create);
  static CreateBillingPortalSessionRequest? _defaultInstance;

  /// Where Stripe redirects the browser when the user leaves the portal.
  @$pb.TagNumber(1)
  $core.String get returnUrl => $_getSZ(0);
  @$pb.TagNumber(1)
  set returnUrl($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasReturnUrl() => $_has(0);
  @$pb.TagNumber(1)
  void clearReturnUrl() => $_clearField(1);
}

class CreateBillingPortalSessionResponse extends $pb.GeneratedMessage {
  factory CreateBillingPortalSessionResponse({
    $core.String? url,
  }) {
    final result = create();
    if (url != null) result.url = url;
    return result;
  }

  CreateBillingPortalSessionResponse._();

  factory CreateBillingPortalSessionResponse.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CreateBillingPortalSessionResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateBillingPortalSessionResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'moth.billing.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'url')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateBillingPortalSessionResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateBillingPortalSessionResponse copyWith(
          void Function(CreateBillingPortalSessionResponse) updates) =>
      super.copyWith((message) =>
              updates(message as CreateBillingPortalSessionResponse))
          as CreateBillingPortalSessionResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateBillingPortalSessionResponse create() =>
      CreateBillingPortalSessionResponse._();
  @$core.override
  CreateBillingPortalSessionResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CreateBillingPortalSessionResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateBillingPortalSessionResponse>(
          create);
  static CreateBillingPortalSessionResponse? _defaultInstance;

  /// The Stripe-hosted Billing Portal URL to redirect the browser to.
  @$pb.TagNumber(1)
  $core.String get url => $_getSZ(0);
  @$pb.TagNumber(1)
  set url($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasUrl() => $_has(0);
  @$pb.TagNumber(1)
  void clearUrl() => $_clearField(1);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
