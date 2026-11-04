package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// AgreementSection is one headed block of a drafted roommate agreement.
type AgreementSection struct {
	Heading string   `json:"heading"`
	Points  []string `json:"points"`
}

type agreementOut struct {
	Sections []AgreementSection `json:"sections"`
}

const agreementSchema = `{"type":"object","properties":{"sections":{"type":"array","minItems":4,"maxItems":4,"items":{"type":"object","properties":{"heading":{"type":"string"},"points":{"type":"array","items":{"type":"string"}}},"required":["heading","points"]}}},"required":["sections"]}`

func agreementPrompt(a, b string, flagged []string) string {
	topics := strings.Join(flagged, ", ")
	if topics == "" {
		topics = "general living habits"
	}
	return "Draft an informal roommate agreement for " + a + " and " + b + ". Reply with JSON only: " + agreementSchema + "\n" +
		"Rules: exactly 4 sections with these headings: House rules, Bills, Guests, Moving out. " +
		"2-4 short practical points each, plain language, no legal jargon. Address these flagged topics where they fit: " + topics + ". " +
		"Never invent facts about either person. This is a conversation starter, not legal advice."
}

func validateAgreement(o agreementOut) error {
	if len(o.Sections) != 4 {
		return fmt.Errorf("ai: want 4 sections, got %d", len(o.Sections))
	}
	want := map[string]bool{"House rules": true, "Bills": true, "Guests": true, "Moving out": true}
	for _, s := range o.Sections {
		if !want[s.Heading] {
			return fmt.Errorf("ai: bad heading %q", s.Heading)
		}
		delete(want, s.Heading)
		if len(s.Points) < 2 || len(s.Points) > 5 {
			return fmt.Errorf("ai: want 2-5 points in %q", s.Heading)
		}
		for _, p := range s.Points {
			if strings.TrimSpace(p) == "" || len(p) > 300 {
				return fmt.Errorf("ai: bad point in %q", s.Heading)
			}
		}
	}
	return nil
}

// DraftAgreement (T8) drafts a roommate agreement for a and b.
// ok=false means the caller shows the template version.
func DraftAgreement(ctx context.Context, llm LLM, a, b string, flagged []string) ([]AgreementSection, bool) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	out, err := CompleteJSON(ctx, llm, agreementPrompt(a, b, flagged), agreementSchema, validateAgreement)
	if err != nil {
		return nil, false
	}
	return out.Sections, true
}

// TemplateAgreement is the deterministic fallback: the same four
// sections with generic starter points.
func TemplateAgreement() []AgreementSection {
	return []AgreementSection{
		{Heading: "House rules", Points: []string{
			"Agree on quiet hours for weeknights and weekends.",
			"Split shared chores on a weekly rotation and write it down.",
			"Keep common areas tidy daily; deep-clean together monthly.",
		}},
		{Heading: "Bills", Points: []string{
			"List every shared bill, its due date, and who pays first.",
			"Agree on the split (equal or custom) before moving day.",
			"Settle balances at least once a month.",
		}},
		{Heading: "Guests", Points: []string{
			"Set how much notice overnight guests need.",
			"Cap overnight stays per week unless you both agree otherwise.",
			"No long-term guests without a written heads-up.",
		}},
		{Heading: "Moving out", Points: []string{
			"Give at least 30 days notice before moving out.",
			"The person leaving helps find or approve a replacement.",
			"Return shared deposits and settle the ledger on exit.",
		}},
	}
}
