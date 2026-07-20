import type { Theme, ThemeColors } from '../gen/moth/auth/v1/config_pb.js'

/**
 * One complete palette of the moth design system: each color role and its
 * "on" (foreground) counterpart, as `#RRGGBB` strings. Server-side
 * validation guarantees WCAG AA contrast between every pair.
 */
export interface MothThemeColors {
  primary: string
  onPrimary: string
  background: string
  onBackground: string
  surface: string
  onSurface: string
  error: string
  onError: string
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
  revisionId: string
  /** Light palette. */
  colors: MothThemeColors
  /** Dark palette (admin overrides merged server-side, or derived locally). */
  darkColors: MothThemeColors
  /** Font family display name (from the server's curated set). */
  fontFamily: string
  /** Absolute URL of the font file to load; undefined = system font. */
  fontUrl?: string
  /** Global text-size multiplier. */
  fontScale: number
  /** Base spacing step in CSS pixels. */
  spacingUnit: number
  /** Component corner radius in CSS pixels. */
  cornerRadius: number
  /** Absolute logo URLs per color scheme. */
  logoLightUrl?: string
  logoDarkUrl?: string
  /** Optional legal links rendered in the login screen footer. */
  termsUrl?: string
  privacyUrl?: string
}

const fallbackLight: MothThemeColors = {
  primary: '#6750A4',
  onPrimary: '#FFFFFF',
  background: '#FFFBFE',
  onBackground: '#1C1B1F',
  surface: '#FFFBFE',
  onSurface: '#1C1B1F',
  error: '#B3261E',
  onError: '#FFFFFF',
}

/**
 * The theme every project starts from (and the offline fallback when
 * nothing is cached yet): the server's built-in default.
 */
export function fallbackTheme(): MothTheme {
  return {
    revisionId: '',
    colors: { ...fallbackLight },
    darkColors: deriveDarkColors(fallbackLight),
    fontFamily: 'Inter',
    fontScale: 1,
    spacingUnit: 8,
    cornerRadius: 12,
  }
}

/**
 * Maps the theme message from `GetProjectConfig`. Fields an older server
 * leaves empty fall back to the defaults, and a missing dark palette is
 * derived locally with the same algorithm the server uses.
 */
export function themeFromProto(proto: Theme): MothTheme {
  const fallback = fallbackTheme()
  const light = proto.colors
    ? colorsFromProto(proto.colors, fallback.colors)
    : fallback.colors
  const theme: MothTheme = {
    revisionId: proto.revisionId,
    colors: light,
    darkColors: proto.darkColors
      ? colorsFromProto(proto.darkColors, deriveDarkColors(light))
      : deriveDarkColors(light),
    fontFamily: proto.fontFamily === '' ? fallback.fontFamily : proto.fontFamily,
    fontScale: proto.fontScale > 0 ? proto.fontScale : fallback.fontScale,
    spacingUnit: proto.spacingUnit > 0 ? proto.spacingUnit : fallback.spacingUnit,
    cornerRadius:
      proto.cornerRadius >= 0 ? proto.cornerRadius : fallback.cornerRadius,
  }
  if (proto.fontUrl !== '') theme.fontUrl = proto.fontUrl
  if (proto.logoLightUrl !== '') theme.logoLightUrl = proto.logoLightUrl
  if (proto.logoDarkUrl !== '') theme.logoDarkUrl = proto.logoDarkUrl
  if (proto.termsUrl !== '') theme.termsUrl = proto.termsUrl
  if (proto.privacyUrl !== '') theme.privacyUrl = proto.privacyUrl
  return theme
}

function colorsFromProto(
  proto: ThemeColors,
  fallback: MothThemeColors,
): MothThemeColors {
  const parse = (hex: string, fb: string) =>
    hexPattern.test(hex) ? hex.toUpperCase() : fb
  return {
    primary: parse(proto.primary, fallback.primary),
    onPrimary: parse(proto.onPrimary, fallback.onPrimary),
    background: parse(proto.background, fallback.background),
    onBackground: parse(proto.onBackground, fallback.onBackground),
    surface: parse(proto.surface, fallback.surface),
    onSurface: parse(proto.onSurface, fallback.onSurface),
    error: parse(proto.error, fallback.error),
    onError: parse(proto.onError, fallback.onError),
  }
}

/**
 * The CSS custom properties for `theme`, ready to set on the `.moth-root`
 * wrapper. Light and dark palettes are emitted side by side
 * (`--moth-l-*` / `--moth-d-*`); the injected stylesheet resolves the live
 * `--moth-*` tokens from them per `prefers-color-scheme`.
 */
export function themeCssVars(theme: MothTheme): Record<string, string> {
  const vars: Record<string, string> = {
    '--moth-font': `'${theme.fontFamily}', system-ui, sans-serif`,
    '--moth-font-scale': String(theme.fontScale),
    '--moth-unit': `${theme.spacingUnit}px`,
    '--moth-radius': `${theme.cornerRadius}px`,
  }
  const put = (prefix: 'l' | 'd', colors: MothThemeColors) => {
    vars[`--moth-${prefix}-primary`] = colors.primary
    vars[`--moth-${prefix}-on-primary`] = colors.onPrimary
    vars[`--moth-${prefix}-background`] = colors.background
    vars[`--moth-${prefix}-on-background`] = colors.onBackground
    vars[`--moth-${prefix}-surface`] = colors.surface
    vars[`--moth-${prefix}-on-surface`] = colors.onSurface
    vars[`--moth-${prefix}-error`] = colors.error
    vars[`--moth-${prefix}-on-error`] = colors.onError
  }
  put('l', theme.colors)
  put('d', theme.darkColors)
  return vars
}

const loadedFonts = new Set<string>()

/**
 * Loads the theme's font file (when it names one) into `document.fonts` so
 * the moth surfaces render it. Idempotent per URL; failures are swallowed —
 * text simply stays on the system font.
 */
export async function ensureThemeFont(theme: MothTheme): Promise<void> {
  const url = theme.fontUrl
  if (
    url === undefined ||
    loadedFonts.has(url) ||
    typeof document === 'undefined' ||
    typeof FontFace === 'undefined'
  ) {
    return
  }
  // Mark before awaiting so a raced second call cannot double-register.
  loadedFonts.add(url)
  try {
    const face = new FontFace(theme.fontFamily, `url(${JSON.stringify(url)})`)
    await face.load()
    document.fonts.add(face)
  } catch {
    loadedFonts.delete(url) // allow a later retry
  }
}

// ------------------------------------------------------------- color math
//
// Mirrors internal/theme on the server (color.go + derive.go) so a locally
// derived dark palette matches what the server would have sent.

const hexPattern = /^#[0-9a-fA-F]{6}$/

/** Parses a strict `#RRGGBB` hex color into [r, g, b]; null when malformed. */
export function parseHexColor(hex: string): [number, number, number] | null {
  if (!hexPattern.test(hex)) return null
  const v = parseInt(hex.slice(1), 16)
  return [(v >> 16) & 0xff, (v >> 8) & 0xff, v & 0xff]
}

function formatHex([r, g, b]: [number, number, number]): string {
  const v = (r << 16) | (g << 8) | b
  return `#${v.toString(16).padStart(6, '0').toUpperCase()}`
}

/**
 * Derives a dark palette from a light one with the exact algorithm the
 * server uses (internal/theme/derive.go): background and surface blend
 * 88% / 84% toward black, primary and error blend 40% toward white, and
 * every on* color becomes black or white — whichever contrasts more.
 */
export function deriveDarkColors(light: MothThemeColors): MothThemeColors {
  const white: [number, number, number] = [255, 255, 255]
  const black: [number, number, number] = [0, 0, 0]
  const of = (hex: string) => parseHexColor(hex) ?? black
  const primary = mix(of(light.primary), white, 0.4)
  const background = mix(of(light.background), black, 0.88)
  const surface = mix(of(light.surface), black, 0.84)
  const error = mix(of(light.error), white, 0.4)
  return {
    primary: formatHex(primary),
    onPrimary: bestOn(primary),
    background: formatHex(background),
    onBackground: bestOn(background),
    surface: formatHex(surface),
    onSurface: bestOn(surface),
    error: formatHex(error),
    onError: bestOn(error),
  }
}

/** WCAG 2.x contrast ratio between two `#RRGGBB` colors (1..21). */
export function contrastRatio(a: string, b: string): number {
  const la = luminance(parseHexColor(a) ?? [0, 0, 0])
  const lb = luminance(parseHexColor(b) ?? [0, 0, 0])
  const hi = Math.max(la, lb)
  const lo = Math.min(la, lb)
  return (hi + 0.05) / (lo + 0.05)
}

// Channel-wise sRGB blend with round-half-up — naive on purpose, matching
// the server byte-for-byte.
function mix(
  color: [number, number, number],
  toward: [number, number, number],
  t: number,
): [number, number, number] {
  const blend = (a: number, b: number) => Math.round(a * (1 - t) + b * t)
  return [
    blend(color[0], toward[0]),
    blend(color[1], toward[1]),
    blend(color[2], toward[2]),
  ]
}

// Black or white, whichever has the higher contrast (white wins ties).
function bestOn(color: [number, number, number]): string {
  const l = luminance(color)
  const whiteRatio = (1 + 0.05) / (l + 0.05)
  const blackRatio = (l + 0.05) / 0.05
  return whiteRatio >= blackRatio ? '#FFFFFF' : '#000000'
}

function luminance([r, g, b]: [number, number, number]): number {
  const channel = (v: number) => {
    const s = v / 255
    return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4)
  }
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b)
}
