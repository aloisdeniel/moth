// Package acme wires golang.org/x/crypto/acme/autocert so a bare VPS can get
// a Let's Encrypt certificate straight from the single moth binary via
// --acme-domain, with no reverse proxy in front.
//
// The serve loop uses it in three pieces: Manager builds the autocert.Manager
// (certificates cached on disk under the data dir), TLSConfig turns it into a
// *tls.Config for the HTTPS listener, and the manager's own HTTPHandler must
// wrap the plain-HTTP listener on :80 to answer the ACME http-01 challenge
// and redirect everything else to https.
package acme

import (
	"crypto/tls"
	"fmt"
	"path/filepath"

	"golang.org/x/crypto/acme/autocert"
)

// CacheDirName is the subdirectory of the data dir where issued certificates
// and account keys are cached, so they survive restarts and stay off the
// public surface.
const CacheDirName = "acme"

// Manager builds an autocert.Manager that will obtain and renew certificates
// for the given domains, accepting the Let's Encrypt terms and caching state
// under dataDir/acme. At least one domain is required. The HostPolicy is
// pinned to exactly these domains, so the manager never requests a
// certificate for an attacker-supplied SNI.
func Manager(dataDir string, domains ...string) (*autocert.Manager, error) {
	if len(domains) == 0 {
		return nil, fmt.Errorf("acme: at least one domain is required")
	}
	for _, d := range domains {
		if d == "" {
			return nil, fmt.Errorf("acme: empty domain in %v", domains)
		}
	}
	return &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache(filepath.Join(dataDir, CacheDirName)),
		HostPolicy: autocert.HostWhitelist(domains...),
	}, nil
}

// TLSConfig returns the *tls.Config for the HTTPS server: it fetches
// certificates from the manager on demand and advertises the ACME TLS-ALPN-01
// protocol so challenges can also be answered on the TLS listener.
func TLSConfig(m *autocert.Manager) *tls.Config {
	return m.TLSConfig()
}
