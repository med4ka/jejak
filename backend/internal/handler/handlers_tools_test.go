package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestNormalizePhone pins the four documented input shapes (local, plus,
// international, bare national) plus every rejection rule: length window
// 10-15 digits, digit-only, non-empty.
func TestNormalizePhone(t *testing.T) {
	okCases := []struct{ in, want string }{
		{"08123456789", "628123456789"},   // local trunk prefix
		{"+628123456789", "628123456789"}, // e.164 with plus
		{"628123456789", "628123456789"},  // already international
		{"8123456789", "628123456789"},    // bare national
		{"0812-3456-789", "628123456789"}, // pasted dashes
		{"0812 3456 789", "628123456789"}, // pasted spaces
		{"(0812) 3456 789", "628123456789"},
		{"00628123456789", "628123456789"}, // 00 international prefix
	}
	for _, tc := range okCases {
		got, err := NormalizePhone(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("NormalizePhone(%q) = (%q, %v), want (%q, nil)", tc.in, got, err, tc.want)
		}
	}

	badCases := []struct{ in, wantMsg string }{
		{"", "required"},
		{"   ", "required"},
		{"123", "10-15"},                // too short after normalize
		{"0812345", "10-15"},            // 62812345 = 8 digits
		{"628123456789012345", "10-15"}, // too long
		{"0812345678a", "digits only"},  // letter
		{"notaphone", "digits only"},    // word
	}
	for _, tc := range badCases {
		_, err := NormalizePhone(tc.in)
		if err == nil {
			t.Errorf("NormalizePhone(%q) accepted, want error containing %q", tc.in, tc.wantMsg)
			continue
		}
		if !strings.Contains(err.Error(), tc.wantMsg) {
			t.Errorf("NormalizePhone(%q) error = %q, want containing %q", tc.in, err, tc.wantMsg)
		}
	}
}

// TestHandleWhatsAppLinkCreated checks the happy path: digits normalized,
// message percent-encoded (spaces as %20, never "+"), and shorten defaulting
// to ON with the created short link recorded in the store.
func TestHandleWhatsAppLinkCreated(t *testing.T) {
	h := newTestHandler(&fakeStore{})
	body := `{"phone":"0812 3456 789","message":"Halo, saya mau tanya produk"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tools/whatsapp-link", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.HandleWhatsAppLink(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rr.Code, rr.Body.String())
	}
	m := decodeJSONBody(t, rr)
	wantWA := "https://wa.me/628123456789?text=Halo%2C%20saya%20mau%20tanya%20produk"
	if m["wa_link"] != wantWA {
		t.Errorf("wa_link = %q, want %q", m["wa_link"], wantWA)
	}
	short, _ := m["short_url"].(string)
	if !strings.Contains(short, "/r/") {
		t.Errorf("short_url = %q, want containing /r/", short)
	}
	st := h.Store.(*fakeStore)
	if len(st.created) != 1 || st.created[0].originalURL != wantWA {
		t.Fatalf("created = %+v, want one row with the wa.me URL", st.created)
	}
}

// TestHandleWhatsAppLinkNoMessage: a chat link without a prefilled message
// must be a bare wa.me/<digits> (no dangling ?text=).
func TestHandleWhatsAppLinkNoMessage(t *testing.T) {
	h := newTestHandler(&fakeStore{})
	req := httptest.NewRequest(http.MethodPost, "/api/tools/whatsapp-link",
		strings.NewReader(`{"phone":"+628123456789","message":"  ","shorten":false}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.HandleWhatsAppLink(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rr.Code, rr.Body.String())
	}
	m := decodeJSONBody(t, rr)
	if m["wa_link"] != "https://wa.me/628123456789" {
		t.Errorf("wa_link = %q, want bare wa.me link", m["wa_link"])
	}
	if _, exists := m["short_url"]; exists {
		t.Errorf("short_url present despite shorten=false: %v", m)
	}
	if len(h.Store.(*fakeStore).created) != 0 {
		t.Errorf("a link was created despite shorten=false")
	}
}

// TestHandleWhatsAppLinkValidation: invalid phone and oversized message are
// 400 field errors, and NOTHING is stored.
func TestHandleWhatsAppLinkValidation(t *testing.T) {
	cases := []struct{ name, body, field string }{
		{"short phone", `{"phone":"123","message":""}`, "phone"},
		{"letter phone", `{"phone":"0812abc","message":""}`, "phone"},
		{"message over 500", `{"phone":"08123456789","message":"` + strings.Repeat("x", 501) + `"}`, "message"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler(&fakeStore{})
			req := httptest.NewRequest(http.MethodPost, "/api/tools/whatsapp-link", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			h.HandleWhatsAppLink(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body: %s)", rr.Code, rr.Body.String())
			}
			m := decodeJSONBody(t, rr)
			if m["field"] != tc.field {
				t.Errorf("field = %v, want %q", m["field"], tc.field)
			}
			if len(h.Store.(*fakeStore).created) != 0 {
				t.Errorf("a link was created despite the 400")
			}
		})
	}
}

// TestHandleDeepLinkGenerate: supported URL answers 200 with platform and
// scheme; a non-e-commerce URL is 404 (never a 500 or a fake success).
func TestHandleDeepLinkGenerate(t *testing.T) {
	h := newTestHandler(&fakeStore{})

	t.Run("shopee product", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/deeplink/generate",
			strings.NewReader(`{"url":"https://shopee.co.id/product/111/222"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.HandleDeepLinkGenerate(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body: %s)", rr.Code, rr.Body.String())
		}
		m := decodeJSONBody(t, rr)
		if m["platform"] != "shopee" || m["deep_link"] != "shopeeid://product/111/222" {
			t.Errorf("resp = %v", m)
		}
		if m["web_fallback"] != "https://shopee.co.id/product/111/222" {
			t.Errorf("web_fallback = %v", m["web_fallback"])
		}
	})

	t.Run("ordinary URL is 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/deeplink/generate",
			strings.NewReader(`{"url":"https://example.com/product/1/2"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.HandleDeepLinkGenerate(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404 (body: %s)", rr.Code, rr.Body.String())
		}
	})

	t.Run("invalid URL is 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/deeplink/generate",
			strings.NewReader(`{"url":"javascript:alert(1)"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.HandleDeepLinkGenerate(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (body: %s)", rr.Code, rr.Body.String())
		}
	})
}
