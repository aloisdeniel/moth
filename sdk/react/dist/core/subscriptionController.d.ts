import type { MothClient } from './client.js';
import type { WebStorageLike } from './tokenStore.js';
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
export declare class MothSubscriptionController {
    #private;
    constructor(client: MothClient, options?: {
        storage?: WebStorageLike;
    });
    /**
     * Begins tracking. Idempotent, and restartable after {@link dispose}
     * (React StrictMode mounts effects twice); failures are swallowed — the
     * current value simply stays.
     */
    start(): void;
    dispose(): void;
}
