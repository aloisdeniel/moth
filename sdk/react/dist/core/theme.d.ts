import type { Theme } from '../gen/moth/auth/v1/config_pb.js';
/**
 * One complete palette of the moth design system: each color role and its
 * "on" (foreground) counterpart, as `#RRGGBB` strings. Server-side
 * validation guarantees WCAG AA contrast between every pair.
 */
export interface MothThemeColors {
    primary: string;
    onPrimary: string;
    background: string;
    onBackground: string;
    surface: string;
    onSurface: string;
    error: string;
    onError: string;
}
/**
 * A project's design system, fully resolved and ready to render: the public
 * form of the theme configured in the moth admin (delivered inside
 * `GetProjectConfig`). The moth components consume it exclusively — as CSS
 * custom properties — so a project's brand applies without a release.
 */
export interface MothTheme {
    /**
     * Identifies this version of the theme; echoed as `known_theme_revision`
     * so an unchanged theme is not re-sent. Empty for the fallback.
     */
    revisionId: string;
    /** Light palette. */
    colors: MothThemeColors;
    /** Dark palette (admin overrides merged server-side, or derived locally). */
    darkColors: MothThemeColors;
    /** Font family display name (from the server's curated set). */
    fontFamily: string;
    /** Absolute URL of the font file to load; undefined = system font. */
    fontUrl?: string;
    /** Global text-size multiplier. */
    fontScale: number;
    /** Base spacing step in CSS pixels. */
    spacingUnit: number;
    /** Component corner radius in CSS pixels. */
    cornerRadius: number;
    /** Absolute logo URLs per color scheme. */
    logoLightUrl?: string;
    logoDarkUrl?: string;
    /** Optional legal links rendered in the login screen footer. */
    termsUrl?: string;
    privacyUrl?: string;
}
/**
 * The theme every project starts from (and the offline fallback when
 * nothing is cached yet): the server's built-in default.
 */
export declare function fallbackTheme(): MothTheme;
/**
 * Maps the theme message from `GetProjectConfig`. Fields an older server
 * leaves empty fall back to the defaults, and a missing dark palette is
 * derived locally with the same algorithm the server uses.
 */
export declare function themeFromProto(proto: Theme): MothTheme;
/**
 * The CSS custom properties for `theme`, ready to set on the `.moth-root`
 * wrapper. Light and dark palettes are emitted side by side
 * (`--moth-l-*` / `--moth-d-*`); the injected stylesheet resolves the live
 * `--moth-*` tokens from them per `prefers-color-scheme`.
 */
export declare function themeCssVars(theme: MothTheme): Record<string, string>;
/**
 * Loads the theme's font file (when it names one) into `document.fonts` so
 * the moth surfaces render it. Idempotent per URL; failures are swallowed —
 * text simply stays on the system font.
 */
export declare function ensureThemeFont(theme: MothTheme): Promise<void>;
/** Parses a strict `#RRGGBB` hex color into [r, g, b]; null when malformed. */
export declare function parseHexColor(hex: string): [number, number, number] | null;
/**
 * Derives a dark palette from a light one with the exact algorithm the
 * server uses (internal/theme/derive.go): background and surface blend
 * 88% / 84% toward black, primary and error blend 40% toward white, and
 * every on* color becomes black or white — whichever contrasts more.
 */
export declare function deriveDarkColors(light: MothThemeColors): MothThemeColors;
/** WCAG 2.x contrast ratio between two `#RRGGBB` colors (1..21). */
export declare function contrastRatio(a: string, b: string): number;
