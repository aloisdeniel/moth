import type { MothClient } from '../core/client.js';
import type { MothCustomerInfo, MothEntitlement } from '../core/customerInfo.js';
import type { MothPushPermission, MothPushStatus } from '../core/push.js';
import type { MothAuthState, MothUser } from '../core/user.js';
export interface UseMothResult {
    client: MothClient;
    /** The auth state; re-renders the component on every change. */
    state: MothAuthState;
    /** The signed-in user, or null. */
    user: MothUser | null;
    customerInfo: MothCustomerInfo;
    signOut: (options?: {
        allDevices?: boolean;
    }) => Promise<void>;
    /** Forces a token refresh and returns the updated user. */
    refreshUser: () => Promise<MothUser>;
    /** Deletes the account after re-authentication with the password. */
    deleteAccount: (password: string) => Promise<void>;
    /** Re-fetches the customer info from the server. */
    refreshCustomerInfo: () => Promise<MothCustomerInfo>;
}
/**
 * The moth client, auth state and common actions. Re-renders on every auth
 * or entitlement change.
 */
export declare function useMoth(): UseMothResult;
/** The signed-in user, or null. Re-renders on auth changes. */
export declare function useMothUser(): MothUser | null;
/**
 * The current subscription state — always valid, the free tier while signed
 * out. Re-renders on every entitlement change (cache hit on launch,
 * background refresh, checkout return, sign-out).
 */
export declare function useMothCustomerInfo(): MothCustomerInfo;
export interface UseMothEntitlementResult {
    /** Whether the user currently holds the entitlement. */
    active: boolean;
    /** The held entitlement (expiry, source), when active. */
    entitlement?: MothEntitlement;
}
/**
 * Whether the signed-in user holds `identifier` (e.g. `'pro'`) — the single
 * question app code should ask to gate a feature. Re-renders when the
 * entitlement flips, including at its expiry time (an expired cached
 * entitlement never keeps gating open).
 */
export declare function useMothEntitlement(identifier: string): UseMothEntitlementResult;
export interface UseMothPushResult {
    /**
     * Where Web Push stands for this installation: `unavailable` (project has
     * no push / no VAPID key), `unsupported` (browser lacks the Push API),
     * `idle`, `subscribed`, or `denied`. Environment problems are states,
     * never exceptions.
     */
    status: MothPushStatus;
    /** The browser notification permission (`'unknown'` until asked). */
    permission: MothPushPermission;
    /**
     * Prompts for permission, subscribes the app's service worker's
     * `PushManager` and registers the device; resolves to the new status. A
     * typed no-op when `unavailable`/`unsupported` — it never throws for
     * environment reasons.
     */
    subscribe: () => Promise<MothPushStatus>;
    /** Unsubscribes the browser subscription and revokes the registration. */
    unsubscribe: () => Promise<void>;
}
/**
 * Web Push subscription state and actions — a settings-screen toggle in one
 * hook. Re-renders on every push status/permission change; while signed in
 * an existing subscription is re-registered on every launch, and sign-out
 * revokes it automatically. The app owns its service worker (display and
 * click handling); see the README for a minimal `sw.js`.
 */
export declare function useMothPush(): UseMothPushResult;
