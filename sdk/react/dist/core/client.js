var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var _MothClient_instances, _MothClient_store, _MothClient_refreshSkewMs, _MothClient_now, _MothClient_checkoutPollAttempts, _MothClient_checkoutPollIntervalMs, _MothClient_navigateFn, _MothClient_auth, _MothClient_projectConfig, _MothClient_billing, _MothClient_state, _MothClient_session, _MothClient_refreshing, _MothClient_generation, _MothClient_customerInfo, _MothClient_stateListeners, _MothClient_infoListeners, _MothClient_lastRawConfig, _MothClient_lastRawPaywall, _MothClient_run, _MothClient_authed, _MothClient_expiresSoon, _MothClient_refresh, _MothClient_settleRefresh, _MothClient_doRefresh, _MothClient_openSession, _MothClient_startSession, _MothClient_updateUser, _MothClient_persist, _MothClient_clearSession, _MothClient_applyCustomerInfo, _MothClient_setCustomerInfo, _MothClient_emitCustomerInfo, _MothClient_setState, _MothClient_logStorageFailure, _MothClient_returnUrl, _MothClient_currentHref, _MothClient_navigate, _MothClient_sleep;
import { createClient } from '@connectrpc/connect';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import { AuthService, OAuthProvider, } from '../gen/moth/auth/v1/auth_pb.js';
import { ConfigService } from '../gen/moth/auth/v1/config_pb.js';
import { BillingService } from '../gen/moth/billing/v1/billing_pb.js';
import { currentLocaleOf } from './config.js';
import { copyUpdateFromProto } from './copy.js';
import { MothCustomerInfo } from './customerInfo.js';
import { mapConnectError, MothError, MothInvalidAccessTokenError, MothInvalidRefreshTokenError, MothInvalidTokenError, MothRefreshTokenReusedError, MothUserDisabledError, } from './errors.js';
import { customClaimsOf } from './jwt.js';
import { MothOffering, paywallFromProto, } from './offering.js';
import { checkoutReturnParam } from './purchase.js';
import { themeFromProto } from './theme.js';
import { createTokenStore } from './tokenStore.js';
import { createMothTransport, withMothHeaders } from './transport.js';
import { mothAuthLoading, mothSignedOut } from './user.js';
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
    constructor(config, options = {}) {
        _MothClient_instances.add(this);
        _MothClient_store.set(this, void 0);
        _MothClient_refreshSkewMs.set(this, void 0);
        _MothClient_now.set(this, void 0);
        _MothClient_checkoutPollAttempts.set(this, void 0);
        _MothClient_checkoutPollIntervalMs.set(this, void 0);
        _MothClient_navigateFn.set(this, void 0);
        _MothClient_auth.set(this, void 0);
        _MothClient_projectConfig.set(this, void 0);
        _MothClient_billing.set(this, void 0);
        _MothClient_state.set(this, mothAuthLoading);
        _MothClient_session.set(this, null);
        _MothClient_refreshing.set(this, null
        /**
         * Bumped on every session transition — clear (signOut, rejected refresh)
         * AND start (signIn, signUp, OAuth exchange, changePassword) — so an
         * in-flight refresh that completes afterwards can tell the session it
         * started from is gone and must neither resurrect it (a stale success
         * would overwrite a newer session's tokens) nor clear the newer session
         * (a stale rejection).
         */
        );
        /**
         * Bumped on every session transition — clear (signOut, rejected refresh)
         * AND start (signIn, signUp, OAuth exchange, changePassword) — so an
         * in-flight refresh that completes afterwards can tell the session it
         * started from is gone and must neither resurrect it (a stale success
         * would overwrite a newer session's tokens) nor clear the newer session
         * (a stale rejection).
         */
        _MothClient_generation.set(this, 0);
        _MothClient_customerInfo.set(this, MothCustomerInfo.free());
        _MothClient_stateListeners.set(this, new Set());
        _MothClient_infoListeners.set(this, new Set());
        _MothClient_lastRawConfig.set(this, null
        // -------------------------------------------------------------- billing
        /**
         * Fetches the signed-in user's subscription state, updates
         * {@link currentCustomerInfo} and notifies subscribers. Cheap and safe to
         * call on every launch. Throws when signed out.
         */
        );
        _MothClient_lastRawPaywall.set(this, null
        /**
         * Creates a Stripe-hosted Checkout session for `productIdentifier` and
         * returns its URL. Prefer {@link purchase}, which also performs the
         * redirect and defaults the return URLs.
         */
        );
        this.config = config;
        __classPrivateFieldSet(this, _MothClient_store, createTokenStore(config.publishableKey, config.storage), "f");
        __classPrivateFieldSet(this, _MothClient_refreshSkewMs, options.refreshSkewMs ?? 30000, "f");
        __classPrivateFieldSet(this, _MothClient_now, options.now ?? (() => Date.now()), "f");
        __classPrivateFieldSet(this, _MothClient_checkoutPollAttempts, options.checkoutPollAttempts ?? 5, "f");
        __classPrivateFieldSet(this, _MothClient_checkoutPollIntervalMs, options.checkoutPollIntervalMs ?? 2000, "f");
        __classPrivateFieldSet(this, _MothClient_navigateFn, options.navigate, "f");
        const transport = withMothHeaders(options.transport ?? createMothTransport(config), config, () => __classPrivateFieldGet(this, _MothClient_session, "f") === null
            ? {}
            : { accessToken: __classPrivateFieldGet(this, _MothClient_session, "f").accessToken });
        __classPrivateFieldSet(this, _MothClient_auth, createClient(AuthService, transport), "f");
        __classPrivateFieldSet(this, _MothClient_projectConfig, createClient(ConfigService, transport), "f");
        __classPrivateFieldSet(this, _MothClient_billing, createClient(BillingService, transport), "f");
    }
    // ---------------------------------------------------------------- state
    /** The current auth state (`loading` until {@link restore} completes). */
    get currentState() {
        return __classPrivateFieldGet(this, _MothClient_state, "f");
    }
    /** The signed-in user, or null. */
    get currentUser() {
        return __classPrivateFieldGet(this, _MothClient_state, "f").status === 'signedIn' ? __classPrivateFieldGet(this, _MothClient_state, "f").user : null;
    }
    /**
     * The locale the SDK negotiates copy for: `config.locale` when the app
     * pinned one, otherwise the live browser language.
     */
    get currentLocale() {
        return currentLocaleOf(this.config);
    }
    /**
     * Subscribes to auth state changes. The current state is REPLAYED to the
     * new subscriber immediately (synchronously), then every subsequent change
     * is delivered. Returns the unsubscribe function.
     */
    onAuthStateChanged(listener) {
        __classPrivateFieldGet(this, _MothClient_stateListeners, "f").add(listener);
        listener(__classPrivateFieldGet(this, _MothClient_state, "f"));
        return () => __classPrivateFieldGet(this, _MothClient_stateListeners, "f").delete(listener);
    }
    // -------------------------------------------------------- entitlements
    /**
     * The signed-in user's current subscription state. Always valid — an
     * empty {@link MothCustomerInfo} (the free `none` tier) until the first
     * {@link getCustomerInfo}, and while signed out.
     */
    get currentCustomerInfo() {
        return __classPrivateFieldGet(this, _MothClient_customerInfo, "f");
    }
    /**
     * Subscribes to subscription-state changes. Like
     * {@link onAuthStateChanged}, the current value is replayed to every new
     * subscriber. Returns the unsubscribe function.
     */
    onEntitlementsChanged(listener) {
        __classPrivateFieldGet(this, _MothClient_infoListeners, "f").add(listener);
        listener(__classPrivateFieldGet(this, _MothClient_customerInfo, "f"));
        return () => __classPrivateFieldGet(this, _MothClient_infoListeners, "f").delete(listener);
    }
    /**
     * Seeds {@link currentCustomerInfo} from a cache (stale-while-revalidate)
     * so subscribers reflect the last known entitlements before the first
     * {@link getCustomerInfo} lands. Deduplicated; the server stays
     * authoritative and overwrites on the next billing RPC.
     */
    primeCustomerInfo(info) {
        __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_setCustomerInfo).call(this, info);
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
    async restore() {
        const generation = __classPrivateFieldGet(this, _MothClient_generation, "f");
        let stored = null;
        try {
            stored = await __classPrivateFieldGet(this, _MothClient_store, "f").load();
        }
        catch (err) {
            // A broken token store must never wedge startup on the loading
            // state: start signed out.
            __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_logStorageFailure).call(this, 'load', err);
        }
        // A session opened while the store was loading (or while the refresh
        // below was in flight) wins over the stored snapshot: never clobber it
        // with stale state.
        if (generation !== __classPrivateFieldGet(this, _MothClient_generation, "f"))
            return __classPrivateFieldGet(this, _MothClient_state, "f");
        if (stored === null) {
            __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_setState).call(this, mothSignedOut);
            return __classPrivateFieldGet(this, _MothClient_state, "f");
        }
        __classPrivateFieldSet(this, _MothClient_session, stored, "f");
        if (!__classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_expiresSoon).call(this, stored)) {
            __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_setState).call(this, { status: 'signedIn', user: stored.user });
            return __classPrivateFieldGet(this, _MothClient_state, "f");
        }
        try {
            await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_refresh).call(this);
        }
        catch {
            // #refresh clears the session when the token was rejected; otherwise
            // (network failure) stay signed in on the stored snapshot. When the
            // generation moved, a concurrent transition (a fresh sign-in, a
            // clear) already owns the state — leave it alone.
            if (generation === __classPrivateFieldGet(this, _MothClient_generation, "f") && __classPrivateFieldGet(this, _MothClient_session, "f") !== null) {
                __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_setState).call(this, { status: 'signedIn', user: stored.user });
            }
        }
        return __classPrivateFieldGet(this, _MothClient_state, "f");
    }
    /**
     * Returns a valid access token for the signed-in user, refreshing it
     * first when it expires within the refresh skew. Concurrent callers share
     * a single refresh RPC. Throws when signed out.
     */
    async accessToken() {
        const session = __classPrivateFieldGet(this, _MothClient_session, "f");
        if (session === null)
            throw new Error('moth: not signed in');
        if (!__classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_expiresSoon).call(this, session))
            return session.accessToken;
        return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_refresh).call(this);
    }
    /**
     * Forces a token refresh and returns the updated user. Throws when the
     * session ended (e.g. a concurrent {@link signOut}) before it completed.
     */
    async refresh() {
        await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_refresh).call(this);
        const session = __classPrivateFieldGet(this, _MothClient_session, "f");
        if (session === null)
            throw new Error('moth: not signed in');
        return session.user;
    }
    // ----------------------------------------------------- email / password
    /** Registers a new email/password user, subject to project policy. */
    async signUp(params) {
        return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_run).call(this, async () => {
            const resp = await __classPrivateFieldGet(this, _MothClient_auth, "f").signUp({
                email: params.email,
                password: params.password,
                displayName: params.displayName ?? '',
                deviceInfo: params.deviceInfo ?? '',
            });
            if (resp.tokens !== undefined) {
                const user = await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_openSession).call(this, resp.user, resp.tokens);
                return { user, signedIn: true };
            }
            const result = { signedIn: false };
            if (resp.user !== undefined)
                result.user = userFromProto(resp.user, {});
            return result;
        });
    }
    /** Exchanges email/password for a session. */
    async signIn(params) {
        return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_run).call(this, async () => {
            const resp = await __classPrivateFieldGet(this, _MothClient_auth, "f").signIn({
                email: params.email,
                password: params.password,
                deviceInfo: params.deviceInfo ?? '',
            });
            return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_openSession).call(this, resp.user, resp.tokens);
        });
    }
    /**
     * Revokes the current session server-side (best effort — local sign-out
     * happens even when the revocation RPC fails) and clears the stored
     * session. With `allDevices` every session of the user is revoked.
     */
    async signOut(options = {}) {
        // An in-flight refresh must settle first: it may be rotating the
        // refresh token right now (revoke the current one, not a stale
        // predecessor) and, left running, it would re-open the session after
        // the sign-out cleared it.
        await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_settleRefresh).call(this);
        const session = __classPrivateFieldGet(this, _MothClient_session, "f");
        if (session === null) {
            __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_setState).call(this, mothSignedOut);
            return;
        }
        try {
            await __classPrivateFieldGet(this, _MothClient_auth, "f").signOut({
                refreshToken: session.refreshToken,
                allDevices: options.allDevices ?? false,
            });
        }
        catch {
            // Best effort; the local session is cleared regardless.
        }
        finally {
            // A refresh kicked off while the RPC was in flight must not
            // resurrect the session either.
            await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_settleRefresh).call(this);
            await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_clearSession).call(this);
        }
    }
    /**
     * Changes the password (requires the current one). Every other session is
     * revoked; this device continues on a fresh token pair.
     */
    async changePassword(params) {
        await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_authed).call(this, async () => {
            const resp = await __classPrivateFieldGet(this, _MothClient_auth, "f").changePassword(params);
            // The session may have ended (concurrent signOut) while the RPC was
            // in flight; don't resurrect it from the response.
            const session = __classPrivateFieldGet(this, _MothClient_session, "f");
            if (session === null)
                throw new Error('moth: not signed in');
            if (resp.tokens === undefined)
                throw new Error('moth: no tokens');
            const user = {
                ...session.user,
                claims: customClaimsOf(resp.tokens.accessToken),
            };
            await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_startSession).call(this, resp.tokens, user);
        });
    }
    // ---------------------------------------------------------- social auth
    /**
     * Signs in (or up) with a provider ID token obtained from a Google/Apple
     * flow the app ran itself. `rawNonce`, `authorizationCode`, `givenName`
     * and `familyName` are Apple-only.
     */
    async signInWithOAuth(params) {
        return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_run).call(this, async () => {
            const resp = await __classPrivateFieldGet(this, _MothClient_auth, "f").signInWithOAuth({
                provider: providerToProto(params.provider),
                idToken: params.idToken,
                nonce: params.rawNonce ?? '',
                authorizationCode: params.authorizationCode ?? '',
                givenName: params.givenName ?? '',
                familyName: params.familyName ?? '',
                deviceInfo: params.deviceInfo ?? '',
            });
            return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_openSession).call(this, resp.user, resp.tokens);
        });
    }
    /**
     * Trades the one-time `code` from the web-redirect OAuth flow
     * (`GET /oauth/{provider}/start` → consent → callback → redirect back
     * with `?code=...`) for a session.
     */
    async exchangeOAuthCode(code, options = {}) {
        return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_run).call(this, async () => {
            const resp = await __classPrivateFieldGet(this, _MothClient_auth, "f").exchangeOAuthCode({
                code,
                deviceInfo: options.deviceInfo ?? '',
            });
            return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_openSession).call(this, resp.user, resp.tokens);
        });
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
    oauthStartUrl(provider, redirectUri) {
        const slug = this.config.projectSlug;
        if (slug === undefined || slug === '') {
            throw new Error('moth: the web-redirect OAuth flow needs MothConfig.projectSlug');
        }
        const url = new URL(`/oauth/${provider}/start`, this.config.endpoint);
        url.searchParams.set('project', slug);
        if (redirectUri !== undefined && redirectUri !== '') {
            url.searchParams.set('redirect', redirectUri);
        }
        return url.toString();
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
    signInWithRedirect(provider, options = {}) {
        let redirect = options.redirectUri;
        if (redirect === undefined && typeof window !== 'undefined') {
            const url = new URL(window.location.href);
            url.hash = '';
            redirect = url.toString();
        }
        __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_navigate).call(this, this.oauthStartUrl(provider, redirect));
    }
    /**
     * Removes the signed-in user's identity for `provider`. Refused with
     * `MothLastLoginMethodError` when it would leave no way to sign in.
     */
    async unlinkIdentity(provider) {
        await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_authed).call(this, () => __classPrivateFieldGet(this, _MothClient_auth, "f").unlinkIdentity({ provider: providerToProto(provider) }));
    }
    // -------------------------------------------------------------- profile
    /** Fetches the signed-in user from the server and updates {@link currentUser}. */
    async getMe() {
        return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_authed).call(this, async () => {
            const resp = await __classPrivateFieldGet(this, _MothClient_auth, "f").getMe({});
            return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_updateUser).call(this, resp.user);
        });
    }
    /** Updates profile fields; only defined arguments are sent. */
    async updateMe(params) {
        return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_authed).call(this, async () => {
            // Only defined fields are sent (the proto fields carry presence, so
            // an omitted field leaves the profile value untouched).
            const req = {};
            if (params.displayName !== undefined)
                req.displayName = params.displayName;
            if (params.avatarUrl !== undefined)
                req.avatarUrl = params.avatarUrl;
            const resp = await __classPrivateFieldGet(this, _MothClient_auth, "f").updateMe(req);
            return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_updateUser).call(this, resp.user);
        });
    }
    /**
     * Permanently deletes the account after fresh re-authentication with
     * `password`, then clears the local session.
     */
    async deleteAccount(params) {
        await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_authed).call(this, () => __classPrivateFieldGet(this, _MothClient_auth, "f").deleteAccount({ password: params.password }));
        // As in signOut: a refresh started while the RPC was in flight must
        // not re-open the (now deleted) session after it is cleared.
        await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_settleRefresh).call(this);
        await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_clearSession).call(this);
    }
    // ---------------------------------------------------------- email flows
    /** (Re)sends the verification email. Never reveals whether an account exists. */
    async requestEmailVerification(email) {
        await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_run).call(this, () => __classPrivateFieldGet(this, _MothClient_auth, "f").requestEmailVerification({ email }));
    }
    /** Consumes a verification token from the email link. */
    async confirmEmailVerification(token) {
        await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_run).call(this, () => __classPrivateFieldGet(this, _MothClient_auth, "f").confirmEmailVerification({ token }));
    }
    /** Emails a password-reset link. Never reveals whether an account exists. */
    async requestPasswordReset(email) {
        await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_run).call(this, () => __classPrivateFieldGet(this, _MothClient_auth, "f").requestPasswordReset({ email }));
    }
    /** Consumes a reset token and sets the new password; every session is revoked. */
    async confirmPasswordReset(params) {
        await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_run).call(this, () => __classPrivateFieldGet(this, _MothClient_auth, "f").confirmPasswordReset(params));
    }
    /** Sends a confirmation link to `newEmail`; the account switches once verified. */
    async requestEmailChange(newEmail) {
        await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_authed).call(this, () => __classPrivateFieldGet(this, _MothClient_auth, "f").requestEmailChange({ newEmail }));
    }
    /** Consumes an email-change (or revert) token and applies the address. */
    async confirmEmailChange(token) {
        await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_run).call(this, () => __classPrivateFieldGet(this, _MothClient_auth, "f").confirmEmailChange({ token }));
    }
    // --------------------------------------------------------------- config
    /**
     * Fetches the project's public configuration (enabled providers, password
     * policy, theme, localized copy). Pass the cached revisions as
     * `knownThemeRevision` / `knownCopyRevision`: when they still match, the
     * server omits the body and the corresponding field stays undefined (keep
     * the cached copy).
     */
    async getProjectConfig(options = {}) {
        return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_run).call(this, async () => {
            const resp = await __classPrivateFieldGet(this, _MothClient_projectConfig, "f").getProjectConfig({
                knownThemeRevision: options.knownThemeRevision ?? '',
                knownCopyRevision: options.knownCopyRevision ?? '',
            });
            const blank = (s) => s === undefined || s === '' ? undefined : s;
            const google = {
                enabled: resp.google?.enabled ?? false,
            };
            const webClientId = blank(resp.google?.webClientId);
            if (webClientId !== undefined)
                google.webClientId = webClientId;
            const iosClientId = blank(resp.google?.iosClientId);
            if (iosClientId !== undefined)
                google.iosClientId = iosClientId;
            const androidClientId = blank(resp.google?.androidClientId);
            if (androidClientId !== undefined)
                google.androidClientId = androidClientId;
            const config = {
                google,
                apple: { enabled: resp.apple?.enabled ?? false },
                passwordMinLength: resp.passwordMinLength,
                signUpOpen: resp.signUpOpen,
            };
            if (resp.theme !== undefined)
                config.theme = themeFromProto(resp.theme);
            if (resp.copy !== undefined)
                config.copy = copyUpdateFromProto(resp.copy);
            // The controllers cache the raw wire messages; hand them through.
            __classPrivateFieldSet(this, _MothClient_lastRawConfig, resp, "f");
            return config;
        });
    }
    /**
     * The raw wire messages of the last GetProjectConfig response, for the
     * config caches (they persist the payload exactly as delivered).
     */
    get lastRawProjectConfig() {
        return __classPrivateFieldGet(this, _MothClient_lastRawConfig, "f");
    }
    // -------------------------------------------------------------- billing
    /**
     * Fetches the signed-in user's subscription state, updates
     * {@link currentCustomerInfo} and notifies subscribers. Cheap and safe to
     * call on every launch. Throws when signed out.
     */
    async getCustomerInfo() {
        return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_authed).call(this, async () => {
            // Capture whose session issues this RPC: a response landing after a
            // sign-out (or after a different user signed in) must not be
            // published as the current user's.
            const userId = __classPrivateFieldGet(this, _MothClient_session, "f")?.user.id;
            const resp = await __classPrivateFieldGet(this, _MothClient_billing, "f").getCustomerInfo({});
            return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_applyCustomerInfo).call(this, resp.customerInfo, userId);
        });
    }
    /**
     * The products of `offering` (empty selects the project's default), for a
     * paywall to display. Publishable-key only — safe before sign-in.
     */
    async getOfferings(options = {}) {
        return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_run).call(this, async () => {
            const resp = await __classPrivateFieldGet(this, _MothClient_billing, "f").getOfferings({
                offering: options.offering ?? '',
            });
            return resp.offering !== undefined
                ? MothOffering.fromProto(resp.offering)
                : new MothOffering('', true, []);
        });
    }
    /**
     * The project's public paywall configuration, or null when
     * `knownPaywallRevision` still matches the current revision (keep the
     * cached copy). Publishable-key only — safe before sign-in.
     */
    async getPaywall(options = {}) {
        return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_run).call(this, async () => {
            const resp = await __classPrivateFieldGet(this, _MothClient_billing, "f").getPaywall({
                knownPaywallRevision: options.knownPaywallRevision ?? '',
            });
            __classPrivateFieldSet(this, _MothClient_lastRawPaywall, resp.paywall ?? null, "f");
            return resp.paywall !== undefined ? paywallFromProto(resp.paywall) : null;
        });
    }
    /** The raw wire message of the last non-empty GetPaywall response. */
    get lastRawPaywall() {
        return __classPrivateFieldGet(this, _MothClient_lastRawPaywall, "f");
    }
    /**
     * Creates a Stripe-hosted Checkout session for `productIdentifier` and
     * returns its URL. Prefer {@link purchase}, which also performs the
     * redirect and defaults the return URLs.
     */
    async createCheckoutSession(params) {
        return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_authed).call(this, async () => {
            const resp = await __classPrivateFieldGet(this, _MothClient_billing, "f").createCheckoutSession(params);
            return resp.url;
        });
    }
    /** Creates a Stripe Billing Portal session and returns its URL. */
    async createBillingPortalSession(params) {
        return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_authed).call(this, async () => {
            const resp = await __classPrivateFieldGet(this, _MothClient_billing, "f").createBillingPortalSession(params);
            return resp.url;
        });
    }
    /**
     * Buys `product` through Stripe-hosted Checkout: creates the session and
     * redirects the browser to it. The success/cancel URLs default to the
     * current location with a `moth_checkout=success|cancel` query parameter,
     * which {@link handleCheckoutReturn} consumes on return. Expected
     * failures resolve as result values — this never throws.
     */
    async purchase(product, options = {}) {
        const identifier = typeof product === 'string' ? product : product.identifier;
        if (typeof product !== 'string' && product.stripePriceId === '') {
            // The tier does not ship on the web (no Stripe price). The paywall
            // renders such tiers disabled; this guards direct calls the same way
            // the server would, without a round-trip.
            return {
                status: 'error',
                message: 'this tier is not available for purchase on the web',
                reason: 'PRODUCT_NOT_ON_STORE',
            };
        }
        try {
            const url = await this.createCheckoutSession({
                productIdentifier: identifier,
                successUrl: options.successUrl ?? __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_returnUrl).call(this, 'success'),
                cancelUrl: options.cancelUrl ?? __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_returnUrl).call(this, 'cancel'),
            });
            __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_navigate).call(this, url);
            return { status: 'redirect' };
        }
        catch (err) {
            const mapped = err instanceof MothError ? err : mapConnectError(err);
            const result = {
                status: 'error',
                message: mapped.message,
            };
            if (mapped.reason !== undefined)
                result.reason = mapped.reason;
            return result;
        }
    }
    /**
     * Opens the Stripe Billing Portal (subscription management) by redirect.
     * `returnUrl` defaults to the current location. Throws typed errors
     * (e.g. `MothNoBillingHistoryError`) — callers surface them in UI.
     */
    async manageBilling(options = {}) {
        const url = await this.createBillingPortalSession({
            returnUrl: options.returnUrl ?? __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_currentHref).call(this),
        });
        __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_navigate).call(this, url);
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
    async handleCheckoutReturn() {
        if (typeof window === 'undefined')
            return null;
        const url = new URL(window.location.href);
        const marker = url.searchParams.get(checkoutReturnParam);
        if (marker === null)
            return null;
        url.searchParams.delete(checkoutReturnParam);
        window.history.replaceState(window.history.state, '', url.toString());
        if (marker !== 'success')
            return { status: 'cancelled' };
        if (__classPrivateFieldGet(this, _MothClient_session, "f") === null)
            return { status: 'pending' };
        const before = __classPrivateFieldGet(this, _MothClient_customerInfo, "f");
        for (let attempt = 0; attempt < __classPrivateFieldGet(this, _MothClient_checkoutPollAttempts, "f"); attempt++) {
            if (attempt > 0)
                await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_sleep).call(this, __classPrivateFieldGet(this, _MothClient_checkoutPollIntervalMs, "f"));
            let info;
            try {
                info = await this.getCustomerInfo();
            }
            catch {
                continue; // transient failure: keep polling
            }
            if (info.activeEntitlements.length > 0 &&
                (before.activeEntitlements.length === 0 || !info.equals(before))) {
                return { status: 'purchased' };
            }
        }
        // The webhook has not landed yet; entitlements flip when it does.
        return { status: 'pending' };
    }
    /**
     * Drops every subscription. Re-entrant: subscribing again afterwards
     * works (React StrictMode mounts effects twice), so this is a reset, not
     * a poison pill.
     */
    dispose() {
        __classPrivateFieldGet(this, _MothClient_stateListeners, "f").clear();
        __classPrivateFieldGet(this, _MothClient_infoListeners, "f").clear();
    }
}
_MothClient_store = new WeakMap(), _MothClient_refreshSkewMs = new WeakMap(), _MothClient_now = new WeakMap(), _MothClient_checkoutPollAttempts = new WeakMap(), _MothClient_checkoutPollIntervalMs = new WeakMap(), _MothClient_navigateFn = new WeakMap(), _MothClient_auth = new WeakMap(), _MothClient_projectConfig = new WeakMap(), _MothClient_billing = new WeakMap(), _MothClient_state = new WeakMap(), _MothClient_session = new WeakMap(), _MothClient_refreshing = new WeakMap(), _MothClient_generation = new WeakMap(), _MothClient_customerInfo = new WeakMap(), _MothClient_stateListeners = new WeakMap(), _MothClient_infoListeners = new WeakMap(), _MothClient_lastRawConfig = new WeakMap(), _MothClient_lastRawPaywall = new WeakMap(), _MothClient_instances = new WeakSet(), _MothClient_run = 
// ------------------------------------------------------------ internals
/** Maps transport errors to the typed {@link MothError} hierarchy. */
async function _MothClient_run(fn) {
    try {
        return await fn();
    }
    catch (err) {
        throw mapConnectError(err);
    }
}, _MothClient_authed = 
/**
 * Ensures a fresh Bearer token is attached, then runs `fn`. When the
 * server nonetheless rejects the access token (the client-computed expiry
 * drifted — machine slept mid-call, TTL shortened server-side), the call
 * refreshes reactively and retries exactly once instead of surfacing
 * "session expired" to a session whose refresh token is still valid.
 */
async function _MothClient_authed(fn) {
    await this.accessToken();
    try {
        return await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_run).call(this, fn);
    }
    catch (err) {
        if (!(err instanceof MothInvalidAccessTokenError))
            throw err;
        if (__classPrivateFieldGet(this, _MothClient_session, "f") === null)
            throw err;
        await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_refresh).call(this);
        return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_run).call(this, fn);
    }
}, _MothClient_expiresSoon = function _MothClient_expiresSoon(session) {
    return session.expiresAtMs <= __classPrivateFieldGet(this, _MothClient_now, "f").call(this) + __classPrivateFieldGet(this, _MothClient_refreshSkewMs, "f");
}, _MothClient_refresh = function _MothClient_refresh() {
    const inflight = __classPrivateFieldGet(this, _MothClient_refreshing, "f");
    if (inflight !== null)
        return inflight;
    const refresh = __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_doRefresh).call(this).finally(() => {
        __classPrivateFieldSet(this, _MothClient_refreshing, null, "f");
    });
    __classPrivateFieldSet(this, _MothClient_refreshing, refresh, "f");
    return refresh;
}, _MothClient_settleRefresh = 
/** Waits for an in-flight refresh to settle, ignoring its outcome. */
async function _MothClient_settleRefresh() {
    const inflight = __classPrivateFieldGet(this, _MothClient_refreshing, "f");
    if (inflight === null)
        return;
    try {
        await inflight;
    }
    catch {
        // Handled by the refresh's own callers.
    }
}, _MothClient_doRefresh = async function _MothClient_doRefresh() {
    const session = __classPrivateFieldGet(this, _MothClient_session, "f");
    if (session === null)
        throw new Error('moth: not signed in');
    const generation = __classPrivateFieldGet(this, _MothClient_generation, "f");
    let resp;
    try {
        resp = await __classPrivateFieldGet(this, _MothClient_auth, "f").refreshToken({
            refreshToken: session.refreshToken,
        });
    }
    catch (err) {
        const mapped = mapConnectError(err);
        // A rejected refresh token means the session is gone (rotated-out,
        // revoked or stolen): end up signed out with storage cleared.
        // Transient failures (network, server errors) keep the session. When
        // the session already ended concurrently there is nothing left to
        // clear (and a newer session must not be clobbered).
        if (generation === __classPrivateFieldGet(this, _MothClient_generation, "f") &&
            (mapped instanceof MothInvalidRefreshTokenError ||
                mapped instanceof MothRefreshTokenReusedError ||
                mapped instanceof MothInvalidTokenError ||
                mapped instanceof MothUserDisabledError)) {
            await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_clearSession).call(this);
        }
        throw mapped;
    }
    // The session ended (signOut/deleteAccount) while the RPC was in
    // flight: committing the fresh tokens would silently sign the user
    // back in.
    if (generation !== __classPrivateFieldGet(this, _MothClient_generation, "f")) {
        throw new Error('moth: signed out during token refresh');
    }
    if (resp.tokens === undefined) {
        throw new MothError('moth: refresh returned no tokens');
    }
    await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_openSession).call(this, resp.user, resp.tokens);
    return resp.tokens.accessToken;
}, _MothClient_openSession = async function _MothClient_openSession(user, tokens) {
    if (user === undefined || tokens === undefined) {
        throw new MothError('moth: response missing user or tokens');
    }
    return __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_startSession).call(this, tokens, userFromProto(user, customClaimsOf(tokens.accessToken)));
}, _MothClient_startSession = async function _MothClient_startSession(tokens, user) {
    var _a;
    // Any session transition invalidates in-flight refreshes: a refresh
    // started under the previous session must neither clear this one (on a
    // stale rejection) nor overwrite its tokens (on a stale success).
    __classPrivateFieldSet(this, _MothClient_generation, (_a = __classPrivateFieldGet(this, _MothClient_generation, "f"), _a++, _a), "f");
    const session = {
        accessToken: tokens.accessToken,
        refreshToken: tokens.refreshToken,
        expiresAtMs: __classPrivateFieldGet(this, _MothClient_now, "f").call(this) + Number(tokens.expiresIn) * 1000,
        user,
    };
    __classPrivateFieldSet(this, _MothClient_session, session, "f");
    await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_persist).call(this, session);
    __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_setState).call(this, { status: 'signedIn', user });
    return user;
}, _MothClient_updateUser = 
/** Replaces the cached user snapshot after GetMe/UpdateMe. */
async function _MothClient_updateUser(user) {
    if (user === undefined)
        throw new MothError('moth: response missing user');
    const session = __classPrivateFieldGet(this, _MothClient_session, "f");
    const claims = session === null ? {} : session.user.claims;
    const mothUser = userFromProto(user, claims);
    if (session !== null) {
        const updated = { ...session, user: mothUser };
        __classPrivateFieldSet(this, _MothClient_session, updated, "f");
        await __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_persist).call(this, updated);
        __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_setState).call(this, { status: 'signedIn', user: mothUser });
    }
    return mothUser;
}, _MothClient_persist = 
/**
 * Saves the session, swallowing storage failures: the in-memory session
 * is fully usable (the server accepted the credentials) — it just won't
 * survive a reload. Throwing here would fail a sign-in that actually
 * succeeded.
 */
async function _MothClient_persist(session) {
    try {
        await __classPrivateFieldGet(this, _MothClient_store, "f").save(session);
    }
    catch (err) {
        __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_logStorageFailure).call(this, 'save', err);
    }
}, _MothClient_clearSession = async function _MothClient_clearSession() {
    var _a;
    __classPrivateFieldSet(this, _MothClient_session, null, "f");
    __classPrivateFieldSet(this, _MothClient_generation, (_a = __classPrivateFieldGet(this, _MothClient_generation, "f"), _a++, _a), "f");
    try {
        await __classPrivateFieldGet(this, _MothClient_store, "f").clear();
    }
    catch (err) {
        // Sign-out must complete locally even when storage misbehaves.
        __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_logStorageFailure).call(this, 'clear', err);
    }
    // Emit the signed-out auth state BEFORE the free customer-info reset:
    // the subscription controller listens to both, and it must observe the
    // sign-out (which drops its user id) before the free snapshot arrives,
    // so it does not persist the empty snapshot over the outgoing user's
    // cached entitlements (breaking instant gating when they sign back in).
    __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_setState).call(this, mothSignedOut);
    __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_setCustomerInfo).call(this, MothCustomerInfo.free());
}, _MothClient_applyCustomerInfo = function _MothClient_applyCustomerInfo(proto, issuedForUserId) {
    if (issuedForUserId === undefined ||
        __classPrivateFieldGet(this, _MothClient_session, "f")?.user.id !== issuedForUserId) {
        return __classPrivateFieldGet(this, _MothClient_customerInfo, "f");
    }
    const info = proto === undefined
        ? MothCustomerInfo.free()
        : MothCustomerInfo.fromProto(proto);
    __classPrivateFieldSet(this, _MothClient_customerInfo, info, "f");
    __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_emitCustomerInfo).call(this, info);
    return info;
}, _MothClient_setCustomerInfo = function _MothClient_setCustomerInfo(info) {
    if (info.equals(__classPrivateFieldGet(this, _MothClient_customerInfo, "f")))
        return;
    __classPrivateFieldSet(this, _MothClient_customerInfo, info, "f");
    __classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_emitCustomerInfo).call(this, info);
}, _MothClient_emitCustomerInfo = function _MothClient_emitCustomerInfo(info) {
    for (const listener of [...__classPrivateFieldGet(this, _MothClient_infoListeners, "f")])
        listener(info);
}, _MothClient_setState = function _MothClient_setState(state) {
    __classPrivateFieldSet(this, _MothClient_state, state, "f");
    for (const listener of [...__classPrivateFieldGet(this, _MothClient_stateListeners, "f")])
        listener(state);
}, _MothClient_logStorageFailure = function _MothClient_logStorageFailure(op, err) {
    if (typeof console !== 'undefined') {
        console.warn(`moth: token store ${op} failed:`, err);
    }
}, _MothClient_returnUrl = function _MothClient_returnUrl(outcome) {
    const url = new URL(__classPrivateFieldGet(this, _MothClient_instances, "m", _MothClient_currentHref).call(this));
    url.searchParams.set(checkoutReturnParam, outcome);
    return url.toString();
}, _MothClient_currentHref = function _MothClient_currentHref() {
    if (typeof window === 'undefined')
        return this.config.endpoint;
    return window.location.href;
}, _MothClient_navigate = function _MothClient_navigate(url) {
    if (__classPrivateFieldGet(this, _MothClient_navigateFn, "f") !== undefined) {
        __classPrivateFieldGet(this, _MothClient_navigateFn, "f").call(this, url);
        return;
    }
    if (typeof window === 'undefined')
        return;
    window.location.assign(url);
}, _MothClient_sleep = function _MothClient_sleep(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
};
function userFromProto(user, claims) {
    const mothUser = {
        id: user.id,
        email: user.email,
        emailVerified: user.emailVerified,
        claims,
    };
    if (user.displayName !== '')
        mothUser.displayName = user.displayName;
    if (user.avatarUrl !== '')
        mothUser.avatarUrl = user.avatarUrl;
    if (user.createTime !== undefined) {
        mothUser.createTime = timestampDate(user.createTime);
    }
    return mothUser;
}
function providerToProto(provider) {
    return provider === 'google'
        ? OAuthProvider.OAUTH_PROVIDER_GOOGLE
        : OAuthProvider.OAUTH_PROVIDER_APPLE;
}
