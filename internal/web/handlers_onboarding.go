package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gentcod/flatmate/internal/ai"
	"github.com/gentcod/flatmate/internal/match"
	"github.com/gentcod/flatmate/internal/profile"
)

// maxInterviewTurns caps user messages per interview; the last turn
// always finishes into the questionnaire.
const maxInterviewTurns = 6

func (s *Server) handleOnboardingShow(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	if iv, ok := s.Interviews.Get(id); ok && len(iv.Turns) > 0 {
		s.renderInterview(w, r, id, "")
		return
	}
	data := map[string]any{"Title": "Describe your habits", "Authed": true,
		"Dims": buildQdims(nil), "CSRF": s.csrfToken(w, r)}
	if ob, found, err := profile.GetOnboarding(r.Context(), s.DB, id); err == nil && found {
		data["Text"] = ob.Description
	}
	s.render(w, "onboarding", data)
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
	if len(text) < 3 {
		if _, ok := s.Interviews.Get(id); !ok {
			s.render(w, "onboarding", map[string]any{"Title": "Describe your habits", "Authed": true,
				"Error": "Write a little more (a sentence or two).", "Text": text,
				"Dims": buildQdims(nil), "CSRF": s.csrfToken(w, r)})
			return
		}
		s.renderInterview(w, r, id, "Say a little more than that.")
		return
	}
	if len(text) > 2000 {
		text = text[:2000]
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
		s.interviewFallback(w, r, id, transcript, "Couldn't understand that — review the form below.")
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
		s.finishInterview(w, r, id, merged, transcript)
		return
	}
	s.renderInterview(w, r, id, "")
}

// renderInterview shows the active conversation plus the reply box.
func (s *Server) renderInterview(w http.ResponseWriter, r *http.Request, id int64, errMsg string) {
	data := map[string]any{"Title": "Describe your habits", "Authed": true,
		"Dims": buildQdims(nil), "CSRF": s.csrfToken(w, r),
		"Interview": true, "Total": len(match.Dimensions)}
	if iv, ok := s.Interviews.Get(id); ok {
		data["Turns"] = iv.Turns
		data["Done"] = len(iv.Drafts)
	}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	s.render(w, "onboarding", data)
}

// finishInterview ends the conversation into a prefilled questionnaire.
func (s *Server) finishInterview(w http.ResponseWriter, r *http.Request, id int64, seed map[string]profile.Answer, transcript string) {
	s.Interviews.Drop(id)
	s.renderQuestionnaire(w, r, buildQdims(seed), "", false, transcript,
		"Pre-filled from your conversation — review, adjust, and save to confirm.")
}

// interviewFallback drops to the plain form, keeping whatever the
// conversation established so far.
func (s *Server) interviewFallback(w http.ResponseWriter, r *http.Request, id int64, transcript, msg string) {
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
	s.renderQuestionnaire(w, r, buildQdims(saved), msg, false, transcript, "")
}

func (s *Server) interviewCapped(id int64) bool {
	iv, ok := s.Interviews.Get(id)
	return ok && iv.UserTurns >= maxInterviewTurns
}
