import { Code, ConnectError } from '@connectrpc/connect';
/**
 * The `google.rpc.ErrorInfo` domain moth uses
 * (internal/server/rpc/auth/errors.go).
 */
export declare const mothErrorDomain = "moth.dev";
/**
 * Base class for every error surfaced by the moth SDK.
 *
 * The server attaches a stable machine-readable reason to its errors as a
 * `google.rpc.ErrorInfo` detail (domain `moth.dev`); the SDK maps each known
 * reason to a subclass so callers catch types instead of matching strings.
 * Unknown reasons and non-moth failures surface as the base class, with
 * {@link reason} still populated when one was present.
 */
export declare class MothError extends Error {
    /** The `ErrorInfo` reason, e.g. `INVALID_CREDENTIALS`, or undefined. */
    readonly reason?: string;
    /** The gRPC status code, when the failure came from a Connect error. */
    readonly code?: Code;
    constructor(message: string, options?: {
        reason?: string;
        code?: Code;
    });
}
/** Wrong email/password combination (`INVALID_CREDENTIALS`). */
export declare class MothInvalidCredentialsError extends MothError {
}
/** The project requires a verified email before sign-in (`EMAIL_NOT_VERIFIED`). */
export declare class MothEmailNotVerifiedError extends MothError {
}
/** The email is already registered (`EMAIL_ALREADY_EXISTS`). */
export declare class MothEmailAlreadyExistsError extends MothError {
}
/** Public sign-up is closed for this project (`SIGNUP_CLOSED`). */
export declare class MothSignUpClosedError extends MothError {
}
/** The password does not meet the project's policy (`WEAK_PASSWORD`). */
export declare class MothWeakPasswordError extends MothError {
}
/** The email address is malformed (`INVALID_EMAIL`). */
export declare class MothInvalidEmailError extends MothError {
}
/** An email verification / reset / change token is invalid or expired (`INVALID_TOKEN`). */
export declare class MothInvalidTokenError extends MothError {
}
/** The refresh token is unknown, revoked or expired (`INVALID_REFRESH_TOKEN`). The SDK clears the session. */
export declare class MothInvalidRefreshTokenError extends MothError {
}
/** An already-rotated refresh token was presented; the family is revoked (`REFRESH_TOKEN_REUSED`). The SDK clears the session. */
export declare class MothRefreshTokenReusedError extends MothError {
}
/** The access token is missing, malformed or expired (`INVALID_ACCESS_TOKEN`). */
export declare class MothInvalidAccessTokenError extends MothError {
}
/** The account is disabled (`USER_DISABLED`). */
export declare class MothUserDisabledError extends MothError {
}
/** Too many attempts; retry later (`RATE_LIMITED`). */
export declare class MothRateLimitedError extends MothError {
}
/** The social provider is not enabled for this project (`PROVIDER_DISABLED`). */
export declare class MothProviderDisabledError extends MothError {
}
/** The provider ID token failed verification (`INVALID_PROVIDER_TOKEN`). */
export declare class MothInvalidProviderTokenError extends MothError {
}
/** The one-time OAuth exchange code is invalid, expired or used (`INVALID_OAUTH_CODE`). */
export declare class MothInvalidOAuthCodeError extends MothError {
}
/** The OAuth redirect URI is not allowed (`INVALID_REDIRECT`). */
export declare class MothInvalidRedirectError extends MothError {
}
/** Unlinking this identity would leave no way to sign in (`LAST_LOGIN_METHOD`). */
export declare class MothLastLoginMethodError extends MothError {
}
/** The email's domain is not allowed to sign up (`EMAIL_DOMAIN_NOT_ALLOWED`). */
export declare class MothEmailDomainNotAllowedError extends MothError {
}
/** The project has no store credentials configured (`BILLING_NOT_CONFIGURED`). */
export declare class MothBillingNotConfiguredError extends MothError {
}
/** The purchase receipt was rejected or malformed (`INVALID_RECEIPT`). */
export declare class MothInvalidReceiptError extends MothError {
}
/** The store could not be reached; retrying may succeed (`STORE_UNAVAILABLE`). */
export declare class MothStoreUnavailableError extends MothError {
}
/** The tier does not ship on the requested store (`PRODUCT_NOT_ON_STORE`). */
export declare class MothProductNotOnStoreError extends MothError {
}
/** The user has no purchase history on the store (`NO_BILLING_HISTORY`). */
export declare class MothNoBillingHistoryError extends MothError {
}
/**
 * The server could not be reached (connection, timeout, transport failure).
 * The session is kept — retrying may succeed.
 */
export declare class MothNetworkError extends MothError {
}
/**
 * Maps a transport error to the typed {@link MothError} hierarchy using the
 * `google.rpc.ErrorInfo` reason (domain `moth.dev`) the server attaches.
 * Unknown reasons map to the base class — new server reasons never break the
 * SDK. Reason-less Unavailable / DeadlineExceeded / Aborted failures become
 * {@link MothNetworkError}. Non-Connect errors are wrapped as
 * {@link MothNetworkError} too (fetch-level failures).
 */
export declare function mapConnectError(error: unknown): MothError;
/**
 * Extracts the moth `ErrorInfo` reason from a Connect error's details, or
 * undefined. Handles both wire-decoded details (`{type, value}` with raw
 * bytes) and in-process details (`{desc, value}` pairs, e.g. from
 * `createRouterTransport`), so no generated ErrorInfo schema is needed.
 */
export declare function mothReasonOf(error: ConnectError): string | undefined;
/** Decodes the `reason`/`domain` fields of a serialized google.rpc.ErrorInfo. */
export declare function decodeErrorInfo(bytes: Uint8Array): {
    reason?: string;
    domain?: string;
};
/** Encodes a google.rpc.ErrorInfo `{reason, domain}` (used by tests/fakes). */
export declare function encodeErrorInfo(reason: string, domain: string): Uint8Array;
