package web

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Limiter is a sliding-window rate limiter keyed by client string.
// Construct with NewLimiter. Safe for concurrent use.
type Limiter struct {
	mu     sync.Mutex
	n      int
	window time.Duration
	hits   map[string][]time.Time
}

// NewLimiter allows n hits per window for each key.
func NewLimiter(n int, window time.Duration) *Limiter {
	return &Limiter{n: n, window: window, hits: map[string][]time.Time{}}
}

// Allow reports whether key may proceed, recording this hit.
func (l *Limiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.hits) > 10000 {
		l.hits = map[string][]time.Time{} // shed load rather than grow forever
	}
	cutoff := now.Add(-l.window)
	kept := make([]time.Time, 0, len(l.hits[key])+1)
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.n {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
}

// clientIP keys rate limits by origin. Behind a proxy the leftmost
// X-Forwarded-For entry wins; otherwise the peer address.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if ip, _, _ := strings.Cut(fwd, ","); strings.TrimSpace(ip) != "" {
			return strings.TrimSpace(ip)
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
