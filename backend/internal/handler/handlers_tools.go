package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"jejak/internal/deeplink"
	"jejak/internal/middleware"
	"jejak/internal/ratelimit"
	"jejak/internal/shortener"
)

// HandleDeepLinkGenerate handles POST /api/deeplink/generate: a pure,
// side-effect-free lookup that turns an e-commerce URL into its app deep
// link. No session, no rate limit, no DB access: the work is one URL parse,
// so the only "cost" an abuser can rack up is regex matching they could do
// client-side anyway. Non-e-commerce URLs answer 404 (the caller asked a
// question this endpoint cannot answer, not a server failure).
func (h *Handler) HandleDeepLinkGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		URL string `json:"url"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	// Same gate as doShorten: javascript:, data:, "//evil" and empty input
	// must never leave this handler as a "deep link" candidate.
	if !validRemoteURL(req.URL) {
		http.Error(w, "url must be a valid http(s) URL", http.StatusBadRequest)
		return
	}
	platform, scheme := deeplink.DetectEcommerce(req.URL)
	if platform == "" {
		http.Error(w, "URL is not a supported e-commerce link", http.StatusNotFound)
		return
	}
	note := "Opens in app if installed, otherwise web"
	if scheme == "" {
		// TikTok and friends: the web page itself hands off to the app.
		note = "Opens on the web; the app takes over via universal link"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"original_url": req.URL,
		"platform":     platform,
		"deep_link":    scheme,
		"web_fallback": req.URL,
		"note":         note,
	})
}

// HandleWhatsAppLink handles POST /api/tools/whatsapp-link: build a
// wa.me click-to-chat link from a phone number plus a prefilled message
// (the WhatsApp click-to-chat contract: digits only, no "+", message as the
// percent-encoded "text" query parameter). shorten=true (the default) also
// creates a Jejak short link so the click lands in the owner's analytics:
// device, referrer, and counts all flow through the normal /r/{code}
// redirect like any other link.
func (h *Handler) HandleWhatsAppLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Phone   string `json:"phone"`
		Message string `json:"message"`
		// Shorten nil = true: omitted bodies follow the documented default
		// (the UI toggle ships ON), while an explicit false skips creation.
		Shorten *bool `json:"shorten"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	digits, err := NormalizePhone(req.Phone)
	if err != nil {
		writeFieldError(w, http.StatusBadRequest, "phone", err.Error())
		return
	}
	// 500 matches the UI counter; the message reaches a public URL, so the
	// server enforces the cap instead of trusting the textarea.
	if len([]rune(req.Message)) > 500 {
		writeFieldError(w, http.StatusBadRequest, "message", "message must be at most 500 characters")
		return
	}
	waLink := buildWhatsAppLink(digits, req.Message)

	resp := map[string]string{"wa_link": waLink}
	shorten := req.Shorten == nil || *req.Shorten
	if shorten {
		// Same bucket as POST /api/shorten (same effect: a new row in urls).
		// Sharing the key means a spammer cannot double their budget by
		// alternating between the two endpoints.
		rlKey := "shorten\x00" + ratelimit.ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For"))
		if h.ShortenLimiter != nil {
			if !h.ShortenLimiter.Allow(rlKey) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "Too many requests, try again later", http.StatusTooManyRequests)
				return
			}
			h.ShortenLimiter.Record(rlKey)
		}
		code, err := h.createShortLink(middleware.CreatorID(r), waLink)
		if err != nil {
			h.Logger.Printf("WhatsApp link createShortLink failed: %v", err)
			http.Error(w, "Failed to store URL", http.StatusInternalServerError)
			return
		}
		resp["short_url"] = absoluteShortURL(r, code)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// NormalizePhone converts local/international phone input to the bare
// international digit string wa.me expects (e.g. "628123456789").
// Accepted shapes: "0812…" (Indonesian local), "+62812…", "62812…",
// "812…" (bare national), any spacing/dashes/parens the user pastes.
// Anything that does not resolve to 10-15 digits is rejected: too short
// cannot be a real subscriber number, too long cannot be one either, and
// non-digits (letters) are never part of a phone number.
func NormalizePhone(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "+")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "(", "")
	s = strings.ReplaceAll(s, ")", "")
	s = strings.TrimPrefix(s, "00") // international prefix "00 62…" style
	if s == "" {
		return "", errors.New("phone is required")
	}
	switch {
	case strings.HasPrefix(s, "62"):
		// Already international Indonesian.
	case strings.HasPrefix(s, "0"):
		// Local format "0812…" -> "62812…" (drop the trunk prefix).
		s = "62" + s[1:]
	default:
		// Bare national "812…" or another country: assume Indonesia, the
		// target audience of this tool (documented behavior).
		s = "62" + s
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return "", errors.New("phone must contain digits only")
		}
	}
	if len(s) < 10 || len(s) > 15 {
		return "", errors.New("phone must be 10-15 digits (contoh: 081234567890)")
	}
	return s, nil
}

// buildWhatsAppLink assembles the wa.me URL. QueryEscape would emit "+" for
// spaces, which wa.me treats as a literal plus in some clients: "%20" is
// unambiguous in every decoder, so spaces are re-escaped after encoding.
func buildWhatsAppLink(digits, message string) string {
	link := "https://wa.me/" + digits
	if msg := strings.TrimSpace(message); msg != "" {
		link += "?text=" + strings.ReplaceAll(url.QueryEscape(msg), "+", "%20")
	}
	return link
}

// createShortLink stores a random-code short link pointing at targetURL and
// warms the redirect cache, reusing the exact create semantics of doShorten
// (random 6-char code, empty tag list, no expiry). It is intentionally NOT a
// refactor of doShorten: the shortener's request/response contract stays
// untouched; this helper only covers the "internal" creation needed by the
// WhatsApp tool.
func (h *Handler) createShortLink(creatorID *int64, targetURL string) (string, error) {
	code, err := shortener.GenerateShortCode(6)
	if err != nil {
		return "", err
	}
	if err := h.Store.CreateURL(code, targetURL, creatorID, "[]", nil, ""); err != nil {
		return "", err
	}
	if h.Cache != nil {
		if b, err := json.Marshal(redirectTarget{URL: targetURL}); err == nil {
			if err := h.Cache.Set(code, string(b), redirectTTL(nil)); err != nil {
				h.Logger.Printf("Warning: failed to populate cache: %v", err)
			}
		}
	}
	return code, nil
}

// absoluteShortURL mirrors the scheme+Host building block of doShorten
// (TLS, exact X-Forwarded-Proto https upgrade, request Host) for endpoints
// that return JSON instead of the raw URL body.
func absoluteShortURL(r *http.Request, code string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/r/" + code
}
