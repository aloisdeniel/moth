import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'user.dart';

/// One persisted session: the token pair plus a snapshot of the user, so a
/// restored session can render without a network round-trip.
class StoredSession {
  const StoredSession({
    required this.accessToken,
    required this.refreshToken,
    required this.expiresAt,
    required this.user,
  });

  factory StoredSession.fromJson(Map<String, Object?> json) => StoredSession(
    accessToken: json['access_token'] as String,
    refreshToken: json['refresh_token'] as String,
    expiresAt: DateTime.parse(json['expires_at'] as String),
    user: MothUser.fromJson(json['user']! as Map<String, Object?>),
  );

  final String accessToken;
  final String refreshToken;

  /// When the access token expires (computed from `expires_in` at receipt).
  final DateTime expiresAt;

  final MothUser user;

  StoredSession copyWith({MothUser? user}) => StoredSession(
    accessToken: accessToken,
    refreshToken: refreshToken,
    expiresAt: expiresAt,
    user: user ?? this.user,
  );

  Map<String, Object?> toJson() => {
    'access_token': accessToken,
    'refresh_token': refreshToken,
    'expires_at': expiresAt.toUtc().toIso8601String(),
    'user': user.toJson(),
  };
}

/// Where [MothClient] persists the session. The default is
/// [SecureTokenStore]; swap in an [InMemoryTokenStore] for tests or
/// ephemeral sessions.
abstract class TokenStore {
  Future<StoredSession?> load();
  Future<void> save(StoredSession session);
  Future<void> clear();
}

/// Keeps the session in memory only — nothing survives a restart.
class InMemoryTokenStore implements TokenStore {
  StoredSession? _session;

  @override
  Future<StoredSession?> load() async => _session;

  @override
  Future<void> save(StoredSession session) async => _session = session;

  @override
  Future<void> clear() async => _session = null;
}

/// Persists the session in the platform's secure storage via
/// `flutter_secure_storage` (Keychain on iOS/macOS, Keystore-backed
/// encrypted storage on Android). Entries are namespaced by publishable key
/// so two projects on one device never collide.
class SecureTokenStore implements TokenStore {
  SecureTokenStore({
    required String publishableKey,
    this._storage = const FlutterSecureStorage(),
  }) : _key = 'moth_session_$publishableKey';

  final FlutterSecureStorage _storage;
  final String _key;

  @override
  Future<StoredSession?> load() async {
    final String? raw;
    try {
      raw = await _storage.read(key: _key);
    } on Object {
      // Secure storage itself failed — Keystore entry invalidated after a
      // backup/restore or OS reinstall, Keychain locked before first
      // unlock, ... Treat as signed out instead of crashing startup; the
      // entry is kept because a locked-Keychain read can succeed later.
      return null;
    }
    if (raw == null) return null;
    try {
      return StoredSession.fromJson(jsonDecode(raw) as Map<String, Object?>);
    } on Object {
      // Unreadable entry (corruption, format change): treat as signed out.
      await _deleteEntry();
      return null;
    }
  }

  @override
  Future<void> save(StoredSession session) =>
      _storage.write(key: _key, value: jsonEncode(session.toJson()));

  @override
  Future<void> clear() => _storage.delete(key: _key);

  Future<void> _deleteEntry() async {
    try {
      await _storage.delete(key: _key);
    } on Object {
      // Best effort — the entry was unreadable anyway.
    }
  }
}
