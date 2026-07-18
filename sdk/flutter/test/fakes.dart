// In-process fake moth server implementing the generated service bases, so
// client tests exercise the real wire path (channel, metadata, status
// details) without a Go binary.
import 'dart:async';
import 'dart:convert';
import 'dart:ui';

import 'package:fixnum/fixnum.dart';
import 'package:grpc/grpc.dart';
import 'package:grpc/protos.dart' show ErrorInfo;
// The Any/Status containers for grpc-status-details-bin are not re-exported
// by package:grpc; tests build the trailer by hand exactly like the Go
// server does.
// ignore: implementation_imports
import 'package:grpc/src/generated/google/protobuf/any.pb.dart';
// ignore: implementation_imports
import 'package:grpc/src/generated/google/rpc/status.pb.dart' as rpc;
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_auth/src/gen/moth/auth/v1/auth.pbgrpc.dart';
import 'package:moth_auth/src/gen/moth/auth/v1/config.pbgrpc.dart';
import 'package:moth_auth/src/gen/moth/billing/v1/billing.pbgrpc.dart'
    as billing;

/// A syntactically valid JWT with [payload] and a fake signature — the SDK
/// only ever decodes the payload.
String makeJwt(Map<String, Object?> payload) {
  String enc(Object o) =>
      base64Url.encode(utf8.encode(jsonEncode(o))).replaceAll('=', '');
  return '${enc({'alg': 'ES256', 'typ': 'JWT'})}.${enc(payload)}.sig';
}

/// A GrpcError carrying a moth `google.rpc.ErrorInfo` reason in
/// `grpc-status-details-bin`, mirroring internal/server/rpc/auth/errors.go.
GrpcError mothError(int code, String reason, String message) {
  final status = rpc.Status(
    code: code,
    message: message,
    details: [Any.pack(ErrorInfo(reason: reason, domain: 'moth.dev'))],
  );
  final bin = base64Url.encode(status.writeToBuffer()).replaceAll('=', '');
  return GrpcError.custom(code, message, null, null, {
    'grpc-status-details-bin': bin,
  });
}

enum SignUpMode { tokens, userOnly, empty }

/// A [TokenStore] whose operations fail on demand, standing in for broken
/// platform secure storage (invalidated Keystore entry, locked Keychain).
class ThrowingTokenStore extends InMemoryTokenStore {
  bool throwOnLoad = false;
  bool throwOnSave = false;
  bool throwOnClear = false;

  @override
  Future<StoredSession?> load() {
    if (throwOnLoad) throw Exception('storage read failed');
    return super.load();
  }

  @override
  Future<void> save(StoredSession session) {
    if (throwOnSave) throw Exception('storage write failed');
    return super.save(session);
  }

  @override
  Future<void> clear() {
    if (throwOnClear) throw Exception('storage delete failed');
    return super.clear();
  }
}

class FakeAuthService extends AuthServiceBase {
  /// Client metadata of the most recent call, per method name.
  final metadataByMethod = <String, Map<String, String>>{};

  /// Thrown once by the next RPC (any method), then cleared.
  GrpcError? nextError;

  /// Thrown by every RefreshToken call while set.
  GrpcError? refreshError;

  /// `expires_in` stamped on minted token pairs.
  Duration accessTokenTtl = const Duration(hours: 1);

  /// Artificial latency inside RefreshToken, to pile up concurrent callers.
  Duration refreshDelay = Duration.zero;

  SignUpMode signUpMode = SignUpMode.tokens;

  /// The most recent SignInWithOAuth request, for nonce/token assertions.
  SignInWithOAuthRequest? lastOAuthRequest;

  /// The most recent SignOut request, for revoked-token assertions.
  SignOutRequest? lastSignOutRequest;

  int refreshCalls = 0;
  int tokenCounter = 0;
  final refreshTokensSeen = <String>[];

  User get userProto => User(
    id: 'user-1',
    email: 'jane@example.com',
    emailVerified: true,
    displayName: 'Jane',
  );

  TokenPair mintTokens() {
    tokenCounter++;
    return TokenPair(
      accessToken: makeJwt({
        'sub': 'user-1',
        'email': 'jane@example.com',
        'claims': {'role': 'admin'},
        'n': tokenCounter,
      }),
      refreshToken: 'rt_$tokenCounter',
      expiresIn: Int64(accessTokenTtl.inSeconds),
    );
  }

  void _enter(String method, ServiceCall call) {
    metadataByMethod[method] = Map.of(call.clientMetadata ?? const {});
    final err = nextError;
    if (err != null) {
      nextError = null;
      throw err;
    }
  }

  @override
  Future<SignUpResponse> signUp(ServiceCall call, SignUpRequest request) async {
    _enter('SignUp', call);
    return switch (signUpMode) {
      SignUpMode.tokens => SignUpResponse(
        user: userProto,
        tokens: mintTokens(),
      ),
      SignUpMode.userOnly => SignUpResponse(user: userProto),
      SignUpMode.empty => SignUpResponse(),
    };
  }

  @override
  Future<SignInResponse> signIn(ServiceCall call, SignInRequest request) async {
    _enter('SignIn', call);
    return SignInResponse(user: userProto, tokens: mintTokens());
  }

  @override
  Future<RefreshTokenResponse> refreshToken(
    ServiceCall call,
    RefreshTokenRequest request,
  ) async {
    _enter('RefreshToken', call);
    refreshCalls++;
    refreshTokensSeen.add(request.refreshToken);
    final err = refreshError;
    if (err != null) throw err;
    await Future<void>.delayed(refreshDelay);
    return RefreshTokenResponse(user: userProto, tokens: mintTokens());
  }

  @override
  Future<SignOutResponse> signOut(
    ServiceCall call,
    SignOutRequest request,
  ) async {
    _enter('SignOut', call);
    lastSignOutRequest = request;
    return SignOutResponse();
  }

  @override
  Future<GetMeResponse> getMe(ServiceCall call, GetMeRequest request) async {
    _enter('GetMe', call);
    return GetMeResponse(user: userProto);
  }

  @override
  Future<UpdateMeResponse> updateMe(
    ServiceCall call,
    UpdateMeRequest request,
  ) async {
    _enter('UpdateMe', call);
    final user = userProto;
    if (request.hasDisplayName()) user.displayName = request.displayName;
    if (request.hasAvatarUrl()) user.avatarUrl = request.avatarUrl;
    return UpdateMeResponse(user: user);
  }

  @override
  Future<ChangePasswordResponse> changePassword(
    ServiceCall call,
    ChangePasswordRequest request,
  ) async {
    _enter('ChangePassword', call);
    return ChangePasswordResponse(tokens: mintTokens());
  }

  @override
  Future<RequestEmailVerificationResponse> requestEmailVerification(
    ServiceCall call,
    RequestEmailVerificationRequest request,
  ) async {
    _enter('RequestEmailVerification', call);
    return RequestEmailVerificationResponse();
  }

  @override
  Future<ConfirmEmailVerificationResponse> confirmEmailVerification(
    ServiceCall call,
    ConfirmEmailVerificationRequest request,
  ) async {
    _enter('ConfirmEmailVerification', call);
    return ConfirmEmailVerificationResponse();
  }

  @override
  Future<RequestPasswordResetResponse> requestPasswordReset(
    ServiceCall call,
    RequestPasswordResetRequest request,
  ) async {
    _enter('RequestPasswordReset', call);
    return RequestPasswordResetResponse();
  }

  @override
  Future<ConfirmPasswordResetResponse> confirmPasswordReset(
    ServiceCall call,
    ConfirmPasswordResetRequest request,
  ) async {
    _enter('ConfirmPasswordReset', call);
    return ConfirmPasswordResetResponse();
  }

  @override
  Future<RequestEmailChangeResponse> requestEmailChange(
    ServiceCall call,
    RequestEmailChangeRequest request,
  ) async {
    _enter('RequestEmailChange', call);
    return RequestEmailChangeResponse();
  }

  @override
  Future<ConfirmEmailChangeResponse> confirmEmailChange(
    ServiceCall call,
    ConfirmEmailChangeRequest request,
  ) async {
    _enter('ConfirmEmailChange', call);
    return ConfirmEmailChangeResponse();
  }

  @override
  Future<SignInWithOAuthResponse> signInWithOAuth(
    ServiceCall call,
    SignInWithOAuthRequest request,
  ) async {
    _enter('SignInWithOAuth', call);
    lastOAuthRequest = request;
    return SignInWithOAuthResponse(user: userProto, tokens: mintTokens());
  }

  @override
  Future<ExchangeOAuthCodeResponse> exchangeOAuthCode(
    ServiceCall call,
    ExchangeOAuthCodeRequest request,
  ) async {
    _enter('ExchangeOAuthCode', call);
    return ExchangeOAuthCodeResponse(user: userProto, tokens: mintTokens());
  }

  @override
  Future<UnlinkIdentityResponse> unlinkIdentity(
    ServiceCall call,
    UnlinkIdentityRequest request,
  ) async {
    _enter('UnlinkIdentity', call);
    return UnlinkIdentityResponse();
  }

  @override
  Future<DeleteAccountResponse> deleteAccount(
    ServiceCall call,
    DeleteAccountRequest request,
  ) async {
    _enter('DeleteAccount', call);
    return DeleteAccountResponse();
  }
}

class FakeConfigService extends ConfigServiceBase {
  Map<String, String>? lastMetadata;
  GetProjectConfigRequest? lastRequest;
  int calls = 0;

  /// While set, every GetProjectConfig call waits for the gate before
  /// replying — lets tests assert on intermediate client state (e.g. the
  /// cached theme rendering) while the network response is held back.
  Completer<void>? gate;

  /// Served to every GetProjectConfig call; tests mutate it for variants.
  GetProjectConfigResponse response = GetProjectConfigResponse(
    google: GoogleConfig(
      enabled: true,
      webClientId: 'web-id',
      androidClientId: 'android-id',
    ),
    apple: AppleConfig(enabled: false),
    passwordMinLength: 10,
    signUpOpen: true,
  );

  @override
  Future<GetProjectConfigResponse> getProjectConfig(
    ServiceCall call,
    GetProjectConfigRequest request,
  ) async {
    lastMetadata = Map.of(call.clientMetadata ?? const {});
    lastRequest = request;
    calls++;
    // Snapshot the response at request time (before the gate), so a test that
    // holds one call gated and mutates `response` for a second, concurrent call
    // still serves each call the value it was configured with when it arrived.
    final resp = response.deepCopy();
    final gate = this.gate;
    if (gate != null) await gate.future;
    // The theme caching contract: a matching known revision omits the
    // theme body, exactly like internal/server/rpc/auth/config.go.
    if (resp.hasTheme() &&
        request.knownThemeRevision == resp.theme.revisionId) {
      resp.clearTheme();
    }
    // The copy caching contract: a matching known revision keeps the locale +
    // revision but omits the messages body, exactly like the server.
    if (resp.hasCopy() && request.knownCopyRevision == resp.copy.copyRevision) {
      resp.copy.messages.clear();
    }
    return resp;
  }
}

/// In-process fake for moth.billing.v1. Tests set [customerInfo], [offering]
/// and [paywall] and drive success/error paths via [nextError],
/// [purchaseError] and [customerInfoAfterPurchase].
class FakeBillingService extends billing.BillingServiceBase {
  /// Returned by GetCustomerInfo / SubmitPurchase / RestorePurchases. Free
  /// (empty entitlements) by default.
  billing.CustomerInfo customerInfo = billing.CustomerInfo();

  /// Returned by GetOfferings for the default (empty) tag.
  billing.Offering offering = billing.Offering(
    identifier: 'default',
    isDefault: true,
  );

  /// Per-tag offerings; a request whose tag is present here wins over
  /// [offering]. Lets tests give a non-default offering distinct products.
  final offeringsByTag = <String, billing.Offering>{};

  /// Returned by GetPaywall; omitted when the request's known revision matches.
  billing.Paywall paywall = billing.Paywall(
    revisionId: 'pw-1',
    headline: 'Unlock Premium',
    subtitle: 'Get the full experience with a subscription.',
    layout: billing.PaywallLayout.PAYWALL_LAYOUT_TILES,
  );

  /// When set, SubmitPurchase installs it as the new [customerInfo] and returns
  /// it (simulating the server deriving entitlements from the receipt).
  billing.CustomerInfo? customerInfoAfterPurchase;

  /// Thrown once by the next RPC (any method), then cleared.
  GrpcError? nextError;

  /// Thrown by every SubmitPurchase while set.
  GrpcError? purchaseError;

  /// While set, GetCustomerInfo waits for the gate before replying — lets
  /// tests observe the cached snapshot rendering while the refresh is held.
  Completer<void>? getCustomerInfoGate;

  final metadataByMethod = <String, Map<String, String>>{};
  billing.SubmitPurchaseRequest? lastSubmit;
  billing.RestorePurchasesRequest? lastRestore;
  billing.GetOfferingsRequest? lastOfferingsRequest;
  billing.GetPaywallRequest? lastPaywallRequest;
  int getCustomerInfoCalls = 0;
  int getOfferingsCalls = 0;
  int getPaywallCalls = 0;

  void _enter(String method, ServiceCall call) {
    metadataByMethod[method] = Map.of(call.clientMetadata ?? const {});
    final err = nextError;
    if (err != null) {
      nextError = null;
      throw err;
    }
  }

  @override
  Future<billing.GetCustomerInfoResponse> getCustomerInfo(
    ServiceCall call,
    billing.GetCustomerInfoRequest request,
  ) async {
    _enter('GetCustomerInfo', call);
    getCustomerInfoCalls++;
    final gate = getCustomerInfoGate;
    if (gate != null) await gate.future;
    return billing.GetCustomerInfoResponse(customerInfo: customerInfo);
  }

  @override
  Future<billing.SubmitPurchaseResponse> submitPurchase(
    ServiceCall call,
    billing.SubmitPurchaseRequest request,
  ) async {
    _enter('SubmitPurchase', call);
    lastSubmit = request;
    final err = purchaseError;
    if (err != null) throw err;
    final next = customerInfoAfterPurchase;
    if (next != null) customerInfo = next;
    return billing.SubmitPurchaseResponse(customerInfo: customerInfo);
  }

  @override
  Future<billing.RestorePurchasesResponse> restorePurchases(
    ServiceCall call,
    billing.RestorePurchasesRequest request,
  ) async {
    _enter('RestorePurchases', call);
    lastRestore = request;
    return billing.RestorePurchasesResponse(customerInfo: customerInfo);
  }

  @override
  Future<billing.GetOfferingsResponse> getOfferings(
    ServiceCall call,
    billing.GetOfferingsRequest request,
  ) async {
    _enter('GetOfferings', call);
    getOfferingsCalls++;
    lastOfferingsRequest = request;
    final tagged = offeringsByTag[request.offering];
    return billing.GetOfferingsResponse(offering: tagged ?? offering);
  }

  @override
  Future<billing.GetPaywallResponse> getPaywall(
    ServiceCall call,
    billing.GetPaywallRequest request,
  ) async {
    _enter('GetPaywall', call);
    getPaywallCalls++;
    lastPaywallRequest = request;
    final resp = billing.GetPaywallResponse();
    if (request.knownPaywallRevision != paywall.revisionId) {
      resp.paywall = paywall;
    }
    return resp;
  }

  // The Stripe web-billing RPCs (milestone 17/18) exist on the generated
  // base but have no Flutter SDK surface; the fake only needs them to
  // compile.
  @override
  Future<billing.CreateCheckoutSessionResponse> createCheckoutSession(
    ServiceCall call,
    billing.CreateCheckoutSessionRequest request,
  ) async {
    _enter('CreateCheckoutSession', call);
    return billing.CreateCheckoutSessionResponse();
  }

  @override
  Future<billing.CreateBillingPortalSessionResponse> createBillingPortalSession(
    ServiceCall call,
    billing.CreateBillingPortalSessionRequest request,
  ) async {
    _enter('CreateBillingPortalSession', call);
    return billing.CreateBillingPortalSessionResponse();
  }
}

/// A [MothBillingAdapter] whose native outcomes are set by the test: returns
/// [nextReceipt] (a signed Apple transaction by default), null when [cancel]
/// is set, or throws [throwOnPurchase].
class FakeBillingAdapter implements MothBillingAdapter {
  MothPurchaseReceipt? nextReceipt;
  MothBillingException? throwOnPurchase;
  bool cancel = false;
  MothRestoreReceipts restoreResult = const MothRestoreReceipts(
    store: MothStore.apple,
    receipts: ['restore-jws'],
  );

  /// Store products returned by [productsFor] (empty = not implemented).
  List<MothStoreProduct> storeProducts = const [];

  /// Out-of-band receipts (Ask to Buy approvals, renewals) pushed by tests
  /// via `updates.add(...)`; surfaced on [transactionUpdates].
  final StreamController<MothPurchaseReceipt> updates =
      StreamController<MothPurchaseReceipt>.broadcast();

  MothOfferingProduct? lastProduct;
  int purchaseCalls = 0;
  int restoreCalls = 0;
  int productsForCalls = 0;

  @override
  Future<MothPurchaseReceipt?> purchase(MothOfferingProduct product) async {
    purchaseCalls++;
    lastProduct = product;
    final err = throwOnPurchase;
    if (err != null) throw err;
    if (cancel) return null;
    return nextReceipt ??
        MothPurchaseReceipt(
          store: MothStore.apple,
          productIdentifier: product.identifier,
          appleJwsTransaction: 'jws-${product.identifier}',
        );
  }

  @override
  Future<MothRestoreReceipts> restore() async {
    restoreCalls++;
    return restoreResult;
  }

  @override
  Future<List<MothStoreProduct>> productsFor(MothOffering offering) async {
    productsForCalls++;
    return storeProducts;
  }

  @override
  Stream<MothPurchaseReceipt> get transactionUpdates => updates.stream;
}

class FakeMoth {
  FakeMoth(this.server, this.auth, this.config, this.billing);

  final Server server;
  final FakeAuthService auth;
  final FakeConfigService config;
  final FakeBillingService billing;

  int get port => server.port!;

  Future<void> shutdown() => server.shutdown();
}

Future<FakeMoth> startFakeMoth() async {
  final auth = FakeAuthService();
  final config = FakeConfigService();
  final billingService = FakeBillingService();
  final server = Server.create(services: [auth, config, billingService]);
  await server.serve(address: 'localhost', port: 0);
  return FakeMoth(server, auth, config, billingService);
}

MothClient newClient(
  FakeMoth moth, {
  TokenStore? store,
  Duration skew = const Duration(seconds: 30),
  Locale? locale,
  String? appName,
  Duration configCacheTtl = const Duration(hours: 1),
}) => MothClient(
  MothConfig(
    endpoint: Uri.parse('http://localhost:${moth.port}'),
    publishableKey: 'pk_test',
    locale: locale,
    appName: appName,
    configCacheTtl: configCacheTtl,
  ),
  tokenStore: store ?? InMemoryTokenStore(),
  refreshSkew: skew,
);
