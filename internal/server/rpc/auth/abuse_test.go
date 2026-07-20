package authrpc

import (
	"testing"

	"connectrpc.com/connect"

	"github.com/aloisdeniel/moth/internal/store"
)

func TestDomainAllowed(t *testing.T) {
	cases := []struct {
		name       string
		domain     string
		allow      []string
		block      []string
		wantResult bool
	}{
		{"no lists allows any", "example.com", nil, nil, true},
		{"allowlist exact match", "example.com", []string{"example.com"}, nil, true},
		{"allowlist rejects other", "evil.com", []string{"example.com"}, nil, false},
		{"allowlist wildcard apex", "acme.io", []string{"*.acme.io"}, nil, true},
		{"allowlist wildcard sub", "eu.acme.io", []string{"*.acme.io"}, nil, true},
		{"allowlist wildcard rejects other", "acme.io.evil.com", []string{"*.acme.io"}, nil, false},
		{"blocklist rejects", "spam.com", nil, []string{"spam.com"}, false},
		{"blocklist wildcard rejects sub", "x.spam.io", nil, []string{"*.spam.io"}, false},
		{"block wins over allow", "sub.acme.io", []string{"*.acme.io"}, []string{"sub.acme.io"}, false},
		{"allow passes, block misses", "ok.acme.io", []string{"*.acme.io"}, []string{"bad.acme.io"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := domainAllowed(tc.domain, tc.allow, tc.block); got != tc.wantResult {
				t.Fatalf("domainAllowed(%q) = %v, want %v", tc.domain, got, tc.wantResult)
			}
		})
	}
}

func TestCheckSignupEmail(t *testing.T) {
	settings := store.ProjectSettings{
		SignupEmailAllowlist: []string{"example.com"},
		SignupEmailBlocklist: []string{"blocked.example.com"},
	}
	if err := checkSignupEmail("jane@example.com", settings); err != nil {
		t.Fatalf("allowed domain rejected: %v", err)
	}
	err := checkSignupEmail("jane@evil.com", settings)
	if err == nil {
		t.Fatal("disallowed domain accepted")
	}
	if got := connect.CodeOf(err); got != connect.CodePermissionDenied {
		t.Fatalf("code = %v, want PermissionDenied", got)
	}
	if got := ErrorReason(err); got != ReasonEmailDomainNotAllowed {
		t.Fatalf("reason = %q, want %q", got, ReasonEmailDomainNotAllowed)
	}
}
