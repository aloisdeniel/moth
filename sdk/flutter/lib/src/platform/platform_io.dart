import 'dart:io' show Platform;

/// `ios`, `android`, `macos`, `linux`, `windows` or `fuchsia`, as reported
/// in `x-moth-platform` metadata.
String currentPlatform() => Platform.operatingSystem;

/// The OS version string, as reported in push-registration metadata.
String currentOsVersion() => Platform.operatingSystemVersion;
