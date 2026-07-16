// Command moth is the single-binary auth server for mobile apps.
package main

import (
	"fmt"
	"os"

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
	return config.Load(o, os.Getenv)
}
