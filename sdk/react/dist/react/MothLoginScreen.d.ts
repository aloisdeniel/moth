export interface MothLoginScreenProps {
    /** Headline override; defaults to the localized mode title. */
    title?: string;
}
/**
 * Batteries-included sign-in / sign-up / forgot-password flow.
 *
 * `MothProvider` shows it by default while signed out. On mount it fetches
 * the project's public config and adapts: the sign-up toggle only appears
 * when public sign-up is open, password validation uses the project's
 * minimum length, and Google/Apple buttons appear per the enabled providers
 * (via the milestone-04 web-redirect flow — requires
 * `MothConfig.projectSlug` and the app's origin registered in the admin
 * under Providers → "Redirect origins (web)"). The OAuth round-trip
 * returns to the current URL with its fragment stripped (the server
 * refuses redirect URIs containing `#`), so hash-routed apps land back on
 * the fragment-less URL after Google/Apple sign-in and should restore
 * their route themselves. Every visual token
 * comes from the project's theme; every string from the negotiated
 * localized copy with the bundled floor as offline fallback.
 */
export declare function MothLoginScreen(props: MothLoginScreenProps): import("react").JSX.Element;
