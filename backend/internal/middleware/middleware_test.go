package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"jejak/internal/auth"
	"jejak/internal/ratelimit"
)

// capturedWriter records what identity a wrapped handler saw.
type probeResult struct {
	id      *int64
	written string
}

func probe() (http.Handler, *probeResult) {
	res := &probeResult{}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res.id = CreatorID(r)
		if res.id == nil {
			res.written = "anon"
		} else {
			res.written = "user"
		}
		w.WriteHeader(http.StatusOK)
	})
	return h, res
}

func TestRequireAuthAnonymous401(t *testing.T) {
	h, _ := probe()
	st := auth.NewMemoryStore()
	wrapped := RequireAuth(st)(h)

	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not a JSON error envelope: %v (%q)", err, rr.Body.String())
	}
	if body.Code != "AUTH_REQUIRED" || body.Message == "" {
		t.Errorf("body = %+v, want code AUTH_REQUIRED with a fallback message", body)
	}
}

func TestRequireAuthNilStore401(t *testing.T) {
	// Auth disabled must fail safe: required routes reject everyone.
	h, _ := probe()
	wrapped := RequireAuth(nil)(h)

	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (nil store)", rr.Code)
	}
}

func TestRequireAuthValidSession(t *testing.T) {
	h, res := probe()
	st := auth.NewMemoryStore()
	token := mustCreateSession(t, st, 42)
	wrapped := RequireAuth(st)(h)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if res.id == nil || *res.id != 42 {
		t.Fatalf("creator seen by handler = %v, want 42", res.id)
	}
	// Sliding TTL: a valid hit re-issues the session cookie.
	if got := rr.Header().Get("Set-Cookie"); got == "" {
		t.Error("valid session did not re-issue Set-Cookie (sliding TTL broken)")
	}
}

func TestRequireAuthInvalidToken401(t *testing.T) {
	h, _ := probe()
	st := auth.NewMemoryStore()
	wrapped := RequireAuth(st)(h)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "bogus-token"})
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestOptionalAuthAnonymousPassesThrough(t *testing.T) {
	h, res := probe()
	st := auth.NewMemoryStore()
	wrapped := OptionalAuth(st)(h)

	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (anonymous allowed)", rr.Code)
	}
	if res.id != nil || res.written != "anon" {
		t.Fatalf("handler saw id=%v written=%q, want anonymous", res.id, res.written)
	}
}

func TestOptionalAuthWithSessionSetsCreator(t *testing.T) {
	h, res := probe()
	st := auth.NewMemoryStore()
	token := mustCreateSession(t, st, 7)
	wrapped := OptionalAuth(st)(h)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)
	if res.id == nil || *res.id != 7 {
		t.Fatalf("handler saw id=%v, want 7", res.id)
	}
}

func TestWithCreatorRoundTrip(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = WithCreator(req, 99)
	if id := CreatorID(req); id == nil || *id != 99 {
		t.Fatalf("CreatorID after WithCreator = %v, want 99", id)
	}

	plain := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := CreatorID(plain); got != nil {
		t.Fatalf("CreatorID on plain request = %v, want nil", got)
	}
}

func TestRateLimit(t *testing.T) {
	h, _ := probe()
	lim := ratelimit.NewLimiterWithMax(1)
	wrapped := RateLimit(lim, func(r *http.Request) string { return "bucket-a" })(h)

	first := httptest.NewRecorder()
	wrapped.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want 200", first.Code)
	}

	second := httptest.NewRecorder()
	wrapped.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", second.Code)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Error("429 missing Retry-After header")
	}
}

func TestRateLimitKeysBucketSeparated(t *testing.T) {
	h, _ := probe()
	lim := ratelimit.NewLimiterWithMax(1)
	keyOf := func(bucket string) func(*http.Request) string {
		return func(*http.Request) string { return bucket }
	}
	wrappedA := RateLimit(lim, keyOf("a"))(h)
	wrappedB := RateLimit(lim, keyOf("b"))(h)

	// Consume bucket a; bucket b must be unaffected.
	a1 := httptest.NewRecorder()
	wrappedA.ServeHTTP(a1, httptest.NewRequest(http.MethodGet, "/", nil))
	b1 := httptest.NewRecorder()
	wrappedB.ServeHTTP(b1, httptest.NewRequest(http.MethodGet, "/", nil))
	a2 := httptest.NewRecorder()
	wrappedA.ServeHTTP(a2, httptest.NewRequest(http.MethodGet, "/", nil))
	if b1.Code != http.StatusOK {
		t.Fatalf("bucket b after a consumed: status = %d, want 200", b1.Code)
	}
	if a2.Code != http.StatusTooManyRequests {
		t.Fatalf("bucket a second hit: status = %d, want 429", a2.Code)
	}
}

func TestRateLimitNilLimiterDisabled(t *testing.T) {
	h, _ := probe()
	wrapped := RateLimit(nil, func(r *http.Request) string { return "x" })(h)
	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		wrapped.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (nil limiter = disabled)", rr.Code)
		}
	}
}

func mustCreateSession(t *testing.T, st auth.Store, creatorID int64) string {
	t.Helper()
	token, err := st.Create(creatorID)
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}
	return token
}
