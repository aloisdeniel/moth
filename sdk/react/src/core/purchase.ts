/**
 * The typed outcome of `MothClient.purchase(product)` /
 * `handleCheckoutReturn()`. Expected outcomes are values, never exceptions —
 * mirroring the Flutter SDK's `MothPurchaseResult`, adapted to the
 * redirect-based web flow:
 *
 * - `redirect`: the browser is navigating to Stripe-hosted Checkout; the
 *   page unloads and the outcome arrives on return.
 * - `purchased`: entitlements are updated (after a checkout return).
 * - `pending`: checkout succeeded but the webhook has not landed yet; the
 *   entitlement will flip as soon as it does (subscribers re-render).
 * - `alreadyOwned`: the user already owns this product.
 * - `cancelled`: the user backed out of checkout.
 * - `error`: the purchase failed; `reason` carries the server `ErrorInfo`
 *   reason when the failure was server-side.
 */
export type MothPurchaseResult =
  | { status: 'redirect' }
  | { status: 'purchased' }
  | { status: 'pending' }
  | { status: 'alreadyOwned' }
  | { status: 'cancelled' }
  | { status: 'error'; message: string; reason?: string }

/**
 * The query parameter the SDK appends to the checkout success / cancel URLs
 * so `handleCheckoutReturn()` can recognize the return navigation.
 */
export const checkoutReturnParam = 'moth_checkout'
