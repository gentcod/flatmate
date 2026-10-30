package util

import "testing"

func TestLoopbackAddr(t *testing.T) {
	cases := map[string]bool{
		":8080": true, "localhost:8080": true, "127.0.0.1:8080": true,
		"[::1]:8080": true, "localhost": true,
		"0.0.0.0:8080": false, "example.com:443": false, "192.168.1.2:8080": false,
	}
	for addr, want := range cases {
		if got := LoopbackAddr(addr); got != want {
			t.Errorf("LoopbackAddr(%q) = %v want %v", addr, got, want)
		}
	}
}

func TestSecureCookies(t *testing.T) {
	if (Config{Addr: ":8080"}).SecureCookies() {
		t.Error("loopback default: want insecure")
	}
	if !(Config{Addr: "0.0.0.0:8080"}).SecureCookies() {
		t.Error("public default: want secure")
	}
	if !(Config{Addr: ":8080", CookieSecure: "1"}).SecureCookies() {
		t.Error("explicit 1: want secure")
	}
	if (Config{Addr: "0.0.0.0:8080", CookieSecure: "0"}).SecureCookies() {
		t.Error("explicit 0: want insecure")
	}
}
