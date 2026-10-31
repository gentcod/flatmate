package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/mail"
	"sort"
	"strconv"
	"strings"

	"github.com/gentcod/flatmate/internal/ai"
	"github.com/gentcod/flatmate/internal/auth"
	"github.com/gentcod/flatmate/internal/database"
	"github.com/gentcod/flatmate/internal/match"
	"github.com/gentcod/flatmate/internal/profile"
)

// JSON API for the SPA client. These handlers mirror the form handlers
// (same validation, limiters, and AI touchpoints) but speak JSON:
// successes are objects with a 2xx status, failures are
// {"error": "..."} with 400/401/403/404/409/429/500.

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body.")
		return false
	}
	return true
}

func (s *Server) requireAuthAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := s.userID(r)
		if !ok {
			s.Sessions.ClearCookie(w)
			writeErr(w, http.StatusUnauthorized, "Log in to continue.")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, id)))
	})
}

func (s *Server) checkCSRFAPI(w http.ResponseWriter, r *http.Request) bool {
	if !auth.CheckCSRF(r) {
		writeErr(w, http.StatusForbidden, "Bad form token; reload and try again.")
		return false
	}
	return true
}

// registerAPI wires the /api/v1 routes. Always on, in both UI modes,
// so the Vite dev server can proxy to a legacy-mode Go process.
func (s *Server) registerAPI(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/csrf", s.apiCSRF)
	mux.HandleFunc("GET /api/v1/session", s.apiSession)
	mux.HandleFunc("POST /api/v1/signup", s.apiSignup)
	mux.HandleFunc("POST /api/v1/login", s.apiLogin)
	mux.HandleFunc("POST /api/v1/logout", s.apiLogout)
	mux.Handle("GET /api/v1/profile", s.requireAuthAPI(http.HandlerFunc(s.apiProfileGet)))
	mux.Handle("PUT /api/v1/profile", s.requireAuthAPI(http.HandlerFunc(s.apiProfilePut)))
	mux.Handle("GET /api/v1/questionnaire", s.requireAuthAPI(http.HandlerFunc(s.apiQuestionnaireGet)))
	mux.Handle("PUT /api/v1/questionnaire", s.requireAuthAPI(http.HandlerFunc(s.apiQuestionnairePut)))
	mux.Handle("GET /api/v1/matches", s.requireAuthAPI(http.HandlerFunc(s.apiMatchesList)))
	mux.Handle("GET /api/v1/matches/{id}", s.requireAuthAPI(http.HandlerFunc(s.apiMatchDetail)))
	mux.Handle("POST /api/v1/matches/{id}/save", s.requireAuthAPI(http.HandlerFunc(s.apiSave)))
	mux.Handle("POST /api/v1/matches/{id}/unsave", s.requireAuthAPI(http.HandlerFunc(s.apiUnsave)))
	mux.Handle("POST /api/v1/matches/{id}/dismiss", s.requireAuthAPI(http.HandlerFunc(s.apiDismiss)))
	mux.Handle("POST /api/v1/matches/{id}/interest", s.requireAuthAPI(http.HandlerFunc(s.apiInterest)))
	mux.Handle("POST /api/v1/matches/{id}/block", s.requireAuthAPI(http.HandlerFunc(s.apiBlock)))
	mux.Handle("POST /api/v1/matches/{id}/report", s.requireAuthAPI(http.HandlerFunc(s.apiReport)))
	mux.Handle("GET /api/v1/onboarding", s.requireAuthAPI(http.HandlerFunc(s.apiOnboardingGet)))
	mux.Handle("POST /api/v1/onboarding", s.requireAuthAPI(http.HandlerFunc(s.apiOnboardingPost)))
	mux.Handle("POST /api/v1/account/deactivate", s.requireAuthAPI(http.HandlerFunc(s.apiDeactivate)))
}

// apiCSRF mints the double-submit cookie and hands the token to the
// SPA, which echoes it in the X-CSRF-Token header.
func (s *Server) apiCSRF(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"csrf_token": s.csrfToken(w, r)})
}

func (s *Server) apiSession(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{"authed": false}
	if id, ok := s.userID(r); ok {
		out["authed"] = true
		out["user_id"] = id
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) apiSignup(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	email := auth.NormalizeEmail(in.Email)
	if !s.LoginLimiter.Allow(clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "Too many attempts — try again in a minute.")
		return
	}
	if _, err := mail.ParseAddress(email); err != nil || len(email) > 254 {
		writeErr(w, http.StatusBadRequest, "Enter a valid email address.")
		return
	}
	if len(in.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "Password must be at least 8 characters.")
		return
	}
	hash := auth.HashEmail(email)
	exists, err := database.New(s.DB).CountUsersByEmailHash(r.Context(), hash)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	if exists > 0 {
		writeErr(w, http.StatusConflict, "That email is already registered. Log in instead.")
		return
	}
	pw, err := auth.HashPassword(in.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	res, err := database.New(s.DB).CreateUser(r.Context(), database.CreateUserParams{
		EmailHash: hash, PassHash: pw, CreatedAt: nowRFC3339(),
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeErr(w, http.StatusConflict, "That email is already registered. Log in instead.")
			return
		}
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	tok, err := s.Sessions.Issue(id, sessionTTL)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	s.Sessions.SetCookie(w, tok, sessionTTL)
	writeJSON(w, http.StatusCreated, map[string]any{"user_id": id})
}

func (s *Server) apiLogin(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	email := auth.NormalizeEmail(in.Email)
	if !s.LoginLimiter.Allow(clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "Too many attempts — try again in a minute.")
		return
	}
	u, err := database.New(s.DB).GetUserByEmailHash(r.Context(), auth.HashEmail(email))
	if err != nil || u.DeactivatedAt.Valid || !auth.CheckPassword(u.PassHash, in.Password) {
		writeErr(w, http.StatusUnauthorized, "Invalid email or password.")
		return
	}
	tok, err := s.Sessions.Issue(u.ID, sessionTTL)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "Invalid email or password.")
		return
	}
	s.Sessions.SetCookie(w, tok, sessionTTL)
	writeJSON(w, http.StatusOK, map[string]any{"user_id": u.ID})
}

func (s *Server) apiLogout(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	s.Sessions.ClearCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// apiProfile mirrors profile.Profile for JSON.
type apiProfile struct {
	FirstName      string   `json:"first_name"`
	Mode           string   `json:"mode"`
	AreaText       string   `json:"area_text"`
	Lat            *float64 `json:"lat"`
	Lng            *float64 `json:"lng"`
	Relocating     bool     `json:"relocating"`
	BudgetMin      int64    `json:"budget_min"`
	BudgetMax      int64    `json:"budget_max"`
	Currency       string   `json:"currency"`
	MoveEarliest   string   `json:"move_earliest"`
	MoveLatest     string   `json:"move_latest"`
	DurationMonths int      `json:"duration_months"`
	Gender         string   `json:"gender"`
	GenderPref     string   `json:"gender_pref"`
	Occupation     string   `json:"occupation"`
	Bio            string   `json:"bio"`
	ContactHandle  string   `json:"contact_handle"`
	Searching      bool     `json:"searching"`
}

func toAPIProfile(p *profile.Profile) apiProfile {
	return apiProfile{
		FirstName: p.FirstName, Mode: p.Mode, AreaText: p.AreaText,
		Lat: p.Lat, Lng: p.Lng, Relocating: p.Relocating,
		BudgetMin: p.BudgetMin, BudgetMax: p.BudgetMax, Currency: p.Currency,
		MoveEarliest: p.MoveEarliest, MoveLatest: p.MoveLatest,
		DurationMonths: p.DurationMonths, Gender: p.Gender, GenderPref: p.GenderPref,
		Occupation: p.Occupation, Bio: p.Bio, ContactHandle: p.ContactHandle,
		Searching: p.Searching,
	}
}

func (s *Server) apiProfileGet(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	p, found, err := profile.Get(r.Context(), s.DB, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return
	}
	if !found {
		p = &profile.Profile{UserID: id, Mode: profile.ModeHaveRoom, Currency: "USD",
			DurationMonths: 6, GenderPref: profile.GenderPrefAnyone, Searching: true}
	}
	writeJSON(w, http.StatusOK, map[string]any{"found": found, "profile": toAPIProfile(p)})
}

func (s *Server) apiProfilePut(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	var in apiProfile
	if !decodeJSON(w, r, &in) {
		return
	}
	p := &profile.Profile{UserID: id,
		FirstName: strings.TrimSpace(in.FirstName), Mode: in.Mode,
		AreaText: strings.TrimSpace(in.AreaText), Lat: in.Lat, Lng: in.Lng,
		Relocating: in.Relocating, BudgetMin: in.BudgetMin, BudgetMax: in.BudgetMax,
		Currency:       strings.ToUpper(strings.TrimSpace(in.Currency)),
		MoveEarliest:   strings.TrimSpace(in.MoveEarliest),
		DurationMonths: in.DurationMonths, Gender: in.Gender, GenderPref: in.GenderPref,
		Occupation: strings.TrimSpace(in.Occupation), Bio: strings.TrimSpace(in.Bio),
		ContactHandle: strings.TrimSpace(in.ContactHandle), Searching: in.Searching,
	}
	if p.Currency == "" {
		p.Currency = "USD"
	}
	s.maybeNormalizeArea(r, p)
	if err := profile.Upsert(r.Context(), s.DB, p); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	saved, found, err := profile.Get(r.Context(), s.DB, id)
	if err != nil || !found {
		writeErr(w, http.StatusInternalServerError, "Save succeeded but the profile could not be reloaded.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"found": true, "profile": toAPIProfile(saved)})
}

// apiQdim is one questionnaire row for the client.
type apiQdim struct {
	Dim      string   `json:"dim"`
	Label    string   `json:"label"`
	Hint     string   `json:"hint"`
	Question string   `json:"question"`
	Options  []string `json:"options"`
	Value    string   `json:"value"`
	Hard     bool     `json:"hard"`
	Weight   int      `json:"weight"`
}

// apiAnswer is one stored answer for JSON.
type apiAnswer struct {
	Dimension string `json:"dimension"`
	Value     string `json:"value"`
	Hard      bool   `json:"hard"`
	Weight    int    `json:"weight"`
}

func (s *Server) apiQuestionnaireGet(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	saved, err := profile.GetAnswers(r.Context(), s.DB, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return
	}
	out := map[string]any{"description": s.savedDescription(r, id)}
	if iv, ok := s.Interviews.Get(id); ok && len(iv.Drafts) > 0 {
		for k, v := range iv.Drafts {
			saved[k] = v
		}
		out["notice"] = "Drafted so far — review, adjust, and save to confirm."
	}
	dims := buildQdims(saved)
	prompts := s.questionPrompts(r, id)
	apiDims := make([]apiQdim, 0, len(dims))
	for _, d := range dims {
		q := d.Question
		if q == "" {
			q = prompts[d.Dim]
		}
		apiDims = append(apiDims, apiQdim{Dim: d.Dim, Label: d.Label, Hint: d.Hint,
			Question: q, Options: d.Options, Value: d.Value, Hard: d.Hard, Weight: d.Weight})
	}
	out["dims"] = apiDims
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) apiQuestionnairePut(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	var in struct {
		Answers []apiAnswer `json:"answers"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	byDim := map[string]apiAnswer{}
	for _, a := range in.Answers {
		byDim[a.Dimension] = a
	}
	var answers []profile.Answer
	for _, d := range match.Dimensions {
		a, ok := byDim[d.Name]
		if !ok || strings.TrimSpace(a.Value) == "" {
			writeErr(w, http.StatusBadRequest, "Answer every question before saving.")
			return
		}
		wt := a.Weight
		if wt <= 0 {
			wt = 1
		}
		answers = append(answers, profile.Answer{Dimension: d.Name,
			Value: strings.TrimSpace(a.Value), Hard: a.Hard, Weight: wt})
	}
	if err := profile.SaveAnswers(r.Context(), s.DB, id, answers); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.Interviews.Drop(id) // answers confirmed; any interview is done
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// apiCandidate loads the caller's engine input, reporting which setup
// step is missing with 409 instead of redirecting.
func (s *Server) apiCandidate(w http.ResponseWriter, r *http.Request, id int64) (match.Candidate, bool) {
	p, found, err := profile.Get(r.Context(), s.DB, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return match.Candidate{}, false
	}
	if !found {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "Complete your profile first.", "setup_required": "profile"})
		return match.Candidate{}, false
	}
	a, err := profile.GetAnswers(r.Context(), s.DB, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return match.Candidate{}, false
	}
	if len(a) != len(match.Dimensions) {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "Answer the questionnaire first.", "setup_required": "questionnaire"})
		return match.Candidate{}, false
	}
	return toMatchCandidate(p, a), true
}

type apiMatch struct {
	ID         int64  `json:"id"`
	FirstName  string `json:"first_name"`
	AreaText   string `json:"area_text"`
	Gender     string `json:"gender"`
	Occupation string `json:"occupation"`
	Bio        string `json:"bio"`
	Total      int    `json:"total"`
	TopAlign   string `json:"top_align"`
	Saved      bool   `json:"saved"`
	Mutual     bool   `json:"mutual"`
}

func (s *Server) apiMatchesList(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	me, ok := s.apiCandidate(w, r, id)
	if !ok {
		return
	}
	others := s.loadCandidates(r, id)
	blocked := s.blockSet(r, id)
	acts := s.actionSet(r, id)
	toMe := s.interestToMe(r, id)
	reported := s.reportedByMe(r, id)
	areaFilter := strings.TrimSpace(r.URL.Query().Get("area"))
	area := strings.ToLower(areaFilter)
	views := []apiMatch{}
	for _, c := range others {
		if blocked[c.id] || reported[c.id] || acts[c.id]["dismiss"] {
			continue
		}
		if area != "" && !strings.Contains(strings.ToLower(c.p.AreaText), area) {
			continue
		}
		cand := toMatchCandidate(c.p, c.a)
		if !match.Eligible(me, cand, false, false) {
			continue
		}
		bd := match.Score(me, cand, s.Weights)
		top := ""
		if len(bd.Alignments) > 0 {
			top = bd.Alignments[0]
		}
		views = append(views, apiMatch{
			ID: c.id, FirstName: c.p.FirstName, AreaText: c.p.AreaText,
			Gender: profile.GenderLabel(c.p.Gender), Occupation: c.p.Occupation,
			Bio: c.p.Bio, Total: bd.Total, TopAlign: top,
			Saved: acts[c.id]["save"], Mutual: acts[c.id]["interest"] && toMe[c.id],
		})
	}
	sort.Slice(views, func(i, j int) bool { return views[i].Total > views[j].Total })
	writeJSON(w, http.StatusOK, map[string]any{"matches": views, "empty": len(views) == 0,
		"searching": s.searching(r, id), "area_filter": areaFilter, "nudge": s.profileNudge(r, id)})
}

type apiDetail struct {
	ID         int64  `json:"id"`
	FirstName  string `json:"first_name"`
	AreaText   string `json:"area_text"`
	Gender     string `json:"gender"`
	Occupation string `json:"occupation"`
	Bio        string `json:"bio"`
	ModeLabel  string `json:"mode_label"`
	BudgetStr  string `json:"budget_str"`
	MoveWindow string `json:"move_window"`
	Contact    string `json:"contact"`
	Mutual     bool   `json:"mutual"`
	Saved      bool   `json:"saved"`
	MyInterest bool   `json:"my_interest"`
	Ineligible bool   `json:"ineligible"`
}

type apiDimResult struct {
	Name    string  `json:"name"`
	Score   int     `json:"score"`
	Weight  float64 `json:"weight"`
	Aligned bool    `json:"aligned"`
	Discuss bool    `json:"discuss"`
	Note    string  `json:"note"`
}

type apiBreakdown struct {
	Total         int            `json:"total"`
	Practical     int            `json:"practical"`
	Lifestyle     int            `json:"lifestyle"`
	Interpersonal int            `json:"interpersonal"`
	Dimensions    []apiDimResult `json:"dimensions"`
	Alignments    []string       `json:"alignments"`
	Differences   []string       `json:"differences"`
	Discuss       []string       `json:"discuss"`
}

func toAPIBreakdown(bd match.Breakdown) apiBreakdown {
	dims := make([]apiDimResult, 0, len(bd.Dimensions))
	for _, dr := range bd.Dimensions {
		dims = append(dims, apiDimResult{Name: dr.Name, Score: dr.Score,
			Weight: dr.Weight, Aligned: dr.Aligned, Discuss: dr.Discuss, Note: dr.Note})
	}
	return apiBreakdown{Total: bd.Total, Practical: bd.Practical,
		Lifestyle: bd.Lifestyle, Interpersonal: bd.Interpersonal,
		Dimensions: dims, Alignments: bd.Alignments,
		Differences: bd.Differences, Discuss: bd.Discuss}
}

func (s *Server) apiMatchDetail(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	target, ok := pathID(r)
	if !ok || target == id {
		writeErr(w, http.StatusNotFound, "Not found.")
		return
	}
	me, ok := s.apiCandidate(w, r, id)
	if !ok {
		return
	}
	p, found, err := profile.Get(r.Context(), s.DB, target)
	a, aerr := profile.GetAnswers(r.Context(), s.DB, target)
	if err != nil || aerr != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return
	}
	if !found || len(a) != len(match.Dimensions) ||
		s.blockedPair(r, id, target) || s.deactivated(r, target) || !p.Searching {
		writeErr(w, http.StatusNotFound, "Not found.")
		return
	}
	dv := apiDetail{ID: target, FirstName: p.FirstName, AreaText: p.AreaText,
		Bio: p.Bio, ModeLabel: modeLabel(p.Mode), Gender: profile.GenderLabel(p.Gender), Occupation: p.Occupation,
		BudgetStr:  commas(p.BudgetMin) + " – " + commas(p.BudgetMax) + " " + p.Currency,
		MoveWindow: p.MoveEarliest + " → " + p.MoveLatest + " (" + strconv.Itoa(p.DurationMonths) + " mo)"}
	cand := toMatchCandidate(p, a)
	if !match.Eligible(me, cand, false, false) {
		dv.Ineligible = true
		writeJSON(w, http.StatusOK, map[string]any{"m": dv, "ineligible": true})
		return
	}
	bd := match.Score(me, cand, s.Weights)
	acts := s.actionSet(r, id)
	their, _ := database.New(s.DB).CountInterestFrom(r.Context(), database.CountInterestFromParams{Actor: target, Target: id})
	mutual := acts[target]["interest"] && their > 0
	dv.Saved = acts[target]["save"]
	dv.MyInterest = acts[target]["interest"]
	dv.Mutual = mutual
	if mutual && p.ContactHandle != "" {
		dv.Contact = p.ContactHandle
		database.New(s.DB).AddAction(r.Context(), database.AddActionParams{
			Actor: id, Target: target, Kind: "reveal", CreatedAt: nowRFC3339(),
		})
	}
	out := map[string]any{"m": dv, "bd": toAPIBreakdown(bd), "top": topStrengths(bd)}
	var flagged []string
	for _, dr := range bd.Dimensions {
		if dr.Discuss && dr.Name != "budget" && dr.Name != "area" && dr.Name != "dates" {
			flagged = append(flagged, dr.Name)
		}
	}
	if s.AILimiter.Allow("user:" + strconv.FormatInt(id, 10)) {
		if bullets, _, ok := ai.ExplainMatch(r.Context(), s.AI, p.FirstName, bd); ok {
			out["bullets"], out["enhanced"] = bullets, true
			if qs, ok := ai.AskDiscuss(r.Context(), s.AI, flagged); ok {
				out["discuss_q"] = qs
			}
			if br, ok := ai.SuggestBridges(r.Context(), s.AI, flagged); ok {
				out["bridges"] = br
			}
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) apiActionTarget(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	id := authedID(r)
	target, ok := pathID(r)
	if !ok || target == id {
		writeErr(w, http.StatusNotFound, "Not found.")
		return 0, 0, false
	}
	return id, target, true
}

func (s *Server) apiStampAction(w http.ResponseWriter, r *http.Request, kind string) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id, target, ok := s.apiActionTarget(w, r)
	if !ok {
		return
	}
	database.New(s.DB).AddAction(r.Context(), database.AddActionParams{
		Actor: id, Target: target, Kind: kind, CreatedAt: nowRFC3339(),
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) apiSave(w http.ResponseWriter, r *http.Request) { s.apiStampAction(w, r, "save") }
func (s *Server) apiInterest(w http.ResponseWriter, r *http.Request) {
	s.apiStampAction(w, r, "interest")
}
func (s *Server) apiDismiss(w http.ResponseWriter, r *http.Request) {
	s.apiStampAction(w, r, "dismiss")
}

func (s *Server) apiUnsave(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id, target, ok := s.apiActionTarget(w, r)
	if !ok {
		return
	}
	database.New(s.DB).RemoveAction(r.Context(), database.RemoveActionParams{
		Actor: id, Target: target, Kind: "save",
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) apiBlock(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id, target, ok := s.apiActionTarget(w, r)
	if !ok {
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Block error.")
		return
	}
	q := database.New(s.DB).WithTx(tx)
	q.AddBlock(r.Context(), database.AddBlockParams{
		Blocker: id, Blocked: target, CreatedAt: nowRFC3339(),
	})
	q.DeleteActionsBetween(r.Context(), database.DeleteActionsBetweenParams{Actor: id, Target: target})
	q.DeleteActionsBetween(r.Context(), database.DeleteActionsBetweenParams{Actor: target, Target: id})
	if err := tx.Commit(); err != nil {
		writeErr(w, http.StatusInternalServerError, "Block error.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) apiReport(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id, target, ok := s.apiActionTarget(w, r)
	if !ok {
		return
	}
	var in struct {
		Reason string `json:"reason"`
	}
	// A missing body just means "no reason given", like the form.
	if r.ContentLength != 0 {
		if !decodeJSON(w, r, &in) {
			return
		}
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		reason = "No reason given"
	}
	if len(reason) > 500 {
		reason = reason[:500]
	}
	n, _ := database.New(s.DB).CountReport(r.Context(), database.CountReportParams{Reporter: id, Target: target})
	if n == 0 {
		database.New(s.DB).AddReport(r.Context(), database.AddReportParams{
			Reporter: id, Target: target, Reason: reason, CreatedAt: nowRFC3339(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) apiOnboardingGet(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	out := map[string]any{"total": len(match.Dimensions), "active": false,
		"turns": []map[string]string{}, "drafts_done": 0, "text": ""}
	if ob, found, err := profile.GetOnboarding(r.Context(), s.DB, id); err == nil && found {
		out["text"] = ob.Description
		if ob.Drafts != "" {
			var drafts []apiAnswer
			if json.Unmarshal([]byte(ob.Drafts), &drafts) == nil {
				out["drafts"] = drafts
			}
		}
	}
	if iv, ok := s.Interviews.Get(id); ok && len(iv.Turns) > 0 {
		out["active"] = true
		out["turns"] = toAPITurns(iv.Turns)
		out["drafts_done"] = len(iv.Drafts)
	}
	writeJSON(w, http.StatusOK, out)
}

func toAPITurns(turns []Turn) []map[string]string {
	out := make([]map[string]string, 0, len(turns))
	for _, t := range turns {
		out = append(out, map[string]string{"role": t.Role, "text": t.Text})
	}
	return out
}

func toAPISeed(seed map[string]profile.Answer) map[string]apiAnswer {
	out := map[string]apiAnswer{}
	for k, v := range seed {
		out[k] = apiAnswer{Dimension: v.Dimension, Value: v.Value, Hard: v.Hard, Weight: v.Weight}
	}
	return out
}

func (s *Server) apiOnboardingPost(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	if !s.AILimiter.Allow("user:" + strconv.FormatInt(id, 10)) {
		s.apiInterviewFallback(w, r, id, "", "Slow down a touch — the plain form is below.")
		return
	}
	var in struct {
		Message string `json:"message"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	text := strings.TrimSpace(in.Message)
	if len(text) > 2000 {
		text = text[:2000]
	}
	if len(text) < 3 {
		if _, ok := s.Interviews.Get(id); !ok {
			writeErr(w, http.StatusBadRequest, "Write a little more (a sentence or two).")
			return
		}
		s.apiInterviewState(w, r, id, "Say a little more than that.")
		return
	}
	// Append the user turn, then run one interview step over the transcript.
	var transcript string
	var covered []string
	s.Interviews.Update(id, func(iv *Interview) {
		iv.Turns = append(iv.Turns, Turn{Role: "user", Text: text})
		iv.UserTurns++
		transcript = formatTranscript(iv.Turns)
		for dim := range iv.Drafts {
			covered = append(covered, dim)
		}
	})
	question, drafts, done, err := ai.InterviewStep(r.Context(), s.AI, transcript, covered)
	if err != nil {
		s.apiInterviewFallback(w, r, id, transcript, "Couldn't understand that — review the form below.")
		return
	}
	merged := map[string]profile.Answer{}
	s.Interviews.Update(id, func(iv *Interview) {
		for _, d := range drafts {
			iv.Drafts[d.Dimension] = profile.Answer{Dimension: d.Dimension, Value: d.Value, Hard: d.Hard, Weight: d.Weight}
		}
		if !done {
			iv.Turns = append(iv.Turns, Turn{Role: "assistant", Text: question})
		}
		transcript = formatTranscript(iv.Turns)
		for k, v := range iv.Drafts {
			merged[k] = v
		}
	})
	raw, _ := json.Marshal(drafts)
	profile.SaveOnboarding(r.Context(), s.DB, id, transcript, string(raw))
	if done || s.interviewCapped(id) {
		s.Interviews.Drop(id)
		writeJSON(w, http.StatusOK, map[string]any{
			"done": true, "seed": toAPISeed(merged), "transcript": transcript,
			"message": "Pre-filled from your conversation — review, adjust, and save to confirm.",
		})
		return
	}
	s.apiInterviewState(w, r, id, "")
}

// apiInterviewState returns the live conversation for the chat UI.
func (s *Server) apiInterviewState(w http.ResponseWriter, r *http.Request, id int64, errMsg string) {
	out := map[string]any{"done": false, "total": len(match.Dimensions),
		"turns": []map[string]string{}, "drafts_done": 0}
	if iv, ok := s.Interviews.Get(id); ok {
		out["turns"] = toAPITurns(iv.Turns)
		out["drafts_done"] = len(iv.Drafts)
	}
	if errMsg != "" {
		out["error"] = errMsg
	}
	writeJSON(w, http.StatusOK, out)
}

// apiInterviewFallback drops to the plain form, keeping whatever the
// conversation established so far as a seed.
func (s *Server) apiInterviewFallback(w http.ResponseWriter, r *http.Request, id int64, transcript, msg string) {
	saved, _ := profile.GetAnswers(r.Context(), s.DB, id)
	if saved == nil {
		saved = map[string]profile.Answer{}
	}
	if iv, ok := s.Interviews.Get(id); ok {
		for k, v := range iv.Drafts {
			saved[k] = v
		}
		if transcript == "" {
			transcript = formatTranscript(iv.Turns)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"fallback": true, "message": msg,
		"seed": toAPISeed(saved), "transcript": transcript,
	})
}

func (s *Server) apiDeactivate(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	var in struct {
		Confirm bool `json:"confirm"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if !in.Confirm {
		writeErr(w, http.StatusBadRequest, "Confirmation required.")
		return
	}
	database.New(s.DB).DeactivateUser(r.Context(), database.DeactivateUserParams{
		DeactivatedAt: sql.NullString{String: nowRFC3339(), Valid: true}, ID: id,
	})
	s.Sessions.ClearCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
