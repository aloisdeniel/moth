import { bundledCopy } from './i18n/bundledCopy.js';
const placeholder = /\{([a-zA-Z][a-zA-Z0-9_]*)\}/g;
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
    constructor(locale, revisionId = '', messages = {}) {
        this.locale = locale;
        this.revisionId = revisionId;
        this.messages = messages;
    }
    /**
     * The bundled-only floor for `locale`: no server messages yet, so
     * {@link value} resolves straight from the bundled catalog (English
     * fallback per key). The starting value before the first fetch.
     */
    static bundled(locale) {
        return new MothCopy(locale);
    }
    /**
     * The localized string for `key`, with any `{name}` placeholders replaced
     * from `vars` (a literal `{name}` → value substitution, mirroring the
     * server's placeholder contract — no pluralization). Falls back to the
     * bundled catalog then English; an unknown key returns the key itself.
     */
    value(key, vars) {
        let template = this.messages[key];
        if (template === undefined || template === '') {
            template = bundledCopy(this.locale)[key];
        }
        template ?? (template = key);
        if (vars === undefined || Object.keys(vars).length === 0)
            return template;
        return template.replace(placeholder, (match, name) => {
            const replacement = vars[name];
            return replacement ?? match;
        });
    }
}
/** Maps the Copy message of a GetProjectConfig response. */
export function copyUpdateFromProto(copy) {
    const update = {
        locale: copy.locale === '' ? 'en' : copy.locale,
        revisionId: copy.copyRevision,
        source: copy,
    };
    if (Object.keys(copy.messages).length > 0) {
        update.messages = { ...copy.messages };
    }
    return update;
}
/** The language subtag of a BCP-47 tag (`fr-CA` → `fr`). */
export function languageOf(locale) {
    return locale.split(/[-_]/)[0]?.toLowerCase() ?? 'en';
}
