package web

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gentcod/flatmate/internal/ai"
	"github.com/gentcod/flatmate/internal/match"
	"github.com/gentcod/flatmate/internal/profile"
)

func (s *Server) handleOnboardingShow(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	data := map[string]any{"Title": "Describe your habits", "Authed": true,
		"Dims": buildQdims(nil), "CSRF": s.csrfToken(w, r)}
	if ob, found, err := profile.GetOnboarding(r.Context(), s.DB, id); err == nil && found {
		data["Text"] = ob.Description
	}
	s.render(w, r, "onboarding", data)
}

func (s *Server) handleOnboardingPost(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	if !s.AILimiter.Allow("user:" + strconv.FormatInt(id, 10)) {
		s.interviewFallback(w, r, id, "", "Slow down a touch — the plain form is below.")
		return
	}
	text := strings.TrimSpace(r.FormValue("message"))
	if text == "" {
		text = strings.TrimSpace(r.FormValue("description")) // opening message field
	}
	if len(text) > 2000 {
		text = string([]rune(text)[:2000])
	}
	if len(text) < 3 {
		s.render(w, r, "onboarding", map[string]any{"Title": "Describe your habits", "Authed": true,
			"Error": "Write a little more (a sentence or two).", "Text": text,
			"Dims": buildQdims(nil), "CSRF": s.csrfToken(w, r)})
		return
	}
	// One shot: a single message, a single extraction. Whatever the
	// model cannot cover, the user answers in the form.
	transcript := "user: " + text + "\n"
	drafts, err := ai.ExtractAnswers(r.Context(), s.AI, transcript)
	if err != nil {
		log.Printf("interview step: %v", err)
		msg := "Couldn't draft answers from that — fill in the form below."
		if errors.Is(err, ai.ErrNotConfigured) {
			msg = "AI chat isn't available right now — the form below works the same."
		}
		s.interviewFallback(w, r, id, transcript, msg)
		return
	}
	seed := map[string]profile.Answer{}
	for _, d := range drafts {
		seed[d.Dimension] = profile.Answer{Dimension: d.Dimension, Value: d.Value, Hard: d.Hard, Weight: d.Weight}
	}
	raw, _ := json.Marshal(drafts)
	profile.SaveOnboarding(r.Context(), s.DB, id, transcript, string(raw))
	s.finishInterview(w, r, id, seed, transcript)
}

// finishInterview ends the one-shot extraction into a prefilled questionnaire.
func (s *Server) finishInterview(w http.ResponseWriter, r *http.Request, id int64, seed map[string]profile.Answer, transcript string) {
	s.renderQuestionnaire(w, r, buildQdims(seed), "", false, transcript,
		"Drafted "+strconv.Itoa(len(seed))+" of "+strconv.Itoa(len(match.Dimensions))+" from your message — answer the rest below.")
}

// interviewFallback drops to the plain form with saved answers.
func (s *Server) interviewFallback(w http.ResponseWriter, r *http.Request, id int64, transcript, msg string) {
	saved, _ := profile.GetAnswers(r.Context(), s.DB, id)
	if saved == nil {
		saved = map[string]profile.Answer{}
	}
	s.renderQuestionnaire(w, r, buildQdims(saved), msg, false, transcript, "")
}
