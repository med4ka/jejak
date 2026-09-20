package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"jejak/internal/middleware"
)

// HandleClaimLinks lets a just-logged-in creator claim anonymous links made
// in THIS browser (POST /api/links/claim, body {"short_codes": [...]}).
// Auth required; store only flips rows that are still ownerless
// (see LEARN on ClaimLinks — owned links can never be stolen this way).
// Returns {"claimed": n}; unknown/already-owned codes count 0, not an error.
func (h *Handler) HandleClaimLinks(w http.ResponseWriter, r *http.Request) {
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		http.Error(w, "Login required", http.StatusUnauthorized)
		return
	}

	var req struct {
		ShortCodes []string `json:"short_codes"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.ShortCodes) > 50 {
		http.Error(w, "max 50 codes per claim", http.StatusBadRequest)
		return
	}
	codes := make([]string, 0, len(req.ShortCodes))
	for _, c := range req.ShortCodes {
		c = strings.TrimSpace(c)
		if c == "" || len(c) > 30 {
			http.Error(w, "Invalid short code in list", http.StatusBadRequest)
			return
		}
		codes = append(codes, c)
	}

	claimed, err := h.Store.ClaimLinks(*creatorID, codes)
	if err != nil {
		h.Logger.Printf("ClaimLinks failed: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
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
// Body: {"order": ["codeA", "codeB", ...]} — position = index in array.
// Auth required; store scopes every row to the caller (link orang lain -> 404,
// bukan 403, supaya tidak membocorkan keberadaan short-code orang).
func (h *Handler) HandleReorderLinks(w http.ResponseWriter, r *http.Request) {
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		http.Error(w, "Login required", http.StatusUnauthorized)
		return
	}

	var req struct {
		Order []string `json:"order"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Order == nil {
		http.Error(w, "order is required", http.StatusBadRequest)
		return
	}
	if len(req.Order) > 200 {
		http.Error(w, "max 200 links per reorder", http.StatusBadRequest)
		return
	}

	if err := h.Store.ReorderLinks(*creatorID, req.Order); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Unknown short code or not yours", http.StatusNotFound)
			return
		}
		h.Logger.Printf("ReorderLinks failed: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// HandleUpdateLink edits an owned link (PUT /api/links/{short_code}):
// device_rules (Smart Link), tags, is_featured (link unggulan), dan is_active
// (disable/enable) dalam 1 endpoint. Setiap field opsional (pointer) — field
// yang tidak dikirim TIDAK disentuh, supaya toggle "unggulan" / "nonaktifkan"
// dari dashboard tidak menghapus device_rules/tags yang ada. Field yang
// dikirim = full-replace (mirip dengan HandleUpdateMyProfile). Auth wajib;
// scope ke baris milik sendiri (WHERE creator_id) — link orang lain / kode
// kosong → 404 (bukan 403, jangan bocorkan keberadaan short-code orang). URL
// rules divalidasi http(s) kalau diisi; kalau kosong = fallback ke
// original_url. Featured di-set lewat transaksi radio di store (1 per
// creator). is_active=off menonaktifkan URL publik (redirect menjawab 410,
// lihat HandleRedirect — Fase 13 lifecycle). Cache di-invalidate sesudah
// update supaya redirect berikutnya baca device_rules/is_active terbaru.
func (h *Handler) HandleUpdateLink(shortCode string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		http.Error(w, "Login required", http.StatusUnauthorized)
		return
	}

	var req struct {
		DeviceRules *map[string]string `json:"device_rules"`
		Tags        *[]string          `json:"tags"`
		IsFeatured  *bool              `json:"is_featured"`
		IsActive    *bool              `json:"is_active"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.DeviceRules == nil && req.Tags == nil && req.IsFeatured == nil && req.IsActive == nil {
		http.Error(w, "Nothing to update", http.StatusBadRequest)
		return
	}

	if req.DeviceRules != nil {
		for key, ruleURL := range *req.DeviceRules {
			if !validHTTPURL(ruleURL) {
				http.Error(w, "Invalid URL for "+key, http.StatusBadRequest)
				return
			}
		}
	}

	tags := []string{}
	if req.Tags != nil {
		var ok bool
		tags, ok = normalizeTags(*req.Tags)
		if !ok {
			http.Error(w, "Invalid tags (max 5 tags, 1-20 chars each)", http.StatusBadRequest)
			return
		}
	}
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		http.Error(w, "Invalid tags", http.StatusBadRequest)
		return
	}

	rules := map[string]string{}
	if req.DeviceRules != nil {
		rules = *req.DeviceRules
	}
	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		http.Error(w, "Invalid device_rules", http.StatusBadRequest)
		return
	}

	// UpdateLink dipanggil hanya kalau ada field smart link/tags yang dikirim —
	// toggle featured saja tidak boleh menimpa data rules/tags existing.
	if req.DeviceRules != nil || req.Tags != nil {
		if err := h.Store.UpdateLink(*creatorID, shortCode, string(rulesJSON), string(tagsJSON)); err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Unknown short code or not yours", http.StatusNotFound)
				return
			}
			h.Logger.Printf("UpdateLink failed for %q: %v", shortCode, err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	if req.IsFeatured != nil {
		if err := h.Store.SetFeaturedLink(*creatorID, shortCode, *req.IsFeatured); err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Unknown short code or not yours", http.StatusNotFound)
				return
			}
			h.Logger.Printf("SetFeaturedLink failed for %q: %v", shortCode, err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	// Toggle is_active (disable/enable) bukan bagian dari UpdateLink: kolom
	// terpisah dan tidak boleh menimpa rules/tags. Endpoint publik redirect
	// akan menjawab 410 untuk link yang di-disable (lihat HandleRedirect);
	// dashboard tetap menampilkannya supaya bisa di hidupkan kembali.
	if req.IsActive != nil {
		if err := h.Store.SetLinkActive(*creatorID, shortCode, *req.IsActive); err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Unknown short code or not yours", http.StatusNotFound)
				return
			}
			h.Logger.Printf("SetLinkActive failed for %q: %v", shortCode, err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	// Stale cache tidak lagi cocok dengan DB -> buang supaya redirect berikutnya
	// membaca device_rules terbaru (cache-aside invalidate-on-write).
	if h.Cache != nil {
		if err := h.Cache.Delete(shortCode); err != nil {
			h.Logger.Printf("Warning: failed to invalidate cache for %q: %v", shortCode, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}
