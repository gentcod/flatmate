package web

import (
	"strings"
	"sync"
	"time"

	"github.com/gentcod/flatmate/internal/profile"
)

// Turn is one interview message. Role is "user" or "assistant".
type Turn struct {
	Role string
	Text string
}

// Interview is one user's in-progress onboarding conversation.
// It lives only in memory: a restart simply restarts the interview,
// and every turn is also persisted to onboarding_drafts for reference.
type Interview struct {
	Turns     []Turn
	Drafts    map[string]profile.Answer
	UserTurns int
	Updated   time.Time
}

// Interviews tracks active interviews by user ID. Safe for concurrent use.
type Interviews struct {
	mu  sync.Mutex
	ttl time.Duration
	all map[int64]*Interview
}

// NewInterviews tracks interviews, forgetting ones idle past ttl.
func NewInterviews(ttl time.Duration) *Interviews {
	return &Interviews{ttl: ttl, all: map[int64]*Interview{}}
}

// Update runs fn against the user's interview under lock, creating an
// empty one on first use. The idle clock restarts on every update.
func (s *Interviews) Update(id int64, fn func(iv *Interview)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(time.Now())
	iv, ok := s.all[id]
	if !ok {
		iv = &Interview{Drafts: map[string]profile.Answer{}}
		s.all[id] = iv
	}
	fn(iv)
	iv.Updated = time.Now()
}

// Get returns a copy of the user's interview, if one is active.
func (s *Interviews) Get(id int64) (*Interview, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(time.Now())
	iv, ok := s.all[id]
	if !ok {
		return nil, false
	}
	cp := &Interview{Turns: append([]Turn(nil), iv.Turns...),
		Drafts: map[string]profile.Answer{}, UserTurns: iv.UserTurns, Updated: iv.Updated}
	for k, v := range iv.Drafts {
		cp.Drafts[k] = v
	}
	return cp, true
}

// Drop forgets a user's interview, if any.
func (s *Interviews) Drop(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.all, id)
}

func (s *Interviews) pruneLocked(now time.Time) {
	for id, iv := range s.all {
		if now.Sub(iv.Updated) > s.ttl {
			delete(s.all, id)
		}
	}
}

// formatTranscript flattens turns for the model prompt and the saved record.
func formatTranscript(turns []Turn) string {
	var b strings.Builder
	for _, t := range turns {
		b.WriteString(t.Role)
		b.WriteString(": ")
		b.WriteString(t.Text)
		b.WriteString("\n")
	}
	return b.String()
}
