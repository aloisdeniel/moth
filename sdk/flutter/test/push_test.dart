// Push registration lifecycle: MothPushController against the in-process
// fake push service (register payloads, launch/rotation/permission
// re-registration, non-fatal failures) and the MothApp/MothScope wiring
// (sign-out unregisters before the session drops, no-adapter no-op).
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:grpc/grpc.dart' show GrpcError;
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_auth/src/gen/moth/auth/v1/config.pb.dart' as authpb;
import 'package:moth_auth/src/gen/moth/push/v1/push.pbenum.dart' as pbe;

import 'fakes.dart';
import 'widget_helpers.dart';

Future<void> _waitUntil(bool Function() cond, {String? reason}) async {
  final deadline = DateTime.now().add(const Duration(seconds: 5));
  while (!cond()) {
    if (DateTime.now().isAfter(deadline)) {
      fail('timed out waiting for ${reason ?? 'condition'}');
    }
    await Future<void>.delayed(const Duration(milliseconds: 10));
  }
}

/// A real-time window for work that must NOT have happened; polling can't
/// prove a negative, so give the machinery time to misbehave first.
Future<void> _settleQuietly() =>
    Future<void>.delayed(const Duration(milliseconds: 150));

void main() {
  group('MothPushController', () {
    late FakeMoth moth;
    late MothClient client;
    late FakePushAdapter adapter;
    late MothMemoryPushDeviceIdStore deviceIds;
    late MothPushController controller;

    setUp(() async {
      moth = await startFakeMoth();
      client = newClient(moth, locale: const Locale('fr', 'FR'));
      adapter = FakePushAdapter();
      deviceIds = MothMemoryPushDeviceIdStore();
      controller = MothPushController(
        client: client,
        adapter: adapter,
        deviceIdStore: deviceIds,
      );
      await controller.start();
    });

    tearDown(() async {
      controller.dispose();
      await client.dispose();
      await moth.shutdown();
    });

    Future<MothUser> signIn() =>
        client.signIn(email: 'jane@example.com', password: 'pw');

    test('sign-in registers the device with token, permission, a stable '
        'device id and metadata', () async {
      await signIn();
      await _waitUntil(() => moth.push.registerCalls == 1, reason: 'register');

      final req = moth.push.lastRegister!;
      expect(req.target, pbe.PushTarget.PUSH_TARGET_FCM);
      expect(req.token, 'fcm-token-1');
      expect(req.permission, pbe.PushPermission.PUSH_PERMISSION_GRANTED);
      // A generated v4 UUID, persisted for the next registration.
      expect(req.deviceId, hasLength(36));
      expect(await deviceIds.load(), req.deviceId);
      // Metadata the SDK can source itself: platform, OS version, locale.
      expect(req.metadata.platform, isNotEmpty);
      expect(req.metadata.osVersion, isNotEmpty);
      expect(req.metadata.locale, 'fr-FR');
      // Billing-style call: publishable key + Bearer JWT.
      final headers = moth.push.metadataByMethod['RegisterDevice']!;
      expect(headers['x-moth-key'], 'pk_test');
      expect(headers['authorization'], startsWith('Bearer '));

      await _waitUntil(() => controller.value.registered, reason: 'status');
      expect(
        controller.value,
        const MothPushStatus(
          available: true,
          permission: MothPushPermission.granted,
          registered: true,
        ),
      );
    });

    test('adapter metadata wins over SDK-sourced fields', () async {
      adapter.metadata = const MothPushDeviceMetadata(
        model: 'Pixel 9',
        appVersion: '2.4.1+87',
        osVersion: 'Android 16',
      );
      await signIn();
      await _waitUntil(() => moth.push.registerCalls == 1, reason: 'register');
      final metadata = moth.push.lastRegister!.metadata;
      expect(metadata.model, 'Pixel 9');
      expect(metadata.appVersion, '2.4.1+87');
      expect(metadata.osVersion, 'Android 16');
      // Fields the adapter left empty are still filled by the SDK.
      expect(metadata.platform, isNotEmpty);
      expect(metadata.locale, 'fr-FR');
    });

    test(
      'a launch while signed in re-registers with the same device id',
      () async {
        final store = InMemoryTokenStore();
        final relaunchClient = newClient(
          moth,
          store: store,
          locale: const Locale('fr', 'FR'),
        );
        final firstController = MothPushController(
          client: relaunchClient,
          adapter: adapter,
          deviceIdStore: deviceIds,
        );
        await firstController.start();
        await relaunchClient.signIn(email: 'jane@example.com', password: 'pw');
        await _waitUntil(
          () => moth.push.registerCalls == 1,
          reason: 'register',
        );
        final deviceId = moth.push.lastRegister!.deviceId;
        firstController.dispose();
        await relaunchClient.dispose();

        // Next launch: a fresh client restores the persisted session; the
        // controller re-registers (the server upserts) with the persisted id.
        final secondClient = newClient(
          moth,
          store: store,
          locale: const Locale('fr', 'FR'),
        );
        final secondController = MothPushController(
          client: secondClient,
          adapter: adapter,
          deviceIdStore: deviceIds,
        );
        await secondController.start();
        await secondClient.restore();
        await _waitUntil(
          () => moth.push.registerCalls == 2,
          reason: 're-register on launch',
        );
        expect(moth.push.lastRegister!.deviceId, deviceId);
        secondController.dispose();
        await secondClient.dispose();
      },
    );

    test('token rotation re-registers with the new token', () async {
      await signIn();
      await _waitUntil(() => moth.push.registerCalls == 1, reason: 'register');
      final deviceId = moth.push.lastRegister!.deviceId;

      adapter.rotate('fcm-token-2');
      await _waitUntil(
        () => moth.push.registerCalls == 2,
        reason: 're-register on rotation',
      );
      final req = moth.push.lastRegister!;
      expect(req.token, 'fcm-token-2');
      expect(req.deviceId, deviceId);
    });

    test('requestPermission prompts once and re-registers with the updated '
        'permission', () async {
      await signIn();
      await _waitUntil(() => moth.push.registerCalls == 1, reason: 'register');
      expect(
        moth.push.lastRegister!.permission,
        pbe.PushPermission.PUSH_PERMISSION_GRANTED,
      );

      adapter.requestResult = MothPushPermission.denied;
      final result = await controller.requestPermission();
      expect(result, MothPushPermission.denied);
      expect(adapter.requestPermissionCalls, 1);
      await _waitUntil(
        () => moth.push.registerCalls == 2,
        reason: 're-register on permission change',
      );
      expect(
        moth.push.lastRegister!.permission,
        pbe.PushPermission.PUSH_PERMISSION_DENIED,
      );
      expect(controller.value.permission, MothPushPermission.denied);
    });

    test('push disabled in the project config never registers', () async {
      moth.config.response.push = authpb.PushConfig(enabled: false);
      await signIn();
      await _waitUntil(() => moth.config.calls >= 1, reason: 'config fetch');
      await _settleQuietly();
      expect(moth.push.registerCalls, 0);
      expect(adapter.getTokenCalls, 0);
      expect(controller.value, MothPushStatus.unavailable);
    });

    test(
      'no credential yet reports the permission without registering',
      () async {
        adapter.token = null;
        await signIn();
        await _waitUntil(() => adapter.getTokenCalls >= 1, reason: 'getToken');
        await _settleQuietly();
        expect(moth.push.registerCalls, 0);
        expect(
          controller.value,
          const MothPushStatus(
            available: true,
            permission: MothPushPermission.granted,
          ),
        );
      },
    );

    test('registration failure is non-fatal: sign-in completes, retry is the '
        'next trigger', () async {
      moth.push.registerError = GrpcError.unavailable('push down');
      final user = await signIn();
      expect(user.email, 'jane@example.com');
      await _waitUntil(() => moth.push.registerCalls == 1, reason: 'attempt');
      await _settleQuietly();
      // Still signed in; push simply reports not-registered.
      expect(client.currentState, isA<MothSignedIn>());
      expect(controller.value.registered, isFalse);

      // The next trigger (a token rotation) retries and succeeds.
      moth.push.registerError = null;
      adapter.rotate('fcm-token-2');
      await _waitUntil(() => controller.value.registered, reason: 'retry');
    });

    test(
      'a broken adapter (getToken throws) never surfaces into auth',
      () async {
        adapter.throwOnGetToken = StateError('no Firebase config');
        await signIn();
        await _waitUntil(() => adapter.getTokenCalls >= 1, reason: 'getToken');
        await _settleQuietly();
        expect(client.currentState, isA<MothSignedIn>());
        expect(moth.push.registerCalls, 0);
        expect(controller.value.registered, isFalse);
      },
    );

    test('unregisterForSignOut revokes this installation but keeps the '
        'device id', () async {
      await signIn();
      await _waitUntil(() => moth.push.registerCalls == 1, reason: 'register');
      final deviceId = moth.push.lastRegister!.deviceId;

      await controller.unregisterForSignOut();
      expect(moth.push.unregisterCalls, 1);
      expect(moth.push.lastUnregister!.deviceId, deviceId);
      expect(controller.value.registered, isFalse);
      // The id identifies the installation, not the user: it survives.
      expect(await deviceIds.load(), deviceId);
    });
  });

  group('MothApp push wiring', () {
    late FakeMoth moth;
    late MothClient client;

    Future<void> start(WidgetTester tester) async {
      moth = await runReal(tester, startFakeMoth);
      client = newClient(moth);
    }

    Future<void> stop(WidgetTester tester) async {
      await settle(tester, client.dispose());
      await settle(tester, moth.shutdown());
    }

    Future<void> signInClient(WidgetTester tester) async {
      // Let the initial login screen finish mounting first: signing in while
      // its config fetch is mid-flight leaves an orphaned RPC that blocks the
      // channel shutdown in stop().
      await pumpUntilFound(tester, find.byKey(MothLoginScreen.submitButtonKey));
      await settle(
        tester,
        client.signIn(email: 'jane@example.com', password: 'pw'),
      );
      await tester.pump();
    }

    testWidgets('sign-out through MothScope unregisters before the session '
        'drops', (tester) async {
      await start(tester);
      final adapter = FakePushAdapter();
      await tester.pumpWidget(
        MothApp(
          client: client,
          pushAdapter: adapter,
          pushDeviceIdStore: MothMemoryPushDeviceIdStore(),
          child: const MaterialApp(home: Text('app-home')),
        ),
      );
      await signInClient(tester);
      // Wait for the registration to round-trip and surface on the scope.
      await pumpUntil(tester, () {
        final home = find.text('app-home').evaluate();
        if (home.isEmpty) return false;
        return MothScope.of(home.first).pushStatus.registered;
      }, reason: 'registered status on the scope');
      expect(moth.push.registerCalls, 1);

      final scope = MothScope.of(tester.element(find.text('app-home')));
      expect(scope.pushStatus.permission, MothPushPermission.granted);

      await settle(tester, scope.signOut());
      expect(moth.push.unregisterCalls, 1);
      expect(
        moth.push.lastUnregister!.deviceId,
        moth.push.lastRegister!.deviceId,
      );
      // The revocation ran while the session could still authenticate it —
      // strictly before the SignOut RPC dropped the session.
      expect(
        moth.callLog.indexOf('UnregisterDevice'),
        lessThan(moth.callLog.indexOf('SignOut')),
      );
      expect(
        moth.push.metadataByMethod['UnregisterDevice']!['authorization'],
        startsWith('Bearer '),
      );
      await pumpUntilFound(tester, find.byKey(MothLoginScreen.submitButtonKey));
      await stop(tester);
    });

    testWidgets('no adapter is a clean no-op', (tester) async {
      await start(tester);
      await tester.pumpWidget(
        MothApp(
          client: client,
          child: const MaterialApp(home: Text('app-home')),
        ),
      );
      await signInClient(tester);
      await pumpUntilFound(tester, find.text('app-home'));

      final scope = MothScope.of(tester.element(find.text('app-home')));
      expect(scope.pushController, isNull);
      expect(scope.pushStatus, MothPushStatus.unavailable);
      // requestPushPermission never prompts anything and never throws.
      final permission = await settle(tester, scope.requestPushPermission());
      expect(permission, MothPushPermission.unknown);

      await settle(tester, scope.signOut());
      await pumpUntilFound(tester, find.byKey(MothLoginScreen.submitButtonKey));
      expect(moth.push.registerCalls, 0);
      expect(moth.push.unregisterCalls, 0);
      await stop(tester);
    });

    testWidgets('registration failure never blocks the signed-in UI', (
      tester,
    ) async {
      await start(tester);
      moth.push.registerError = GrpcError.unavailable('push down');
      final adapter = FakePushAdapter();
      await tester.pumpWidget(
        MothApp(
          client: client,
          pushAdapter: adapter,
          pushDeviceIdStore: MothMemoryPushDeviceIdStore(),
          child: const MaterialApp(home: Text('app-home')),
        ),
      );
      await signInClient(tester);
      await pumpUntilFound(tester, find.text('app-home'));
      await pumpUntil(
        tester,
        () => moth.push.registerCalls == 1,
        reason: 'failed registration attempt',
      );
      await tester.pump();
      final scope = MothScope.of(tester.element(find.text('app-home')));
      expect(scope.state, isA<MothSignedIn>());
      expect(scope.pushStatus.registered, isFalse);

      // The retry policy is simply the next trigger: heal the server and
      // rotate the token — the registration lands without any auth churn.
      moth.push.registerError = null;
      adapter.rotate('fcm-token-2');
      await pumpUntil(
        tester,
        () => MothScope.of(
          tester.element(find.text('app-home')),
        ).pushStatus.registered,
        reason: 'retry to register',
      );
      await stop(tester);
    });
  });
}
