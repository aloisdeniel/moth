// Tests for the baked-in project config seed (generated_config.dart +
// bootstrap.dart): a server-generated package decodes its base64 config/paywall
// once into a MothBootstrap, which seeds the login screen and the theme/copy/
// paywall caches so the first frame renders correctly with no network round-trip.
import 'dart:convert';
import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_auth/src/bootstrap.dart';
import 'package:moth_auth/src/gen/moth/auth/v1/config.pb.dart' as authpb;
import 'package:moth_auth/src/gen/moth/billing/v1/billing.pb.dart' as billpb;

import 'theme_test.dart' show fullProtoTheme;

void main() {
  // A representative server-baked config: enabled providers, a policy, a theme
  // and default-locale copy — exactly what the server marshals into the seed.
  authpb.GetProjectConfigResponse configResponse() =>
      authpb.GetProjectConfigResponse(
        google: authpb.GoogleConfig(enabled: true, webClientId: 'web.apps'),
        apple: authpb.AppleConfig(enabled: true),
        passwordMinLength: 10,
        signUpOpen: true,
        push: authpb.PushConfig(enabled: false),
        theme: fullProtoTheme(),
        copy: authpb.Copy(
          locale: 'en',
          copyRevision: 'en|rev1',
          messages: {'sign_in.title': 'Welcome'}.entries,
        ),
      );

  billpb.Paywall paywall() =>
      billpb.Paywall(revisionId: 'pw1', headline: 'Go Pro');

  String cfgB64() => base64.encode(configResponse().writeToBuffer());
  String pwB64() => base64.encode(paywall().writeToBuffer());

  group('MothBootstrap.decode', () {
    test('null for the canonical package (empty seed)', () {
      expect(MothBootstrap.decode('', ''), isNull);
    });

    test('null (never throws) on malformed base64', () {
      expect(MothBootstrap.decode('!! not base64 !!', ''), isNull);
    });

    test('decodes providers and password policy for the login screen', () {
      final b = MothBootstrap.decode(cfgB64(), pwB64())!;
      final c = b.projectConfig;
      expect(c.google.enabled, isTrue);
      expect(c.google.webClientId, 'web.apps');
      expect(c.apple.enabled, isTrue);
      expect(c.passwordMinLength, 10);
      expect(c.signUpOpen, isTrue);
      expect(c.theme, isNotNull);
    });

    test('seeds the theme cache with the baked theme', () {
      final b = MothBootstrap.decode(cfgB64(), pwB64())!;
      final seeded = b.seededTheme!;
      expect(seeded.theme.revisionId, fullProtoTheme().revisionId);
      // Stamped at the epoch so it always revalidates (stale-while-revalidate).
      expect(seeded.fetchedAt.millisecondsSinceEpoch, 0);
    });

    test('seeds copy only for the baked locale', () {
      final b = MothBootstrap.decode(cfgB64(), pwB64())!;
      final en = b.seededCopy(const Locale('en'))!;
      expect(en.copy.revisionId, 'en|rev1');
      expect(en.copy.messages['sign_in.title'], 'Welcome');
      // A different language is a miss (fetched on first use).
      expect(b.seededCopy(const Locale('fr')), isNull);
    });

    test('seeds the paywall cache when a paywall is baked in', () {
      final b = MothBootstrap.decode(cfgB64(), pwB64())!;
      expect(b.seededPaywall!.paywall.revisionId, 'pw1');
    });

    test('no paywall seed when the paywall blob is absent', () {
      final b = MothBootstrap.decode(cfgB64(), '')!;
      expect(b.seededPaywall, isNull);
    });
  });

  group('MothConfig.generated', () {
    test('asserts in a non-generated build (empty placeholders)', () {
      // This test suite runs against the canonical package, whose placeholders
      // are empty — MothConfig.generated() must refuse rather than build a
      // config with an empty endpoint.
      expect(MothConfig.generated, throwsA(isA<AssertionError>()));
    });
  });
}
