package config

import (
	"os"
	"path/filepath"
	"testing"
)

func ptr(s string) *string { return &s }

func noEnv(string) string { return "" }

func inDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
	return dir
}

func TestDefaults(t *testing.T) {
	inDir(t)
	cfg, err := Load(Overrides{}, noEnv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != DefaultAddr || cfg.DataDir != DefaultDataDir || cfg.BaseURL != DefaultBaseURL {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestPrecedenceFlagOverEnvOverFileOverDefault(t *testing.T) {
	dir := inDir(t)
	file := filepath.Join(dir, "moth.toml")
	err := os.WriteFile(file, []byte(
		"addr = \":1111\"\ndata_dir = \"/from-file\"\nbase_url = \"http://file.example\"\n"), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]string{
		"MOTH_ADDR":     ":2222",
		"MOTH_DATA_DIR": "/from-env",
	}
	getenv := func(k string) string { return env[k] }

	// Flag beats env beats file beats default.
	cfg, err := Load(Overrides{Addr: ptr(":3333")}, getenv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":3333" {
		t.Errorf("addr: flag should win, got %q", cfg.Addr)
	}
	if cfg.DataDir != "/from-env" {
		t.Errorf("data dir: env should win over file, got %q", cfg.DataDir)
	}
	if cfg.BaseURL != "http://file.example" {
		t.Errorf("base url: file should win over default, got %q", cfg.BaseURL)
	}
}

func TestExplicitConfigFileMustExist(t *testing.T) {
	inDir(t)
	if _, err := Load(Overrides{File: "nope.toml"}, noEnv); err == nil {
		t.Fatal("expected error for missing explicit config file")
	}
	// A missing default file is fine.
	if _, err := Load(Overrides{}, noEnv); err != nil {
		t.Fatalf("missing default file should not error: %v", err)
	}
}

func TestConfigFileFromEnv(t *testing.T) {
	dir := inDir(t)
	custom := filepath.Join(dir, "custom.toml")
	if err := os.WriteFile(custom, []byte("addr = \":9999\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(Overrides{}, func(k string) string {
		if k == "MOTH_CONFIG" {
			return custom
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":9999" {
		t.Fatalf("addr from MOTH_CONFIG file, got %q", cfg.Addr)
	}
}

func TestInvalidBaseURL(t *testing.T) {
	inDir(t)
	if _, err := Load(Overrides{BaseURL: ptr("not-a-url")}, noEnv); err == nil {
		t.Fatal("expected error for invalid base URL")
	}
}

func TestSecureAndOrigin(t *testing.T) {
	c := Config{BaseURL: "https://auth.example.com/x"}
	if !c.Secure() {
		t.Error("https base URL should be secure")
	}
	if got := c.BaseOrigin(); got != "https://auth.example.com" {
		t.Errorf("origin: got %q", got)
	}
	c = Config{BaseURL: "http://localhost:8080"}
	if c.Secure() {
		t.Error("http base URL should not be secure")
	}
}
