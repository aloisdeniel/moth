/** Default {@link MothConfig.configCacheTtlMs}: one hour. */
export const defaultConfigCacheTtlMs = 60 * 60 * 1000;
/** The TTL for `config`, applying the default. */
export function configCacheTtlMs(config) {
    return config.configCacheTtlMs ?? defaultConfigCacheTtlMs;
}
/**
 * The locale the SDK negotiates copy for: the pinned {@link MothConfig.locale}
 * when set, otherwise the live browser language. Sent as `x-moth-language`.
 */
export function currentLocaleOf(config) {
    if (config.locale)
        return config.locale;
    if (typeof navigator !== 'undefined' && navigator.language) {
        return navigator.language;
    }
    return 'en';
}
