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
var _MothConfigController_instances, _MothConfigController_client, _MothConfigController_storage, _MothConfigController_namespace, _MothConfigController_now, _MothConfigController_theme, _MothConfigController_copy, _MothConfigController_projectConfig, _MothConfigController_listeners, _MothConfigController_started, _MothConfigController_disposed, _MothConfigController_generation, _MothConfigController_ensureFetch, _MothConfigController_ttlMs, _MothConfigController_isFresh, _MothConfigController_themeCache, _MothConfigController_copyCache, _MothConfigController_loadThemeCache, _MothConfigController_loadCopyCache, _MothConfigController_fetch, _MothConfigController_notify;
import { fromBinary, toBinary } from '@bufbuild/protobuf';
import { CopySchema, ThemeSchema } from '../gen/moth/auth/v1/config_pb.js';
import { BlobCache, cacheNamespace, copyCacheKey, defaultCacheStorage, themeCacheKey, } from './cache.js';
import { configCacheTtlMs } from './config.js';
import { languageOf, MothCopy } from './copy.js';
import { fallbackTheme, themeFromProto } from './theme.js';
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
    constructor(client, options = {}) {
        _MothConfigController_instances.add(this);
        _MothConfigController_client.set(this, void 0);
        _MothConfigController_storage.set(this, void 0);
        _MothConfigController_namespace.set(this, void 0);
        _MothConfigController_now.set(this, void 0);
        _MothConfigController_theme.set(this, fallbackTheme());
        _MothConfigController_copy.set(this, void 0);
        _MothConfigController_projectConfig.set(this, null);
        _MothConfigController_listeners.set(this, new Set());
        _MothConfigController_started.set(this, false);
        _MothConfigController_disposed.set(this, false
        /**
         * Bumped every time a fetch is initiated; a fetch only applies its result
         * when its captured generation is still current (a superseded locale's
         * response must never clobber the current one).
         */
        );
        /**
         * Bumped every time a fetch is initiated; a fetch only applies its result
         * when its captured generation is still current (a superseded locale's
         * response must never clobber the current one).
         */
        _MothConfigController_generation.set(this, 0);
        /** The in-flight {@link ensureProjectConfig} fetch, when one is running. */
        _MothConfigController_ensureFetch.set(this, null);
        __classPrivateFieldSet(this, _MothConfigController_client, client, "f");
        __classPrivateFieldSet(this, _MothConfigController_storage, options.storage ?? defaultCacheStorage(), "f");
        __classPrivateFieldSet(this, _MothConfigController_namespace, cacheNamespace(client.config.publishableKey), "f");
        __classPrivateFieldSet(this, _MothConfigController_now, options.now ?? (() => Date.now()), "f");
        __classPrivateFieldSet(this, _MothConfigController_copy, MothCopy.bundled(client.currentLocale), "f");
    }
    /** The current theme (fallback until a cache or the server delivers one). */
    get theme() {
        return __classPrivateFieldGet(this, _MothConfigController_theme, "f");
    }
    /** The current localized copy (bundled floor until one arrives). */
    get copy() {
        return __classPrivateFieldGet(this, _MothConfigController_copy, "f");
    }
    /**
     * The last fetched project config (providers, password policy), or null
     * before the first round-trip. Use {@link ensureProjectConfig} to fetch.
     */
    get projectConfig() {
        return __classPrivateFieldGet(this, _MothConfigController_projectConfig, "f");
    }
    /** Subscribes to any change (theme, copy or project config); replays nothing. */
    subscribe(listener) {
        __classPrivateFieldGet(this, _MothConfigController_listeners, "f").add(listener);
        return () => __classPrivateFieldGet(this, _MothConfigController_listeners, "f").delete(listener);
    }
    /**
     * Loads the cached theme/copy (publishing them immediately when present),
     * then — unless both cache entries are still younger than the TTL
     * (download-once: a fresh cache means zero config RPCs) — revalidates
     * from the server in the background. Idempotent; failures are swallowed —
     * the current values simply stay.
     */
    async start() {
        // Restartable after dispose() (React StrictMode mounts effects twice);
        // the fetch is still performed at most once per instance.
        __classPrivateFieldSet(this, _MothConfigController_disposed, false, "f");
        if (__classPrivateFieldGet(this, _MothConfigController_started, "f"))
            return;
        __classPrivateFieldSet(this, _MothConfigController_started, true, "f");
        const themeFresh = __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_loadThemeCache).call(this);
        const copyFresh = __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_loadCopyCache).call(this);
        __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_notify).call(this);
        if (themeFresh && copyFresh)
            return;
        await __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_fetch).call(this);
    }
    /**
     * Asks the server for the current config (echoing the revisions already
     * held, so unchanged bodies are not re-sent), applies and caches new
     * revisions. Always performs the round-trip — the download-once TTL only
     * gates the automatic revalidation in {@link start}. Safe to call any
     * time; network failures keep the current values.
     */
    async refresh() {
        var _a;
        // The browser locale may have changed since the last fetch: reload
        // that locale's cached floor first so the fetch starts from it.
        const locale = __classPrivateFieldGet(this, _MothConfigController_client, "f").currentLocale;
        if (languageOf(locale) !== languageOf(__classPrivateFieldGet(this, _MothConfigController_copy, "f").locale)) {
            __classPrivateFieldSet(this, _MothConfigController_generation, (_a = __classPrivateFieldGet(this, _MothConfigController_generation, "f"), _a++, _a), "f"); // discard any in-flight fetch for the old locale
            __classPrivateFieldSet(this, _MothConfigController_copy, MothCopy.bundled(locale), "f");
            __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_loadCopyCache).call(this);
            __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_notify).call(this);
        }
        await __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_fetch).call(this);
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
    async ensureProjectConfig() {
        const cached = __classPrivateFieldGet(this, _MothConfigController_projectConfig, "f");
        if (cached !== null)
            return cached;
        __classPrivateFieldSet(this, _MothConfigController_ensureFetch, __classPrivateFieldGet(this, _MothConfigController_ensureFetch, "f") ?? __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_fetch).call(this, { rethrow: true }).finally(() => {
            __classPrivateFieldSet(this, _MothConfigController_ensureFetch, null, "f");
        }), "f");
        await __classPrivateFieldGet(this, _MothConfigController_ensureFetch, "f");
        const fetched = __classPrivateFieldGet(this, _MothConfigController_projectConfig, "f");
        if (fetched === null)
            throw new Error('moth: project config unavailable');
        return fetched;
    }
    dispose() {
        __classPrivateFieldSet(this, _MothConfigController_disposed, true, "f");
        __classPrivateFieldGet(this, _MothConfigController_listeners, "f").clear();
    }
}
_MothConfigController_client = new WeakMap(), _MothConfigController_storage = new WeakMap(), _MothConfigController_namespace = new WeakMap(), _MothConfigController_now = new WeakMap(), _MothConfigController_theme = new WeakMap(), _MothConfigController_copy = new WeakMap(), _MothConfigController_projectConfig = new WeakMap(), _MothConfigController_listeners = new WeakMap(), _MothConfigController_started = new WeakMap(), _MothConfigController_disposed = new WeakMap(), _MothConfigController_generation = new WeakMap(), _MothConfigController_ensureFetch = new WeakMap(), _MothConfigController_instances = new WeakSet(), _MothConfigController_ttlMs = function _MothConfigController_ttlMs() {
    return configCacheTtlMs(__classPrivateFieldGet(this, _MothConfigController_client, "f").config);
}, _MothConfigController_isFresh = function _MothConfigController_isFresh(fetchedAtMs) {
    return __classPrivateFieldGet(this, _MothConfigController_now, "f").call(this) - fetchedAtMs < __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_ttlMs).call(this);
}, _MothConfigController_themeCache = function _MothConfigController_themeCache() {
    return new BlobCache(__classPrivateFieldGet(this, _MothConfigController_storage, "f"), themeCacheKey(__classPrivateFieldGet(this, _MothConfigController_namespace, "f")));
}, _MothConfigController_copyCache = function _MothConfigController_copyCache(language) {
    return new BlobCache(__classPrivateFieldGet(this, _MothConfigController_storage, "f"), copyCacheKey(__classPrivateFieldGet(this, _MothConfigController_namespace, "f"), language));
}, _MothConfigController_loadThemeCache = function _MothConfigController_loadThemeCache() {
    const blob = __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_themeCache).call(this).load();
    if (blob === null)
        return false;
    try {
        __classPrivateFieldSet(this, _MothConfigController_theme, themeFromProto(fromBinary(ThemeSchema, blob.payload)), "f");
    }
    catch {
        __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_themeCache).call(this).remove();
        return false;
    }
    return __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_isFresh).call(this, blob.fetchedAtMs);
}, _MothConfigController_loadCopyCache = function _MothConfigController_loadCopyCache() {
    const locale = __classPrivateFieldGet(this, _MothConfigController_client, "f").currentLocale;
    const blob = __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_copyCache).call(this, languageOf(locale)).load();
    if (blob === null)
        return false;
    try {
        const copy = fromBinary(CopySchema, blob.payload);
        __classPrivateFieldSet(this, _MothConfigController_copy, new MothCopy(copy.locale === '' ? locale : copy.locale, copy.copyRevision, { ...copy.messages }), "f");
    }
    catch {
        __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_copyCache).call(this, languageOf(locale)).remove();
        return false;
    }
    return __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_isFresh).call(this, blob.fetchedAtMs);
}, _MothConfigController_fetch = async function _MothConfigController_fetch(options = {}) {
    var _a;
    const generation = __classPrivateFieldSet(this, _MothConfigController_generation, (_a = __classPrivateFieldGet(this, _MothConfigController_generation, "f"), ++_a), "f");
    let config;
    try {
        config = await __classPrivateFieldGet(this, _MothConfigController_client, "f").getProjectConfig({
            knownThemeRevision: __classPrivateFieldGet(this, _MothConfigController_theme, "f").revisionId,
            knownCopyRevision: __classPrivateFieldGet(this, _MothConfigController_copy, "f").revisionId,
        });
    }
    catch (err) {
        if (options.rethrow === true)
            throw err;
        return; // network failure: keep the current values
    }
    if (__classPrivateFieldGet(this, _MothConfigController_disposed, "f"))
        return;
    const raw = __classPrivateFieldGet(this, _MothConfigController_client, "f").lastRawProjectConfig;
    const now = __classPrivateFieldGet(this, _MothConfigController_now, "f").call(this);
    const current = generation === __classPrivateFieldGet(this, _MothConfigController_generation, "f");
    if (current)
        __classPrivateFieldSet(this, _MothConfigController_projectConfig, config, "f");
    // Theme: omitted body = revision matched — restart the TTL window.
    if (config.theme === undefined) {
        __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_themeCache).call(this).touch(now);
    }
    else {
        if (raw?.theme !== undefined) {
            __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_themeCache).call(this).save({
                payload: toBinary(ThemeSchema, raw.theme),
                revision: config.theme.revisionId,
                locale: '',
                fetchedAtMs: now,
            });
        }
        if (current)
            __classPrivateFieldSet(this, _MothConfigController_theme, config.theme, "f");
    }
    // Copy: same contract, keyed by the negotiated locale's language.
    const update = config.copy;
    if (update !== undefined) {
        if (update.messages === undefined) {
            __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_copyCache).call(this, languageOf(update.locale)).touch(now);
        }
        else {
            if (update.source !== undefined) {
                __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_copyCache).call(this, languageOf(update.locale)).save({
                    payload: toBinary(CopySchema, update.source),
                    revision: update.revisionId,
                    locale: update.locale,
                    fetchedAtMs: now,
                });
            }
            // A superseded fetch (locale switched mid-request) must not
            // overwrite the current locale's copy — last request wins.
            if (current) {
                __classPrivateFieldSet(this, _MothConfigController_copy, new MothCopy(update.locale, update.revisionId, update.messages), "f");
            }
        }
    }
    if (current)
        __classPrivateFieldGet(this, _MothConfigController_instances, "m", _MothConfigController_notify).call(this);
}, _MothConfigController_notify = function _MothConfigController_notify() {
    if (__classPrivateFieldGet(this, _MothConfigController_disposed, "f"))
        return;
    for (const listener of [...__classPrivateFieldGet(this, _MothConfigController_listeners, "f")])
        listener();
};
