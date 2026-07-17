package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed locales/*.json
var localeFS embed.FS

// bundles holds the parsed non-English locale files: locale → key → value.
// English is not stored here — it is the Go-defined canonical source
// (englishByKey) and the fallback for any key a bundled locale omits.
var bundles = map[Locale]map[Key]string{}

// loadBundles parses every embedded locale JSON file. It is called once from
// the package init after the catalog maps are built. Files are trusted (they
// ship in the binary), so a malformed file or an unknown key is a build-time
// programming error and panics.
func loadBundles() {
	for _, loc := range BundledLocales {
		if loc == DefaultLocale {
			continue // English lives in Go
		}
		raw, err := localeFS.ReadFile("locales/" + string(loc) + ".json")
		if err != nil {
			panic(fmt.Sprintf("i18n: missing embedded locale file for %q: %v", loc, err))
		}
		var m map[Key]string
		if err := json.Unmarshal(raw, &m); err != nil {
			panic(fmt.Sprintf("i18n: parse locale %q: %v", loc, err))
		}
		for k := range m {
			if !IsKey(k) {
				panic(fmt.Sprintf("i18n: locale %q has unknown key %q", loc, k))
			}
		}
		bundles[loc] = m
	}
}

// bundledValue returns the bundled string for a key in a locale, falling back
// to English when the locale is unknown or omits the key. ok is false only for
// an unknown key. English is guaranteed non-empty for every known key, so a
// known key never yields an empty string.
func bundledValue(k Key, l Locale) (string, bool) {
	en, ok := englishByKey[k]
	if !ok {
		return "", false
	}
	if m := bundles[NormalizeLocale(string(l))]; m != nil {
		if v, has := m[k]; has && v != "" {
			return v, true
		}
	}
	return en, true
}
