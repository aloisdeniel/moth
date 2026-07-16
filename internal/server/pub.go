package server

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/aloisdeniel/moth/sdk"
)

// pubPackageName is the only package this instance serves; the moth_auth
// Flutter SDK ships inside the binary and its version tracks the moth
// release (plan/05, "Package serving").
const pubPackageName = "moth_auth"

// pubContentType is the media type of the pub hosted repository API v2.
const pubContentType = "application/vnd.pub.v2+json"

var (
	// pubspecVersionLine matches the version field the archive builder
	// rewrites to the moth build version.
	pubspecVersionLine = regexp.MustCompile(`(?m)^version: .+$`)
	// versionDartLine matches the mothSdkVersion constant the archive
	// builder rewrites alongside the pubspec, so the version the SDK
	// reports in x-moth-sdk-version metadata matches the served package.
	versionDartLine = regexp.MustCompile(`(?m)^const String mothSdkVersion = '[^']*';$`)
	// semverRe accepts what Dart's pub_semver calls a version:
	// major.minor.patch with optional pre-release and/or build suffix
	// (1.2.3, 1.2.3-rc.1, 1.2.3+42, 1.2.3-rc.1+42).
	semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)
)

// pubArchive is the single served version of the embedded SDK package,
// built once at startup so the sha256 in the version listing always matches
// the served bytes.
type pubArchive struct {
	version string
	pubspec map[string]any // decoded (rewritten) pubspec, for the listing
	tarball []byte
	sha256  string // hex of tarball
}

// pubVersion maps the moth build version to a valid Dart semver: releases
// drop the "v" prefix (v1.2.3 → 1.2.3). Dev builds (no release ldflags)
// become a pre-release derived from the embedded SDK tree hash — the pub
// client caches hosted packages by (host, name, version) and records the
// archive sha256 in pubspec.lock, so a fixed dev version would serve stale
// cached SDKs and trip content-hash mismatches as the tree changes. Any
// other version is a mis-tagged release and is rejected instead of being
// silently mis-stamped.
func pubVersion(mothVersion, treeHash string) (string, error) {
	if mothVersion == "dev" {
		// The "h" prefix keeps the identifier alphanumeric even when the
		// hash prefix happens to be all digits (numeric semver identifiers
		// must not have leading zeros).
		return "0.0.0-dev.h" + treeHash[:8], nil
	}
	v := strings.TrimPrefix(mothVersion, "v")
	if !semverRe.MatchString(v) {
		return "", fmt.Errorf("build version %q is not a valid semver; the moth_auth package cannot be stamped (use VERSION=vX.Y.Z or VERSION=dev)", mothVersion)
	}
	return v, nil
}

// sdkTreeHash hashes the embedded SDK tree (paths and contents) so dev
// builds get a package version unique to the tree they serve.
func sdkTreeHash() (string, error) {
	h := sha256.New()
	err := fs.WalkDir(sdk.FS, "flutter", func(p string, d fs.DirEntry, err error) error {
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
		return "", fmt.Errorf("hash embedded SDK tree: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// buildPubArchive assembles the moth_auth package tarball from the embedded
// sdk/flutter tree, stamping the pubspec version and the mothSdkVersion
// Dart constant with the moth build version. The output is deterministic
// for a given binary: fs.WalkDir yields lexical order and every tar header
// carries fixed metadata.
func buildPubArchive(mothVersion string) (*pubArchive, error) {
	treeHash, err := sdkTreeHash()
	if err != nil {
		return nil, err
	}
	ver, err := pubVersion(mothVersion, treeHash)
	if err != nil {
		return nil, err
	}
	a := &pubArchive{version: ver}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	err = fs.WalkDir(sdk.FS, "flutter", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := sdk.FS.ReadFile(p)
		if err != nil {
			return err
		}
		// pub archives are rooted at the package directory.
		name := strings.TrimPrefix(p, "flutter/")
		if name == "pubspec.yaml" {
			if !pubspecVersionLine.Match(raw) {
				return fmt.Errorf("embedded pubspec.yaml has no version field")
			}
			raw = pubspecVersionLine.ReplaceAll(raw, []byte("version: "+a.version))
			if err := yaml.Unmarshal(raw, &a.pubspec); err != nil {
				return fmt.Errorf("decode pubspec.yaml: %w", err)
			}
		}
		if name == "lib/src/version.dart" {
			// The runtime constant the SDK reports in x-moth-sdk-version
			// metadata must match the stamped pubspec version.
			if !versionDartLine.Match(raw) {
				return fmt.Errorf("embedded lib/src/version.dart has no mothSdkVersion constant")
			}
			raw = versionDartLine.ReplaceAll(raw,
				[]byte("const String mothSdkVersion = '"+a.version+"';"))
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
		return nil, fmt.Errorf("build pub archive: %w", err)
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	if a.pubspec == nil {
		return nil, fmt.Errorf("embedded SDK tree has no pubspec.yaml")
	}

	a.tarball = buf.Bytes()
	sum := sha256.Sum256(a.tarball)
	a.sha256 = hex.EncodeToString(sum[:])
	return a, nil
}

// archivePath is the URL path of the tarball, relative to the base URL.
func (a *pubArchive) archivePath() string {
	return "/pub/packages/" + pubPackageName + "/versions/" + a.version + ".tar.gz"
}

// handlePubVersions implements GET /pub/api/packages/{package} from the pub
// hosted repository API v2: the version listing the `dart pub` client
// resolves against. Exactly one version is served — the binary's own.
func (s *Server) handlePubVersions(w http.ResponseWriter, r *http.Request) {
	if r.PathValue("package") != pubPackageName {
		http.NotFound(w, r)
		return
	}
	version := map[string]any{
		"version":        s.pub.version,
		"pubspec":        s.pub.pubspec,
		"archive_url":    strings.TrimSuffix(s.cfg.BaseURL, "/") + s.pub.archivePath(),
		"archive_sha256": s.pub.sha256,
	}
	w.Header().Set("Content-Type", pubContentType)
	json.NewEncoder(w).Encode(map[string]any{
		"name":     pubPackageName,
		"latest":   version,
		"versions": []any{version},
	})
}

// handlePubArchive serves the package tarball at
// GET /pub/packages/{package}/versions/{version}.tar.gz.
func (s *Server) handlePubArchive(w http.ResponseWriter, r *http.Request) {
	if r.PathValue("package") != pubPackageName ||
		r.PathValue("file") != s.pub.version+".tar.gz" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(s.pub.tarball)
}
