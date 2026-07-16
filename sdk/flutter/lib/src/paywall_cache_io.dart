import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:path_provider/path_provider.dart';

import 'offering.dart';
import 'paywall_cache.dart';

MothPaywallCache createPaywallCache(String publishableKey) =>
    MothFilePaywallCache(publishableKey: publishableKey);

/// File-backed [MothPaywallCache] under the application support directory:
/// `<support>/moth/<key-hash>/paywall.json`. The paywall config is not secret
/// (the server re-delivers it), so plain files — not secure storage — are the
/// right place.
class MothFilePaywallCache implements MothPaywallCache {
  MothFilePaywallCache({
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
    return File('${dir.path}/paywall.json');
  }

  @override
  Future<MothPaywall?> load() async {
    final file = await _file();
    if (!await file.exists()) return null;
    return MothPaywall.fromJson(
      jsonDecode(await file.readAsString()) as Map<String, Object?>,
    );
  }

  @override
  Future<void> save(MothPaywall paywall) async =>
      (await _file()).writeAsString(jsonEncode(paywall.toJson()));
}
