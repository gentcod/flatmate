package web

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gentcod/flatmate/internal/ai"
	"github.com/gentcod/flatmate/internal/match"
	"github.com/gentcod/flatmate/internal/profile"
)

func (s *Server) handleProfileShow(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	p, found, err := profile.Get(r.Context(), s.DB, id)
	if err != nil {
		http.Error(w, "load error", http.StatusInternalServerError)
		return
	}
	if !found {
		p = &profile.Profile{UserID: id, Mode: profile.ModeHaveRoom, Currency: "USD",
			DurationMonths: 6, GenderPref: profile.GenderPrefAnyone, Searching: true}
	}
	s.renderProfile(w, r, p, "")
}

func (s *Server) renderProfile(w http.ResponseWriter, r *http.Request, p *profile.Profile, errMsg string) {
	data := map[string]any{
		"Title": "Your profile", "Authed": true, "P": p,
		"Relocating": p.Relocating, "CSRF": s.csrfToken(w, r),
	}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	if p.Lat != nil {
		data["Lat"] = strconv.FormatFloat(*p.Lat, 'f', 4, 64)
	}
	if p.Lng != nil {
		data["Lng"] = strconv.FormatFloat(*p.Lng, 'f', 4, 64)
	}
	s.render(w, "profile", data)
}

func (s *Server) handleProfilePost(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	p := &profile.Profile{UserID: id}
	p.FirstName = strings.TrimSpace(r.FormValue("first_name"))
	p.Mode = r.FormValue("mode")
	p.AreaText = strings.TrimSpace(r.FormValue("area_text"))
	p.Relocating = r.FormValue("relocating") == "1"
	p.Currency = strings.ToUpper(strings.TrimSpace(r.FormValue("currency")))
	if p.Currency == "" {
		p.Currency = "USD"
	}
	p.MoveEarliest = strings.TrimSpace(r.FormValue("move_earliest"))
	p.Gender = r.FormValue("gender")
	p.GenderPref = r.FormValue("gender_pref")
	p.Searching = r.FormValue("searching") == "1"
	p.Occupation = strings.TrimSpace(r.FormValue("occupation"))
	p.Bio = strings.TrimSpace(r.FormValue("bio"))
	p.ContactHandle = strings.TrimSpace(r.FormValue("contact_handle"))
	fail := func(msg string) { s.renderProfile(w, r, p, msg) }

	var err error
	if p.DurationMonths, err = strconv.Atoi(strings.TrimSpace(r.FormValue("duration_months"))); err != nil {
		fail("Stay duration must be a whole number of months, e.g. 6.")
		return
	}
	if p.BudgetMin, err = parseMoney(r.FormValue("budget_min")); err != nil {
		fail("Budget must be plain numbers, e.g. 400000.")
		return
	}
	if p.BudgetMax, err = parseMoney(r.FormValue("budget_max")); err != nil {
		fail("Budget must be plain numbers, e.g. 400000.")
		return
	}
	if v := strings.TrimSpace(r.FormValue("lat")); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			fail("Location coordinates are invalid.")
			return
		}
		p.Lat = &f
	}
	if v := strings.TrimSpace(r.FormValue("lng")); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			fail("Location coordinates are invalid.")
			return
		}
		p.Lng = &f
	}
	s.maybeNormalizeArea(r, p)
	if err := profile.Upsert(r.Context(), s.DB, p); err != nil {
		fail(err.Error())
		return
	}
	http.Redirect(w, r, "/questionnaire", http.StatusSeeOther)
}

// maybeNormalizeArea cleans messy area text via the model, keeping the
// user's wording on any failure. The tight timeout and rate limit keep
// saves snappy and cheap.
func (s *Server) maybeNormalizeArea(r *http.Request, p *profile.Profile) {
	if strings.TrimSpace(p.AreaText) == "" {
		return
	}
	if !s.AILimiter.Allow("user:" + strconv.FormatInt(p.UserID, 10)) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if out, err := ai.NormalizeArea(ctx, s.AI, p.AreaText); err == nil && out.AreaText != "" {
		p.AreaText = out.AreaText
	}
}

// parseMoney accepts digits with optional commas/spaces.
func parseMoney(v string) (int64, error) {
	v = strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(v), ",", ""), " ", "")
	return strconv.ParseInt(v, 10, 64)
}

// qdim is one questionnaire row for the template.
type qdim struct {
	Dim      string
	Label    string
	Hint     string
	Question string // AI-drafted scenario prompt, "" for direct axes
	Options  []string
	Value    string
	Hard     bool
	Weight   int
}

// generatedDims are questionnaire axes that get AI-drafted scenario
// questions. Factual or precision-critical axes (sleep, expenses,
// smoking, pets, wfh, employment) keep direct wording: scenarios add
// no signal there and risk muddying exact answers.
var generatedDims = []string{"cleanliness", "noise", "guests", "privacy", "social", "disagreements"}

// questionPrompts drafts one scenario question per generated axis.
// Any failure yields an empty map and the form falls back to labels.
func (s *Server) questionPrompts(r *http.Request, id int64) map[string]string {
	allowed := map[string]bool{}
	for _, d := range generatedDims {
		allowed[d] = true
	}
	if !s.AILimiter.Allow("user:" + strconv.FormatInt(id, 10)) {
		return map[string]string{}
	}
	picked, err := ai.GenerateQuestions(r.Context(), s.AI, generatedDims)
	if err != nil {
		return map[string]string{}
	}
	out := map[string]string{}
	for dim, q := range picked {
		if allowed[dim] {
			out[dim] = q
		}
	}
	return out
}

var qlabels = map[string][2]string{
	"cleanliness":   {"Cleanliness & tidiness", "1 = relaxed, 5 = spotless"},
	"noise":         {"Noise tolerance", "1 = need quiet, 5 = fine with noise"},
	"sleep":         {"Sleep schedule", "When do you usually wind down?"},
	"guests":        {"Guests & visitors", "1 = rarely, 5 = often (including overnight)"},
	"privacy":       {"Need for privacy", "1 = happy sharing space, 5 = need lots of own space"},
	"social":        {"Social home vs quiet home", "What vibe should the flat have?"},
	"expenses":      {"Shared expenses", "How should bills be split?"},
	"disagreements": {"Handling disagreements", "How do you resolve friction?"},
	"smoking":       {"Smoking", "Cigarettes, shisha, and the like."},
	"pets":          {"Pets", "Animals living in the flat."},
	"wfh":           {"Working from home", "Days per week working from the flat (0–5)."},
	"employment":    {"Employment status", "What best describes your work right now?"},
}

func (s *Server) handleQuestionnaireShow(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	saved, err := profile.GetAnswers(r.Context(), s.DB, id)
	if err != nil {
		http.Error(w, "load error", http.StatusInternalServerError)
		return
	}
	notice := ""
	if r.URL.Query().Get("draft") == "1" {
		if iv, ok := s.Interviews.Get(id); ok && len(iv.Drafts) > 0 {
			for k, v := range iv.Drafts {
				saved[k] = v
			}
			notice = "Drafted so far — review, adjust, and save to confirm."
		}
	}
	s.renderQuestionnaire(w, r, buildQdims(saved), "", r.URL.Query().Get("saved") == "1", s.savedDescription(r, id), notice)
}

// savedDescription returns the user's AI onboarding text for reference,
// or "" when they never used the description flow.
func (s *Server) savedDescription(r *http.Request, id int64) string {
	if ob, found, err := profile.GetOnboarding(r.Context(), s.DB, id); err == nil && found {
		return ob.Description
	}
	return ""
}

func buildQdims(saved map[string]profile.Answer) []qdim {
	out := make([]qdim, 0, len(match.Dimensions))
	for _, d := range match.Dimensions {
		q := qdim{Dim: d.Name, Weight: 1}
		if l, ok := qlabels[d.Name]; ok {
			q.Label, q.Hint = l[0], l[1]
		} else {
			q.Label = d.Name
		}
		if d.Type == match.Choice {
			q.Options = d.Choices
		} else {
			for i := d.Min; i <= d.Max; i++ {
				q.Options = append(q.Options, strconv.Itoa(i))
			}
		}
		if a, ok := saved[d.Name]; ok {
			q.Value, q.Hard, q.Weight = a.Value, a.Hard, a.Weight
		}
		out = append(out, q)
	}
	return out
}

func (s *Server) renderQuestionnaire(w http.ResponseWriter, r *http.Request, dims []qdim, errMsg string, saved bool, description, notice string) {
	prompts := s.questionPrompts(r, authedID(r))
	for i := range dims {
		if dims[i].Question == "" {
			if q, ok := prompts[dims[i].Dim]; ok {
				dims[i].Question = q
			}
		}
	}
	data := map[string]any{
		"Title": "Questionnaire", "Authed": true, "Dims": dims, "Saved": saved,
		"Description": description, "CSRF": s.csrfToken(w, r),
	}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	if notice != "" {
		data["Notice"] = notice
	}
	s.render(w, "questionnaire", data)
}

func (s *Server) handleQuestionnairePost(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	var answers []profile.Answer
	dims := buildQdims(nil)
	for i, d := range match.Dimensions {
		v := strings.TrimSpace(r.FormValue("value_" + d.Name))
		wt, err := strconv.Atoi(r.FormValue("weight_" + d.Name))
		if err != nil {
			wt = 1
		}
		dims[i].Value, dims[i].Weight = v, wt
		dims[i].Hard = r.FormValue("hard_"+d.Name) == "1"
		if v == "" {
			s.renderQuestionnaire(w, r, dims, "Answer every question before saving.", false, s.savedDescription(r, id), "")
			return
		}
		answers = append(answers, profile.Answer{
			Dimension: d.Name, Value: v, Hard: dims[i].Hard, Weight: wt,
		})
	}
	if err := profile.SaveAnswers(r.Context(), s.DB, id, answers); err != nil {
		s.renderQuestionnaire(w, r, dims, err.Error(), false, s.savedDescription(r, id), "")
		return
	}
	s.Interviews.Drop(id) // answers confirmed; any interview is done
	http.Redirect(w, r, "/questionnaire?saved=1", http.StatusSeeOther)
}
