import { PushPermission, PushTarget } from '../gen/moth/push/v1/push_pb.js';
import { base64Decode } from './cache.js';
/** Maps an SDK push target to the wire enum. */
export function pushTargetToProto(target) {
    switch (target) {
        case 'apns':
            return PushTarget.APNS;
        case 'fcm':
            return PushTarget.FCM;
        case 'webpush':
            return PushTarget.WEBPUSH;
    }
}
/** Maps an SDK permission state to the wire enum. */
export function pushPermissionToProto(permission) {
    switch (permission) {
        case 'granted':
            return PushPermission.GRANTED;
        case 'provisional':
            return PushPermission.PROVISIONAL;
        case 'denied':
            return PushPermission.DENIED;
        case 'unknown':
            return PushPermission.UNKNOWN;
    }
}
/**
 * Maps the browser's `Notification.permission` to the SDK permission state
 * (`'default'` — not asked yet — maps to `'unknown'`).
 */
export function pushPermissionFromNotification(permission) {
    switch (permission) {
        case 'granted':
            return 'granted';
        case 'denied':
            return 'denied';
        default:
            return 'unknown';
    }
}
/**
 * Decodes a VAPID public key (base64url, as stored in the project config)
 * into the `applicationServerKey` bytes `PushManager.subscribe` expects.
 */
export function vapidKeyBytes(key) {
    const base64 = key.replace(/-/g, '+').replace(/_/g, '/');
    return base64Decode(base64 + '='.repeat((4 - (base64.length % 4)) % 4));
}
