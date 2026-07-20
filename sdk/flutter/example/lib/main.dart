// Example app for moth_auth: wraps the app in [MothApp], reads auth state
// from [MothScope] and calls a sample backend with the authenticated http
// client. See README.md for how to run it against a local moth instance.
import 'dart:io' show Platform;

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_billing/moth_billing.dart';
import 'package:moth_push/moth_push.dart';

import 'home_screen.dart';
import 'oauth_adapter.dart';

/// Connection settings, injected at build time:
///
/// ```sh
/// flutter run \
///   --dart-define=MOTH_PUBLISHABLE_KEY=pk_... \
///   --dart-define=MOTH_ENDPOINT=http://localhost:8080 \
///   --dart-define=API_BASE=http://localhost:8081
/// ```
const mothEndpoint = String.fromEnvironment(
  'MOTH_ENDPOINT',
  defaultValue: 'http://localhost:8080',
);
const mothPublishableKey = String.fromEnvironment('MOTH_PUBLISHABLE_KEY');

/// Base URL of the sample backend (scripts/example_backend in the moth
/// repository) that verifies moth JWTs against the project JWKS.
const apiBase = String.fromEnvironment(
  'API_BASE',
  defaultValue: 'http://localhost:8081',
);

/// Rewrites localhost to the Android emulator's host alias: on the emulator
/// `localhost` is the device itself, `10.0.2.2` is the machine running moth.
Uri resolveLocalhost(Uri uri) {
  if (kIsWeb || !Platform.isAndroid) return uri;
  if (uri.host != 'localhost' && uri.host != '127.0.0.1') return uri;
  return uri.replace(host: '10.0.2.2');
}

void main() {
  runApp(const ExampleApp());
}

class ExampleApp extends StatefulWidget {
  const ExampleApp({super.key});

  @override
  State<ExampleApp> createState() => _ExampleAppState();
}

class _ExampleAppState extends State<ExampleApp> {
  // moth's first-party billing (StoreKit 2 / Play Billing) owns the native
  // method-channel handler for its lifetime (MothApp subscribes to its
  // transaction updates), so own it here rather than rebuilding it every
  // frame.
  final _billingAdapter = MothStoreBilling();

  // moth's first-party push registration (APNs / FCM). Wiring the adapter is
  // the whole opt-in: while signed in the SDK keeps the server's device
  // registry current; the OS permission prompt still only appears when the
  // app calls requestPushPermission() (see the notifications card on the
  // home screen). Fixed for the MothApp's lifetime.
  final _pushAdapter = MothNativePush();

  /// The language shown on the moth screens. `null` follows the device; the
  /// switcher below overrides it via [MothConfig.locale] so you can see the
  /// login and paywall screens re-localize live — the project's custom copy
  /// from a running instance when it has it, the SDK's bundled translations
  /// otherwise (and offline).
  Locale? _localeOverride;

  /// null = follow the device, then a short tour of the bundled locales.
  static const _localeCycle = <Locale?>[
    null,
    Locale('en'),
    Locale('fr'),
    Locale('de'),
    Locale('ja'),
  ];

  void _cycleLocale() {
    final next =
        (_localeCycle.indexOf(_localeOverride) + 1) % _localeCycle.length;
    setState(() => _localeOverride = _localeCycle[next]);
  }

  @override
  void dispose() {
    _billingAdapter.dispose();
    _pushAdapter.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (mothPublishableKey.isEmpty) return const _MissingKeyScreen();
    // A MaterialApp at the root supplies the moth localization delegates (so
    // MaterialLocalizations resolve for every bundled language) and lets the
    // language switcher overlay float above whatever MothApp shows.
    return MaterialApp(
      title: 'moth example',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(colorSchemeSeed: Colors.indigo),
      locale: _localeOverride,
      localizationsDelegates: mothLocalizationsDelegates,
      supportedLocales: mothSupportedLocales,
      builder: (context, child) => Stack(
        children: [
          ?child,
          _LanguageSwitcher(locale: _localeOverride, onPressed: _cycleLocale),
        ],
      ),
      home: MothApp(
        // Re-keyed on the locale so a switch rebuilds the client with the new
        // language; the moth screens then re-fetch and re-render localized.
        key: ValueKey(_localeOverride),
        config: MothConfig(
          endpoint: resolveLocalhost(Uri.parse(mothEndpoint)),
          publishableKey: mothPublishableKey,
          locale: _localeOverride,
          // Fills the {app} placeholder in the bundled copy (offline / before
          // the config loads); the server interpolates its own project name.
          appName: 'moth example',
        ),
        oauthAdapter: ExampleOAuthAdapter(),
        // moth_billing runs the native store purchase for MothScope.purchase
        // and the paywall; the server validates the resulting receipt.
        billingAdapter: _billingAdapter,
        // moth_push keeps the push-device registry current for the signed-in
        // user (register on launch and token rotation, unregister on
        // sign-out). No adapter, no push — nothing else changes.
        pushAdapter: _pushAdapter,
        // Signed out -> the SDK's default MothLoginScreen; signed in -> child.
        child: const HomeScreen(),
      ),
    );
  }
}

/// A floating pill that cycles the moth screens through the bundled languages,
/// so the demo shows the login and paywall re-localizing on the fly.
class _LanguageSwitcher extends StatelessWidget {
  const _LanguageSwitcher({required this.locale, required this.onPressed});

  final Locale? locale;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final label = switch (locale?.languageCode) {
      null => 'Device',
      'en' => 'English',
      'fr' => 'Français',
      'de' => 'Deutsch',
      'ja' => '日本語',
      final code => code,
    };
    return SafeArea(
      child: Align(
        alignment: Alignment.topRight,
        child: Padding(
          padding: const EdgeInsets.all(8),
          child: Material(
            color: Colors.black.withValues(alpha: 0.6),
            borderRadius: BorderRadius.circular(999),
            child: InkWell(
              borderRadius: BorderRadius.circular(999),
              onTap: onPressed,
              child: Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: 14,
                  vertical: 8,
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(Icons.language, size: 18, color: Colors.white),
                    const SizedBox(width: 6),
                    Text(label, style: const TextStyle(color: Colors.white)),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Shown instead of the app when no publishable key was provided, so a
/// plain `flutter run` explains what to do rather than failing opaquely.
class _MissingKeyScreen extends StatelessWidget {
  const _MissingKeyScreen();

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      home: Scaffold(
        appBar: AppBar(title: const Text('moth example')),
        body: const Padding(
          padding: EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'No publishable key',
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold),
              ),
              SizedBox(height: 12),
              Text(
                'Pass your project\'s publishable key when launching:\n\n'
                'flutter run --dart-define=MOTH_PUBLISHABLE_KEY=pk_...\n\n'
                'Create a project in the moth admin (http://localhost:8080'
                '/admin) and copy the key from its setup page. See '
                'README.md for the full walkthrough.',
              ),
            ],
          ),
        ),
      ),
    );
  }
}
