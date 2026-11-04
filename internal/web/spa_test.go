package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegisterSPAHeaders(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"index.html":           "<!doctype html><title>x</title>",
		"sw.js":                "// worker",
		"assets/app.js":        "console.log(1)",
		"manifest.webmanifest": `{"name":"x"}`,
	}
	for name, body := range files {
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mux := http.NewServeMux()
	(&Server{ClientDist: dir}).registerSPA(mux)
	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		return rec
	}
	if rec := get("/sw.js"); rec.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("sw.js Cache-Control=%q, want no-cache", rec.Header().Get("Cache-Control"))
	}
	if rec := get("/assets/app.js"); rec.Header().Get("Cache-Control") == "no-cache" {
		t.Errorf("hashed asset must stay cacheable, got no-cache")
	}
	if rec := get("/manifest.webmanifest"); rec.Header().Get("Content-Type") != "application/manifest+json" {
		t.Errorf("manifest Content-Type=%q", rec.Header().Get("Content-Type"))
	}
	rec := get("/matches")
	if rec.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("fallback Cache-Control=%q, want no-cache", rec.Header().Get("Cache-Control"))
	}
	if !strings.Contains(rec.Body.String(), "<title>x</title>") {
		t.Errorf("fallback body=%q, want index.html", rec.Body.String())
	}
}

func TestPlaceholdersRoute(t *testing.T) {
	staticDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(staticDir, "placeholders"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "placeholders", "room-1.svg"), []byte("<svg/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	dist := t.TempDir()
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("spa"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, ui := range []string{"legacy", "spa"} {
		mux := (&Server{UI: ui, ClientDist: dist}).NewMux(staticDir)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", "/placeholders/room-1.svg", nil))
		if rec.Code != http.StatusOK {
			t.Errorf("ui=%s: status %d, want 200", ui, rec.Code)
		}
	}
}
