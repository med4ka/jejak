package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"jejak/internal/apierror"
	"jejak/internal/auth"
	"jejak/internal/middleware"
)

// optionalTime represents the 3 cases of the expires_at field on PUT
// /api/links/{code}:
//
//   - field not sent      → Set=false, T=nil  (do not touch the DB)
//   - "expires_at": null  → Set=true,  T=nil  (CLEAR: remove the expiry)
//   - "expires_at": "..." → Set=true,  T=&t   (set/change, RFC3339 → UTC)
//
// json.RawMessage is used (rather than UnmarshalJSON on a pointer) so the
// "null" case behaves deterministically: encoding/json has a null-vs-pointer
// edge case that is easy to misread; RawMessage distinguishes absent (len 0)
// vs "null" vs string explicitly.
type optionalTime struct {
	Set bool
	T   *time.Time
}

// HandleClaimLinks lets a just-logged-in creator claim anonymous links made
// in THIS browser (POST /api/links/claim, body {"short_codes": [...]}).
// Auth required; the store only flips rows that are still ownerless
// (see the rationale on ClaimLinks: owned links can never be stolen this way).
// Returns {"claimed": n}; unknown or already-owned codes count 0, not an error.
func (h *Handler) HandleClaimLinks(w http.ResponseWriter, r *http.Request) {
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		apierror.WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Login required")
		return
	}

	var req struct {
		ShortCodes []string `json:"short_codes"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		apierror.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
		return
	}
	if len(req.ShortCodes) > 50 {
		apierror.WriteError(w, http.StatusBadRequest, "LINK_CLAIM_LIMIT", "max 50 codes per claim")
		return
	}
	codes := make([]string, 0, len(req.ShortCodes))
	for _, c := range req.ShortCodes {
		c = strings.TrimSpace(c)
		if c == "" || len(c) > 30 {
			apierror.WriteError(w, http.StatusBadRequest, "LINK_INVALID_CODE", "Invalid short code in list")
			return
		}
		codes = append(codes, c)
	}

	claimed, err := h.Store.ClaimLinks(*creatorID, codes)
	if err != nil {
		h.Logger.Printf("ClaimLinks failed: %v", err)
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"claimed": claimed})
}

// normalizeTags cleans raw tag input: trim, lowercase, drop empties and
// duplicates (order preserved). Returns ok=false when any tag breaks the
// limits (max 5 tags, each 1-20 chars) so the caller can 400 clearly.
func normalizeTags(raw []string) ([]string, bool) {
	clean := []string{}
	seen := make(map[string]bool)
	for _, t := range raw {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || seen[t] {
			continue
		}
		if len(t) > 20 {
			return nil, false
		}
		seen[t] = true
		clean = append(clean, t)
	}
	if len(clean) > 5 {
		return nil, false
	}
	return clean, true
}

// HandleReorderLinks saves a new link order (PUT /api/links/reorder).
// Body: {"order": ["codeA", "codeB", ...]}: position = index in array.
// Auth required; the store scopes every row to the caller (another user's
// link -> 404, not 403, so the existence of someone else's short code is not
// disclosed).
func (h *Handler) HandleReorderLinks(w http.ResponseWriter, r *http.Request) {
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		apierror.WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Login required")
		return
	}

	var req struct {
		Order []string `json:"order"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		apierror.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
		return
	}
	if req.Order == nil {
		apierror.WriteError(w, http.StatusBadRequest, "LINK_ORDER_REQUIRED", "order is required")
		return
	}
	if len(req.Order) > 200 {
		apierror.WriteError(w, http.StatusBadRequest, "LINK_REORDER_LIMIT", "max 200 links per reorder")
		return
	}

	if err := h.Store.ReorderLinks(*creatorID, req.Order); err != nil {
		if err == sql.ErrNoRows {
			apierror.WriteError(w, http.StatusNotFound, "LINK_NOT_FOUND", "Unknown short code or not yours")
			return
		}
		h.Logger.Printf("ReorderLinks failed: %v", err)
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// HandleUpdateLink edits an owned link (PUT /api/links/{short_code}):
// device_rules (Smart Link), tags, is_featured (featured link), and is_active
// (disable/enable) in a single endpoint. Every field is optional (pointer):
// fields that are not sent are NOT touched, so a dashboard "feature" or
// "disable" toggle never wipes the existing device_rules/tags. A sent field =
// full-replace (similar to HandleUpdateMyProfile). Auth mandatory; scoped to
// the caller's own rows (WHERE creator_id): another user's link / empty code
// → 404 (not 403, to avoid disclosing the existence of someone else's short
// code). Rule URLs are validated as http(s) when set; empty = fallback to
// original_url. Featured is set through a radio transaction in the store (1
// per creator). is_active=off disables the public URL (the redirect answers
// 410, see HandleRedirect: Phase 13 lifecycle). The cache is invalidated
// after the update so the next redirect reads the latest
// device_rules/is_active.
//
// Link management (2026-09-30): added expires_at (optional, optionalTime:
// see that type). The frontend ONLY sends this field when the value CHANGES
// (an already-expired old link sends nothing → no 422); clear = null.
// Malformed format → 400, valid but <= 1 hour from now → 422 + field error
// (same as doShorten: the same UTC rule).
func (h *Handler) HandleUpdateLink(shortCode string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		apierror.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		apierror.WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Login required")
		return
	}

	var req struct {
		DeviceRules *map[string]string `json:"device_rules"`
		Tags        *[]string          `json:"tags"`
		IsFeatured  *bool              `json:"is_featured"`
		IsActive    *bool              `json:"is_active"`
		ExpiresAt   json.RawMessage    `json:"expires_at"`
		// Password (migration 17): nil = field not sent = keep the current
		// gate; "" = REMOVE the password (link becomes public); a value =
		// set/replace (validated 4-72 chars, bcrypt-hashed here). The
		// string-pointer distinguishes absent from clear - a plain string
		// could not. A password change also invalidates every old access
		// cookie (the cookie stores the digest OF the hash).
		Password *string `json:"password"`
		// FallbackURL (migration 17): nil = keep; "" = clear; a value =
		// where visitors go while the destination is broken (http(s), same
		// validRemoteURL rule as original_url: it must point at the
		// Internet).
		FallbackURL *string `json:"fallback_url"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		apierror.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
		return
	}
	if req.DeviceRules == nil && req.Tags == nil && req.IsFeatured == nil && req.IsActive == nil && len(req.ExpiresAt) == 0 && req.Password == nil && req.FallbackURL == nil {
		apierror.WriteError(w, http.StatusBadRequest, "LINK_NOTHING_TO_UPDATE", "Nothing to update")
		return
	}

	// password: absent = keep (no write), "" = clear, value = set. Same
	// 4-72 rule as doShorten: bcrypt truncates at 72 bytes silently, so a
	// longer password must be rejected instead of half-hashed.
	if req.Password != nil {
		pw := *req.Password
		hash := ""
		if pw != "" {
			if len(pw) < 4 || len(pw) > 72 {
				apierror.WriteFieldError(w, http.StatusBadRequest, "password", "LINK_PASSWORD_INVALID", "password harus 4-72 karakter")
				return
			}
			hashed, err := auth.HashPassword(pw)
			if err != nil {
				h.Logger.Printf("HashPassword failed: %v", err)
				apierror.WriteError(w, http.StatusInternalServerError, "LINK_PASSWORD_HASH_ERROR", "Failed to hash password")
				return
			}
			hash = hashed
		}
		if err := h.Store.SetLinkPassword(*creatorID, shortCode, hash); err != nil {
			if err == sql.ErrNoRows {
				apierror.WriteError(w, http.StatusNotFound, "LINK_NOT_FOUND", "Unknown short code or not yours")
				return
			}
			h.Logger.Printf("SetLinkPassword failed for %q: %v", shortCode, err)
			apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
			return
		}
	}

	// fallback_url: absent = keep, "" = clear, value = set (must be a real
	// http(s) destination - it is where visitors end up).
	if req.FallbackURL != nil {
		fb := *req.FallbackURL
		if fb != "" && !validRemoteURL(fb) {
			apierror.WriteFieldError(w, http.StatusBadRequest, "fallback_url", "LINK_FALLBACK_URL_INVALID", "fallback_url harus URL http(s) yang valid")
			return
		}
		if err := h.Store.SetLinkFallback(*creatorID, shortCode, fb); err != nil {
			if err == sql.ErrNoRows {
				apierror.WriteError(w, http.StatusNotFound, "LINK_NOT_FOUND", "Unknown short code or not yours")
				return
			}
			h.Logger.Printf("SetLinkFallback failed for %q: %v", shortCode, err)
			apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
			return
		}
	}

	// expires_at: 3 cases (see optionalTime). RFC3339 → UTC; must be > 1 hour
	// from now (422 otherwise): consistent with doShorten.
	expires := optionalTime{}
	if len(req.ExpiresAt) > 0 {
		if string(req.ExpiresAt) == "null" {
			expires.Set = true // clear
		} else {
			var s string
			if err := json.Unmarshal(req.ExpiresAt, &s); err != nil {
				apierror.WriteFieldError(w, http.StatusBadRequest, "expires_at", "LINK_EXPIRES_AT_INVALID", "expires_at must be RFC3339 UTC or null")
				return
			}
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				apierror.WriteFieldError(w, http.StatusBadRequest, "expires_at", "LINK_EXPIRES_AT_INVALID", "expires_at must be RFC3339 UTC (contoh: 2026-10-01T09:00:00Z)")
				return
			}
			u := t.UTC()
			if !u.After(time.Now().UTC().Add(time.Hour)) {
				apierror.WriteFieldError(w, http.StatusUnprocessableEntity, "expires_at", "LINK_EXPIRES_AT_TOO_SOON", "expires_at must be more than 1 hour in the future")
				return
			}
			expires.Set = true
			expires.T = &u
		}
	}

	if req.DeviceRules != nil {
		for key, ruleURL := range *req.DeviceRules {
			if !validHTTPURL(ruleURL) {
				apierror.WriteError(w, http.StatusBadRequest, "LINK_INVALID_DEVICE_URL", "Invalid URL for "+key)
				return
			}
		}
	}

	tags := []string{}
	if req.Tags != nil {
		var ok bool
		tags, ok = normalizeTags(*req.Tags)
		if !ok {
			apierror.WriteError(w, http.StatusBadRequest, "LINK_INVALID_TAGS", "Invalid tags (max 5 tags, 1-20 chars each)")
			return
		}
	}
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		apierror.WriteError(w, http.StatusBadRequest, "LINK_INVALID_TAGS", "Invalid tags")
		return
	}

	rules := map[string]string{}
	if req.DeviceRules != nil {
		rules = *req.DeviceRules
	}
	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		apierror.WriteError(w, http.StatusBadRequest, "LINK_INVALID_DEVICE_RULES", "Invalid device_rules")
		return
	}

	// UpdateLink runs only when smart-link/tags fields were sent: a
	// featured-only toggle must not overwrite existing rules/tags data.
	if req.DeviceRules != nil || req.Tags != nil {
		if err := h.Store.UpdateLink(*creatorID, shortCode, string(rulesJSON), string(tagsJSON)); err != nil {
			if err == sql.ErrNoRows {
				apierror.WriteError(w, http.StatusNotFound, "LINK_NOT_FOUND", "Unknown short code or not yours")
				return
			}
			h.Logger.Printf("UpdateLink failed for %q: %v", shortCode, err)
			apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
			return
		}
	}

	if req.IsFeatured != nil {
		if err := h.Store.SetFeaturedLink(*creatorID, shortCode, *req.IsFeatured); err != nil {
			if err == sql.ErrNoRows {
				apierror.WriteError(w, http.StatusNotFound, "LINK_NOT_FOUND", "Unknown short code or not yours")
				return
			}
			h.Logger.Printf("SetFeaturedLink failed for %q: %v", shortCode, err)
			apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
			return
		}
	}

	// The is_active toggle (disable/enable) is not part of UpdateLink: it is a
	// separate column and must not overwrite rules/tags. The public redirect
	// endpoint answers 410 for a disabled link (see HandleRedirect); the
	// dashboard still lists it so it can be re-enabled.
	if req.IsActive != nil {
		if err := h.Store.SetLinkActive(*creatorID, shortCode, *req.IsActive); err != nil {
			if err == sql.ErrNoRows {
				apierror.WriteError(w, http.StatusNotFound, "LINK_NOT_FOUND", "Unknown short code or not yours")
				return
			}
			h.Logger.Printf("SetLinkActive failed for %q: %v", shortCode, err)
			apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
			return
		}
	}

	// expires_at: Set → write the value OR clear it (T=nil when null). Scoped
	// in the store (WHERE creator_id) → another user's link = 404, consistent
	// with the other update paths.
	if expires.Set {
		if err := h.Store.SetLinkExpiry(*creatorID, shortCode, expires.T); err != nil {
			if err == sql.ErrNoRows {
				apierror.WriteError(w, http.StatusNotFound, "LINK_NOT_FOUND", "Unknown short code or not yours")
				return
			}
			h.Logger.Printf("SetLinkExpiry failed for %q: %v", shortCode, err)
			apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
			return
		}
	}

	// The stale cache no longer matches the DB -> discard it so the next
	// redirect reads the latest device_rules (cache-aside invalidate-on-write).
	if h.Cache != nil {
		if err := h.Cache.Delete(shortCode); err != nil {
			h.Logger.Printf("Warning: failed to invalidate cache for %q: %v", shortCode, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}
