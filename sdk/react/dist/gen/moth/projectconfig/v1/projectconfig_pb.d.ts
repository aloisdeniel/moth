import type { GenEnum, GenFile, GenMessage } from "@bufbuild/protobuf/codegenv2";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file moth/projectconfig/v1/projectconfig.proto.
 */
export declare const file_moth_projectconfig_v1_projectconfig: GenFile;
/**
 * LegalLinks are the optional legal URLs rendered near signup and on the
 * paywall footer.
 *
 * @generated from message moth.projectconfig.v1.LegalLinks
 */
export type LegalLinks = Message<"moth.projectconfig.v1.LegalLinks"> & {
    /**
     * @generated from field: string terms_url = 1;
     */
    termsUrl: string;
    /**
     * @generated from field: string privacy_url = 2;
     */
    privacyUrl: string;
};
/**
 * Describes the message moth.projectconfig.v1.LegalLinks.
 * Use `create(LegalLinksSchema)` to create a new message.
 */
export declare const LegalLinksSchema: GenMessage<LegalLinks>;
/**
 * ThemeColors is a complete palette: every role and its "on" (foreground)
 * counterpart, as #RRGGBB values.
 *
 * @generated from message moth.projectconfig.v1.ThemeColors
 */
export type ThemeColors = Message<"moth.projectconfig.v1.ThemeColors"> & {
    /**
     * @generated from field: string primary = 1;
     */
    primary: string;
    /**
     * @generated from field: string on_primary = 2;
     */
    onPrimary: string;
    /**
     * @generated from field: string background = 3;
     */
    background: string;
    /**
     * @generated from field: string on_background = 4;
     */
    onBackground: string;
    /**
     * @generated from field: string surface = 5;
     */
    surface: string;
    /**
     * @generated from field: string on_surface = 6;
     */
    onSurface: string;
    /**
     * @generated from field: string error = 7;
     */
    error: string;
    /**
     * @generated from field: string on_error = 8;
     */
    onError: string;
};
/**
 * Describes the message moth.projectconfig.v1.ThemeColors.
 * Use `create(ThemeColorsSchema)` to create a new message.
 */
export declare const ThemeColorsSchema: GenMessage<ThemeColors>;
/**
 * ThemeColorOverrides is a partial dark palette: any empty field is derived
 * from the light palette instead (see internal/theme.DeriveDark).
 *
 * @generated from message moth.projectconfig.v1.ThemeColorOverrides
 */
export type ThemeColorOverrides = Message<"moth.projectconfig.v1.ThemeColorOverrides"> & {
    /**
     * @generated from field: string primary = 1;
     */
    primary: string;
    /**
     * @generated from field: string on_primary = 2;
     */
    onPrimary: string;
    /**
     * @generated from field: string background = 3;
     */
    background: string;
    /**
     * @generated from field: string on_background = 4;
     */
    onBackground: string;
    /**
     * @generated from field: string surface = 5;
     */
    surface: string;
    /**
     * @generated from field: string on_surface = 6;
     */
    onSurface: string;
    /**
     * @generated from field: string error = 7;
     */
    error: string;
    /**
     * @generated from field: string on_error = 8;
     */
    onError: string;
};
/**
 * Describes the message moth.projectconfig.v1.ThemeColorOverrides.
 * Use `create(ThemeColorOverridesSchema)` to create a new message.
 */
export declare const ThemeColorOverridesSchema: GenMessage<ThemeColorOverrides>;
/**
 * ThemeTypography selects one of the curated embedded fonts and a global
 * size multiplier.
 *
 * @generated from message moth.projectconfig.v1.ThemeTypography
 */
export type ThemeTypography = Message<"moth.projectconfig.v1.ThemeTypography"> & {
    /**
     * @generated from field: string font_family = 1;
     */
    fontFamily: string;
    /**
     * @generated from field: double scale = 2;
     */
    scale: number;
};
/**
 * Describes the message moth.projectconfig.v1.ThemeTypography.
 * Use `create(ThemeTypographySchema)` to create a new message.
 */
export declare const ThemeTypographySchema: GenMessage<ThemeTypography>;
/**
 * ThemeSpacing is the base spacing grid step in logical pixels.
 *
 * @generated from message moth.projectconfig.v1.ThemeSpacing
 */
export type ThemeSpacing = Message<"moth.projectconfig.v1.ThemeSpacing"> & {
    /**
     * @generated from field: int32 unit = 1;
     */
    unit: number;
};
/**
 * Describes the message moth.projectconfig.v1.ThemeSpacing.
 * Use `create(ThemeSpacingSchema)` to create a new message.
 */
export declare const ThemeSpacingSchema: GenMessage<ThemeSpacing>;
/**
 * ThemeShape controls component rounding, in logical pixels.
 *
 * @generated from message moth.projectconfig.v1.ThemeShape
 */
export type ThemeShape = Message<"moth.projectconfig.v1.ThemeShape"> & {
    /**
     * @generated from field: int32 corner_radius = 1;
     */
    cornerRadius: number;
};
/**
 * Describes the message moth.projectconfig.v1.ThemeShape.
 * Use `create(ThemeShapeSchema)` to create a new message.
 */
export declare const ThemeShapeSchema: GenMessage<ThemeShape>;
/**
 * ThemeLogo holds the server-managed asset paths of the uploaded logos,
 * one per color scheme ("/assets/{project}/logo-light.png"). Empty = none.
 *
 * @generated from message moth.projectconfig.v1.ThemeLogo
 */
export type ThemeLogo = Message<"moth.projectconfig.v1.ThemeLogo"> & {
    /**
     * @generated from field: string light = 1;
     */
    light: string;
    /**
     * @generated from field: string dark = 2;
     */
    dark: string;
};
/**
 * Describes the message moth.projectconfig.v1.ThemeLogo.
 * Use `create(ThemeLogoSchema)` to create a new message.
 */
export declare const ThemeLogoSchema: GenMessage<ThemeLogo>;
/**
 * StoredTheme is one project's complete design system as persisted on the
 * project row and in theme_revisions (milestone 06, re-encoded from JSON to
 * protobuf). internal/theme owns validation and dark-palette derivation;
 * this message owns only the shape.
 *
 * @generated from message moth.projectconfig.v1.StoredTheme
 */
export type StoredTheme = Message<"moth.projectconfig.v1.StoredTheme"> & {
    /**
     * version is the document schema version (internal/theme.SchemaVersion).
     *
     * @generated from field: int32 version = 1;
     */
    version: number;
    /**
     * @generated from field: moth.projectconfig.v1.ThemeColors colors = 2;
     */
    colors?: ThemeColors | undefined;
    /**
     * dark_colors optionally overrides individual dark-palette colors;
     * absent = fully derived from colors.
     *
     * @generated from field: moth.projectconfig.v1.ThemeColorOverrides dark_colors = 3;
     */
    darkColors?: ThemeColorOverrides | undefined;
    /**
     * @generated from field: moth.projectconfig.v1.ThemeTypography typography = 4;
     */
    typography?: ThemeTypography | undefined;
    /**
     * @generated from field: moth.projectconfig.v1.ThemeSpacing spacing = 5;
     */
    spacing?: ThemeSpacing | undefined;
    /**
     * @generated from field: moth.projectconfig.v1.ThemeShape shape = 6;
     */
    shape?: ThemeShape | undefined;
    /**
     * @generated from field: moth.projectconfig.v1.ThemeLogo logo = 7;
     */
    logo?: ThemeLogo | undefined;
    /**
     * @generated from field: moth.projectconfig.v1.LegalLinks legal = 8;
     */
    legal?: LegalLinks | undefined;
};
/**
 * Describes the message moth.projectconfig.v1.StoredTheme.
 * Use `create(StoredThemeSchema)` to create a new message.
 */
export declare const StoredThemeSchema: GenMessage<StoredTheme>;
/**
 * StoredPaywall is one project's paywall configuration as persisted on the
 * project row and in paywall_revisions (milestone 13, re-encoded from JSON
 * to protobuf). Colors/typography always inherit from the theme — the
 * paywall owns no design tokens.
 *
 * @generated from message moth.projectconfig.v1.StoredPaywall
 */
export type StoredPaywall = Message<"moth.projectconfig.v1.StoredPaywall"> & {
    /**
     * version is the document schema version (internal/paywall.SchemaVersion).
     *
     * @generated from field: int32 version = 1;
     */
    version: number;
    /**
     * @generated from field: string headline = 2;
     */
    headline: string;
    /**
     * @generated from field: string subtitle = 3;
     */
    subtitle: string;
    /**
     * @generated from field: repeated string benefits = 4;
     */
    benefits: string[];
    /**
     * offering names the product offering the paywall presents; empty = the
     * project's default offering.
     *
     * @generated from field: string offering = 5;
     */
    offering: string;
    /**
     * highlighted_identifier marks the "most popular" tier; empty = none.
     *
     * @generated from field: string highlighted_identifier = 6;
     */
    highlightedIdentifier: string;
    /**
     * @generated from field: string layout = 7;
     */
    layout: string;
    /**
     * @generated from field: moth.projectconfig.v1.LegalLinks legal = 8;
     */
    legal?: LegalLinks | undefined;
};
/**
 * Describes the message moth.projectconfig.v1.StoredPaywall.
 * Use `create(StoredPaywallSchema)` to create a new message.
 */
export declare const StoredPaywallSchema: GenMessage<StoredPaywall>;
/**
 * StoredPush is one project's push settings as persisted on the project row
 * (milestone 20). Plain config, no secrets: only the Web Push VAPID PUBLIC
 * key ever lives here — the private key stays with the developer's sender
 * and never touches moth. Delivered to clients through the public
 * moth.auth.v1.GetProjectConfig response.
 *
 * @generated from message moth.projectconfig.v1.StoredPush
 */
export type StoredPush = Message<"moth.projectconfig.v1.StoredPush"> & {
    /**
     * version is the document schema version (internal/push.SchemaVersion).
     *
     * @generated from field: int32 version = 1;
     */
    version: number;
    /**
     * Master switch for the push registry; when false the client-facing
     * moth.push.v1 RPCs refuse registrations.
     *
     * @generated from field: bool enabled = 2;
     */
    enabled: boolean;
    /**
     * VAPID public key (base64url, uncompressed P-256 point) browser clients
     * subscribe with; empty when the project does not use Web Push.
     *
     * @generated from field: string webpush_vapid_public_key = 3;
     */
    webpushVapidPublicKey: string;
};
/**
 * Describes the message moth.projectconfig.v1.StoredPush.
 * Use `create(StoredPushSchema)` to create a new message.
 */
export declare const StoredPushSchema: GenMessage<StoredPush>;
/**
 * StoredProfile is one project's setup profile as persisted on the project
 * row (milestone 22): the creation wizard's answers — platforms, sign-in
 * intent, monetization and push intent. It records what the app *intends*,
 * so surfaces can tell "doesn't want Apple sign-in" apart from "hasn't
 * configured it yet"; it is never a second source of config truth. Absent
 * (empty column) on projects created before the wizard, where surfaces
 * behave exactly as before.
 *
 * @generated from message moth.projectconfig.v1.StoredProfile
 */
export type StoredProfile = Message<"moth.projectconfig.v1.StoredProfile"> & {
    /**
     * version is the document schema version (internal/profile.SchemaVersion).
     *
     * @generated from field: int32 version = 1;
     */
    version: number;
    /**
     * platforms the app ships on. Non-empty in every valid profile; drives
     * every platform branch (store credentials, VAPID, setup snippets).
     *
     * @generated from field: repeated moth.projectconfig.v1.Platform platforms = 2;
     */
    platforms: Platform[];
    /**
     * google_sign_in / apple_sign_in record the social sign-in intent.
     * Email/password is always on and needs no flag.
     *
     * @generated from field: bool google_sign_in = 3;
     */
    googleSignIn: boolean;
    /**
     * @generated from field: bool apple_sign_in = 4;
     */
    appleSignIn: boolean;
    /**
     * sells_subscriptions records the monetization intent (milestones 11/12).
     *
     * @generated from field: bool sells_subscriptions = 5;
     */
    sellsSubscriptions: boolean;
    /**
     * sends_pushes records the push intent (milestone 20).
     *
     * @generated from field: bool sends_pushes = 6;
     */
    sendsPushes: boolean;
    /**
     * checklist_dismissed hides the overview checklist card; it never fakes
     * completeness — the derived items stay computable either way.
     *
     * @generated from field: bool checklist_dismissed = 7;
     */
    checklistDismissed: boolean;
};
/**
 * Describes the message moth.projectconfig.v1.StoredProfile.
 * Use `create(StoredProfileSchema)` to create a new message.
 */
export declare const StoredProfileSchema: GenMessage<StoredProfile>;
/**
 * CopyLocaleMessages is one locale's copy overrides: catalog message key
 * (e.g. "sign_in.title") to the operator-customized string.
 *
 * @generated from message moth.projectconfig.v1.CopyLocaleMessages
 */
export type CopyLocaleMessages = Message<"moth.projectconfig.v1.CopyLocaleMessages"> & {
    /**
     * @generated from field: map<string, string> messages = 1;
     */
    messages: {
        [key: string]: string;
    };
};
/**
 * Describes the message moth.projectconfig.v1.CopyLocaleMessages.
 * Use `create(CopyLocaleMessagesSchema)` to create a new message.
 */
export declare const CopyLocaleMessagesSchema: GenMessage<CopyLocaleMessages>;
/**
 * StoredCopy is one project's localization overrides as persisted on the
 * project row and in copy_revisions (milestone 15, re-encoded from JSON to
 * protobuf): BCP-47 locale tag to that locale's key overrides. Bundled
 * catalog defaults live in the binary (internal/i18n), never here.
 *
 * @generated from message moth.projectconfig.v1.StoredCopy
 */
export type StoredCopy = Message<"moth.projectconfig.v1.StoredCopy"> & {
    /**
     * @generated from field: map<string, moth.projectconfig.v1.CopyLocaleMessages> locales = 1;
     */
    locales: {
        [key: string]: CopyLocaleMessages;
    };
};
/**
 * Describes the message moth.projectconfig.v1.StoredCopy.
 * Use `create(StoredCopySchema)` to create a new message.
 */
export declare const StoredCopySchema: GenMessage<StoredCopy>;
/**
 * CacheEnvelope wraps a config payload the Flutter SDK persists on device
 * (theme, paywall, copy — milestone 16 caches, re-encoded from JSON to
 * protobuf). payload is the serialized wire message exactly as the server
 * delivered it (moth.auth.v1.Theme / moth.billing.v1.Paywall /
 * moth.auth.v1.Copy), so the cache and the wire share one schema. The SDK
 * serves the cached payload without any network call until
 * fetched_at_unix_ms + its configured TTL has passed, then revalidates
 * cheaply with the known_*_revision request fields.
 *
 * @generated from message moth.projectconfig.v1.CacheEnvelope
 */
export type CacheEnvelope = Message<"moth.projectconfig.v1.CacheEnvelope"> & {
    /**
     * @generated from field: bytes payload = 1;
     */
    payload: Uint8Array;
    /**
     * revision is the server revision the payload came from
     * (theme/paywall/copy revision id) — the revalidation key.
     *
     * @generated from field: string revision = 2;
     */
    revision: string;
    /**
     * locale is the negotiated BCP-47 tag for locale-keyed payloads (copy);
     * empty for locale-independent payloads.
     *
     * @generated from field: string locale = 3;
     */
    locale: string;
    /**
     * fetched_at_unix_ms is when the payload was fetched or last revalidated,
     * Unix milliseconds UTC.
     *
     * @generated from field: int64 fetched_at_unix_ms = 4;
     */
    fetchedAtUnixMs: bigint;
};
/**
 * Describes the message moth.projectconfig.v1.CacheEnvelope.
 * Use `create(CacheEnvelopeSchema)` to create a new message.
 */
export declare const CacheEnvelopeSchema: GenMessage<CacheEnvelope>;
/**
 * Platform is an app platform a project ships on — the wizard's platform
 * multi-select (milestone 22). Platforms drive which setup surfaces, wizard
 * branches and checklist items apply to the project.
 *
 * @generated from enum moth.projectconfig.v1.Platform
 */
export declare enum Platform {
    /**
     * @generated from enum value: PLATFORM_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * @generated from enum value: PLATFORM_IOS = 1;
     */
    IOS = 1,
    /**
     * @generated from enum value: PLATFORM_ANDROID = 2;
     */
    ANDROID = 2,
    /**
     * @generated from enum value: PLATFORM_WEB = 3;
     */
    WEB = 3
}
/**
 * Describes the enum moth.projectconfig.v1.Platform.
 */
export declare const PlatformSchema: GenEnum<Platform>;
