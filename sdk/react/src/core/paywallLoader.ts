import { fromBinary, toBinary } from '@bufbuild/protobuf'
import { PaywallSchema } from '../gen/moth/billing/v1/billing_pb.js'
import {
  BlobCache,
  cacheNamespace,
  defaultCacheStorage,
  paywallCacheKey,
} from './cache.js'
import type { MothClient } from './client.js'
import { configCacheTtlMs } from './config.js'
import { emptyPaywall, paywallFromProto, type MothPaywall } from './offering.js'
import type { WebStorageLike } from './tokenStore.js'

/**
 * Loads the paywall configuration with the same download-once, revision-
 * cached discipline as the theme/copy: the cached blob is served without
 * any network call while younger than `configCacheTtlMs`; once stale, the
 * cached revision is echoed so the server can omit an unchanged body
 * (which restarts the TTL window); a new body replaces the cache. Network
 * failures fall back to the cached (possibly stale) config; with no cache
 * at all they rethrow so the paywall can show its error state.
 */
export async function loadPaywall(
  client: MothClient,
  options: { storage?: WebStorageLike; now?: () => number } = {},
): Promise<MothPaywall> {
  const storage = options.storage ?? defaultCacheStorage()
  const now = options.now ?? (() => Date.now())
  const cache = new BlobCache(
    storage,
    paywallCacheKey(cacheNamespace(client.config.publishableKey)),
  )
  const blob = cache.load()
  let cached: MothPaywall | null = null
  if (blob !== null) {
    try {
      cached = paywallFromProto(fromBinary(PaywallSchema, blob.payload))
    } catch {
      cache.remove()
    }
  }
  if (
    cached !== null &&
    blob !== null &&
    now() - blob.fetchedAtMs < configCacheTtlMs(client.config)
  ) {
    return cached // download-once: fresh cache, zero RPCs
  }
  let fetched: MothPaywall | null
  try {
    fetched = await client.getPaywall({
      knownPaywallRevision: cached?.revisionId ?? '',
    })
  } catch (err) {
    if (cached !== null) return cached // offline: stale beats nothing
    throw err
  }
  const stamp = now()
  if (fetched === null) {
    // Omitted-body match: the cached payload is confirmed current —
    // restart its download-once TTL window.
    if (cached !== null) {
      cache.touch(stamp)
      return cached
    }
    return emptyPaywall
  }
  const raw = client.lastRawPaywall
  if (raw !== null) {
    cache.save({
      payload: toBinary(PaywallSchema, raw),
      revision: fetched.revisionId,
      locale: '',
      fetchedAtMs: stamp,
    })
  }
  return fetched
}
