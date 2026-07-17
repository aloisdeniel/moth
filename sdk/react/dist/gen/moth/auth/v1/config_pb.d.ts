import type { GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file moth/auth/v1/config.proto.
 */
export declare const file_moth_auth_v1_config: GenFile;
/**
 * GoogleConfig is the public part of a project's Sign in with Google
 * configuration.
 *
 * @generated from message moth.auth.v1.GoogleConfig
 */
export type GoogleConfig = Message<"moth.auth.v1.GoogleConfig"> & {
    /**
     * @generated from field: bool enabled = 1;
     */
    enabled: boolean;
    /**
     * OAuth client IDs the native flows initialize with. Client IDs are
     * public values (the secret never leaves the server).
     *
     * @generated from field: string web_client_id = 2;
     */
    webClientId: string;
    /**
     * @generated from field: string ios_client_id = 3;
     */
    iosClientId: string;
    /**
     * @generated from field: string android_client_id = 4;
     */
    androidClientId: string;
};
/**
 * Describes the message moth.auth.v1.GoogleConfig.
 * Use `create(GoogleConfigSchema)` to create a new message.
 */
export declare const GoogleConfigSchema: GenMessage<GoogleConfig>;
/**
 * AppleConfig is the public part of a project's Sign in with Apple
 * configuration.
 *
 * @generated from message moth.auth.v1.AppleConfig
 */
export type AppleConfig = Message<"moth.auth.v1.AppleConfig"> & {
    /**
     * @generated from field: bool enabled = 1;
     */
    enabled: boolean;
};
/**
 * Describes the message moth.auth.v1.AppleConfig.
 * Use `create(AppleConfigSchema)` to create a new message.
 */
export declare const AppleConfigSchema: GenMessage<AppleConfig>;
/**
 * Theme is the public, fully resolved form of the project's design system,
 * ready to render: dark colors are already derived server-side, asset
 * references are absolute URLs. Binary assets (logo images, font files)
 * stay plain-HTTP downloads with cache headers — they don't belong in RPC
 * responses.
 *
 * @generated from message moth.auth.v1.Theme
 */
export type Theme = Message<"moth.auth.v1.Theme"> & {
    /**
     * Identifies this version of the theme; changes on every admin edit.
     * Cache the theme keyed by this value and echo it as
     * GetProjectConfigRequest.known_theme_revision.
     *
     * @generated from field: string revision_id = 1;
     */
    revisionId: string;
    /**
     * Light palette, "#RRGGBB" values.
     *
     * @generated from field: moth.auth.v1.ThemeColors colors = 2;
     */
    colors?: ThemeColors | undefined;
    /**
     * Dark palette, fully resolved (admin overrides merged with derived
     * values); render it when the device is in dark mode.
     *
     * @generated from field: moth.auth.v1.ThemeColors dark_colors = 3;
     */
    darkColors?: ThemeColors | undefined;
    /**
     * Font family name (from the server's curated set).
     *
     * @generated from field: string font_family = 4;
     */
    fontFamily: string;
    /**
     * Absolute URL of the font file to download and register; cacheable.
     *
     * @generated from field: string font_url = 5;
     */
    fontUrl: string;
    /**
     * Global text-size multiplier.
     *
     * @generated from field: double font_scale = 6;
     */
    fontScale: number;
    /**
     * Base spacing step in logical pixels.
     *
     * @generated from field: int32 spacing_unit = 7;
     */
    spacingUnit: number;
    /**
     * Component corner radius in logical pixels.
     *
     * @generated from field: int32 corner_radius = 8;
     */
    cornerRadius: number;
    /**
     * Absolute logo URLs per color scheme; empty when no logo is set.
     *
     * @generated from field: string logo_light_url = 9;
     */
    logoLightUrl: string;
    /**
     * @generated from field: string logo_dark_url = 10;
     */
    logoDarkUrl: string;
    /**
     * Optional legal links rendered in the login screen footer.
     *
     * @generated from field: string terms_url = 11;
     */
    termsUrl: string;
    /**
     * @generated from field: string privacy_url = 12;
     */
    privacyUrl: string;
};
/**
 * Describes the message moth.auth.v1.Theme.
 * Use `create(ThemeSchema)` to create a new message.
 */
export declare const ThemeSchema: GenMessage<Theme>;
/**
 * ThemeColors is a complete palette: each color role and its "on"
 * (foreground) counterpart. Server-side validation guarantees WCAG AA
 * contrast (>= 4.5:1) between every pair.
 *
 * @generated from message moth.auth.v1.ThemeColors
 */
export type ThemeColors = Message<"moth.auth.v1.ThemeColors"> & {
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
 * Describes the message moth.auth.v1.ThemeColors.
 * Use `create(ThemeColorsSchema)` to create a new message.
 */
export declare const ThemeColorsSchema: GenMessage<ThemeColors>;
/**
 * Copy is the resolved, localized copy for the negotiated locale: the message
 * key → localized-string map the SDK renders its auth screens from
 * (sign_in.*, sign_up.*, password_reset.*, verify_email.*), already merged
 * bundled-default → project-override. The locale is negotiated server-side
 * from the request's Accept-Language / x-moth-language metadata against the
 * project's available locales; the client never dictates raw copy.
 *
 * @generated from message moth.auth.v1.Copy
 */
export type Copy = Message<"moth.auth.v1.Copy"> & {
    /**
     * Opaque cache token identifying this (locale, override-revision) pair. It
     * changes whenever the negotiated locale or the project's copy overrides
     * change. Cache `messages` keyed by this value and echo it as
     * GetProjectConfigRequest.known_copy_revision; the response omits `messages`
     * when it still matches (see the caching contract on the request).
     *
     * @generated from field: string copy_revision = 1;
     */
    copyRevision: string;
    /**
     * The negotiated BCP-47 locale this copy is for (e.g. "fr"). Echoed so the
     * client sets lang/dir correctly and re-requests when the device language
     * changes; always present even when `messages` is omitted.
     *
     * @generated from field: string locale = 2;
     */
    locale: string;
    /**
     * Resolved message key → localized string for the negotiated locale.
     *
     * @generated from field: map<string, string> messages = 3;
     */
    messages: {
        [key: string]: string;
    };
};
/**
 * Describes the message moth.auth.v1.Copy.
 * Use `create(CopySchema)` to create a new message.
 */
export declare const CopySchema: GenMessage<Copy>;
/**
 * @generated from message moth.auth.v1.GetProjectConfigRequest
 */
export type GetProjectConfigRequest = Message<"moth.auth.v1.GetProjectConfigRequest"> & {
    /**
     * Theme caching contract: pass the revision_id of the theme the client
     * has cached (empty on first call). When it still matches the current
     * revision, the response omits `theme` entirely — the client keeps
     * rendering its cached copy. When it differs (or was empty), `theme` is
     * present and the client replaces its cache.
     *
     * @generated from field: string known_theme_revision = 1;
     */
    knownThemeRevision: string;
    /**
     * Copy caching contract (identical shape to the theme one, but keyed by the
     * negotiated locale too): pass the copy_revision the client has cached for
     * the locale it is about to render (empty on first call). When it still
     * matches the token the server computes for the negotiated locale, the
     * response's `copy` carries the locale + copy_revision but omits `messages`
     * (stale-while-revalidate); when it differs (or was empty), `messages` is
     * present and the client replaces its cache. The negotiated locale comes
     * from Accept-Language / x-moth-language metadata, never from this body.
     *
     * @generated from field: string known_copy_revision = 2;
     */
    knownCopyRevision: string;
};
/**
 * Describes the message moth.auth.v1.GetProjectConfigRequest.
 * Use `create(GetProjectConfigRequestSchema)` to create a new message.
 */
export declare const GetProjectConfigRequestSchema: GenMessage<GetProjectConfigRequest>;
/**
 * @generated from message moth.auth.v1.GetProjectConfigResponse
 */
export type GetProjectConfigResponse = Message<"moth.auth.v1.GetProjectConfigResponse"> & {
    /**
     * @generated from field: moth.auth.v1.GoogleConfig google = 1;
     */
    google?: GoogleConfig | undefined;
    /**
     * @generated from field: moth.auth.v1.AppleConfig apple = 2;
     */
    apple?: AppleConfig | undefined;
    /**
     * Minimum accepted password length.
     *
     * @generated from field: int32 password_min_length = 3;
     */
    passwordMinLength: number;
    /**
     * Whether the public SignUp RPC is open.
     *
     * @generated from field: bool sign_up_open = 4;
     */
    signUpOpen: boolean;
    /**
     * The project's design system. Omitted when
     * GetProjectConfigRequest.known_theme_revision matches the current
     * revision (see the caching contract there); always present otherwise,
     * including for projects on the built-in default theme.
     *
     * @generated from field: moth.auth.v1.Theme theme = 5;
     */
    theme?: Theme | undefined;
    /**
     * The localized copy for the negotiated locale. Always present (it carries
     * the negotiated locale + copy_revision so the client caches per (locale,
     * revision)); its `messages` map is omitted when
     * GetProjectConfigRequest.known_copy_revision matches, present otherwise —
     * including for projects with no copy overrides (fully bundled defaults).
     *
     * @generated from field: moth.auth.v1.Copy copy = 6;
     */
    copy?: Copy | undefined;
};
/**
 * Describes the message moth.auth.v1.GetProjectConfigResponse.
 * Use `create(GetProjectConfigResponseSchema)` to create a new message.
 */
export declare const GetProjectConfigResponseSchema: GenMessage<GetProjectConfigResponse>;
/**
 * ConfigService exposes a project's public, non-secret configuration to the
 * mobile SDK, so the login screen can render exactly the sign-in methods
 * the project enables. Authenticated like AuthService: every call carries
 * the project's publishable key in `x-moth-key: pk_...` request metadata.
 *
 * Later milestones extend GetProjectConfigResponse: SDK bootstrap values in
 * 05, login-screen branding/theme in 06. Fields are only ever added.
 *
 * @generated from service moth.auth.v1.ConfigService
 */
export declare const ConfigService: GenService<{
    /**
     * GetProjectConfig returns the project configuration a client may see.
     * Never includes secrets; only values that are safe to embed in an app.
     *
     * @generated from rpc moth.auth.v1.ConfigService.GetProjectConfig
     */
    getProjectConfig: {
        methodKind: "unary";
        input: typeof GetProjectConfigRequestSchema;
        output: typeof GetProjectConfigResponseSchema;
    };
}>;
