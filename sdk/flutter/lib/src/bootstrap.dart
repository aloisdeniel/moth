import 'dart:convert';
import 'dart:ui';

import 'copy.dart';
import 'copy_cache.dart';
import 'gen/moth/auth/v1/config.pb.dart' as pbconfig;
import 'gen/moth/billing/v1/billing.pb.dart' as pbbilling;
import 'generated_config.dart';
import 'locale.dart';
import 'offering.dart';
import 'paywall_cache.dart';
import 'project_config.dart';
import 'theme.dart';
import 'theme_cache.dart';

/// The project configuration a moth instance bakes into a server-generated
/// package (see `generated_config.dart`), decoded once. It lets the SDK render
/// the right providers, theme, copy and paywall on the very first frame with
/// no network round-trip — the bundled *floor* that the caches then revalidate
/// against in the background so admin edits still ship without an app release.
///
/// [instance] is `null` for the canonical package pulled from the generic
/// `/pub` repository (empty placeholders); every consumer then behaves exactly
/// as before.
class MothBootstrap {
  MothBootstrap._(this.config, this.paywall);

  /// The decoded public project config (providers, password policy, sign-up
  /// state, push, theme, and default-locale copy).
  final pbconfig.GetProjectConfigResponse config;

  /// The decoded public paywall, or null when the project has none baked in.
  final pbbilling.Paywall? paywall;

  static bool _loaded = false;
  static MothBootstrap? _instance;

  /// The baked config for a server-generated build, or null for the canonical
  /// package (or when the seed is absent/malformed — never throws).
  static MothBootstrap? get instance {
    if (_loaded) return _instance;
    _loaded = true;
    return _instance = decode(mothConfigB64, mothPaywallB64);
  }

  /// Decodes the base64 config/paywall seeds into a [MothBootstrap], or null
  /// when [configB64] is empty (the canonical package) or the bytes are
  /// malformed — decoding never throws. Exposed for tests; production reads the
  /// baked constants through [instance].
  static MothBootstrap? decode(String configB64, String paywallB64) {
    if (configB64.isEmpty) return null;
    try {
      final config = pbconfig.GetProjectConfigResponse.fromBuffer(
        base64.decode(configB64),
      );
      final paywall = paywallB64.isEmpty
          ? null
          : pbbilling.Paywall.fromBuffer(base64.decode(paywallB64));
      return MothBootstrap._(config, paywall);
    } on Object {
      // A malformed seed is never fatal: fall back to the network path.
      return null;
    }
  }

  // The bundled floor is stamped at the epoch so it always reads as "stale":
  // the caches serve it instantly, then the controllers revalidate on the
  // first launch (stale-while-revalidate) to pick up any admin edits.
  static final DateTime _epoch = DateTime.fromMillisecondsSinceEpoch(
    0,
    isUtc: true,
  );

  static String? _blank(String s) => s.isEmpty ? null : s;

  /// The baked public config as the SDK model, so the login screen renders the
  /// right providers and password policy on the first frame.
  MothProjectConfig get projectConfig => MothProjectConfig(
    google: MothGoogleConfig(
      enabled: config.google.enabled,
      webClientId: _blank(config.google.webClientId),
      iosClientId: _blank(config.google.iosClientId),
      androidClientId: _blank(config.google.androidClientId),
    ),
    apple: MothAppleConfig(enabled: config.apple.enabled),
    passwordMinLength: config.passwordMinLength,
    signUpOpen: config.signUpOpen,
    push: MothPushConfig(
      enabled: config.push.enabled,
      webpushVapidPublicKey: _blank(config.push.webpushVapidPublicKey),
    ),
    theme: config.hasTheme() ? MothTheme.fromProto(config.theme) : null,
  );

  /// The baked theme as a cache entry, for the theme cache to seed a cold
  /// cache. Null when no theme is baked in.
  MothCachedTheme? get seededTheme => config.hasTheme()
      ? MothCachedTheme(theme: MothTheme.fromProto(config.theme), fetchedAt: _epoch)
      : null;

  /// The baked copy as a cache entry for [locale], for the copy cache to seed
  /// a cold cache. Null unless [locale] matches the baked default-locale copy
  /// (other languages fetch on first use).
  MothCachedCopy? seededCopy(Locale locale) {
    if (!config.hasCopy()) return null;
    final wire = config.copy;
    final baked = mothLocaleFromTag(wire.locale);
    if (baked.languageCode != locale.languageCode) return null;
    return MothCachedCopy(
      copy: MothCopy(
        locale: baked,
        revisionId: wire.copyRevision,
        messages: Map<String, String>.of(wire.messages),
        source: wire,
      ),
      fetchedAt: _epoch,
    );
  }

  /// The baked paywall as a cache entry, for the paywall cache to seed a cold
  /// cache. Null when no paywall is baked in.
  MothCachedPaywall? get seededPaywall {
    final pw = paywall;
    if (pw == null) return null;
    return MothCachedPaywall(paywall: MothPaywall.fromProto(pw), fetchedAt: _epoch);
  }
}
