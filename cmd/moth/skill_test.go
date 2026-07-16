package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/cli"
	"github.com/aloisdeniel/moth/internal/config"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/server"
	adminrpc "github.com/aloisdeniel/moth/internal/server/rpc/admin"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

// newSkillTestServer starts a full moth server whose public base URL is
// its real listen address, so both the admin RPCs and the plain-HTTP pub
// listing the export reads resolve against the same host. It returns the
// server and a usable PAT.
func newSkillTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()

	st, err := store.Open(filepath.Join(dir, "moth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	master, err := keys.LoadOrCreateMasterKey(dir, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewUnstartedServer(http.NotFoundHandler())
	baseURL := "http://" + ts.Listener.Addr().String()
	srv, err := server.New(server.Options{
		Config: config.Config{Addr: ":0", DataDir: dir, BaseURL: baseURL},
		Store:  st,
		Master: master,
		Logger: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	ts.Config.Handler = srv.Handler()
	ts.Start()
	t.Cleanup(ts.Close)

	// An admin with a PAT, created straight in the store — the Bearer RPC
	// path itself is covered by the admin PAT tests.
	now := time.Now()
	admin := store.Admin{ID: adminrpc.NewID(), Email: "op@example.com",
		PasswordHash: "unused", CreatedAt: now, UpdatedAt: now}
	if err := st.CreateAdmin(context.Background(), admin); err != nil {
		t.Fatal(err)
	}
	pat := token.New(token.PATPrefix)
	if err := st.CreatePAT(context.Background(), store.PersonalAccessToken{
		ID: adminrpc.NewID(), AdminID: admin.ID, Name: "skill-test",
		TokenHash: token.Hash(pat), CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	return ts, pat
}

// configureContext points the CLI config file (isolated via
// XDG_CONFIG_HOME) at the test server.
func configureContext(t *testing.T, url, pat string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := cli.ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	cfg := cli.Config{}
	cfg.SetContext("test", cli.Context{URL: url, Token: pat})
	if err := cli.SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
}

func runCLI(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	var out, errb bytes.Buffer
	root := newRootCmd()
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs(args)
	err = root.Execute()
	return out.String(), errb.String(), err
}

func readSkillFile(t *testing.T, dir, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(raw)
}

func TestSkillExportInterpolatesProjectValues(t *testing.T) {
	ts, pat := newSkillTestServer(t)
	configureContext(t, ts.URL, pat)
	ctx := context.Background()

	client := cli.New(ts.URL, pat)
	create, err := client.Projects.CreateProject(ctx,
		connect.NewRequest(&adminv1.CreateProjectRequest{Name: "Demo App"}))
	if err != nil {
		t.Fatal(err)
	}
	p := create.Msg.Project
	settings := p.Settings
	settings.Google = &adminv1.GoogleProviderConfig{
		Enabled:     true,
		WebClientId: "web-123.apps.googleusercontent.com",
		IosClientId: "ios-123.apps.googleusercontent.com",
	}
	if _, err := client.Projects.UpdateProject(ctx, connect.NewRequest(
		&adminv1.UpdateProjectRequest{Id: p.Id, Name: p.Name, Settings: settings})); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	stdout, stderr, err := runCLI(t,
		"skill", "export", "--project", p.Slug, "--dir", dir, "--context", "test")
	if err != nil {
		t.Fatalf("skill export: %v (stderr: %s)", err, stderr)
	}
	if !strings.Contains(stdout, "SKILL.md") {
		t.Errorf("stdout should list the written files, got:\n%s", stdout)
	}

	skillMD := readSkillFile(t, dir, "SKILL.md")
	for _, want := range []string{p.PublishableKey, p.Slug, ts.URL, "Sign in with Google | enabled"} {
		if !strings.Contains(skillMD, want) {
			t.Errorf("SKILL.md misses %q", want)
		}
	}
	jwks := ts.URL + "/p/" + p.Slug + "/.well-known/jwks.json"
	if backend := readSkillFile(t, dir, "references/backend-verification.md"); !strings.Contains(backend, jwks) {
		t.Errorf("backend-verification.md misses the real JWKS URL %s", jwks)
	}
	if providers := readSkillFile(t, dir, "references/provider-setup.md"); !strings.Contains(providers, "web-123.apps.googleusercontent.com") {
		t.Error("provider-setup.md misses the configured Google web client ID")
	}
	all := skillMD +
		readSkillFile(t, dir, "references/flutter-integration.md") +
		readSkillFile(t, dir, "references/backend-verification.md") +
		readSkillFile(t, dir, "references/provider-setup.md") +
		readSkillFile(t, dir, "references/cli-administration.md")
	for _, stale := range []string{"MOTH_BASE_URL", "pk_YOUR_PUBLISHABLE_KEY", "MOTH_SDK_VERSION"} {
		if strings.Contains(all, stale) {
			t.Errorf("interpolated export still contains placeholder %q", stale)
		}
	}
	// The pubspec constraint comes from the instance's own pub listing;
	// dev builds serve a pre-release, which must be pinned exactly.
	if flutter := readSkillFile(t, dir, "references/flutter-integration.md"); !strings.Contains(flutter, "version: 0.0.0-dev.") {
		t.Error("flutter-integration.md should pin the served pre-release SDK version exactly")
	}
}

func TestSkillExportWithoutProjectKeepsPlaceholders(t *testing.T) {
	// No context configured, no server running: the placeholder export
	// must work fully offline.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()

	stdout, stderr, err := runCLI(t, "skill", "export", "--dir", dir, "--json")
	if err != nil {
		t.Fatalf("skill export: %v (stderr: %s)", err, stderr)
	}
	var report struct {
		Dir          string   `json:"dir"`
		Files        []string `json:"files"`
		Interpolated bool     `json:"interpolated"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("--json output is not JSON: %v\n%s", err, stdout)
	}
	if report.Interpolated || report.Dir != dir || len(report.Files) == 0 {
		t.Errorf("unexpected --json report: %+v", report)
	}
	if !strings.Contains(readSkillFile(t, dir, "SKILL.md"), "pk_YOUR_PUBLISHABLE_KEY") {
		t.Error("un-interpolated SKILL.md misses the publishable-key placeholder")
	}
}

func TestSkillExportGenericFormat(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()

	if _, stderr, err := runCLI(t, "skill", "export", "--dir", dir, "--format", "generic"); err != nil {
		t.Fatalf("skill export: %v (stderr: %s)", err, stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); !os.IsNotExist(err) {
		t.Error("generic export must not write SKILL.md")
	}
	if doc := readSkillFile(t, dir, "README.md"); strings.HasPrefix(doc, "---") {
		t.Error("generic export must not emit frontmatter")
	}
}

// TestSkillCLISurfaceMatchesCommands guards the skill's CLI teaching
// against drift from the actual command tree: every `moth ...` invocation
// inside the exported skill's fenced code blocks must resolve to a real
// command, and every long flag shown must exist on that command. (The
// plan's release-time assembly from the milestone-09 docs tree replaces
// this eventually; until then a renamed command or flag fails here.)
func TestSkillCLISurfaceMatchesCommands(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	if _, stderr, err := runCLI(t, "skill", "export", "--dir", dir); err != nil {
		t.Fatalf("skill export: %v (stderr: %s)", err, stderr)
	}

	root := newRootCmd()
	checked := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		inFence := false
		for _, line := range strings.Split(string(raw), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "```") {
				inFence = !inFence
				continue
			}
			if !inFence || !strings.HasPrefix(trimmed, "moth ") {
				continue
			}
			if cut, _, ok := strings.Cut(trimmed, " #"); ok {
				trimmed = strings.TrimSpace(cut)
			}
			// "cmd-a / cmd-b" lists two invocations on one line.
			for _, invocation := range strings.Split(trimmed, " / ") {
				for _, tokens := range expandCommandAlternatives(strings.Fields(invocation)) {
					checkSkillInvocation(t, root, rel, tokens)
					checked++
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked < 10 {
		t.Fatalf("only %d moth invocations found in the skill — the scan is broken", checked)
	}
}

// expandCommandAlternatives expands "moth project dump|apply" style tokens
// into one concrete token list per alternative.
func expandCommandAlternatives(tokens []string) [][]string {
	out := [][]string{{}}
	for _, tok := range tokens {
		if strings.Contains(tok, "|") && !strings.HasPrefix(tok, "-") {
			var next [][]string
			for _, alt := range strings.Split(tok, "|") {
				for _, prefix := range out {
					expanded := append(slices.Clone(prefix), alt)
					next = append(next, expanded)
				}
			}
			out = next
			continue
		}
		for i := range out {
			out[i] = append(out[i], tok)
		}
	}
	return out
}

// checkSkillInvocation resolves one `moth ...` token list against the real
// command tree and verifies the long flags it shows exist there.
func checkSkillInvocation(t *testing.T, root *cobra.Command, file string, tokens []string) {
	t.Helper()
	line := strings.Join(tokens, " ")
	cur := root
	i := 1 // tokens[0] is "moth"
	for i < len(tokens) {
		var child *cobra.Command
		for _, c := range cur.Commands() {
			if c.Name() == tokens[i] {
				child = c
				break
			}
		}
		if child == nil {
			break
		}
		cur = child
		i++
	}
	if cur == root {
		t.Errorf("%s teaches %q, which is not a moth command", file, line)
		return
	}
	if !cur.Runnable() && i < len(tokens) && !strings.HasPrefix(tokens[i], "-") {
		t.Errorf("%s teaches %q: %q is a command group and %q is not one of its subcommands",
			file, line, cur.CommandPath(), tokens[i])
		return
	}
	for ; i < len(tokens); i++ {
		if !strings.HasPrefix(tokens[i], "--") {
			continue
		}
		name := strings.TrimPrefix(tokens[i], "--")
		name, _, _ = strings.Cut(name, "=")
		if cur.Flags().Lookup(name) == nil && cur.InheritedFlags().Lookup(name) == nil {
			t.Errorf("%s teaches %q, but %q has no --%s flag", file, line, cur.CommandPath(), name)
		}
	}
}

func TestSkillExportProjectRequiresContext(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	_, _, err := runCLI(t, "skill", "export", "--project", "demo", "--dir", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "no context configured") {
		t.Fatalf("want a no-context error, got %v", err)
	}
}
