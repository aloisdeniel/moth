package server

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/aloisdeniel/moth/internal/fonts"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/theme"
)

// themedData fills the design-system fields of a hosted page from the
// project's theme: the generated CSS custom-properties block, logo URLs and
// legal links. Every hosted page renders through this, so an admin theme
// edit shows up on the very next request.
func themedData(p store.Project, data pageData) pageData {
	t, rev := authrpc.ProjectTheme(p)
	data.Project = p.Name
	data.ThemeCSS = pageThemeCSS(t)
	// Same-origin paths (revision-keyed, see /assets): the pages live on
	// the moth origin itself.
	data.LogoLight = authrpc.AssetURL("", t.Logo.Light, rev)
	data.LogoDark = authrpc.AssetURL("", t.Logo.Dark, rev)
	if data.LogoDark == "" {
		// Dark scheme falls back to the light logo rather than no logo.
		data.LogoDark = data.LogoLight
	}
	data.TermsURL = t.Legal.TermsURL
	data.PrivacyURL = t.Legal.PrivacyURL
	return data
}

// pageThemeCSS renders a theme as the CSS the hosted-page template inlines:
// the @font-face rules for the curated font (same-origin /assets/fonts
// URLs, nothing external) and a custom-properties block, with the dark
// palette behind prefers-color-scheme. All values come from a validated
// theme (strict #RRGGBB colors, curated fonts, bounded numbers), so the
// block is safe to emit as template.CSS.
func pageThemeCSS(t theme.Theme) template.CSS {
	var b strings.Builder
	family := "-apple-system, 'Segoe UI', Roboto, sans-serif"
	if f, ok := fonts.ByName(t.Typography.FontFamily); ok {
		if face, ok := fonts.FaceCSS(f.ID, "/assets/fonts"); ok {
			b.WriteString(face)
		}
		if fam, ok := fonts.FamilyCSS(f.ID); ok {
			family = fam
		}
	}
	writePalette := func(c theme.Colors) {
		fmt.Fprintf(&b, "  --primary: %s; --on-primary: %s;\n", c.Primary, c.OnPrimary)
		fmt.Fprintf(&b, "  --background: %s; --on-background: %s;\n", c.Background, c.OnBackground)
		fmt.Fprintf(&b, "  --surface: %s; --on-surface: %s;\n", c.Surface, c.OnSurface)
		fmt.Fprintf(&b, "  --error: %s; --on-error: %s;\n", c.Error, c.OnError)
	}
	b.WriteString(":root {\n")
	writePalette(t.Colors)
	fmt.Fprintf(&b, "  --font-family: %s;\n", family)
	fmt.Fprintf(&b, "  --font-scale: %g;\n", t.Typography.Scale)
	fmt.Fprintf(&b, "  --space: %dpx;\n", t.Spacing.Unit)
	fmt.Fprintf(&b, "  --radius: %dpx;\n", t.Shape.CornerRadius)
	b.WriteString("}\n@media (prefers-color-scheme: dark) {\n:root {\n")
	writePalette(t.EffectiveDark())
	b.WriteString("}\n}\n")
	return template.CSS(b.String())
}
