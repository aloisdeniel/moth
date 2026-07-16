package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigPathHonorsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg/conf")
	path, err := ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("/xdg/conf", "moth", "config.toml"); path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}

	t.Setenv("XDG_CONFIG_HOME", "")
	path, err = ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	if want := filepath.Join(home, ".config", "moth", "config.toml"); path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.toml")

	// A missing file loads as an empty config.
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CurrentContext != "" || len(cfg.Contexts) != 0 {
		t.Fatalf("missing file should load empty, got %+v", cfg)
	}

	cfg.SetContext("prod", Context{URL: "https://auth.example.com", Token: "moth_pat_abc"})
	cfg.SetContext("dev", Context{URL: "http://localhost:8080", Token: "moth_pat_xyz"})
	if cfg.CurrentContext != "dev" {
		t.Fatalf("SetContext should update current-context, got %q", cfg.CurrentContext)
	}
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}

	// Credentials file must not be group/world readable.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("config perm = %o, want 600", perm)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CurrentContext != "dev" {
		t.Fatalf("current-context = %q, want dev", loaded.CurrentContext)
	}
	if got := loaded.Contexts["prod"]; got != (Context{URL: "https://auth.example.com", Token: "moth_pat_abc"}) {
		t.Fatalf("prod context = %+v", got)
	}
	if got := loaded.Contexts["dev"]; got != (Context{URL: "http://localhost:8080", Token: "moth_pat_xyz"}) {
		t.Fatalf("dev context = %+v", got)
	}
}

// SaveConfig must tighten a pre-existing loose-mode file: os.WriteFile
// only applies 0600 on create, and the file holds the plaintext PAT.
func TestSaveConfigTightensExistingPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("# seeded by a dotfiles manager\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var cfg Config
	cfg.SetContext("prod", Context{URL: "https://auth.example.com", Token: "moth_pat_abc"})
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("config perm = %o, want 600", perm)
	}
}

func TestConfigResolve(t *testing.T) {
	cfg := Config{
		CurrentContext: "dev",
		Contexts: map[string]Context{
			"dev":    {URL: "http://localhost:8080", Token: "moth_pat_xyz"},
			"prod":   {URL: "https://auth.example.com", Token: "moth_pat_abc"},
			"broken": {URL: "https://half.example.com"},
		},
	}

	cases := []struct {
		name     string
		override string
		wantName string
		wantErr  string
	}{
		{"current context by default", "", "dev", ""},
		{"explicit override", "prod", "prod", ""},
		{"unknown context", "staging", "", "not found"},
		{"incomplete context", "broken", "", "incomplete"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			name, ctx, err := cfg.Resolve(tc.override)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if name != tc.wantName || ctx != cfg.Contexts[tc.wantName] {
				t.Fatalf("resolved %q %+v, want %q", name, ctx, tc.wantName)
			}
		})
	}

	// A pristine config points the operator at moth login.
	if _, _, err := (Config{}).Resolve(""); err == nil || !strings.Contains(err.Error(), "moth login") {
		t.Fatalf("empty config: err = %v, want login hint", err)
	}
}
