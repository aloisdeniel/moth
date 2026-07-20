import 'dart:async';
import 'dart:developer' as developer;

import 'package:grpc/service_api.dart' as grpc;

import 'version.dart';

/// Reads the `x-moth-version` response metadata the server attaches to
/// every RPC and, in debug builds only, warns once per server version when
/// the server's major version diverges from the SDK's — the wire contract
/// is guaranteed within a major version (plan/05, "version coupling").
class VersionCheckInterceptor extends grpc.ClientInterceptor {
  /// Server versions already warned about, so a skewed server does not spam
  /// the log on every call.
  static final Set<String> _warned = {};

  @override
  grpc.ResponseFuture<R> interceptUnary<Q, R>(
    grpc.ClientMethod<Q, R> method,
    Q request,
    grpc.CallOptions options,
    grpc.ClientUnaryInvoker<Q, R> invoker,
  ) {
    final call = invoker(method, request, options);
    // Asserts only run in debug builds; release builds skip the check
    // entirely.
    assert(() {
      unawaited(
        call.headers.then((headers) {
          final server = headers['x-moth-version'] ?? headers['X-Moth-Version'];
          if (server == null || !_warned.add(server)) return;
          final warning = mothVersionMismatch(server);
          if (warning != null) {
            developer.log(warning, name: 'moth', level: 900 /* warning */);
          }
        }, onError: (Object _) {}),
      );
      return true;
    }());
    return call;
  }
}

/// Returns a warning when [serverVersion] and [sdkVersion] have different
/// major versions, or null when they match or either side has no parsable
/// major (e.g. a `dev` server build). Exposed for tests;
/// [VersionCheckInterceptor] logs it in debug builds.
String? mothVersionMismatch(
  String? serverVersion, {
  String sdkVersion = mothSdkVersion,
}) {
  final server = _major(serverVersion);
  final sdk = _major(sdkVersion);
  if (server == null || sdk == null || server == sdk) return null;
  return 'moth server version $serverVersion does not match SDK version '
      '$sdkVersion (major $server vs $sdk). The wire contract is only '
      'guaranteed within a major version — update the moth_auth dependency '
      'from your instance\'s /pub repository.';
}

final _majorRe = RegExp(r'^(\d+)\.\d+\.\d+');

int? _major(String? version) {
  if (version == null) return null;
  final v = version.startsWith('v') ? version.substring(1) : version;
  final match = _majorRe.firstMatch(v);
  if (match == null) return null;
  return int.parse(match.group(1)!);
}
