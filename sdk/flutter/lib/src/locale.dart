import 'dart:ui';

/// The device's current locale, read from the engine's
/// [PlatformDispatcher] (the same source [WidgetsApp] resolves from). An app
/// pins a fixed language by passing `MothConfig(locale: ...)`, which wins over
/// this.
Locale mothDeviceLocale() => PlatformDispatcher.instance.locale;

/// The BCP-47 language tag for [locale] (e.g. `fr` or `fr-CA`), sent as the
/// `x-moth-language` metadata header so the server negotiates the copy locale.
String mothLanguageTag(Locale locale) => locale.toLanguageTag();

/// Parses a BCP-47 tag (`fr`, `fr-CA`, `zh-Hant`, `zh_Hant_TW`) back into a
/// [Locale]. Tolerates both `-` and `_` separators; an empty or malformed tag
/// falls back to English.
Locale mothLocaleFromTag(String tag) {
  final parts = tag.split(RegExp('[-_]')).where((p) => p.isNotEmpty).toList();
  if (parts.isEmpty) return const Locale('en');
  final language = parts[0];
  if (parts.length >= 2 && parts[1].length == 4) {
    // Second subtag is a script (e.g. Hant); a region may follow.
    return Locale.fromSubtags(
      languageCode: language,
      scriptCode: parts[1],
      countryCode: parts.length >= 3 ? parts[2] : null,
    );
  }
  return Locale(language, parts.length >= 2 ? parts[1] : null);
}
