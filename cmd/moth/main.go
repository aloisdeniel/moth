// Command moth is the single-binary auth server for mobile apps.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	// Embed the IANA timezone database: the analytics rollup timezone
	// (ProjectSettings.rollup_timezone) must resolve on hosts without
	// /usr/share/zoneinfo (scratch containers, Windows) — one binary,
	// everything embedded.
	_ "time/tzdata"

	"github.com/spf13/cobra"

	"github.com/aloisdeniel/moth/internal/config"
)

// rootFlags are shared by every subcommand that touches the instance.
type rootFlags struct {
	addr    string
	dataDir string
	baseURL string
	file    string
	// Operational flags registered only on 'serve' (see addServeOpsFlags).
	logFormat      string
	reflection     bool
	backupDir      string
	backupInterval time.Duration
	trustedProxies string
	acmeDomain     string
}

// newRootCmd assembles the full command tree. Docs generation and tests
// build the same tree, so the rendered CLI reference always matches the
// real binary.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "moth",
		Short:         "moth — authentication for your mobile apps in one binary",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newServeCmd(), newAdminCmd(), newVersionCmd())
	// Local data-directory operations.
	root.AddCommand(newBackupCmd(), newRestoreCmd())
	// Remote client mode: kubectl-style commands against a configured
	// context (see 'moth login').
	root.AddCommand(newLoginCmd(), newProjectCmd(), newUserCmd(),
		newStatsCmd(), newInstanceCmd(), newTokenCmd())
	// Provider-console orchestration + health checks (milestone 08).
	root.AddCommand(newSetupCmd(), newDoctorCmd())
	// Agent-facing artifacts: the exported skill and the generated docs.
	root.AddCommand(newSkillCmd(), newDocsCmd())
	return root
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(exitCode(err))
	}
}

// addConfigFlags registers the shared config flags on cmd.
func addConfigFlags(cmd *cobra.Command, f *rootFlags) {
	cmd.Flags().StringVar(&f.addr, "addr", config.DefaultAddr, "listen address")
	cmd.Flags().StringVar(&f.dataDir, "data-dir", config.DefaultDataDir, "data directory (database, keys, uploads)")
	cmd.Flags().StringVar(&f.baseURL, "base-url", config.DefaultBaseURL, "public base URL of this instance")
	cmd.Flags().StringVar(&f.file, "config", "", "config file (default "+config.DefaultFile+" if present)")
}

// addServeOpsFlags registers the operational flags that only apply to a
// running server, so they don't clutter every subcommand's help.
func addServeOpsFlags(cmd *cobra.Command, f *rootFlags) {
	cmd.Flags().StringVar(&f.logFormat, "log-format", config.DefaultLogFormat, "log handler: text or json")
	cmd.Flags().BoolVar(&f.reflection, "reflection", false, "enable gRPC server reflection in release builds")
	cmd.Flags().StringVar(&f.backupDir, "backup-dir", "", "directory for scheduled automatic backups (empty disables)")
	cmd.Flags().DurationVar(&f.backupInterval, "backup-interval", config.DefaultBackupInterval, "interval between scheduled backups")
	cmd.Flags().StringVar(&f.trustedProxies, "trusted-proxies", "", "comma-separated CIDRs/IPs whose X-Forwarded-For is trusted for client-IP rate limiting")
	cmd.Flags().StringVar(&f.acmeDomain, "acme-domain", "", "comma-separated hostname(s) to obtain a Let's Encrypt certificate for; enables built-in HTTPS on :443 and http-01 on :80")
}

// splitCommaList parses a comma-separated flag value, trimming spaces and
// dropping empty entries.
func splitCommaList(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// resolveConfig applies the flag > env > file > default precedence; only
// flags the user actually set on the command line override lower levels.
func resolveConfig(cmd *cobra.Command, f *rootFlags) (config.Config, error) {
	o := config.Overrides{File: f.file}
	if cmd.Flags().Changed("addr") {
		o.Addr = &f.addr
	}
	if cmd.Flags().Changed("data-dir") {
		o.DataDir = &f.dataDir
	}
	if cmd.Flags().Changed("base-url") {
		o.BaseURL = &f.baseURL
	}
	if cmd.Flags().Changed("log-format") {
		o.LogFormat = &f.logFormat
	}
	if cmd.Flags().Changed("reflection") {
		o.Reflection = &f.reflection
	}
	if cmd.Flags().Changed("backup-dir") {
		o.BackupDir = &f.backupDir
	}
	if cmd.Flags().Changed("backup-interval") {
		o.BackupInterval = &f.backupInterval
	}
	if cmd.Flags().Changed("trusted-proxies") {
		list := splitCommaList(f.trustedProxies)
		o.TrustedProxies = &list
	}
	if cmd.Flags().Changed("acme-domain") {
		list := splitCommaList(f.acmeDomain)
		o.AcmeDomains = &list
	}
	return config.Load(o, os.Getenv)
}
