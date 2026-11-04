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
			"Top":       []string{"x"},
			"Bridges":   []ai.Bridge{{Dimension: "sleep", Suggestion: "s"}},
			"Reviews":   []reviewView{{AuthorName: "Bo", Rating: 5, Body: "great"}},
			"ReviewAvg": "5.0", "ReviewCount": 1,
			"M": detailView{ID: 2, FirstName: "Bo", AreaText: "Yaba", Gender: "Man", Occupation: "Nurse",
				Bio: "b", ModeLabel: "Has a room", BudgetStr: "1 – 2 NGN",
				MoveWindow: "x → y (6 mo)", Contact: "c",
				Mutual: true, Saved: true, MyInterest: true}},
		"onboarding": {"Title": "o", "Authed": true, "Text": "t", "CSRF": "t",
			"Dims": []qdim{{Dim: "sleep", Label: "Sleep", Hint: "h"}}},
		"listings": {"Title": "Room listings", "Authed": true, "CSRF": "t", "AreaFilter": "",
			"Listings": []listingView{{ID: 1, UserID: 2, OwnerName: "Bo", Title: "Room",
				Description: "d", AreaText: "Yaba", MonthlyRent: 100, Currency: "NGN", PhotoURL: "/placeholders/room-1.svg"}}},
		"listing": {"Title": "Room", "Authed": true, "CSRF": "t", "Mine": false,
			"Mutual": true, "MyInterest": true, "Contact": "c", "ContactSelf": true,
			"L": listingView{ID: 1, UserID: 1, OwnerName: "Ada", Title: "Room",
				Description: "d", AreaText: "Yaba", MonthlyRent: 100, Currency: "NGN", Mine: true, PhotoURL: "/placeholders/room-2.svg"}},
		"chat": {"Title": "Chats", "Authed": true, "CSRF": "t",
			"Conversations": []convoView{{PeerID: 2, PeerName: "Bo", LastBody: "hi"}}},
		"thread": {"Title": "Bo", "Authed": true, "CSRF": "t", "PeerID": 2, "PeerName": "Bo",
			"Messages": []msgView{{Mine: false, Body: "hi"}, {Mine: true, Body: "hello"}}},
		"expenses": {"Title": "Shared expenses", "Authed": true, "CSRF": "t", "PeerID": 2, "PeerName": "Bo",
			"Expenses": []expenseView{{ID: 1, PaidByMe: true, Amount: 100, Currency: "NGN", Split: "equal", OwedToMe: 50}},
			"Balances": []balanceView{{Currency: "NGN", Amount: 50}}},
		"agreement": {"Title": "Roommate agreement", "Authed": true, "CSRF": "t", "PeerID": 2, "PeerName": "Bo", "Enhanced": false,
			"Sections": []ai.AgreementSection{{Heading: "Bills", Points: []string{"a", "b"}}}},
		"admin_reports": {"Title": "Reports", "Authed": true, "IsAdmin": true, "CSRF": "t",
			"Reports": []apiReport{{ID: 1, Reporter: 2, ReporterName: "Bo", Target: 3, TargetName: "Cy", Reason: "spam", CreatedAt: "2026-01-01"}}},
		"admin_metrics": {"Title": "Metrics", "Authed": true, "IsAdmin": true,
			"users": 3, "active_users": 2, "open_reports": 1, "listings": 4, "reviews": 5, "expenses": 6},
		"landing": {"Title": "Find flatmates who match your rhythm", "CSRF": "t"},
		"alerts": {"Title": "Alerts", "Authed": true, "CSRF": "t", "UnreadAlerts": int64(1),
			"Alerts": []alertView{
				{Kind: "interest", ActorID: 2, ActorName: "Bo", RefID: 2, Body: "Bo showed interest in you.", Unread: true, At: "2026-01-01"},
				{Kind: "listing_review", ActorID: 2, ActorName: "Bo", RefID: 7, Body: "Bo left a stay review on Sunny room.", At: "2026-01-02"},
			}},
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
}
