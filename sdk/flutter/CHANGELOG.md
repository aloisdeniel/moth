# Changelog

Versions of this package track the moth server version; the changelog below
is per SDK milestone until the first stamped release.

## 0.1.0

- Initial core: `MothClient` covering the full moth.auth.v1 surface
  (email/password, social sign-in, profile, email flows, project config),
  typed `MothException` hierarchy, secure session persistence with automatic
  single-flight token refresh, `authStateChanges`, and an `http` client
  wrapper for calling your own backend.
