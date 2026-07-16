import 'package:grpc/service_api.dart';

/// Fallback for platforms with neither `dart:io` nor JS interop.
ClientChannel createChannel(Uri endpoint) =>
    throw UnsupportedError('moth_auth: unsupported platform');
