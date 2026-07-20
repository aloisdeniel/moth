import 'package:grpc/grpc.dart';
import 'package:grpc/service_api.dart' as api;

/// Native HTTP/2 gRPC channel (iOS, Android, desktop). TLS follows the
/// endpoint scheme; plain `http` is supported for local development.
api.ClientChannel createChannel(Uri endpoint) {
  final secure = endpoint.scheme == 'https';
  return ClientChannel(
    endpoint.host,
    port: endpoint.hasPort ? endpoint.port : (secure ? 443 : 80),
    options: ChannelOptions(
      credentials: secure
          ? const ChannelCredentials.secure()
          : const ChannelCredentials.insecure(),
    ),
  );
}
