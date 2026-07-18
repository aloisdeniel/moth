// @moth/react — auth, entitlements and a themed login/paywall for apps
// backed by a moth instance. The `core/` layer is framework-free (a future
// Vue/Svelte binding reuses it unchanged); the React layer is thin bindings.

export { mothSdkVersion } from './version.js'

// Core client.
export {
  MothClient,
  type MothClientOptions,
  type MothOAuthProvider,
  type MothSignUpResult,
} from './core/client.js'
export {
  defaultConfigCacheTtlMs,
  type MothConfig,
} from './core/config.js'
export {
  mothAuthLoading,
  mothSignedOut,
  type MothAuthState,
  type MothUser,
} from './core/user.js'
export {
  InMemoryTokenStore,
  WebStorageTokenStore,
  type StoredSession,
  type TokenStore,
} from './core/tokenStore.js'
export { createMothFetch } from './core/fetch.js'
export { customClaimsOf, decodeJwtPayload } from './core/jwt.js'

// Billing models.
export {
  MothCustomerInfo,
  subscriptionStatusIsActive,
  type MothActiveSubscription,
  type MothEntitlement,
  type MothEntitlementSource,
  type MothStore,
  type MothSubscriptionStatus,
} from './core/customerInfo.js'
export {
  MothOffering,
  productHasTrial,
  type MothOfferingProduct,
  type MothPaywall,
  type MothPaywallLayout,
} from './core/offering.js'
export { checkoutReturnParam, type MothPurchaseResult } from './core/purchase.js'

// Push.
export type {
  MothPushDeviceMetadata,
  MothPushPermission,
  MothPushStatus,
  MothPushTarget,
} from './core/push.js'
export { MothPushController } from './core/pushController.js'

// Config, theme and copy.
export type {
  MothAppleConfig,
  MothGoogleConfig,
  MothProjectConfig,
  MothPushConfig,
} from './core/projectConfig.js'
export {
  contrastRatio,
  deriveDarkColors,
  fallbackTheme,
  themeCssVars,
  themeFromProto,
  type MothTheme,
  type MothThemeColors,
} from './core/theme.js'
export { MothCopy, type MothCopyUpdate } from './core/copy.js'
export { bundledCopy, mothBundledLocales } from './core/i18n/bundledCopy.js'
export { MothConfigController } from './core/configController.js'
export { MothSubscriptionController } from './core/subscriptionController.js'
export { loadPaywall } from './core/paywallLoader.js'

// Errors.
export {
  mapConnectError,
  mothErrorDomain,
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
} from './core/errors.js'

// React layer.
export {
  MothProvider,
  MothSurface,
  useMothContext,
  useMothCopy,
  useMothTheme,
  type MothContextValue,
  type MothProviderProps,
} from './react/context.js'
export {
  useMoth,
  useMothCustomerInfo,
  useMothEntitlement,
  useMothPush,
  useMothUser,
  type UseMothEntitlementResult,
  type UseMothPushResult,
  type UseMothResult,
} from './react/hooks.js'
export { MothGate, type MothGateProps } from './react/MothGate.js'
export {
  MothLoginScreen,
  type MothLoginScreenProps,
} from './react/MothLoginScreen.js'
export {
  MothPaywallHeader,
  MothPaywallScreen,
  MothPurchaseButton,
  MothTierCard,
  priceLabel,
  type MothPaywallScreenProps,
} from './react/MothPaywallScreen.js'
export { friendlyMothErrorMessage } from './react/friendlyErrors.js'
