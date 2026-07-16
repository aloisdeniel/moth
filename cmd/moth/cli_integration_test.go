package main

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCLIAgainstRealServer builds the moth binary, boots a real instance,
// mints a PAT with the local admin commands and drives the remote client
// commands against it — including the milestone acceptance criterion that
// a second 'moth project apply -f' of the same spec reports zero changes.
func TestCLIAgainstRealServer(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test; skipped with -short")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not available")
	}

	bin := filepath.Join(t.TempDir(), "moth")
	if out, err := exec.Command(goBin, "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build moth: %v\n%s", err, out)
	}

	dataDir := filepath.Join(t.TempDir(), "data")
	xdgDir := filepath.Join(t.TempDir(), "xdg")
	workDir := t.TempDir() // no moth.toml here
	env := append(os.Environ(),
		"XDG_CONFIG_HOME="+xdgDir,
		"MOTH_CONTEXT=",
	)

	run := func(stdin string, args ...string) (string, error) {
		cmd := exec.Command(bin, args...)
		cmd.Dir = workDir
		cmd.Env = env
		if stdin != "" {
			cmd.Stdin = strings.NewReader(stdin)
		}
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	mustRun := func(stdin string, args ...string) string {
		t.Helper()
		out, err := run(stdin, args...)
		if err != nil {
			t.Fatalf("moth %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return out
	}

	// Local bootstrap: admin account + PAT, no server needed. --json is the
	// scripting contract — no prose scraping.
	mustRun("", "admin", "create", "--email", "ops@example.com", "--password", "correct horse", "--data-dir", dataDir)
	var minted struct {
		Token    string
		Metadata struct{ Id string }
	}
	tokenOut := mustRun("", "admin", "token", "create", "--email", "ops@example.com", "--name", "it", "--data-dir", dataDir, "--json")
	if err := json.Unmarshal([]byte(tokenOut), &minted); err != nil {
		t.Fatalf("admin token create --json: %v\n%s", err, tokenOut)
	}
	pat := minted.Token
	if !strings.HasPrefix(pat, "moth_pat_") || minted.Metadata.Id == "" {
		t.Fatalf("unexpected admin token create --json output:\n%s", tokenOut)
	}
	listOut := mustRun("", "admin", "token", "list", "--email", "ops@example.com", "--data-dir", dataDir, "--json")
	var localList struct {
		Tokens []struct{ Id, Name string }
	}
	if err := json.Unmarshal([]byte(listOut), &localList); err != nil {
		t.Fatalf("admin token list --json: %v\n%s", err, listOut)
	}
	if len(localList.Tokens) != 1 || localList.Tokens[0].Id != minted.Metadata.Id {
		t.Fatalf("admin token list --json = %+v", localList)
	}

	// Boot the real server on a free port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	baseURL := "http://" + addr

	serve := exec.Command(bin, "serve", "--addr", addr, "--data-dir", dataDir, "--base-url", baseURL)
	serve.Dir = workDir
	serve.Env = env
	if err := serve.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = serve.Process.Kill()
		_ = serve.Wait()
	})
	waitForServer(t, baseURL+"/healthz")

	// moth login: scripted path, token piped on stdin.
	loginOut := mustRun(pat+"\n", "login", baseURL, "--name", "it")
	if !strings.Contains(loginOut, "ops@example.com") {
		t.Fatalf("login should report the admin identity:\n%s", loginOut)
	}
	if _, err := os.Stat(filepath.Join(xdgDir, "moth", "config.toml")); err != nil {
		t.Fatalf("login should write the XDG config file: %v", err)
	}

	// project create / list / get over the explicit context, JSON output.
	var created struct {
		Project struct {
			Id, Slug, Name string
		}
		SecretKey string
	}
	out := mustRun("", "--context", "it", "project", "create", "Demo", "--slug", "demo", "--json")
	if err := json.Unmarshal([]byte(out), &created); err != nil {
		t.Fatalf("create --json: %v\n%s", err, out)
	}
	if created.Project.Slug != "demo" {
		t.Fatalf("created slug = %q", created.Project.Slug)
	}
	if created.SecretKey != "" {
		t.Fatalf("secret key printed without --show-secret:\n%s", out)
	}

	var list struct {
		Projects []struct{ Slug string }
	}
	out = mustRun("", "--context", "it", "project", "list", "--json")
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		t.Fatalf("list --json: %v\n%s", err, out)
	}
	if len(list.Projects) != 1 || list.Projects[0].Slug != "demo" {
		t.Fatalf("list = %+v", list)
	}

	var got struct{ Id, Slug string }
	out = mustRun("", "--context", "it", "project", "get", "demo", "--json")
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("get --json: %v\n%s", err, out)
	}
	if got.Id != created.Project.Id {
		t.Fatalf("get resolved %q, want %q", got.Id, created.Project.Id)
	}

	// Unknown project exits with the not-found code.
	if out, err := run("", "--context", "it", "project", "get", "nope", "--json"); err == nil {
		t.Fatalf("get nope should fail:\n%s", out)
	} else if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 4 {
		t.Fatalf("get nope exit = %v, want 4\n%s", err, out)
	}

	// Declarative apply: create a second project from a hand-written spec,
	// then re-apply it — the second run must report zero changes.
	specPath := filepath.Join(workDir, "moth.yaml")
	spec := "name: Demo Two\nslug: demo-two\nsettings:\n  allow_public_signup: true\n  require_email_verification: true\n"
	if err := os.WriteFile(specPath, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}
	type applyOut struct {
		Plan struct {
			Create         bool `json:"create"`
			UpdateName     bool `json:"update_name"`
			UpdateSettings bool `json:"update_settings"`
			UpdateTheme    bool `json:"update_theme"`
			ResetTheme     bool `json:"reset_theme"`
		} `json:"plan"`
		Applied bool `json:"applied"`
	}
	var first, second applyOut
	out = mustRun("", "--context", "it", "project", "apply", "-f", specPath, "--yes", "--json")
	if err := json.Unmarshal([]byte(out), &first); err != nil {
		t.Fatalf("apply --json: %v\n%s", err, out)
	}
	if !first.Applied || !first.Plan.Create {
		t.Fatalf("first apply should create: %+v\n%s", first, out)
	}
	out = mustRun("", "--context", "it", "project", "apply", "-f", specPath, "--yes", "--json")
	if err := json.Unmarshal([]byte(out), &second); err != nil {
		t.Fatalf("re-apply --json: %v\n%s", err, out)
	}
	if second.Applied || second.Plan.Create || second.Plan.UpdateName ||
		second.Plan.UpdateSettings || second.Plan.UpdateTheme || second.Plan.ResetTheme {
		t.Fatalf("second apply must be a no-op: %+v\n%s", second, out)
	}

	// dump → apply is equally idempotent (the acceptance criterion's
	// dump-derived form).
	dump := mustRun("", "--context", "it", "project", "dump", "demo")
	dumpPath := filepath.Join(workDir, "dump.yaml")
	if err := os.WriteFile(dumpPath, []byte(dump), 0o600); err != nil {
		t.Fatal(err)
	}
	var reapplied applyOut
	out = mustRun("", "--context", "it", "project", "apply", "-f", dumpPath, "--yes", "--json")
	if err := json.Unmarshal([]byte(out), &reapplied); err != nil {
		t.Fatalf("apply dump --json: %v\n%s", err, out)
	}
	if reapplied.Applied {
		t.Fatalf("applying a fresh dump must be a no-op: %+v\n%s", reapplied, out)
	}

	// project update --name.
	var updated struct {
		Project struct{ Name string }
	}
	out = mustRun("", "--context", "it", "project", "update", "demo", "--name", "Demo v2", "--json")
	if err := json.Unmarshal([]byte(out), &updated); err != nil {
		t.Fatalf("update --json: %v\n%s", err, out)
	}
	if updated.Project.Name != "Demo v2" {
		t.Fatalf("update renamed to %q", updated.Project.Name)
	}

	// project keys show: the offline-verification values.
	var keysShow struct {
		Key      struct{ Kid, Algorithm string }
		JwksUrl  string
		Issuer   string
		Audience string
	}
	out = mustRun("", "--context", "it", "project", "keys", "show", "demo", "--json")
	if err := json.Unmarshal([]byte(out), &keysShow); err != nil {
		t.Fatalf("keys show --json: %v\n%s", err, out)
	}
	if keysShow.Key.Algorithm != "ES256" || keysShow.Audience != "demo" ||
		!strings.Contains(keysShow.JwksUrl, "/p/demo/.well-known/jwks.json") {
		t.Fatalf("keys show = %+v", keysShow)
	}

	// Secret-key lifecycle: create --show-secret prints the sk_ key
	// (the only terminal recovery route), regenerate-secret refuses to
	// rotate without --show-secret, and rotates+prints with it.
	var withSecret struct {
		Project   struct{ Slug string }
		SecretKey string
	}
	out = mustRun("", "--context", "it", "project", "create", "Keys", "--slug", "keys", "--show-secret", "--json")
	if err := json.Unmarshal([]byte(out), &withSecret); err != nil {
		t.Fatalf("create --show-secret --json: %v\n%s", err, out)
	}
	if !strings.HasPrefix(withSecret.SecretKey, "sk_") {
		t.Fatalf("create --show-secret should print the sk_ key:\n%s", out)
	}
	if out, err := run("", "--context", "it", "project", "keys", "regenerate-secret", "keys", "--yes", "--json"); err == nil {
		t.Fatalf("regenerate-secret without --show-secret must refuse (it would burn the only copy):\n%s", out)
	}
	var regen struct{ SecretKey string }
	out = mustRun("", "--context", "it", "project", "keys", "regenerate-secret", "keys", "--yes", "--show-secret", "--json")
	if err := json.Unmarshal([]byte(out), &regen); err != nil {
		t.Fatalf("regenerate-secret --json: %v\n%s", err, out)
	}
	if !strings.HasPrefix(regen.SecretKey, "sk_") || regen.SecretKey == withSecret.SecretKey {
		t.Fatalf("regenerate-secret should mint a fresh sk_ key:\n%s", out)
	}
	mustRun("", "--context", "it", "project", "delete", "keys", "--yes")

	// The user group: create, list, get, claims, disable/enable, sessions.
	var createdUser struct {
		User struct{ Id, Email string }
	}
	out = mustRun("", "--context", "it", "user", "create", "u1@example.com", "--project", "demo",
		"--password", "correct horse", "--verified", "--display-name", "User One", "--json")
	if err := json.Unmarshal([]byte(out), &createdUser); err != nil {
		t.Fatalf("user create --json: %v\n%s", err, out)
	}
	var users struct {
		Users []struct{ Id, Email string }
	}
	out = mustRun("", "--context", "it", "user", "list", "--project", "demo", "--json")
	if err := json.Unmarshal([]byte(out), &users); err != nil {
		t.Fatalf("user list --json: %v\n%s", err, out)
	}
	if len(users.Users) != 1 || users.Users[0].Id != createdUser.User.Id {
		t.Fatalf("user list = %+v", users)
	}
	var gotUser struct {
		User struct {
			Id           string
			CustomClaims string
			Disabled     bool
		}
	}
	out = mustRun("", "--context", "it", "user", "claims", "set", "u1@example.com", `{"role":"admin"}`,
		"--project", "demo", "--json")
	if err := json.Unmarshal([]byte(out), &gotUser); err != nil {
		t.Fatalf("claims set --json: %v\n%s", err, out)
	}
	if gotUser.User.CustomClaims != `{"role":"admin"}` {
		t.Fatalf("claims set = %+v", gotUser)
	}
	out = mustRun("", "--context", "it", "user", "disable", createdUser.User.Id, "--project", "demo", "--json")
	if err := json.Unmarshal([]byte(out), &gotUser); err != nil {
		t.Fatalf("user disable --json: %v\n%s", err, out)
	}
	if !gotUser.User.Disabled {
		t.Fatalf("user disable = %+v", gotUser)
	}
	mustRun("", "--context", "it", "user", "enable", createdUser.User.Id, "--project", "demo")
	gotUser.User.Disabled = false // protojson omits false fields; reset before re-parsing
	out = mustRun("", "--context", "it", "user", "get", "u1@example.com", "--project", "demo", "--json")
	if err := json.Unmarshal([]byte(out), &gotUser); err != nil {
		t.Fatalf("user get --json: %v\n%s", err, out)
	}
	if gotUser.User.Id != createdUser.User.Id || gotUser.User.Disabled {
		t.Fatalf("user get = %+v", gotUser)
	}
	mustRun("", "--context", "it", "user", "sessions", "revoke", "u1@example.com", "--project", "demo", "--yes")

	// project export → import round trip (users-JSON data lifecycle).
	exportPath := filepath.Join(workDir, "users.json")
	mustRun("", "--context", "it", "project", "export", "demo", "-o", exportPath)
	var exported struct {
		Project string `json:"project"`
		Users   []struct {
			Email        string `json:"email"`
			CustomClaims string `json:"custom_claims"`
		} `json:"users"`
	}
	raw, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &exported); err != nil {
		t.Fatalf("export document: %v\n%s", err, raw)
	}
	if exported.Project != "demo" || len(exported.Users) != 1 ||
		exported.Users[0].Email != "u1@example.com" || exported.Users[0].CustomClaims != `{"role":"admin"}` {
		t.Fatalf("export = %+v", exported)
	}
	mustRun("", "--context", "it", "project", "create", "Copy", "--slug", "copy")
	var imported struct {
		Created int `json:"created"`
		Skipped int `json:"skipped"`
	}
	out = mustRun("", "--context", "it", "project", "import", "copy", "-f", exportPath, "--yes", "--json")
	if err := json.Unmarshal([]byte(out), &imported); err != nil {
		t.Fatalf("import --json: %v\n%s", err, out)
	}
	if imported.Created != 1 || imported.Skipped != 0 {
		t.Fatalf("import = %+v", imported)
	}
	// Idempotent: a re-import skips the existing user.
	out = mustRun("", "--context", "it", "project", "import", "copy", "-f", exportPath, "--yes", "--json")
	if err := json.Unmarshal([]byte(out), &imported); err != nil {
		t.Fatalf("re-import --json: %v\n%s", err, out)
	}
	if imported.Created != 0 || imported.Skipped != 1 {
		t.Fatalf("re-import = %+v", imported)
	}
	out = mustRun("", "--context", "it", "user", "get", "u1@example.com", "--project", "copy", "--json")
	if err := json.Unmarshal([]byte(out), &gotUser); err != nil {
		t.Fatalf("imported user get --json: %v\n%s", err, out)
	}
	if gotUser.User.CustomClaims != `{"role":"admin"}` {
		t.Fatalf("import should restore claims, got %+v", gotUser)
	}
	mustRun("", "--context", "it", "project", "delete", "copy", "--yes")

	// stats get: the tiles render even for a quiet project.
	var stats struct {
		Tiles struct{ TotalUsers string }
	}
	out = mustRun("", "--context", "it", "stats", "get", "--project", "demo", "--json")
	if err := json.Unmarshal([]byte(out), &stats); err != nil {
		t.Fatalf("stats get --json: %v\n%s", err, out)
	}
	if stats.Tiles.TotalUsers != "1" {
		t.Fatalf("stats tiles = %+v\n%s", stats, out)
	}

	// instance get: base URL matches what the CLI dialed.
	var instance struct{ BaseUrl string }
	out = mustRun("", "--context", "it", "instance", "get", "--json")
	if err := json.Unmarshal([]byte(out), &instance); err != nil {
		t.Fatalf("instance get --json: %v\n%s", err, out)
	}
	if instance.BaseUrl != baseURL {
		t.Fatalf("instance base url = %q, want %q", instance.BaseUrl, baseURL)
	}

	// doctor: instance + project checks against the real binary. No SMTP is
	// configured, so the overall status is WARN, never FAIL (exit 0).
	var doctorRep struct {
		Status string `json:"status"`
		Checks []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"checks"`
	}
	out = mustRun("", "--context", "it", "doctor", "--project", "demo", "--json")
	if err := json.Unmarshal([]byte(out), &doctorRep); err != nil {
		t.Fatalf("doctor --json: %v\n%s", err, out)
	}
	if doctorRep.Status != "WARN" {
		t.Fatalf("doctor status = %s\n%s", doctorRep.Status, out)
	}
	wantPass := map[string]bool{
		"instance: health endpoint":             false,
		"instance: pub endpoint serves the SDK": false,
		"instance: base URL sanity":             false,
		"project: JWKS reachable":               false,
	}
	for _, c := range doctorRep.Checks {
		if _, ok := wantPass[c.Name]; ok && c.Status == "PASS" {
			wantPass[c.Name] = true
		}
	}
	for name, passed := range wantPass {
		if !passed {
			t.Fatalf("doctor check %q did not pass:\n%s", name, out)
		}
	}
	// A typoed project slug exits with the not-found code.
	if out, err := run("", "--context", "it", "setup", "google", "--project", "nope", "--gcp-project", "demo-project"); err == nil {
		t.Fatalf("setup google --project nope should fail:\n%s", out)
	} else if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 4 {
		t.Fatalf("setup google --project nope exit = %v, want 4\n%s", err, out)
	}

	// Remote PAT management: list shows the login token, revoke kills it.
	tokensOut := mustRun("", "--context", "it", "token", "list")
	if !strings.Contains(tokensOut, "it") {
		t.Fatalf("token list should include the login token:\n%s", tokensOut)
	}

	// Destructive op without --yes and without a TTY fails closed.
	if out, err := run("", "--context", "it", "project", "delete", "demo-two"); err == nil {
		t.Fatalf("delete without --yes should fail:\n%s", out)
	}
	mustRun("", "--context", "it", "project", "delete", "demo-two", "--yes")

	// Revoking the PAT makes the very next call fail with the auth code.
	mustRun("", "admin", "token", "revoke", "--email", "ops@example.com", "--id", minted.Metadata.Id, "--data-dir", dataDir)
	if out, err := run("", "--context", "it", "project", "list", "--json"); err == nil {
		t.Fatalf("revoked PAT should fail:\n%s", out)
	} else if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 3 {
		t.Fatalf("revoked PAT exit = %v, want 3\n%s", err, out)
	}
}

func waitForServer(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server at %s did not come up", url)
}
