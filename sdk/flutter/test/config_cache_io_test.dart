// Tests for the file-backed config caches' protobuf format: each cache file
// is a moth.projectconfig.v1.CacheEnvelope whose payload is the raw wire message
// exactly as the server delivered it (moth.auth.v1.Theme /
// moth.billing.v1.Paywall / moth.auth.v1.Copy), and any legacy-JSON or
// corrupt file is a cache miss — deleted best-effort, never a crash.
import 'dart:convert';
import 'dart:io';
import 'dart:ui';

import 'package:fixnum/fixnum.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_auth/src/copy_cache_io.dart';
import 'package:moth_auth/src/gen/moth/auth/v1/config.pb.dart' as authpb;
import 'package:moth_auth/src/gen/moth/billing/v1/billing.pb.dart' as billpb;
import 'package:moth_auth/src/gen/moth/projectconfig/v1/projectconfig.pb.dart'
    as storagepb;
import 'package:moth_auth/src/paywall_cache_io.dart';
import 'package:moth_auth/src/theme_cache_io.dart';

import 'theme_test.dart' show fullProtoTheme;

void main() {
  late Directory base;
  Future<Directory> baseDirectory() async => base;

  setUp(() async {
    base = await Directory.systemTemp.createTemp('moth_cache_test');
  });

  tearDown(() async {
    await base.delete(recursive: true);
  });

  /// The single cache directory `<base>/moth/<key-hash>` the caches write
  /// into (created on first use).
  Directory cacheDir() =>
      Directory('${base.path}/moth').listSync().single as Directory;

  File cacheFile(String name) => File('${cacheDir().path}/$name');

  storagepb.CacheEnvelope readEnvelope(File file) =>
      storagepb.CacheEnvelope.fromBuffer(file.readAsBytesSync());

  final fetchedAt = DateTime.utc(2026, 7, 17, 12, 30);

  group('MothFileThemeCache', () {
    test('persists a CacheEnvelope wrapping the raw Theme wire message and '
        'round-trips it', () async {
      final proto = fullProtoTheme();
      final cache = MothFileThemeCache(
        publishableKey: 'pk_test',
        baseDirectory: baseDirectory,
      );
      await cache.saveTheme(MothTheme.fromProto(proto), fetchedAt: fetchedAt);

      final file = cacheFile('theme.pb');
      expect(file.existsSync(), isTrue);
      final envelope = readEnvelope(file);
      expect(envelope.revision, 'rev-1');
      expect(envelope.locale, isEmpty); // locale-independent payload
      expect(
        envelope.fetchedAtUnixMs.toInt(),
        fetchedAt.millisecondsSinceEpoch,
      );
      // The payload is the wire message exactly as the server delivered it.
      expect(
        authpb.Theme.fromBuffer(envelope.payload).writeToBuffer(),
        proto.writeToBuffer(),
      );

      final loaded = await cache.loadTheme();
      expect(loaded, isNotNull);
      expect(loaded!.fetchedAt, fetchedAt);
      expect(loaded.theme, MothTheme.fromProto(proto));
      expect(loaded.theme.source!.writeToBuffer(), proto.writeToBuffer());
    });

    test('a legacy JSON cache file is a miss and is deleted', () async {
      final cache = MothFileThemeCache(
        publishableKey: 'pk_test',
        baseDirectory: baseDirectory,
      );
      // Prime the directory, then plant the pre-protobuf cache file.
      await cache.saveTheme(
        MothTheme.fromProto(fullProtoTheme()),
        fetchedAt: fetchedAt,
      );
      cacheFile('theme.pb').deleteSync();
      final legacy = cacheFile('theme.json')
        ..writeAsStringSync(
          jsonEncode(MothTheme.fromProto(fullProtoTheme()).toJson()),
        );

      expect(await cache.loadTheme(), isNull);
      expect(legacy.existsSync(), isFalse);
    });

    test('a corrupt cache file is a miss and is deleted', () async {
      final cache = MothFileThemeCache(
        publishableKey: 'pk_test',
        baseDirectory: baseDirectory,
      );
      await cache.saveTheme(
        MothTheme.fromProto(fullProtoTheme()),
        fetchedAt: fetchedAt,
      );
      final file = cacheFile('theme.pb')
        ..writeAsStringSync('{"revisionId": "rev-1", "not": "protobuf"}');

      expect(await cache.loadTheme(), isNull);
      expect(file.existsSync(), isFalse);

      // A subsequent save starts clean again.
      await cache.saveTheme(
        MothTheme.fromProto(fullProtoTheme()),
        fetchedAt: fetchedAt,
      );
      expect((await cache.loadTheme())!.theme.revisionId, 'rev-1');
    });

    test('touchTheme re-stamps fetched_at and keeps the payload', () async {
      final proto = fullProtoTheme();
      final cache = MothFileThemeCache(
        publishableKey: 'pk_test',
        baseDirectory: baseDirectory,
      );
      await cache.saveTheme(MothTheme.fromProto(proto), fetchedAt: fetchedAt);

      final later = fetchedAt.add(const Duration(hours: 3));
      await cache.touchTheme(later);
      final loaded = await cache.loadTheme();
      expect(loaded!.fetchedAt, later);
      expect(loaded.theme.source!.writeToBuffer(), proto.writeToBuffer());
    });
  });

  group('MothFilePaywallCache', () {
    final proto = billpb.Paywall(
      revisionId: 'pw-1',
      headline: 'Go Pro',
      subtitle: 'Everything unlocked.',
      benefits: ['Exports', 'Support'],
      offering: 'default',
      highlightedProductIdentifier: 'yearly',
      layout: billpb.PaywallLayout.PAYWALL_LAYOUT_COMPACT,
      termsUrl: 'https://example.com/terms',
      privacyUrl: 'https://example.com/privacy',
    );

    test('persists a CacheEnvelope wrapping the raw Paywall wire message '
        'and round-trips it', () async {
      final cache = MothFilePaywallCache(
        publishableKey: 'pk_test',
        baseDirectory: baseDirectory,
      );
      await cache.save(MothPaywall.fromProto(proto), fetchedAt: fetchedAt);

      final envelope = readEnvelope(cacheFile('paywall.pb'));
      expect(envelope.revision, 'pw-1');
      expect(envelope.locale, isEmpty); // locale-independent payload
      expect(
        envelope.fetchedAtUnixMs.toInt(),
        fetchedAt.millisecondsSinceEpoch,
      );
      expect(
        billpb.Paywall.fromBuffer(envelope.payload).writeToBuffer(),
        proto.writeToBuffer(),
      );

      final loaded = await cache.load();
      expect(loaded, isNotNull);
      expect(loaded!.fetchedAt, fetchedAt);
      expect(loaded.paywall.revisionId, 'pw-1');
      expect(loaded.paywall.headline, 'Go Pro');
      expect(loaded.paywall.benefits, ['Exports', 'Support']);
      expect(loaded.paywall.layout, MothPaywallLayout.compact);
      expect(loaded.paywall.source!.writeToBuffer(), proto.writeToBuffer());
    });

    test('legacy JSON and corrupt files are misses and are deleted', () async {
      final cache = MothFilePaywallCache(
        publishableKey: 'pk_test',
        baseDirectory: baseDirectory,
      );
      await cache.save(MothPaywall.fromProto(proto), fetchedAt: fetchedAt);

      // Legacy pre-protobuf file next to a corrupted current one.
      final legacy = cacheFile('paywall.json')
        ..writeAsStringSync(jsonEncode(MothPaywall.fromProto(proto).toJson()));
      final file = cacheFile('paywall.pb')..writeAsStringSync('not protobuf');

      expect(await cache.load(), isNull);
      expect(legacy.existsSync(), isFalse);
      expect(file.existsSync(), isFalse);
    });

    test('touch re-stamps fetched_at and keeps the payload', () async {
      final cache = MothFilePaywallCache(
        publishableKey: 'pk_test',
        baseDirectory: baseDirectory,
      );
      await cache.save(MothPaywall.fromProto(proto), fetchedAt: fetchedAt);

      final later = fetchedAt.add(const Duration(hours: 3));
      await cache.touch(later);
      final loaded = await cache.load();
      expect(loaded!.fetchedAt, later);
      expect(loaded.paywall.headline, 'Go Pro');
    });
  });

  group('MothFileCopyCache', () {
    final proto = authpb.Copy(
      copyRevision: 'c1',
      locale: 'fr',
      messages: {
        'sign_in.title': 'Connexion',
        'sign_up.title': 'Créer',
      }.entries,
    );

    MothCopy copyModel() => MothCopy(
      locale: const Locale('fr'),
      revisionId: 'c1',
      messages: {'sign_in.title': 'Connexion', 'sign_up.title': 'Créer'},
      source: proto,
    );

    test('persists a CacheEnvelope wrapping the raw Copy wire message, '
        'locale stamped, and round-trips it', () async {
      final cache = MothFileCopyCache(
        publishableKey: 'pk_test',
        baseDirectory: baseDirectory,
      );
      await cache.save(copyModel(), fetchedAt: fetchedAt);

      final file = cacheDir()
          .listSync()
          .whereType<File>()
          .where(
            (f) =>
                f.path.endsWith('.pb') &&
                f.uri.pathSegments.last.startsWith('copy_'),
          )
          .single;
      final envelope = readEnvelope(file);
      expect(envelope.revision, 'c1');
      expect(envelope.locale, 'fr'); // locale-keyed payload
      expect(
        envelope.fetchedAtUnixMs.toInt(),
        fetchedAt.millisecondsSinceEpoch,
      );
      expect(
        authpb.Copy.fromBuffer(envelope.payload).writeToBuffer(),
        proto.writeToBuffer(),
      );

      // Load by a region-tagged device locale: same language, same file.
      final loaded = await cache.load(const Locale('fr', 'CA'));
      expect(loaded, isNotNull);
      expect(loaded!.fetchedAt, fetchedAt);
      expect(loaded.copy.revisionId, 'c1');
      expect(loaded.copy.locale.languageCode, 'fr');
      expect(loaded.copy.value('sign_in.title'), 'Connexion');
      expect(loaded.copy.source!.writeToBuffer(), proto.writeToBuffer());
    });

    test('legacy JSON and corrupt files are misses and are deleted', () async {
      final cache = MothFileCopyCache(
        publishableKey: 'pk_test',
        baseDirectory: baseDirectory,
      );
      await cache.save(copyModel(), fetchedAt: fetchedAt);
      final pbFile = cacheDir().listSync().whereType<File>().single;
      final hash = pbFile.uri.pathSegments.last
          .replaceFirst('copy_', '')
          .replaceFirst('.pb', '');

      // Legacy pre-protobuf file for the same language next to a corrupted
      // current one.
      final legacy = cacheFile('copy_$hash.json')
        ..writeAsStringSync(jsonEncode(copyModel().toJson()));
      pbFile.writeAsStringSync(jsonEncode(copyModel().toJson()));

      expect(await cache.load(const Locale('fr')), isNull);
      expect(legacy.existsSync(), isFalse);
      expect(pbFile.existsSync(), isFalse);

      // The next save/load round-trips cleanly again.
      await cache.save(copyModel(), fetchedAt: fetchedAt);
      expect((await cache.load(const Locale('fr')))!.copy.revisionId, 'c1');
    });

    test('touch re-stamps fetched_at for the locale', () async {
      final cache = MothFileCopyCache(
        publishableKey: 'pk_test',
        baseDirectory: baseDirectory,
      );
      await cache.save(copyModel(), fetchedAt: fetchedAt);

      final later = fetchedAt.add(const Duration(hours: 3));
      await cache.touch(const Locale('fr'), later);
      final loaded = await cache.load(const Locale('fr'));
      expect(loaded!.fetchedAt, later);
      expect(loaded.copy.value('sign_up.title'), 'Créer');

      // Touching a locale with no envelope is a no-op, not an error.
      await cache.touch(const Locale('de'), later);
      expect(await cache.load(const Locale('de')), isNull);
    });
  });

  test('an envelope with a wildly corrupt payload is a miss, not a '
      'crash', () async {
    final cache = MothFileThemeCache(
      publishableKey: 'pk_test',
      baseDirectory: baseDirectory,
    );
    await cache.saveTheme(
      MothTheme.fromProto(fullProtoTheme()),
      fetchedAt: fetchedAt,
    );
    // A valid envelope whose payload is not a Theme message.
    final envelope = storagepb.CacheEnvelope(
      payload: [0xFF, 0xFF, 0xFF, 0xFF],
      revision: 'rev-x',
      fetchedAtUnixMs: Int64(fetchedAt.millisecondsSinceEpoch),
    );
    cacheFile('theme.pb').writeAsBytesSync(envelope.writeToBuffer());

    expect(await cache.loadTheme(), isNull);
    expect(cacheFile('theme.pb').existsSync(), isFalse);
  });
}
