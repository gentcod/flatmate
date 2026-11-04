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
	"github.com/gentcod/flatmate/internal/database"
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
		MsgLimiter:   NewLimiter(1000, time.Minute),
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
	if method == "POST" || method == "PUT" || method == "DELETE" {
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

func signupFull(t *testing.T, c *apiClient, email string) int64 {
	t.Helper()
	id := signup(t, c, email)
	if rec := c.do("PUT", "/api/v1/profile", validProfile()); rec.Code != http.StatusOK {
		t.Fatalf("put profile: status %d: %s", rec.Code, rec.Body.String())
	}
	if rec := c.do("PUT", "/api/v1/questionnaire", map[string]any{"answers": fullAnswers()}); rec.Code != http.StatusOK {
		t.Fatalf("save answers: status %d: %s", rec.Code, rec.Body.String())
	}
	return id
}

func expressInterest(t *testing.T, c *apiClient, target int64) {
	t.Helper()
	rec := c.do("POST", "/api/v1/matches/"+strconv.FormatInt(target, 10)+"/interest", map[string]string{})
	if rec.Code != http.StatusOK {
		t.Fatalf("interest: status %d: %s", rec.Code, rec.Body.String())
	}
}

func proposeStay(t *testing.T, c *apiClient, peer, listing int64) int64 {
	t.Helper()
	body := map[string]any{"peer_id": peer}
	if listing > 0 {
		body["listing_id"] = listing
	}
	rec := c.do("POST", "/api/v1/stays", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("propose stay: status %d: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	decodeBody(t, rec, &got)
	id, _ := got["id"].(float64)
	return int64(id)
}

func stayAction(t *testing.T, c *apiClient, stayID int64, action string) {
	t.Helper()
	rec := c.do("POST", "/api/v1/stays/"+strconv.FormatInt(stayID, 10)+"/"+action, map[string]string{})
	if rec.Code != http.StatusOK {
		t.Fatalf("stay %s: status %d: %s", action, rec.Code, rec.Body.String())
	}
}

func validListing() map[string]any {
	return map[string]any{
		"title": "Sunny room in Yaba", "description": "Furnished, shared kitchen.",
		"area_text": "Yaba, Lagos", "monthly_rent": 150000, "currency": "NGN",
	}
}

func TestAPIListings(t *testing.T) {
	mux := testAPIServer(t)
	a := newAPIClient(t, mux)
	b := newAPIClient(t, mux)
	signupFull(t, a, "ada@example.com")
	signupFull(t, b, "bo@example.com")

	var created map[string]any
	rec := a.do("POST", "/api/v1/listings", validListing())
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: status %d: %s", rec.Code, rec.Body.String())
	}
	decodeBody(t, rec, &created)
	idPath := "/api/v1/listings/" + strconv.FormatInt(int64(created["id"].(float64)), 10)

	var list map[string]any
	decodeBody(t, a.do("GET", "/api/v1/listings", nil), &list)
	items, _ := list["listings"].([]any)
	if len(items) != 1 {
		t.Fatalf("list: %d items, want 1", len(items))
	}
	first, _ := items[0].(map[string]any)
	if first["title"] != "Sunny room in Yaba" || first["owner_name"] != "Ada" || first["mine"] != true {
		t.Errorf("listing fields: %v", first)
	}

	var one map[string]any
	decodeBody(t, b.do("GET", idPath, nil), &one)
	got, _ := one["listing"].(map[string]any)
	if got["mine"] != false {
		t.Errorf("peer's view: mine=%v, want false", got["mine"])
	}

	edit := validListing()
	edit["title"] = "Cozy room in Yaba"
	rec = a.do("PUT", idPath, edit)
	if rec.Code != http.StatusOK {
		t.Fatalf("owner edit: status %d", rec.Code)
	}
	if rec := b.do("PUT", idPath, edit); rec.Code != http.StatusNotFound {
		t.Errorf("peer edit: status %d, want 404", rec.Code)
	}
	if rec := b.do("DELETE", idPath, nil); rec.Code == http.StatusOK {
		t.Errorf("peer delete: status %d, want non-200", rec.Code)
	}
	_ = rec

	bad := validListing()
	bad["title"] = ""
	if rec := a.do("POST", "/api/v1/listings", bad); rec.Code != http.StatusBadRequest {
		t.Errorf("empty title: status %d, want 400", rec.Code)
	}

	hidden := validProfile()
	hidden["searching"] = false
	if rec := b.do("PUT", "/api/v1/profile", hidden); rec.Code != http.StatusOK {
		t.Fatalf("hide b: status %d", rec.Code)
	}
	rec = b.do("POST", "/api/v1/listings", validListing())
	if rec.Code != http.StatusCreated {
		t.Fatalf("hidden create: status %d", rec.Code)
	}
	var bCreated map[string]any
	decodeBody(t, rec, &bCreated)
	bPath := "/api/v1/listings/" + strconv.FormatInt(int64(bCreated["id"].(float64)), 10)
	decodeBody(t, a.do("GET", "/api/v1/listings", nil), &list)
	items, _ = list["listings"].([]any)
	if len(items) != 1 {
		t.Errorf("hidden owner's listing visible: %d items, want 1", len(items))
	}
	if rec := a.do("GET", bPath, nil); rec.Code != http.StatusNotFound {
		t.Errorf("hidden listing detail: status %d, want 404", rec.Code)
	}

	if rec := a.do("DELETE", idPath, nil); rec.Code != http.StatusOK {
		t.Fatalf("owner delete: status %d: %s", rec.Code, rec.Body.String())
	}
	decodeBody(t, a.do("GET", "/api/v1/listings", nil), &list)
	items, _ = list["listings"].([]any)
	if len(items) != 0 {
		t.Errorf("after delete: %d items, want 0", len(items))
	}
}

func TestAPIMessaging(t *testing.T) {
	mux := testAPIServer(t)
	a := newAPIClient(t, mux)
	b := newAPIClient(t, mux)
	aID := signupFull(t, a, "ada@example.com")
	bID := signupFull(t, b, "bo@example.com")
	aThread := "/api/v1/conversations/" + strconv.FormatInt(bID, 10)

	send := func(c *apiClient, peer int64, body string) *httptest.ResponseRecorder {
		return c.do("POST", "/api/v1/conversations/"+strconv.FormatInt(peer, 10),
			map[string]string{"body": body})
	}
	if rec := send(a, bID, "hello"); rec.Code != http.StatusForbidden {
		t.Errorf("pre-mutual send: status %d, want 403", rec.Code)
	}
	expressInterest(t, a, bID)
	expressInterest(t, b, aID)
	if rec := send(a, bID, ""); rec.Code != http.StatusBadRequest {
		t.Errorf("empty body: status %d, want 400", rec.Code)
	}
	if rec := send(a, bID, "hello"); rec.Code != http.StatusCreated {
		t.Fatalf("send: status %d: %s", rec.Code, rec.Body.String())
	}
	if rec := send(b, aID, "hi there"); rec.Code != http.StatusCreated {
		t.Fatalf("reply: status %d: %s", rec.Code, rec.Body.String())
	}
	var thread map[string]any
	decodeBody(t, a.do("GET", aThread, nil), &thread)
	msgs, _ := thread["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("thread: %d messages, want 2", len(msgs))
	}
	m0, _ := msgs[0].(map[string]any)
	m1, _ := msgs[1].(map[string]any)
	if m0["body"] != "hello" || m0["from_me"] != true || m1["body"] != "hi there" || m1["from_me"] != false {
		t.Errorf("thread order/flags: %v %v", m0, m1)
	}
	peer, _ := thread["peer"].(map[string]any)
	if peer["first_name"] != "Ada" {
		t.Errorf("peer name: %v", peer)
	}
	var inbox map[string]any
	decodeBody(t, a.do("GET", "/api/v1/conversations", nil), &inbox)
	convos, _ := inbox["conversations"].([]any)
	if len(convos) != 1 {
		t.Fatalf("inbox: %d conversations, want 1", len(convos))
	}
	cv, _ := convos[0].(map[string]any)
	if cv["peer_name"] != "Ada" || cv["last_body"] != "hi there" || cv["last_from_me"] != false {
		t.Errorf("inbox entry: %v", cv)
	}

	hidden := validProfile()
	hidden["searching"] = false
	if rec := b.do("PUT", "/api/v1/profile", hidden); rec.Code != http.StatusOK {
		t.Fatalf("hide b: status %d", rec.Code)
	}
	if rec := send(a, bID, "still there?"); rec.Code != http.StatusNotFound {
		t.Errorf("send to hidden: status %d, want 404", rec.Code)
	}
	if rec := a.do("GET", aThread, nil); rec.Code != http.StatusOK {
		t.Errorf("hidden history: status %d, want 200", rec.Code)
	}

	if rec := a.do("POST", "/api/v1/matches/"+strconv.FormatInt(bID, 10)+"/block", map[string]string{}); rec.Code != http.StatusOK {
		t.Fatalf("block: status %d", rec.Code)
	}
	if rec := a.do("GET", aThread, nil); rec.Code != http.StatusNotFound {
		t.Errorf("blocked history: status %d, want 404", rec.Code)
	}
}

func TestAPIChatReadFlow(t *testing.T) {
	mux := testAPIServer(t)
	a := newAPIClient(t, mux)
	b := newAPIClient(t, mux)
	aID := signupFull(t, a, "ada@example.com")
	bID := signupFull(t, b, "bo@example.com")
	expressInterest(t, a, bID)
	expressInterest(t, b, aID)

	unread := func(c *apiClient) float64 {
		t.Helper()
		var inbox map[string]any
		decodeBody(t, c.do("GET", "/api/v1/conversations", nil), &inbox)
		convos, _ := inbox["conversations"].([]any)
		if len(convos) != 1 {
			t.Fatalf("inbox: %d conversations, want 1", len(convos))
		}
		cv, _ := convos[0].(map[string]any)
		n, _ := cv["unread_count"].(float64)
		return n
	}
	if rec := a.do("POST", "/api/v1/conversations/"+strconv.FormatInt(bID, 10), map[string]string{"body": "hello"}); rec.Code != http.StatusCreated {
		t.Fatalf("send: status %d", rec.Code)
	}
	if n := unread(b); n != 1 {
		t.Errorf("b unread before read: %v, want 1", n)
	}
	if rec := b.do("GET", "/api/v1/conversations/"+strconv.FormatInt(aID, 10), nil); rec.Code != http.StatusOK {
		t.Fatalf("open thread: status %d", rec.Code)
	}
	if n := unread(b); n != 0 {
		t.Errorf("b unread after read: %v, want 0", n)
	}
	if n := unread(a); n != 0 {
		t.Errorf("a unread: %v, want 0", n)
	}
}

func TestAPIReviews(t *testing.T) {
	mux := testAPIServer(t)
	a := newAPIClient(t, mux)
	b := newAPIClient(t, mux)
	aID := signupFull(t, a, "ada@example.com")
	bID := signupFull(t, b, "bo@example.com")
	expressInterest(t, a, bID)
	expressInterest(t, b, aID)
	stayAction(t, b, proposeStay(t, a, bID, 0), "accept")
	_ = aID
	path := "/api/v1/users/" + strconv.FormatInt(bID, 10) + "/reviews"

	put := func(rating int64, body string) *httptest.ResponseRecorder {
		return a.do("PUT", path, map[string]any{"rating": rating, "body": body})
	}
	if rec := put(5, "Tidy and quiet, great flatmate."); rec.Code != http.StatusOK {
		t.Fatalf("review: status %d: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	decodeBody(t, a.do("GET", path, nil), &got)
	if got["count"] != float64(1) || got["avg"] != float64(5) {
		t.Errorf("after review: count=%v avg=%v", got["count"], got["avg"])
	}
	revs, _ := got["reviews"].([]any)
	first, _ := revs[0].(map[string]any)
	if first["author_name"] != "Ada" || first["body"] != "Tidy and quiet, great flatmate." {
		t.Errorf("review fields: %v", first)
	}
	if first["verified_stay"] != true {
		t.Errorf("verified_stay: %v, want true", first["verified_stay"])
	}
	if rec := put(3, "Fine, mostly."); rec.Code != http.StatusOK {
		t.Fatalf("re-review: status %d", rec.Code)
	}
	decodeBody(t, a.do("GET", path, nil), &got)
	if got["count"] != float64(1) || got["avg"] != float64(3) {
		t.Errorf("after upsert: count=%v avg=%v, want 1/3", got["count"], got["avg"])
	}
	if rec := put(6, "x"); rec.Code != http.StatusBadRequest {
		t.Errorf("rating 6: status %d, want 400", rec.Code)
	}
	if rec := put(4, ""); rec.Code != http.StatusBadRequest {
		t.Errorf("empty body: status %d, want 400", rec.Code)
	}
	selfPath := "/api/v1/users/" + strconv.FormatInt(aID, 10) + "/reviews"
	if rec := a.do("PUT", selfPath, map[string]any{"rating": 5, "body": "me"}); rec.Code != http.StatusNotFound {
		t.Errorf("self review: status %d, want 404", rec.Code)
	}
	c := newAPIClient(t, mux)
	cID := signupFull(t, c, "cy@example.com")
	expressInterest(t, c, bID)
	expressInterest(t, b, cID)
	if rec := c.do("PUT", "/api/v1/users/"+strconv.FormatInt(bID, 10)+"/reviews", map[string]any{"rating": 5, "body": "no stay yet"}); rec.Code != http.StatusForbidden {
		t.Errorf("review without stay: status %d, want 403", rec.Code)
	}

	var detail map[string]any
	decodeBody(t, a.do("GET", "/api/v1/matches/"+strconv.FormatInt(bID, 10), nil), &detail)
	if detail["review_count"] != float64(1) || detail["review_avg"] != float64(3) {
		t.Errorf("detail reviews: count=%v avg=%v", detail["review_count"], detail["review_avg"])
	}

	if rec := a.do("POST", "/api/v1/matches/"+strconv.FormatInt(bID, 10)+"/block", map[string]string{}); rec.Code != http.StatusOK {
		t.Fatalf("block: status %d", rec.Code)
	}
	if rec := a.do("GET", path, nil); rec.Code != http.StatusNotFound {
		t.Errorf("blocked reviews read: status %d, want 404", rec.Code)
	}
}

func TestAPIStays(t *testing.T) {
	mux := testAPIServer(t)
	a := newAPIClient(t, mux)
	b := newAPIClient(t, mux)
	aID := signupFull(t, a, "ada@example.com")
	bID := signupFull(t, b, "bo@example.com")
	if rec := a.do("POST", "/api/v1/stays", map[string]any{"peer_id": bID}); rec.Code != http.StatusForbidden {
		t.Fatalf("propose non-mutual: status %d, want 403", rec.Code)
	}
	expressInterest(t, a, bID)
	expressInterest(t, b, aID)
	stayID := proposeStay(t, a, bID, 0)
	if rec := a.do("POST", "/api/v1/stays", map[string]any{"peer_id": bID}); rec.Code != http.StatusConflict {
		t.Errorf("double propose: status %d, want 409", rec.Code)
	}
	if rec := a.do("POST", "/api/v1/stays/"+strconv.FormatInt(stayID, 10)+"/accept", map[string]string{}); rec.Code != http.StatusConflict {
		t.Errorf("self accept: status %d, want 409", rec.Code)
	}
	stayAction(t, b, stayID, "accept")
	var detail map[string]any
	decodeBody(t, a.do("GET", "/api/v1/matches/"+strconv.FormatInt(bID, 10), nil), &detail)
	m, _ := detail["m"].(map[string]any)
	if m["stay_status"] != "active" || m["can_review"] != true {
		t.Errorf("detail stay: status=%v can_review=%v", m["stay_status"], m["can_review"])
	}
	stayAction(t, a, stayID, "end")
	var inbox map[string]any
	decodeBody(t, b.do("GET", "/api/v1/stays", nil), &inbox)
	stays, _ := inbox["stays"].([]any)
	if len(stays) != 1 {
		t.Fatalf("stays: %d, want 1", len(stays))
	}
	first, _ := stays[0].(map[string]any)
	if first["status"] != "ended" || first["peer_name"] != "Ada" {
		t.Errorf("stay row: %v", first)
	}
	_ = aID
}

func TestAPIListingReviews(t *testing.T) {
	mux := testAPIServer(t)
	a := newAPIClient(t, mux)
	b := newAPIClient(t, mux)
	c := newAPIClient(t, mux)
	aID := signupFull(t, a, "ada@example.com")
	bID := signupFull(t, b, "bo@example.com")
	_ = signupFull(t, c, "cy@example.com")
	var created map[string]any
	if rec := a.do("POST", "/api/v1/listings", validListing()); rec.Code != http.StatusCreated {
		t.Fatalf("create: status %d: %s", rec.Code, rec.Body.String())
	} else {
		decodeBody(t, rec, &created)
	}
	lid := int64(created["id"].(float64))
	path := "/api/v1/listings/" + strconv.FormatInt(lid, 10)
	if rec := a.do("PUT", path+"/reviews", map[string]any{"rating": 5, "body": "mine", "kind": "visit"}); rec.Code != http.StatusForbidden {
		t.Errorf("owner review: status %d, want 403", rec.Code)
	}
	expressInterest(t, a, bID)
	expressInterest(t, b, aID)
	if rec := b.do("PUT", path+"/reviews", map[string]any{"rating": 5, "body": "lived here", "kind": "stay"}); rec.Code != http.StatusForbidden {
		t.Errorf("stay review without stay: status %d, want 403", rec.Code)
	}
	if rec := b.do("PUT", path+"/reviews", map[string]any{"rating": 5, "body": "x", "kind": "bogus"}); rec.Code != http.StatusBadRequest {
		t.Errorf("bad kind: status %d, want 400", rec.Code)
	}
	if rec := c.do("PUT", path+"/reviews", map[string]any{"rating": 4, "body": "saw it, nice", "kind": "visit"}); rec.Code != http.StatusOK {
		t.Fatalf("visit review: status %d: %s", rec.Code, rec.Body.String())
	}
	stayID := proposeStay(t, a, bID, lid)
	stayAction(t, b, stayID, "accept")
	var detail map[string]any
	decodeBody(t, a.do("GET", path, nil), &detail)
	l, _ := detail["listing"].(map[string]any)
	if l["status"] != "rented" {
		t.Errorf("listing status after accept: %v, want rented", l["status"])
	}
	if rec := b.do("PUT", path+"/reviews", map[string]any{"rating": 5, "body": "lived here, great", "kind": "stay"}); rec.Code != http.StatusOK {
		t.Fatalf("stay review: status %d: %s", rec.Code, rec.Body.String())
	}
	var revs map[string]any
	decodeBody(t, c.do("GET", path+"/reviews", nil), &revs)
	if revs["stay_count"] != float64(1) || revs["visit_count"] != float64(1) {
		t.Errorf("counts: stay=%v visit=%v, want 1/1", revs["stay_count"], revs["visit_count"])
	}
	if revs["trust_score"] != 4.75 {
		t.Errorf("trust score: %v, want 4.75", revs["trust_score"])
	}
	if rec := a.do("POST", path+"/status", map[string]any{"status": "available"}); rec.Code != http.StatusOK {
		t.Fatalf("relist: status %d: %s", rec.Code, rec.Body.String())
	}
	decodeBody(t, a.do("GET", path, nil), &detail)
	l, _ = detail["listing"].(map[string]any)
	if l["status"] != "available" {
		t.Errorf("listing status after relist: %v, want available", l["status"])
	}
	if rec := b.do("POST", path+"/status", map[string]any{"status": "rented"}); rec.Code != http.StatusNotFound {
		t.Errorf("non-owner status: status %d, want 404", rec.Code)
	}
	_ = aID
}

func TestAPIExpenses(t *testing.T) {
	mux := testAPIServer(t)
	a := newAPIClient(t, mux)
	b := newAPIClient(t, mux)
	aID := signupFull(t, a, "ada@example.com")
	bID := signupFull(t, b, "bo@example.com")
	path := "/api/v1/expenses/" + strconv.FormatInt(bID, 10)
	post := func(body map[string]any) *httptest.ResponseRecorder {
		return a.do("POST", path, body)
	}
	if rec := post(map[string]any{"amount": 10000, "currency": "NGN", "split": "equal"}); rec.Code != http.StatusForbidden {
		t.Errorf("pre-mutual: status %d, want 403", rec.Code)
	}
	expressInterest(t, a, bID)
	expressInterest(t, b, aID)
	if rec := post(map[string]any{"amount": 0, "split": "equal"}); rec.Code != http.StatusBadRequest {
		t.Errorf("zero amount: status %d, want 400", rec.Code)
	}
	if rec := post(map[string]any{"amount": 100, "split": "bogus"}); rec.Code != http.StatusBadRequest {
		t.Errorf("bad split: status %d, want 400", rec.Code)
	}
	if rec := post(map[string]any{"amount": 100, "split": "custom", "payer_share": 30, "peer_share": 50}); rec.Code != http.StatusBadRequest {
		t.Errorf("mismatched shares: status %d, want 400", rec.Code)
	}
	if rec := post(map[string]any{"amount": 10000, "currency": "NGN", "note": "power bill", "split": "equal"}); rec.Code != http.StatusCreated {
		t.Fatalf("create: status %d: %s", rec.Code, rec.Body.String())
	}
	var get map[string]any
	decodeBody(t, a.do("GET", path, nil), &get)
	items, _ := get["expenses"].([]any)
	bals, _ := get["balances"].([]any)
	if len(items) != 1 || len(bals) != 1 {
		t.Fatalf("ledger: %d expenses %d balances", len(items), len(bals))
	}
	b0, _ := bals[0].(map[string]any)
	if b0["currency"] != "NGN" || b0["amount"] != float64(5000) {
		t.Errorf("balance: %v, want NGN 5000", b0)
	}
	rec := b.do("POST", "/api/v1/expenses/"+strconv.FormatInt(aID, 10), map[string]any{
		"amount": 6000, "currency": "NGN", "split": "custom", "payer_share": 2000, "peer_share": 4000,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("peer create: status %d: %s", rec.Code, rec.Body.String())
	}
	decodeBody(t, a.do("GET", path, nil), &get)
	bals, _ = get["balances"].([]any)
	b0, _ = bals[0].(map[string]any)
	if b0["amount"] != float64(1000) {
		t.Errorf("balance after reply: %v, want 1000", b0["amount"])
	}
	c := newAPIClient(t, mux)
	signup(t, c, "cee@example.com")
	items, _ = get["expenses"].([]any)
	first, _ := items[0].(map[string]any)
	delPath := "/api/v1/expenses/item/" + strconv.FormatInt(int64(first["id"].(float64)), 10)
	if rec := c.do("DELETE", delPath, nil); rec.Code != http.StatusNotFound {
		t.Errorf("stranger delete: status %d, want 404", rec.Code)
	}
	if rec := a.do("DELETE", delPath, nil); rec.Code != http.StatusOK {
		t.Fatalf("delete: status %d", rec.Code)
	}
	decodeBody(t, a.do("GET", path, nil), &get)
	items, _ = get["expenses"].([]any)
	if len(items) != 1 {
		t.Errorf("after delete: %d expenses, want 1", len(items))
	}
	if rec := a.do("POST", "/api/v1/matches/"+strconv.FormatInt(bID, 10)+"/block", map[string]string{}); rec.Code != http.StatusOK {
		t.Fatalf("block: status %d", rec.Code)
	}
	if rec := a.do("GET", path, nil); rec.Code != http.StatusNotFound {
		t.Errorf("blocked ledger: status %d, want 404", rec.Code)
	}
}

func TestAPISearches(t *testing.T) {
	mux := testAPIServer(t)
	a := newAPIClient(t, mux)
	b := newAPIClient(t, mux)
	signupFull(t, a, "ada@example.com")
	signupFull(t, b, "bo@example.com")
	postListing := func(c *apiClient, area string, rent int64) {
		l := validListing()
		l["area_text"] = area
		l["monthly_rent"] = rent
		if rec := c.do("POST", "/api/v1/listings", l); rec.Code != http.StatusCreated {
			t.Fatalf("create listing: status %d", rec.Code)
		}
	}
	postListing(a, "Yaba, Lagos", 150000)
	if rec := a.do("POST", "/api/v1/searches", map[string]any{"name": "", "area": "yaba"}); rec.Code != http.StatusBadRequest {
		t.Errorf("empty name: status %d, want 400", rec.Code)
	}
	var created map[string]any
	rec := a.do("POST", "/api/v1/searches", map[string]any{"name": "yaba under 200k", "area": "yaba", "max_rent": 200000})
	if rec.Code != http.StatusCreated {
		t.Fatalf("save search: status %d", rec.Code)
	}
	decodeBody(t, rec, &created)
	time.Sleep(1100 * time.Millisecond) // created_at has 1s resolution
	postListing(b, "Yaba, Lagos", 100000)
	postListing(b, "Brixton, London", 100000)
	postListing(b, "Yaba, Lagos", 500000)
	var list map[string]any
	decodeBody(t, a.do("GET", "/api/v1/searches", nil), &list)
	items, _ := list["searches"].([]any)
	if len(items) != 1 {
		t.Fatalf("searches: %d, want 1", len(items))
	}
	s0, _ := items[0].(map[string]any)
	if s0["new_listings"] != float64(1) {
		t.Errorf("new_listings = %v, want 1", s0["new_listings"])
	}
	delPath := "/api/v1/searches/" + strconv.FormatInt(int64(created["id"].(float64)), 10)
	if rec := a.do("DELETE", delPath, nil); rec.Code != http.StatusOK {
		t.Fatalf("delete: status %d", rec.Code)
	}
	decodeBody(t, a.do("GET", "/api/v1/searches", nil), &list)
	items, _ = list["searches"].([]any)
	if len(items) != 0 {
		t.Errorf("after delete: %d, want 0", len(items))
	}
}

func TestAPIAgreement(t *testing.T) {
	mux := testAPIServer(t)
	a := newAPIClient(t, mux)
	b := newAPIClient(t, mux)
	aID := signupFull(t, a, "ada@example.com")
	bID := signupFull(t, b, "bo@example.com")
	path := "/api/v1/matches/" + strconv.FormatInt(bID, 10) + "/agreement"
	var out map[string]any
	rec := a.do("POST", path, map[string]string{})
	if rec.Code != http.StatusOK {
		t.Fatalf("draft: status %d: %s", rec.Code, rec.Body.String())
	}
	decodeBody(t, rec, &out)
	secs, _ := out["sections"].([]any)
	if len(secs) != 4 {
		t.Fatalf("sections: %d, want 4", len(secs))
	}
	s0, _ := secs[0].(map[string]any)
	if s0["heading"] != "House rules" {
		t.Errorf("first heading: %v", s0["heading"])
	}
	if out["enhanced"] != false {
		t.Errorf("disabled AI: enhanced=%v", out["enhanced"])
	}
	self := "/api/v1/matches/" + strconv.FormatInt(aID, 10) + "/agreement"
	if rec := a.do("POST", self, map[string]string{}); rec.Code != http.StatusNotFound {
		t.Errorf("self: status %d, want 404", rec.Code)
	}
	hidden := validProfile()
	hidden["searching"] = false
	b.do("PUT", "/api/v1/profile", hidden)
	if rec := a.do("POST", path, map[string]string{}); rec.Code != http.StatusNotFound {
		t.Errorf("hidden: status %d, want 404", rec.Code)
	}
}

func TestAPINLSearch(t *testing.T) {
	mux := testAPIServer(t)
	a := newAPIClient(t, mux)
	b := newAPIClient(t, mux)
	c := newAPIClient(t, mux)
	signupFull(t, a, "ada@example.com")
	signupFull(t, b, "bo@example.com")
	signup(t, c, "cee@example.com")
	london := validProfile()
	london["first_name"] = "Cee"
	london["area_text"] = "Brixton, London"
	if rec := c.do("PUT", "/api/v1/profile", london); rec.Code != http.StatusOK {
		t.Fatalf("put profile: status %d", rec.Code)
	}
	if rec := c.do("PUT", "/api/v1/questionnaire", map[string]any{"answers": fullAnswers()}); rec.Code != http.StatusOK {
		t.Fatalf("save answers: status %d", rec.Code)
	}
	search := func(query string) map[string]any {
		var out map[string]any
		rec := a.do("POST", "/api/v1/matches/search", map[string]string{"query": query})
		if rec.Code != http.StatusOK {
			t.Fatalf("search %q: status %d: %s", query, rec.Code, rec.Body.String())
		}
		decodeBody(t, rec, &out)
		return out
	}
	out := search("london")
	if ms, _ := out["matches"].([]any); len(ms) != 1 {
		t.Errorf("london: %d matches, want 1", len(ms))
	}
	if out["enhanced"] != false {
		t.Errorf("disabled AI: enhanced=%v", out["enhanced"])
	}
	out = search("")
	if ms, _ := out["matches"].([]any); len(ms) != 2 {
		t.Errorf("empty: %d matches, want 2", len(ms))
	}
}

func TestNLKeep(t *testing.T) {
	p := &profile.Profile{AreaText: "Yaba, Lagos", BudgetMin: 100}
	a := map[string]profile.Answer{
		"smoking": {Dimension: "smoking", Value: "no"},
		"pets":    {Dimension: "pets", Value: "none"},
		"wfh":     {Dimension: "wfh", Value: "2"},
	}
	cases := []struct {
		name string
		f    ai.SearchFilters
		want bool
	}{
		{"empty", ai.SearchFilters{WFHMax: -1}, true},
		{"area hit", ai.SearchFilters{Area: "yaba", WFHMax: -1}, true},
		{"area miss", ai.SearchFilters{Area: "london", WFHMax: -1}, false},
		{"rent ok", ai.SearchFilters{MaxRent: 150, WFHMax: -1}, true},
		{"rent low", ai.SearchFilters{MaxRent: 50, WFHMax: -1}, false},
		{"non-smoker", ai.SearchFilters{Smoking: "no", WFHMax: -1}, true},
		{"smoker wanted", ai.SearchFilters{Smoking: "yes", WFHMax: -1}, false},
		{"no pets", ai.SearchFilters{Pets: "none", WFHMax: -1}, true},
		{"wfh cap ok", ai.SearchFilters{WFHMax: 2}, true},
		{"wfh cap low", ai.SearchFilters{WFHMax: 1}, false},
	}
	for _, tc := range cases {
		if got := nlKeep(tc.f, p, a); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestAPIContactSelfConfirm(t *testing.T) {
	c := newAPIClient(t, testAPIServer(t))
	signup(t, c, "ada@example.com")
	var get map[string]any
	decodeBody(t, c.do("GET", "/api/v1/profile", nil), &get)
	defs, _ := get["profile"].(map[string]any)
	if defs["contact_self_confirmed"] != false {
		t.Errorf("fresh default: contact_self_confirmed=%v, want false", defs["contact_self_confirmed"])
	}
	p := validProfile()
	p["contact_self_confirmed"] = true
	if rec := c.do("PUT", "/api/v1/profile", p); rec.Code != http.StatusOK {
		t.Fatalf("put: status %d, want 200", rec.Code)
	}
	decodeBody(t, c.do("GET", "/api/v1/profile", nil), &get)
	saved, _ := get["profile"].(map[string]any)
	if saved["contact_self_confirmed"] != true {
		t.Errorf("saved: contact_self_confirmed=%v, want true", saved["contact_self_confirmed"])
	}
}

func TestAPIListingPhotos(t *testing.T) {
	mux := testAPIServer(t)
	a := newAPIClient(t, mux)
	signupFull(t, a, "ada@example.com")

	var created map[string]any
	rec := a.do("POST", "/api/v1/listings", validListing())
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: status %d: %s", rec.Code, rec.Body.String())
	}
	decodeBody(t, rec, &created)
	idPath := "/api/v1/listings/" + strconv.FormatInt(int64(created["id"].(float64)), 10)

	var one map[string]any
	decodeBody(t, a.do("GET", idPath, nil), &one)
	got, _ := one["listing"].(map[string]any)
	if got["photo_url"] != "" {
		t.Errorf("default photo_url=%v, want empty", got["photo_url"])
	}

	withPhoto := validListing()
	withPhoto["photo_url"] = "/placeholders/room-1.svg"
	if rec := a.do("PUT", idPath, withPhoto); rec.Code != http.StatusOK {
		t.Fatalf("set photo: status %d: %s", rec.Code, rec.Body.String())
	}
	decodeBody(t, a.do("GET", idPath, nil), &one)
	got, _ = one["listing"].(map[string]any)
	if got["photo_url"] != "/placeholders/room-1.svg" {
		t.Errorf("photo_url=%v", got["photo_url"])
	}

	bad := validListing()
	bad["photo_url"] = "javascript:alert(1)"
	if rec := a.do("POST", "/api/v1/listings", bad); rec.Code != http.StatusBadRequest {
		t.Errorf("bad scheme create: status %d, want 400", rec.Code)
	}
	bad["photo_url"] = "ftp://x/y.png"
	if rec := a.do("PUT", idPath, bad); rec.Code != http.StatusBadRequest {
		t.Errorf("bad scheme put: status %d, want 400", rec.Code)
	}
}

func TestAPIAdmin(t *testing.T) {
	srv := testAPIServerFull(t)
	mux := srv.NewMux("../../web/static")
	boss := newAPIClient(t, mux)
	user := newAPIClient(t, mux)
	bossID := signupFull(t, boss, "boss@example.com")
	malloryID := signupFull(t, user, "mallory@example.com")
	q := database.New(srv.DB)
	if err := q.SetUserAdmin(context.Background(), database.SetUserAdminParams{IsAdmin: 1, ID: bossID}); err != nil {
		t.Fatal(err)
	}
	var sess map[string]any
	decodeBody(t, boss.do("GET", "/api/v1/session", nil), &sess)
	if sess["is_admin"] != true {
		t.Fatalf("session is_admin=%v, want true", sess["is_admin"])
	}
	if rec := user.do("GET", "/api/v1/admin/reports", nil); rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin reports: status %d, want 403", rec.Code)
	}
	if rec := user.do("GET", "/api/v1/admin/metrics", nil); rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin metrics: status %d, want 403", rec.Code)
	}
	reportURL := "/api/v1/matches/" + strconv.FormatInt(bossID, 10) + "/report"
	if rec := user.do("POST", reportURL, map[string]string{"reason": "spam"}); rec.Code != http.StatusOK {
		t.Fatalf("report: status %d: %s", rec.Code, rec.Body.String())
	}
	var list map[string]any
	decodeBody(t, boss.do("GET", "/api/v1/admin/reports", nil), &list)
	reports, _ := list["reports"].([]any)
	if len(reports) != 1 {
		t.Fatalf("reports=%d, want 1", len(reports))
	}
	rep, _ := reports[0].(map[string]any)
	if rep["reason"] != "spam" || rep["reporter"] != float64(malloryID) || rep["target"] != float64(bossID) {
		t.Fatalf("report fields: %v", rep)
	}
	if rep["reporter_name"] == "" || rep["target_name"] == "" {
		t.Fatalf("names unresolved: %v", rep)
	}
	if rep["resolved_at"] != nil {
		t.Fatalf("resolved_at=%v, want nil", rep["resolved_at"])
	}
	repID := strconv.FormatInt(int64(rep["id"].(float64)), 10)
	resolve := "/api/v1/admin/reports/" + repID + "/resolve"
	if rec := boss.do("POST", resolve, map[string]string{"action": "ban"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad action: status %d, want 400", rec.Code)
	}
	if rec := boss.do("POST", "/api/v1/admin/reports/999/resolve", map[string]string{"action": "dismiss"}); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown report: status %d, want 404", rec.Code)
	}
	if rec := boss.do("POST", resolve, map[string]string{"action": "dismiss"}); rec.Code != http.StatusOK {
		t.Fatalf("dismiss: status %d: %s", rec.Code, rec.Body.String())
	}
	if rec := boss.do("POST", resolve, map[string]string{"action": "dismiss"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("re-resolve: status %d, want 400", rec.Code)
	}
	decodeBody(t, boss.do("GET", "/api/v1/admin/reports", nil), &list)
	rep, _ = list["reports"].([]any)[0].(map[string]any)
	if rep["resolved_at"] == nil || rep["resolution"] != "dismiss" {
		t.Fatalf("after dismiss: %v", rep)
	}
	// Deactivation path: boss reports mallory, then deactivates her.
	malloryURL := "/api/v1/matches/" + strconv.FormatInt(malloryID, 10) + "/report"
	if rec := boss.do("POST", malloryURL, map[string]string{"reason": "abuse"}); rec.Code != http.StatusOK {
		t.Fatalf("report 2: status %d: %s", rec.Code, rec.Body.String())
	}
	decodeBody(t, boss.do("GET", "/api/v1/admin/reports", nil), &list)
	var openID string
	for _, r := range list["reports"].([]any) {
		m, _ := r.(map[string]any)
		if m["resolved_at"] == nil {
			openID = strconv.FormatInt(int64(m["id"].(float64)), 10)
		}
	}
	if openID == "" {
		t.Fatal("no open report found")
	}
	if rec := boss.do("POST", "/api/v1/admin/reports/"+openID+"/resolve", map[string]string{"action": "deactivate"}); rec.Code != http.StatusOK {
		t.Fatalf("deactivate: status %d: %s", rec.Code, rec.Body.String())
	}
	deact, err := q.GetUserDeactivation(context.Background(), malloryID)
	if err != nil || !deact.Valid {
		t.Fatalf("mallory deactivated=%v err=%v, want set", deact, err)
	}
	var metrics map[string]any
	decodeBody(t, boss.do("GET", "/api/v1/admin/metrics", nil), &metrics)
	if metrics["users"] != float64(2) || metrics["active_users"] != float64(1) || metrics["open_reports"] != float64(0) {
		t.Fatalf("metrics=%v", metrics)
	}
	for _, k := range []string{"listings", "reviews", "expenses"} {
		if _, ok := metrics[k]; !ok {
			t.Errorf("metrics missing %q: %v", k, metrics)
		}
	}
	// Self-deactivation guard: a fresh user reports the admin.
	zed := newAPIClient(t, mux)
	signupFull(t, zed, "zed@example.com")
	if rec := zed.do("POST", reportURL, map[string]string{"reason": "x"}); rec.Code != http.StatusOK {
		t.Fatalf("report 3: status %d: %s", rec.Code, rec.Body.String())
	}
	decodeBody(t, boss.do("GET", "/api/v1/admin/reports", nil), &list)
	for _, r := range list["reports"].([]any) {
		m, _ := r.(map[string]any)
		if m["resolved_at"] == nil {
			openID = strconv.FormatInt(int64(m["id"].(float64)), 10)
		}
	}
	if rec := boss.do("POST", "/api/v1/admin/reports/"+openID+"/resolve", map[string]string{"action": "deactivate"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("self-deactivate: status %d, want 400", rec.Code)
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
