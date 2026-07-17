import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';

import 'bundled_copy.dart';

/// The locales the moth SDK screens are localized for — the same curated set
/// the server bundles (English first). Spread into your app's
/// `MaterialApp.supportedLocales` so the framework resolves a matching locale
/// for the moth login and paywall screens.
final List<Locale> mothSupportedLocales = mothBundledLocales
    .map((code) => Locale(code))
    .toList(growable: false);

/// The `LocalizationsDelegate`s that resolve framework strings
/// (`MaterialLocalizations`, `WidgetsLocalizations`, `CupertinoLocalizations`)
/// for every [mothSupportedLocales] language, so moth's screens render dates,
/// tooltips and semantics in the device language without extra app setup.
///
/// [MothApp] installs these on the shell it wraps its own screens in. An app
/// with its own `MaterialApp` that wants the same coverage spreads them into
/// its `localizationsDelegates` alongside its own:
///
/// ```dart
/// MaterialApp(
///   localizationsDelegates: const [
///     ...mothLocalizationsDelegates,
///     MyAppLocalizations.delegate,
///   ],
///   supportedLocales: mothSupportedLocales,
/// );
/// ```
const List<LocalizationsDelegate<dynamic>> mothLocalizationsDelegates =
    <LocalizationsDelegate<dynamic>>[
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ];
