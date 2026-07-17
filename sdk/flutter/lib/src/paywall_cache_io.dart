import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:fixnum/fixnum.dart';
import 'package:path_provider/path_provider.dart';

import 'gen/moth/billing/v1/billing.pb.dart' as pb;
import 'gen/moth/storage/v1/storage.pb.dart' as storagepb;
import 'offering.dart';
import 'paywall_cache.dart';

MothPaywallCache createPaywallCache(String publishableKey) =>
    MothFilePaywallCache(publishableKey: publishableKey);

/// File-backed [MothPaywallCache] under the application support directory:
/// `<support>/moth/<key-hash>/paywall.pb` — a `moth.storage.v1.CacheEnvelope`
/// wrapping the raw `moth.billing.v1.Paywall` wire message. The paywall
/// config is not secret (the server re-delivers it), so plain files — not
/// secure storage — are the right place.
///
/// A legacy JSON cache file from a pre-protobuf SDK (`paywall.json`), like
/// any unparseable file, is treated as a cache miss and deleted best-effort
/// — the config is simply refetched.
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

  Future<Directory> _dir() async {
    final base = await _baseDirectory();
    return Directory('${base.path}/moth/$_namespace').create(recursive: true);
  }

  Future<File> _file() async => File('${(await _dir()).path}/paywall.pb');

  @override
  Future<MothCachedPaywall?> load() async {
    final dir = await _dir();
    // Best-effort cleanup of the pre-protobuf JSON cache; its content is
    // not migrated (the config is refetched on the next revalidation).
    await _deleteBestEffort(File('${dir.path}/paywall.json'));
    final file = File('${dir.path}/paywall.pb');
    if (!await file.exists()) return null;
    try {
      final envelope = storagepb.CacheEnvelope.fromBuffer(
        await file.readAsBytes(),
      );
      if (!envelope.hasPayload()) throw const FormatException('no payload');
      return MothCachedPaywall(
        paywall: MothPaywall.fromProto(pb.Paywall.fromBuffer(envelope.payload)),
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
  Future<void> save(MothPaywall paywall, {required DateTime fetchedAt}) async {
    // The payload is the raw wire message exactly as the server delivered
    // it; a hand-built config (no source) is re-encoded from the model.
    final wire = paywall.source ?? _paywallToProto(paywall);
    final envelope = storagepb.CacheEnvelope(
      payload: wire.writeToBuffer(),
      revision: paywall.revisionId,
      fetchedAtUnixMs: Int64(fetchedAt.millisecondsSinceEpoch),
    );
    await (await _file()).writeAsBytes(envelope.writeToBuffer());
  }

  @override
  Future<void> touch(DateTime fetchedAt) async {
    final file = await _file();
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

/// Re-encodes a hand-built [MothPaywall] (no retained wire message) into
/// the wire schema, so every saved config round-trips through the same
/// envelope.
pb.Paywall _paywallToProto(MothPaywall paywall) => pb.Paywall(
  revisionId: paywall.revisionId,
  headline: paywall.headline,
  subtitle: paywall.subtitle,
  benefits: paywall.benefits,
  offering: paywall.offering,
  highlightedProductIdentifier: paywall.highlightedProductIdentifier,
  layout: switch (paywall.layout) {
    MothPaywallLayout.list => pb.PaywallLayout.PAYWALL_LAYOUT_LIST,
    MothPaywallLayout.compact => pb.PaywallLayout.PAYWALL_LAYOUT_COMPACT,
    MothPaywallLayout.tiles => pb.PaywallLayout.PAYWALL_LAYOUT_TILES,
  },
  termsUrl: paywall.termsUrl ?? '',
  privacyUrl: paywall.privacyUrl ?? '',
);
