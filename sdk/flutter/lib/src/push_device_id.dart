import 'dart:math';

import 'push_device_id_stub.dart'
    if (dart.library.io) 'push_device_id_io.dart'
    as impl;

/// Where the SDK persists this installation's stable push `device_id` — a
/// client-generated UUID sent with every `RegisterDevice` call so one
/// physical device re-registering replaces its own row instead of
/// accumulating.
///
/// The id identifies the **installation, not the user**: it survives
/// sign-out on purpose (the server's newest-owner-wins upsert makes reuse
/// across accounts safe, and a returning user then supersedes their own old
/// row instead of creating a duplicate). It is not a secret — the server
/// scopes every registration to the authenticated user. All methods may
/// throw (broken storage); callers treat failures as a miss and fall back to
/// a fresh id.
abstract class MothPushDeviceIdStore {
  /// The persisted id, or null when none was generated yet.
  Future<String?> load();
  Future<void> save(String deviceId);
}

/// The platform default store, namespaced by publishable key so two projects
/// on one device never collide.
MothPushDeviceIdStore defaultPushDeviceIdStore(String publishableKey) =>
    impl.createPushDeviceIdStore(publishableKey);

/// Keeps the id in memory only — nothing survives a restart. The default on
/// Flutter Web, and handy in tests.
class MothMemoryPushDeviceIdStore implements MothPushDeviceIdStore {
  String? _deviceId;

  @override
  Future<String?> load() async => _deviceId;

  @override
  Future<void> save(String deviceId) async => _deviceId = deviceId;
}

/// A random version-4 UUID for a fresh installation id.
String generatePushDeviceId() {
  final random = Random.secure();
  final bytes = List<int>.generate(16, (_) => random.nextInt(256));
  bytes[6] = (bytes[6] & 0x0f) | 0x40; // version 4
  bytes[8] = (bytes[8] & 0x3f) | 0x80; // RFC 4122 variant
  final hex = bytes.map((b) => b.toRadixString(16).padLeft(2, '0')).join();
  return '${hex.substring(0, 8)}-${hex.substring(8, 12)}-'
      '${hex.substring(12, 16)}-${hex.substring(16, 20)}-${hex.substring(20)}';
}
