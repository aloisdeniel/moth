import 'dart:io' show Platform;

/// `ios`, `android`, `macos`, `linux`, `windows` or `fuchsia`, as reported
/// in `x-moth-platform` metadata.
String currentPlatform() => Platform.operatingSystem;
