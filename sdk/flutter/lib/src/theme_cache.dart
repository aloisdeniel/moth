import 'dart:typed_data';

import 'theme.dart';
import 'theme_cache_stub.dart'
    if (dart.library.io) 'theme_cache_io.dart'
    as impl;

/// A cached [MothTheme] together with the moment it was fetched or last
/// revalidated against the server — the timestamp that drives the
/// download-once TTL ([MothConfig.configCacheTtl]).
class MothCachedTheme {
  const MothCachedTheme({required this.theme, required this.fetchedAt});

  final MothTheme theme;

  /// When the theme was fetched or last revalidated (UTC).
  final DateTime fetchedAt;
}

/// Where the SDK persists the last delivered theme and downloaded font
/// files, so a launch renders the project's branding immediately
/// (stale-while-revalidate) and fonts are not re-downloaded.
///
/// The file cache stores a `moth.storage.v1.CacheEnvelope` protobuf whose
/// payload is the raw `moth.auth.v1.Theme` wire message exactly as the
/// server delivered it, stamped with the fetch time that drives the
/// download-once TTL. The default is a file cache under the app's support
/// directory (an in-memory cache on Flutter Web); swap in a
/// [MothMemoryThemeCache] for tests. All methods may throw (broken
/// storage); callers treat failures as cache misses.
abstract class MothThemeCache {
  /// The cached theme and its fetch time, or null on a miss (nothing
  /// cached, or an unreadable/legacy cache file).
  Future<MothCachedTheme?> loadTheme();

  /// Persists [theme] stamped with [fetchedAt] (when it was fetched).
  Future<void> saveTheme(MothTheme theme, {required DateTime fetchedAt});

  /// Re-stamps the cached theme's fetch time after the server confirmed the
  /// cached revision is still current (an omitted-body revalidation), so
  /// the download-once TTL window restarts. No-op on a miss.
  Future<void> touchTheme(DateTime fetchedAt);

  /// Cached font file bytes for [url], or null.
  Future<Uint8List?> loadFontBytes(String url);
  Future<void> saveFontBytes(String url, Uint8List bytes);
}

/// The platform default cache, namespaced by publishable key so two
/// projects on one device never collide.
MothThemeCache defaultThemeCache(String publishableKey) =>
    impl.createThemeCache(publishableKey);

/// Keeps the theme and font bytes in memory only — nothing survives a
/// restart. The default on Flutter Web, and handy in tests.
class MothMemoryThemeCache implements MothThemeCache {
  MothCachedTheme? _entry;
  final _fonts = <String, Uint8List>{};

  @override
  Future<MothCachedTheme?> loadTheme() async => _entry;

  @override
  Future<void> saveTheme(MothTheme theme, {required DateTime fetchedAt}) async {
    _entry = MothCachedTheme(theme: theme, fetchedAt: fetchedAt);
  }

  @override
  Future<void> touchTheme(DateTime fetchedAt) async {
    final entry = _entry;
    if (entry == null) return;
    _entry = MothCachedTheme(theme: entry.theme, fetchedAt: fetchedAt);
  }

  @override
  Future<Uint8List?> loadFontBytes(String url) async => _fonts[url];

  @override
  Future<void> saveFontBytes(String url, Uint8List bytes) async =>
      _fonts[url] = bytes;
}
