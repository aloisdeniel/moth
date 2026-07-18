import { type Client, type Transport } from '@connectrpc/connect';
import { ConfigService } from '../gen/moth/auth/v1/config_pb.js';
import { type MothConfig } from './config.js';
import { MothCustomerInfo } from './customerInfo.js';
import { MothOffering, type MothOfferingProduct, type MothPaywall } from './offering.js';
import type { MothProjectConfig } from './projectConfig.js';
import { type MothPurchaseResult } from './purchase.js';
import { type MothPushDeviceMetadata, type MothPushPermission, type MothPushTarget } from './push.js';
import { type MothAuthState, type MothUser } from './user.js';
/** A social sign-in provider supported by moth. */
export type MothOAuthProvider = 'google' | 'apple';
/**
 * Result of {@link MothClient.signUp}. Depending on project policy the server
 * returns the user with tokens (signed in immediately), the user without
 * tokens (email verification required first) or nothing at all
 * (enumeration-safe projects).
 */
export interface MothSignUpResult {
    user?: MothUser;
    /** True when sign-up also opened a session (tokens were returned). */
    signedIn: boolean;
}
/** Constructor options beyond the config. */
export interface MothClientOptions {
    /**
     * Transport override (tests use `createRouterTransport`); defaults to the
     * gRPC-Web transport for `config.endpoint`. The moth metadata headers are
     * attached either way.
     */
    transport?: Transport;
    /** Access tokens expiring within this window refresh proactively. */
    refreshSkewMs?: number;
    /** Clock override for tests. */
    now?: () => number;
    /** Checkout-return polling budget: attempts x interval. */
    checkoutPollAttempts?: number;
    checkoutPollIntervalMs?: number;
    /** Redirect override for tests; defaults to `window.location.assign`. */
    navigate?: (url: string) => void;
}
type Unsubscribe = () => void;
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
export declare class MothClient {
    #private;
    readonly config: MothConfig;
    constructor(config: MothConfig, options?: MothClientOptions);
    /** The current auth state (`loading` until {@link restore} completes). */
    get currentState(): MothAuthState;
    /** The signed-in user, or null. */
    get currentUser(): MothUser | null;
    /**
     * The locale the SDK negotiates copy for: `config.locale` when the app
     * pinned one, otherwise the live browser language.
     */
    get currentLocale(): string;
    /**
     * Subscribes to auth state changes. The current state is REPLAYED to the
     * new subscriber immediately (synchronously), then every subsequent change
     * is delivered. Returns the unsubscribe function.
     */
    onAuthStateChanged(listener: (state: MothAuthState) => void): Unsubscribe;
    /**
     * Registers work that must run at the start of {@link signOut}, while the
     * session (and its Bearer token) is still valid — e.g. the push
     * controller revoking this installation's device registration. Hooks are
     * awaited best-effort: a failing hook never blocks the sign-out. Returns
     * the unsubscribe function.
     */
    onBeforeSignOut(hook: () => void | Promise<void>): Unsubscribe;
    /**
     * The signed-in user's current subscription state. Always valid — an
     * empty {@link MothCustomerInfo} (the free `none` tier) until the first
     * {@link getCustomerInfo}, and while signed out.
     */
    get currentCustomerInfo(): MothCustomerInfo;
    /**
     * Subscribes to subscription-state changes. Like
     * {@link onAuthStateChanged}, the current value is replayed to every new
     * subscriber. Returns the unsubscribe function.
     */
    onEntitlementsChanged(listener: (info: MothCustomerInfo) => void): Unsubscribe;
    /**
     * Seeds {@link currentCustomerInfo} from a cache (stale-while-revalidate)
     * so subscribers reflect the last known entitlements before the first
     * {@link getCustomerInfo} lands. Deduplicated; the server stays
     * authoritative and overwrites on the next billing RPC.
     */
    primeCustomerInfo(info: MothCustomerInfo): void;
    /**
     * Restores a persisted session from the token store. Call once at
     * startup; until it completes {@link currentState} is `loading`.
     *
     * A stored session whose access token is still fresh signs in without a
     * network round-trip. An expired one is refreshed; when the server
     * rejects the refresh token the session is cleared, while transient
     * network failures keep it (the next {@link accessToken} call retries).
     */
    restore(): Promise<MothAuthState>;
    /**
     * Returns a valid access token for the signed-in user, refreshing it
     * first when it expires within the refresh skew. Concurrent callers share
     * a single refresh RPC. Throws when signed out.
     */
    accessToken(): Promise<string>;
    /**
     * Forces a token refresh and returns the updated user. Throws when the
     * session ended (e.g. a concurrent {@link signOut}) before it completed.
     */
    refresh(): Promise<MothUser>;
    /** Registers a new email/password user, subject to project policy. */
    signUp(params: {
        email: string;
        password: string;
        displayName?: string;
        deviceInfo?: string;
    }): Promise<MothSignUpResult>;
    /** Exchanges email/password for a session. */
    signIn(params: {
        email: string;
        password: string;
        deviceInfo?: string;
    }): Promise<MothUser>;
    /**
     * Revokes the current session server-side (best effort — local sign-out
     * happens even when the revocation RPC fails) and clears the stored
     * session. With `allDevices` every session of the user is revoked.
     */
    signOut(options?: {
        allDevices?: boolean;
    }): Promise<void>;
    /**
     * Changes the password (requires the current one). Every other session is
     * revoked; this device continues on a fresh token pair.
     */
    changePassword(params: {
        currentPassword: string;
        newPassword: string;
    }): Promise<void>;
    /**
     * Signs in (or up) with a provider ID token obtained from a Google/Apple
     * flow the app ran itself. `rawNonce`, `authorizationCode`, `givenName`
     * and `familyName` are Apple-only.
     */
    signInWithOAuth(params: {
        provider: MothOAuthProvider;
        idToken: string;
        rawNonce?: string;
        authorizationCode?: string;
        givenName?: string;
        familyName?: string;
        deviceInfo?: string;
    }): Promise<MothUser>;
    /**
     * Trades the one-time `code` from the web-redirect OAuth flow
     * (`GET /oauth/{provider}/start` → consent → callback → redirect back
     * with `?code=...`) for a session.
     */
    exchangeOAuthCode(code: string, options?: {
        deviceInfo?: string;
    }): Promise<MothUser>;
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
    oauthStartUrl(provider: MothOAuthProvider, redirectUri?: string): string;
    /**
     * Starts the web-redirect OAuth flow: navigates the browser to
     * {@link oauthStartUrl}. `redirectUri` defaults to the current URL with
     * its fragment stripped — the server refuses redirect URIs containing
     * `#` (anything appended after it would land in the fragment instead of
     * the query), so hash-routed apps (e.g. HashRouter) return to the
     * fragment-less URL and should restore their route after
     * {@link exchangeOAuthCode}. Throws when `config.projectSlug` is unset.
     */
    signInWithRedirect(provider: MothOAuthProvider, options?: {
        redirectUri?: string;
    }): void;
    /**
     * Removes the signed-in user's identity for `provider`. Refused with
     * `MothLastLoginMethodError` when it would leave no way to sign in.
     */
    unlinkIdentity(provider: MothOAuthProvider): Promise<void>;
    /** Fetches the signed-in user from the server and updates {@link currentUser}. */
    getMe(): Promise<MothUser>;
    /** Updates profile fields; only defined arguments are sent. */
    updateMe(params: {
        displayName?: string;
        avatarUrl?: string;
    }): Promise<MothUser>;
    /**
     * Permanently deletes the account after fresh re-authentication with
     * `password`, then clears the local session.
     */
    deleteAccount(params: {
        password: string;
    }): Promise<void>;
    /** (Re)sends the verification email. Never reveals whether an account exists. */
    requestEmailVerification(email: string): Promise<void>;
    /** Consumes a verification token from the email link. */
    confirmEmailVerification(token: string): Promise<void>;
    /** Emails a password-reset link. Never reveals whether an account exists. */
    requestPasswordReset(email: string): Promise<void>;
    /** Consumes a reset token and sets the new password; every session is revoked. */
    confirmPasswordReset(params: {
        token: string;
        newPassword: string;
    }): Promise<void>;
    /** Sends a confirmation link to `newEmail`; the account switches once verified. */
    requestEmailChange(newEmail: string): Promise<void>;
    /** Consumes an email-change (or revert) token and applies the address. */
    confirmEmailChange(token: string): Promise<void>;
    /**
     * Fetches the project's public configuration (enabled providers, password
     * policy, theme, localized copy). Pass the cached revisions as
     * `knownThemeRevision` / `knownCopyRevision`: when they still match, the
     * server omits the body and the corresponding field stays undefined (keep
     * the cached copy).
     */
    getProjectConfig(options?: {
        knownThemeRevision?: string;
        knownCopyRevision?: string;
    }): Promise<MothProjectConfig>;
    /**
     * The raw wire messages of the last GetProjectConfig response, for the
     * config caches (they persist the payload exactly as delivered).
     */
    get lastRawProjectConfig(): Awaited<ReturnType<Client<typeof ConfigService>['getProjectConfig']>> | null;
    /**
     * Fetches the signed-in user's subscription state, updates
     * {@link currentCustomerInfo} and notifies subscribers. Cheap and safe to
     * call on every launch. Throws when signed out.
     */
    getCustomerInfo(): Promise<MothCustomerInfo>;
    /**
     * The products of `offering` (empty selects the project's default), for a
     * paywall to display. Publishable-key only — safe before sign-in.
     */
    getOfferings(options?: {
        offering?: string;
    }): Promise<MothOffering>;
    /**
     * The project's public paywall configuration, or null when
     * `knownPaywallRevision` still matches the current revision (keep the
     * cached copy). Publishable-key only — safe before sign-in.
     */
    getPaywall(options?: {
        knownPaywallRevision?: string;
    }): Promise<MothPaywall | null>;
    /** The raw wire message of the last non-empty GetPaywall response. */
    get lastRawPaywall(): (import("@bufbuild/protobuf").Message<"moth.billing.v1.Paywall"> & {
        revisionId: string;
        headline: string;
        subtitle: string;
        benefits: string[];
        offering: string;
        highlightedProductIdentifier: string;
        layout: import("../gen/moth/billing/v1/billing_pb.js").PaywallLayout;
        termsUrl: string;
        privacyUrl: string;
    }) | null;
    /**
     * Creates a Stripe-hosted Checkout session for `productIdentifier` and
     * returns its URL. Prefer {@link purchase}, which also performs the
     * redirect and defaults the return URLs.
     */
    createCheckoutSession(params: {
        productIdentifier: string;
        successUrl: string;
        cancelUrl: string;
    }): Promise<string>;
    /** Creates a Stripe Billing Portal session and returns its URL. */
    createBillingPortalSession(params: {
        returnUrl: string;
    }): Promise<string>;
    /**
     * Buys `product` through Stripe-hosted Checkout: creates the session and
     * redirects the browser to it. The success/cancel URLs default to the
     * current location with a `moth_checkout=success|cancel` query parameter,
     * which {@link handleCheckoutReturn} consumes on return. Expected
     * failures resolve as result values — this never throws.
     */
    purchase(product: MothOfferingProduct | string, options?: {
        successUrl?: string;
        cancelUrl?: string;
    }): Promise<MothPurchaseResult>;
    /**
     * Opens the Stripe Billing Portal (subscription management) by redirect.
     * `returnUrl` defaults to the current location. Throws typed errors
     * (e.g. `MothNoBillingHistoryError`) — callers surface them in UI.
     */
    manageBilling(options?: {
        returnUrl?: string;
    }): Promise<void>;
    /**
     * Detects a return from Stripe Checkout (the `moth_checkout` query
     * parameter {@link purchase} added to the return URLs), strips it from
     * the address bar, and — on success — re-fetches the customer info with a
     * short poll to absorb webhook latency. Call once on startup after
     * {@link restore}; `MothProvider` does this automatically.
     *
     * Returns null when the current URL carries no checkout-return marker.
     */
    handleCheckoutReturn(): Promise<MothPurchaseResult | null>;
    /**
     * Upserts this installation's push registration
     * (`moth.push.v1.PushService.RegisterDevice`). Idempotent by design —
     * call it on every launch, token rotation and permission change with the
     * same stable `deviceId`; the registry replaces the row. Throws when
     * signed out (registrations always hang off the signed-in user).
     */
    registerPushDevice(params: {
        target: MothPushTarget;
        /**
         * The push credential: APNs/FCM token or the serialized Web Push
         * subscription (JSON with endpoint + keys).
         */
        token: string;
        /** Client-generated stable installation id. */
        deviceId: string;
        permission?: MothPushPermission;
        metadata?: MothPushDeviceMetadata;
    }): Promise<void>;
    /**
     * Revokes this installation's push registration (`signed_out`).
     * Idempotent: unknown or already-revoked device ids succeed. Throws when
     * signed out — call it *before* dropping the session.
     */
    unregisterPushDevice(deviceId: string): Promise<void>;
    /**
     * Drops every subscription. Re-entrant: subscribing again afterwards
     * works (React StrictMode mounts effects twice), so this is a reset, not
     * a poison pill.
     */
    dispose(): void;
}
export {};
