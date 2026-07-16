import type { Theme } from "../gen/moth/admin/v1/theme_pb";

// Client-side mirror of internal/theme: color math (WCAG 2.x), the dark
// palette derivation and the curated font list. The server re-validates
// everything on save — this exists so the editor can warn live and render
// the preview without a round-trip. Keep in sync with internal/theme.

// ---------- Colors ----------

// ThemeColorKey lists the palette roles in editor order: each role directly
// followed by its "on" (foreground) counterpart.
export const COLOR_PAIRS = [
  { role: "primary", on: "onPrimary", label: "Primary", onLabel: "On primary" },
  { role: "background", on: "onBackground", label: "Background", onLabel: "On background" },
  { role: "surface", on: "onSurface", label: "Surface", onLabel: "On surface" },
  { role: "error", on: "onError", label: "Error", onLabel: "On error" },
] as const;

export type ColorKey =
  | "primary"
  | "onPrimary"
  | "background"
  | "onBackground"
  | "surface"
  | "onSurface"
  | "error"
  | "onError";

export type Palette = Record<ColorKey, string>;

// MIN_CONTRAST is the WCAG AA ratio required between every color/on pair
// (internal/theme.MinContrast).
export const MIN_CONTRAST = 4.5;

// isHexColor accepts the schema's canonical form only: strict #RRGGBB.
export function isHexColor(s: string): boolean {
  return /^#[0-9a-fA-F]{6}$/.test(s);
}

export function normalizeHex(s: string): string {
  return s.toUpperCase();
}

type RGB = { r: number; g: number; b: number };

function parseHex(s: string): RGB {
  // Callers pre-validate with isHexColor; black on garbage, like the server.
  if (!isHexColor(s)) return { r: 0, g: 0, b: 0 };
  return {
    r: parseInt(s.slice(1, 3), 16),
    g: parseInt(s.slice(3, 5), 16),
    b: parseInt(s.slice(5, 7), 16),
  };
}

function toHex(c: RGB): string {
  const h = (v: number) => v.toString(16).padStart(2, "0").toUpperCase();
  return `#${h(c.r)}${h(c.g)}${h(c.b)}`;
}

// channelLuminance linearizes one sRGB channel per WCAG 2.x.
function channelLuminance(v: number): number {
  const s = v / 255;
  return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
}

function luminance(c: RGB): number {
  return (
    0.2126 * channelLuminance(c.r) +
    0.7152 * channelLuminance(c.g) +
    0.0722 * channelLuminance(c.b)
  );
}

// contrastRatio is the WCAG 2.x contrast ratio between two hex colors,
// from 1 (identical luminance) to 21 (black on white); order-independent.
export function contrastRatio(a: string, b: string): number {
  let la = luminance(parseHex(a));
  let lb = luminance(parseHex(b));
  if (la < lb) [la, lb] = [lb, la];
  return (la + 0.05) / (lb + 0.05);
}

// ---------- Dark palette derivation (mirror of internal/theme.DeriveDark) ----------

const DARK_BACKGROUND_BLEND = 0.88;
const DARK_SURFACE_BLEND = 0.84;
const DARK_ACCENT_BLEND = 0.4;

const WHITE: RGB = { r: 255, g: 255, b: 255 };
const BLACK: RGB = { r: 0, g: 0, b: 0 };

function mix(c: RGB, o: RGB, t: number): RGB {
  const blend = (a: number, b: number) => Math.round(a * (1 - t) + b * t);
  return { r: blend(c.r, o.r), g: blend(c.g, o.g), b: blend(c.b, o.b) };
}

function blendHex(hex: string, toward: RGB, t: number): string {
  return toHex(mix(parseHex(hex), toward, t));
}

// bestOn picks black or white, whichever contrasts more with hex — always
// at least sqrt(21) ≈ 4.58, so derived on* colors are AA by construction.
function bestOn(hex: string): string {
  return contrastRatio(hex, "#FFFFFF") >= contrastRatio(hex, "#000000") ? "#FFFFFF" : "#000000";
}

// deriveDark computes the effective dark palette: explicit overrides where
// present (non-empty), derived values everywhere else. Surfaces blend
// toward black (surface a little less, so elevation still reads), accents
// blend toward white, and each on* becomes black or white by max contrast.
export function deriveDark(light: Palette, overrides?: Partial<Record<ColorKey, string>>): Palette {
  const ov = (k: ColorKey, derived: string) => {
    const v = overrides?.[k];
    return v ? v : derived;
  };
  const d = {
    primary: ov("primary", blendHex(light.primary, WHITE, DARK_ACCENT_BLEND)),
    background: ov("background", blendHex(light.background, BLACK, DARK_BACKGROUND_BLEND)),
    surface: ov("surface", blendHex(light.surface, BLACK, DARK_SURFACE_BLEND)),
    error: ov("error", blendHex(light.error, WHITE, DARK_ACCENT_BLEND)),
  };
  return {
    ...d,
    onPrimary: ov("onPrimary", bestOn(d.primary)),
    onBackground: ov("onBackground", bestOn(d.background)),
    onSurface: ov("onSurface", bestOn(d.surface)),
    onError: ov("onError", bestOn(d.error)),
  };
}

// ---------- Typography ----------

// FONT_FAMILIES mirrors internal/fonts's registry (the set the server
// accepts): display name, the same-origin URL of the embedded variable
// font, its weight range, and a CSS fallback stack per face so the preview
// still renders sensibly while a face downloads.
const SANS_FALLBACK = `-apple-system, "Segoe UI", Roboto, sans-serif`;
export const FONT_FAMILIES: { name: string; url: string; weights: string; stack: string }[] = [
  {
    name: "Inter",
    url: "/assets/fonts/inter/Inter.ttf",
    weights: "100 900",
    stack: `"Inter", ${SANS_FALLBACK}`,
  },
  {
    name: "Source Sans 3",
    url: "/assets/fonts/sourcesans3/SourceSans3.ttf",
    weights: "200 900",
    stack: `"Source Sans 3", ${SANS_FALLBACK}`,
  },
  {
    name: "Nunito Sans",
    url: "/assets/fonts/nunitosans/NunitoSans.ttf",
    weights: "200 1000",
    stack: `"Nunito Sans", ${SANS_FALLBACK}`,
  },
  {
    name: "Lora",
    url: "/assets/fonts/lora/Lora.ttf",
    weights: "400 700",
    stack: `"Lora", Georgia, "Times New Roman", serif`,
  },
  {
    name: "JetBrains Mono",
    url: "/assets/fonts/jetbrainsmono/JetBrainsMono.ttf",
    weights: "100 800",
    stack: `"JetBrains Mono", "SF Mono", Menlo, Consolas, monospace`,
  },
];

export function fontStack(name: string): string {
  return FONT_FAMILIES.find((f) => f.name === name)?.stack ?? `"${name}", ${SANS_FALLBACK}`;
}

// ensurePreviewFonts injects @font-face rules for the curated faces (the
// client-side twin of internal/fonts.FaceCSS, same-origin /assets/fonts
// URLs), so the live preview renders the actual embedded font a theme
// selects rather than whatever happens to be installed locally. Idempotent;
// the browser only downloads a face once the preview uses it.
let previewFontsInjected = false;
export function ensurePreviewFonts(): void {
  if (previewFontsInjected || typeof document === "undefined") return;
  previewFontsInjected = true;
  const style = document.createElement("style");
  style.setAttribute("data-moth-preview-fonts", "");
  style.textContent = FONT_FAMILIES.map(
    (f) =>
      `@font-face { font-family: "${f.name}"; font-style: normal; ` +
      `font-weight: ${f.weights}; font-display: swap; ` +
      `src: url("${f.url}") format("truetype"); }`,
  ).join("\n");
  document.head.appendChild(style);
}

// Token ranges (internal/theme).
export const MIN_SCALE = 0.8;
export const MAX_SCALE = 1.4;
export const MIN_SPACING_UNIT = 4;
export const MAX_SPACING_UNIT = 16;
export const MAX_CORNER_RADIUS = 32;

// MAX_LOGO_BYTES is the UploadLogo size cap (512 KiB).
export const MAX_LOGO_BYTES = 512 * 1024;

// ---------- Editor state ----------

// EditorTheme is the tab's working copy of the token set: a plain object so
// React state updates stay structural. Logo paths are deliberately absent —
// they are server-managed (UploadLogo/DeleteLogo) and read from the latest
// GetTheme response instead.
export type EditorTheme = {
  colors: Palette;
  // darkEnabled toggles the explicit dark override group; when false the
  // whole dark palette is derived and `dark` is not sent.
  darkEnabled: boolean;
  dark: Palette;
  fontFamily: string;
  scale: number;
  spacingUnit: number;
  cornerRadius: number;
  termsUrl: string;
  privacyUrl: string;
};

const FALLBACK_PALETTE: Palette = {
  // internal/theme.Default() — only relevant when the server response is
  // missing a message, which GetTheme never does in practice.
  primary: "#6750A4",
  onPrimary: "#FFFFFF",
  background: "#FFFBFE",
  onBackground: "#1C1B1F",
  surface: "#FFFBFE",
  onSurface: "#1C1B1F",
  error: "#B3261E",
  onError: "#FFFFFF",
};

// editorFromProto seeds the working copy from a GetTheme (or restore/reset)
// response. Partial dark overrides are expanded to the full effective dark
// palette: the editor edits complete palettes and sends all eight fields
// back when overriding.
export function editorFromProto(theme: Theme | undefined): EditorTheme {
  const c = theme?.colors;
  const colors: Palette = c
    ? {
        primary: c.primary,
        onPrimary: c.onPrimary,
        background: c.background,
        onBackground: c.onBackground,
        surface: c.surface,
        onSurface: c.onSurface,
        error: c.error,
        onError: c.onError,
      }
    : { ...FALLBACK_PALETTE };
  const o = theme?.darkColors;
  const overrides: Partial<Record<ColorKey, string>> = {};
  if (o) {
    for (const k of Object.keys(FALLBACK_PALETTE) as ColorKey[]) {
      if (o[k]) overrides[k] = o[k];
    }
  }
  const darkEnabled = Object.keys(overrides).length > 0;
  return {
    colors,
    darkEnabled,
    dark: deriveDark(colors, overrides),
    fontFamily: theme?.typography?.fontFamily || FONT_FAMILIES[0].name,
    scale: theme?.typography?.scale || 1.0,
    spacingUnit: theme?.spacing?.unit || 8,
    cornerRadius: theme?.shape?.cornerRadius ?? 12,
    termsUrl: theme?.legal?.termsUrl ?? "",
    privacyUrl: theme?.legal?.privacyUrl ?? "",
  };
}

// editorToProto builds the UpdateTheme payload. Logo is omitted (output
// only); an enabled dark group sends all eight fields as explicit
// overrides, a disabled one sends nothing (fully derived).
export function editorToProto(t: EditorTheme) {
  return {
    colors: { ...t.colors },
    darkColors: t.darkEnabled ? { ...t.dark } : undefined,
    typography: { fontFamily: t.fontFamily, scale: t.scale },
    spacing: { unit: t.spacingUnit },
    shape: { cornerRadius: t.cornerRadius },
    legal: { termsUrl: t.termsUrl.trim(), privacyUrl: t.privacyUrl.trim() },
  };
}

// effectiveDark returns the dark palette the preview and the contrast
// checks use for the current editor state.
export function effectiveDark(t: EditorTheme): Palette {
  return t.darkEnabled ? t.dark : deriveDark(t.colors);
}

// ContrastIssue describes one failing color/on pair.
export type ContrastIssue = { scheme: "light" | "dark"; label: string; ratio: number };

// contrastIssues checks every color/on pair of both effective palettes
// against WCAG AA; save is blocked while any remain (the server re-checks).
export function contrastIssues(t: EditorTheme): ContrastIssue[] {
  const issues: ContrastIssue[] = [];
  const check = (scheme: "light" | "dark", p: Palette) => {
    for (const pair of COLOR_PAIRS) {
      const ratio = contrastRatio(p[pair.role], p[pair.on]);
      if (ratio < MIN_CONTRAST) issues.push({ scheme, label: pair.label.toLowerCase(), ratio });
    }
  };
  check("light", t.colors);
  check("dark", effectiveDark(t));
  return issues;
}
