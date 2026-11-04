package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SearchFilters is a structured flatmate search parsed from prose.
// Zero values mean "no constraint", except WFHMax where -1 does.
type SearchFilters struct {
	Area    string `json:"area"`
	MaxRent int64  `json:"max_rent"`
	Smoking string `json:"smoking"` // "no" when the user rules smoking out
	Pets    string `json:"pets"`    // "none" when the user rules pets out
	WFHMax  int64  `json:"wfh_max"` // -1 means no limit
}

type searchOut struct {
	Area      string `json:"area"`
	MaxRent   int64  `json:"max_rent"`
	NoSmoking bool   `json:"no_smoking"`
	NoPets    bool   `json:"no_pets"`
	WFHMax    int64  `json:"wfh_max"`
}

const searchSchema = `{"type":"object","properties":{"area":{"type":"string"},"max_rent":{"type":"integer"},"no_smoking":{"type":"boolean"},"no_pets":{"type":"boolean"},"wfh_max":{"type":"integer"}},"required":["area","max_rent","no_smoking","no_pets","wfh_max"]}`

func searchPrompt(q string) string {
	return "Parse this flatmate search into structured filters. Reply with JSON only: " + searchSchema + "\n" +
		"Rules: area is a neighborhood or city substring, empty when none is mentioned. max_rent is a monthly rent ceiling in whole units, 0 when none. no_smoking/no_pets are true only when the user rules them out. wfh_max caps days working from home 0-5, -1 when unmentioned. Never invent values.\nSearch:\n" + q
}

func validateSearch(o searchOut) error {
	if len(o.Area) > 120 {
		return fmt.Errorf("ai: area too long")
	}
	if o.MaxRent < 0 || o.MaxRent > 1000000000000 {
		return fmt.Errorf("ai: bad max_rent")
	}
	if o.WFHMax < -1 || o.WFHMax > 5 {
		return fmt.Errorf("ai: bad wfh_max")
	}
	return nil
}

// ParseSearch (T9) parses a natural-language match search.
// ok=false means the caller treats the query as a plain area filter.
func ParseSearch(ctx context.Context, llm LLM, q string) (SearchFilters, bool) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	out, err := CompleteJSON(ctx, llm, searchPrompt(q), searchSchema, validateSearch)
	if err != nil {
		return SearchFilters{}, false
	}
	f := SearchFilters{Area: strings.TrimSpace(out.Area), MaxRent: out.MaxRent, WFHMax: out.WFHMax}
	if out.NoSmoking {
		f.Smoking = "no"
	}
	if out.NoPets {
		f.Pets = "none"
	}
	return f, true
}

// KeywordSearch is the deterministic fallback: the whole query becomes
// an area substring filter.
func KeywordSearch(q string) SearchFilters {
	return SearchFilters{Area: strings.TrimSpace(q), WFHMax: -1}
}
