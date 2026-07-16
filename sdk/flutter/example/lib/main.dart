// Example app for moth_auth: wraps the app in [MothApp], reads auth state
// from [MothScope] and calls a sample backend with the authenticated http
// client. See README.md for how to run it against a local moth instance.
import 'dart:io' show Platform;

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:moth_auth/moth_auth.dart';

import 'billing_adapter.dart';
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
  // The store billing adapter subscribes to the purchase stream for its
  // lifetime, so own it here rather than rebuilding it every frame.
  final _billingAdapter = ExampleBillingAdapter();

  @override
  void dispose() {
    _billingAdapter.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (mothPublishableKey.isEmpty) return const _MissingKeyScreen();
    return MothApp(
      config: MothConfig(
        endpoint: resolveLocalhost(Uri.parse(mothEndpoint)),
        publishableKey: mothPublishableKey,
      ),
      oauthAdapter: ExampleOAuthAdapter(),
      // Runs native store purchases for MothScope.purchase and the paywall.
      billingAdapter: _billingAdapter,
      // Signed out -> the SDK's default MothLoginScreen; signed in -> child.
      child: MaterialApp(
        title: 'moth example',
        theme: ThemeData(colorSchemeSeed: Colors.indigo),
        home: const HomeScreen(),
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
