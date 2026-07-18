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
var _MothPushController_instances, _MothPushController_client, _MothPushController_config, _MothPushController_storage, _MothPushController_namespace, _MothPushController_registered, _MothPushController_deviceIdCache, _MothPushController_userId, _MothPushController_disposed, _MothPushController_unsubscribers, _MothPushController_listeners, _MothPushController_supported, _MothPushController_available, _MothPushController_vapidKey, _MothPushController_ensureVapidKey, _MothPushController_sync, _MothPushController_currentSubscription, _MothPushController_register, _MothPushController_unregisterForSignOut, _MothPushController_deviceId, _MothPushController_persistedDeviceId, _MothPushController_deviceIdStorageKey, _MothPushController_notify;
import { cacheNamespace, defaultCacheStorage } from './cache.js';
import { pushPermissionFromNotification, vapidKeyBytes, } from './push.js';
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
    constructor(client, configController, options = {}) {
        _MothPushController_instances.add(this);
        _MothPushController_client.set(this, void 0);
        _MothPushController_config.set(this, void 0);
        _MothPushController_storage.set(this, void 0);
        _MothPushController_namespace.set(this, void 0);
        /** Whether this installation is registered for the current session. */
        _MothPushController_registered.set(this, false
        /** The installation id, once read or minted (survives broken storage). */
        );
        /** The installation id, once read or minted (survives broken storage). */
        _MothPushController_deviceIdCache.set(this, null
        /** The user id the launch sync last ran for; null while signed out. */
        );
        /** The user id the launch sync last ran for; null while signed out. */
        _MothPushController_userId.set(this, null);
        _MothPushController_disposed.set(this, false);
        _MothPushController_unsubscribers.set(this, []);
        _MothPushController_listeners.set(this, new Set());
        __classPrivateFieldSet(this, _MothPushController_client, client, "f");
        __classPrivateFieldSet(this, _MothPushController_config, configController, "f");
        __classPrivateFieldSet(this, _MothPushController_storage, options.storage ?? defaultCacheStorage(), "f");
        __classPrivateFieldSet(this, _MothPushController_namespace, cacheNamespace(client.config.publishableKey), "f");
    }
    /** The current push status; see {@link MothPushStatus}. */
    get status() {
        if (!__classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_supported).call(this))
            return 'unsupported';
        if (!__classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_available).call(this))
            return 'unavailable';
        const permission = this.permission;
        if (permission === 'denied')
            return 'denied';
        return __classPrivateFieldGet(this, _MothPushController_registered, "f") ? 'subscribed' : 'idle';
    }
    /** The browser notification permission (`'unknown'` until asked). */
    get permission() {
        if (typeof Notification === 'undefined')
            return 'unknown';
        return pushPermissionFromNotification(Notification.permission);
    }
    /** Subscribes to status/permission changes; replays nothing. */
    onChange(listener) {
        __classPrivateFieldGet(this, _MothPushController_listeners, "f").add(listener);
        return () => __classPrivateFieldGet(this, _MothPushController_listeners, "f").delete(listener);
    }
    /**
     * Begins tracking: while signed in, an existing browser subscription is
     * (re-)registered on every launch, and sign-out revokes the registration
     * before the session drops. Idempotent, and restartable after
     * {@link dispose} (React StrictMode mounts effects twice).
     */
    start() {
        if (__classPrivateFieldGet(this, _MothPushController_unsubscribers, "f").length > 0)
            return;
        __classPrivateFieldSet(this, _MothPushController_disposed, false, "f");
        __classPrivateFieldGet(this, _MothPushController_unsubscribers, "f").push(__classPrivateFieldGet(this, _MothPushController_client, "f").onAuthStateChanged((state) => {
            if (state.status === 'signedIn') {
                if (__classPrivateFieldGet(this, _MothPushController_userId, "f") === state.user.id)
                    return; // token refresh etc.
                __classPrivateFieldSet(this, _MothPushController_userId, state.user.id, "f");
                void __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_sync).call(this);
            }
            else {
                __classPrivateFieldSet(this, _MothPushController_userId, null, "f");
                // The registration belongs to a session; without one there is
                // nothing registered (a sign-out elsewhere already revoked it,
                // an expired session gets re-registered on the next sign-in).
                __classPrivateFieldSet(this, _MothPushController_registered, false, "f");
                __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_notify).call(this);
            }
        }), __classPrivateFieldGet(this, _MothPushController_client, "f").onBeforeSignOut(() => __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_unregisterForSignOut).call(this)), 
        // The status flips from `unavailable` when the project config lands.
        __classPrivateFieldGet(this, _MothPushController_config, "f").subscribe(() => __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_notify).call(this)));
    }
    dispose() {
        __classPrivateFieldSet(this, _MothPushController_disposed, true, "f");
        __classPrivateFieldSet(this, _MothPushController_userId, null, "f");
        for (const unsubscribe of __classPrivateFieldGet(this, _MothPushController_unsubscribers, "f"))
            unsubscribe();
        __classPrivateFieldSet(this, _MothPushController_unsubscribers, [], "f");
        __classPrivateFieldGet(this, _MothPushController_listeners, "f").clear();
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
    async subscribe() {
        if (!__classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_supported).call(this))
            return this.status;
        const vapidKey = await __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_ensureVapidKey).call(this);
        if (vapidKey === null)
            return this.status;
        let permission;
        try {
            permission = await Notification.requestPermission();
        }
        catch {
            permission = Notification.permission;
        }
        __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_notify).call(this); // the permission may have changed either way
        if (permission !== 'granted')
            return this.status;
        try {
            const registration = await navigator.serviceWorker.ready;
            const subscription = await registration.pushManager.subscribe({
                userVisibleOnly: true,
                applicationServerKey: toBufferSource(vapidKeyBytes(vapidKey)),
            });
            await __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_register).call(this, subscription);
            __classPrivateFieldSet(this, _MothPushController_registered, true, "f");
        }
        catch {
            // The browser refused the subscription, or the registry RPC failed:
            // stay idle. The subscription (when created) is found again by the
            // next launch's sync, so the registration self-heals.
        }
        __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_notify).call(this);
        return this.status;
    }
    /**
     * Unsubscribes the browser's push subscription and revokes the
     * registration (`UnregisterDevice`), best-effort — like {@link subscribe},
     * this never throws for environment reasons.
     */
    async unsubscribe() {
        if (__classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_supported).call(this)) {
            try {
                const subscription = await __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_currentSubscription).call(this);
                if (subscription !== null)
                    await subscription.unsubscribe();
            }
            catch {
                // Best effort.
            }
        }
        __classPrivateFieldSet(this, _MothPushController_registered, false, "f");
        try {
            await __classPrivateFieldGet(this, _MothPushController_client, "f").unregisterPushDevice(__classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_deviceId).call(this));
        }
        catch {
            // Signed out, offline, ...: the registry sweep (or the next
            // sign-out) catches up; unregistration is idempotent.
        }
        __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_notify).call(this);
    }
}
_MothPushController_client = new WeakMap(), _MothPushController_config = new WeakMap(), _MothPushController_storage = new WeakMap(), _MothPushController_namespace = new WeakMap(), _MothPushController_registered = new WeakMap(), _MothPushController_deviceIdCache = new WeakMap(), _MothPushController_userId = new WeakMap(), _MothPushController_disposed = new WeakMap(), _MothPushController_unsubscribers = new WeakMap(), _MothPushController_listeners = new WeakMap(), _MothPushController_instances = new WeakSet(), _MothPushController_supported = function _MothPushController_supported() {
    return (typeof window !== 'undefined' &&
        'PushManager' in window &&
        typeof navigator !== 'undefined' &&
        'serviceWorker' in navigator &&
        typeof Notification !== 'undefined');
}, _MothPushController_available = function _MothPushController_available() {
    return __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_vapidKey).call(this) !== null;
}, _MothPushController_vapidKey = function _MothPushController_vapidKey() {
    const push = __classPrivateFieldGet(this, _MothPushController_config, "f").projectConfig?.push;
    if (push === undefined || !push.enabled)
        return null;
    const key = push.webpushVapidPublicKey;
    return key === undefined || key === '' ? null : key;
}, _MothPushController_ensureVapidKey = 
/** The VAPID key, fetching the project config when none arrived yet. */
async function _MothPushController_ensureVapidKey() {
    try {
        await __classPrivateFieldGet(this, _MothPushController_config, "f").ensureProjectConfig();
    }
    catch {
        // Offline: fall through to whatever config is (not) cached.
    }
    return __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_vapidKey).call(this);
}, _MothPushController_sync = 
/**
 * Re-registers an existing subscription after a sign-in (and thus on
 * every launch while signed in) — upsert semantics make this carefree,
 * and it doubles as the liveness heartbeat.
 */
async function _MothPushController_sync() {
    if (!__classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_supported).call(this))
        return;
    const vapidKey = await __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_ensureVapidKey).call(this);
    __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_notify).call(this); // the config fetch may have flipped `unavailable`
    if (vapidKey === null || __classPrivateFieldGet(this, _MothPushController_disposed, "f"))
        return;
    if (Notification.permission !== 'granted')
        return;
    try {
        const subscription = await __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_currentSubscription).call(this);
        if (subscription === null || __classPrivateFieldGet(this, _MothPushController_disposed, "f"))
            return;
        await __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_register).call(this, subscription);
        __classPrivateFieldSet(this, _MothPushController_registered, true, "f");
        __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_notify).call(this);
    }
    catch {
        // Non-fatal: sign-in never blocks on push; the next launch retries.
    }
}, _MothPushController_currentSubscription = 
/**
 * The current browser subscription, or null. Deliberately via
 * `getRegistration()` (not `.ready`, which never settles when the app
 * registered no service worker — only the explicit {@link subscribe}
 * opt-in may wait on one).
 */
async function _MothPushController_currentSubscription() {
    const registration = await navigator.serviceWorker.getRegistration();
    if (registration === undefined)
        return null;
    return registration.pushManager.getSubscription();
}, _MothPushController_register = async function _MothPushController_register(subscription) {
    await __classPrivateFieldGet(this, _MothPushController_client, "f").registerPushDevice({
        target: 'webpush',
        token: JSON.stringify(subscription),
        deviceId: __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_deviceId).call(this),
        permission: this.permission,
        metadata: { platform: 'web', locale: __classPrivateFieldGet(this, _MothPushController_client, "f").currentLocale },
    });
}, _MothPushController_unregisterForSignOut = 
/**
 * Revokes the registration while the session is still live; best-effort.
 * Gated on the persisted installation id — not on this launch's
 * `#registered` flag — so a still-active row from a previous session is
 * revoked even when this launch's sync failed or is still in flight.
 */
async function _MothPushController_unregisterForSignOut() {
    __classPrivateFieldSet(this, _MothPushController_registered, false, "f");
    const deviceId = __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_persistedDeviceId).call(this);
    // Never registered on this installation: nothing to revoke.
    if (deviceId === null)
        return;
    try {
        await __classPrivateFieldGet(this, _MothPushController_client, "f").unregisterPushDevice(deviceId);
    }
    catch {
        // Non-fatal: the sign-out proceeds; the row goes stale server-side.
    }
    __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_notify).call(this);
}, _MothPushController_deviceId = function _MothPushController_deviceId() {
    const existing = __classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_persistedDeviceId).call(this);
    if (existing !== null)
        return existing;
    const id = randomDeviceId();
    __classPrivateFieldSet(this, _MothPushController_deviceIdCache, id, "f");
    try {
        __classPrivateFieldGet(this, _MothPushController_storage, "f").setItem(__classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_deviceIdStorageKey).call(this), id);
    }
    catch {
        // Best effort — worst case the next launch registers a new row and
        // the old one goes stale.
    }
    return id;
}, _MothPushController_persistedDeviceId = function _MothPushController_persistedDeviceId() {
    if (__classPrivateFieldGet(this, _MothPushController_deviceIdCache, "f") !== null)
        return __classPrivateFieldGet(this, _MothPushController_deviceIdCache, "f");
    try {
        const existing = __classPrivateFieldGet(this, _MothPushController_storage, "f").getItem(__classPrivateFieldGet(this, _MothPushController_instances, "m", _MothPushController_deviceIdStorageKey).call(this));
        if (existing !== null && existing !== '') {
            __classPrivateFieldSet(this, _MothPushController_deviceIdCache, existing, "f");
            return existing;
        }
    }
    catch {
        // Storage misbehaving: treat as never registered.
    }
    return null;
}, _MothPushController_deviceIdStorageKey = function _MothPushController_deviceIdStorageKey() {
    return `moth_${__classPrivateFieldGet(this, _MothPushController_namespace, "f")}_push_device`;
}, _MothPushController_notify = function _MothPushController_notify() {
    if (__classPrivateFieldGet(this, _MothPushController_disposed, "f"))
        return;
    for (const listener of [...__classPrivateFieldGet(this, _MothPushController_listeners, "f")])
        listener();
};
function randomDeviceId() {
    if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
        return crypto.randomUUID();
    }
    let id = '';
    for (let i = 0; i < 32; i++)
        id += Math.floor(Math.random() * 16).toString(16);
    return id;
}
/**
 * Re-wraps decoded key bytes in a plain ArrayBuffer: `applicationServerKey`
 * accepts a BufferSource, and a `Uint8Array<ArrayBufferLike>` view is not
 * assignable to it under TS 5.7's split ArrayBuffer types.
 */
function toBufferSource(bytes) {
    const buffer = new ArrayBuffer(bytes.length);
    new Uint8Array(buffer).set(bytes);
    return buffer;
}
