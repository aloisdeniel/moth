package server

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/aloisdeniel/moth/sdk"
)

// npmPackageName is the only package this instance serves; the @moth/react
// SDK ships inside the binary as a committed build and its version tracks
// the moth release (plan/18, "Package serving").
const npmPackageName = "@moth/react"

var (
	// packageJSONVersionLine matches the version field the archive builder
	// rewrites to the moth build version.
	packageJSONVersionLine = regexp.MustCompile(`(?m)^  "version": "[^"]*",$`)
	// versionJSLine matches the mothSdkVersion constant the archive builder
	// rewrites alongside package.json, so the version the SDK reports in
	// x-moth-sdk-version metadata matches the served package.
	versionJSLine = regexp.MustCompile(`(?m)^export const mothSdkVersion = '[^']*';$`)
	// versionDTSLine matches the same placeholder in the generated type
	// declaration, stamped when present so the types mirror the runtime.
	versionDTSLine = regexp.MustCompile(`(?m)^export declare const mothSdkVersion = "[^"]*";$`)
)

// npmArchive is the single served version of the embedded npm package,
// built once at startup so the integrity hashes in the packument always
// match the served bytes.
type npmArchive struct {
	version   string
	pkg       map[string]any // decoded (rewritten) package.json, for the packument
	tarball   []byte
	shasum    string // hex sha1 of tarball (npm "dist.shasum")
	integrity string // "sha512-" + base64 of tarball (npm "dist.integrity")
}

// reactTreeHash hashes the embedded React SDK tree (paths and contents) so
// dev builds get a package version unique to the tree they serve — npm
// caches tarballs by integrity, so a fixed dev version would trip
// content-hash mismatches as the tree changes (same rationale as
// sdkTreeHash for /pub).
func reactTreeHash() (string, error) {
	h := sha256.New()
	err := fs.WalkDir(sdk.FS, "react", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := sdk.FS.ReadFile(p)
		if err != nil {
			return err
		}
		// hash.Hash writes never fail.
		_, _ = fmt.Fprintf(h, "%s\x00%d\x00", p, len(raw))
		_, _ = h.Write(raw)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("hash embedded react SDK tree: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// buildNpmArchive assembles the @moth/react package tarball from the
// embedded sdk/react tree, stamping the package.json version and the
// mothSdkVersion constant with the moth build version. The output is
// deterministic for a given binary: fs.WalkDir yields lexical order and
// every tar header carries fixed metadata.
func buildNpmArchive(mothVersion string) (*npmArchive, error) {
	treeHash, err := reactTreeHash()
	if err != nil {
		return nil, err
	}
	// The moth build version maps to npm semver exactly as it does to Dart
	// semver: releases drop the "v" prefix, dev builds become a tree-hash
	// pre-release.
	ver, err := pubVersion(mothVersion, treeHash)
	if err != nil {
		return nil, err
	}
	a := &npmArchive{version: ver}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	err = fs.WalkDir(sdk.FS, "react", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := sdk.FS.ReadFile(p)
		if err != nil {
			return err
		}
		// npm tarballs are rooted at a mandatory "package/" directory
		// (clients strip one leading path component).
		name := "package/" + strings.TrimPrefix(p, "react/")
		switch name {
		case "package/package.json":
			if !packageJSONVersionLine.Match(raw) {
				return fmt.Errorf("embedded package.json has no version field")
			}
			raw = packageJSONVersionLine.ReplaceAll(raw,
				[]byte(`  "version": "`+a.version+`",`))
			if err := json.Unmarshal(raw, &a.pkg); err != nil {
				return fmt.Errorf("decode package.json: %w", err)
			}
		case "package/dist/version.js":
			// The runtime constant the SDK reports in x-moth-sdk-version
			// metadata must match the stamped package.json version.
			if !versionJSLine.Match(raw) {
				return fmt.Errorf("embedded dist/version.js has no mothSdkVersion constant")
			}
			raw = versionJSLine.ReplaceAll(raw,
				[]byte("export const mothSdkVersion = '"+a.version+"';"))
		case "package/dist/version.d.ts":
			// The declared literal type carries the same placeholder; stamp
			// it when present so the types mirror the runtime constant.
			raw = versionDTSLine.ReplaceAll(raw,
				[]byte(`export declare const mothSdkVersion = "`+a.version+`";`))
		}
		hdr := &tar.Header{
			Name:    name,
			Mode:    0o644,
			Size:    int64(len(raw)),
			ModTime: time.Unix(0, 0), // fixed: archive bytes depend only on content
			Format:  tar.FormatUSTAR,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		_, err = tw.Write(raw)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("build npm archive: %w", err)
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	if a.pkg == nil {
		return nil, fmt.Errorf("embedded react SDK tree has no package.json")
	}

	a.tarball = buf.Bytes()
	sum512 := sha512.Sum512(a.tarball)
	a.integrity = "sha512-" + base64.StdEncoding.EncodeToString(sum512[:])
	sum1 := sha1.Sum(a.tarball)
	a.shasum = hex.EncodeToString(sum1[:])
	return a, nil
}

// tarballName is the conventional npm tarball basename (unscoped name).
func (a *npmArchive) tarballName() string {
	return "react-" + a.version + ".tgz"
}

// tarballPath is the URL path of the tarball, relative to the base URL.
func (a *npmArchive) tarballPath() string {
	return "/npm/" + npmPackageName + "/-/" + a.tarballName()
}

// handleNpmPackument implements GET /npm/@moth/react: the packument the npm
// client resolves against. Exactly one version is served — the binary's
// own. It is registered on two routes because clients disagree on encoding
// the scoped slash: npm and pnpm request /npm/@moth%2freact (one escaped
// segment, matched by the {pkg} wildcard), yarn and bun request the literal
// two-segment /npm/@moth/react (matched exactly, no path value).
func (s *Server) handleNpmPackument(w http.ResponseWriter, r *http.Request) {
	if pkg := r.PathValue("pkg"); pkg != "" && pkg != npmPackageName {
		http.NotFound(w, r)
		return
	}
	version := maps.Clone(s.npm.pkg)
	version["dist"] = map[string]any{
		"tarball":   strings.TrimSuffix(s.cfg.BaseURL, "/") + s.npm.tarballPath(),
		"shasum":    s.npm.shasum,
		"integrity": s.npm.integrity,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"name":      npmPackageName,
		"dist-tags": map[string]any{"latest": s.npm.version},
		"versions":  map[string]any{s.npm.version: version},
	})
}

// handleNpmTarball serves the package tarball at
// GET /npm/@moth/react/-/react-{version}.tgz.
func (s *Server) handleNpmTarball(w http.ResponseWriter, r *http.Request) {
	if r.PathValue("file") != s.npm.tarballName() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(s.npm.tarball)
}
