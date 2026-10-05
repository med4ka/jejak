package handler

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/skip2/go-qrcode"

	"jejak/internal/apierror"
	"jejak/internal/db"
	"jejak/internal/middleware"
	"jejak/internal/shortener"
)

// Bulk endpoint limits (link management, 2026-09-30):
//
//	10-100 URLs per import request (outside that → 400, never silently
//	truncated: the client learns the spec), body max 100KB (far above 100
//	plain-text URLs at ~10-20KB; this cap is tighter than the general
//	maxJSONBodyBytes of 1MB), and bulk QR max 200 ZIP entries (beyond that use
//	the ?tag= filter: one request could otherwise produce a ZIP of tens of
//	MB, which must not be executed silently).
const (
	bulkMinURLs      = 10
	bulkMaxURLs      = 100
	maxBulkBodyBytes = 100 << 10 // 100 KB
	bulkQRMaxLinks   = 200
)

// bulkError is one failed row in a bulk response: line is the 1-based index
// into the input array (not a byte offset) so the UI can mark exactly the
// offending text row.
type bulkError struct {
	Line  int    `json:"line"`
	URL   string `json:"url"`
	Error string `json:"error"`
}

// HandleBulkShorten is POST /api/links/bulk: a mass import of 10-100 URLs at
// once (hundreds of one-by-one "Shorten" clicks do not scale). Unlike
// doShorten: a per-row JSON response (created + errors) rather than one
// plain-text short URL; one row's failure does NOT cancel the others: rows
// are validated individually and then inserted through CreateURLsBatch (ON
// CONFLICT DO NOTHING, see db.go). Failure reasons: invalid_url |
// duplicate_in_request | duplicate (the code collides with a row already in
// the DB).
//
// Auth + rate limit are wired in main.go: RequireAuth sits OUTSIDE RateLimit
// (a 401 must not consume import quota): the checks are repeated here
// defensively (the pattern used by all handlers).
func (h *Handler) HandleBulkShorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierror.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		apierror.WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Login required")
		return
	}

	// Manual decode with a 100KB cap (not decodeJSON: the bulk cap is tighter
	// than the 1MB default). An oversized body = 400, consistent with the
	// decodeJSON rationale (a clear error, not 413/500).
	r.Body = http.MaxBytesReader(w, r.Body, maxBulkBodyBytes)
	var req struct {
		URLs []string `json:"urls"`
		Tag  string   `json:"tag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
		return
	}

	total := len(req.URLs)
	if total < bulkMinURLs || total > bulkMaxURLs {
		apierror.WriteFieldError(w, http.StatusBadRequest, "urls", "BULK_URLS_INVALID",
			fmt.Sprintf("urls must contain %d-%d entries (got %d)", bulkMinURLs, bulkMaxURLs, total))
		return
	}

	// tag is optional: ONE tag applies to every imported link (the frontend
	// sends a string; it is re-normalized server-side: trim/lowercase/limits).
	tagsJSON := []byte("[]")
	if t := strings.TrimSpace(req.Tag); t != "" {
		tags, ok := normalizeTags([]string{t})
		if !ok {
			apierror.WriteFieldError(w, http.StatusBadRequest, "tag", "BULK_TAG_INVALID", "invalid tag (1-20 chars)")
			return
		}
		if b, err := json.Marshal(tags); err == nil {
			tagsJSON = b
		}
	}

	failures := make([]bulkError, 0, 8)
	seen := make(map[string]bool)       // dedupe in-request (URL already validated)
	batchCodes := make(map[string]bool) // code already taken by another row in the batch
	var items []db.BulkURL
	var lines []int // parallel to items: original row numbers for errors

	for i, raw := range req.URLs {
		line := i + 1
		u := strings.TrimSpace(raw)
		if !validRemoteURL(u) {
			failures = append(failures, bulkError{Line: line, URL: u, Error: "invalid_url"})
			continue
		}
		if seen[u] {
			failures = append(failures, bulkError{Line: line, URL: u, Error: "duplicate_in_request"})
			continue
		}
		seen[u] = true

		code, err := shortener.GenerateShortCode(6)
		if err != nil {
			h.Logger.Printf("GenerateShortCode failed: %v", err)
			apierror.WriteError(w, http.StatusInternalServerError, "LINK_CODE_ERROR", "Failed to generate short code")
			return
		}
		// Code collision ACROSS rows of the same batch: very rare (6 random
		// chars) but handled: regenerate at most 5x, otherwise leave it to
		// the ON CONFLICT in the DB.
		for tries := 0; batchCodes[code] && tries < 5; tries++ {
			if code, err = shortener.GenerateShortCode(6); err != nil {
				h.Logger.Printf("GenerateShortCode failed: %v", err)
				apierror.WriteError(w, http.StatusInternalServerError, "LINK_CODE_ERROR", "Failed to generate short code")
				return
			}
		}
		batchCodes[code] = true

		items = append(items, db.BulkURL{ShortCode: code, OriginalURL: u, TagsJSON: string(tagsJSON)})
		lines = append(lines, line)
	}

	// Not a single valid row: 400 plus the reason each row failed (more useful
	// than a 201 containing zero links).
	if len(items) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    "BULK_NO_VALID_URLS",
			"message": "no valid urls to import",
			"errors":  failures,
			"summary": map[string]int{"total": total, "created": 0, "failed": len(failures)},
		})
		return
	}

	// One transaction, one round-trip; code "" = ON CONFLICT (already taken).
	codes, err := h.Store.CreateURLsBatch(creatorID, items)
	if err != nil {
		h.Logger.Printf("CreateURLsBatch failed: %v", err)
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
		return
	}

	scheme := requestScheme(r)
	created := make([]map[string]string, 0, len(items))
	for i, code := range codes {
		it := items[i]
		if code == "" {
			failures = append(failures, bulkError{Line: lines[i], URL: it.OriginalURL, Error: "duplicate"})
			continue
		}
		// Populate the cache write-through, same as doShorten (bulk imports
		// carry no expiry → default TTL; redirectTTL(nil) = 300).
		if h.Cache != nil {
			if b, err := json.Marshal(redirectTarget{URL: it.OriginalURL}); err == nil {
				if err := h.Cache.Set(code, string(b), redirectTTL(nil)); err != nil {
					h.Logger.Printf("Warning: failed to populate cache for %q: %v", code, err)
				}
			}
		}
		created = append(created, map[string]string{
			"short_code":   code,
			"short_url":    scheme + "://" + r.Host + "/r/" + code,
			"original_url": it.OriginalURL,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"created": created,
		"errors":  failures,
		"summary": map[string]int{"total": total, "created": len(created), "failed": len(failures)},
	})
}

// HandleBulkQR is GET /api/links/qr-bulk.zip: a ZIP download of QR PNGs
// (512px, Medium level) for ALL of a creator's links, or filtered with
// ?tag=kerja. Specification: 0 matches → 400 (not an empty ZIP), > 200 links
// → 400 (use the filter), and the PNGs are generated BEFORE the ZIP headers
// are sent: an encode failure midway could no longer be turned into a 500
// JSON response. QR content = the absolute short URL (scheme + host from the
// request, exactly as short_url is built in doShorten) so it scans directly.
func (h *Handler) HandleBulkQR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apierror.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		apierror.WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Login required")
		return
	}

	filterTag := strings.TrimSpace(r.URL.Query().Get("tag"))
	if filterTag != "" {
		tags, ok := normalizeTags([]string{filterTag})
		if !ok || len(tags) == 0 {
			apierror.WriteFieldError(w, http.StatusBadRequest, "tag", "BULK_TAG_INVALID", "invalid tag (1-20 chars)")
			return
		}
		filterTag = tags[0]
	}

	links, err := h.Store.ListLinksByCreatorPrimary(*creatorID)
	if err != nil {
		h.Logger.Printf("ListLinksByCreatorPrimary failed: %v", err)
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
		return
	}

	var matches []db.Link
	for _, l := range links {
		if filterTag != "" {
			has := false
			for _, t := range l.Tags {
				if t == filterTag {
					has = true
					break
				}
			}
			if !has {
				continue
			}
		}
		matches = append(matches, l)
	}
	if len(matches) == 0 {
		apierror.WriteError(w, http.StatusBadRequest, "QR_NO_MATCH", "No links match this tag")
		return
	}
	if len(matches) > bulkQRMaxLinks {
		apierror.WriteFieldError(w, http.StatusBadRequest, "tag", "QR_TOO_MANY",
			fmt.Sprintf("too many links (%d > %d): filter with ?tag=", len(matches), bulkQRMaxLinks))
		return
	}

	// Pre-generate every PNG before writing a single response byte: if a QR
	// fails, the answer can still be a proper 500 JSON.
	scheme := requestScheme(r)
	pngs := make([][]byte, len(matches))
	for i, l := range matches {
		qrURL := scheme + "://" + r.Host + "/r/" + l.ShortCode
		png, err := qrcode.Encode(qrURL, qrcode.Medium, 512)
		if err != nil {
			h.Logger.Printf("QR encode failed for %q: %v", l.ShortCode, err)
			apierror.WriteError(w, http.StatusInternalServerError, "QR_GENERATE_ERROR", "Failed to generate QR code")
			return
		}
		pngs[i] = png
	}

	filename := "jejak-qr.zip"
	if filterTag != "" {
		filename = "jejak-qr-" + safeFilenamePart(filterTag) + ".zip"
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	zw := zip.NewWriter(w)
	for i, l := range matches {
		f, err := zw.Create(l.ShortCode + ".png")
		if err != nil {
			h.Logger.Printf("zip.Create failed: %v", err)
			return // headers already sent: nothing left but to stop
		}
		if _, err := f.Write(pngs[i]); err != nil {
			h.Logger.Printf("zip write failed for %q: %v", l.ShortCode, err)
			return
		}
	}
	if err := zw.Close(); err != nil {
		h.Logger.Printf("zip close failed: %v", err)
	}
}

// requestScheme derives the short URL's scheme from the request:
// X-Forwarded-Proto wins over TLS (a proxy sits in front), default http: the
// same logic used to build short_url in doShorten (rationale documented
// there).
func requestScheme(r *http.Request) string {
	if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
		return p
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

// safeFilenamePart sanitizes a tag for use in a ZIP file name: only
// [a-z0-9_-] survive and everything else becomes "-", preventing path/quote
// injection from a user-supplied tag (even after normalization, tags are not
// charset-limited).
func safeFilenamePart(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "tag"
	}
	return out
}
