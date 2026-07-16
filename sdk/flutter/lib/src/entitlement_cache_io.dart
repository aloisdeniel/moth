import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:path_provider/path_provider.dart';

import 'customer_info.dart';
import 'entitlement_cache.dart';

MothEntitlementCache createEntitlementCache(String publishableKey) =>
    MothFileEntitlementCache(publishableKey: publishableKey);

/// File-backed [MothEntitlementCache] under the application support directory:
/// `<support>/moth/<key-hash>/entitlements_<user-hash>.json`. Entitlement
/// state is not secret (the server re-validates it), so plain files — not
/// secure storage — are the right place.
class MothFileEntitlementCache implements MothEntitlementCache {
  MothFileEntitlementCache({
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

  Future<File> _file(String userId) async =>
      File('${(await _dir()).path}/entitlements_${_hash(userId)}.json');

  @override
  Future<MothCustomerInfo?> load(String userId) async {
    final file = await _file(userId);
    if (!await file.exists()) return null;
    return MothCustomerInfo.fromJson(
      jsonDecode(await file.readAsString()) as Map<String, Object?>,
    );
  }

  @override
  Future<void> save(String userId, MothCustomerInfo info) async =>
      (await _file(userId)).writeAsString(jsonEncode(info.toJson()));
}
