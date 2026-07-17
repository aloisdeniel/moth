import { create, toBinary, fromBinary } from '@bufbuild/protobuf';
import { timestampDate, timestampFromDate } from '@bufbuild/protobuf/wkt';
import { CustomerInfoSchema, EntitlementSource, Store, SubscriptionStatus, } from '../gen/moth/billing/v1/billing_pb.js';
/** Whether `status` keeps the subscription's access. */
export function subscriptionStatusIsActive(status) {
    return (status === 'active' ||
        status === 'trialing' ||
        status === 'inGracePeriod' ||
        status === 'inBillingRetry');
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
export class MothCustomerInfo {
    constructor(activeEntitlements = [], subscriptions = []) {
        this.activeEntitlements = activeEntitlements;
        this.subscriptions = subscriptions;
    }
    /** The valid, empty state: no entitlements, the free `none` tier. */
    static free() {
        return freeInfo;
    }
    static fromProto(proto) {
        return new MothCustomerInfo(proto.activeEntitlements.map((e) => {
            const ent = {
                identifier: e.identifier,
                source: sourceFromProto(e.source),
                productIdentifier: e.productIdentifier,
            };
            if (e.expireTime)
                ent.expireTime = timestampDate(e.expireTime);
            return ent;
        }), proto.subscriptions.map((s) => {
            const sub = {
                productIdentifier: s.productIdentifier,
                status: statusFromProto(s.status),
                autoRenew: s.autoRenew,
                isSandbox: s.isSandbox,
            };
            const store = storeFromProto(s.store);
            if (store)
                sub.store = store;
            if (s.currentPeriodEnd) {
                sub.currentPeriodEnd = timestampDate(s.currentPeriodEnd);
            }
            return sub;
        }));
    }
    /**
     * Whether the user currently holds the entitlement `identifier` (e.g.
     * `pro`) — the single question app code should ask to gate a feature. An
     * entitlement past its expiry no longer counts, so a cached snapshot
     * cannot grant lapsed access.
     */
    hasEntitlement(identifier, now = Date.now()) {
        return this.entitlement(identifier, now) !== undefined;
    }
    /** The held entitlement with `identifier`, or undefined. */
    entitlement(identifier, now = Date.now()) {
        return this.activeEntitlements.find((e) => e.identifier === identifier &&
            (e.expireTime === undefined || e.expireTime.getTime() > now));
    }
    equals(other) {
        return (this.activeEntitlements.length === other.activeEntitlements.length &&
            this.subscriptions.length === other.subscriptions.length &&
            this.activeEntitlements.every((e, i) => entitlementEquals(e, other.activeEntitlements[i])) &&
            this.subscriptions.every((s, i) => subscriptionEquals(s, other.subscriptions[i])));
    }
    /**
     * Re-encodes this snapshot into the `moth.billing.v1.CustomerInfo` wire
     * schema — the payload the per-user cache persists (protobuf, never JSON).
     */
    toProtoBytes() {
        return toBinary(CustomerInfoSchema, create(CustomerInfoSchema, {
            activeEntitlements: this.activeEntitlements.map((e) => ({
                identifier: e.identifier,
                source: sourceToProto(e.source),
                productIdentifier: e.productIdentifier,
                ...(e.expireTime ? { expireTime: timestampFromDate(e.expireTime) } : {}),
            })),
            subscriptions: this.subscriptions.map((s) => ({
                productIdentifier: s.productIdentifier,
                store: storeToProto(s.store),
                status: statusToProto(s.status),
                autoRenew: s.autoRenew,
                isSandbox: s.isSandbox,
                ...(s.currentPeriodEnd
                    ? { currentPeriodEnd: timestampFromDate(s.currentPeriodEnd) }
                    : {}),
            })),
        }));
    }
    static fromProtoBytes(bytes) {
        return MothCustomerInfo.fromProto(fromBinary(CustomerInfoSchema, bytes));
    }
}
const freeInfo = new MothCustomerInfo();
function entitlementEquals(a, b) {
    return (a.identifier === b.identifier &&
        a.source === b.source &&
        a.productIdentifier === b.productIdentifier &&
        (a.expireTime?.getTime() ?? -1) === (b.expireTime?.getTime() ?? -1));
}
function subscriptionEquals(a, b) {
    return (a.productIdentifier === b.productIdentifier &&
        a.store === b.store &&
        a.status === b.status &&
        a.autoRenew === b.autoRenew &&
        a.isSandbox === b.isSandbox &&
        (a.currentPeriodEnd?.getTime() ?? -1) === (b.currentPeriodEnd?.getTime() ?? -1));
}
function sourceFromProto(source) {
    switch (source) {
        case EntitlementSource.STORE:
            return 'store';
        case EntitlementSource.GRANT:
            return 'grant';
        default:
            return 'none';
    }
}
function sourceToProto(source) {
    switch (source) {
        case 'store':
            return EntitlementSource.STORE;
        case 'grant':
            return EntitlementSource.GRANT;
        default:
            return EntitlementSource.NONE;
    }
}
function storeFromProto(store) {
    switch (store) {
        case Store.APPLE:
            return 'apple';
        case Store.GOOGLE:
            return 'google';
        case Store.STRIPE:
            return 'stripe';
        default:
            return undefined;
    }
}
function storeToProto(store) {
    switch (store) {
        case 'apple':
            return Store.APPLE;
        case 'google':
            return Store.GOOGLE;
        case 'stripe':
            return Store.STRIPE;
        default:
            return Store.UNSPECIFIED;
    }
}
function statusFromProto(status) {
    switch (status) {
        case SubscriptionStatus.ACTIVE:
            return 'active';
        case SubscriptionStatus.TRIALING:
            return 'trialing';
        case SubscriptionStatus.IN_GRACE_PERIOD:
            return 'inGracePeriod';
        case SubscriptionStatus.IN_BILLING_RETRY:
            return 'inBillingRetry';
        case SubscriptionStatus.PAUSED:
            return 'paused';
        case SubscriptionStatus.EXPIRED:
            return 'expired';
        case SubscriptionStatus.REVOKED:
            return 'revoked';
        default:
            return 'unspecified';
    }
}
function statusToProto(status) {
    switch (status) {
        case 'active':
            return SubscriptionStatus.ACTIVE;
        case 'trialing':
            return SubscriptionStatus.TRIALING;
        case 'inGracePeriod':
            return SubscriptionStatus.IN_GRACE_PERIOD;
        case 'inBillingRetry':
            return SubscriptionStatus.IN_BILLING_RETRY;
        case 'paused':
            return SubscriptionStatus.PAUSED;
        case 'expired':
            return SubscriptionStatus.EXPIRED;
        case 'revoked':
            return SubscriptionStatus.REVOKED;
        default:
            return SubscriptionStatus.UNSPECIFIED;
    }
}
