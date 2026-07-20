// This is a generated file - do not edit.
//
// Generated from moth/projectconfig/v1/projectconfig.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import 'projectconfig.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'projectconfig.pbenum.dart';

/// LegalLinks are the optional legal URLs rendered near signup and on the
/// paywall footer.
class LegalLinks extends $pb.GeneratedMessage {
  factory LegalLinks({
    $core.String? termsUrl,
    $core.String? privacyUrl,
  }) {
    final result = create();
    if (termsUrl != null) result.termsUrl = termsUrl;
    if (privacyUrl != null) result.privacyUrl = privacyUrl;
    return result;
  }

  LegalLinks._();

  factory LegalLinks.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory LegalLinks.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LegalLinks',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'moth.projectconfig.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'termsUrl')
    ..aOS(2, _omitFieldNames ? '' : 'privacyUrl')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LegalLinks clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LegalLinks copyWith(void Function(LegalLinks) updates) =>
      super.copyWith((message) => updates(message as LegalLinks)) as LegalLinks;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LegalLinks create() => LegalLinks._();
  @$core.override
  LegalLinks createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static LegalLinks getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LegalLinks>(create);
  static LegalLinks? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get termsUrl => $_getSZ(0);
  @$pb.TagNumber(1)
  set termsUrl($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTermsUrl() => $_has(0);
  @$pb.TagNumber(1)
  void clearTermsUrl() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get privacyUrl => $_getSZ(1);
  @$pb.TagNumber(2)
  set privacyUrl($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPrivacyUrl() => $_has(1);
  @$pb.TagNumber(2)
  void clearPrivacyUrl() => $_clearField(2);
}

/// ThemeColors is a complete palette: every role and its "on" (foreground)
/// counterpart, as #RRGGBB values.
class ThemeColors extends $pb.GeneratedMessage {
  factory ThemeColors({
    $core.String? primary,
    $core.String? onPrimary,
    $core.String? background,
    $core.String? onBackground,
    $core.String? surface,
    $core.String? onSurface,
    $core.String? error,
    $core.String? onError,
  }) {
    final result = create();
    if (primary != null) result.primary = primary;
    if (onPrimary != null) result.onPrimary = onPrimary;
    if (background != null) result.background = background;
    if (onBackground != null) result.onBackground = onBackground;
    if (surface != null) result.surface = surface;
    if (onSurface != null) result.onSurface = onSurface;
    if (error != null) result.error = error;
    if (onError != null) result.onError = onError;
    return result;
  }

  ThemeColors._();

  factory ThemeColors.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ThemeColors.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ThemeColors',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'moth.projectconfig.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'primary')
    ..aOS(2, _omitFieldNames ? '' : 'onPrimary')
    ..aOS(3, _omitFieldNames ? '' : 'background')
    ..aOS(4, _omitFieldNames ? '' : 'onBackground')
    ..aOS(5, _omitFieldNames ? '' : 'surface')
    ..aOS(6, _omitFieldNames ? '' : 'onSurface')
    ..aOS(7, _omitFieldNames ? '' : 'error')
    ..aOS(8, _omitFieldNames ? '' : 'onError')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ThemeColors clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ThemeColors copyWith(void Function(ThemeColors) updates) =>
      super.copyWith((message) => updates(message as ThemeColors))
          as ThemeColors;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ThemeColors create() => ThemeColors._();
  @$core.override
  ThemeColors createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ThemeColors getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ThemeColors>(create);
  static ThemeColors? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get primary => $_getSZ(0);
  @$pb.TagNumber(1)
  set primary($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPrimary() => $_has(0);
  @$pb.TagNumber(1)
  void clearPrimary() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get onPrimary => $_getSZ(1);
  @$pb.TagNumber(2)
  set onPrimary($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOnPrimary() => $_has(1);
  @$pb.TagNumber(2)
  void clearOnPrimary() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get background => $_getSZ(2);
  @$pb.TagNumber(3)
  set background($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasBackground() => $_has(2);
  @$pb.TagNumber(3)
  void clearBackground() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get onBackground => $_getSZ(3);
  @$pb.TagNumber(4)
  set onBackground($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasOnBackground() => $_has(3);
  @$pb.TagNumber(4)
  void clearOnBackground() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get surface => $_getSZ(4);
  @$pb.TagNumber(5)
  set surface($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSurface() => $_has(4);
  @$pb.TagNumber(5)
  void clearSurface() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get onSurface => $_getSZ(5);
  @$pb.TagNumber(6)
  set onSurface($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasOnSurface() => $_has(5);
  @$pb.TagNumber(6)
  void clearOnSurface() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get error => $_getSZ(6);
  @$pb.TagNumber(7)
  set error($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasError() => $_has(6);
  @$pb.TagNumber(7)
  void clearError() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get onError => $_getSZ(7);
  @$pb.TagNumber(8)
  set onError($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasOnError() => $_has(7);
  @$pb.TagNumber(8)
  void clearOnError() => $_clearField(8);
}

/// ThemeColorOverrides is a partial dark palette: any empty field is derived
/// from the light palette instead (see internal/theme.DeriveDark).
class ThemeColorOverrides extends $pb.GeneratedMessage {
  factory ThemeColorOverrides({
    $core.String? primary,
    $core.String? onPrimary,
    $core.String? background,
    $core.String? onBackground,
    $core.String? surface,
    $core.String? onSurface,
    $core.String? error,
    $core.String? onError,
  }) {
    final result = create();
    if (primary != null) result.primary = primary;
    if (onPrimary != null) result.onPrimary = onPrimary;
    if (background != null) result.background = background;
    if (onBackground != null) result.onBackground = onBackground;
    if (surface != null) result.surface = surface;
    if (onSurface != null) result.onSurface = onSurface;
    if (error != null) result.error = error;
    if (onError != null) result.onError = onError;
    return result;
  }

  ThemeColorOverrides._();

  factory ThemeColorOverrides.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ThemeColorOverrides.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ThemeColorOverrides',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'moth.projectconfig.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'primary')
    ..aOS(2, _omitFieldNames ? '' : 'onPrimary')
    ..aOS(3, _omitFieldNames ? '' : 'background')
    ..aOS(4, _omitFieldNames ? '' : 'onBackground')
    ..aOS(5, _omitFieldNames ? '' : 'surface')
    ..aOS(6, _omitFieldNames ? '' : 'onSurface')
    ..aOS(7, _omitFieldNames ? '' : 'error')
    ..aOS(8, _omitFieldNames ? '' : 'onError')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ThemeColorOverrides clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ThemeColorOverrides copyWith(void Function(ThemeColorOverrides) updates) =>
      super.copyWith((message) => updates(message as ThemeColorOverrides))
          as ThemeColorOverrides;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ThemeColorOverrides create() => ThemeColorOverrides._();
  @$core.override
  ThemeColorOverrides createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ThemeColorOverrides getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ThemeColorOverrides>(create);
  static ThemeColorOverrides? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get primary => $_getSZ(0);
  @$pb.TagNumber(1)
  set primary($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPrimary() => $_has(0);
  @$pb.TagNumber(1)
  void clearPrimary() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get onPrimary => $_getSZ(1);
  @$pb.TagNumber(2)
  set onPrimary($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOnPrimary() => $_has(1);
  @$pb.TagNumber(2)
  void clearOnPrimary() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get background => $_getSZ(2);
  @$pb.TagNumber(3)
  set background($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasBackground() => $_has(2);
  @$pb.TagNumber(3)
  void clearBackground() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get onBackground => $_getSZ(3);
  @$pb.TagNumber(4)
  set onBackground($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasOnBackground() => $_has(3);
  @$pb.TagNumber(4)
  void clearOnBackground() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get surface => $_getSZ(4);
  @$pb.TagNumber(5)
  set surface($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSurface() => $_has(4);
  @$pb.TagNumber(5)
  void clearSurface() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get onSurface => $_getSZ(5);
  @$pb.TagNumber(6)
  set onSurface($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasOnSurface() => $_has(5);
  @$pb.TagNumber(6)
  void clearOnSurface() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get error => $_getSZ(6);
  @$pb.TagNumber(7)
  set error($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasError() => $_has(6);
  @$pb.TagNumber(7)
  void clearError() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get onError => $_getSZ(7);
  @$pb.TagNumber(8)
  set onError($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasOnError() => $_has(7);
  @$pb.TagNumber(8)
  void clearOnError() => $_clearField(8);
}

/// ThemeTypography selects one of the curated embedded fonts and a global
/// size multiplier.
class ThemeTypography extends $pb.GeneratedMessage {
  factory ThemeTypography({
    $core.String? fontFamily,
    $core.double? scale,
  }) {
    final result = create();
    if (fontFamily != null) result.fontFamily = fontFamily;
    if (scale != null) result.scale = scale;
    return result;
  }

  ThemeTypography._();

  factory ThemeTypography.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ThemeTypography.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ThemeTypography',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'moth.projectconfig.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'fontFamily')
    ..aD(2, _omitFieldNames ? '' : 'scale')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ThemeTypography clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ThemeTypography copyWith(void Function(ThemeTypography) updates) =>
      super.copyWith((message) => updates(message as ThemeTypography))
          as ThemeTypography;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ThemeTypography create() => ThemeTypography._();
  @$core.override
  ThemeTypography createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ThemeTypography getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ThemeTypography>(create);
  static ThemeTypography? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get fontFamily => $_getSZ(0);
  @$pb.TagNumber(1)
  set fontFamily($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasFontFamily() => $_has(0);
  @$pb.TagNumber(1)
  void clearFontFamily() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.double get scale => $_getN(1);
  @$pb.TagNumber(2)
  set scale($core.double value) => $_setDouble(1, value);
  @$pb.TagNumber(2)
  $core.bool hasScale() => $_has(1);
  @$pb.TagNumber(2)
  void clearScale() => $_clearField(2);
}

/// ThemeSpacing is the base spacing grid step in logical pixels.
class ThemeSpacing extends $pb.GeneratedMessage {
  factory ThemeSpacing({
    $core.int? unit,
  }) {
    final result = create();
    if (unit != null) result.unit = unit;
    return result;
  }

  ThemeSpacing._();

  factory ThemeSpacing.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ThemeSpacing.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ThemeSpacing',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'moth.projectconfig.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'unit')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ThemeSpacing clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ThemeSpacing copyWith(void Function(ThemeSpacing) updates) =>
      super.copyWith((message) => updates(message as ThemeSpacing))
          as ThemeSpacing;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ThemeSpacing create() => ThemeSpacing._();
  @$core.override
  ThemeSpacing createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ThemeSpacing getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ThemeSpacing>(create);
  static ThemeSpacing? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get unit => $_getIZ(0);
  @$pb.TagNumber(1)
  set unit($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasUnit() => $_has(0);
  @$pb.TagNumber(1)
  void clearUnit() => $_clearField(1);
}

/// ThemeShape controls component rounding, in logical pixels.
class ThemeShape extends $pb.GeneratedMessage {
  factory ThemeShape({
    $core.int? cornerRadius,
  }) {
    final result = create();
    if (cornerRadius != null) result.cornerRadius = cornerRadius;
    return result;
  }

  ThemeShape._();

  factory ThemeShape.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ThemeShape.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ThemeShape',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'moth.projectconfig.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'cornerRadius')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ThemeShape clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ThemeShape copyWith(void Function(ThemeShape) updates) =>
      super.copyWith((message) => updates(message as ThemeShape)) as ThemeShape;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ThemeShape create() => ThemeShape._();
  @$core.override
  ThemeShape createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ThemeShape getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ThemeShape>(create);
  static ThemeShape? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get cornerRadius => $_getIZ(0);
  @$pb.TagNumber(1)
  set cornerRadius($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCornerRadius() => $_has(0);
  @$pb.TagNumber(1)
  void clearCornerRadius() => $_clearField(1);
}

/// ThemeLogo holds the server-managed asset paths of the uploaded logos,
/// one per color scheme ("/assets/{project}/logo-light.png"). Empty = none.
class ThemeLogo extends $pb.GeneratedMessage {
  factory ThemeLogo({
    $core.String? light,
    $core.String? dark,
  }) {
    final result = create();
    if (light != null) result.light = light;
    if (dark != null) result.dark = dark;
    return result;
  }

  ThemeLogo._();

  factory ThemeLogo.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ThemeLogo.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ThemeLogo',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'moth.projectconfig.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'light')
    ..aOS(2, _omitFieldNames ? '' : 'dark')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ThemeLogo clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ThemeLogo copyWith(void Function(ThemeLogo) updates) =>
      super.copyWith((message) => updates(message as ThemeLogo)) as ThemeLogo;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ThemeLogo create() => ThemeLogo._();
  @$core.override
  ThemeLogo createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ThemeLogo getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ThemeLogo>(create);
  static ThemeLogo? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get light => $_getSZ(0);
  @$pb.TagNumber(1)
  set light($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasLight() => $_has(0);
  @$pb.TagNumber(1)
  void clearLight() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get dark => $_getSZ(1);
  @$pb.TagNumber(2)
  set dark($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDark() => $_has(1);
  @$pb.TagNumber(2)
  void clearDark() => $_clearField(2);
}

/// StoredTheme is one project's complete design system as persisted on the
/// project row and in theme_revisions (milestone 06, re-encoded from JSON to
/// protobuf). internal/theme owns validation and dark-palette derivation;
/// this message owns only the shape.
class StoredTheme extends $pb.GeneratedMessage {
  factory StoredTheme({
    $core.int? version,
    ThemeColors? colors,
    ThemeColorOverrides? darkColors,
    ThemeTypography? typography,
    ThemeSpacing? spacing,
    ThemeShape? shape,
    ThemeLogo? logo,
    LegalLinks? legal,
  }) {
    final result = create();
    if (version != null) result.version = version;
    if (colors != null) result.colors = colors;
    if (darkColors != null) result.darkColors = darkColors;
    if (typography != null) result.typography = typography;
    if (spacing != null) result.spacing = spacing;
    if (shape != null) result.shape = shape;
    if (logo != null) result.logo = logo;
    if (legal != null) result.legal = legal;
    return result;
  }

  StoredTheme._();

  factory StoredTheme.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StoredTheme.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StoredTheme',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'moth.projectconfig.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'version')
    ..aOM<ThemeColors>(2, _omitFieldNames ? '' : 'colors',
        subBuilder: ThemeColors.create)
    ..aOM<ThemeColorOverrides>(3, _omitFieldNames ? '' : 'darkColors',
        subBuilder: ThemeColorOverrides.create)
    ..aOM<ThemeTypography>(4, _omitFieldNames ? '' : 'typography',
        subBuilder: ThemeTypography.create)
    ..aOM<ThemeSpacing>(5, _omitFieldNames ? '' : 'spacing',
        subBuilder: ThemeSpacing.create)
    ..aOM<ThemeShape>(6, _omitFieldNames ? '' : 'shape',
        subBuilder: ThemeShape.create)
    ..aOM<ThemeLogo>(7, _omitFieldNames ? '' : 'logo',
        subBuilder: ThemeLogo.create)
    ..aOM<LegalLinks>(8, _omitFieldNames ? '' : 'legal',
        subBuilder: LegalLinks.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StoredTheme clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StoredTheme copyWith(void Function(StoredTheme) updates) =>
      super.copyWith((message) => updates(message as StoredTheme))
          as StoredTheme;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StoredTheme create() => StoredTheme._();
  @$core.override
  StoredTheme createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StoredTheme getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StoredTheme>(create);
  static StoredTheme? _defaultInstance;

  /// version is the document schema version (internal/theme.SchemaVersion).
  @$pb.TagNumber(1)
  $core.int get version => $_getIZ(0);
  @$pb.TagNumber(1)
  set version($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  ThemeColors get colors => $_getN(1);
  @$pb.TagNumber(2)
  set colors(ThemeColors value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasColors() => $_has(1);
  @$pb.TagNumber(2)
  void clearColors() => $_clearField(2);
  @$pb.TagNumber(2)
  ThemeColors ensureColors() => $_ensure(1);

  /// dark_colors optionally overrides individual dark-palette colors;
  /// absent = fully derived from colors.
  @$pb.TagNumber(3)
  ThemeColorOverrides get darkColors => $_getN(2);
  @$pb.TagNumber(3)
  set darkColors(ThemeColorOverrides value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasDarkColors() => $_has(2);
  @$pb.TagNumber(3)
  void clearDarkColors() => $_clearField(3);
  @$pb.TagNumber(3)
  ThemeColorOverrides ensureDarkColors() => $_ensure(2);

  @$pb.TagNumber(4)
  ThemeTypography get typography => $_getN(3);
  @$pb.TagNumber(4)
  set typography(ThemeTypography value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasTypography() => $_has(3);
  @$pb.TagNumber(4)
  void clearTypography() => $_clearField(4);
  @$pb.TagNumber(4)
  ThemeTypography ensureTypography() => $_ensure(3);

  @$pb.TagNumber(5)
  ThemeSpacing get spacing => $_getN(4);
  @$pb.TagNumber(5)
  set spacing(ThemeSpacing value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasSpacing() => $_has(4);
  @$pb.TagNumber(5)
  void clearSpacing() => $_clearField(5);
  @$pb.TagNumber(5)
  ThemeSpacing ensureSpacing() => $_ensure(4);

  @$pb.TagNumber(6)
  ThemeShape get shape => $_getN(5);
  @$pb.TagNumber(6)
  set shape(ThemeShape value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasShape() => $_has(5);
  @$pb.TagNumber(6)
  void clearShape() => $_clearField(6);
  @$pb.TagNumber(6)
  ThemeShape ensureShape() => $_ensure(5);

  @$pb.TagNumber(7)
  ThemeLogo get logo => $_getN(6);
  @$pb.TagNumber(7)
  set logo(ThemeLogo value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasLogo() => $_has(6);
  @$pb.TagNumber(7)
  void clearLogo() => $_clearField(7);
  @$pb.TagNumber(7)
  ThemeLogo ensureLogo() => $_ensure(6);

  @$pb.TagNumber(8)
  LegalLinks get legal => $_getN(7);
  @$pb.TagNumber(8)
  set legal(LegalLinks value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasLegal() => $_has(7);
  @$pb.TagNumber(8)
  void clearLegal() => $_clearField(8);
  @$pb.TagNumber(8)
  LegalLinks ensureLegal() => $_ensure(7);
}

/// StoredPaywall is one project's paywall configuration as persisted on the
/// project row and in paywall_revisions (milestone 13, re-encoded from JSON
/// to protobuf). Colors/typography always inherit from the theme — the
/// paywall owns no design tokens.
class StoredPaywall extends $pb.GeneratedMessage {
  factory StoredPaywall({
    $core.int? version,
    $core.String? headline,
    $core.String? subtitle,
    $core.Iterable<$core.String>? benefits,
    $core.String? offering,
    $core.String? highlightedIdentifier,
    $core.String? layout,
    LegalLinks? legal,
  }) {
    final result = create();
    if (version != null) result.version = version;
    if (headline != null) result.headline = headline;
    if (subtitle != null) result.subtitle = subtitle;
    if (benefits != null) result.benefits.addAll(benefits);
    if (offering != null) result.offering = offering;
    if (highlightedIdentifier != null)
      result.highlightedIdentifier = highlightedIdentifier;
    if (layout != null) result.layout = layout;
    if (legal != null) result.legal = legal;
    return result;
  }

  StoredPaywall._();

  factory StoredPaywall.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StoredPaywall.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StoredPaywall',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'moth.projectconfig.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'version')
    ..aOS(2, _omitFieldNames ? '' : 'headline')
    ..aOS(3, _omitFieldNames ? '' : 'subtitle')
    ..pPS(4, _omitFieldNames ? '' : 'benefits')
    ..aOS(5, _omitFieldNames ? '' : 'offering')
    ..aOS(6, _omitFieldNames ? '' : 'highlightedIdentifier')
    ..aOS(7, _omitFieldNames ? '' : 'layout')
    ..aOM<LegalLinks>(8, _omitFieldNames ? '' : 'legal',
        subBuilder: LegalLinks.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StoredPaywall clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StoredPaywall copyWith(void Function(StoredPaywall) updates) =>
      super.copyWith((message) => updates(message as StoredPaywall))
          as StoredPaywall;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StoredPaywall create() => StoredPaywall._();
  @$core.override
  StoredPaywall createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StoredPaywall getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StoredPaywall>(create);
  static StoredPaywall? _defaultInstance;

  /// version is the document schema version (internal/paywall.SchemaVersion).
  @$pb.TagNumber(1)
  $core.int get version => $_getIZ(0);
  @$pb.TagNumber(1)
  set version($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get headline => $_getSZ(1);
  @$pb.TagNumber(2)
  set headline($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasHeadline() => $_has(1);
  @$pb.TagNumber(2)
  void clearHeadline() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get subtitle => $_getSZ(2);
  @$pb.TagNumber(3)
  set subtitle($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSubtitle() => $_has(2);
  @$pb.TagNumber(3)
  void clearSubtitle() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<$core.String> get benefits => $_getList(3);

  /// offering names the product offering the paywall presents; empty = the
  /// project's default offering.
  @$pb.TagNumber(5)
  $core.String get offering => $_getSZ(4);
  @$pb.TagNumber(5)
  set offering($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasOffering() => $_has(4);
  @$pb.TagNumber(5)
  void clearOffering() => $_clearField(5);

  /// highlighted_identifier marks the "most popular" tier; empty = none.
  @$pb.TagNumber(6)
  $core.String get highlightedIdentifier => $_getSZ(5);
  @$pb.TagNumber(6)
  set highlightedIdentifier($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasHighlightedIdentifier() => $_has(5);
  @$pb.TagNumber(6)
  void clearHighlightedIdentifier() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get layout => $_getSZ(6);
  @$pb.TagNumber(7)
  set layout($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasLayout() => $_has(6);
  @$pb.TagNumber(7)
  void clearLayout() => $_clearField(7);

  @$pb.TagNumber(8)
  LegalLinks get legal => $_getN(7);
  @$pb.TagNumber(8)
  set legal(LegalLinks value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasLegal() => $_has(7);
  @$pb.TagNumber(8)
  void clearLegal() => $_clearField(8);
  @$pb.TagNumber(8)
  LegalLinks ensureLegal() => $_ensure(7);
}

/// StoredPush is one project's push settings as persisted on the project row
/// (milestone 20). Plain config, no secrets: only the Web Push VAPID PUBLIC
/// key ever lives here — the private key stays with the developer's sender
/// and never touches moth. Delivered to clients through the public
/// moth.auth.v1.GetProjectConfig response.
class StoredPush extends $pb.GeneratedMessage {
  factory StoredPush({
    $core.int? version,
    $core.bool? enabled,
    $core.String? webpushVapidPublicKey,
  }) {
    final result = create();
    if (version != null) result.version = version;
    if (enabled != null) result.enabled = enabled;
    if (webpushVapidPublicKey != null)
      result.webpushVapidPublicKey = webpushVapidPublicKey;
    return result;
  }

  StoredPush._();

  factory StoredPush.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StoredPush.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StoredPush',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'moth.projectconfig.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'version')
    ..aOB(2, _omitFieldNames ? '' : 'enabled')
    ..aOS(3, _omitFieldNames ? '' : 'webpushVapidPublicKey')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StoredPush clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StoredPush copyWith(void Function(StoredPush) updates) =>
      super.copyWith((message) => updates(message as StoredPush)) as StoredPush;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StoredPush create() => StoredPush._();
  @$core.override
  StoredPush createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StoredPush getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StoredPush>(create);
  static StoredPush? _defaultInstance;

  /// version is the document schema version (internal/push.SchemaVersion).
  @$pb.TagNumber(1)
  $core.int get version => $_getIZ(0);
  @$pb.TagNumber(1)
  set version($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearVersion() => $_clearField(1);

  /// Master switch for the push registry; when false the client-facing
  /// moth.push.v1 RPCs refuse registrations.
  @$pb.TagNumber(2)
  $core.bool get enabled => $_getBF(1);
  @$pb.TagNumber(2)
  set enabled($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEnabled() => $_has(1);
  @$pb.TagNumber(2)
  void clearEnabled() => $_clearField(2);

  /// VAPID public key (base64url, uncompressed P-256 point) browser clients
  /// subscribe with; empty when the project does not use Web Push.
  @$pb.TagNumber(3)
  $core.String get webpushVapidPublicKey => $_getSZ(2);
  @$pb.TagNumber(3)
  set webpushVapidPublicKey($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasWebpushVapidPublicKey() => $_has(2);
  @$pb.TagNumber(3)
  void clearWebpushVapidPublicKey() => $_clearField(3);
}

/// StoredProfile is one project's setup profile as persisted on the project
/// row (milestone 22): the creation wizard's answers — platforms, sign-in
/// intent, monetization and push intent. It records what the app *intends*,
/// so surfaces can tell "doesn't want Apple sign-in" apart from "hasn't
/// configured it yet"; it is never a second source of config truth. Absent
/// (empty column) on projects created before the wizard, where surfaces
/// behave exactly as before.
class StoredProfile extends $pb.GeneratedMessage {
  factory StoredProfile({
    $core.int? version,
    $core.Iterable<Platform>? platforms,
    $core.bool? googleSignIn,
    $core.bool? appleSignIn,
    $core.bool? sellsSubscriptions,
    $core.bool? sendsPushes,
    $core.bool? checklistDismissed,
  }) {
    final result = create();
    if (version != null) result.version = version;
    if (platforms != null) result.platforms.addAll(platforms);
    if (googleSignIn != null) result.googleSignIn = googleSignIn;
    if (appleSignIn != null) result.appleSignIn = appleSignIn;
    if (sellsSubscriptions != null)
      result.sellsSubscriptions = sellsSubscriptions;
    if (sendsPushes != null) result.sendsPushes = sendsPushes;
    if (checklistDismissed != null)
      result.checklistDismissed = checklistDismissed;
    return result;
  }

  StoredProfile._();

  factory StoredProfile.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StoredProfile.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StoredProfile',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'moth.projectconfig.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'version')
    ..pc<Platform>(2, _omitFieldNames ? '' : 'platforms', $pb.PbFieldType.KE,
        valueOf: Platform.valueOf,
        enumValues: Platform.values,
        defaultEnumValue: Platform.PLATFORM_UNSPECIFIED)
    ..aOB(3, _omitFieldNames ? '' : 'googleSignIn')
    ..aOB(4, _omitFieldNames ? '' : 'appleSignIn')
    ..aOB(5, _omitFieldNames ? '' : 'sellsSubscriptions')
    ..aOB(6, _omitFieldNames ? '' : 'sendsPushes')
    ..aOB(7, _omitFieldNames ? '' : 'checklistDismissed')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StoredProfile clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StoredProfile copyWith(void Function(StoredProfile) updates) =>
      super.copyWith((message) => updates(message as StoredProfile))
          as StoredProfile;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StoredProfile create() => StoredProfile._();
  @$core.override
  StoredProfile createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StoredProfile getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StoredProfile>(create);
  static StoredProfile? _defaultInstance;

  /// version is the document schema version (internal/profile.SchemaVersion).
  @$pb.TagNumber(1)
  $core.int get version => $_getIZ(0);
  @$pb.TagNumber(1)
  set version($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearVersion() => $_clearField(1);

  /// platforms the app ships on. Non-empty in every valid profile; drives
  /// every platform branch (store credentials, VAPID, setup snippets).
  @$pb.TagNumber(2)
  $pb.PbList<Platform> get platforms => $_getList(1);

  /// google_sign_in / apple_sign_in record the social sign-in intent.
  /// Email/password is always on and needs no flag.
  @$pb.TagNumber(3)
  $core.bool get googleSignIn => $_getBF(2);
  @$pb.TagNumber(3)
  set googleSignIn($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasGoogleSignIn() => $_has(2);
  @$pb.TagNumber(3)
  void clearGoogleSignIn() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get appleSignIn => $_getBF(3);
  @$pb.TagNumber(4)
  set appleSignIn($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasAppleSignIn() => $_has(3);
  @$pb.TagNumber(4)
  void clearAppleSignIn() => $_clearField(4);

  /// sells_subscriptions records the monetization intent (milestones 11/12).
  @$pb.TagNumber(5)
  $core.bool get sellsSubscriptions => $_getBF(4);
  @$pb.TagNumber(5)
  set sellsSubscriptions($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSellsSubscriptions() => $_has(4);
  @$pb.TagNumber(5)
  void clearSellsSubscriptions() => $_clearField(5);

  /// sends_pushes records the push intent (milestone 20).
  @$pb.TagNumber(6)
  $core.bool get sendsPushes => $_getBF(5);
  @$pb.TagNumber(6)
  set sendsPushes($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasSendsPushes() => $_has(5);
  @$pb.TagNumber(6)
  void clearSendsPushes() => $_clearField(6);

  /// checklist_dismissed hides the overview checklist card; it never fakes
  /// completeness — the derived items stay computable either way.
  @$pb.TagNumber(7)
  $core.bool get checklistDismissed => $_getBF(6);
  @$pb.TagNumber(7)
  set checklistDismissed($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasChecklistDismissed() => $_has(6);
  @$pb.TagNumber(7)
  void clearChecklistDismissed() => $_clearField(7);
}

/// CopyLocaleMessages is one locale's copy overrides: catalog message key
/// (e.g. "sign_in.title") to the operator-customized string.
class CopyLocaleMessages extends $pb.GeneratedMessage {
  factory CopyLocaleMessages({
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? messages,
  }) {
    final result = create();
    if (messages != null) result.messages.addEntries(messages);
    return result;
  }

  CopyLocaleMessages._();

  factory CopyLocaleMessages.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CopyLocaleMessages.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CopyLocaleMessages',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'moth.projectconfig.v1'),
      createEmptyInstance: create)
    ..m<$core.String, $core.String>(1, _omitFieldNames ? '' : 'messages',
        entryClassName: 'CopyLocaleMessages.MessagesEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('moth.projectconfig.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CopyLocaleMessages clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CopyLocaleMessages copyWith(void Function(CopyLocaleMessages) updates) =>
      super.copyWith((message) => updates(message as CopyLocaleMessages))
          as CopyLocaleMessages;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CopyLocaleMessages create() => CopyLocaleMessages._();
  @$core.override
  CopyLocaleMessages createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CopyLocaleMessages getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CopyLocaleMessages>(create);
  static CopyLocaleMessages? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbMap<$core.String, $core.String> get messages => $_getMap(0);
}

/// StoredCopy is one project's localization overrides as persisted on the
/// project row and in copy_revisions (milestone 15, re-encoded from JSON to
/// protobuf): BCP-47 locale tag to that locale's key overrides. Bundled
/// catalog defaults live in the binary (internal/i18n), never here.
class StoredCopy extends $pb.GeneratedMessage {
  factory StoredCopy({
    $core.Iterable<$core.MapEntry<$core.String, CopyLocaleMessages>>? locales,
  }) {
    final result = create();
    if (locales != null) result.locales.addEntries(locales);
    return result;
  }

  StoredCopy._();

  factory StoredCopy.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StoredCopy.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StoredCopy',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'moth.projectconfig.v1'),
      createEmptyInstance: create)
    ..m<$core.String, CopyLocaleMessages>(1, _omitFieldNames ? '' : 'locales',
        entryClassName: 'StoredCopy.LocalesEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OM,
        valueCreator: CopyLocaleMessages.create,
        valueDefaultOrMaker: CopyLocaleMessages.getDefault,
        packageName: const $pb.PackageName('moth.projectconfig.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StoredCopy clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StoredCopy copyWith(void Function(StoredCopy) updates) =>
      super.copyWith((message) => updates(message as StoredCopy)) as StoredCopy;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StoredCopy create() => StoredCopy._();
  @$core.override
  StoredCopy createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StoredCopy getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StoredCopy>(create);
  static StoredCopy? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbMap<$core.String, CopyLocaleMessages> get locales => $_getMap(0);
}

/// CacheEnvelope wraps a config payload the Flutter SDK persists on device
/// (theme, paywall, copy — milestone 16 caches, re-encoded from JSON to
/// protobuf). payload is the serialized wire message exactly as the server
/// delivered it (moth.auth.v1.Theme / moth.billing.v1.Paywall /
/// moth.auth.v1.Copy), so the cache and the wire share one schema. The SDK
/// serves the cached payload without any network call until
/// fetched_at_unix_ms + its configured TTL has passed, then revalidates
/// cheaply with the known_*_revision request fields.
class CacheEnvelope extends $pb.GeneratedMessage {
  factory CacheEnvelope({
    $core.List<$core.int>? payload,
    $core.String? revision,
    $core.String? locale,
    $fixnum.Int64? fetchedAtUnixMs,
  }) {
    final result = create();
    if (payload != null) result.payload = payload;
    if (revision != null) result.revision = revision;
    if (locale != null) result.locale = locale;
    if (fetchedAtUnixMs != null) result.fetchedAtUnixMs = fetchedAtUnixMs;
    return result;
  }

  CacheEnvelope._();

  factory CacheEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CacheEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CacheEnvelope',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'moth.projectconfig.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'payload', $pb.PbFieldType.OY)
    ..aOS(2, _omitFieldNames ? '' : 'revision')
    ..aOS(3, _omitFieldNames ? '' : 'locale')
    ..aInt64(4, _omitFieldNames ? '' : 'fetchedAtUnixMs')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CacheEnvelope clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CacheEnvelope copyWith(void Function(CacheEnvelope) updates) =>
      super.copyWith((message) => updates(message as CacheEnvelope))
          as CacheEnvelope;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CacheEnvelope create() => CacheEnvelope._();
  @$core.override
  CacheEnvelope createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CacheEnvelope getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CacheEnvelope>(create);
  static CacheEnvelope? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get payload => $_getN(0);
  @$pb.TagNumber(1)
  set payload($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPayload() => $_has(0);
  @$pb.TagNumber(1)
  void clearPayload() => $_clearField(1);

  /// revision is the server revision the payload came from
  /// (theme/paywall/copy revision id) — the revalidation key.
  @$pb.TagNumber(2)
  $core.String get revision => $_getSZ(1);
  @$pb.TagNumber(2)
  set revision($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearRevision() => $_clearField(2);

  /// locale is the negotiated BCP-47 tag for locale-keyed payloads (copy);
  /// empty for locale-independent payloads.
  @$pb.TagNumber(3)
  $core.String get locale => $_getSZ(2);
  @$pb.TagNumber(3)
  set locale($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasLocale() => $_has(2);
  @$pb.TagNumber(3)
  void clearLocale() => $_clearField(3);

  /// fetched_at_unix_ms is when the payload was fetched or last revalidated,
  /// Unix milliseconds UTC.
  @$pb.TagNumber(4)
  $fixnum.Int64 get fetchedAtUnixMs => $_getI64(3);
  @$pb.TagNumber(4)
  set fetchedAtUnixMs($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasFetchedAtUnixMs() => $_has(3);
  @$pb.TagNumber(4)
  void clearFetchedAtUnixMs() => $_clearField(4);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
