import 'offering.dart';
import 'paywall_cache_stub.dart'
    if (dart.library.io) 'paywall_cache_io.dart'
    as impl;

/// Where the SDK persists the last delivered [MothPaywall] config, keyed by
/// its [MothPaywall.revisionId], so a launch renders the project's paywall
/// copy immediately (stale-while-revalidate) and `GetPaywall` can omit the
/// body when the cached revision still matches — the same pattern as the
/// theme cache in milestone 06.
///
/// The paywall config is not secret, so a plain file (not secure storage) is
/// the right place. All methods may throw (broken storage); callers treat
/// failures as cache misses.
abstract class MothPaywallCache {
  /// The cached paywall config, or null on a miss.
  Future<MothPaywall?> load();
  Future<void> save(MothPaywall paywall);
}

/// The platform default cache, namespaced by publishable key so two projects
/// on one device never collide.
MothPaywallCache defaultPaywallCache(String publishableKey) =>
    impl.createPaywallCache(publishableKey);

/// Keeps the paywall config in memory only — nothing survives a restart. The
/// default on Flutter Web, and handy in tests.
class MothMemoryPaywallCache implements MothPaywallCache {
  MothPaywall? _paywall;

  @override
  Future<MothPaywall?> load() async => _paywall;

  @override
  Future<void> save(MothPaywall paywall) async => _paywall = paywall;
}
