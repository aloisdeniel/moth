import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:crypto/crypto.dart';
import 'package:fixnum/fixnum.dart';
import 'package:path_provider/path_provider.dart';

import 'bootstrap.dart';
import 'gen/moth/auth/v1/config.pb.dart' as pb;
import 'gen/moth/projectconfig/v1/projectconfig.pb.dart' as storagepb;
import 'theme.dart';
import 'theme_cache.dart';

MothThemeCache createThemeCache(String publishableKey) =>
    MothFileThemeCache(publishableKey: publishableKey);

/// File-backed [MothThemeCache] under the application support directory:
/// `<support>/moth/<key-hash>/theme.pb` — a `moth.projectconfig.v1.CacheEnvelope`
/// wrapping the raw `moth.auth.v1.Theme` wire message — plus one
/// `font_<url-hash>` file per downloaded font. Nothing here is secret — the
/// theme is public project configuration — so plain files (not secure
/// storage) are the right place.
///
/// A legacy JSON cache file from a pre-protobuf SDK (`theme.json`), like
/// any unparseable file, is treated as a cache miss and deleted best-effort
/// — the theme is simply refetched.
class MothFileThemeCache implements MothThemeCache {
  MothFileThemeCache({
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

  Future<File> _themeFile() async => File('${(await _dir()).path}/theme.pb');

  Future<File> _fontFile(String url) async =>
      File('${(await _dir()).path}/font_${_hash(url)}');

  @override
  Future<MothCachedTheme?> loadTheme() async {
    final dir = await _dir();
    // Best-effort cleanup of the pre-protobuf JSON cache; its content is
    // not migrated (the theme is refetched on the next revalidation).
    await _deleteBestEffort(File('${dir.path}/theme.json'));
    final file = File('${dir.path}/theme.pb');
    // Cold cache: seed from the package's baked-in theme (a server-generated
    // build), so the login screen renders branded on the very first launch
    // with no network round-trip. Null for the canonical package.
    if (!await file.exists()) return MothBootstrap.instance?.seededTheme;
    try {
      final envelope = storagepb.CacheEnvelope.fromBuffer(
        await file.readAsBytes(),
      );
      if (!envelope.hasPayload()) throw const FormatException('no payload');
      return MothCachedTheme(
        theme: MothTheme.fromProto(pb.Theme.fromBuffer(envelope.payload)),
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
  Future<void> saveTheme(MothTheme theme, {required DateTime fetchedAt}) async {
    // The payload is the raw wire message exactly as the server delivered
    // it; a hand-built theme (no source) is re-encoded from the model.
    final wire = theme.source ?? _themeToProto(theme);
    final envelope = storagepb.CacheEnvelope(
      payload: wire.writeToBuffer(),
      revision: theme.revisionId,
      fetchedAtUnixMs: Int64(fetchedAt.millisecondsSinceEpoch),
    );
    await (await _themeFile()).writeAsBytes(envelope.writeToBuffer());
  }

  @override
  Future<void> touchTheme(DateTime fetchedAt) async {
    final file = await _themeFile();
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

  @override
  Future<Uint8List?> loadFontBytes(String url) async {
    final file = await _fontFile(url);
    if (!await file.exists()) return null;
    return file.readAsBytes();
  }

  @override
  Future<void> saveFontBytes(String url, Uint8List bytes) async =>
      (await _fontFile(url)).writeAsBytes(bytes);
}

Future<void> _deleteBestEffort(File file) async {
  try {
    if (await file.exists()) await file.delete();
  } on Object {
    // Best effort only.
  }
}

/// Re-encodes a hand-built [MothTheme] (no retained wire message) into the
/// wire schema, so every saved theme round-trips through the same envelope.
pb.Theme _themeToProto(MothTheme theme) => pb.Theme(
  revisionId: theme.revisionId,
  colors: _colorsToProto(theme.colors),
  darkColors: _colorsToProto(theme.darkColors),
  fontFamily: theme.fontFamily,
  fontUrl: theme.fontUrl ?? '',
  fontScale: theme.fontScale,
  spacingUnit: theme.spacingUnit.round(),
  cornerRadius: theme.cornerRadius.round(),
  logoLightUrl: theme.logoLightUrl ?? '',
  logoDarkUrl: theme.logoDarkUrl ?? '',
  termsUrl: theme.termsUrl ?? '',
  privacyUrl: theme.privacyUrl ?? '',
);

pb.ThemeColors _colorsToProto(MothThemeColors colors) => pb.ThemeColors(
  primary: mothHexColor(colors.primary),
  onPrimary: mothHexColor(colors.onPrimary),
  background: mothHexColor(colors.background),
  onBackground: mothHexColor(colors.onBackground),
  surface: mothHexColor(colors.surface),
  onSurface: mothHexColor(colors.onSurface),
  error: mothHexColor(colors.error),
  onError: mothHexColor(colors.onError),
);
