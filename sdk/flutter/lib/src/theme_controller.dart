import 'dart:async';
import 'dart:developer' as developer;

import 'package:flutter/foundation.dart';

import 'client.dart';
import 'project_config.dart';
import 'theme.dart';
import 'theme_cache.dart';
import 'theme_fonts.dart';

/// Owns the project theme for a widget tree: a [ValueListenable] that
/// starts at [MothTheme.fallback], flips to the disk-cached theme as soon
/// as it loads, and then to the server's current theme once a background
/// `GetProjectConfig` round-trip confirms (or replaces) it —
/// stale-while-revalidate, so the login screen renders branded immediately
/// and picks up admin edits on the next launch.
///
/// [MothApp] creates one automatically; instantiate one only when
/// composing custom signed-out UI from the themed building blocks.
class MothThemeController extends ValueNotifier<MothTheme> {
  MothThemeController({
    required MothClient client,
    MothThemeCache? cache,
    MothFontLoader? fontLoader,
  }) : _client = client,
       _cache = cache ?? defaultThemeCache(client.config.publishableKey),
       _fonts = fontLoader ?? MothFontLoader(),
       super(MothTheme.fallback());

  final MothClient _client;
  final MothThemeCache _cache;
  final MothFontLoader _fonts;

  bool _started = false;
  bool _disposed = false;

  /// Loads the cached theme (rendering it immediately when present), then
  /// refreshes from the server in the background. Idempotent; failures are
  /// swallowed — the current value simply stays.
  Future<void> start() async {
    if (_started) return;
    _started = true;
    MothTheme? cached;
    try {
      cached = await _cache.loadTheme();
    } on Object catch (err) {
      _log('theme cache load failed: $err');
    }
    if (cached != null && !_disposed) {
      value = cached;
      // Fonts usually resolve from the disk cache before the network
      // refresh returns; don't serialize the two.
      unawaited(_applyFont(cached));
    }
    await refresh();
  }

  /// Asks the server for the current theme (echoing the revision already
  /// held, so an unchanged theme is not re-sent), applies and caches a new
  /// revision. Safe to call any time; network failures keep the current
  /// value.
  Future<void> refresh() async {
    final MothProjectConfig config;
    try {
      config = await _client.getProjectConfig(
        knownThemeRevision: value.revisionId,
      );
    } on Object catch (err) {
      _log('theme refresh failed: $err');
      return;
    }
    final theme = config.theme;
    if (theme == null || _disposed) return; // revision matched (or too late)
    try {
      await _cache.saveTheme(theme);
    } on Object catch (err) {
      _log('theme cache save failed: $err');
    }
    // The controller may have been disposed while the write was in flight;
    // notifying a disposed ChangeNotifier asserts in debug builds.
    if (_disposed) return;
    value = theme;
    await _applyFont(theme);
  }

  /// Registers the theme's font and republishes the theme with
  /// [MothTheme.resolvedFontFamily] set, swapping text off the system
  /// font. No-op when the theme carries no font URL or loading fails.
  Future<void> _applyFont(MothTheme theme) async {
    final url = theme.fontUrl;
    if (url == null) return;
    final family = await _fonts.ensure(
      fontFamily: theme.fontFamily,
      url: url,
      cache: _cache,
    );
    if (family == null || _disposed) return;
    // Only stamp the family when the theme it was loaded for is still
    // current (a refresh may have raced past this download).
    if (value.fontUrl == url) {
      value = value.copyWith(resolvedFontFamily: family);
    }
  }

  void _log(String message) {
    assert(() {
      developer.log('moth: $message', name: 'moth', level: 900 /* warning */);
      return true;
    }());
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}
