package server

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	billingv1 "github.com/aloisdeniel/moth/gen/moth/billing/v1"
)

// projectListing fetches a per-project pub version listing.
func projectListing(t *testing.T, e *testEnv, slug, pkg string) pubListing {
	t.Helper()
	resp, err := e.client.Get(e.url + "/p/" + slug + "/pub/api/packages/" + pkg)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("listing %s/%s: status %d", slug, pkg, resp.StatusCode)
	}
	var l pubListing
	if err := json.NewDecoder(resp.Body).Decode(&l); err != nil {
		t.Fatal(err)
	}
	return l
}

// tarFile downloads a per-project tarball and returns one member's bytes.
func tarFile(t *testing.T, e *testEnv, slug, pkg, version, member string) []byte {
	t.Helper()
	resp, err := e.client.Get(e.url + "/p/" + slug + "/pub/packages/" + pkg + "/versions/" + version + ".tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("archive %s/%s@%s: status %d", slug, pkg, version, resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Name == member {
			b, err := io.ReadAll(tr)
			if err != nil {
				t.Fatal(err)
			}
			return b
		}
	}
	t.Fatalf("member %q not found in %s/%s@%s", member, slug, pkg, version)
	return nil
}

// genConstValue extracts the value of a `const String NAME = '...';` line from
// the generated_config.dart source.
func genConstValue(t *testing.T, src, name string) string {
	t.Helper()
	prefix := "const String " + name + " = '"
	for _, line := range strings.Split(src, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSuffix(strings.TrimPrefix(line, prefix), "';")
		}
	}
	t.Fatalf("no %s constant in generated_config.dart:\n%s", name, src)
	return ""
}

func TestPubProjectPreconfiguredPackage(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Bird Spotter")

	l := projectListing(t, e, p.Slug, "moth_auth")

	// The version encodes the config revisions (the `.r<hash>` identifier), so
	// a config edit bumps it.
	if !strings.Contains(l.Latest.Version, ".r") {
		t.Fatalf("version %q does not encode a config revision", l.Latest.Version)
	}
	// The listing's pubspec version matches the served version.
	if got := l.Latest.Pubspec["version"]; got != l.Latest.Version {
		t.Fatalf("pubspec version %v != listing version %q", got, l.Latest.Version)
	}

	// The tarball's generated_config.dart carries the project's real values.
	src := string(tarFile(t, e, p.Slug, "moth_auth", l.Latest.Version, "lib/src/generated_config.dart"))
	if got := genConstValue(t, src, "mothEndpoint"); got != "http://localhost:8080" {
		t.Fatalf("mothEndpoint = %q, want the instance base URL", got)
	}
	if got := genConstValue(t, src, "mothPublishableKey"); got != p.PublishableKey {
		t.Fatalf("mothPublishableKey = %q, want %q", got, p.PublishableKey)
	}

	// The config seed decodes to a real GetProjectConfigResponse with the
	// theme and default-locale copy baked in (no round-trip needed on launch).
	cfgB64 := genConstValue(t, src, "mothConfigB64")
	if cfgB64 == "" {
		t.Fatal("mothConfigB64 is empty")
	}
	cfgBytes, err := base64.StdEncoding.DecodeString(cfgB64)
	if err != nil {
		t.Fatalf("mothConfigB64 not valid base64: %v", err)
	}
	var cfg authv1.GetProjectConfigResponse
	if err := proto.Unmarshal(cfgBytes, &cfg); err != nil {
		t.Fatalf("mothConfigB64 not a GetProjectConfigResponse: %v", err)
	}
	if cfg.Theme == nil || cfg.Theme.RevisionId == "" {
		t.Fatal("baked config has no theme")
	}
	if cfg.Copy == nil || cfg.Copy.CopyRevision == "" || len(cfg.Copy.Messages) == 0 {
		t.Fatal("baked config has no localized copy messages")
	}

	// The paywall seed decodes to a real Paywall (the built-in default here).
	pwB64 := genConstValue(t, src, "mothPaywallB64")
	pwBytes, err := base64.StdEncoding.DecodeString(pwB64)
	if err != nil {
		t.Fatalf("mothPaywallB64 not valid base64: %v", err)
	}
	var pw billingv1.Paywall
	if err := proto.Unmarshal(pwBytes, &pw); err != nil {
		t.Fatalf("mothPaywallB64 not a Paywall: %v", err)
	}
	if pw.RevisionId == "" {
		t.Fatal("baked paywall has no revision id")
	}

	// The companion moth_billing pins moth_auth against THIS project's pub URL
	// and the same version, so the whole set resolves per project.
	billing := projectListing(t, e, p.Slug, "moth_billing")
	if billing.Latest.Version != l.Latest.Version {
		t.Fatalf("companion version %q != moth_auth version %q", billing.Latest.Version, l.Latest.Version)
	}
	deps, _ := billing.Latest.Pubspec["dependencies"].(map[string]any)
	dep, _ := deps["moth_auth"].(map[string]any)
	wantHosted := "http://localhost:8080/p/" + p.Slug + "/pub"
	if dep["hosted"] != wantHosted {
		t.Fatalf("companion moth_auth hosted = %v, want %q", dep["hosted"], wantHosted)
	}
	if dep["version"] != l.Latest.Version {
		t.Fatalf("companion moth_auth version pin = %v, want %q", dep["version"], l.Latest.Version)
	}
}

func TestPubProjectVersionBumpsOnConfigEdit(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Config Edit App")

	before := projectListing(t, e, p.Slug, "moth_auth").Latest.Version

	// Editing the copy mints a new copy revision on the project row, which the
	// package version folds in.
	if _, err := e.copySvc().UpdateProjectCopy(context.Background(), connect.NewRequest(&adminv1.UpdateProjectCopyRequest{
		ProjectId: p.Id, Locale: "en", Values: map[string]string{"sign_in.title": "Welcome"},
	})); err != nil {
		t.Fatal(err)
	}

	after := projectListing(t, e, p.Slug, "moth_auth").Latest.Version
	if before == after {
		t.Fatalf("version did not change after a config edit: still %q", after)
	}
}

func TestPubProjectUnknownSlug(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")

	resp, err := e.client.Get(e.url + "/p/does-not-exist/pub/api/packages/moth_auth")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown slug: status %d, want 404", resp.StatusCode)
	}
}

func TestPubVersionForProject(t *testing.T) {
	const base = "1.0.0"
	const url = "https://auth.example.com/p/app/pub"

	v1 := pubVersionForProject(base, url, "themeA|copyA|paywallA")
	// Deterministic.
	if v1 != pubVersionForProject(base, url, "themeA|copyA|paywallA") {
		t.Fatal("pubVersionForProject is not deterministic")
	}
	// A changed revision signature changes the version.
	if v1 == pubVersionForProject(base, url, "themeB|copyA|paywallA") {
		t.Fatal("version did not change when the revision signature changed")
	}
	// A valid semver build-metadata suffix (`+u….r…`).
	if !semverRe.MatchString(v1) {
		t.Fatalf("version %q is not valid semver", v1)
	}
	if !strings.Contains(v1, ".r") {
		t.Fatalf("version %q lacks the revision identifier", v1)
	}
}
