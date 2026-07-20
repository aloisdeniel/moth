import { fromBinary, toBinary } from '@bufbuild/protobuf'
import { CopySchema, ThemeSchema } from '../gen/moth/auth/v1/config_pb.js'
import {
  BlobCache,
  cacheNamespace,
  copyCacheKey,
  defaultCacheStorage,
  themeCacheKey,
} from './cache.js'
import type { MothClient } from './client.js'
import { configCacheTtlMs } from './config.js'
import { languageOf, MothCopy } from './copy.js'
import type { MothProjectConfig } from './projectConfig.js'
import { fallbackTheme, themeFromProto, type MothTheme } from './theme.js'
import type { WebStorageLike } from './tokenStore.js'

/**
 * Owns the project theme and localized copy for a UI tree: starts from the
 * fallback theme and the bundled copy floor, flips to the cached blobs as
 * soon as they load, then to the server's current config once a
 * revalidation round-trip confirms (or replaces) them.
 *
 * Download-once TTL: each cached blob records when it was fetched or last
 * revalidated. While that moment is younger than
 * `MothConfig.configCacheTtlMs`, {@link start} performs **zero** config
 * RPCs. Once expired, it revalidates cheaply — echoing the cached revisions
 * so the server omits unchanged bodies (an omitted-body match restarts the
 * window). {@link refresh} always hits the server.
 *
 * `MothProvider` creates one automatically; instantiate one only when
 * composing custom UI from the core alone.
 */
export class MothConfigController {
  readonly #client: MothClient
  readonly #storage: WebStorageLike
  readonly #namespace: string
  readonly #now: () => number

  #theme: MothTheme = fallbackTheme()
  #copy: MothCopy
  #projectConfig: MothProjectConfig | null = null

  #listeners = new Set<() => void>()
  #started = false
  #disposed = false

  /**
   * Bumped every time a fetch is initiated; a fetch only applies its result
   * when its captured generation is still current (a superseded locale's
   * response must never clobber the current one).
   */
  #generation = 0

  constructor(
    client: MothClient,
    options: { storage?: WebStorageLike; now?: () => number } = {},
  ) {
    this.#client = client
    this.#storage = options.storage ?? defaultCacheStorage()
    this.#namespace = cacheNamespace(client.config.publishableKey)
    this.#now = options.now ?? (() => Date.now())
    this.#copy = MothCopy.bundled(client.currentLocale)
  }

  /** The current theme (fallback until a cache or the server delivers one). */
  get theme(): MothTheme {
    return this.#theme
  }

  /** The current localized copy (bundled floor until one arrives). */
  get copy(): MothCopy {
    return this.#copy
  }

  /**
   * The last fetched project config (providers, password policy), or null
   * before the first round-trip. Use {@link ensureProjectConfig} to fetch.
   */
  get projectConfig(): MothProjectConfig | null {
    return this.#projectConfig
  }

  /** Subscribes to any change (theme, copy or project config); replays nothing. */
  subscribe(listener: () => void): () => void {
    this.#listeners.add(listener)
    return () => this.#listeners.delete(listener)
  }

  /**
   * Loads the cached theme/copy (publishing them immediately when present),
   * then — unless both cache entries are still younger than the TTL
   * (download-once: a fresh cache means zero config RPCs) — revalidates
   * from the server in the background. Idempotent; failures are swallowed —
   * the current values simply stay.
   */
  async start(): Promise<void> {
    // Restartable after dispose() (React StrictMode mounts effects twice);
    // the fetch is still performed at most once per instance.
    this.#disposed = false
    if (this.#started) return
    this.#started = true
    const themeFresh = this.#loadThemeCache()
    const copyFresh = this.#loadCopyCache()
    this.#notify()
    if (themeFresh && copyFresh) return
    await this.#fetch()
  }

  /**
   * Asks the server for the current config (echoing the revisions already
   * held, so unchanged bodies are not re-sent), applies and caches new
   * revisions. Always performs the round-trip — the download-once TTL only
   * gates the automatic revalidation in {@link start}. Safe to call any
   * time; network failures keep the current values.
   */
  async refresh(): Promise<void> {
    // The browser locale may have changed since the last fetch: reload
    // that locale's cached floor first so the fetch starts from it.
    const locale = this.#client.currentLocale
    if (languageOf(locale) !== languageOf(this.#copy.locale)) {
      this.#generation++ // discard any in-flight fetch for the old locale
      this.#copy = MothCopy.bundled(locale)
      this.#loadCopyCache()
      this.#notify()
    }
    await this.#fetch()
  }

  /**
   * The project config (providers, password policy, sign-up open), fetching
   * it when no round-trip has happened yet. The login screen calls this on
   * mount so policy is always current even when the caches were fresh.
   *
   * Single-flight: concurrent callers (React StrictMode mounts the login
   * screen's effect twice) share one fetch. Two independent fetches would
   * race the generation guard — the superseded one resolves without ever
   * setting `#projectConfig` and would report the config unavailable even
   * though its round-trip succeeded.
   */
  async ensureProjectConfig(): Promise<MothProjectConfig> {
    const cached = this.#projectConfig
    if (cached !== null) return cached
    this.#ensureFetch ??= this.#fetch({ rethrow: true }).finally(() => {
      this.#ensureFetch = null
    })
    await this.#ensureFetch
    const fetched = this.#projectConfig
    if (fetched === null) throw new Error('moth: project config unavailable')
    return fetched
  }

  /** The in-flight {@link ensureProjectConfig} fetch, when one is running. */
  #ensureFetch: Promise<void> | null = null

  dispose(): void {
    this.#disposed = true
    this.#listeners.clear()
  }

  // ------------------------------------------------------------ internals

  #ttlMs(): number {
    return configCacheTtlMs(this.#client.config)
  }

  #isFresh(fetchedAtMs: number): boolean {
    return this.#now() - fetchedAtMs < this.#ttlMs()
  }

  #themeCache(): BlobCache {
    return new BlobCache(this.#storage, themeCacheKey(this.#namespace))
  }

  #copyCache(language: string): BlobCache {
    return new BlobCache(this.#storage, copyCacheKey(this.#namespace, language))
  }

  /** Loads the cached theme; returns whether the entry is still fresh. */
  #loadThemeCache(): boolean {
    const blob = this.#themeCache().load()
    if (blob === null) return false
    try {
      this.#theme = themeFromProto(fromBinary(ThemeSchema, blob.payload))
    } catch {
      this.#themeCache().remove()
      return false
    }
    return this.#isFresh(blob.fetchedAtMs)
  }

  /** Loads the cached copy for the current locale; returns freshness. */
  #loadCopyCache(): boolean {
    const locale = this.#client.currentLocale
    const blob = this.#copyCache(languageOf(locale)).load()
    if (blob === null) return false
    try {
      const copy = fromBinary(CopySchema, blob.payload)
      this.#copy = new MothCopy(
        copy.locale === '' ? locale : copy.locale,
        copy.copyRevision,
        { ...copy.messages },
      )
    } catch {
      this.#copyCache(languageOf(locale)).remove()
      return false
    }
    return this.#isFresh(blob.fetchedAtMs)
  }

  async #fetch(options: { rethrow?: boolean } = {}): Promise<void> {
    const generation = ++this.#generation
    let config: MothProjectConfig
    try {
      config = await this.#client.getProjectConfig({
        knownThemeRevision: this.#theme.revisionId,
        knownCopyRevision: this.#copy.revisionId,
      })
    } catch (err) {
      if (options.rethrow === true) throw err
      return // network failure: keep the current values
    }
    if (this.#disposed) return
    const raw = this.#client.lastRawProjectConfig
    const now = this.#now()
    const current = generation === this.#generation
    if (current) this.#projectConfig = config

    // Theme: omitted body = revision matched — restart the TTL window.
    if (config.theme === undefined) {
      this.#themeCache().touch(now)
    } else {
      if (raw?.theme !== undefined) {
        this.#themeCache().save({
          payload: toBinary(ThemeSchema, raw.theme),
          revision: config.theme.revisionId,
          locale: '',
          fetchedAtMs: now,
        })
      }
      if (current) this.#theme = config.theme
    }

    // Copy: same contract, keyed by the negotiated locale's language.
    const update = config.copy
    if (update !== undefined) {
      if (update.messages === undefined) {
        this.#copyCache(languageOf(update.locale)).touch(now)
      } else {
        if (update.source !== undefined) {
          this.#copyCache(languageOf(update.locale)).save({
            payload: toBinary(CopySchema, update.source),
            revision: update.revisionId,
            locale: update.locale,
            fetchedAtMs: now,
          })
        }
        // A superseded fetch (locale switched mid-request) must not
        // overwrite the current locale's copy — last request wins.
        if (current) {
          this.#copy = new MothCopy(
            update.locale,
            update.revisionId,
            update.messages,
          )
        }
      }
    }
    if (current) this.#notify()
  }

  #notify(): void {
    if (this.#disposed) return
    for (const listener of [...this.#listeners]) listener()
  }
}
