import 'package:http/http.dart' as http;

import 'client.dart';

/// An [http.Client] that attaches `Authorization: Bearer <access token>` —
/// kept fresh via [MothClient.accessToken] — to every request. Drop it in
/// wherever the app calls its own backend; the backend verifies the JWT
/// against the project JWKS.
///
/// ```dart
/// final api = authenticatedClient(moth);
/// final resp = await api.get(Uri.parse('https://api.example.com/todos'));
/// ```
///
/// (dio users: see the README for the equivalent three-line interceptor —
/// the SDK deliberately does not depend on dio.)
class MothHttpClient extends http.BaseClient {
  MothHttpClient(this._moth, {http.Client? inner})
    : _inner = inner ?? http.Client();

  final MothClient _moth;
  final http.Client _inner;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    request.headers['authorization'] = 'Bearer ${await _moth.accessToken()}';
    return _inner.send(request);
  }

  @override
  void close() => _inner.close();
}

/// Wraps [inner] (or a fresh [http.Client]) so every request carries a valid
/// moth access token.
http.Client authenticatedClient(MothClient moth, {http.Client? inner}) =>
    MothHttpClient(moth, inner: inner);
