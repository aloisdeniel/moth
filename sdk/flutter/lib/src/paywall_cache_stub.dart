import 'paywall_cache.dart';

/// Web (and any non-`dart:io`) fallback: the paywall config is kept in memory
/// only.
MothPaywallCache createPaywallCache(String publishableKey) =>
    MothMemoryPaywallCache();
