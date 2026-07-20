// This is a generated file - do not edit.
//
// Generated from moth/billing/v1/billing.proto.

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

@$core.Deprecated('Use paywallLayoutDescriptor instead')
const PaywallLayout$json = {
  '1': 'PaywallLayout',
  '2': [
    {'1': 'PAYWALL_LAYOUT_UNSPECIFIED', '2': 0},
    {'1': 'PAYWALL_LAYOUT_TILES', '2': 1},
    {'1': 'PAYWALL_LAYOUT_LIST', '2': 2},
    {'1': 'PAYWALL_LAYOUT_COMPACT', '2': 3},
  ],
};

/// Descriptor for `PaywallLayout`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List paywallLayoutDescriptor = $convert.base64Decode(
    'Cg1QYXl3YWxsTGF5b3V0Eh4KGlBBWVdBTExfTEFZT1VUX1VOU1BFQ0lGSUVEEAASGAoUUEFZV0'
    'FMTF9MQVlPVVRfVElMRVMQARIXChNQQVlXQUxMX0xBWU9VVF9MSVNUEAISGgoWUEFZV0FMTF9M'
    'QVlPVVRfQ09NUEFDVBAD');

@$core.Deprecated('Use storeDescriptor instead')
const Store$json = {
  '1': 'Store',
  '2': [
    {'1': 'STORE_UNSPECIFIED', '2': 0},
    {'1': 'STORE_APPLE', '2': 1},
    {'1': 'STORE_GOOGLE', '2': 2},
    {'1': 'STORE_STRIPE', '2': 3},
  ],
};

/// Descriptor for `Store`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List storeDescriptor = $convert.base64Decode(
    'CgVTdG9yZRIVChFTVE9SRV9VTlNQRUNJRklFRBAAEg8KC1NUT1JFX0FQUExFEAESEAoMU1RPUk'
    'VfR09PR0xFEAISEAoMU1RPUkVfU1RSSVBFEAM=');

@$core.Deprecated('Use subscriptionStatusDescriptor instead')
const SubscriptionStatus$json = {
  '1': 'SubscriptionStatus',
  '2': [
    {'1': 'SUBSCRIPTION_STATUS_UNSPECIFIED', '2': 0},
    {'1': 'SUBSCRIPTION_STATUS_ACTIVE', '2': 1},
    {'1': 'SUBSCRIPTION_STATUS_TRIALING', '2': 2},
    {'1': 'SUBSCRIPTION_STATUS_IN_GRACE_PERIOD', '2': 3},
    {'1': 'SUBSCRIPTION_STATUS_IN_BILLING_RETRY', '2': 4},
    {'1': 'SUBSCRIPTION_STATUS_PAUSED', '2': 5},
    {'1': 'SUBSCRIPTION_STATUS_EXPIRED', '2': 6},
    {'1': 'SUBSCRIPTION_STATUS_REVOKED', '2': 7},
  ],
};

/// Descriptor for `SubscriptionStatus`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List subscriptionStatusDescriptor = $convert.base64Decode(
    'ChJTdWJzY3JpcHRpb25TdGF0dXMSIwofU1VCU0NSSVBUSU9OX1NUQVRVU19VTlNQRUNJRklFRB'
    'AAEh4KGlNVQlNDUklQVElPTl9TVEFUVVNfQUNUSVZFEAESIAocU1VCU0NSSVBUSU9OX1NUQVRV'
    'U19UUklBTElORxACEicKI1NVQlNDUklQVElPTl9TVEFUVVNfSU5fR1JBQ0VfUEVSSU9EEAMSKA'
    'okU1VCU0NSSVBUSU9OX1NUQVRVU19JTl9CSUxMSU5HX1JFVFJZEAQSHgoaU1VCU0NSSVBUSU9O'
    'X1NUQVRVU19QQVVTRUQQBRIfChtTVUJTQ1JJUFRJT05fU1RBVFVTX0VYUElSRUQQBhIfChtTVU'
    'JTQ1JJUFRJT05fU1RBVFVTX1JFVk9LRUQQBw==');

@$core.Deprecated('Use entitlementSourceDescriptor instead')
const EntitlementSource$json = {
  '1': 'EntitlementSource',
  '2': [
    {'1': 'ENTITLEMENT_SOURCE_UNSPECIFIED', '2': 0},
    {'1': 'ENTITLEMENT_SOURCE_STORE', '2': 1},
    {'1': 'ENTITLEMENT_SOURCE_GRANT', '2': 2},
    {'1': 'ENTITLEMENT_SOURCE_NONE', '2': 3},
  ],
};

/// Descriptor for `EntitlementSource`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List entitlementSourceDescriptor = $convert.base64Decode(
    'ChFFbnRpdGxlbWVudFNvdXJjZRIiCh5FTlRJVExFTUVOVF9TT1VSQ0VfVU5TUEVDSUZJRUQQAB'
    'IcChhFTlRJVExFTUVOVF9TT1VSQ0VfU1RPUkUQARIcChhFTlRJVExFTUVOVF9TT1VSQ0VfR1JB'
    'TlQQAhIbChdFTlRJVExFTUVOVF9TT1VSQ0VfTk9ORRAD');

@$core.Deprecated('Use offeringDescriptor instead')
const Offering$json = {
  '1': 'Offering',
  '2': [
    {'1': 'identifier', '3': 1, '4': 1, '5': 9, '10': 'identifier'},
    {'1': 'is_default', '3': 2, '4': 1, '5': 8, '10': 'isDefault'},
    {
      '1': 'products',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.moth.billing.v1.OfferingProduct',
      '10': 'products'
    },
  ],
};

/// Descriptor for `Offering`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List offeringDescriptor = $convert.base64Decode(
    'CghPZmZlcmluZxIeCgppZGVudGlmaWVyGAEgASgJUgppZGVudGlmaWVyEh0KCmlzX2RlZmF1bH'
    'QYAiABKAhSCWlzRGVmYXVsdBI8Cghwcm9kdWN0cxgDIAMoCzIgLm1vdGguYmlsbGluZy52MS5P'
    'ZmZlcmluZ1Byb2R1Y3RSCHByb2R1Y3Rz');

@$core.Deprecated('Use offeringProductDescriptor instead')
const OfferingProduct$json = {
  '1': 'OfferingProduct',
  '2': [
    {'1': 'identifier', '3': 1, '4': 1, '5': 9, '10': 'identifier'},
    {'1': 'display_name', '3': 2, '4': 1, '5': 9, '10': 'displayName'},
    {'1': 'apple_product_id', '3': 3, '4': 1, '5': 9, '10': 'appleProductId'},
    {'1': 'google_product_id', '3': 4, '4': 1, '5': 9, '10': 'googleProductId'},
    {'1': 'billing_period', '3': 5, '4': 1, '5': 9, '10': 'billingPeriod'},
    {
      '1': 'price_amount_micros',
      '3': 6,
      '4': 1,
      '5': 3,
      '10': 'priceAmountMicros'
    },
    {'1': 'currency', '3': 7, '4': 1, '5': 9, '10': 'currency'},
    {'1': 'trial_period', '3': 8, '4': 1, '5': 9, '10': 'trialPeriod'},
    {
      '1': 'intro_price_amount_micros',
      '3': 9,
      '4': 1,
      '5': 3,
      '10': 'introPriceAmountMicros'
    },
    {'1': 'intro_period', '3': 10, '4': 1, '5': 9, '10': 'introPeriod'},
    {'1': 'entitlements', '3': 11, '4': 3, '5': 9, '10': 'entitlements'},
    {'1': 'sort_order', '3': 12, '4': 1, '5': 5, '10': 'sortOrder'},
    {'1': 'highlighted', '3': 13, '4': 1, '5': 8, '10': 'highlighted'},
    {'1': 'stripe_price_id', '3': 14, '4': 1, '5': 9, '10': 'stripePriceId'},
  ],
};

/// Descriptor for `OfferingProduct`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List offeringProductDescriptor = $convert.base64Decode(
    'Cg9PZmZlcmluZ1Byb2R1Y3QSHgoKaWRlbnRpZmllchgBIAEoCVIKaWRlbnRpZmllchIhCgxkaX'
    'NwbGF5X25hbWUYAiABKAlSC2Rpc3BsYXlOYW1lEigKEGFwcGxlX3Byb2R1Y3RfaWQYAyABKAlS'
    'DmFwcGxlUHJvZHVjdElkEioKEWdvb2dsZV9wcm9kdWN0X2lkGAQgASgJUg9nb29nbGVQcm9kdW'
    'N0SWQSJQoOYmlsbGluZ19wZXJpb2QYBSABKAlSDWJpbGxpbmdQZXJpb2QSLgoTcHJpY2VfYW1v'
    'dW50X21pY3JvcxgGIAEoA1IRcHJpY2VBbW91bnRNaWNyb3MSGgoIY3VycmVuY3kYByABKAlSCG'
    'N1cnJlbmN5EiEKDHRyaWFsX3BlcmlvZBgIIAEoCVILdHJpYWxQZXJpb2QSOQoZaW50cm9fcHJp'
    'Y2VfYW1vdW50X21pY3JvcxgJIAEoA1IWaW50cm9QcmljZUFtb3VudE1pY3JvcxIhCgxpbnRyb1'
    '9wZXJpb2QYCiABKAlSC2ludHJvUGVyaW9kEiIKDGVudGl0bGVtZW50cxgLIAMoCVIMZW50aXRs'
    'ZW1lbnRzEh0KCnNvcnRfb3JkZXIYDCABKAVSCXNvcnRPcmRlchIgCgtoaWdobGlnaHRlZBgNIA'
    'EoCFILaGlnaGxpZ2h0ZWQSJgoPc3RyaXBlX3ByaWNlX2lkGA4gASgJUg1zdHJpcGVQcmljZUlk');

@$core.Deprecated('Use paywallDescriptor instead')
const Paywall$json = {
  '1': 'Paywall',
  '2': [
    {'1': 'revision_id', '3': 1, '4': 1, '5': 9, '10': 'revisionId'},
    {'1': 'headline', '3': 2, '4': 1, '5': 9, '10': 'headline'},
    {'1': 'subtitle', '3': 3, '4': 1, '5': 9, '10': 'subtitle'},
    {'1': 'benefits', '3': 4, '4': 3, '5': 9, '10': 'benefits'},
    {'1': 'offering', '3': 5, '4': 1, '5': 9, '10': 'offering'},
    {
      '1': 'highlighted_product_identifier',
      '3': 6,
      '4': 1,
      '5': 9,
      '10': 'highlightedProductIdentifier'
    },
    {
      '1': 'layout',
      '3': 7,
      '4': 1,
      '5': 14,
      '6': '.moth.billing.v1.PaywallLayout',
      '10': 'layout'
    },
    {'1': 'terms_url', '3': 8, '4': 1, '5': 9, '10': 'termsUrl'},
    {'1': 'privacy_url', '3': 9, '4': 1, '5': 9, '10': 'privacyUrl'},
  ],
};

/// Descriptor for `Paywall`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List paywallDescriptor = $convert.base64Decode(
    'CgdQYXl3YWxsEh8KC3JldmlzaW9uX2lkGAEgASgJUgpyZXZpc2lvbklkEhoKCGhlYWRsaW5lGA'
    'IgASgJUghoZWFkbGluZRIaCghzdWJ0aXRsZRgDIAEoCVIIc3VidGl0bGUSGgoIYmVuZWZpdHMY'
    'BCADKAlSCGJlbmVmaXRzEhoKCG9mZmVyaW5nGAUgASgJUghvZmZlcmluZxJECh5oaWdobGlnaH'
    'RlZF9wcm9kdWN0X2lkZW50aWZpZXIYBiABKAlSHGhpZ2hsaWdodGVkUHJvZHVjdElkZW50aWZp'
    'ZXISNgoGbGF5b3V0GAcgASgOMh4ubW90aC5iaWxsaW5nLnYxLlBheXdhbGxMYXlvdXRSBmxheW'
    '91dBIbCgl0ZXJtc191cmwYCCABKAlSCHRlcm1zVXJsEh8KC3ByaXZhY3lfdXJsGAkgASgJUgpw'
    'cml2YWN5VXJs');

@$core.Deprecated('Use getOfferingsRequestDescriptor instead')
const GetOfferingsRequest$json = {
  '1': 'GetOfferingsRequest',
  '2': [
    {'1': 'offering', '3': 1, '4': 1, '5': 9, '10': 'offering'},
  ],
};

/// Descriptor for `GetOfferingsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getOfferingsRequestDescriptor =
    $convert.base64Decode(
        'ChNHZXRPZmZlcmluZ3NSZXF1ZXN0EhoKCG9mZmVyaW5nGAEgASgJUghvZmZlcmluZw==');

@$core.Deprecated('Use getOfferingsResponseDescriptor instead')
const GetOfferingsResponse$json = {
  '1': 'GetOfferingsResponse',
  '2': [
    {
      '1': 'offering',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.moth.billing.v1.Offering',
      '10': 'offering'
    },
  ],
};

/// Descriptor for `GetOfferingsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getOfferingsResponseDescriptor = $convert.base64Decode(
    'ChRHZXRPZmZlcmluZ3NSZXNwb25zZRI1CghvZmZlcmluZxgBIAEoCzIZLm1vdGguYmlsbGluZy'
    '52MS5PZmZlcmluZ1IIb2ZmZXJpbmc=');

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
      '6': '.moth.billing.v1.Copy.MessagesEntry',
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
    'ABKAlSBmxvY2FsZRI/CghtZXNzYWdlcxgDIAMoCzIjLm1vdGguYmlsbGluZy52MS5Db3B5Lk1l'
    'c3NhZ2VzRW50cnlSCG1lc3NhZ2VzGjsKDU1lc3NhZ2VzRW50cnkSEAoDa2V5GAEgASgJUgNrZX'
    'kSFAoFdmFsdWUYAiABKAlSBXZhbHVlOgI4AQ==');

@$core.Deprecated('Use getPaywallRequestDescriptor instead')
const GetPaywallRequest$json = {
  '1': 'GetPaywallRequest',
  '2': [
    {
      '1': 'known_paywall_revision',
      '3': 1,
      '4': 1,
      '5': 9,
      '10': 'knownPaywallRevision'
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

/// Descriptor for `GetPaywallRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getPaywallRequestDescriptor = $convert.base64Decode(
    'ChFHZXRQYXl3YWxsUmVxdWVzdBI0ChZrbm93bl9wYXl3YWxsX3JldmlzaW9uGAEgASgJUhRrbm'
    '93blBheXdhbGxSZXZpc2lvbhIuChNrbm93bl9jb3B5X3JldmlzaW9uGAIgASgJUhFrbm93bkNv'
    'cHlSZXZpc2lvbg==');

@$core.Deprecated('Use getPaywallResponseDescriptor instead')
const GetPaywallResponse$json = {
  '1': 'GetPaywallResponse',
  '2': [
    {
      '1': 'paywall',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.moth.billing.v1.Paywall',
      '10': 'paywall'
    },
    {
      '1': 'copy',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.moth.billing.v1.Copy',
      '10': 'copy'
    },
  ],
};

/// Descriptor for `GetPaywallResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getPaywallResponseDescriptor = $convert.base64Decode(
    'ChJHZXRQYXl3YWxsUmVzcG9uc2USMgoHcGF5d2FsbBgBIAEoCzIYLm1vdGguYmlsbGluZy52MS'
    '5QYXl3YWxsUgdwYXl3YWxsEikKBGNvcHkYAiABKAsyFS5tb3RoLmJpbGxpbmcudjEuQ29weVIE'
    'Y29weQ==');

@$core.Deprecated('Use customerInfoDescriptor instead')
const CustomerInfo$json = {
  '1': 'CustomerInfo',
  '2': [
    {
      '1': 'active_entitlements',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.moth.billing.v1.Entitlement',
      '10': 'activeEntitlements'
    },
    {
      '1': 'subscriptions',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.moth.billing.v1.ActiveSubscription',
      '10': 'subscriptions'
    },
  ],
};

/// Descriptor for `CustomerInfo`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List customerInfoDescriptor = $convert.base64Decode(
    'CgxDdXN0b21lckluZm8STQoTYWN0aXZlX2VudGl0bGVtZW50cxgBIAMoCzIcLm1vdGguYmlsbG'
    'luZy52MS5FbnRpdGxlbWVudFISYWN0aXZlRW50aXRsZW1lbnRzEkkKDXN1YnNjcmlwdGlvbnMY'
    'AiADKAsyIy5tb3RoLmJpbGxpbmcudjEuQWN0aXZlU3Vic2NyaXB0aW9uUg1zdWJzY3JpcHRpb2'
    '5z');

@$core.Deprecated('Use entitlementDescriptor instead')
const Entitlement$json = {
  '1': 'Entitlement',
  '2': [
    {'1': 'identifier', '3': 1, '4': 1, '5': 9, '10': 'identifier'},
    {
      '1': 'expire_time',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'expireTime'
    },
    {
      '1': 'source',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.moth.billing.v1.EntitlementSource',
      '10': 'source'
    },
    {
      '1': 'product_identifier',
      '3': 4,
      '4': 1,
      '5': 9,
      '10': 'productIdentifier'
    },
  ],
};

/// Descriptor for `Entitlement`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List entitlementDescriptor = $convert.base64Decode(
    'CgtFbnRpdGxlbWVudBIeCgppZGVudGlmaWVyGAEgASgJUgppZGVudGlmaWVyEjsKC2V4cGlyZV'
    '90aW1lGAIgASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIKZXhwaXJlVGltZRI6CgZz'
    'b3VyY2UYAyABKA4yIi5tb3RoLmJpbGxpbmcudjEuRW50aXRsZW1lbnRTb3VyY2VSBnNvdXJjZR'
    'ItChJwcm9kdWN0X2lkZW50aWZpZXIYBCABKAlSEXByb2R1Y3RJZGVudGlmaWVy');

@$core.Deprecated('Use activeSubscriptionDescriptor instead')
const ActiveSubscription$json = {
  '1': 'ActiveSubscription',
  '2': [
    {
      '1': 'product_identifier',
      '3': 1,
      '4': 1,
      '5': 9,
      '10': 'productIdentifier'
    },
    {
      '1': 'store',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.moth.billing.v1.Store',
      '10': 'store'
    },
    {
      '1': 'status',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.moth.billing.v1.SubscriptionStatus',
      '10': 'status'
    },
    {
      '1': 'current_period_end',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'currentPeriodEnd'
    },
    {'1': 'auto_renew', '3': 5, '4': 1, '5': 8, '10': 'autoRenew'},
    {'1': 'is_sandbox', '3': 6, '4': 1, '5': 8, '10': 'isSandbox'},
  ],
};

/// Descriptor for `ActiveSubscription`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List activeSubscriptionDescriptor = $convert.base64Decode(
    'ChJBY3RpdmVTdWJzY3JpcHRpb24SLQoScHJvZHVjdF9pZGVudGlmaWVyGAEgASgJUhFwcm9kdW'
    'N0SWRlbnRpZmllchIsCgVzdG9yZRgCIAEoDjIWLm1vdGguYmlsbGluZy52MS5TdG9yZVIFc3Rv'
    'cmUSOwoGc3RhdHVzGAMgASgOMiMubW90aC5iaWxsaW5nLnYxLlN1YnNjcmlwdGlvblN0YXR1c1'
    'IGc3RhdHVzEkgKEmN1cnJlbnRfcGVyaW9kX2VuZBgEIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5U'
    'aW1lc3RhbXBSEGN1cnJlbnRQZXJpb2RFbmQSHQoKYXV0b19yZW5ldxgFIAEoCFIJYXV0b1Jlbm'
    'V3Eh0KCmlzX3NhbmRib3gYBiABKAhSCWlzU2FuZGJveA==');

@$core.Deprecated('Use getCustomerInfoRequestDescriptor instead')
const GetCustomerInfoRequest$json = {
  '1': 'GetCustomerInfoRequest',
};

/// Descriptor for `GetCustomerInfoRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getCustomerInfoRequestDescriptor =
    $convert.base64Decode('ChZHZXRDdXN0b21lckluZm9SZXF1ZXN0');

@$core.Deprecated('Use getCustomerInfoResponseDescriptor instead')
const GetCustomerInfoResponse$json = {
  '1': 'GetCustomerInfoResponse',
  '2': [
    {
      '1': 'customer_info',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.moth.billing.v1.CustomerInfo',
      '10': 'customerInfo'
    },
  ],
};

/// Descriptor for `GetCustomerInfoResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getCustomerInfoResponseDescriptor =
    $convert.base64Decode(
        'ChdHZXRDdXN0b21lckluZm9SZXNwb25zZRJCCg1jdXN0b21lcl9pbmZvGAEgASgLMh0ubW90aC'
        '5iaWxsaW5nLnYxLkN1c3RvbWVySW5mb1IMY3VzdG9tZXJJbmZv');

@$core.Deprecated('Use submitPurchaseRequestDescriptor instead')
const SubmitPurchaseRequest$json = {
  '1': 'SubmitPurchaseRequest',
  '2': [
    {
      '1': 'store',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.moth.billing.v1.Store',
      '10': 'store'
    },
    {
      '1': 'product_identifier',
      '3': 2,
      '4': 1,
      '5': 9,
      '10': 'productIdentifier'
    },
    {
      '1': 'apple_jws_transaction',
      '3': 3,
      '4': 1,
      '5': 9,
      '9': 0,
      '10': 'appleJwsTransaction'
    },
    {
      '1': 'google_purchase_token',
      '3': 4,
      '4': 1,
      '5': 9,
      '9': 0,
      '10': 'googlePurchaseToken'
    },
    {
      '1': 'google_subscription_id',
      '3': 5,
      '4': 1,
      '5': 9,
      '10': 'googleSubscriptionId'
    },
  ],
  '8': [
    {'1': 'receipt'},
  ],
};

/// Descriptor for `SubmitPurchaseRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List submitPurchaseRequestDescriptor = $convert.base64Decode(
    'ChVTdWJtaXRQdXJjaGFzZVJlcXVlc3QSLAoFc3RvcmUYASABKA4yFi5tb3RoLmJpbGxpbmcudj'
    'EuU3RvcmVSBXN0b3JlEi0KEnByb2R1Y3RfaWRlbnRpZmllchgCIAEoCVIRcHJvZHVjdElkZW50'
    'aWZpZXISNAoVYXBwbGVfandzX3RyYW5zYWN0aW9uGAMgASgJSABSE2FwcGxlSndzVHJhbnNhY3'
    'Rpb24SNAoVZ29vZ2xlX3B1cmNoYXNlX3Rva2VuGAQgASgJSABSE2dvb2dsZVB1cmNoYXNlVG9r'
    'ZW4SNAoWZ29vZ2xlX3N1YnNjcmlwdGlvbl9pZBgFIAEoCVIUZ29vZ2xlU3Vic2NyaXB0aW9uSW'
    'RCCQoHcmVjZWlwdA==');

@$core.Deprecated('Use submitPurchaseResponseDescriptor instead')
const SubmitPurchaseResponse$json = {
  '1': 'SubmitPurchaseResponse',
  '2': [
    {
      '1': 'customer_info',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.moth.billing.v1.CustomerInfo',
      '10': 'customerInfo'
    },
  ],
};

/// Descriptor for `SubmitPurchaseResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List submitPurchaseResponseDescriptor =
    $convert.base64Decode(
        'ChZTdWJtaXRQdXJjaGFzZVJlc3BvbnNlEkIKDWN1c3RvbWVyX2luZm8YASABKAsyHS5tb3RoLm'
        'JpbGxpbmcudjEuQ3VzdG9tZXJJbmZvUgxjdXN0b21lckluZm8=');

@$core.Deprecated('Use restorePurchasesRequestDescriptor instead')
const RestorePurchasesRequest$json = {
  '1': 'RestorePurchasesRequest',
  '2': [
    {
      '1': 'store',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.moth.billing.v1.Store',
      '10': 'store'
    },
    {'1': 'receipts', '3': 2, '4': 3, '5': 9, '10': 'receipts'},
  ],
};

/// Descriptor for `RestorePurchasesRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List restorePurchasesRequestDescriptor =
    $convert.base64Decode(
        'ChdSZXN0b3JlUHVyY2hhc2VzUmVxdWVzdBIsCgVzdG9yZRgBIAEoDjIWLm1vdGguYmlsbGluZy'
        '52MS5TdG9yZVIFc3RvcmUSGgoIcmVjZWlwdHMYAiADKAlSCHJlY2VpcHRz');

@$core.Deprecated('Use restorePurchasesResponseDescriptor instead')
const RestorePurchasesResponse$json = {
  '1': 'RestorePurchasesResponse',
  '2': [
    {
      '1': 'customer_info',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.moth.billing.v1.CustomerInfo',
      '10': 'customerInfo'
    },
  ],
};

/// Descriptor for `RestorePurchasesResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List restorePurchasesResponseDescriptor =
    $convert.base64Decode(
        'ChhSZXN0b3JlUHVyY2hhc2VzUmVzcG9uc2USQgoNY3VzdG9tZXJfaW5mbxgBIAEoCzIdLm1vdG'
        'guYmlsbGluZy52MS5DdXN0b21lckluZm9SDGN1c3RvbWVySW5mbw==');

@$core.Deprecated('Use createCheckoutSessionRequestDescriptor instead')
const CreateCheckoutSessionRequest$json = {
  '1': 'CreateCheckoutSessionRequest',
  '2': [
    {
      '1': 'product_identifier',
      '3': 1,
      '4': 1,
      '5': 9,
      '10': 'productIdentifier'
    },
    {'1': 'success_url', '3': 2, '4': 1, '5': 9, '10': 'successUrl'},
    {'1': 'cancel_url', '3': 3, '4': 1, '5': 9, '10': 'cancelUrl'},
  ],
};

/// Descriptor for `CreateCheckoutSessionRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createCheckoutSessionRequestDescriptor =
    $convert.base64Decode(
        'ChxDcmVhdGVDaGVja291dFNlc3Npb25SZXF1ZXN0Ei0KEnByb2R1Y3RfaWRlbnRpZmllchgBIA'
        'EoCVIRcHJvZHVjdElkZW50aWZpZXISHwoLc3VjY2Vzc191cmwYAiABKAlSCnN1Y2Nlc3NVcmwS'
        'HQoKY2FuY2VsX3VybBgDIAEoCVIJY2FuY2VsVXJs');

@$core.Deprecated('Use createCheckoutSessionResponseDescriptor instead')
const CreateCheckoutSessionResponse$json = {
  '1': 'CreateCheckoutSessionResponse',
  '2': [
    {'1': 'url', '3': 1, '4': 1, '5': 9, '10': 'url'},
  ],
};

/// Descriptor for `CreateCheckoutSessionResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createCheckoutSessionResponseDescriptor =
    $convert.base64Decode(
        'Ch1DcmVhdGVDaGVja291dFNlc3Npb25SZXNwb25zZRIQCgN1cmwYASABKAlSA3VybA==');

@$core.Deprecated('Use createBillingPortalSessionRequestDescriptor instead')
const CreateBillingPortalSessionRequest$json = {
  '1': 'CreateBillingPortalSessionRequest',
  '2': [
    {'1': 'return_url', '3': 1, '4': 1, '5': 9, '10': 'returnUrl'},
  ],
};

/// Descriptor for `CreateBillingPortalSessionRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createBillingPortalSessionRequestDescriptor =
    $convert.base64Decode(
        'CiFDcmVhdGVCaWxsaW5nUG9ydGFsU2Vzc2lvblJlcXVlc3QSHQoKcmV0dXJuX3VybBgBIAEoCV'
        'IJcmV0dXJuVXJs');

@$core.Deprecated('Use createBillingPortalSessionResponseDescriptor instead')
const CreateBillingPortalSessionResponse$json = {
  '1': 'CreateBillingPortalSessionResponse',
  '2': [
    {'1': 'url', '3': 1, '4': 1, '5': 9, '10': 'url'},
  ],
};

/// Descriptor for `CreateBillingPortalSessionResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createBillingPortalSessionResponseDescriptor =
    $convert.base64Decode(
        'CiJDcmVhdGVCaWxsaW5nUG9ydGFsU2Vzc2lvblJlc3BvbnNlEhAKA3VybBgBIAEoCVIDdXJs');
