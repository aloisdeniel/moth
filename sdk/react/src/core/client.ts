import { createClient, type Client, type Transport } from '@connectrpc/connect'
import { timestampDate } from '@bufbuild/protobuf/wkt'
import {
  AuthService,
  OAuthProvider,
  type TokenPair,
  type User,
} from '../gen/moth/auth/v1/auth_pb.js'
import { ConfigService } from '../gen/moth/auth/v1/config_pb.js'
import { BillingService } from '../gen/moth/billing/v1/billing_pb.js'
import { currentLocaleOf, type MothConfig } from './config.js'
import { copyUpdateFromProto } from './copy.js'
import { MothCustomerInfo } from './customerInfo.js'
import {
  mapConnectError,
  MothError,
  MothInvalidAccessTokenError,
  MothInvalidRefreshTokenError,
  MothInvalidTokenError,
  MothRefreshTokenReusedError,
  MothUserDisabledError,
} from './errors.js'
import { customClaimsOf } from './jwt.js'
import {
  MothOffering,
  paywallFromProto,
  type MothOfferingProduct,
  type MothPaywall,
} from './offering.js'
import type { MothProjectConfig } from './projectConfig.js'
import { checkoutReturnParam, type MothPurchaseResult } from './purchase.js'
import { themeFromProto } from './theme.js'
import { createTokenStore, type StoredSession, type TokenStore } from './tokenStore.js'
import { createMothTransport, withMothHeaders } from './transport.js'
import { mothAuthLoading, mothSignedOut, type MothAuthState, type MothUser } from './user.js'

/** A social sign-in provider supported by moth. */
export type MothOAuthProvider = 'google' | 'apple'

/**
 * Result of {@link MothClient.signUp}. Depending on project policy the server
 * returns the user with tokens (signed in immediately), the user without
 * tokens (email verification required first) or nothing at all
 * (enumeration-safe projects).
 */
export interface MothSignUpResult {
  user?: MothUser
  /** True when sign-up also opened a session (tokens were returned). */
  signedIn: boolean
}

/** Constructor options beyond the config. */
export interface MothClientOptions {
  /**
   * Transport override (tests use `createRouterTransport`); defaults to the
   * gRPC-Web transport for `config.endpoint`. The moth metadata headers are
   * attached either way.
   */
  transport?: Transport
  /** Access tokens expiring within this window refresh proactively. */
  refreshSkewMs?: number
  /** Clock override for tests. */
  now?: () => number
  /** Checkout-return polling budget: attempts x interval. */
  checkoutPollAttempts?: number
  checkoutPollIntervalMs?: number
  /** Redirect override for tests; defaults to `window.location.assign`. */
  navigate?: (url: string) => void
}

type Unsubscribe = () => void

/**
 * Client for the moth.auth.v1 / moth.billing.v1 end-user API — the
 * framework-free core of `@moth/react` (the React layer is thin bindings on
 * top; a Vue or Svelte layer would reuse this class unchanged).
 *
 * Attaches the project's publishable key and — once signed in — a Bearer
 * access token to every call, persists the session in a {@link TokenStore}
 * and refreshes the access token automatically (single-flight, with
 * proactive skew).
 *
 * ```ts
 * const moth = new MothClient({
 *   endpoint: 'https://auth.example.com',
 *   publishableKey: 'pk_...',
 * })
 * await moth.restore() // loading -> signedIn | signedOut
 * ```
 */
export class MothClient {
  readonly config: MothConfig

  readonly #store: TokenStore
  readonly #refreshSkewMs: number
  readonly #now: () => number
  readonly #checkoutPollAttempts: number
  readonly #checkoutPollIntervalMs: number
  readonly #navigateFn: ((url: string) => void) | undefined

  readonly #auth: Client<typeof AuthService>
  readonly #projectConfig: Client<typeof ConfigService>
  readonly #billing: Client<typeof BillingService>

  #state: MothAuthState = mothAuthLoading
  #session: StoredSession | null = null
  #refreshing: Promise<string> | null = null

  /**
   * Bumped on every session transition — clear (signOut, rejected refresh)
   * AND start (signIn, signUp, OAuth exchange, changePassword) — so an
   * in-flight refresh that completes afterwards can tell the session it
   * started from is gone and must neither resurrect it (a stale success
   * would overwrite a newer session's tokens) nor clear the newer session
   * (a stale rejection).
   */
  #generation = 0

  #customerInfo: MothCustomerInfo = MothCustomerInfo.free()

  #stateListeners = new Set<(state: MothAuthState) => void>()
  #infoListeners = new Set<(info: MothCustomerInfo) => void>()

  constructor(config: MothConfig, options: MothClientOptions = {}) {
    this.config = config
    this.#store = createTokenStore(config.publishableKey, config.storage)
    this.#refreshSkewMs = options.refreshSkewMs ?? 30_000
    this.#now = options.now ?? (() => Date.now())
    this.#checkoutPollAttempts = options.checkoutPollAttempts ?? 5
    this.#checkoutPollIntervalMs = options.checkoutPollIntervalMs ?? 2_000
    this.#navigateFn = options.navigate
    const transport = withMothHeaders(
      options.transport ?? createMothTransport(config),
      config,
      () =>
        this.#session === null
          ? {}
          : { accessToken: this.#session.accessToken },
    )
    this.#auth = createClient(AuthService, transport)
    this.#projectConfig = createClient(ConfigService, transport)
    this.#billing = createClient(BillingService, transport)
  }

  // ---------------------------------------------------------------- state

  /** The current auth state (`loading` until {@link restore} completes). */
  get currentState(): MothAuthState {
    return this.#state
  }

  /** The signed-in user, or null. */
  get currentUser(): MothUser | null {
    return this.#state.status === 'signedIn' ? this.#state.user : null
  }

  /**
   * The locale the SDK negotiates copy for: `config.locale` when the app
   * pinned one, otherwise the live browser language.
   */
  get currentLocale(): string {
    return currentLocaleOf(this.config)
  }

  /**
   * Subscribes to auth state changes. The current state is REPLAYED to the
   * new subscriber immediately (synchronously), then every subsequent change
   * is delivered. Returns the unsubscribe function.
   */
  onAuthStateChanged(listener: (state: MothAuthState) => void): Unsubscribe {
    this.#stateListeners.add(listener)
    listener(this.#state)
    return () => this.#stateListeners.delete(listener)
  }

  // -------------------------------------------------------- entitlements

  /**
   * The signed-in user's current subscription state. Always valid — an
   * empty {@link MothCustomerInfo} (the free `none` tier) until the first
   * {@link getCustomerInfo}, and while signed out.
   */
  get currentCustomerInfo(): MothCustomerInfo {
    return this.#customerInfo
  }

  /**
   * Subscribes to subscription-state changes. Like
   * {@link onAuthStateChanged}, the current value is replayed to every new
   * subscriber. Returns the unsubscribe function.
   */
  onEntitlementsChanged(
    listener: (info: MothCustomerInfo) => void,
  ): Unsubscribe {
    this.#infoListeners.add(listener)
    listener(this.#customerInfo)
    return () => this.#infoListeners.delete(listener)
  }

  /**
   * Seeds {@link currentCustomerInfo} from a cache (stale-while-revalidate)
   * so subscribers reflect the last known entitlements before the first
   * {@link getCustomerInfo} lands. Deduplicated; the server stays
   * authoritative and overwrites on the next billing RPC.
   */
  primeCustomerInfo(info: MothCustomerInfo): void {
    this.#setCustomerInfo(info)
  }

  // -------------------------------------------------------------- session

  /**
   * Restores a persisted session from the token store. Call once at
   * startup; until it completes {@link currentState} is `loading`.
   *
   * A stored session whose access token is still fresh signs in without a
   * network round-trip. An expired one is refreshed; when the server
   * rejects the refresh token the session is cleared, while transient
   * network failures keep it (the next {@link accessToken} call retries).
   */
  async restore(): Promise<MothAuthState> {
    const generation = this.#generation
    let stored: StoredSession | null = null
    try {
      stored = await this.#store.load()
    } catch (err) {
      // A broken token store must never wedge startup on the loading
      // state: start signed out.
      this.#logStorageFailure('load', err)
    }
    // A session opened while the store was loading (or while the refresh
    // below was in flight) wins over the stored snapshot: never clobber it
    // with stale state.
    if (generation !== this.#generation) return this.#state
    if (stored === null) {
      this.#setState(mothSignedOut)
      return this.#state
    }
    this.#session = stored
    if (!this.#expiresSoon(stored)) {
      this.#setState({ status: 'signedIn', user: stored.user })
      return this.#state
    }
    try {
      await this.#refresh()
    } catch {
      // #refresh clears the session when the token was rejected; otherwise
      // (network failure) stay signed in on the stored snapshot. When the
      // generation moved, a concurrent transition (a fresh sign-in, a
      // clear) already owns the state — leave it alone.
      if (generation === this.#generation && this.#session !== null) {
        this.#setState({ status: 'signedIn', user: stored.user })
      }
    }
    return this.#state
  }

  /**
   * Returns a valid access token for the signed-in user, refreshing it
   * first when it expires within the refresh skew. Concurrent callers share
   * a single refresh RPC. Throws when signed out.
   */
  async accessToken(): Promise<string> {
    const session = this.#session
    if (session === null) throw new Error('moth: not signed in')
    if (!this.#expiresSoon(session)) return session.accessToken
    return this.#refresh()
  }

  /**
   * Forces a token refresh and returns the updated user. Throws when the
   * session ended (e.g. a concurrent {@link signOut}) before it completed.
   */
  async refresh(): Promise<MothUser> {
    await this.#refresh()
    const session = this.#session
    if (session === null) throw new Error('moth: not signed in')
    return session.user
  }

  // ----------------------------------------------------- email / password

  /** Registers a new email/password user, subject to project policy. */
  async signUp(params: {
    email: string
    password: string
    displayName?: string
    deviceInfo?: string
  }): Promise<MothSignUpResult> {
    return this.#run(async () => {
      const resp = await this.#auth.signUp({
        email: params.email,
        password: params.password,
        displayName: params.displayName ?? '',
        deviceInfo: params.deviceInfo ?? '',
      })
      if (resp.tokens !== undefined) {
        const user = await this.#openSession(resp.user, resp.tokens)
        return { user, signedIn: true }
      }
      const result: MothSignUpResult = { signedIn: false }
      if (resp.user !== undefined) result.user = userFromProto(resp.user, {})
      return result
    })
  }

  /** Exchanges email/password for a session. */
  async signIn(params: {
    email: string
    password: string
    deviceInfo?: string
  }): Promise<MothUser> {
    return this.#run(async () => {
      const resp = await this.#auth.signIn({
        email: params.email,
        password: params.password,
        deviceInfo: params.deviceInfo ?? '',
      })
      return this.#openSession(resp.user, resp.tokens)
    })
  }

  /**
   * Revokes the current session server-side (best effort — local sign-out
   * happens even when the revocation RPC fails) and clears the stored
   * session. With `allDevices` every session of the user is revoked.
   */
  async signOut(options: { allDevices?: boolean } = {}): Promise<void> {
    // An in-flight refresh must settle first: it may be rotating the
    // refresh token right now (revoke the current one, not a stale
    // predecessor) and, left running, it would re-open the session after
    // the sign-out cleared it.
    await this.#settleRefresh()
    const session = this.#session
    if (session === null) {
      this.#setState(mothSignedOut)
      return
    }
    try {
      await this.#auth.signOut({
        refreshToken: session.refreshToken,
        allDevices: options.allDevices ?? false,
      })
    } catch {
      // Best effort; the local session is cleared regardless.
    } finally {
      // A refresh kicked off while the RPC was in flight must not
      // resurrect the session either.
      await this.#settleRefresh()
      await this.#clearSession()
    }
  }

  /**
   * Changes the password (requires the current one). Every other session is
   * revoked; this device continues on a fresh token pair.
   */
  async changePassword(params: {
    currentPassword: string
    newPassword: string
  }): Promise<void> {
    await this.#authed(async () => {
      const resp = await this.#auth.changePassword(params)
      // The session may have ended (concurrent signOut) while the RPC was
      // in flight; don't resurrect it from the response.
      const session = this.#session
      if (session === null) throw new Error('moth: not signed in')
      if (resp.tokens === undefined) throw new Error('moth: no tokens')
      const user: MothUser = {
        ...session.user,
        claims: customClaimsOf(resp.tokens.accessToken),
      }
      await this.#startSession(resp.tokens, user)
    })
  }

  // ---------------------------------------------------------- social auth

  /**
   * Signs in (or up) with a provider ID token obtained from a Google/Apple
   * flow the app ran itself. `rawNonce`, `authorizationCode`, `givenName`
   * and `familyName` are Apple-only.
   */
  async signInWithOAuth(params: {
    provider: MothOAuthProvider
    idToken: string
    rawNonce?: string
    authorizationCode?: string
    givenName?: string
    familyName?: string
    deviceInfo?: string
  }): Promise<MothUser> {
    return this.#run(async () => {
      const resp = await this.#auth.signInWithOAuth({
        provider: providerToProto(params.provider),
        idToken: params.idToken,
        nonce: params.rawNonce ?? '',
        authorizationCode: params.authorizationCode ?? '',
        givenName: params.givenName ?? '',
        familyName: params.familyName ?? '',
        deviceInfo: params.deviceInfo ?? '',
      })
      return this.#openSession(resp.user, resp.tokens)
    })
  }

  /**
   * Trades the one-time `code` from the web-redirect OAuth flow
   * (`GET /oauth/{provider}/start` → consent → callback → redirect back
   * with `?code=...`) for a session.
   */
  async exchangeOAuthCode(
    code: string,
    options: { deviceInfo?: string } = {},
  ): Promise<MothUser> {
    return this.#run(async () => {
      const resp = await this.#auth.exchangeOAuthCode({
        code,
        deviceInfo: options.deviceInfo ?? '',
      })
      return this.#openSession(resp.user, resp.tokens)
    })
  }

  /**
   * The URL that starts the web-redirect OAuth flow for `provider`. The
   * browser should navigate there (`window.location.assign`); after consent
   * the server redirects to `redirectUri` — which must target a redirect
   * destination registered for the project: for a browser SPA, register
   * the app's origin in the admin (Providers → "Redirect origins (web)");
   * the exact origin is matched, any path works — with a one-time `code`
   * query parameter for {@link exchangeOAuthCode}. Requires
   * `config.projectSlug`.
   */
  oauthStartUrl(provider: MothOAuthProvider, redirectUri?: string): string {
    const slug = this.config.projectSlug
    if (slug === undefined || slug === '') {
      throw new Error(
        'moth: the web-redirect OAuth flow needs MothConfig.projectSlug',
      )
    }
    const url = new URL(
      `/oauth/${provider}/start`,
      this.config.endpoint,
    )
    url.searchParams.set('project', slug)
    if (redirectUri !== undefined && redirectUri !== '') {
      url.searchParams.set('redirect', redirectUri)
    }
    return url.toString()
  }

  /**
   * Starts the web-redirect OAuth flow: navigates the browser to
   * {@link oauthStartUrl}. `redirectUri` defaults to the current URL with
   * its fragment stripped — the server refuses redirect URIs containing
   * `#` (anything appended after it would land in the fragment instead of
   * the query), so hash-routed apps (e.g. HashRouter) return to the
   * fragment-less URL and should restore their route after
   * {@link exchangeOAuthCode}. Throws when `config.projectSlug` is unset.
   */
  signInWithRedirect(
    provider: MothOAuthProvider,
    options: { redirectUri?: string } = {},
  ): void {
    let redirect = options.redirectUri
    if (redirect === undefined && typeof window !== 'undefined') {
      const url = new URL(window.location.href)
      url.hash = ''
      redirect = url.toString()
    }
    this.#navigate(this.oauthStartUrl(provider, redirect))
  }

  /**
   * Removes the signed-in user's identity for `provider`. Refused with
   * `MothLastLoginMethodError` when it would leave no way to sign in.
   */
  async unlinkIdentity(provider: MothOAuthProvider): Promise<void> {
    await this.#authed(() =>
      this.#auth.unlinkIdentity({ provider: providerToProto(provider) }),
    )
  }

  // -------------------------------------------------------------- profile

  /** Fetches the signed-in user from the server and updates {@link currentUser}. */
  async getMe(): Promise<MothUser> {
    return this.#authed(async () => {
      const resp = await this.#auth.getMe({})
      return this.#updateUser(resp.user)
    })
  }

  /** Updates profile fields; only defined arguments are sent. */
  async updateMe(params: {
    displayName?: string
    avatarUrl?: string
  }): Promise<MothUser> {
    return this.#authed(async () => {
      // Only defined fields are sent (the proto fields carry presence, so
      // an omitted field leaves the profile value untouched).
      const req: { displayName?: string; avatarUrl?: string } = {}
      if (params.displayName !== undefined) req.displayName = params.displayName
      if (params.avatarUrl !== undefined) req.avatarUrl = params.avatarUrl
      const resp = await this.#auth.updateMe(req)
      return this.#updateUser(resp.user)
    })
  }

  /**
   * Permanently deletes the account after fresh re-authentication with
   * `password`, then clears the local session.
   */
  async deleteAccount(params: { password: string }): Promise<void> {
    await this.#authed(() =>
      this.#auth.deleteAccount({ password: params.password }),
    )
    // As in signOut: a refresh started while the RPC was in flight must
    // not re-open the (now deleted) session after it is cleared.
    await this.#settleRefresh()
    await this.#clearSession()
  }

  // ---------------------------------------------------------- email flows

  /** (Re)sends the verification email. Never reveals whether an account exists. */
  async requestEmailVerification(email: string): Promise<void> {
    await this.#run(() => this.#auth.requestEmailVerification({ email }))
  }

  /** Consumes a verification token from the email link. */
  async confirmEmailVerification(token: string): Promise<void> {
    await this.#run(() => this.#auth.confirmEmailVerification({ token }))
  }

  /** Emails a password-reset link. Never reveals whether an account exists. */
  async requestPasswordReset(email: string): Promise<void> {
    await this.#run(() => this.#auth.requestPasswordReset({ email }))
  }

  /** Consumes a reset token and sets the new password; every session is revoked. */
  async confirmPasswordReset(params: {
    token: string
    newPassword: string
  }): Promise<void> {
    await this.#run(() => this.#auth.confirmPasswordReset(params))
  }

  /** Sends a confirmation link to `newEmail`; the account switches once verified. */
  async requestEmailChange(newEmail: string): Promise<void> {
    await this.#authed(() => this.#auth.requestEmailChange({ newEmail }))
  }

  /** Consumes an email-change (or revert) token and applies the address. */
  async confirmEmailChange(token: string): Promise<void> {
    await this.#run(() => this.#auth.confirmEmailChange({ token }))
  }

  // --------------------------------------------------------------- config

  /**
   * Fetches the project's public configuration (enabled providers, password
   * policy, theme, localized copy). Pass the cached revisions as
   * `knownThemeRevision` / `knownCopyRevision`: when they still match, the
   * server omits the body and the corresponding field stays undefined (keep
   * the cached copy).
   */
  async getProjectConfig(
    options: { knownThemeRevision?: string; knownCopyRevision?: string } = {},
  ): Promise<MothProjectConfig> {
    return this.#run(async () => {
      const resp = await this.#projectConfig.getProjectConfig({
        knownThemeRevision: options.knownThemeRevision ?? '',
        knownCopyRevision: options.knownCopyRevision ?? '',
      })
      const blank = (s: string | undefined) =>
        s === undefined || s === '' ? undefined : s
      const google: MothProjectConfig['google'] = {
        enabled: resp.google?.enabled ?? false,
      }
      const webClientId = blank(resp.google?.webClientId)
      if (webClientId !== undefined) google.webClientId = webClientId
      const iosClientId = blank(resp.google?.iosClientId)
      if (iosClientId !== undefined) google.iosClientId = iosClientId
      const androidClientId = blank(resp.google?.androidClientId)
      if (androidClientId !== undefined) google.androidClientId = androidClientId
      const config: MothProjectConfig = {
        google,
        apple: { enabled: resp.apple?.enabled ?? false },
        passwordMinLength: resp.passwordMinLength,
        signUpOpen: resp.signUpOpen,
      }
      if (resp.theme !== undefined) config.theme = themeFromProto(resp.theme)
      if (resp.copy !== undefined) config.copy = copyUpdateFromProto(resp.copy)
      // The controllers cache the raw wire messages; hand them through.
      this.#lastRawConfig = resp
      return config
    })
  }

  /**
   * The raw wire messages of the last GetProjectConfig response, for the
   * config caches (they persist the payload exactly as delivered).
   */
  get lastRawProjectConfig():
    | Awaited<ReturnType<Client<typeof ConfigService>['getProjectConfig']>>
    | null {
    return this.#lastRawConfig
  }

  #lastRawConfig: Awaited<
    ReturnType<Client<typeof ConfigService>['getProjectConfig']>
  > | null = null

  // -------------------------------------------------------------- billing

  /**
   * Fetches the signed-in user's subscription state, updates
   * {@link currentCustomerInfo} and notifies subscribers. Cheap and safe to
   * call on every launch. Throws when signed out.
   */
  async getCustomerInfo(): Promise<MothCustomerInfo> {
    return this.#authed(async () => {
      // Capture whose session issues this RPC: a response landing after a
      // sign-out (or after a different user signed in) must not be
      // published as the current user's.
      const userId = this.#session?.user.id
      const resp = await this.#billing.getCustomerInfo({})
      return this.#applyCustomerInfo(resp.customerInfo, userId)
    })
  }

  /**
   * The products of `offering` (empty selects the project's default), for a
   * paywall to display. Publishable-key only — safe before sign-in.
   */
  async getOfferings(
    options: { offering?: string } = {},
  ): Promise<MothOffering> {
    return this.#run(async () => {
      const resp = await this.#billing.getOfferings({
        offering: options.offering ?? '',
      })
      return resp.offering !== undefined
        ? MothOffering.fromProto(resp.offering)
        : new MothOffering('', true, [])
    })
  }

  /**
   * The project's public paywall configuration, or null when
   * `knownPaywallRevision` still matches the current revision (keep the
   * cached copy). Publishable-key only — safe before sign-in.
   */
  async getPaywall(
    options: { knownPaywallRevision?: string } = {},
  ): Promise<MothPaywall | null> {
    return this.#run(async () => {
      const resp = await this.#billing.getPaywall({
        knownPaywallRevision: options.knownPaywallRevision ?? '',
      })
      this.#lastRawPaywall = resp.paywall ?? null
      return resp.paywall !== undefined ? paywallFromProto(resp.paywall) : null
    })
  }

  /** The raw wire message of the last non-empty GetPaywall response. */
  get lastRawPaywall() {
    return this.#lastRawPaywall
  }

  #lastRawPaywall: NonNullable<
    Awaited<ReturnType<Client<typeof BillingService>['getPaywall']>>['paywall']
  > | null = null

  /**
   * Creates a Stripe-hosted Checkout session for `productIdentifier` and
   * returns its URL. Prefer {@link purchase}, which also performs the
   * redirect and defaults the return URLs.
   */
  async createCheckoutSession(params: {
    productIdentifier: string
    successUrl: string
    cancelUrl: string
  }): Promise<string> {
    return this.#authed(async () => {
      const resp = await this.#billing.createCheckoutSession(params)
      return resp.url
    })
  }

  /** Creates a Stripe Billing Portal session and returns its URL. */
  async createBillingPortalSession(params: {
    returnUrl: string
  }): Promise<string> {
    return this.#authed(async () => {
      const resp = await this.#billing.createBillingPortalSession(params)
      return resp.url
    })
  }

  /**
   * Buys `product` through Stripe-hosted Checkout: creates the session and
   * redirects the browser to it. The success/cancel URLs default to the
   * current location with a `moth_checkout=success|cancel` query parameter,
   * which {@link handleCheckoutReturn} consumes on return. Expected
   * failures resolve as result values — this never throws.
   */
  async purchase(
    product: MothOfferingProduct | string,
    options: { successUrl?: string; cancelUrl?: string } = {},
  ): Promise<MothPurchaseResult> {
    const identifier =
      typeof product === 'string' ? product : product.identifier
    if (typeof product !== 'string' && product.stripePriceId === '') {
      // The tier does not ship on the web (no Stripe price). The paywall
      // renders such tiers disabled; this guards direct calls the same way
      // the server would, without a round-trip.
      return {
        status: 'error',
        message: 'this tier is not available for purchase on the web',
        reason: 'PRODUCT_NOT_ON_STORE',
      }
    }
    try {
      const url = await this.createCheckoutSession({
        productIdentifier: identifier,
        successUrl: options.successUrl ?? this.#returnUrl('success'),
        cancelUrl: options.cancelUrl ?? this.#returnUrl('cancel'),
      })
      this.#navigate(url)
      return { status: 'redirect' }
    } catch (err) {
      const mapped = err instanceof MothError ? err : mapConnectError(err)
      const result: MothPurchaseResult = {
        status: 'error',
        message: mapped.message,
      }
      if (mapped.reason !== undefined) result.reason = mapped.reason
      return result
    }
  }

  /**
   * Opens the Stripe Billing Portal (subscription management) by redirect.
   * `returnUrl` defaults to the current location. Throws typed errors
   * (e.g. `MothNoBillingHistoryError`) — callers surface them in UI.
   */
  async manageBilling(options: { returnUrl?: string } = {}): Promise<void> {
    const url = await this.createBillingPortalSession({
      returnUrl: options.returnUrl ?? this.#currentHref(),
    })
    this.#navigate(url)
  }

  /**
   * Detects a return from Stripe Checkout (the `moth_checkout` query
   * parameter {@link purchase} added to the return URLs), strips it from
   * the address bar, and — on success — re-fetches the customer info with a
   * short poll to absorb webhook latency. Call once on startup after
   * {@link restore}; `MothProvider` does this automatically.
   *
   * Returns null when the current URL carries no checkout-return marker.
   */
  async handleCheckoutReturn(): Promise<MothPurchaseResult | null> {
    if (typeof window === 'undefined') return null
    const url = new URL(window.location.href)
    const marker = url.searchParams.get(checkoutReturnParam)
    if (marker === null) return null
    url.searchParams.delete(checkoutReturnParam)
    window.history.replaceState(window.history.state, '', url.toString())
    if (marker !== 'success') return { status: 'cancelled' }
    if (this.#session === null) return { status: 'pending' }
    const before = this.#customerInfo
    for (let attempt = 0; attempt < this.#checkoutPollAttempts; attempt++) {
      if (attempt > 0) await this.#sleep(this.#checkoutPollIntervalMs)
      let info: MothCustomerInfo
      try {
        info = await this.getCustomerInfo()
      } catch {
        continue // transient failure: keep polling
      }
      if (
        info.activeEntitlements.length > 0 &&
        (before.activeEntitlements.length === 0 || !info.equals(before))
      ) {
        return { status: 'purchased' }
      }
    }
    // The webhook has not landed yet; entitlements flip when it does.
    return { status: 'pending' }
  }

  /**
   * Drops every subscription. Re-entrant: subscribing again afterwards
   * works (React StrictMode mounts effects twice), so this is a reset, not
   * a poison pill.
   */
  dispose(): void {
    this.#stateListeners.clear()
    this.#infoListeners.clear()
  }

  // ------------------------------------------------------------ internals

  /** Maps transport errors to the typed {@link MothError} hierarchy. */
  async #run<T>(fn: () => Promise<T>): Promise<T> {
    try {
      return await fn()
    } catch (err) {
      throw mapConnectError(err)
    }
  }

  /**
   * Ensures a fresh Bearer token is attached, then runs `fn`. When the
   * server nonetheless rejects the access token (the client-computed expiry
   * drifted — machine slept mid-call, TTL shortened server-side), the call
   * refreshes reactively and retries exactly once instead of surfacing
   * "session expired" to a session whose refresh token is still valid.
   */
  async #authed<T>(fn: () => Promise<T>): Promise<T> {
    await this.accessToken()
    try {
      return await this.#run(fn)
    } catch (err) {
      if (!(err instanceof MothInvalidAccessTokenError)) throw err
      if (this.#session === null) throw err
      await this.#refresh()
      return this.#run(fn)
    }
  }

  #expiresSoon(session: StoredSession): boolean {
    return session.expiresAtMs <= this.#now() + this.#refreshSkewMs
  }

  #refresh(): Promise<string> {
    const inflight = this.#refreshing
    if (inflight !== null) return inflight
    const refresh = this.#doRefresh().finally(() => {
      this.#refreshing = null
    })
    this.#refreshing = refresh
    return refresh
  }

  /** Waits for an in-flight refresh to settle, ignoring its outcome. */
  async #settleRefresh(): Promise<void> {
    const inflight = this.#refreshing
    if (inflight === null) return
    try {
      await inflight
    } catch {
      // Handled by the refresh's own callers.
    }
  }

  async #doRefresh(): Promise<string> {
    const session = this.#session
    if (session === null) throw new Error('moth: not signed in')
    const generation = this.#generation
    let resp
    try {
      resp = await this.#auth.refreshToken({
        refreshToken: session.refreshToken,
      })
    } catch (err) {
      const mapped = mapConnectError(err)
      // A rejected refresh token means the session is gone (rotated-out,
      // revoked or stolen): end up signed out with storage cleared.
      // Transient failures (network, server errors) keep the session. When
      // the session already ended concurrently there is nothing left to
      // clear (and a newer session must not be clobbered).
      if (
        generation === this.#generation &&
        (mapped instanceof MothInvalidRefreshTokenError ||
          mapped instanceof MothRefreshTokenReusedError ||
          mapped instanceof MothInvalidTokenError ||
          mapped instanceof MothUserDisabledError)
      ) {
        await this.#clearSession()
      }
      throw mapped
    }
    // The session ended (signOut/deleteAccount) while the RPC was in
    // flight: committing the fresh tokens would silently sign the user
    // back in.
    if (generation !== this.#generation) {
      throw new Error('moth: signed out during token refresh')
    }
    if (resp.tokens === undefined) {
      throw new MothError('moth: refresh returned no tokens')
    }
    await this.#openSession(resp.user, resp.tokens)
    return resp.tokens.accessToken
  }

  async #openSession(
    user: User | undefined,
    tokens: TokenPair | undefined,
  ): Promise<MothUser> {
    if (user === undefined || tokens === undefined) {
      throw new MothError('moth: response missing user or tokens')
    }
    return this.#startSession(
      tokens,
      userFromProto(user, customClaimsOf(tokens.accessToken)),
    )
  }

  async #startSession(tokens: TokenPair, user: MothUser): Promise<MothUser> {
    // Any session transition invalidates in-flight refreshes: a refresh
    // started under the previous session must neither clear this one (on a
    // stale rejection) nor overwrite its tokens (on a stale success).
    this.#generation++
    const session: StoredSession = {
      accessToken: tokens.accessToken,
      refreshToken: tokens.refreshToken,
      expiresAtMs: this.#now() + Number(tokens.expiresIn) * 1000,
      user,
    }
    this.#session = session
    await this.#persist(session)
    this.#setState({ status: 'signedIn', user })
    return user
  }

  /** Replaces the cached user snapshot after GetMe/UpdateMe. */
  async #updateUser(user: User | undefined): Promise<MothUser> {
    if (user === undefined) throw new MothError('moth: response missing user')
    const session = this.#session
    const claims = session === null ? {} : session.user.claims
    const mothUser = userFromProto(user, claims)
    if (session !== null) {
      const updated: StoredSession = { ...session, user: mothUser }
      this.#session = updated
      await this.#persist(updated)
      this.#setState({ status: 'signedIn', user: mothUser })
    }
    return mothUser
  }

  /**
   * Saves the session, swallowing storage failures: the in-memory session
   * is fully usable (the server accepted the credentials) — it just won't
   * survive a reload. Throwing here would fail a sign-in that actually
   * succeeded.
   */
  async #persist(session: StoredSession): Promise<void> {
    try {
      await this.#store.save(session)
    } catch (err) {
      this.#logStorageFailure('save', err)
    }
  }

  async #clearSession(): Promise<void> {
    this.#session = null
    this.#generation++
    try {
      await this.#store.clear()
    } catch (err) {
      // Sign-out must complete locally even when storage misbehaves.
      this.#logStorageFailure('clear', err)
    }
    // Emit the signed-out auth state BEFORE the free customer-info reset:
    // the subscription controller listens to both, and it must observe the
    // sign-out (which drops its user id) before the free snapshot arrives,
    // so it does not persist the empty snapshot over the outgoing user's
    // cached entitlements (breaking instant gating when they sign back in).
    this.#setState(mothSignedOut)
    this.#setCustomerInfo(MothCustomerInfo.free())
  }

  // Publishes a fresh CustomerInfo from a billing RPC. Always emits — even
  // when it equals the last value — so a stale-while-revalidate cache that
  // rendered a cached snapshot still learns the confirmed server truth.
  //
  // `issuedForUserId` is the signed-in user id captured when the RPC was
  // issued: a response that outlived its session (sign-out, or another user
  // signed in meanwhile) is DROPPED — publishing it would hand user A's
  // entitlements to user B, and the subscription controller would persist
  // them under B's cache key. The current (correct) value is returned
  // instead so callers still get a truthful answer for the active session.
  #applyCustomerInfo(
    proto: Parameters<typeof MothCustomerInfo.fromProto>[0] | undefined,
    issuedForUserId: string | undefined,
  ): MothCustomerInfo {
    if (
      issuedForUserId === undefined ||
      this.#session?.user.id !== issuedForUserId
    ) {
      return this.#customerInfo
    }
    const info =
      proto === undefined
        ? MothCustomerInfo.free()
        : MothCustomerInfo.fromProto(proto)
    this.#customerInfo = info
    this.#emitCustomerInfo(info)
    return info
  }

  // Resets to a value on a lifecycle change (sign-out); deduplicated, so a
  // no-op reset does not churn listeners.
  #setCustomerInfo(info: MothCustomerInfo): void {
    if (info.equals(this.#customerInfo)) return
    this.#customerInfo = info
    this.#emitCustomerInfo(info)
  }

  #emitCustomerInfo(info: MothCustomerInfo): void {
    for (const listener of [...this.#infoListeners]) listener(info)
  }

  #setState(state: MothAuthState): void {
    this.#state = state
    for (const listener of [...this.#stateListeners]) listener(state)
  }

  #logStorageFailure(op: string, err: unknown): void {
    if (typeof console !== 'undefined') {
      console.warn(`moth: token store ${op} failed:`, err)
    }
  }

  #returnUrl(outcome: 'success' | 'cancel'): string {
    const url = new URL(this.#currentHref())
    url.searchParams.set(checkoutReturnParam, outcome)
    return url.toString()
  }

  #currentHref(): string {
    if (typeof window === 'undefined') return this.config.endpoint
    return window.location.href
  }

  #navigate(url: string): void {
    if (this.#navigateFn !== undefined) {
      this.#navigateFn(url)
      return
    }
    if (typeof window === 'undefined') return
    window.location.assign(url)
  }

  #sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms))
  }
}

function userFromProto(user: User, claims: Record<string, unknown>): MothUser {
  const mothUser: MothUser = {
    id: user.id,
    email: user.email,
    emailVerified: user.emailVerified,
    claims,
  }
  if (user.displayName !== '') mothUser.displayName = user.displayName
  if (user.avatarUrl !== '') mothUser.avatarUrl = user.avatarUrl
  if (user.createTime !== undefined) {
    mothUser.createTime = timestampDate(user.createTime)
  }
  return mothUser
}

function providerToProto(provider: MothOAuthProvider): OAuthProvider {
  return provider === 'google'
    ? OAuthProvider.OAUTH_PROVIDER_GOOGLE
    : OAuthProvider.OAUTH_PROVIDER_APPLE
}
