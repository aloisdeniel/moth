package netutil

import "testing"

func mustProxies(t *testing.T, cidrs ...string) *TrustedProxies {
	t.Helper()
	tp, err := ParseTrustedProxies(cidrs)
	if err != nil {
		t.Fatalf("ParseTrustedProxies(%v): %v", cidrs, err)
	}
	return tp
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name    string
		proxies *TrustedProxies
		remote  string
		xff     string
		want    string
	}{
		{
			name:    "no trusted proxies ignores xff",
			proxies: mustProxies(t),
			remote:  "203.0.113.7:44321",
			xff:     "1.2.3.4",
			want:    "203.0.113.7",
		},
		{
			name:    "untrusted peer ignores spoofed xff",
			proxies: mustProxies(t, "10.0.0.0/8"),
			remote:  "203.0.113.7:5000",
			xff:     "1.2.3.4, 5.6.7.8",
			want:    "203.0.113.7",
		},
		{
			name:    "trusted peer honours xff",
			proxies: mustProxies(t, "10.0.0.0/8"),
			remote:  "10.1.2.3:5000",
			xff:     "198.51.100.9",
			want:    "198.51.100.9",
		},
		{
			name:    "walks past chained trusted proxies",
			proxies: mustProxies(t, "10.0.0.0/8"),
			remote:  "10.0.0.1:5000",
			xff:     "198.51.100.9, 10.0.0.9, 10.0.0.8",
			want:    "198.51.100.9",
		},
		{
			name:    "spoofed client prefix does not fool right-walk",
			proxies: mustProxies(t, "10.0.0.0/8"),
			remote:  "10.0.0.1:5000",
			// Attacker injected 1.1.1.1; the real edge is 203.0.113.5.
			xff:  "1.1.1.1, 203.0.113.5, 10.0.0.9",
			want: "203.0.113.5",
		},
		{
			name:    "all hops trusted falls back to leftmost",
			proxies: mustProxies(t, "10.0.0.0/8"),
			remote:  "10.0.0.1:5000",
			xff:     "10.0.0.5, 10.0.0.6",
			want:    "10.0.0.5",
		},
		{
			name:    "trusted peer empty xff returns peer",
			proxies: mustProxies(t, "10.0.0.0/8"),
			remote:  "10.0.0.1:5000",
			xff:     "",
			want:    "10.0.0.1",
		},
		{
			name:    "bare host without port",
			proxies: mustProxies(t),
			remote:  "203.0.113.7",
			xff:     "1.2.3.4",
			want:    "203.0.113.7",
		},
		{
			name:    "ipv6 trusted peer",
			proxies: mustProxies(t, "::1/128"),
			remote:  "[::1]:5000",
			xff:     "2001:db8::1",
			want:    "2001:db8::1",
		},
		{
			name:    "single-host cidr from bare ip",
			proxies: mustProxies(t, "127.0.0.1"),
			remote:  "127.0.0.1:9000",
			xff:     "8.8.8.8",
			want:    "8.8.8.8",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.proxies.ClientIP(tc.remote, tc.xff); got != tc.want {
				t.Fatalf("ClientIP(%q, %q) = %q, want %q", tc.remote, tc.xff, got, tc.want)
			}
		})
	}
}

func TestNilTrustedProxies(t *testing.T) {
	var tp *TrustedProxies
	if got := tp.ClientIP("203.0.113.7:1234", "1.2.3.4"); got != "203.0.113.7" {
		t.Fatalf("nil ClientIP = %q, want peer", got)
	}
	if tp.Trusts("10.0.0.1") {
		t.Fatal("nil TrustedProxies must trust nothing")
	}
}

func TestParseTrustedProxiesInvalid(t *testing.T) {
	if _, err := ParseTrustedProxies([]string{"not-an-ip"}); err == nil {
		t.Fatal("expected error for invalid CIDR")
	}
}
