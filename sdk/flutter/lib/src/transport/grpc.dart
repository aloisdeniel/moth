/// Platform-appropriate grpc surface. Both libraries export the shared API
/// (`GrpcError`, `StatusCode`, `CallOptions`, ...); only the channel
/// implementations differ, and those are behind `channel/`.
library;

export 'package:grpc/grpc.dart'
    if (dart.library.js_interop) 'package:grpc/grpc_web.dart';
