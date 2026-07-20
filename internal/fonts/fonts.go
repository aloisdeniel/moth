// Package fonts ships the curated set of open-license typefaces a project
// theme can pick from (plan/06). Embedding a small, hand-picked catalogue —
// instead of accepting arbitrary uploads — keeps mobile rendering
// predictable and the binary self-contained. Every face is a single
// variable-weight TTF under the SIL Open Font License; the OFL requires the
// license text to accompany the font, so each family's OFL.txt is embedded
// and served alongside its binary.
package fonts

import (
	"embed"
	"fmt"
	"net/http"
	"strings"
)

//go:embed data
var dataFS embed.FS

// DefaultID is the font every new project theme starts with.
const DefaultID = "inter"

// Font is one embedded typeface the theme editor can offer.
type Font struct {
	// ID is the stable identifier stored in the project theme JSON.
	ID string
	// Name is the display name in the admin selector and the CSS
	// font-family the face registers as.
	Name string
	// License is the face's license identifier.
	License string
	// Files are the font binaries, as paths relative to the serving base
	// URL (e.g. "inter/Inter.ttf" under "/assets/fonts/").
	Files []string
	// Fallback is the CSS font-family stack used after Name, so text
	// renders sensibly while the face downloads or if it fails to.
	Fallback string

	// weights is the CSS font-weight range the variable file covers,
	// emitted in the @font-face rule.
	weights string
}

// registry lists the embedded faces, default first. Weight ranges match the
// wght axis of each upstream variable font.
var registry = []Font{
	{
		ID:       "inter",
		Name:     "Inter",
		License:  "OFL-1.1",
		Files:    []string{"inter/Inter.ttf"},
		Fallback: "-apple-system, 'Segoe UI', Roboto, sans-serif",
		weights:  "100 900",
	},
	{
		ID:       "sourcesans3",
		Name:     "Source Sans 3",
		License:  "OFL-1.1",
		Files:    []string{"sourcesans3/SourceSans3.ttf"},
		Fallback: "-apple-system, 'Segoe UI', Roboto, sans-serif",
		weights:  "200 900",
	},
	{
		ID:       "nunitosans",
		Name:     "Nunito Sans",
		License:  "OFL-1.1",
		Files:    []string{"nunitosans/NunitoSans.ttf"},
		Fallback: "-apple-system, 'Segoe UI', Roboto, sans-serif",
		weights:  "200 1000",
	},
	{
		ID:       "lora",
		Name:     "Lora",
		License:  "OFL-1.1",
		Files:    []string{"lora/Lora.ttf"},
		Fallback: "Georgia, 'Times New Roman', serif",
		weights:  "400 700",
	},
	{
		ID:       "jetbrainsmono",
		Name:     "JetBrains Mono",
		License:  "OFL-1.1",
		Files:    []string{"jetbrainsmono/JetBrainsMono.ttf"},
		Fallback: "ui-monospace, 'SF Mono', Menlo, Consolas, monospace",
		weights:  "100 800",
	},
}

var byID = func() map[string]Font {
	m := make(map[string]Font, len(registry))
	for _, f := range registry {
		m[f.ID] = f
	}
	return m
}()

// servable maps every request path the handler answers to its content type:
// each registered binary plus the family's OFL.txt.
var servable = func() map[string]string {
	m := make(map[string]string)
	for _, f := range registry {
		for _, file := range f.Files {
			m[file] = "font/ttf"
		}
		m[f.ID+"/OFL.txt"] = "text/plain; charset=utf-8"
	}
	return m
}()

// List returns the embedded fonts in selector order, default first.
func List() []Font {
	out := make([]Font, len(registry))
	copy(out, registry)
	return out
}

// Get returns the font registered under id.
func Get(id string) (Font, bool) {
	f, ok := byID[id]
	return f, ok
}

// ByName returns the font whose display name matches name — the form a
// project theme stores in typography.fontFamily.
func ByName(name string) (Font, bool) {
	for _, f := range registry {
		if f.Name == name {
			return f, true
		}
	}
	return Font{}, false
}

// Files returns the binary file paths for id, relative to the serving base
// URL, or false for an unknown id.
func Files(id string) ([]string, bool) {
	f, ok := byID[id]
	if !ok {
		return nil, false
	}
	return append([]string(nil), f.Files...), true
}

// FaceCSS returns the @font-face rule(s) registering font id, with each
// source URL rooted at baseURL (e.g. "/assets/fonts"), or false for an
// unknown id. Hosted pages inline the result; the admin preview mirrors it
// client-side (web/admin/src/lib/theme.ts ensurePreviewFonts).
func FaceCSS(id, baseURL string) (string, bool) {
	f, ok := byID[id]
	if !ok {
		return "", false
	}
	base := strings.TrimSuffix(baseURL, "/")
	var b strings.Builder
	for _, file := range f.Files {
		fmt.Fprintf(&b, "@font-face {\n"+
			"  font-family: %q;\n"+
			"  font-style: normal;\n"+
			"  font-weight: %s;\n"+
			"  font-display: swap;\n"+
			"  src: url(%q) format(\"truetype\");\n"+
			"}\n", f.Name, f.weights, base+"/"+file)
	}
	return b.String(), true
}

// FamilyCSS returns the full CSS font-family value for id — the face's name
// followed by its fallback stack — or false for an unknown id.
func FamilyCSS(id string) (string, bool) {
	f, ok := byID[id]
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%q, %s", f.Name, f.Fallback), true
}

// Handler serves the embedded font binaries and their license texts. The
// server mounts it under the public assets prefix:
//
//	mux.Handle("/assets/fonts/", http.StripPrefix("/assets/fonts/", fonts.Handler()))
//
// Responses carry a one-year immutable Cache-Control: the embedded bytes
// only change with a new moth release, and clients that miss the update
// simply keep rendering the face they already have.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p := strings.TrimPrefix(r.URL.Path, "/")
		ct, ok := servable[p]
		if !ok {
			http.NotFound(w, r)
			return
		}
		raw, err := dataFS.ReadFile("data/" + p)
		if err != nil {
			// Registered but not embedded — a build defect the tests catch.
			http.Error(w, "font asset missing", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Write(raw)
	})
}
