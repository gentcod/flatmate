package web

import (
	"io"
	"testing"

	"github.com/gentcod/flatmate/internal/ai"
	"github.com/gentcod/flatmate/internal/match"
	"github.com/gentcod/flatmate/internal/profile"
)

// TestTemplatesRender executes every page with representative data.
// Template field names are unchecked at compile time, so this guards
// the handler/view/template contract (e.g. .P.DurationMonths, .M.Gender).
func TestTemplatesRender(t *testing.T) {
	pages, err := LoadPages("../../web/templates")
	if err != nil {
		t.Fatal(err)
	}
	p := &profile.Profile{UserID: 1, FirstName: "Ada", Mode: profile.ModeHaveRoom,
		AreaText: "Yaba", BudgetMin: 1, BudgetMax: 2, Currency: "NGN",
		MoveEarliest: "2026-12-01", MoveLatest: "2027-06-01", DurationMonths: 6,
		Gender: profile.GenderWoman, GenderPref: profile.GenderPrefAnyone,
		Occupation: "Nurse", Bio: "b", ContactHandle: "c", Searching: true}
	bd := match.Breakdown{Total: 80, Practical: 80, Lifestyle: 80, Interpersonal: 80,
		Dimensions:  []match.DimensionResult{{Name: "sleep", Score: 100, Weight: 1, Aligned: true, Note: "x"}},
		Alignments:  []string{"a"},
		Differences: []string{"d"},
		Discuss:     []string{"t"}}
	cases := map[string]map[string]any{
		"signup": {"Title": "Sign up", "CSRF": "t"},
		"login":  {"Title": "Log in", "Email": "a@b.c", "CSRF": "t"},
		"profile": {"Title": "Your profile", "Authed": true, "P": p,
			"Relocating": false, "CSRF": "t"},
		"questionnaire": {"Title": "q", "Authed": true, "Saved": true, "Description": "early riser",
			"CSRF": "t",
			"Dims": []qdim{{Dim: "sleep", Label: "Sleep", Hint: "h", Question: "How long do you leave the trash?",
				Options: []string{"early"}, Value: "early", Weight: 1}}},
		"matches": {"Title": "Matches", "Authed": true, "Empty": false, "CSRF": "t", "Searching": false,
			"AreaFilter": "", "Nudge": "Add a short bio.",
			"Matches": []matchView{{ID: 2, FirstName: "Bo", AreaText: "Yaba",
				Gender: "Man", Occupation: "Nurse", Bio: "b", Total: 80, TopAlign: "top",
				Saved: true, Mutual: true}}},
		"match_detail": {"Title": "Bo", "Authed": true, "BD": bd, "CSRF": "t",
			"Bullets": []string{"b"}, "Enhanced": true, "DiscussQ": []string{"q"},
			"Top":     []string{"x"},
			"Bridges": []ai.Bridge{{Dimension: "sleep", Suggestion: "s"}},
			"M": detailView{ID: 2, FirstName: "Bo", AreaText: "Yaba", Gender: "Man", Occupation: "Nurse",
				Bio: "b", ModeLabel: "Has a room", BudgetStr: "1 – 2 NGN",
				MoveWindow: "x → y (6 mo)", Contact: "c",
				Mutual: true, Saved: true, MyInterest: true}},
		"onboarding": {"Title": "o", "Authed": true, "Text": "t", "CSRF": "t",
			"Dims": []qdim{{Dim: "sleep", Label: "Sleep", Hint: "h"}}},
	}
	for name, data := range cases {
		tmpl, ok := pages[name]
		if !ok {
			t.Errorf("missing page %q", name)
			continue
		}
		if err := tmpl.ExecuteTemplate(io.Discard, "base", data); err != nil {
			t.Errorf("render %s: %v", name, err)
		}
	}
	iv := map[string]any{"Title": "o", "Authed": true, "CSRF": "t", "Interview": true,
		"Turns": []Turn{{Role: "user", Text: "hi"}, {Role: "assistant", Text: "q?"}},
		"Done":  1, "Total": 12}
	if err := pages["onboarding"].ExecuteTemplate(io.Discard, "base", iv); err != nil {
		t.Errorf("render onboarding interview: %v", err)
	}
}
