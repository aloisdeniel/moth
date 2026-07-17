import 'offering.dart';
import 'paywall_cache_stub.dart'
    if (dart.library.io) 'paywall_cache_io.dart'
    as impl;

/// A cached [MothPaywall] together with the moment it was fetched or last
/// revalidated against the server — the timestamp that drives the
/// download-once TTL ([MothConfig.configCacheTtl]).
class MothCachedPaywall {
  const MothCachedPaywall({required this.paywall, required this.fetchedAt});

  final MothPaywall paywall;

  /// When the paywall config was fetched or last revalidated (UTC).
  final DateTime fetchedAt;
}

/// Where the SDK persists the last delivered [MothPaywall] config, keyed by
/// its [MothPaywall.revisionId], so a launch renders the project's paywall
/// copy immediately (stale-while-revalidate) and `GetPaywall` can omit the
/// body when the cached revision still matches — the same pattern as the
/// theme cache in milestone 06.
///
/// The file cache stores a `moth.projectconfig.v1.CacheEnvelope` protobuf whose
/// payload is the raw `moth.billing.v1.Paywall` wire message exactly as the
/// server delivered it, stamped with the fetch time that drives the
/// download-once TTL. The paywall config is not secret, so a plain file
/// (not secure storage) is the right place. All methods may throw (broken
/// storage); callers treat failures as cache misses.
abstract class MothPaywallCache {
  /// The cached paywall config and its fetch time, or null on a miss
  /// (nothing cached, or an unreadable/legacy cache file).
  Future<MothCachedPaywall?> load();

  /// Persists [paywall] stamped with [fetchedAt] (when it was fetched).
  Future<void> save(MothPaywall paywall, {required DateTime fetchedAt});

  /// Re-stamps the cached config's fetch time after the server confirmed
  /// the cached revision is still current (an omitted-body revalidation),
  /// so the download-once TTL window restarts. No-op on a miss.
  Future<void> touch(DateTime fetchedAt);
}

/// The platform default cache, namespaced by publishable key so two projects
/// on one device never collide.
MothPaywallCache defaultPaywallCache(String publishableKey) =>
    impl.createPaywallCache(publishableKey);

/// Keeps the paywall config in memory only — nothing survives a restart. The
/// default on Flutter Web, and handy in tests.
class MothMemoryPaywallCache implements MothPaywallCache {
  MothCachedPaywall? _entry;

  @override
  Future<MothCachedPaywall?> load() async => _entry;

  @override
  Future<void> save(MothPaywall paywall, {required DateTime fetchedAt}) async {
    _entry = MothCachedPaywall(paywall: paywall, fetchedAt: fetchedAt);
  }

  @override
  Future<void> touch(DateTime fetchedAt) async {
    final entry = _entry;
    if (entry == null) return;
    _entry = MothCachedPaywall(paywall: entry.paywall, fetchedAt: fetchedAt);
  }
}
