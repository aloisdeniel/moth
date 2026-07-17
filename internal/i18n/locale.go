package i18n

import (
	"sort"
	"strconv"
	"strings"
)

// Locale is a BCP-47 language tag in moth's canonical form: a lowercase
// language subtag, optionally followed by "-" and an uppercase region subtag
// (e.g. "en", "fr", "pt-BR"). Use NormalizeLocale to canonicalize input.
type Locale string

// DefaultLocale is the ultimate fallback: bundled English is guaranteed
// complete for every key, so negotiation can always terminate here.
const DefaultLocale Locale = "en"

// BundledLocales is the curated set of locales moth ships translations for.
// English is mandatory and canonical (defined in Go); the others ship as
// embedded JSON (bundle.go) and are machine-quality baselines — good enough to
// render every screen, meant to be refined by project overrides. The set is
// kept small so the copy space stays reviewable and the binary self-contained.
var BundledLocales = []Locale{"en", "fr", "de", "es", "pt", "it", "ja"}

var bundledSet = func() map[Locale]bool {
	m := map[Locale]bool{}
	for _, l := range BundledLocales {
		m[l] = true
	}
	return m
}()

// IsBundled reports whether moth ships translations for a locale.
func IsBundled(l Locale) bool { return bundledSet[NormalizeLocale(string(l))] }

// NormalizeLocale canonicalizes a raw language tag: it trims whitespace,
// lowercases the language subtag, uppercases a two-letter region subtag, and
// drops anything beyond language-region (scripts, variants). "" stays "".
func NormalizeLocale(raw string) Locale {
	s := strings.TrimSpace(raw)
	if s == "" || s == "*" {
		return Locale(s)
	}
	s = strings.ReplaceAll(s, "_", "-")
	parts := strings.Split(s, "-")
	lang := strings.ToLower(parts[0])
	if len(parts) == 1 {
		return Locale(lang)
	}
	region := parts[1]
	if len(region) == 2 {
		return Locale(lang + "-" + strings.ToUpper(region))
	}
	// 3-digit UN M.49 region or a script subtag: keep language only for
	// script, keep region uppercased for M.49.
	if len(region) == 3 && isDigits(region) {
		return Locale(lang + "-" + region)
	}
	return Locale(lang)
}

// Base returns the language-only part of a locale ("fr-CA" → "fr").
func (l Locale) Base() Locale {
	s := string(l)
	if i := strings.IndexByte(s, '-'); i >= 0 {
		return Locale(s[:i])
	}
	return l
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// weighted pairs a parsed Accept-Language tag with its quality and original
// position, for a stable quality sort.
type weighted struct {
	loc Locale
	q   float64
	pos int
}

// ParseAcceptLanguage parses an HTTP Accept-Language header (RFC 7231) into
// canonical locales ordered by descending quality, ties broken by header
// order. Tags with q=0 are dropped, as is the "*" wildcard; malformed q values
// default to 1.0. An empty or all-dropped header yields nil.
func ParseAcceptLanguage(header string) []Locale {
	if strings.TrimSpace(header) == "" {
		return nil
	}
	var ws []weighted
	for i, raw := range strings.Split(header, ",") {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		q := 1.0
		if semi := strings.IndexByte(tag, ';'); semi >= 0 {
			params := tag[semi+1:]
			tag = strings.TrimSpace(tag[:semi])
			if v, ok := parseQ(params); ok {
				q = v
			}
		}
		loc := NormalizeLocale(tag)
		if loc == "" || loc == "*" || q <= 0 {
			continue
		}
		ws = append(ws, weighted{loc: loc, q: q, pos: i})
	}
	sort.SliceStable(ws, func(i, j int) bool {
		if ws[i].q != ws[j].q {
			return ws[i].q > ws[j].q
		}
		return ws[i].pos < ws[j].pos
	})
	out := make([]Locale, 0, len(ws))
	for _, w := range ws {
		out = append(out, w.loc)
	}
	return out
}

// parseQ extracts the q value from an Accept-Language parameter segment
// ("q=0.8"). It returns ok=false when no q is present.
func parseQ(params string) (float64, bool) {
	for _, p := range strings.Split(params, ";") {
		p = strings.TrimSpace(p)
		if k, v, ok := strings.Cut(p, "="); ok && strings.TrimSpace(k) == "q" {
			f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
			if err != nil {
				return 1.0, true
			}
			return f, true
		}
	}
	return 0, false
}

// Negotiate picks the best available locale for a quality-ordered request
// list, deterministically:
//
//  1. exact match — a requested tag equal to an available locale;
//  2. base-language match — a requested tag whose base language matches an
//     available locale's base (covers fr-CA→fr and fr→fr-CA);
//  3. the project default locale, if available;
//  4. DefaultLocale (bundled English), the guaranteed-complete fallback.
//
// available is the union of bundled locales and the project's override
// locales; the caller supplies it. requested and available are normalized
// defensively, so callers may pass raw tags.
func Negotiate(requested, available []Locale, defaultLocale Locale) Locale {
	avail := make([]Locale, 0, len(available))
	availSet := map[Locale]bool{}
	for _, a := range available {
		n := NormalizeLocale(string(a))
		if n == "" {
			continue
		}
		if !availSet[n] {
			availSet[n] = true
			avail = append(avail, n)
		}
	}

	// 1. exact.
	for _, r := range requested {
		n := NormalizeLocale(string(r))
		if availSet[n] {
			return n
		}
	}
	// 2. base-language match, honoring request order.
	for _, r := range requested {
		rb := NormalizeLocale(string(r)).Base()
		if availSet[rb] { // requested region tag, available has the base
			return rb
		}
		for _, a := range avail { // requested base, available has a region tag
			if a.Base() == rb {
				return a
			}
		}
	}
	// 3. project default.
	if d := NormalizeLocale(string(defaultLocale)); d != "" && availSet[d] {
		return d
	}
	// 4. bundled English.
	return DefaultLocale
}
