// Golden tests for MothLoginScreen: 3 reference themes in English + the
// default theme in French, each × light/dark.
//
// Tagged `golden` and excluded from the default `flutter test` run and CI:
// rasterization differs across platforms/engine builds, so the committed
// images are only stable on the machine flavor that generated them. Run
// (or regenerate with UPDATE=1) via `make sdk-goldens`.
@Tags(['golden'])
library;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_auth/src/gen/moth/auth/v1/config.pb.dart' as pbconfig;

import '../fakes.dart';
import '../widget_helpers.dart';

/// The three reference themes of the milestone's acceptance criteria: the
/// built-in default, a cool corporate look and a warm rounded one with
/// dark-color overrides.
final referenceThemes = <String, MothTheme>{
  'default': MothTheme.fallback(),
  'ocean': MothTheme.fromProto(
    pbconfig.Theme(
      colors: pbconfig.ThemeColors(
        primary: '#0B6E99',
        onPrimary: '#FFFFFF',
        background: '#F7FAFC',
        onBackground: '#102A43',
        surface: '#FFFFFF',
        onSurface: '#102A43',
        error: '#B00020',
        onError: '#FFFFFF',
      ),
      fontScale: 0.9,
      spacingUnit: 6,
      cornerRadius: 2,
      termsUrl: 'https://example.com/terms',
      privacyUrl: 'https://example.com/privacy',
    ),
  ),
  'sunset': MothTheme.fromProto(
    pbconfig.Theme(
      colors: pbconfig.ThemeColors(
        primary: '#C8481F',
        onPrimary: '#FFFFFF',
        background: '#FFF8F2',
        onBackground: '#33201A',
        surface: '#FFFFFF',
        onSurface: '#33201A',
        error: '#8C1D18',
        onError: '#FFFFFF',
      ),
      darkColors: pbconfig.ThemeColors(
        primary: '#FFB59D',
        onPrimary: '#3B0900',
        background: '#1F1410',
        onBackground: '#F5E4DD',
        surface: '#2A1B15',
        onSurface: '#F5E4DD',
        error: '#FFB4AB',
        onError: '#3B0900',
      ),
      fontScale: 1.1,
      spacingUnit: 10,
      cornerRadius: 28,
      privacyUrl: 'https://example.com/privacy',
    ),
  ),
};

void main() {
  late FakeMoth moth;
  late MothClient client;

  // (image-stem, theme, device locale). English covers all three themes; the
  // "other language" (French) covers the default theme, per the milestone.
  final cases = <(String, MothTheme, Locale)>[
    for (final MapEntry(key: name, value: theme) in referenceThemes.entries)
      (name, theme, const Locale('en')),
    ('default_fr', referenceThemes['default']!, const Locale('fr')),
  ];

  for (final (name, theme, locale) in cases) {
    for (final brightness in Brightness.values) {
      final mode = brightness == Brightness.dark ? 'dark' : 'light';
      testWidgets('MothLoginScreen $name $mode', (tester) async {
        tester.view.physicalSize = const Size(390, 844);
        tester.view.devicePixelRatio = 1.0;
        addTearDown(tester.view.reset);
        tester.platformDispatcher.platformBrightnessTestValue = brightness;
        addTearDown(tester.platformDispatcher.clearPlatformBrightnessTestValue);

        moth = await runReal(tester, startFakeMoth);
        // Both providers on, so the goldens cover the full surface.
        moth.config.response = pbconfig.GetProjectConfigResponse(
          google: pbconfig.GoogleConfig(enabled: true),
          apple: pbconfig.AppleConfig(enabled: true),
          passwordMinLength: 8,
          signUpOpen: true,
        );
        // The device locale drives the bundled copy; appName fills {app}.
        client = newClient(moth, locale: locale, appName: 'Aurora');

        await tester.pumpWidget(
          MaterialApp(
            debugShowCheckedModeBanner: false,
            localizationsDelegates: mothLocalizationsDelegates,
            supportedLocales: mothSupportedLocales,
            locale: locale,
            home: MothLoginScreen(client: client, theme: theme),
          ),
        );
        await pumpUntilFound(
          tester,
          find.byKey(MothLoginScreen.submitButtonKey),
        );
        await tester.pump();

        await expectLater(
          find.byType(MothLoginScreen),
          matchesGoldenFile('goldens/login_${name}_$mode.png'),
        );

        await settle(tester, client.dispose());
        await settle(tester, moth.shutdown());
      });
    }
  }
}
