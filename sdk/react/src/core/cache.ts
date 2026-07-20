import { create, fromBinary, toBinary } from '@bufbuild/protobuf'
import { CacheEnvelopeSchema } from '../gen/moth/projectconfig/v1/projectconfig_pb.js'
import { sha256Hex } from './sha256.js'
import type { WebStorageLike } from './tokenStore.js'

// The SDK's config caches (theme, localized copy, paywall, per-user customer
// info). Every cached blob is a `moth.projectconfig.v1.CacheEnvelope`
// protobuf — payload = the raw wire message exactly as the server delivered
// it — base64-encoded into web storage. Cached blobs are protobuf, never
// JSON: one schema for the wire and the cache (project rule).
//
// Keys are namespaced by sha256(publishableKey)[:16] so two projects on one
// origin never collide; the copy cache is additionally keyed by language.

/** A cached blob: the raw wire payload plus revalidation metadata. */
export interface CachedBlob {
  /** The serialized wire message exactly as the server delivered it. */
  payload: Uint8Array
  /** The server revision the payload came from — the revalidation key. */
  revision: string
  /** Negotiated locale for locale-keyed payloads (copy); empty otherwise. */
  locale: string
  /** When the payload was fetched or last revalidated (Unix ms UTC). */
  fetchedAtMs: number
}

/** The cache-key namespace for a publishable key. */
export function cacheNamespace(publishableKey: string): string {
  return sha256Hex(publishableKey).slice(0, 16)
}

/** In-memory Web Storage stand-in (tests, storage-less environments). */
export class MemoryStorage implements WebStorageLike {
  #map = new Map<string, string>()

  getItem(key: string): string | null {
    return this.#map.get(key) ?? null
  }

  setItem(key: string, value: string): void {
    this.#map.set(key, value)
  }

  removeItem(key: string): void {
    this.#map.delete(key)
  }
}

/** localStorage when available, else an in-memory fallback. */
export function defaultCacheStorage(): WebStorageLike {
  try {
    if (typeof window !== 'undefined' && window.localStorage) {
      return window.localStorage
    }
  } catch {
    // Storage disabled — fall through.
  }
  return new MemoryStorage()
}

/**
 * One config-blob slot (theme, paywall, one copy locale, one user's customer
 * info) persisted as a base64 CacheEnvelope. All failures — storage throwing,
 * corrupt entries — surface as cache misses / no-ops, never as errors.
 */
export class BlobCache {
  readonly #storage: WebStorageLike
  readonly #key: string

  constructor(storage: WebStorageLike, key: string) {
    this.#storage = storage
    this.#key = key
  }

  load(): CachedBlob | null {
    let raw: string | null
    try {
      raw = this.#storage.getItem(this.#key)
    } catch {
      return null
    }
    if (raw === null) return null
    try {
      const envelope = fromBinary(CacheEnvelopeSchema, base64Decode(raw))
      if (envelope.payload.length === 0) throw new Error('no payload')
      return {
        payload: envelope.payload,
        revision: envelope.revision,
        locale: envelope.locale,
        fetchedAtMs: Number(envelope.fetchedAtUnixMs),
      }
    } catch {
      // Corrupt or foreign content: a cache miss, never a crash. Drop the
      // entry so the next save starts clean.
      this.remove()
      return null
    }
  }

  save(blob: CachedBlob): void {
    try {
      const envelope = create(CacheEnvelopeSchema, {
        payload: blob.payload,
        revision: blob.revision,
        locale: blob.locale,
        fetchedAtUnixMs: BigInt(Math.trunc(blob.fetchedAtMs)),
      })
      this.#storage.setItem(
        this.#key,
        base64Encode(toBinary(CacheEnvelopeSchema, envelope)),
      )
    } catch {
      // Best effort — the config is re-delivered next launch.
    }
  }

  /**
   * Re-stamps the cached blob's fetch time after the server confirmed the
   * cached revision is still current (an omitted-body revalidation), so the
   * download-once TTL window restarts. No-op on a miss.
   */
  touch(fetchedAtMs: number): void {
    const blob = this.load()
    if (blob === null) return
    this.save({ ...blob, fetchedAtMs })
  }

  remove(): void {
    try {
      this.#storage.removeItem(this.#key)
    } catch {
      // Best effort.
    }
  }
}

/** The theme cache slot for a project. */
export function themeCacheKey(namespace: string): string {
  return `moth_${namespace}_theme`
}

/** The paywall cache slot for a project. */
export function paywallCacheKey(namespace: string): string {
  return `moth_${namespace}_paywall`
}

/**
 * The copy cache slot for a project + language. Keyed by *language* only
 * (`en-US` and the server-negotiated `en` share a slot) so an offline
 * relaunch on a region-tagged browser still finds the cached copy.
 */
export function copyCacheKey(namespace: string, language: string): string {
  return `moth_${namespace}_copy_${language}`
}

/** The per-user customer-info cache slot. User ids are UUIDs — safe as-is. */
export function customerInfoCacheKey(
  namespace: string,
  userId: string,
): string {
  return `moth_${namespace}_ci_${userId}`
}

export function base64Encode(bytes: Uint8Array): string {
  let bin = ''
  for (let i = 0; i < bytes.length; i += 0x8000) {
    bin += String.fromCharCode(...bytes.subarray(i, i + 0x8000))
  }
  return btoa(bin)
}

export function base64Decode(text: string): Uint8Array {
  const bin = atob(text)
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  return bytes
}
