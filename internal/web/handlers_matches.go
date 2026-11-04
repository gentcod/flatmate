package web

import (
	"database/sql"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gentcod/flatmate/internal/ai"
	"github.com/gentcod/flatmate/internal/database"
	"github.com/gentcod/flatmate/internal/match"
	"github.com/gentcod/flatmate/internal/profile"
)

type matchView struct {
	ID         int64
	FirstName  string
	AreaText   string
	Gender     string
	Occupation string
	Bio        string
	Total      int
	TopAlign   string
	Saved      bool
	Mutual     bool
}

type detailView struct {
	ID          int64
	FirstName   string
	AreaText    string
	Gender      string
	Occupation  string
	Bio         string
	ModeLabel   string
	BudgetStr   string
	MoveWindow  string
	Contact     string
	ContactSelf bool
	Mutual      bool
	Saved       bool
	MyInterest  bool
	Ineligible  bool
	StayID      int64
	StayStatus  string
	StayByMe    bool
	CanReview   bool
}

type candidateBundle struct {
	id int64
	p  *profile.Profile
	a  map[string]profile.Answer
}

func (s *Server) handleMatchesShow(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	me, ok := s.myCandidate(w, r, id)
	if !ok {
		return
	}
	others := s.loadCandidates(r, id)
	blocked := s.blockSet(r, id)
	acts := s.actionSet(r, id)
	toMe := s.interestToMe(r, id)
	reported := s.reportedByMe(r, id)
	area := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("area")))
	filters, nlQuery := s.nlFilters(r, id)
	var views []matchView
	for _, c := range others {
		if blocked[c.id] || reported[c.id] || acts[c.id]["dismiss"] {
			continue
		}
		if area != "" && !strings.Contains(strings.ToLower(c.p.AreaText), area) || !nlKeep(filters, c.p, c.a) {
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
		views = append(views, matchView{
			ID: c.id, FirstName: c.p.FirstName, AreaText: c.p.AreaText,
			Gender: profile.GenderLabel(c.p.Gender), Occupation: c.p.Occupation,
			Bio: c.p.Bio, Total: bd.Total, TopAlign: top,
			Saved: acts[c.id]["save"], Mutual: acts[c.id]["interest"] && toMe[c.id],
		})
	}
	sort.Slice(views, func(i, j int) bool { return views[i].Total > views[j].Total })
	s.render(w, r, "matches", map[string]any{
		"Title": "Matches", "Authed": true, "Matches": views, "Empty": len(views) == 0,
		"Searching": s.searching(r, id), "AreaFilter": strings.TrimSpace(r.URL.Query().Get("area")),
		"Nudge": s.profileNudge(r, id), "NLQuery": nlQuery, "NLDesc": nlDescribe(filters),
		"Proposals": s.stayProposals(r, id), "CSRF": s.csrfToken(w, r),
	})
}

func (s *Server) handleMatchDetail(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	target, ok := pathID(r)
	if !ok || target == id {
		http.NotFound(w, r)
		return
	}
	me, ok := s.myCandidate(w, r, id)
	if !ok {
		return
	}
	p, found, err := profile.Get(r.Context(), s.DB, target)
	a, aerr := profile.GetAnswers(r.Context(), s.DB, target)
	if err != nil || aerr != nil {
		http.Error(w, "load error", http.StatusInternalServerError)
		return
	}
	if !found || len(a) != len(match.Dimensions) ||
		s.blockedPair(r, id, target) || s.deactivated(r, target) || !p.Searching {
		http.NotFound(w, r)
		return
	}
	dv := detailView{ID: target, FirstName: p.FirstName, AreaText: p.AreaText,
		Bio: p.Bio, ModeLabel: modeLabel(p.Mode), Gender: profile.GenderLabel(p.Gender), Occupation: p.Occupation,
		BudgetStr:  commas(p.BudgetMin) + " – " + commas(p.BudgetMax) + " " + p.Currency,
		MoveWindow: p.MoveEarliest + " → " + p.MoveLatest + " (" + strconv.Itoa(p.DurationMonths) + " mo)"}
	data := map[string]any{"Title": p.FirstName, "Authed": true, "M": dv, "CSRF": s.csrfToken(w, r)}
	if revs, err := database.New(s.DB).ListReviewsForUser(r.Context(), target); err == nil && len(revs) > 0 {
		views := make([]reviewView, 0, len(revs))
		for _, rv := range revs {
			views = append(views, reviewView{AuthorName: s.authorName(r, rv.Author), Rating: rv.Rating, Body: rv.Body, VerifiedStay: s.verifiedStay(r, rv.Author, target)})
		}
		data["Reviews"] = views
		data["ReviewAvg"] = strconv.FormatFloat(reviewAvg(revs), 'f', 1, 64)
		data["ReviewCount"] = len(revs)
	}
	if msg := strings.TrimSpace(r.URL.Query().Get("err")); msg != "" {
		data["Error"] = msg
	}
	cand := toMatchCandidate(p, a)
	if !match.Eligible(me, cand, false, false) {
		dv.Ineligible = true
		data["M"] = dv
		s.render(w, r, "match_detail", data)
		return
	}
	bd := match.Score(me, cand, s.Weights)
	data["Top"] = topStrengths(bd)
	acts := s.actionSet(r, id)
	their, _ := database.New(s.DB).CountInterestFrom(r.Context(), database.CountInterestFromParams{Actor: target, Target: id})
	mutual := acts[target]["interest"] && their > 0
	dv.Saved = acts[target]["save"]
	dv.MyInterest = acts[target]["interest"]
	dv.Mutual = mutual
	if st, ok := s.stayBetween(r, id, target); ok {
		dv.StayID, dv.StayStatus = st.ID, st.Status
		dv.StayByMe = st.ProposedBy == id
		dv.CanReview = st.Status == StayActive || st.Status == StayEnded
	}
	if mutual && p.ContactHandle != "" {
		dv.Contact = p.ContactHandle
		dv.ContactSelf = p.ContactSelfConfirmed
		database.New(s.DB).AddAction(r.Context(), database.AddActionParams{
			Actor: id, Target: target, Kind: "reveal", CreatedAt: nowRFC3339(),
		})
	}
	var flagged []string
	for _, dr := range bd.Dimensions {
		if dr.Discuss && dr.Name != "budget" && dr.Name != "area" && dr.Name != "dates" {
			flagged = append(flagged, dr.Name)
		}
	}
	if s.AILimiter.Allow("user:" + strconv.FormatInt(id, 10)) {
		if bullets, _, ok := ai.ExplainMatch(r.Context(), s.AI, p.FirstName, bd); ok {
			data["Bullets"], data["Enhanced"] = bullets, true
			if qs, ok := ai.AskDiscuss(r.Context(), s.AI, flagged); ok {
				data["DiscussQ"] = qs
			}
			if br, ok := ai.SuggestBridges(r.Context(), s.AI, flagged); ok {
				data["Bridges"] = br
			}
		}
	}
	data["M"] = dv
	data["BD"] = bd
	s.render(w, r, "match_detail", data)
}

// myCandidate loads the caller's engine input, redirecting to complete
// profile or questionnaire first.
func (s *Server) myCandidate(w http.ResponseWriter, r *http.Request, id int64) (match.Candidate, bool) {
	p, found, err := profile.Get(r.Context(), s.DB, id)
	if err != nil {
		http.Error(w, "load error", http.StatusInternalServerError)
		return match.Candidate{}, false
	}
	if !found {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return match.Candidate{}, false
	}
	a, err := profile.GetAnswers(r.Context(), s.DB, id)
	if err != nil {
		http.Error(w, "load error", http.StatusInternalServerError)
		return match.Candidate{}, false
	}
	if len(a) != len(match.Dimensions) {
		http.Redirect(w, r, "/questionnaire", http.StatusSeeOther)
		return match.Candidate{}, false
	}
	return toMatchCandidate(p, a), true
}

func toMatchCandidate(p *profile.Profile, a map[string]profile.Answer) match.Candidate {
	c := match.Candidate{
		UserID: p.UserID, AreaText: p.AreaText, Relocating: p.Relocating,
		BudgetMin: p.BudgetMin, BudgetMax: p.BudgetMax,
		MoveEarliest: p.MoveEarliest, MoveLatest: p.MoveLatest,
		Gender: p.Gender, GenderPref: p.GenderPref,
		Answers: map[string]match.Response{},
	}
	if p.Lat != nil && p.Lng != nil {
		c.HasCoords = true
		c.Lat, c.Lng = *p.Lat, *p.Lng
	}
	for dim, ans := range a {
		c.Answers[dim] = match.Response{Value: ans.Value, Hard: ans.Hard, Weight: ans.Weight}
	}
	return c
}

func (s *Server) loadCandidates(r *http.Request, me int64) []candidateBundle {
	ids, err := database.New(s.DB).ActiveUserIDs(r.Context(), me)
	if err != nil {
		return nil
	}
	var out []candidateBundle
	for _, id := range ids {
		p, found, err := profile.Get(r.Context(), s.DB, id)
		if err != nil || !found {
			continue
		}
		if !p.Searching {
			continue // hidden users never appear in matching
		}
		a, err := profile.GetAnswers(r.Context(), s.DB, id)
		if err != nil || len(a) != len(match.Dimensions) {
			continue
		}
		out = append(out, candidateBundle{id: id, p: p, a: a})
	}
	return out
}

// profileNudge suggests the next optional profile field worth filling,
// or "" when the profile is complete.
func (s *Server) profileNudge(r *http.Request, id int64) string {
	p, found, err := profile.Get(r.Context(), s.DB, id)
	if err != nil || !found {
		return ""
	}
	switch {
	case strings.TrimSpace(p.Bio) == "":
		return "Add a short bio so matches know who you are."
	case strings.TrimSpace(p.Occupation) == "":
		return "Add your occupation — it's shown to your matches."
	case strings.TrimSpace(p.ContactHandle) == "":
		return "Add a contact handle so mutual matches can reach you."
	}
	return ""
}

// searching reports whether the user is visible in matching.
func (s *Server) searching(r *http.Request, id int64) bool {
	p, found, err := profile.Get(r.Context(), s.DB, id)
	return err == nil && found && p.Searching
}

func (s *Server) blockSet(r *http.Request, me int64) map[int64]bool {
	out := map[int64]bool{}
	q := database.New(s.DB)
	if rows, err := q.ListBlocksByBlocker(r.Context(), me); err == nil {
		for _, b := range rows {
			out[b.Blocked] = true
		}
	}
	if rows, err := q.ListBlocksByBlocked(r.Context(), me); err == nil {
		for _, b := range rows {
			out[b.Blocker] = true
		}
	}
	return out
}

func (s *Server) blockedPair(r *http.Request, a, b int64) bool {
	q := database.New(s.DB)
	fwd, _ := q.CountBlockDirection(r.Context(), database.CountBlockDirectionParams{Blocker: a, Blocked: b})
	rev, _ := q.CountBlockDirection(r.Context(), database.CountBlockDirectionParams{Blocker: b, Blocked: a})
	return fwd+rev > 0
}

func (s *Server) deactivated(r *http.Request, id int64) bool {
	d, err := database.New(s.DB).GetUserDeactivation(r.Context(), id)
	if err != nil {
		return true
	}
	return d.Valid
}

func (s *Server) actionSet(r *http.Request, me int64) map[int64]map[string]bool {
	out := map[int64]map[string]bool{}
	rows, err := database.New(s.DB).ListActionsByActor(r.Context(), me)
	if err != nil {
		return out
	}
	for _, a := range rows {
		if out[a.Target] == nil {
			out[a.Target] = map[string]bool{}
		}
		out[a.Target][a.Kind] = true
	}
	return out
}

func (s *Server) interestToMe(r *http.Request, me int64) map[int64]bool {
	out := map[int64]bool{}
	actors, err := database.New(s.DB).ListInterestActors(r.Context(), me)
	if err != nil {
		return out
	}
	for _, actor := range actors {
		out[actor] = true
	}
	return out
}

func (s *Server) reportedByMe(r *http.Request, me int64) map[int64]bool {
	out := map[int64]bool{}
	targets, err := database.New(s.DB).ListReportTargets(r.Context(), me)
	if err != nil {
		return out
	}
	for _, target := range targets {
		out[target] = true
	}
	return out
}

func pathID(r *http.Request) (int64, bool) {
	n, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return n, err == nil && n > 0
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	s.stampAction(w, r, "save", true)
}

func (s *Server) handleInterest(w http.ResponseWriter, r *http.Request) {
	s.stampAction(w, r, "interest", true)
}

func (s *Server) handleDismiss(w http.ResponseWriter, r *http.Request) {
	s.stampAction(w, r, "dismiss", false)
}

func (s *Server) stampAction(w http.ResponseWriter, r *http.Request, kind string, detail bool) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	target, ok := pathID(r)
	if !ok || target == id {
		http.NotFound(w, r)
		return
	}
	if kind == "interest" {
		if n, _ := database.New(s.DB).CountInterestFrom(r.Context(), database.CountInterestFromParams{Actor: id, Target: target}); n == 0 {
			s.notify(r, target, "interest", id, id, s.authorName(r, id)+" showed interest in you.")
		}
	}
	database.New(s.DB).AddAction(r.Context(), database.AddActionParams{
		Actor: id, Target: target, Kind: kind, CreatedAt: nowRFC3339(),
	})
	if detail {
		http.Redirect(w, r, "/match/"+strconv.FormatInt(target, 10), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/matches", http.StatusSeeOther)
	}
}

func (s *Server) handleUnsave(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	target, ok := pathID(r)
	if !ok || target == id {
		http.NotFound(w, r)
		return
	}
	database.New(s.DB).RemoveAction(r.Context(), database.RemoveActionParams{
		Actor: id, Target: target, Kind: "save",
	})
	http.Redirect(w, r, "/match/"+strconv.FormatInt(target, 10), http.StatusSeeOther)
}

func (s *Server) handleBlock(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	target, ok := pathID(r)
	if !ok || target == id {
		http.NotFound(w, r)
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "block error", http.StatusInternalServerError)
		return
	}
	q := database.New(s.DB).WithTx(tx)
	q.AddBlock(r.Context(), database.AddBlockParams{
		Blocker: id, Blocked: target, CreatedAt: nowRFC3339(),
	})
	q.DeleteActionsBetween(r.Context(), database.DeleteActionsBetweenParams{Actor: id, Target: target})
	q.DeleteActionsBetween(r.Context(), database.DeleteActionsBetweenParams{Actor: target, Target: id})
	if err := tx.Commit(); err != nil {
		http.Error(w, "block error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/matches", http.StatusSeeOther)
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	target, ok := pathID(r)
	if !ok || target == id {
		http.NotFound(w, r)
		return
	}
	reason := strings.TrimSpace(r.FormValue("reason"))
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
	http.Redirect(w, r, "/matches", http.StatusSeeOther)
}

func (s *Server) handleDeactivate(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	if r.FormValue("confirm") != "1" {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}
	database.New(s.DB).DeactivateUser(r.Context(), database.DeactivateUserParams{
		DeactivatedAt: sql.NullString{String: nowRFC3339(), Valid: true}, ID: id,
	})
	s.Sessions.ClearCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// topStrengths names the three highest-scoring dimensions behind a
// total. Ties break toward higher answer weight. Display only;
// scoring is untouched.
func topStrengths(bd match.Breakdown) []string {
	dims := append([]match.DimensionResult(nil), bd.Dimensions...)
	sort.Slice(dims, func(i, j int) bool {
		if dims[i].Score != dims[j].Score {
			return dims[i].Score > dims[j].Score
		}
		return dims[i].Weight > dims[j].Weight
	})
	var out []string
	for _, dr := range dims {
		if len(out) == 3 {
			break
		}
		out = append(out, dr.Note)
	}
	return out
}

func modeLabel(m string) string {
	switch m {
	case profile.ModeHaveRoom:
		return "Has a room"
	case profile.ModeNeedRoom:
		return "Needs a room"
	case profile.ModeFindTogether:
		return "Wants to find a place together"
	}
	return m
}

func commas(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	var out []byte
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
