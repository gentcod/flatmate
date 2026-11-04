package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestValidateExtract(t *testing.T) {
	good := extractOut{Answers: []Draft{{Dimension: "sleep", Value: "late", Weight: 2}}}
	if err := validateExtract(good); err != nil {
		t.Errorf("valid: %v", err)
	}
	if err := validateExtract(extractOut{}); err == nil {
		t.Error("empty: want error")
	}
}

func TestValidateBridges(t *testing.T) {
	good := bridgesOut{Bridges: []Bridge{{Dimension: "sleep", Suggestion: "Agree on quiet hours after 11pm."}}}
	if err := validateBridges(good); err != nil {
		t.Errorf("valid: %v", err)
	}
	for i, o := range []bridgesOut{
		{},
		{Bridges: []Bridge{{Dimension: "sleep", Suggestion: ""}}},
		{Bridges: []Bridge{{Dimension: "vibes", Suggestion: "Be chill."}}},
	} {
		if err := validateBridges(o); err == nil {
			t.Errorf("case %d: want error", i)
		}
	}
}

// stubLLM replays scripted replies for touchpoint tests.
type stubLLM struct {
	replies []string
	calls   int
}

var errStubExhausted = errors.New("stub replies exhausted")

func (s *stubLLM) CompleteRaw(ctx context.Context, prompt, schema string) (json.RawMessage, error) {
	s.calls++
	if s.calls > len(s.replies) {
		return nil, errStubExhausted
	}
	return json.RawMessage(s.replies[s.calls-1]), nil
}

func TestValidateQuestionSets(t *testing.T) {
	good := questionGenOut{Questions: []questionVariant{
		{Dimension: "cleanliness", Variants: []string{"How long do you leave the trash?", "How often do you mop?"}},
	}}
	if err := validateQuestionGen(good); err != nil {
		t.Errorf("valid: %v", err)
	}
	bad := []questionGenOut{
		{},
		{Questions: []questionVariant{{Dimension: "vibes", Variants: []string{"q?"}}}},
		{Questions: []questionVariant{
			{Dimension: "sleep", Variants: []string{"a?"}},
			{Dimension: "sleep", Variants: []string{"b?"}},
		}},
		{Questions: []questionVariant{{Dimension: "sleep"}}},
		{Questions: []questionVariant{{Dimension: "sleep", Variants: []string{""}}}},
	}
	for i, o := range bad {
		if err := validateQuestionGen(o); err == nil {
			t.Errorf("case %d: want error", i)
		}
	}
}

func TestGenerateQuestionsPicksVariant(t *testing.T) {
	stub := &stubLLM{replies: []string{
		`{"questions":[{"dimension":"cleanliness","variants":["How long do you leave the trash?","How often do you mop?"]}]}`,
	}}
	got, err := GenerateQuestions(context.Background(), stub, []string{"cleanliness"})
	if err != nil {
		t.Fatal(err)
	}
	if q := got["cleanliness"]; q != "How long do you leave the trash?" && q != "How often do you mop?" {
		t.Errorf("picked = %q", q)
	}
	if _, err := GenerateQuestions(context.Background(), stub, nil); err != nil {
		t.Errorf("empty dims: %v", err)
	}
}

func TestTailTranscript(t *testing.T) {
	short := "user: hi"
	if got := tailTranscript(short); got != short {
		t.Errorf("short: got %q, want unchanged", got)
	}
	long := strings.Repeat("x", maxTranscriptRunes+100)
	got := tailTranscript(long)
	if len([]rune(got)) > maxTranscriptRunes+50 {
		t.Errorf("long: tail %d runes, want about %d", len([]rune(got)), maxTranscriptRunes)
	}
	if !strings.HasPrefix(got, "[earlier conversation omitted]") {
		t.Error("long: want omission marker at the front")
	}
	if !strings.HasSuffix(got, strings.Repeat("x", maxTranscriptRunes)) {
		t.Error("long: want the most recent runes kept")
	}
}

func TestExtractAnswersRetries(t *testing.T) {
	stub := &stubLLM{replies: []string{
		`not json`,
		`{"answers":[{"dimension":"sleep","value":"late","weight":0}]}`,
	}}
	drafts, err := ExtractAnswers(context.Background(), stub, "user: i'm a night owl")
	if err != nil || len(drafts) != 1 {
		t.Fatalf("drafts=%+v err=%v", drafts, err)
	}
	if drafts[0].Weight != 1 {
		t.Errorf("weight default = %d", drafts[0].Weight)
	}
	if stub.calls != 2 {
		t.Errorf("calls = %d want 2", stub.calls)
	}
}
