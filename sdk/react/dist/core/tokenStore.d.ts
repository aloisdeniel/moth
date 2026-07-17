import type { MothUser } from './user.js';
/**
 * One persisted session: the token pair plus a snapshot of the user, so a
 * restored session can render without a network round-trip.
 */
export interface StoredSession {
    accessToken: string;
    refreshToken: string;
    /** When the access token expires (Unix ms, computed from `expires_in`). */
    expiresAtMs: number;
    user: MothUser;
}
/**
 * Where `MothClient` persists the session. The default is localStorage
 * (namespaced by publishable key); swap in `'session'`, `'memory'` or a
 * custom implementation via `MothConfig.storage`. Methods may be sync or
 * async; failures never fail an operation (the client logs and continues).
 */
export interface TokenStore {
    load(): StoredSession | null | Promise<StoredSession | null>;
    save(session: StoredSession): void | Promise<void>;
    clear(): void | Promise<void>;
}
/** Keeps the session in memory only — nothing survives a reload. */
export declare class InMemoryTokenStore implements TokenStore {
    #private;
    load(): StoredSession | null;
    save(session: StoredSession): void;
    clear(): void;
}
/** The subset of the Web Storage API the SDK relies on. */
export interface WebStorageLike {
    getItem(key: string): string | null;
    setItem(key: string, value: string): void;
    removeItem(key: string): void;
}
/**
 * Persists the session as JSON in a Web Storage area, under
 * `moth_session_<publishableKey>` so two projects on one origin never
 * collide. A corrupted entry is deleted and treated as signed out; storage
 * failures (quota, privacy modes) surface as misses / no-ops so they never
 * fail an operation.
 */
export declare class WebStorageTokenStore implements TokenStore {
    #private;
    constructor(publishableKey: string, storage: WebStorageLike);
    load(): StoredSession | null;
    save(session: StoredSession): void;
    clear(): void;
}
/**
 * Builds the token store for a `MothConfig.storage` option. Unavailable web
 * storage (server-side rendering, storage disabled) degrades to in-memory.
 */
export declare function createTokenStore(publishableKey: string, storage: 'local' | 'session' | 'memory' | TokenStore | undefined): TokenStore;
