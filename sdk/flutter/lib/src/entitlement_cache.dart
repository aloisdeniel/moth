import 'customer_info.dart';
import 'entitlement_cache_stub.dart'
    if (dart.library.io) 'entitlement_cache_io.dart'
    as impl;

/// Where the SDK persists the last known [MothCustomerInfo] per user, so
/// gating is instant on launch (stale-while-revalidate) before the background
/// `GetCustomerInfo` confirms it.
///
/// The cache is keyed by user id so switching accounts never leaks
/// entitlements. Entries are a convenience, never the authority — the server
/// stays the source of truth. All methods may throw (broken storage); callers
/// treat failures as cache misses.
abstract class MothEntitlementCache {
  /// The cached info for [userId], or null on a miss.
  Future<MothCustomerInfo?> load(String userId);
  Future<void> save(String userId, MothCustomerInfo info);
}

/// The platform default cache, namespaced by publishable key so two projects
/// on one device never collide.
MothEntitlementCache defaultEntitlementCache(String publishableKey) =>
    impl.createEntitlementCache(publishableKey);

/// Keeps entitlements in memory only — nothing survives a restart. The default
/// on Flutter Web, and handy in tests.
class MothMemoryEntitlementCache implements MothEntitlementCache {
  final _byUser = <String, MothCustomerInfo>{};

  @override
  Future<MothCustomerInfo?> load(String userId) async => _byUser[userId];

  @override
  Future<void> save(String userId, MothCustomerInfo info) async =>
      _byUser[userId] = info;
}
