// Tests for MothThemeController's stale-while-revalidate flow and
// MothFontLoader's byte caching, against the in-process fake server.
import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_auth/src/gen/moth/auth/v1/config.pb.dart' as pb;

import 'fakes.dart';
import 'theme_test.dart' show fullProtoTheme;

/// A cache whose saveTheme blocks on [gate], signalling [saving] on entry —
/// pins MothThemeController.refresh() mid-write for dispose-race tests.
class _GatedSaveCache extends MothMemoryThemeCache {
  final saving = Completer<void>();
  final gate = Completer<void>();

  @override
  Future<void> saveTheme(MothTheme theme) async {
    if (!saving.isCompleted) saving.complete();
    await gate.future;
    await super.saveTheme(theme);
  }
}

void main() {
  late FakeMoth moth;
  late MothClient client;

  setUp(() async {
    moth = await startFakeMoth();
    client = newClient(moth);
  });

  tearDown(() async {
    await client.dispose();
    await moth.shutdown();
  });

  pb.Theme serverTheme(String revision, {String primary = '#0B6E99'}) =>
      fullProtoTheme()
        ..revisionId = revision
        ..colors.primary = primary
        ..clearFontUrl(); // no font downloads in these tests

  test('first launch: fallback immediately, server theme applied and '
      'cached', () async {
    moth.config.response.theme = serverTheme('rev-1');
    final cache = MothMemoryThemeCache();
    final controller = MothThemeController(client: client, cache: cache);
    addTearDown(controller.dispose);

    expect(controller.value, MothTheme.fallback());
    await controller.start();
    expect(controller.value.revisionId, 'rev-1');
    expect(mothHexColor(controller.value.colors.primary), '#0B6E99');
    // First call carries no known revision, and the theme got cached.
    expect(moth.config.lastRequest!.knownThemeRevision, '');
    expect((await cache.loadTheme())!.revisionId, 'rev-1');
  });

  test('stale-while-revalidate: cached theme renders first, refresh swaps '
      'to the new revision', () async {
    final cache = MothMemoryThemeCache();
    await cache.saveTheme(
      MothTheme.fromProto(serverTheme('rev-1', primary: '#C8102E')),
    );
    moth.config.response.theme = serverTheme('rev-2', primary: '#2E7D32');
    // Hold the server response: the cached theme must be published while
    // the network round-trip is still in flight — a controller that blocks
    // rendering on the RPC would preserve the notification order but stall
    // on the fallback theme for the whole RTT.
    moth.config.gate = Completer<void>();

    final controller = MothThemeController(client: client, cache: cache);
    addTearDown(controller.dispose);
    final revisions = <String>[];
    final sawCached = Completer<void>();
    controller.addListener(() {
      revisions.add(controller.value.revisionId);
      if (controller.value.revisionId == 'rev-1' && !sawCached.isCompleted) {
        sawCached.complete();
      }
    });

    final started = controller.start();
    await sawCached.future.timeout(const Duration(seconds: 5));
    // Cached revision rendered with the response still gated.
    expect(revisions, ['rev-1']);

    moth.config.gate!.complete();
    await started;
    // ...then the server's new revision landed.
    expect(revisions, ['rev-1', 'rev-2']);
    expect(mothHexColor(controller.value.colors.primary), '#2E7D32');
    // The refresh echoed the cached revision.
    expect(moth.config.lastRequest!.knownThemeRevision, 'rev-1');
    // The new revision replaced the cache.
    expect((await cache.loadTheme())!.revisionId, 'rev-2');
  });

  test(
    'matching revision: server omits the theme, cached copy stays',
    () async {
      final cache = MothMemoryThemeCache();
      final theme = MothTheme.fromProto(serverTheme('rev-1'));
      await cache.saveTheme(theme);
      moth.config.response.theme = serverTheme('rev-1');

      final controller = MothThemeController(client: client, cache: cache);
      addTearDown(controller.dispose);
      var notifications = 0;
      controller.addListener(() => notifications++);

      await controller.start();
      expect(moth.config.lastRequest!.knownThemeRevision, 'rev-1');
      expect(controller.value, theme);
      // One flip (fallback -> cached); the confirming refresh changed
      // nothing.
      expect(notifications, 1);
    },
  );

  test('dispose while the cache write is in flight stays silent', () async {
    // refresh() suspends between receiving the theme and publishing it
    // (the disk write); disposing in that window must not notify a
    // disposed ChangeNotifier, which asserts in debug builds.
    moth.config.response.theme = serverTheme('rev-1');
    final cache = _GatedSaveCache();
    final controller = MothThemeController(client: client, cache: cache);

    final refreshing = controller.refresh();
    await cache.saving.future.timeout(const Duration(seconds: 5));
    controller.dispose();
    cache.gate.complete();
    await refreshing; // must complete without the dispose assertion
    expect(controller.value.revisionId, isNot('rev-1'));
  });

  test('offline start keeps the cached theme', () async {
    final cache = MothMemoryThemeCache();
    await cache.saveTheme(MothTheme.fromProto(serverTheme('rev-1')));
    await moth.shutdown(); // nothing listening anymore

    final offline = MothClient(
      MothConfig(
        endpoint: Uri.parse('http://localhost:1'), // connection refused
        publishableKey: 'pk_test',
      ),
      tokenStore: InMemoryTokenStore(),
    );
    addTearDown(offline.dispose);
    final controller = MothThemeController(client: offline, cache: cache);
    addTearDown(controller.dispose);

    await controller.start();
    expect(controller.value.revisionId, 'rev-1');

    moth = await startFakeMoth(); // for tearDown symmetry
  });

  group('MothFontLoader', () {
    test('caches downloaded bytes and skips the re-download', () async {
      final cache = MothMemoryThemeCache();
      var fetches = 0;
      // Deliberately not a real font: registration fails (the system font
      // stays), but the fetch/cache path is what's under test.
      final loader = MothFontLoader(
        fetch: (url) async {
          fetches++;
          return Uint8List.fromList([1, 2, 3]);
        },
      );

      const url = 'https://auth.example.com/assets/fonts/inter.ttf';
      await loader.ensure(fontFamily: 'Inter', url: url, cache: cache);
      expect(fetches, 1);
      expect(await cache.loadFontBytes(url), isNotNull);

      // Second attempt reads the cache, no network.
      await loader.ensure(fontFamily: 'Inter', url: url, cache: cache);
      expect(fetches, 1);
    });

    test('failed fetch falls back to the system font (returns null)', () async {
      final loader = MothFontLoader(fetch: (url) async => null);
      final family = await loader.ensure(
        fontFamily: 'Inter',
        // Distinct URL: registration is process-wide, keyed by URL.
        url: 'https://auth.example.com/assets/fonts/unreachable.ttf',
        cache: MothMemoryThemeCache(),
      );
      expect(family, isNull);
    });
  });
}
