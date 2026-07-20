package fonts

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
)

// ttfMagic is the sfnt version tag opening every TrueType-flavoured font.
var ttfMagic = []byte{0x00, 0x01, 0x00, 0x00}

func TestRegistryComplete(t *testing.T) {
	fonts := List()
	if len(fonts) == 0 {
		t.Fatal("List returned no fonts")
	}
	if fonts[0].ID != DefaultID {
		t.Fatalf("List()[0].ID = %q, want default %q first", fonts[0].ID, DefaultID)
	}
	if _, ok := Get(DefaultID); !ok {
		t.Fatalf("default font %q not registered", DefaultID)
	}

	seen := map[string]bool{}
	var total int
	for _, f := range fonts {
		if seen[f.ID] {
			t.Errorf("duplicate font id %q", f.ID)
		}
		seen[f.ID] = true
		if f.Name == "" || f.License == "" || f.Fallback == "" || f.weights == "" {
			t.Errorf("font %q has empty metadata: %+v", f.ID, f)
		}

		files, ok := Files(f.ID)
		if !ok || len(files) == 0 {
			t.Fatalf("Files(%q) = %v, %v; want at least one file", f.ID, files, ok)
		}
		for _, file := range files {
			raw, err := dataFS.ReadFile("data/" + file)
			if err != nil {
				t.Fatalf("font %q: file %q not embedded: %v", f.ID, file, err)
			}
			if len(raw) == 0 {
				t.Fatalf("font %q: file %q is empty", f.ID, file)
			}
			if !bytes.HasPrefix(raw, ttfMagic) {
				t.Errorf("font %q: file %q does not start with TTF magic bytes", f.ID, file)
			}
			total += len(raw)
		}

		lic, err := dataFS.ReadFile("data/" + f.ID + "/OFL.txt")
		if err != nil {
			t.Fatalf("font %q: OFL.txt not embedded: %v", f.ID, err)
		}
		if !strings.Contains(string(lic), "SIL OPEN FONT LICENSE") {
			t.Errorf("font %q: OFL.txt does not look like the OFL", f.ID)
		}
	}
	t.Logf("%d fonts, %d bytes embedded", len(fonts), total)
}

func TestFilesUnknown(t *testing.T) {
	if files, ok := Files("comic-sans"); ok {
		t.Fatalf("Files for unknown id returned %v", files)
	}
}

func TestHandlerServesFont(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/inter/Inter.ttf", nil))

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "font/ttf" {
		t.Errorf("Content-Type = %q, want font/ttf", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q", got)
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), ttfMagic) {
		t.Error("body does not start with TTF magic bytes")
	}
}

func TestHandlerServesLicense(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/inter/OFL.txt", nil))

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
	if !strings.Contains(rec.Body.String(), "SIL OPEN FONT LICENSE") {
		t.Error("body does not look like the OFL")
	}
}

func TestHandlerRejects(t *testing.T) {
	cases := []struct {
		name, method, path string
		want               int
	}{
		{"unknown font", "GET", "/nosuch/NoSuch.ttf", 404},
		{"unregistered path", "GET", "/inter/METADATA.pb", 404},
		{"directory", "GET", "/inter/", 404},
		{"traversal", "GET", "/../fonts.go", 404},
		{"post", "POST", "/inter/Inter.ttf", 405},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			Handler().ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}

func TestFaceCSS(t *testing.T) {
	css, ok := FaceCSS("inter", "/assets/fonts/")
	if !ok {
		t.Fatal("FaceCSS(inter) not ok")
	}
	for _, want := range []string{
		"@font-face {",
		`font-family: "Inter";`,
		"font-weight: 100 900;",
		"font-display: swap;",
		`src: url("/assets/fonts/inter/Inter.ttf") format("truetype");`,
	} {
		if !strings.Contains(css, want) {
			t.Errorf("FaceCSS missing %q in:\n%s", want, css)
		}
	}
	if _, ok := FaceCSS("nosuch", "/assets/fonts"); ok {
		t.Error("FaceCSS for unknown id reported ok")
	}
}

func TestFamilyCSS(t *testing.T) {
	fam, ok := FamilyCSS("lora")
	if !ok {
		t.Fatal("FamilyCSS(lora) not ok")
	}
	if want := `"Lora", Georgia, 'Times New Roman', serif`; fam != want {
		t.Errorf("FamilyCSS = %q, want %q", fam, want)
	}
	if _, ok := FamilyCSS("nosuch"); ok {
		t.Error("FamilyCSS for unknown id reported ok")
	}
}
