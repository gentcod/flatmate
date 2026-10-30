package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCSRFTokenRoundTrip(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	tok := EnsureCSRFToken(rec, req, false)
	if len(tok) != 64 {
		t.Fatalf("token = %q", tok)
	}
	post := httptest.NewRequest("POST", "/", strings.NewReader("csrf_token="+tok))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range rec.Result().Cookies() {
		post.AddCookie(c)
	}
	if again := EnsureCSRFToken(httptest.NewRecorder(), post, false); again != tok {
		t.Errorf("stable = %q want %q", again, tok)
	}
	if !CheckCSRF(post) {
		t.Error("valid pair rejected")
	}
}

func TestCSRFRejects(t *testing.T) {
	rec := httptest.NewRecorder()
	tok := EnsureCSRFToken(rec, httptest.NewRequest("GET", "/", nil), false)
	post := func(cookie, field string) *http.Request {
		r := httptest.NewRequest("POST", "/", strings.NewReader("csrf_token="+field))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: "csrf", Value: cookie})
		}
		return r
	}
	zeros := strings.Repeat("0", 64)
	cases := map[string]*http.Request{
		"no cookie":     post("", tok),
		"no field":      post(tok, ""),
		"mismatch":      post(tok, zeros),
		"short cookie":  post("abc", tok),
		"short field":   post(tok, "abc"),
		"non-hex field": post(tok, strings.Repeat("z", 64)),
	}
	for name, r := range cases {
		if CheckCSRF(r) {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestSessionsSecureFlag(t *testing.T) {
	s, err := NewSessions(strings.Repeat("ab", 32), true)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	s.SetCookie(rec, "t", time.Hour)
	found := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" && c.Secure && c.HttpOnly {
			found = true
		}
	}
	if !found {
		t.Error("session cookie missing Secure/HttpOnly")
	}
	if _, err := NewSessions("xyz", false); err == nil {
		t.Error("bad key: want error")
	}
	if _, err := NewEphemeralSessions(false); err != nil {
		t.Errorf("ephemeral: %v", err)
	}
}
