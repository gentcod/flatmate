// Package web serves pages and handlers over stdlib net/http.
package web

import (
	"context"
	"database/sql"
	"html/template"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gentcod/flatmate/internal/ai"
	"github.com/gentcod/flatmate/internal/auth"
	"github.com/gentcod/flatmate/internal/database"
	"github.com/gentcod/flatmate/internal/match"
)

// Server holds shared dependencies.
type Server struct {
	DB           *sql.DB
	Sessions     *auth.Sessions
	Pages        map[string]*template.Template
	Weights      match.Weights
	AI           ai.LLM
	CookieSecure bool
	LoginLimiter *Limiter
	AILimiter    *Limiter
	Interviews   *Interviews
}

const sessionTTL = 7 * 24 * time.Hour

type ctxKey string

const userKey ctxKey = "userID"

// LoadPages parses base.html plus each page into its own template set.
func LoadPages(dir string) (map[string]*template.Template, error) {
	base, err := template.ParseFiles(filepath.Join(dir, "base.html"))
	if err != nil {
		return nil, err
	}
	pages := []string{"signup", "login", "profile", "questionnaire", "matches", "match_detail", "onboarding"}
	out := make(map[string]*template.Template, len(pages))
	for _, name := range pages {
		t, err := base.Clone()
		if err != nil {
			return nil, err
		}
		if _, err := t.ParseFiles(filepath.Join(dir, name+".html")); err != nil {
			return nil, err
		}
		out[name] = t
	}
	return out, nil
}

// NewMux wires routes.
func (s *Server) NewMux(staticDir string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("GET /{$}", s.handleHome)
	mux.HandleFunc("GET /signup", s.handleSignupShow)
	mux.HandleFunc("POST /signup", s.handleSignupPost)
	mux.HandleFunc("GET /login", s.handleLoginShow)
	mux.HandleFunc("POST /login", s.handleLoginPost)
	mux.HandleFunc("POST /logout", s.handleLogout)
	mux.Handle("GET /profile", s.requireAuth(http.HandlerFunc(s.handleProfileShow)))
	mux.Handle("POST /profile", s.requireAuth(http.HandlerFunc(s.handleProfilePost)))
	mux.Handle("GET /questionnaire", s.requireAuth(http.HandlerFunc(s.handleQuestionnaireShow)))
	mux.Handle("POST /questionnaire", s.requireAuth(http.HandlerFunc(s.handleQuestionnairePost)))
	mux.Handle("GET /matches", s.requireAuth(http.HandlerFunc(s.handleMatchesShow)))
	mux.Handle("GET /match/{id}", s.requireAuth(http.HandlerFunc(s.handleMatchDetail)))
	mux.Handle("POST /match/{id}/save", s.requireAuth(http.HandlerFunc(s.handleSave)))
	mux.Handle("POST /match/{id}/unsave", s.requireAuth(http.HandlerFunc(s.handleUnsave)))
	mux.Handle("POST /match/{id}/dismiss", s.requireAuth(http.HandlerFunc(s.handleDismiss)))
	mux.Handle("POST /match/{id}/interest", s.requireAuth(http.HandlerFunc(s.handleInterest)))
	mux.Handle("POST /match/{id}/block", s.requireAuth(http.HandlerFunc(s.handleBlock)))
	mux.Handle("POST /match/{id}/report", s.requireAuth(http.HandlerFunc(s.handleReport)))
	mux.Handle("POST /account/deactivate", s.requireAuth(http.HandlerFunc(s.handleDeactivate)))
	mux.Handle("GET /onboarding", s.requireAuth(http.HandlerFunc(s.handleOnboardingShow)))
	mux.Handle("POST /onboarding", s.requireAuth(http.HandlerFunc(s.handleOnboardingPost)))
	return mux
}

func (s *Server) render(w http.ResponseWriter, page string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	if _, ok := data["Title"]; !ok {
		data["Title"] = "Flatmate"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.Pages[page].ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// csrfToken returns this request's form token, minting the CSRF
// cookie on first sight. Every rendered form echoes it back.
func (s *Server) csrfToken(w http.ResponseWriter, r *http.Request) string {
	return auth.EnsureCSRFToken(w, r, s.CookieSecure)
}

// checkCSRF rejects forged POSTs with 403.
func (s *Server) checkCSRF(w http.ResponseWriter, r *http.Request) bool {
	if !auth.CheckCSRF(r) {
		http.Error(w, "bad form token; please reload and try again", http.StatusForbidden)
		return false
	}
	return true
}

// userID returns the authenticated user ID from the request, if any.
func (s *Server) userID(r *http.Request) (int64, bool) {
	tok, ok := auth.CookieToken(r)
	if !ok {
		return 0, false
	}
	id, err := s.Sessions.Verify(tok)
	if err != nil {
		return 0, false
	}
	deactivated, err := database.New(s.DB).GetUserDeactivation(r.Context(), id)
	if err != nil || deactivated.Valid {
		return 0, false
	}
	return id, true
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := s.userID(r)
		if !ok {
			s.Sessions.ClearCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, id)))
	})
}

func authedID(r *http.Request) int64 {
	id, _ := r.Context().Value(userKey).(int64)
	return id
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.userID(r); ok {
		http.Redirect(w, r, "/matches", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
