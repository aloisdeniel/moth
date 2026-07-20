import { Store, SubscriptionStatus } from "../gen/moth/admin/v1/subscription_pb";

// storeLabel renders the app-store enum for display.
export function storeLabel(store: Store): string {
  switch (store) {
    case Store.APPLE:
      return "Apple";
    case Store.GOOGLE:
      return "Google";
    case Store.STRIPE:
      return "Stripe";
    default:
      return "—";
  }
}

type Tone = "neutral" | "success" | "warning" | "danger";

// subscriptionStatusMeta maps a store status to a human label and a Badge
// tone. The tone follows the entitlement-derivation matrix (plan/11): the
// statuses that keep access are success/warning, the ones that revoke it are
// neutral/danger — never a functional color used decoratively.
export function subscriptionStatusMeta(s: SubscriptionStatus): { label: string; tone: Tone } {
  switch (s) {
    case SubscriptionStatus.ACTIVE:
      return { label: "Active", tone: "success" };
    case SubscriptionStatus.TRIALING:
      return { label: "Trialing", tone: "success" };
    case SubscriptionStatus.IN_GRACE_PERIOD:
      return { label: "Grace period", tone: "warning" };
    case SubscriptionStatus.IN_BILLING_RETRY:
      return { label: "Billing retry", tone: "warning" };
    case SubscriptionStatus.PAUSED:
      return { label: "Paused", tone: "neutral" };
    case SubscriptionStatus.EXPIRED:
      return { label: "Expired", tone: "neutral" };
    case SubscriptionStatus.REVOKED:
      return { label: "Revoked", tone: "danger" };
    default:
      return { label: "Unknown", tone: "neutral" };
  }
}

// A subscription in one of these statuses grants its product's entitlements
// (grace & billing-retry keep access, matching store policy).
export function statusGrantsAccess(s: SubscriptionStatus): boolean {
  return (
    s === SubscriptionStatus.ACTIVE ||
    s === SubscriptionStatus.TRIALING ||
    s === SubscriptionStatus.IN_GRACE_PERIOD ||
    s === SubscriptionStatus.IN_BILLING_RETRY
  );
}

// formatPrice renders store price metadata (micros = price * 1_000_000).
export function formatPrice(micros: bigint, currency: string): string {
  if (micros === 0n) return "—";
  const amount = Number(micros) / 1_000_000;
  return `${amount.toFixed(2)}${currency ? ` ${currency}` : ""}`;
}

// formatMoney renders a store-reported revenue amount (micros) in its own
// currency — always per currency, never blended. `fractionDigits` lets axis
// ticks drop the cents while tooltips/tiles keep them. Falls back to a plain
// "12.00 USD" when the ISO code is unknown to Intl.
export function formatMoney(micros: bigint, currency: string, fractionDigits = 2): string {
  const amount = Number(micros) / 1_000_000;
  if (currency) {
    try {
      return new Intl.NumberFormat(undefined, {
        style: "currency",
        currency,
        maximumFractionDigits: fractionDigits,
      }).format(amount);
    } catch {
      // Unknown ISO-4217 code — fall through to the plain form.
    }
  }
  return `${amount.toFixed(fractionDigits)}${currency ? ` ${currency}` : ""}`;
}
