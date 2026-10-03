package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jejak/internal/auth"
	"jejak/internal/ratelimit"
)

// TestShortenDangerousSchemes400 proves the open-redirect/XSS gadgets are
// rejected at the door: every non-http(s) scheme that a browser would still
// try to execute (file:, data:, vbscript:) must never reach the DB, because
// the stored value later comes back verbatim in the redirect Location.
func TestShortenDangerousSchemes400(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"javascript", `{"url":"javascript:alert(1)"}`},
		{"data", `{"url":"data:text/html,<script>alert(1)</script>"}`},
		{"file", `{"url":"file:///etc/passwd"}`},
		{"vbscript", `{"url":"vbscript:msgbox(1)"}`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s := &fakeStore{urls: map[string]string{}}
			h := newTestHandler(s)
			req := postJSON("/api/shorten", tt.body)
			req.Host = "example.com"
			rr := httptest.NewRecorder()
			h.HandleShorten(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
			}
			if len(s.created) != 0 {
				t.Fatalf("dangerous URL stored: %+v", s.created)
			}
		})
	}
}

// TestShortenTraversalSlug400: a slug is a short_code, never a path. Slashes
// and dot segments must fail the format check before any store lookup, so
// "../etc/passwd" can neither become a code nor probe the filesystem.
func TestShortenTraversalSlug400(t *testing.T) {
	s := &fakeStore{urls: map[string]string{}}
	h := newTestHandler(s)
	req := postJSON("/api/shorten", `{"url":"https://example.com/x","slug":"../etc/passwd"}`)
	req.Host = "example.com"
	rr := httptest.NewRecorder()
	h.HandleShorten(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("traversal slug status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	if len(s.created) != 0 {
		t.Fatalf("traversal slug stored: %+v", s.created)
	}
}

// TestDecodeJSONOversize400 documents the deliberate contract: a JSON body
// over maxJSONBodyBytes (1 MB) fails decode with 400 (decodeJSON), NOT 413:
// one error path across every JSON handler. The multipart avatar endpoint is
// the 413 case (see TestHandleUploadAvatar).
func TestDecodeJSONOversize400(t *testing.T) {
	s := &fakeStore{urls: map[string]string{}}
	h := newTestHandler(s)
	big := `{"url":"https://example.com/` + strings.Repeat("a", maxJSONBodyBytes+64) + `"}`
	req := postJSON("/api/shorten", big)
	req.Host = "example.com"
	rr := httptest.NewRecorder()
	h.HandleShorten(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("oversize JSON status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	if len(s.created) != 0 {
		t.Fatalf("oversize body stored a row: %+v", s.created)
	}
}

// TestDecodeJSONMalformed400: syntactically invalid JSON (or a non-JSON body
// on a JSON endpoint) is a client error, never a 500.
func TestDecodeJSONMalformed400(t *testing.T) {
	s := &fakeStore{urls: map[string]string{}}
	h := newTestHandler(s)
	for _, body := range []string{`{"url":`, `not json at all`, `[1,2,3`} {
		req := postJSON("/api/shorten", body)
		req.Host = "example.com"
		rr := httptest.NewRecorder()
		h.HandleShorten(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("body %q status = %d, want 400", body, rr.Code)
		}
	}
}

// TestHandleRegisterRateLimit proves the new anti-spam throttle: 10
// registrations/minute per IP, the 11th attempt 429 with Retry-After.
// Every attempt consumes the budget (no credential separates success from
// failure at this point).
func TestHandleRegisterRateLimit(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)
	h.Auth = auth.NewMemoryStore()
	h.RegisterLimiter = ratelimit.NewLimiterWithMax(10)

	for i := 1; i <= 10; i++ {
		req := postJSON("/api/register", `{"username":"user`+string(rune('a'+i))+`","display_name":"U","password":"password123"}`)
		req.RemoteAddr = "198.51.100.7:5555"
		rr := httptest.NewRecorder()
		h.HandleRegister(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("attempt %d: status = %d, want 201; body=%s", i, rr.Code, rr.Body.String())
		}
	}

	req := postJSON("/api/register", `{"username":"toolate","display_name":"U","password":"password123"}`)
	req.RemoteAddr = "198.51.100.7:5555"
	rr := httptest.NewRecorder()
	h.HandleRegister(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("attempt 11: status = %d, want 429", rr.Code)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Error("429 missing Retry-After")
	}
}

// TestHandleShortenRateLimit proves the spam-link cap: 30 shorten
// attempts/minute per IP, the 31st 429.
func TestHandleShortenRateLimit(t *testing.T) {
	s := &fakeStore{urls: map[string]string{}}
	h := newTestHandler(s)
	h.ShortenLimiter = ratelimit.NewLimiterWithMax(30)

	for i := 0; i < 30; i++ {
		req := postJSON("/api/shorten", `{"url":"https://example.com/x"}`)
		req.Host = "example.com"
		req.RemoteAddr = "203.0.113.9:9000"
		rr := httptest.NewRecorder()
		h.HandleShorten(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("request %d: status = %d, want 201; body=%s", i+1, rr.Code, rr.Body.String())
		}
	}

	req := postJSON("/api/shorten", `{"url":"https://example.com/x"}`)
	req.Host = "example.com"
	req.RemoteAddr = "203.0.113.9:9000"
	rr := httptest.NewRecorder()
	h.HandleShorten(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("request 31: status = %d, want 429", rr.Code)
	}
}

// TestShortenSameSlugIdempotentConflict fires the SAME create twice: the
// first 201 writes one row, the duplicate gets 409 (UNIQUE backstop) and
// must not write a second row: a retry can never fork the data.
func TestShortenSameSlugIdempotentConflict(t *testing.T) {
	s := &fakeStore{urls: map[string]string{}}
	h := newTestHandler(s)

	for i, want := range []int{http.StatusCreated, http.StatusConflict} {
		req := postJSON("/api/shorten", `{"url":"https://example.com/x","slug":"stable-slug"}`)
		req.Host = "example.com"
		rr := httptest.NewRecorder()
		h.HandleShorten(rr, req)
		if rr.Code != want {
			t.Fatalf("attempt %d: status = %d, want %d; body=%s", i+1, rr.Code, want, rr.Body.String())
		}
	}
	if len(s.created) != 1 {
		t.Fatalf("rows created = %d, want exactly 1", len(s.created))
	}
}

// TestNoCORSHeaders: the API ships with no Access-Control-Allow-* headers at
// all (no CORS route, no wildcard): a foreign origin can neither read JSON
// responses nor issue state-changing calls (browser SOP + SameSite=Lax
// cookie). If a CORS header ever appears, the exposure must be a conscious,
// reviewed decision.
func TestNoCORSHeaders(t *testing.T) {
	s := &fakeStore{urls: map[string]string{}}
	h := newTestHandler(s)

	req := postJSON("/api/shorten", `{"url":"https://example.com/x"}`)
	req.Host = "example.com"
	req.Header.Set("Origin", "https://evil.example")
	rr := httptest.NewRecorder()
	h.HandleShorten(rr, req)

	for _, hdr := range []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Credentials",
		"Access-Control-Allow-Methods",
	} {
		if v := rr.Header().Get(hdr); v != "" {
			t.Errorf("%s = %q, want absent (no CORS)", hdr, v)
		}
	}
}
