import { Code, ConnectError } from '@connectrpc/connect'
import { describe, expect, it } from 'vitest'
import { mothConnectError } from '../test/fake.js'
import {
  decodeErrorInfo,
  encodeErrorInfo,
  mapConnectError,
  MothBillingNotConfiguredError,
  MothEmailAlreadyExistsError,
  MothEmailDomainNotAllowedError,
  MothEmailNotVerifiedError,
  MothError,
  MothInvalidAccessTokenError,
  MothInvalidCredentialsError,
  MothInvalidEmailError,
  MothInvalidOAuthCodeError,
  MothInvalidProviderTokenError,
  MothInvalidReceiptError,
  MothInvalidRedirectError,
  MothInvalidRefreshTokenError,
  MothInvalidTokenError,
  MothLastLoginMethodError,
  MothNetworkError,
  MothNoBillingHistoryError,
  MothProductNotOnStoreError,
  MothProviderDisabledError,
  MothRateLimitedError,
  MothRefreshTokenReusedError,
  MothSignUpClosedError,
  MothStoreUnavailableError,
  MothUserDisabledError,
  MothWeakPasswordError,
} from './errors.js'

describe('error mapping table', () => {
  const table: [string, new (m: string) => MothError][] = [
    ['INVALID_CREDENTIALS', MothInvalidCredentialsError],
    ['EMAIL_NOT_VERIFIED', MothEmailNotVerifiedError],
    ['EMAIL_ALREADY_EXISTS', MothEmailAlreadyExistsError],
    ['SIGNUP_CLOSED', MothSignUpClosedError],
    ['WEAK_PASSWORD', MothWeakPasswordError],
    ['INVALID_EMAIL', MothInvalidEmailError],
    ['INVALID_TOKEN', MothInvalidTokenError],
    ['INVALID_REFRESH_TOKEN', MothInvalidRefreshTokenError],
    ['REFRESH_TOKEN_REUSED', MothRefreshTokenReusedError],
    ['INVALID_ACCESS_TOKEN', MothInvalidAccessTokenError],
    ['USER_DISABLED', MothUserDisabledError],
    ['RATE_LIMITED', MothRateLimitedError],
    ['PROVIDER_DISABLED', MothProviderDisabledError],
    ['INVALID_PROVIDER_TOKEN', MothInvalidProviderTokenError],
    ['INVALID_OAUTH_CODE', MothInvalidOAuthCodeError],
    ['INVALID_REDIRECT', MothInvalidRedirectError],
    ['LAST_LOGIN_METHOD', MothLastLoginMethodError],
    ['EMAIL_DOMAIN_NOT_ALLOWED', MothEmailDomainNotAllowedError],
    ['BILLING_NOT_CONFIGURED', MothBillingNotConfiguredError],
    ['INVALID_RECEIPT', MothInvalidReceiptError],
    ['STORE_UNAVAILABLE', MothStoreUnavailableError],
    ['PRODUCT_NOT_ON_STORE', MothProductNotOnStoreError],
    ['NO_BILLING_HISTORY', MothNoBillingHistoryError],
  ]

  it.each(table)('%s maps to its class', (reason, cls) => {
    const mapped = mapConnectError(
      mothConnectError(Code.InvalidArgument, reason, 'boom'),
    )
    expect(mapped).toBeInstanceOf(cls)
    expect(mapped.reason).toBe(reason)
    expect(mapped.message).toBe('boom')
  })

  it('maps an unknown reason to the base MothError, reason preserved', () => {
    const mapped = mapConnectError(
      mothConnectError(Code.FailedPrecondition, 'BRAND_NEW_REASON'),
    )
    expect(mapped.constructor).toBe(MothError)
    expect(mapped.reason).toBe('BRAND_NEW_REASON')
  })

  it('ignores ErrorInfo details from foreign domains', () => {
    const err = new ConnectError('other', Code.InvalidArgument)
    err.details.push({
      type: 'google.rpc.ErrorInfo',
      value: encodeErrorInfo('SOME_REASON', 'example.com'),
    })
    const mapped = mapConnectError(err)
    expect(mapped.constructor).toBe(MothError)
    expect(mapped.reason).toBeUndefined()
  })

  it.each([Code.Unavailable, Code.DeadlineExceeded, Code.Aborted])(
    'reason-less code %s becomes MothNetworkError',
    (code) => {
      const mapped = mapConnectError(new ConnectError('down', code))
      expect(mapped).toBeInstanceOf(MothNetworkError)
    },
  )

  it('a reason-less Internal error stays the base class with its code', () => {
    const mapped = mapConnectError(new ConnectError('boom', Code.Internal))
    expect(mapped.constructor).toBe(MothError)
    expect(mapped.code).toBe(Code.Internal)
  })

  it('wraps non-Connect failures as MothNetworkError', () => {
    const mapped = mapConnectError(new TypeError('fetch failed'))
    expect(mapped).toBeInstanceOf(MothNetworkError)
    expect(mapped.message).toBe('fetch failed')
  })

  it('passes through an already-mapped MothError', () => {
    const original = new MothInvalidCredentialsError('x')
    expect(mapConnectError(original)).toBe(original)
  })
})

describe('ErrorInfo wire codec', () => {
  it('round-trips reason and domain', () => {
    const bytes = encodeErrorInfo('INVALID_CREDENTIALS', 'moth.dev')
    expect(decodeErrorInfo(bytes)).toEqual({
      reason: 'INVALID_CREDENTIALS',
      domain: 'moth.dev',
    })
  })

  it('tolerates unknown fields and garbage', () => {
    expect(decodeErrorInfo(new Uint8Array([0xff, 0x01, 0x02]))).toEqual({})
    // Field 3 (metadata) present alongside reason.
    const withExtra = new Uint8Array([
      ...encodeErrorInfo('R', 'moth.dev'),
      0x1a,
      0x01,
      0x00,
    ])
    expect(decodeErrorInfo(withExtra).reason).toBe('R')
  })
})
