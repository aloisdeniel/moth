// Package cli implements the moth binary's remote-client mode: named
// contexts (server URL + personal access token) stored in
// ~/.config/moth/config.toml, connect clients for the moth.admin.v1
// services authenticated by that token, and the declarative
// dump/apply logic built on ProjectSpec.
package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Context is one named server + credential pair.
type Context struct {
	// URL is the server base URL ("https://auth.example.com").
	URL string `toml:"url"`
	// Token is a personal access token (moth_pat_...).
	Token string `toml:"token"`
}

// Config is the on-disk CLI configuration, kubectl-style.
type Config struct {
	// CurrentContext is the context used when --context/MOTH_CONTEXT is
	// not given.
	CurrentContext string             `toml:"current-context,omitempty"`
	Contexts       map[string]Context `toml:"contexts,omitempty"`
}

// ConfigPath returns the CLI config file location:
// $XDG_CONFIG_HOME/moth/config.toml, falling back to
// ~/.config/moth/config.toml.
func ConfigPath() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "moth", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "moth", "config.toml"), nil
}

// LoadConfig reads the config file; a missing file is an empty config.
func LoadConfig(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

// SaveConfig writes the config file (0600 — it holds credentials),
// creating parent directories as needed.
func SaveConfig(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return err
	}
	// os.WriteFile only applies the mode when it creates the file; a
	// pre-existing config (seeded by a dotfiles manager, copied from another
	// machine) keeps its old — possibly world-readable — permissions, so
	// tighten explicitly: the file holds the plaintext PAT.
	return os.Chmod(path, 0o600)
}

// SetContext adds or replaces a named context and makes it current.
func (c *Config) SetContext(name string, ctx Context) {
	if c.Contexts == nil {
		c.Contexts = map[string]Context{}
	}
	c.Contexts[name] = ctx
	c.CurrentContext = name
}

// Resolve picks the context to use: the named one when name is not empty
// (--context / MOTH_CONTEXT), the current-context otherwise. The returned
// errors tell the operator how to fix an unconfigured CLI.
func (c Config) Resolve(name string) (string, Context, error) {
	if name == "" {
		name = c.CurrentContext
	}
	if name == "" {
		return "", Context{}, errors.New(
			"no context configured: run 'moth login <url>' first, or select one with --context/MOTH_CONTEXT")
	}
	ctx, ok := c.Contexts[name]
	if !ok {
		return "", Context{}, fmt.Errorf(
			"context %q not found: run 'moth login <url> --name %s' to create it", name, name)
	}
	if ctx.URL == "" || ctx.Token == "" {
		return "", Context{}, fmt.Errorf(
			"context %q is incomplete (missing url or token): re-run 'moth login <url> --name %s'", name, name)
	}
	return name, ctx, nil
}
