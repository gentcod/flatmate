package web

import (
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/gentcod/flatmate/internal/auth"
	"github.com/gentcod/flatmate/internal/database"
)

func (s *Server) handleSignupShow(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.userID(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	s.render(w, "signup", map[string]any{"Title": "Sign up", "CSRF": s.csrfToken(w, r)})
}

func (s *Server) handleSignupPost(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	email := auth.NormalizeEmail(r.FormValue("email"))
	password := r.FormValue("password")
	fail := func(msg string) {
		s.render(w, "signup", map[string]any{
			"Title": "Sign up", "Error": msg, "Email": email, "CSRF": s.csrfToken(w, r),
		})
	}
	if !s.LoginLimiter.Allow(clientIP(r)) {
		fail("Too many attempts — try again in a minute.")
		return
	}
	if _, err := mail.ParseAddress(email); err != nil || len(email) > 254 {
		fail("Enter a valid email address.")
		return
	}
	if len(password) < 8 {
		fail("Password must be at least 8 characters.")
		return
	}
	hash := auth.HashEmail(email)
	exists, err := database.New(s.DB).CountUsersByEmailHash(r.Context(), hash)
	if err != nil {
		fail("Something went wrong. Try again.")
		return
	}
	if exists > 0 {
		fail("That email is already registered. Log in instead.")
		return
	}
	pw, err := auth.HashPassword(password)
	if err != nil {
		fail("Something went wrong. Try again.")
		return
	}
	res, err := database.New(s.DB).CreateUser(r.Context(), database.CreateUserParams{
		EmailHash: hash, PassHash: pw, CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			fail("That email is already registered. Log in instead.")
			return
		}
		fail("Something went wrong. Try again.")
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		fail("Something went wrong. Try again.")
		return
	}
	tok, err := s.Sessions.Issue(id, sessionTTL)
	if err != nil {
		fail("Something went wrong. Try again.")
		return
	}
	s.Sessions.SetCookie(w, tok, sessionTTL)
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func (s *Server) handleLoginShow(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.userID(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	s.render(w, "login", map[string]any{"Title": "Log in", "CSRF": s.csrfToken(w, r)})
}

func (s *Server) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	email := auth.NormalizeEmail(r.FormValue("email"))
	password := r.FormValue("password")
	fail := func(msg string) {
		s.render(w, "login", map[string]any{
			"Title": "Log in", "Error": msg, "Email": email, "CSRF": s.csrfToken(w, r),
		})
	}
	if !s.LoginLimiter.Allow(clientIP(r)) {
		fail("Too many attempts — try again in a minute.")
		return
	}
	u, err := database.New(s.DB).GetUserByEmailHash(r.Context(), auth.HashEmail(email))
	if err != nil || u.DeactivatedAt.Valid || !auth.CheckPassword(u.PassHash, password) {
		fail("Invalid email or password.")
		return
	}
	tok, err := s.Sessions.Issue(u.ID, sessionTTL)
	if err != nil {
		fail("Invalid email or password.")
		return
	}
	s.Sessions.SetCookie(w, tok, sessionTTL)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	s.Sessions.ClearCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
