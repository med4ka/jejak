package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jejak/internal/db"
	"jejak/internal/middleware"
	"jejak/internal/shortener"
)

func TestHandleShorten(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantCode  int
		wantSlice string // substring expected in response body
		check     func(t *testing.T, s *fakeStore, rr *httptest.ResponseRecorder)
	}{
		{
			name:      "random code success",
			body:      `{"url":"https://example.com/page"}`,
			wantCode:  http.StatusCreated,
			wantSlice: "/r/",
			check: func(t *testing.T, s *fakeStore, rr *httptest.ResponseRecorder) {
				if len(s.created) != 1 {
					t.Fatalf("CreateURL called %d times, want 1", len(s.created))
				}
				code := s.created[0].code
				if len(code) != 6 {
					t.Fatalf("generated code length = %d, want 6", len(code))
				}
				for _, c := range code {
					if !validCodeChar(byte(c)) {
						t.Fatalf("generated code %q contains disallowed char %q", code, c)
					}
				}
				// Creator is nil (anonymous request) — ownership must stay NULL.
				if s.created[0].creatorID != nil {
					t.Fatalf("anonymous shorten got creatorID %v, want nil", *s.created[0].creatorID)
				}
			},
		},
		{
			name:      "custom slug success",
			body:      `{"url":"https://example.com/x","slug":"mylink"}`,
			wantCode:  http.StatusCreated,
			wantSlice: "/r/mylink",
			check: func(t *testing.T, s *fakeStore, rr *httptest.ResponseRecorder) {
				if len(s.created) != 1 || s.created[0].code != "mylink" {
					t.Fatalf("CreateURL got %v, want code mylink", s.created)
				}
			},
		},
		{
			name:      "slug already taken -> 409",
			body:      `{"url":"https://example.com/x","slug":"taken"}`,
			wantCode:  http.StatusConflict,
			wantSlice: "taken",
		},
		{
			name:      "reserved slug -> 400",
			body:      `{"url":"https://example.com/x","slug":"dashboard"}`,
			wantCode:  http.StatusBadRequest,
			wantSlice: "reserved",
		},
		{
			name:      "javascript scheme -> 400",
			body:      `{"url":"javascript:alert(1)"}`,
			wantCode:  http.StatusBadRequest,
			wantSlice: "http(s)",
		},
		{
			name:     "missing url -> 400",
			body:     `{"url":""}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "scheme-relative url -> 400",
			body:     `{"url":"//evil.example.com"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "local uploads path -> 400",
			body:     `{"url":"/uploads/avatars/1.png"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid slug format -> 400",
			body:     `{"url":"https://example.com/x","slug":"ab"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:      "more than 5 tags -> 400",
			body:      `{"url":"https://example.com/x","tags":["a","b","c","d","e","f"]}`,
			wantCode:  http.StatusBadRequest,
			wantSlice: "tags",
		},
		{
			name:     "tag longer than 20 chars -> 400",
			body:     `{"url":"https://example.com/x","tags":["aaaaaaaaaaaaaaaaaaaaaaaaa"]}`,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &fakeStore{urls: map[string]string{"taken": "https://existing.com"}}
			h := newTestHandler(s)

			req := postJSON("/api/shorten", tt.body)
			req.Host = "example.com"
			rr := httptest.NewRecorder()

			h.HandleShorten(rr, req)

			if rr.Code != tt.wantCode {
				t.Fatalf("status = %d (body %q), want %d", rr.Code, rr.Body.String(), tt.wantCode)
			}
			if tt.wantSlice != "" && !strings.Contains(strings.ToLower(rr.Body.String()), tt.wantSlice) {
				t.Errorf("body %q missing %q", rr.Body.String(), tt.wantSlice)
			}
			if tt.check != nil {
				tt.check(t, s, rr)
			}
		})
	}
}

func validCodeChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// TestHandleShortenReservedAllWords drives every ReservedSlugs entry through
// the handler to prove the endpoint itself rejects them (not just the helper).
func TestHandleShortenReservedAllWords(t *testing.T) {
	s := &fakeStore{urls: map[string]string{}}
	h := newTestHandler(s)

	for _, reserved := range shortener.ReservedSlugs {
		req := postJSON("/api/shorten", `{"url":"https://example.com/x","slug":"`+reserved+`"}`)
		req.Host = "example.com"
		rr := httptest.NewRecorder()
		h.HandleShorten(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("slug %q -> status %d, want 400", reserved, rr.Code)
		}
	}
}

// TestParseTarget handles both Smart Link JSON cache entries and legacy bare-URL
// entries written before Smart Link (backward compat for warm caches).
func TestParseTarget(t *testing.T) {
	jsonEntry := `{"url":"https://default.com","device_rules":{"ios":"https://ios.example.com"}}`
	target, ok := parseTarget(jsonEntry)
	if !ok {
		t.Fatal("JSON target should parse")
	}
	if target.URL != "https://default.com" || target.DeviceRules["ios"] != "https://ios.example.com" {
		t.Fatalf("parsed target wrong: %+v", target)
	}

	// Legacy: cache holds nothing but a bare URL string.
	if _, ok := parseTarget("https://legacy.example.com"); ok {
		t.Fatal("bare URL must NOT parse as a JSON target (caller falls back)")
	}

	// Corrupt / empty cache values must fail parse, not panic.
	if _, ok := parseTarget(""); ok {
		t.Fatal("empty value must not parse")
	}
	if _, ok := parseTarget(`{"url":`); ok {
		t.Fatal("corrupt JSON must not parse")
	}
}

func TestDetectDevice(t *testing.T) {
	tests := []struct {
		ua   string
		want string
	}{
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit", "ios"},
		{"Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit", "ios"},
		{"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit", "android"},
		{"Mozilla/5.0 (X11; Linux x86_64) Chrome/120 Safari", ""}, // desktop
		{"", ""},
		{"iPhone", "ios"}, // bare keyword still matches
	}
	for _, tt := range tests {
		if got := detectDevice(tt.ua); got != tt.want {
			t.Errorf("detectDevice(%q) = %q, want %q", tt.ua, got, tt.want)
		}
	}
}

func TestRedirectTargetPickURL(t *testing.T) {
	target := redirectTarget{URL: "https://default.com", DeviceRules: map[string]string{
		"ios":     "https://app-store.example.com/x",
		"android": "https://play.example.com/x",
	}}
	tests := []struct {
		device string
		want   string
	}{
		{"ios", "https://app-store.example.com/x"},
		{"android", "https://play.example.com/x"},
		{"", "https://default.com"},        // desktop: no matching key
		{"windows", "https://default.com"}, // unknown key
	}
	for _, tt := range tests {
		if got := target.pickURL(tt.device); got != tt.want {
			t.Errorf("pickURL(%q) = %q, want %q", tt.device, got, tt.want)
		}
	}

	// Empty value in a matched rule -> fallback to default.
	empty := redirectTarget{URL: "https://default.com", DeviceRules: map[string]string{"ios": "   "}}
	if got := empty.pickURL("ios"); got != "https://default.com" {
		t.Errorf("pickURL with blank rule = %q, want default", got)
	}
}

// TestHandleRedirectDeviceRouting proves the redirect path routes by device:
// iPhone UA -> iOS rule URL, desktop UA -> default URL (no matching rule).
func TestHandleRedirectDeviceRouting(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)

	s.link = db.Link{
		ShortCode: "abc123", OriginalURL: "https://default.com",
		DeviceRules: map[string]string{"ios": "https://ios.example.com"},
		IsActive:    true,
	}

	tests := []struct {
		name string
		ua   string
		want string
	}{
		{"iphone-routes-to-ios", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)", "https://ios.example.com"},
		{"desktop-uses-default", "Mozilla/5.0 (X11; Linux x86_64) Chrome/120", "https://default.com"},
		{"android-no-rule-uses-default", "Mozilla/5.0 (Linux; Android 14; Pixel 8)", "https://default.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/r/abc123", nil)
			req.Host = "example.com"
			req.Header.Set("User-Agent", tt.ua)
			rr := httptest.NewRecorder()
			h.HandleRedirect("abc123", rr, req)
			if rr.Code != http.StatusFound {
				t.Fatalf("status = %d, want 302", rr.Code)
			}
			if got := rr.Header().Get("Location"); got != tt.want {
				t.Fatalf("Location = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestHandleRedirectUnknownLink404 ensures the smarter read path still 404s
// for unknown codes (no nil deref on zero Link).
func TestHandleRedirectUnknownLink404(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)
	req := httptest.NewRequest(http.MethodGet, "/r/nope12", nil)
	rr := httptest.NewRecorder()
	h.HandleRedirect("nope12", rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

// TestHandleRedirectDisabledLink410 proves the Fase 13 lifecycle is honored:
// a link with is_active=false must NOT redirect (410 Gone) and must not leak
// into a Location header. DB path (cache nil).
func TestHandleRedirectDisabledLink410(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)
	s.link = db.Link{
		ShortCode: "off01", OriginalURL: "https://default.com",
		IsActive: false,
	}

	req := httptest.NewRequest(http.MethodGet, "/r/off01", nil)
	rr := httptest.NewRecorder()
	h.HandleRedirect("off01", rr, req)
	if rr.Code != http.StatusGone {
		t.Fatalf("status = %d, want 410 (body %q)", rr.Code, rr.Body.String())
	}
	if loc := rr.Header().Get("Location"); loc != "" {
		t.Fatalf("disabled link must not redirect, got Location %q", loc)
	}
}

// TestHandleRedirectDisabledLinkCacheHit410 proves a cache entry marked
// disabled is honored (410) AND evicted, so re-enabling the link makes the
// very next request read the fresh DB row instead of a stale 300s entry.
func TestHandleRedirectDisabledLinkCacheHit410(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)
	h.Cache = &mapCache{data: map[string]string{"off01": `{"url":"https://default.com","disabled":true}`}}

	req := httptest.NewRequest(http.MethodGet, "/r/off01", nil)
	rr := httptest.NewRecorder()
	h.HandleRedirect("off01", rr, req)
	if rr.Code != http.StatusGone {
		t.Fatalf("status = %d, want 410 (body %q)", rr.Code, rr.Body.String())
	}
	if _, ok := h.Cache.Get("off01"); ok {
		t.Fatal("disabled cache entry not evicted after 410")
	}
}

// TestHandleRedirectCacheHitActiveStillRedirects locks in that a cache hit for
// an ACTIVE link keeps routing by device (and cleanly handles a cache that is
// writable — the legacy code path always touched the store first).
func TestHandleRedirectCacheHitActiveStillRedirects(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)
	h.Cache = &mapCache{data: map[string]string{
		"abc123": `{"url":"https://default.com","device_rules":{"ios":"https://ios.example.com"}}`,
	}}

	req := httptest.NewRequest(http.MethodGet, "/r/abc123", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)")
	rr := httptest.NewRecorder()
	h.HandleRedirect("abc123", rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "https://ios.example.com" {
		t.Fatalf("Location = %q, want https://ios.example.com", got)
	}
}

// mapCache is a trivial in-memory cache.Cache double for redirect tests.
type mapCache struct {
	data map[string]string
}

func (c *mapCache) Get(key string) (string, bool) {
	v, ok := c.data[key]
	return v, ok
}

func (c *mapCache) Set(key, value string, ttl int64) error {
	if c.data == nil {
		c.data = map[string]string{}
	}
	c.data[key] = value
	return nil
}

func (c *mapCache) Delete(key string) error {
	delete(c.data, key)
	return nil
}

// TestHandleListLinksAuthAndScoping proves GET /api/links is NOT public anymore:
// anonymous -> 401 (no data leak), and a logged-in caller gets ONLY their own
// links (store called with their creator id), not the global table.
func TestHandleListLinksAuthAndScoping(t *testing.T) {
	s := &fakeStore{
		creatorLinks: []db.Link{
			{ShortCode: "mine01", OriginalURL: "https://mine.example.com"},
		},
	}
	h := newTestHandler(s)

	anon := httptest.NewRequest(http.MethodGet, "/api/links", nil)
	rrAnon := httptest.NewRecorder()
	h.HandleListLinks(rrAnon, anon)
	if rrAnon.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous status = %d, want 401", rrAnon.Code)
	}

	authed := middleware.WithCreator(httptest.NewRequest(http.MethodGet, "/api/links", nil), 7)
	rrAuth := httptest.NewRecorder()
	h.HandleListLinks(rrAuth, authed)
	if rrAuth.Code != http.StatusOK {
		t.Fatalf("authed status = %d, want 200 (body %q)", rrAuth.Code, rrAuth.Body.String())
	}
	if !strings.Contains(rrAuth.Body.String(), "mine01") {
		t.Fatalf("body %q missing caller's own link", rrAuth.Body.String())
	}
}
