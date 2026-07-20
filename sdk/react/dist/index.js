// @moth/react — auth, entitlements and a themed login/paywall for apps
// backed by a moth instance. The `core/` layer is framework-free (a future
// Vue/Svelte binding reuses it unchanged); the React layer is thin bindings.
export { mothSdkVersion } from './version.js';
// Core client.
export { MothClient, } from './core/client.js';
export { defaultConfigCacheTtlMs, } from './core/config.js';
export { mothAuthLoading, mothSignedOut, } from './core/user.js';
export { InMemoryTokenStore, WebStorageTokenStore, } from './core/tokenStore.js';
export { createMothFetch } from './core/fetch.js';
export { customClaimsOf, decodeJwtPayload } from './core/jwt.js';
// Billing models.
export { MothCustomerInfo, subscriptionStatusIsActive, } from './core/customerInfo.js';
export { MothOffering, productHasTrial, } from './core/offering.js';
export { checkoutReturnParam } from './core/purchase.js';
export { MothPushController } from './core/pushController.js';
export { contrastRatio, deriveDarkColors, fallbackTheme, themeCssVars, themeFromProto, } from './core/theme.js';
export { MothCopy } from './core/copy.js';
export { bundledCopy, mothBundledLocales } from './core/i18n/bundledCopy.js';
export { MothConfigController } from './core/configController.js';
export { MothSubscriptionController } from './core/subscriptionController.js';
export { loadPaywall } from './core/paywallLoader.js';
// Errors.
export { mapConnectError, mothErrorDomain, MothBillingNotConfiguredError, MothEmailAlreadyExistsError, MothEmailDomainNotAllowedError, MothEmailNotVerifiedError, MothError, MothInvalidAccessTokenError, MothInvalidCredentialsError, MothInvalidEmailError, MothInvalidOAuthCodeError, MothInvalidProviderTokenError, MothInvalidReceiptError, MothInvalidRedirectError, MothInvalidRefreshTokenError, MothInvalidTokenError, MothLastLoginMethodError, MothNetworkError, MothNoBillingHistoryError, MothProductNotOnStoreError, MothProviderDisabledError, MothRateLimitedError, MothRefreshTokenReusedError, MothSignUpClosedError, MothStoreUnavailableError, MothUserDisabledError, MothWeakPasswordError, } from './core/errors.js';
// React layer.
export { MothProvider, MothSurface, useMothContext, useMothCopy, useMothTheme, } from './react/context.js';
export { useMoth, useMothCustomerInfo, useMothEntitlement, useMothPush, useMothUser, } from './react/hooks.js';
export { MothGate } from './react/MothGate.js';
export { MothLoginScreen, } from './react/MothLoginScreen.js';
export { MothPaywallHeader, MothPaywallScreen, MothPurchaseButton, MothTierCard, priceLabel, } from './react/MothPaywallScreen.js';
export { friendlyMothErrorMessage } from './react/friendlyErrors.js';
