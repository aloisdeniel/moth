package authrpc

import (
	"context"
	"net/http"
	"strings"

	"github.com/aloisdeniel/moth/internal/i18n"
	"github.com/aloisdeniel/moth/internal/store"
)

// Localization wiring: read the request's language, negotiate the best locale
// the project actually has (bundled ∪ its override locales), and resolve the
// bundled catalog copy merged with the project's overrides. Shared by the
// public config path (GetProjectConfig), the billing paywall path (GetPaywall),
// the hosted pages and the transactional emails, so one negotiation and one
// catalog serve every surface. The catalog itself lives in internal/i18n.

const (
	// LanguageHeader is the SDK's explicit locale request metadata. When
	// present it wins over Accept-Language (plan/15); the SDK sets it from the
	// device language in milestone 16.
	LanguageHeader = "x-moth-language"
	// LocaleHeader echoes the negotiated locale back on plain-HTTP and RPC
	// responses so the client caches per (locale, revision) correctly.
	LocaleHeader = "x-moth-locale"
)

// ProjectDefaultLocale is the locale a project falls back to before any
// requested locale is available. No per-project default-locale storage exists
// yet, so it is bundled English — always fully translated, so negotiation can
// always terminate on it.
const ProjectDefaultLocale = i18n.DefaultLocale

// ConfigCopyScreens are the SDK auth screens whose copy GetProjectConfig ships
// to the login/sign-up flows.
var ConfigCopyScreens = []i18n.Screen{
	i18n.ScreenSignIn, i18n.ScreenSignUp, i18n.ScreenPasswordReset, i18n.ScreenVerifyEmail,
}

// Localizer resolves catalog copy for a request's negotiated locale against a
// project's stored overrides. Build it with NewLocalizer (or
// NewFallbackLocalizer when a copy read fails and the surface must still
// render).
type Localizer struct {
	// Locale is the negotiated BCP-47 locale.
	Locale    i18n.Locale
	storeRev  string
	overrides i18n.Overrides
}

// NewLocalizer loads the project's copy overrides, negotiates the best locale
// for the request headers (x-moth-language, then Accept-Language) against the
// project's available locales (bundled ∪ its override locales) and returns a
// Localizer ready to resolve any screen's copy.
func NewLocalizer(ctx context.Context, st store.CopyStore, projectID string, h http.Header) (Localizer, error) {
	ov, storeRev, err := st.GetProjectCopy(ctx, projectID)
	if err != nil {
		return Localizer{}, err
	}
	return newLocalizer(ov, storeRev, h), nil
}

// NewFallbackLocalizer returns a Localizer on the project default locale with
// no overrides — the safe fallback when the copy store cannot be read, so a
// hosted page or email still renders bundled defaults.
func NewFallbackLocalizer() Localizer {
	return Localizer{Locale: ProjectDefaultLocale}
}

func newLocalizer(ov store.CopyOverrides, storeRev string, h http.Header) Localizer {
	loc := i18n.Negotiate(requestedLocales(h), availableLocales(ov), ProjectDefaultLocale)
	return Localizer{Locale: loc, storeRev: storeRev, overrides: toI18nOverrides(ov)}
}

// Token is the opaque (locale, override-revision) cache token echoed as
// Copy.copy_revision. It changes whenever the negotiated locale or the
// project's stored overrides change, so a locale switch re-sends the copy body
// even at the same store revision (the plan/15 caching contract).
func (l Localizer) Token() string {
	rev := l.storeRev
	if rev == "" {
		rev = "default"
	}
	return string(l.Locale) + "|" + rev
}

// Messages resolves every key of the given screens for the negotiated locale
// (bundled default merged with the project's overrides) and interpolates vars
// (e.g. {"app": project name}). Placeholders with no matching var are left
// verbatim for the client to fill at render time.
func (l Localizer) Messages(screens []i18n.Screen, vars map[string]string) map[string]string {
	var keys []i18n.Key
	for _, s := range screens {
		keys = append(keys, i18n.ScreenKeys(s)...)
	}
	resolved := i18n.Resolve(keys, l.Locale, ProjectDefaultLocale, l.overrides)
	out := make(map[string]string, len(resolved))
	for k, v := range resolved {
		out[string(k)] = i18n.Interpolate(v, vars)
	}
	return out
}

// Value resolves a single catalog key for the negotiated locale, interpolated
// with vars; "" for an unknown key.
func (l Localizer) Value(k i18n.Key, vars map[string]string) string {
	resolved := i18n.Resolve([]i18n.Key{k}, l.Locale, ProjectDefaultLocale, l.overrides)
	return i18n.Interpolate(resolved[k], vars)
}

// Dir is the text direction for the negotiated locale. No RTL locale is
// bundled yet, so it is always "ltr" (plan/15: RTL out of scope for the
// initial set).
func (l Localizer) Dir() string { return "ltr" }

// requestedLocales extracts the quality-ordered requested locales: the SDK's
// x-moth-language metadata wins when present, else the browser Accept-Language.
func requestedLocales(h http.Header) []i18n.Locale {
	if h == nil {
		return nil
	}
	if v := strings.TrimSpace(h.Get(LanguageHeader)); v != "" {
		if reqs := i18n.ParseAcceptLanguage(v); len(reqs) > 0 {
			return reqs
		}
	}
	return i18n.ParseAcceptLanguage(h.Get("Accept-Language"))
}

// availableLocales is the union of moth's bundled locales and the project's
// override locales — the set negotiation matches a request against.
func availableLocales(ov store.CopyOverrides) []i18n.Locale {
	set := map[i18n.Locale]bool{}
	var out []i18n.Locale
	add := func(l i18n.Locale) {
		if l != "" && !set[l] {
			set[l] = true
			out = append(out, l)
		}
	}
	for _, l := range i18n.BundledLocales {
		add(l)
	}
	for locale := range ov {
		add(i18n.NormalizeLocale(locale))
	}
	return out
}

// toI18nOverrides converts a store copy document into the i18n override shape,
// normalizing locale tags. Returns nil for an empty document (all bundled
// defaults).
func toI18nOverrides(ov store.CopyOverrides) i18n.Overrides {
	if len(ov) == 0 {
		return nil
	}
	out := make(i18n.Overrides, len(ov))
	for locale, msgs := range ov {
		m := make(map[i18n.Key]string, len(msgs))
		for k, v := range msgs {
			m[i18n.Key(k)] = v
		}
		out[i18n.NormalizeLocale(locale)] = m
	}
	return out
}

// copyValidator adapts the bundled catalog to store.CopyValidator so the store
// validates a project's overrides (known key, required placeholders, length)
// without importing internal/i18n.
type copyValidator struct{}

func (copyValidator) ValidateCopyOverrides(ov store.CopyOverrides) error {
	return i18n.ValidateOverrides(toI18nOverrides(ov))
}

// NewCopyValidator returns the store.CopyValidator backed by moth's bundled
// catalog. The admin CopyService passes it to store.UpdateProjectCopy.
func NewCopyValidator() store.CopyValidator { return copyValidator{} }
