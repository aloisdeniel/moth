package authrpc

import (
	"testing"

	"connectrpc.com/connect"
)

func TestCheckRedirectURI(t *testing.T) {
	schemes := []string{"myapp"}
	origins := []string{"https://app.example.com", "http://localhost:5173", "https://other.example.com:8443"}
	cases := []struct {
		name     string
		redirect string
		ok       bool
	}{
		// Custom schemes: matched on the scheme alone, as before.
		{"registered scheme", "myapp://auth", true},
		{"registered scheme case-insensitive", "MyApp://auth", true},
		{"unregistered scheme", "otherapp://auth", false},
		{"no scheme", "auth/callback", false},
		{"unparsable", "://", false},

		// Web origins: exact origin match.
		{"registered origin", "https://app.example.com", true},
		{"registered origin with path", "https://app.example.com/auth/callback", true},
		{"registered origin with query", "https://app.example.com/cb?next=%2Fhome", true},
		{"origin case-insensitive", "HTTPS://App.Example.COM/cb", true},
		{"default https port normalized", "https://app.example.com:443/cb", true},
		{"registered non-default port", "https://other.example.com:8443/cb", true},
		{"wrong port", "https://app.example.com:8080/cb", false},
		{"missing registered port", "https://other.example.com/cb", false},
		{"wrong host", "https://evil.example.com/cb", false},
		{"subdomain of registered origin", "https://sub.app.example.com/cb", false},
		{"registered origin as suffix", "https://evilapp.example.com/cb", false},
		{"http for registered https origin", "http://app.example.com/cb", false},
		{"http localhost registered", "http://localhost:5173/cb", true},
		{"http localhost wrong port", "http://localhost:3000/cb", false},
		{"fragment refused", "https://app.example.com/cb#frag", false},
		{"empty fragment refused", "https://app.example.com/cb#", false},
		{"userinfo refused", "https://user@app.example.com/cb", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkRedirectURI(tc.redirect, schemes, origins)
			if tc.ok && err != nil {
				t.Fatalf("checkRedirectURI(%q) = %v, want ok", tc.redirect, err)
			}
			if !tc.ok {
				if err == nil {
					t.Fatalf("checkRedirectURI(%q) = ok, want refusal", tc.redirect)
				}
				if connect.CodeOf(err) != connect.CodeInvalidArgument || ErrorReason(err) != ReasonInvalidRedirect {
					t.Fatalf("checkRedirectURI(%q) error: %v", tc.redirect, err)
				}
			}
		})
	}
	// Without registered origins, every http(s) redirect stays refused —
	// the pre-origins behavior.
	if err := checkRedirectURI("https://app.example.com/cb", schemes, nil); err == nil {
		t.Fatal("https redirect with no registered origins should be refused")
	}
}

// A stored origin that predates write-time canonicalization (or was written
// directly into the settings JSON) still matches: both sides are
// canonicalized at check time.
func TestCheckRedirectURINonCanonicalStoredOrigin(t *testing.T) {
	origins := []string{"HTTPS://App.Example.com:443/"}
	if err := checkRedirectURI("https://app.example.com/cb", nil, origins); err != nil {
		t.Fatalf("non-canonical stored origin should match: %v", err)
	}
}

func TestNormalizeRedirectOrigin(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // "" means an error is expected
	}{
		{"https origin", "https://app.example.com", "https://app.example.com"},
		{"lowercases scheme and host", "HTTPS://App.Example.COM", "https://app.example.com"},
		{"strips trailing slash", "https://app.example.com/", "https://app.example.com"},
		{"strips default https port", "https://app.example.com:443", "https://app.example.com"},
		{"keeps explicit port", "https://app.example.com:8443", "https://app.example.com:8443"},
		{"trims whitespace", "  https://app.example.com ", "https://app.example.com"},
		{"http localhost", "http://localhost:5173", "http://localhost:5173"},
		{"http loopback ip", "http://127.0.0.1:3000", "http://127.0.0.1:3000"},
		{"http ipv6 loopback", "http://[::1]:3000", "http://[::1]:3000"},
		{"strips default http port on localhost", "http://localhost:80", "http://localhost"},
		{"http non-localhost", "http://app.example.com", ""},
		{"custom scheme", "myapp://auth", ""},
		{"no scheme", "app.example.com", ""},
		{"missing host", "https://", ""},
		{"path refused", "https://app.example.com/auth", ""},
		{"query refused", "https://app.example.com?x=1", ""},
		{"fragment refused", "https://app.example.com#frag", ""},
		{"userinfo refused", "https://user@app.example.com", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeRedirectOrigin(tc.in)
			if tc.want == "" {
				if err == nil {
					t.Fatalf("NormalizeRedirectOrigin(%q) = %q, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeRedirectOrigin(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeRedirectOrigin(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
