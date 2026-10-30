package web

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestLimiterWindow(t *testing.T) {
	l := NewLimiter(2, 60*time.Millisecond)
	if !l.Allow("a") || !l.Allow("a") {
		t.Fatal("first two allowed")
	}
	if l.Allow("a") {
		t.Error("third within window denied")
	}
	if !l.Allow("b") {
		t.Error("other key allowed")
	}
	time.Sleep(70 * time.Millisecond)
	if !l.Allow("a") {
		t.Error("after window allowed")
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	if got := clientIP(r); got != "192.0.2.1" {
		t.Errorf("peer = %q", got)
	}
	r.Header.Set("X-Forwarded-For", "203.0.113.7, 70.41.3.18")
	if got := clientIP(r); got != "203.0.113.7" {
		t.Errorf("forwarded = %q", got)
	}
}
