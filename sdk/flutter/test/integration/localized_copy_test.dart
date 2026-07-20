// End-to-end localization acceptance against a real moth binary (milestone
// 16): spawns `bin/moth serve` on a throwaway data dir, creates an admin +
// project over the connect JSON endpoints, writes a French copy override for
// the sign-in and paywall screens via the admin CopyService, then drives the
// SDK's MothClient with `x-moth-language: fr` and asserts:
//   * GetProjectConfig negotiates `fr` and returns the project's French copy
//     (proving the SDK actually put x-moth-language on the wire — an English
//     control client with no locale gets the English copy);
//   * GetPaywall returns the French paywall copy the paywall screen renders.
//
// Excluded from the default `flutter test` run (see dart_test.yaml); run via
// `make sdk-e2e` after `make build`.
@Tags(['integration'])
library;

import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:moth_auth/moth_auth.dart';

const adminEmail = 'admin@example.com';
const adminPassword = 'admin-password-1';

// Custom French wording, deliberately distinct from the SDK's bundled French
// floor, so a match proves the project override (the "server ceiling") landed,
// not merely that `fr` was negotiated.
const frSignInTitle = 'Bienvenue à nouveau';
const frSignInSubmit = 'Se connecter maintenant';
const frPaywallCta = 'Souscrire maintenant';

void main() {
  final binary = _findMothBinary();

  Directory? dataDir;
  Process? server;
  final serverLog = StringBuffer();
  late Uri base;

  setUpAll(() async {
    if (binary == null) return;
    dataDir = await Directory.systemTemp.createTemp('moth-sdk-i18n-e2e-');

    final create = await Process.run(binary, [
      'admin',
      'create',
      '--email',
      adminEmail,
      '--password',
      adminPassword,
      '--data-dir',
      dataDir!.path,
    ]);
    expect(create.exitCode, 0, reason: '${create.stdout}\n${create.stderr}');

    for (var attempt = 1; ; attempt++) {
      final port = await _freePort();
      base = Uri.parse('http://127.0.0.1:$port');
      final candidate = await Process.start(binary, [
        'serve',
        '--addr',
        '127.0.0.1:$port',
        '--data-dir',
        dataDir!.path,
        '--base-url',
        '$base',
      ]);
      candidate.stdout.transform(utf8.decoder).listen(serverLog.write);
      candidate.stderr.transform(utf8.decoder).listen(serverLog.write);
      if (await _becomesHealthy(base, candidate)) {
        server = candidate;
        break;
      }
      candidate.kill();
      await candidate.exitCode;
      if (attempt == 3) {
        fail('moth server did not become healthy at $base:\n$serverLog');
      }
    }
  });

  tearDownAll(() async {
    server?.kill();
    await server?.exitCode;
    await dataDir?.delete(recursive: true);
    printOnFailure('moth server log:\n$serverLog');
  });

  test('fr negotiation + project copy overrides render on device', () async {
    if (binary == null) {
      markTestSkipped('bin/moth not found — run `make build` first');
      return;
    }

    // -------------------------------------------------- admin: project + copy
    final cookie = await _adminLogin(base);
    final project = await _rpc(
      base,
      '/moth.admin.v1.ProjectService/CreateProject',
      {'name': 'sdk i18n e2e'},
      cookie: cookie,
    );
    final proj = project['project']! as Map<String, Object?>;
    final projectId = proj['id'] as String;
    final publishableKey = proj['publishableKey'] as String;
    expect(publishableKey, startsWith('pk_'));

    // Write a French override for the two SDK screens under test.
    final saved = await _rpc(
      base,
      '/moth.admin.v1.CopyService/UpdateProjectCopy',
      {
        'projectId': projectId,
        'locale': 'fr',
        'values': {
          'sign_in.title': frSignInTitle,
          'sign_in.submit': frSignInSubmit,
          'paywall.cta': frPaywallCta,
        },
      },
      cookie: cookie,
    );
    expect(
      (saved['revisionId'] as String?) ?? '',
      isNotEmpty,
      reason: 'the override save creates a revision',
    );

    // ------------------------------------------------------ SDK: French client
    // MothConfig(locale: fr) makes the client attach `x-moth-language: fr` on
    // every call via its interceptor.
    final frClient = MothClient(
      MothConfig(
        endpoint: base,
        publishableKey: publishableKey,
        locale: const Locale('fr'),
      ),
    );
    addTearDown(frClient.dispose);
    expect(frClient.currentLocale, const Locale('fr'));

    final frConfig = await frClient.getProjectConfig();
    final frCopy = frConfig.copy;
    expect(frCopy, isNotNull, reason: 'config carries the negotiated copy');
    expect(
      frCopy!.locale,
      const Locale('fr'),
      reason: 'server negotiated fr from x-moth-language',
    );
    expect(frCopy.revisionId, startsWith('fr|'));
    expect(frCopy.messages, isNotNull);
    expect(frCopy.messages!['sign_in.title'], frSignInTitle);
    expect(frCopy.messages!['sign_in.submit'], frSignInSubmit);
    // A key with no override still falls back to the bundled French default.
    expect(frCopy.messages!['sign_in.email_label'], 'E-mail');

    // What the SDK's MothCopy would actually resolve at render time.
    final rendered = MothCopy(
      locale: frCopy.locale,
      revisionId: frCopy.revisionId,
      messages: frCopy.messages!,
    );
    expect(rendered.value('sign_in.title'), frSignInTitle);
    expect(rendered.value('sign_in.submit'), frSignInSubmit);

    // ------------------------------------------------------ English control
    // No locale override → device locale (whatever the host is) with no
    // Accept-Language forwarded by the SDK falls to the project default
    // (English). This is the negative control proving the fr result above came
    // from the header, not from a fixed server default.
    final enClient = MothClient(
      MothConfig(
        endpoint: base,
        publishableKey: publishableKey,
        locale: const Locale('en'),
      ),
    );
    addTearDown(enClient.dispose);
    final enConfig = await enClient.getProjectConfig();
    expect(enConfig.copy!.locale, const Locale('en'));
    expect(enConfig.copy!.messages!['sign_in.title'], 'Sign in');
    expect(
      enConfig.copy!.messages!['sign_in.title'],
      isNot(frSignInTitle),
      reason: 'English client must not see the French override',
    );

    // ------------------------------------------------------ paywall copy (fr)
    // The SDK client's getPaywall drops the Copy body, so assert the French
    // paywall copy the paywall screen renders at the RPC level: the very
    // GetPaywall response the SDK issues with x-moth-language: fr.
    final paywall = await frClient.getPaywall();
    expect(paywall, isNotNull, reason: 'default paywall config is returned');

    final paywallResp = await _rpc(
      base,
      '/moth.billing.v1.BillingService/GetPaywall',
      const <String, Object?>{},
      headers: {'x-moth-language': 'fr', 'x-moth-key': publishableKey},
    );
    final pwCopy = paywallResp['copy']! as Map<String, Object?>;
    expect(pwCopy['locale'], 'fr');
    final pwMessages = (pwCopy['messages']! as Map<String, Object?>)
        .cast<String, String>();
    expect(pwMessages['paywall.cta'], frPaywallCta);
    // Unoverridden paywall key falls back to the bundled French default.
    expect(pwMessages['paywall.restore'], 'Restaurer les achats');
  });
}

String? _findMothBinary() {
  var dir = Directory.current;
  for (var i = 0; i < 5; i++) {
    final candidate = File('${dir.path}/bin/moth');
    if (candidate.existsSync()) return candidate.path;
    final parent = dir.parent;
    if (parent.path == dir.path) break;
    dir = parent;
  }
  return null;
}

Future<int> _freePort() async {
  final socket = await ServerSocket.bind(InternetAddress.loopbackIPv4, 0);
  final port = socket.port;
  await socket.close();
  return port;
}

Future<bool> _becomesHealthy(Uri base, Process server) async {
  var exited = false;
  unawaited(server.exitCode.then((_) => exited = true));
  for (var i = 0; i < 100; i++) {
    if (exited) return false;
    try {
      final resp = await http.get(base.replace(path: '/healthz'));
      if (resp.statusCode == 200) return true;
    } on Exception {
      // Not listening yet.
    }
    await Future<void>.delayed(const Duration(milliseconds: 100));
  }
  return false;
}

Future<Map<String, Object?>> _rpc(
  Uri base,
  String procedure,
  Map<String, Object?> body, {
  String? cookie,
  Map<String, String>? headers,
}) async {
  final resp = await http.post(
    base.replace(path: procedure),
    headers: {
      'content-type': 'application/json',
      'cookie': ?cookie,
      ...?headers,
    },
    body: jsonEncode(body),
  );
  expect(resp.statusCode, 200, reason: '$procedure: ${resp.body}');
  return jsonDecode(resp.body) as Map<String, Object?>;
}

Future<String> _adminLogin(Uri base) async {
  final resp = await http.post(
    base.replace(path: '/moth.admin.v1.SessionService/Login'),
    headers: {'content-type': 'application/json'},
    body: jsonEncode({'email': adminEmail, 'password': adminPassword}),
  );
  expect(resp.statusCode, 200, reason: 'admin login: ${resp.body}');
  final setCookie = resp.headers['set-cookie'];
  expect(setCookie, isNotNull, reason: 'login response carries the session');
  return setCookie!.split(';').first;
}
