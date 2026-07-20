import type { MothClient } from './client.js';
import type { MothConfigController } from './configController.js';
import { type MothPushPermission, type MothPushStatus } from './push.js';
import type { WebStorageLike } from './tokenStore.js';
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
export declare class MothPushController {
    #private;
    constructor(client: MothClient, configController: MothConfigController, options?: {
        storage?: WebStorageLike;
    });
    /** The current push status; see {@link MothPushStatus}. */
    get status(): MothPushStatus;
    /** The browser notification permission (`'unknown'` until asked). */
    get permission(): MothPushPermission;
    /** Subscribes to status/permission changes; replays nothing. */
    onChange(listener: () => void): () => void;
    /**
     * Begins tracking: while signed in, an existing browser subscription is
     * (re-)registered on every launch, and sign-out revokes the registration
     * before the session drops. Idempotent, and restartable after
     * {@link dispose} (React StrictMode mounts effects twice).
     */
    start(): void;
    dispose(): void;
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
    subscribe(): Promise<MothPushStatus>;
    /**
     * Unsubscribes the browser's push subscription and revokes the
     * registration (`UnregisterDevice`), best-effort — like {@link subscribe},
     * this never throws for environment reasons.
     */
    unsubscribe(): Promise<void>;
}
