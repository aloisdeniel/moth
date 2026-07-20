package httpsec

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serve(p Policy, r *http.Request, h http.Handler) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	if h == nil {
		h = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	}
	p.Wrap(h).ServeHTTP(rec, r)
	return rec
}

func TestAdminPolicyHasNoUnsafeInline(t *testing.T) {
	p := DefaultAdminPolicy()
	if strings.Contains(p.ContentSecurityPolicy, "unsafe-inline") {
		t.Fatalf("admin CSP must not use unsafe-inline: %s", p.ContentSecurityPolicy)
	}
	if strings.Contains(p.ContentSecurityPolicy, "unsafe-eval") {
		t.Fatalf("admin CSP must not use unsafe-eval: %s", p.ContentSecurityPolicy)
	}
	if p.UsesNonce() {
		t.Fatal("admin policy is external-asset only and needs no nonce")
	}
	if !strings.Contains(p.ContentSecurityPolicy, "script-src 'self'") {
		t.Fatalf("admin CSP missing script-src 'self': %s", p.ContentSecurityPolicy)
	}
}

func TestHostedPolicyNonceNotUnsafeInline(t *testing.T) {
	p := DefaultHostedPolicy()
	if strings.Contains(p.ContentSecurityPolicy, "unsafe-inline") {
		t.Fatalf("hosted CSP must not use unsafe-inline: %s", p.ContentSecurityPolicy)
	}
	if !p.UsesNonce() {
		t.Fatal("hosted policy must use a nonce for its inline theme <style>")
	}
}

func TestAdminHeaders(t *testing.T) {
	rec := serve(DefaultAdminPolicy(), httptest.NewRequest("GET", "/admin", nil), nil)
	h := rec.Header()
	if got := h.Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'self'") {
		t.Fatalf("CSP = %q", got)
	}
	if got := h.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
	if got := h.Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q", got)
	}
	if got := h.Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Fatalf("Referrer-Policy = %q", got)
	}
	// No HSTS on the default admin policy (HSTS off, http request).
	if got := h.Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("unexpected HSTS = %q", got)
	}
}

func TestNoncePerRequestAndInContext(t *testing.T) {
	var seen []string
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = append(seen, NonceFromContext(r.Context()))
	})
	p := DefaultHostedPolicy()

	rec1 := serve(p, httptest.NewRequest("GET", "/p/x/verify", nil), handler)
	serve(p, httptest.NewRequest("GET", "/p/x/verify", nil), handler)

	if len(seen) != 2 || seen[0] == "" || seen[1] == "" {
		t.Fatalf("nonce not exposed in context: %v", seen)
	}
	if seen[0] == seen[1] {
		t.Fatal("nonce must be fresh per request")
	}
	// The nonce in the header matches the one handed to the handler and the
	// placeholder is fully substituted.
	csp := rec1.Header().Get("Content-Security-Policy")
	if strings.Contains(csp, noncePlaceholder) {
		t.Fatalf("placeholder not substituted: %s", csp)
	}
	if !strings.Contains(csp, "'nonce-"+seen[0]+"'") {
		t.Fatalf("header nonce %q not in CSP %q", seen[0], csp)
	}
}

func TestHSTSOnlyOverHTTPS(t *testing.T) {
	p := DefaultAdminPolicy()
	p.HSTS = true
	p.HSTSIncludeSubdomains = true

	// Plain http: no HSTS.
	rec := serve(p, httptest.NewRequest("GET", "/admin", nil), nil)
	if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("HSTS emitted over http: %q", got)
	}

	// Direct TLS: HSTS present.
	req := httptest.NewRequest("GET", "https://x/admin", nil)
	req.TLS = &tls.ConnectionState{}
	rec = serve(p, req, nil)
	got := rec.Header().Get("Strict-Transport-Security")
	if !strings.HasPrefix(got, "max-age=") || !strings.Contains(got, "includeSubDomains") {
		t.Fatalf("HSTS over TLS = %q", got)
	}

	// Forwarded https via proxy: HSTS present.
	req = httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec = serve(p, req, nil)
	if got := rec.Header().Get("Strict-Transport-Security"); got == "" {
		t.Fatal("HSTS not emitted for forwarded https")
	}
}
