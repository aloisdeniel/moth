/// Connection settings for a moth project.
///
/// The values to paste here are shown on the project's setup-instructions
/// page in the moth admin.
class MothConfig {
  const MothConfig({required this.endpoint, required this.publishableKey});

  /// Base URL of the moth server, e.g. `https://auth.example.com`.
  ///
  /// TLS follows the scheme: `https` uses secure transport, plain `http` is
  /// supported for local development (e.g. `http://localhost:8080`).
  final Uri endpoint;

  /// The project's publishable key (`pk_...`), attached to every call as
  /// `x-moth-key` metadata. Safe to embed in the app.
  final String publishableKey;
}
