package theme

// Dark-palette derivation blend factors (toward black for surfaces, toward
// white for accents).
const (
	darkBackgroundBlend = 0.88
	darkSurfaceBlend    = 0.84
	darkAccentBlend     = 0.40
)

var (
	white = Color{R: 255, G: 255, B: 255}
	black = Color{}
)

// DeriveDark computes the effective dark palette from the light palette
// and the optional per-field overrides. The algorithm, applied to every
// field left empty in o (a nil o derives everything), is deterministic:
//
//  1. background and surface: the light value blended 88% / 84% toward
//     black — lands in the dark-neutral range while keeping a trace of the
//     brand tint, with surface slightly lighter than background so
//     elevation still reads.
//  2. primary and error: the light value blended 40% toward white — the
//     conventional pastel shift that keeps saturated brand colors legible
//     on dark surfaces.
//  3. every on* color: whichever of black/white contrasts more with its
//     (derived or overridden) counterpart. This always meets WCAG AA:
//     CR(c, white) x CR(c, black) = 21 for any c, so the larger of the two
//     is at least sqrt(21) ~ 4.58 >= MinContrast.
//
// light is assumed valid (see Validate); unparseable inputs derive from
// black.
func DeriveDark(light Colors, o *ColorOverrides) Colors {
	var ov ColorOverrides
	if o != nil {
		ov = *o
	}
	d := Colors{
		Primary:    override(ov.Primary, blendHex(light.Primary, white, darkAccentBlend)),
		Background: override(ov.Background, blendHex(light.Background, black, darkBackgroundBlend)),
		Surface:    override(ov.Surface, blendHex(light.Surface, black, darkSurfaceBlend)),
		Error:      override(ov.Error, blendHex(light.Error, white, darkAccentBlend)),
	}
	d.OnPrimary = override(ov.OnPrimary, bestOn(d.Primary))
	d.OnBackground = override(ov.OnBackground, bestOn(d.Background))
	d.OnSurface = override(ov.OnSurface, bestOn(d.Surface))
	d.OnError = override(ov.OnError, bestOn(d.Error))
	return d
}

func override(explicit, derived string) string {
	if explicit != "" {
		return explicit
	}
	return derived
}

func blendHex(hex string, toward Color, t float64) string {
	c, _ := ParseColor(hex) // zero (black) on error; inputs are pre-validated
	return mix(c, toward, t).Hex()
}

// bestOn picks black or white, whichever has the higher contrast ratio
// against hex.
func bestOn(hex string) string {
	c, _ := ParseColor(hex)
	if ContrastRatio(c, white) >= ContrastRatio(c, black) {
		return white.Hex()
	}
	return black.Hex()
}
