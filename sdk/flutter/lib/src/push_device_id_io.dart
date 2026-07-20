import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:path_provider/path_provider.dart';

import 'push_device_id.dart';

MothPushDeviceIdStore createPushDeviceIdStore(String publishableKey) =>
    MothFilePushDeviceIdStore(publishableKey: publishableKey);

/// File-backed [MothPushDeviceIdStore] under the application support
/// directory: `<support>/moth/<key-hash>/push_device_id`. The id is not a
/// secret (the server scopes registrations to the authenticated user), so a
/// plain file — the same layer as the other config caches — is the right
/// place; it lives outside the token store so sign-out keeps it.
class MothFilePushDeviceIdStore implements MothPushDeviceIdStore {
  MothFilePushDeviceIdStore({
    required String publishableKey,
    Future<Directory> Function()? baseDirectory,
  }) : _baseDirectory = baseDirectory ?? getApplicationSupportDirectory,
       _namespace = _hash(publishableKey);

  final Future<Directory> Function() _baseDirectory;
  final String _namespace;

  static String _hash(String s) =>
      sha256.convert(utf8.encode(s)).toString().substring(0, 16);

  Future<File> _file() async {
    final base = await _baseDirectory();
    final dir = await Directory(
      '${base.path}/moth/$_namespace',
    ).create(recursive: true);
    return File('${dir.path}/push_device_id');
  }

  @override
  Future<String?> load() async {
    final file = await _file();
    if (!await file.exists()) return null;
    final id = (await file.readAsString()).trim();
    return id.isEmpty ? null : id;
  }

  @override
  Future<void> save(String deviceId) async =>
      (await _file()).writeAsString(deviceId);
}
