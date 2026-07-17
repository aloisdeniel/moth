import 'dart:convert';
import 'dart:io';
import 'dart:ui';

import 'package:crypto/crypto.dart';
import 'package:fixnum/fixnum.dart';
import 'package:path_provider/path_provider.dart';

import 'copy.dart';
import 'copy_cache.dart';
import 'gen/moth/auth/v1/config.pb.dart' as pb;
import 'gen/moth/storage/v1/storage.pb.dart' as storagepb;
import 'locale.dart';

MothCopyCache createCopyCache(String publishableKey) =>
    MothFileCopyCache(publishableKey: publishableKey);

/// File-backed [MothCopyCache] under the application support directory:
/// `<support>/moth/<key-hash>/copy_<locale-hash>.pb`, one file per locale —
/// each a `moth.storage.v1.CacheEnvelope` wrapping the raw
/// `moth.auth.v1.Copy` wire message, with the envelope's `locale` set to the
/// negotiated tag. The copy is not secret (the server re-delivers it), so
/// plain files — not secure storage — are the right place.
///
/// A legacy JSON cache file from a pre-protobuf SDK
/// (`copy_<locale-hash>.json`), like any unparseable file, is treated as a
/// cache miss and deleted best-effort — the copy is simply refetched.
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

  Future<Directory> _dir() async {
    final base = await _baseDirectory();
    return Directory('${base.path}/moth/$_namespace').create(recursive: true);
  }

  // Keyed by language only: the load key (device/override locale) may carry a
  // region while the save key (server-negotiated locale) is language-only, so
  // hashing the languageCode keeps them the same file (see MothCopyCache).
  Future<File> _file(Locale locale) async =>
      File('${(await _dir()).path}/copy_${_hash(locale.languageCode)}.pb');

  @override
  Future<MothCachedCopy?> load(Locale locale) async {
    final dir = await _dir();
    final hash = _hash(locale.languageCode);
    // Best-effort cleanup of the pre-protobuf JSON cache; its content is
    // not migrated (the copy is refetched on the next revalidation).
    await _deleteBestEffort(File('${dir.path}/copy_$hash.json'));
    final file = File('${dir.path}/copy_$hash.pb');
    if (!await file.exists()) return null;
    try {
      final envelope = storagepb.CacheEnvelope.fromBuffer(
        await file.readAsBytes(),
      );
      if (!envelope.hasPayload()) throw const FormatException('no payload');
      final wire = pb.Copy.fromBuffer(envelope.payload);
      return MothCachedCopy(
        copy: MothCopy(
          locale: mothLocaleFromTag(wire.locale),
          revisionId: wire.copyRevision,
          messages: Map<String, String>.of(wire.messages),
          source: wire,
        ),
        fetchedAt: DateTime.fromMillisecondsSinceEpoch(
          envelope.fetchedAtUnixMs.toInt(),
          isUtc: true,
        ),
      );
    } on Object {
      // Corrupt or foreign content (e.g. a legacy format): a cache miss,
      // never a crash. Drop the file so the next save starts clean.
      await _deleteBestEffort(file);
      return null;
    }
  }

  @override
  Future<void> save(MothCopy copy, {required DateTime fetchedAt}) async {
    // The payload is the raw wire message exactly as the server delivered
    // it; hand-built copy (no source) is re-encoded from the model.
    final wire =
        copy.source ??
        pb.Copy(
          copyRevision: copy.revisionId,
          locale: mothLanguageTag(copy.locale),
          messages: copy.messages.entries,
        );
    final envelope = storagepb.CacheEnvelope(
      payload: wire.writeToBuffer(),
      revision: copy.revisionId,
      locale: mothLanguageTag(copy.locale),
      fetchedAtUnixMs: Int64(fetchedAt.millisecondsSinceEpoch),
    );
    await (await _file(copy.locale)).writeAsBytes(envelope.writeToBuffer());
  }

  @override
  Future<void> touch(Locale locale, DateTime fetchedAt) async {
    final file = await _file(locale);
    if (!await file.exists()) return;
    try {
      final envelope = storagepb.CacheEnvelope.fromBuffer(
        await file.readAsBytes(),
      );
      envelope.fetchedAtUnixMs = Int64(fetchedAt.millisecondsSinceEpoch);
      await file.writeAsBytes(envelope.writeToBuffer());
    } on Object {
      await _deleteBestEffort(file);
    }
  }
}

Future<void> _deleteBestEffort(File file) async {
  try {
    if (await file.exists()) await file.delete();
  } on Object {
    // Best effort only.
  }
}
