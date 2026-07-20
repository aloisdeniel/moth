import { create, fromJson, toJson } from "@bufbuild/protobuf";
import type { JsonValue, MessageInitShape } from "@bufbuild/protobuf";

import { CopyScreen, CopyService } from "../../gen/moth/admin/v1/copy_pb";
import { PaywallLayout, PaywallService, PaywallConfigSchema } from "../../gen/moth/admin/v1/paywall_pb";
import type { PaywallConfig } from "../../gen/moth/admin/v1/paywall_pb";
import {
  LogoVariant,
  ThemeLogoSchema,
  ThemeSchema,
  ThemeService,
} from "../../gen/moth/admin/v1/theme_pb";
import type { Theme } from "../../gen/moth/admin/v1/theme_pb";
import { PROJECT_MAIN, PROJECT_SIDE, demoId } from "../ids";
import { handle } from "../transport";
import { daysAgo, invalidArgument, notFound, now, randomId, ts } from "../util";
import type { Millis } from "../util";

// Content slice: per-project theme tokens, localization copy overrides and
// paywall config — the three "protobuf blob" configs (CopyService,
// ThemeService, PaywallService). Theme and paywall payloads are stored as
// proto JSON (toJson/fromJson of the generated schemas) so round-trips are
// lossless and the document stays plain JSON; logo images are stored as
// base64 data: URIs (the SPA renders ThemeLogo paths directly as <img src>,
// so a data URI displays without a server).

// ---------- State ----------

interface StoredLogo {
  // A full data: URI ("data:image/png;base64,…") — served back verbatim as
  // the ThemeLogo light/dark path.
  dataUri: string;
  contentType: string;
}

interface ThemeRevisionState {
  revisionId: string;
  createTime: Millis;
  // moth.admin.v1.Theme as proto JSON (logo excluded; logos are per-project).
  theme: JsonValue;
}

interface ThemeState {
  // Current moth.admin.v1.Theme as proto JSON (logo excluded); null renders
  // the built-in defaults.
  current: JsonValue | null;
  revisionId: string;
  // Newest first, at most REVISION_KEEP.
  revisions: ThemeRevisionState[];
  logoLight: StoredLogo | null;
  logoDark: StoredLogo | null;
}

// Copy overrides document: locale tag -> catalog key -> override string.
type CopyDoc = Record<string, Record<string, string>>;

interface CopyRevisionState {
  revisionId: string;
  createTime: Millis;
  overrides: CopyDoc;
}

interface CopyState {
  overrides: CopyDoc;
  revisionId: string;
  revisions: CopyRevisionState[];
}

interface PaywallRevisionState {
  revisionId: string;
  createTime: Millis;
  // moth.admin.v1.PaywallConfig as proto JSON.
  config: JsonValue;
}

interface PaywallState {
  current: JsonValue | null;
  revisionId: string;
  revisions: PaywallRevisionState[];
}

export interface ContentSlice {
  themes: Record<string, ThemeState>;
  copies: Record<string, CopyState>;
  paywalls: Record<string, PaywallState>;
}

// The server keeps the last 10 revisions of each config for undo.
const REVISION_KEEP = 10;

// ---------- Bundled copy catalog (mirror of internal/i18n, editor screens) ----------

// The five screens the admin copy editor exposes, with the bundled English
// and French strings. Other bundled locales are not embedded in the demo:
// their editors fall back to the English defaults.
const CATALOG: { key: string; screen: CopyScreen; en: string; fr: string }[] = [
  // sign_in
  { key: "sign_in.title", screen: CopyScreen.SIGN_IN, en: "Sign in", fr: "Connexion" },
  { key: "sign_in.subtitle", screen: CopyScreen.SIGN_IN, en: "Welcome back to {app}.", fr: "Bon retour sur {app}." },
  { key: "sign_in.email_label", screen: CopyScreen.SIGN_IN, en: "Email", fr: "E-mail" },
  { key: "sign_in.password_label", screen: CopyScreen.SIGN_IN, en: "Password", fr: "Mot de passe" },
  { key: "sign_in.submit", screen: CopyScreen.SIGN_IN, en: "Sign in", fr: "Se connecter" },
  { key: "sign_in.forgot_password", screen: CopyScreen.SIGN_IN, en: "Forgot password?", fr: "Mot de passe oublié ?" },
  { key: "sign_in.no_account", screen: CopyScreen.SIGN_IN, en: "Don't have an account?", fr: "Vous n'avez pas de compte ?" },
  { key: "sign_in.switch_to_sign_up", screen: CopyScreen.SIGN_IN, en: "Sign up", fr: "S'inscrire" },
  { key: "sign_in.error_invalid", screen: CopyScreen.SIGN_IN, en: "Incorrect email or password.", fr: "E-mail ou mot de passe incorrect." },
  // sign_up
  { key: "sign_up.title", screen: CopyScreen.SIGN_UP, en: "Create account", fr: "Créer un compte" },
  { key: "sign_up.subtitle", screen: CopyScreen.SIGN_UP, en: "Create your {app} account.", fr: "Créez votre compte {app}." },
  { key: "sign_up.email_label", screen: CopyScreen.SIGN_UP, en: "Email", fr: "E-mail" },
  { key: "sign_up.password_label", screen: CopyScreen.SIGN_UP, en: "Password", fr: "Mot de passe" },
  { key: "sign_up.submit", screen: CopyScreen.SIGN_UP, en: "Create account", fr: "Créer un compte" },
  { key: "sign_up.have_account", screen: CopyScreen.SIGN_UP, en: "Already have an account?", fr: "Vous avez déjà un compte ?" },
  { key: "sign_up.switch_to_sign_in", screen: CopyScreen.SIGN_UP, en: "Sign in", fr: "Se connecter" },
  { key: "sign_up.legal", screen: CopyScreen.SIGN_UP, en: "By continuing you agree to our Terms and Privacy Policy.", fr: "En continuant, vous acceptez nos conditions d'utilisation et notre politique de confidentialité." },
  { key: "sign_up.error_email_taken", screen: CopyScreen.SIGN_UP, en: "An account with this email already exists.", fr: "Un compte avec cet e-mail existe déjà." },
  // password_reset
  { key: "password_reset.title", screen: CopyScreen.PASSWORD_RESET, en: "Reset password", fr: "Réinitialiser le mot de passe" },
  { key: "password_reset.subtitle", screen: CopyScreen.PASSWORD_RESET, en: "Enter your email and we'll send you a reset link.", fr: "Saisissez votre e-mail et nous vous enverrons un lien de réinitialisation." },
  { key: "password_reset.email_label", screen: CopyScreen.PASSWORD_RESET, en: "Email", fr: "E-mail" },
  { key: "password_reset.submit", screen: CopyScreen.PASSWORD_RESET, en: "Send reset link", fr: "Envoyer le lien" },
  { key: "password_reset.back_to_sign_in", screen: CopyScreen.PASSWORD_RESET, en: "Back to sign in", fr: "Retour à la connexion" },
  { key: "password_reset.sent", screen: CopyScreen.PASSWORD_RESET, en: "If an account exists for {email}, a reset link is on its way.", fr: "Si un compte existe pour {email}, un lien de réinitialisation est en route." },
  { key: "password_reset.new_password_label", screen: CopyScreen.PASSWORD_RESET, en: "New password", fr: "Nouveau mot de passe" },
  { key: "password_reset.new_password_submit", screen: CopyScreen.PASSWORD_RESET, en: "Set new password", fr: "Définir le mot de passe" },
  { key: "password_reset.success", screen: CopyScreen.PASSWORD_RESET, en: "Your password has been reset.", fr: "Votre mot de passe a été réinitialisé." },
  // verify_email
  { key: "verify_email.title", screen: CopyScreen.VERIFY_EMAIL, en: "Verify your email", fr: "Vérifiez votre e-mail" },
  { key: "verify_email.subtitle", screen: CopyScreen.VERIFY_EMAIL, en: "We sent a verification link to {email}.", fr: "Nous avons envoyé un lien de vérification à {email}." },
  { key: "verify_email.resend", screen: CopyScreen.VERIFY_EMAIL, en: "Resend email", fr: "Renvoyer l'e-mail" },
  { key: "verify_email.success", screen: CopyScreen.VERIFY_EMAIL, en: "Your email address is verified.", fr: "Votre adresse e-mail est vérifiée." },
  { key: "verify_email.expired", screen: CopyScreen.VERIFY_EMAIL, en: "This verification link has expired.", fr: "Ce lien de vérification a expiré." },
  // paywall
  { key: "paywall.title", screen: CopyScreen.PAYWALL, en: "Unlock {app} Premium", fr: "Débloquez {app} Premium" },
  { key: "paywall.subtitle", screen: CopyScreen.PAYWALL, en: "Get unlimited access to every feature.", fr: "Accédez sans limite à toutes les fonctionnalités." },
  { key: "paywall.cta", screen: CopyScreen.PAYWALL, en: "Continue", fr: "Continuer" },
  { key: "paywall.restore", screen: CopyScreen.PAYWALL, en: "Restore purchases", fr: "Restaurer les achats" },
  { key: "paywall.terms", screen: CopyScreen.PAYWALL, en: "Payment is charged to your account. Cancel anytime.", fr: "Le paiement est débité de votre compte. Annulez à tout moment." },
];

const CATALOG_BY_KEY = new Map(CATALOG.map((e) => [e.key, e]));

// The rune-length cap internal/i18n enforces on any override string.
const MAX_COPY_VALUE_LENGTH = 400;

const DEFAULT_LOCALE = "en";

// The locales the demo bundles translations for (the full server set is
// larger; see CATALOG above).
const BUNDLED_LOCALES = ["en", "fr"];

const LOCALE_DISPLAY_NAMES: Record<string, string> = { en: "English", fr: "French" };

const PLACEHOLDER_RE = /\{([a-zA-Z][a-zA-Z0-9_]*)\}/g;

// requiredPlaceholders derives a key's placeholder contract from its English
// default, exactly like the server.
function requiredPlaceholders(en: string): string[] {
  const seen = new Set<string>();
  for (const m of en.matchAll(PLACEHOLDER_RE)) seen.add(m[1]);
  return [...seen].sort();
}

// bundledValue returns the bundled default for one key and locale (English
// for locales the demo does not embed).
function bundledValue(entry: (typeof CATALOG)[number], locale: string): string {
  return locale === "fr" ? entry.fr : entry.en;
}

// normalizeLocale lower-cases the tag and collapses regioned variants of the
// demo-bundled locales onto their primary subtag ("fr-CA" -> "fr").
function normalizeLocale(raw: string): string {
  const tag = raw.trim().toLowerCase().replace(/_/g, "-");
  const primary = tag.split("-")[0];
  return BUNDLED_LOCALES.includes(primary) ? primary : tag;
}

// ---------- Built-in defaults (mirrors internal/theme and internal/paywall) ----------

function defaultThemeInit(): MessageInitShape<typeof ThemeSchema> {
  return {
    colors: {
      primary: "#6750A4",
      onPrimary: "#FFFFFF",
      background: "#FFFBFE",
      onBackground: "#1C1B1F",
      surface: "#FFFBFE",
      onSurface: "#1C1B1F",
      error: "#B3261E",
      onError: "#FFFFFF",
    },
    typography: { fontFamily: "Inter", scale: 1.0 },
    spacing: { unit: 8 },
    shape: { cornerRadius: 12 },
    legal: { termsUrl: "", privacyUrl: "" },
  };
}

function defaultPaywallInit(): MessageInitShape<typeof PaywallConfigSchema> {
  return {
    headline: "Unlock Premium",
    subtitle: "Get the full experience with a subscription.",
    benefits: [
      "Unlimited access to every feature",
      "Priority support",
      "New features first",
    ],
    offering: "",
    highlightedProductIdentifier: "",
    layout: PaywallLayout.TILES,
    legal: { termsUrl: "", privacyUrl: "" },
  };
}

// ---------- Small helpers ----------

function clone<T>(v: T): T {
  return JSON.parse(JSON.stringify(v)) as T;
}

function emptyThemeState(): ThemeState {
  return { current: null, revisionId: "", revisions: [], logoLight: null, logoDark: null };
}

function emptyCopyState(): CopyState {
  return { overrides: {}, revisionId: "", revisions: [] };
}

function emptyPaywallState(): PaywallState {
  return { current: null, revisionId: "", revisions: [] };
}

// themeState/copyState/paywallState lazily create a defaults entry, so
// projects created interactively (the wizard) work without cross-slice
// coordination.
function themeState(s: ContentSlice, projectId: string): ThemeState {
  if (projectId === "") throw notFound("project");
  let t = s.themes[projectId];
  if (!t) {
    t = emptyThemeState();
    s.themes[projectId] = t;
  }
  return t;
}

function copyState(s: ContentSlice, projectId: string): CopyState {
  if (projectId === "") throw notFound("project");
  let c = s.copies[projectId];
  if (!c) {
    c = emptyCopyState();
    s.copies[projectId] = c;
  }
  return c;
}

function paywallState(s: ContentSlice, projectId: string): PaywallState {
  if (projectId === "") throw notFound("project");
  let p = s.paywalls[projectId];
  if (!p) {
    p = emptyPaywallState();
    s.paywalls[projectId] = p;
  }
  return p;
}

// themeMessage rebuilds the current (or a revision's) Theme message from the
// stored proto JSON and injects the per-project logo data URIs, mirroring the
// server (logo assets are per project, not per revision).
function themeMessage(t: ThemeState, themeJson: JsonValue | null): Theme {
  const msg =
    themeJson !== null
      ? fromJson(ThemeSchema, themeJson)
      : create(ThemeSchema, defaultThemeInit());
  msg.logo = create(ThemeLogoSchema, {
    lightPath: t.logoLight?.dataUri ?? "",
    darkPath: t.logoDark?.dataUri ?? "",
  });
  return msg;
}

// storeTheme serializes an incoming Theme message as proto JSON with the
// output-only logo field stripped (logos are managed by UploadLogo/DeleteLogo).
function storeTheme(theme: Theme): JsonValue {
  const json = toJson(ThemeSchema, theme);
  if (json !== null && typeof json === "object" && !Array.isArray(json)) {
    delete (json as Record<string, JsonValue>).logo;
  }
  return json;
}

function pushThemeRevision(t: ThemeState, themeJson: JsonValue, at: Millis): string {
  const id = randomId();
  t.revisions.unshift({ revisionId: id, createTime: at, theme: clone(themeJson) });
  t.revisions = t.revisions.slice(0, REVISION_KEEP);
  t.current = clone(themeJson);
  t.revisionId = id;
  return id;
}

function paywallMessage(configJson: JsonValue | null): PaywallConfig {
  return configJson !== null
    ? fromJson(PaywallConfigSchema, configJson)
    : create(PaywallConfigSchema, defaultPaywallInit());
}

function pushPaywallRevision(p: PaywallState, configJson: JsonValue, at: Millis): string {
  const id = randomId();
  p.revisions.unshift({ revisionId: id, createTime: at, config: clone(configJson) });
  p.revisions = p.revisions.slice(0, REVISION_KEEP);
  p.current = clone(configJson);
  p.revisionId = id;
  return id;
}

// pushCopyRevision snapshots the (already-mutated) overrides document as a
// new revision. A document with no overrides at all is the bundled default:
// the revision is still recorded, but the current revision id goes empty,
// mirroring the server's contract.
function pushCopyRevision(c: CopyState, at: Millis): string {
  const id = randomId();
  c.revisions.unshift({ revisionId: id, createTime: at, overrides: clone(c.overrides) });
  c.revisions = c.revisions.slice(0, REVISION_KEEP);
  const empty = Object.keys(c.overrides).length === 0;
  c.revisionId = empty ? "" : id;
  return c.revisionId;
}

function bytesToBase64(data: Uint8Array): string {
  let s = "";
  const chunk = 0x8000;
  for (let i = 0; i < data.length; i += chunk) {
    s += String.fromCharCode(...Array.from(data.subarray(i, i + chunk)));
  }
  return btoa(s);
}

function isHttpUrl(s: string): boolean {
  return /^https?:\/\//.test(s);
}

// ---------- Seed ----------

const AURORA_TERMS = "https://aurorajournal.app/terms";
const AURORA_PRIVACY = "https://aurorajournal.app/privacy";

// Aurora's theme history: an early brand pass, the indigo/Lora identity, and
// the current night-sky revision with an explicit dark palette. Every palette
// pair clears WCAG AA (the editor blocks saves below 4.5:1).
function auroraThemeRev1(): JsonValue {
  return toJson(
    ThemeSchema,
    create(ThemeSchema, {
      colors: {
        primary: "#5B4FC8",
        onPrimary: "#FFFFFF",
        background: "#FFFFFF",
        onBackground: "#1B1B22",
        surface: "#FFFFFF",
        onSurface: "#1B1B22",
        error: "#B3261E",
        onError: "#FFFFFF",
      },
      typography: { fontFamily: "Inter", scale: 1.0 },
      spacing: { unit: 8 },
      shape: { cornerRadius: 12 },
      legal: { termsUrl: "", privacyUrl: "" },
    }),
  );
}

function auroraThemeRev2(): JsonValue {
  return toJson(
    ThemeSchema,
    create(ThemeSchema, {
      colors: {
        primary: "#4338CA",
        onPrimary: "#FFFFFF",
        background: "#F8FAFF",
        onBackground: "#1E1B2E",
        surface: "#FFFFFF",
        onSurface: "#232136",
        error: "#B3261E",
        onError: "#FFFFFF",
      },
      typography: { fontFamily: "Lora", scale: 1.0 },
      spacing: { unit: 8 },
      shape: { cornerRadius: 12 },
      legal: { termsUrl: AURORA_TERMS, privacyUrl: AURORA_PRIVACY },
    }),
  );
}

function auroraThemeCurrent(): JsonValue {
  return toJson(
    ThemeSchema,
    create(ThemeSchema, {
      colors: {
        primary: "#4338CA",
        onPrimary: "#FFFFFF",
        background: "#F8FAFF",
        onBackground: "#1E1B2E",
        surface: "#FFFFFF",
        onSurface: "#232136",
        error: "#B3261E",
        onError: "#FFFFFF",
      },
      // A hand-tuned night-sky dark palette: deep navy surfaces with an
      // indigo-glow accent, fitting the aurora story.
      darkColors: {
        primary: "#8B9DFF",
        onPrimary: "#0A0E2A",
        background: "#0B1026",
        onBackground: "#E4E9FF",
        surface: "#151B36",
        onSurface: "#DDE3FA",
        error: "#FFB4AB",
        onError: "#3B0907",
      },
      typography: { fontFamily: "Lora", scale: 1.05 },
      spacing: { unit: 8 },
      shape: { cornerRadius: 16 },
      legal: { termsUrl: AURORA_TERMS, privacyUrl: AURORA_PRIVACY },
    }),
  );
}

// Aurora's current copy overrides: the default locale carries the branded
// headlines (placeholder contracts preserved), French pins the two most
// visible strings so the locale shows as customized.
function auroraCopyCurrent(): CopyDoc {
  return {
    en: {
      "sign_in.title": "Welcome back to Aurora Journal",
      "sign_in.subtitle": "Your journal is ready — pick up tonight's entry in {app}.",
      "sign_up.subtitle": "Create your {app} account and start your first entry.",
      "paywall.title": "Unlock {app} Pro",
      "paywall.subtitle": "Unlimited entries, mood insights and encrypted sync.",
    },
    fr: {
      "sign_in.title": "Bon retour sur Aurora Journal",
      "paywall.title": "Débloquez {app} Pro",
    },
  };
}

function auroraPaywallRev1(): JsonValue {
  return toJson(
    PaywallConfigSchema,
    create(PaywallConfigSchema, {
      headline: "Unlock Aurora Pro",
      subtitle: "Keep every memory, beautifully organized.",
      benefits: [
        "Unlimited journal entries",
        "Mood trends & insights",
        "Encrypted sync",
      ],
      offering: "",
      highlightedProductIdentifier: "",
      layout: PaywallLayout.TILES,
      legal: { termsUrl: "", privacyUrl: "" },
    }),
  );
}

function auroraPaywallCurrent(): JsonValue {
  return toJson(
    PaywallConfigSchema,
    create(PaywallConfigSchema, {
      headline: "Go deeper with Aurora Pro",
      subtitle: "Unlimited entries, mood trends and encrypted sync across all your devices.",
      benefits: [
        "Unlimited journal entries",
        "Mood trends & yearly insights",
        "End-to-end encrypted sync",
        "Export to PDF & Markdown",
      ],
      offering: "",
      // The Aurora Pro annual tier (a catalog identifier, never a store SKU).
      highlightedProductIdentifier: "annual",
      layout: PaywallLayout.TILES,
      legal: { termsUrl: AURORA_TERMS, privacyUrl: AURORA_PRIVACY },
    }),
  );
}

export function seedContent(): ContentSlice {
  const themeRevisionIds = [demoId("them", 1), demoId("them", 2), demoId("them", 3)];
  const copyRevisionIds = [demoId("cpyr", 1), demoId("cpyr", 2), demoId("cpyr", 3)];
  const paywallRevisionIds = [demoId("payw", 1), demoId("payw", 2)];

  const auroraTheme: ThemeState = {
    current: auroraThemeCurrent(),
    revisionId: themeRevisionIds[2],
    // Newest first.
    revisions: [
      { revisionId: themeRevisionIds[2], createTime: daysAgo(9), theme: auroraThemeCurrent() },
      { revisionId: themeRevisionIds[1], createTime: daysAgo(30), theme: auroraThemeRev2() },
      { revisionId: themeRevisionIds[0], createTime: daysAgo(62), theme: auroraThemeRev1() },
    ],
    // No seeded logo: uploads land as data: URIs (see UploadLogo below); the
    // preview's monogram fallback covers the empty state.
    logoLight: null,
    logoDark: null,
  };

  const auroraCopy: CopyState = {
    overrides: auroraCopyCurrent(),
    revisionId: copyRevisionIds[2],
    revisions: [
      { revisionId: copyRevisionIds[2], createTime: daysAgo(2), overrides: auroraCopyCurrent() },
      {
        revisionId: copyRevisionIds[1],
        createTime: daysAgo(12),
        overrides: {
          en: {
            "sign_in.title": "Welcome back to Aurora Journal",
            "sign_in.subtitle": "Your journal is ready — pick up tonight's entry in {app}.",
            "paywall.title": "Unlock {app} Pro",
          },
        },
      },
      {
        revisionId: copyRevisionIds[0],
        createTime: daysAgo(40),
        overrides: { en: { "sign_in.title": "Welcome back to Aurora Journal" } },
      },
    ],
  };

  const auroraPaywall: PaywallState = {
    current: auroraPaywallCurrent(),
    revisionId: paywallRevisionIds[1],
    revisions: [
      { revisionId: paywallRevisionIds[1], createTime: daysAgo(8), config: auroraPaywallCurrent() },
      { revisionId: paywallRevisionIds[0], createTime: daysAgo(30), config: auroraPaywallRev1() },
    ],
  };

  return {
    themes: {
      [PROJECT_MAIN.id]: auroraTheme,
      // Skylark never customized anything: built-in defaults everywhere.
      [PROJECT_SIDE.id]: emptyThemeState(),
    },
    copies: {
      [PROJECT_MAIN.id]: auroraCopy,
      [PROJECT_SIDE.id]: emptyCopyState(),
    },
    paywalls: {
      [PROJECT_MAIN.id]: auroraPaywall,
      [PROJECT_SIDE.id]: emptyPaywallState(),
    },
  };
}

// ---------- Handlers ----------

export function registerContent(): void {
  // ----- ThemeService -----

  handle(ThemeService.method.getTheme, (state: ContentSlice, req) => {
    const t = themeState(state, req.projectId);
    return {
      theme: themeMessage(t, t.current),
      revisionId: t.revisionId,
      isDefault: t.current === null,
    };
  });

  handle(ThemeService.method.updateTheme, (state: ContentSlice, req) => {
    if (!req.theme) throw invalidArgument("a theme is required");
    const c = req.theme.colors;
    if (!c) throw invalidArgument("a full light palette is required");
    const hex = /^#[0-9a-fA-F]{6}$/;
    for (const v of [
      c.primary, c.onPrimary, c.background, c.onBackground,
      c.surface, c.onSurface, c.error, c.onError,
    ]) {
      if (!hex.test(v)) throw invalidArgument(`invalid color ${JSON.stringify(v)}: want #RRGGBB`);
    }
    const legal = req.theme.legal;
    for (const u of [legal?.termsUrl ?? "", legal?.privacyUrl ?? ""]) {
      if (u !== "" && !isHttpUrl(u)) {
        throw invalidArgument("legal links must be absolute http(s) URLs");
      }
    }
    const t = themeState(state, req.projectId);
    const revisionId = pushThemeRevision(t, storeTheme(req.theme), now());
    return { theme: themeMessage(t, t.current), revisionId };
  });

  handle(ThemeService.method.listThemeRevisions, (state: ContentSlice, req) => {
    const t = themeState(state, req.projectId);
    const limit = req.limit > 0 ? req.limit : t.revisions.length;
    return {
      revisions: t.revisions.slice(0, limit).map((rev) => ({
        revisionId: rev.revisionId,
        theme: themeMessage(t, rev.theme),
        createTime: ts(rev.createTime),
      })),
    };
  });

  handle(ThemeService.method.restoreThemeRevision, (state: ContentSlice, req) => {
    const t = themeState(state, req.projectId);
    const rev = t.revisions.find((r) => r.revisionId === req.revisionId);
    if (!rev) throw notFound("theme revision");
    const revisionId = pushThemeRevision(t, rev.theme, now());
    return { theme: themeMessage(t, t.current), revisionId };
  });

  handle(ThemeService.method.resetTheme, (state: ContentSlice, req) => {
    const t = themeState(state, req.projectId);
    t.current = null;
    t.revisionId = "";
    return { theme: themeMessage(t, null) };
  });

  handle(ThemeService.method.uploadLogo, (state: ContentSlice, req) => {
    if (req.variant !== LogoVariant.LIGHT && req.variant !== LogoVariant.DARK) {
      throw invalidArgument("a logo variant is required");
    }
    if (req.contentType !== "image/png" && req.contentType !== "image/svg+xml") {
      throw invalidArgument("logo must be image/png or image/svg+xml");
    }
    if (req.data.length === 0) throw invalidArgument("logo image is empty");
    if (req.data.length > 512 * 1024) {
      throw invalidArgument("logo is too large — the limit is 512 KiB");
    }
    const t = themeState(state, req.projectId);
    const logo: StoredLogo = {
      dataUri: `data:${req.contentType};base64,${bytesToBase64(req.data)}`,
      contentType: req.contentType,
    };
    if (req.variant === LogoVariant.LIGHT) t.logoLight = logo;
    else t.logoDark = logo;
    // An upload creates a theme revision, like the server; on a never-themed
    // project the saved document is the built-in default token set.
    const base = t.current ?? toJson(ThemeSchema, create(ThemeSchema, defaultThemeInit()));
    const revisionId = pushThemeRevision(t, base, now());
    return { theme: themeMessage(t, t.current), revisionId, path: logo.dataUri };
  });

  handle(ThemeService.method.deleteLogo, (state: ContentSlice, req) => {
    if (req.variant !== LogoVariant.LIGHT && req.variant !== LogoVariant.DARK) {
      throw invalidArgument("a logo variant is required");
    }
    const t = themeState(state, req.projectId);
    if (req.variant === LogoVariant.LIGHT) t.logoLight = null;
    else t.logoDark = null;
    const base = t.current ?? toJson(ThemeSchema, create(ThemeSchema, defaultThemeInit()));
    const revisionId = pushThemeRevision(t, base, now());
    return { theme: themeMessage(t, t.current), revisionId };
  });

  // ----- CopyService -----

  handle(CopyService.method.getProjectCopy, (state: ContentSlice, req) => {
    const c = copyState(state, req.projectId);
    const locale = normalizeLocale(req.locale === "" ? DEFAULT_LOCALE : req.locale);
    const entries =
      req.screen === CopyScreen.UNSPECIFIED
        ? CATALOG
        : CATALOG.filter((e) => e.screen === req.screen);
    if (entries.length === 0) throw invalidArgument(`unknown copy screen ${req.screen}`);
    const overrides = c.overrides[locale] ?? {};
    // A non-default locale inherits the default locale's override as its
    // effective default (layer 2 of the server's resolution).
    const defOverrides = locale === DEFAULT_LOCALE ? {} : (c.overrides[DEFAULT_LOCALE] ?? {});
    return {
      keys: entries.map((e) => {
        let def = bundledValue(e, locale);
        const dv = defOverrides[e.key];
        if (dv !== undefined && dv !== "") def = dv;
        return {
          key: e.key,
          screen: e.screen,
          defaultValue: def,
          overrideValue: overrides[e.key] ?? "",
          placeholders: requiredPlaceholders(e.en),
          maxLength: MAX_COPY_VALUE_LENGTH,
        };
      }),
      locale,
      revisionId: c.revisionId,
      isDefault: c.revisionId === "",
    };
  });

  handle(CopyService.method.updateProjectCopy, (state: ContentSlice, req) => {
    if (req.locale === "") throw invalidArgument("locale is required");
    const locale = normalizeLocale(req.locale);
    const c = copyState(state, req.projectId);
    // Validate against the catalog before mutating anything.
    for (const [key, value] of Object.entries(req.values)) {
      const entry = CATALOG_BY_KEY.get(key);
      if (!entry) throw invalidArgument(`unknown copy key "${key}"`);
      if (value === "") continue;
      if (value.length > MAX_COPY_VALUE_LENGTH) {
        throw invalidArgument(
          `${key}: value is over the ${MAX_COPY_VALUE_LENGTH}-character limit`,
        );
      }
      for (const ph of requiredPlaceholders(entry.en)) {
        if (!value.includes(`{${ph}}`)) {
          throw invalidArgument(`${key}: missing required placeholder {${ph}}`);
        }
      }
    }
    // Merge key-by-key: a non-empty value upserts the override, an empty
    // value clears it; keys not sent are left untouched.
    const doc = c.overrides[locale] ?? {};
    for (const [key, value] of Object.entries(req.values)) {
      if (value !== "") doc[key] = value;
      else delete doc[key];
    }
    if (Object.keys(doc).length === 0) delete c.overrides[locale];
    else c.overrides[locale] = doc;
    return { revisionId: pushCopyRevision(c, now()) };
  });

  handle(CopyService.method.listCopyRevisions, (state: ContentSlice, req) => {
    const c = copyState(state, req.projectId);
    const limit = req.limit > 0 ? req.limit : c.revisions.length;
    return {
      revisions: c.revisions.slice(0, limit).map((rev) => ({
        revisionId: rev.revisionId,
        createTime: ts(rev.createTime),
        locales: Object.keys(rev.overrides).sort(),
      })),
    };
  });

  handle(CopyService.method.restoreCopyRevision, (state: ContentSlice, req) => {
    const c = copyState(state, req.projectId);
    const rev = c.revisions.find((r) => r.revisionId === req.revisionId);
    if (!rev) throw notFound("copy revision");
    c.overrides = clone(rev.overrides);
    return { revisionId: pushCopyRevision(c, now()) };
  });

  handle(CopyService.method.resetCopy, (state: ContentSlice, req) => {
    if (req.locale === "") throw invalidArgument("locale is required");
    const locale = normalizeLocale(req.locale);
    const c = copyState(state, req.projectId);
    if (req.key === "") {
      delete c.overrides[locale];
    } else {
      const doc = c.overrides[locale];
      if (doc) {
        delete doc[req.key];
        if (Object.keys(doc).length === 0) delete c.overrides[locale];
      }
    }
    return { revisionId: pushCopyRevision(c, now()) };
  });

  handle(CopyService.method.listLocales, (state: ContentSlice, req) => {
    const c = copyState(state, req.projectId);
    const customized = new Set(Object.keys(c.overrides).map(normalizeLocale));
    const tags = [...new Set([...BUNDLED_LOCALES, ...customized])].sort();
    return {
      locales: tags.map((tag) => ({
        tag,
        displayName: LOCALE_DISPLAY_NAMES[tag] ?? tag,
        bundled: BUNDLED_LOCALES.includes(tag),
        customized: customized.has(tag),
        isDefault: tag === DEFAULT_LOCALE,
      })),
      defaultLocale: DEFAULT_LOCALE,
    };
  });

  // ----- PaywallService -----

  handle(PaywallService.method.getPaywallConfig, (state: ContentSlice, req) => {
    const p = paywallState(state, req.projectId);
    return {
      config: paywallMessage(p.current),
      revisionId: p.revisionId,
      isDefault: p.current === null,
    };
  });

  handle(PaywallService.method.updatePaywallConfig, (state: ContentSlice, req) => {
    const cfg = req.config;
    if (!cfg) throw invalidArgument("a paywall config is required");
    if (cfg.headline.trim() === "") throw invalidArgument("a headline is required");
    if (cfg.headline.length > 80) throw invalidArgument("headline is over the 80-character limit");
    if (cfg.subtitle.length > 160) throw invalidArgument("subtitle is over the 160-character limit");
    if (cfg.benefits.length > 8) throw invalidArgument("at most 8 benefits are allowed");
    for (const b of cfg.benefits) {
      if (b.trim() === "") throw invalidArgument("benefits must not be empty");
      if (b.length > 120) throw invalidArgument("a benefit is over the 120-character limit");
    }
    for (const u of [cfg.legal?.termsUrl ?? "", cfg.legal?.privacyUrl ?? ""]) {
      if (u !== "" && !isHttpUrl(u)) {
        throw invalidArgument("legal links must be absolute http(s) URLs");
      }
    }
    const p = paywallState(state, req.projectId);
    const stored = toJson(PaywallConfigSchema, cfg);
    const revisionId = pushPaywallRevision(p, stored, now());
    return { config: paywallMessage(p.current), revisionId };
  });

  handle(PaywallService.method.listPaywallRevisions, (state: ContentSlice, req) => {
    const p = paywallState(state, req.projectId);
    const limit = req.limit > 0 ? req.limit : p.revisions.length;
    return {
      revisions: p.revisions.slice(0, limit).map((rev) => ({
        revisionId: rev.revisionId,
        config: paywallMessage(rev.config),
        createTime: ts(rev.createTime),
      })),
    };
  });

  handle(PaywallService.method.restorePaywallRevision, (state: ContentSlice, req) => {
    const p = paywallState(state, req.projectId);
    const rev = p.revisions.find((r) => r.revisionId === req.revisionId);
    if (!rev) throw notFound("paywall revision");
    const revisionId = pushPaywallRevision(p, rev.config, now());
    return { config: paywallMessage(p.current), revisionId };
  });

  handle(PaywallService.method.resetPaywall, (state: ContentSlice, req) => {
    const p = paywallState(state, req.projectId);
    p.current = null;
    p.revisionId = "";
    return { config: paywallMessage(null) };
  });
}
