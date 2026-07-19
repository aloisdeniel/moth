package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"

	"github.com/aloisdeniel/moth/internal/server/rpc/auth"
	billingrpc "github.com/aloisdeniel/moth/internal/server/rpc/billing"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/version"
)

// Per-project pub repository (/p/{slug}/pub): each project serves its own
// preconfigured build of the identically-named moth_auth package (plus the
// companions), with the endpoint, publishable key and public config baked in
// so the SDK needs no manual configuration and no initial request. Everything
// served is public — the public config and the publishable key are designed
// to ship in the app — so these endpoints need no credential, exactly like the
// generic /pub.
//
// Archives are built on demand (projects × config revisions is unbounded, so
// they cannot be prebuilt at startup like the generic set) and cached per slug
// at the current revision; a config edit changes the version and rebuilds.

// projectPubCache memoises the built package set for a project at its current
// config revision, bounded by an LRU cap. A config edit changes the derived
// version, which misses the cache and rebuilds.
type projectPubCache struct {
	mu    sync.Mutex
	max   int
	tick  uint64
	slots map[string]*projectPubSet // keyed by slug
}

type projectPubSet struct {
	version  string
	archives map[string]*pubArchive
	used     uint64
}

func newProjectPubCache(max int) *projectPubCache {
	return &projectPubCache{max: max, slots: make(map[string]*projectPubSet)}
}

// revSignature is the config fingerprint folded into the package version, so a
// theme, copy or paywall edit bumps it. Empty ids (never-customized documents)
// use the same "default" sentinel the SDK-facing revisions use.
func revSignature(p store.Project) string {
	def := func(s string) string {
		if s == "" {
			return "default"
		}
		return s
	}
	return def(p.ThemeRevisionID) + "|" + def(p.CopyRevisionID) + "|" + def(p.PaywallRevisionID)
}

// projectArchives returns the built package set for a project, building and
// caching it on a miss (or when the project's config revision has changed).
// Returns store.ErrNotFound for an unknown slug.
func (s *Server) projectArchives(ctx context.Context, slug string) (*projectPubSet, error) {
	p, err := s.store.GetProjectBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimSuffix(s.cfg.BaseURL, "/")
	pubURL := baseURL + "/p/" + p.Slug + "/pub"

	treeHash, err := sdkTreeHash()
	if err != nil {
		return nil, err
	}
	base, err := pubVersion(version.Version, treeHash)
	if err != nil {
		return nil, err
	}
	ver := pubVersionForProject(base, pubURL, revSignature(p))

	// Fast path: cached at this exact version.
	s.projectPub.mu.Lock()
	if set := s.projectPub.slots[slug]; set != nil && set.version == ver {
		s.projectPub.tick++
		set.used = s.projectPub.tick
		s.projectPub.mu.Unlock()
		return set, nil
	}
	s.projectPub.mu.Unlock()

	// Build outside the lock (CPU-bound; other projects should not block).
	inj, err := s.projectGenConfig(ctx, p, baseURL)
	if err != nil {
		return nil, err
	}
	archives := make(map[string]*pubArchive, len(pubPackages))
	for _, spec := range pubPackages {
		a, err := buildPubArchive(spec, ver, pubURL, inj)
		if err != nil {
			return nil, fmt.Errorf("build %s pub archive for project %s: %w", spec.name, slug, err)
		}
		archives[spec.name] = a
	}
	set := &projectPubSet{version: ver, archives: archives}

	s.projectPub.mu.Lock()
	defer s.projectPub.mu.Unlock()
	// Another request may have built the same version concurrently; prefer the
	// existing entry so callers share bytes.
	if existing := s.projectPub.slots[slug]; existing != nil && existing.version == ver {
		s.projectPub.tick++
		existing.used = s.projectPub.tick
		return existing, nil
	}
	s.projectPub.tick++
	set.used = s.projectPub.tick
	s.projectPub.slots[slug] = set
	s.projectPub.evictLocked()
	return set, nil
}

// evictLocked drops the least-recently-used slots until the cache is within
// its cap. Caller holds mu.
func (c *projectPubCache) evictLocked() {
	for len(c.slots) > c.max {
		var oldestKey string
		var oldest uint64
		first := true
		for k, v := range c.slots {
			if first || v.used < oldest {
				oldestKey, oldest, first = k, v.used, false
			}
		}
		delete(c.slots, oldestKey)
	}
}

// projectGenConfig assembles the per-project injection: the endpoint, the
// publishable key, and base64 seeds of the public config (moth.auth.v1
// GetProjectConfigResponse) and paywall (moth.billing.v1 Paywall) the SDK
// decodes at startup to render offline with no round-trip.
func (s *Server) projectGenConfig(ctx context.Context, p store.Project, baseURL string) (*genConfigInjection, error) {
	// nil header negotiates the project default locale — the bundled floor.
	cfg, _, err := authrpc.BuildProjectConfigResponse(ctx, s.store, p, baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("assemble project config: %w", err)
	}
	cfgBytes, err := proto.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal project config: %w", err)
	}
	pwBytes, err := proto.Marshal(billingrpc.PublicPaywall(p))
	if err != nil {
		return nil, fmt.Errorf("marshal project paywall: %w", err)
	}
	return &genConfigInjection{
		endpoint:       baseURL,
		publishableKey: p.PublishableKey,
		configB64:      base64.StdEncoding.EncodeToString(cfgBytes),
		paywallB64:     base64.StdEncoding.EncodeToString(pwBytes),
	}, nil
}

// handlePubProjectVersions implements the per-project version listing:
// GET /p/{slug}/pub/api/packages/{package}. Exactly one version is served per
// package — the project's own current build.
func (s *Server) handlePubProjectVersions(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	set, err := s.projectArchives(r.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		s.internalError(w, r, err)
		return
	}
	a, ok := set.archives[r.PathValue("package")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	baseURL := strings.TrimSuffix(s.cfg.BaseURL, "/")
	archiveURL := baseURL + "/p/" + slug + "/pub/packages/" + a.name + "/versions/" + a.version + ".tar.gz"
	ver := map[string]any{
		"version":        a.version,
		"pubspec":        a.pubspec,
		"archive_url":    archiveURL,
		"archive_sha256": a.sha256,
	}
	w.Header().Set("Content-Type", pubContentType)
	json.NewEncoder(w).Encode(map[string]any{
		"name":     a.name,
		"latest":   ver,
		"versions": []any{ver},
	})
}

// handlePubProjectArchive serves a per-project package tarball:
// GET /p/{slug}/pub/packages/{package}/versions/{version}.tar.gz.
func (s *Server) handlePubProjectArchive(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	set, err := s.projectArchives(r.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		s.internalError(w, r, err)
		return
	}
	a, ok := set.archives[r.PathValue("package")]
	if !ok || r.PathValue("file") != a.version+".tar.gz" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(a.tarball)
}
