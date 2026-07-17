var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var _InMemoryTokenStore_session, _WebStorageTokenStore_storage, _WebStorageTokenStore_key;
/** Keeps the session in memory only — nothing survives a reload. */
export class InMemoryTokenStore {
    constructor() {
        _InMemoryTokenStore_session.set(this, null);
    }
    load() {
        return __classPrivateFieldGet(this, _InMemoryTokenStore_session, "f");
    }
    save(session) {
        __classPrivateFieldSet(this, _InMemoryTokenStore_session, session, "f");
    }
    clear() {
        __classPrivateFieldSet(this, _InMemoryTokenStore_session, null, "f");
    }
}
_InMemoryTokenStore_session = new WeakMap();
/**
 * Persists the session as JSON in a Web Storage area, under
 * `moth_session_<publishableKey>` so two projects on one origin never
 * collide. A corrupted entry is deleted and treated as signed out; storage
 * failures (quota, privacy modes) surface as misses / no-ops so they never
 * fail an operation.
 */
export class WebStorageTokenStore {
    constructor(publishableKey, storage) {
        _WebStorageTokenStore_storage.set(this, void 0);
        _WebStorageTokenStore_key.set(this, void 0);
        __classPrivateFieldSet(this, _WebStorageTokenStore_storage, storage, "f");
        __classPrivateFieldSet(this, _WebStorageTokenStore_key, `moth_session_${publishableKey}`, "f");
    }
    load() {
        let raw;
        try {
            raw = __classPrivateFieldGet(this, _WebStorageTokenStore_storage, "f").getItem(__classPrivateFieldGet(this, _WebStorageTokenStore_key, "f"));
        }
        catch {
            // Storage itself failed (disabled, sandboxed iframe): signed out.
            return null;
        }
        if (raw === null)
            return null;
        try {
            const parsed = JSON.parse(raw);
            if (typeof parsed.access_token !== 'string' ||
                typeof parsed.refresh_token !== 'string' ||
                typeof parsed.expires_at_ms !== 'number' ||
                typeof parsed.user?.id !== 'string') {
                throw new Error('malformed session');
            }
            const user = {
                id: parsed.user.id,
                email: parsed.user.email ?? '',
                emailVerified: parsed.user.email_verified ?? false,
                claims: parsed.user.claims ?? {},
            };
            if (parsed.user.display_name)
                user.displayName = parsed.user.display_name;
            if (parsed.user.avatar_url)
                user.avatarUrl = parsed.user.avatar_url;
            if (parsed.user.create_time) {
                const t = new Date(parsed.user.create_time);
                if (!Number.isNaN(t.getTime()))
                    user.createTime = t;
            }
            return {
                accessToken: parsed.access_token,
                refreshToken: parsed.refresh_token,
                expiresAtMs: parsed.expires_at_ms,
                user,
            };
        }
        catch {
            // Unreadable entry (corruption, format change): treat as signed out.
            try {
                __classPrivateFieldGet(this, _WebStorageTokenStore_storage, "f").removeItem(__classPrivateFieldGet(this, _WebStorageTokenStore_key, "f"));
            }
            catch {
                // Best effort — the entry was unreadable anyway.
            }
            return null;
        }
    }
    save(session) {
        const user = {
            id: session.user.id,
            email: session.user.email,
            email_verified: session.user.emailVerified,
        };
        if (session.user.displayName)
            user['display_name'] = session.user.displayName;
        if (session.user.avatarUrl)
            user['avatar_url'] = session.user.avatarUrl;
        if (session.user.createTime) {
            user['create_time'] = session.user.createTime.toISOString();
        }
        if (Object.keys(session.user.claims).length > 0) {
            user['claims'] = session.user.claims;
        }
        __classPrivateFieldGet(this, _WebStorageTokenStore_storage, "f").setItem(__classPrivateFieldGet(this, _WebStorageTokenStore_key, "f"), JSON.stringify({
            access_token: session.accessToken,
            refresh_token: session.refreshToken,
            expires_at_ms: session.expiresAtMs,
            user,
        }));
    }
    clear() {
        __classPrivateFieldGet(this, _WebStorageTokenStore_storage, "f").removeItem(__classPrivateFieldGet(this, _WebStorageTokenStore_key, "f"));
    }
}
_WebStorageTokenStore_storage = new WeakMap(), _WebStorageTokenStore_key = new WeakMap();
/**
 * Builds the token store for a `MothConfig.storage` option. Unavailable web
 * storage (server-side rendering, storage disabled) degrades to in-memory.
 */
export function createTokenStore(publishableKey, storage) {
    if (storage !== undefined && typeof storage === 'object')
        return storage;
    if (storage === 'memory')
        return new InMemoryTokenStore();
    const web = webStorage(storage === 'session' ? 'session' : 'local');
    if (web === null)
        return new InMemoryTokenStore();
    return new WebStorageTokenStore(publishableKey, web);
}
function webStorage(kind) {
    try {
        if (typeof window === 'undefined')
            return null;
        return kind === 'session' ? window.sessionStorage : window.localStorage;
    }
    catch {
        return null;
    }
}
