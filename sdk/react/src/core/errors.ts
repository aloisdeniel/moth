import { Code, ConnectError } from '@connectrpc/connect'

/**
 * The `google.rpc.ErrorInfo` domain moth uses
 * (internal/server/rpc/auth/errors.go).
 */
export const mothErrorDomain = 'moth.dev'

/**
 * Base class for every error surfaced by the moth SDK.
 *
 * The server attaches a stable machine-readable reason to its errors as a
 * `google.rpc.ErrorInfo` detail (domain `moth.dev`); the SDK maps each known
 * reason to a subclass so callers catch types instead of matching strings.
 * Unknown reasons and non-moth failures surface as the base class, with
 * {@link reason} still populated when one was present.
 */
export class MothError extends Error {
  /** The `ErrorInfo` reason, e.g. `INVALID_CREDENTIALS`, or undefined. */
  readonly reason?: string

  /** The gRPC status code, when the failure came from a Connect error. */
  readonly code?: Code

  constructor(message: string, options?: { reason?: string; code?: Code }) {
    super(message)
    this.name = new.target.name
    if (options?.reason !== undefined) this.reason = options.reason
    if (options?.code !== undefined) this.code = options.code
  }
}

/** Wrong email/password combination (`INVALID_CREDENTIALS`). */
export class MothInvalidCredentialsError extends MothError {}
/** The project requires a verified email before sign-in (`EMAIL_NOT_VERIFIED`). */
export class MothEmailNotVerifiedError extends MothError {}
/** The email is already registered (`EMAIL_ALREADY_EXISTS`). */
export class MothEmailAlreadyExistsError extends MothError {}
/** Public sign-up is closed for this project (`SIGNUP_CLOSED`). */
export class MothSignUpClosedError extends MothError {}
/** The password does not meet the project's policy (`WEAK_PASSWORD`). */
export class MothWeakPasswordError extends MothError {}
/** The email address is malformed (`INVALID_EMAIL`). */
export class MothInvalidEmailError extends MothError {}
/** An email verification / reset / change token is invalid or expired (`INVALID_TOKEN`). */
export class MothInvalidTokenError extends MothError {}
/** The refresh token is unknown, revoked or expired (`INVALID_REFRESH_TOKEN`). The SDK clears the session. */
export class MothInvalidRefreshTokenError extends MothError {}
/** An already-rotated refresh token was presented; the family is revoked (`REFRESH_TOKEN_REUSED`). The SDK clears the session. */
export class MothRefreshTokenReusedError extends MothError {}
/** The access token is missing, malformed or expired (`INVALID_ACCESS_TOKEN`). */
export class MothInvalidAccessTokenError extends MothError {}
/** The account is disabled (`USER_DISABLED`). */
export class MothUserDisabledError extends MothError {}
/** Too many attempts; retry later (`RATE_LIMITED`). */
export class MothRateLimitedError extends MothError {}
/** The social provider is not enabled for this project (`PROVIDER_DISABLED`). */
export class MothProviderDisabledError extends MothError {}
/** The provider ID token failed verification (`INVALID_PROVIDER_TOKEN`). */
export class MothInvalidProviderTokenError extends MothError {}
/** The one-time OAuth exchange code is invalid, expired or used (`INVALID_OAUTH_CODE`). */
export class MothInvalidOAuthCodeError extends MothError {}
/** The OAuth redirect URI is not allowed (`INVALID_REDIRECT`). */
export class MothInvalidRedirectError extends MothError {}
/** Unlinking this identity would leave no way to sign in (`LAST_LOGIN_METHOD`). */
export class MothLastLoginMethodError extends MothError {}
/** The email's domain is not allowed to sign up (`EMAIL_DOMAIN_NOT_ALLOWED`). */
export class MothEmailDomainNotAllowedError extends MothError {}
/** The project has no store credentials configured (`BILLING_NOT_CONFIGURED`). */
export class MothBillingNotConfiguredError extends MothError {}
/** The purchase receipt was rejected or malformed (`INVALID_RECEIPT`). */
export class MothInvalidReceiptError extends MothError {}
/** The store could not be reached; retrying may succeed (`STORE_UNAVAILABLE`). */
export class MothStoreUnavailableError extends MothError {}
/** The tier does not ship on the requested store (`PRODUCT_NOT_ON_STORE`). */
export class MothProductNotOnStoreError extends MothError {}
/** The user has no purchase history on the store (`NO_BILLING_HISTORY`). */
export class MothNoBillingHistoryError extends MothError {}

/**
 * The server could not be reached (connection, timeout, transport failure).
 * The session is kept — retrying may succeed.
 */
export class MothNetworkError extends MothError {}

type MothErrorClass = new (
  message: string,
  options?: { reason?: string; code?: Code },
) => MothError

const byReason: Record<string, MothErrorClass> = {
  INVALID_CREDENTIALS: MothInvalidCredentialsError,
  EMAIL_NOT_VERIFIED: MothEmailNotVerifiedError,
  EMAIL_ALREADY_EXISTS: MothEmailAlreadyExistsError,
  SIGNUP_CLOSED: MothSignUpClosedError,
  WEAK_PASSWORD: MothWeakPasswordError,
  INVALID_EMAIL: MothInvalidEmailError,
  INVALID_TOKEN: MothInvalidTokenError,
  INVALID_REFRESH_TOKEN: MothInvalidRefreshTokenError,
  REFRESH_TOKEN_REUSED: MothRefreshTokenReusedError,
  INVALID_ACCESS_TOKEN: MothInvalidAccessTokenError,
  USER_DISABLED: MothUserDisabledError,
  RATE_LIMITED: MothRateLimitedError,
  PROVIDER_DISABLED: MothProviderDisabledError,
  INVALID_PROVIDER_TOKEN: MothInvalidProviderTokenError,
  INVALID_OAUTH_CODE: MothInvalidOAuthCodeError,
  INVALID_REDIRECT: MothInvalidRedirectError,
  LAST_LOGIN_METHOD: MothLastLoginMethodError,
  EMAIL_DOMAIN_NOT_ALLOWED: MothEmailDomainNotAllowedError,
  BILLING_NOT_CONFIGURED: MothBillingNotConfiguredError,
  INVALID_RECEIPT: MothInvalidReceiptError,
  STORE_UNAVAILABLE: MothStoreUnavailableError,
  PRODUCT_NOT_ON_STORE: MothProductNotOnStoreError,
  NO_BILLING_HISTORY: MothNoBillingHistoryError,
}

/**
 * Maps a transport error to the typed {@link MothError} hierarchy using the
 * `google.rpc.ErrorInfo` reason (domain `moth.dev`) the server attaches.
 * Unknown reasons map to the base class — new server reasons never break the
 * SDK. Reason-less Unavailable / DeadlineExceeded / Aborted failures become
 * {@link MothNetworkError}. Non-Connect errors are wrapped as
 * {@link MothNetworkError} too (fetch-level failures).
 */
export function mapConnectError(error: unknown): MothError {
  if (error instanceof MothError) return error
  if (!(error instanceof ConnectError)) {
    const message = error instanceof Error ? error.message : String(error)
    return new MothNetworkError(message)
  }
  const message = error.rawMessage || 'request failed'
  const reason = mothReasonOf(error)
  if (reason !== undefined) {
    const cls = byReason[reason] ?? MothError
    return new cls(message, { reason, code: error.code })
  }
  switch (error.code) {
    case Code.Unavailable:
    case Code.DeadlineExceeded:
    case Code.Aborted:
      return new MothNetworkError(message, { code: error.code })
    default:
      return new MothError(message, { code: error.code })
  }
}

/**
 * Extracts the moth `ErrorInfo` reason from a Connect error's details, or
 * undefined. Handles both wire-decoded details (`{type, value}` with raw
 * bytes) and in-process details (`{desc, value}` pairs, e.g. from
 * `createRouterTransport`), so no generated ErrorInfo schema is needed.
 */
export function mothReasonOf(error: ConnectError): string | undefined {
  for (const detail of error.details) {
    if ('type' in detail && typeof detail.type === 'string') {
      // Incoming detail: type name + raw wire bytes.
      if (detail.type !== 'google.rpc.ErrorInfo') continue
      const info = decodeErrorInfo(detail.value)
      if (info.domain === mothErrorDomain && info.reason) return info.reason
    } else if ('desc' in detail) {
      // Outgoing detail (in-process transports): schema + init shape.
      if (detail.desc.typeName !== 'google.rpc.ErrorInfo') continue
      const value = detail.value as { reason?: string; domain?: string }
      if (value.domain === mothErrorDomain && value.reason) return value.reason
    }
  }
  return undefined
}

// google.rpc.ErrorInfo wire format: field 1 = reason (string), field 2 =
// domain (string). A tiny hand-rolled reader keeps the SDK free of a
// generated googleapis dependency (mirroring the Dart SDK's structural
// matching).

/** Decodes the `reason`/`domain` fields of a serialized google.rpc.ErrorInfo. */
export function decodeErrorInfo(bytes: Uint8Array): {
  reason?: string
  domain?: string
} {
  const out: { reason?: string; domain?: string } = {}
  let i = 0
  const decoder = new TextDecoder()
  while (i < bytes.length) {
    let tag = 0
    let shift = 0
    for (;;) {
      const b = bytes[i]
      if (b === undefined) return out
      i++
      tag |= (b & 0x7f) << shift
      if ((b & 0x80) === 0) break
      shift += 7
    }
    const field = tag >>> 3
    const wireType = tag & 0x7
    if (wireType === 2) {
      let len = 0
      shift = 0
      for (;;) {
        const b = bytes[i]
        if (b === undefined) return out
        i++
        len |= (b & 0x7f) << shift
        if ((b & 0x80) === 0) break
        shift += 7
      }
      const value = bytes.subarray(i, i + len)
      i += len
      if (field === 1) out.reason = decoder.decode(value)
      else if (field === 2) out.domain = decoder.decode(value)
    } else if (wireType === 0) {
      for (;;) {
        const b = bytes[i]
        if (b === undefined) return out
        i++
        if ((b & 0x80) === 0) break
      }
    } else {
      return out // unknown wire type: stop rather than misparse
    }
  }
  return out
}

/** Encodes a google.rpc.ErrorInfo `{reason, domain}` (used by tests/fakes). */
export function encodeErrorInfo(reason: string, domain: string): Uint8Array {
  const encoder = new TextEncoder()
  const parts: number[] = []
  const put = (field: number, value: string) => {
    const bytes = encoder.encode(value)
    parts.push((field << 3) | 2)
    let len = bytes.length
    for (;;) {
      if (len < 0x80) {
        parts.push(len)
        break
      }
      parts.push((len & 0x7f) | 0x80)
      len >>>= 7
    }
    for (const b of bytes) parts.push(b)
  }
  put(1, reason)
  put(2, domain)
  return new Uint8Array(parts)
}
