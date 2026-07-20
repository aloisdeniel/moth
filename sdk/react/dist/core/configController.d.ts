import type { MothClient } from './client.js';
import { MothCopy } from './copy.js';
import type { MothProjectConfig } from './projectConfig.js';
import { type MothTheme } from './theme.js';
import type { WebStorageLike } from './tokenStore.js';
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
export declare class MothConfigController {
    #private;
    constructor(client: MothClient, options?: {
        storage?: WebStorageLike;
        now?: () => number;
    });
    /** The current theme (fallback until a cache or the server delivers one). */
    get theme(): MothTheme;
    /** The current localized copy (bundled floor until one arrives). */
    get copy(): MothCopy;
    /**
     * The last fetched project config (providers, password policy), or null
     * before the first round-trip. Use {@link ensureProjectConfig} to fetch.
     */
    get projectConfig(): MothProjectConfig | null;
    /** Subscribes to any change (theme, copy or project config); replays nothing. */
    subscribe(listener: () => void): () => void;
    /**
     * Loads the cached theme/copy (publishing them immediately when present),
     * then — unless both cache entries are still younger than the TTL
     * (download-once: a fresh cache means zero config RPCs) — revalidates
     * from the server in the background. Idempotent; failures are swallowed —
     * the current values simply stay.
     */
    start(): Promise<void>;
    /**
     * Asks the server for the current config (echoing the revisions already
     * held, so unchanged bodies are not re-sent), applies and caches new
     * revisions. Always performs the round-trip — the download-once TTL only
     * gates the automatic revalidation in {@link start}. Safe to call any
     * time; network failures keep the current values.
     */
    refresh(): Promise<void>;
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
    ensureProjectConfig(): Promise<MothProjectConfig>;
    dispose(): void;
}
