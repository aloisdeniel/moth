import { type CustomerInfo } from '../gen/moth/billing/v1/billing_pb.js';
/** Which store a purchase or subscription belongs to. */
export type MothStore = 'apple' | 'google' | 'stripe';
/** Why an entitlement is active. */
export type MothEntitlementSource = 'store' | 'grant' | 'none';
/**
 * The store's renewal state, mapped to a small set common to every store.
 * active/trialing/inGracePeriod/inBillingRetry keep access;
 * paused/expired/revoked do not.
 */
export type MothSubscriptionStatus = 'unspecified' | 'active' | 'trialing' | 'inGracePeriod' | 'inBillingRetry' | 'paused' | 'expired' | 'revoked';
/** Whether `status` keeps the subscription's access. */
export declare function subscriptionStatusIsActive(status: MothSubscriptionStatus): boolean;
/**
 * One active capability the user holds (e.g. `pro`), with its expiry and why
 * it is active. Apps gate on {@link identifier}, never on a product id.
 */
export interface MothEntitlement {
    /** Stable identifier the app checks (e.g. `pro`). */
    identifier: string;
    /** When the entitlement lapses; undefined for a non-expiring grant. */
    expireTime?: Date;
    /** Why it is active (store subscription vs operator grant). */
    source: MothEntitlementSource;
    /** The moth product identifier that granted it; empty for grants. */
    productIdentifier: string;
}
/** One of the user's store subscriptions (may be inactive, for history). */
export interface MothActiveSubscription {
    productIdentifier: string;
    /** The store this subscription lives on; undefined when unspecified. */
    store?: MothStore;
    status: MothSubscriptionStatus;
    /** End of the current paid (or trial) period. */
    currentPeriodEnd?: Date;
    autoRenew: boolean;
    /** Whether this subscription is a sandbox/test purchase. */
    isSandbox: boolean;
}
/**
 * The complete subscription picture for one user, from
 * `moth.billing.v1.GetCustomerInfo`.
 *
 * A never-paid user, a free-tier user, and a user in a project with no
 * products all get a well-formed instance with empty entitlements (the
 * built-in `none` tier) — never an error. Gate with
 * {@link MothCustomerInfo.hasEntitlement}; never special-case "never paid".
 */
export declare class MothCustomerInfo {
    readonly activeEntitlements: readonly MothEntitlement[];
    readonly subscriptions: readonly MothActiveSubscription[];
    constructor(activeEntitlements?: readonly MothEntitlement[], subscriptions?: readonly MothActiveSubscription[]);
    /** The valid, empty state: no entitlements, the free `none` tier. */
    static free(): MothCustomerInfo;
    static fromProto(proto: CustomerInfo): MothCustomerInfo;
    /**
     * Whether the user currently holds the entitlement `identifier` (e.g.
     * `pro`) — the single question app code should ask to gate a feature. An
     * entitlement past its expiry no longer counts, so a cached snapshot
     * cannot grant lapsed access.
     */
    hasEntitlement(identifier: string, now?: number): boolean;
    /** The held entitlement with `identifier`, or undefined. */
    entitlement(identifier: string, now?: number): MothEntitlement | undefined;
    equals(other: MothCustomerInfo): boolean;
    /**
     * Re-encodes this snapshot into the `moth.billing.v1.CustomerInfo` wire
     * schema — the payload the per-user cache persists (protobuf, never JSON).
     */
    toProtoBytes(): Uint8Array;
    static fromProtoBytes(bytes: Uint8Array): MothCustomerInfo;
}
