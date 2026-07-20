package server

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// The public /assets/ surface serves the binary halves of a project theme —
// things that don't belong in RPC responses:
//
//	/assets/fonts/{...}          embedded font binaries and their licenses
//	/assets/{projectID}/{file}   uploaded logos (allowlisted names only)
//
// Logo URLs handed to clients are revision-keyed (?v={themeRevision}), so
// the files can be served immutable: any theme change mints a new revision
// id and with it a new URL, and stale caches never survive an edit.

// logoAssets maps every servable uploaded-asset filename to its content
// type. Requests for anything else 404 — combined with the UUID check on
// the directory segment, path traversal has nothing to grab.
var logoAssets = map[string]string{
	"logo-light.png": "image/png",
	"logo-light.svg": "image/svg+xml",
	"logo-dark.png":  "image/png",
	"logo-dark.svg":  "image/svg+xml",
}

func (s *Server) handleAsset(w http.ResponseWriter, r *http.Request) {
	rest := r.PathValue("path")
	if strings.HasPrefix(rest, "fonts/") {
		s.fonts.ServeHTTP(w, r)
		return
	}
	projectID, file, found := strings.Cut(rest, "/")
	ct, servable := logoAssets[file]
	if !found || !servable {
		http.NotFound(w, r)
		return
	}
	// Upload directories are keyed by server-generated project UUIDs;
	// anything else cannot name a servable file.
	if _, err := uuid.Parse(projectID); err != nil {
		http.NotFound(w, r)
		return
	}
	// Both path segments are constrained above: projectID must parse as a
	// UUID and file must be a key in logoAssets, so neither can carry a
	// traversal payload. gosec's taint analysis can't see those guards.
	raw, err := os.ReadFile(filepath.Join(s.uploads, projectID, file)) // #nosec G304,G703 -- projectID is a parsed UUID, file is allowlisted
	if errors.Is(err, fs.ErrNotExist) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	sum := sha256.Sum256(raw)
	etag := `"` + hex.EncodeToString(sum[:16]) + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Type", ct)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// Uploaded SVGs are sanitized at upload time; a CSP on the response
	// keeps even a hypothetical leftover script inert when the file is
	// opened directly.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	if etagMatch(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Write(raw)
}

// etagMatch reports whether an If-None-Match header value matches etag (a
// quoted strong validator) under RFC 9110 semantics: "*" matches any
// stored response, the value is a comma-separated list, and comparison is
// weak — a W/ prefix (a proxy may have weakened the validator, e.g. after
// re-coding the response) is ignored.
func etagMatch(header, etag string) bool {
	if strings.TrimSpace(header) == "*" {
		return true
	}
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		candidate = strings.TrimPrefix(candidate, "W/")
		if candidate == etag {
			return true
		}
	}
	return false
}
