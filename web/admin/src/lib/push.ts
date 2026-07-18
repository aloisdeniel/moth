import { PushPermission, PushRevokeReason, PushTarget } from "../gen/moth/admin/v1/push_pb";

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
