package setup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Runner abstracts the local helper tools the setup flows shell out to
// (gcloud, keytool), so tests substitute canned output and CI never needs
// the tools installed.
type Runner interface {
	// LookPath reports where name resolves on PATH, or an error.
	LookPath(name string) (string, error)
	// Output runs the command with the extra "KEY=value" environment
	// variables (nil for none — secrets travel through the environment,
	// never the world-readable argv) and returns its combined output; a
	// non-zero exit is an error that carries the output for diagnostics.
	Output(ctx context.Context, env []string, name string, args ...string) ([]byte, error)
}

// ExecRunner runs the real tools.
type ExecRunner struct{}

// LookPath implements Runner via exec.LookPath.
func (ExecRunner) LookPath(name string) (string, error) { return exec.LookPath(name) }

// Output implements Runner via exec.CommandContext.
func (ExecRunner) Output(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s: %w: %s", name, err, out)
	}
	return out, nil
}
