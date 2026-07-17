import 'dart:ui';

import 'copy.dart';
import 'copy_cache_stub.dart' if (dart.library.io) 'copy_cache_io.dart' as impl;

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
/// The copy is not secret (the server re-delivers it), so a plain file — not
/// secure storage — is the right place. All methods may throw (broken
/// storage); callers treat failures as cache misses.
abstract class MothCopyCache {
  /// The cached copy for [locale]'s language, or null on a miss.
  Future<MothCopy?> load(Locale locale);

  /// Persists [copy] under its own locale.
  Future<void> save(MothCopy copy);
}

/// The platform default cache, namespaced by publishable key so two projects
/// on one device never collide.
MothCopyCache defaultCopyCache(String publishableKey) =>
    impl.createCopyCache(publishableKey);

/// Keeps the copy in memory only — nothing survives a restart. The default on
/// Flutter Web, and handy in tests.
class MothMemoryCopyCache implements MothCopyCache {
  final _byLocale = <String, MothCopy>{};

  @override
  Future<MothCopy?> load(Locale locale) async => _byLocale[locale.languageCode];

  @override
  Future<void> save(MothCopy copy) async =>
      _byLocale[copy.locale.languageCode] = copy;
}
