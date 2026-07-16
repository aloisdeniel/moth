import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/src/version_check.dart';

void main() {
  group('mothVersionMismatch', () {
    test('warns when the server major differs from the SDK major', () {
      expect(
        mothVersionMismatch('2.0.0', sdkVersion: '1.4.2'),
        contains('major 2 vs 1'),
      );
      expect(mothVersionMismatch('v2.1.0', sdkVersion: '1.0.0'), isNotNull);
      // A dev SDK (0.0.0-dev.*) against a release server is a mismatch.
      expect(
        mothVersionMismatch('1.2.3', sdkVersion: '0.0.0-dev.h1a2b3c4d'),
        isNotNull,
      );
    });

    test('is silent when the majors match', () {
      expect(mothVersionMismatch('1.9.0', sdkVersion: '1.0.0'), isNull);
      expect(mothVersionMismatch('v1.2.3-rc.1', sdkVersion: '1.2.3'), isNull);
    });

    test('is silent when either side has no parsable major', () {
      expect(mothVersionMismatch('dev', sdkVersion: '1.0.0'), isNull);
      expect(mothVersionMismatch(null, sdkVersion: '1.0.0'), isNull);
      expect(mothVersionMismatch('', sdkVersion: '1.0.0'), isNull);
      expect(mothVersionMismatch('1.0.0', sdkVersion: 'dev'), isNull);
    });
  });
}
