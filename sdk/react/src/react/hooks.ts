import { useEffect, useMemo, useState } from 'react'
import type { MothClient } from '../core/client.js'
import type { MothCustomerInfo, MothEntitlement } from '../core/customerInfo.js'
import type { MothPushPermission, MothPushStatus } from '../core/push.js'
import type { MothAuthState, MothUser } from '../core/user.js'
import { useMothContext } from './context.js'

export interface UseMothResult {
  client: MothClient
  /** The auth state; re-renders the component on every change. */
  state: MothAuthState
  /** The signed-in user, or null. */
  user: MothUser | null
  customerInfo: MothCustomerInfo
  signOut: (options?: { allDevices?: boolean }) => Promise<void>
  /** Forces a token refresh and returns the updated user. */
  refreshUser: () => Promise<MothUser>
  /** Deletes the account after re-authentication with the password. */
  deleteAccount: (password: string) => Promise<void>
  /** Re-fetches the customer info from the server. */
  refreshCustomerInfo: () => Promise<MothCustomerInfo>
}

/**
 * The moth client, auth state and common actions. Re-renders on every auth
 * or entitlement change.
 */
export function useMoth(): UseMothResult {
  const { client, state, customerInfo } = useMothContext()
  return useMemo(
    () => ({
      client,
      state,
      user: state.status === 'signedIn' ? state.user : null,
      customerInfo,
      signOut: (options) => client.signOut(options),
      refreshUser: () => client.refresh(),
      deleteAccount: (password) => client.deleteAccount({ password }),
      refreshCustomerInfo: () => client.getCustomerInfo(),
    }),
    [client, state, customerInfo],
  )
}

/** The signed-in user, or null. Re-renders on auth changes. */
export function useMothUser(): MothUser | null {
  const { state } = useMothContext()
  return state.status === 'signedIn' ? state.user : null
}

/**
 * The current subscription state — always valid, the free tier while signed
 * out. Re-renders on every entitlement change (cache hit on launch,
 * background refresh, checkout return, sign-out).
 */
export function useMothCustomerInfo(): MothCustomerInfo {
  return useMothContext().customerInfo
}

export interface UseMothEntitlementResult {
  /** Whether the user currently holds the entitlement. */
  active: boolean
  /** The held entitlement (expiry, source), when active. */
  entitlement?: MothEntitlement
}

/**
 * Whether the signed-in user holds `identifier` (e.g. `'pro'`) — the single
 * question app code should ask to gate a feature. Re-renders when the
 * entitlement flips, including at its expiry time (an expired cached
 * entitlement never keeps gating open).
 */
export function useMothEntitlement(
  identifier: string,
): UseMothEntitlementResult {
  const customerInfo = useMothCustomerInfo()
  const [expiredTick, setExpiredTick] = useState(0)
  const entitlement = customerInfo.entitlement(identifier)

  // Schedule a re-render at the entitlement's expiry so gating flips
  // without any server round-trip (the focus/return refetches then
  // confirm against the server).
  useEffect(() => {
    const expireTime = entitlement?.expireTime
    if (expireTime === undefined) return
    const delay = expireTime.getTime() - Date.now()
    if (delay <= 0) return
    // setTimeout clamps very large delays; cap at 24h. The tick is a
    // dependency so a capped firing re-runs this effect — `entitlement`
    // alone is reference-stable across the re-render — recomputing the
    // remaining delay and re-arming until the expiry actually passes; an
    // entitlement expiring more than 24h out still flips the gate on time.
    const timer = setTimeout(
      () => setExpiredTick((t) => t + 1),
      Math.min(delay + 50, 24 * 60 * 60 * 1000),
    )
    return () => clearTimeout(timer)
  }, [entitlement, expiredTick])

  const result: UseMothEntitlementResult = { active: entitlement !== undefined }
  if (entitlement !== undefined) result.entitlement = entitlement
  return result
}

export interface UseMothPushResult {
  /**
   * Where Web Push stands for this installation: `unavailable` (project has
   * no push / no VAPID key), `unsupported` (browser lacks the Push API),
   * `idle`, `subscribed`, or `denied`. Environment problems are states,
   * never exceptions.
   */
  status: MothPushStatus
  /** The browser notification permission (`'unknown'` until asked). */
  permission: MothPushPermission
  /**
   * Prompts for permission, subscribes the app's service worker's
   * `PushManager` and registers the device; resolves to the new status. A
   * typed no-op when `unavailable`/`unsupported` — it never throws for
   * environment reasons.
   */
  subscribe: () => Promise<MothPushStatus>
  /** Unsubscribes the browser subscription and revokes the registration. */
  unsubscribe: () => Promise<void>
}

/**
 * Web Push subscription state and actions — a settings-screen toggle in one
 * hook. Re-renders on every push status/permission change; while signed in
 * an existing subscription is re-registered on every launch, and sign-out
 * revokes it automatically. The app owns its service worker (display and
 * click handling); see the README for a minimal `sw.js`.
 */
export function useMothPush(): UseMothPushResult {
  const { pushController } = useMothContext()
  const [, setTick] = useState(0)
  useEffect(
    () => pushController.onChange(() => setTick((t) => t + 1)),
    [pushController],
  )
  return {
    status: pushController.status,
    permission: pushController.permission,
    subscribe: () => pushController.subscribe(),
    unsubscribe: () => pushController.unsubscribe(),
  }
}
