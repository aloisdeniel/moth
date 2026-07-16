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
// every test binary) stamps on the served package.
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
	return v
}

func TestPubVersionListing(t *testing.T) {
	e := newTestEnv(t, "")
	devVersion := devPubVersion(t)

	resp, err := e.client.Get(e.url + "/pub/api/packages/moth_auth")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("listing: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/vnd.pub.v2+json" {
		t.Fatalf("content type: %s", ct)
	}
	var listing pubListing
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		t.Fatal(err)
	}
	if listing.Name != "moth_auth" || len(listing.Versions) != 1 {
		t.Fatalf("listing: %+v", listing)
	}
	v := listing.Versions[0]
	// The test binary is a dev build (no release ldflags).
	if v.Version != devVersion || listing.Latest.Version != v.Version {
		t.Fatalf("version: %q (latest %q), want %q", v.Version, listing.Latest.Version, devVersion)
	}
	// archive_url is absolute, rooted at the configured base URL (the
	// testEnv config, not the httptest listener).
	wantURL := "http://localhost:8080/pub/packages/moth_auth/versions/" + devVersion + ".tar.gz"
	if v.ArchiveURL != wantURL {
		t.Fatalf("archive_url: %q, want %q", v.ArchiveURL, wantURL)
	}
	if name := v.Pubspec["name"]; name != "moth_auth" {
		t.Fatalf("pubspec.name: %v", name)
	}
	if got := v.Pubspec["version"]; got != v.Version {
		t.Fatalf("pubspec.version: %v, want %s", got, v.Version)
	}

	// The advertised sha256 matches the served tarball bytes.
	tarball := fetchPubTarball(t, e)
	sum := sha256.Sum256(tarball)
	if hex.EncodeToString(sum[:]) != v.ArchiveSHA256 {
		t.Fatal("archive_sha256 does not match the served tarball")
	}
}

func TestPubArchiveContents(t *testing.T) {
	e := newTestEnv(t, "")
	tarball := fetchPubTarball(t, e)

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

// TestPubArchiveReproducible verifies the determinism the advertised
// archive_sha256 depends on: two builds from the same embedded tree must be
// byte-identical (replicas built from the same source advertise the same
// hash, and pubspec.lock content hashes stay valid across restarts).
func TestPubArchiveReproducible(t *testing.T) {
	a, err := buildPubArchive("dev")
	if err != nil {
		t.Fatal(err)
	}
	b, err := buildPubArchive("dev")
	if err != nil {
		t.Fatal(err)
	}
	if a.sha256 != b.sha256 || !bytes.Equal(a.tarball, b.tarball) {
		t.Fatalf("archive not reproducible: %s != %s", a.sha256, b.sha256)
	}
}

// TestBuildPubArchiveRejectsBadVersion covers the release-build guard: an
// unparsable version must fail archive construction (and with it server
// startup) rather than mis-stamp the package.
func TestBuildPubArchiveRejectsBadVersion(t *testing.T) {
	if _, err := buildPubArchive("release-1.2.3"); err == nil {
		t.Fatal("buildPubArchive accepted an invalid release version")
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
	app := t.TempDir()
	pubspec := `name: pubtest
environment:
  sdk: ^3.9.0
dependencies:
  flutter:
    sdk: flutter
  moth_auth:
    hosted: ` + baseURL + `/pub
    version: ` + srv.pub.version + `
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

	// The resolved package really is ours, with the stamped version.
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
	for _, p := range cfg.Packages {
		if p.Name != "moth_auth" {
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
		if !strings.Contains(string(got), "version: "+srv.pub.version) {
			t.Fatalf("downloaded pubspec version not stamped:\n%s", got)
		}
		return
	}
	t.Fatalf("moth_auth missing from package_config.json:\n%s", raw)
}

func fetchPubTarball(t *testing.T, e *testEnv) []byte {
	t.Helper()
	resp, err := e.client.Get(e.url + "/pub/packages/moth_auth/versions/" + devPubVersion(t) + ".tar.gz")
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
