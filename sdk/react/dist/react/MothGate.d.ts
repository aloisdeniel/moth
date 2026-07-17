import { type ReactNode } from 'react';
export interface MothGateProps {
    /** The entitlement (e.g. `'pro'`) that unlocks `children`. */
    entitlement: string;
    /** Shown while the user lacks it; defaults to `<MothPaywallScreen />`. */
    fallback?: ReactNode;
    children: ReactNode;
}
/**
 * Gates `children` behind an entitlement:
 *
 * ```tsx
 * <MothGate entitlement="pro">
 *   <ProFeatures />
 * </MothGate>
 * ```
 *
 * A user who holds the entitlement sees `children` — and flips to them
 * instantly the moment it arrives (checkout return, background refresh),
 * with zero catalog RPCs. A user who lacks it sees the `fallback` paywall,
 * with one crucial nuance: the gate resolves the **paywall's own offering**
 * first, and when no product there grants the entitlement it falls through
 * to `children` — never block when there is nothing to sell, so a project
 * with no billing configured still runs the whole auth story. The verdict
 * is cached per client, so remounts are free. A catalog-load failure shows
 * the paywall (which has its own retry/empty states) but never latches:
 * the gate retries with backoff (and afresh on remount) until the offering
 * answers, then applies the fall-through rule.
 */
export declare function MothGate(props: MothGateProps): import("react").JSX.Element;
