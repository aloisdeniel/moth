import type { PaywallConfig } from "../gen/moth/admin/v1/paywall_pb";
import { PaywallLayout } from "../gen/moth/admin/v1/paywall_pb";

// Client-side helpers for the admin paywall editor (milestone 13). The
// paywall owns no design tokens of its own — colors, typography, spacing,
// radius and logo inherit from the theme (see lib/theme). This module only
// mirrors the copy/layout config the SDK's MothPaywallScreen renders. The
// server re-validates everything on save.

// LAYOUT_OPTIONS lists the selectable layout variants in editor order; the
// UNSPECIFIED zero value is never offered (TILES is the default).
export const LAYOUT_OPTIONS: { value: PaywallLayout; label: string; help: string }[] = [
  { value: PaywallLayout.TILES, label: "Tiles", help: "One card per tier, side by side." },
  { value: PaywallLayout.LIST, label: "List", help: "Tiers stacked as full-width rows." },
  { value: PaywallLayout.COMPACT, label: "Compact", help: "A single selected tier." },
];

// Bounds mirror internal/paywall validation so the editor can warn live.
export const MAX_HEADLINE = 80;
export const MAX_SUBTITLE = 160;
export const MAX_BENEFITS = 8;
export const MAX_BENEFIT_LENGTH = 120;

// EditorPaywall is the tab's working copy of a paywall config: a plain
// object so React state updates stay structural.
export type EditorPaywall = {
  headline: string;
  subtitle: string;
  benefits: string[];
  offering: string;
  highlightedProductIdentifier: string;
  layout: PaywallLayout;
  termsUrl: string;
  privacyUrl: string;
};

// DEFAULT_PAYWALL is only used when the server response is missing a config
// message, which GetPaywallConfig never does in practice (it returns the
// built-in defaults with is_default=true).
const DEFAULT_PAYWALL: EditorPaywall = {
  headline: "Unlock everything",
  subtitle: "Go further with a subscription.",
  benefits: [],
  offering: "",
  highlightedProductIdentifier: "",
  layout: PaywallLayout.TILES,
  termsUrl: "",
  privacyUrl: "",
};

// paywallFromProto seeds the working copy from a GetPaywallConfig (or
// restore/reset) response.
export function paywallFromProto(c: PaywallConfig | undefined): EditorPaywall {
  if (!c) return { ...DEFAULT_PAYWALL, benefits: [] };
  return {
    headline: c.headline,
    subtitle: c.subtitle,
    benefits: [...c.benefits],
    offering: c.offering,
    highlightedProductIdentifier: c.highlightedProductIdentifier,
    layout: c.layout === PaywallLayout.UNSPECIFIED ? PaywallLayout.TILES : c.layout,
    termsUrl: c.legal?.termsUrl ?? "",
    privacyUrl: c.legal?.privacyUrl ?? "",
  };
}

// paywallToProto builds the UpdatePaywallConfig payload (a plain object
// matching PaywallConfig; connect encodes it against the schema).
export function paywallToProto(e: EditorPaywall) {
  return {
    headline: e.headline.trim(),
    subtitle: e.subtitle.trim(),
    benefits: e.benefits.map((b) => b.trim()).filter((b) => b !== ""),
    offering: e.offering.trim(),
    highlightedProductIdentifier: e.highlightedProductIdentifier.trim(),
    layout: e.layout,
    legal: { termsUrl: e.termsUrl.trim(), privacyUrl: e.privacyUrl.trim() },
  };
}
