import 'package:flutter/services.dart' show PlatformException;
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';

/// Fakes the platform secure storage without touching a method channel.
/// [readResult] is returned by read(); [throwOnRead]/[throwOnDelete] mimic
/// the PlatformExceptions flutter_secure_storage surfaces when the Android
/// Keystore entry was invalidated or the iOS Keychain is locked.
class _FakeSecureStorage extends FlutterSecureStorage {
  _FakeSecureStorage({this.readResult, this.throwOnRead = false});

  final String? readResult;
  final bool throwOnRead;
  bool throwOnDelete = false;
  int deletes = 0;

  @override
  Future<String?> read({
    required String key,
    IOSOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WebOptions? webOptions,
    MacOsOptions? mOptions,
    WindowsOptions? wOptions,
  }) async {
    if (throwOnRead) {
      throw PlatformException(code: 'BadPaddingException');
    }
    return readResult;
  }

  @override
  Future<void> delete({
    required String key,
    IOSOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WebOptions? webOptions,
    MacOsOptions? mOptions,
    WindowsOptions? wOptions,
  }) async {
    deletes++;
    if (throwOnDelete) {
      throw PlatformException(code: 'errSecInteractionNotAllowed');
    }
  }
}

void main() {
  group('SecureTokenStore.load', () {
    test(
      'a throwing platform read restores to signed out, keeping the entry',
      () async {
        final storage = _FakeSecureStorage(throwOnRead: true);
        final store = SecureTokenStore(
          publishableKey: 'pk_test',
          storage: storage,
        );

        expect(await store.load(), isNull);
        // The entry may become readable again (e.g. Keychain unlock): kept.
        expect(storage.deletes, 0);
      },
    );

    test('an unreadable entry is deleted and treated as signed out', () async {
      final storage = _FakeSecureStorage(readResult: 'not json');
      final store = SecureTokenStore(
        publishableKey: 'pk_test',
        storage: storage,
      );

      expect(await store.load(), isNull);
      expect(storage.deletes, 1);
    });

    test('a failing delete of an unreadable entry is swallowed', () async {
      final storage = _FakeSecureStorage(readResult: '{"bad": true}')
        ..throwOnDelete = true;
      final store = SecureTokenStore(
        publishableKey: 'pk_test',
        storage: storage,
      );

      expect(await store.load(), isNull);
    });
  });
}
