import 'dart:async';
import 'dart:developer' as developer;
import 'dart:ui';

import 'package:grpc/service_api.dart' as grpc;

import 'auth_state.dart';
import 'channel/channel_stub.dart'
    if (dart.library.io) 'channel/channel_io.dart'
    if (dart.library.js_interop) 'channel/channel_web.dart';
import 'config.dart';
import 'copy.dart';
import 'customer_info.dart';
import 'errors.dart';
import 'exceptions.dart';
import 'gen/moth/auth/v1/auth.pbgrpc.dart' as pb;
import 'gen/moth/auth/v1/config.pbgrpc.dart' as pbconfig;
import 'gen/moth/billing/v1/billing.pbgrpc.dart' as pbbilling;
import 'gen/moth/push/v1/push.pbgrpc.dart' as pbpush;
import 'jwt.dart';
import 'locale.dart';
import 'offering.dart';
import 'platform/platform_stub.dart'
    if (dart.library.io) 'platform/platform_io.dart'
    if (dart.library.js_interop) 'platform/platform_web.dart';
import 'project_config.dart';
import 'push.dart';
import 'theme.dart';
import 'token_store.dart';
import 'transport/grpc.dart' show GrpcError;
import 'user.dart';
import 'version.dart';
import 'version_check.dart';

/// A social sign-in provider supported by moth.
enum MothOAuthProvider { google, apple }

extension on MothOAuthProvider {
  pb.OAuthProvider get proto => switch (this) {
    MothOAuthProvider.google => pb.OAuthProvider.OAUTH_PROVIDER_GOOGLE,
    MothOAuthProvider.apple => pb.OAuthProvider.OAUTH_PROVIDER_APPLE,
  };
}

/// Result of [MothClient.signUp]. Depending on project policy the server
/// returns the user with tokens (signed in immediately), the user without
/// tokens (email verification required first) or nothing at all
/// (enumeration-safe projects).
class MothSignUpResult {
  const MothSignUpResult({this.user, required this.signedIn});

  final MothUser? user;

  /// True when sign-up also opened a session (tokens were returned).
  final bool signedIn;
}

/// Client for the moth.auth.v1 end-user API.
///
/// Owns the gRPC channel (native HTTP/2 on iOS/Android/desktop, gRPC-Web on
/// Flutter Web), attaches the project's publishable key and — once signed in
/// — a Bearer access token to every call, persists the session in
/// [TokenStore] and refreshes the access token automatically.
///
/// Typical startup:
///
/// ```dart
/// final moth = MothClient(MothConfig(
///   endpoint: Uri.parse('https://auth.example.com'),
///   publishableKey: 'pk_...',
/// ));
/// await moth.restore(); // MothAuthLoading -> signedIn | signedOut
/// ```
class MothClient {
  MothClient(
    this.config, {
    TokenStore? tokenStore,
    this._deviceInfo = '',
    this._refreshSkew = const Duration(seconds: 30),
  }) : _store =
           tokenStore ??
           SecureTokenStore(publishableKey: config.publishableKey) {
    _channel = createChannel(config.endpoint);
    final options = grpc.CallOptions(providers: [_attachMetadata]);
    // Debug builds compare the server's x-moth-version response metadata
    // against the SDK version and warn on a major-version mismatch.
    final interceptors = [VersionCheckInterceptor()];
    _auth = pb.AuthServiceClient(
      _channel,
      options: options,
      interceptors: interceptors,
    );
    _projectConfig = pbconfig.ConfigServiceClient(
      _channel,
      options: options,
      interceptors: interceptors,
    );
    _billing = pbbilling.BillingServiceClient(
      _channel,
      options: options,
      interceptors: interceptors,
    );
    _push = pbpush.PushServiceClient(
      _channel,
      options: options,
      interceptors: interceptors,
    );
  }

  final MothConfig config;
  final TokenStore _store;
  final String _deviceInfo;

  /// Access tokens expiring within this window are refreshed proactively.
  final Duration _refreshSkew;

  late final grpc.ClientChannel _channel;
  late final pb.AuthServiceClient _auth;
  late final pbconfig.ConfigServiceClient _projectConfig;
  late final pbbilling.BillingServiceClient _billing;
  late final pbpush.PushServiceClient _push;

  MothAuthState _state = const MothAuthLoading();
  final _states = StreamController<MothAuthState>.broadcast();
  StoredSession? _session;
  Future<String>? _refreshing;

  MothCustomerInfo _customerInfo = const MothCustomerInfo.free();
  final _customerInfos = StreamController<MothCustomerInfo>.broadcast();

  MothPushConfig? _pushConfig;

  /// Bumped on every sign-out so an in-flight refresh that completes
  /// afterwards can tell the session it started from is gone and must not
  /// be resurrected.
  int _generation = 0;

  /// The current auth state ([MothAuthLoading] until [restore] completes).
  MothAuthState get currentState => _state;

  /// The locale the SDK negotiates copy for: [MothConfig.locale] when the app
  /// pinned one, otherwise the live device locale. Sent as `x-moth-language`
  /// on every call and used by [MothCopyController].
  Locale get currentLocale => config.locale ?? mothDeviceLocale();

  /// The signed-in user, or null.
  MothUser? get currentUser => switch (_state) {
    MothSignedIn(:final user) => user,
    _ => null,
  };

  /// Auth state changes. Every listener immediately receives the current
  /// state, then every subsequent change (backed by a broadcast stream).
  Stream<MothAuthState> get authStateChanges {
    late StreamController<MothAuthState> controller;
    StreamSubscription<MothAuthState>? forward;
    controller = StreamController<MothAuthState>(
      onListen: () {
        controller.add(_state);
        forward = _states.stream.listen(
          controller.add,
          onError: controller.addError,
          onDone: controller.close,
        );
      },
      onCancel: () => forward?.cancel(),
    );
    return controller.stream;
  }

  // ------------------------------------------------------------ entitlements

  /// The signed-in user's current subscription state. Always valid — an empty
  /// [MothCustomerInfo] (the free `none` tier) until the first
  /// [getCustomerInfo], and while signed out.
  MothCustomerInfo get currentCustomerInfo => _customerInfo;

  /// Seeds [currentCustomerInfo] from an on-device cache (stale-while-
  /// revalidate) so both [currentCustomerInfo] and [customerInfoChanges]
  /// reflect the last known entitlements before the first [getCustomerInfo]
  /// lands — non-widget subscribers then agree with the cached widget state.
  /// Deduplicated; the server stays authoritative and overwrites on the next
  /// billing RPC. Called by [MothSubscriptionController]; rarely needed
  /// directly.
  void primeCustomerInfo(MothCustomerInfo info) => _setCustomerInfo(info);

  /// Subscription-state changes. Like [authStateChanges], every listener
  /// immediately receives the current value, then every subsequent change
  /// (cache hit on launch, background refresh, purchase, restore, sign-out).
  Stream<MothCustomerInfo> get customerInfoChanges {
    late StreamController<MothCustomerInfo> controller;
    StreamSubscription<MothCustomerInfo>? forward;
    controller = StreamController<MothCustomerInfo>(
      onListen: () {
        controller.add(_customerInfo);
        forward = _customerInfos.stream.listen(
          controller.add,
          onError: controller.addError,
          onDone: controller.close,
        );
      },
      onCancel: () => forward?.cancel(),
    );
    return controller.stream;
  }

  // ---------------------------------------------------------------- session

  /// Restores a persisted session from the [TokenStore]. Call once at
  /// startup; until it completes [currentState] is [MothAuthLoading].
  ///
  /// A stored session whose access token is still fresh signs in without a
  /// network round-trip. An expired one is refreshed; when the server
  /// rejects the refresh token the session is cleared, while transient
  /// network failures keep it (the next [accessToken] call retries).
  Future<MothAuthState> restore() async {
    StoredSession? stored;
    try {
      stored = await _store.load();
    } on Object catch (err) {
      // A broken token store (secure storage unavailable, custom store
      // bug) must never wedge startup on the loading state: start signed
      // out, as documented — failures surface through the state stream.
      _logStorageFailure('load', err);
    }
    if (stored == null) {
      _setState(const MothSignedOut());
      return _state;
    }
    _session = stored;
    if (!_expiresSoon(stored)) {
      _setState(MothSignedIn(stored.user));
      return _state;
    }
    try {
      await _refresh();
    } on MothException {
      // _refresh clears the session when the token was rejected; otherwise
      // (network failure) stay signed in on the stored snapshot.
      if (_session != null) _setState(MothSignedIn(stored.user));
    }
    return _state;
  }

  /// Returns a valid access token for the signed-in user, refreshing it
  /// first when it expires within the refresh skew. Concurrent callers
  /// share a single refresh RPC. Throws [StateError] when signed out.
  Future<String> accessToken() async {
    final session = _session;
    if (session == null) throw StateError('moth: not signed in');
    if (!_expiresSoon(session)) return session.accessToken;
    return _refresh();
  }

  /// Forces a token refresh and returns the updated user. Throws
  /// [StateError] when the session ended (e.g. a concurrent [signOut])
  /// before the refresh completed.
  Future<MothUser> refresh() async {
    await _refresh();
    final session = _session;
    if (session == null) throw StateError('moth: not signed in');
    return session.user;
  }

  // ------------------------------------------------------- email / password

  /// Registers a new email/password user, subject to project policy.
  Future<MothSignUpResult> signUp({
    required String email,
    required String password,
    String? displayName,
    String? deviceInfo,
  }) => _run(() async {
    final resp = await _auth.signUp(
      pb.SignUpRequest(
        email: email,
        password: password,
        displayName: displayName ?? '',
        deviceInfo: deviceInfo ?? _deviceInfo,
      ),
    );
    if (resp.hasTokens()) {
      final user = await _openSession(resp.user, resp.tokens);
      return MothSignUpResult(user: user, signedIn: true);
    }
    return MothSignUpResult(
      user: resp.hasUser() ? _userFromProto(resp.user, const {}) : null,
      signedIn: false,
    );
  });

  /// Exchanges email/password for a session.
  Future<MothUser> signIn({
    required String email,
    required String password,
    String? deviceInfo,
  }) => _run(() async {
    final resp = await _auth.signIn(
      pb.SignInRequest(
        email: email,
        password: password,
        deviceInfo: deviceInfo ?? _deviceInfo,
      ),
    );
    return _openSession(resp.user, resp.tokens);
  });

  /// Revokes the current session server-side (best effort — local sign-out
  /// happens even when the revocation RPC fails) and clears the stored
  /// session. With [allDevices] every session of the user is revoked.
  Future<void> signOut({bool allDevices = false}) async {
    // An in-flight refresh must settle first: it may be rotating the
    // refresh token right now (revoke the current one, not a stale
    // predecessor) and, left running, it would re-open the session after
    // the sign-out cleared it.
    await _settleRefresh();
    final session = _session;
    if (session == null) {
      _setState(const MothSignedOut());
      return;
    }
    try {
      await _auth.signOut(
        pb.SignOutRequest(
          refreshToken: session.refreshToken,
          allDevices: allDevices,
        ),
      );
    } on GrpcError {
      // Best effort; the local session is cleared regardless.
    } finally {
      // A refresh kicked off while the RPC was in flight (e.g. a
      // background accessToken() call) must not resurrect the session
      // either.
      await _settleRefresh();
      await _clearSession();
    }
  }

  /// Changes the password (requires the current one). Every other session
  /// is revoked; this device continues on a fresh token pair.
  Future<void> changePassword({
    required String currentPassword,
    required String newPassword,
  }) => _authed(() async {
    final resp = await _auth.changePassword(
      pb.ChangePasswordRequest(
        currentPassword: currentPassword,
        newPassword: newPassword,
      ),
    );
    // The session may have ended (concurrent signOut) while the RPC was in
    // flight; don't resurrect it from the response.
    final session = _session;
    if (session == null) throw StateError('moth: not signed in');
    final user = session.user.copyWith(
      claims: customClaimsOf(resp.tokens.accessToken),
    );
    await _startSession(resp.tokens, user);
  });

  // ------------------------------------------------------------ social auth

  /// Signs in (or up) with a provider ID token obtained from a native
  /// Google/Apple flow. [rawNonce] is the per-attempt nonce the app
  /// generated; [authorizationCode], [givenName] and [familyName] are
  /// Apple-only.
  Future<MothUser> signInWithOAuth({
    required MothOAuthProvider provider,
    required String idToken,
    String? rawNonce,
    String? authorizationCode,
    String? givenName,
    String? familyName,
    String? deviceInfo,
  }) => _run(() async {
    final resp = await _auth.signInWithOAuth(
      pb.SignInWithOAuthRequest(
        provider: provider.proto,
        idToken: idToken,
        nonce: rawNonce ?? '',
        authorizationCode: authorizationCode ?? '',
        givenName: givenName ?? '',
        familyName: familyName ?? '',
        deviceInfo: deviceInfo ?? _deviceInfo,
      ),
    );
    return _openSession(resp.user, resp.tokens);
  });

  /// Trades the one-time code from the web-redirect OAuth fallback flow for
  /// a session.
  Future<MothUser> exchangeOAuthCode(String code, {String? deviceInfo}) =>
      _run(() async {
        final resp = await _auth.exchangeOAuthCode(
          pb.ExchangeOAuthCodeRequest(
            code: code,
            deviceInfo: deviceInfo ?? _deviceInfo,
          ),
        );
        return _openSession(resp.user, resp.tokens);
      });

  /// Removes the signed-in user's identity for [provider]. Refused with
  /// [MothLastLoginMethod] when it would leave no way to sign in.
  Future<void> unlinkIdentity(MothOAuthProvider provider) => _authed(
    () => _auth.unlinkIdentity(
      pb.UnlinkIdentityRequest(provider: provider.proto),
    ),
  );

  // ---------------------------------------------------------------- profile

  /// Fetches the signed-in user from the server and updates [currentUser].
  Future<MothUser> getMe() => _authed(() async {
    final resp = await _auth.getMe(pb.GetMeRequest());
    return _updateUser(resp.user);
  });

  /// Updates profile fields; only non-null arguments are sent.
  Future<MothUser> updateMe({String? displayName, String? avatarUrl}) =>
      _authed(() async {
        final req = pb.UpdateMeRequest();
        if (displayName != null) req.displayName = displayName;
        if (avatarUrl != null) req.avatarUrl = avatarUrl;
        final resp = await _auth.updateMe(req);
        return _updateUser(resp.user);
      });

  /// Permanently deletes the account after fresh re-authentication with
  /// [password], then clears the local session.
  Future<void> deleteAccount({required String password}) async {
    await _authed(
      () => _auth.deleteAccount(pb.DeleteAccountRequest(password: password)),
    );
    // As in signOut: a refresh started while the RPC was in flight must
    // not re-open the (now deleted) session after it is cleared.
    await _settleRefresh();
    await _clearSession();
  }

  // ------------------------------------------------------------ email flows

  /// (Re)sends the verification email. Always succeeds so responses never
  /// reveal whether an account exists.
  Future<void> requestEmailVerification(String email) => _run(
    () => _auth.requestEmailVerification(
      pb.RequestEmailVerificationRequest(email: email),
    ),
  );

  /// Consumes a verification token from the email link.
  Future<void> confirmEmailVerification(String token) => _run(
    () => _auth.confirmEmailVerification(
      pb.ConfirmEmailVerificationRequest(token: token),
    ),
  );

  /// Emails a password-reset link. Always succeeds so responses never
  /// reveal whether an account exists.
  Future<void> requestPasswordReset(String email) => _run(
    () => _auth.requestPasswordReset(
      pb.RequestPasswordResetRequest(email: email),
    ),
  );

  /// Consumes a reset token and sets the new password; every session of the
  /// user is revoked.
  Future<void> confirmPasswordReset({
    required String token,
    required String newPassword,
  }) => _run(
    () => _auth.confirmPasswordReset(
      pb.ConfirmPasswordResetRequest(token: token, newPassword: newPassword),
    ),
  );

  /// Sends a confirmation link to [newEmail]; the account switches only
  /// once that address is verified.
  Future<void> requestEmailChange(String newEmail) => _authed(
    () => _auth.requestEmailChange(
      pb.RequestEmailChangeRequest(newEmail: newEmail),
    ),
  );

  /// Consumes an email-change (or revert) token and applies the address.
  Future<void> confirmEmailChange(String token) => _run(
    () => _auth.confirmEmailChange(pb.ConfirmEmailChangeRequest(token: token)),
  );

  // ----------------------------------------------------------------- config

  /// Fetches the project's public configuration (enabled providers, client
  /// IDs, password policy, theme) so login UI adapts without an app
  /// release. Pass the [MothTheme.revisionId] already cached as
  /// [knownThemeRevision]: when it still matches, the server omits the
  /// theme body and [MothProjectConfig.theme] is null (keep the cached
  /// copy).
  Future<MothProjectConfig> getProjectConfig({
    String knownThemeRevision = '',
    String knownCopyRevision = '',
  }) => _run(() async {
    final resp = await _projectConfig.getProjectConfig(
      pbconfig.GetProjectConfigRequest(
        knownThemeRevision: knownThemeRevision,
        knownCopyRevision: knownCopyRevision,
      ),
    );
    String? blank(String s) => s.isEmpty ? null : s;
    // Cache the push section for the registration flow: enabled gates
    // RegisterDevice, the VAPID public key is the Web Push subscribe input.
    final push = MothPushConfig(
      enabled: resp.push.enabled,
      webpushVapidPublicKey: blank(resp.push.webpushVapidPublicKey),
    );
    _pushConfig = push;
    return MothProjectConfig(
      google: MothGoogleConfig(
        enabled: resp.google.enabled,
        webClientId: blank(resp.google.webClientId),
        iosClientId: blank(resp.google.iosClientId),
        androidClientId: blank(resp.google.androidClientId),
      ),
      apple: MothAppleConfig(enabled: resp.apple.enabled),
      passwordMinLength: resp.passwordMinLength,
      signUpOpen: resp.signUpOpen,
      push: push,
      theme: resp.hasTheme() ? MothTheme.fromProto(resp.theme) : null,
      copy: resp.hasCopy() ? _copyUpdate(resp.copy) : null,
    );
  });

  // The Copy caching contract mirrors the theme: the negotiated locale and
  // revision are always present; `messages` is omitted (empty) when the
  // client's knownCopyRevision still matched, meaning "keep the cached copy".
  MothCopyUpdate _copyUpdate(pbconfig.Copy copy) => MothCopyUpdate(
    locale: mothLocaleFromTag(copy.locale),
    revisionId: copy.copyRevision,
    messages: copy.messages.isEmpty
        ? null
        : Map<String, String>.of(copy.messages),
    source: copy,
  );

  // --------------------------------------------------------------- billing

  /// Fetches the signed-in user's subscription state, updates
  /// [currentCustomerInfo] and notifies [customerInfoChanges]. Cheap and safe
  /// to call on every launch. Throws [StateError] when signed out.
  ///
  /// On-device caching (stale-while-revalidate) and background refresh on
  /// launch are done by [MothSubscriptionController] — inserted automatically
  /// by [MothApp] — not by this raw RPC.
  Future<MothCustomerInfo> getCustomerInfo() => _authed(() async {
    final resp = await _billing.getCustomerInfo(
      pbbilling.GetCustomerInfoRequest(),
    );
    return _applyCustomerInfo(resp.customerInfo);
  });

  /// The products of [offering] (empty selects the project's default), for a
  /// paywall to display. Publishable-key only — safe before sign-in.
  Future<MothOffering> getOfferings({String offering = ''}) => _run(() async {
    final resp = await _billing.getOfferings(
      pbbilling.GetOfferingsRequest(offering: offering),
    );
    return MothOffering.fromProto(resp.offering);
  });

  /// The project's public paywall configuration, or null when
  /// [knownPaywallRevision] still matches the current revision (keep the
  /// cached copy — stale-while-revalidate, like the theme). Publishable-key
  /// only — safe before sign-in.
  Future<MothPaywall?> getPaywall({String knownPaywallRevision = ''}) => _run(
    () async {
      final resp = await _billing.getPaywall(
        pbbilling.GetPaywallRequest(knownPaywallRevision: knownPaywallRevision),
      );
      return resp.hasPaywall() ? MothPaywall.fromProto(resp.paywall) : null;
    },
  );

  /// Hands moth the receipt of a purchase the app just completed natively;
  /// the server validates it, re-derives entitlements and returns the fresh
  /// state. Prefer `MothScope.of(context).purchase` — it runs the native
  /// purchase through the [MothBillingAdapter] first.
  Future<MothCustomerInfo> submitPurchase({
    required MothStore store,
    required String productIdentifier,
    String? appleJwsTransaction,
    String? googlePurchaseToken,
    String? googleSubscriptionId,
  }) => _authed(() async {
    final req = pbbilling.SubmitPurchaseRequest(
      store: store.proto,
      productIdentifier: productIdentifier,
    );
    if (appleJwsTransaction != null) {
      req.appleJwsTransaction = appleJwsTransaction;
    }
    if (googlePurchaseToken != null) {
      req.googlePurchaseToken = googlePurchaseToken;
    }
    if (googleSubscriptionId != null) {
      req.googleSubscriptionId = googleSubscriptionId;
    }
    final resp = await _billing.submitPurchase(req);
    return _applyCustomerInfo(resp.customerInfo);
  });

  /// Re-links the store's existing purchases (identified by [receipts]) to the
  /// current user and returns the fresh state. Prefer
  /// `MothScope.of(context).restorePurchases` — it reads the receipts from the
  /// [MothBillingAdapter] first.
  Future<MothCustomerInfo> restorePurchases({
    required MothStore store,
    required List<String> receipts,
  }) => _authed(() async {
    final resp = await _billing.restorePurchases(
      pbbilling.RestorePurchasesRequest(store: store.proto, receipts: receipts),
    );
    return _applyCustomerInfo(resp.customerInfo);
  });

  // ------------------------------------------------------------------ push

  /// The push section of the last fetched project config, or null before the
  /// first [getProjectConfig] of this client's lifetime.
  MothPushConfig? get currentPushConfig => _pushConfig;

  /// The project's public push configuration, cached from the last
  /// [getProjectConfig] (fetched on demand otherwise). `enabled` gates the
  /// registration flow; the VAPID public key is for Web Push subscribes.
  /// Refreshed whenever any caller fetches the project config — a flipped
  /// switch is picked up on the next launch at the latest.
  Future<MothPushConfig> getPushConfig() async {
    final cached = _pushConfig;
    if (cached != null) return cached;
    await getProjectConfig();
    return _pushConfig ?? const MothPushConfig(enabled: false);
  }

  /// Upserts the signed-in user's push registration for this installation.
  /// Idempotent by design — call it on every launch/rotation/permission
  /// change without bookkeeping; the newest owner of a token wins. Prefer
  /// wiring a [MothPushAdapter] into [MothApp], which calls this
  /// automatically at the right moments. Throws [StateError] when signed
  /// out.
  Future<void> registerPushDevice({
    required MothPushTarget target,
    required String token,
    required String deviceId,
    MothPushPermission permission = MothPushPermission.unknown,
    MothPushDeviceMetadata metadata = const MothPushDeviceMetadata(),
  }) => _authed(
    () => _push.registerDevice(
      pbpush.RegisterDeviceRequest(
        target: target.proto,
        token: token,
        deviceId: deviceId,
        permission: permission.proto,
        metadata: pbpush.PushDeviceMetadata(
          platform: metadata.platform,
          model: metadata.model,
          osVersion: metadata.osVersion,
          appVersion: metadata.appVersion,
          locale: metadata.locale,
        ),
      ),
    ),
  );

  /// Revokes the signed-in user's registration for [deviceId]
  /// (`signed_out`). Unknown or already-revoked ids succeed — safe to
  /// repeat. Called by the SDK's sign-out flow before the session drops.
  Future<void> unregisterPushDevice({required String deviceId}) => _authed(
    () => _push.unregisterDevice(
      pbpush.UnregisterDeviceRequest(deviceId: deviceId),
    ),
  );

  /// Shuts down the channel and closes [authStateChanges].
  Future<void> dispose() async {
    await _channel.shutdown();
    await _states.close();
    await _customerInfos.close();
  }

  // -------------------------------------------------------------- internals

  Future<void> _attachMetadata(Map<String, String> metadata, String uri) async {
    metadata['x-moth-key'] = config.publishableKey;
    metadata['x-moth-platform'] = currentPlatform();
    metadata['x-moth-sdk-version'] = mothSdkVersion;
    metadata['x-moth-language'] = mothLanguageTag(currentLocale);
    final session = _session;
    if (session != null) {
      metadata['authorization'] = 'Bearer ${session.accessToken}';
    }
  }

  /// Maps transport errors to the typed [MothException] hierarchy.
  Future<T> _run<T>(Future<T> Function() fn) async {
    try {
      return await fn();
    } on GrpcError catch (err) {
      throw mapGrpcError(err);
    }
  }

  /// Ensures a fresh Bearer token is attached, then runs [fn]. When the
  /// server nonetheless rejects the access token (the client-computed
  /// expiry drifted — device slept mid-call, TTL shortened server-side),
  /// the call refreshes reactively and retries once instead of surfacing
  /// "session expired" to a session whose refresh token is still valid.
  Future<T> _authed<T>(Future<T> Function() fn) async {
    await accessToken();
    try {
      return await _run(fn);
    } on MothInvalidAccessToken {
      if (_session == null) rethrow;
      await _refresh();
      return _run(fn);
    }
  }

  bool _expiresSoon(StoredSession session) =>
      !session.expiresAt.isAfter(DateTime.now().toUtc().add(_refreshSkew));

  Future<String> _refresh() {
    final inflight = _refreshing;
    if (inflight != null) return inflight;
    final refresh = _doRefresh().whenComplete(() => _refreshing = null);
    _refreshing = refresh;
    return refresh;
  }

  /// Waits for an in-flight refresh to settle, ignoring its outcome (its
  /// own callers handle errors).
  Future<void> _settleRefresh() async {
    final inflight = _refreshing;
    if (inflight == null) return;
    try {
      await inflight;
    } on Object {
      // Handled by the refresh's own callers.
    }
  }

  Future<String> _doRefresh() async {
    final session = _session;
    if (session == null) throw StateError('moth: not signed in');
    final generation = _generation;
    try {
      final resp = await _auth.refreshToken(
        pb.RefreshTokenRequest(refreshToken: session.refreshToken),
      );
      // The session ended (signOut/deleteAccount) while the RPC was in
      // flight: committing the fresh tokens would silently sign the user
      // back in.
      if (generation != _generation) {
        throw StateError('moth: signed out during token refresh');
      }
      await _openSession(resp.user, resp.tokens);
      return resp.tokens.accessToken;
    } on GrpcError catch (err) {
      final mapped = mapGrpcError(err);
      // A rejected refresh token means the session is gone (rotated-out,
      // revoked or stolen): end up signed out with storage cleared.
      // Transient failures (network, server errors) keep the session.
      // When the session already ended concurrently there is nothing left
      // to clear (and a newer session must not be clobbered).
      if (generation == _generation &&
          (mapped is MothInvalidRefreshToken ||
              mapped is MothRefreshTokenReused ||
              mapped is MothInvalidToken ||
              mapped is MothUserDisabled)) {
        await _clearSession();
      }
      throw mapped;
    }
  }

  Future<MothUser> _openSession(pb.User user, pb.TokenPair tokens) =>
      _startSession(
        tokens,
        _userFromProto(user, customClaimsOf(tokens.accessToken)),
      );

  Future<MothUser> _startSession(pb.TokenPair tokens, MothUser user) async {
    final session = StoredSession(
      accessToken: tokens.accessToken,
      refreshToken: tokens.refreshToken,
      expiresAt: DateTime.now().toUtc().add(
        Duration(seconds: tokens.expiresIn.toInt()),
      ),
      user: user,
    );
    _session = session;
    await _persist(session);
    _setState(MothSignedIn(user));
    return user;
  }

  /// Replaces the cached user snapshot after GetMe/UpdateMe.
  Future<MothUser> _updateUser(pb.User user) async {
    final session = _session;
    final claims = session == null
        ? const <String, Object?>{}
        : session.user.claims;
    final mothUser = _userFromProto(user, claims);
    if (session != null) {
      final updated = session.copyWith(user: mothUser);
      _session = updated;
      await _persist(updated);
      _setState(MothSignedIn(mothUser));
    }
    return mothUser;
  }

  /// Saves the session, swallowing storage failures: the in-memory session
  /// is fully usable (the server accepted the credentials) — it just won't
  /// survive a restart. Throwing here would fail a sign-in that actually
  /// succeeded, as an untyped platform exception no caller expects.
  Future<void> _persist(StoredSession session) async {
    try {
      await _store.save(session);
    } on Object catch (err) {
      _logStorageFailure('save', err);
    }
  }

  Future<void> _clearSession() async {
    _session = null;
    _generation++;
    try {
      await _store.clear();
    } on Object catch (err) {
      // Sign-out must complete locally even when storage misbehaves.
      _logStorageFailure('clear', err);
    }
    // Emit the signed-out auth state BEFORE the free customer-info reset: a
    // MothSubscriptionController listens to both, and it must observe the
    // sign-out (which drops its user id) before the free snapshot arrives, so
    // it does not persist the empty snapshot over the outgoing user's cached
    // entitlements (breaking instant gating when they sign back in).
    _setState(const MothSignedOut());
    _setCustomerInfo(const MothCustomerInfo.free());
  }

  // Publishes a fresh CustomerInfo from a billing RPC. Always emits — even
  // when it equals the last value — so a stale-while-revalidate controller
  // that rendered a cached snapshot still learns the confirmed server truth.
  // Server is authority; on-device caching lives in
  // MothSubscriptionController.
  MothCustomerInfo _applyCustomerInfo(pbbilling.CustomerInfo proto) {
    final info = MothCustomerInfo.fromProto(proto);
    _customerInfo = info;
    if (!_customerInfos.isClosed) _customerInfos.add(info);
    return info;
  }

  // Resets to a value on a lifecycle change (sign-out); deduplicated, so a
  // no-op reset does not churn listeners.
  void _setCustomerInfo(MothCustomerInfo info) {
    if (info == _customerInfo) return;
    _customerInfo = info;
    if (!_customerInfos.isClosed) _customerInfos.add(info);
  }

  void _logStorageFailure(String op, Object err) {
    assert(() {
      developer.log(
        'moth: token store $op failed: $err',
        name: 'moth',
        level: 900 /* warning */,
      );
      return true;
    }());
  }

  void _setState(MothAuthState state) {
    _state = state;
    if (!_states.isClosed) _states.add(state);
  }

  MothUser _userFromProto(pb.User user, Map<String, Object?> claims) =>
      MothUser(
        id: user.id,
        email: user.email,
        emailVerified: user.emailVerified,
        displayName: user.displayName.isEmpty ? null : user.displayName,
        avatarUrl: user.avatarUrl.isEmpty ? null : user.avatarUrl,
        createTime: user.hasCreateTime() ? user.createTime.toDateTime() : null,
        claims: claims,
      );
}
