// Package config resolves the moth server configuration.
//
// Precedence, highest first: command-line flags > MOTH_* environment
// variables > config file (moth.toml, optional) > built-in defaults.
package config

import (
	"fmt"
	"net/url"
	"os"

	"github.com/BurntSushi/toml"
)

// Defaults.
const (
	DefaultAddr    = ":8080"
	DefaultDataDir = "./data"
	DefaultBaseURL = "http://localhost:8080"

	// DefaultFile is the config file loaded when it exists and no explicit
	// path was given.
	DefaultFile = "moth.toml"
)

// Config is the fully resolved server configuration.
type Config struct {
	// Addr is the listen address, e.g. ":8080".
	Addr string
	// DataDir holds the SQLite database, uploads and key material.
	DataDir string
	// BaseURL is the public URL of this instance, used to build absolute
	// links (setup URL, JWKS URLs) and to decide cookie security.
	BaseURL string
}

// Overrides carries values set explicitly on the command line. Nil fields
// were not set and fall through to the next precedence level.
type Overrides struct {
	Addr    *string
	DataDir *string
	BaseURL *string
	// File is an explicit config file path. When empty, DefaultFile is
	// loaded if it exists.
	File string
}

type fileConfig struct {
	Addr    *string `toml:"addr"`
	DataDir *string `toml:"data_dir"`
	BaseURL *string `toml:"base_url"`
}

// Load resolves the configuration. getenv is injectable for tests; pass
// os.Getenv in production.
func Load(o Overrides, getenv func(string) string) (Config, error) {
	cfg := Config{
		Addr:    DefaultAddr,
		DataDir: DefaultDataDir,
		BaseURL: DefaultBaseURL,
	}

	path := o.File
	if path == "" {
		if p := getenv("MOTH_CONFIG"); p != "" {
			path = p
		}
	}
	explicit := path != ""
	if path == "" {
		path = DefaultFile
	}
	var fc fileConfig
	if _, err := toml.DecodeFile(path, &fc); err != nil {
		if explicit || !os.IsNotExist(err) {
			return Config{}, fmt.Errorf("config file %s: %w", path, err)
		}
	}
	apply(&cfg, fc.Addr, fc.DataDir, fc.BaseURL)

	envOpt := func(key string) *string {
		if v := getenv(key); v != "" {
			return &v
		}
		return nil
	}
	apply(&cfg, envOpt("MOTH_ADDR"), envOpt("MOTH_DATA_DIR"), envOpt("MOTH_BASE_URL"))

	apply(&cfg, o.Addr, o.DataDir, o.BaseURL)

	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return Config{}, fmt.Errorf("base URL %q is not a valid absolute URL", cfg.BaseURL)
	}
	return cfg, nil
}

// Secure reports whether the instance is served over https, which controls
// the Secure attribute on session cookies.
func (c Config) Secure() bool {
	u, err := url.Parse(c.BaseURL)
	return err == nil && u.Scheme == "https"
}

// BaseOrigin returns the scheme://host part of the base URL, used as the
// allowed CORS origin for gRPC-Web calls.
func (c Config) BaseOrigin() string {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func apply(cfg *Config, addr, dataDir, baseURL *string) {
	if addr != nil {
		cfg.Addr = *addr
	}
	if dataDir != nil {
		cfg.DataDir = *dataDir
	}
	if baseURL != nil {
		cfg.BaseURL = *baseURL
	}
}
