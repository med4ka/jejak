package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSetCookieFlags pins the session cookie security contract: HttpOnly
// (JS cannot read it), SameSite=Lax (CSRF surface limited to top-level
// navigations), Path=/ (dashboard everywhere), MaxAge = SessionTTL, and
// Secure exactly as SecureCookies dictates (COOKIE_SECURE wiring).
func TestSetCookieFlags(t *testing.T) {
	rr := httptest.NewRecorder()
	SetCookie(rr, "tok-123")

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	c := cookies[0]
	if c.Name != CookieName {
		t.Errorf("name = %q, want %q", c.Name, CookieName)
	}
	if c.Value != "tok-123" {
		t.Errorf("value = %q, want tok-123", c.Value)
	}
	if !c.HttpOnly {
		t.Error("HttpOnly missing")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", c.SameSite)
	}
	if c.Path != "/" {
		t.Errorf("Path = %q, want /", c.Path)
	}
	if c.MaxAge != int(SessionTTL.Seconds()) {
		t.Errorf("MaxAge = %d, want %d", c.MaxAge, int(SessionTTL.Seconds()))
	}
	if c.Secure != SecureCookies {
		t.Errorf("Secure = %v, want SecureCookies=%v", c.Secure, SecureCookies)
	}

	// Production wiring: COOKIE_SECURE=true must mark the cookie Secure.
	SecureCookies = true
	defer func() { SecureCookies = false }()
	rr2 := httptest.NewRecorder()
	SetCookie(rr2, "tok-456")
	if got := rr2.Result().Cookies()[0]; !got.Secure {
		t.Error("SecureCookies=true did not set Secure")
	}
}

// TestClearCookieMatchesSet: the clearing cookie must carry the SAME
// attributes as the issued one (a mismatched Secure/Path would make the
// browser keep the original) and expire immediately.
func TestClearCookieMatchesSet(t *testing.T) {
	rr := httptest.NewRecorder()
	ClearCookie(rr)

	c := rr.Result().Cookies()[0]
	if c.Name != CookieName {
		t.Errorf("name = %q, want %q", c.Name, CookieName)
	}
	if c.Value != "" {
		t.Errorf("value = %q, want empty", c.Value)
	}
	if c.MaxAge >= 0 {
		t.Errorf("MaxAge = %d, want negative (immediate expiry)", c.MaxAge)
	}
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" {
		t.Errorf("attributes diverge from SetCookie: %+v", c)
	}
}

// TestTokenFromRequest: cookie present → value; absent → "" (never an
// error: callers treat "" as anonymous).
func TestTokenFromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := TokenFromRequest(req); got != "" {
		t.Fatalf("no cookie: %q, want empty", got)
	}
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "abc"})
	if got := TokenFromRequest(req); got != "abc" {
		t.Fatalf("with cookie: %q, want abc", got)
	}
	// A foreign cookie must not be mistaken for the session.
	req.AddCookie(&http.Cookie{Name: "other", Value: "zzz"})
	if got := TokenFromRequest(req); got != "abc" {
		t.Fatalf("after foreign cookie: %q, want abc", got)
	}
}

// TestSessionTTLPositive sanity: cookie MaxAge and the store TTL agree on a
// sane lifetime (a zero/negative TTL would make sessions die instantly).
func TestSessionTTLPositive(t *testing.T) {
	if SessionTTL <= 0 {
		t.Fatalf("SessionTTL = %v, want > 0", SessionTTL)
	}
	if !strings.Contains(CookieName, "session") && CookieName == "" {
		t.Fatal("CookieName empty")
	}
}
