package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jejak/internal/auth"
	"jejak/internal/db"
	"jejak/internal/middleware"
	"jejak/internal/ratelimit"
)

// newAccountHandler prepares a handler plus a fakeStore holding one creator
// (id 9, password "secret-1234") and an in-memory session store.
func newAccountHandler(t *testing.T) (*Handler, *fakeStore) {
	t.Helper()
	hash, err := auth.HashPassword("secret-1234")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	s := &fakeStore{creator: db.Creator{ID: 9, Username: "alice", PasswordHash: hash}}
	h := newTestHandler(s)
	h.Auth = auth.NewMemoryStore()
	return h, s
}

// authReq builds a request for any method with the session cookie plus auth
// context (the authedReq/authedPut pattern, extended to DELETE) and returns
// the session token so tests can verify revocation.
func authReq(h *Handler, method, path, body string) (*http.Request, string) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	token, _ := h.Auth.Create(9)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	return middleware.WithCreator(req, 9), token
}

func decodeJSONBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("response bukan JSON valid: %q (%v)", rr.Body.String(), err)
	}
	return m
}

func TestHandleUpdateEmailSuccess(t *testing.T) {
	h, s := newAccountHandler(t)
	req, _ := authReq(h, http.MethodPut, "/api/account/email", `{"email":"baru@example.com","password":"secret-1234"}`)
	rr := httptest.NewRecorder()
	h.HandleUpdateEmail(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rr.Code, rr.Body.String())
	}
	if len(s.emailCalls) != 1 || s.emailCalls[0] != "baru@example.com" {
		t.Fatalf("emailCalls = %v, want [baru@example.com]", s.emailCalls)
	}
	body := decodeJSONBody(t, rr)
	if body["ok"] != true || body["email"] != "baru@example.com" {
		t.Fatalf("body = %v, want ok=true email=baru@example.com", body)
	}
	// The password hash must never leak into an account response.
	if strings.Contains(rr.Body.String(), "$2a$") {
		t.Fatal("password hash bocor ke response")
	}
}

func TestHandleUpdateEmailWrongPassword(t *testing.T) {
	h, s := newAccountHandler(t)
	req, _ := authReq(h, http.MethodPut, "/api/account/email", `{"email":"baru@example.com","password":"salah"}`)
	rr := httptest.NewRecorder()
	h.HandleUpdateEmail(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
	if len(s.emailCalls) != 0 {
		t.Fatalf("email tetap ditulis walau password salah: %v", s.emailCalls)
	}
}

func TestHandleUpdateEmailTaken(t *testing.T) {
	h, s := newAccountHandler(t)
	s.emailErr = db.ErrEmailTaken
	req, _ := authReq(h, http.MethodPut, "/api/account/email", `{"email":"punyaorang@example.com","password":"secret-1234"}`)
	rr := httptest.NewRecorder()
	h.HandleUpdateEmail(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rr.Code)
	}
}

func TestHandleUpdateEmailInvalidFormat(t *testing.T) {
	h, _ := newAccountHandler(t)
	req, _ := authReq(h, http.MethodPut, "/api/account/email", `{"email":"bukan-email","password":"secret-1234"}`)
	rr := httptest.NewRecorder()
	h.HandleUpdateEmail(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHandleUpdateEmailAnonymous(t *testing.T) {
	h, _ := newAccountHandler(t)
	req := putJSON("/api/account/email", `{"email":"x@example.com","password":"y"}`)
	rr := httptest.NewRecorder()
	h.HandleUpdateEmail(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestHandleUpdatePasswordSuccessRevokesOtherDevices(t *testing.T) {
	h, s := newAccountHandler(t)
	otherToken, _ := h.Auth.Create(9) // "another device"
	req, currentToken := authReq(h, http.MethodPut, "/api/account/password", `{"current_password":"secret-1234","new_password":"new-pass-9999"}`)
	rr := httptest.NewRecorder()
	h.HandleUpdatePassword(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rr.Code, rr.Body.String())
	}
	if len(s.passwordHashes) != 1 {
		t.Fatalf("passwordHashes = %v, want 1 hash", s.passwordHashes)
	}
	if s.passwordHashes[0] == "new-pass-9999" {
		t.Fatal("password disimpan plaintext: harus bcrypt")
	}
	if !auth.CheckPassword(s.passwordHashes[0], "new-pass-9999") {
		t.Fatal("hash tersimpan tidak cocok dengan password baru")
	}
	if _, ok := h.Auth.Get(otherToken); ok {
		t.Fatal("sesi device lain masih hidup setelah ganti password")
	}
	if _, ok := h.Auth.Get(currentToken); !ok {
		t.Fatal("sesi peminta ikut ter-revoke: harusnya disisihkan")
	}
}

func TestHandleUpdatePasswordWrongCurrent(t *testing.T) {
	h, s := newAccountHandler(t)
	req, _ := authReq(h, http.MethodPut, "/api/account/password", `{"current_password":"salah","new_password":"new-pass-9999"}`)
	rr := httptest.NewRecorder()
	h.HandleUpdatePassword(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
	if len(s.passwordHashes) != 0 {
		t.Fatal("password ditulis walau password lama salah")
	}
}

func TestHandleUpdatePasswordTooShort(t *testing.T) {
	h, _ := newAccountHandler(t)
	req, _ := authReq(h, http.MethodPut, "/api/account/password", `{"current_password":"secret-1234","new_password":"pendek"}`)
	rr := httptest.NewRecorder()
	h.HandleUpdatePassword(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHandleLogoutAll(t *testing.T) {
	h, _ := newAccountHandler(t)
	devA, _ := h.Auth.Create(9)
	devB, _ := h.Auth.Create(9)
	req, _ := authReq(h, http.MethodPost, "/api/account/logout-all", `{}`)
	rr := httptest.NewRecorder()
	h.HandleLogoutAll(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	for _, tok := range []string{devA, devB} {
		if _, ok := h.Auth.Get(tok); ok {
			t.Fatal("masih ada sesi setelah logout-all")
		}
	}
	sc := rr.Header().Get("Set-Cookie")
	if !strings.Contains(sc, auth.CookieName+"=;") && !strings.Contains(sc, "Max-Age=0") {
		t.Fatalf("Set-Cookie clear tidak ada: %q", sc)
	}
}

func TestHandleLogoutAllAnonymous(t *testing.T) {
	h, _ := newAccountHandler(t)
	req := postJSON("/api/account/logout-all", `{}`)
	rr := httptest.NewRecorder()
	h.HandleLogoutAll(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestHandleDeleteAccountSuccess(t *testing.T) {
	h, s := newAccountHandler(t)
	devA, _ := h.Auth.Create(9)
	req, _ := authReq(h, http.MethodDelete, "/api/account", `{"confirmation":"HAPUS","password":"secret-1234"}`)
	rr := httptest.NewRecorder()
	h.HandleDeleteAccount(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rr.Code, rr.Body.String())
	}
	if len(s.deletedAccounts) != 1 || s.deletedAccounts[0] != 9 {
		t.Fatalf("deletedAccounts = %v, want [9]", s.deletedAccounts)
	}
	if _, ok := h.Auth.Get(devA); ok {
		t.Fatal("sesi masih hidup setelah hapus akun")
	}
	sc := rr.Header().Get("Set-Cookie")
	if !strings.Contains(sc, auth.CookieName+"=;") && !strings.Contains(sc, "Max-Age=0") {
		t.Fatalf("cookie tidak di-clear: %q", sc)
	}
}

func TestHandleDeleteAccountNeedsHAPUS(t *testing.T) {
	h, s := newAccountHandler(t)
	req, _ := authReq(h, http.MethodDelete, "/api/account", `{"confirmation":"hapus","password":"secret-1234"}`)
	rr := httptest.NewRecorder()
	h.HandleDeleteAccount(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	if len(s.deletedAccounts) != 0 {
		t.Fatal("akun terhapus tanpa konfirmasi HAPUS")
	}
}

func TestHandleDeleteAccountWrongPassword(t *testing.T) {
	h, s := newAccountHandler(t)
	req, _ := authReq(h, http.MethodDelete, "/api/account", `{"confirmation":"HAPUS","password":"salah"}`)
	rr := httptest.NewRecorder()
	h.HandleDeleteAccount(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
	if len(s.deletedAccounts) != 0 {
		t.Fatal("akun terhapus walau password salah")
	}
}

// TestAccountEndpointsRateLimit: 5 requests/minute per IP: the 6th attempt
// gets 429 (same pattern as TestHandleLoginRateLimit, but through the
// middleware.RateLimit wrapper exactly as wired up in cmd/server/main.go).
func TestAccountEndpointsRateLimit(t *testing.T) {
	h, _ := newAccountHandler(t)
	h.AccountLimiter = ratelimit.NewLimiter()
	rl := middleware.RateLimit(h.AccountLimiter, func(r *http.Request) string {
		return "acct:" + ratelimit.ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For"))
	})
	endpoint := rl(middleware.OptionalAuth(h.Auth)(http.HandlerFunc(h.HandleLogoutAll)))

	for i := 1; i <= ratelimit.MaxAttempts; i++ {
		req, _ := authReq(h, http.MethodPost, "/api/account/logout-all", `{}`)
		req.RemoteAddr = "198.51.100.7:4000"
		rr := httptest.NewRecorder()
		endpoint.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("attempt %d: status = %d, want 200", i, rr.Code)
		}
	}
	req, _ := authReq(h, http.MethodPost, "/api/account/logout-all", `{}`)
	req.RemoteAddr = "198.51.100.7:4000"
	rr := httptest.NewRecorder()
	endpoint.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("attempt %d: status = %d, want 429", ratelimit.MaxAttempts+1, rr.Code)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Error("429 tanpa header Retry-After")
	}
}
