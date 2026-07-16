// Shared helpers for widget tests that talk to the in-process fake gRPC
// server from fakes.dart.
//
// flutter_test runs the test body in a fake-async zone, so real socket I/O
// (the gRPC calls) only progresses inside `tester.runAsync` windows; the
// helpers below alternate short real-time waits with pumps until the tree
// (or the fake server) reaches the expected state.
import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';

/// Runs [body] with real async processing and a non-nullable result.
Future<T> runReal<T>(WidgetTester tester, Future<T> Function() body) async =>
    (await tester.runAsync(body)) as T;

/// Awaits [future] by alternating real-I/O windows and fake-zone pumps.
///
/// Client futures started in the test zone need both: the socket events
/// only arrive while real async runs, and the continuations they schedule
/// are fake-zone microtasks that only run on pump. Blocking on such a
/// future inside `runAsync` would deadlock.
Future<T> settle<T>(WidgetTester tester, Future<T> future) async {
  var done = false;
  T? result;
  Object? error;
  StackTrace? stack;
  future.then(
    (value) {
      result = value;
      done = true;
    },
    onError: (Object err, StackTrace st) {
      error = err;
      stack = st;
      done = true;
    },
  );
  await pumpUntil(tester, () => done, reason: 'future to complete');
  if (error != null) Error.throwWithStackTrace(error!, stack!);
  return result as T;
}

/// Pumps frames, letting real I/O progress in between, until [condition]
/// holds.
Future<void> pumpUntil(
  WidgetTester tester,
  bool Function() condition, {
  Duration timeout = const Duration(seconds: 10),
  String? reason,
}) async {
  final deadline = DateTime.now().add(timeout);
  while (!condition()) {
    if (DateTime.now().isAfter(deadline)) {
      fail(
        'pumpUntil timed out${reason == null ? '' : ' waiting for $reason'}',
      );
    }
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 20)),
    );
    // Advance the fake clock too, so zero/short-delay timers created in
    // the test zone (e.g. by the gRPC internals) get to run.
    await tester.pump(const Duration(milliseconds: 20));
  }
}

Future<void> pumpUntilFound(
  WidgetTester tester,
  Finder finder, {
  Duration timeout = const Duration(seconds: 10),
}) => pumpUntil(
  tester,
  () => finder.evaluate().isNotEmpty,
  timeout: timeout,
  reason: '$finder',
);

/// A token store whose first load blocks until [gate] completes, pinning
/// the client in [MothAuthLoading] for as long as the test needs.
class GatedTokenStore implements TokenStore {
  final _inner = InMemoryTokenStore();
  final gate = Completer<void>();

  @override
  Future<StoredSession?> load() async {
    await gate.future;
    return _inner.load();
  }

  @override
  Future<void> save(StoredSession session) => _inner.save(session);

  @override
  Future<void> clear() => _inner.clear();
}

class RecordingAdapter implements MothOAuthAdapter {
  MothGoogleConfig? googleConfig;
  String? hashedNonce;

  @override
  Future<MothGoogleCredential?> getGoogleIdToken(
    MothGoogleConfig config,
  ) async {
    googleConfig = config;
    return const MothGoogleCredential(idToken: 'google-id-token');
  }

  @override
  Future<MothAppleCredential?> getAppleCredential({
    required String hashedNonce,
  }) async {
    this.hashedNonce = hashedNonce;
    return const MothAppleCredential(idToken: 'apple-id-token');
  }
}
