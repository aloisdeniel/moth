package server

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestDocsRouteServesEmbeddedContent hits the /docs route on the fully
// assembled server and checks the embedded, rendered documentation comes
// back — the milestone-10 "/docs embedding pipeline" acceptance.
func TestDocsRouteServesEmbeddedContent(t *testing.T) {
	e := newTestEnv(t, "")

	cases := []struct {
		path string
		want string // a phrase that must appear in the rendered HTML
	}{
		{"/docs", "self-hosted authentication"},
		{"/docs/quick-start", "<h2"},      // markdown rendered to HTML
		{"/docs/installation", "systemd"}, // a deep page
		{"/docs/cli/reference", "moth"},   // nested slug + generated ref
	}
	for _, tc := range cases {
		resp, err := e.client.Get(e.url + tc.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tc.path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s: status %d, want 200", tc.path, resp.StatusCode)
			continue
		}
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("GET %s: content-type %q", tc.path, ct)
		}
		if !strings.Contains(string(body), tc.want) {
			t.Errorf("GET %s: rendered HTML missing %q", tc.path, tc.want)
		}
	}

	// Unknown pages 404 rather than serving the shell.
	resp, err := e.client.Get(e.url + "/docs/does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown docs page: status %d, want 404", resp.StatusCode)
	}
}
