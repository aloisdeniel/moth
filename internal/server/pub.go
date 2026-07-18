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

// pubPackageSpec describes one Flutter package this instance serves from the
// embedded sdk/ tree; the packages ship inside the binary and their versions
// track the moth release (plan/05, "Package serving"; plan/19 grows the set).
type pubPackageSpec struct {
	name string // pub package name
	dir  string // package directory under sdk.FS
	// versionDart marks packages carrying a lib/src/version.dart
	// mothSdkVersion constant that must be stamped alongside the pubspec.
	versionDart bool
	// mothAuthDep marks companion packages declaring the placeholder
	// moth_auth hosted dependency the archive builder rewrites to this
	// instance's /pub URL and stamped version.
	mothAuthDep bool
}

// pubPackages is the served package set. Adding a companion means embedding
// its publishable subset in sdk/embed.go and appending a spec here.
var pubPackages = []pubPackageSpec{
	{name: "moth_auth", dir: "flutter", versionDart: true},
	{name: "moth_billing", dir: "flutter_billing", mothAuthDep: true},
	{name: "moth_push", dir: "flutter_push", mothAuthDep: true},
}

// pubContentType is the media type of the pub hosted repository API v2.
const pubContentType = "application/vnd.pub.v2+json"

// mothAuthDepPlaceholder is the exact hosted-dependency block companion
// packages declare in their pubspec; buildPubArchives rewrites it so the
// dependency resolves against the serving instance at the stamped version.
// Local development points at ../flutter via pubspec_overrides.yaml instead,
// which is never embedded.
const mothAuthDepPlaceholder = "  moth_auth:\n" +
	"    hosted: https://moth.invalid/pub\n" +
	"    version: 0.1.0\n"

var (
	// pubspecVersionLine matches the top-level version field the archive
	// builder rewrites to the moth build version (anchored at column zero,
	// so an indented dependency version pin never matches).
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

// pubArchive is the single served version of one embedded SDK package,
// built once at startup so the sha256 in the version listing always matches
// the served bytes.
type pubArchive struct {
	name    string
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
		return "", fmt.Errorf("build version %q is not a valid semver; the pub packages cannot be stamped (use VERSION=vX.Y.Z or VERSION=dev)", mothVersion)
	}
	return v, nil
}

// pubVersionForURL appends a build-metadata identifier derived from pubURL
// to the stamped version. Companion tarballs bake the instance's /pub URL
// into their moth_auth hosted dependency, so the served bytes are a function
// of the URL as well as the trees — and the pub client caches archives by
// (host, name, version) and pins archive_sha256 in pubspec.lock, so the same
// (name, version) must always mean the same bytes even when base_url changes
// while the old host stays reachable (http→https, a www/apex alias, an
// internal hostname behind the same proxy). The whole package set gets the
// suffix: companions pin moth_auth's exact stamped version, so the versions
// must stay identical across packages.
func pubVersionForURL(version, pubURL string) string {
	sum := sha256.Sum256([]byte(pubURL))
	id := "u" + hex.EncodeToString(sum[:4])
	if strings.Contains(version, "+") {
		// The build version already carries metadata (VERSION=v1.2.3+42):
		// append a further dot-separated identifier, never a second '+'.
		return version + "." + id
	}
	return version + "+" + id
}

// sdkTreeHash hashes every embedded pub package tree (paths and contents) so
// dev builds get a package version unique to the trees they serve. One hash
// covers the whole set: companion packages pin moth_auth's exact stamped
// version inside their tarball, so a moth_auth change must bump every
// package's dev version to keep pub's per-(name, version) cache coherent.
func sdkTreeHash() (string, error) {
	h := sha256.New()
	for _, spec := range pubPackages {
		err := fs.WalkDir(sdk.FS, spec.dir, func(p string, d fs.DirEntry, err error) error {
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
			return "", fmt.Errorf("hash embedded SDK tree %s: %w", spec.dir, err)
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// buildPubArchives assembles the tarball for every package in pubPackages
// from the embedded sdk/ tree, stamping each with the same version derived
// from the moth build version and the instance's /pub URL
// (pubVersionForURL). baseURL is this instance's public base URL
// (config BaseURL, the same root the version listing derives archive_url
// from): it is baked into companion pubspecs so their moth_auth dependency
// resolves against this instance.
func buildPubArchives(mothVersion, baseURL string) (map[string]*pubArchive, error) {
	treeHash, err := sdkTreeHash()
	if err != nil {
		return nil, err
	}
	ver, err := pubVersion(mothVersion, treeHash)
	if err != nil {
		return nil, err
	}
	pubURL := strings.TrimSuffix(baseURL, "/") + "/pub"
	// The URL is baked into companion tarballs, so it must be part of the
	// version too — (name, version) → bytes stays injective across base_url
	// changes.
	ver = pubVersionForURL(ver, pubURL)
	archives := make(map[string]*pubArchive, len(pubPackages))
	for _, spec := range pubPackages {
		a, err := buildPubArchive(spec, ver, pubURL)
		if err != nil {
			return nil, fmt.Errorf("build %s pub archive: %w", spec.name, err)
		}
		archives[spec.name] = a
	}
	return archives, nil
}

// buildPubArchive assembles one package tarball, rewriting the pubspec
// version (and, per the spec, the mothSdkVersion Dart constant and the
// moth_auth hosted-dependency placeholder). The output is deterministic for
// a given binary and base URL: fs.WalkDir yields lexical order and every
// tar header carries fixed metadata.
func buildPubArchive(spec pubPackageSpec, version, pubURL string) (*pubArchive, error) {
	a := &pubArchive{name: spec.name, version: version}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	err := fs.WalkDir(sdk.FS, spec.dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := sdk.FS.ReadFile(p)
		if err != nil {
			return err
		}
		// pub archives are rooted at the package directory.
		name := strings.TrimPrefix(p, spec.dir+"/")
		if name == "pubspec.yaml" {
			if !pubspecVersionLine.Match(raw) {
				return fmt.Errorf("embedded pubspec.yaml has no version field")
			}
			raw = pubspecVersionLine.ReplaceAll(raw, []byte("version: "+a.version))
			if spec.mothAuthDep {
				// The companion must resolve the exact moth_auth this
				// instance serves — a missing placeholder means the two can
				// drift, so fail the build loudly.
				if !bytes.Contains(raw, []byte(mothAuthDepPlaceholder)) {
					return fmt.Errorf("embedded pubspec.yaml has no moth_auth hosted-dependency placeholder")
				}
				dep := "  moth_auth:\n" +
					"    hosted: " + pubURL + "\n" +
					"    version: " + a.version + "\n"
				raw = bytes.ReplaceAll(raw, []byte(mothAuthDepPlaceholder), []byte(dep))
			}
			if err := yaml.Unmarshal(raw, &a.pubspec); err != nil {
				return fmt.Errorf("decode pubspec.yaml: %w", err)
			}
		}
		if spec.versionDart && name == "lib/src/version.dart" {
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
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	if a.pubspec == nil {
		return nil, fmt.Errorf("embedded tree has no pubspec.yaml")
	}

	a.tarball = buf.Bytes()
	sum := sha256.Sum256(a.tarball)
	a.sha256 = hex.EncodeToString(sum[:])
	return a, nil
}

// archivePath is the URL path of the tarball, relative to the base URL.
func (a *pubArchive) archivePath() string {
	return "/pub/packages/" + a.name + "/versions/" + a.version + ".tar.gz"
}

// handlePubVersions implements GET /pub/api/packages/{package} from the pub
// hosted repository API v2: the version listing the `dart pub` client
// resolves against. Exactly one version is served per package — the
// binary's own.
func (s *Server) handlePubVersions(w http.ResponseWriter, r *http.Request) {
	a, ok := s.pub[r.PathValue("package")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	version := map[string]any{
		"version":        a.version,
		"pubspec":        a.pubspec,
		"archive_url":    strings.TrimSuffix(s.cfg.BaseURL, "/") + a.archivePath(),
		"archive_sha256": a.sha256,
	}
	w.Header().Set("Content-Type", pubContentType)
	json.NewEncoder(w).Encode(map[string]any{
		"name":     a.name,
		"latest":   version,
		"versions": []any{version},
	})
}

// handlePubArchive serves the package tarball at
// GET /pub/packages/{package}/versions/{version}.tar.gz.
func (s *Server) handlePubArchive(w http.ResponseWriter, r *http.Request) {
	a, ok := s.pub[r.PathValue("package")]
	if !ok || r.PathValue("file") != a.version+".tar.gz" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(a.tarball)
}
