package acme

import (
	"path/filepath"
	"testing"

	"golang.org/x/crypto/acme/autocert"
)

func TestManager(t *testing.T) {
	dir := t.TempDir()
	m, err := Manager(dir, "auth.example.com")
	if err != nil {
		t.Fatalf("Manager: %v", err)
	}
	if _, ok := m.Cache.(autocert.DirCache); !ok {
		t.Fatalf("cache type = %T, want DirCache", m.Cache)
	}
	if dc, _ := m.Cache.(autocert.DirCache); string(dc) != filepath.Join(dir, CacheDirName) {
		t.Fatalf("cache dir = %q, want under %q", string(dc), dir)
	}
	if m.HostPolicy == nil {
		t.Fatal("HostPolicy must pin the whitelist")
	}
	// Whitelisted host passes; anything else is refused.
	if err := m.HostPolicy(t.Context(), "auth.example.com"); err != nil {
		t.Fatalf("whitelisted host rejected: %v", err)
	}
	if err := m.HostPolicy(t.Context(), "evil.example.com"); err == nil {
		t.Fatal("non-whitelisted host must be rejected")
	}
}

func TestTLSConfig(t *testing.T) {
	m, err := Manager(t.TempDir(), "auth.example.com")
	if err != nil {
		t.Fatal(err)
	}
	cfg := TLSConfig(m)
	if cfg.GetCertificate == nil {
		t.Fatal("TLS config must fetch certificates on demand")
	}
	var hasACMEALPN bool
	for _, p := range cfg.NextProtos {
		if p == "acme-tls/1" {
			hasACMEALPN = true
		}
	}
	if !hasACMEALPN {
		t.Fatalf("TLS config must advertise acme-tls/1, got %v", cfg.NextProtos)
	}
}

func TestManagerRequiresDomain(t *testing.T) {
	if _, err := Manager(t.TempDir()); err == nil {
		t.Fatal("expected error with no domains")
	}
	if _, err := Manager(t.TempDir(), ""); err == nil {
		t.Fatal("expected error with empty domain")
	}
}
