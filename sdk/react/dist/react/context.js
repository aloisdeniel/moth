import { jsx as _jsx, Fragment as _Fragment } from "react/jsx-runtime";
import { createContext, useContext, useEffect, useInsertionEffect, useMemo, useState, } from 'react';
import { MothClient } from '../core/client.js';
import { MothConfigController } from '../core/configController.js';
import { MothSubscriptionController } from '../core/subscriptionController.js';
import { ensureThemeFont, themeCssVars } from '../core/theme.js';
import { MothLoginScreen } from './MothLoginScreen.js';
import { ensureMothStyles } from './styles.js';
const MothContext = createContext(null);
/** The provider's context value; throws outside a `MothProvider`. */
export function useMothContext() {
    const value = useContext(MothContext);
    if (value === null) {
        throw new Error('moth: this component must be rendered under <MothProvider>');
    }
    return value;
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
export function MothProvider(props) {
    const { config, client: externalClient, requireAuth = true } = props;
    if ((config === undefined) === (externalClient === undefined)) {
        throw new Error('moth: provide exactly one of config or client');
    }
    // The client and controllers are created once, for the lifetime of the
    // provider (config/client are fixed, as on MothApp in Flutter).
    const [owned] = useState(() => {
        const client = externalClient ?? new MothClient(config);
        return {
            client,
            ownsClient: externalClient === undefined,
            configController: new MothConfigController(client),
            subscriptions: new MothSubscriptionController(client),
        };
    });
    const { client, configController } = owned;
    const [state, setState] = useState(client.currentState);
    const [customerInfo, setCustomerInfo] = useState(client.currentCustomerInfo);
    const [, setConfigTick] = useState(0);
    useEffect(() => {
        const unsubscribers = [
            client.onAuthStateChanged(setState),
            client.onEntitlementsChanged(setCustomerInfo),
            configController.subscribe(() => setConfigTick((t) => t + 1)),
        ];
        owned.subscriptions.start();
        void configController.start();
        if (client.currentState.status === 'loading') {
            // Failures surface through the state stream (restore keeps or clears
            // the session itself), then the checkout-return marker is consumed.
            void client.restore().then(() => client.handleCheckoutReturn());
        }
        else {
            void client.handleCheckoutReturn();
        }
        // Refetch entitlements when the tab regains focus (checkout in another
        // tab, admin grants, subscription lapses).
        const onFocus = () => {
            if (client.currentState.status === 'signedIn') {
                client.getCustomerInfo().catch(() => undefined);
            }
        };
        window.addEventListener('focus', onFocus);
        // A browser-language change reloads that locale's cached copy floor and
        // refetches (no-op when MothConfig.locale pins a language).
        const onLanguageChange = () => void configController.refresh();
        window.addEventListener('languagechange', onLanguageChange);
        return () => {
            window.removeEventListener('focus', onFocus);
            window.removeEventListener('languagechange', onLanguageChange);
            for (const unsubscribe of unsubscribers)
                unsubscribe();
            owned.subscriptions.dispose();
            configController.dispose();
            if (owned.ownsClient)
                client.dispose();
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [owned]);
    const value = useMemo(() => ({ client, state, customerInfo, configController }), [client, state, customerInfo, configController]);
    let body;
    let ownSurface = false;
    if (requireAuth) {
        switch (state.status) {
            case 'loading':
                body = props.loadingFallback ?? _jsx(MothSplash, {});
                ownSurface = true;
                break;
            case 'signedOut':
                body = props.signedOut ?? _jsx(MothLoginScreen, {});
                ownSurface = true;
                break;
            case 'signedIn':
                body = props.children;
                break;
        }
    }
    else {
        body = props.children;
    }
    if (ownSurface) {
        body = _jsx(MothSurface, { children: body });
    }
    return (_jsx(MothContext.Provider, { value: value, children: _jsx(MothRemount, { children: body }, ownSurface ? 'moth' : 'app') }));
}
function MothRemount(props) {
    return _jsx(_Fragment, { children: props.children });
}
/**
 * Wraps a moth-owned surface: injects the (idempotent) stylesheet and
 * renders the `.moth-root` scope carrying the theme's CSS custom
 * properties — light and dark palettes side by side, resolved by the
 * stylesheet per `prefers-color-scheme`. Never rendered around the app's
 * own children.
 */
export function MothSurface(props) {
    const { configController } = useMothContext();
    const theme = configController.theme;
    useInsertionEffect(() => {
        ensureMothStyles();
    }, []);
    useEffect(() => {
        void ensureThemeFont(theme);
    }, [theme]);
    return (_jsx("div", { className: "moth-root", style: themeCssVars(theme), children: props.children }));
}
/** The theme for the current moth surface. */
export function useMothTheme() {
    return useMothContext().configController.theme;
}
/** The localized copy for the current moth surface. */
export function useMothCopy() {
    return useMothContext().configController.copy;
}
function MothSplash() {
    return (_jsx("div", { className: "moth-screen", children: _jsx("div", { className: "moth-spinner", role: "progressbar", "aria-label": "Loading" }) }));
}
