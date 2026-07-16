package theme

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
)

// assetPathPrefix is where uploaded theme assets are served from; logo
// paths in a theme must point there.
const assetPathPrefix = "/assets/"

// Validate checks every token of the theme, including the effective dark
// palette after derivation, and returns the first violation. A theme that
// validates is guaranteed to render legibly: WCAG AA contrast on every
// color/on-color pair, a curated font, and in-range scale/spacing/radius.
func (t Theme) Validate() error {
	if t.Version != SchemaVersion {
		return fmt.Errorf("unsupported schema version %d (want %d)", t.Version, SchemaVersion)
	}
	for _, f := range colorFields(t.Colors) {
		if _, err := ParseColor(f.value); err != nil {
			return fmt.Errorf("colors.%s: %w", f.name, err)
		}
	}
	if t.DarkColors != nil {
		for _, f := range colorFields(Colors(*t.DarkColors)) {
			if f.value == "" {
				continue // derived
			}
			if _, err := ParseColor(f.value); err != nil {
				return fmt.Errorf("darkColors.%s: %w", f.name, err)
			}
		}
	}
	if err := checkContrast("colors", t.Colors); err != nil {
		return err
	}
	// Derived on* colors always pass (see DeriveDark); this catches
	// illegible explicit overrides.
	if err := checkContrast("darkColors", t.EffectiveDark()); err != nil {
		return err
	}
	if !slices.Contains(FontFamilies, t.Typography.FontFamily) {
		return fmt.Errorf("typography.fontFamily: unknown font %q (available: %s)",
			t.Typography.FontFamily, strings.Join(FontFamilies, ", "))
	}
	if t.Typography.Scale < MinScale || t.Typography.Scale > MaxScale {
		return fmt.Errorf("typography.scale: %g out of range [%g, %g]", t.Typography.Scale, MinScale, MaxScale)
	}
	if t.Spacing.Unit < MinSpacingUnit || t.Spacing.Unit > MaxSpacingUnit {
		return fmt.Errorf("spacing.unit: %d out of range [%d, %d]", t.Spacing.Unit, MinSpacingUnit, MaxSpacingUnit)
	}
	if t.Shape.CornerRadius < 0 || t.Shape.CornerRadius > MaxCornerRadius {
		return fmt.Errorf("shape.cornerRadius: %d out of range [0, %d]", t.Shape.CornerRadius, MaxCornerRadius)
	}
	for name, path := range map[string]string{"light": t.Logo.Light, "dark": t.Logo.Dark} {
		if path != "" && !strings.HasPrefix(path, assetPathPrefix) {
			return fmt.Errorf("logo.%s: %q is not a managed asset path (want %s...)", name, path, assetPathPrefix)
		}
	}
	for name, u := range map[string]string{"termsUrl": t.Legal.TermsURL, "privacyUrl": t.Legal.PrivacyURL} {
		if u == "" {
			continue
		}
		if err := validHTTPURL(u); err != nil {
			return fmt.Errorf("legal.%s: %w", name, err)
		}
	}
	return nil
}

type colorField struct {
	name, value string
}

func colorFields(c Colors) []colorField {
	return []colorField{
		{"primary", c.Primary},
		{"onPrimary", c.OnPrimary},
		{"background", c.Background},
		{"onBackground", c.OnBackground},
		{"surface", c.Surface},
		{"onSurface", c.OnSurface},
		{"error", c.Error},
		{"onError", c.OnError},
	}
}

// checkContrast enforces MinContrast between each color and its on*
// counterpart. Unparseable values error here too, so the check is safe to
// call on palettes that skipped the per-field hex validation.
func checkContrast(section string, c Colors) error {
	pairs := []struct {
		name   string
		bg, fg string
	}{
		{"primary/onPrimary", c.Primary, c.OnPrimary},
		{"background/onBackground", c.Background, c.OnBackground},
		{"surface/onSurface", c.Surface, c.OnSurface},
		{"error/onError", c.Error, c.OnError},
	}
	for _, p := range pairs {
		bg, err := ParseColor(p.bg)
		if err != nil {
			return fmt.Errorf("%s.%s: %w", section, p.name, err)
		}
		fg, err := ParseColor(p.fg)
		if err != nil {
			return fmt.Errorf("%s.%s: %w", section, p.name, err)
		}
		if cr := ContrastRatio(bg, fg); cr < MinContrast {
			return fmt.Errorf("%s: %s contrast %.2f:1 is below the WCAG AA minimum %.1f:1",
				section, p.name, cr, MinContrast)
		}
	}
	return nil
}

func validHTTPURL(s string) error {
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("%q is not an absolute http(s) URL", s)
	}
	return nil
}
