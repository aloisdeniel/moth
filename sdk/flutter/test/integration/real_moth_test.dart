// End-to-end test against a real moth binary: spawns `bin/moth serve` on a
// random port with a throwaway data dir, creates an admin + project over
// the connect JSON endpoints, then drives MothClient through the full
// lifecycle — signup, sign-in, getMe, transparent access-token refresh
// (5-second TTL project), session restore, sign-out.
//
// Excluded from the default `flutter test` run (see dart_test.yaml); run
// via `make sdk-e2e` after `make build`.
@Tags(['integration'])
library;

import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:moth_auth/moth_auth.dart';

const adminEmail = 'admin@example.com';
const adminPassword = 'admin-password-1';
const userEmail = 'jane@example.com';
const userPassword = 'password-123';

void main() {
  final binary = _findMothBinary();

  // Nullable (not late) so tearDownAll can clean up whatever exists when
  // setUpAll failed partway through.
  Directory? dataDir;
  Process? server;
  final serverLog = StringBuffer();
  late Uri base;

  setUpAll(() async {
    if (binary == null) return;
    dataDir = await Directory.systemTemp.createTemp('moth-sdk-e2e-');

    // Create the admin account directly in the data dir, before serving.
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

    // _freePort closes its probe socket before `moth serve` binds the
    // port, so another process can grab it in between; retry on a fresh
    // port when the server dies instead of becoming healthy.
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
    // Every step is guarded so a partial setUpAll still cleans up and the
    // server log still surfaces.
    server?.kill();
    await server?.exitCode;
    await dataDir?.delete(recursive: true);
    printOnFailure('moth server log:\n$serverLog');
  });

  test('signup → sign-in → refresh → restore → sign-out', () async {
    if (binary == null) {
      markTestSkipped('bin/moth not found — run `make build` first');
      return;
    }

    // -------------------------------------------------- admin: project setup
    // Admin RPCs are connect endpoints; plain JSON POSTs work fine.
    final cookie = await _adminLogin(base);
    final project = await _createProject(base, cookie, name: 'sdk e2e');
    final projectId = project['id'] as String;
    final publishableKey = project['publishableKey'] as String;
    expect(publishableKey, startsWith('pk_'));

    // 5-second access tokens so expiry happens mid-test; no email
    // verification so SignUp opens a session immediately.
    await _updateSettings(base, cookie, projectId, {
      'passwordMinLength': 8,
      'requireEmailVerification': false,
      'allowPublicSignup': true,
      'accessTokenTtlSeconds': 5,
      'refreshTokenTtlDays': 30,
    });

    // ------------------------------------------------------------ SDK client
    final store = InMemoryTokenStore();
    final client = MothClient(
      MothConfig(endpoint: base, publishableKey: publishableKey),
      tokenStore: store,
      // No proactive refresh window: tokens refresh only once they are
      // actually expired, so the refresh below is provably expiry-driven.
      refreshSkew: Duration.zero,
    );
    addTearDown(client.dispose);
    final states = <MothAuthState>[];
    client.authStateChanges.listen(states.add);

    expect(await client.restore(), isA<MothSignedOut>());

    // The project's public config reflects the settings we just wrote.
    final config = await client.getProjectConfig();
    expect(config.signUpOpen, isTrue);
    expect(config.passwordMinLength, 8);
    expect(config.google.enabled, isFalse);
    expect(config.apple.enabled, isFalse);

    // Sign-up opens a session (verification not required).
    final signUp = await client.signUp(
      email: userEmail,
      password: userPassword,
    );
    expect(signUp.signedIn, isTrue);
    expect(client.currentUser?.email, userEmail);

    final me = await client.getMe();
    expect(me.email, userEmail);
    expect(me.emailVerified, isFalse);

    final tokenA = await client.accessToken();
    final payload = _jwtPayload(tokenA);
    expect(payload['iss'], '$base/p/${project['slug']}');
    expect(
      (payload['exp'] as int) - (payload['iat'] as int),
      5,
      reason: 'project mints 5-second access tokens',
    );

    // ------------------------------------------------- transparent refresh
    // Let the access token expire, then make an authenticated call: the
    // client must refresh under the hood and succeed with a new token.
    await Future<void>.delayed(const Duration(seconds: 6));
    final refreshedMe = await client.getMe();
    expect(refreshedMe.email, userEmail);
    final tokenB = await client.accessToken();
    expect(tokenB, isNot(tokenA));

    // The authenticated http client wires the same token into plain HTTP
    // calls; moth's own healthz endpoint just proves the header flows.
    final api = authenticatedClient(client);
    addTearDown(api.close);
    final resp = await api.get(base.replace(path: '/healthz'));
    expect(resp.statusCode, 200);

    // ------------------------------------------------------ session restore
    // A second client over the same store is "the app after a restart".
    final restored = MothClient(
      MothConfig(endpoint: base, publishableKey: publishableKey),
      tokenStore: store,
      refreshSkew: Duration.zero,
    );
    addTearDown(restored.dispose);
    expect(await restored.restore(), isA<MothSignedIn>());
    expect(restored.currentUser?.email, userEmail);

    // -------------------------------------------------------- typed errors
    await expectLater(
      client.signIn(email: userEmail, password: 'wrong-password'),
      throwsA(isA<MothInvalidCredentials>()),
    );
    // The failed attempt must not have touched the session.
    expect(client.currentState, isA<MothSignedIn>());

    // ------------------------------------------------------------- sign-out
    await client.signOut();
    expect(client.currentState, isA<MothSignedOut>());
    expect(await store.load(), isNull);
    // Stream delivery is asynchronous; drain pending events before
    // asserting on the observed sequence.
    await pumpEventQueue();
    expect(states.first, isA<MothAuthLoading>());
    expect(states.last, isA<MothSignedOut>());
    expect(states.whereType<MothSignedIn>(), isNotEmpty);

    // Signing in again works (same credentials, fresh session).
    final again = await client.signIn(email: userEmail, password: userPassword);
    expect(again.email, userEmail);
  });
}

/// Locates bin/moth by walking up from the package directory to the
/// repository root, or null when it has not been built.
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

/// Polls /healthz until [server] answers, returning false as soon as the
/// process exits (e.g. the port was taken between probe and bind) or when
/// it never becomes healthy.
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

/// Calls a connect unary RPC with a JSON body and returns the decoded
/// response, failing the test on non-200s.
Future<Map<String, Object?>> _rpc(
  Uri base,
  String procedure,
  Map<String, Object?> body, {
  String? cookie,
}) async {
  final resp = await http.post(
    base.replace(path: procedure),
    headers: {'content-type': 'application/json', 'cookie': ?cookie},
    body: jsonEncode(body),
  );
  expect(resp.statusCode, 200, reason: '$procedure: ${resp.body}');
  return jsonDecode(resp.body) as Map<String, Object?>;
}

/// Logs the admin in and returns the session cookie ("name=value").
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

Future<Map<String, Object?>> _createProject(
  Uri base,
  String cookie, {
  required String name,
}) async {
  final resp = await _rpc(base, '/moth.admin.v1.ProjectService/CreateProject', {
    'name': name,
  }, cookie: cookie);
  return resp['project']! as Map<String, Object?>;
}

Future<void> _updateSettings(
  Uri base,
  String cookie,
  String projectId,
  Map<String, Object?> settings,
) async {
  await _rpc(base, '/moth.admin.v1.ProjectService/UpdateProject', {
    'id': projectId,
    'settings': settings,
    'updateMask': 'settings',
  }, cookie: cookie);
}

Map<String, Object?> _jwtPayload(String token) {
  final segment = token.split('.')[1];
  return jsonDecode(utf8.decode(base64Url.decode(base64Url.normalize(segment))))
      as Map<String, Object?>;
}
