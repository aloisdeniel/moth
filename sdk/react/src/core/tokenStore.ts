import type { MothUser } from './user.js'

/**
 * One persisted session: the token pair plus a snapshot of the user, so a
 * restored session can render without a network round-trip.
 */
export interface StoredSession {
  accessToken: string
  refreshToken: string
  /** When the access token expires (Unix ms, computed from `expires_in`). */
  expiresAtMs: number
  user: MothUser
}

/**
 * Where `MothClient` persists the session. The default is localStorage
 * (namespaced by publishable key); swap in `'session'`, `'memory'` or a
 * custom implementation via `MothConfig.storage`. Methods may be sync or
 * async; failures never fail an operation (the client logs and continues).
 */
export interface TokenStore {
  load(): StoredSession | null | Promise<StoredSession | null>
  save(session: StoredSession): void | Promise<void>
  clear(): void | Promise<void>
}

/** Keeps the session in memory only — nothing survives a reload. */
export class InMemoryTokenStore implements TokenStore {
  #session: StoredSession | null = null

  load(): StoredSession | null {
    return this.#session
  }

  save(session: StoredSession): void {
    this.#session = session
  }

  clear(): void {
    this.#session = null
  }
}

/** The subset of the Web Storage API the SDK relies on. */
export interface WebStorageLike {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
  removeItem(key: string): void
}

/**
 * Persists the session as JSON in a Web Storage area, under
 * `moth_session_<publishableKey>` so two projects on one origin never
 * collide. A corrupted entry is deleted and treated as signed out; storage
 * failures (quota, privacy modes) surface as misses / no-ops so they never
 * fail an operation.
 */
export class WebStorageTokenStore implements TokenStore {
  readonly #storage: WebStorageLike
  readonly #key: string

  constructor(publishableKey: string, storage: WebStorageLike) {
    this.#storage = storage
    this.#key = `moth_session_${publishableKey}`
  }

  load(): StoredSession | null {
    let raw: string | null
    try {
      raw = this.#storage.getItem(this.#key)
    } catch {
      // Storage itself failed (disabled, sandboxed iframe): signed out.
      return null
    }
    if (raw === null) return null
    try {
      const parsed = JSON.parse(raw) as {
        access_token: string
        refresh_token: string
        expires_at_ms: number
        user: {
          id: string
          email: string
          email_verified?: boolean
          display_name?: string
          avatar_url?: string
          create_time?: string
          claims?: Record<string, unknown>
        }
      }
      if (
        typeof parsed.access_token !== 'string' ||
        typeof parsed.refresh_token !== 'string' ||
        typeof parsed.expires_at_ms !== 'number' ||
        typeof parsed.user?.id !== 'string'
      ) {
        throw new Error('malformed session')
      }
      const user: MothUser = {
        id: parsed.user.id,
        email: parsed.user.email ?? '',
        emailVerified: parsed.user.email_verified ?? false,
        claims: parsed.user.claims ?? {},
      }
      if (parsed.user.display_name) user.displayName = parsed.user.display_name
      if (parsed.user.avatar_url) user.avatarUrl = parsed.user.avatar_url
      if (parsed.user.create_time) {
        const t = new Date(parsed.user.create_time)
        if (!Number.isNaN(t.getTime())) user.createTime = t
      }
      return {
        accessToken: parsed.access_token,
        refreshToken: parsed.refresh_token,
        expiresAtMs: parsed.expires_at_ms,
        user,
      }
    } catch {
      // Unreadable entry (corruption, format change): treat as signed out.
      try {
        this.#storage.removeItem(this.#key)
      } catch {
        // Best effort — the entry was unreadable anyway.
      }
      return null
    }
  }

  save(session: StoredSession): void {
    const user: Record<string, unknown> = {
      id: session.user.id,
      email: session.user.email,
      email_verified: session.user.emailVerified,
    }
    if (session.user.displayName) user['display_name'] = session.user.displayName
    if (session.user.avatarUrl) user['avatar_url'] = session.user.avatarUrl
    if (session.user.createTime) {
      user['create_time'] = session.user.createTime.toISOString()
    }
    if (Object.keys(session.user.claims).length > 0) {
      user['claims'] = session.user.claims
    }
    this.#storage.setItem(
      this.#key,
      JSON.stringify({
        access_token: session.accessToken,
        refresh_token: session.refreshToken,
        expires_at_ms: session.expiresAtMs,
        user,
      }),
    )
  }

  clear(): void {
    this.#storage.removeItem(this.#key)
  }
}

/**
 * Builds the token store for a `MothConfig.storage` option. Unavailable web
 * storage (server-side rendering, storage disabled) degrades to in-memory.
 */
export function createTokenStore(
  publishableKey: string,
  storage: 'local' | 'session' | 'memory' | TokenStore | undefined,
): TokenStore {
  if (storage !== undefined && typeof storage === 'object') return storage
  if (storage === 'memory') return new InMemoryTokenStore()
  const web = webStorage(storage === 'session' ? 'session' : 'local')
  if (web === null) return new InMemoryTokenStore()
  return new WebStorageTokenStore(publishableKey, web)
}

function webStorage(kind: 'local' | 'session'): WebStorageLike | null {
  try {
    if (typeof window === 'undefined') return null
    return kind === 'session' ? window.sessionStorage : window.localStorage
  } catch {
    return null
  }
}
