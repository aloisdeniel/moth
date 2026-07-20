package authrpc

import (
	"context"
	"strings"

	"connectrpc.com/connect"

	"github.com/aloisdeniel/moth/internal/store"
)

// checkSignupEmail enforces the per-project signup email-domain allow/block
// lists. It is called on both password SignUp and social auto-create so a
// blocked domain cannot slip in through either door. An empty allowlist
// permits every domain; the blocklist is evaluated after the allowlist.
func checkSignupEmail(email string, settings store.ProjectSettings) error {
	if domainAllowed(emailDomain(email), settings.SignupEmailAllowlist, settings.SignupEmailBlocklist) {
		return nil
	}
	return newError(connect.CodePermissionDenied, ReasonEmailDomainNotAllowed,
		"this email domain is not allowed to sign up for this project")
}

// emailDomain returns the lowercased domain part of an email, or "".
func emailDomain(email string) string {
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(email[at+1:]))
}

// domainAllowed reports whether domain passes the allow/block lists. When
// allowlist is non-empty the domain must match one of its patterns; then, in
// either case, it must not match any blocklist pattern.
func domainAllowed(domain string, allowlist, blocklist []string) bool {
	if domain == "" {
		// A malformed address never has a valid domain; validEmail rejects it
		// earlier, so treat it as not allowed defensively.
		return len(allowlist) == 0
	}
	if len(allowlist) > 0 && !matchesAnyDomain(domain, allowlist) {
		return false
	}
	if matchesAnyDomain(domain, blocklist) {
		return false
	}
	return true
}

func matchesAnyDomain(domain string, patterns []string) bool {
	for _, p := range patterns {
		if matchDomainPattern(p, domain) {
			return true
		}
	}
	return false
}

// matchDomainPattern matches a single domain glob against a domain. Supported
// forms: an exact domain ("example.com"); a wildcard ("*.acme.io") matching
// the apex and any subdomain; a leading-dot suffix (".acme.io") matching any
// subdomain.
func matchDomainPattern(pattern, domain string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if pattern == "" {
		return false
	}
	switch {
	case strings.HasPrefix(pattern, "*."):
		base := pattern[2:]
		return domain == base || strings.HasSuffix(domain, "."+base)
	case strings.HasPrefix(pattern, "."):
		return strings.HasSuffix(domain, pattern)
	default:
		return domain == pattern
	}
}

// verifyCaptcha is the documented CAPTCHA hook. v1 ships it as a deliberate
// no-op: even when a project configures captcha_verify_url, enforcement stays
// off until the SDK is extended to carry a CAPTCHA token end to end. Wiring
// the call site now keeps the hook a one-function change later.
//
// TODO(post-v1): POST the token to settings.CaptchaVerifyURL and reject the
// request when the provider reports failure.
func (h *Handler) verifyCaptcha(_ context.Context, _ store.ProjectSettings, _ string) error {
	// Intentionally a no-op stub — see the package/plan CAPTCHA notes.
	return nil
}
