package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jejak/internal/auth"
	"jejak/internal/db"
	"jejak/internal/middleware"
)

// newAuthedLinkHandler = newTestHandler + a session store (authedPut needs
// h.Auth to mint a session cookie).
func newAuthedLinkHandler(s *fakeStore) *Handler {
	h := newTestHandler(s)
	h.Auth = auth.NewMemoryStore()
	return h
}

// TestHandleShortenExpiryValidation pins down the two expiry error tiers:
// malformed format = 400, valid but <= 1 hour from now = 422 plus a field
// error (the request is well-formed, its semantics cannot be met). Success
// stores pure UTC (the zone-less TIMESTAMP column is absolute).
func TestHandleShortenExpiryValidation(t *testing.T) {
	tests := []struct {
		name      string
		expires   string // "" = field absen
		wantCode  int
		wantField bool // body mentions "expires_at"
		check     func(t *testing.T, s *fakeStore)
	}{
		{
			name:     "malformed -> 400 + field",
			expires:  "not-a-timestamp",
			wantCode: http.StatusBadRequest, wantField: true,
		},
		{
			name:     "past value -> 422 + field",
			expires:  "2020-01-01T00:00:00Z",
			wantCode: http.StatusUnprocessableEntity, wantField: true,
		},
		{
			name:     "within 1 hour -> 422",
			expires:  time.Now().UTC().Add(30 * time.Minute).Format(time.RFC3339),
			wantCode: http.StatusUnprocessableEntity, wantField: true,
		},
		{
			name:     "48h ahead -> 201 stored UTC",
			expires:  time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339),
			wantCode: http.StatusCreated,
			check: func(t *testing.T, s *fakeStore) {
				if len(s.created) != 1 || s.created[0].expiresAt == nil {
					t.Fatalf("created = %+v, want 1 with expiresAt", s.created)
				}
				got := *s.created[0].expiresAt
				if got.Location() != time.UTC {
					t.Errorf("expiresAt location = %v, want UTC", got.Location())
				}
				if !got.After(time.Now().UTC()) {
					t.Errorf("expiresAt %v not in the future", got)
				}
			},
		},
		{
			name:     "absent -> 201 no expiry",
			expires:  "",
			wantCode: http.StatusCreated,
			check: func(t *testing.T, s *fakeStore) {
				if len(s.created) != 1 || s.created[0].expiresAt != nil {
					t.Fatalf("created = %+v, want 1 with nil expiresAt", s.created)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &fakeStore{}
			h := newTestHandler(s)
			body := `{"url":"https://example.com/page"}`
			if tt.expires != "" {
				body = `{"url":"https://example.com/page","expires_at":"` + tt.expires + `"}`
			}
			req := postJSON("/api/shorten", body)
			req.Host = "example.com"
			rr := httptest.NewRecorder()
			h.HandleShorten(rr, req)

			if rr.Code != tt.wantCode {
				t.Fatalf("status = %d (body %q), want %d", rr.Code, rr.Body.String(), tt.wantCode)
			}
			if tt.wantField && !strings.Contains(rr.Body.String(), "expires_at") {
				t.Errorf("body %q must mention field expires_at", rr.Body.String())
			}
			if tt.check != nil {
				tt.check(t, s)
			}
		})
	}
}

// TestHandleUpdateLinkExpiry pins down the 3 optionalTime cases: set (RFC3339
// > 1h), clear (null), and validation (400/422), plus the 404 scoping for a
// code the caller does not own.
func TestHandleUpdateLinkExpiry(t *testing.T) {
	t.Run("set future value", func(t *testing.T) {
		s := &fakeStore{}
		s.CreateURL("mylink", "https://example.com", ptrInt64(7), "[]", nil)
		h := newAuthedLinkHandler(s)
		future := time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("mylink", rr, authedPut(s, h, 7, "/api/links/mylink", `{"expires_at":"`+future+`"}`))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %q)", rr.Code, rr.Body.String())
		}
		if len(s.expiryCalls) != 1 || s.expiryCalls[0].expiresAt == nil {
			t.Fatalf("expiryCalls = %+v, want 1 set call", s.expiryCalls)
		}
	})

	t.Run("null clears expiry", func(t *testing.T) {
		s := &fakeStore{}
		s.CreateURL("mylink", "https://example.com", ptrInt64(7), "[]", nil)
		h := newAuthedLinkHandler(s)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("mylink", rr, authedPut(s, h, 7, "/api/links/mylink", `{"expires_at":null}`))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %q)", rr.Code, rr.Body.String())
		}
		if len(s.expiryCalls) != 1 || s.expiryCalls[0].expiresAt != nil {
			t.Fatalf("expiryCalls = %+v, want 1 clear call (nil)", s.expiryCalls)
		}
	})

	t.Run("past value -> 422 untouched", func(t *testing.T) {
		s := &fakeStore{}
		s.CreateURL("mylink", "https://example.com", ptrInt64(7), "[]", nil)
		h := newAuthedLinkHandler(s)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("mylink", rr, authedPut(s, h, 7, "/api/links/mylink",
			`{"expires_at":"2020-01-01T00:00:00Z"}`))
		if rr.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422 (body %q)", rr.Code, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), "expires_at") {
			t.Errorf("body must mention field expires_at")
		}
		if len(s.expiryCalls) != 0 {
			t.Fatal("store must not be touched on 422")
		}
	})

	t.Run("malformed -> 400", func(t *testing.T) {
		s := &fakeStore{}
		s.CreateURL("mylink", "https://example.com", ptrInt64(7), "[]", nil)
		h := newAuthedLinkHandler(s)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("mylink", rr, authedPut(s, h, 7, "/api/links/mylink", `{"expires_at":123}`))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
		if len(s.expiryCalls) != 0 {
			t.Fatal("store must not be touched on 400")
		}
	})

	t.Run("not yours -> 404", func(t *testing.T) {
		s := &fakeStore{} // code never created → ErrNoRows
		h := newAuthedLinkHandler(s)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("stranger", rr, authedPut(s, h, 7, "/api/links/stranger", `{"expires_at":null}`))
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rr.Code)
		}
	})
}

// TestHandleRedirectExpired410DB: a link whose expires_at has passed in the
// DB → 410 HTML (not plain text), NO Location, NO cache write, NO logClick.
// This check follows is_active (a state) because it is a TIME condition.
func TestHandleRedirectExpired410DB(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)
	past := time.Now().UTC().Add(-time.Minute)
	s.link = db.Link{
		ShortCode: "old001", OriginalURL: "https://default.com",
		IsActive: true, ExpiresAt: &past,
	}
	h.Cache = &mapCache{}

	req := httptest.NewRequest(http.MethodGet, "/r/old001", nil)
	rr := httptest.NewRecorder()
	h.HandleRedirect("old001", rr, req)

	if rr.Code != http.StatusGone {
		t.Fatalf("status = %d, want 410 (body %q)", rr.Code, rr.Body.String())
	}
	if loc := rr.Header().Get("Location"); loc != "" {
		t.Fatalf("expired link must not redirect, got Location %q", loc)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(rr.Body.String(), "kedaluwarsa") {
		t.Errorf("body must say kedaluwarsa, got %q", rr.Body.String())
	}
	if _, ok := h.Cache.Get("old001"); ok {
		t.Error("expired link must not be written to cache")
	}
}

// TestHandleRedirectExpiredCacheHit410: a cache entry holding an expires_at
// that has already passed → 410 + EVICT (the next request reads the DB), with
// no Location.
func TestHandleRedirectExpiredCacheHit410(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)
	h.Cache = &mapCache{data: map[string]string{
		"old002": `{"url":"https://default.com","expires_at":"2020-01-01T00:00:00Z"}`,
	}}

	req := httptest.NewRequest(http.MethodGet, "/r/old002", nil)
	rr := httptest.NewRecorder()
	h.HandleRedirect("old002", rr, req)

	if rr.Code != http.StatusGone {
		t.Fatalf("status = %d, want 410 (body %q)", rr.Code, rr.Body.String())
	}
	if loc := rr.Header().Get("Location"); loc != "" {
		t.Fatalf("expired cache hit must not redirect, got Location %q", loc)
	}
	if _, ok := h.Cache.Get("old002"); ok {
		t.Fatal("expired cache entry not evicted after 410")
	}
	if !strings.Contains(rr.Body.String(), "kedaluwarsa") {
		t.Errorf("body must say kedaluwarsa, got %q", rr.Body.String())
	}
}

// TestHandleRedirectActiveExpiryClampsTTL: a link that has NOT passed but is
// close to its deadline → normal 302, and the cache TTL is clamped to the
// remaining lifetime (not 300 seconds: an entry must not outlive
// expires_at).
func TestHandleRedirectActiveExpiryClampsTTL(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)
	near := time.Now().UTC().Add(120 * time.Second)
	s.link = db.Link{
		ShortCode: "soon01", OriginalURL: "https://default.com",
		IsActive: true, ExpiresAt: &near,
	}
	h.Cache = &mapCache{}

	req := httptest.NewRequest(http.MethodGet, "/r/soon01", nil)
	rr := httptest.NewRecorder()
	h.HandleRedirect("soon01", rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 (link belum lewat)", rr.Code)
	}
	mc := h.Cache.(*mapCache)
	if mc.ttls["soon01"] < 100 || mc.ttls["soon01"] > 121 {
		t.Fatalf("cache TTL = %d, want ~120 (diklamp ke sisa expiry)", mc.ttls["soon01"])
	}
}

// TestHandleRedirectActiveLinkStill302: REGRESSION for an ordinary link: an
// active link with no expiry, no device rule and no cache still 302s to
// original_url (no expiry change may break the normal redirect path).
func TestHandleRedirectActiveLinkStill302(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)
	s.link = db.Link{
		ShortCode: "plain1", OriginalURL: "https://example.org/target",
		IsActive: true, // ExpiresAt nil
	}
	h.Cache = &mapCache{}

	req := httptest.NewRequest(http.MethodGet, "/r/plain1", nil)
	req.Host = "sho.rt"
	rr := httptest.NewRecorder()
	h.HandleRedirect("plain1", rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "https://example.org/target" {
		t.Fatalf("Location = %q, want original", loc)
	}
	// Long-standing behavior that must not change: the cache is filled with
	// the default TTL when a link has no expiry (redirectTTL(nil)=300).
	if h.Cache != nil {
		if mc := h.Cache.(*mapCache); mc.ttls["plain1"] != 300 {
			t.Errorf("TTL = %d, want 300", mc.ttls["plain1"])
		}
	}
}

// TestHandleListLinksStatusJSON: scanLinks fills the derived Status: the
// list response carries expires_at + status (active/scheduled/expired) for the
// dashboard badge. (In the fake, the values already match scanLinks output.)
func TestHandleListLinksStatusJSON(t *testing.T) {
	future := time.Now().UTC().Add(48 * time.Hour)
	past := time.Now().UTC().Add(-time.Hour)
	s := &fakeStore{creatorLinks: []db.Link{
		{ShortCode: "a00001", OriginalURL: "https://a.com", IsActive: true, Status: "active"},
		{ShortCode: "b00002", OriginalURL: "https://b.com", IsActive: true, ExpiresAt: &future, Status: "scheduled"},
		{ShortCode: "c00003", OriginalURL: "https://c.com", IsActive: true, ExpiresAt: &past, Status: "expired"},
	}}
	h := newTestHandler(s)

	rr := httptest.NewRecorder()
	h.HandleListLinks(rr, middleware.WithCreator(
		httptest.NewRequest(http.MethodGet, "/api/links", nil), 7))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{
		`"status":"active"`, `"status":"scheduled"`, `"status":"expired"`, `"expires_at"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %s", want)
		}
	}
}

// ptrInt64 is a tiny helper for building owned-link fixtures.
func ptrInt64(v int64) *int64 { return &v }
