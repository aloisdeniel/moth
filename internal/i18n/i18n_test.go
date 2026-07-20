package i18n

import (
	"strings"
	"testing"
)

// TestCatalogCompleteness asserts every bundled locale renders a non-empty
// string for every key (via English fallback when a translation is missing),
// so no screen can ever show a blank field.
func TestCatalogCompleteness(t *testing.T) {
	for _, loc := range BundledLocales {
		for _, k := range AllKeys() {
			got := Render(k, loc, nil)
			if got == "" {
				t.Errorf("Render(%s, %s) is empty", k, loc)
			}
		}
	}
}

// TestBundledLocaleFilesOnlyKnownKeys is enforced at init (loadBundles
// panics), but this pins the invariant and that every locale file parsed.
func TestBundledLocalesParsed(t *testing.T) {
	for _, loc := range BundledLocales {
		if loc == DefaultLocale {
			continue
		}
		if bundles[loc] == nil {
			t.Errorf("locale %s did not load", loc)
		}
	}
}

// TestBundledLocalesComplete asserts every non-English bundle carries a
// non-empty translation for every catalog key — WITHOUT the English fallback
// that Render/TestCatalogCompleteness would mask a gap with. This pins the
// plan/15 invariant that "every key has a bundled default in every bundled
// locale", so adding a catalog key without translating it fails the build.
func TestBundledLocalesComplete(t *testing.T) {
	for _, loc := range BundledLocales {
		if loc == DefaultLocale {
			continue // English is the Go-defined source, not a bundle file.
		}
		m := bundles[loc]
		for _, k := range AllKeys() {
			if v, ok := m[k]; !ok || strings.TrimSpace(v) == "" {
				t.Errorf("locale %s is missing a translation for %s", loc, k)
			}
		}
	}
}

// TestEnglishNonEmpty guards the canonical source: every key has a non-empty
// English default (the fallback that makes every render non-empty).
func TestEnglishNonEmpty(t *testing.T) {
	for _, k := range AllKeys() {
		v, ok := English(k)
		if !ok || strings.TrimSpace(v) == "" {
			t.Errorf("English(%s) empty/missing", k)
		}
	}
}

func TestScreenKeysPartition(t *testing.T) {
	var total int
	for _, s := range Screens() {
		ks := ScreenKeys(s)
		if len(ks) == 0 {
			t.Errorf("screen %s has no keys", s)
		}
		for _, k := range ks {
			if sc, _ := ScreenOf(k); sc != s {
				t.Errorf("key %s reported screen %s, want %s", k, sc, s)
			}
		}
		total += len(ks)
	}
	if total != len(AllKeys()) {
		t.Errorf("screen keys sum to %d, AllKeys has %d", total, len(AllKeys()))
	}
	if ScreenKeys("nope") != nil {
		t.Error("unknown screen should yield nil")
	}
}

func TestParseAcceptLanguage(t *testing.T) {
	tests := []struct {
		header string
		want   []Locale
	}{
		{"", nil},
		{"   ", nil},
		{"en", []Locale{"en"}},
		{"fr-CA", []Locale{"fr-CA"}},
		{"fr-ca", []Locale{"fr-CA"}},
		{"en-US,en;q=0.9,fr;q=0.8", []Locale{"en-US", "en", "fr"}},
		{"fr;q=0.5,de;q=0.9", []Locale{"de", "fr"}},  // reordered by q
		{"de;q=0,fr", []Locale{"fr"}},                // q=0 dropped
		{"*;q=0.5,ja", []Locale{"ja"}},               // wildcard dropped
		{"en;q=bad,fr;q=0.1", []Locale{"en", "fr"}},  // bad q -> 1.0
		{"pt-BR, pt;q=0.9", []Locale{"pt-BR", "pt"}}, // stable tie & order
	}
	for _, tt := range tests {
		got := ParseAcceptLanguage(tt.header)
		if !equalLocales(got, tt.want) {
			t.Errorf("ParseAcceptLanguage(%q) = %v, want %v", tt.header, got, tt.want)
		}
	}
}

func TestNegotiate(t *testing.T) {
	avail := []Locale{"en", "fr", "de", "pt-BR"}
	tests := []struct {
		name      string
		requested []Locale
		def       Locale
		want      Locale
	}{
		{"exact", []Locale{"fr"}, "en", "fr"},
		{"base fr-CA->fr", []Locale{"fr-CA"}, "en", "fr"},
		{"base fr->pt-BR not matched, fallback de wanted", []Locale{"nl", "de"}, "en", "de"},
		{"request base matches region avail (pt->pt-BR)", []Locale{"pt"}, "en", "pt-BR"},
		{"unknown -> project default", []Locale{"zh"}, "de", "de"},
		{"unknown & default unavailable -> english", []Locale{"zh"}, "ru", "en"},
		{"empty request -> project default", nil, "fr", "fr"},
		{"quality order picks higher", []Locale{"de", "fr"}, "en", "de"},
		{"exact beats base across list", []Locale{"fr-CA", "de"}, "en", "de"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Negotiate(tt.requested, avail, tt.def); got != tt.want {
				t.Errorf("Negotiate(%v, def=%s) = %s, want %s", tt.requested, tt.def, got, tt.want)
			}
		})
	}
}

func TestNegotiateEndToEndFromHeader(t *testing.T) {
	// Acceptance: Accept-Language: fr resolves to fr against bundled locales.
	got := Negotiate(ParseAcceptLanguage("fr"), BundledLocales, "en")
	if got != "fr" {
		t.Fatalf("got %s, want fr", got)
	}
}

func TestInterpolate(t *testing.T) {
	tests := []struct {
		s    string
		vars map[string]string
		want string
	}{
		{"Welcome to {app}.", map[string]string{"app": "Acme"}, "Welcome to Acme."},
		{"no placeholders", map[string]string{"app": "x"}, "no placeholders"},
		{"{a} and {b}", map[string]string{"a": "1", "b": "2"}, "1 and 2"},
		{"{missing}", map[string]string{"other": "x"}, "{missing}"}, // left verbatim
		{"{app}", nil, "{app}"},
	}
	for _, tt := range tests {
		if got := Interpolate(tt.s, tt.vars); got != tt.want {
			t.Errorf("Interpolate(%q) = %q, want %q", tt.s, got, tt.want)
		}
	}
}

func TestRenderInterpolates(t *testing.T) {
	got := Render(SignInSubtitle, "en", map[string]string{"app": "Acme"})
	if got != "Welcome back to Acme." {
		t.Errorf("got %q", got)
	}
	if fr := Render(SignInSubtitle, "fr", map[string]string{"app": "Acme"}); !strings.Contains(fr, "Acme") {
		t.Errorf("fr render lost placeholder value: %q", fr)
	}
	if Render(Key("bogus.key"), "en", nil) != "" {
		t.Error("unknown key should render empty")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		key     Key
		value   string
		wantErr bool
	}{
		{"ok plain", SignInTitle, "Connexion", false},
		{"ok with placeholder", SignInSubtitle, "Salut {app}", false},
		{"missing required placeholder", SignInSubtitle, "Salut", true},
		{"unknown placeholder", SignInTitle, "Hi {app}", true},
		{"empty", SignInTitle, "   ", true},
		{"unknown key", Key("nope.x"), "y", true},
		{"too long", SignInTitle, strings.Repeat("x", MaxValueLength+1), true},
		{"at limit", SignInTitle, strings.Repeat("x", MaxValueLength), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%s, %q) err=%v, wantErr=%v", tt.key, tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestResolveMergePrecedence(t *testing.T) {
	ov := Overrides{
		"en": {SignInTitle: "EN base override"},
		"fr": {SignInTitle: "FR override"},
	}
	// negotiated fr, project default en: fr override wins.
	got := Resolve([]Key{SignInTitle}, "fr", "en", ov)
	if got[SignInTitle] != "FR override" {
		t.Errorf("fr resolve = %q, want FR override", got[SignInTitle])
	}
	// negotiated de (no de override), default en: default-locale override applies.
	got = Resolve([]Key{SignInTitle}, "de", "en", ov)
	if got[SignInTitle] != "EN base override" {
		t.Errorf("de resolve = %q, want EN base override", got[SignInTitle])
	}
	// negotiated de, unedited key: bundled German default.
	got = Resolve([]Key{SignInSubmit}, "de", "en", ov)
	if want, _ := bundledValue(SignInSubmit, "de"); got[SignInSubmit] != want {
		t.Errorf("de bundled = %q, want %q", got[SignInSubmit], want)
	}
	// no overrides at all: bundled default.
	got = Resolve([]Key{SignInTitle}, "fr", "en", nil)
	if want, _ := bundledValue(SignInTitle, "fr"); got[SignInTitle] != want {
		t.Errorf("nil-override resolve = %q, want %q", got[SignInTitle], want)
	}
}

func TestResolveAllKeysNonEmpty(t *testing.T) {
	got := Resolve(nil, "ja", "en", nil)
	if len(got) != len(AllKeys()) {
		t.Fatalf("resolved %d keys, want %d", len(got), len(AllKeys()))
	}
	for k, v := range got {
		if v == "" {
			t.Errorf("resolved %s empty", k)
		}
	}
}

func TestValidateOverrides(t *testing.T) {
	good := Overrides{"fr": {SignInSubtitle: "Salut {app}"}}
	if err := ValidateOverrides(good); err != nil {
		t.Errorf("good overrides rejected: %v", err)
	}
	bad := Overrides{"fr": {SignInSubtitle: "Salut"}}
	if err := ValidateOverrides(bad); err == nil {
		t.Error("bad overrides accepted")
	}
}

func TestNormalizeLocale(t *testing.T) {
	tests := []struct{ in, want string }{
		{"EN", "en"},
		{"fr-ca", "fr-CA"},
		{"pt_BR", "pt-BR"},
		{"zh-Hant", "zh"},
		{"es-419", "es-419"},
		{"  de  ", "de"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := NormalizeLocale(tt.in); got != Locale(tt.want) {
			t.Errorf("NormalizeLocale(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func equalLocales(a, b []Locale) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
