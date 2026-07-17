import { type ReactNode } from 'react';
import { MothClient } from '../core/client.js';
import type { MothConfig } from '../core/config.js';
import { MothConfigController } from '../core/configController.js';
import type { MothCopy } from '../core/copy.js';
import { MothCustomerInfo } from '../core/customerInfo.js';
import { type MothTheme } from '../core/theme.js';
import type { MothAuthState } from '../core/user.js';
export interface MothContextValue {
    client: MothClient;
    state: MothAuthState;
    customerInfo: MothCustomerInfo;
    configController: MothConfigController;
}
/** The provider's context value; throws outside a `MothProvider`. */
export declare function useMothContext(): MothContextValue;
export interface MothProviderProps {
    /** Connection settings; the provider creates (and owns) the client. */
    config?: MothConfig;
    /** An externally owned client (e.g. one also used outside React). */
    client?: MothClient;
    /** Shown while the session restore is in flight. */
    loadingFallback?: ReactNode;
    /** Shown while signed out; defaults to `<MothLoginScreen />`. */
    signedOut?: ReactNode;
    /** When false, children render regardless of auth state. */
    requireAuth?: boolean;
    children: ReactNode;
}
/**
 * Top-level component that owns a {@link MothClient} and gates `children`
 * behind authentication:
 *
 * ```tsx
 * <MothProvider config={{ endpoint: 'https://auth.example.com', publishableKey: 'pk_...' }}>
 *   <App />
 * </MothProvider>
 * ```
 *
 * On mount it restores the persisted session, then renders per state:
 * `loading` → `loadingFallback` (default: a spinner), `signedOut` →
 * `signedOut` (default: {@link MothLoginScreen}), `signedIn` → `children`.
 * With `requireAuth={false}` children always render and read the state via
 * the hooks, which are available below this component either way.
 *
 * The surfaces the provider owns (loading and signed-out) render with the
 * project's admin-configured theme and localized copy, cached
 * download-once/stale-while-revalidate. The app's own subtree is
 * deliberately left alone — the moth theme only ever applies to moth
 * surfaces. Flipping between a moth-owned surface and the app remounts the
 * subtree (a fresh `key`), so app state never survives a sign-out
 * underneath the login screen.
 */
export declare function MothProvider(props: MothProviderProps): import("react").JSX.Element;
/**
 * Wraps a moth-owned surface: injects the (idempotent) stylesheet and
 * renders the `.moth-root` scope carrying the theme's CSS custom
 * properties — light and dark palettes side by side, resolved by the
 * stylesheet per `prefers-color-scheme`. Never rendered around the app's
 * own children.
 */
export declare function MothSurface(props: {
    children: ReactNode;
}): import("react").JSX.Element;
/** The theme for the current moth surface. */
export declare function useMothTheme(): MothTheme;
/** The localized copy for the current moth surface. */
export declare function useMothCopy(): MothCopy;
