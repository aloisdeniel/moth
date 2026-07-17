var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var _MemoryStorage_map, _BlobCache_storage, _BlobCache_key;
import { create, fromBinary, toBinary } from '@bufbuild/protobuf';
import { CacheEnvelopeSchema } from '../gen/moth/projectconfig/v1/projectconfig_pb.js';
import { sha256Hex } from './sha256.js';
/** The cache-key namespace for a publishable key. */
export function cacheNamespace(publishableKey) {
    return sha256Hex(publishableKey).slice(0, 16);
}
/** In-memory Web Storage stand-in (tests, storage-less environments). */
export class MemoryStorage {
    constructor() {
        _MemoryStorage_map.set(this, new Map());
    }
    getItem(key) {
        return __classPrivateFieldGet(this, _MemoryStorage_map, "f").get(key) ?? null;
    }
    setItem(key, value) {
        __classPrivateFieldGet(this, _MemoryStorage_map, "f").set(key, value);
    }
    removeItem(key) {
        __classPrivateFieldGet(this, _MemoryStorage_map, "f").delete(key);
    }
}
_MemoryStorage_map = new WeakMap();
/** localStorage when available, else an in-memory fallback. */
export function defaultCacheStorage() {
    try {
        if (typeof window !== 'undefined' && window.localStorage) {
            return window.localStorage;
        }
    }
    catch {
        // Storage disabled — fall through.
    }
    return new MemoryStorage();
}
/**
 * One config-blob slot (theme, paywall, one copy locale, one user's customer
 * info) persisted as a base64 CacheEnvelope. All failures — storage throwing,
 * corrupt entries — surface as cache misses / no-ops, never as errors.
 */
export class BlobCache {
    constructor(storage, key) {
        _BlobCache_storage.set(this, void 0);
        _BlobCache_key.set(this, void 0);
        __classPrivateFieldSet(this, _BlobCache_storage, storage, "f");
        __classPrivateFieldSet(this, _BlobCache_key, key, "f");
    }
    load() {
        let raw;
        try {
            raw = __classPrivateFieldGet(this, _BlobCache_storage, "f").getItem(__classPrivateFieldGet(this, _BlobCache_key, "f"));
        }
        catch {
            return null;
        }
        if (raw === null)
            return null;
        try {
            const envelope = fromBinary(CacheEnvelopeSchema, base64Decode(raw));
            if (envelope.payload.length === 0)
                throw new Error('no payload');
            return {
                payload: envelope.payload,
                revision: envelope.revision,
                locale: envelope.locale,
                fetchedAtMs: Number(envelope.fetchedAtUnixMs),
            };
        }
        catch {
            // Corrupt or foreign content: a cache miss, never a crash. Drop the
            // entry so the next save starts clean.
            this.remove();
            return null;
        }
    }
    save(blob) {
        try {
            const envelope = create(CacheEnvelopeSchema, {
                payload: blob.payload,
                revision: blob.revision,
                locale: blob.locale,
                fetchedAtUnixMs: BigInt(Math.trunc(blob.fetchedAtMs)),
            });
            __classPrivateFieldGet(this, _BlobCache_storage, "f").setItem(__classPrivateFieldGet(this, _BlobCache_key, "f"), base64Encode(toBinary(CacheEnvelopeSchema, envelope)));
        }
        catch {
            // Best effort — the config is re-delivered next launch.
        }
    }
    /**
     * Re-stamps the cached blob's fetch time after the server confirmed the
     * cached revision is still current (an omitted-body revalidation), so the
     * download-once TTL window restarts. No-op on a miss.
     */
    touch(fetchedAtMs) {
        const blob = this.load();
        if (blob === null)
            return;
        this.save({ ...blob, fetchedAtMs });
    }
    remove() {
        try {
            __classPrivateFieldGet(this, _BlobCache_storage, "f").removeItem(__classPrivateFieldGet(this, _BlobCache_key, "f"));
        }
        catch {
            // Best effort.
        }
    }
}
_BlobCache_storage = new WeakMap(), _BlobCache_key = new WeakMap();
/** The theme cache slot for a project. */
export function themeCacheKey(namespace) {
    return `moth_${namespace}_theme`;
}
/** The paywall cache slot for a project. */
export function paywallCacheKey(namespace) {
    return `moth_${namespace}_paywall`;
}
/**
 * The copy cache slot for a project + language. Keyed by *language* only
 * (`en-US` and the server-negotiated `en` share a slot) so an offline
 * relaunch on a region-tagged browser still finds the cached copy.
 */
export function copyCacheKey(namespace, language) {
    return `moth_${namespace}_copy_${language}`;
}
/** The per-user customer-info cache slot. User ids are UUIDs — safe as-is. */
export function customerInfoCacheKey(namespace, userId) {
    return `moth_${namespace}_ci_${userId}`;
}
export function base64Encode(bytes) {
    let bin = '';
    for (let i = 0; i < bytes.length; i += 0x8000) {
        bin += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
    }
    return btoa(bin);
}
export function base64Decode(text) {
    const bin = atob(text);
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++)
        bytes[i] = bin.charCodeAt(i);
    return bytes;
}
