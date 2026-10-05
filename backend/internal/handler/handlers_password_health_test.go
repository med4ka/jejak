package handler

// Tests for the migration 17 features (2026-10-04): per-link password gate,
// health fallback pages, manual check trigger, notification feed, and the
// new PUT fields (password / fallback_url).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jejak/internal/auth"
	"jejak/internal/db"
	"jejak/internal/health"
	"jejak/internal/middleware"
)

// bcryptHashOf returns a real bcrypt hash for the given plaintext (cost 10,
// auth.HashPassword) - tests must use a real hash: CheckPassword verifies
// with bcrypt, not with string comparison.
func bcryptHashOf(t *testing.T, pw string) string {
	t.Helper()
	h, err := auth.HashPassword(pw)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return h
}

// dbLink builds the minimal active redirect row for gate tests.
func dbLink(code, url string) db.Link {
	return db.Link{
		ShortCode:    code,
		OriginalURL:  url,
		IsActive:     true,
		HealthStatus: health.StatusUnknown,
	}
}

// authedGet is authedReq for GET (the notification feed checks Method).
// The session cookie is not consulted when handlers are called directly -
// middleware.CreatorID comes from WithCreator (same as every other test).
func authedGet(h *Handler, creatorID int64, path string) *http.Request {
	return middleware.WithCreator(httptest.NewRequest(http.MethodGet, path, nil), creatorID)
}

// TestHandleShortenPassword pins the create-time password rules: a valid
// password is bcrypt-hashed (never stored plaintext), and the 4-72 length
// guard answers 400 with a "password" field error (bcrypt would silently
// truncate at 72 bytes otherwise).
func TestHandleShortenPassword(t *testing.T) {
	t.Run("password is hashed", func(t *testing.T) {
		s := &fakeStore{}
		h := newTestHandler(s)
		req := postJSON("/api/shorten", `{"url":"https://example.com","password":"test123"}`)
		req.Host = "example.com"
		rr := httptest.NewRecorder()
		h.HandleShorten(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("status = %d (body %q), want 201", rr.Code, rr.Body.String())
		}
		got := s.created[0].passwordHash
		if got == "" || got == "test123" || !strings.HasPrefix(got, "$2") {
			t.Fatalf("passwordHash = %q, want bcrypt hash", got)
		}
	})

	t.Run("no password stores empty hash", func(t *testing.T) {
		s := &fakeStore{}
		h := newTestHandler(s)
		req := postJSON("/api/shorten", `{"url":"https://example.com"}`)
		req.Host = "example.com"
		rr := httptest.NewRecorder()
		h.HandleShorten(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", rr.Code)
		}
		if s.created[0].passwordHash != "" {
			t.Fatalf("passwordHash = %q, want empty", s.created[0].passwordHash)
		}
	})

	for _, tc := range []struct {
		name string
		pw   string
	}{
		{"too short", "abc"},
		{"too long (73)", strings.Repeat("x", 73)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &fakeStore{}
			h := newTestHandler(s)
			body, _ := json.Marshal(map[string]string{"url": "https://example.com", "password": tc.pw})
			req := postJSON("/api/shorten", string(body))
			req.Host = "example.com"
			rr := httptest.NewRecorder()
			h.HandleShorten(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rr.Code)
			}
			if !strings.Contains(rr.Body.String(), "password") {
				t.Fatalf("body %q must mention field password", rr.Body.String())
			}
		})
	}
}

// TestPasswordRedirectGate walks the full visitor flow:
// form -> wrong password (no cookie, no click) -> right password (cookie) ->
// redirect with cookie (302 + click logged).
func TestPasswordRedirectGate(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)
	hash := bcryptHashOf(t, "test123")
	s.link = dbLink("pw001", "https://example.com/dest")
	s.link.PasswordHash = hash

	// 1. GET without cookie: gate page, NO click logged.
	rr := httptest.NewRecorder()
	h.HandleRedirect("pw001", rr, httptest.NewRequest(http.MethodGet, "/r/pw001", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("gate status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Link ini dilindungi password") {
		t.Fatalf("gate body must show the password form, got %q", rr.Body.String())
	}
	if len(s.clickEvents) != 0 {
		t.Fatalf("clicks = %d before verification, want 0", len(s.clickEvents))
	}

	// 2. Wrong password: PRG to ?e=1, no cookie set, still no click.
	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/r/pw001/verify", strings.NewReader("password=salah-banget"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.HandlePasswordVerify("pw001", rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("wrong-password status = %d, want 303", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/r/pw001?e=1" {
		t.Fatalf("Location = %q, want /r/pw001?e=1", loc)
	}
	if len(rr.Result().Cookies()) != 0 {
		t.Fatal("wrong password must not set a cookie")
	}
	if len(s.clickEvents) != 0 {
		t.Fatalf("clicks = %d after failed verify, want 0", len(s.clickEvents))
	}

	// The e=1 flag must render the form WITH the error message.
	rr = httptest.NewRecorder()
	h.HandleRedirect("pw001", rr, httptest.NewRequest(http.MethodGet, "/r/pw001?e=1", nil))
	if !strings.Contains(rr.Body.String(), "Password salah") {
		t.Fatalf("form after e=1 must show the error, got %q", rr.Body.String())
	}

	// 3. Correct password: cookie + PRG back to the link.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/r/pw001/verify", strings.NewReader("password=test123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.HandlePasswordVerify("pw001", rr, req)
	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/r/pw001" {
		t.Fatalf("verify status/loc = %d/%q, want 303 /r/pw001", rr.Code, rr.Header().Get("Location"))
	}
	var access *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == "jejak_link_access_pw001" {
			access = c
		}
	}
	if access == nil {
		t.Fatal("successful verify must set the access cookie")
	}
	if access.Value == hash {
		t.Fatal("cookie must hold the SHA-256 digest, never the bcrypt hash itself")
	}
	if !access.HttpOnly || access.Path != "/r/" {
		t.Fatalf("cookie flags = httpOnly:%v path:%q, want true + /r/", access.HttpOnly, access.Path)
	}

	// 4. GET with the cookie: real redirect + exactly one logged click.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/r/pw001", nil)
	req.AddCookie(access)
	h.HandleRedirect("pw001", rr, req)
	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "https://example.com/dest" {
		t.Fatalf("redirect = %d %q, want 302 to destination", rr.Code, rr.Header().Get("Location"))
	}
	if len(s.clickEvents) != 1 {
		t.Fatalf("clicks = %d after verified redirect, want 1", len(s.clickEvents))
	}
}

// TestPasswordFormLocales: the gate page renders from Accept-Language
// (internal/i18n); no header = Indonesian, en/de = translated. The error
// kind (?e=1) is translated too.
func TestPasswordFormLocales(t *testing.T) {
	setup := func() (*Handler, *fakeStore) {
		s := &fakeStore{}
		s.link = dbLink("pwloc", "https://example.com/dest")
		s.link.PasswordHash = bcryptHashOf(t, "test123")
		return &Handler{Store: s}, s
	}
	cases := []struct {
		name     string
		accept   string
		query    string
		contains []string
		absent   []string
	}{
		{"default Indonesian", "", "", []string{"Link ini dilindungi password", "Buka link"}, nil},
		{"English", "en-US,en;q=0.9", "", []string{"This link is password-protected", "Open link"}, []string{"dilindungi"}},
		{"German", "de-DE,de;q=0.9", "", []string{"passwortgeschützt", "Link öffnen"}, []string{"dilindungi"}},
		{"English wrong-password", "en", "?e=1", []string{"Wrong password. Try again."}, []string{"Password salah"}},
		{"German rate-limit text key exists", "de", "", []string{"Gib das Passwort ein"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := setup()
			req := httptest.NewRequest(http.MethodGet, "/r/pwloc"+tc.query, nil)
			if tc.accept != "" {
				req.Header.Set("Accept-Language", tc.accept)
			}
			rr := httptest.NewRecorder()
			h.HandleRedirect("pwloc", rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rr.Code)
			}
			body := rr.Body.String()
			for _, want := range tc.contains {
				if !strings.Contains(body, want) {
					t.Errorf("body misses %q (Accept-Language %q)", want, tc.accept)
				}
			}
			for _, notWant := range tc.absent {
				if strings.Contains(body, notWant) {
					t.Errorf("body leaks untranslated %q (Accept-Language %q)", notWant, tc.accept)
				}
			}
			if !strings.Contains(body, `lang="`+tcLang(tc.accept)+`"`) {
				t.Errorf("html lang attribute wrong for Accept-Language %q", tc.accept)
			}
		})
	}
}

func tcLang(accept string) string {
	switch {
	case strings.HasPrefix(accept, "en"):
		return "en"
	case strings.HasPrefix(accept, "de"):
		return "de"
	default:
		return "id"
	}
}

// TestHealthGateRedirect: broken destination answers IN PLACE (fallback
// interstitial or 503) without logging a click; healthy redirects normally.
func TestHealthGateRedirect(t *testing.T) {
	t.Run("broken with fallback serves interstitial", func(t *testing.T) {
		s := &fakeStore{}
		h := newTestHandler(s)
		s.link = dbLink("hb001", "https://example.com/dest")
		s.link.HealthStatus = health.StatusBroken
		s.link.FallbackURL = "https://backup.example/id"

		rr := httptest.NewRecorder()
		h.HandleRedirect("hb001", rr, httptest.NewRequest(http.MethodGet, "/r/hb001", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (interstitial)", rr.Code)
		}
		body := rr.Body.String()
		if !strings.Contains(body, "Link ini dialihkan karena tujuan asli tidak tersedia") {
			t.Fatalf("body must carry the banner text, got %q", body)
		}
		if !strings.Contains(body, "https://backup.example/id") {
			t.Fatalf("body must refresh to the fallback URL, got %q", body)
		}
		if len(s.clickEvents) != 0 {
			t.Fatalf("clicks = %d, want 0 (never reached a destination)", len(s.clickEvents))
		}
	})

	t.Run("broken without fallback serves 503", func(t *testing.T) {
		s := &fakeStore{}
		h := newTestHandler(s)
		s.link = dbLink("hb002", "https://example.com/dest")
		s.link.HealthStatus = health.StatusBroken

		rr := httptest.NewRecorder()
		h.HandleRedirect("hb002", rr, httptest.NewRequest(http.MethodGet, "/r/hb002", nil))
		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "Link sedang bermasalah") {
			t.Fatalf("body = %q, want the 503 page", rr.Body.String())
		}
		if len(s.clickEvents) != 0 {
			t.Fatalf("clicks = %d, want 0", len(s.clickEvents))
		}
	})

	t.Run("timeout status still redirects", func(t *testing.T) {
		s := &fakeStore{}
		h := newTestHandler(s)
		s.link = dbLink("hb003", "https://example.com/dest")
		s.link.HealthStatus = health.StatusTimeout

		rr := httptest.NewRecorder()
		h.HandleRedirect("hb003", rr, httptest.NewRequest(http.MethodGet, "/r/hb003", nil))
		if rr.Code != http.StatusFound {
			t.Fatalf("status = %d, want 302 (timeout is not a gate)", rr.Code)
		}
		if len(s.clickEvents) != 1 {
			t.Fatalf("clicks = %d, want 1", len(s.clickEvents))
		}
	})
}

// TestHandleUpdateLinkPasswordAndFallback pins the 3-state PUT contract:
// absent = keep (no write), "" = clear, value = set (password bcrypt-hashed,
// fallback validated as http(s)).
func TestHandleUpdateLinkPasswordAndFallback(t *testing.T) {
	t.Run("set password", func(t *testing.T) {
		s := &fakeStore{}
		s.CreateURL("mylink", "https://example.com", ptrInt64(7), "[]", nil, "")
		h := newAuthedLinkHandler(s)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("mylink", rr, authedPut(s, h, 7, "/api/links/mylink", `{"password":"rahasia"}`))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d (body %q), want 200", rr.Code, rr.Body.String())
		}
		if len(s.passwordCalls) != 1 || !strings.HasPrefix(s.passwordCalls[0].hash, "$2") {
			t.Fatalf("passwordCalls = %+v, want 1 bcrypt hash", s.passwordCalls)
		}
	})

	t.Run("clear password", func(t *testing.T) {
		s := &fakeStore{}
		s.CreateURL("mylink", "https://example.com", ptrInt64(7), "[]", nil, "")
		h := newAuthedLinkHandler(s)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("mylink", rr, authedPut(s, h, 7, "/api/links/mylink", `{"password":""}`))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if len(s.passwordCalls) != 1 || s.passwordCalls[0].hash != "" {
			t.Fatalf("passwordCalls = %+v, want one empty hash (clear)", s.passwordCalls)
		}
	})

	t.Run("absent password does not touch the column", func(t *testing.T) {
		s := &fakeStore{}
		s.CreateURL("mylink", "https://example.com", ptrInt64(7), "[]", nil, "")
		h := newAuthedLinkHandler(s)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("mylink", rr, authedPut(s, h, 7, "/api/links/mylink", `{"is_active":true}`))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if len(s.passwordCalls) != 0 {
			t.Fatalf("passwordCalls = %+v, want none (field absent)", s.passwordCalls)
		}
	})

	t.Run("password too short is a field error", func(t *testing.T) {
		s := &fakeStore{}
		s.CreateURL("mylink", "https://example.com", ptrInt64(7), "[]", nil, "")
		h := newAuthedLinkHandler(s)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("mylink", rr, authedPut(s, h, 7, "/api/links/mylink", `{"password":"ab"}`))
		if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "password") {
			t.Fatalf("status/body = %d/%q, want 400 field error", rr.Code, rr.Body.String())
		}
	})

	t.Run("fallback set and clear", func(t *testing.T) {
		s := &fakeStore{}
		s.CreateURL("mylink", "https://example.com", ptrInt64(7), "[]", nil, "")
		h := newAuthedLinkHandler(s)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("mylink", rr, authedPut(s, h, 7, "/api/links/mylink", `{"fallback_url":"https://backup.example/x"}`))
		if rr.Code != http.StatusOK {
			t.Fatalf("set status = %d, want 200", rr.Code)
		}
		if len(s.fallbackCalls) != 1 || s.fallbackCalls[0].url != "https://backup.example/x" {
			t.Fatalf("fallbackCalls = %+v", s.fallbackCalls)
		}
		rr = httptest.NewRecorder()
		h.HandleUpdateLink("mylink", rr, authedPut(s, h, 7, "/api/links/mylink", `{"fallback_url":""}`))
		if rr.Code != http.StatusOK || s.fallbackCalls[1].url != "" {
			t.Fatalf("clear: status = %d, call = %+v", rr.Code, s.fallbackCalls[1])
		}
	})

	t.Run("invalid fallback URL rejected", func(t *testing.T) {
		s := &fakeStore{}
		s.CreateURL("mylink", "https://example.com", ptrInt64(7), "[]", nil, "")
		h := newAuthedLinkHandler(s)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("mylink", rr, authedPut(s, h, 7, "/api/links/mylink", `{"fallback_url":"javascript:alert(1)"}`))
		if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "fallback_url") {
			t.Fatalf("status/body = %d/%q, want 400 field error", rr.Code, rr.Body.String())
		}
	})
}

// TestNotificationsEndpoints: feed shape (list + unread) and owner-scoped
// read marker (unknown id -> 404, no id probing).
func TestNotificationsEndpoints(t *testing.T) {
	s := &fakeStore{}
	s.notifList = []db.Notification{
		{ID: 5, Type: "link_broken", ShortCode: "abc123", Message: "Link /abc123 tidak bisa dijangkau"},
	}
	s.notifUnread = 1
	h := newAuthedLinkHandler(s)

	rr := httptest.NewRecorder()
	h.HandleListNotifications(rr, authedGet(h, 7, "/api/notifications"))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", rr.Code)
	}
	var got struct {
		Notifications []db.Notification `json:"notifications"`
		Unread        int64             `json:"unread"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v (body %q)", err, rr.Body.String())
	}
	if len(got.Notifications) != 1 || got.Unread != 1 {
		t.Fatalf("feed = %+v, want 1 item / unread 1", got)
	}
	if got.Notifications[0].ShortCode != "abc123" {
		t.Fatalf("short_code = %q", got.Notifications[0].ShortCode)
	}

	// Mark read: existing id -> 200 + recorded; unknown id -> 404.
	rr = httptest.NewRecorder()
	req := authedPut(s, h, 7, "/api/notifications/5/read", "")
	h.HandleMarkNotificationRead("5", rr, req)
	if rr.Code != http.StatusOK || len(s.readCalls) != 1 || s.readCalls[0] != 5 {
		t.Fatalf("read: status = %d, calls = %v", rr.Code, s.readCalls)
	}
	rr = httptest.NewRecorder()
	req = authedPut(s, h, 7, "/api/notifications/999/read", "")
	h.HandleMarkNotificationRead("999", rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("unknown id status = %d, want 404", rr.Code)
	}
}

// TestHandleCheckHealth: ownership runs BEFORE any outbound request (a
// foreign code is 404 with zero checks recorded), and a valid owner check
// reports the classified status back.
func TestHandleCheckHealth(t *testing.T) {
	t.Run("foreign code is 404", func(t *testing.T) {
		s := &fakeStore{}
		h := newAuthedLinkHandler(s)
		s.link = dbLink("hx001", "https://example.com")
		creator := int64(99) // owned by someone else
		s.link.CreatorID = &creator
		h.Checker = &health.Checker{Store: s}

		rr := httptest.NewRecorder()
		req := authedReq(s, h, 7, "/api/links/hx001/check-health", "")
		req.Method = http.MethodPost
		h.HandleCheckHealth("hx001", rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rr.Code)
		}
		if len(s.healthCalls) != 0 {
			t.Fatalf("healthCalls = %d, want 0 (no outbound probe for a foreign code)", len(s.healthCalls))
		}
	})

	t.Run("nil checker is 503", func(t *testing.T) {
		s := &fakeStore{}
		h := newAuthedLinkHandler(s)
		rr := httptest.NewRecorder()
		req := authedReq(s, h, 7, "/api/links/x/check-health", "")
		req.Method = http.MethodPost
		h.HandleCheckHealth("x", rr, req)
		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rr.Code)
		}
	})

	t.Run("owner check runs and reports status", func(t *testing.T) {
		// Live destination: 200 -> healthy.
		dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer dest.Close()

		s := &fakeStore{}
		h := newAuthedLinkHandler(s)
		creator := int64(7)
		s.link = dbLink("hx002", dest.URL)
		s.link.CreatorID = &creator
		h.Checker = &health.Checker{Store: s}

		rr := httptest.NewRecorder()
		req := authedReq(s, h, 7, "/api/links/hx002/check-health", "")
		req.Method = http.MethodPost
		h.HandleCheckHealth("hx002", rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d (body %q), want 200", rr.Code, rr.Body.String())
		}
		var out struct {
			ShortCode     string `json:"short_code"`
			HealthStatus  string `json:"health_status"`
			LastHealthStr string `json:"last_health_check"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if out.HealthStatus != health.StatusHealthy || out.ShortCode != "hx002" {
			t.Fatalf("response = %+v, want healthy/hx002", out)
		}
		if len(s.healthCalls) != 1 || s.healthCalls[0].status != health.StatusHealthy {
			t.Fatalf("healthCalls = %+v, want one healthy write", s.healthCalls)
		}
	})
}
