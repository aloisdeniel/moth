var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var _MothSubscriptionController_instances, _MothSubscriptionController_client, _MothSubscriptionController_storage, _MothSubscriptionController_namespace, _MothSubscriptionController_userId, _MothSubscriptionController_disposed, _MothSubscriptionController_unsubscribers, _MothSubscriptionController_cache, _MothSubscriptionController_onInfo, _MothSubscriptionController_loadAndRefresh;
import { BlobCache, cacheNamespace, customerInfoCacheKey, defaultCacheStorage, } from './cache.js';
import { MothCustomerInfo } from './customerInfo.js';
/**
 * Owns the signed-in user's subscription state: on each transition to a new
 * signed-in user it loads that user's cached snapshot (publishing it via
 * `primeCustomerInfo` so gating is instant), then refreshes from the server
 * in the background; every billing RPC result is persisted per user —
 * stale-while-revalidate, the same shape as the theme controller.
 *
 * The cache is keyed by user id so switching accounts never leaks
 * entitlements, and it is a convenience, never the authority — the server
 * stays the source of truth. The cached blob is a CacheEnvelope-wrapped
 * `moth.billing.v1.CustomerInfo` wire message (protobuf, never JSON).
 *
 * `MothProvider` creates one automatically.
 */
export class MothSubscriptionController {
    constructor(client, options = {}) {
        _MothSubscriptionController_instances.add(this);
        _MothSubscriptionController_client.set(this, void 0);
        _MothSubscriptionController_storage.set(this, void 0);
        _MothSubscriptionController_namespace.set(this, void 0);
        /** The user id the current value belongs to; null while signed out. */
        _MothSubscriptionController_userId.set(this, null);
        _MothSubscriptionController_disposed.set(this, false);
        _MothSubscriptionController_unsubscribers.set(this, []);
        __classPrivateFieldSet(this, _MothSubscriptionController_client, client, "f");
        __classPrivateFieldSet(this, _MothSubscriptionController_storage, options.storage ?? defaultCacheStorage(), "f");
        __classPrivateFieldSet(this, _MothSubscriptionController_namespace, cacheNamespace(client.config.publishableKey), "f");
    }
    /**
     * Begins tracking. Idempotent, and restartable after {@link dispose}
     * (React StrictMode mounts effects twice); failures are swallowed — the
     * current value simply stays.
     */
    start() {
        if (__classPrivateFieldGet(this, _MothSubscriptionController_unsubscribers, "f").length > 0)
            return;
        __classPrivateFieldSet(this, _MothSubscriptionController_disposed, false, "f");
        // The entitlement subscription must be attached before the auth one so
        // a fresh GetCustomerInfo triggered by the sign-in transition is not
        // missed — and, on sign-out, the auth listener drops the user id
        // BEFORE the free reset arrives (the client emits signedOut first), so
        // the outgoing user's cached entitlements are never overwritten.
        __classPrivateFieldGet(this, _MothSubscriptionController_unsubscribers, "f").push(__classPrivateFieldGet(this, _MothSubscriptionController_client, "f").onEntitlementsChanged((info) => __classPrivateFieldGet(this, _MothSubscriptionController_instances, "m", _MothSubscriptionController_onInfo).call(this, info)), __classPrivateFieldGet(this, _MothSubscriptionController_client, "f").onAuthStateChanged((state) => {
            if (state.status === 'signedIn') {
                if (__classPrivateFieldGet(this, _MothSubscriptionController_userId, "f") === state.user.id)
                    return; // token refresh etc.
                __classPrivateFieldSet(this, _MothSubscriptionController_userId, state.user.id, "f");
                void __classPrivateFieldGet(this, _MothSubscriptionController_instances, "m", _MothSubscriptionController_loadAndRefresh).call(this, state.user.id);
            }
            else {
                __classPrivateFieldSet(this, _MothSubscriptionController_userId, null, "f");
            }
        }));
    }
    dispose() {
        __classPrivateFieldSet(this, _MothSubscriptionController_disposed, true, "f");
        __classPrivateFieldSet(this, _MothSubscriptionController_userId, null, "f");
        for (const unsubscribe of __classPrivateFieldGet(this, _MothSubscriptionController_unsubscribers, "f"))
            unsubscribe();
        __classPrivateFieldSet(this, _MothSubscriptionController_unsubscribers, [], "f");
    }
}
_MothSubscriptionController_client = new WeakMap(), _MothSubscriptionController_storage = new WeakMap(), _MothSubscriptionController_namespace = new WeakMap(), _MothSubscriptionController_userId = new WeakMap(), _MothSubscriptionController_disposed = new WeakMap(), _MothSubscriptionController_unsubscribers = new WeakMap(), _MothSubscriptionController_instances = new WeakSet(), _MothSubscriptionController_cache = function _MothSubscriptionController_cache(userId) {
    return new BlobCache(__classPrivateFieldGet(this, _MothSubscriptionController_storage, "f"), customerInfoCacheKey(__classPrivateFieldGet(this, _MothSubscriptionController_namespace, "f"), userId));
}, _MothSubscriptionController_onInfo = function _MothSubscriptionController_onInfo(info) {
    const userId = __classPrivateFieldGet(this, _MothSubscriptionController_userId, "f");
    if (userId === null || __classPrivateFieldGet(this, _MothSubscriptionController_disposed, "f"))
        return;
    // Persist the latest server truth for instant gating next launch.
    try {
        __classPrivateFieldGet(this, _MothSubscriptionController_instances, "m", _MothSubscriptionController_cache).call(this, userId).save({
            payload: info.toProtoBytes(),
            revision: '',
            locale: '',
            fetchedAtMs: Date.now(),
        });
    }
    catch {
        // Best effort.
    }
}, _MothSubscriptionController_loadAndRefresh = async function _MothSubscriptionController_loadAndRefresh(userId) {
    try {
        const blob = __classPrivateFieldGet(this, _MothSubscriptionController_instances, "m", _MothSubscriptionController_cache).call(this, userId).load();
        if (blob !== null && __classPrivateFieldGet(this, _MothSubscriptionController_userId, "f") === userId && !__classPrivateFieldGet(this, _MothSubscriptionController_disposed, "f")) {
            // Mirror the cached snapshot into the client so all subscribers and
            // currentCustomerInfo agree with it until the refresh confirms.
            __classPrivateFieldGet(this, _MothSubscriptionController_client, "f").primeCustomerInfo(MothCustomerInfo.fromProtoBytes(blob.payload));
        }
    }
    catch {
        // Broken cache — treat as a miss.
    }
    // Background refresh; the result arrives via onEntitlementsChanged.
    // Failures (offline) keep the cached (or free) snapshot.
    try {
        await __classPrivateFieldGet(this, _MothSubscriptionController_client, "f").getCustomerInfo();
    }
    catch {
        // Best effort.
    }
};
