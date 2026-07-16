import 'dart:typed_data';

import 'theme.dart';
import 'theme_cache_stub.dart'
    if (dart.library.io) 'theme_cache_io.dart'
    as impl;

/// Where the SDK persists the last delivered theme and downloaded font
/// files, so a launch renders the project's branding immediately
/// (stale-while-revalidate) and fonts are not re-downloaded.
///
/// The default is a file cache under the app's support directory (an
/// in-memory cache on Flutter Web); swap in a [MothMemoryThemeCache] for
/// tests. All methods may throw (broken storage); callers treat failures
/// as cache misses.
abstract class MothThemeCache {
  Future<MothTheme?> loadTheme();
  Future<void> saveTheme(MothTheme theme);

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
  MothTheme? _theme;
  final _fonts = <String, Uint8List>{};

  @override
  Future<MothTheme?> loadTheme() async => _theme;

  @override
  Future<void> saveTheme(MothTheme theme) async => _theme = theme;

  @override
  Future<Uint8List?> loadFontBytes(String url) async => _fonts[url];

  @override
  Future<void> saveFontBytes(String url, Uint8List bytes) async =>
      _fonts[url] = bytes;
}
