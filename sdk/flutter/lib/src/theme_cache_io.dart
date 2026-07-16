import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:crypto/crypto.dart';
import 'package:path_provider/path_provider.dart';

import 'theme.dart';
import 'theme_cache.dart';

MothThemeCache createThemeCache(String publishableKey) =>
    MothFileThemeCache(publishableKey: publishableKey);

/// File-backed [MothThemeCache] under the application support directory:
/// `<support>/moth/<key-hash>/theme.json` plus one `font_<url-hash>` file
/// per downloaded font. Nothing here is secret — the theme is public
/// project configuration — so plain files (not secure storage) are the
/// right place.
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

  Future<File> _themeFile() async => File('${(await _dir()).path}/theme.json');

  Future<File> _fontFile(String url) async =>
      File('${(await _dir()).path}/font_${_hash(url)}');

  @override
  Future<MothTheme?> loadTheme() async {
    final file = await _themeFile();
    if (!await file.exists()) return null;
    return MothTheme.fromJson(
      jsonDecode(await file.readAsString()) as Map<String, Object?>,
    );
  }

  @override
  Future<void> saveTheme(MothTheme theme) async =>
      (await _themeFile()).writeAsString(jsonEncode(theme.toJson()));

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
