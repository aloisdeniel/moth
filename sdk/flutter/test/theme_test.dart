// Unit tests for MothTheme: proto mapping, dark-palette derivation (parity
// with the server's internal/theme/derive.go), ThemeData mapping and the
// JSON cache round-trip.
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_auth/src/gen/moth/auth/v1/config.pb.dart' as pb;

pb.Theme fullProtoTheme() => pb.Theme(
  revisionId: 'rev-1',
  colors: pb.ThemeColors(
    primary: '#0B6E99',
    onPrimary: '#FFFFFF',
    background: '#F7FAFC',
    onBackground: '#102A43',
    surface: '#FFFFFF',
    onSurface: '#102A43',
    error: '#B00020',
    onError: '#FFFFFF',
  ),
  darkColors: pb.ThemeColors(
    primary: '#66B8D6',
    onPrimary: '#000000',
    background: '#0B1420',
    onBackground: '#FFFFFF',
    surface: '#12202E',
    onSurface: '#FFFFFF',
    error: '#FF8A80',
    onError: '#000000',
  ),
  fontFamily: 'Lato',
  fontUrl: 'https://auth.example.com/assets/fonts/lato.ttf',
  fontScale: 1.2,
  spacingUnit: 10,
  cornerRadius: 4,
  logoLightUrl: 'https://auth.example.com/assets/p1/logo-light.png',
  logoDarkUrl: 'https://auth.example.com/assets/p1/logo-dark.png',
  termsUrl: 'https://example.com/terms',
  privacyUrl: 'https://example.com/privacy',
);

void main() {
  group('MothTheme.fallback', () {
    test('matches the server default (internal/theme Default())', () {
      final theme = MothTheme.fallback();
      expect(mothHexColor(theme.colors.primary), '#6750A4');
      expect(mothHexColor(theme.colors.onPrimary), '#FFFFFF');
      expect(mothHexColor(theme.colors.background), '#FFFBFE');
      expect(mothHexColor(theme.colors.onBackground), '#1C1B1F');
      expect(mothHexColor(theme.colors.surface), '#FFFBFE');
      expect(mothHexColor(theme.colors.onSurface), '#1C1B1F');
      expect(mothHexColor(theme.colors.error), '#B3261E');
      expect(mothHexColor(theme.colors.onError), '#FFFFFF');
      expect(theme.fontFamily, 'Inter');
      expect(theme.fontScale, 1.0);
      expect(theme.spacingUnit, 8);
      expect(theme.cornerRadius, 12);
      expect(theme.logoLightUrl, isNull);
      expect(theme.termsUrl, isNull);
    });
  });

  group('mothDeriveDarkColors', () {
    test('matches the server derivation byte-for-byte', () {
      // Expected values printed by internal/theme's DeriveDark for the
      // default palette. The same constants are pinned on the Go side
      // (internal/theme TestDeriveDarkDefaultConstants), so a derivation
      // change must move both files together.
      final dark = MothTheme.fallback().darkColors;
      expect(mothHexColor(dark.primary), '#A496C8');
      expect(mothHexColor(dark.onPrimary), '#000000');
      expect(mothHexColor(dark.background), '#1F1E1E');
      expect(mothHexColor(dark.onBackground), '#FFFFFF');
      expect(mothHexColor(dark.surface), '#292829');
      expect(mothHexColor(dark.onSurface), '#FFFFFF');
      expect(mothHexColor(dark.error), '#D17D78');
      expect(mothHexColor(dark.onError), '#000000');
    });

    test('always yields WCAG AA pairs', () {
      for (final primary in ['#000000', '#FFFFFF', '#0B6E99', '#C8102E']) {
        final light = MothThemeColors(
          primary: mothParseHexColor(primary)!,
          onPrimary: const Color(0xFFFFFFFF),
          background: const Color(0xFFF5F5F5),
          onBackground: const Color(0xFF111111),
          surface: const Color(0xFFFFFFFF),
          onSurface: const Color(0xFF111111),
          error: const Color(0xFFB00020),
          onError: const Color(0xFFFFFFFF),
        );
        final dark = mothDeriveDarkColors(light);
        for (final (a, b) in [
          (dark.primary, dark.onPrimary),
          (dark.background, dark.onBackground),
          (dark.surface, dark.onSurface),
          (dark.error, dark.onError),
        ]) {
          expect(
            mothContrastRatio(a, b),
            greaterThanOrEqualTo(4.5),
            reason: 'primary $primary: ${mothHexColor(a)}/${mothHexColor(b)}',
          );
        }
      }
    });
  });

  group('color helpers', () {
    test('parse and format round-trip', () {
      expect(mothHexColor(mothParseHexColor('#0b6e99')!), '#0B6E99');
      expect(mothParseHexColor('0B6E99'), isNull);
      expect(mothParseHexColor('#0B6E9'), isNull);
      expect(mothParseHexColor('#0B6E999'), isNull);
      expect(mothParseHexColor('#GGGGGG'), isNull);
    });

    test('contrast ratio matches WCAG reference points', () {
      const white = Color(0xFFFFFFFF);
      const black = Color(0xFF000000);
      expect(mothContrastRatio(black, white), closeTo(21.0, 1e-9));
      expect(mothContrastRatio(white, black), closeTo(21.0, 1e-9));
      expect(
        mothContrastRatio(const Color(0xFF777777), white),
        closeTo(4.48, 0.005),
      );
    });
  });

  group('MothTheme.fromProto', () {
    test('maps every field of a fully resolved theme', () {
      final theme = MothTheme.fromProto(fullProtoTheme());
      expect(theme.revisionId, 'rev-1');
      expect(mothHexColor(theme.colors.primary), '#0B6E99');
      expect(mothHexColor(theme.darkColors.primary), '#66B8D6');
      expect(mothHexColor(theme.darkColors.background), '#0B1420');
      expect(theme.fontFamily, 'Lato');
      expect(theme.fontUrl, 'https://auth.example.com/assets/fonts/lato.ttf');
      expect(theme.fontScale, closeTo(1.2, 1e-6));
      expect(theme.spacingUnit, 10);
      expect(theme.cornerRadius, 4);
      expect(
        theme.logoLightUrl,
        'https://auth.example.com/assets/p1/logo-light.png',
      );
      expect(
        theme.logoDarkUrl,
        'https://auth.example.com/assets/p1/logo-dark.png',
      );
      expect(theme.termsUrl, 'https://example.com/terms');
      expect(theme.privacyUrl, 'https://example.com/privacy');
      expect(theme.resolvedFontFamily, isNull);
    });

    test('derives the dark palette when the message omits it', () {
      final proto = fullProtoTheme()..clearDarkColors();
      final theme = MothTheme.fromProto(proto);
      final derived = mothDeriveDarkColors(theme.colors);
      expect(theme.darkColors, derived);
      expect(
        mothContrastRatio(theme.darkColors.primary, theme.darkColors.onPrimary),
        greaterThanOrEqualTo(4.5),
      );
    });

    test('falls back per field on empty or malformed values', () {
      final theme = MothTheme.fromProto(
        pb.Theme(
          colors: pb.ThemeColors(primary: '#0B6E99', onPrimary: 'oops'),
        ),
      );
      final fallback = MothTheme.fallback();
      expect(mothHexColor(theme.colors.primary), '#0B6E99');
      expect(theme.colors.onPrimary, fallback.colors.onPrimary);
      expect(theme.colors.background, fallback.colors.background);
      expect(theme.fontFamily, fallback.fontFamily);
      expect(theme.fontScale, fallback.fontScale);
      expect(theme.spacingUnit, fallback.spacingUnit);
      expect(theme.fontUrl, isNull);
      expect(theme.logoLightUrl, isNull);
    });
  });

  group('toThemeData', () {
    final theme = MothTheme.fromProto(fullProtoTheme());

    test('light: tokens land in the ColorScheme and scaffold', () {
      final data = theme.toThemeData(Brightness.light);
      expect(data.colorScheme.brightness, Brightness.light);
      expect(data.colorScheme.primary, theme.colors.primary);
      expect(data.colorScheme.onPrimary, theme.colors.onPrimary);
      expect(data.colorScheme.surface, theme.colors.surface);
      expect(data.colorScheme.onSurface, theme.colors.onSurface);
      expect(data.colorScheme.error, theme.colors.error);
      expect(data.scaffoldBackgroundColor, theme.colors.background);
    });

    test('dark: uses the dark palette', () {
      final data = theme.toThemeData(Brightness.dark);
      expect(data.colorScheme.brightness, Brightness.dark);
      expect(data.colorScheme.primary, theme.darkColors.primary);
      expect(data.colorScheme.surface, theme.darkColors.surface);
      expect(data.scaffoldBackgroundColor, theme.darkColors.background);
    });

    test('text styles keep the palette colors (merge direction)', () {
      // Regression test: Flutter's geometry text themes (englishLike) are
      // declared `inherit: false`, so merging the color variant *into*
      // them returns the geometry verbatim, drops every text color, and
      // themed text paints with the engine default (white) — invisible on
      // a light background.
      final light = theme.toThemeData(Brightness.light).textTheme;
      expect(light.bodyMedium!.color, theme.colors.onSurface);
      expect(light.headlineMedium!.color, theme.colors.onSurface);
      expect(light.titleLarge!.color, theme.colors.onSurface);
      expect(light.bodySmall!.color, theme.colors.onSurface);
      final dark = theme.toThemeData(Brightness.dark).textTheme;
      expect(dark.bodyMedium!.color, theme.darkColors.onSurface);
      expect(dark.headlineMedium!.color, theme.darkColors.onSurface);
    });

    test('font scale multiplies every text size', () {
      final base = MothTheme.fallback().toThemeData(Brightness.light);
      final scaled = theme.toThemeData(Brightness.light);
      expect(
        scaled.textTheme.bodyMedium!.fontSize,
        closeTo(base.textTheme.bodyMedium!.fontSize! * 1.2, 1e-6),
      );
      expect(
        scaled.textTheme.headlineMedium!.fontSize,
        closeTo(base.textTheme.headlineMedium!.fontSize! * 1.2, 1e-6),
      );
    });

    test('corner radius shapes inputs and buttons', () {
      final data = theme.toThemeData(Brightness.light);
      final border = data.inputDecorationTheme.border! as OutlineInputBorder;
      expect(border.borderRadius, BorderRadius.circular(4));
      final buttonShape =
          data.filledButtonTheme.style!.shape!.resolve(const {})!
              as RoundedRectangleBorder;
      expect(buttonShape.borderRadius, BorderRadius.circular(4));
    });

    test('registered font family applies once resolved', () {
      final loaded = theme.copyWith(resolvedFontFamily: 'moth Lato');
      final data = loaded.toThemeData(Brightness.light);
      expect(data.textTheme.bodyMedium!.fontFamily, 'moth Lato');
      // Until then the system font stays.
      expect(
        theme.toThemeData(Brightness.light).textTheme.bodyMedium!.fontFamily,
        isNot('moth Lato'),
      );
    });
  });

  group('JSON cache round-trip', () {
    test('full theme survives encode/decode', () {
      final theme = MothTheme.fromProto(fullProtoTheme());
      final decoded = MothTheme.fromJson(
        jsonDecode(jsonEncode(theme.toJson())) as Map<String, Object?>,
      );
      expect(decoded, theme);
    });

    test('minimal theme survives encode/decode', () {
      final theme = MothTheme.fallback();
      final decoded = MothTheme.fromJson(
        jsonDecode(jsonEncode(theme.toJson())) as Map<String, Object?>,
      );
      expect(decoded, theme);
    });

    test('corrupt colors throw (treated as a cache miss upstream)', () {
      expect(
        () => MothTheme.fromJson({
          'colors': {'primary': 'nope'},
        }),
        throwsFormatException,
      );
    });
  });
}
