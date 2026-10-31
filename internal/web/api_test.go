package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gentcod/flatmate/internal/ai"
	"github.com/gentcod/flatmate/internal/auth"
	"github.com/gentcod/flatmate/internal/db"
	"github.com/gentcod/flatmate/internal/match"
	"github.com/gentcod/flatmate/internal/profile"
)

func testAPIServer(t *testing.T) *http.ServeMux {
	t.Helper()
	return testAPIServerFull(t).NewMux("../../web/static")
}

func testAPIServerFull(t *testing.T) *Server {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if err := db.Migrate(d, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	sessions, err := auth.NewEphemeralSessions(false)
	if err != nil {
		t.Fatal(err)
	}
	weights, err := match.LoadWeights("../../weights.yaml")
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{DB: d, Sessions: sessions, Weights: weights, AI: ai.Disabled(),
		LoginLimiter: NewLimiter(1000, time.Minute),
		AILimiter:    NewLimiter(1000, time.Minute),
		Interviews:   NewInterviews(time.Minute)}
	return srv
}

// apiClient drives the JSON API like the SPA: it keeps cookies and
// the CSRF token across requests.
type apiClient struct {
	t    *testing.T
	mux  *http.ServeMux
	jar  map[string]string
	csrf string
}

func newAPIClient(t *testing.T, mux *http.ServeMux) *apiClient {
	t.Helper()
	c := &apiClient{t: t, mux: mux, jar: map[string]string{}}
	rec := c.do("GET", "/api/v1/csrf", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("csrf: status %d", rec.Code)
	}
	var out map[string]string
	decodeBody(t, rec, &out)
	c.csrf = out["csrf_token"]
	if c.csrf == "" {
		t.Fatal("csrf: empty token")
	}
	return c
}

func (c *apiClient) do(method, path string, body any) *httptest.ResponseRecorder {
	c.t.Helper()
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range c.jar {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	if method == "POST" || method == "PUT" {
		req.Header.Set("X-CSRF-Token", c.csrf)
	}
	rec := httptest.NewRecorder()
	c.mux.ServeHTTP(rec, req)
	for _, sc := range rec.Result().Cookies() {
		c.jar[sc.Name] = sc.Value
	}
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(v); err != nil {
		t.Fatalf("decode status-%d body: %v", rec.Code, err)
	}
}

func signup(t *testing.T, c *apiClient, email string) int64 {
	t.Helper()
	rec := c.do("POST", "/api/v1/signup", map[string]string{
		"email": email, "password": "Seedpass123!",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("signup %s: status %d: %s", email, rec.Code, rec.Body.String())
	}
	var out map[string]any
	decodeBody(t, rec, &out)
	id, _ := out["user_id"].(float64)
	return int64(id)
}

func validProfile() map[string]any {
	return map[string]any{
		"first_name": "Ada", "mode": profile.ModeHaveRoom, "area_text": "Yaba",
		"relocating": false, "budget_min": 100, "budget_max": 200, "currency": "NGN",
		"move_earliest": "2026-12-01", "duration_months": 6,
		"gender": profile.GenderWoman, "gender_pref": profile.GenderPrefAnyone,
		"occupation": "Nurse", "bio": "Quiet", "contact_handle": "@ada",
		"searching": true,
	}
}

func fullAnswers() []map[string]any {
	var out []map[string]any
	for _, d := range match.Dimensions {
		v := "3"
		if len(d.Choices) > 0 {
			v = d.Choices[0]
		} else {
			n := 3
			if n < d.Min {
				n = d.Min
			}
			if n > d.Max {
				n = d.Max
			}
			v = strconv.Itoa(n)
		}
		out = append(out, map[string]any{
			"dimension": d.Name, "value": v, "hard": false, "weight": 1,
		})
	}
	return out
}

func TestAPISessionLoggedOut(t *testing.T) {
	c := newAPIClient(t, testAPIServer(t))
	rec := c.do("GET", "/api/v1/session", nil)
	var session map[string]any
	decodeBody(t, rec, &session)
	if session["authed"] != false {
		t.Errorf("logged-out session: got %v", session)
	}
	if rec := c.do("GET", "/api/v1/profile", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("logged-out profile: status %d, want 401", rec.Code)
	}
}

func TestAPICSRFRejected(t *testing.T) {
	mux := testAPIServer(t)
	c := &apiClient{t: t, mux: mux, jar: map[string]string{}} // no csrf fetch
	rec := c.do("POST", "/api/v1/signup", map[string]string{
		"email": "x@example.com", "password": "Seedpass123!",
	})
	if rec.Code != http.StatusForbidden {
		t.Errorf("missing token: status %d, want 403", rec.Code)
	}
}

func TestAPISignupLoginFlow(t *testing.T) {
	c := newAPIClient(t, testAPIServer(t))
	id := signup(t, c, "ada@example.com")
	if id <= 0 {
		t.Fatalf("signup: bad id %d", id)
	}
	var session map[string]any
	decodeBody(t, c.do("GET", "/api/v1/session", nil), &session)
	if session["authed"] != true {
		t.Errorf("after signup: authed=%v", session["authed"])
	}
	if rec := c.do("POST", "/api/v1/logout", map[string]string{}); rec.Code != http.StatusOK {
		t.Errorf("logout: status %d", rec.Code)
	}
	decodeBody(t, c.do("GET", "/api/v1/session", nil), &session)
	if session["authed"] != false {
		t.Errorf("after logout: authed=%v", session["authed"])
	}
	rec := c.do("POST", "/api/v1/login", map[string]string{
		"email": "ada@example.com", "password": "Seedpass123!",
	})
	if rec.Code != http.StatusOK {
		t.Errorf("login: status %d: %s", rec.Code, rec.Body.String())
	}
	rec = c.do("POST", "/api/v1/login", map[string]string{
		"email": "ada@example.com", "password": "wrongpass1",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("bad password: status %d, want 401", rec.Code)
	}
	rec = c.do("POST", "/api/v1/signup", map[string]string{
		"email": "ada@example.com", "password": "Seedpass123!",
	})
	if rec.Code != http.StatusConflict {
		t.Errorf("duplicate signup: status %d, want 409", rec.Code)
	}
	rec = c.do("POST", "/api/v1/signup", map[string]string{
		"email": "not-an-email", "password": "Seedpass123!",
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad email: status %d, want 400", rec.Code)
	}
	rec = c.do("POST", "/api/v1/signup", map[string]string{
		"email": "new@example.com", "password": "short",
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("short password: status %d, want 400", rec.Code)
	}
}

func TestAPIProfileRoundTrip(t *testing.T) {
	c := newAPIClient(t, testAPIServer(t))
	signup(t, c, "ada@example.com")

	var get map[string]any
	decodeBody(t, c.do("GET", "/api/v1/profile", nil), &get)
	if get["found"] != false {
		t.Fatalf("fresh profile: found=%v", get["found"])
	}
	defs, _ := get["profile"].(map[string]any)
	if defs["currency"] != "USD" || defs["duration_months"] != float64(6) || defs["searching"] != true {
		t.Errorf("fresh defaults: %v", defs)
	}

	var put map[string]any
	rec := c.do("PUT", "/api/v1/profile", validProfile())
	if rec.Code != http.StatusOK {
		t.Fatalf("put profile: status %d: %s", rec.Code, rec.Body.String())
	}
	decodeBody(t, rec, &put)
	saved, _ := put["profile"].(map[string]any)
	if saved["move_latest"] != "2027-06-01" {
		t.Errorf("derived move_latest = %v, want 2027-06-01", saved["move_latest"])
	}
	if saved["searching"] != true {
		t.Errorf("saved searching = %v, want true", saved["searching"])
	}
	off := validProfile()
	off["searching"] = false
	if rec := c.do("PUT", "/api/v1/profile", off); rec.Code != http.StatusOK {
		t.Fatalf("hide profile: status %d", rec.Code)
	}
	decodeBody(t, c.do("GET", "/api/v1/profile", nil), &get)
	hidden, _ := get["profile"].(map[string]any)
	if hidden["searching"] != false {
		t.Errorf("after hide: searching=%v, want false", hidden["searching"])
	}

	decodeBody(t, c.do("GET", "/api/v1/profile", nil), &get)
	if get["found"] != true {
		t.Errorf("after save: found=%v", get["found"])
	}

	bad := validProfile()
	bad["duration_months"] = 0
	if rec := c.do("PUT", "/api/v1/profile", bad); rec.Code != http.StatusBadRequest {
		t.Errorf("bad duration: status %d, want 400", rec.Code)
	}
}

func TestAPIQuestionnaire(t *testing.T) {
	c := newAPIClient(t, testAPIServer(t))
	signup(t, c, "ada@example.com")

	rec := c.do("PUT", "/api/v1/questionnaire", map[string]any{
		"answers": fullAnswers()[:3],
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("partial answers: status %d, want 400", rec.Code)
	}
	rec = c.do("PUT", "/api/v1/questionnaire", map[string]any{"answers": fullAnswers()})
	if rec.Code != http.StatusOK {
		t.Fatalf("save answers: status %d: %s", rec.Code, rec.Body.String())
	}
	var get map[string]any
	decodeBody(t, c.do("GET", "/api/v1/questionnaire", nil), &get)
	dims, _ := get["dims"].([]any)
	if len(dims) != len(match.Dimensions) {
		t.Fatalf("dims = %d, want %d", len(dims), len(match.Dimensions))
	}
	first, _ := dims[0].(map[string]any)
	if first["value"] == "" || first["label"] == "" || len(first["options"].([]any)) == 0 {
		t.Errorf("first dim missing value/label/options: %v", first)
	}
}

func TestAPIMatchesSetupFlow(t *testing.T) {
	c := newAPIClient(t, testAPIServer(t))
	signup(t, c, "ada@example.com")

	rec := c.do("GET", "/api/v1/matches", nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("no profile: status %d, want 409", rec.Code)
	}
	var conflict map[string]string
	decodeBody(t, rec, &conflict)
	if conflict["setup_required"] != "profile" {
		t.Errorf("no profile: setup_required=%q", conflict["setup_required"])
	}

	if rec := c.do("PUT", "/api/v1/profile", validProfile()); rec.Code != http.StatusOK {
		t.Fatalf("put profile: status %d", rec.Code)
	}
	rec = c.do("GET", "/api/v1/matches", nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("no answers: status %d, want 409", rec.Code)
	}
	decodeBody(t, rec, &conflict)
	if conflict["setup_required"] != "questionnaire" {
		t.Errorf("no answers: setup_required=%q", conflict["setup_required"])
	}

	if rec := c.do("PUT", "/api/v1/questionnaire", map[string]any{"answers": fullAnswers()}); rec.Code != http.StatusOK {
		t.Fatalf("save answers: status %d", rec.Code)
	}
	var list map[string]any
	decodeBody(t, c.do("GET", "/api/v1/matches", nil), &list)
	if list["empty"] != true {
		t.Errorf("solo user: empty=%v", list["empty"])
	}

	if rec := c.do("GET", "/api/v1/matches/9999", nil); rec.Code != http.StatusNotFound {
		t.Errorf("missing match: status %d, want 404", rec.Code)
	}
}

func TestAPIOnboardingFallback(t *testing.T) {
	// AI is disabled in tests, so the interview always falls back to
	// the plain form seed instead of failing.
	c := newAPIClient(t, testAPIServer(t))
	signup(t, c, "ada@example.com")

	var get map[string]any
	decodeBody(t, c.do("GET", "/api/v1/onboarding", nil), &get)
	if get["active"] != false {
		t.Errorf("fresh onboarding: active=%v", get["active"])
	}
	var post map[string]any
	rec := c.do("POST", "/api/v1/onboarding", map[string]string{
		"message": "I keep the kitchen spotless and sleep early.",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("interview post: status %d: %s", rec.Code, rec.Body.String())
	}
	decodeBody(t, rec, &post)
	if post["fallback"] != true {
		t.Errorf("disabled AI: fallback=%v, want true (%v)", post["fallback"], post)
	}
}

func TestAPIVisibility(t *testing.T) {
	mux := testAPIServer(t)
	a := newAPIClient(t, mux)
	b := newAPIClient(t, mux)
	signup(t, a, "ada@example.com")
	bID := signup(t, b, "bo@example.com")
	for _, c := range []*apiClient{a, b} {
		if rec := c.do("PUT", "/api/v1/profile", validProfile()); rec.Code != http.StatusOK {
			t.Fatalf("put profile: status %d: %s", rec.Code, rec.Body.String())
		}
		if rec := c.do("PUT", "/api/v1/questionnaire", map[string]any{"answers": fullAnswers()}); rec.Code != http.StatusOK {
			t.Fatalf("save answers: status %d: %s", rec.Code, rec.Body.String())
		}
	}
	list := func(c *apiClient) map[string]any {
		var out map[string]any
		decodeBody(t, c.do("GET", "/api/v1/matches", nil), &out)
		return out
	}
	got := list(a)
	ms, _ := got["matches"].([]any)
	if len(ms) != 1 {
		t.Fatalf("visible pair: matches=%d, want 1", len(ms))
	}
	first, _ := ms[0].(map[string]any)
	if id, _ := first["id"].(float64); int64(id) != bID {
		t.Errorf("visible pair: match id=%v, want %d", first["id"], bID)
	}
	if got["searching"] != true {
		t.Errorf("own searching flag: %v, want true", got["searching"])
	}

	hidden := validProfile()
	hidden["searching"] = false
	if rec := b.do("PUT", "/api/v1/profile", hidden); rec.Code != http.StatusOK {
		t.Fatalf("hide: status %d", rec.Code)
	}
	got = list(a)
	if got["empty"] != true {
		t.Errorf("after hide: empty=%v, want true", got["empty"])
	}
	if rec := a.do("GET", "/api/v1/matches/"+strconv.FormatInt(bID, 10), nil); rec.Code != http.StatusNotFound {
		t.Errorf("hidden detail: status %d, want 404", rec.Code)
	}

	if rec := b.do("PUT", "/api/v1/profile", validProfile()); rec.Code != http.StatusOK {
		t.Fatalf("re-show: status %d", rec.Code)
	}
	got = list(a)
	ms, _ = got["matches"].([]any)
	if len(ms) != 1 {
		t.Errorf("after re-show: matches=%d, want 1", len(ms))
	}
}

func TestAPIAreaFilter(t *testing.T) {
	mux := testAPIServer(t)
	a := newAPIClient(t, mux)
	b := newAPIClient(t, mux)
	c := newAPIClient(t, mux)
	signup(t, a, "ada@example.com")
	signup(t, b, "bo@example.com")
	signup(t, c, "cee@example.com")
	london := validProfile()
	london["first_name"] = "Cee"
	london["area_text"] = "Brixton, London"
	setups := []struct {
		client  *apiClient
		profile map[string]any
	}{{a, validProfile()}, {b, validProfile()}, {c, london}}
	for _, s := range setups {
		if rec := s.client.do("PUT", "/api/v1/profile", s.profile); rec.Code != http.StatusOK {
			t.Fatalf("put profile: status %d: %s", rec.Code, rec.Body.String())
		}
		if rec := s.client.do("PUT", "/api/v1/questionnaire", map[string]any{"answers": fullAnswers()}); rec.Code != http.StatusOK {
			t.Fatalf("save answers: status %d: %s", rec.Code, rec.Body.String())
		}
	}
	count := func(path string) int {
		var out map[string]any
		decodeBody(t, a.do("GET", path, nil), &out)
		ms, _ := out["matches"].([]any)
		return len(ms)
	}
	if n := count("/api/v1/matches"); n != 2 {
		t.Errorf("unfiltered: matches=%d, want 2", n)
	}
	if n := count("/api/v1/matches?area=london"); n != 1 {
		t.Errorf("area=london: matches=%d, want 1", n)
	}
	if n := count("/api/v1/matches?area=YABA"); n != 1 {
		t.Errorf("area=YABA: matches=%d, want 1 (case-insensitive)", n)
	}
	if n := count("/api/v1/matches?area=nowhere"); n != 0 {
		t.Errorf("area=nowhere: matches=%d, want 0", n)
	}
}

func TestAPIOnboardingDrafts(t *testing.T) {
	srv := testAPIServerFull(t)
	c := newAPIClient(t, srv.NewMux("../../web/static"))
	id := signup(t, c, "ada@example.com")
	err := profile.SaveOnboarding(context.Background(), srv.DB, id,
		"user: I sleep early\n",
		`[{"dimension":"sleep","value":"early","hard":false,"weight":2}]`)
	if err != nil {
		t.Fatal(err)
	}
	var get map[string]any
	decodeBody(t, c.do("GET", "/api/v1/onboarding", nil), &get)
	if get["text"] != "user: I sleep early\n" {
		t.Errorf("text = %q", get["text"])
	}
	drafts, _ := get["drafts"].([]any)
	if len(drafts) != 1 {
		t.Fatalf("drafts = %d, want 1", len(drafts))
	}
	d, _ := drafts[0].(map[string]any)
	if d["dimension"] != "sleep" || d["value"] != "early" {
		t.Errorf("draft = %v", d)
	}
}

func TestAPIProfileNudge(t *testing.T) {
	c := newAPIClient(t, testAPIServer(t))
	signup(t, c, "ada@example.com")
	if rec := c.do("PUT", "/api/v1/questionnaire", map[string]any{"answers": fullAnswers()}); rec.Code != http.StatusOK {
		t.Fatalf("save answers: status %d", rec.Code)
	}
	nudge := func(p map[string]any) string {
		if rec := c.do("PUT", "/api/v1/profile", p); rec.Code != http.StatusOK {
			t.Fatalf("put profile: status %d: %s", rec.Code, rec.Body.String())
		}
		var out map[string]any
		decodeBody(t, c.do("GET", "/api/v1/matches", nil), &out)
		n, _ := out["nudge"].(string)
		return n
	}
	if n := nudge(validProfile()); n != "" {
		t.Errorf("complete profile: nudge=%q, want empty", n)
	}
	noBio := validProfile()
	noBio["bio"] = ""
	if n := nudge(noBio); !strings.Contains(n, "bio") {
		t.Errorf("missing bio: nudge=%q", n)
	}
	noWork := validProfile()
	noWork["occupation"] = ""
	if n := nudge(noWork); !strings.Contains(n, "occupation") {
		t.Errorf("missing occupation: nudge=%q", n)
	}
	noContact := validProfile()
	noContact["contact_handle"] = ""
	if n := nudge(noContact); !strings.Contains(n, "contact") {
		t.Errorf("missing contact: nudge=%q", n)
	}
}

func TestAPIDeactivate(t *testing.T) {
	c := newAPIClient(t, testAPIServer(t))
	signup(t, c, "ada@example.com")

	rec := c.do("POST", "/api/v1/account/deactivate", map[string]bool{"confirm": false})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("unconfirmed: status %d, want 400", rec.Code)
	}
	rec = c.do("POST", "/api/v1/account/deactivate", map[string]bool{"confirm": true})
	if rec.Code != http.StatusOK {
		t.Fatalf("deactivate: status %d: %s", rec.Code, rec.Body.String())
	}
	var session map[string]any
	decodeBody(t, c.do("GET", "/api/v1/session", nil), &session)
	if session["authed"] != false {
		t.Errorf("after deactivate: authed=%v", session["authed"])
	}
}
