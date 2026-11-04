package web

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gentcod/flatmate/internal/admin"
	"github.com/gentcod/flatmate/internal/database"
	"github.com/gentcod/flatmate/internal/profile"
)

func (s *Server) requireAdminAPI(next http.Handler) http.Handler {
	return s.requireAuthAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !admin.IsAdmin(r.Context(), database.New(s.DB), authedID(r)) {
			writeErr(w, http.StatusForbidden, "Admins only.")
			return
		}
		next.ServeHTTP(w, r)
	}))
}

type apiReport struct {
	ID           int64   `json:"id"`
	Reporter     int64   `json:"reporter"`
	ReporterName string  `json:"reporter_name"`
	Target       int64   `json:"target"`
	TargetName   string  `json:"target_name"`
	Reason       string  `json:"reason"`
	CreatedAt    string  `json:"created_at"`
	ResolvedAt   *string `json:"resolved_at"`
	Resolution   string  `json:"resolution"`
}

func (s *Server) displayName(r *http.Request, id int64) string {
	if p, found, err := profile.Get(r.Context(), s.DB, id); err == nil && found {
		return p.FirstName
	}
	return fmt.Sprintf("user #%d", id)
}

func (s *Server) reportViews(r *http.Request, rows []database.Report) []apiReport {
	out := make([]apiReport, 0, len(rows))
	for _, row := range rows {
		rep := apiReport{
			ID: row.ID, Reporter: row.Reporter, Target: row.Target,
			ReporterName: s.displayName(r, row.Reporter),
			TargetName:   s.displayName(r, row.Target),
			Reason:       row.Reason, CreatedAt: row.CreatedAt,
			Resolution: row.Resolution,
		}
		if row.ResolvedAt.Valid {
			v := row.ResolvedAt.String
			rep.ResolvedAt = &v
		}
		out = append(out, rep)
	}
	return out
}

// applyResolution validates and applies an admin action on a report.
// It returns the HTTP status and message the caller should render.
func (s *Server) applyResolution(r *http.Request, reportID int64, action string) (int, string) {
	if action != "dismiss" && action != "deactivate" {
		return http.StatusBadRequest, "Unknown action."
	}
	q := database.New(s.DB)
	rep, err := q.GetReport(r.Context(), reportID)
	if err != nil {
		if err == sql.ErrNoRows {
			return http.StatusNotFound, "Report not found."
		}
		return http.StatusInternalServerError, "Something went wrong. Try again."
	}
	if rep.ResolvedAt.Valid {
		return http.StatusBadRequest, "Already resolved."
	}
	if action == "deactivate" {
		if rep.Target == authedID(r) {
			return http.StatusBadRequest, "You can't deactivate yourself."
		}
		if err := q.DeactivateUser(r.Context(), database.DeactivateUserParams{
			DeactivatedAt: sql.NullString{String: nowRFC3339(), Valid: true}, ID: rep.Target,
		}); err != nil {
			return http.StatusInternalServerError, "Something went wrong. Try again."
		}
	}
	if err := q.ResolveReport(r.Context(), database.ResolveReportParams{
		ResolvedAt: sql.NullString{String: nowRFC3339(), Valid: true},
		Resolution: action, ID: reportID,
	}); err != nil {
		return http.StatusInternalServerError, "Something went wrong. Try again."
	}
	return http.StatusOK, "ok"
}

func (s *Server) apiReportsList(w http.ResponseWriter, r *http.Request) {
	rows, err := database.New(s.DB).ListReports(r.Context(), 200)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reports": s.reportViews(r, rows)})
}

func (s *Server) apiReportResolve(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id, ok := pathID(r)
	if !ok {
		writeErr(w, http.StatusNotFound, "Report not found.")
		return
	}
	var in struct {
		Action string `json:"action"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if status, msg := s.applyResolution(r, id, in.Action); status != http.StatusOK {
		writeErr(w, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) metricCounts(r *http.Request) (map[string]any, error) {
	q := database.New(s.DB)
	ctx := r.Context()
	users, err := q.CountAllUsers(ctx)
	if err != nil {
		return nil, err
	}
	active, err := q.CountActiveUsers(ctx)
	if err != nil {
		return nil, err
	}
	open, err := q.CountOpenReports(ctx)
	if err != nil {
		return nil, err
	}
	listings, err := q.CountListings(ctx)
	if err != nil {
		return nil, err
	}
	reviews, err := q.CountReviews(ctx)
	if err != nil {
		return nil, err
	}
	expenses, err := q.CountExpenses(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"users": users, "active_users": active, "open_reports": open,
		"listings": listings, "reviews": reviews, "expenses": expenses,
	}, nil
}

func (s *Server) apiMetrics(w http.ResponseWriter, r *http.Request) {
	counts, err := s.metricCounts(r)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	writeJSON(w, http.StatusOK, counts)
}
