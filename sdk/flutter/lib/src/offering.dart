import 'gen/moth/billing/v1/billing.pb.dart' as pb;
import 'gen/moth/billing/v1/billing.pbenum.dart' as pbe;

/// One purchasable tier as the paywall needs it: enough to render a card and
/// match the native store product.
///
/// Price/period are display + analytics metadata; the native store read stays
/// authoritative for the localized price actually charged. The app never gates
/// on [identifier] — it gates on [entitlements] — but the SDK uses it to drive
/// purchases.
class MothOfferingProduct {
  const MothOfferingProduct({
    required this.identifier,
    this.displayName = '',
    this.appleProductId = '',
    this.googleProductId = '',
    this.billingPeriod = '',
    this.priceAmountMicros = 0,
    this.currency = '',
    this.trialPeriod = '',
    this.introPriceAmountMicros = 0,
    this.introPeriod = '',
    this.entitlements = const [],
    this.sortOrder = 0,
    this.highlighted = false,
  });

  factory MothOfferingProduct.fromProto(pb.OfferingProduct proto) =>
      MothOfferingProduct(
        identifier: proto.identifier,
        displayName: proto.displayName,
        appleProductId: proto.appleProductId,
        googleProductId: proto.googleProductId,
        billingPeriod: proto.billingPeriod,
        priceAmountMicros: proto.priceAmountMicros.toInt(),
        currency: proto.currency,
        trialPeriod: proto.trialPeriod,
        introPriceAmountMicros: proto.introPriceAmountMicros.toInt(),
        introPeriod: proto.introPeriod,
        entitlements: List<String>.unmodifiable(proto.entitlements),
        sortOrder: proto.sortOrder,
        highlighted: proto.highlighted,
      );

  /// Stable moth catalog identifier (e.g. `monthly`).
  final String identifier;
  final String displayName;

  /// Store SKUs; either may be empty when the tier ships on one store only.
  final String appleProductId;
  final String googleProductId;

  /// ISO-8601 period descriptors (e.g. `P1M`); empty when unset.
  final String billingPeriod;

  /// Price in micros of [currency] (1,000,000 = one unit). Display metadata;
  /// the native store price is authoritative.
  final int priceAmountMicros;
  final String currency;

  /// Trial/intro descriptors (display + analytics only).
  final String trialPeriod;
  final int introPriceAmountMicros;
  final String introPeriod;

  /// The stable entitlement identifiers this product grants while active.
  final List<String> entitlements;
  final int sortOrder;

  /// Whether this tier is the paywall's highlighted "most popular" tier.
  final bool highlighted;

  /// Whether this product offers a free trial.
  bool get hasTrial => trialPeriod.isNotEmpty;

  /// The store SKU for [store], or the moth [identifier] when the store SKU
  /// is not set.
  String storeProductId(pbe.Store store) {
    final sku = store == pbe.Store.STORE_APPLE
        ? appleProductId
        : googleProductId;
    return sku.isEmpty ? identifier : sku;
  }
}

/// The ordered set of products a paywall presents — the products sharing an
/// `offering` tag, in sort order. Every project has a default offering.
class MothOffering {
  const MothOffering({
    required this.identifier,
    this.isDefault = false,
    this.products = const [],
  });

  factory MothOffering.fromProto(pb.Offering proto) => MothOffering(
    identifier: proto.identifier,
    isDefault: proto.isDefault,
    products: proto.products
        .map(MothOfferingProduct.fromProto)
        .toList(growable: false),
  );

  /// Offering tag; `default` for the project's default offering.
  final String identifier;
  final bool isDefault;

  /// The products to display, in paywall order.
  final List<MothOfferingProduct> products;

  /// True when there is nothing to sell.
  bool get isEmpty => products.isEmpty;

  /// The product with [identifier], or null.
  MothOfferingProduct? productById(String identifier) =>
      products.where((p) => p.identifier == identifier).firstOrNull;

  /// Whether any product in this offering grants [entitlement].
  bool grants(String entitlement) =>
      products.any((p) => p.entitlements.contains(entitlement));
}

/// The rendering variant the paywall screen uses; the token space
/// (colors/spacing/radius) always comes from the theme.
enum MothPaywallLayout {
  /// One card per tier, side by side (the default).
  tiles,

  /// Tiers stacked as full-width rows.
  list,

  /// A single selected tier with a period toggle.
  compact;

  static MothPaywallLayout fromProto(pbe.PaywallLayout proto) =>
      switch (proto) {
        pbe.PaywallLayout.PAYWALL_LAYOUT_LIST => MothPaywallLayout.list,
        pbe.PaywallLayout.PAYWALL_LAYOUT_COMPACT => MothPaywallLayout.compact,
        _ => MothPaywallLayout.tiles,
      };

  static MothPaywallLayout fromName(String name) => switch (name) {
    'list' => MothPaywallLayout.list,
    'compact' => MothPaywallLayout.compact,
    _ => MothPaywallLayout.tiles,
  };
}

/// The public, render-ready paywall configuration, from
/// `moth.billing.v1.GetPaywall`. Copy and layout only — colors/typography
/// inherit from the [MothTheme].
class MothPaywall {
  const MothPaywall({
    this.revisionId = '',
    this.headline = 'Unlock Premium',
    this.subtitle = '',
    this.benefits = const [],
    this.offering = '',
    this.highlightedProductIdentifier = '',
    this.layout = MothPaywallLayout.tiles,
    this.termsUrl,
    this.privacyUrl,
  });

  factory MothPaywall.fromProto(pb.Paywall proto) {
    String? blank(String s) => s.isEmpty ? null : s;
    return MothPaywall(
      revisionId: proto.revisionId,
      headline: proto.headline,
      subtitle: proto.subtitle,
      benefits: List<String>.unmodifiable(proto.benefits),
      offering: proto.offering,
      highlightedProductIdentifier: proto.highlightedProductIdentifier,
      layout: MothPaywallLayout.fromProto(proto.layout),
      termsUrl: blank(proto.termsUrl),
      privacyUrl: blank(proto.privacyUrl),
    );
  }

  /// Rebuilds a paywall persisted with [toJson] (the client-side revision
  /// cache).
  factory MothPaywall.fromJson(Map<String, Object?> json) {
    String? blank(Object? s) => (s is String && s.isNotEmpty) ? s : null;
    return MothPaywall(
      revisionId: json['revisionId'] as String? ?? '',
      headline: json['headline'] as String? ?? '',
      subtitle: json['subtitle'] as String? ?? '',
      benefits: List<String>.unmodifiable(
        (json['benefits'] as List<Object?>? ?? const []).cast<String>(),
      ),
      offering: json['offering'] as String? ?? '',
      highlightedProductIdentifier:
          json['highlightedProductIdentifier'] as String? ?? '',
      layout: MothPaywallLayout.fromName(json['layout'] as String? ?? 'tiles'),
      termsUrl: blank(json['termsUrl']),
      privacyUrl: blank(json['privacyUrl']),
    );
  }

  /// Identifies this version of the paywall config; changes on every admin
  /// edit. Cache the paywall keyed by this value and echo it as
  /// `knownPaywallRevision`.
  final String revisionId;
  final String headline;
  final String subtitle;

  /// Feature/benefit bullets, in display order.
  final List<String> benefits;

  /// The offering tag whose products this paywall lists; empty selects the
  /// default offering.
  final String offering;

  /// The product identifier to render as "most popular"; empty for none.
  final String highlightedProductIdentifier;
  final MothPaywallLayout layout;

  /// Optional legal links rendered in the paywall footer.
  final String? termsUrl;
  final String? privacyUrl;

  /// Serializes the config for the client-side revision cache.
  Map<String, Object?> toJson() => {
    'revisionId': revisionId,
    'headline': headline,
    'subtitle': subtitle,
    'benefits': benefits,
    'offering': offering,
    'highlightedProductIdentifier': highlightedProductIdentifier,
    'layout': layout.name,
    if (termsUrl != null) 'termsUrl': termsUrl,
    if (privacyUrl != null) 'privacyUrl': privacyUrl,
  };
}
