package server

import (
	"encoding/csv"
	"net/http"
	"strings"
	"time"

	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

// auditExportMaxRows caps a single CSV export so an unbounded log cannot pin a
// request streaming the whole table; ample for operator review and well under
// any spreadsheet's row limit.
const auditExportMaxRows = 100_000

// auditExportPage is the keyset page size used to stream the export.
const auditExportPage = 1000

// handleExportAudit streams the audit log as a CSV download:
// GET /admin/export/audit.csv[?project_id=&actor_id=&action=&from=&to=].
// Like the stats export it rides plain HTTP (a browser download is not an
// RPC) and is authenticated by the same admin credential as the RPCs — either
// the session cookie or a personal access token.
func (s *Server) handleExportAudit(w http.ResponseWriter, r *http.Request) {
	if !s.adminHTTPAuthed(r) {
		writeJSONError(w, http.StatusUnauthorized, "admin session or personal access token required")
		return
	}
	q := r.URL.Query()
	filter := store.AuditFilter{
		ProjectID: q.Get("project_id"),
		ActorID:   q.Get("actor_id"),
		Action:    q.Get("action"),
		Limit:     auditExportPage,
	}
	if v := q.Get("from"); v != "" {
		t, err := parseAuditBound(v)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid from timestamp")
			return
		}
		filter.From = t
	}
	if v := q.Get("to"); v != "" {
		t, err := parseAuditBound(v)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid to timestamp")
			return
		}
		filter.To = t
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=audit-log.csv")

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id", "create_time", "actor_type", "actor_id",
		"actor_label", "action", "target_type", "target_id", "project_id",
		"ip", "summary"})

	written := 0
	for written < auditExportMaxRows {
		entries, err := s.store.ListAudit(r.Context(), filter)
		if err != nil {
			// Headers are already sent; log and stop rather than write a
			// misleading trailer.
			s.log.Error("stream audit csv", "error", err.Error())
			break
		}
		if len(entries) == 0 {
			break
		}
		for _, e := range entries {
			_ = cw.Write([]string{
				csvSafe(e.ID),
				e.CreatedAt.UTC().Format(time.RFC3339),
				csvSafe(e.ActorType), csvSafe(e.ActorID), csvSafe(e.ActorLabel),
				csvSafe(e.Action), csvSafe(e.TargetType), csvSafe(e.TargetID),
				csvSafe(e.ProjectID), csvSafe(e.IP), csvSafe(e.Summary),
			})
			written++
			if written >= auditExportMaxRows {
				break
			}
		}
		if len(entries) < filter.Limit {
			break
		}
		// Keyset-page from the oldest id of this batch (ListAudit is
		// newest-first, so the last row is the oldest).
		filter.AfterID = entries[len(entries)-1].ID
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		s.log.Error("flush audit csv", "error", err.Error())
	}
}

// parseAuditBound accepts a full RFC3339 timestamp or a bare YYYY-MM-DD date.
func parseAuditBound(v string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", v)
}

// csvSafe defuses spreadsheet formula injection: a field a spreadsheet would
// evaluate as a formula (leading =, +, -, @ or a control character) is
// prefixed with a single quote so it is imported as literal text. Audit
// summaries and labels contain user- and admin-controlled strings (emails,
// project names), so every cell is passed through this.
func csvSafe(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}

// adminHTTPAuthed reports whether the request carries a live admin credential:
// the session cookie or a usable personal access token. Mirrors the admin RPC
// interceptor for the plain-HTTP download endpoints.
func (s *Server) adminHTTPAuthed(r *http.Request) bool {
	if s.adminAuthed(r) {
		return true
	}
	auth := r.Header.Get("Authorization")
	const scheme = "Bearer "
	if len(auth) <= len(scheme) || !strings.EqualFold(auth[:len(scheme)], scheme) {
		return false
	}
	tok := strings.TrimSpace(auth[len(scheme):])
	if !strings.HasPrefix(tok, token.PATPrefix) {
		return false
	}
	pat, err := s.store.GetPATByHash(r.Context(), token.Hash(tok))
	if err != nil {
		return false
	}
	return pat.Usable(time.Now())
}
