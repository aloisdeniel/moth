import { PaywallLayout, } from '../gen/moth/billing/v1/billing_pb.js';
/** Whether this product offers a free trial. */
export function productHasTrial(product) {
    return product.trialPeriod !== '';
}
export function offeringProductFromProto(proto) {
    return {
        identifier: proto.identifier,
        displayName: proto.displayName,
        appleProductId: proto.appleProductId,
        googleProductId: proto.googleProductId,
        stripePriceId: proto.stripePriceId,
        billingPeriod: proto.billingPeriod,
        priceAmountMicros: Number(proto.priceAmountMicros),
        currency: proto.currency,
        trialPeriod: proto.trialPeriod,
        introPriceAmountMicros: Number(proto.introPriceAmountMicros),
        introPeriod: proto.introPeriod,
        entitlements: [...proto.entitlements],
        sortOrder: proto.sortOrder,
        highlighted: proto.highlighted,
    };
}
/**
 * The ordered set of products a paywall presents — the products sharing an
 * `offering` tag, in sort order. Every project has a default offering.
 */
export class MothOffering {
    constructor(identifier, isDefault = false, products = []) {
        this.identifier = identifier;
        this.isDefault = isDefault;
        this.products = products;
    }
    static fromProto(proto) {
        return new MothOffering(proto.identifier, proto.isDefault, proto.products.map(offeringProductFromProto));
    }
    /** True when there is nothing to sell. */
    get isEmpty() {
        return this.products.length === 0;
    }
    /** The product with `identifier`, or undefined. */
    productById(identifier) {
        return this.products.find((p) => p.identifier === identifier);
    }
    /** Whether any product in this offering grants `entitlement`. */
    grants(entitlement) {
        return this.products.some((p) => p.entitlements.includes(entitlement));
    }
}
export const emptyPaywall = {
    revisionId: '',
    headline: '',
    subtitle: '',
    benefits: [],
    offering: '',
    highlightedProductIdentifier: '',
    layout: 'tiles',
};
export function paywallFromProto(proto) {
    const paywall = {
        revisionId: proto.revisionId,
        headline: proto.headline,
        subtitle: proto.subtitle,
        benefits: [...proto.benefits],
        offering: proto.offering,
        highlightedProductIdentifier: proto.highlightedProductIdentifier,
        layout: layoutFromProto(proto.layout),
    };
    if (proto.termsUrl !== '')
        paywall.termsUrl = proto.termsUrl;
    if (proto.privacyUrl !== '')
        paywall.privacyUrl = proto.privacyUrl;
    return paywall;
}
function layoutFromProto(proto) {
    switch (proto) {
        case PaywallLayout.LIST:
            return 'list';
        case PaywallLayout.COMPACT:
            return 'compact';
        default:
            return 'tiles';
    }
}
