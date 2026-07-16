import 'package:flutter_test/flutter_test.dart';
import 'package:grpc/grpc.dart' show GrpcError, StatusCode;
import 'package:moth_auth/moth_auth.dart';

import 'fakes.dart';

void main() {
  late FakeMoth moth;
  late MothClient client;

  setUp(() async {
    moth = await startFakeMoth();
    client = newClient(moth);
  });

  tearDown(() async {
    await client.dispose();
    await moth.shutdown();
  });

  Future<MothException> failingSignIn() async {
    try {
      await client.signIn(email: 'jane@example.com', password: 'pw');
      fail('expected a MothException');
    } on MothException catch (err) {
      return err;
    }
  }

  // Every ErrorInfo reason from internal/server/rpc/auth/errors.go, with the
  // status code the server pairs it with.
  final cases = <(String, int, Matcher)>[
    (
      'INVALID_CREDENTIALS',
      StatusCode.unauthenticated,
      isA<MothInvalidCredentials>(),
    ),
    (
      'EMAIL_NOT_VERIFIED',
      StatusCode.permissionDenied,
      isA<MothEmailNotVerified>(),
    ),
    (
      'EMAIL_ALREADY_EXISTS',
      StatusCode.alreadyExists,
      isA<MothEmailAlreadyExists>(),
    ),
    ('SIGNUP_CLOSED', StatusCode.permissionDenied, isA<MothSignUpClosed>()),
    ('WEAK_PASSWORD', StatusCode.invalidArgument, isA<MothWeakPassword>()),
    ('INVALID_EMAIL', StatusCode.invalidArgument, isA<MothInvalidEmail>()),
    ('INVALID_TOKEN', StatusCode.invalidArgument, isA<MothInvalidToken>()),
    (
      'INVALID_REFRESH_TOKEN',
      StatusCode.unauthenticated,
      isA<MothInvalidRefreshToken>(),
    ),
    (
      'REFRESH_TOKEN_REUSED',
      StatusCode.unauthenticated,
      isA<MothRefreshTokenReused>(),
    ),
    (
      'INVALID_ACCESS_TOKEN',
      StatusCode.unauthenticated,
      isA<MothInvalidAccessToken>(),
    ),
    ('USER_DISABLED', StatusCode.permissionDenied, isA<MothUserDisabled>()),
    ('RATE_LIMITED', StatusCode.resourceExhausted, isA<MothRateLimited>()),
    (
      'PROVIDER_DISABLED',
      StatusCode.failedPrecondition,
      isA<MothProviderDisabled>(),
    ),
    (
      'INVALID_PROVIDER_TOKEN',
      StatusCode.unauthenticated,
      isA<MothInvalidProviderToken>(),
    ),
    (
      'INVALID_OAUTH_CODE',
      StatusCode.unauthenticated,
      isA<MothInvalidOAuthCode>(),
    ),
    (
      'INVALID_REDIRECT',
      StatusCode.invalidArgument,
      isA<MothInvalidRedirect>(),
    ),
    (
      'LAST_LOGIN_METHOD',
      StatusCode.failedPrecondition,
      isA<MothLastLoginMethod>(),
    ),
  ];

  for (final (reason, code, matcher) in cases) {
    test('$reason maps to a typed exception', () async {
      moth.auth.nextError = mothError(code, reason, 'boom');
      final err = await failingSignIn();
      expect(err, matcher);
      expect(err.reason, reason);
      expect(err.message, 'boom');
    });
  }

  test('an unknown moth reason surfaces on the base MothException', () async {
    moth.auth.nextError = mothError(
      StatusCode.failedPrecondition,
      'SOMETHING_NEW',
      'later',
    );
    final err = await failingSignIn();
    expect(err.runtimeType, MothException);
    expect(err.reason, 'SOMETHING_NEW');
  });

  test('a status without ErrorInfo keeps a null reason', () async {
    moth.auth.nextError = GrpcError.internal('boom');
    final err = await failingSignIn();
    expect(err.runtimeType, MothException);
    expect(err.reason, isNull);
  });

  test('unreachable server maps to MothNetworkError', () async {
    final offline = MothClient(
      MothConfig(
        // Port 1 is never listening locally.
        endpoint: Uri.parse('http://localhost:1'),
        publishableKey: 'pk_test',
      ),
      tokenStore: InMemoryTokenStore(),
    );
    addTearDown(offline.dispose);
    await expectLater(
      offline.signIn(email: 'jane@example.com', password: 'pw'),
      throwsA(isA<MothNetworkError>()),
    );
  });
}
