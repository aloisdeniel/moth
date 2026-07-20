// Widget tests for the moth_auth widget layer, against the in-process fake
// gRPC server from fakes.dart (async plumbing in widget_helpers.dart).
import 'dart:convert';

import 'package:crypto/crypto.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_auth/src/gen/moth/auth/v1/auth.pbenum.dart' as pb;
import 'package:moth_auth/src/gen/moth/auth/v1/config.pb.dart' as pbconfig;

import 'fakes.dart';
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

  Future<void> signInClient(WidgetTester tester) async {
    await settle(
      tester,
      client.signIn(email: 'jane@example.com', password: 'pw'),
    );
    await tester.pump();
  }

  group('MothApp', () {
    testWidgets('gates loading → signedOut → signedIn → signedOut', (
      tester,
    ) async {
      final store = GatedTokenStore();
      await start(tester, store: store);
      await tester.pumpWidget(
        MothApp(
          client: client,
          child: const MaterialApp(home: Text('app-home')),
        ),
      );

      // Restore is gated: splash with a progress indicator.
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      expect(find.text('app-home'), findsNothing);

      // Empty store -> signed out -> default MothLoginScreen.
      store.gate.complete();
      await pumpUntilFound(tester, find.byKey(MothLoginScreen.submitButtonKey));
      expect(find.text('app-home'), findsNothing);

      // Signing in swaps to the app.
      await signInClient(tester);
      expect(find.text('app-home'), findsOneWidget);
      expect(find.byType(MothLoginScreen), findsNothing);

      // Sign out through MothScope: back to the login screen.
      final scope = MothScope.of(tester.element(find.text('app-home')));
      await settle(tester, scope.signOut());
      await pumpUntilFound(tester, find.byKey(MothLoginScreen.submitButtonKey));
      expect(find.text('app-home'), findsNothing);

      await stop(tester);
    });

    testWidgets('requireAuth: false always renders child', (tester) async {
      await start(tester, store: GatedTokenStore());
      await tester.pumpWidget(
        MothApp(
          client: client,
          requireAuth: false,
          child: const MaterialApp(home: Text('open')),
        ),
      );
      // Even while the state is still loading.
      expect(client.currentState, isA<MothAuthLoading>());
      expect(find.text('open'), findsOneWidget);
      await stop(tester);
    });
  });

  group('MothScope', () {
    testWidgets('rebuilds dependents on every state transition', (
      tester,
    ) async {
      await start(tester);
      final seen = <Type>[];
      await tester.pumpWidget(
        MothApp(
          client: client,
          requireAuth: false,
          child: Directionality(
            textDirection: TextDirection.ltr,
            child: Builder(
              builder: (context) {
                final scope = MothScope.of(context);
                seen.add(scope.state.runtimeType);
                return Text('user:${scope.user?.email}');
              },
            ),
          ),
        ),
      );
      expect(seen.first, MothAuthLoading);

      await pumpUntil(tester, () => seen.contains(MothSignedOut));
      expect(find.text('user:null'), findsOneWidget);

      await signInClient(tester);
      expect(seen.last, MothSignedIn);
      expect(find.text('user:jane@example.com'), findsOneWidget);

      await settle(tester, client.signOut());
      await tester.pump();
      expect(seen.last, MothSignedOut);
      expect(find.text('user:null'), findsOneWidget);

      await stop(tester);
    });
  });

  group('MothLoginScreen', () {
    Future<void> pumpLoginScreen(
      WidgetTester tester, {
      MothOAuthAdapter? adapter,
    }) async {
      await tester.pumpWidget(
        MaterialApp(
          home: MothLoginScreen(client: client, adapter: adapter),
        ),
      );
      await pumpUntilFound(tester, find.byKey(MothLoginScreen.submitButtonKey));
    }

    testWidgets('validates email and password per project config', (
      tester,
    ) async {
      await start(tester);
      await pumpLoginScreen(tester);

      await tester.enterText(
        find.byKey(MothLoginScreen.emailFieldKey),
        'not-an-email',
      );
      await tester.enterText(
        find.byKey(MothLoginScreen.passwordFieldKey),
        'whatever',
      );
      await tester.tap(find.byKey(MothLoginScreen.submitButtonKey));
      await tester.pump();
      expect(find.text('Enter a valid email address'), findsOneWidget);
      expect(moth.auth.metadataByMethod.containsKey('SignIn'), isFalse);

      // Sign-up mode enforces the configured minimum length (10 in the
      // fake config).
      await tester.tap(find.byKey(MothLoginScreen.toggleModeKey));
      await tester.pump();
      // The sign-up title and submit button both read 'Create account'.
      expect(find.text('Create account'), findsWidgets);
      await tester.enterText(
        find.byKey(MothLoginScreen.emailFieldKey),
        'jane@example.com',
      );
      await tester.enterText(
        find.byKey(MothLoginScreen.passwordFieldKey),
        'short',
      );
      await tester.tap(find.byKey(MothLoginScreen.submitButtonKey));
      await tester.pump();
      expect(find.text('Use at least 10 characters'), findsOneWidget);
      expect(moth.auth.metadataByMethod.containsKey('SignUp'), isFalse);

      await stop(tester);
    });

    testWidgets('sign-in failure shows friendly error banner', (tester) async {
      await start(tester);
      await pumpLoginScreen(tester);
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
      expect(find.text('Incorrect email or password.'), findsOneWidget);

      await stop(tester);
    });

    testWidgets(
      'sign-up without immediate session prompts email verification',
      (tester) async {
        await start(tester);
        moth.auth.signUpMode = SignUpMode.userOnly;
        await pumpLoginScreen(tester);

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
        expect(find.textContaining('check your inbox'), findsOneWidget);
        // Back on the sign-in form, ready for the post-verification login.
        expect(find.text('Sign in'), findsWidgets);
        expect(client.currentState, isNot(isA<MothSignedIn>()));

        await stop(tester);
      },
    );

    testWidgets('forgot-password requests a reset and confirms', (
      tester,
    ) async {
      await start(tester);
      await pumpLoginScreen(tester);

      await tester.tap(find.byKey(MothLoginScreen.forgotPasswordKey));
      await tester.pump();
      expect(find.text('Reset password'), findsOneWidget);

      await tester.enterText(
        find.byKey(MothLoginScreen.emailFieldKey),
        'jane@example.com',
      );
      await tester.tap(find.byKey(MothLoginScreen.sendResetKey));
      await pumpUntil(
        tester,
        () => moth.auth.metadataByMethod.containsKey('RequestPasswordReset'),
        reason: 'RequestPasswordReset RPC',
      );
      await pumpUntilFound(tester, find.text('Check your email'));
      expect(find.textContaining('jane@example.com'), findsOneWidget);

      await tester.tap(find.byKey(MothLoginScreen.backToSignInKey));
      await tester.pump();
      expect(find.byKey(MothLoginScreen.submitButtonKey), findsOneWidget);

      await stop(tester);
    });

    testWidgets('provider buttons follow project config; missing adapter '
        'explains itself', (tester) async {
      await start(tester);
      // Default fake config: Google enabled, Apple disabled.
      await pumpLoginScreen(tester);
      expect(find.byKey(MothLoginScreen.googleButtonKey), findsOneWidget);
      expect(find.byKey(MothLoginScreen.appleButtonKey), findsNothing);

      await tester.tap(find.byKey(MothLoginScreen.googleButtonKey));
      await tester.pump();
      expect(find.textContaining('MothOAuthAdapter'), findsOneWidget);
      expect(
        moth.auth.metadataByMethod.containsKey('SignInWithOAuth'),
        isFalse,
      );

      await stop(tester);
    });

    testWidgets('apple-only closed-signup config; adapter drives the flow', (
      tester,
    ) async {
      await start(tester);
      moth.config.response = pbconfig.GetProjectConfigResponse(
        google: pbconfig.GoogleConfig(enabled: false),
        apple: pbconfig.AppleConfig(enabled: true),
        passwordMinLength: 8,
        signUpOpen: false,
      );
      final adapter = RecordingAdapter();
      await pumpLoginScreen(tester, adapter: adapter);

      expect(find.byKey(MothLoginScreen.googleButtonKey), findsNothing);
      expect(find.byKey(MothLoginScreen.appleButtonKey), findsOneWidget);
      // Sign-up closed: no toggle.
      expect(find.byKey(MothLoginScreen.toggleModeKey), findsNothing);

      await tester.tap(find.byKey(MothLoginScreen.appleButtonKey));
      await pumpUntil(
        tester,
        () => moth.auth.lastOAuthRequest != null,
        reason: 'SignInWithOAuth RPC',
      );
      final request = moth.auth.lastOAuthRequest!;
      expect(request.provider, pb.OAuthProvider.OAUTH_PROVIDER_APPLE);
      expect(request.idToken, 'apple-id-token');
      // The adapter got the SHA-256 of the raw nonce the server received.
      expect(
        adapter.hashedNonce,
        sha256.convert(utf8.encode(request.nonce)).toString(),
      );
      await pumpUntil(tester, () => client.currentState is MothSignedIn);

      await stop(tester);
    });
  });

  group('showMothDeleteAccountDialog', () {
    testWidgets('re-auths, surfaces server errors, deletes and signs out', (
      tester,
    ) async {
      await start(tester);
      await signInClient(tester);
      await tester.pumpWidget(
        MothApp(
          client: client,
          child: MaterialApp(
            home: Builder(
              builder: (context) => Center(
                child: TextButton(
                  onPressed: () => showMothDeleteAccountDialog(context),
                  child: const Text('delete'),
                ),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('delete'));
      await tester.pump();
      expect(find.text('Delete account?'), findsOneWidget);

      // Wrong password: the typed error is shown inside the dialog.
      moth.auth.nextError = mothError(16, 'INVALID_CREDENTIALS', 'nope');
      await tester.enterText(
        find.byKey(mothDeletePasswordFieldKey),
        'wrong-password',
      );
      await tester.tap(find.byKey(mothDeleteConfirmButtonKey));
      await pumpUntilFound(tester, find.text('Incorrect email or password.'));
      expect(find.text('Delete account?'), findsOneWidget);

      // Second attempt succeeds: dialog closes, state flips to signed out
      // and MothApp falls back to the login screen.
      await tester.tap(find.byKey(mothDeleteConfirmButtonKey));
      await pumpUntil(tester, () => client.currentState is MothSignedOut);
      await pumpUntilFound(tester, find.byKey(MothLoginScreen.submitButtonKey));
      expect(find.text('Delete account?'), findsNothing);

      await stop(tester);
    });
  });
}
