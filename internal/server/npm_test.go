package server

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aloisdeniel/moth/internal/config"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/store"
)

// npmPackument mirrors the registry document npm resolves against.
type npmPackument struct {
	Name     string                  `json:"name"`
	DistTags map[string]string       `json:"dist-tags"`
	Versions map[string]npmVersionTJ `json:"versions"`
}

type npmVersionTJ struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Dependencies     map[string]string `json:"dependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
	Dist             struct {
		Tarball   string `json:"tarball"`
		Shasum    string `json:"shasum"`
		Integrity string `json:"integrity"`
	} `json:"dist"`
}

// devNpmVersion returns the version a dev build (no release ldflags, i.e.
// every test binary) stamps on the served package.
func devNpmVersion(t *testing.T) string {
	t.Helper()
	hash, err := reactTreeHash()
	if err != nil {
		t.Fatal(err)
	}
	v, err := pubVersion("dev", hash)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestNpmPackument(t *testing.T) {
	e := newTestEnv(t, "")
	devVersion := devNpmVersion(t)

	// npm and pnpm percent-encode the scoped slash.
	resp, err := e.client.Get(e.url + "/npm/@moth%2freact")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("packument: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content type: %s", ct)
	}
	var doc npmPackument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if doc.Name != "@moth/react" || len(doc.Versions) != 1 {
		t.Fatalf("packument: %+v", doc)
	}
	// The test binary is a dev build (no release ldflags).
	if doc.DistTags["latest"] != devVersion {
		t.Fatalf("dist-tags.latest: %q, want %q", doc.DistTags["latest"], devVersion)
	}
	v, ok := doc.Versions[devVersion]
	if !ok {
		t.Fatalf("versions misses %q: %+v", devVersion, doc.Versions)
	}
	if v.Name != "@moth/react" || v.Version != devVersion {
		t.Fatalf("version object: name %q version %q", v.Name, v.Version)
	}
	if _, ok := v.Dependencies["@connectrpc/connect-web"]; !ok {
		t.Fatalf("dependencies missing from packument: %+v", v.Dependencies)
	}
	if _, ok := v.PeerDependencies["react"]; !ok {
		t.Fatalf("peerDependencies missing from packument: %+v", v.PeerDependencies)
	}
	// dist.tarball is absolute, rooted at the configured base URL (the
	// testEnv config, not the httptest listener).
	wantURL := "http://localhost:8080/npm/@moth/react/-/react-" + devVersion + ".tgz"
	if v.Dist.Tarball != wantURL {
		t.Fatalf("dist.tarball: %q, want %q", v.Dist.Tarball, wantURL)
	}

	// The advertised hashes match the served tarball bytes: npm verifies
	// the sha1 shasum, pnpm the sha512 integrity.
	tarball := fetchNpmTarball(t, e)
	sum1 := sha1.Sum(tarball)
	if hex.EncodeToString(sum1[:]) != v.Dist.Shasum {
		t.Fatal("dist.shasum does not match the served tarball")
	}
	sum512 := sha512.Sum512(tarball)
	if "sha512-"+base64.StdEncoding.EncodeToString(sum512[:]) != v.Dist.Integrity {
		t.Fatal("dist.integrity does not match the served tarball")
	}
}

// TestNpmPackumentBothURLForms covers the client split on scoped-name
// encoding: npm/pnpm request one percent-encoded segment, yarn/bun the
// literal two-segment path. Both must serve the identical packument.
func TestNpmPackumentBothURLForms(t *testing.T) {
	e := newTestEnv(t, "")
	bodies := make([][]byte, 0, 2)
	for _, path := range []string{"/npm/@moth%2freact", "/npm/@moth/react"} {
		resp, err := e.client.Get(e.url + path)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d", path, resp.StatusCode)
		}
		bodies = append(bodies, raw)
	}
	if !bytes.Equal(bodies[0], bodies[1]) {
		t.Fatalf("encoded and literal packuments differ:\n%s\n%s", bodies[0], bodies[1])
	}
}

func TestNpmTarballContents(t *testing.T) {
	e := newTestEnv(t, "")
	tarball := fetchNpmTarball(t, e)

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

	devVersion := devNpmVersion(t)
	// npm mandates the package/ root (clients strip one leading component).
	for name := range files {
		if !strings.HasPrefix(name, "package/") {
			t.Errorf("entry %s is not under package/", name)
		}
	}
	pkgJSON, ok := files["package/package.json"]
	if !ok {
		t.Fatal("tarball has no package/package.json")
	}
	var pkg struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(pkgJSON), &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.Name != "@moth/react" || pkg.Version != devVersion {
		t.Fatalf("package.json not stamped: name %q version %q, want version %q",
			pkg.Name, pkg.Version, devVersion)
	}
	// The runtime constant is stamped alongside package.json so the SDK
	// reports the served version in x-moth-sdk-version metadata.
	versionJS, ok := files["package/dist/version.js"]
	if !ok {
		t.Fatal("tarball has no package/dist/version.js")
	}
	if !strings.Contains(versionJS, "export const mothSdkVersion = '"+devVersion+"';") {
		t.Fatalf("mothSdkVersion not stamped:\n%s", versionJS)
	}
	for _, want := range []string{
		"package/dist/index.js", "package/dist/index.d.ts",
		"package/README.md", "package/LICENSE", "package/CHANGELOG.md",
	} {
		if _, ok := files[want]; !ok {
			t.Errorf("tarball misses %s", want)
		}
	}
	for name := range files {
		if strings.HasPrefix(name, "package/node_modules/") ||
			strings.HasPrefix(name, "package/src/") ||
			name == "package/package-lock.json" {
			t.Errorf("tarball leaks %s", name)
		}
	}
}

// TestNpmArchiveReproducible verifies the determinism the advertised
// integrity hashes depend on: two builds from the same embedded tree must
// be byte-identical (replicas built from the same source advertise the same
// integrity, and package-lock.json hashes stay valid across restarts).
func TestNpmArchiveReproducible(t *testing.T) {
	a, err := buildNpmArchive("dev")
	if err != nil {
		t.Fatal(err)
	}
	b, err := buildNpmArchive("dev")
	if err != nil {
		t.Fatal(err)
	}
	if a.integrity != b.integrity || !bytes.Equal(a.tarball, b.tarball) {
		t.Fatalf("archive not reproducible: %s != %s", a.integrity, b.integrity)
	}
}

// TestBuildNpmArchiveRejectsBadVersion covers the release-build guard: an
// unparsable version must fail archive construction (and with it server
// startup) rather than mis-stamp the package.
func TestBuildNpmArchiveRejectsBadVersion(t *testing.T) {
	if _, err := buildNpmArchive("release-1.2.3"); err == nil {
		t.Fatal("buildNpmArchive accepted an invalid release version")
	}
}

func TestNpmNotFound(t *testing.T) {
	e := newTestEnv(t, "")
	devVersion := devNpmVersion(t)
	for _, path := range []string{
		"/npm/other-package",
		"/npm/@other%2fpackage",
		"/npm/@moth/react/-/react-9.9.9.tgz",
		"/npm/@moth/react/-/react-" + devVersion + ".tar.gz",
		"/npm/@moth/react/-/other-" + devVersion + ".tgz",
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

// TestNpmInstall resolves and downloads @moth/react from a running moth
// handler with the real npm client — the milestone's acceptance criterion.
// It needs npm and network access to registry.npmjs.org for the transitive
// dependencies, so it skips when npm is absent or -short is set. npm
// follows dist.tarball from the packument, which is why the server's
// BaseURL points at the address actually listened on.
func TestNpmInstall(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: skipped in -short mode")
	}
	npm, err := exec.LookPath("npm")
	if err != nil {
		t.Skip("integration test: npm not on PATH")
	}

	// The npm client follows dist.tarball from the packument, so BaseURL
	// must match the address actually listened on: listen first, then
	// assemble.
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

	// A minimal consumer project routing only the @moth scope at this
	// instance; everything else (transitive deps, the react peer) still
	// resolves from the default registry.
	app := t.TempDir()
	if err := os.WriteFile(filepath.Join(app, "package.json"),
		[]byte(`{"name":"npmtest","private":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, ".npmrc"),
		[]byte("@moth:registry="+baseURL+"/npm\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	// Pin the exact served version (a dev build serves a 0.0.0-dev.*
	// pre-release, which ranges would not match).
	cmd := exec.CommandContext(ctx, npm, "install",
		"@moth/react@"+srv.npm.version, "--no-audit", "--no-fund")
	cmd.Dir = app
	// A private npm cache keeps the developer's real cache clean (npm
	// caches tarballs by integrity, so each dev tree would otherwise leave
	// an entry behind).
	cmd.Env = append(os.Environ(),
		"npm_config_cache="+filepath.Join(t.TempDir(), "npm-cache"),
		"npm_config_update_notifier=false")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("npm install: %v\n%s", err, out)
	}

	// The installed package really is ours, with the stamped version.
	raw, err := os.ReadFile(filepath.Join(app, "node_modules", "@moth", "react", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pkg struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.Name != "@moth/react" || pkg.Version != srv.npm.version {
		t.Fatalf("installed package.json: name %q version %q, want version %q",
			pkg.Name, pkg.Version, srv.npm.version)
	}
	if _, err := os.Stat(filepath.Join(app, "node_modules", "@moth", "react", "dist", "index.js")); err != nil {
		t.Fatalf("installed package misses dist/index.js: %v", err)
	}
}

func fetchNpmTarball(t *testing.T, e *testEnv) []byte {
	t.Helper()
	resp, err := e.client.Get(e.url + "/npm/@moth/react/-/react-" + devNpmVersion(t) + ".tgz")
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
