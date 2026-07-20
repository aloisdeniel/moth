import { Fragment as _Fragment, jsx as _jsx } from "react/jsx-runtime";
import { useEffect, useState } from 'react';
import { loadPaywall } from '../core/paywallLoader.js';
import { MothSurface, useMothContext } from './context.js';
import { useMothEntitlement } from './hooks.js';
import { MothPaywallScreen } from './MothPaywallScreen.js';
/**
 * Per-client cache of resolved "does the paywall's offering sell this
 * entitlement" verdicts. A successful resolution is cached for the client's
 * lifetime (the catalog does not change within a session), so remounting a
 * gate costs zero RPCs; a failed resolution is evicted immediately, so the
 * next attempt (backoff retry or remount) asks again instead of latching
 * the failure.
 */
const gateVerdicts = new WeakMap();
function resolveGateBlocks(client, entitlement) {
    let verdicts = gateVerdicts.get(client);
    if (verdicts === undefined) {
        verdicts = new Map();
        gateVerdicts.set(client, verdicts);
    }
    const cached = verdicts.get(entitlement);
    if (cached !== undefined)
        return cached;
    const map = verdicts;
    const promise = (async () => {
        // Resolve the SAME offering the paywall will present, not the default
        // one: the paywall config can point at a non-default offering, and the
        // gated entitlement may be granted only by products there. loadPaywall
        // shares the paywall screen's own revision-cached blob, so a warm
        // cache costs at most the single offerings RPC.
        const paywall = await loadPaywall(client);
        const offering = await client.getOfferings({
            offering: paywall.offering,
        });
        return offering.grants(entitlement);
    })();
    map.set(entitlement, promise);
    promise.catch(() => map.delete(entitlement));
    return promise;
}
const retryBaseDelayMs = 3000;
const retryMaxDelayMs = 30000;
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
export function MothGate(props) {
    const { client } = useMothContext();
    const { active } = useMothEntitlement(props.entitlement);
    const [resolution, setResolution] = useState('resolving');
    useEffect(() => {
        if (active)
            return; // nothing to resolve while the entitlement is held
        let cancelled = false;
        let timer;
        let delay = retryBaseDelayMs;
        const attempt = () => {
            resolveGateBlocks(client, props.entitlement).then((blocks) => {
                if (!cancelled)
                    setResolution(blocks ? 'blocks' : 'open');
            }, () => {
                if (cancelled)
                    return;
                // Couldn't load the catalog: show the paywall (it has its own
                // retry/empty state) — but the failed verdict was evicted, so
                // keep retrying with backoff until the offering answers; a
                // project that sells nothing for this entitlement then falls
                // through to children per the never-block rule.
                setResolution('blocks');
                timer = setTimeout(attempt, delay);
                delay = Math.min(delay * 2, retryMaxDelayMs);
            });
        };
        attempt();
        return () => {
            cancelled = true;
            if (timer !== undefined)
                clearTimeout(timer);
        };
    }, [client, props.entitlement, active]);
    if (active)
        return _jsx(_Fragment, { children: props.children });
    if (resolution === 'resolving') {
        return (_jsx(MothSurface, { children: _jsx("div", { className: "moth-screen", children: _jsx("div", { className: "moth-spinner", role: "progressbar", "aria-label": "Loading" }) }) }));
    }
    if (resolution === 'open')
        return _jsx(_Fragment, { children: props.children });
    // A custom fallback is the app's own UI — rendered bare; only the
    // default paywall is a moth-owned surface (theme scope + stylesheet).
    if (props.fallback !== undefined)
        return _jsx(_Fragment, { children: props.fallback });
    return (_jsx(MothSurface, { children: _jsx(MothPaywallScreen, {}) }));
}
