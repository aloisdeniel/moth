import type { Copy } from '../gen/moth/auth/v1/config_pb.js'
import { bundledCopy } from './i18n/bundledCopy.js'

const placeholder = /\{([a-zA-Z][a-zA-Z0-9_]*)\}/g

/**
 * The resolved, localized copy for a locale: the message key → localized
 * string map the SDK's screens render from (`sign_in.*`, `sign_up.*`,
 * `paywall.*`), for the locale the server negotiated from the browser
 * language.
 *
 * Resolution is **server override → bundled → English**: {@link value}
 * returns the server-delivered string for a key when present, otherwise the
 * SDK's bundled catalog for the locale, which itself fills in English for
 * any key the locale lacks. So a screen is always fully localized —
 * instantly from the bundle before the config loads and offline, refined by
 * the project's own wording once it arrives.
 */
export class MothCopy {
  /** The negotiated locale this copy is for. */
  readonly locale: string

  /**
   * Opaque cache token for this `(locale, override-revision)` pair; empty
   * for a bundled-only floor. Echoed as `known_copy_revision`.
   */
  readonly revisionId: string

  /** Server-delivered message key → localized string; empty for the floor. */
  readonly messages: Readonly<Record<string, string>>

  constructor(
    locale: string,
    revisionId = '',
    messages: Readonly<Record<string, string>> = {},
  ) {
    this.locale = locale
    this.revisionId = revisionId
    this.messages = messages
  }

  /**
   * The bundled-only floor for `locale`: no server messages yet, so
   * {@link value} resolves straight from the bundled catalog (English
   * fallback per key). The starting value before the first fetch.
   */
  static bundled(locale: string): MothCopy {
    return new MothCopy(locale)
  }

  /**
   * The localized string for `key`, with any `{name}` placeholders replaced
   * from `vars` (a literal `{name}` → value substitution, mirroring the
   * server's placeholder contract — no pluralization). Falls back to the
   * bundled catalog then English; an unknown key returns the key itself.
   */
  value(key: string, vars?: Record<string, string>): string {
    let template: string | undefined = this.messages[key]
    if (template === undefined || template === '') {
      template = bundledCopy(this.locale)[key]
    }
    template ??= key
    if (vars === undefined || Object.keys(vars).length === 0) return template
    return template.replace(placeholder, (match, name: string) => {
      const replacement = vars[name]
      return replacement ?? match
    })
  }
}

/**
 * The copy carried by a `GetProjectConfig` response: the negotiated locale
 * and revision are always present; {@link messages} is undefined when the
 * server confirmed the `knownCopyRevision` still matches (keep the cached
 * copy — stale-while-revalidate, like the theme).
 */
export interface MothCopyUpdate {
  locale: string
  revisionId: string
  /** Resolved key → string when the revision changed; undefined when unchanged. */
  messages?: Record<string, string>
  /**
   * The wire message this update was mapped from — the payload the config
   * cache persists, so cache and wire share one schema.
   */
  source?: Copy
}

/** Maps the Copy message of a GetProjectConfig response. */
export function copyUpdateFromProto(copy: Copy): MothCopyUpdate {
  const update: MothCopyUpdate = {
    locale: copy.locale === '' ? 'en' : copy.locale,
    revisionId: copy.copyRevision,
    source: copy,
  }
  if (Object.keys(copy.messages).length > 0) {
    update.messages = { ...copy.messages }
  }
  return update
}

/** The language subtag of a BCP-47 tag (`fr-CA` → `fr`). */
export function languageOf(locale: string): string {
  return locale.split(/[-_]/)[0]?.toLowerCase() ?? 'en'
}
