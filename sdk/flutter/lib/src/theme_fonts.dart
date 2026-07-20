import 'dart:typed_data';

import 'package:flutter/services.dart' show FontLoader;
import 'package:http/http.dart' as http;

import 'theme_cache.dart';

/// Downloads a theme's font file and registers it with the engine via
/// [FontLoader], caching the bytes on disk so later launches skip the
/// download. Registration is process-wide and deduplicated by URL.
class MothFontLoader {
  MothFontLoader({Future<Uint8List?> Function(Uri url)? fetch})
    : _fetch = fetch ?? _httpFetch;

  final Future<Uint8List?> Function(Uri url) _fetch;

  /// URLs already registered (or in flight) in this process, with the
  /// family name they resolve to; failed loads are evicted so a later
  /// theme refresh can retry.
  static final _registered = <String, Future<String?>>{};

  /// The engine family name a theme font registers under. Prefixed so a
  /// server font can never shadow a family the app bundles itself.
  static String familyNameFor(String fontFamily) => 'moth $fontFamily';

  /// Ensures the font at [url] is registered under
  /// `familyNameFor(fontFamily)`, downloading (or reading [cache]) as
  /// needed. Returns the registered family name, or null when the font
  /// could not be loaded — callers keep the system font.
  Future<String?> ensure({
    required String fontFamily,
    required String url,
    required MothThemeCache cache,
  }) async {
    final pending = _registered[url];
    if (pending != null) return pending;
    final load = _load(fontFamily, url, cache);
    _registered[url] = load;
    final family = await load;
    if (family == null) _registered.remove(url);
    return family;
  }

  Future<String?> _load(
    String fontFamily,
    String url,
    MothThemeCache cache,
  ) async {
    try {
      final bytes = await _bytesFor(url, cache);
      if (bytes == null) return null;
      final family = familyNameFor(fontFamily);
      await (FontLoader(
        family,
      )..addFont(Future.value(ByteData.sublistView(bytes)))).load();
      return family;
    } on Object {
      // Unreachable server, corrupt cache entry, unparseable font — the
      // system font stays in place; the next refresh retries.
      return null;
    }
  }

  Future<Uint8List?> _bytesFor(String url, MothThemeCache cache) async {
    try {
      final cached = await cache.loadFontBytes(url);
      if (cached != null) return cached;
    } on Object {
      // Broken cache reads fall through to a fresh download.
    }
    final uri = Uri.tryParse(url);
    if (uri == null || !uri.isAbsolute) return null;
    final bytes = await _fetch(uri);
    if (bytes == null) return null;
    try {
      await cache.saveFontBytes(url, bytes);
    } on Object {
      // Best effort — the font still registers for this session.
    }
    return bytes;
  }

  static Future<Uint8List?> _httpFetch(Uri url) async {
    final resp = await http.get(url);
    return resp.statusCode == 200 ? resp.bodyBytes : null;
  }
}
