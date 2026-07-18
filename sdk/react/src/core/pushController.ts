import { cacheNamespace, defaultCacheStorage } from './cache.js'
import type { MothClient } from './client.js'
import type { MothConfigController } from './configController.js'
import {
  pushPermissionFromNotification,
  vapidKeyBytes,
  type MothPushPermission,
  type MothPushStatus,
} from './push.js'
import type { WebStorageLike } from './tokenStore.js'

/**
 * Owns Web Push registration for a browser installation, against
 * `moth.push.v1` (framework-free; `useMothPush` is a thin binding):
 *
 * - {@link subscribe} runs the whole opt-in: browser permission prompt →
 *   `PushManager.subscribe` with the project's VAPID public key (from the
 *   public project config) → `RegisterDevice` with a stable, persisted
 *   installation id.
 * - While signed in, an existing browser subscription is re-registered on
 *   every launch — the registry's upsert semantics are the retry policy, so
 *   there is no client-side "am I registered?" bookkeeping to corrupt.
 * - Sign-out revokes the registration *before* the session drops (via
 *   `MothClient.onBeforeSignOut`), best-effort.
 *
 * Environment problems are states, never exceptions: a project without push
 * (or before the config arrives) reports `unavailable`, a browser without
 * the Push API reports `unsupported` (feature-detected), and `subscribe()`
 * is a typed no-op in both. Registration failures are non-fatal by design —
 * auth never blocks on push; the next launch retries.
 *
 * The app supplies its own service worker (display and click handling stay
 * app code); the SDK only manages the subscription and the registry row.
 *
 * `MothProvider` creates one automatically.
 */
export class MothPushController {
  readonly #client: MothClient
  readonly #config: MothConfigController
  readonly #storage: WebStorageLike
  readonly #namespace: string

  /** Whether this installation is registered for the current session. */
  #registered = false
  /** The installation id, once read or minted (survives broken storage). */
  #deviceIdCache: string | null = null
  /** The user id the launch sync last ran for; null while signed out. */
  #userId: string | null = null
  #disposed = false
  #unsubscribers: (() => void)[] = []
  #listeners = new Set<() => void>()

  constructor(
    client: MothClient,
    configController: MothConfigController,
    options: { storage?: WebStorageLike } = {},
  ) {
    this.#client = client
    this.#config = configController
    this.#storage = options.storage ?? defaultCacheStorage()
    this.#namespace = cacheNamespace(client.config.publishableKey)
  }

  /** The current push status; see {@link MothPushStatus}. */
  get status(): MothPushStatus {
    if (!this.#supported()) return 'unsupported'
    if (!this.#available()) return 'unavailable'
    const permission = this.permission
    if (permission === 'denied') return 'denied'
    return this.#registered ? 'subscribed' : 'idle'
  }

  /** The browser notification permission (`'unknown'` until asked). */
  get permission(): MothPushPermission {
    if (typeof Notification === 'undefined') return 'unknown'
    return pushPermissionFromNotification(Notification.permission)
  }

  /** Subscribes to status/permission changes; replays nothing. */
  onChange(listener: () => void): () => void {
    this.#listeners.add(listener)
    return () => this.#listeners.delete(listener)
  }

  /**
   * Begins tracking: while signed in, an existing browser subscription is
   * (re-)registered on every launch, and sign-out revokes the registration
   * before the session drops. Idempotent, and restartable after
   * {@link dispose} (React StrictMode mounts effects twice).
   */
  start(): void {
    if (this.#unsubscribers.length > 0) return
    this.#disposed = false
    this.#unsubscribers.push(
      this.#client.onAuthStateChanged((state) => {
        if (state.status === 'signedIn') {
          if (this.#userId === state.user.id) return // token refresh etc.
          this.#userId = state.user.id
          void this.#sync()
        } else {
          this.#userId = null
          // The registration belongs to a session; without one there is
          // nothing registered (a sign-out elsewhere already revoked it,
          // an expired session gets re-registered on the next sign-in).
          this.#registered = false
          this.#notify()
        }
      }),
      this.#client.onBeforeSignOut(() => this.#unregisterForSignOut()),
      // The status flips from `unavailable` when the project config lands.
      this.#config.subscribe(() => this.#notify()),
    )
  }

  dispose(): void {
    this.#disposed = true
    this.#userId = null
    for (const unsubscribe of this.#unsubscribers) unsubscribe()
    this.#unsubscribers = []
    this.#listeners.clear()
  }

  /**
   * Runs the Web Push opt-in: requests the browser notification permission,
   * subscribes the app's service worker's `PushManager` with the project's
   * VAPID public key, and registers the serialized subscription
   * (`target: webpush`). Resolves to the resulting {@link status}; never
   * throws for environment reasons — an unsupported browser or a project
   * without push resolve unchanged (`unsupported` / `unavailable`), a
   * denied prompt resolves `denied`, and a registry failure is non-fatal
   * (the subscription survives; the next launch retries the registration).
   */
  async subscribe(): Promise<MothPushStatus> {
    if (!this.#supported()) return this.status
    const vapidKey = await this.#ensureVapidKey()
    if (vapidKey === null) return this.status
    let permission: NotificationPermission
    try {
      permission = await Notification.requestPermission()
    } catch {
      permission = Notification.permission
    }
    this.#notify() // the permission may have changed either way
    if (permission !== 'granted') return this.status
    try {
      const registration = await navigator.serviceWorker.ready
      const subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: toBufferSource(vapidKeyBytes(vapidKey)),
      })
      await this.#register(subscription)
      this.#registered = true
    } catch {
      // The browser refused the subscription, or the registry RPC failed:
      // stay idle. The subscription (when created) is found again by the
      // next launch's sync, so the registration self-heals.
    }
    this.#notify()
    return this.status
  }

  /**
   * Unsubscribes the browser's push subscription and revokes the
   * registration (`UnregisterDevice`), best-effort — like {@link subscribe},
   * this never throws for environment reasons.
   */
  async unsubscribe(): Promise<void> {
    if (this.#supported()) {
      try {
        const subscription = await this.#currentSubscription()
        if (subscription !== null) await subscription.unsubscribe()
      } catch {
        // Best effort.
      }
    }
    this.#registered = false
    try {
      await this.#client.unregisterPushDevice(this.#deviceId())
    } catch {
      // Signed out, offline, ...: the registry sweep (or the next
      // sign-out) catches up; unregistration is idempotent.
    }
    this.#notify()
  }

  // ------------------------------------------------------------ internals

  /** Feature detection — never user-agent sniffing. */
  #supported(): boolean {
    return (
      typeof window !== 'undefined' &&
      'PushManager' in window &&
      typeof navigator !== 'undefined' &&
      'serviceWorker' in navigator &&
      typeof Notification !== 'undefined'
    )
  }

  /** Whether the (already-fetched) project config enables Web Push. */
  #available(): boolean {
    return this.#vapidKey() !== null
  }

  #vapidKey(): string | null {
    const push = this.#config.projectConfig?.push
    if (push === undefined || !push.enabled) return null
    const key = push.webpushVapidPublicKey
    return key === undefined || key === '' ? null : key
  }

  /** The VAPID key, fetching the project config when none arrived yet. */
  async #ensureVapidKey(): Promise<string | null> {
    try {
      await this.#config.ensureProjectConfig()
    } catch {
      // Offline: fall through to whatever config is (not) cached.
    }
    return this.#vapidKey()
  }

  /**
   * Re-registers an existing subscription after a sign-in (and thus on
   * every launch while signed in) — upsert semantics make this carefree,
   * and it doubles as the liveness heartbeat.
   */
  async #sync(): Promise<void> {
    if (!this.#supported()) return
    const vapidKey = await this.#ensureVapidKey()
    this.#notify() // the config fetch may have flipped `unavailable`
    if (vapidKey === null || this.#disposed) return
    if (Notification.permission !== 'granted') return
    try {
      const subscription = await this.#currentSubscription()
      if (subscription === null || this.#disposed) return
      await this.#register(subscription)
      this.#registered = true
      this.#notify()
    } catch {
      // Non-fatal: sign-in never blocks on push; the next launch retries.
    }
  }

  /**
   * The current browser subscription, or null. Deliberately via
   * `getRegistration()` (not `.ready`, which never settles when the app
   * registered no service worker — only the explicit {@link subscribe}
   * opt-in may wait on one).
   */
  async #currentSubscription(): Promise<PushSubscription | null> {
    const registration = await navigator.serviceWorker.getRegistration()
    if (registration === undefined) return null
    return registration.pushManager.getSubscription()
  }

  async #register(subscription: PushSubscription): Promise<void> {
    await this.#client.registerPushDevice({
      target: 'webpush',
      token: JSON.stringify(subscription),
      deviceId: this.#deviceId(),
      permission: this.permission,
      metadata: { platform: 'web', locale: this.#client.currentLocale },
    })
  }

  /**
   * Revokes the registration while the session is still live; best-effort.
   * Gated on the persisted installation id — not on this launch's
   * `#registered` flag — so a still-active row from a previous session is
   * revoked even when this launch's sync failed or is still in flight.
   */
  async #unregisterForSignOut(): Promise<void> {
    this.#registered = false
    const deviceId = this.#persistedDeviceId()
    // Never registered on this installation: nothing to revoke.
    if (deviceId === null) return
    try {
      await this.#client.unregisterPushDevice(deviceId)
    } catch {
      // Non-fatal: the sign-out proceeds; the row goes stale server-side.
    }
    this.#notify()
  }

  /**
   * The stable installation id, persisted alongside the SDK's other storage
   * (namespaced per publishable key) so re-registrations replace this
   * installation's row instead of accumulating.
   */
  #deviceId(): string {
    const existing = this.#persistedDeviceId()
    if (existing !== null) return existing
    const id = randomDeviceId()
    this.#deviceIdCache = id
    try {
      this.#storage.setItem(this.#deviceIdStorageKey(), id)
    } catch {
      // Best effort — worst case the next launch registers a new row and
      // the old one goes stale.
    }
    return id
  }

  /**
   * The already-persisted installation id, or null when this installation
   * never registered — the sign-out gate: no id, nothing to revoke. Never
   * mints one (that is {@link #deviceId}, on registration only).
   */
  #persistedDeviceId(): string | null {
    if (this.#deviceIdCache !== null) return this.#deviceIdCache
    try {
      const existing = this.#storage.getItem(this.#deviceIdStorageKey())
      if (existing !== null && existing !== '') {
        this.#deviceIdCache = existing
        return existing
      }
    } catch {
      // Storage misbehaving: treat as never registered.
    }
    return null
  }

  #deviceIdStorageKey(): string {
    return `moth_${this.#namespace}_push_device`
  }

  #notify(): void {
    if (this.#disposed) return
    for (const listener of [...this.#listeners]) listener()
  }
}

function randomDeviceId(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  let id = ''
  for (let i = 0; i < 32; i++) id += Math.floor(Math.random() * 16).toString(16)
  return id
}

/**
 * Re-wraps decoded key bytes in a plain ArrayBuffer: `applicationServerKey`
 * accepts a BufferSource, and a `Uint8Array<ArrayBufferLike>` view is not
 * assignable to it under TS 5.7's split ArrayBuffer types.
 */
function toBufferSource(bytes: Uint8Array): ArrayBuffer {
  const buffer = new ArrayBuffer(bytes.length)
  new Uint8Array(buffer).set(bytes)
  return buffer
}
