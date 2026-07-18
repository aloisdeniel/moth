/// Fallback for platforms with neither `dart:io` nor JS interop.
String currentPlatform() => 'unknown';

/// No OS version to report without `dart:io`.
String currentOsVersion() => '';
