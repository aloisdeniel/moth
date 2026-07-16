// Widget tests for the themed login surface: MothTheme consumption in
// MothLoginScreen and its building blocks, logo and legal links, and the
// MothApp stale-while-revalidate wiring.
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';

import 'fakes.dart';
import 'theme_test.dart' show fullProtoTheme;
import 'widget_helpers.dart';

void main() {
  late FakeMoth moth;
  late MothClient client;

  Future<void> start(WidgetTester tester, {TokenStore? store}) async {
    moth = await runReal(tester, startFakeMoth);
    client = newClient(moth, store: store);
  }

  Future<void> stop(WidgetTester tester) async {
    await settle(tester, client.dispose());
    await settle(tester, moth.shutdown());
  }

  final referenceTheme = MothTheme.fromProto(
    fullProtoTheme()
      ..clearFontUrl() // no font downloads in widget tests
      ..clearLogoLightUrl()
      ..clearLogoDarkUrl(),
  );

  ThemeData themeAt(WidgetTester tester, Key key) =>
      Theme.of(tester.element(find.byKey(key)));

  group('MothLoginScreen theming', () {
    Future<void> pumpLoginScreen(
      WidgetTester tester, {
      MothTheme? theme,
    }) async {
      await tester.pumpWidget(
        MaterialApp(
          home: MothLoginScreen(client: client, theme: theme),
        ),
      );
      await pumpUntilFound(tester, find.text('Welcome'));
    }

    testWidgets('explicit theme drives colors, radius and spacing', (
      tester,
    ) async {
      await start(tester);
      await pumpLoginScreen(tester, theme: referenceTheme);

      final theme = themeAt(tester, MothLoginScreen.submitButtonKey);
      expect(theme.colorScheme.primary, referenceTheme.colors.primary);
      expect(theme.colorScheme.error, referenceTheme.colors.error);
      expect(theme.scaffoldBackgroundColor, referenceTheme.colors.background);
      // The button theme in effect at the submit button carries the theme
      // radius.
      expect(
        theme.filledButtonTheme.style!.shape!.resolve(const {}),
        RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(referenceTheme.cornerRadius),
        ),
      );
      // Text styles carry the palette color and the scaled sizes — a
      // sub-widget reverting to hardcoded styles or a broken text-theme
      // merge shows up here, not only in the (local-only) goldens.
      expect(
        theme.textTheme.headlineMedium!.color,
        referenceTheme.colors.onSurface,
      );
      expect(
        theme.textTheme.bodyMedium!.fontSize,
        closeTo(
          Typography.material2021().englishLike.bodyMedium!.fontSize! *
              referenceTheme.fontScale,
          1e-6,
        ),
      );
      // Layout paddings derive from the theme's spacing unit (10, not the
      // default 8): the screen pads its scroll view by space(3).
      final scroll = tester.widget<SingleChildScrollView>(
        find.descendant(
          of: find.byType(MothLoginScreen),
          matching: find.byType(SingleChildScrollView),
        ),
      );
      expect(scroll.padding, EdgeInsets.all(referenceTheme.spacingUnit * 3));

      await stop(tester);
    });

    testWidgets('dark mode renders the dark palette', (tester) async {
      tester.platformDispatcher.platformBrightnessTestValue = Brightness.dark;
      addTearDown(tester.platformDispatcher.clearPlatformBrightnessTestValue);
      await start(tester);
      await pumpLoginScreen(tester, theme: referenceTheme);

      final theme = themeAt(tester, MothLoginScreen.submitButtonKey);
      expect(theme.brightness, Brightness.dark);
      expect(theme.colorScheme.primary, referenceTheme.darkColors.primary);
      expect(
        theme.scaffoldBackgroundColor,
        referenceTheme.darkColors.background,
      );

      await stop(tester);
    });

    testWidgets('standalone screen picks up the server-delivered theme', (
      tester,
    ) async {
      await start(tester);
      moth.config.response.theme = fullProtoTheme()
        ..clearFontUrl()
        ..clearLogoLightUrl()
        ..clearLogoDarkUrl();
      await pumpLoginScreen(tester); // no explicit theme

      final theme = themeAt(tester, MothLoginScreen.submitButtonKey);
      expect(theme.colorScheme.primary, referenceTheme.colors.primary);

      await stop(tester);
    });

    testWidgets('logo renders when set, with the variant per brightness', (
      tester,
    ) async {
      await start(tester);
      final themed = MothTheme.fromProto(fullProtoTheme()..clearFontUrl());
      await pumpLoginScreen(tester, theme: themed);

      final light = tester.widget<Image>(
        find.descendant(
          of: find.byKey(MothLoginScreen.logoKey),
          matching: find.byType(Image),
        ),
      );
      expect((light.image as NetworkImage).url, themed.logoLightUrl);

      tester.platformDispatcher.platformBrightnessTestValue = Brightness.dark;
      addTearDown(tester.platformDispatcher.clearPlatformBrightnessTestValue);
      await tester.pump();
      final dark = tester.widget<Image>(
        find.descendant(
          of: find.byKey(MothLoginScreen.logoKey),
          matching: find.byType(Image),
        ),
      );
      expect((dark.image as NetworkImage).url, themed.logoDarkUrl);

      await stop(tester);
    });

    testWidgets('no logo, no legal links -> nothing rendered', (tester) async {
      await start(tester);
      await pumpLoginScreen(tester); // fake config carries no theme
      expect(find.byKey(MothLoginScreen.logoKey), findsNothing);
      expect(find.byKey(MothLoginScreen.termsLinkKey), findsNothing);
      expect(find.byKey(MothLoginScreen.privacyLinkKey), findsNothing);
      await stop(tester);
    });

    testWidgets('legal links render in the footer', (tester) async {
      await start(tester);
      await pumpLoginScreen(tester, theme: referenceTheme);
      expect(find.byKey(MothLoginScreen.termsLinkKey), findsOneWidget);
      expect(find.text('Terms of Service'), findsOneWidget);
      expect(find.byKey(MothLoginScreen.privacyLinkKey), findsOneWidget);
      expect(find.text('Privacy Policy'), findsOneWidget);
      await stop(tester);
    });
  });

  group('MothApp theming', () {
    testWidgets('login screen refreshes from fallback to the server theme '
        'and caches it', (tester) async {
      await start(tester);
      moth.config.response.theme = fullProtoTheme()
        ..clearFontUrl()
        ..clearLogoLightUrl()
        ..clearLogoDarkUrl();
      final cache = MothMemoryThemeCache();
      await tester.pumpWidget(
        MothApp(
          client: client,
          themeCache: cache,
          child: const MaterialApp(home: Text('app-home')),
        ),
      );
      await pumpUntilFound(tester, find.text('Welcome'));

      // The background refresh lands and the login screen re-themes.
      await pumpUntil(
        tester,
        () =>
            themeAt(
              tester,
              MothLoginScreen.submitButtonKey,
            ).colorScheme.primary ==
            referenceTheme.colors.primary,
        reason: 'server theme to apply',
      );
      expect((await settle(tester, cache.loadTheme()))!.revisionId, 'rev-1');

      await stop(tester);
    });

    testWidgets('explicit MothApp theme wins; no fetch of the server theme', (
      tester,
    ) async {
      await start(tester);
      // A server theme with a visibly different primary: if MothApp
      // installed the server theme instead of the explicit one, this green
      // would render and the expectations below would fail.
      moth.config.response.theme = fullProtoTheme()
        ..colors.primary = '#2E7D32'
        ..clearFontUrl()
        ..clearLogoLightUrl()
        ..clearLogoDarkUrl();
      await tester.pumpWidget(
        MothApp(
          client: client,
          theme: referenceTheme,
          child: const MaterialApp(home: Text('app-home')),
        ),
      );
      await pumpUntilFound(tester, find.text('Welcome'));
      final theme = themeAt(tester, MothLoginScreen.submitButtonKey);
      expect(theme.colorScheme.primary, referenceTheme.colors.primary);
      expect(theme.colorScheme.primary, isNot(mothParseHexColor('#2E7D32')));
      // Exactly one config call — the login screen's own fetch. A theme
      // controller revalidating the server theme would add a second.
      expect(moth.config.calls, 1);
      await stop(tester);
    });
  });

  group('building blocks standalone', () {
    testWidgets('MothEmailForm validates then hands over credentials', (
      tester,
    ) async {
      String? email;
      String? password;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: MothEmailForm(
              mode: MothEmailFormMode.signUp,
              passwordMinLength: 10,
              onSubmit: (e, p) {
                email = e;
                password = p;
              },
            ),
          ),
        ),
      );
      await tester.enterText(
        find.byKey(MothEmailForm.emailFieldKey),
        'jane@example.com',
      );
      await tester.enterText(
        find.byKey(MothEmailForm.passwordFieldKey),
        'short',
      );
      await tester.tap(find.byKey(MothEmailForm.submitButtonKey));
      await tester.pump();
      expect(find.text('Use at least 10 characters'), findsOneWidget);
      expect(email, isNull);

      await tester.enterText(
        find.byKey(MothEmailForm.passwordFieldKey),
        'long-enough-password',
      );
      await tester.tap(find.byKey(MothEmailForm.submitButtonKey));
      await tester.pump();
      expect(email, 'jane@example.com');
      expect(password, 'long-enough-password');
    });

    testWidgets('MothProviderButtons renders exactly the enabled providers', (
      tester,
    ) async {
      const config = MothProjectConfig(
        google: MothGoogleConfig(enabled: true),
        apple: MothAppleConfig(enabled: true),
        passwordMinLength: 8,
        signUpOpen: true,
      );
      MothOAuthProvider? selected;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: MothProviderButtons(
              config: config,
              onSelected: (provider) => selected = provider,
            ),
          ),
        ),
      );
      expect(find.byKey(MothProviderButtons.googleButtonKey), findsOneWidget);
      expect(find.byKey(MothProviderButtons.appleButtonKey), findsOneWidget);
      await tester.tap(find.byKey(MothProviderButtons.appleButtonKey));
      expect(selected, MothOAuthProvider.apple);
    });
  });
}
