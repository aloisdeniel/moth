import {
  createContext,
  useContext,
  useEffect,
  useInsertionEffect,
  useMemo,
  useState,
  type CSSProperties,
  type ReactNode,
} from 'react'
import { MothClient } from '../core/client.js'
import type { MothConfig } from '../core/config.js'
import { MothConfigController } from '../core/configController.js'
import type { MothCopy } from '../core/copy.js'
import { MothCustomerInfo } from '../core/customerInfo.js'
import { MothPushController } from '../core/pushController.js'
import { MothSubscriptionController } from '../core/subscriptionController.js'
import { ensureThemeFont, themeCssVars, type MothTheme } from '../core/theme.js'
import type { MothAuthState } from '../core/user.js'
import { MothLoginScreen } from './MothLoginScreen.js'
import { ensureMothStyles } from './styles.js'

export interface MothContextValue {
  client: MothClient
  state: MothAuthState
  customerInfo: MothCustomerInfo
  configController: MothConfigController
  pushController: MothPushController
}

const MothContext = createContext<MothContextValue | null>(null)

/** The provider's context value; throws outside a `MothProvider`. */
export function useMothContext(): MothContextValue {
  const value = useContext(MothContext)
  if (value === null) {
    throw new Error('moth: this component must be rendered under <MothProvider>')
  }
  return value
}

export interface MothProviderProps {
  /** Connection settings; the provider creates (and owns) the client. */
  config?: MothConfig
  /** An externally owned client (e.g. one also used outside React). */
  client?: MothClient
  /** Shown while the session restore is in flight. */
  loadingFallback?: ReactNode
  /** Shown while signed out; defaults to `<MothLoginScreen />`. */
  signedOut?: ReactNode
  /** When false, children render regardless of auth state. */
  requireAuth?: boolean
  children: ReactNode
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
export function MothProvider(props: MothProviderProps) {
  const { config, client: externalClient, requireAuth = true } = props
  if ((config === undefined) === (externalClient === undefined)) {
    throw new Error('moth: provide exactly one of config or client')
  }

  // The client and controllers are created once, for the lifetime of the
  // provider (config/client are fixed, as on MothApp in Flutter).
  const [owned] = useState(() => {
    const client = externalClient ?? new MothClient(config!)
    const configController = new MothConfigController(client)
    return {
      client,
      ownsClient: externalClient === undefined,
      configController,
      subscriptions: new MothSubscriptionController(client),
      pushController: new MothPushController(client, configController),
    }
  })
  const { client, configController, pushController } = owned

  const [state, setState] = useState<MothAuthState>(client.currentState)
  const [customerInfo, setCustomerInfo] = useState<MothCustomerInfo>(
    client.currentCustomerInfo,
  )
  const [, setConfigTick] = useState(0)

  useEffect(() => {
    const unsubscribers = [
      client.onAuthStateChanged(setState),
      client.onEntitlementsChanged(setCustomerInfo),
      configController.subscribe(() => setConfigTick((t) => t + 1)),
    ]
    owned.subscriptions.start()
    // Before restore(): the push controller's sign-in listener must see the
    // restored session's transition so a signed-in launch re-registers an
    // existing subscription (upsert semantics keep this carefree).
    pushController.start()
    void configController.start()
    if (client.currentState.status === 'loading') {
      // Failures surface through the state stream (restore keeps or clears
      // the session itself), then the checkout-return marker is consumed.
      void client.restore().then(() => client.handleCheckoutReturn())
    } else {
      void client.handleCheckoutReturn()
    }
    // Refetch entitlements when the tab regains focus (checkout in another
    // tab, admin grants, subscription lapses).
    const onFocus = () => {
      if (client.currentState.status === 'signedIn') {
        client.getCustomerInfo().catch(() => undefined)
      }
    }
    window.addEventListener('focus', onFocus)
    // A browser-language change reloads that locale's cached copy floor and
    // refetches (no-op when MothConfig.locale pins a language).
    const onLanguageChange = () => void configController.refresh()
    window.addEventListener('languagechange', onLanguageChange)
    return () => {
      window.removeEventListener('focus', onFocus)
      window.removeEventListener('languagechange', onLanguageChange)
      for (const unsubscribe of unsubscribers) unsubscribe()
      owned.subscriptions.dispose()
      pushController.dispose()
      configController.dispose()
      if (owned.ownsClient) client.dispose()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [owned])

  const value = useMemo<MothContextValue>(
    () => ({ client, state, customerInfo, configController, pushController }),
    [client, state, customerInfo, configController, pushController],
  )

  let body: ReactNode
  let ownSurface = false
  if (requireAuth) {
    switch (state.status) {
      case 'loading':
        body = props.loadingFallback ?? <MothSplash />
        ownSurface = true
        break
      case 'signedOut':
        body = props.signedOut ?? <MothLoginScreen />
        ownSurface = true
        break
      case 'signedIn':
        body = props.children
        break
    }
  } else {
    body = props.children
  }
  if (ownSurface) {
    body = <MothSurface>{body}</MothSurface>
  }
  return (
    <MothContext.Provider value={value}>
      {/* Distinct keys per side of the gate: a flip must fully remount the
          subtree, never update it in place — otherwise the app's state
          (open dialogs, routers) would survive a sign-out underneath the
          login screen. */}
      <MothRemount key={ownSurface ? 'moth' : 'app'}>{body}</MothRemount>
    </MothContext.Provider>
  )
}

function MothRemount(props: { children: ReactNode }) {
  return <>{props.children}</>
}

/**
 * Wraps a moth-owned surface: injects the (idempotent) stylesheet and
 * renders the `.moth-root` scope carrying the theme's CSS custom
 * properties — light and dark palettes side by side, resolved by the
 * stylesheet per `prefers-color-scheme`. Never rendered around the app's
 * own children.
 */
export function MothSurface(props: { children: ReactNode }) {
  const { configController } = useMothContext()
  const theme = configController.theme
  useInsertionEffect(() => {
    ensureMothStyles()
  }, [])
  useEffect(() => {
    void ensureThemeFont(theme)
  }, [theme])
  return (
    <div className="moth-root" style={themeCssVars(theme) as CSSProperties}>
      {props.children}
    </div>
  )
}

/** The theme for the current moth surface. */
export function useMothTheme(): MothTheme {
  return useMothContext().configController.theme
}

/** The localized copy for the current moth surface. */
export function useMothCopy(): MothCopy {
  return useMothContext().configController.copy
}

function MothSplash() {
  return (
    <div className="moth-screen">
      <div className="moth-spinner" role="progressbar" aria-label="Loading" />
    </div>
  )
}
