/** The BCP-47 language codes the SDK bundles fallback copy for, English first. */
export declare const mothBundledLocales: readonly string[];
/**
 * The bundled fallback copy for `locale`'s language, as a message key ->
 * string map, with English filled in for any key the language lacks (so
 * every bundled-screen key always resolves non-empty). A language the SDK
 * does not bundle returns the full English map.
 */
export declare function bundledCopy(locale: string): Record<string, string>;
