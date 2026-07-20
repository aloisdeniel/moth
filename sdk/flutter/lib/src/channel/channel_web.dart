import 'package:grpc/grpc_web.dart';
import 'package:grpc/service_api.dart' as api;

/// gRPC-Web channel for Flutter Web. The moth server speaks gRPC-Web on the
/// same endpoint.
api.ClientChannel createChannel(Uri endpoint) =>
    GrpcWebClientChannel.xhr(endpoint);
