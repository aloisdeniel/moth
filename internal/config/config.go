// Package config resolves the moth server configuration.
//
// Precedence, highest first: command-line flags > MOTH_* environment
// variables > config file (moth.toml, optional) > built-in defaults.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
)

// Defaults.
const (
	DefaultAddr     = ":8080"
	DefaultDataDir  = "./data"
	DefaultBaseURL  = "http://localhost:8080"
	DefaultSMTPPort = 587

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
	// SMTP configures outgoing email; when Host is empty, emails are
	// logged to the console instead (dev default).
	SMTP SMTP
}

// SMTP is the outgoing email relay configuration.
type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	// From is the sender address on every email.
	From string
}

// Enabled reports whether a real SMTP relay is configured.
func (s SMTP) Enabled() bool { return s.Host != "" }

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
	Addr    *string         `toml:"addr"`
	DataDir *string         `toml:"data_dir"`
	BaseURL *string         `toml:"base_url"`
	SMTP    *fileSMTPConfig `toml:"smtp"`
}

type fileSMTPConfig struct {
	Host     *string `toml:"host"`
	Port     *int    `toml:"port"`
	Username *string `toml:"username"`
	Password *string `toml:"password"`
	From     *string `toml:"from"`
}

// Load resolves the configuration. getenv is injectable for tests; pass
// os.Getenv in production.
func Load(o Overrides, getenv func(string) string) (Config, error) {
	cfg := Config{
		Addr:    DefaultAddr,
		DataDir: DefaultDataDir,
		BaseURL: DefaultBaseURL,
		SMTP:    SMTP{Port: DefaultSMTPPort},
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
	if fc.SMTP != nil {
		applySMTP(&cfg.SMTP, fc.SMTP.Host, fc.SMTP.Port, fc.SMTP.Username, fc.SMTP.Password, fc.SMTP.From)
	}

	envOpt := func(key string) *string {
		if v := getenv(key); v != "" {
			return &v
		}
		return nil
	}
	apply(&cfg, envOpt("MOTH_ADDR"), envOpt("MOTH_DATA_DIR"), envOpt("MOTH_BASE_URL"))
	var envPort *int
	if v := getenv("MOTH_SMTP_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p <= 0 || p > 65535 {
			return Config{}, fmt.Errorf("MOTH_SMTP_PORT %q is not a valid port", v)
		}
		envPort = &p
	}
	applySMTP(&cfg.SMTP, envOpt("MOTH_SMTP_HOST"), envPort,
		envOpt("MOTH_SMTP_USERNAME"), envOpt("MOTH_SMTP_PASSWORD"), envOpt("MOTH_SMTP_FROM"))

	apply(&cfg, o.Addr, o.DataDir, o.BaseURL)

	if cfg.SMTP.Enabled() && cfg.SMTP.From == "" {
		return Config{}, fmt.Errorf("smtp.from is required when an SMTP host is configured")
	}

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

func applySMTP(s *SMTP, host *string, port *int, username, password, from *string) {
	if host != nil {
		s.Host = *host
	}
	if port != nil {
		s.Port = *port
	}
	if username != nil {
		s.Username = *username
	}
	if password != nil {
		s.Password = *password
	}
	if from != nil {
		s.From = *from
	}
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
