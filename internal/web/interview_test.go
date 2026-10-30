package web

import (
	"testing"
	"time"

	"github.com/gentcod/flatmate/internal/profile"
)

func TestInterviewsRoundTrip(t *testing.T) {
	s := NewInterviews(time.Hour)
	if _, ok := s.Get(1); ok {
		t.Fatal("fresh user has interview")
	}
	s.Update(1, func(iv *Interview) {
		iv.Turns = append(iv.Turns, Turn{Role: "user", Text: "hi"})
		iv.Drafts["sleep"] = profile.Answer{Dimension: "sleep", Value: "late"}
		iv.UserTurns = 1
	})
	got, ok := s.Get(1)
	if !ok || len(got.Turns) != 1 || got.Drafts["sleep"].Value != "late" || got.UserTurns != 1 {
		t.Fatalf("round trip: %+v %v", got, ok)
	}
	got.Drafts["sleep"] = profile.Answer{}
	again, _ := s.Get(1)
	if again.Drafts["sleep"].Value != "late" {
		t.Error("store shares drafts map with reader")
	}
	s.Drop(1)
	if _, ok := s.Get(1); ok {
		t.Error("dropped interview still present")
	}
}

func TestInterviewsExpire(t *testing.T) {
	s := NewInterviews(20 * time.Millisecond)
	s.Update(1, func(iv *Interview) { iv.UserTurns = 1 })
	time.Sleep(30 * time.Millisecond)
	if _, ok := s.Get(1); ok {
		t.Error("expired interview kept")
	}
}

func TestFormatTranscript(t *testing.T) {
	got := formatTranscript([]Turn{{Role: "user", Text: "hi"}, {Role: "assistant", Text: "q?"}})
	if got != "user: hi\nassistant: q?\n" {
		t.Errorf("transcript = %q", got)
	}
}
