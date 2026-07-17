import type { WebStorageLike } from './tokenStore.js';
/** A cached blob: the raw wire payload plus revalidation metadata. */
export interface CachedBlob {
    /** The serialized wire message exactly as the server delivered it. */
    payload: Uint8Array;
    /** The server revision the payload came from — the revalidation key. */
    revision: string;
    /** Negotiated locale for locale-keyed payloads (copy); empty otherwise. */
    locale: string;
    /** When the payload was fetched or last revalidated (Unix ms UTC). */
    fetchedAtMs: number;
}
/** The cache-key namespace for a publishable key. */
export declare function cacheNamespace(publishableKey: string): string;
/** In-memory Web Storage stand-in (tests, storage-less environments). */
export declare class MemoryStorage implements WebStorageLike {
    #private;
    getItem(key: string): string | null;
    setItem(key: string, value: string): void;
    removeItem(key: string): void;
}
/** localStorage when available, else an in-memory fallback. */
export declare function defaultCacheStorage(): WebStorageLike;
/**
 * One config-blob slot (theme, paywall, one copy locale, one user's customer
 * info) persisted as a base64 CacheEnvelope. All failures — storage throwing,
 * corrupt entries — surface as cache misses / no-ops, never as errors.
 */
export declare class BlobCache {
    #private;
    constructor(storage: WebStorageLike, key: string);
    load(): CachedBlob | null;
    save(blob: CachedBlob): void;
    /**
     * Re-stamps the cached blob's fetch time after the server confirmed the
     * cached revision is still current (an omitted-body revalidation), so the
     * download-once TTL window restarts. No-op on a miss.
     */
    touch(fetchedAtMs: number): void;
    remove(): void;
}
/** The theme cache slot for a project. */
export declare function themeCacheKey(namespace: string): string;
/** The paywall cache slot for a project. */
export declare function paywallCacheKey(namespace: string): string;
/**
 * The copy cache slot for a project + language. Keyed by *language* only
 * (`en-US` and the server-negotiated `en` share a slot) so an offline
 * relaunch on a region-tagged browser still finds the cached copy.
 */
export declare function copyCacheKey(namespace: string, language: string): string;
/** The per-user customer-info cache slot. User ids are UUIDs — safe as-is. */
export declare function customerInfoCacheKey(namespace: string, userId: string): string;
export declare function base64Encode(bytes: Uint8Array): string;
export declare function base64Decode(text: string): Uint8Array;
