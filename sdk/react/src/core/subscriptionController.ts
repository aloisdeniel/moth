import {
  BlobCache,
  cacheNamespace,
  customerInfoCacheKey,
  defaultCacheStorage,
} from './cache.js'
import type { MothClient } from './client.js'
import { MothCustomerInfo } from './customerInfo.js'
import type { WebStorageLike } from './tokenStore.js'

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
  readonly #client: MothClient
  readonly #storage: WebStorageLike
  readonly #namespace: string

  /** The user id the current value belongs to; null while signed out. */
  #userId: string | null = null
  #disposed = false
  #unsubscribers: (() => void)[] = []

  constructor(client: MothClient, options: { storage?: WebStorageLike } = {}) {
    this.#client = client
    this.#storage = options.storage ?? defaultCacheStorage()
    this.#namespace = cacheNamespace(client.config.publishableKey)
  }

  /**
   * Begins tracking. Idempotent, and restartable after {@link dispose}
   * (React StrictMode mounts effects twice); failures are swallowed — the
   * current value simply stays.
   */
  start(): void {
    if (this.#unsubscribers.length > 0) return
    this.#disposed = false
    // The entitlement subscription must be attached before the auth one so
    // a fresh GetCustomerInfo triggered by the sign-in transition is not
    // missed — and, on sign-out, the auth listener drops the user id
    // BEFORE the free reset arrives (the client emits signedOut first), so
    // the outgoing user's cached entitlements are never overwritten.
    this.#unsubscribers.push(
      this.#client.onEntitlementsChanged((info) => this.#onInfo(info)),
      this.#client.onAuthStateChanged((state) => {
        if (state.status === 'signedIn') {
          if (this.#userId === state.user.id) return // token refresh etc.
          this.#userId = state.user.id
          void this.#loadAndRefresh(state.user.id)
        } else {
          this.#userId = null
        }
      }),
    )
  }

  dispose(): void {
    this.#disposed = true
    this.#userId = null
    for (const unsubscribe of this.#unsubscribers) unsubscribe()
    this.#unsubscribers = []
  }

  #cache(userId: string): BlobCache {
    return new BlobCache(
      this.#storage,
      customerInfoCacheKey(this.#namespace, userId),
    )
  }

  #onInfo(info: MothCustomerInfo): void {
    const userId = this.#userId
    if (userId === null || this.#disposed) return
    // Persist the latest server truth for instant gating next launch.
    try {
      this.#cache(userId).save({
        payload: info.toProtoBytes(),
        revision: '',
        locale: '',
        fetchedAtMs: Date.now(),
      })
    } catch {
      // Best effort.
    }
  }

  async #loadAndRefresh(userId: string): Promise<void> {
    try {
      const blob = this.#cache(userId).load()
      if (blob !== null && this.#userId === userId && !this.#disposed) {
        // Mirror the cached snapshot into the client so all subscribers and
        // currentCustomerInfo agree with it until the refresh confirms.
        this.#client.primeCustomerInfo(
          MothCustomerInfo.fromProtoBytes(blob.payload),
        )
      }
    } catch {
      // Broken cache — treat as a miss.
    }
    // Background refresh; the result arrives via onEntitlementsChanged.
    // Failures (offline) keep the cached (or free) snapshot.
    try {
      await this.#client.getCustomerInfo()
    } catch {
      // Best effort.
    }
  }
}
