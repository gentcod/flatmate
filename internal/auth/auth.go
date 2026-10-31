// Package auth handles password hashing, email lookup hashes, and
// PASETO session tokens carried in an HttpOnly cookie.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"hash"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/o1egl/paseto"
)

const (
	pwVersion byte = 0x01
	pwSaltLen      = 16
	pwKeyLen       = 32
	pwIter         = 210_000
)

// HashPassword derives a salted PBKDF2-SHA256 blob: version || salt || key.
func HashPassword(password string) ([]byte, error) {
	salt := make([]byte, pwSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	key := pbkdf2(password, salt, pwIter, pwKeyLen)
	out := make([]byte, 0, 1+pwSaltLen+pwKeyLen)
	out = append(out, pwVersion)
	out = append(out, salt...)
	out = append(out, key...)
	return out, nil
}

// CheckPassword reports whether password matches the stored blob.
func CheckPassword(blob []byte, password string) bool {
	if len(blob) != 1+pwSaltLen+pwKeyLen || blob[0] != pwVersion {
		return false
	}
	want := pbkdf2(password, blob[1:1+pwSaltLen], pwIter, pwKeyLen)
	return subtle.ConstantTimeCompare(want, blob[1+pwSaltLen:]) == 1
}

func pbkdf2(password string, salt []byte, iter, keyLen int) []byte {
	dk := make([]byte, 0, keyLen)
	var block uint32 = 1
	for len(dk) < keyLen {
		dk = append(dk, pbkdf2Block(sha256.New, password, salt, iter, block)...)
		block++
	}
	return dk[:keyLen]
}

func pbkdf2Block(h func() hash.Hash, password string, salt []byte, iter int, block uint32) []byte {
	mac := hmac.New(h, []byte(password))
	mac.Write(salt)
	mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
	u := mac.Sum(nil)
	out := make([]byte, len(u))
	copy(out, u)
	for i := 1; i < iter; i++ {
		mac = hmac.New(h, []byte(password))
		mac.Write(u)
		u = mac.Sum(nil)
		for j := range out {
			out[j] ^= u[j]
		}
	}
	return out
}

// NormalizeEmail lowercases and trims an address for hashing.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// HashEmail returns the SHA-256 lookup hash of a normalized email.
func HashEmail(email string) []byte {
	sum := sha256.Sum256([]byte(NormalizeEmail(email)))
	return sum[:]
}

// Sessions issues and verifies PASETO V2 local tokens.
type Sessions struct {
	v2     *paseto.V2
	key    []byte
	Secure bool // set the Secure flag on cookies; false for local HTTP dev
}

// NewSessions builds a session manager from a 64-char hex key (32 bytes).
func NewSessions(hexKey string, secure bool) (*Sessions, error) {
	key, err := hex.DecodeString(strings.TrimSpace(hexKey))
	if err != nil {
		return nil, fmt.Errorf("decode session key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("session key must be 32 bytes, got %d", len(key))
	}
	return &Sessions{v2: paseto.NewV2(), key: key, Secure: secure}, nil
}

// NewEphemeralSessions builds a manager with a random key.
// Sessions die on restart. Demo/dev use only.
func NewEphemeralSessions(secure bool) (*Sessions, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return &Sessions{v2: paseto.NewV2(), key: key, Secure: secure}, nil
}

// Issue mints a token for userID valid for ttl.
func (s *Sessions) Issue(userID int64, ttl time.Duration) (string, error) {
	now := time.Now()
	tok := paseto.JSONToken{
		Subject:    strconv.FormatInt(userID, 10),
		IssuedAt:   now,
		Expiration: now.Add(ttl),
	}
	return s.v2.Encrypt(s.key, tok, nil)
}

// Verify checks a token and returns the subject user ID.
func (s *Sessions) Verify(token string) (int64, error) {
	var tok paseto.JSONToken
	if err := s.v2.Decrypt(token, s.key, &tok, nil); err != nil {
		return 0, err
	}
	if err := tok.Validate(); err != nil {
		return 0, err
	}
	return strconv.ParseInt(tok.Subject, 10, 64)
}

const cookieName = "session"

// SetCookie writes the session cookie.
func (s *Sessions) SetCookie(w http.ResponseWriter, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.Secure,
		Expires:  time.Now().Add(ttl),
	})
}

// ClearCookie removes the session cookie. Flags mirror SetCookie so
// browsers actually drop a Secure cookie.
func (s *Sessions) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: s.Secure, MaxAge: -1})
}

// CookieToken reads the session cookie value.
func CookieToken(r *http.Request) (string, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return "", false
	}
	return c.Value, true
}

const csrfCookieName = "csrf"

// EnsureCSRFToken returns this request's CSRF token, minting and setting
// the double-submit cookie on first sight. Server-rendered forms echo the
// token in a hidden csrf_token field; CheckCSRF compares the two. No
// server-side storage, so it works for logged-out forms too.
func EnsureCSRFToken(w http.ResponseWriter, r *http.Request, secure bool) string {
	if c, err := r.Cookie(csrfCookieName); err == nil && validCSRFToken(c.Value) {
		return c.Value
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "" // fail closed: the POST check rejects empty tokens
	}
	tok := hex.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{Name: csrfCookieName, Value: tok, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure})
	return tok
}

func validCSRFToken(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

// CheckCSRF reports whether the request token matches the CSRF cookie.
// Browser forms send it as the csrf_token field; the JSON API sends it
// as the X-CSRF-Token header. Either satisfies the double-submit check.
func CheckCSRF(r *http.Request) bool {
	c, err := r.Cookie(csrfCookieName)
	if err != nil || !validCSRFToken(c.Value) {
		return false
	}
	f := r.Header.Get("X-CSRF-Token")
	if f == "" {
		f = r.FormValue("csrf_token")
	}
	if !validCSRFToken(f) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(c.Value), []byte(f)) == 1
}
