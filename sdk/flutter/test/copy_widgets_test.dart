// Widget tests for the localized SDK screens: MothLoginScreen (sign-in and
// sign-up) and MothPaywallScreen render the negotiated project copy when the
// server delivers it and the bundled fallback otherwise, driven by the
// in-process fake gRPC server from fakes.dart.
import 'package:fixnum/fixnum.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_auth/src/gen/moth/auth/v1/config.pb.dart' as pbconfig;
import 'package:moth_auth/src/gen/moth/billing/v1/billing.pb.dart' as pb;

import 'fakes.dart';
import 'widget_helpers.dart';

/// English strings that must never survive a French render of the sign-in /
/// sign-up form (the "no hardcoded English" guard).
const _englishSignInStrings = <String>[
  'Sign in',
  'Forgot password?',
  "Don't have an account?",
  'Sign up',
  'Password',
  'Create account',
  'Already have an account?',
  'Continue with Google',
  'Continue with Apple',
];

/// A theme carrying terms/privacy URLs, so the login screen renders its legal
/// footer (the links only appear when the URLs are set).
MothTheme _themeWithLegalLinks() => MothTheme.fromProto(
  pbconfig.Theme(
    termsUrl: 'https://example.com/terms',
    privacyUrl: 'https://example.com/privacy',
  ),
);

void main() {
  late FakeMoth moth;
  late MothClient client;

  Future<void> pumpLogin(
    WidgetTester tester, {
    Locale locale = const Locale('fr'),
    String appName = 'Aurora',
  }) async {
    moth = await runReal(tester, startFakeMoth);
    client = newClient(moth, locale: locale, appName: appName);
    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: mothLocalizationsDelegates,
        supportedLocales: mothSupportedLocales,
        home: MothLoginScreen(client: client),
      ),
    );
    await pumpUntilFound(tester, find.byKey(MothLoginScreen.submitButtonKey));
  }

  Future<void> stop(WidgetTester tester) async {
    await settle(tester, client.dispose());
    await settle(tester, moth.shutdown());
  }

  group('MothLoginScreen localization', () {
    testWidgets('renders the bundled French copy offline (no server copy)', (
      tester,
    ) async {
      // The default fake config carries no Copy, so the screen falls back to
      // the SDK's bundled French catalog for the fr device locale.
      await pumpLogin(tester);

      expect(find.text('Connexion'), findsOneWidget); // sign_in.title
      expect(find.text('Se connecter'), findsWidgets); // sign_in.submit
      expect(find.text('Mot de passe oublié ?'), findsOneWidget);
      expect(find.text('Vous n\'avez pas de compte ?'), findsOneWidget);
      expect(find.text('S\'inscrire'), findsOneWidget);
      // The {app} placeholder is filled from MothConfig.appName offline.
      expect(find.text('Bon retour sur Aurora.'), findsOneWidget);

      for (final english in _englishSignInStrings) {
        expect(
          find.text(english),
          findsNothing,
          reason: 'French render still shows English "$english"',
        );
      }

      await stop(tester);
    });

    testWidgets('sign-up mode renders the bundled French copy', (tester) async {
      await pumpLogin(tester);
      await tester.tap(find.byKey(MothLoginScreen.toggleModeKey));
      await tester.pump();

      expect(find.text('Créer un compte'), findsWidgets); // title + submit
      expect(find.text('Vous avez déjà un compte ?'), findsOneWidget);
      expect(find.text('Créez votre compte Aurora.'), findsOneWidget);

      await stop(tester);
    });

    testWidgets('renders the server-delivered French copy override', (
      tester,
    ) async {
      moth = await runReal(tester, startFakeMoth);
      // The project customized its French sign-in title; the server negotiates
      // fr and returns the override, which wins over the bundled default.
      moth.config.response = pbconfig.GetProjectConfigResponse(
        google: pbconfig.GoogleConfig(enabled: false),
        apple: pbconfig.AppleConfig(enabled: false),
        passwordMinLength: 8,
        signUpOpen: true,
        copy: pbconfig.Copy(
          locale: 'fr',
          copyRevision: 'c1',
          messages: <String, String>{
            'sign_in.title': 'Bienvenue chez Aurora',
            'sign_in.submit': 'Entrer',
          }.entries,
        ),
      );
      client = newClient(moth, locale: const Locale('fr'));
      await tester.pumpWidget(
        MaterialApp(
          localizationsDelegates: mothLocalizationsDelegates,
          supportedLocales: mothSupportedLocales,
          home: MothLoginScreen(client: client),
        ),
      );
      await pumpUntilFound(tester, find.text('Bienvenue chez Aurora'));

      expect(find.text('Entrer'), findsWidgets); // server override submit
      // A key the override omitted still resolves from the bundled French.
      expect(find.text('Mot de passe oublié ?'), findsOneWidget);
      expect(find.text('Connexion'), findsNothing); // bundled title superseded

      await stop(tester);
    });

    testWidgets('form validators render in French, never English', (
      tester,
    ) async {
      await pumpLogin(tester);

      await tester.enterText(
        find.byKey(MothLoginScreen.emailFieldKey),
        'not-an-email',
      );
      await tester.tap(find.byKey(MothLoginScreen.submitButtonKey));
      await tester.pump();

      expect(find.text('Saisissez une adresse e-mail valide'), findsOneWidget);
      expect(find.text('Enter a valid email address'), findsNothing);
      expect(find.text('Enter your email address'), findsNothing);

      await stop(tester);
    });

    testWidgets('sign-up verification info banner renders in French', (
      tester,
    ) async {
      moth = await runReal(tester, startFakeMoth);
      moth.auth.signUpMode = SignUpMode.userOnly; // no session → verify first
      client = newClient(moth, locale: const Locale('fr'), appName: 'Aurora');
      await tester.pumpWidget(
        MaterialApp(
          localizationsDelegates: mothLocalizationsDelegates,
          supportedLocales: mothSupportedLocales,
          home: MothLoginScreen(client: client),
        ),
      );
      await pumpUntilFound(tester, find.byKey(MothLoginScreen.submitButtonKey));

      await tester.tap(find.byKey(MothLoginScreen.toggleModeKey));
      await tester.pump();
      await tester.enterText(
        find.byKey(MothLoginScreen.emailFieldKey),
        'jane@example.com',
      );
      await tester.enterText(
        find.byKey(MothLoginScreen.passwordFieldKey),
        'long-enough-password',
      );
      await tester.tap(find.byKey(MothLoginScreen.submitButtonKey));
      await pumpUntilFound(tester, find.byKey(MothLoginScreen.infoBannerKey));

      // The bundled French sign_up.verify_sent, not the English placeholder.
      expect(
        find.textContaining('vérifier votre adresse e-mail'),
        findsWidgets,
      );
      expect(find.textContaining('check your inbox'), findsNothing);

      await stop(tester);
    });

    testWidgets('the friendly error banner renders in French', (tester) async {
      await pumpLogin(tester);
      moth.auth.nextError = mothError(16, 'INVALID_CREDENTIALS', 'nope');

      await tester.enterText(
        find.byKey(MothLoginScreen.emailFieldKey),
        'jane@example.com',
      );
      await tester.enterText(
        find.byKey(MothLoginScreen.passwordFieldKey),
        'wrong-password',
      );
      await tester.tap(find.byKey(MothLoginScreen.submitButtonKey));
      await pumpUntilFound(tester, find.byKey(MothLoginScreen.errorBannerKey));

      // friendlyMothErrorMessage resolves sign_in.error_invalid in French.
      expect(find.text('E-mail ou mot de passe incorrect.'), findsOneWidget);
      expect(find.text('Incorrect email or password.'), findsNothing);

      await stop(tester);
    });

    testWidgets('legal footer link labels render in French', (tester) async {
      moth = await runReal(tester, startFakeMoth);
      client = newClient(moth, locale: const Locale('fr'), appName: 'Aurora');
      await tester.pumpWidget(
        MaterialApp(
          localizationsDelegates: mothLocalizationsDelegates,
          supportedLocales: mothSupportedLocales,
          home: MothLoginScreen(client: client, theme: _themeWithLegalLinks()),
        ),
      );
      await pumpUntilFound(tester, find.byKey(MothLoginScreen.submitButtonKey));

      expect(
        find.descendant(
          of: find.byKey(MothLoginScreen.termsLinkKey),
          matching: find.text('Conditions d\'utilisation'),
        ),
        findsOneWidget,
      );
      expect(
        find.descendant(
          of: find.byKey(MothLoginScreen.privacyLinkKey),
          matching: find.text('Politique de confidentialité'),
        ),
        findsOneWidget,
      );
      expect(find.text('Terms of Service'), findsNothing);
      expect(find.text('Privacy Policy'), findsNothing);

      await stop(tester);
    });
  });

  group('MothPaywallScreen localization', () {
    pb.Offering offering() => pb.Offering(
      identifier: 'default',
      isDefault: true,
      products: [
        pb.OfferingProduct(
          identifier: 'monthly',
          displayName: 'Mensuel',
          appleProductId: 'com.example.monthly',
          billingPeriod: 'P1M',
          priceAmountMicros: Int64(9990000),
          currency: 'EUR',
          entitlements: ['pro'],
        ),
      ],
    );

    testWidgets('renders bundled French CTA/restore + server French header', (
      tester,
    ) async {
      moth = await runReal(tester, startFakeMoth);
      moth.billing.offering = offering();
      // The server delivers the paywall headline/subtitle already localized;
      // the CTA and restore come from the SDK's bundled French catalog.
      moth.billing.paywall = pb.Paywall(
        revisionId: 'pw-fr',
        headline: 'Débloquez Premium',
        subtitle: 'Accès complet à tout.',
        layout: pb.PaywallLayout.PAYWALL_LAYOUT_TILES,
      );
      client = newClient(moth, locale: const Locale('fr'));
      await tester.pumpWidget(
        MaterialApp(
          localizationsDelegates: mothLocalizationsDelegates,
          supportedLocales: mothSupportedLocales,
          home: MothPaywallScreen(client: client),
        ),
      );
      await pumpUntilFound(tester, find.text('Débloquez Premium'));

      // Bundled French: paywall.cta + paywall.restore.
      expect(find.text('Continuer'), findsOneWidget);
      expect(find.text('Restaurer les achats'), findsOneWidget);
      expect(find.text('Restore purchases'), findsNothing);
      expect(find.text('Subscribe'), findsNothing);

      await stop(tester);
    });

    testWidgets('tier badges, price period and footer render in French', (
      tester,
    ) async {
      moth = await runReal(tester, startFakeMoth);
      moth.billing.offering = pb.Offering(
        identifier: 'default',
        isDefault: true,
        products: [
          pb.OfferingProduct(
            identifier: 'yearly',
            displayName: 'Annuel',
            appleProductId: 'com.example.yearly',
            billingPeriod: 'P1Y',
            priceAmountMicros: Int64(59990000),
            currency: 'EUR',
            trialPeriod: 'P1W',
            entitlements: ['pro'],
            highlighted: true,
          ),
        ],
      );
      moth.billing.paywall = pb.Paywall(
        revisionId: 'pw-fr',
        headline: 'Débloquez Premium',
        subtitle: 'Accès complet à tout.',
        layout: pb.PaywallLayout.PAYWALL_LAYOUT_TILES,
        termsUrl: 'https://example.com/terms',
        privacyUrl: 'https://example.com/privacy',
      );
      client = newClient(moth, locale: const Locale('fr'));
      await tester.pumpWidget(
        MaterialApp(
          localizationsDelegates: mothLocalizationsDelegates,
          supportedLocales: mothSupportedLocales,
          home: MothPaywallScreen(client: client),
        ),
      );
      await pumpUntilFound(tester, find.text('Débloquez Premium'));

      expect(find.text('Le plus populaire'), findsOneWidget);
      expect(find.text('Essai gratuit d\'une semaine'), findsOneWidget);
      expect(find.text('€59.99 / an'), findsOneWidget);
      expect(find.text('Conditions d\'utilisation'), findsOneWidget);
      expect(find.text('Politique de confidentialité'), findsOneWidget);
      // No English survivors.
      expect(find.text('Most popular'), findsNothing);
      expect(find.text('1-week free trial'), findsNothing);
      expect(find.text('Terms of Service'), findsNothing);
      expect(find.text('Privacy Policy'), findsNothing);

      await stop(tester);
    });
  });
}
