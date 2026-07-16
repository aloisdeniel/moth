// Package httpsec is the security-header middleware for moth's HTML and JSON
// surfaces: a strict Content-Security-Policy, HSTS (https only),
// X-Content-Type-Options, Referrer-Policy and frame options.
//
// CSP and unsafe-inline. The two families of pages moth serves are covered by
// two default policies, neither of which uses 'unsafe-inline':
//
//   - The admin SPA (DefaultAdminPolicy) is a Vite production build: its only
//     script and stylesheet are external files loaded with src=/href=, so a
//     plain script-src 'self'; style-src 'self' is sufficient — no inline
//     code, no nonce, no hash.
//   - The hosted pages (DefaultHostedPolicy) — verify / reset / confirm-email
//     — render one inline <style> block carrying the project's theme CSS.
//     Rather than weaken the policy with 'unsafe-inline', the middleware mints
//     a fresh base64 nonce per request, substitutes it for the "%NONCE%"
//     token in the policy, and exposes it via NonceFromContext so the template
//     can stamp nonce="…" onto that <style> element. script-src is 'none'
//     because the hosted pages ship no JavaScript at all.
//
// A per-request nonce (not a static hash) is used for the hosted pages
// because the inline CSS is dynamic — it embeds each project's theme tokens —
// so its hash is not stable across projects or theme edits.
package httpsec

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// noncePlaceholder is replaced with a fresh per-request nonce wherever it
// appears in Policy.ContentSecurityPolicy.
const noncePlaceholder = "%NONCE%"

// Policy is the set of security headers to emit. Construct one of the
// defaults and adjust as needed, or build your own.
type Policy struct {
	// ContentSecurityPolicy is the Content-Security-Policy header value. When
	// it contains the token %NONCE%, each request gets a fresh nonce
	// substituted in and made available through NonceFromContext.
	ContentSecurityPolicy string
	// HSTS enables Strict-Transport-Security. It is only emitted on requests
	// served over https (or forwarded as https), never plain http, so a
	// misconfiguration cannot lock clients out of an http instance.
	HSTS bool
	// HSTSMaxAge is the max-age of the HSTS header; defaults to 365 days when
	// zero and HSTS is enabled.
	HSTSMaxAge time.Duration
	// HSTSIncludeSubdomains adds includeSubDomains to the HSTS header.
	HSTSIncludeSubdomains bool
	// FrameOptions sets X-Frame-Options (e.g. "DENY", "SAMEORIGIN"); empty
	// omits the header (rely on CSP frame-ancestors instead).
	FrameOptions string
	// ReferrerPolicy sets Referrer-Policy; empty omits the header.
	ReferrerPolicy string
	// ContentTypeOptions emits X-Content-Type-Options: nosniff when true.
	ContentTypeOptions bool
}

// DefaultAdminPolicy returns the strict policy for the admin SPA and JSON
// APIs: no inline code, no nonce. HSTS is left to the caller to enable when
// the instance is served over https.
func DefaultAdminPolicy() Policy {
	return Policy{
		ContentSecurityPolicy: strings.Join([]string{
			"default-src 'self'",
			"script-src 'self'",
			"style-src 'self'",
			"img-src 'self' data:",
			"font-src 'self' data:",
			"connect-src 'self'",
			"frame-ancestors 'none'",
			"base-uri 'self'",
			"form-action 'self'",
			"object-src 'none'",
		}, "; "),
		FrameOptions:       "DENY",
		ReferrerPolicy:     "strict-origin-when-cross-origin",
		ContentTypeOptions: true,
	}
}

// DefaultHostedPolicy returns the policy for the server-rendered hosted pages.
// It permits the single inline <style> block via a per-request nonce and
// forbids scripts entirely.
func DefaultHostedPolicy() Policy {
	return Policy{
		ContentSecurityPolicy: strings.Join([]string{
			"default-src 'self'",
			"script-src 'none'",
			"style-src 'nonce-" + noncePlaceholder + "'",
			"img-src 'self' data:",
			"font-src 'self' data:",
			"frame-ancestors 'none'",
			"base-uri 'none'",
			"form-action 'self'",
			"object-src 'none'",
		}, "; "),
		FrameOptions:       "DENY",
		ReferrerPolicy:     "strict-origin-when-cross-origin",
		ContentTypeOptions: true,
	}
}

// UsesNonce reports whether the policy's CSP embeds the nonce placeholder.
func (p Policy) UsesNonce() bool {
	return strings.Contains(p.ContentSecurityPolicy, noncePlaceholder)
}

type nonceKey struct{}

// NonceFromContext returns the CSP nonce assigned to the current request, or
// "" when the active policy does not use a nonce. Templates read it to stamp
// nonce="…" onto their inline <style>/<script> elements.
func NonceFromContext(ctx context.Context) string {
	n, _ := ctx.Value(nonceKey{}).(string)
	return n
}

// Wrap returns an http.Handler that applies p's headers to every response
// from next.
func (p Policy) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		csp := p.ContentSecurityPolicy
		if p.UsesNonce() {
			nonce := newNonce()
			csp = strings.ReplaceAll(csp, noncePlaceholder, nonce)
			r = r.WithContext(context.WithValue(r.Context(), nonceKey{}, nonce))
		}

		h := w.Header()
		if csp != "" {
			h.Set("Content-Security-Policy", csp)
		}
		if p.ContentTypeOptions {
			h.Set("X-Content-Type-Options", "nosniff")
		}
		if p.FrameOptions != "" {
			h.Set("X-Frame-Options", p.FrameOptions)
		}
		if p.ReferrerPolicy != "" {
			h.Set("Referrer-Policy", p.ReferrerPolicy)
		}
		if p.HSTS && isHTTPS(r) {
			h.Set("Strict-Transport-Security", p.hstsValue())
		}
		next.ServeHTTP(w, r)
	})
}

func (p Policy) hstsValue() string {
	age := p.HSTSMaxAge
	if age <= 0 {
		age = 365 * 24 * time.Hour
	}
	v := "max-age=" + strconv.FormatInt(int64(age.Seconds()), 10)
	if p.HSTSIncludeSubdomains {
		v += "; includeSubDomains"
	}
	return v
}

// isHTTPS reports whether the request reached moth over TLS, directly
// (r.TLS set) or via a trusted proxy that set X-Forwarded-Proto: https.
// Callers that terminate TLS at a proxy should only rely on this behind a
// trusted-proxy check upstream.
func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func newNonce() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return base64.StdEncoding.EncodeToString(b[:])
}
