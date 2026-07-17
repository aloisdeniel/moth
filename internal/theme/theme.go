// Package theme defines the per-project design system: a small, versioned
// set of tokens (colors, typography, spacing, corner radius, logo, legal
// links) that every end-user surface — the Flutter login screen, hosted web
// pages, emails — renders from. The token space is deliberately constrained
// so that every accepted theme produces a legible screen: Validate enforces
// WCAG AA contrast, curated fonts and sane ranges.
package theme

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	projectconfigv1 "github.com/aloisdeniel/moth/gen/moth/projectconfig/v1"
	"github.com/aloisdeniel/moth/internal/fonts"
)

// SchemaVersion is the version stamped on every encoded theme document.
// Parse rejects documents with a different version; future schema changes
// bump it and add an explicit upgrade path.
const SchemaVersion = 1

// Theme is one project's complete design system. It is persisted as a
// moth.projectconfig.v1.StoredTheme protobuf document (see Encode/Parse).
type Theme struct {
	// Version is the schema version of the document (SchemaVersion).
	Version int
	// Colors is the light palette. All fields are required #RRGGBB values.
	Colors Colors
	// DarkColors optionally overrides individual dark-palette colors.
	// Omitted fields (and a nil struct) are derived from Colors — see
	// DeriveDark for the algorithm.
	DarkColors *ColorOverrides
	Typography Typography
	Spacing    Spacing
	Shape      Shape
	Logo       Logo
	Legal      Legal
}

// Colors is a complete palette: every role and its "on" (foreground)
// counterpart. Values are #RRGGBB.
type Colors struct {
	Primary      string
	OnPrimary    string
	Background   string
	OnBackground string
	Surface      string
	OnSurface    string
	Error        string
	OnError      string
}

// ColorOverrides is a partial palette: any empty field is derived from the
// light palette instead.
type ColorOverrides struct {
	Primary      string
	OnPrimary    string
	Background   string
	OnBackground string
	Surface      string
	OnSurface    string
	Error        string
	OnError      string
}

// Typography selects one of the curated embedded fonts and a global size
// multiplier.
type Typography struct {
	// FontFamily must be one of FontFamilies.
	FontFamily string
	// Scale multiplies every text size; MinScale..MaxScale.
	Scale float64
}

// Spacing is the base spacing grid.
type Spacing struct {
	// Unit is the base spacing step in logical pixels;
	// MinSpacingUnit..MaxSpacingUnit.
	Unit int
}

// Shape controls component rounding.
type Shape struct {
	// CornerRadius in logical pixels; 0..MaxCornerRadius.
	CornerRadius int
}

// Logo holds the server-managed asset paths of the uploaded logos, one per
// color scheme ("/assets/{project}/logo-light.png"). Empty = no logo.
type Logo struct {
	Light string
	Dark  string
}

// Legal holds the optional legal links rendered near signup.
type Legal struct {
	TermsURL   string
	PrivacyURL string
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

// Encode serializes the theme as its canonical storage document (a
// moth.projectconfig.v1.StoredTheme protobuf message), stamping the current schema
// version. An encoded document is never empty: empty stored bytes keep
// meaning "the built-in default theme".
func Encode(t Theme) ([]byte, error) {
	t.Version = SchemaVersion
	raw, err := proto.Marshal(ToProto(t))
	if err != nil {
		return nil, fmt.Errorf("encode theme: %w", err)
	}
	return raw, nil
}

// Parse decodes a stored theme document (moth.projectconfig.v1.StoredTheme). It
// rejects documents from a different schema version — including empty input,
// which callers treat as "default theme" before parsing; it does not
// validate token values (Validate does).
func Parse(raw []byte) (Theme, error) {
	var msg projectconfigv1.StoredTheme
	if err := proto.Unmarshal(raw, &msg); err != nil {
		return Theme{}, fmt.Errorf("parse theme: %w", err)
	}
	if msg.Version != SchemaVersion {
		return Theme{}, fmt.Errorf("parse theme: unsupported schema version %d (want %d)", msg.Version, SchemaVersion)
	}
	return FromProto(&msg), nil
}

// ToProto converts the domain theme into its storage message.
func ToProto(t Theme) *projectconfigv1.StoredTheme {
	msg := &projectconfigv1.StoredTheme{
		Version: int32(t.Version),
		Colors: &projectconfigv1.ThemeColors{
			Primary:      t.Colors.Primary,
			OnPrimary:    t.Colors.OnPrimary,
			Background:   t.Colors.Background,
			OnBackground: t.Colors.OnBackground,
			Surface:      t.Colors.Surface,
			OnSurface:    t.Colors.OnSurface,
			Error:        t.Colors.Error,
			OnError:      t.Colors.OnError,
		},
		Typography: &projectconfigv1.ThemeTypography{
			FontFamily: t.Typography.FontFamily,
			Scale:      t.Typography.Scale,
		},
		Spacing: &projectconfigv1.ThemeSpacing{Unit: int32(t.Spacing.Unit)},
		Shape:   &projectconfigv1.ThemeShape{CornerRadius: int32(t.Shape.CornerRadius)},
		Logo:    &projectconfigv1.ThemeLogo{Light: t.Logo.Light, Dark: t.Logo.Dark},
		Legal:   &projectconfigv1.LegalLinks{TermsUrl: t.Legal.TermsURL, PrivacyUrl: t.Legal.PrivacyURL},
	}
	if t.DarkColors != nil {
		msg.DarkColors = &projectconfigv1.ThemeColorOverrides{
			Primary:      t.DarkColors.Primary,
			OnPrimary:    t.DarkColors.OnPrimary,
			Background:   t.DarkColors.Background,
			OnBackground: t.DarkColors.OnBackground,
			Surface:      t.DarkColors.Surface,
			OnSurface:    t.DarkColors.OnSurface,
			Error:        t.DarkColors.Error,
			OnError:      t.DarkColors.OnError,
		}
	}
	return msg
}

// FromProto converts a storage message into the domain theme. Nil
// sub-messages become zero values; an absent dark_colors stays nil so the
// dark palette derives fully from the light one.
func FromProto(msg *projectconfigv1.StoredTheme) Theme {
	t := Theme{Version: int(msg.GetVersion())}
	if c := msg.GetColors(); c != nil {
		t.Colors = Colors{
			Primary:      c.Primary,
			OnPrimary:    c.OnPrimary,
			Background:   c.Background,
			OnBackground: c.OnBackground,
			Surface:      c.Surface,
			OnSurface:    c.OnSurface,
			Error:        c.Error,
			OnError:      c.OnError,
		}
	}
	if d := msg.GetDarkColors(); d != nil {
		t.DarkColors = &ColorOverrides{
			Primary:      d.Primary,
			OnPrimary:    d.OnPrimary,
			Background:   d.Background,
			OnBackground: d.OnBackground,
			Surface:      d.Surface,
			OnSurface:    d.OnSurface,
			Error:        d.Error,
			OnError:      d.OnError,
		}
	}
	if ty := msg.GetTypography(); ty != nil {
		t.Typography = Typography{FontFamily: ty.FontFamily, Scale: ty.Scale}
	}
	if sp := msg.GetSpacing(); sp != nil {
		t.Spacing = Spacing{Unit: int(sp.Unit)}
	}
	if sh := msg.GetShape(); sh != nil {
		t.Shape = Shape{CornerRadius: int(sh.CornerRadius)}
	}
	if l := msg.GetLogo(); l != nil {
		t.Logo = Logo{Light: l.Light, Dark: l.Dark}
	}
	if l := msg.GetLegal(); l != nil {
		t.Legal = Legal{TermsURL: l.TermsUrl, PrivacyURL: l.PrivacyUrl}
	}
	return t
}

// EffectiveDark returns the dark palette actually rendered: explicit
// DarkColors overrides where present, derived values everywhere else.
func (t Theme) EffectiveDark() Colors {
	return DeriveDark(t.Colors, t.DarkColors)
}
