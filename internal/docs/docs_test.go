package docs

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mounted mirrors how internal/server mounts the handler: stripped of the
// /docs prefix, so a request path of "/docs/<slug>" reaches render(<slug>).
func mounted() http.Handler {
	mux := http.NewServeMux()
	h := http.StripPrefix("/docs", Handler())
	mux.Handle("GET /docs", h)
	mux.Handle("GET /docs/", h)
	return mux
}

// TestHandlerServesIndex renders the docs index and checks the chrome and
// content both come through.
func TestHandlerServesIndex(t *testing.T) {
	for _, path := range []string{"/", ""} {
		rec := httptest.NewRecorder()
		mounted().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/docs"+path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("index %q: status = %d, want 200", path, rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "<nav>") || !strings.Contains(body, "self-hosted authentication") {
			t.Fatalf("index %q: missing chrome or content:\n%s", path, body[:min(len(body), 400)])
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Fatalf("index %q: content-type = %q", path, ct)
		}
	}
}

// TestHandlerRendersEveryPage proves every navigable slug is present in the
// embedded tree and renders to non-trivial HTML — a stale sidebar or a
// missing sync would fail here.
func TestHandlerRendersEveryPage(t *testing.T) {
	for _, slug := range Slugs() {
		rec := httptest.NewRecorder()
		mounted().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/docs/"+slug, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("slug %q: status = %d, want 200", slug, rec.Code)
			continue
		}
		if rec.Body.Len() < 500 {
			t.Errorf("slug %q: body only %d bytes", slug, rec.Body.Len())
		}
	}
}

// TestHandlerRendersMarkdown checks the markdown actually became HTML (a
// fenced code block and a heading), not passed through as raw text.
func TestHandlerRendersMarkdown(t *testing.T) {
	rec := httptest.NewRecorder()
	mounted().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/docs/quick-start", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "<h2") {
		t.Errorf("quick-start: no rendered <h2> heading")
	}
	if !strings.Contains(body, "<pre>") && !strings.Contains(body, "<code>") {
		t.Errorf("quick-start: no rendered code block")
	}
}

func TestHandlerUnknownSlug404(t *testing.T) {
	rec := httptest.NewRecorder()
	mounted().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/docs/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown slug: status = %d, want 404", rec.Code)
	}
}

func TestHandlerNoRawFrontmatter(t *testing.T) {
	rec := httptest.NewRecorder()
	mounted().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/docs/installation", nil))
	if body, _ := io.ReadAll(rec.Body); strings.Contains(string(body), ":::note") {
		t.Errorf("installation: Starlight admonition leaked into rendered HTML")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
