package web

import (
	"net/http"

	"github.com/gentcod/flatmate/internal/database"
)

func (s *Server) handleAdminReports(w http.ResponseWriter, r *http.Request) {
	rows, err := database.New(s.DB).ListReports(r.Context(), 200)
	if err != nil {
		http.Error(w, "load error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "admin_reports", map[string]any{"Title": "Reports", "Authed": true,
		"IsAdmin": true, "Reports": s.reportViews(r, rows), "CSRF": s.csrfToken(w, r)})
}

func (s *Server) handleAdminResolve(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id, ok := pathID(r)
	if !ok {
		http.Error(w, "report not found", http.StatusNotFound)
		return
	}
	if status, msg := s.applyResolution(r, id, r.FormValue("action")); status != http.StatusOK {
		http.Error(w, msg, status)
		return
	}
	http.Redirect(w, r, "/admin/reports", http.StatusSeeOther)
}

func (s *Server) handleAdminMetrics(w http.ResponseWriter, r *http.Request) {
	counts, err := s.metricCounts(r)
	if err != nil {
		http.Error(w, "load error", http.StatusInternalServerError)
		return
	}
	counts["Title"] = "Metrics"
	counts["Authed"] = true
	counts["IsAdmin"] = true
	s.render(w, r, "admin_metrics", counts)
}
