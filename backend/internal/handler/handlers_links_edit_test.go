package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jejak/internal/auth"
)

// TestHandleClaimLinksAnonymouslyBlocked: claim requires login: anonymous
// request must be rejected before touching the store.
func TestHandleClaimLinksAnonymouslyBlocked(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)
	h.Auth = auth.NewMemoryStore()

	req := postJSON("/api/links/claim", `{"short_codes":["abc123"]}`)
	rr := httptest.NewRecorder()
	h.HandleClaimLinks(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous claim status = %d, want 401", rr.Code)
	}
	if len(s.claimCodes) != 0 {
		t.Fatalf("store touched for anonymous claim: %v", s.claimCodes)
	}
}

// TestHandleClaimLinksUsedCodesCallsStore checks the endpoint forwards the
// advertised codes to the store and echoes the claimed count.
func TestHandleClaimLinksUsedCodesCallsStore(t *testing.T) {
	s := &fakeStore{claimResult: 2}
	h := newTestHandler(s)
	h.Auth = auth.NewMemoryStore()

	req := authedReq(s, h, 42, "/api/links/claim", `{"short_codes":["abc123","def456"]}`)
	rr := httptest.NewRecorder()
	h.HandleClaimLinks(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d (body %q), want 200", rr.Code, rr.Body.String())
	}
	if len(s.claimCodes) != 2 || s.claimCodes[0] != "abc123" || s.claimCodes[1] != "def456" {
		t.Fatalf("store received %v, want [abc123 def456]", s.claimCodes)
	}
	if body := strings.TrimSpace(rr.Body.String()); body != `{"claimed":2}` {
		t.Fatalf("body = %q, want {\"claimed\":2}", body)
	}
}

// TestHandleClaimLinksBadBody: malformed / over-limit bodies must 400 without
// touching the store.
func TestHandleClaimLinksBadBody(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"not-json", `{not json`},
		{"too-many", `{"short_codes":["a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a","a"]}`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s := &fakeStore{}
			h := newTestHandler(s)
			h.Auth = auth.NewMemoryStore()

			req := authedReq(s, h, 1, "/api/links/claim", tt.body)
			rr := httptest.NewRecorder()
			h.HandleClaimLinks(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d (body %q), want 400", rr.Code, rr.Body.String())
			}
			if len(s.claimCodes) != 0 {
				t.Fatalf("store touched on bad body: %v", s.claimCodes)
			}
		})
	}
}

func TestHandleUpdateLink(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		anon     bool
		notMine  bool
		wantCode int
	}{
		{"success-ios-android-tags", `{"device_rules":{"ios":"https://ios.example.com","android":"https://android.example.com"},"tags":["smart"]}`, false, false, http.StatusOK},
		{"success-empty-rules", `{"device_rules":{},"tags":[]}`, false, false, http.StatusOK},
		{"anonymous-401", `{"device_rules":{"ios":"https://x.com"}}`, true, false, http.StatusUnauthorized},
		{"not-owner-404", `{"device_rules":{"ios":"https://x.com"}}`, false, true, http.StatusNotFound},
		{"bad-ios-url-400", `{"device_rules":{"ios":"not-a-url"}}`, false, false, http.StatusBadRequest},
		{"malformed-json-400", `{nope`, false, false, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &fakeStore{created: []createCall{{code: "mine01"}}}
			h := newTestHandler(s)
			h.Auth = auth.NewMemoryStore()

			var req *http.Request
			if tt.anon {
				req = putJSON("/api/links/mine01", tt.body)
			} else {
				req = authedPut(s, h, 42, "/api/links/mine01", tt.body)
			}
			code := "mine01"
			if tt.notMine {
				code = "theirs9"
			}
			rr := httptest.NewRecorder()
			h.HandleUpdateLink(code, rr, req)
			if rr.Code != tt.wantCode {
				t.Fatalf("status = %d (body %q), want %d", rr.Code, rr.Body.String(), tt.wantCode)
			}
		})
	}
}

// TestHandleUpdateLinkSuccessPersists asserts the store actually received the
// full-replace device_rules + tags JSON for the owned code.
func TestHandleUpdateLinkSuccessPersists(t *testing.T) {
	s := &fakeStore{created: []createCall{{code: "mine01"}}}
	h := newTestHandler(s)
	h.Auth = auth.NewMemoryStore()

	req := authedPut(s, h, 42, "/api/links/mine01",
		`{"device_rules":{"ios":"https://ios.example.com/"},"tags":["deeplink","mobile"]}`)
	rr := httptest.NewRecorder()
	h.HandleUpdateLink("mine01", rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d (body %q), want 200", rr.Code, rr.Body.String())
	}
	if len(s.updateCalls) != 1 {
		t.Fatalf("UpdateLink called %d times, want 1", len(s.updateCalls))
	}
	call := s.updateCalls[0]
	if call.code != "mine01" {
		t.Errorf("updated code = %q, want mine01", call.code)
	}
	if call.deviceRulesJSON != `{"ios":"https://ios.example.com/"}` {
		t.Errorf("device_rules = %q, want full-replace device_rules", call.deviceRulesJSON)
	}
	if call.tagsJSON != `["deeplink","mobile"]` {
		t.Errorf("tags = %q, want full-replace tags", call.tagsJSON)
	}
}

func TestHandleUpdateLinkFeatured(t *testing.T) {
	t.Run("feature-true-only-does-not-touch-rules", func(t *testing.T) {
		s := &fakeStore{created: []createCall{{code: "mine01"}}}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()

		req := authedPut(s, h, 42, "/api/links/mine01", `{"is_featured":true}`)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("mine01", rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d (body %q), want 200", rr.Code, rr.Body.String())
		}
		if len(s.featureCalls) != 1 || !s.featureCalls[0].featured || s.featureCalls[0].code != "mine01" {
			t.Errorf("SetFeaturedLink calls = %+v, want one featured=true for mine01", s.featureCalls)
		}
		if len(s.updateCalls) != 0 {
			t.Errorf("UpdateLink called %d times, want 0 (toggle tidak boleh menimpa rules/tags)", len(s.updateCalls))
		}
	})

	t.Run("feature-false-unfeatures", func(t *testing.T) {
		s := &fakeStore{created: []createCall{{code: "mine01"}}}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()

		req := authedPut(s, h, 42, "/api/links/mine01", `{"is_featured":false}`)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("mine01", rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d (body %q), want 200", rr.Code, rr.Body.String())
		}
		if len(s.featureCalls) != 1 || s.featureCalls[0].featured {
			t.Errorf("SetFeaturedLink calls = %+v, want one featured=false", s.featureCalls)
		}
	})

	t.Run("full-update-and-feature-combined", func(t *testing.T) {
		s := &fakeStore{created: []createCall{{code: "mine01"}}}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()

		req := authedPut(s, h, 42, "/api/links/mine01",
			`{"device_rules":{"ios":"https://ios.example.com/"},"tags":["smart"],"is_featured":true}`)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("mine01", rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d (body %q), want 200", rr.Code, rr.Body.String())
		}
		if len(s.updateCalls) != 1 {
			t.Errorf("UpdateLink called %d times, want 1", len(s.updateCalls))
		}
		if len(s.featureCalls) != 1 || !s.featureCalls[0].featured {
			t.Errorf("SetFeaturedLink calls = %+v, want one featured=true", s.featureCalls)
		}
	})

	t.Run("nothing-to-update-400", func(t *testing.T) {
		s := &fakeStore{created: []createCall{{code: "mine01"}}}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()

		req := authedPut(s, h, 42, "/api/links/mine01", `{}`)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("mine01", rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d (body %q), want 400", rr.Code, rr.Body.String())
		}
		if len(s.featureCalls) != 0 && len(s.updateCalls) != 0 {
			t.Error("empty body should not touch the store")
		}
	})

	t.Run("feature-not-owner-404", func(t *testing.T) {
		s := &fakeStore{created: []createCall{{code: "mine01"}}}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()

		req := authedPut(s, h, 42, "/api/links/theirs9", `{"is_featured":true}`)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("theirs9", rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d (body %q), want 404", rr.Code, rr.Body.String())
		}
	})
}

// TestHandleUpdateLinkIsActive covers the Phase 13 disable/enable toggle:
// {is_active} alone must reach SetLinkActive, must NOT touch UpdateLink
// (rules/tags untouched), and must be 404 for links the caller doesn't own.
func TestHandleUpdateLinkIsActive(t *testing.T) {
	t.Run("disable-false-calls-set-active", func(t *testing.T) {
		s := &fakeStore{created: []createCall{{code: "mine01"}}}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()

		req := authedPut(s, h, 42, "/api/links/mine01", `{"is_active":false}`)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("mine01", rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d (body %q), want 200", rr.Code, rr.Body.String())
		}
		if len(s.activeCalls) != 1 || s.activeCalls[0].code != "mine01" || s.activeCalls[0].active {
			t.Errorf("SetLinkActive calls = %+v, want one active=false for mine01", s.activeCalls)
		}
		if len(s.updateCalls) != 0 {
			t.Errorf("UpdateLink called %d times, want 0 (toggle tidak boleh menimpa rules/tags)", len(s.updateCalls))
		}
	})

	t.Run("enable-true-reactivates", func(t *testing.T) {
		s := &fakeStore{created: []createCall{{code: "mine01"}}}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()

		req := authedPut(s, h, 42, "/api/links/mine01", `{"is_active":true}`)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("mine01", rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d (body %q), want 200", rr.Code, rr.Body.String())
		}
		if len(s.activeCalls) != 1 || !s.activeCalls[0].active {
			t.Errorf("SetLinkActive calls = %+v, want one active=true", s.activeCalls)
		}
	})

	t.Run("not-owner-404", func(t *testing.T) {
		s := &fakeStore{created: []createCall{{code: "mine01"}}}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()

		req := authedPut(s, h, 42, "/api/links/theirs9", `{"is_active":false}`)
		rr := httptest.NewRecorder()
		h.HandleUpdateLink("theirs9", rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d (body %q), want 404", rr.Code, rr.Body.String())
		}
		if len(s.activeCalls) != 0 {
			t.Errorf("SetLinkActive called for unowned code: %v", s.activeCalls)
		}
	})
}
