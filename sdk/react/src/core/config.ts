import type { TokenStore } from './tokenStore.js'

/**
 * Connection settings for a moth project. The values to paste here are shown
 * on the project's setup-instructions page in the moth admin.
 */
export interface MothConfig {
  /** Base URL of the moth server, e.g. `https://auth.example.com`. */
  endpoint: string

  /**
   * The project's publishable key (`pk_...`), attached to every call as the
   * `x-moth-key` header. Safe to embed in the app.
   */
  publishableKey: string

  /**
   * Overrides the browser locale for language negotiation and localized
   * copy. Leave unset to follow `navigator.language`. Sent as the
   * `x-moth-language` header (a BCP-47 tag) on every call.
   */
  locale?: string

  /**
   * The app's display name, substituted for the `{app}` placeholder in the
   * SDK's bundled fallback copy. Only used offline / before the first
   * GetProjectConfig — the server interpolates its own project name into the
   * copy it delivers.
   */
  appName?: string

  /**
   * The project's URL slug, needed only for the web-redirect OAuth flow
   * (`GET /oauth/{provider}/start?project={slug}&...`). Shown on the
   * setup-instructions page. Without it the Google/Apple buttons on
   * `MothLoginScreen` are hidden.
   */
  projectSlug?: string

  /**
   * How long the cached config blobs (theme, localized copy, paywall) are
   * served without any network revalidation — download once, then start
   * quietly until the TTL expires. Defaults to one hour. Use 0 to
   * revalidate on every start; explicit refreshes always hit the server.
   */
  configCacheTtlMs?: number

  /**
   * Where the session (rotating refresh token + user snapshot) persists:
   * `'local'` (localStorage, the default — survives restarts),
   * `'session'` (sessionStorage — per-tab), `'memory'` (nothing survives a
   * reload), or a custom {@link TokenStore}.
   *
   * The XSS trade-off is real and documented rather than papered over: any
   * script running on your origin can read web storage. Rotation-reuse
   * detection server-side limits the blast radius of a stolen refresh
   * token, but a strict CSP is your first line of defense.
   */
  storage?: 'local' | 'session' | 'memory' | TokenStore
}

/** Default {@link MothConfig.configCacheTtlMs}: one hour. */
export const defaultConfigCacheTtlMs = 60 * 60 * 1000

/** The TTL for `config`, applying the default. */
export function configCacheTtlMs(config: MothConfig): number {
  return config.configCacheTtlMs ?? defaultConfigCacheTtlMs
}

/**
 * The locale the SDK negotiates copy for: the pinned {@link MothConfig.locale}
 * when set, otherwise the live browser language. Sent as `x-moth-language`.
 */
export function currentLocaleOf(config: MothConfig): string {
  if (config.locale) return config.locale
  if (typeof navigator !== 'undefined' && navigator.language) {
    return navigator.language
  }
  return 'en'
}
