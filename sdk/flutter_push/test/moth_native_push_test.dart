// Contract tests for MothNativePush against a fake method-channel host: the
// (target, token) credential and permission strings handed to RegisterDevice
// are the contract.
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_push/moth_push.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
  final calls = <MethodCall>[];

  /// Installs the fake host for this test.
  void host(Future<Object?> Function(MethodCall call) handler) {
    messenger.setMockMethodCallHandler(MothNativePush.channel, (call) {
      calls.add(call);
      return handler(call);
    });
  }

  /// Delivers a native-to-Dart call, as the platform sides do for token
  /// rotation.
  Future<void> nativeCall(MethodCall call) async {
    await messenger.handlePlatformMessage(
      'moth_push',
      const StandardMethodCodec().encodeMethodCall(call),
      (_) {},
    );
  }

  setUp(calls.clear);
  tearDown(() {
    messenger.setMockMethodCallHandler(MothNativePush.channel, null);
  });

  group('permissions', () {
    for (final (raw, mapped) in [
      ('granted', MothPushPermission.granted),
      ('provisional', MothPushPermission.provisional),
      ('denied', MothPushPermission.denied),
      ('unknown', MothPushPermission.unknown),
    ]) {
      test('requestPermission maps "$raw"', () async {
        host((call) async {
          expect(call.method, 'requestPermission');
          expect(call.arguments, {'provisional': false});
          return raw;
        });
        expect(await MothNativePush().requestPermission(), mapped);
      });

      test('permissionStatus maps "$raw"', () async {
        host((call) async {
          expect(call.method, 'permissionStatus');
          expect(call.arguments, isNull);
          return raw;
        });
        expect(await MothNativePush().permissionStatus(), mapped);
      });
    }

    test('an unexpected native state maps to unknown, never throws', () async {
      host((call) async => 'something-new');
      expect(
        await MothNativePush().permissionStatus(),
        MothPushPermission.unknown,
      );
    });

    test('the provisional flag reaches the native side', () async {
      host((call) async {
        expect(call.arguments, {'provisional': true});
        return 'provisional';
      });
      expect(
        await MothNativePush(provisional: true).requestPermission(),
        MothPushPermission.provisional,
      );
    });
  });

  group('getToken', () {
    test('returns the APNs credential RegisterDevice expects', () async {
      host((call) async {
        expect(call.method, 'getToken');
        return {'target': 'apns', 'token': 'a1b2c3'};
      });
      expect(
        await MothNativePush().getToken(),
        const MothPushToken(target: MothPushTarget.apns, token: 'a1b2c3'),
      );
    });

    test('returns the FCM credential RegisterDevice expects', () async {
      host((call) async => {'target': 'fcm', 'token': 'fcm-reg-token'});
      expect(
        await MothNativePush().getToken(),
        const MothPushToken(target: MothPushTarget.fcm, token: 'fcm-reg-token'),
      );
    });

    test('no credential yet resolves to null, not an error', () async {
      host((call) async => null);
      expect(await MothNativePush().getToken(), isNull);
    });

    test('an unknown target resolves to null, not an error', () async {
      host((call) async => {'target': 'carrier-pigeon', 'token': 'coo'});
      expect(await MothNativePush().getToken(), isNull);
    });

    test('missing Firebase config surfaces as the documented '
        'firebase-not-initialized PlatformException', () async {
      host((call) async {
        throw PlatformException(
          code: 'firebase-not-initialized',
          message:
              'Firebase is not initialized. moth_push uses Firebase '
              'Cloud Messaging on Android, which needs your app\'s own '
              'Firebase project: download google-services.json into '
              'android/app/ and apply the com.google.gms.google-services '
              'Gradle plugin, then rebuild. See the moth_push README.',
        );
      });
      await expectLater(
        MothNativePush().getToken(),
        throwsA(
          isA<PlatformException>()
              .having((e) => e.code, 'code', 'firebase-not-initialized')
              // The message is actionable, not a bare stack trace.
              .having(
                (e) => e.message,
                'message',
                contains('google-services.json'),
              ),
        ),
      );
    });
  });

  group('onTokenRefresh', () {
    test('native rotation callbacks feed the stream', () async {
      final push = MothNativePush();
      final received = <MothPushToken>[];
      final sub = push.onTokenRefresh.listen(received.add);
      await nativeCall(
        const MethodCall('onTokenRefresh', {
          'target': 'apns',
          'token': 'rotated-1',
        }),
      );
      await nativeCall(
        const MethodCall('onTokenRefresh', {
          'target': 'fcm',
          'token': 'rotated-2',
        }),
      );
      await null; // let the broadcast stream deliver
      expect(received, const [
        MothPushToken(target: MothPushTarget.apns, token: 'rotated-1'),
        MothPushToken(target: MothPushTarget.fcm, token: 'rotated-2'),
      ]);
      await sub.cancel();
      push.dispose();
    });

    test('a malformed rotation payload is dropped, not thrown', () async {
      final push = MothNativePush();
      final received = <MothPushToken>[];
      final sub = push.onTokenRefresh.listen(received.add);
      await nativeCall(const MethodCall('onTokenRefresh', {'token': ''}));
      await nativeCall(const MethodCall('onTokenRefresh'));
      await null;
      expect(received, isEmpty);
      await sub.cancel();
      push.dispose();
    });

    test('rotation after dispose is dropped safely', () async {
      final push = MothNativePush();
      push.dispose();
      await nativeCall(
        const MethodCall('onTokenRefresh', {'target': 'fcm', 'token': 'late'}),
      );
      // No throw is the assertion: next launch re-registers anyway.
    });
  });

  group('deviceMetadata', () {
    test('surfaces the native model and app version', () async {
      host((call) async {
        expect(call.method, 'deviceMetadata');
        return {'model': 'iPhone16,1', 'appVersion': '2.4.1+87'};
      });
      final metadata = await MothNativePush().deviceMetadata();
      expect(metadata.model, 'iPhone16,1');
      expect(metadata.appVersion, '2.4.1+87');
      // The SDK fills these from Dart; the plugin leaves them empty.
      expect(metadata.platform, isEmpty);
      expect(metadata.osVersion, isEmpty);
      expect(metadata.locale, isEmpty);
    });

    test('tolerates a sparse native reply', () async {
      host((call) async => <Object?, Object?>{});
      final metadata = await MothNativePush().deviceMetadata();
      expect(metadata.model, isEmpty);
      expect(metadata.appVersion, isEmpty);
    });
  });
}
