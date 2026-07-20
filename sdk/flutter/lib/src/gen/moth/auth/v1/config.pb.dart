// This is a generated file - do not edit.
//
// Generated from moth/auth/v1/config.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

/// GoogleConfig is the public part of a project's Sign in with Google
/// configuration.
class GoogleConfig extends $pb.GeneratedMessage {
  factory GoogleConfig({
    $core.bool? enabled,
    $core.String? webClientId,
    $core.String? iosClientId,
    $core.String? androidClientId,
  }) {
    final result = create();
    if (enabled != null) result.enabled = enabled;
    if (webClientId != null) result.webClientId = webClientId;
    if (iosClientId != null) result.iosClientId = iosClientId;
    if (androidClientId != null) result.androidClientId = androidClientId;
    return result;
  }

  GoogleConfig._();

  factory GoogleConfig.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GoogleConfig.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GoogleConfig',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.auth.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'enabled')
    ..aOS(2, _omitFieldNames ? '' : 'webClientId')
    ..aOS(3, _omitFieldNames ? '' : 'iosClientId')
    ..aOS(4, _omitFieldNames ? '' : 'androidClientId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GoogleConfig clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GoogleConfig copyWith(void Function(GoogleConfig) updates) =>
      super.copyWith((message) => updates(message as GoogleConfig))
          as GoogleConfig;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GoogleConfig create() => GoogleConfig._();
  @$core.override
  GoogleConfig createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GoogleConfig getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GoogleConfig>(create);
  static GoogleConfig? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get enabled => $_getBF(0);
  @$pb.TagNumber(1)
  set enabled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEnabled() => $_has(0);
  @$pb.TagNumber(1)
  void clearEnabled() => $_clearField(1);

  /// OAuth client IDs the native flows initialize with. Client IDs are
  /// public values (the secret never leaves the server).
  @$pb.TagNumber(2)
  $core.String get webClientId => $_getSZ(1);
  @$pb.TagNumber(2)
  set webClientId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasWebClientId() => $_has(1);
  @$pb.TagNumber(2)
  void clearWebClientId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get iosClientId => $_getSZ(2);
  @$pb.TagNumber(3)
  set iosClientId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasIosClientId() => $_has(2);
  @$pb.TagNumber(3)
  void clearIosClientId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get androidClientId => $_getSZ(3);
  @$pb.TagNumber(4)
  set androidClientId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasAndroidClientId() => $_has(3);
  @$pb.TagNumber(4)
  void clearAndroidClientId() => $_clearField(4);
}

/// AppleConfig is the public part of a project's Sign in with Apple
/// configuration.
class AppleConfig extends $pb.GeneratedMessage {
  factory AppleConfig({
    $core.bool? enabled,
  }) {
    final result = create();
    if (enabled != null) result.enabled = enabled;
    return result;
  }

  AppleConfig._();

  factory AppleConfig.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AppleConfig.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AppleConfig',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.auth.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'enabled')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AppleConfig clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AppleConfig copyWith(void Function(AppleConfig) updates) =>
      super.copyWith((message) => updates(message as AppleConfig))
          as AppleConfig;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AppleConfig create() => AppleConfig._();
  @$core.override
  AppleConfig createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AppleConfig getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AppleConfig>(create);
  static AppleConfig? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get enabled => $_getBF(0);
  @$pb.TagNumber(1)
  set enabled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEnabled() => $_has(0);
  @$pb.TagNumber(1)
  void clearEnabled() => $_clearField(1);
}

/// PushConfig is the public part of a project's push-notification
/// configuration (milestone 20): everything the SDK needs to register the
/// device with moth's push registry. Public values only — the Web Push VAPID
/// public key is designed to be embedded in clients; the matching private key
/// stays with the developer's sender and never touches moth.
class PushConfig extends $pb.GeneratedMessage {
  factory PushConfig({
    $core.bool? enabled,
    $core.String? webpushVapidPublicKey,
  }) {
    final result = create();
    if (enabled != null) result.enabled = enabled;
    if (webpushVapidPublicKey != null)
      result.webpushVapidPublicKey = webpushVapidPublicKey;
    return result;
  }

  PushConfig._();

  factory PushConfig.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PushConfig.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PushConfig',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.auth.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'enabled')
    ..aOS(2, _omitFieldNames ? '' : 'webpushVapidPublicKey')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PushConfig clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PushConfig copyWith(void Function(PushConfig) updates) =>
      super.copyWith((message) => updates(message as PushConfig)) as PushConfig;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PushConfig create() => PushConfig._();
  @$core.override
  PushConfig createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PushConfig getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PushConfig>(create);
  static PushConfig? _defaultInstance;

  /// Whether the project accepts device registrations (moth.push.v1).
  @$pb.TagNumber(1)
  $core.bool get enabled => $_getBF(0);
  @$pb.TagNumber(1)
  set enabled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEnabled() => $_has(0);
  @$pb.TagNumber(1)
  void clearEnabled() => $_clearField(1);

  /// VAPID public key (base64url, uncompressed P-256 point) the browser SDK
  /// passes as `applicationServerKey` when subscribing; empty when the
  /// project does not use Web Push.
  @$pb.TagNumber(2)
  $core.String get webpushVapidPublicKey => $_getSZ(1);
  @$pb.TagNumber(2)
  set webpushVapidPublicKey($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasWebpushVapidPublicKey() => $_has(1);
  @$pb.TagNumber(2)
  void clearWebpushVapidPublicKey() => $_clearField(2);
}

/// Theme is the public, fully resolved form of the project's design system,
/// ready to render: dark colors are already derived server-side, asset
/// references are absolute URLs. Binary assets (logo images, font files)
/// stay plain-HTTP downloads with cache headers — they don't belong in RPC
/// responses.
class Theme extends $pb.GeneratedMessage {
  factory Theme({
    $core.String? revisionId,
    ThemeColors? colors,
    ThemeColors? darkColors,
    $core.String? fontFamily,
    $core.String? fontUrl,
    $core.double? fontScale,
    $core.int? spacingUnit,
    $core.int? cornerRadius,
    $core.String? logoLightUrl,
    $core.String? logoDarkUrl,
    $core.String? termsUrl,
    $core.String? privacyUrl,
  }) {
    final result = create();
    if (revisionId != null) result.revisionId = revisionId;
    if (colors != null) result.colors = colors;
    if (darkColors != null) result.darkColors = darkColors;
    if (fontFamily != null) result.fontFamily = fontFamily;
    if (fontUrl != null) result.fontUrl = fontUrl;
    if (fontScale != null) result.fontScale = fontScale;
    if (spacingUnit != null) result.spacingUnit = spacingUnit;
    if (cornerRadius != null) result.cornerRadius = cornerRadius;
    if (logoLightUrl != null) result.logoLightUrl = logoLightUrl;
    if (logoDarkUrl != null) result.logoDarkUrl = logoDarkUrl;
    if (termsUrl != null) result.termsUrl = termsUrl;
    if (privacyUrl != null) result.privacyUrl = privacyUrl;
    return result;
  }

  Theme._();

  factory Theme.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Theme.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Theme',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'revisionId')
    ..aOM<ThemeColors>(2, _omitFieldNames ? '' : 'colors',
        subBuilder: ThemeColors.create)
    ..aOM<ThemeColors>(3, _omitFieldNames ? '' : 'darkColors',
        subBuilder: ThemeColors.create)
    ..aOS(4, _omitFieldNames ? '' : 'fontFamily')
    ..aOS(5, _omitFieldNames ? '' : 'fontUrl')
    ..aD(6, _omitFieldNames ? '' : 'fontScale')
    ..aI(7, _omitFieldNames ? '' : 'spacingUnit')
    ..aI(8, _omitFieldNames ? '' : 'cornerRadius')
    ..aOS(9, _omitFieldNames ? '' : 'logoLightUrl')
    ..aOS(10, _omitFieldNames ? '' : 'logoDarkUrl')
    ..aOS(11, _omitFieldNames ? '' : 'termsUrl')
    ..aOS(12, _omitFieldNames ? '' : 'privacyUrl')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Theme clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Theme copyWith(void Function(Theme) updates) =>
      super.copyWith((message) => updates(message as Theme)) as Theme;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Theme create() => Theme._();
  @$core.override
  Theme createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Theme getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Theme>(create);
  static Theme? _defaultInstance;

  /// Identifies this version of the theme; changes on every admin edit.
  /// Cache the theme keyed by this value and echo it as
  /// GetProjectConfigRequest.known_theme_revision.
  @$pb.TagNumber(1)
  $core.String get revisionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set revisionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRevisionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRevisionId() => $_clearField(1);

  /// Light palette, "#RRGGBB" values.
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

  /// Dark palette, fully resolved (admin overrides merged with derived
  /// values); render it when the device is in dark mode.
  @$pb.TagNumber(3)
  ThemeColors get darkColors => $_getN(2);
  @$pb.TagNumber(3)
  set darkColors(ThemeColors value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasDarkColors() => $_has(2);
  @$pb.TagNumber(3)
  void clearDarkColors() => $_clearField(3);
  @$pb.TagNumber(3)
  ThemeColors ensureDarkColors() => $_ensure(2);

  /// Font family name (from the server's curated set).
  @$pb.TagNumber(4)
  $core.String get fontFamily => $_getSZ(3);
  @$pb.TagNumber(4)
  set fontFamily($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasFontFamily() => $_has(3);
  @$pb.TagNumber(4)
  void clearFontFamily() => $_clearField(4);

  /// Absolute URL of the font file to download and register; cacheable.
  @$pb.TagNumber(5)
  $core.String get fontUrl => $_getSZ(4);
  @$pb.TagNumber(5)
  set fontUrl($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasFontUrl() => $_has(4);
  @$pb.TagNumber(5)
  void clearFontUrl() => $_clearField(5);

  /// Global text-size multiplier.
  @$pb.TagNumber(6)
  $core.double get fontScale => $_getN(5);
  @$pb.TagNumber(6)
  set fontScale($core.double value) => $_setDouble(5, value);
  @$pb.TagNumber(6)
  $core.bool hasFontScale() => $_has(5);
  @$pb.TagNumber(6)
  void clearFontScale() => $_clearField(6);

  /// Base spacing step in logical pixels.
  @$pb.TagNumber(7)
  $core.int get spacingUnit => $_getIZ(6);
  @$pb.TagNumber(7)
  set spacingUnit($core.int value) => $_setSignedInt32(6, value);
  @$pb.TagNumber(7)
  $core.bool hasSpacingUnit() => $_has(6);
  @$pb.TagNumber(7)
  void clearSpacingUnit() => $_clearField(7);

  /// Component corner radius in logical pixels.
  @$pb.TagNumber(8)
  $core.int get cornerRadius => $_getIZ(7);
  @$pb.TagNumber(8)
  set cornerRadius($core.int value) => $_setSignedInt32(7, value);
  @$pb.TagNumber(8)
  $core.bool hasCornerRadius() => $_has(7);
  @$pb.TagNumber(8)
  void clearCornerRadius() => $_clearField(8);

  /// Absolute logo URLs per color scheme; empty when no logo is set.
  @$pb.TagNumber(9)
  $core.String get logoLightUrl => $_getSZ(8);
  @$pb.TagNumber(9)
  set logoLightUrl($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasLogoLightUrl() => $_has(8);
  @$pb.TagNumber(9)
  void clearLogoLightUrl() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.String get logoDarkUrl => $_getSZ(9);
  @$pb.TagNumber(10)
  set logoDarkUrl($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasLogoDarkUrl() => $_has(9);
  @$pb.TagNumber(10)
  void clearLogoDarkUrl() => $_clearField(10);

  /// Optional legal links rendered in the login screen footer.
  @$pb.TagNumber(11)
  $core.String get termsUrl => $_getSZ(10);
  @$pb.TagNumber(11)
  set termsUrl($core.String value) => $_setString(10, value);
  @$pb.TagNumber(11)
  $core.bool hasTermsUrl() => $_has(10);
  @$pb.TagNumber(11)
  void clearTermsUrl() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.String get privacyUrl => $_getSZ(11);
  @$pb.TagNumber(12)
  set privacyUrl($core.String value) => $_setString(11, value);
  @$pb.TagNumber(12)
  $core.bool hasPrivacyUrl() => $_has(11);
  @$pb.TagNumber(12)
  void clearPrivacyUrl() => $_clearField(12);
}

/// ThemeColors is a complete palette: each color role and its "on"
/// (foreground) counterpart. Server-side validation guarantees WCAG AA
/// contrast (>= 4.5:1) between every pair.
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
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.auth.v1'),
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

/// Copy is the resolved, localized copy for the negotiated locale: the message
/// key → localized-string map the SDK renders its auth screens from
/// (sign_in.*, sign_up.*, password_reset.*, verify_email.*), already merged
/// bundled-default → project-override. The locale is negotiated server-side
/// from the request's Accept-Language / x-moth-language metadata against the
/// project's available locales; the client never dictates raw copy.
class Copy extends $pb.GeneratedMessage {
  factory Copy({
    $core.String? copyRevision,
    $core.String? locale,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? messages,
  }) {
    final result = create();
    if (copyRevision != null) result.copyRevision = copyRevision;
    if (locale != null) result.locale = locale;
    if (messages != null) result.messages.addEntries(messages);
    return result;
  }

  Copy._();

  factory Copy.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Copy.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Copy',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'copyRevision')
    ..aOS(2, _omitFieldNames ? '' : 'locale')
    ..m<$core.String, $core.String>(3, _omitFieldNames ? '' : 'messages',
        entryClassName: 'Copy.MessagesEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('moth.auth.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Copy clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Copy copyWith(void Function(Copy) updates) =>
      super.copyWith((message) => updates(message as Copy)) as Copy;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Copy create() => Copy._();
  @$core.override
  Copy createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Copy getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Copy>(create);
  static Copy? _defaultInstance;

  /// Opaque cache token identifying this (locale, override-revision) pair. It
  /// changes whenever the negotiated locale or the project's copy overrides
  /// change. Cache `messages` keyed by this value and echo it as
  /// GetProjectConfigRequest.known_copy_revision; the response omits `messages`
  /// when it still matches (see the caching contract on the request).
  @$pb.TagNumber(1)
  $core.String get copyRevision => $_getSZ(0);
  @$pb.TagNumber(1)
  set copyRevision($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCopyRevision() => $_has(0);
  @$pb.TagNumber(1)
  void clearCopyRevision() => $_clearField(1);

  /// The negotiated BCP-47 locale this copy is for (e.g. "fr"). Echoed so the
  /// client sets lang/dir correctly and re-requests when the device language
  /// changes; always present even when `messages` is omitted.
  @$pb.TagNumber(2)
  $core.String get locale => $_getSZ(1);
  @$pb.TagNumber(2)
  set locale($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLocale() => $_has(1);
  @$pb.TagNumber(2)
  void clearLocale() => $_clearField(2);

  /// Resolved message key → localized string for the negotiated locale.
  @$pb.TagNumber(3)
  $pb.PbMap<$core.String, $core.String> get messages => $_getMap(2);
}

class GetProjectConfigRequest extends $pb.GeneratedMessage {
  factory GetProjectConfigRequest({
    $core.String? knownThemeRevision,
    $core.String? knownCopyRevision,
  }) {
    final result = create();
    if (knownThemeRevision != null)
      result.knownThemeRevision = knownThemeRevision;
    if (knownCopyRevision != null) result.knownCopyRevision = knownCopyRevision;
    return result;
  }

  GetProjectConfigRequest._();

  factory GetProjectConfigRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetProjectConfigRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetProjectConfigRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'knownThemeRevision')
    ..aOS(2, _omitFieldNames ? '' : 'knownCopyRevision')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProjectConfigRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProjectConfigRequest copyWith(
          void Function(GetProjectConfigRequest) updates) =>
      super.copyWith((message) => updates(message as GetProjectConfigRequest))
          as GetProjectConfigRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetProjectConfigRequest create() => GetProjectConfigRequest._();
  @$core.override
  GetProjectConfigRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetProjectConfigRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetProjectConfigRequest>(create);
  static GetProjectConfigRequest? _defaultInstance;

  /// Theme caching contract: pass the revision_id of the theme the client
  /// has cached (empty on first call). When it still matches the current
  /// revision, the response omits `theme` entirely — the client keeps
  /// rendering its cached copy. When it differs (or was empty), `theme` is
  /// present and the client replaces its cache.
  @$pb.TagNumber(1)
  $core.String get knownThemeRevision => $_getSZ(0);
  @$pb.TagNumber(1)
  set knownThemeRevision($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasKnownThemeRevision() => $_has(0);
  @$pb.TagNumber(1)
  void clearKnownThemeRevision() => $_clearField(1);

  /// Copy caching contract (identical shape to the theme one, but keyed by the
  /// negotiated locale too): pass the copy_revision the client has cached for
  /// the locale it is about to render (empty on first call). When it still
  /// matches the token the server computes for the negotiated locale, the
  /// response's `copy` carries the locale + copy_revision but omits `messages`
  /// (stale-while-revalidate); when it differs (or was empty), `messages` is
  /// present and the client replaces its cache. The negotiated locale comes
  /// from Accept-Language / x-moth-language metadata, never from this body.
  @$pb.TagNumber(2)
  $core.String get knownCopyRevision => $_getSZ(1);
  @$pb.TagNumber(2)
  set knownCopyRevision($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasKnownCopyRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearKnownCopyRevision() => $_clearField(2);
}

class GetProjectConfigResponse extends $pb.GeneratedMessage {
  factory GetProjectConfigResponse({
    GoogleConfig? google,
    AppleConfig? apple,
    $core.int? passwordMinLength,
    $core.bool? signUpOpen,
    Theme? theme,
    Copy? copy,
    PushConfig? push,
  }) {
    final result = create();
    if (google != null) result.google = google;
    if (apple != null) result.apple = apple;
    if (passwordMinLength != null) result.passwordMinLength = passwordMinLength;
    if (signUpOpen != null) result.signUpOpen = signUpOpen;
    if (theme != null) result.theme = theme;
    if (copy != null) result.copy = copy;
    if (push != null) result.push = push;
    return result;
  }

  GetProjectConfigResponse._();

  factory GetProjectConfigResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetProjectConfigResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetProjectConfigResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'moth.auth.v1'),
      createEmptyInstance: create)
    ..aOM<GoogleConfig>(1, _omitFieldNames ? '' : 'google',
        subBuilder: GoogleConfig.create)
    ..aOM<AppleConfig>(2, _omitFieldNames ? '' : 'apple',
        subBuilder: AppleConfig.create)
    ..aI(3, _omitFieldNames ? '' : 'passwordMinLength')
    ..aOB(4, _omitFieldNames ? '' : 'signUpOpen')
    ..aOM<Theme>(5, _omitFieldNames ? '' : 'theme', subBuilder: Theme.create)
    ..aOM<Copy>(6, _omitFieldNames ? '' : 'copy', subBuilder: Copy.create)
    ..aOM<PushConfig>(7, _omitFieldNames ? '' : 'push',
        subBuilder: PushConfig.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProjectConfigResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProjectConfigResponse copyWith(
          void Function(GetProjectConfigResponse) updates) =>
      super.copyWith((message) => updates(message as GetProjectConfigResponse))
          as GetProjectConfigResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetProjectConfigResponse create() => GetProjectConfigResponse._();
  @$core.override
  GetProjectConfigResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetProjectConfigResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetProjectConfigResponse>(create);
  static GetProjectConfigResponse? _defaultInstance;

  @$pb.TagNumber(1)
  GoogleConfig get google => $_getN(0);
  @$pb.TagNumber(1)
  set google(GoogleConfig value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasGoogle() => $_has(0);
  @$pb.TagNumber(1)
  void clearGoogle() => $_clearField(1);
  @$pb.TagNumber(1)
  GoogleConfig ensureGoogle() => $_ensure(0);

  @$pb.TagNumber(2)
  AppleConfig get apple => $_getN(1);
  @$pb.TagNumber(2)
  set apple(AppleConfig value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasApple() => $_has(1);
  @$pb.TagNumber(2)
  void clearApple() => $_clearField(2);
  @$pb.TagNumber(2)
  AppleConfig ensureApple() => $_ensure(1);

  /// Minimum accepted password length.
  @$pb.TagNumber(3)
  $core.int get passwordMinLength => $_getIZ(2);
  @$pb.TagNumber(3)
  set passwordMinLength($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPasswordMinLength() => $_has(2);
  @$pb.TagNumber(3)
  void clearPasswordMinLength() => $_clearField(3);

  /// Whether the public SignUp RPC is open.
  @$pb.TagNumber(4)
  $core.bool get signUpOpen => $_getBF(3);
  @$pb.TagNumber(4)
  set signUpOpen($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSignUpOpen() => $_has(3);
  @$pb.TagNumber(4)
  void clearSignUpOpen() => $_clearField(4);

  /// The project's design system. Omitted when
  /// GetProjectConfigRequest.known_theme_revision matches the current
  /// revision (see the caching contract there); always present otherwise,
  /// including for projects on the built-in default theme.
  @$pb.TagNumber(5)
  Theme get theme => $_getN(4);
  @$pb.TagNumber(5)
  set theme(Theme value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasTheme() => $_has(4);
  @$pb.TagNumber(5)
  void clearTheme() => $_clearField(5);
  @$pb.TagNumber(5)
  Theme ensureTheme() => $_ensure(4);

  /// The localized copy for the negotiated locale. Always present (it carries
  /// the negotiated locale + copy_revision so the client caches per (locale,
  /// revision)); its `messages` map is omitted when
  /// GetProjectConfigRequest.known_copy_revision matches, present otherwise —
  /// including for projects with no copy overrides (fully bundled defaults).
  @$pb.TagNumber(6)
  Copy get copy => $_getN(5);
  @$pb.TagNumber(6)
  set copy(Copy value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasCopy() => $_has(5);
  @$pb.TagNumber(6)
  void clearCopy() => $_clearField(6);
  @$pb.TagNumber(6)
  Copy ensureCopy() => $_ensure(5);

  /// The project's public push configuration. Always present; enabled=false
  /// when the project never configured push.
  @$pb.TagNumber(7)
  PushConfig get push => $_getN(6);
  @$pb.TagNumber(7)
  set push(PushConfig value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasPush() => $_has(6);
  @$pb.TagNumber(7)
  void clearPush() => $_clearField(7);
  @$pb.TagNumber(7)
  PushConfig ensurePush() => $_ensure(6);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
