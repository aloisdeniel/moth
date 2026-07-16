package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/aloisdeniel/moth/internal/cli"
)

// clientOpts are the flags shared by every remote (client-mode) command
// group: which context to talk to and whether to emit machine JSON.
type clientOpts struct {
	context string
	json    bool
}

// addClientFlags registers --context and --json as persistent flags on a
// client command group.
func addClientFlags(cmd *cobra.Command, o *clientOpts) {
	cmd.PersistentFlags().StringVar(&o.context, "context", "",
		"named context from the CLI config (default: MOTH_CONTEXT, then current-context)")
	cmd.PersistentFlags().BoolVar(&o.json, "json", false, "print machine-readable JSON")
}

// resolveContext resolves the CLI context (--context > MOTH_CONTEXT >
// current-context) from the config file. Commands that only need RPC
// clients use dial; setup/doctor/skill also need the context URL itself
// (redirect/return/pub URLs), so the context is the shared primitive.
func (o *clientOpts) resolveContext() (cli.Context, error) {
	path, err := cli.ConfigPath()
	if err != nil {
		return cli.Context{}, err
	}
	cfg, err := cli.LoadConfig(path)
	if err != nil {
		return cli.Context{}, err
	}
	name := o.context
	if name == "" {
		name = os.Getenv("MOTH_CONTEXT")
	}
	_, ctx, err := cfg.Resolve(name)
	return ctx, err
}

// dial resolves the context and returns authenticated admin clients for it.
func (o *clientOpts) dial() (*cli.Client, error) {
	ctx, err := o.resolveContext()
	if err != nil {
		return nil, err
	}
	return cli.New(ctx.URL, ctx.Token), nil
}

// dialURL resolves the context and returns clients plus the server base
// URL, which the setup and doctor flows need to build the redirect/return
// URLs they register and probe.
func (o *clientOpts) dialURL() (*cli.Client, string, error) {
	ctx, err := o.resolveContext()
	if err != nil {
		return nil, "", err
	}
	return cli.New(ctx.URL, ctx.Token), ctx.URL, nil
}

// exitCode maps an error to the process exit code, so scripts can branch
// without parsing stderr: 2 bad input / already exists, 3 authentication,
// 4 not found, 5 server unreachable, 1 anything else.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ce *connect.Error
	if errors.As(err, &ce) {
		switch ce.Code() {
		case connect.CodeInvalidArgument, connect.CodeAlreadyExists:
			return 2
		case connect.CodeUnauthenticated, connect.CodePermissionDenied:
			return 3
		case connect.CodeNotFound:
			return 4
		case connect.CodeUnavailable:
			return 5
		}
	}
	return 1
}

// printJSON writes stable JSON for a proto message to stdout.
func printJSON(cmd *cobra.Command, msg proto.Message) error {
	data, err := cli.MarshalJSON(msg)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(data)
	return err
}

// table renders rows with aligned columns to stdout.
func table(cmd *cobra.Command, header []string, rows [][]string) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, strings.Join(header, "\t")); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintln(w, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return w.Flush()
}

// confirm gates a destructive operation: --yes skips the prompt, otherwise
// an interactive "yes" is required (and a non-TTY refuses outright).
func confirm(cmd *cobra.Command, yes bool, action string) error {
	if yes {
		return nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("%s needs confirmation; re-run with --yes", action)
	}
	fmt.Fprintf(os.Stderr, "%s — type 'yes' to confirm: ", action)
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(line) != "yes" {
		return errors.New("aborted")
	}
	return nil
}

// jsonMarshalIndent renders a plain Go value with the same two-space
// indent as the proto JSON output.
func jsonMarshalIndent(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// fmtTime renders a proto timestamp for table output; "-" when unset.
func fmtTime(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return "-"
	}
	return ts.AsTime().Local().Format("2006-01-02 15:04")
}

// fmtBool renders a boolean as yes/no for table output.
func fmtBool(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// dateFlagDefaults returns the default --from/--to values: the last n days
// ending today (local time), as the "YYYY-MM-DD" strings GetStats expects.
func dateFlagDefaults(n int) (from, to string) {
	now := time.Now()
	return now.AddDate(0, 0, -(n - 1)).Format("2006-01-02"), now.Format("2006-01-02")
}
