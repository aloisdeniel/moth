import 'dart:ui';

import 'gen/moth/auth/v1/config.pb.dart' as pb;
import 'i18n/bundled_copy.dart';
import 'locale.dart';

/// The resolved, localized copy for a locale: the message key → localized
/// string map the SDK's auth screens render from (`sign_in.*`, `sign_up.*`,
/// `paywall.*`), for the locale the server negotiated from the device's
/// language.
///
/// Resolution is **server override → bundled → English**: [value] returns the
/// server-delivered string for a key when present, otherwise the SDK's
/// [bundledCopy] for the locale, which itself fills in English for any key the
/// locale lacks. So a screen is always fully localized — instantly from the
/// bundle before the config loads and offline, refined by the project's own
/// wording once [MothCopyController] fetches it.
class MothCopy {
  const MothCopy({
    required this.locale,
    required this.revisionId,
    required this.messages,
    this.source,
  });

  /// The bundled-only floor for [locale]: no server messages yet, so [value]
  /// resolves straight from the bundled catalog (English fallback per key).
  /// The starting value before the first `GetProjectConfig`.
  const MothCopy.bundled(this.locale)
    : revisionId = '',
      messages = const <String, String>{},
      source = null;

  /// The negotiated locale this copy is for (from the server, echoed even
  /// when its `messages` were omitted).
  final Locale locale;

  /// Opaque cache token for this `(locale, override-revision)` pair; empty for
  /// a bundled-only floor. Echoed as `known_copy_revision` on the next fetch.
  final String revisionId;

  /// Server-delivered message key → localized string; empty for the bundled
  /// floor.
  final Map<String, String> messages;

  /// The wire message this copy was mapped from — the raw payload the
  /// on-device config cache persists, so cache and wire share one schema.
  /// Null for the bundled floor and hand-built copy. Derivation metadata
  /// only; not part of equality.
  final pb.Copy? source;

  static final RegExp _placeholder = RegExp(r'\{([a-zA-Z][a-zA-Z0-9_]*)\}');

  /// The localized string for [key], with any `{name}` placeholders replaced
  /// from [vars] (a literal `{name}` → value substitution, mirroring the
  /// server's placeholder contract — no pluralization). Falls back to the
  /// bundled catalog then English; an unknown key returns the key itself.
  String value(String key, {Map<String, String>? vars}) {
    var template = messages[key];
    if (template == null || template.isEmpty) {
      template = bundledCopy(locale)[key];
    }
    template ??= key;
    if (vars == null || vars.isEmpty) return template;
    return template.replaceAllMapped(_placeholder, (m) {
      final name = m[1]!;
      final replacement = vars[name];
      return replacement ?? m[0]!;
    });
  }

  Map<String, Object?> toJson() => <String, Object?>{
    'locale': mothLanguageTag(locale),
    'revision': revisionId,
    'messages': messages,
  };

  factory MothCopy.fromJson(Map<String, Object?> json) => MothCopy(
    locale: mothLocaleFromTag(json['locale'] as String),
    revisionId: json['revision'] as String,
    messages: (json['messages'] as Map).cast<String, String>(),
  );

  @override
  bool operator ==(Object other) =>
      other is MothCopy &&
      other.locale == locale &&
      other.revisionId == revisionId;

  @override
  int get hashCode => Object.hash(locale, revisionId);
}

/// The copy carried by a `GetProjectConfig` response: the negotiated locale
/// and revision are always present; [messages] is null when the server
/// confirmed the `knownCopyRevision` the client sent still matches (keep the
/// cached copy — stale-while-revalidate, like the theme).
class MothCopyUpdate {
  const MothCopyUpdate({
    required this.locale,
    required this.revisionId,
    this.messages,
    this.source,
  });

  final Locale locale;
  final String revisionId;

  /// Resolved key → string when the revision changed (or on first fetch); null
  /// when unchanged.
  final Map<String, String>? messages;

  /// The wire message this update was mapped from, handed through to the
  /// on-device config cache as its persisted payload.
  final pb.Copy? source;
}
