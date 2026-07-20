import { PushPermission, PushRevokeReason, PushTarget } from "../gen/moth/admin/v1/push_pb";

// vapidKeyError mirrors the server-side shape check so a typo surfaces
// before the save round-trip: base64url (no padding) decoding to an
// uncompressed P-256 public point (65 bytes starting 0x04). Empty is valid —
// the project simply does not use Web Push. Shared by the Settings tab's
// push section and the creation wizard's push step.
export function vapidKeyError(key: string): string {
  if (key === "") return "";
  if (!/^[A-Za-z0-9_-]+$/.test(key)) {
    return "Must be base64url without padding (A–Z a–z 0–9 - _).";
  }
  let raw: string;
  try {
    raw = atob(key.replace(/-/g, "+").replace(/_/g, "/"));
  } catch {
    return "Not valid base64url.";
  }
  if (raw.length !== 65 || raw.charCodeAt(0) !== 0x04) {
    return "Not an uncompressed P-256 public key (expected 65 bytes starting with 0x04).";
  }
  return "";
}

// pushTargetLabel renders the push-target enum for display: which service
// the credential belongs to (which API the sender must call), not the OS.
export function pushTargetLabel(t: PushTarget): string {
  switch (t) {
    case PushTarget.APNS:
      return "APNs";
    case PushTarget.FCM:
      return "FCM";
    case PushTarget.WEBPUSH:
      return "Web Push";
    default:
      return "—";
  }
}

type Tone = "neutral" | "success" | "warning" | "danger";

// pushPermissionMeta maps the OS notification-permission state to a label
// and Badge tone: granted delivers alerts (success), provisional delivers
// quietly (warning), denied keeps only data pushes working (danger).
export function pushPermissionMeta(p: PushPermission): { label: string; tone: Tone } {
  switch (p) {
    case PushPermission.GRANTED:
      return { label: "Granted", tone: "success" };
    case PushPermission.PROVISIONAL:
      return { label: "Provisional", tone: "warning" };
    case PushPermission.DENIED:
      return { label: "Denied", tone: "danger" };
    default:
      return { label: "Unknown", tone: "neutral" };
  }
}

// pushRevokeReasonLabel renders why a registration was revoked.
export function pushRevokeReasonLabel(r: PushRevokeReason): string {
  switch (r) {
    case PushRevokeReason.SIGNED_OUT:
      return "signed out";
    case PushRevokeReason.REPORTED_INVALID:
      return "reported invalid";
    case PushRevokeReason.STALE:
      return "stale";
    case PushRevokeReason.REPLACED:
      return "replaced";
    case PushRevokeReason.ADMIN:
      return "by admin";
    default:
      return "";
  }
}
