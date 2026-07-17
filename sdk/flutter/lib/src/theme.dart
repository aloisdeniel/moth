import 'dart:math' as math;

import 'package:flutter/material.dart';

import 'gen/moth/auth/v1/config.pb.dart' as pb;

/// One complete palette of the moth design system: each color role and its
/// "on" (foreground) counterpart. Server-side validation guarantees WCAG AA
/// contrast (>= 4.5:1) between every pair.
@immutable
class MothThemeColors {
  const MothThemeColors({
    required this.primary,
    required this.onPrimary,
    required this.background,
    required this.onBackground,
    required this.surface,
    required this.onSurface,
    required this.error,
    required this.onError,
  });

  factory MothThemeColors.fromJson(Map<String, Object?> json) {
    Color parse(String field) {
      final color = mothParseHexColor(json[field] as String? ?? '');
      if (color == null) {
        throw FormatException('invalid color for $field: ${json[field]}');
      }
      return color;
    }

    return MothThemeColors(
      primary: parse('primary'),
      onPrimary: parse('onPrimary'),
      background: parse('background'),
      onBackground: parse('onBackground'),
      surface: parse('surface'),
      onSurface: parse('onSurface'),
      error: parse('error'),
      onError: parse('onError'),
    );
  }

  final Color primary;
  final Color onPrimary;
  final Color background;
  final Color onBackground;
  final Color surface;
  final Color onSurface;
  final Color error;
  final Color onError;

  Map<String, Object?> toJson() => {
    'primary': mothHexColor(primary),
    'onPrimary': mothHexColor(onPrimary),
    'background': mothHexColor(background),
    'onBackground': mothHexColor(onBackground),
    'surface': mothHexColor(surface),
    'onSurface': mothHexColor(onSurface),
    'error': mothHexColor(error),
    'onError': mothHexColor(onError),
  };

  @override
  bool operator ==(Object other) =>
      other is MothThemeColors &&
      other.primary == primary &&
      other.onPrimary == onPrimary &&
      other.background == background &&
      other.onBackground == onBackground &&
      other.surface == surface &&
      other.onSurface == onSurface &&
      other.error == error &&
      other.onError == onError;

  @override
  int get hashCode => Object.hash(
    primary,
    onPrimary,
    background,
    onBackground,
    surface,
    onSurface,
    error,
    onError,
  );
}

/// A project's design system, fully resolved and ready to render: the
/// public form of the theme configured in the moth admin (delivered inside
/// `GetProjectConfig`), mapped to Flutter types.
///
/// [toThemeData] turns it into a Material [ThemeData]; the moth widgets
/// ([MothLoginScreen] and its building blocks) consume it exclusively, so a
/// project's brand applies without an app release.
@immutable
class MothTheme {
  const MothTheme({
    this.revisionId = '',
    required this.colors,
    required this.darkColors,
    this.fontFamily = 'Inter',
    this.fontUrl,
    this.fontScale = 1.0,
    this.spacingUnit = 8,
    this.cornerRadius = 12,
    this.logoLightUrl,
    this.logoDarkUrl,
    this.termsUrl,
    this.privacyUrl,
    this.resolvedFontFamily,
    this.source,
  });

  /// The theme every project starts from (and the offline fallback when
  /// nothing is cached yet): the server's built-in default — the Material
  /// baseline palette, an 8px grid and 12px corners.
  factory MothTheme.fallback() => _fallback;

  static final MothTheme _fallback = () {
    const light = MothThemeColors(
      primary: Color(0xFF6750A4),
      onPrimary: Color(0xFFFFFFFF),
      background: Color(0xFFFFFBFE),
      onBackground: Color(0xFF1C1B1F),
      surface: Color(0xFFFFFBFE),
      onSurface: Color(0xFF1C1B1F),
      error: Color(0xFFB3261E),
      onError: Color(0xFFFFFFFF),
    );
    return MothTheme(colors: light, darkColors: mothDeriveDarkColors(light));
  }();

  /// Maps the theme message from `GetProjectConfig`. The message is fully
  /// resolved server-side; fields an older server leaves empty fall back to
  /// the defaults (and a missing dark palette is derived locally with the
  /// same algorithm the server uses).
  factory MothTheme.fromProto(pb.Theme proto) {
    final fallback = MothTheme.fallback();
    final light = proto.hasColors()
        ? _colorsFromProto(proto.colors, fallback.colors)
        : fallback.colors;
    final derivedDark = mothDeriveDarkColors(light);
    String? blank(String s) => s.isEmpty ? null : s;
    return MothTheme(
      revisionId: proto.revisionId,
      colors: light,
      darkColors: proto.hasDarkColors()
          ? _colorsFromProto(proto.darkColors, derivedDark)
          : derivedDark,
      fontFamily: proto.fontFamily.isEmpty
          ? fallback.fontFamily
          : proto.fontFamily,
      fontUrl: blank(proto.fontUrl),
      fontScale: proto.fontScale > 0 ? proto.fontScale : fallback.fontScale,
      spacingUnit: proto.spacingUnit > 0
          ? proto.spacingUnit.toDouble()
          : fallback.spacingUnit,
      cornerRadius: proto.cornerRadius >= 0
          ? proto.cornerRadius.toDouble()
          : fallback.cornerRadius,
      logoLightUrl: blank(proto.logoLightUrl),
      logoDarkUrl: blank(proto.logoDarkUrl),
      termsUrl: blank(proto.termsUrl),
      privacyUrl: blank(proto.privacyUrl),
      source: proto,
    );
  }

  factory MothTheme.fromJson(Map<String, Object?> json) {
    final colors = MothThemeColors.fromJson(
      json['colors']! as Map<String, Object?>,
    );
    final darkJson = json['darkColors'] as Map<String, Object?>?;
    return MothTheme(
      revisionId: json['revisionId'] as String? ?? '',
      colors: colors,
      darkColors: darkJson == null
          ? mothDeriveDarkColors(colors)
          : MothThemeColors.fromJson(darkJson),
      fontFamily: json['fontFamily'] as String? ?? 'Inter',
      fontUrl: json['fontUrl'] as String?,
      fontScale: (json['fontScale'] as num?)?.toDouble() ?? 1.0,
      spacingUnit: (json['spacingUnit'] as num?)?.toDouble() ?? 8,
      cornerRadius: (json['cornerRadius'] as num?)?.toDouble() ?? 12,
      logoLightUrl: json['logoLightUrl'] as String?,
      logoDarkUrl: json['logoDarkUrl'] as String?,
      termsUrl: json['termsUrl'] as String?,
      privacyUrl: json['privacyUrl'] as String?,
    );
  }

  /// Identifies this version of the theme; echoed to the server as
  /// `known_theme_revision` so an unchanged theme is not re-sent. Empty for
  /// [MothTheme.fallback] and hand-built themes.
  final String revisionId;

  /// Light palette.
  final MothThemeColors colors;

  /// Dark palette (admin overrides merged with derived values server-side,
  /// or derived locally via [mothDeriveDarkColors]).
  final MothThemeColors darkColors;

  /// Font family display name (from the server's curated set).
  final String fontFamily;

  /// Absolute URL of the font file to download and register; null when the
  /// server did not provide one (system font applies).
  final String? fontUrl;

  /// Global text-size multiplier.
  final double fontScale;

  /// Base spacing step in logical pixels; every gap and padding in the moth
  /// widgets is a multiple of it.
  final double spacingUnit;

  /// Component corner radius in logical pixels.
  final double cornerRadius;

  /// Absolute logo URLs per color scheme; null when no logo is set.
  final String? logoLightUrl;
  final String? logoDarkUrl;

  /// Optional legal links rendered in the login screen footer.
  final String? termsUrl;
  final String? privacyUrl;

  /// The family name the theme's font was registered under once its file
  /// finished downloading (see `MothFontLoader`). Null until then — text
  /// renders in the system font and swaps when the font is ready.
  final String? resolvedFontFamily;

  /// The wire message this theme was mapped from ([MothTheme.fromProto]) —
  /// the raw payload the on-device config cache persists, so cache and wire
  /// share one schema. Null for hand-built themes and [MothTheme.fallback].
  /// Derivation metadata only; not part of equality.
  final pb.Theme? source;

  /// [spacingUnit] times [units] — spacing helper for themed layouts.
  double space(double units) => spacingUnit * units;

  MothTheme copyWith({String? resolvedFontFamily}) => MothTheme(
    revisionId: revisionId,
    colors: colors,
    darkColors: darkColors,
    fontFamily: fontFamily,
    fontUrl: fontUrl,
    fontScale: fontScale,
    spacingUnit: spacingUnit,
    cornerRadius: cornerRadius,
    logoLightUrl: logoLightUrl,
    logoDarkUrl: logoDarkUrl,
    termsUrl: termsUrl,
    privacyUrl: privacyUrl,
    resolvedFontFamily: resolvedFontFamily ?? this.resolvedFontFamily,
    source: source,
  );

  /// Builds the Material theme for [brightness] from the tokens: the
  /// palette lands in the [ColorScheme], text sizes scale by [fontScale],
  /// inputs and buttons share [cornerRadius], and control heights follow
  /// [spacingUnit].
  ThemeData toThemeData(Brightness brightness) {
    final palette = brightness == Brightness.dark ? darkColors : colors;
    final scheme = ColorScheme(
      brightness: brightness,
      primary: palette.primary,
      onPrimary: palette.onPrimary,
      secondary: palette.primary,
      onSecondary: palette.onPrimary,
      error: palette.error,
      onError: palette.onError,
      surface: palette.surface,
      onSurface: palette.onSurface,
      // Supporting roles derived from the tokens; the on* of each container
      // keeps the AA-checked pairing of its base role.
      errorContainer: palette.error,
      onErrorContainer: palette.onError,
      secondaryContainer: palette.primary,
      onSecondaryContainer: palette.onPrimary,
      onSurfaceVariant: palette.onSurface.withValues(alpha: 0.72),
      outline: palette.onSurface.withValues(alpha: 0.45),
      outlineVariant: palette.onSurface.withValues(alpha: 0.22),
    );
    final radius = BorderRadius.circular(cornerRadius);
    final shape = RoundedRectangleBorder(borderRadius: radius);
    final controlHeight = Size.fromHeight(spacingUnit * 6);
    // Merge the color variant into the geometry (concrete font sizes), the
    // same direction as ThemeData.localize: the geometry styles carry
    // `inherit: false`, so merging the other way around would return them
    // verbatim and drop every text color.
    final typography = Typography.material2021(colorScheme: scheme);
    final textTheme = typography.englishLike
        .merge(
          brightness == Brightness.dark ? typography.white : typography.black,
        )
        .apply(fontFamily: resolvedFontFamily, fontSizeFactor: fontScale);
    final base = ThemeData(
      useMaterial3: true,
      colorScheme: scheme,
      scaffoldBackgroundColor: palette.background,
      fontFamily: resolvedFontFamily,
    );
    return base.copyWith(
      textTheme: textTheme,
      inputDecorationTheme: base.inputDecorationTheme.copyWith(
        border: OutlineInputBorder(borderRadius: radius),
        enabledBorder: OutlineInputBorder(
          borderRadius: radius,
          borderSide: BorderSide(color: scheme.outline),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: radius,
          borderSide: BorderSide(color: scheme.primary, width: 2),
        ),
        errorBorder: OutlineInputBorder(
          borderRadius: radius,
          borderSide: BorderSide(color: scheme.error),
        ),
        focusedErrorBorder: OutlineInputBorder(
          borderRadius: radius,
          borderSide: BorderSide(color: scheme.error, width: 2),
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: scheme.primary,
          foregroundColor: scheme.onPrimary,
          shape: shape,
          minimumSize: controlHeight,
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: scheme.onSurface,
          side: BorderSide(color: scheme.outline),
          shape: shape,
          minimumSize: controlHeight,
        ),
      ),
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          foregroundColor: scheme.primary,
          shape: shape,
        ),
      ),
      progressIndicatorTheme: ProgressIndicatorThemeData(color: scheme.primary),
      dividerTheme: DividerThemeData(color: scheme.outlineVariant),
    );
  }

  Map<String, Object?> toJson() => {
    'revisionId': revisionId,
    'colors': colors.toJson(),
    'darkColors': darkColors.toJson(),
    'fontFamily': fontFamily,
    if (fontUrl != null) 'fontUrl': fontUrl,
    'fontScale': fontScale,
    'spacingUnit': spacingUnit,
    'cornerRadius': cornerRadius,
    if (logoLightUrl != null) 'logoLightUrl': logoLightUrl,
    if (logoDarkUrl != null) 'logoDarkUrl': logoDarkUrl,
    if (termsUrl != null) 'termsUrl': termsUrl,
    if (privacyUrl != null) 'privacyUrl': privacyUrl,
  };

  @override
  bool operator ==(Object other) =>
      other is MothTheme &&
      other.revisionId == revisionId &&
      other.colors == colors &&
      other.darkColors == darkColors &&
      other.fontFamily == fontFamily &&
      other.fontUrl == fontUrl &&
      other.fontScale == fontScale &&
      other.spacingUnit == spacingUnit &&
      other.cornerRadius == cornerRadius &&
      other.logoLightUrl == logoLightUrl &&
      other.logoDarkUrl == logoDarkUrl &&
      other.termsUrl == termsUrl &&
      other.privacyUrl == privacyUrl &&
      other.resolvedFontFamily == resolvedFontFamily;

  @override
  int get hashCode => Object.hash(
    revisionId,
    colors,
    darkColors,
    fontFamily,
    fontUrl,
    fontScale,
    spacingUnit,
    cornerRadius,
    logoLightUrl,
    logoDarkUrl,
    termsUrl,
    privacyUrl,
    resolvedFontFamily,
  );

  static MothThemeColors _colorsFromProto(
    pb.ThemeColors proto,
    MothThemeColors fallback,
  ) {
    Color parse(String hex, Color fallback) =>
        mothParseHexColor(hex) ?? fallback;
    return MothThemeColors(
      primary: parse(proto.primary, fallback.primary),
      onPrimary: parse(proto.onPrimary, fallback.onPrimary),
      background: parse(proto.background, fallback.background),
      onBackground: parse(proto.onBackground, fallback.onBackground),
      surface: parse(proto.surface, fallback.surface),
      onSurface: parse(proto.onSurface, fallback.onSurface),
      error: parse(proto.error, fallback.error),
      onError: parse(proto.onError, fallback.onError),
    );
  }
}

// ------------------------------------------------------------- color math
//
// Mirrors internal/theme on the server (color.go + derive.go) so a locally
// derived dark palette matches what the server would have sent.

final _hexPattern = RegExp(r'^#[0-9a-fA-F]{6}$');

/// Parses a strict `#RRGGBB` hex color; null when malformed.
Color? mothParseHexColor(String hex) {
  if (!_hexPattern.hasMatch(hex)) return null;
  return Color(0xFF000000 | int.parse(hex.substring(1), radix: 16));
}

/// Formats a color as uppercase `#RRGGBB` (alpha dropped).
String mothHexColor(Color color) {
  final rgb = color.toARGB32() & 0xFFFFFF;
  return '#${rgb.toRadixString(16).padLeft(6, '0').toUpperCase()}';
}

/// Derives a dark palette from a light one with the exact algorithm the
/// server uses (internal/theme/derive.go): background and surface blend
/// 88% / 84% toward black, primary and error blend 40% toward white, and
/// every on* color becomes black or white — whichever contrasts more,
/// which always meets WCAG AA.
MothThemeColors mothDeriveDarkColors(MothThemeColors light) {
  const backgroundBlend = 0.88;
  const surfaceBlend = 0.84;
  const accentBlend = 0.40;
  const white = Color(0xFFFFFFFF);
  const black = Color(0xFF000000);
  final primary = _mix(light.primary, white, accentBlend);
  final background = _mix(light.background, black, backgroundBlend);
  final surface = _mix(light.surface, black, surfaceBlend);
  final error = _mix(light.error, white, accentBlend);
  return MothThemeColors(
    primary: primary,
    onPrimary: _bestOn(primary),
    background: background,
    onBackground: _bestOn(background),
    surface: surface,
    onSurface: _bestOn(surface),
    error: error,
    onError: _bestOn(error),
  );
}

/// WCAG 2.x contrast ratio between two colors: 1 (identical luminance) to
/// 21 (black on white); order does not matter.
double mothContrastRatio(Color a, Color b) {
  var la = _luminance(a);
  var lb = _luminance(b);
  if (la < lb) {
    final t = la;
    la = lb;
    lb = t;
  }
  return (la + 0.05) / (lb + 0.05);
}

/// Blends [color] toward [toward] by [t], channel-wise on sRGB bytes with
/// round-half-up — naive on purpose, matching the server byte-for-byte.
Color _mix(Color color, Color toward, double t) {
  int blend(int a, int b) => (a * (1 - t) + b * t).round();
  final c = color.toARGB32();
  final o = toward.toARGB32();
  return Color.fromARGB(
    0xFF,
    blend((c >> 16) & 0xFF, (o >> 16) & 0xFF),
    blend((c >> 8) & 0xFF, (o >> 8) & 0xFF),
    blend(c & 0xFF, o & 0xFF),
  );
}

/// Black or white, whichever has the higher contrast against [color]
/// (white wins ties, as on the server).
Color _bestOn(Color color) {
  const white = Color(0xFFFFFFFF);
  const black = Color(0xFF000000);
  return mothContrastRatio(color, white) >= mothContrastRatio(color, black)
      ? white
      : black;
}

/// WCAG 2.x relative luminance: 0 for black, 1 for white.
double _luminance(Color color) {
  final argb = color.toARGB32();
  double channel(int v) {
    final s = v / 255;
    return s <= 0.03928
        ? s / 12.92
        : math.pow((s + 0.055) / 1.055, 2.4).toDouble();
  }

  return 0.2126 * channel((argb >> 16) & 0xFF) +
      0.7152 * channel((argb >> 8) & 0xFF) +
      0.0722 * channel(argb & 0xFF);
}
