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
	"strings"
	"time"

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

	// DefaultLogFormat is the slog handler used when none is configured.
	DefaultLogFormat = "text"
	// DefaultBackupInterval is how often scheduled backups run when a backup
	// directory is configured but no interval is given.
	DefaultBackupInterval = 24 * time.Hour

	// Default per-minute rate-limit tiers for the credential-facing surfaces.
	DefaultRateLimitIPPerMinute      = 60
	DefaultRateLimitAccountPerMinute = 10
	DefaultRateLimitProjectPerMinute = 600
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
	// LogFormat selects the slog handler: "text" (default) or "json" for
	// structured logs suitable for log aggregation.
	LogFormat string
	// Reflection enables gRPC server reflection in release builds. Dev
	// builds always enable it; this flag turns it on in production.
	Reflection bool
	// BackupDir, when set, enables scheduled automatic backups written to
	// this directory while serving. Empty disables the scheduler.
	BackupDir string
	// BackupInterval is the period between scheduled backups. Zero falls
	// back to DefaultBackupInterval when BackupDir is set.
	BackupInterval time.Duration
	// TrustedProxies are the CIDR networks (or bare IPs) whose
	// X-Forwarded-For headers are believed when deriving the client IP for
	// rate limiting behind a reverse proxy. Empty trusts no proxy.
	TrustedProxies []string
	// RateLimit configures the credential-facing rate-limit tiers.
	RateLimit RateLimit
	// AcmeDomains, when non-empty, enables the built-in ACME/Let's Encrypt
	// client: the server obtains and renews a certificate for these hostnames
	// and serves HTTPS on :443, answering http-01 challenges (and redirecting
	// plain HTTP) on :80. Empty disables it (plain HTTP on Addr).
	AcmeDomains []string
}

// RateLimit holds the per-minute limits of the three rate-limit tiers. A
// non-positive value disables that tier.
type RateLimit struct {
	IPPerMinute      int
	AccountPerMinute int
	ProjectPerMinute int
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
	Addr           *string
	DataDir        *string
	BaseURL        *string
	LogFormat      *string
	Reflection     *bool
	BackupDir      *string
	BackupInterval *time.Duration
	TrustedProxies *[]string
	AcmeDomains    *[]string
	// File is an explicit config file path. When empty, DefaultFile is
	// loaded if it exists.
	File string
}

type fileConfig struct {
	Addr           *string         `toml:"addr"`
	DataDir        *string         `toml:"data_dir"`
	BaseURL        *string         `toml:"base_url"`
	LogFormat      *string         `toml:"log_format"`
	Reflection     *bool           `toml:"reflection"`
	BackupDir      *string         `toml:"backup_dir"`
	BackupInterval *string         `toml:"backup_interval"`
	TrustedProxies []string        `toml:"trusted_proxies"`
	AcmeDomains    []string        `toml:"acme_domains"`
	RateLimit      *fileRateLimit  `toml:"ratelimit"`
	SMTP           *fileSMTPConfig `toml:"smtp"`
}

type fileRateLimit struct {
	IPPerMinute      *int `toml:"ip_per_minute"`
	AccountPerMinute *int `toml:"account_per_minute"`
	ProjectPerMinute *int `toml:"project_per_minute"`
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
		Addr:      DefaultAddr,
		DataDir:   DefaultDataDir,
		BaseURL:   DefaultBaseURL,
		LogFormat: DefaultLogFormat,
		SMTP:      SMTP{Port: DefaultSMTPPort},
		RateLimit: RateLimit{
			IPPerMinute:      DefaultRateLimitIPPerMinute,
			AccountPerMinute: DefaultRateLimitAccountPerMinute,
			ProjectPerMinute: DefaultRateLimitProjectPerMinute,
		},
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
	var fileInterval *time.Duration
	if fc.BackupInterval != nil {
		d, err := time.ParseDuration(*fc.BackupInterval)
		if err != nil {
			return Config{}, fmt.Errorf("backup_interval %q is not a valid duration", *fc.BackupInterval)
		}
		fileInterval = &d
	}
	applyOps(&cfg, fc.LogFormat, fc.Reflection, fc.BackupDir, fileInterval)
	if fc.TrustedProxies != nil {
		cfg.TrustedProxies = fc.TrustedProxies
	}
	if fc.AcmeDomains != nil {
		cfg.AcmeDomains = fc.AcmeDomains
	}
	if fc.RateLimit != nil {
		applyRateLimit(&cfg.RateLimit, fc.RateLimit.IPPerMinute, fc.RateLimit.AccountPerMinute, fc.RateLimit.ProjectPerMinute)
	}
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

	var envReflection *bool
	if v := getenv("MOTH_REFLECTION"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("MOTH_REFLECTION %q is not a valid boolean", v)
		}
		envReflection = &b
	}
	var envInterval *time.Duration
	if v := getenv("MOTH_BACKUP_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("MOTH_BACKUP_INTERVAL %q is not a valid duration", v)
		}
		envInterval = &d
	}
	applyOps(&cfg, envOpt("MOTH_LOG_FORMAT"), envReflection, envOpt("MOTH_BACKUP_DIR"), envInterval)

	if v := getenv("MOTH_TRUSTED_PROXIES"); v != "" {
		cfg.TrustedProxies = splitList(v)
	}
	if v := getenv("MOTH_ACME_DOMAINS"); v != "" {
		cfg.AcmeDomains = splitList(v)
	}
	envIP, err := envInt(getenv, "MOTH_RATELIMIT_IP_PER_MINUTE")
	if err != nil {
		return Config{}, err
	}
	envAcct, err := envInt(getenv, "MOTH_RATELIMIT_ACCOUNT_PER_MINUTE")
	if err != nil {
		return Config{}, err
	}
	envProj, err := envInt(getenv, "MOTH_RATELIMIT_PROJECT_PER_MINUTE")
	if err != nil {
		return Config{}, err
	}
	applyRateLimit(&cfg.RateLimit, envIP, envAcct, envProj)

	apply(&cfg, o.Addr, o.DataDir, o.BaseURL)
	applyOps(&cfg, o.LogFormat, o.Reflection, o.BackupDir, o.BackupInterval)
	if o.TrustedProxies != nil {
		cfg.TrustedProxies = *o.TrustedProxies
	}
	if o.AcmeDomains != nil {
		cfg.AcmeDomains = *o.AcmeDomains
	}

	if cfg.LogFormat != "text" && cfg.LogFormat != "json" {
		return Config{}, fmt.Errorf("log format %q is not one of text, json", cfg.LogFormat)
	}
	if cfg.BackupDir != "" && cfg.BackupInterval <= 0 {
		cfg.BackupInterval = DefaultBackupInterval
	}

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

func applyOps(cfg *Config, logFormat *string, reflection *bool, backupDir *string, backupInterval *time.Duration) {
	if logFormat != nil {
		cfg.LogFormat = *logFormat
	}
	if reflection != nil {
		cfg.Reflection = *reflection
	}
	if backupDir != nil {
		cfg.BackupDir = *backupDir
	}
	if backupInterval != nil {
		cfg.BackupInterval = *backupInterval
	}
}

func applyRateLimit(rl *RateLimit, ip, account, project *int) {
	if ip != nil {
		rl.IPPerMinute = *ip
	}
	if account != nil {
		rl.AccountPerMinute = *account
	}
	if project != nil {
		rl.ProjectPerMinute = *project
	}
}

// envInt reads an optional non-negative integer environment variable.
func envInt(getenv func(string) string, key string) (*int, error) {
	v := getenv(key)
	if v == "" {
		return nil, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return nil, fmt.Errorf("%s %q is not a valid non-negative integer", key, v)
	}
	return &n, nil
}

// splitList parses a comma-separated list, trimming spaces and dropping empty
// entries.
func splitList(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
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
