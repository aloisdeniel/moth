package authrpc

import (
	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/internal/fonts"
	mailpkg "github.com/aloisdeniel/moth/internal/mail"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/theme"
)

// DefaultThemeRevision is the revision id reported for projects rendering
// the built-in default theme. It is a stable non-empty sentinel so the
// GetProjectConfig caching contract ("theme omitted when
// known_theme_revision matches") also works for never-customized projects:
// an empty known_theme_revision (first call) never matches.
const DefaultThemeRevision = "default"

// ProjectTheme returns the theme a project renders with and the revision id
// identifying it: the stored theme, or the built-in default (with
// DefaultThemeRevision) when the project never customized anything.
func ProjectTheme(p store.Project) (theme.Theme, string) {
	if p.Theme == "" {
		return theme.Default(), DefaultThemeRevision
	}
	t, err := theme.Parse([]byte(p.Theme))
	if err != nil {
		// Stored themes were validated at write time; treat corruption as
		// "no theme" rather than breaking every login surface.
		return theme.Default(), DefaultThemeRevision
	}
	rev := p.ThemeRevisionID
	if rev == "" {
		rev = DefaultThemeRevision
	}
	return t, rev
}

// AssetURL builds the absolute, revision-keyed URL of a managed theme asset
// path ("/assets/{project}/logo-light.png"). The ?v= query defeats stale
// caches: the asset is served immutable, and every theme change mints a new
// revision id and therefore a new URL.
func AssetURL(baseURL, path, revision string) string {
	if path == "" {
		return ""
	}
	return baseURL + path + "?v=" + revision
}

// publicTheme resolves a project's theme into the public form the SDK
// renders from: the dark palette fully derived, asset references absolute
// URLs.
func publicTheme(p store.Project, baseURL string) *authv1.Theme {
	t, rev := ProjectTheme(p)
	msg := &authv1.Theme{
		RevisionId:   rev,
		Colors:       publicColors(t.Colors),
		DarkColors:   publicColors(t.EffectiveDark()),
		FontFamily:   t.Typography.FontFamily,
		FontScale:    t.Typography.Scale,
		SpacingUnit:  int32(t.Spacing.Unit),
		CornerRadius: int32(t.Shape.CornerRadius),
		LogoLightUrl: AssetURL(baseURL, t.Logo.Light, rev),
		LogoDarkUrl:  AssetURL(baseURL, t.Logo.Dark, rev),
		TermsUrl:     t.Legal.TermsURL,
		PrivacyUrl:   t.Legal.PrivacyURL,
	}
	if f, ok := fonts.ByName(t.Typography.FontFamily); ok && len(f.Files) > 0 {
		// The embedded font files only change with a moth release, so the
		// URL needs no revision key (they are served immutable regardless).
		msg.FontUrl = baseURL + "/assets/fonts/" + f.Files[0]
	}
	return msg
}

func publicColors(c theme.Colors) *authv1.ThemeColors {
	return &authv1.ThemeColors{
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

// Brand is the email branding derived from a project's theme: name, light
// logo and primary-color accents. Shared with the admin handlers, which
// send user-facing mail for the same projects.
func (h *Handler) Brand(p store.Project) mailpkg.Brand {
	t, rev := ProjectTheme(p)
	return mailpkg.Brand{
		Name:     p.Name,
		LogoURL:  AssetURL(h.baseURL, t.Logo.Light, rev),
		Accent:   t.Colors.Primary,
		OnAccent: t.Colors.OnPrimary,
	}
}
