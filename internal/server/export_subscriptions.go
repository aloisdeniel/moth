package server

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aloisdeniel/moth/internal/store"
)

// Subscription revenue export range: months, matching AnalyticsService.
const (
	exportMonthFormat    = "2006-01"
	defaultExportMonths  = 12
	maxExportRangeMonths = 60
)

// handleExportSubscriptions streams a project's subscription revenue rollup as
// a CSV download:
// GET /admin/export/subscriptions.csv?project_id=ID[&from=YYYY-MM][&to=YYYY-MM].
// Like the stats and audit exports it rides plain HTTP (a browser download is
// not an RPC) and is authenticated by the same admin credential — the session
// cookie or a personal access token. It reads ONLY the pre-aggregated rollup
// tables, never the raw subscription_events, and is per currency (one row per
// month/currency) — never a fabricated blended total.
func (s *Server) handleExportSubscriptions(w http.ResponseWriter, r *http.Request) {
	if !s.adminHTTPAuthed(r) {
		writeJSONError(w, http.StatusUnauthorized, "admin session or personal access token required")
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

	// Default: the last 12 months ending with the current (in-progress) month
	// in the project's rollup timezone — revenue accrues through this month.
	localNow := time.Now().In(project.Settings.RollupLocation())
	to := time.Date(localNow.Year(), localNow.Month(), 1, 0, 0, 0, 0, time.UTC)
	if v := q.Get("to"); v != "" {
		if to, err = time.Parse(exportMonthFormat, v); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid to month")
			return
		}
	}
	from := to.AddDate(0, -(defaultExportMonths - 1), 0)
	if v := q.Get("from"); v != "" {
		if from, err = time.Parse(exportMonthFormat, v); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid from month")
			return
		}
	}
	if from.After(to) {
		writeJSONError(w, http.StatusBadRequest, "from is after to")
		return
	}
	if months := int(to.Year()-from.Year())*12 + int(to.Month()) - int(from.Month()); months >= maxExportRangeMonths {
		writeJSONError(w, http.StatusBadRequest, "range longer than 60 months")
		return
	}

	rows, err := s.store.GetSubscriptionStats(r.Context(), project.ID,
		from.Format(exportMonthFormat), to.Format(exportMonthFormat))
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=subscriptions-%s-%s-%s.csv",
			project.Slug, from.Format(exportMonthFormat), to.Format(exportMonthFormat)))

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"period", "currency", "revenue_micros", "active_subscribers",
		"new_subscribers", "renewals", "churned", "trials_started", "trials_converted",
		"store_apple_revenue_micros", "store_google_revenue_micros"})
	for _, row := range rows {
		// currency is store-reported; run it through csvSafe so a hostile
		// value cannot smuggle a spreadsheet formula. Numbers are safe as-is.
		_ = cw.Write([]string{
			row.Period,
			csvSafe(row.Currency),
			strconv.FormatInt(row.RevenueMicros, 10),
			strconv.Itoa(row.ActiveSubscribers),
			strconv.Itoa(row.NewSubscribers),
			strconv.Itoa(row.Renewals),
			strconv.Itoa(row.Churned),
			strconv.Itoa(row.TrialsStarted),
			strconv.Itoa(row.TrialsConverted),
			strconv.FormatInt(row.StoreAppleRevenueMicros, 10),
			strconv.FormatInt(row.StoreGoogleRevenueMicros, 10),
		})
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		s.log.Error("stream subscriptions csv", "error", err.Error())
	}
}
