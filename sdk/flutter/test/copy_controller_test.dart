// Tests for MothCopyController's stale-while-revalidate flow keyed by
// (locale, revision), the x-moth-language header, and offline fallback,
// against the in-process fake server.
import 'dart:async';
import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_auth/src/gen/moth/auth/v1/config.pb.dart' as pb;

import 'fakes.dart';

pb.Copy serverCopy(
  String revision, {
  String locale = 'fr',
  Map<String, String>? messages,
}) => pb.Copy(
  copyRevision: revision,
  locale: locale,
  messages: (messages ?? {'sign_in.title': 'Copy $revision'}).entries,
);

/// A cache fetch time safely outside the default one-hour download-once TTL,
/// so seeded caches still trigger the revalidation round-trip under test.
DateTime stale() => DateTime.now().toUtc().subtract(const Duration(hours: 2));

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

  group('x-moth-language header', () {
    test('attaches the pinned MothConfig.locale as a BCP-47 tag', () async {
      final fr = newClient(moth, locale: const Locale('fr', 'CA'));
      addTearDown(fr.dispose);
      await fr.getProjectConfig();
      expect(moth.config.lastMetadata!['x-moth-language'], 'fr-CA');
    });

    test('falls back to the device locale when none is pinned', () async {
      await client.getProjectConfig();
      expect(
        moth.config.lastMetadata!['x-moth-language'],
        mothLanguageTag(client.currentLocale),
      );
      expect(moth.config.lastMetadata!['x-moth-language'], isNotEmpty);
    });
  });

  group('MothCopyController', () {
    test('first launch: bundled floor immediately, server copy applied '
        'and cached', () async {
      moth.config.response.copy = serverCopy(
        'c1',
        messages: {'sign_in.title': 'FR1'},
      );
      final cache = MothMemoryCopyCache();
      final controller = MothCopyController(
        client: client,
        cache: cache,
        localeOf: () => const Locale('fr'),
      );
      addTearDown(controller.dispose);

      // Bundled French floor before any fetch.
      expect(controller.value.revisionId, '');
      expect(controller.value.value('sign_in.title'), 'Connexion');

      await controller.start();
      expect(controller.value.revisionId, 'c1');
      expect(controller.value.value('sign_in.title'), 'FR1');
      // First call carries no known revision, and the copy got cached.
      expect(moth.config.lastRequest!.knownCopyRevision, '');
      expect((await cache.load(const Locale('fr')))!.copy.revisionId, 'c1');
    });

    test('stale-while-revalidate: cached copy renders first, refresh swaps '
        'to the new revision', () async {
      final cache = MothMemoryCopyCache();
      await cache.save(
        const MothCopy(
          locale: Locale('fr'),
          revisionId: 'c1',
          messages: {'sign_in.title': 'FR1'},
        ),
        fetchedAt: stale(),
      );
      moth.config.response.copy = serverCopy(
        'c2',
        messages: {'sign_in.title': 'FR2'},
      );
      moth.config.gate = Completer<void>();

      final controller = MothCopyController(
        client: client,
        cache: cache,
        localeOf: () => const Locale('fr'),
      );
      addTearDown(controller.dispose);
      final revisions = <String>[];
      final sawCached = Completer<void>();
      controller.addListener(() {
        revisions.add(controller.value.revisionId);
        if (controller.value.revisionId == 'c1' && !sawCached.isCompleted) {
          sawCached.complete();
        }
      });

      final started = controller.start();
      await sawCached.future.timeout(const Duration(seconds: 5));
      // Cached revision rendered with the response still gated.
      expect(revisions, ['c1']);

      moth.config.gate!.complete();
      await started;
      // ...then the server's new revision landed.
      expect(revisions, ['c1', 'c2']);
      expect(controller.value.value('sign_in.title'), 'FR2');
      // The refresh echoed the cached revision, and the cache was replaced.
      expect(moth.config.lastRequest!.knownCopyRevision, 'c1');
      expect((await cache.load(const Locale('fr')))!.copy.revisionId, 'c2');
    });

    test(
      'matching revision: server omits messages, cached copy stays',
      () async {
        final cache = MothMemoryCopyCache();
        await cache.save(
          const MothCopy(
            locale: Locale('fr'),
            revisionId: 'c1',
            messages: {'sign_in.title': 'FR1'},
          ),
          fetchedAt: stale(),
        );
        moth.config.response.copy = serverCopy(
          'c1',
          messages: {'sign_in.title': 'FR1'},
        );

        final controller = MothCopyController(
          client: client,
          cache: cache,
          localeOf: () => const Locale('fr'),
        );
        addTearDown(controller.dispose);
        var notifications = 0;
        controller.addListener(() => notifications++);

        await controller.start();
        expect(moth.config.lastRequest!.knownCopyRevision, 'c1');
        expect(controller.value.revisionId, 'c1');
        expect(controller.value.value('sign_in.title'), 'FR1');
        // One flip (bundled floor -> cached); the confirming refresh (messages
        // omitted) changed nothing.
        expect(notifications, 1);
      },
    );

    test('device locale change reloads the floor and refetches', () async {
      var locale = const Locale('fr');
      moth.config.response.copy = serverCopy(
        'fr1',
        locale: 'fr',
        messages: {'sign_in.title': 'FR'},
      );
      final cache = MothMemoryCopyCache();
      final controller = MothCopyController(
        client: client,
        cache: cache,
        localeOf: () => locale,
      );
      addTearDown(controller.dispose);

      await controller.start();
      expect(controller.value.locale.languageCode, 'fr');
      expect(controller.value.value('sign_in.title'), 'FR');

      // Switch device language to German; the server now negotiates de.
      locale = const Locale('de');
      moth.config.response.copy = serverCopy(
        'de1',
        locale: 'de',
        messages: {'sign_in.title': 'DE'},
      );
      await controller.refresh();
      expect(controller.value.locale.languageCode, 'de');
      expect(controller.value.value('sign_in.title'), 'DE');
      // New locale had no cache, so the refetch sent an empty known revision.
      expect(moth.config.lastRequest!.knownCopyRevision, '');
      expect((await cache.load(const Locale('de')))!.copy.revisionId, 'de1');
    });

    test(
      'caches under the language, so a region-tagged device reloads it',
      () async {
        // The device reports a region-qualified locale (en-US); the server
        // negotiates the language-only catalog locale (en). Cache load (device
        // locale) and save (negotiated locale) must land on the same key.
        moth.config.response.copy = serverCopy(
          'e1',
          locale: 'en',
          messages: {'sign_in.title': 'EN1'},
        );
        final cache = MothMemoryCopyCache();
        final controller = MothCopyController(
          client: client,
          cache: cache,
          localeOf: () => const Locale('en', 'US'),
        );
        addTearDown(controller.dispose);

        await controller.start();
        expect(controller.value.revisionId, 'e1');
        // Loading by the raw device locale (en-US) finds the copy saved under
        // the negotiated en — the round-trip that a language-keyed cache makes
        // work and a full-tag one would miss.
        final reloaded = await cache.load(const Locale('en', 'US'));
        expect(reloaded, isNotNull);
        expect(reloaded!.copy.revisionId, 'e1');
        expect(reloaded.copy.value('sign_in.title'), 'EN1');
      },
    );

    test(
      'a superseded in-flight fetch never clobbers the current locale',
      () async {
        var locale = const Locale('fr');
        // The fr fetch is gated; while it is held, the device switches to de and
        // the de fetch completes first. The stale fr response must be discarded.
        moth.config.response.copy = serverCopy(
          'fr1',
          locale: 'fr',
          messages: {'sign_in.title': 'FR'},
        );
        final frGate = Completer<void>();
        moth.config.gate = frGate;
        final controller = MothCopyController(
          client: client,
          cache: MothMemoryCopyCache(),
          localeOf: () => locale,
        );
        addTearDown(controller.dispose);

        final started = controller.start(); // _fetch(fr) blocks on the gate
        // Let the gated fr request reach the server before switching locale.
        await Future<void>.delayed(const Duration(milliseconds: 50));

        locale = const Locale('de');
        moth.config.response.copy = serverCopy(
          'de1',
          locale: 'de',
          messages: {'sign_in.title': 'DE'},
        );
        moth.config.gate = null; // the de fetch is not gated
        await controller.refresh(); // _load(de) + de fetch complete
        expect(controller.value.locale.languageCode, 'de');
        expect(controller.value.value('sign_in.title'), 'DE');

        // Now release the stale fr fetch: its result is for a superseded locale.
        frGate.complete();
        await started;
        expect(controller.value.locale.languageCode, 'de');
        expect(controller.value.value('sign_in.title'), 'DE');
      },
    );

    test('download-once TTL: a fresh cache serves with zero config '
        'RPCs', () async {
      moth.config.response.copy = serverCopy(
        'c2',
        messages: {'sign_in.title': 'FR2'},
      );
      final cache = MothMemoryCopyCache();
      await cache.save(
        const MothCopy(
          locale: Locale('fr'),
          revisionId: 'c1',
          messages: {'sign_in.title': 'FR1'},
        ),
        fetchedAt: DateTime.now().toUtc(),
      );
      final controller = MothCopyController(
        client: client,
        cache: cache,
        localeOf: () => const Locale('fr'),
      );
      addTearDown(controller.dispose);

      await controller.start();
      // Within the TTL the cached copy is served as-is, no revalidation.
      expect(controller.value.revisionId, 'c1');
      expect(controller.value.value('sign_in.title'), 'FR1');
      expect(moth.config.calls, 0);
    });

    test('download-once TTL: an expired cache revalidates once; the omitted '
        'messages refresh fetched_at so the next launch is quiet '
        'again', () async {
      moth.config.response.copy = serverCopy(
        'c1',
        messages: {'sign_in.title': 'FR1'},
      );
      final cache = MothMemoryCopyCache();
      await cache.save(
        const MothCopy(
          locale: Locale('fr'),
          revisionId: 'c1',
          messages: {'sign_in.title': 'FR1'},
        ),
        fetchedAt: stale(),
      );

      final first = MothCopyController(
        client: client,
        cache: cache,
        localeOf: () => const Locale('fr'),
      );
      addTearDown(first.dispose);
      await first.start();
      // Expired: exactly one cheap revalidation, echoing the revision (the
      // server omits the unchanged messages).
      expect(moth.config.calls, 1);
      expect(moth.config.lastRequest!.knownCopyRevision, 'c1');
      // The omitted-body match re-stamped fetched_at...
      final entry = await cache.load(const Locale('fr'));
      expect(
        DateTime.now().toUtc().difference(entry!.fetchedAt),
        lessThan(const Duration(minutes: 1)),
      );

      // ...so a second launch is quiet: cache only, no config RPC.
      final second = MothCopyController(
        client: client,
        cache: cache,
        localeOf: () => const Locale('fr'),
      );
      addTearDown(second.dispose);
      await second.start();
      expect(moth.config.calls, 1);
      expect(second.value.revisionId, 'c1');
    });

    test('download-once TTL: explicit refresh() with an unchanged locale '
        'forces a fetch', () async {
      moth.config.response.copy = serverCopy(
        'c2',
        messages: {'sign_in.title': 'FR2'},
      );
      final cache = MothMemoryCopyCache();
      await cache.save(
        const MothCopy(
          locale: Locale('fr'),
          revisionId: 'c1',
          messages: {'sign_in.title': 'FR1'},
        ),
        fetchedAt: DateTime.now().toUtc(),
      );
      final controller = MothCopyController(
        client: client,
        cache: cache,
        localeOf: () => const Locale('fr'),
      );
      addTearDown(controller.dispose);

      await controller.start();
      expect(moth.config.calls, 0);
      await controller.refresh();
      expect(moth.config.calls, 1);
      expect(controller.value.revisionId, 'c2');
    });

    test('locale change to a locale with a fresh envelope serves from the '
        'cache, no fetch', () async {
      var locale = const Locale('fr');
      final cache = MothMemoryCopyCache();
      final now = DateTime.now().toUtc();
      await cache.save(
        const MothCopy(
          locale: Locale('fr'),
          revisionId: 'fr1',
          messages: {'sign_in.title': 'FR'},
        ),
        fetchedAt: now,
      );
      await cache.save(
        const MothCopy(
          locale: Locale('de'),
          revisionId: 'de1',
          messages: {'sign_in.title': 'DE'},
        ),
        fetchedAt: now,
      );
      final controller = MothCopyController(
        client: client,
        cache: cache,
        localeOf: () => locale,
      );
      addTearDown(controller.dispose);

      await controller.start();
      expect(moth.config.calls, 0);

      // The new locale's envelope is fresh: the TTL is not bypassed.
      locale = const Locale('de');
      await controller.refresh();
      expect(moth.config.calls, 0);
      expect(controller.value.locale.languageCode, 'de');
      expect(controller.value.value('sign_in.title'), 'DE');
    });

    test('locale change to a locale without a fresh envelope bypasses the '
        'TTL and fetches', () async {
      var locale = const Locale('fr');
      final cache = MothMemoryCopyCache();
      await cache.save(
        const MothCopy(
          locale: Locale('fr'),
          revisionId: 'fr1',
          messages: {'sign_in.title': 'FR'},
        ),
        fetchedAt: DateTime.now().toUtc(),
      );
      final controller = MothCopyController(
        client: client,
        cache: cache,
        localeOf: () => locale,
      );
      addTearDown(controller.dispose);

      await controller.start();
      expect(moth.config.calls, 0); // fr envelope is fresh

      // de has no envelope at all: the switch must fetch despite fr's
      // freshness.
      locale = const Locale('de');
      moth.config.response.copy = serverCopy(
        'de1',
        locale: 'de',
        messages: {'sign_in.title': 'DE'},
      );
      await controller.refresh();
      expect(moth.config.calls, 1);
      expect(controller.value.locale.languageCode, 'de');
      expect(controller.value.value('sign_in.title'), 'DE');
      expect((await cache.load(const Locale('de')))!.copy.revisionId, 'de1');
    });

    test('offline start keeps the bundled localized floor', () async {
      await moth.shutdown(); // nothing listening anymore
      final offline = MothClient(
        MothConfig(
          endpoint: Uri.parse('http://localhost:1'), // connection refused
          publishableKey: 'pk_test',
          locale: const Locale('fr'),
        ),
        tokenStore: InMemoryTokenStore(),
      );
      addTearDown(offline.dispose);
      final controller = MothCopyController(
        client: offline,
        cache: MothMemoryCopyCache(),
      );
      addTearDown(controller.dispose);

      await controller.start();
      // No network, no cache: the bundled French floor still renders.
      expect(controller.value.revisionId, '');
      expect(controller.value.value('sign_in.title'), 'Connexion');

      moth = await startFakeMoth(); // for tearDown symmetry
    });
  });
}
