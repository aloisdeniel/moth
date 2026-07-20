import { PushPermission, PushTarget } from '../gen/moth/push/v1/push_pb.js'
import { base64Decode } from './cache.js'

/** Which push service a registered credential belongs to. */
export type MothPushTarget = 'apns' | 'fcm' | 'webpush'

/** The OS/browser-level notification-permission state the SDK observed. */
export type MothPushPermission =
  | 'granted'
  | 'provisional'
  | 'denied'
  | 'unknown'

/**
 * Where `useMothPush` / `MothPushController` stand:
 *
 * - `unavailable` — the project has push disabled or no VAPID public key
 *   (or the config has not arrived yet); `subscribe()` is a typed no-op.
 * - `unsupported` — the browser lacks the Web Push API (feature-detected,
 *   never user-agent sniffed); `subscribe()` is a typed no-op.
 * - `idle` — ready; `subscribe()` will prompt and register.
 * - `subscribed` — this installation is registered with the moth registry
 *   for the signed-in user.
 * - `denied` — the user denied the browser notification permission.
 */
export type MothPushStatus =
  | 'unavailable'
  | 'unsupported'
  | 'idle'
  | 'subscribed'
  | 'denied'

/** Display metadata stored alongside a registration (all fields optional). */
export interface MothPushDeviceMetadata {
  /** OS family (`'web'`, `'ios'`, ...) — display only. */
  platform?: string
  model?: string
  osVersion?: string
  appVersion?: string
  /** BCP-47 locale of the device, for sender-side locale targeting. */
  locale?: string
}

/** Maps an SDK push target to the wire enum. */
export function pushTargetToProto(target: MothPushTarget): PushTarget {
  switch (target) {
    case 'apns':
      return PushTarget.APNS
    case 'fcm':
      return PushTarget.FCM
    case 'webpush':
      return PushTarget.WEBPUSH
  }
}

/** Maps an SDK permission state to the wire enum. */
export function pushPermissionToProto(
  permission: MothPushPermission,
): PushPermission {
  switch (permission) {
    case 'granted':
      return PushPermission.GRANTED
    case 'provisional':
      return PushPermission.PROVISIONAL
    case 'denied':
      return PushPermission.DENIED
    case 'unknown':
      return PushPermission.UNKNOWN
  }
}

/**
 * Maps the browser's `Notification.permission` to the SDK permission state
 * (`'default'` — not asked yet — maps to `'unknown'`).
 */
export function pushPermissionFromNotification(
  permission: NotificationPermission,
): MothPushPermission {
  switch (permission) {
    case 'granted':
      return 'granted'
    case 'denied':
      return 'denied'
    default:
      return 'unknown'
  }
}

/**
 * Decodes a VAPID public key (base64url, as stored in the project config)
 * into the `applicationServerKey` bytes `PushManager.subscribe` expects.
 */
export function vapidKeyBytes(key: string): Uint8Array {
  const base64 = key.replace(/-/g, '+').replace(/_/g, '/')
  return base64Decode(base64 + '='.repeat((4 - (base64.length % 4)) % 4))
}
