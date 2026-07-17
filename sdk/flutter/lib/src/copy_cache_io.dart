import 'dart:convert';
import 'dart:io';
import 'dart:ui';

import 'package:crypto/crypto.dart';
import 'package:path_provider/path_provider.dart';

import 'copy.dart';
import 'copy_cache.dart';

MothCopyCache createCopyCache(String publishableKey) =>
    MothFileCopyCache(publishableKey: publishableKey);

/// File-backed [MothCopyCache] under the application support directory:
/// `<support>/moth/<key-hash>/copy_<locale-hash>.json`, one file per locale
/// (the JSON carries the `(locale, revision)` pair). The copy is not secret
/// (the server re-delivers it), so plain files — not secure storage — are the
/// right place.
class MothFileCopyCache implements MothCopyCache {
  MothFileCopyCache({
    required String publishableKey,
    Future<Directory> Function()? baseDirectory,
  }) : _baseDirectory = baseDirectory ?? getApplicationSupportDirectory,
       _namespace = _hash(publishableKey);

  final Future<Directory> Function() _baseDirectory;
  final String _namespace;

  static String _hash(String s) =>
      sha256.convert(utf8.encode(s)).toString().substring(0, 16);

  // Keyed by language only: the load key (device/override locale) may carry a
  // region while the save key (server-negotiated locale) is language-only, so
  // hashing the languageCode keeps them the same file (see MothCopyCache).
  Future<File> _file(Locale locale) async {
    final base = await _baseDirectory();
    final dir = await Directory(
      '${base.path}/moth/$_namespace',
    ).create(recursive: true);
    return File('${dir.path}/copy_${_hash(locale.languageCode)}.json');
  }

  @override
  Future<MothCopy?> load(Locale locale) async {
    final file = await _file(locale);
    if (!await file.exists()) return null;
    return MothCopy.fromJson(
      jsonDecode(await file.readAsString()) as Map<String, Object?>,
    );
  }

  @override
  Future<void> save(MothCopy copy) async =>
      (await _file(copy.locale)).writeAsString(jsonEncode(copy.toJson()));
}
