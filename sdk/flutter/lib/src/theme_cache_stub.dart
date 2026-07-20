import 'theme_cache.dart';

/// Platforms without dart:io (Flutter Web) cache in memory only: the theme
/// re-fetches per session, which is what a web page does anyway.
MothThemeCache createThemeCache(String publishableKey) =>
    MothMemoryThemeCache();
