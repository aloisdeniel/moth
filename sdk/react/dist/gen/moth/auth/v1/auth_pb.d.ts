import type { GenEnum, GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file moth/auth/v1/auth.proto.
 */
export declare const file_moth_auth_v1_auth: GenFile;
/**
 * User is the caller's own account as exposed to the app.
 *
 * @generated from message moth.auth.v1.User
 */
export type User = Message<"moth.auth.v1.User"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string email = 2;
     */
    email: string;
    /**
     * @generated from field: bool email_verified = 3;
     */
    emailVerified: boolean;
    /**
     * @generated from field: string display_name = 4;
     */
    displayName: string;
    /**
     * @generated from field: string avatar_url = 5;
     */
    avatarUrl: string;
    /**
     * @generated from field: google.protobuf.Timestamp create_time = 6;
     */
    createTime?: Timestamp | undefined;
};
/**
 * Describes the message moth.auth.v1.User.
 * Use `create(UserSchema)` to create a new message.
 */
export declare const UserSchema: GenMessage<User>;
/**
 * TokenPair is one authenticated session: a short-lived ES256 JWT plus the
 * opaque rotating refresh token that renews it.
 *
 * @generated from message moth.auth.v1.TokenPair
 */
export type TokenPair = Message<"moth.auth.v1.TokenPair"> & {
    /**
     * @generated from field: string access_token = 1;
     */
    accessToken: string;
    /**
     * @generated from field: string refresh_token = 2;
     */
    refreshToken: string;
    /**
     * Access token lifetime in seconds.
     *
     * @generated from field: int64 expires_in = 3;
     */
    expiresIn: bigint;
};
/**
 * Describes the message moth.auth.v1.TokenPair.
 * Use `create(TokenPairSchema)` to create a new message.
 */
export declare const TokenPairSchema: GenMessage<TokenPair>;
/**
 * @generated from message moth.auth.v1.SignUpRequest
 */
export type SignUpRequest = Message<"moth.auth.v1.SignUpRequest"> & {
    /**
     * @generated from field: string email = 1;
     */
    email: string;
    /**
     * @generated from field: string password = 2;
     */
    password: string;
    /**
     * @generated from field: string display_name = 3;
     */
    displayName: string;
    /**
     * Free-form device description stored with the session, e.g. "iPhone 15".
     *
     * @generated from field: string device_info = 4;
     */
    deviceInfo: string;
    /**
     * Optional CAPTCHA solution, verified when the project configures a
     * captcha_verify_url (off by default; enforcement is a documented hook).
     *
     * @generated from field: string captcha_token = 5;
     */
    captchaToken: string;
};
/**
 * Describes the message moth.auth.v1.SignUpRequest.
 * Use `create(SignUpRequestSchema)` to create a new message.
 */
export declare const SignUpRequestSchema: GenMessage<SignUpRequest>;
/**
 * @generated from message moth.auth.v1.SignUpResponse
 */
export type SignUpResponse = Message<"moth.auth.v1.SignUpResponse"> & {
    /**
     * Unset when project policy withholds it: enumeration-safe projects
     * always return an empty response, and projects requiring verification
     * return the user without tokens.
     *
     * @generated from field: moth.auth.v1.User user = 1;
     */
    user?: User | undefined;
    /**
     * Set only when the user may sign in immediately.
     *
     * @generated from field: moth.auth.v1.TokenPair tokens = 2;
     */
    tokens?: TokenPair | undefined;
};
/**
 * Describes the message moth.auth.v1.SignUpResponse.
 * Use `create(SignUpResponseSchema)` to create a new message.
 */
export declare const SignUpResponseSchema: GenMessage<SignUpResponse>;
/**
 * @generated from message moth.auth.v1.SignInRequest
 */
export type SignInRequest = Message<"moth.auth.v1.SignInRequest"> & {
    /**
     * @generated from field: string email = 1;
     */
    email: string;
    /**
     * @generated from field: string password = 2;
     */
    password: string;
    /**
     * @generated from field: string device_info = 3;
     */
    deviceInfo: string;
};
/**
 * Describes the message moth.auth.v1.SignInRequest.
 * Use `create(SignInRequestSchema)` to create a new message.
 */
export declare const SignInRequestSchema: GenMessage<SignInRequest>;
/**
 * @generated from message moth.auth.v1.SignInResponse
 */
export type SignInResponse = Message<"moth.auth.v1.SignInResponse"> & {
    /**
     * @generated from field: moth.auth.v1.User user = 1;
     */
    user?: User | undefined;
    /**
     * @generated from field: moth.auth.v1.TokenPair tokens = 2;
     */
    tokens?: TokenPair | undefined;
};
/**
 * Describes the message moth.auth.v1.SignInResponse.
 * Use `create(SignInResponseSchema)` to create a new message.
 */
export declare const SignInResponseSchema: GenMessage<SignInResponse>;
/**
 * @generated from message moth.auth.v1.RefreshTokenRequest
 */
export type RefreshTokenRequest = Message<"moth.auth.v1.RefreshTokenRequest"> & {
    /**
     * @generated from field: string refresh_token = 1;
     */
    refreshToken: string;
};
/**
 * Describes the message moth.auth.v1.RefreshTokenRequest.
 * Use `create(RefreshTokenRequestSchema)` to create a new message.
 */
export declare const RefreshTokenRequestSchema: GenMessage<RefreshTokenRequest>;
/**
 * @generated from message moth.auth.v1.RefreshTokenResponse
 */
export type RefreshTokenResponse = Message<"moth.auth.v1.RefreshTokenResponse"> & {
    /**
     * @generated from field: moth.auth.v1.User user = 1;
     */
    user?: User | undefined;
    /**
     * @generated from field: moth.auth.v1.TokenPair tokens = 2;
     */
    tokens?: TokenPair | undefined;
};
/**
 * Describes the message moth.auth.v1.RefreshTokenResponse.
 * Use `create(RefreshTokenResponseSchema)` to create a new message.
 */
export declare const RefreshTokenResponseSchema: GenMessage<RefreshTokenResponse>;
/**
 * @generated from message moth.auth.v1.SignOutRequest
 */
export type SignOutRequest = Message<"moth.auth.v1.SignOutRequest"> & {
    /**
     * @generated from field: string refresh_token = 1;
     */
    refreshToken: string;
    /**
     * Revoke every session of the user, not just this one.
     *
     * @generated from field: bool all_devices = 2;
     */
    allDevices: boolean;
};
/**
 * Describes the message moth.auth.v1.SignOutRequest.
 * Use `create(SignOutRequestSchema)` to create a new message.
 */
export declare const SignOutRequestSchema: GenMessage<SignOutRequest>;
/**
 * @generated from message moth.auth.v1.SignOutResponse
 */
export type SignOutResponse = Message<"moth.auth.v1.SignOutResponse"> & {};
/**
 * Describes the message moth.auth.v1.SignOutResponse.
 * Use `create(SignOutResponseSchema)` to create a new message.
 */
export declare const SignOutResponseSchema: GenMessage<SignOutResponse>;
/**
 * @generated from message moth.auth.v1.GetMeRequest
 */
export type GetMeRequest = Message<"moth.auth.v1.GetMeRequest"> & {};
/**
 * Describes the message moth.auth.v1.GetMeRequest.
 * Use `create(GetMeRequestSchema)` to create a new message.
 */
export declare const GetMeRequestSchema: GenMessage<GetMeRequest>;
/**
 * @generated from message moth.auth.v1.GetMeResponse
 */
export type GetMeResponse = Message<"moth.auth.v1.GetMeResponse"> & {
    /**
     * @generated from field: moth.auth.v1.User user = 1;
     */
    user?: User | undefined;
};
/**
 * Describes the message moth.auth.v1.GetMeResponse.
 * Use `create(GetMeResponseSchema)` to create a new message.
 */
export declare const GetMeResponseSchema: GenMessage<GetMeResponse>;
/**
 * @generated from message moth.auth.v1.UpdateMeRequest
 */
export type UpdateMeRequest = Message<"moth.auth.v1.UpdateMeRequest"> & {
    /**
     * @generated from field: optional string display_name = 1;
     */
    displayName?: string | undefined;
    /**
     * @generated from field: optional string avatar_url = 2;
     */
    avatarUrl?: string | undefined;
};
/**
 * Describes the message moth.auth.v1.UpdateMeRequest.
 * Use `create(UpdateMeRequestSchema)` to create a new message.
 */
export declare const UpdateMeRequestSchema: GenMessage<UpdateMeRequest>;
/**
 * @generated from message moth.auth.v1.UpdateMeResponse
 */
export type UpdateMeResponse = Message<"moth.auth.v1.UpdateMeResponse"> & {
    /**
     * @generated from field: moth.auth.v1.User user = 1;
     */
    user?: User | undefined;
};
/**
 * Describes the message moth.auth.v1.UpdateMeResponse.
 * Use `create(UpdateMeResponseSchema)` to create a new message.
 */
export declare const UpdateMeResponseSchema: GenMessage<UpdateMeResponse>;
/**
 * @generated from message moth.auth.v1.ChangePasswordRequest
 */
export type ChangePasswordRequest = Message<"moth.auth.v1.ChangePasswordRequest"> & {
    /**
     * @generated from field: string current_password = 1;
     */
    currentPassword: string;
    /**
     * @generated from field: string new_password = 2;
     */
    newPassword: string;
};
/**
 * Describes the message moth.auth.v1.ChangePasswordRequest.
 * Use `create(ChangePasswordRequestSchema)` to create a new message.
 */
export declare const ChangePasswordRequestSchema: GenMessage<ChangePasswordRequest>;
/**
 * @generated from message moth.auth.v1.ChangePasswordResponse
 */
export type ChangePasswordResponse = Message<"moth.auth.v1.ChangePasswordResponse"> & {
    /**
     * A fresh session for this device; all other sessions are revoked.
     *
     * @generated from field: moth.auth.v1.TokenPair tokens = 1;
     */
    tokens?: TokenPair | undefined;
};
/**
 * Describes the message moth.auth.v1.ChangePasswordResponse.
 * Use `create(ChangePasswordResponseSchema)` to create a new message.
 */
export declare const ChangePasswordResponseSchema: GenMessage<ChangePasswordResponse>;
/**
 * @generated from message moth.auth.v1.RequestEmailVerificationRequest
 */
export type RequestEmailVerificationRequest = Message<"moth.auth.v1.RequestEmailVerificationRequest"> & {
    /**
     * @generated from field: string email = 1;
     */
    email: string;
};
/**
 * Describes the message moth.auth.v1.RequestEmailVerificationRequest.
 * Use `create(RequestEmailVerificationRequestSchema)` to create a new message.
 */
export declare const RequestEmailVerificationRequestSchema: GenMessage<RequestEmailVerificationRequest>;
/**
 * @generated from message moth.auth.v1.RequestEmailVerificationResponse
 */
export type RequestEmailVerificationResponse = Message<"moth.auth.v1.RequestEmailVerificationResponse"> & {};
/**
 * Describes the message moth.auth.v1.RequestEmailVerificationResponse.
 * Use `create(RequestEmailVerificationResponseSchema)` to create a new message.
 */
export declare const RequestEmailVerificationResponseSchema: GenMessage<RequestEmailVerificationResponse>;
/**
 * @generated from message moth.auth.v1.ConfirmEmailVerificationRequest
 */
export type ConfirmEmailVerificationRequest = Message<"moth.auth.v1.ConfirmEmailVerificationRequest"> & {
    /**
     * @generated from field: string token = 1;
     */
    token: string;
};
/**
 * Describes the message moth.auth.v1.ConfirmEmailVerificationRequest.
 * Use `create(ConfirmEmailVerificationRequestSchema)` to create a new message.
 */
export declare const ConfirmEmailVerificationRequestSchema: GenMessage<ConfirmEmailVerificationRequest>;
/**
 * @generated from message moth.auth.v1.ConfirmEmailVerificationResponse
 */
export type ConfirmEmailVerificationResponse = Message<"moth.auth.v1.ConfirmEmailVerificationResponse"> & {};
/**
 * Describes the message moth.auth.v1.ConfirmEmailVerificationResponse.
 * Use `create(ConfirmEmailVerificationResponseSchema)` to create a new message.
 */
export declare const ConfirmEmailVerificationResponseSchema: GenMessage<ConfirmEmailVerificationResponse>;
/**
 * @generated from message moth.auth.v1.RequestPasswordResetRequest
 */
export type RequestPasswordResetRequest = Message<"moth.auth.v1.RequestPasswordResetRequest"> & {
    /**
     * @generated from field: string email = 1;
     */
    email: string;
};
/**
 * Describes the message moth.auth.v1.RequestPasswordResetRequest.
 * Use `create(RequestPasswordResetRequestSchema)` to create a new message.
 */
export declare const RequestPasswordResetRequestSchema: GenMessage<RequestPasswordResetRequest>;
/**
 * @generated from message moth.auth.v1.RequestPasswordResetResponse
 */
export type RequestPasswordResetResponse = Message<"moth.auth.v1.RequestPasswordResetResponse"> & {};
/**
 * Describes the message moth.auth.v1.RequestPasswordResetResponse.
 * Use `create(RequestPasswordResetResponseSchema)` to create a new message.
 */
export declare const RequestPasswordResetResponseSchema: GenMessage<RequestPasswordResetResponse>;
/**
 * @generated from message moth.auth.v1.ConfirmPasswordResetRequest
 */
export type ConfirmPasswordResetRequest = Message<"moth.auth.v1.ConfirmPasswordResetRequest"> & {
    /**
     * @generated from field: string token = 1;
     */
    token: string;
    /**
     * @generated from field: string new_password = 2;
     */
    newPassword: string;
};
/**
 * Describes the message moth.auth.v1.ConfirmPasswordResetRequest.
 * Use `create(ConfirmPasswordResetRequestSchema)` to create a new message.
 */
export declare const ConfirmPasswordResetRequestSchema: GenMessage<ConfirmPasswordResetRequest>;
/**
 * @generated from message moth.auth.v1.ConfirmPasswordResetResponse
 */
export type ConfirmPasswordResetResponse = Message<"moth.auth.v1.ConfirmPasswordResetResponse"> & {};
/**
 * Describes the message moth.auth.v1.ConfirmPasswordResetResponse.
 * Use `create(ConfirmPasswordResetResponseSchema)` to create a new message.
 */
export declare const ConfirmPasswordResetResponseSchema: GenMessage<ConfirmPasswordResetResponse>;
/**
 * @generated from message moth.auth.v1.RequestEmailChangeRequest
 */
export type RequestEmailChangeRequest = Message<"moth.auth.v1.RequestEmailChangeRequest"> & {
    /**
     * @generated from field: string new_email = 1;
     */
    newEmail: string;
};
/**
 * Describes the message moth.auth.v1.RequestEmailChangeRequest.
 * Use `create(RequestEmailChangeRequestSchema)` to create a new message.
 */
export declare const RequestEmailChangeRequestSchema: GenMessage<RequestEmailChangeRequest>;
/**
 * @generated from message moth.auth.v1.RequestEmailChangeResponse
 */
export type RequestEmailChangeResponse = Message<"moth.auth.v1.RequestEmailChangeResponse"> & {};
/**
 * Describes the message moth.auth.v1.RequestEmailChangeResponse.
 * Use `create(RequestEmailChangeResponseSchema)` to create a new message.
 */
export declare const RequestEmailChangeResponseSchema: GenMessage<RequestEmailChangeResponse>;
/**
 * @generated from message moth.auth.v1.ConfirmEmailChangeRequest
 */
export type ConfirmEmailChangeRequest = Message<"moth.auth.v1.ConfirmEmailChangeRequest"> & {
    /**
     * @generated from field: string token = 1;
     */
    token: string;
};
/**
 * Describes the message moth.auth.v1.ConfirmEmailChangeRequest.
 * Use `create(ConfirmEmailChangeRequestSchema)` to create a new message.
 */
export declare const ConfirmEmailChangeRequestSchema: GenMessage<ConfirmEmailChangeRequest>;
/**
 * @generated from message moth.auth.v1.ConfirmEmailChangeResponse
 */
export type ConfirmEmailChangeResponse = Message<"moth.auth.v1.ConfirmEmailChangeResponse"> & {};
/**
 * Describes the message moth.auth.v1.ConfirmEmailChangeResponse.
 * Use `create(ConfirmEmailChangeResponseSchema)` to create a new message.
 */
export declare const ConfirmEmailChangeResponseSchema: GenMessage<ConfirmEmailChangeResponse>;
/**
 * @generated from message moth.auth.v1.SignInWithOAuthRequest
 */
export type SignInWithOAuthRequest = Message<"moth.auth.v1.SignInWithOAuthRequest"> & {
    /**
     * @generated from field: moth.auth.v1.OAuthProvider provider = 1;
     */
    provider: OAuthProvider;
    /**
     * The provider-issued OIDC ID token (JWT).
     *
     * @generated from field: string id_token = 2;
     */
    idToken: string;
    /**
     * The raw per-attempt nonce the SDK generated for this sign-in. The
     * server requires the ID token's `nonce` claim to match (Apple carries
     * its SHA-256 per their scheme), so replayed ID tokens are rejected.
     *
     * @generated from field: string nonce = 3;
     */
    nonce: string;
    /**
     * Apple only: the authorization code from the native flow, exchanged
     * server-side for the refresh token that account deletion later revokes
     * (App Store requirement).
     *
     * @generated from field: string authorization_code = 4;
     */
    authorizationCode: string;
    /**
     * Apple only: the user's name, which Apple exposes solely to the app and
     * solely on first authorization. Client-asserted — used for the initial
     * display name, never for identity resolution.
     *
     * @generated from field: string given_name = 5;
     */
    givenName: string;
    /**
     * @generated from field: string family_name = 6;
     */
    familyName: string;
    /**
     * Free-form device description stored with the session, e.g. "iPhone 15".
     *
     * @generated from field: string device_info = 7;
     */
    deviceInfo: string;
};
/**
 * Describes the message moth.auth.v1.SignInWithOAuthRequest.
 * Use `create(SignInWithOAuthRequestSchema)` to create a new message.
 */
export declare const SignInWithOAuthRequestSchema: GenMessage<SignInWithOAuthRequest>;
/**
 * @generated from message moth.auth.v1.SignInWithOAuthResponse
 */
export type SignInWithOAuthResponse = Message<"moth.auth.v1.SignInWithOAuthResponse"> & {
    /**
     * @generated from field: moth.auth.v1.User user = 1;
     */
    user?: User | undefined;
    /**
     * @generated from field: moth.auth.v1.TokenPair tokens = 2;
     */
    tokens?: TokenPair | undefined;
};
/**
 * Describes the message moth.auth.v1.SignInWithOAuthResponse.
 * Use `create(SignInWithOAuthResponseSchema)` to create a new message.
 */
export declare const SignInWithOAuthResponseSchema: GenMessage<SignInWithOAuthResponse>;
/**
 * @generated from message moth.auth.v1.ExchangeOAuthCodeRequest
 */
export type ExchangeOAuthCodeRequest = Message<"moth.auth.v1.ExchangeOAuthCodeRequest"> & {
    /**
     * The one-time code from the web-redirect callback.
     *
     * @generated from field: string code = 1;
     */
    code: string;
    /**
     * @generated from field: string device_info = 2;
     */
    deviceInfo: string;
};
/**
 * Describes the message moth.auth.v1.ExchangeOAuthCodeRequest.
 * Use `create(ExchangeOAuthCodeRequestSchema)` to create a new message.
 */
export declare const ExchangeOAuthCodeRequestSchema: GenMessage<ExchangeOAuthCodeRequest>;
/**
 * @generated from message moth.auth.v1.ExchangeOAuthCodeResponse
 */
export type ExchangeOAuthCodeResponse = Message<"moth.auth.v1.ExchangeOAuthCodeResponse"> & {
    /**
     * @generated from field: moth.auth.v1.User user = 1;
     */
    user?: User | undefined;
    /**
     * @generated from field: moth.auth.v1.TokenPair tokens = 2;
     */
    tokens?: TokenPair | undefined;
};
/**
 * Describes the message moth.auth.v1.ExchangeOAuthCodeResponse.
 * Use `create(ExchangeOAuthCodeResponseSchema)` to create a new message.
 */
export declare const ExchangeOAuthCodeResponseSchema: GenMessage<ExchangeOAuthCodeResponse>;
/**
 * @generated from message moth.auth.v1.UnlinkIdentityRequest
 */
export type UnlinkIdentityRequest = Message<"moth.auth.v1.UnlinkIdentityRequest"> & {
    /**
     * @generated from field: moth.auth.v1.OAuthProvider provider = 1;
     */
    provider: OAuthProvider;
};
/**
 * Describes the message moth.auth.v1.UnlinkIdentityRequest.
 * Use `create(UnlinkIdentityRequestSchema)` to create a new message.
 */
export declare const UnlinkIdentityRequestSchema: GenMessage<UnlinkIdentityRequest>;
/**
 * @generated from message moth.auth.v1.UnlinkIdentityResponse
 */
export type UnlinkIdentityResponse = Message<"moth.auth.v1.UnlinkIdentityResponse"> & {};
/**
 * Describes the message moth.auth.v1.UnlinkIdentityResponse.
 * Use `create(UnlinkIdentityResponseSchema)` to create a new message.
 */
export declare const UnlinkIdentityResponseSchema: GenMessage<UnlinkIdentityResponse>;
/**
 * @generated from message moth.auth.v1.DeleteAccountRequest
 */
export type DeleteAccountRequest = Message<"moth.auth.v1.DeleteAccountRequest"> & {
    /**
     * Fresh re-authentication: the current password. (Recent social sign-in
     * for social-only users arrives with milestone 04.)
     *
     * @generated from field: string password = 1;
     */
    password: string;
};
/**
 * Describes the message moth.auth.v1.DeleteAccountRequest.
 * Use `create(DeleteAccountRequestSchema)` to create a new message.
 */
export declare const DeleteAccountRequestSchema: GenMessage<DeleteAccountRequest>;
/**
 * @generated from message moth.auth.v1.DeleteAccountResponse
 */
export type DeleteAccountResponse = Message<"moth.auth.v1.DeleteAccountResponse"> & {};
/**
 * Describes the message moth.auth.v1.DeleteAccountResponse.
 * Use `create(DeleteAccountResponseSchema)` to create a new message.
 */
export declare const DeleteAccountResponseSchema: GenMessage<DeleteAccountResponse>;
/**
 * OAuthProvider identifies a supported social sign-in provider.
 * (buf splits "OAuth" as "O_Auth"; the natural OAUTH_ prefix is kept.)
 *
 * @generated from enum moth.auth.v1.OAuthProvider
 */
export declare enum OAuthProvider {
    /**
     * buf:lint:ignore ENUM_VALUE_PREFIX
     *
     * @generated from enum value: OAUTH_PROVIDER_UNSPECIFIED = 0;
     */
    OAUTH_PROVIDER_UNSPECIFIED = 0,
    /**
     * buf:lint:ignore ENUM_VALUE_PREFIX
     *
     * @generated from enum value: OAUTH_PROVIDER_GOOGLE = 1;
     */
    OAUTH_PROVIDER_GOOGLE = 1,
    /**
     * buf:lint:ignore ENUM_VALUE_PREFIX
     *
     * @generated from enum value: OAUTH_PROVIDER_APPLE = 2;
     */
    OAUTH_PROVIDER_APPLE = 2
}
/**
 * Describes the enum moth.auth.v1.OAuthProvider.
 */
export declare const OAuthProviderSchema: GenEnum<OAuthProvider>;
/**
 * AuthService is the public end-user authentication API consumed by mobile
 * apps (via the SDK). Every call carries the project's publishable key in
 * `x-moth-key: pk_...` request metadata; an interceptor resolves it to the
 * project, so users, tokens and emails are always project-scoped.
 *
 * RPCs about the current user (GetMe, UpdateMe, ChangePassword,
 * RequestEmailChange, DeleteAccount) additionally require a valid access
 * token in `authorization: Bearer ...` metadata.
 *
 * Errors carry a google.rpc.ErrorInfo detail with a stable machine-readable
 * `reason` (e.g. INVALID_CREDENTIALS, EMAIL_NOT_VERIFIED) that SDKs map to
 * typed errors.
 *
 * @generated from service moth.auth.v1.AuthService
 */
export declare const AuthService: GenService<{
    /**
     * SignUp registers a new email/password user, subject to project policy
     * (public signup open, password length, email verification). Depending on
     * policy the response may already include tokens, or be empty until the
     * email is verified.
     *
     * @generated from rpc moth.auth.v1.AuthService.SignUp
     */
    signUp: {
        methodKind: "unary";
        input: typeof SignUpRequestSchema;
        output: typeof SignUpResponseSchema;
    };
    /**
     * SignIn exchanges email/password for a token pair. The error is the same
     * whether the email is unknown or the password wrong.
     *
     * @generated from rpc moth.auth.v1.AuthService.SignIn
     */
    signIn: {
        methodKind: "unary";
        input: typeof SignInRequestSchema;
        output: typeof SignInResponseSchema;
    };
    /**
     * RefreshToken rotates the presented refresh token and mints a fresh
     * access token. Presenting an already-rotated token is treated as theft:
     * the whole token family is revoked.
     *
     * @generated from rpc moth.auth.v1.AuthService.RefreshToken
     */
    refreshToken: {
        methodKind: "unary";
        input: typeof RefreshTokenRequestSchema;
        output: typeof RefreshTokenResponseSchema;
    };
    /**
     * SignOut revokes the presented refresh token, or every session of the
     * user with all_devices.
     *
     * @generated from rpc moth.auth.v1.AuthService.SignOut
     */
    signOut: {
        methodKind: "unary";
        input: typeof SignOutRequestSchema;
        output: typeof SignOutResponseSchema;
    };
    /**
     * GetMe returns the user authenticated by the access token.
     *
     * @generated from rpc moth.auth.v1.AuthService.GetMe
     */
    getMe: {
        methodKind: "unary";
        input: typeof GetMeRequestSchema;
        output: typeof GetMeResponseSchema;
    };
    /**
     * UpdateMe updates the user's own profile fields.
     *
     * @generated from rpc moth.auth.v1.AuthService.UpdateMe
     */
    updateMe: {
        methodKind: "unary";
        input: typeof UpdateMeRequestSchema;
        output: typeof UpdateMeResponseSchema;
    };
    /**
     * ChangePassword requires the current password, revokes every other
     * session and returns a fresh token pair for this device.
     *
     * @generated from rpc moth.auth.v1.AuthService.ChangePassword
     */
    changePassword: {
        methodKind: "unary";
        input: typeof ChangePasswordRequestSchema;
        output: typeof ChangePasswordResponseSchema;
    };
    /**
     * RequestEmailVerification (re)sends the verification email. It always
     * returns OK so responses never reveal whether an account exists.
     *
     * @generated from rpc moth.auth.v1.AuthService.RequestEmailVerification
     */
    requestEmailVerification: {
        methodKind: "unary";
        input: typeof RequestEmailVerificationRequestSchema;
        output: typeof RequestEmailVerificationResponseSchema;
    };
    /**
     * ConfirmEmailVerification consumes a verification token from the email
     * link and marks the address verified.
     *
     * @generated from rpc moth.auth.v1.AuthService.ConfirmEmailVerification
     */
    confirmEmailVerification: {
        methodKind: "unary";
        input: typeof ConfirmEmailVerificationRequestSchema;
        output: typeof ConfirmEmailVerificationResponseSchema;
    };
    /**
     * RequestPasswordReset emails a reset link. It always returns OK so
     * responses never reveal whether an account exists.
     *
     * @generated from rpc moth.auth.v1.AuthService.RequestPasswordReset
     */
    requestPasswordReset: {
        methodKind: "unary";
        input: typeof RequestPasswordResetRequestSchema;
        output: typeof RequestPasswordResetResponseSchema;
    };
    /**
     * ConfirmPasswordReset consumes a reset token and sets the new password;
     * every refresh token of the user is revoked.
     *
     * @generated from rpc moth.auth.v1.AuthService.ConfirmPasswordReset
     */
    confirmPasswordReset: {
        methodKind: "unary";
        input: typeof ConfirmPasswordResetRequestSchema;
        output: typeof ConfirmPasswordResetResponseSchema;
    };
    /**
     * RequestEmailChange sends a confirmation link to the new address; the
     * account email only switches once that address is verified.
     *
     * @generated from rpc moth.auth.v1.AuthService.RequestEmailChange
     */
    requestEmailChange: {
        methodKind: "unary";
        input: typeof RequestEmailChangeRequestSchema;
        output: typeof RequestEmailChangeResponseSchema;
    };
    /**
     * ConfirmEmailChange consumes an email-change token and applies the
     * pending address. The previous address receives a notification with a
     * revert link (valid 72 h) that goes through this same RPC.
     *
     * @generated from rpc moth.auth.v1.AuthService.ConfirmEmailChange
     */
    confirmEmailChange: {
        methodKind: "unary";
        input: typeof ConfirmEmailChangeRequestSchema;
        output: typeof ConfirmEmailChangeResponseSchema;
    };
    /**
     * SignInWithOAuth signs in (or up) with a provider ID token obtained by a
     * native Google/Apple flow on the device. The token is verified
     * server-side (signature against the provider JWKS, issuer, audience
     * against the project's configured client/bundle IDs, expiry, nonce);
     * email, name and subject only ever come from the verified token. Account
     * resolution: an existing (provider, subject) identity signs that user
     * in; else a provider-verified email matching an existing user links a
     * new identity to it (when the project's auto_link_verified_email policy
     * allows); else a new user is created.
     *
     * @generated from rpc moth.auth.v1.AuthService.SignInWithOAuth
     */
    signInWithOAuth: {
        methodKind: "unary";
        input: typeof SignInWithOAuthRequestSchema;
        output: typeof SignInWithOAuthResponseSchema;
    };
    /**
     * ExchangeOAuthCode trades the one-time code minted by the web-redirect
     * fallback flow (GET /oauth/{provider}/start → provider consent →
     * callback → redirect back into the app) for a token pair. Codes are
     * single-use and short-lived.
     *
     * @generated from rpc moth.auth.v1.AuthService.ExchangeOAuthCode
     */
    exchangeOAuthCode: {
        methodKind: "unary";
        input: typeof ExchangeOAuthCodeRequestSchema;
        output: typeof ExchangeOAuthCodeResponseSchema;
    };
    /**
     * UnlinkIdentity removes the caller's identity for one provider. Requires
     * a Bearer access token. Refused when it would leave the account without
     * any way to sign in.
     *
     * @generated from rpc moth.auth.v1.AuthService.UnlinkIdentity
     */
    unlinkIdentity: {
        methodKind: "unary";
        input: typeof UnlinkIdentityRequestSchema;
        output: typeof UnlinkIdentityResponseSchema;
    };
    /**
     * DeleteAccount permanently deletes the user after fresh re-authentication
     * (App Store guideline 5.1.1). Identities, sessions and email tokens are
     * cascaded.
     *
     * @generated from rpc moth.auth.v1.AuthService.DeleteAccount
     */
    deleteAccount: {
        methodKind: "unary";
        input: typeof DeleteAccountRequestSchema;
        output: typeof DeleteAccountResponseSchema;
    };
}>;
