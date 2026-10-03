package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"jejak/internal/auth"
	"jejak/internal/db"
	"jejak/internal/ratelimit"
)

// TestHandleLoginRateLimit proves the brute-force throttle: 5 failed attempts
// each 401, the 6th attempt 429 with Retry-After.
func TestHandleLoginRateLimit(t *testing.T) {
	s := &fakeStore{creatorErr: sql.ErrNoRows}
	h := newTestHandler(s)
	h.Auth = auth.NewMemoryStore()
	h.LoginLimiter = ratelimit.NewLimiter()

	const body = `{"username":"ghost","password":"wrong"}`
	for i := 1; i <= ratelimit.MaxAttempts; i++ {
		req := postJSON("/api/login", body)
		req.RemoteAddr = "203.0.113.5:12345"
		rr := httptest.NewRecorder()
		h.HandleLogin(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i, rr.Code)
		}
	}

	// Attempt #6 (MaxAttempts+1) must be throttled.
	req := postJSON("/api/login", body)
	req.RemoteAddr = "203.0.113.5:12345"
	rr := httptest.NewRecorder()
	h.HandleLogin(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("attempt %d: status = %d, want 429", ratelimit.MaxAttempts+1, rr.Code)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Error("429 response missing Retry-After header")
	}
}

// TestHandleLoginSuccessResetsLimiter: a legit login clears the failed-attempt
// bucket, so the owner is not locked out by their own typos.
func TestHandleLoginSuccessResetsLimiter(t *testing.T) {
	hash, _ := auth.HashPassword("correct-pass")
	s := &fakeStore{creator: db.Creator{ID: 9, Username: "alice", PasswordHash: hash}}
	h := newTestHandler(s)
	h.Auth = auth.NewMemoryStore()
	h.LoginLimiter = ratelimit.NewLimiter()

	// 2 wrong attempts, then a correct login.
	for i := 0; i < 2; i++ {
		req := postJSON("/api/login", `{"username":"alice","password":"wrong"}`)
		req.RemoteAddr = "10.0.0.5:1000"
		rr := httptest.NewRecorder()
		h.HandleLogin(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("wrong attempt %d: status %d, want 401", i, rr.Code)
		}
	}

	ok := postJSON("/api/login", `{"username":"alice","password":"correct-pass"}`)
	ok.RemoteAddr = "10.0.0.5:1000"
	rr := httptest.NewRecorder()
	h.HandleLogin(rr, ok)
	if rr.Code != http.StatusOK {
		t.Fatalf("correct login status = %d, want 200", rr.Code)
	}
}
