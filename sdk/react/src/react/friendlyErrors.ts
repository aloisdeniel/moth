import type { MothCopy } from '../core/copy.js'
import {
  MothEmailAlreadyExistsError,
  MothEmailNotVerifiedError,
  MothError,
  MothInvalidAccessTokenError,
  MothInvalidCredentialsError,
  MothInvalidEmailError,
  MothInvalidProviderTokenError,
  MothLastLoginMethodError,
  MothNetworkError,
  MothProviderDisabledError,
  MothRateLimitedError,
  MothSignUpClosedError,
  MothUserDisabledError,
  MothWeakPasswordError,
} from '../core/errors.js'

/**
 * End-user copy for a {@link MothError} (or any error), used by the SDK's
 * built-in screens. Exposed so custom login UIs can reuse the same wording.
 *
 * Every mapped error resolves from the localized catalog — wrong
 * credentials and an already-registered email share the login-form keys,
 * the rest use the shared `error.*` group. The two cases that echo a
 * server-supplied message (a weak-password policy and the generic
 * MothError) are already localized by the server.
 */
export function friendlyMothErrorMessage(error: unknown, copy: MothCopy): string {
  if (error instanceof MothInvalidCredentialsError) {
    return copy.value('sign_in.error_invalid')
  }
  if (error instanceof MothEmailNotVerifiedError) {
    return copy.value('error.email_not_verified')
  }
  if (error instanceof MothEmailAlreadyExistsError) {
    return copy.value('sign_up.error_email_taken')
  }
  if (error instanceof MothSignUpClosedError) {
    return copy.value('error.signup_closed')
  }
  // The server message spells out the policy that was not met.
  if (error instanceof MothWeakPasswordError) return error.message
  if (error instanceof MothInvalidEmailError) {
    return copy.value('error.invalid_email')
  }
  if (error instanceof MothInvalidAccessTokenError) {
    return copy.value('error.session_expired')
  }
  if (error instanceof MothUserDisabledError) {
    return copy.value('error.user_disabled')
  }
  if (error instanceof MothRateLimitedError) {
    return copy.value('error.rate_limited')
  }
  if (error instanceof MothProviderDisabledError) {
    return copy.value('error.provider_disabled')
  }
  if (error instanceof MothInvalidProviderTokenError) {
    return copy.value('error.provider_failed')
  }
  if (error instanceof MothLastLoginMethodError) {
    return copy.value('error.last_login_method')
  }
  if (error instanceof MothNetworkError) {
    return copy.value('error.network')
  }
  if (error instanceof MothError) return error.message
  return copy.value('error.generic')
}
