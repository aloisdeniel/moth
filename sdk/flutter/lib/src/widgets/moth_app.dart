import 'dart:async';

import 'package:flutter/material.dart';

import '../auth_state.dart';
import '../client.dart';
import '../config.dart';
import '../theme.dart';
import '../theme_cache.dart';
import '../theme_controller.dart';
import '../token_store.dart';
import 'moth_login_screen.dart';
import 'moth_scope.dart';
import 'moth_theme_scope.dart';
import 'oauth_adapter.dart';

/// Top-level widget that owns a [MothClient] and gates [child] behind
/// authentication:
///
/// ```dart
/// void main() {
///   runApp(MothApp(
///     config: MothConfig(
///       endpoint: Uri.parse('https://auth.example.com'),
///       publishableKey: 'pk_...',
///     ),
///     child: const MyApp(),
///   ));
/// }
/// ```
///
/// On mount it restores the persisted session, then renders per state:
/// [MothAuthLoading] → [loading] (default: a centered progress indicator),
/// [MothSignedOut] → [signedOut] (default: [MothLoginScreen]),
/// [MothSignedIn] → [child]. With `requireAuth: false` [child] always
/// renders and reads the state itself via [MothScope.of], which is
/// available below this widget either way.
///
/// The screens MothApp owns (loading and signed-out) render with the
/// project's [MothTheme] as configured in the moth admin, refreshed
/// stale-while-revalidate: the last cached theme shows immediately, a
/// background fetch picks up admin edits. [child] — the app itself — keeps
/// the app's own theme untouched; the moth theme only ever applies to moth
/// screens. Pass [theme] to pin a hand-built theme instead (no fetch, no
/// cache), or [themeCache] to change where the delivered theme persists.
///
/// Pass either [config] (the widget creates and disposes the client) or an
/// existing [client] (the caller keeps ownership and disposes it); both
/// are fixed for the lifetime of the widget. When `MothApp` sits above
/// [MaterialApp] — the usual layout — the loading/signed-out screens are
/// wrapped in a minimal `MaterialApp` shell of their own, themed from the
/// project theme.
class MothApp extends StatefulWidget {
  const MothApp({
    super.key,
    this.config,
    this.client,
    this.tokenStore,
    this.oauthAdapter,
    this.theme,
    this.themeCache,
    this.loading,
    this.signedOut,
    this.requireAuth = true,
    required this.child,
  }) : assert(
         (config == null) != (client == null),
         'Provide exactly one of config or client.',
       ),
       assert(
         client == null || tokenStore == null,
         'tokenStore only applies when MothApp creates the client.',
       ),
       assert(
         theme == null || themeCache == null,
         'themeCache only applies when the server theme is used.',
       );

  /// Connection settings; the widget creates (and disposes) the client.
  final MothConfig? config;

  /// An externally owned client, e.g. one also used outside the widget
  /// tree. The caller disposes it.
  final MothClient? client;

  /// Session persistence override for the client created from [config]
  /// (defaults to secure storage).
  final TokenStore? tokenStore;

  /// Bridges the login screen's Google/Apple buttons to the native
  /// sign-in SDKs; exposed to descendants via [MothScope.oauthAdapter].
  final MothOAuthAdapter? oauthAdapter;

  /// Fixed theme for the moth screens; wins over the server-configured
  /// project theme (which is then neither fetched nor cached).
  final MothTheme? theme;

  /// Persistence override for the server-delivered theme (defaults to a
  /// file cache; useful for tests).
  final MothThemeCache? themeCache;

  /// Shown while the session restore is in flight.
  final Widget? loading;

  /// Shown while signed out; defaults to [MothLoginScreen].
  final Widget? signedOut;

  /// When false, [child] renders regardless of auth state.
  final bool requireAuth;

  /// The app itself, rendered once signed in.
  final Widget child;

  @override
  State<MothApp> createState() => _MothAppState();
}

class _MothAppState extends State<MothApp> {
  late final MothClient _client;
  late final bool _ownsClient;
  late MothAuthState _state;
  StreamSubscription<MothAuthState>? _subscription;
  MothThemeController? _theme;

  @override
  void initState() {
    super.initState();
    _ownsClient = widget.client == null;
    _client =
        widget.client ??
        MothClient(widget.config!, tokenStore: widget.tokenStore);
    _state = _client.currentState;
    _subscription = _client.authStateChanges.listen((state) {
      if (!mounted) return;
      setState(() => _state = state);
    });
    if (_state is MothAuthLoading) {
      // Failures surface through the state stream (restore keeps or clears
      // the session itself); nothing to await here.
      unawaited(_client.restore());
    }
    if (widget.theme == null && widget.requireAuth) {
      // Stale-while-revalidate: cached theme first, background refresh
      // after. Started even when the restore will land on signedIn, so the
      // cache is warm for the next sign-out.
      final controller = MothThemeController(
        client: _client,
        cache: widget.themeCache,
      );
      controller.addListener(_onThemeChanged);
      _theme = controller;
      unawaited(controller.start());
    }
  }

  void _onThemeChanged() {
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    _subscription?.cancel();
    _theme?.dispose();
    if (_ownsClient) unawaited(_client.dispose());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    Widget body;
    var ownSurface = false;
    if (widget.requireAuth) {
      switch (_state) {
        case MothAuthLoading():
          body = widget.loading ?? const _MothSplash();
          ownSurface = true;
        case MothSignedOut():
          body = widget.signedOut ?? const MothLoginScreen();
          ownSurface = true;
        case MothSignedIn():
          body = widget.child;
      }
    } else {
      body = widget.child;
    }
    if (ownSurface) {
      // moth-owned screens render with the project theme; the app's own
      // subtree (child) is deliberately left alone.
      final mothTheme = widget.theme ?? _theme?.value ?? MothTheme.fallback();
      body = _MothThemedSurface(theme: mothTheme, child: body);
      // When MothApp is the root of the tree (above the app's
      // MaterialApp), its own surfaces need an app shell for
      // Directionality, Material theming, overlays etc.
      if (Directionality.maybeOf(context) == null) {
        body = MaterialApp(
          debugShowCheckedModeBanner: false,
          theme: mothTheme.toThemeData(Brightness.light),
          darkTheme: mothTheme.toThemeData(Brightness.dark),
          home: body,
        );
      }
    }
    if (widget.requireAuth) {
      // Distinct keys per side of the gate: a flip must fully remount the
      // subtree, never update it in place — otherwise (both sides usually
      // being MaterialApps) the app's navigator state, open routes and
      // dialogs would survive a sign-out underneath the login screen.
      body = KeyedSubtree(key: ValueKey<bool>(ownSurface), child: body);
    }
    return MothScope(
      client: _client,
      state: _state,
      oauthAdapter: widget.oauthAdapter,
      child: body,
    );
  }
}

/// Applies the moth theme to a moth-owned screen: publishes it via
/// [MothThemeScope] and installs the matching Material [Theme] for the
/// ambient brightness.
class _MothThemedSurface extends StatelessWidget {
  const _MothThemedSurface({required this.theme, required this.child});

  final MothTheme theme;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final brightness =
        MediaQuery.maybePlatformBrightnessOf(context) ?? Brightness.light;
    return MothThemeScope(
      theme: theme,
      child: Theme(data: theme.toThemeData(brightness), child: child),
    );
  }
}

class _MothSplash extends StatelessWidget {
  const _MothSplash();

  @override
  Widget build(BuildContext context) =>
      const Scaffold(body: Center(child: CircularProgressIndicator()));
}
