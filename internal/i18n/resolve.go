package i18n

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// MaxValueLength caps the rune length of any project override string. It keeps
// a screen renderable no matter what an operator types — the same "always
// renders sensibly" promise the theme token ranges make.
const MaxValueLength = 400

// Overrides is a project's copy customization: locale → key → value. It is
// stored by the contracts agent (a copy config + revisions, plan/15) and
// passed into Resolve — this package never imports the store. Keys absent from
// a locale fall through to the merge chain below.
type Overrides map[Locale]map[Key]string

// Interpolate substitutes {name} placeholders in s with vars[name]. A
// placeholder with no matching var is left verbatim (so a partial vars map
// degrades visibly rather than silently blanking). Unknown-but-present braces
// that don't form a {name} token are untouched.
func Interpolate(s string, vars map[string]string) string {
	if len(vars) == 0 || !strings.ContainsRune(s, '{') {
		return s
	}
	return placeholderRE.ReplaceAllStringFunc(s, func(tok string) string {
		name := tok[1 : len(tok)-1]
		if v, ok := vars[name]; ok {
			return v
		}
		return tok
	})
}

// Render returns the bundled value for a key in a locale, interpolated with
// vars. It ignores project overrides — use Resolve for those — and is the
// path emails and hosted pages take when a project has no overrides. For a
// known key the result is never empty (English fallback); for an unknown key
// it returns "".
func Render(k Key, l Locale, vars map[string]string) string {
	v, ok := bundledValue(k, l)
	if !ok {
		return ""
	}
	return Interpolate(v, vars)
}

// BundledValue exposes the bundled (override-free) string for a key in a
// locale, with English fallback; ok is false for an unknown key. The admin
// editor uses it to show the "bundled default" hint next to an override field.
func BundledValue(k Key, l Locale) (string, bool) {
	return bundledValue(k, l)
}

// BundledLocale returns the complete bundled key→value map for a locale, every
// key resolved (English fallback for omissions). Handy for the admin editor's
// per-locale hint column.
func BundledLocale(l Locale) map[Key]string {
	out := make(map[Key]string, len(allKeys))
	for _, k := range allKeys {
		v, _ := bundledValue(k, l)
		out[k] = v
	}
	return out
}

// Resolve computes the effective, un-interpolated value for each requested key
// in the negotiated locale, merging three layers in increasing precedence:
//
//  1. bundled default for the negotiated locale (English fallback);
//  2. the project's override for its default locale (the project's rewritten
//     baseline copy, applied to every locale);
//  3. the project's override for the negotiated locale itself.
//
// This is the plan/15 merge order: bundled-default → project-default-locale →
// project-locale. Layer 2 lets a project restate a string once (in its default
// locale) and have it apply everywhere it hasn't been translated; layer 3 then
// refines it per locale. Pass keys=nil to resolve the whole catalog, or a
// screen's keys (ScreenKeys) to resolve one surface. The result is never
// interpolated — call Interpolate at render time with per-request vars.
func Resolve(keys []Key, locale, defaultLocale Locale, ov Overrides) map[Key]string {
	if keys == nil {
		keys = allKeys
	}
	locale = NormalizeLocale(string(locale))
	defaultLocale = NormalizeLocale(string(defaultLocale))
	out := make(map[Key]string, len(keys))
	for _, k := range keys {
		v, ok := bundledValue(k, locale)
		if !ok {
			continue // unknown key: skip
		}
		if dv, has := lookup(ov, defaultLocale, k); has {
			v = dv
		}
		if locale != defaultLocale {
			if lv, has := lookup(ov, locale, k); has {
				v = lv
			}
		}
		out[k] = v
	}
	return out
}

func lookup(ov Overrides, l Locale, k Key) (string, bool) {
	if ov == nil {
		return "", false
	}
	m, ok := ov[l]
	if !ok {
		return "", false
	}
	v, ok := m[k]
	return v, ok
}

// Validate checks a single project override value against the key's contract:
// the key must be known, the value must be non-empty and within MaxValueLength
// runes, and it must contain exactly the placeholder set of the key's English
// default (missing a required placeholder, or introducing an unknown one, is
// rejected). It returns a clear, operator-facing message.
func Validate(k Key, value string) error {
	if !IsKey(k) {
		return fmt.Errorf("unknown copy key %q", k)
	}
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s: value must not be empty", k)
	}
	if n := utf8.RuneCountInString(value); n > MaxValueLength {
		return fmt.Errorf("%s: value is %d characters, over the %d-character limit", k, n, MaxValueLength)
	}
	got := placeholdersIn(value)
	want := requiredPH[k]
	for _, p := range want {
		if !slices.Contains(got, p) {
			return fmt.Errorf("%s: value is missing the required placeholder {%s}", k, p)
		}
	}
	for _, p := range got {
		if !slices.Contains(want, p) {
			return fmt.Errorf("%s: value has an unknown placeholder {%s}", k, p)
		}
	}
	return nil
}

// ValidateOverrides validates every value in an override map and returns the
// first violation, or nil if all pass. Locale keys are canonicalized on the
// fly; an override for a locale moth doesn't bundle is allowed (it becomes the
// full source for that locale, plan/15).
func ValidateOverrides(ov Overrides) error {
	// Deterministic iteration for stable error messages.
	locs := make([]Locale, 0, len(ov))
	for l := range ov {
		locs = append(locs, l)
	}
	slices.Sort(locs)
	for _, l := range locs {
		m := ov[l]
		keys := make([]Key, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			if err := Validate(k, m[k]); err != nil {
				return fmt.Errorf("locale %s: %w", NormalizeLocale(string(l)), err)
			}
		}
	}
	return nil
}
