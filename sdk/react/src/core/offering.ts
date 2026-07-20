import {
  PaywallLayout,
  type Offering,
  type OfferingProduct,
  type Paywall,
} from '../gen/moth/billing/v1/billing_pb.js'

/**
 * One purchasable tier as the paywall needs it: enough to render a card and
 * drive a Stripe Checkout purchase. Price/period are display + analytics
 * metadata. The app never gates on {@link identifier} — it gates on
 * {@link entitlements} — but the SDK uses it to drive purchases.
 */
export interface MothOfferingProduct {
  /** Stable moth catalog identifier (e.g. `monthly`). */
  identifier: string
  displayName: string
  /** Store SKUs; either may be empty when the tier ships on one store only. */
  appleProductId: string
  googleProductId: string
  /**
   * The Stripe Price backing this tier on the web. Empty means the tier is
   * not purchasable on the web — the paywall renders it unavailable rather
   * than hiding it.
   */
  stripePriceId: string
  /** ISO-8601 period descriptor (e.g. `P1M`); empty when unset. */
  billingPeriod: string
  /** Price in micros of {@link currency} (1,000,000 = one unit). */
  priceAmountMicros: number
  currency: string
  /** Trial/intro descriptors (display + analytics only). */
  trialPeriod: string
  introPriceAmountMicros: number
  introPeriod: string
  /** The stable entitlement identifiers this product grants while active. */
  entitlements: readonly string[]
  sortOrder: number
  /** Whether this tier is the paywall's highlighted "most popular" tier. */
  highlighted: boolean
}

/** Whether this product offers a free trial. */
export function productHasTrial(product: MothOfferingProduct): boolean {
  return product.trialPeriod !== ''
}

export function offeringProductFromProto(
  proto: OfferingProduct,
): MothOfferingProduct {
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
  }
}

/**
 * The ordered set of products a paywall presents — the products sharing an
 * `offering` tag, in sort order. Every project has a default offering.
 */
export class MothOffering {
  /** Offering tag; `default` for the project's default offering. */
  readonly identifier: string
  readonly isDefault: boolean
  /** The products to display, in paywall order. */
  readonly products: readonly MothOfferingProduct[]

  constructor(
    identifier: string,
    isDefault = false,
    products: readonly MothOfferingProduct[] = [],
  ) {
    this.identifier = identifier
    this.isDefault = isDefault
    this.products = products
  }

  static fromProto(proto: Offering): MothOffering {
    return new MothOffering(
      proto.identifier,
      proto.isDefault,
      proto.products.map(offeringProductFromProto),
    )
  }

  /** True when there is nothing to sell. */
  get isEmpty(): boolean {
    return this.products.length === 0
  }

  /** The product with `identifier`, or undefined. */
  productById(identifier: string): MothOfferingProduct | undefined {
    return this.products.find((p) => p.identifier === identifier)
  }

  /** Whether any product in this offering grants `entitlement`. */
  grants(entitlement: string): boolean {
    return this.products.some((p) => p.entitlements.includes(entitlement))
  }
}

/**
 * The rendering variant the paywall screen uses; the token space
 * (colors/spacing/radius) always comes from the theme.
 */
export type MothPaywallLayout = 'tiles' | 'list' | 'compact'

/**
 * The public, render-ready paywall configuration, from
 * `moth.billing.v1.GetPaywall`. Copy and layout only — colors/typography
 * inherit from the theme.
 */
export interface MothPaywall {
  /**
   * Identifies this version of the paywall config; changes on every admin
   * edit. Cached keyed by this value and echoed as `knownPaywallRevision`.
   */
  revisionId: string
  headline: string
  subtitle: string
  /** Feature/benefit bullets, in display order. */
  benefits: readonly string[]
  /** The offering tag whose products this paywall lists; empty = default. */
  offering: string
  /** The product identifier to render as "most popular"; empty for none. */
  highlightedProductIdentifier: string
  layout: MothPaywallLayout
  /** Optional legal links rendered in the paywall footer. */
  termsUrl?: string
  privacyUrl?: string
}

export const emptyPaywall: MothPaywall = {
  revisionId: '',
  headline: '',
  subtitle: '',
  benefits: [],
  offering: '',
  highlightedProductIdentifier: '',
  layout: 'tiles',
}

export function paywallFromProto(proto: Paywall): MothPaywall {
  const paywall: MothPaywall = {
    revisionId: proto.revisionId,
    headline: proto.headline,
    subtitle: proto.subtitle,
    benefits: [...proto.benefits],
    offering: proto.offering,
    highlightedProductIdentifier: proto.highlightedProductIdentifier,
    layout: layoutFromProto(proto.layout),
  }
  if (proto.termsUrl !== '') paywall.termsUrl = proto.termsUrl
  if (proto.privacyUrl !== '') paywall.privacyUrl = proto.privacyUrl
  return paywall
}

function layoutFromProto(proto: PaywallLayout): MothPaywallLayout {
  switch (proto) {
    case PaywallLayout.LIST:
      return 'list'
    case PaywallLayout.COMPACT:
      return 'compact'
    default:
      return 'tiles'
  }
}
