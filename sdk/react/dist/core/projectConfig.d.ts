import type { MothCopyUpdate } from './copy.js';
import type { MothTheme } from './theme.js';
/**
 * A project's public, non-secret configuration
 * (`moth.auth.v1.ConfigService.GetProjectConfig`), so login UI can render
 * exactly the sign-in methods the project enables without a release.
 */
export interface MothProjectConfig {
    google: MothGoogleConfig;
    apple: MothAppleConfig;
    /** Minimum accepted password length. */
    passwordMinLength: number;
    /** Whether the public sign-up RPC is open. */
    signUpOpen: boolean;
    /**
     * The project's design system, or undefined when the server confirmed the
     * `knownThemeRevision` is still current (keep the cached theme).
     */
    theme?: MothTheme;
    /**
     * The localized copy for the negotiated locale (locale + revision always
     * present, `messages` only when the revision differed), or undefined when
     * the server predates localized copy.
     */
    copy?: MothCopyUpdate;
}
/**
 * Public part of the project's Sign in with Google configuration. Client IDs
 * are public values; the secret never leaves the server.
 */
export interface MothGoogleConfig {
    enabled: boolean;
    webClientId?: string;
    iosClientId?: string;
    androidClientId?: string;
}
/** Public part of the project's Sign in with Apple configuration. */
export interface MothAppleConfig {
    enabled: boolean;
}
