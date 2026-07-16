package main

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/cli"
	"github.com/aloisdeniel/moth/internal/token"
)

// loginResult is the --json output of `moth login`.
type loginResult struct {
	Context       string `json:"context"`
	ConfigPath    string `json:"config_path"`
	URL           string `json:"url"`
	Admin         string `json:"admin"`
	ServerVersion string `json:"server_version"`
}

func newLoginCmd() *cobra.Command {
	var name string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "login <url>",
		Short: "Authenticate against a moth server and save it as a context",
		Long: `Login prompts for a personal access token (create one in the admin
console under Account, or with 'moth admin token create' on the server
host), validates it against the server and saves the pair as a named
context in the CLI config file. The new context becomes the current one.

When stdin is not a terminal the token is read from it directly, so
scripts can pipe it in: echo "$MOTH_PAT" | moth login https://... --name ci`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			base, err := normalizeServerURL(args[0])
			if err != nil {
				return err
			}
			pat, err := readToken(cmd)
			if err != nil {
				return err
			}
			if !strings.HasPrefix(pat, token.PATPrefix) {
				return fmt.Errorf("that does not look like a personal access token (expected a %s... value)", token.PATPrefix)
			}

			client := cli.New(base, pat)
			who, err := client.Sessions.GetCurrentAdmin(cmd.Context(),
				connect.NewRequest(&adminv1.GetCurrentAdminRequest{}))
			if err != nil {
				return fmt.Errorf("token rejected by %s: %w", base, err)
			}

			if name == "" {
				name = defaultContextName(base)
			}
			path, err := cli.ConfigPath()
			if err != nil {
				return err
			}
			cfg, err := cli.LoadConfig(path)
			if err != nil {
				return err
			}
			cfg.SetContext(name, cli.Context{URL: base, Token: pat})
			if err := cli.SaveConfig(path, cfg); err != nil {
				return err
			}
			if asJSON {
				return printJSONValue(cmd, loginResult{
					Context:       name,
					ConfigPath:    path,
					URL:           base,
					Admin:         who.Msg.GetAdmin().GetEmail(),
					ServerVersion: who.Msg.GetServerVersion(),
				})
			}
			fmt.Printf("context %q saved to %s\nlogged in to %s as %s (server %s)\n",
				name, path, base, who.Msg.GetAdmin().GetEmail(), who.Msg.GetServerVersion())
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "context name (default: the server host)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print machine-readable JSON")
	return cmd
}

// readToken reads the PAT: without echo on a terminal, as a plain line on
// piped stdin (the scripted path).
func readToken(cmd *cobra.Command) (string, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		fmt.Fprint(os.Stderr, "Personal access token: ")
		raw, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(raw)), nil
	}
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("read token from stdin: %w", err)
	}
	pat := strings.TrimSpace(line)
	if pat == "" {
		return "", errors.New("empty token")
	}
	return pat, nil
}

// normalizeServerURL validates the server URL, defaulting the scheme to
// https and stripping any path/trailing slash.
func normalizeServerURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("invalid server URL %q", raw)
	}
	return u.Scheme + "://" + u.Host, nil
}

// defaultContextName derives a context name from the server URL host.
func defaultContextName(base string) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	return u.Host
}
