// This is a generated file - do not edit.
//
// Generated from moth/auth/v1/auth.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:grpc/service_api.dart' as $grpc;
import 'package:protobuf/protobuf.dart' as $pb;

import 'auth.pb.dart' as $0;

export 'auth.pb.dart';

/// AuthService is the public end-user authentication API consumed by mobile
/// apps (via the SDK). Every call carries the project's publishable key in
/// `x-moth-key: pk_...` request metadata; an interceptor resolves it to the
/// project, so users, tokens and emails are always project-scoped.
///
/// RPCs about the current user (GetMe, UpdateMe, ChangePassword,
/// RequestEmailChange, DeleteAccount) additionally require a valid access
/// token in `authorization: Bearer ...` metadata.
///
/// Errors carry a google.rpc.ErrorInfo detail with a stable machine-readable
/// `reason` (e.g. INVALID_CREDENTIALS, EMAIL_NOT_VERIFIED) that SDKs map to
/// typed errors.
@$pb.GrpcServiceName('moth.auth.v1.AuthService')
class AuthServiceClient extends $grpc.Client {
  /// The hostname for this service.
  static const $core.String defaultHost = '';

  /// OAuth scopes needed for the client.
  static const $core.List<$core.String> oauthScopes = [
    '',
  ];

  AuthServiceClient(super.channel, {super.options, super.interceptors});

  /// SignUp registers a new email/password user, subject to project policy
  /// (public signup open, password length, email verification). Depending on
  /// policy the response may already include tokens, or be empty until the
  /// email is verified.
  $grpc.ResponseFuture<$0.SignUpResponse> signUp(
    $0.SignUpRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$signUp, request, options: options);
  }

  /// SignIn exchanges email/password for a token pair. The error is the same
  /// whether the email is unknown or the password wrong.
  $grpc.ResponseFuture<$0.SignInResponse> signIn(
    $0.SignInRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$signIn, request, options: options);
  }

  /// RefreshToken rotates the presented refresh token and mints a fresh
  /// access token. Presenting an already-rotated token is treated as theft:
  /// the whole token family is revoked.
  $grpc.ResponseFuture<$0.RefreshTokenResponse> refreshToken(
    $0.RefreshTokenRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$refreshToken, request, options: options);
  }

  /// SignOut revokes the presented refresh token, or every session of the
  /// user with all_devices.
  $grpc.ResponseFuture<$0.SignOutResponse> signOut(
    $0.SignOutRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$signOut, request, options: options);
  }

  /// GetMe returns the user authenticated by the access token.
  $grpc.ResponseFuture<$0.GetMeResponse> getMe(
    $0.GetMeRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$getMe, request, options: options);
  }

  /// UpdateMe updates the user's own profile fields.
  $grpc.ResponseFuture<$0.UpdateMeResponse> updateMe(
    $0.UpdateMeRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$updateMe, request, options: options);
  }

  /// ChangePassword requires the current password, revokes every other
  /// session and returns a fresh token pair for this device.
  $grpc.ResponseFuture<$0.ChangePasswordResponse> changePassword(
    $0.ChangePasswordRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$changePassword, request, options: options);
  }

  /// RequestEmailVerification (re)sends the verification email. It always
  /// returns OK so responses never reveal whether an account exists.
  $grpc.ResponseFuture<$0.RequestEmailVerificationResponse>
      requestEmailVerification(
    $0.RequestEmailVerificationRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$requestEmailVerification, request,
        options: options);
  }

  /// ConfirmEmailVerification consumes a verification token from the email
  /// link and marks the address verified.
  $grpc.ResponseFuture<$0.ConfirmEmailVerificationResponse>
      confirmEmailVerification(
    $0.ConfirmEmailVerificationRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$confirmEmailVerification, request,
        options: options);
  }

  /// RequestPasswordReset emails a reset link. It always returns OK so
  /// responses never reveal whether an account exists.
  $grpc.ResponseFuture<$0.RequestPasswordResetResponse> requestPasswordReset(
    $0.RequestPasswordResetRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$requestPasswordReset, request, options: options);
  }

  /// ConfirmPasswordReset consumes a reset token and sets the new password;
  /// every refresh token of the user is revoked.
  $grpc.ResponseFuture<$0.ConfirmPasswordResetResponse> confirmPasswordReset(
    $0.ConfirmPasswordResetRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$confirmPasswordReset, request, options: options);
  }

  /// RequestEmailChange sends a confirmation link to the new address; the
  /// account email only switches once that address is verified.
  $grpc.ResponseFuture<$0.RequestEmailChangeResponse> requestEmailChange(
    $0.RequestEmailChangeRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$requestEmailChange, request, options: options);
  }

  /// ConfirmEmailChange consumes an email-change token and applies the
  /// pending address. The previous address receives a notification with a
  /// revert link (valid 72 h) that goes through this same RPC.
  $grpc.ResponseFuture<$0.ConfirmEmailChangeResponse> confirmEmailChange(
    $0.ConfirmEmailChangeRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$confirmEmailChange, request, options: options);
  }

  /// SignInWithOAuth signs in (or up) with a provider ID token obtained by a
  /// native Google/Apple flow on the device. The token is verified
  /// server-side (signature against the provider JWKS, issuer, audience
  /// against the project's configured client/bundle IDs, expiry, nonce);
  /// email, name and subject only ever come from the verified token. Account
  /// resolution: an existing (provider, subject) identity signs that user
  /// in; else a provider-verified email matching an existing user links a
  /// new identity to it (when the project's auto_link_verified_email policy
  /// allows); else a new user is created.
  $grpc.ResponseFuture<$0.SignInWithOAuthResponse> signInWithOAuth(
    $0.SignInWithOAuthRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$signInWithOAuth, request, options: options);
  }

  /// ExchangeOAuthCode trades the one-time code minted by the web-redirect
  /// fallback flow (GET /oauth/{provider}/start → provider consent →
  /// callback → redirect back into the app) for a token pair. Codes are
  /// single-use and short-lived.
  $grpc.ResponseFuture<$0.ExchangeOAuthCodeResponse> exchangeOAuthCode(
    $0.ExchangeOAuthCodeRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$exchangeOAuthCode, request, options: options);
  }

  /// UnlinkIdentity removes the caller's identity for one provider. Requires
  /// a Bearer access token. Refused when it would leave the account without
  /// any way to sign in.
  $grpc.ResponseFuture<$0.UnlinkIdentityResponse> unlinkIdentity(
    $0.UnlinkIdentityRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$unlinkIdentity, request, options: options);
  }

  /// DeleteAccount permanently deletes the user after fresh re-authentication
  /// (App Store guideline 5.1.1). Identities, sessions and email tokens are
  /// cascaded.
  $grpc.ResponseFuture<$0.DeleteAccountResponse> deleteAccount(
    $0.DeleteAccountRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$deleteAccount, request, options: options);
  }

  // method descriptors

  static final _$signUp =
      $grpc.ClientMethod<$0.SignUpRequest, $0.SignUpResponse>(
          '/moth.auth.v1.AuthService/SignUp',
          ($0.SignUpRequest value) => value.writeToBuffer(),
          $0.SignUpResponse.fromBuffer);
  static final _$signIn =
      $grpc.ClientMethod<$0.SignInRequest, $0.SignInResponse>(
          '/moth.auth.v1.AuthService/SignIn',
          ($0.SignInRequest value) => value.writeToBuffer(),
          $0.SignInResponse.fromBuffer);
  static final _$refreshToken =
      $grpc.ClientMethod<$0.RefreshTokenRequest, $0.RefreshTokenResponse>(
          '/moth.auth.v1.AuthService/RefreshToken',
          ($0.RefreshTokenRequest value) => value.writeToBuffer(),
          $0.RefreshTokenResponse.fromBuffer);
  static final _$signOut =
      $grpc.ClientMethod<$0.SignOutRequest, $0.SignOutResponse>(
          '/moth.auth.v1.AuthService/SignOut',
          ($0.SignOutRequest value) => value.writeToBuffer(),
          $0.SignOutResponse.fromBuffer);
  static final _$getMe = $grpc.ClientMethod<$0.GetMeRequest, $0.GetMeResponse>(
      '/moth.auth.v1.AuthService/GetMe',
      ($0.GetMeRequest value) => value.writeToBuffer(),
      $0.GetMeResponse.fromBuffer);
  static final _$updateMe =
      $grpc.ClientMethod<$0.UpdateMeRequest, $0.UpdateMeResponse>(
          '/moth.auth.v1.AuthService/UpdateMe',
          ($0.UpdateMeRequest value) => value.writeToBuffer(),
          $0.UpdateMeResponse.fromBuffer);
  static final _$changePassword =
      $grpc.ClientMethod<$0.ChangePasswordRequest, $0.ChangePasswordResponse>(
          '/moth.auth.v1.AuthService/ChangePassword',
          ($0.ChangePasswordRequest value) => value.writeToBuffer(),
          $0.ChangePasswordResponse.fromBuffer);
  static final _$requestEmailVerification = $grpc.ClientMethod<
          $0.RequestEmailVerificationRequest,
          $0.RequestEmailVerificationResponse>(
      '/moth.auth.v1.AuthService/RequestEmailVerification',
      ($0.RequestEmailVerificationRequest value) => value.writeToBuffer(),
      $0.RequestEmailVerificationResponse.fromBuffer);
  static final _$confirmEmailVerification = $grpc.ClientMethod<
          $0.ConfirmEmailVerificationRequest,
          $0.ConfirmEmailVerificationResponse>(
      '/moth.auth.v1.AuthService/ConfirmEmailVerification',
      ($0.ConfirmEmailVerificationRequest value) => value.writeToBuffer(),
      $0.ConfirmEmailVerificationResponse.fromBuffer);
  static final _$requestPasswordReset = $grpc.ClientMethod<
          $0.RequestPasswordResetRequest, $0.RequestPasswordResetResponse>(
      '/moth.auth.v1.AuthService/RequestPasswordReset',
      ($0.RequestPasswordResetRequest value) => value.writeToBuffer(),
      $0.RequestPasswordResetResponse.fromBuffer);
  static final _$confirmPasswordReset = $grpc.ClientMethod<
          $0.ConfirmPasswordResetRequest, $0.ConfirmPasswordResetResponse>(
      '/moth.auth.v1.AuthService/ConfirmPasswordReset',
      ($0.ConfirmPasswordResetRequest value) => value.writeToBuffer(),
      $0.ConfirmPasswordResetResponse.fromBuffer);
  static final _$requestEmailChange = $grpc.ClientMethod<
          $0.RequestEmailChangeRequest, $0.RequestEmailChangeResponse>(
      '/moth.auth.v1.AuthService/RequestEmailChange',
      ($0.RequestEmailChangeRequest value) => value.writeToBuffer(),
      $0.RequestEmailChangeResponse.fromBuffer);
  static final _$confirmEmailChange = $grpc.ClientMethod<
          $0.ConfirmEmailChangeRequest, $0.ConfirmEmailChangeResponse>(
      '/moth.auth.v1.AuthService/ConfirmEmailChange',
      ($0.ConfirmEmailChangeRequest value) => value.writeToBuffer(),
      $0.ConfirmEmailChangeResponse.fromBuffer);
  static final _$signInWithOAuth =
      $grpc.ClientMethod<$0.SignInWithOAuthRequest, $0.SignInWithOAuthResponse>(
          '/moth.auth.v1.AuthService/SignInWithOAuth',
          ($0.SignInWithOAuthRequest value) => value.writeToBuffer(),
          $0.SignInWithOAuthResponse.fromBuffer);
  static final _$exchangeOAuthCode = $grpc.ClientMethod<
          $0.ExchangeOAuthCodeRequest, $0.ExchangeOAuthCodeResponse>(
      '/moth.auth.v1.AuthService/ExchangeOAuthCode',
      ($0.ExchangeOAuthCodeRequest value) => value.writeToBuffer(),
      $0.ExchangeOAuthCodeResponse.fromBuffer);
  static final _$unlinkIdentity =
      $grpc.ClientMethod<$0.UnlinkIdentityRequest, $0.UnlinkIdentityResponse>(
          '/moth.auth.v1.AuthService/UnlinkIdentity',
          ($0.UnlinkIdentityRequest value) => value.writeToBuffer(),
          $0.UnlinkIdentityResponse.fromBuffer);
  static final _$deleteAccount =
      $grpc.ClientMethod<$0.DeleteAccountRequest, $0.DeleteAccountResponse>(
          '/moth.auth.v1.AuthService/DeleteAccount',
          ($0.DeleteAccountRequest value) => value.writeToBuffer(),
          $0.DeleteAccountResponse.fromBuffer);
}

@$pb.GrpcServiceName('moth.auth.v1.AuthService')
abstract class AuthServiceBase extends $grpc.Service {
  $core.String get $name => 'moth.auth.v1.AuthService';

  AuthServiceBase() {
    $addMethod($grpc.ServiceMethod<$0.SignUpRequest, $0.SignUpResponse>(
        'SignUp',
        signUp_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.SignUpRequest.fromBuffer(value),
        ($0.SignUpResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.SignInRequest, $0.SignInResponse>(
        'SignIn',
        signIn_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.SignInRequest.fromBuffer(value),
        ($0.SignInResponse value) => value.writeToBuffer()));
    $addMethod(
        $grpc.ServiceMethod<$0.RefreshTokenRequest, $0.RefreshTokenResponse>(
            'RefreshToken',
            refreshToken_Pre,
            false,
            false,
            ($core.List<$core.int> value) =>
                $0.RefreshTokenRequest.fromBuffer(value),
            ($0.RefreshTokenResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.SignOutRequest, $0.SignOutResponse>(
        'SignOut',
        signOut_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.SignOutRequest.fromBuffer(value),
        ($0.SignOutResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.GetMeRequest, $0.GetMeResponse>(
        'GetMe',
        getMe_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.GetMeRequest.fromBuffer(value),
        ($0.GetMeResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.UpdateMeRequest, $0.UpdateMeResponse>(
        'UpdateMe',
        updateMe_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.UpdateMeRequest.fromBuffer(value),
        ($0.UpdateMeResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.ChangePasswordRequest,
            $0.ChangePasswordResponse>(
        'ChangePassword',
        changePassword_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.ChangePasswordRequest.fromBuffer(value),
        ($0.ChangePasswordResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.RequestEmailVerificationRequest,
            $0.RequestEmailVerificationResponse>(
        'RequestEmailVerification',
        requestEmailVerification_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.RequestEmailVerificationRequest.fromBuffer(value),
        ($0.RequestEmailVerificationResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.ConfirmEmailVerificationRequest,
            $0.ConfirmEmailVerificationResponse>(
        'ConfirmEmailVerification',
        confirmEmailVerification_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.ConfirmEmailVerificationRequest.fromBuffer(value),
        ($0.ConfirmEmailVerificationResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.RequestPasswordResetRequest,
            $0.RequestPasswordResetResponse>(
        'RequestPasswordReset',
        requestPasswordReset_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.RequestPasswordResetRequest.fromBuffer(value),
        ($0.RequestPasswordResetResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.ConfirmPasswordResetRequest,
            $0.ConfirmPasswordResetResponse>(
        'ConfirmPasswordReset',
        confirmPasswordReset_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.ConfirmPasswordResetRequest.fromBuffer(value),
        ($0.ConfirmPasswordResetResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.RequestEmailChangeRequest,
            $0.RequestEmailChangeResponse>(
        'RequestEmailChange',
        requestEmailChange_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.RequestEmailChangeRequest.fromBuffer(value),
        ($0.RequestEmailChangeResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.ConfirmEmailChangeRequest,
            $0.ConfirmEmailChangeResponse>(
        'ConfirmEmailChange',
        confirmEmailChange_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.ConfirmEmailChangeRequest.fromBuffer(value),
        ($0.ConfirmEmailChangeResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.SignInWithOAuthRequest,
            $0.SignInWithOAuthResponse>(
        'SignInWithOAuth',
        signInWithOAuth_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.SignInWithOAuthRequest.fromBuffer(value),
        ($0.SignInWithOAuthResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.ExchangeOAuthCodeRequest,
            $0.ExchangeOAuthCodeResponse>(
        'ExchangeOAuthCode',
        exchangeOAuthCode_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.ExchangeOAuthCodeRequest.fromBuffer(value),
        ($0.ExchangeOAuthCodeResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.UnlinkIdentityRequest,
            $0.UnlinkIdentityResponse>(
        'UnlinkIdentity',
        unlinkIdentity_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.UnlinkIdentityRequest.fromBuffer(value),
        ($0.UnlinkIdentityResponse value) => value.writeToBuffer()));
    $addMethod(
        $grpc.ServiceMethod<$0.DeleteAccountRequest, $0.DeleteAccountResponse>(
            'DeleteAccount',
            deleteAccount_Pre,
            false,
            false,
            ($core.List<$core.int> value) =>
                $0.DeleteAccountRequest.fromBuffer(value),
            ($0.DeleteAccountResponse value) => value.writeToBuffer()));
  }

  $async.Future<$0.SignUpResponse> signUp_Pre(
      $grpc.ServiceCall $call, $async.Future<$0.SignUpRequest> $request) async {
    return signUp($call, await $request);
  }

  $async.Future<$0.SignUpResponse> signUp(
      $grpc.ServiceCall call, $0.SignUpRequest request);

  $async.Future<$0.SignInResponse> signIn_Pre(
      $grpc.ServiceCall $call, $async.Future<$0.SignInRequest> $request) async {
    return signIn($call, await $request);
  }

  $async.Future<$0.SignInResponse> signIn(
      $grpc.ServiceCall call, $0.SignInRequest request);

  $async.Future<$0.RefreshTokenResponse> refreshToken_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.RefreshTokenRequest> $request) async {
    return refreshToken($call, await $request);
  }

  $async.Future<$0.RefreshTokenResponse> refreshToken(
      $grpc.ServiceCall call, $0.RefreshTokenRequest request);

  $async.Future<$0.SignOutResponse> signOut_Pre($grpc.ServiceCall $call,
      $async.Future<$0.SignOutRequest> $request) async {
    return signOut($call, await $request);
  }

  $async.Future<$0.SignOutResponse> signOut(
      $grpc.ServiceCall call, $0.SignOutRequest request);

  $async.Future<$0.GetMeResponse> getMe_Pre(
      $grpc.ServiceCall $call, $async.Future<$0.GetMeRequest> $request) async {
    return getMe($call, await $request);
  }

  $async.Future<$0.GetMeResponse> getMe(
      $grpc.ServiceCall call, $0.GetMeRequest request);

  $async.Future<$0.UpdateMeResponse> updateMe_Pre($grpc.ServiceCall $call,
      $async.Future<$0.UpdateMeRequest> $request) async {
    return updateMe($call, await $request);
  }

  $async.Future<$0.UpdateMeResponse> updateMe(
      $grpc.ServiceCall call, $0.UpdateMeRequest request);

  $async.Future<$0.ChangePasswordResponse> changePassword_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.ChangePasswordRequest> $request) async {
    return changePassword($call, await $request);
  }

  $async.Future<$0.ChangePasswordResponse> changePassword(
      $grpc.ServiceCall call, $0.ChangePasswordRequest request);

  $async.Future<$0.RequestEmailVerificationResponse>
      requestEmailVerification_Pre($grpc.ServiceCall $call,
          $async.Future<$0.RequestEmailVerificationRequest> $request) async {
    return requestEmailVerification($call, await $request);
  }

  $async.Future<$0.RequestEmailVerificationResponse> requestEmailVerification(
      $grpc.ServiceCall call, $0.RequestEmailVerificationRequest request);

  $async.Future<$0.ConfirmEmailVerificationResponse>
      confirmEmailVerification_Pre($grpc.ServiceCall $call,
          $async.Future<$0.ConfirmEmailVerificationRequest> $request) async {
    return confirmEmailVerification($call, await $request);
  }

  $async.Future<$0.ConfirmEmailVerificationResponse> confirmEmailVerification(
      $grpc.ServiceCall call, $0.ConfirmEmailVerificationRequest request);

  $async.Future<$0.RequestPasswordResetResponse> requestPasswordReset_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.RequestPasswordResetRequest> $request) async {
    return requestPasswordReset($call, await $request);
  }

  $async.Future<$0.RequestPasswordResetResponse> requestPasswordReset(
      $grpc.ServiceCall call, $0.RequestPasswordResetRequest request);

  $async.Future<$0.ConfirmPasswordResetResponse> confirmPasswordReset_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.ConfirmPasswordResetRequest> $request) async {
    return confirmPasswordReset($call, await $request);
  }

  $async.Future<$0.ConfirmPasswordResetResponse> confirmPasswordReset(
      $grpc.ServiceCall call, $0.ConfirmPasswordResetRequest request);

  $async.Future<$0.RequestEmailChangeResponse> requestEmailChange_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.RequestEmailChangeRequest> $request) async {
    return requestEmailChange($call, await $request);
  }

  $async.Future<$0.RequestEmailChangeResponse> requestEmailChange(
      $grpc.ServiceCall call, $0.RequestEmailChangeRequest request);

  $async.Future<$0.ConfirmEmailChangeResponse> confirmEmailChange_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.ConfirmEmailChangeRequest> $request) async {
    return confirmEmailChange($call, await $request);
  }

  $async.Future<$0.ConfirmEmailChangeResponse> confirmEmailChange(
      $grpc.ServiceCall call, $0.ConfirmEmailChangeRequest request);

  $async.Future<$0.SignInWithOAuthResponse> signInWithOAuth_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.SignInWithOAuthRequest> $request) async {
    return signInWithOAuth($call, await $request);
  }

  $async.Future<$0.SignInWithOAuthResponse> signInWithOAuth(
      $grpc.ServiceCall call, $0.SignInWithOAuthRequest request);

  $async.Future<$0.ExchangeOAuthCodeResponse> exchangeOAuthCode_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.ExchangeOAuthCodeRequest> $request) async {
    return exchangeOAuthCode($call, await $request);
  }

  $async.Future<$0.ExchangeOAuthCodeResponse> exchangeOAuthCode(
      $grpc.ServiceCall call, $0.ExchangeOAuthCodeRequest request);

  $async.Future<$0.UnlinkIdentityResponse> unlinkIdentity_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.UnlinkIdentityRequest> $request) async {
    return unlinkIdentity($call, await $request);
  }

  $async.Future<$0.UnlinkIdentityResponse> unlinkIdentity(
      $grpc.ServiceCall call, $0.UnlinkIdentityRequest request);

  $async.Future<$0.DeleteAccountResponse> deleteAccount_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.DeleteAccountRequest> $request) async {
    return deleteAccount($call, await $request);
  }

  $async.Future<$0.DeleteAccountResponse> deleteAccount(
      $grpc.ServiceCall call, $0.DeleteAccountRequest request);
}
