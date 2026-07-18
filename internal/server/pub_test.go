package server

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/config"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/store"
)

// pubListing mirrors the repository-spec-v2 version listing.
type pubListing struct {
	Name     string         `json:"name"`
	Latest   pubVersionTJ   `json:"latest"`
	Versions []pubVersionTJ `json:"versions"`
}

type pubVersionTJ struct {
	Version       string         `json:"version"`
	Pubspec       map[string]any `json:"pubspec"`
	ArchiveURL    string         `json:"archive_url"`
	ArchiveSHA256 string         `json:"archive_sha256"`
}

// devPubVersion returns the version a dev build (no release ldflags, i.e.
// every test binary) stamps on the served package, for the testEnv base URL:
// the version carries a discriminator for the /pub URL baked into companion
// tarballs (see pubVersionForURL).
func devPubVersion(t *testing.T) string {
	t.Helper()
	hash, err := sdkTreeHash()
	if err != nil {
		t.Fatal(err)
	}
	v, err := pubVersion("dev", hash)
	if err != nil {
		t.Fatal(err)
	}
	return pubVersionForURL(v, "http://localhost:8080/pub")
}

func TestPubVersionListing(t *testing.T) {
	e := newTestEnv(t, "")
	devVersion := devPubVersion(t)

	// Every served package is listed, fetchable and stamped with the same
	// version — the SDK-lockstep discipline of plan/19.
	for _, pkg := range []string{"moth_auth", "moth_billing"} {
		resp, err := e.client.Get(e.url + "/pub/api/packages/" + pkg)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s listing: %d", pkg, resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); ct != "application/vnd.pub.v2+json" {
			t.Fatalf("%s content type: %s", pkg, ct)
		}
		var listing pubListing
		if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
			t.Fatal(err)
		}
		if listing.Name != pkg || len(listing.Versions) != 1 {
			t.Fatalf("%s listing: %+v", pkg, listing)
		}
		v := listing.Versions[0]
		// The test binary is a dev build (no release ldflags).
		if v.Version != devVersion || listing.Latest.Version != v.Version {
			t.Fatalf("%s version: %q (latest %q), want %q", pkg, v.Version, listing.Latest.Version, devVersion)
		}
		// archive_url is absolute, rooted at the configured base URL (the
		// testEnv config, not the httptest listener).
		wantURL := "http://localhost:8080/pub/packages/" + pkg + "/versions/" + devVersion + ".tar.gz"
		if v.ArchiveURL != wantURL {
			t.Fatalf("archive_url: %q, want %q", v.ArchiveURL, wantURL)
		}
		if name := v.Pubspec["name"]; name != pkg {
			t.Fatalf("pubspec.name: %v", name)
		}
		if got := v.Pubspec["version"]; got != v.Version {
			t.Fatalf("pubspec.version: %v, want %s", got, v.Version)
		}

		// The advertised sha256 matches the served tarball bytes.
		tarball := fetchPubTarball(t, e, pkg)
		sum := sha256.Sum256(tarball)
		if hex.EncodeToString(sum[:]) != v.ArchiveSHA256 {
			t.Fatalf("%s archive_sha256 does not match the served tarball", pkg)
		}
	}
}

// untarPub extracts a fetched package tarball into a path → content map.
func untarPub(t *testing.T, tarball []byte) map[string]string {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(tarball))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		files[hdr.Name] = string(raw)
	}
	return files
}

func TestPubArchiveContents(t *testing.T) {
	e := newTestEnv(t, "")
	files := untarPub(t, fetchPubTarball(t, e, "moth_auth"))

	devVersion := devPubVersion(t)
	pubspec, ok := files["pubspec.yaml"]
	if !ok {
		t.Fatal("tarball has no pubspec.yaml")
	}
	if !strings.Contains(pubspec, "\nversion: "+devVersion+"\n") {
		t.Fatalf("pubspec version not stamped:\n%s", pubspec)
	}
	// The runtime constant is stamped alongside the pubspec so the SDK
	// reports the served version in x-moth-sdk-version metadata.
	versionDart, ok := files["lib/src/version.dart"]
	if !ok {
		t.Fatal("tarball has no lib/src/version.dart")
	}
	if !strings.Contains(versionDart, "const String mothSdkVersion = '"+devVersion+"';") {
		t.Fatalf("mothSdkVersion not stamped:\n%s", versionDart)
	}
	for _, want := range []string{"lib/moth_auth.dart", "lib/src/client.dart", "README.md", "LICENSE", "CHANGELOG.md"} {
		if _, ok := files[want]; !ok {
			t.Errorf("tarball misses %s", want)
		}
	}
	for name := range files {
		if strings.HasPrefix(name, "example/") || strings.HasPrefix(name, "test/") ||
			strings.HasPrefix(name, "build/") || name == "pubspec.lock" {
			t.Errorf("tarball leaks %s", name)
		}
	}
}

func TestPubBillingArchiveContents(t *testing.T) {
	e := newTestEnv(t, "")
	files := untarPub(t, fetchPubTarball(t, e, "moth_billing"))

	devVersion := devPubVersion(t)
	pubspec, ok := files["pubspec.yaml"]
	if !ok {
		t.Fatal("tarball has no pubspec.yaml")
	}
	if !strings.Contains(pubspec, "\nversion: "+devVersion+"\n") {
		t.Fatalf("pubspec version not stamped:\n%s", pubspec)
	}
	// The placeholder moth_auth hosted dependency is rewritten to this
	// instance's /pub (config BaseURL) at the served moth_auth version, so
	// the two packages resolve in lockstep.
	wantDep := "  moth_auth:\n" +
		"    hosted: http://localhost:8080/pub\n" +
		"    version: " + devVersion + "\n"
	if !strings.Contains(pubspec, wantDep) {
		t.Fatalf("moth_auth dependency not rewritten:\n%s", pubspec)
	}
	if strings.Contains(pubspec, "moth.invalid") {
		t.Fatalf("placeholder host leaked:\n%s", pubspec)
	}
	// The plugin's Dart and native sources ship in the tarball (plan/19: CI
	// asserts the embedded FS contains the native sources).
	for _, want := range []string{
		"lib/moth_billing.dart",
		"lib/src/moth_store_billing.dart",
		"ios/moth_billing.podspec",
		"ios/Classes/MothBillingPlugin.swift",
		"android/build.gradle",
		"android/settings.gradle",
		"android/src/main/AndroidManifest.xml",
		"android/src/main/kotlin/io/moth/billing/MothBillingPlugin.kt",
		"README.md", "LICENSE", "CHANGELOG.md",
	} {
		if _, ok := files[want]; !ok {
			t.Errorf("tarball misses %s", want)
		}
	}
	// pubspec_overrides.yaml is the local-dev path override to ../flutter:
	// serving it would shadow the hosted moth_auth dependency.
	for name := range files {
		if strings.HasPrefix(name, "example/") || strings.HasPrefix(name, "test/") ||
			strings.HasPrefix(name, "build/") || name == "pubspec.lock" ||
			name == "pubspec_overrides.yaml" {
			t.Errorf("tarball leaks %s", name)
		}
	}
}

func TestPubVersionMapping(t *testing.T) {
	const hash = "1a2b3c4d5e6f7788"
	for in, want := range map[string]string{
		"dev":             "0.0.0-dev.h1a2b3c4d", // no release ldflags: tree-hash pre-release
		"v1.2.3":          "1.2.3",
		"1.2.3":           "1.2.3",
		"v2.0.0-rc.1":     "2.0.0-rc.1",
		"v1.2.3-rc.1+42":  "1.2.3-rc.1+42", // pre-release plus build metadata
		"v1.2.3+security": "1.2.3+security",
	} {
		got, err := pubVersion(in, hash)
		if err != nil {
			t.Errorf("pubVersion(%q) failed: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("pubVersion(%q) = %q, want %q", in, got, want)
		}
	}
	// A release build stamped with a version pub would reject must fail
	// loudly instead of silently serving a dev version.
	for _, in := range []string{"nightly-2024", "1.2.3rc1", "release-1.2.3", "1.2"} {
		if got, err := pubVersion(in, hash); err == nil {
			t.Errorf("pubVersion(%q) = %q, want error", in, got)
		}
	}
}

func TestPubVersionForURL(t *testing.T) {
	// Different pub URLs must never share a version: the URL is baked into
	// companion tarballs, and pub's cache and pubspec.lock archive_sha256
	// key on (name, version).
	a := pubVersionForURL("1.2.3", "https://a.example/pub")
	b := pubVersionForURL("1.2.3", "https://b.example/pub")
	if a == b {
		t.Fatalf("same version %q for different pub URLs", a)
	}
	for _, v := range []string{a, b} {
		if !semverRe.MatchString(v) {
			t.Errorf("pubVersionForURL produced invalid semver %q", v)
		}
		if !strings.HasPrefix(v, "1.2.3+u") {
			t.Errorf("version %q does not carry the URL discriminator as build metadata", v)
		}
	}
	// Existing build metadata gains a dot identifier, never a second '+'.
	c := pubVersionForURL("1.2.3+42", "https://a.example/pub")
	if strings.Count(c, "+") != 1 || !strings.HasPrefix(c, "1.2.3+42.u") || !semverRe.MatchString(c) {
		t.Errorf("pubVersionForURL(1.2.3+42) = %q", c)
	}
	// The same URL always maps to the same version (determinism).
	if again := pubVersionForURL("1.2.3", "https://a.example/pub"); again != a {
		t.Errorf("pubVersionForURL not deterministic: %q then %q", a, again)
	}
}

// TestPubArchivesVaryWithBaseURL is the regression test for the base_url
// staleness class: moth_billing's tarball bakes the instance's /pub URL into
// its moth_auth hosted dependency, so two base URLs must never serve
// different bytes under the same (name, version).
func TestPubArchivesVaryWithBaseURL(t *testing.T) {
	for _, mothVersion := range []string{"dev", "v1.2.3"} {
		a, err := buildPubArchives(mothVersion, "https://a.example")
		if err != nil {
			t.Fatal(err)
		}
		b, err := buildPubArchives(mothVersion, "https://b.example")
		if err != nil {
			t.Fatal(err)
		}
		for name := range a {
			if a[name].version == b[name].version {
				t.Errorf("%s %s: same version %q for different base URLs", mothVersion, name, a[name].version)
			}
		}
		// The moth_billing bytes really do differ between the two base URLs
		// (the reason the version must): a same-version check above with
		// identical bytes would be vacuous.
		if a["moth_billing"].sha256 == b["moth_billing"].sha256 {
			t.Errorf("%s: moth_billing bytes unexpectedly identical across base URLs", mothVersion)
		}
	}
}

// TestPubArchiveReproducible verifies the determinism the advertised
// archive_sha256 depends on: two builds from the same embedded trees and
// base URL must be byte-identical (replicas built from the same source and
// config advertise the same hash, and pubspec.lock content hashes stay
// valid across restarts).
func TestPubArchiveReproducible(t *testing.T) {
	a, err := buildPubArchives("dev", "http://localhost:8080")
	if err != nil {
		t.Fatal(err)
	}
	b, err := buildPubArchives("dev", "http://localhost:8080")
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != len(pubPackages) {
		t.Fatalf("built %d archives, want %d", len(a), len(pubPackages))
	}
	for name, av := range a {
		bv := b[name]
		if bv == nil || av.sha256 != bv.sha256 || !bytes.Equal(av.tarball, bv.tarball) {
			t.Fatalf("%s archive not reproducible", name)
		}
	}
}

// TestBuildPubArchivesRejectsBadVersion covers the release-build guard: an
// unparsable version must fail archive construction (and with it server
// startup) rather than mis-stamp the packages.
func TestBuildPubArchivesRejectsBadVersion(t *testing.T) {
	if _, err := buildPubArchives("release-1.2.3", "http://localhost:8080"); err == nil {
		t.Fatal("buildPubArchives accepted an invalid release version")
	}
}

func TestPubNotFound(t *testing.T) {
	e := newTestEnv(t, "")
	devVersion := devPubVersion(t)
	for _, path := range []string{
		"/pub/api/packages/other_package",
		"/pub/packages/other_package/versions/" + devVersion + ".tar.gz",
		"/pub/packages/moth_auth/versions/9.9.9.tar.gz",
		"/pub/packages/moth_auth/versions/" + devVersion + ".zip",
		"/pub/packages/moth_billing/versions/9.9.9.tar.gz",
	} {
		resp, err := e.client.Get(e.url + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: want 404, got %d", path, resp.StatusCode)
		}
	}
}

func TestVersionHeaderOnRPCs(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	ctx := context.Background()

	// Success path: response header.
	resp, err := e.sessions.GetCurrentAdmin(ctx,
		connect.NewRequest(&adminv1.GetCurrentAdminRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got := resp.Header().Get("X-Moth-Version"); got != "dev" {
		t.Fatalf("x-moth-version on success: %q", got)
	}

	// Error path: error metadata.
	anon := adminv1connect.NewProjectServiceClient(http.DefaultClient, e.url)
	_, err = anon.ListProjects(ctx, connect.NewRequest(&adminv1.ListProjectsRequest{}))
	var cerr *connect.Error
	if !errors.As(err, &cerr) {
		t.Fatalf("want connect error, got %v", err)
	}
	if got := cerr.Meta().Get("X-Moth-Version"); got != "dev" {
		t.Fatalf("x-moth-version on error: %q", got)
	}
}

// TestFlutterPubGet resolves and downloads moth_auth from a running moth
// handler with the real pub client — the milestone's acceptance criterion.
// It needs the Flutter toolchain (the package depends on the Flutter SDK)
// and network access to pub.dev for the transitive dependencies, so it
// skips when flutter is absent or -short is set. `dart pub` allows plain
// http only for loopback hosts, which is why the server listens on
// 127.0.0.1 with its BaseURL pointing at the same listener.
func TestFlutterPubGet(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: skipped in -short mode")
	}
	flutter, err := exec.LookPath("flutter")
	if err != nil {
		t.Skip("integration test: flutter not on PATH")
	}

	// The pub client follows archive_url from the listing, so BaseURL must
	// match the address actually listened on: listen first, then assemble.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	baseURL := "http://" + ln.Addr().String()

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
	srv, err := New(Options{
		Config: config.Config{Addr: ln.Addr().String(), DataDir: dir, BaseURL: baseURL},
		Store:  st,
		Master: master,
		Logger: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	hs := &http.Server{Handler: srv.Handler()}
	go hs.Serve(ln)
	t.Cleanup(func() { hs.Close() })

	// A minimal consumer app pinning the exact served version (a dev build
	// serves a 0.0.0-dev.* pre-release, which ranges would not match).
	// moth_auth resolves transitively through the companions' rewritten
	// hosted dependencies — the plan/19 lockstep path.
	version := srv.pub["moth_auth"].version
	app := t.TempDir()
	pubspec := `name: pubtest
environment:
  sdk: ^3.9.0
dependencies:
  flutter:
    sdk: flutter
  moth_billing:
    hosted: ` + baseURL + `/pub
    version: ` + version + `
`
	if err := os.WriteFile(filepath.Join(app, "pubspec.yaml"), []byte(pubspec), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, flutter, "pub", "get")
	cmd.Dir = app
	// A private pub cache keeps the developer's real cache clean (each run
	// would otherwise leave a hosted/127.0.0.1%58<port> entry behind).
	cmd.Env = append(os.Environ(), "PUB_CACHE="+filepath.Join(t.TempDir(), "pub-cache"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("flutter pub get: %v\n%s", err, out)
	}

	// Both resolved packages really are ours, with the stamped version.
	raw, err := os.ReadFile(filepath.Join(app, ".dart_tool", "package_config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Packages []struct {
			Name    string `json:"name"`
			RootURI string `json:"rootUri"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	resolved := map[string]bool{}
	for _, p := range cfg.Packages {
		if p.Name != "moth_auth" && p.Name != "moth_billing" {
			continue
		}
		// rootUri is a percent-encoded file URI into the pub cache.
		u, err := url.Parse(p.RootURI)
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(u.Path, "pubspec.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(got), "version: "+version) {
			t.Fatalf("downloaded %s pubspec version not stamped:\n%s", p.Name, got)
		}
		resolved[p.Name] = true
	}
	if !resolved["moth_auth"] || !resolved["moth_billing"] {
		t.Fatalf("packages missing from package_config.json (got %v):\n%s", resolved, raw)
	}
}

func fetchPubTarball(t *testing.T, e *testEnv, pkg string) []byte {
	t.Helper()
	resp, err := e.client.Get(e.url + "/pub/packages/" + pkg + "/versions/" + devPubVersion(t) + ".tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("tarball: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("tarball content type: %s", ct)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
