// Package theme defines the per-project design system: a small, versioned
// set of tokens (colors, typography, spacing, corner radius, logo, legal
// links) that every end-user surface — the Flutter login screen, hosted web
// pages, emails — renders from. The token space is deliberately constrained
// so that every accepted theme produces a legible screen: Validate enforces
// WCAG AA contrast, curated fonts and sane ranges.
package theme

import (
	"encoding/json"
	"fmt"

	"github.com/aloisdeniel/moth/internal/fonts"
)

// SchemaVersion is the version stamped on every encoded theme document.
// Parse rejects documents with a different version; future schema changes
// bump it and add an explicit upgrade path.
const SchemaVersion = 1

// Theme is one project's complete design system.
type Theme struct {
	// Version is the schema version of the document (SchemaVersion).
	Version int `json:"version"`
	// Colors is the light palette. All fields are required #RRGGBB values.
	Colors Colors `json:"colors"`
	// DarkColors optionally overrides individual dark-palette colors.
	// Omitted fields (and a nil struct) are derived from Colors — see
	// DeriveDark for the algorithm.
	DarkColors *ColorOverrides `json:"darkColors,omitempty"`
	Typography Typography      `json:"typography"`
	Spacing    Spacing         `json:"spacing"`
	Shape      Shape           `json:"shape"`
	Logo       Logo            `json:"logo"`
	Legal      Legal           `json:"legal"`
}

// Colors is a complete palette: every role and its "on" (foreground)
// counterpart. Values are #RRGGBB.
type Colors struct {
	Primary      string `json:"primary"`
	OnPrimary    string `json:"onPrimary"`
	Background   string `json:"background"`
	OnBackground string `json:"onBackground"`
	Surface      string `json:"surface"`
	OnSurface    string `json:"onSurface"`
	Error        string `json:"error"`
	OnError      string `json:"onError"`
}

// ColorOverrides is a partial palette: any empty field is derived from the
// light palette instead.
type ColorOverrides struct {
	Primary      string `json:"primary,omitempty"`
	OnPrimary    string `json:"onPrimary,omitempty"`
	Background   string `json:"background,omitempty"`
	OnBackground string `json:"onBackground,omitempty"`
	Surface      string `json:"surface,omitempty"`
	OnSurface    string `json:"onSurface,omitempty"`
	Error        string `json:"error,omitempty"`
	OnError      string `json:"onError,omitempty"`
}

// Typography selects one of the curated embedded fonts and a global size
// multiplier.
type Typography struct {
	// FontFamily must be one of FontFamilies.
	FontFamily string `json:"fontFamily"`
	// Scale multiplies every text size; MinScale..MaxScale.
	Scale float64 `json:"scale"`
}

// Spacing is the base spacing grid.
type Spacing struct {
	// Unit is the base spacing step in logical pixels;
	// MinSpacingUnit..MaxSpacingUnit.
	Unit int `json:"unit"`
}

// Shape controls component rounding.
type Shape struct {
	// CornerRadius in logical pixels; 0..MaxCornerRadius.
	CornerRadius int `json:"cornerRadius"`
}

// Logo holds the server-managed asset paths of the uploaded logos, one per
// color scheme ("/assets/{project}/logo-light.png"). Empty = no logo.
type Logo struct {
	Light string `json:"light,omitempty"`
	Dark  string `json:"dark,omitempty"`
}

// Legal holds the optional legal links rendered near signup.
type Legal struct {
	TermsURL   string `json:"termsUrl,omitempty"`
	PrivacyURL string `json:"privacyUrl,omitempty"`
}

// FontFamilies is the curated set of open-license fonts embedded in the
// binary (internal/fonts), by display name; Typography.FontFamily must be
// one of them. Arbitrary font uploads are deliberately out of scope — a
// fixed set keeps mobile rendering predictable and the binary
// self-contained.
var FontFamilies = func() []string {
	all := fonts.List()
	names := make([]string, len(all))
	for i, f := range all {
		names[i] = f.Name
	}
	return names
}()

// Token ranges accepted by Validate.
const (
	MinScale        = 0.8
	MaxScale        = 1.4
	MinSpacingUnit  = 4
	MaxSpacingUnit  = 16
	MaxCornerRadius = 32
)

// MinContrast is the WCAG AA contrast ratio required between every color
// and its "on" counterpart, in both the light and the effective dark
// palette.
const MinContrast = 4.5

// Default returns the theme applied to projects that never customized
// anything: the Material baseline palette, Inter at scale 1, an 8px grid
// and 12px corners.
func Default() Theme {
	return Theme{
		Version: SchemaVersion,
		Colors: Colors{
			Primary:      "#6750A4",
			OnPrimary:    "#FFFFFF",
			Background:   "#FFFBFE",
			OnBackground: "#1C1B1F",
			Surface:      "#FFFBFE",
			OnSurface:    "#1C1B1F",
			Error:        "#B3261E",
			OnError:      "#FFFFFF",
		},
		Typography: Typography{FontFamily: "Inter", Scale: 1.0},
		Spacing:    Spacing{Unit: 8},
		Shape:      Shape{CornerRadius: 12},
	}
}

// Encode serializes the theme as its canonical JSON document, stamping the
// current schema version.
func Encode(t Theme) ([]byte, error) {
	t.Version = SchemaVersion
	raw, err := json.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("encode theme: %w", err)
	}
	return raw, nil
}

// Parse decodes a stored theme document. It rejects documents from a
// different schema version; it does not validate token values (Validate
// does).
func Parse(raw []byte) (Theme, error) {
	var t Theme
	if err := json.Unmarshal(raw, &t); err != nil {
		return Theme{}, fmt.Errorf("parse theme: %w", err)
	}
	if t.Version != SchemaVersion {
		return Theme{}, fmt.Errorf("parse theme: unsupported schema version %d (want %d)", t.Version, SchemaVersion)
	}
	return t, nil
}

// EffectiveDark returns the dark palette actually rendered: explicit
// DarkColors overrides where present, derived values everywhere else.
func (t Theme) EffectiveDark() Colors {
	return DeriveDark(t.Colors, t.DarkColors)
}
