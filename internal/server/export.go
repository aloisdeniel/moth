package server

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	adminrpc "github.com/aloisdeniel/moth/internal/server/rpc/admin"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

// Export range: same day format and bounds as AnalyticsService.GetStats.
const (
	exportDateFormat   = "2006-01-02"
	defaultExportDays  = 30
	maxExportRangeDays = 366
)

// handleExportStats streams a project's daily_stats as a CSV download:
// GET /admin/export/stats.csv?project_id=ID[&from=YYYY-MM-DD][&to=YYYY-MM-DD].
// File downloads are a browser affordance, not an RPC, so this rides plain
// HTTP — authenticated by the same admin session cookie as the RPCs.
func (s *Server) handleExportStats(w http.ResponseWriter, r *http.Request) {
	if !s.adminAuthed(r) {
		writeJSONError(w, http.StatusUnauthorized, "admin session required")
		return
	}
	q := r.URL.Query()
	project, err := s.store.GetProject(r.Context(), q.Get("project_id"))
	if errors.Is(err, store.ErrNotFound) {
		writeJSONError(w, http.StatusNotFound, "unknown project")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	// Default: the last 30 completed days in the project's rollup timezone.
	localNow := time.Now().In(project.Settings.RollupLocation())
	to := time.Date(localNow.Year(), localNow.Month(), localNow.Day()-1, 0, 0, 0, 0, time.UTC)
	if v := q.Get("to"); v != "" {
		if to, err = time.Parse(exportDateFormat, v); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid to date")
			return
		}
	}
	from := to.AddDate(0, 0, -(defaultExportDays - 1))
	if v := q.Get("from"); v != "" {
		if from, err = time.Parse(exportDateFormat, v); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid from date")
			return
		}
	}
	if from.After(to) {
		writeJSONError(w, http.StatusBadRequest, "from is after to")
		return
	}
	if to.Sub(from) > maxExportRangeDays*24*time.Hour {
		writeJSONError(w, http.StatusBadRequest, "range longer than 366 days")
		return
	}

	rows, err := s.store.GetDailyStats(r.Context(), project.ID,
		from.Format(exportDateFormat), to.Format(exportDateFormat))
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	byDate := make(map[string]store.DailyStats, len(rows))
	for _, row := range rows {
		byDate[row.Date] = row
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=stats-%s-%s-%s.csv",
			project.Slug, from.Format(exportDateFormat), to.Format(exportDateFormat)))

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"date", "signups", "logins", "dau", "failures",
		"logins_password", "logins_google", "logins_apple",
		"platform_ios", "platform_android", "platform_web", "platform_other"})
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		row := byDate[day.Format(exportDateFormat)]
		_ = cw.Write([]string{
			day.Format(exportDateFormat),
			strconv.Itoa(row.Signups), strconv.Itoa(row.Logins),
			strconv.Itoa(row.DAU), strconv.Itoa(row.Failures),
			strconv.Itoa(row.LoginsPassword), strconv.Itoa(row.LoginsGoogle),
			strconv.Itoa(row.LoginsApple),
			strconv.Itoa(row.PlatformIOS), strconv.Itoa(row.PlatformAndroid),
			strconv.Itoa(row.PlatformWeb), strconv.Itoa(row.PlatformOther),
		})
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		s.log.Error("stream stats csv", "error", err.Error())
	}
}

// adminAuthed reports whether the request carries a live admin session
// cookie — the same check the admin RPC interceptor performs.
func (s *Server) adminAuthed(r *http.Request) bool {
	c, err := r.Cookie(adminrpc.CookieName)
	if err != nil || c.Value == "" {
		return false
	}
	sess, err := s.store.GetSession(r.Context(), token.Hash(c.Value))
	if err != nil {
		return false
	}
	return time.Now().Before(sess.ExpiresAt)
}
