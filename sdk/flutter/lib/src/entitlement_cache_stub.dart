import 'entitlement_cache.dart';

/// Web (and any non-`dart:io`) fallback: entitlements are kept in memory only.
MothEntitlementCache createEntitlementCache(String publishableKey) =>
    MothMemoryEntitlementCache();
