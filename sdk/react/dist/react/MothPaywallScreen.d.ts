import type { MothCopy } from '../core/copy.js';
import { type MothOfferingProduct, type MothPaywall } from '../core/offering.js';
export interface MothPaywallScreenProps {
    /** Paywall copy/layout override; when unset it is fetched (revision-cached). */
    paywall?: MothPaywall;
    /** Called after a checkout redirect begins (e.g. to show a note). */
    onRedirect?: () => void;
    /** Called when the user dismisses the paywall; shows a close button when set. */
    onClose?: () => void;
}
/**
 * Batteries-included paywall screen — the purchasing counterpart to
 * {@link MothLoginScreen}, driven by the same admin-configured paywall
 * config as the mobile SDKs: header (logo + headline + subtitle), benefit
 * bullets, one selectable card per tier (price, trial badge, "most popular"
 * highlight), the purchase button (Stripe-hosted Checkout redirect — no
 * card fields, no Stripe.js), a manage-billing link when subscriptions
 * exist, and terms/privacy links. Tiers without a `stripe_price_id` render
 * as unavailable-on-web rather than disappearing silently.
 *
 * The building blocks ({@link MothPaywallHeader}, {@link MothTierCard},
 * {@link MothPurchaseButton}) are exported for custom paywalls.
 */
export declare function MothPaywallScreen(props: MothPaywallScreenProps): import("react").JSX.Element;
/**
 * The paywall header: logo (when the theme sets one), headline and
 * subtitle. Exported for custom paywalls.
 */
export declare function MothPaywallHeader(props: {
    headline: string;
    subtitle?: string;
    logoLightUrl?: string;
    logoDarkUrl?: string;
}): import("react").JSX.Element;
/**
 * One selectable subscription tier card: name, price/period, a trial badge
 * and the "most popular" highlight. Tiers without a Stripe price render
 * disabled with an unavailable-on-web note. Exported for custom paywalls.
 */
export declare function MothTierCard(props: {
    product: MothOfferingProduct;
    selected?: boolean;
    copy: MothCopy;
    onSelect?: () => void;
}): import("react").JSX.Element;
/**
 * The primary purchase button; label defaults to the localized
 * `paywall.cta`. Exported for custom paywalls.
 */
export declare function MothPurchaseButton(props: {
    product?: MothOfferingProduct | undefined;
    busy?: boolean;
    label?: string;
    onPress?: () => void;
}): import("react").JSX.Element;
/**
 * Formats a tier's price with the billing period suffix (e.g. `$9.99 /
 * month`) from the catalog micros + currency.
 */
export declare function priceLabel(product: MothOfferingProduct, copy: MothCopy): string;
