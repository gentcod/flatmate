package web

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestQuestionPromptsRateLimited(t *testing.T) {
	s := &Server{AILimiter: NewLimiter(1, time.Minute)} // AI nil on purpose
	r := httptest.NewRequest("GET", "/", nil)
	s.AILimiter.Allow("user:0")
	if got := s.questionPrompts(r, 0); len(got) != 0 {
		t.Errorf("limited prompts = %v", got)
	}
}
