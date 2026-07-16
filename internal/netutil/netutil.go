// Package netutil derives the real client IP behind an optional set of
// trusted reverse proxies.
//
// X-Forwarded-For is attacker-controlled: any client can send it. It may
// only be believed when the direct connection peer is itself a proxy we
// operate. ClientIP therefore honours XFF exclusively when the peer is in
// the configured trusted set, and otherwise returns the peer address. The
// same logic feeds both the rate-limit interceptor (per-IP buckets) and the
// plain-HTTP middleware, so a spoofed header can never dodge a limit.
package netutil

import (
	"net"
	"net/netip"
	"strings"
)

// TrustedProxies is the set of CIDR networks whose X-Forwarded-For headers
// are believed. The zero value (and a nil pointer) trusts no proxy, so
// ClientIP always returns the direct peer — the safe default for a
// directly-exposed server.
type TrustedProxies struct {
	nets []netip.Prefix
}

// ParseTrustedProxies builds a TrustedProxies from CIDR strings (e.g.
// "10.0.0.0/8", "127.0.0.1/32", "::1/128"). A bare IP is accepted and
// treated as a single-host network. An empty slice yields a set that trusts
// nothing.
func ParseTrustedProxies(cidrs []string) (*TrustedProxies, error) {
	t := &TrustedProxies{}
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if p, err := netip.ParsePrefix(c); err == nil {
			t.nets = append(t.nets, p.Masked())
			continue
		}
		addr, err := netip.ParseAddr(c)
		if err != nil {
			return nil, err
		}
		addr = addr.Unmap()
		t.nets = append(t.nets, netip.PrefixFrom(addr, addr.BitLen()))
	}
	return t, nil
}

// Trusts reports whether ip (an address string) is within the trusted set.
func (t *TrustedProxies) Trusts(ip string) bool {
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return false
	}
	return t.containsAddr(addr)
}

func (t *TrustedProxies) containsAddr(addr netip.Addr) bool {
	if t == nil {
		return false
	}
	addr = addr.Unmap()
	for _, n := range t.nets {
		if n.Contains(addr) {
			return true
		}
	}
	return false
}

// ClientIP returns the caller's IP for a request whose direct peer is
// remoteAddr ("host:port" or bare host) and whose X-Forwarded-For header is
// xff (possibly empty, possibly a comma-separated chain).
//
// When the peer is not a trusted proxy, xff is ignored entirely and the peer
// host is returned. When the peer is trusted, the chain is walked from the
// right (nearest hop first) and the first address that is not itself a
// trusted proxy is returned — the real client just before it entered our
// proxy tier. If every hop is trusted, the left-most parseable entry wins;
// if none parses, the peer host is returned.
func (t *TrustedProxies) ClientIP(remoteAddr, xff string) string {
	peer := hostOnly(remoteAddr)
	peerAddr, err := netip.ParseAddr(peer)
	if err != nil || t == nil || len(t.nets) == 0 || !t.containsAddr(peerAddr) {
		return peer
	}
	parts := strings.Split(xff, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		if addr, ok := parseHop(parts[i]); ok && !t.containsAddr(addr) {
			return addr.String()
		}
	}
	for _, p := range parts {
		if addr, ok := parseHop(p); ok {
			return addr.String()
		}
	}
	return peer
}

// parseHop parses one X-Forwarded-For element, tolerating surrounding
// whitespace and an accidental :port suffix.
func parseHop(s string) (netip.Addr, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return netip.Addr{}, false
	}
	if addr, err := netip.ParseAddr(s); err == nil {
		return addr.Unmap(), true
	}
	if h, _, err := net.SplitHostPort(s); err == nil {
		if addr, err := netip.ParseAddr(h); err == nil {
			return addr.Unmap(), true
		}
	}
	return netip.Addr{}, false
}

// hostOnly strips a :port from remoteAddr when present, returning the bare
// host. It also unwraps a "[::1]" style bracketed literal.
func hostOnly(remoteAddr string) string {
	remoteAddr = strings.TrimSpace(remoteAddr)
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return h
	}
	return strings.Trim(remoteAddr, "[]")
}
