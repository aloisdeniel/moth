import 'copy_cache.dart';

/// Web (and any non-`dart:io`) fallback: the localized copy is kept in memory
/// only.
MothCopyCache createCopyCache(String publishableKey) => MothMemoryCopyCache();
