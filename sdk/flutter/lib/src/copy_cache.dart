import 'dart:ui';

import 'copy.dart';
import 'copy_cache_stub.dart' if (dart.library.io) 'copy_cache_io.dart' as impl;

/// A cached [MothCopy] together with the moment it was fetched or last
/// revalidated against the server — the timestamp that drives the
/// download-once TTL ([MothConfig.configCacheTtl]).
class MothCachedCopy {
  const MothCachedCopy({required this.copy, required this.fetchedAt});

  final MothCopy copy;

  /// When this locale's copy was fetched or last revalidated (UTC).
  final DateTime fetchedAt;
}

/// Where the SDK persists the last delivered [MothCopy] per locale, keyed by
/// `(language, revisionId)`, so a launch renders the project's localized copy
/// immediately (stale-while-revalidate) and `GetProjectConfig` can omit the
/// `messages` body when the cached revision still matches — the same pattern
/// as the theme cache (milestone 06) and paywall cache (milestone 13), now
/// parameterized by locale.
///
/// The key is the *language* only ([Locale.languageCode]), not the full BCP-47
/// tag: the load key is the device/override locale (which may carry a region,
/// e.g. `en-US`, `pt-BR`), while the save key is the locale the server
/// negotiated (language-only for the bundled set, e.g. `en`, `pt`). Keying on
/// the shared language makes the cache round-trip so an offline relaunch on a
/// region-tagged device still renders the project's cached copy.
///
/// The file cache stores a `moth.storage.v1.CacheEnvelope` protobuf whose
/// payload is the raw `moth.auth.v1.Copy` wire message exactly as the server
/// delivered it, with the envelope's `locale` set to the negotiated tag and
/// the fetch time that drives the download-once TTL. The copy is not secret
/// (the server re-delivers it), so a plain file — not secure storage — is the
/// right place. All methods may throw (broken storage); callers treat
/// failures as cache misses.
abstract class MothCopyCache {
  /// The cached copy for [locale]'s language and its fetch time, or null on
  /// a miss (nothing cached, or an unreadable/legacy cache file).
  Future<MothCachedCopy?> load(Locale locale);

  /// Persists [copy] under its own locale, stamped with [fetchedAt].
  Future<void> save(MothCopy copy, {required DateTime fetchedAt});

  /// Re-stamps the fetch time of [locale]'s cached copy after the server
  /// confirmed the cached revision is still current (an omitted-body
  /// revalidation), so the download-once TTL window restarts. No-op on a
  /// miss.
  Future<void> touch(Locale locale, DateTime fetchedAt);
}

/// The platform default cache, namespaced by publishable key so two projects
/// on one device never collide.
MothCopyCache defaultCopyCache(String publishableKey) =>
    impl.createCopyCache(publishableKey);

/// Keeps the copy in memory only — nothing survives a restart. The default on
/// Flutter Web, and handy in tests.
class MothMemoryCopyCache implements MothCopyCache {
  final _byLocale = <String, MothCachedCopy>{};

  @override
  Future<MothCachedCopy?> load(Locale locale) async =>
      _byLocale[locale.languageCode];

  @override
  Future<void> save(MothCopy copy, {required DateTime fetchedAt}) async {
    _byLocale[copy.locale.languageCode] = MothCachedCopy(
      copy: copy,
      fetchedAt: fetchedAt,
    );
  }

  @override
  Future<void> touch(Locale locale, DateTime fetchedAt) async {
    final entry = _byLocale[locale.languageCode];
    if (entry == null) return;
    _byLocale[locale.languageCode] = MothCachedCopy(
      copy: entry.copy,
      fetchedAt: fetchedAt,
    );
  }
}
