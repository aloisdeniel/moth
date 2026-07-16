import 'gen/moth/billing/v1/billing.pb.dart' as pb;
import 'gen/moth/billing/v1/billing.pbenum.dart' as pbe;

/// Which app store a purchase or subscription belongs to.
enum MothStore {
  apple,
  google;

  pbe.Store get proto => switch (this) {
    MothStore.apple => pbe.Store.STORE_APPLE,
    MothStore.google => pbe.Store.STORE_GOOGLE,
  };

  static MothStore? fromProto(pbe.Store store) => switch (store) {
    pbe.Store.STORE_APPLE => MothStore.apple,
    pbe.Store.STORE_GOOGLE => MothStore.google,
    _ => null,
  };
}

/// Why an entitlement is active.
enum MothEntitlementSource {
  /// Granted by an active store subscription.
  store,

  /// Granted by an operator (promo/comp), independent of store state.
  grant,

  /// The built-in free tier (no active subscription or grant).
  none;

  static MothEntitlementSource fromProto(pbe.EntitlementSource source) =>
      switch (source) {
        pbe.EntitlementSource.ENTITLEMENT_SOURCE_STORE =>
          MothEntitlementSource.store,
        pbe.EntitlementSource.ENTITLEMENT_SOURCE_GRANT =>
          MothEntitlementSource.grant,
        _ => MothEntitlementSource.none,
      };
}

/// The store's renewal state, mapped to a small set common to Apple and
/// Google. active/trialing/inGracePeriod/inBillingRetry keep access;
/// paused/expired/revoked do not.
enum MothSubscriptionStatus {
  unspecified,
  active,
  trialing,
  inGracePeriod,
  inBillingRetry,
  paused,
  expired,
  revoked;

  /// Whether this status keeps the subscription's access.
  bool get isActive => switch (this) {
    MothSubscriptionStatus.active ||
    MothSubscriptionStatus.trialing ||
    MothSubscriptionStatus.inGracePeriod ||
    MothSubscriptionStatus.inBillingRetry => true,
    _ => false,
  };

  static MothSubscriptionStatus fromProto(pbe.SubscriptionStatus status) =>
      switch (status) {
        pbe.SubscriptionStatus.SUBSCRIPTION_STATUS_ACTIVE =>
          MothSubscriptionStatus.active,
        pbe.SubscriptionStatus.SUBSCRIPTION_STATUS_TRIALING =>
          MothSubscriptionStatus.trialing,
        pbe.SubscriptionStatus.SUBSCRIPTION_STATUS_IN_GRACE_PERIOD =>
          MothSubscriptionStatus.inGracePeriod,
        pbe.SubscriptionStatus.SUBSCRIPTION_STATUS_IN_BILLING_RETRY =>
          MothSubscriptionStatus.inBillingRetry,
        pbe.SubscriptionStatus.SUBSCRIPTION_STATUS_PAUSED =>
          MothSubscriptionStatus.paused,
        pbe.SubscriptionStatus.SUBSCRIPTION_STATUS_EXPIRED =>
          MothSubscriptionStatus.expired,
        pbe.SubscriptionStatus.SUBSCRIPTION_STATUS_REVOKED =>
          MothSubscriptionStatus.revoked,
        _ => MothSubscriptionStatus.unspecified,
      };
}

/// One active capability the user holds (e.g. `pro`), with its expiry and why
/// it is active. Apps gate on [identifier], never on a product id.
class MothEntitlement {
  const MothEntitlement({
    required this.identifier,
    this.expireTime,
    this.source = MothEntitlementSource.none,
    this.productIdentifier = '',
  });

  factory MothEntitlement.fromProto(pb.Entitlement proto) => MothEntitlement(
    identifier: proto.identifier,
    expireTime: proto.hasExpireTime()
        ? proto.expireTime.toDateTime().toUtc()
        : null,
    source: MothEntitlementSource.fromProto(proto.source),
    productIdentifier: proto.productIdentifier,
  );

  factory MothEntitlement.fromJson(Map<String, Object?> json) =>
      MothEntitlement(
        identifier: json['identifier'] as String? ?? '',
        expireTime: json['expireTime'] == null
            ? null
            : DateTime.parse(json['expireTime'] as String),
        source: MothEntitlementSource.values.firstWhere(
          (s) => s.name == json['source'],
          orElse: () => MothEntitlementSource.none,
        ),
        productIdentifier: json['productIdentifier'] as String? ?? '',
      );

  /// Stable identifier the app checks (e.g. `pro`).
  final String identifier;

  /// When the entitlement lapses; null for a non-expiring grant.
  final DateTime? expireTime;

  /// Why it is active (store subscription vs operator grant).
  final MothEntitlementSource source;

  /// The moth product identifier that granted it, when [source] is
  /// [MothEntitlementSource.store]; empty for grants.
  final String productIdentifier;

  Map<String, Object?> toJson() => {
    'identifier': identifier,
    if (expireTime != null) 'expireTime': expireTime!.toUtc().toIso8601String(),
    'source': source.name,
    'productIdentifier': productIdentifier,
  };

  @override
  bool operator ==(Object other) =>
      other is MothEntitlement &&
      other.identifier == identifier &&
      other.expireTime == expireTime &&
      other.source == source &&
      other.productIdentifier == productIdentifier;

  @override
  int get hashCode =>
      Object.hash(identifier, expireTime, source, productIdentifier);
}

/// One of the user's store subscriptions (may be inactive, for history /
/// paywall display).
class MothActiveSubscription {
  const MothActiveSubscription({
    required this.productIdentifier,
    required this.store,
    required this.status,
    this.currentPeriodEnd,
    this.autoRenew = false,
    this.isSandbox = false,
  });

  factory MothActiveSubscription.fromProto(pb.ActiveSubscription proto) =>
      MothActiveSubscription(
        productIdentifier: proto.productIdentifier,
        store: MothStore.fromProto(proto.store),
        status: MothSubscriptionStatus.fromProto(proto.status),
        currentPeriodEnd: proto.hasCurrentPeriodEnd()
            ? proto.currentPeriodEnd.toDateTime().toUtc()
            : null,
        autoRenew: proto.autoRenew,
        isSandbox: proto.isSandbox,
      );

  factory MothActiveSubscription.fromJson(Map<String, Object?> json) =>
      MothActiveSubscription(
        productIdentifier: json['productIdentifier'] as String? ?? '',
        store: MothStore.values
            .where((s) => s.name == json['store'])
            .firstOrNull,
        status: MothSubscriptionStatus.values.firstWhere(
          (s) => s.name == json['status'],
          orElse: () => MothSubscriptionStatus.unspecified,
        ),
        currentPeriodEnd: json['currentPeriodEnd'] == null
            ? null
            : DateTime.parse(json['currentPeriodEnd'] as String),
        autoRenew: json['autoRenew'] as bool? ?? false,
        isSandbox: json['isSandbox'] as bool? ?? false,
      );

  final String productIdentifier;

  /// The store this subscription lives on; null when the server reported an
  /// unspecified store.
  final MothStore? store;
  final MothSubscriptionStatus status;

  /// End of the current paid (or trial) period; the renewal date when
  /// [autoRenew] is true.
  final DateTime? currentPeriodEnd;
  final bool autoRenew;

  /// Whether this subscription is a sandbox/test purchase.
  final bool isSandbox;

  Map<String, Object?> toJson() => {
    'productIdentifier': productIdentifier,
    if (store != null) 'store': store!.name,
    'status': status.name,
    if (currentPeriodEnd != null)
      'currentPeriodEnd': currentPeriodEnd!.toUtc().toIso8601String(),
    'autoRenew': autoRenew,
    'isSandbox': isSandbox,
  };

  @override
  bool operator ==(Object other) =>
      other is MothActiveSubscription &&
      other.productIdentifier == productIdentifier &&
      other.store == store &&
      other.status == status &&
      other.currentPeriodEnd == currentPeriodEnd &&
      other.autoRenew == autoRenew &&
      other.isSandbox == isSandbox;

  @override
  int get hashCode => Object.hash(
    productIdentifier,
    store,
    status,
    currentPeriodEnd,
    autoRenew,
    isSandbox,
  );
}

/// The complete subscription picture for one user, from
/// `moth.billing.v1.GetCustomerInfo`.
///
/// A never-paid user, a free-tier user, and a user in a project with no
/// products all get a well-formed instance with empty [activeEntitlements]
/// (the built-in `none` tier) — never an error. Gate features with
/// [hasEntitlement]; never special-case "never paid".
class MothCustomerInfo {
  const MothCustomerInfo({
    this.activeEntitlements = const [],
    this.subscriptions = const [],
  });

  /// The valid, empty state: no entitlements, the free `none` tier.
  const MothCustomerInfo.free() : this();

  factory MothCustomerInfo.fromProto(pb.CustomerInfo proto) => MothCustomerInfo(
    activeEntitlements: proto.activeEntitlements
        .map(MothEntitlement.fromProto)
        .toList(growable: false),
    subscriptions: proto.subscriptions
        .map(MothActiveSubscription.fromProto)
        .toList(growable: false),
  );

  factory MothCustomerInfo.fromJson(Map<String, Object?> json) =>
      MothCustomerInfo(
        activeEntitlements:
            (json['activeEntitlements'] as List<Object?>? ?? const [])
                .map((e) => MothEntitlement.fromJson(e as Map<String, Object?>))
                .toList(growable: false),
        subscriptions: (json['subscriptions'] as List<Object?>? ?? const [])
            .map(
              (e) => MothActiveSubscription.fromJson(e as Map<String, Object?>),
            )
            .toList(growable: false),
      );

  /// The entitlements the user currently holds. Empty means the free `none`
  /// tier — a valid, expected state, not an error.
  final List<MothEntitlement> activeEntitlements;

  /// The user's known subscriptions across stores.
  final List<MothActiveSubscription> subscriptions;

  /// Whether the user currently holds the entitlement [identifier] (e.g.
  /// `pro`). The single question app code should ask to gate a feature.
  bool hasEntitlement(String identifier) =>
      activeEntitlements.any((e) => e.identifier == identifier);

  /// The held entitlement with [identifier], or null.
  MothEntitlement? entitlement(String identifier) =>
      activeEntitlements.where((e) => e.identifier == identifier).firstOrNull;

  Map<String, Object?> toJson() => {
    'activeEntitlements': activeEntitlements
        .map((e) => e.toJson())
        .toList(growable: false),
    'subscriptions': subscriptions
        .map((s) => s.toJson())
        .toList(growable: false),
  };

  @override
  bool operator ==(Object other) =>
      other is MothCustomerInfo &&
      _listEquals(other.activeEntitlements, activeEntitlements) &&
      _listEquals(other.subscriptions, subscriptions);

  @override
  int get hashCode => Object.hash(
    Object.hashAll(activeEntitlements),
    Object.hashAll(subscriptions),
  );
}

bool _listEquals<T>(List<T> a, List<T> b) {
  if (a.length != b.length) return false;
  for (var i = 0; i < a.length; i++) {
    if (a[i] != b[i]) return false;
  }
  return true;
}
