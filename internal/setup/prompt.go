package setup

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// promptAttempts bounds how often a validation failure re-asks before the
// command gives up (protects non-interactive runs fed a bad answer file).
const promptAttempts = 3

// Prompter reads answers for the guided flows. In/out are plain streams so
// tests feed scripted input and assert the transcript.
type Prompter struct {
	raw io.Reader // the unwrapped input, for the terminal check in AskSecret
	in  *bufio.Reader
	out io.Writer
}

// NewPrompter wraps in/out for the guided flows.
func NewPrompter(in io.Reader, out io.Writer) *Prompter {
	return &Prompter{raw: in, in: bufio.NewReader(in), out: out}
}

func (p *Prompter) readLine() (string, error) {
	line, err := p.in.ReadString('\n')
	if err != nil && (line == "" || !errors.Is(err, io.EOF)) {
		return "", fmt.Errorf("read input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// Ask prints the label and reads a line until validate accepts it (which
// may normalize the value) or the attempts run out. An empty validate
// accepts anything, including "".
func (p *Prompter) Ask(label string, validate func(string) (string, error)) (string, error) {
	for attempt := 1; ; attempt++ {
		_, _ = fmt.Fprintf(p.out, "%s: ", label)
		line, err := p.readLine()
		if err != nil {
			return "", err
		}
		if validate == nil {
			return line, nil
		}
		value, err := validate(line)
		if err == nil {
			return value, nil
		}
		if attempt >= promptAttempts {
			return "", fmt.Errorf("giving up after %d attempts: %w", promptAttempts, err)
		}
		_, _ = fmt.Fprintf(p.out, "  %v — try again\n", err)
	}
}

// AskSecret prints the label and reads one secret line. When the input is
// a terminal the echo is disabled, so the value never lands in scrollback
// or session recordings; piped/scripted input falls back to a plain line
// read (nothing echoes there anyway).
func (p *Prompter) AskSecret(label string) (string, error) {
	_, _ = fmt.Fprintf(p.out, "%s: ", label)
	if f, ok := p.raw.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		raw, err := term.ReadPassword(int(f.Fd()))
		_, _ = fmt.Fprintln(p.out)
		if err != nil {
			return "", fmt.Errorf("read input: %w", err)
		}
		return strings.TrimSpace(string(raw)), nil
	}
	return p.readLine()
}

// Confirm asks a yes/no question; empty input means the given default.
func (p *Prompter) Confirm(label string, def bool) (bool, error) {
	hint := "y/N"
	if def {
		hint = "Y/n"
	}
	answer, err := p.Ask(fmt.Sprintf("%s [%s]", label, hint), func(s string) (string, error) {
		switch strings.ToLower(s) {
		case "", "y", "yes", "n", "no":
			return strings.ToLower(s), nil
		}
		return "", errors.New(`answer "y" or "n"`)
	})
	if err != nil {
		return false, err
	}
	if answer == "" {
		return def, nil
	}
	return answer == "y" || answer == "yes", nil
}

// Say writes one line of guidance to the prompt transcript.
func (p *Prompter) Say(format string, args ...any) {
	_, _ = fmt.Fprintf(p.out, format+"\n", args...)
}
