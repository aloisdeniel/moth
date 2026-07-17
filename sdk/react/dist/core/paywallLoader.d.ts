import type { MothClient } from './client.js';
import { type MothPaywall } from './offering.js';
import type { WebStorageLike } from './tokenStore.js';
/**
 * Loads the paywall configuration with the same download-once, revision-
 * cached discipline as the theme/copy: the cached blob is served without
 * any network call while younger than `configCacheTtlMs`; once stale, the
 * cached revision is echoed so the server can omit an unchanged body
 * (which restarts the TTL window); a new body replaces the cache. Network
 * failures fall back to the cached (possibly stale) config; with no cache
 * at all they rethrow so the paywall can show its error state.
 */
export declare function loadPaywall(client: MothClient, options?: {
    storage?: WebStorageLike;
    now?: () => number;
}): Promise<MothPaywall>;
