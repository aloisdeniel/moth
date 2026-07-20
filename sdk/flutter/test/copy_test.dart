// Unit tests for locale resolution, MothCopy resolution/interpolation and the
// bundled fallback catalog — no server needed.
import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';

void main() {
  group('locale', () {
    test('BCP-47 tag for language and language-region', () {
      expect(mothLanguageTag(const Locale('fr')), 'fr');
      expect(mothLanguageTag(const Locale('fr', 'CA')), 'fr-CA');
    });

    test('parses a tag back to a Locale, tolerating both separators', () {
      expect(mothLocaleFromTag('fr'), const Locale('fr'));
      expect(mothLocaleFromTag('fr-CA'), const Locale('fr', 'CA'));
      expect(mothLocaleFromTag('fr_CA'), const Locale('fr', 'CA'));
      expect(mothLocaleFromTag(''), const Locale('en'));
      expect(
        mothLocaleFromTag('zh-Hant-TW'),
        Locale.fromSubtags(
          languageCode: 'zh',
          scriptCode: 'Hant',
          countryCode: 'TW',
        ),
      );
    });

    test('MothConfig.locale overrides the device locale on the client', () {
      final overridden = MothClient(
        MothConfig(
          endpoint: _dummy,
          publishableKey: 'pk',
          locale: const Locale('fr', 'CA'),
        ),
        tokenStore: InMemoryTokenStore(),
      );
      addTearDown(overridden.dispose);
      expect(mothLanguageTag(overridden.currentLocale), 'fr-CA');

      final device = MothClient(
        MothConfig(endpoint: _dummy, publishableKey: 'pk'),
        tokenStore: InMemoryTokenStore(),
      );
      addTearDown(device.dispose);
      // Falls back to the engine's device locale (non-empty in the test host).
      expect(mothLanguageTag(device.currentLocale), isNotEmpty);
      expect(device.currentLocale, mothDeviceLocale());
    });
  });

  group('MothCopy.value', () {
    test('server override wins, then bundled, then English', () {
      const copy = MothCopy(
        locale: Locale('fr'),
        revisionId: 'r1',
        messages: {'sign_in.title': 'Connexion perso'},
      );
      // Server-delivered override.
      expect(copy.value('sign_in.title'), 'Connexion perso');
      // Key not in the override → bundled French default.
      expect(copy.value('sign_in.submit'), 'Se connecter');
    });

    test('a non-bundled locale falls back to English per key', () {
      const copy = MothCopy(locale: Locale('xx'), revisionId: '', messages: {});
      expect(copy.value('sign_in.title'), 'Sign in');
      expect(copy.value('paywall.cta'), 'Continue');
    });

    test('interpolates {placeholders} from vars, leaving unknown ones', () {
      final copy = MothCopy.bundled(const Locale('fr'));
      expect(
        copy.value('sign_in.subtitle', vars: {'app': 'Acme'}),
        'Bon retour sur Acme.',
      );
      // No vars: the placeholder is left intact.
      expect(copy.value('sign_in.subtitle'), 'Bon retour sur {app}.');
    });

    test('an unknown key returns the key itself', () {
      final copy = MothCopy.bundled(const Locale('en'));
      expect(copy.value('nope.missing'), 'nope.missing');
    });

    test(
      'the bundled floor renders localized copy with no server messages',
      () {
        final copy = MothCopy.bundled(const Locale('de'));
        expect(copy.revisionId, '');
        expect(copy.value('sign_in.title'), 'Anmelden');
        expect(copy.value('paywall.cta'), 'Weiter');
      },
    );
  });

  group('bundled catalog', () {
    test('every SDK-screen key resolves non-empty in every bundled locale', () {
      final keys = bundledCopy(const Locale('en')).keys;
      expect(keys, isNotEmpty);
      for (final code in mothBundledLocales) {
        final map = bundledCopy(Locale(code));
        for (final key in keys) {
          expect(map[key], isNotNull, reason: '$code missing $key');
          expect(map[key], isNotEmpty, reason: '$code empty for $key');
        }
      }
    });

    test('non-English locales carry real translations, not English', () {
      expect(bundledCopy(const Locale('fr'))['sign_in.title'], 'Connexion');
      expect(bundledCopy(const Locale('ja'))['paywall.cta'], '続ける');
    });

    test('an unbundled locale returns the full English map', () {
      final map = bundledCopy(const Locale('xx'));
      expect(map['sign_in.title'], 'Sign in');
      expect(map['sign_up.legal'], startsWith('By continuing'));
    });

    test('covers the SDK screens plus the shared error group', () {
      final screens = bundledCopy(
        const Locale('en'),
      ).keys.map((k) => k.split('.').first).toSet();
      expect(screens, {
        'sign_in',
        'sign_up',
        'password_reset',
        'paywall',
        'error',
      });
    });
  });
}

final Uri _dummy = Uri.parse('http://localhost:1');
