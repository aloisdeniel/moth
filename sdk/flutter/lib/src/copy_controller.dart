import 'dart:async';
import 'dart:developer' as developer;
import 'dart:ui';

import 'package:flutter/foundation.dart';

import 'client.dart';
import 'copy.dart';
import 'copy_cache.dart';
import 'locale.dart';
import 'project_config.dart';

/// Owns the localized copy for a widget tree: a [ValueListenable] that starts
/// at the bundled floor for the current locale ([MothCopy.bundled]), flips to
/// the disk-cached copy for that locale as soon as it loads, then to the
/// server's copy once a background `GetProjectConfig` round-trip confirms (or
/// replaces) it — stale-while-revalidate, keyed by `(locale, revision)`, so
/// the login and paywall screens render in the right language instantly and
/// pick up admin copy edits on the next launch.
///
/// The device language drives it: when the locale changes, call [refresh] and
/// the controller reloads that locale's cached floor and refetches. Mirrors
/// [MothThemeController]; [MothApp] creates one automatically.
class MothCopyController extends ValueNotifier<MothCopy> {
  MothCopyController({
    required MothClient client,
    MothCopyCache? cache,
    Locale Function()? localeOf,
  }) : _client = client,
       _cache = cache ?? defaultCopyCache(client.config.publishableKey),
       _localeOf = localeOf ?? (() => client.currentLocale),
       _requested = (localeOf ?? (() => client.currentLocale))(),
       super(MothCopy.bundled((localeOf ?? (() => client.currentLocale))()));

  final MothClient _client;
  final MothCopyCache _cache;
  final Locale Function() _localeOf;

  /// The device/override locale the current value was fetched for; a change
  /// here (device language switched) triggers a reload on [refresh].
  Locale _requested;

  bool _started = false;
  bool _disposed = false;

  /// Bumped every time a fetch is initiated; a fetch only applies its result
  /// when its captured generation is still current. Guards against an
  /// in-flight fetch for a superseded locale (device language switched
  /// mid-request) clobbering the current locale's copy — last request wins.
  int _generation = 0;

  /// Loads the cached copy for the current locale (rendering it immediately
  /// when present), then — unless that locale's cache entry is still younger
  /// than [MothConfig.configCacheTtl] (download-once: a fresh cache means
  /// zero config RPCs on launch) — refreshes from the server in the
  /// background. Idempotent; failures are swallowed — the bundled floor
  /// simply stays.
  Future<void> start() async {
    if (_started) return;
    _started = true;
    await _load(_localeOf());
  }

  /// Refreshes from the server, first switching to the current device locale
  /// (reloading its cached floor) when it changed since the last fetch. Safe
  /// to call any time — on launch, and from a locale-change observer. An
  /// unchanged locale always hits the server (an explicit refresh ignores
  /// the download-once TTL); a locale change fetches only when the new
  /// locale has no fresh cache entry. Network failures keep the current
  /// value.
  Future<void> refresh() async {
    final locale = _localeOf();
    if (mothLanguageTag(locale) != mothLanguageTag(_requested)) {
      await _load(locale);
    } else {
      await _fetch();
    }
  }

  /// Resets the floor to [locale] (its disk-cached copy, else the bundled
  /// defaults) and refetches from the server for it — unless the cached
  /// entry is still within the download-once TTL, in which case it is served
  /// as-is with no round-trip.
  Future<void> _load(Locale locale) async {
    _requested = locale;
    MothCachedCopy? cached;
    try {
      cached = await _cache.load(locale);
    } on Object catch (err) {
      _log('copy cache load failed: $err');
    }
    if (_disposed) return;
    value = cached?.copy ?? MothCopy.bundled(locale);
    // Download-once: a locale whose envelope is still fresh serves from the
    // cache alone; the TTL is bypassed only when the new locale has no fresh
    // envelope. Superseding the generation discards any in-flight fetch for
    // a previous locale, which must not clobber this cached value.
    if (cached != null && _isFresh(cached.fetchedAt)) {
      _generation++;
      return;
    }
    await _fetch();
  }

  bool _isFresh(DateTime fetchedAt) =>
      DateTime.now().toUtc().difference(fetchedAt.toUtc()) <
      _client.config.configCacheTtl;

  /// Asks the server for the copy (echoing the revision already held, so an
  /// unchanged copy is not re-sent), applies and caches a new revision.
  Future<void> _fetch() async {
    final generation = ++_generation;
    final MothProjectConfig config;
    try {
      config = await _client.getProjectConfig(
        knownCopyRevision: value.revisionId,
      );
    } on Object catch (err) {
      _log('copy refresh failed: $err');
      return;
    }
    final update = config.copy;
    // A null update means the server predates localized copy — nothing to
    // apply or cache.
    if (update == null) return;
    final now = DateTime.now().toUtc();
    if (update.messages == null) {
      // Revision matched (messages omitted): the negotiated locale's cached
      // payload is confirmed current — restart its download-once TTL window.
      try {
        await _cache.touch(update.locale, now);
      } on Object catch (err) {
        _log('copy cache touch failed: $err');
      }
      return;
    }
    // Was disposed, or a newer fetch superseded this one (locale switched
    // mid-request) — keep the current value.
    if (_disposed || generation != _generation) return;
    final copy = MothCopy(
      locale: update.locale,
      revisionId: update.revisionId,
      messages: update.messages!,
      source: update.source,
    );
    try {
      await _cache.save(copy, fetchedAt: now);
    } on Object catch (err) {
      _log('copy cache save failed: $err');
    }
    // The controller may have been disposed or superseded while the write was
    // in flight; notifying a disposed ChangeNotifier asserts in debug builds,
    // and a superseded fetch must not overwrite the current locale's copy.
    if (_disposed || generation != _generation) return;
    value = copy;
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
