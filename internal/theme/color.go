package theme

import (
	"fmt"
	"math"
)

// Color is an opaque sRGB color.
type Color struct {
	R, G, B uint8
}

// ParseColor parses a strict "#RRGGBB" hex color (case-insensitive).
// Shorthand (#RGB) and alpha channels are rejected: the theme schema stores
// one canonical form.
func ParseColor(s string) (Color, error) {
	if len(s) != 7 || s[0] != '#' {
		return Color{}, fmt.Errorf("invalid color %q (want #RRGGBB)", s)
	}
	var c Color
	for i, dst := range []*uint8{&c.R, &c.G, &c.B} {
		hi, ok1 := hexNibble(s[1+2*i])
		lo, ok2 := hexNibble(s[2+2*i])
		if !ok1 || !ok2 {
			return Color{}, fmt.Errorf("invalid color %q (want #RRGGBB)", s)
		}
		*dst = hi<<4 | lo
	}
	return c, nil
}

func hexNibble(b byte) (uint8, bool) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', true
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10, true
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10, true
	}
	return 0, false
}

// Hex formats the color as uppercase "#RRGGBB".
func (c Color) Hex() string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

// Luminance is the WCAG 2.x relative luminance of the color: 0 for black,
// 1 for white.
func (c Color) Luminance() float64 {
	return 0.2126*channelLuminance(c.R) + 0.7152*channelLuminance(c.G) + 0.0722*channelLuminance(c.B)
}

// channelLuminance linearizes one sRGB channel per WCAG 2.x.
func channelLuminance(v uint8) float64 {
	s := float64(v) / 255
	if s <= 0.03928 {
		return s / 12.92
	}
	return math.Pow((s+0.055)/1.055, 2.4)
}

// ContrastRatio is the WCAG 2.x contrast ratio between two colors:
// (L1+0.05)/(L2+0.05) with L1 the lighter. Ranges from 1 (identical
// luminance) to 21 (black on white); order does not matter.
func ContrastRatio(a, b Color) float64 {
	la, lb := a.Luminance(), b.Luminance()
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// mix blends c toward o by t (0 = c, 1 = o), channel-wise on sRGB bytes
// with round-half-up. Naive sRGB interpolation on purpose: it is stable,
// dependency-free and good enough for palette derivation.
func mix(c, o Color, t float64) Color {
	blend := func(a, b uint8) uint8 {
		return uint8(math.Round(float64(a)*(1-t) + float64(b)*t))
	}
	return Color{R: blend(c.R, o.R), G: blend(c.G, o.G), B: blend(c.B, o.B)}
}
