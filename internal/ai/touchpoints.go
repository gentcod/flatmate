package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gentcod/flatmate/internal/match"
)

const extractSchema = `{"type":"object","properties":{"answers":{"type":"array","items":{"type":"object","properties":{"dimension":{"type":"string"},"value":{"type":"string"},"hard":{"type":"boolean"},"weight":{"type":"integer","minimum":0,"maximum":3}},"required":["dimension","value"]}}},"required":["answers"]}`

const explainSchema = `{"type":"object","properties":{"bullets":{"type":"array","items":{"type":"string"}},"discuss":{"type":"array","items":{"type":"string"}}},"required":["bullets","discuss"]}`

const questionsSchema = `{"type":"object","properties":{"questions":{"type":"array","items":{"type":"string"}}},"required":["questions"]}`

const normalizeSchema = `{"type":"object","properties":{"area_text":{"type":"string"},"note":{"type":"string"}},"required":["area_text","note"]}`

const interviewSchema = `{"type":"object","properties":{"done":{"type":"boolean"},"question":{"type":"string"},"answers":{"type":"array","items":{"type":"object","properties":{"dimension":{"type":"string"},"value":{"type":"string"},"hard":{"type":"boolean"},"weight":{"type":"integer","minimum":0,"maximum":3}},"required":["dimension","value"]}}},"required":["done","question","answers"]}`

const bridgesSchema = `{"type":"object","properties":{"bridges":{"type":"array","items":{"type":"object","properties":{"dimension":{"type":"string"},"suggestion":{"type":"string"}},"required":["dimension","suggestion"]}}},"required":["bridges"]}`

const questionGenSchema = `{"type":"object","properties":{"questions":{"type":"array","items":{"type":"object","properties":{"dimension":{"type":"string"},"variants":{"type":"array","items":{"type":"string"}}},"required":["dimension","variants"]}}},"required":["questions"]}`

// Draft is one extracted questionnaire answer.
type Draft struct {
	Dimension string `json:"dimension"`
	Value     string `json:"value"`
	Hard      bool   `json:"hard"`
	Weight    int    `json:"weight"`
}

type extractOut struct {
	Answers []Draft `json:"answers"`
}

type explainOut struct {
	Bullets []string `json:"bullets"`
	Discuss []string `json:"discuss"`
}

type questionsOut struct {
	Questions []string `json:"questions"`
}

type normalizeOut struct {
	AreaText string `json:"area_text"`
	Note     string `json:"note"`
}

type interviewOut struct {
	Done     bool    `json:"done"`
	Question string  `json:"question"`
	Answers  []Draft `json:"answers"`
}

// Bridge is one AI-suggested way to handle a flagged difference.
type Bridge struct {
	Dimension  string `json:"dimension"`
	Suggestion string `json:"suggestion"`
}

type bridgesOut struct {
	Bridges []Bridge `json:"bridges"`
}

type questionVariant struct {
	Dimension string   `json:"dimension"`
	Variants  []string `json:"variants"`
}

type questionGenOut struct {
	Questions []questionVariant `json:"questions"`
}

// ExtractAnswers (T1) drafts questionnaire answers from free text.
// The caller shows the plain form on error.
func ExtractAnswers(ctx context.Context, llm LLM, transcript string) ([]Draft, error) {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	out, err := CompleteJSON(ctx, llm, extractPrompt(transcript), extractSchema, validateExtract)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	for i := range out.Answers {
		if out.Answers[i].Weight == 0 {
			out.Answers[i].Weight = 1
		}
	}
	return out.Answers, nil
}

// ExplainMatch (T2) narrates an already-computed breakdown.
// enhanced=false means the caller shows the raw breakdown instead.
func ExplainMatch(ctx context.Context, llm LLM, firstName string, bd match.Breakdown) (bullets, discuss []string, enhanced bool) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	out, err := CompleteJSON(ctx, llm, explainPrompt(firstName, bd), explainSchema, validateExplain)
	if err != nil {
		return nil, nil, false
	}
	return out.Bullets, out.Discuss, true
}

// AskDiscuss (T3) drafts one neutral question per flagged difference.
func AskDiscuss(ctx context.Context, llm LLM, diffs []string) ([]string, bool) {
	if len(diffs) == 0 {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	out, err := CompleteJSON(ctx, llm, questionsPrompt(diffs), questionsSchema, validateQuestions)
	if err != nil {
		return nil, false
	}
	return out.Questions, true
}

// NormalizeArea (T4) cleans messy area text on profile save.
// Failures keep the user's wording.
func NormalizeArea(ctx context.Context, llm LLM, raw string) (normalizeOut, error) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	return CompleteJSON(ctx, llm, normalizePrompt(raw), normalizeSchema, validateNormalize)
}

// InterviewStep (T5) runs one onboarding interview turn: extract every
// answer the transcript supports and ask one follow-up for what is still
// missing. done reports the model believes all dimensions are covered.
func InterviewStep(ctx context.Context, llm LLM, transcript string, covered []string) (question string, drafts []Draft, done bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	out, err := CompleteJSON(ctx, llm, interviewPrompt(transcript, covered), interviewSchema, validateInterview)
	if err != nil {
		return "", nil, false, err
	}
	for i := range out.Answers {
		if out.Answers[i].Weight == 0 {
			out.Answers[i].Weight = 1
		}
	}
	return out.Question, out.Answers, out.Done, nil
}

// SuggestBridges (T6) drafts one practical compromise per flagged
// difference. ok=false means the caller shows the plain discuss list.
func SuggestBridges(ctx context.Context, llm LLM, diffs []string) ([]Bridge, bool) {
	if len(diffs) == 0 {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	out, err := CompleteJSON(ctx, llm, bridgesPrompt(diffs), bridgesSchema, validateBridges)
	if err != nil {
		return nil, false
	}
	return out.Bridges, true
}

// GenerateQuestions (T7) drafts natural scenario questions for the given
// dimensions, picking one random variant each so repeat visits vary.
// The caller falls back to plain labels on error. Dimensions, options,
// and stored values are untouched — only the prompt text is AI-made.
func GenerateQuestions(ctx context.Context, llm LLM, dims []string) (map[string]string, error) {
	if len(dims) == 0 {
		return map[string]string{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	out, err := CompleteJSON(ctx, llm, questionGenPrompt(dims), questionGenSchema, validateQuestionGen)
	if err != nil {
		return nil, err
	}
	picked := map[string]string{}
	for _, q := range out.Questions {
		if len(q.Variants) == 0 {
			continue
		}
		picked[q.Dimension] = q.Variants[rand.Intn(len(q.Variants))]
	}
	return picked, nil
}

// dimensionLines lists every questionnaire axis and its legal values
// for prompts that ask the model to produce answers.
func dimensionLines() string {
	var b strings.Builder
	for _, d := range match.Dimensions {
		if d.Type == match.Choice {
			fmt.Fprintf(&b, "- %s: %s\n", d.Name, strings.Join(d.Choices, "/"))
		} else {
			fmt.Fprintf(&b, "- %s: integer %d-%d\n", d.Name, d.Min, d.Max)
		}
	}
	return b.String()
}

func extractPrompt(transcript string) string {
	var b strings.Builder
	b.WriteString("Extract flatmate questionnaire answers from the description below.\n")
	b.WriteString("Reply with JSON only, matching this schema: " + extractSchema + "\n")
	b.WriteString("Rules: use only these dimensions and values:\n")
	b.WriteString(dimensionLines())
	b.WriteString("hard=true only for explicit deal-breaker language. weight 1-3 by emphasis, default 1. Skip anything not stated. Never infer beyond the text.\n")
	b.WriteString("Description:\n")
	b.WriteString(transcript)
	return b.String()
}

func explainPrompt(firstName string, bd match.Breakdown) string {
	raw, _ := json.Marshal(bd)
	return "Explain this precomputed flatmate match in plain language for " + firstName + ".\n" +
		"Reply with JSON only: " + explainSchema + "\n" +
		"Rules: restate the given scores and notes only. Never invent scores, rankings, or new facts. " +
		"Bullets cover what lines up and what differs; discuss turns flagged gaps into neutral topics.\n" +
		"Breakdown:\n" + string(raw)
}

func questionsPrompt(diffs []string) string {
	return "Write one neutral discussion question per flagged difference. Reply with JSON only: " + questionsSchema + "\n" +
		"Rules: neutral tone, no judgment, address both people.\nDifferences:\n" + strings.Join(diffs, "\n")
}

func normalizePrompt(raw string) string {
	return "Normalize this messy area/listing text. Reply with JSON only: " + normalizeSchema + "\n" +
		"Rules: keep area_text short (neighborhood, city); preserve the original wording in note; never invent an exact address.\nText:\n" + raw
}

func interviewPrompt(transcript string, covered []string) string {
	var b strings.Builder
	b.WriteString("You are interviewing a user to fill a flatmate questionnaire. Reply with JSON only, matching this schema: " + interviewSchema + "\n")
	b.WriteString("Rules: use only these dimensions and values:\n")
	b.WriteString(dimensionLines())
	b.WriteString("Already covered: " + strings.Join(covered, ", ") + " (blank means none yet).\n")
	b.WriteString("answers: EVERY answer extractable from the whole conversation so far (cumulative, not just this turn). hard=true only for explicit deal-breaker language. weight 1-3 by emphasis, default 1. Never infer beyond the text.\n")
	b.WriteString("If every dimension is covered, or the user asks to stop, set done=true and question=\"\". Otherwise done=false and ask exactly ONE short follow-up question (one sentence) targeting an uncovered dimension.\n")
	b.WriteString("Conversation:\n" + transcript)
	return b.String()
}

func bridgesPrompt(diffs []string) string {
	return "Suggest one practical compromise for each flagged flatmate difference. Reply with JSON only: " + bridgesSchema + "\n" +
		"Rules: each suggestion names the difference it addresses, stays neutral and realistic for shared living, and never invents facts about either person.\nDifferences:\n" + strings.Join(diffs, "\n")
}

// scaleAnchors orients generated questions on 1-5 scales. It mirrors the
// questionnaire hints; choice dimensions use their option lists instead.
var scaleAnchors = map[string]string{
	"cleanliness": "1=relaxed about mess, 5=spotless",
	"noise":       "1=needs quiet, 5=fine with noise",
	"guests":      "1=rarely hosts, 5=hosts often",
	"privacy":     "1=happy sharing space, 5=needs lots of own space",
}

func questionGenPrompt(dims []string) string {
	var b strings.Builder
	b.WriteString("Write natural one-sentence scenario questions for a flatmate questionnaire. Reply with JSON only: " + questionGenSchema + "\n")
	b.WriteString("Rules: one entry per dimension below, 2 different everyday scenarios each. Each scenario must be answerable with exactly the listed options. Ask naturally: never name the dimension, the scale, or the options. Keep each under 200 characters.\n")
	for _, name := range dims {
		d, ok := match.ByName(name)
		if !ok {
			continue
		}
		if d.Type == match.Choice {
			fmt.Fprintf(&b, "- %s: %s\n", d.Name, strings.Join(d.Choices, "/"))
		} else if anchor, ok := scaleAnchors[d.Name]; ok {
			fmt.Fprintf(&b, "- %s: rate %s\n", d.Name, anchor)
		} else {
			fmt.Fprintf(&b, "- %s: rate 1-%d\n", d.Name, d.Max)
		}
	}
	return b.String()
}

func validateDraft(a Draft, seen map[string]bool) error {
	if seen[a.Dimension] {
		return fmt.Errorf("ai: duplicate %q", a.Dimension)
	}
	seen[a.Dimension] = true
	if _, ok := match.ByName(a.Dimension); !ok {
		return fmt.Errorf("ai: unknown dimension %q", a.Dimension)
	}
	if !match.ValidValue(a.Dimension, a.Value) {
		return fmt.Errorf("ai: bad value %q for %q", a.Value, a.Dimension)
	}
	if a.Weight < 0 || a.Weight > 3 {
		return fmt.Errorf("ai: bad weight for %q", a.Dimension)
	}
	return nil
}

func validateExtract(o extractOut) error {
	if len(o.Answers) == 0 || len(o.Answers) > len(match.Dimensions) {
		return fmt.Errorf("ai: want 1-%d answers", len(match.Dimensions))
	}
	seen := map[string]bool{}
	for _, a := range o.Answers {
		if err := validateDraft(a, seen); err != nil {
			return err
		}
	}
	return nil
}

func validateInterview(o interviewOut) error {
	if len(o.Question) > 280 {
		return fmt.Errorf("ai: question too long")
	}
	if !o.Done && strings.TrimSpace(o.Question) == "" {
		return fmt.Errorf("ai: want a follow-up question")
	}
	if len(o.Answers) > len(match.Dimensions) {
		return fmt.Errorf("ai: too many answers")
	}
	seen := map[string]bool{}
	for _, a := range o.Answers {
		if err := validateDraft(a, seen); err != nil {
			return err
		}
	}
	return nil
}

func validateQuestionGen(o questionGenOut) error {
	if len(o.Questions) == 0 || len(o.Questions) > len(match.Dimensions) {
		return fmt.Errorf("ai: want 1-%d question sets", len(match.Dimensions))
	}
	seen := map[string]bool{}
	for _, q := range o.Questions {
		if seen[q.Dimension] {
			return fmt.Errorf("ai: duplicate %q", q.Dimension)
		}
		seen[q.Dimension] = true
		if _, ok := match.ByName(q.Dimension); !ok {
			return fmt.Errorf("ai: unknown dimension %q", q.Dimension)
		}
		if len(q.Variants) == 0 || len(q.Variants) > 3 {
			return fmt.Errorf("ai: want 1-3 variants for %q", q.Dimension)
		}
		for _, v := range q.Variants {
			if strings.TrimSpace(v) == "" || len(v) > 280 {
				return fmt.Errorf("ai: bad variant for %q", q.Dimension)
			}
		}
	}
	return nil
}

func validateBridges(o bridgesOut) error {
	if len(o.Bridges) == 0 || len(o.Bridges) > 6 {
		return fmt.Errorf("ai: want 1-6 bridges")
	}
	for _, b := range o.Bridges {
		if _, ok := match.ByName(b.Dimension); !ok {
			return fmt.Errorf("ai: unknown dimension %q", b.Dimension)
		}
		if strings.TrimSpace(b.Suggestion) == "" || len(b.Suggestion) > 280 {
			return fmt.Errorf("ai: bad suggestion for %q", b.Dimension)
		}
	}
	return nil
}

func validateExplain(o explainOut) error {
	if len(o.Bullets) == 0 || len(o.Bullets) > 8 {
		return fmt.Errorf("ai: want 1-8 bullets")
	}
	if len(o.Discuss) > 6 {
		return fmt.Errorf("ai: too many discuss items")
	}
	for _, s := range append(append([]string{}, o.Bullets...), o.Discuss...) {
		if strings.TrimSpace(s) == "" || len(s) > 280 {
			return fmt.Errorf("ai: bad explanation line")
		}
	}
	return nil
}

func validateQuestions(o questionsOut) error {
	if len(o.Questions) == 0 || len(o.Questions) > 6 {
		return fmt.Errorf("ai: want 1-6 questions")
	}
	for _, s := range o.Questions {
		if strings.TrimSpace(s) == "" || len(s) > 280 {
			return fmt.Errorf("ai: bad question")
		}
	}
	return nil
}

func validateNormalize(o normalizeOut) error {
	if len(o.AreaText) == 0 || len(o.AreaText) > 120 {
		return fmt.Errorf("ai: bad area_text")
	}
	if len(o.Note) > 280 {
		return fmt.Errorf("ai: note too long")
	}
	return nil
}
