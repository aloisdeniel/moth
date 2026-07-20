import 'dart:io' show Platform;

import 'package:flutter_test/flutter_test.dart';
import 'package:grpc/grpc.dart' show GrpcError;
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:moth_auth/moth_auth.dart';

import 'fakes.dart';

void main() {
  late FakeMoth moth;
  late InMemoryTokenStore store;
  late MothClient client;

  setUp(() async {
    moth = await startFakeMoth();
    store = InMemoryTokenStore();
    client = newClient(moth, store: store);
  });

  tearDown(() async {
    await client.dispose();
    await moth.shutdown();
  });

  Future<MothUser> signIn() =>
      client.signIn(email: 'jane@example.com', password: 'pw');

  group('metadata', () {
    test('every call carries key, platform and SDK version', () async {
      await signIn();
      final md = moth.auth.metadataByMethod['SignIn']!;
      expect(md['x-moth-key'], 'pk_test');
      expect(md['x-moth-platform'], Platform.operatingSystem);
      expect(md['x-moth-sdk-version'], mothSdkVersion);
      expect(md.containsKey('authorization'), isFalse);
    });

    test('authenticated calls carry the Bearer access token', () async {
      await signIn();
      await client.getMe();
      final md = moth.auth.metadataByMethod['GetMe']!;
      expect(md['authorization'], 'Bearer ${await client.accessToken()}');
    });

    test('ConfigService calls carry the key too', () async {
      await client.getProjectConfig();
      expect(moth.config.lastMetadata!['x-moth-key'], 'pk_test');
    });
  });

  group('sign-in', () {
    test('exposes the user with custom claims from the JWT', () async {
      final user = await signIn();
      expect(user.id, 'user-1');
      expect(user.email, 'jane@example.com');
      expect(user.emailVerified, isTrue);
      expect(user.displayName, 'Jane');
      expect(user.claims, {'role': 'admin'});
      expect(client.currentState, isA<MothSignedIn>());
      expect(client.currentUser?.id, 'user-1');
      expect(await store.load(), isNotNull);
    });

    test('signUp without tokens does not open a session', () async {
      moth.auth.signUpMode = SignUpMode.userOnly;
      final result = await client.signUp(
        email: 'jane@example.com',
        password: 'pw',
      );
      expect(result.signedIn, isFalse);
      expect(result.user?.email, 'jane@example.com');
      expect(client.currentUser, isNull);
    });

    test('signUp with tokens signs in immediately', () async {
      final result = await client.signUp(
        email: 'jane@example.com',
        password: 'pw',
      );
      expect(result.signedIn, isTrue);
      expect(client.currentState, isA<MothSignedIn>());
    });
  });

  group('token refresh', () {
    test(
      'N concurrent accessToken() calls share one RefreshToken RPC',
      () async {
        moth.auth.accessTokenTtl = Duration.zero; // sign-in token expired
        await signIn();
        moth.auth.accessTokenTtl = const Duration(hours: 1);
        moth.auth.refreshDelay = const Duration(milliseconds: 100);

        final tokens = await Future.wait([
          for (var i = 0; i < 5; i++) client.accessToken(),
        ]);

        expect(moth.auth.refreshCalls, 1);
        expect(moth.auth.refreshTokensSeen, ['rt_1']);
        expect(tokens.toSet(), hasLength(1));
      },
    );

    test('a token expiring within the skew is refreshed proactively', () async {
      moth.auth.accessTokenTtl = const Duration(seconds: 10); // < 30s skew
      await signIn();
      final before = await store.load();
      moth.auth.accessTokenTtl = const Duration(hours: 1);

      final token = await client.accessToken();

      expect(moth.auth.refreshCalls, 1);
      expect(token, isNot(before!.accessToken));
    });

    test('a fresh token is returned without a refresh', () async {
      await signIn();
      await client.accessToken();
      expect(moth.auth.refreshCalls, 0);
    });

    test('a rejected (reused) refresh token clears the session', () async {
      moth.auth.accessTokenTtl = Duration.zero;
      await signIn();
      moth.auth.refreshError = mothError(
        16 /* unauthenticated */,
        'REFRESH_TOKEN_REUSED',
        'token reused',
      );

      await expectLater(
        client.accessToken(),
        throwsA(isA<MothRefreshTokenReused>()),
      );

      expect(client.currentState, isA<MothSignedOut>());
      expect(await store.load(), isNull);
    });

    test('a network failure during refresh keeps the session', () async {
      moth.auth.accessTokenTtl = Duration.zero;
      await signIn();
      moth.auth.refreshError = GrpcError.unavailable('down');

      await expectLater(client.accessToken(), throwsA(isA<MothNetworkError>()));

      expect(client.currentState, isA<MothSignedIn>());
      expect(await store.load(), isNotNull);
    });

    test('accessToken() while signed out throws StateError', () async {
      await expectLater(client.accessToken(), throwsStateError);
    });

    test(
      'an access token rejected by the server refreshes and retries',
      () async {
        await signIn();
        // The token looks fresh client-side, but the server rejects it (clock
        // drift, device slept mid-call, TTL shortened server-side).
        moth.auth.nextError = mothError(
          16 /* unauthenticated */,
          'INVALID_ACCESS_TOKEN',
          'token expired',
        );

        final me = await client.getMe();

        expect(me.email, 'jane@example.com');
        expect(moth.auth.refreshCalls, 1);
        expect(client.currentState, isA<MothSignedIn>());
      },
    );
  });

  group('sign-out during refresh', () {
    test(
      'signOut during an in-flight refresh does not resurrect the session',
      () async {
        moth.auth.accessTokenTtl = Duration.zero; // sign-in token expired
        await signIn();
        moth.auth.accessTokenTtl = const Duration(hours: 1);
        moth.auth.refreshDelay = const Duration(milliseconds: 100);

        final states = <MothAuthState>[];
        client.authStateChanges.listen(states.add);

        // A background call (http interceptor, widget) starts a refresh...
        final refreshing = client.accessToken();
        // ...and the user taps "Sign out" while it is in flight.
        await client.signOut();

        await refreshing.then((_) {}, onError: (Object _) {});
        await pumpEventQueue();

        expect(client.currentState, isA<MothSignedOut>());
        expect(await store.load(), isNull);
        expect(states.last, isA<MothSignedOut>());
        // signOut waited for the rotation and revoked the current (rotated)
        // refresh token, not the stale predecessor.
        expect(moth.auth.lastSignOutRequest?.refreshToken, 'rt_2');
      },
    );
  });

  group('session restore', () {
    test('restores a fresh persisted session without a network call', () async {
      await signIn();
      await client.dispose();

      final restored = newClient(moth, store: store);
      addTearDown(restored.dispose);
      expect(restored.currentState, isA<MothAuthLoading>());

      final state = await restored.restore();

      expect(state, isA<MothSignedIn>());
      expect(restored.currentUser?.email, 'jane@example.com');
      expect(moth.auth.refreshCalls, 0);
      await restored.getMe(); // Bearer still attached
    });

    test('refreshes an expired persisted session', () async {
      moth.auth.accessTokenTtl = Duration.zero;
      await signIn();
      await client.dispose();
      moth.auth.accessTokenTtl = const Duration(hours: 1);

      final restored = newClient(moth, store: store);
      addTearDown(restored.dispose);
      final state = await restored.restore();

      expect(state, isA<MothSignedIn>());
      expect(moth.auth.refreshCalls, 1);
    });

    test('an empty store restores to signed out', () async {
      final state = await client.restore();
      expect(state, isA<MothSignedOut>());
    });

    test('a rejected refresh during restore ends signed out', () async {
      moth.auth.accessTokenTtl = Duration.zero;
      await signIn();
      await client.dispose();
      moth.auth.refreshError = mothError(
        16,
        'INVALID_REFRESH_TOKEN',
        'revoked',
      );

      final restored = newClient(moth, store: store);
      addTearDown(restored.dispose);
      final state = await restored.restore();

      expect(state, isA<MothSignedOut>());
      expect(await store.load(), isNull);
    });
  });

  group('auth state stream', () {
    test('replays the current state and emits every transition', () async {
      final states = <MothAuthState>[];
      final sub = client.authStateChanges.listen(states.add);
      addTearDown(sub.cancel);
      await pumpEventQueue();

      await client.restore(); // empty store
      await signIn();
      await client.signOut();
      await pumpEventQueue();

      expect(states, [
        isA<MothAuthLoading>(),
        isA<MothSignedOut>(),
        isA<MothSignedIn>(),
        isA<MothSignedOut>(),
      ]);
    });

    test('a late subscriber immediately receives the current state', () async {
      await signIn();
      expect(await client.authStateChanges.first, isA<MothSignedIn>());
    });
  });

  group('sign out', () {
    test('revokes the session and clears storage', () async {
      await signIn();
      await client.signOut();
      expect(client.currentState, isA<MothSignedOut>());
      expect(await store.load(), isNull);
      expect(moth.auth.metadataByMethod, contains('SignOut'));
    });

    test('ends signed out locally even when the RPC fails', () async {
      await signIn();
      moth.auth.nextError = GrpcError.internal('boom');
      await client.signOut();
      expect(client.currentState, isA<MothSignedOut>());
      expect(await store.load(), isNull);
    });
  });

  group('broken token store', () {
    late ThrowingTokenStore broken;
    late MothClient fragile;

    setUp(() {
      broken = ThrowingTokenStore();
      fragile = newClient(moth, store: broken);
    });

    tearDown(() => fragile.dispose());

    test('a failing load restores to signed out instead of throwing', () async {
      broken.throwOnLoad = true;
      final state = await fragile.restore();
      expect(state, isA<MothSignedOut>());
      expect(fragile.currentState, isA<MothSignedOut>());
    });

    test(
      'a failing save keeps the sign-in (session is in-memory only)',
      () async {
        broken.throwOnSave = true;
        final user = await fragile.signIn(
          email: 'jane@example.com',
          password: 'pw',
        );
        expect(user.email, 'jane@example.com');
        expect(fragile.currentState, isA<MothSignedIn>());
        await fragile.getMe(); // Bearer token still usable
      },
    );

    test('a failing clear still ends signed out locally', () async {
      await fragile.signIn(email: 'jane@example.com', password: 'pw');
      broken.throwOnClear = true;
      await fragile.signOut();
      expect(fragile.currentState, isA<MothSignedOut>());
    });
  });

  group('project config', () {
    test('maps the public configuration', () async {
      final config = await client.getProjectConfig();
      expect(config.google.enabled, isTrue);
      expect(config.google.webClientId, 'web-id');
      expect(config.google.iosClientId, isNull); // blank -> null
      expect(config.google.androidClientId, 'android-id');
      expect(config.apple.enabled, isFalse);
      expect(config.passwordMinLength, 10);
      expect(config.signUpOpen, isTrue);
    });
  });

  group('backend helper', () {
    test('authenticatedClient attaches a fresh Bearer token', () async {
      await signIn();
      http.Request? seen;
      final inner = MockClient((request) async {
        seen = request;
        return http.Response('ok', 200);
      });

      final api = authenticatedClient(client, inner: inner);
      final resp = await api.get(Uri.parse('http://api.example.com/todos'));

      expect(resp.statusCode, 200);
      expect(
        seen!.headers['authorization'],
        'Bearer ${await client.accessToken()}',
      );
    });
  });
}
