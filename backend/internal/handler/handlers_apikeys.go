package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"jejak/internal/db"
	"jejak/internal/middleware"
)

// hashAPIKey hashes an API key with SHA-256 before storage: the same
// principle as passwords: if the database leaks (dump, exposed backup,
// SQLi), api_keys holds only hashes and no key is usable directly. A fast
// hash is preferred over bcrypt because keys are random 128-bit values
// ("jjk_" + 32 hex), so offline brute force stays computationally impossible
// even with SHA-256 (unlike low-entropy user passwords), and the hash is
// verified on every public API request where bcrypt would add ~100 ms per
// call and SHA-256 takes microseconds.
func hashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// generateAPIKey produces a new key: "jjk_" + 32 hex (16 bytes / 128 bits
// from crypto/rand: a CSPRNG; math/rand must never be used for secret
// material). The plaintext exists only in this function's memory and is sent
// in the response ONCE; only its hash reaches the database.
func generateAPIKey() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return "jjk_" + hex.EncodeToString(buf[:]), nil
}

func bearerKeyFromRequest(r *http.Request) (string, bool) {
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		return "", false
	}
	return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), true
}

// keysJSON shapes API keys for responses: NEVER leaks key_hash, and renders
// nullable columns as "" / null (never SQL artifacts). Same principle as
// creatorProfileJSON.
func keysJSON(keys []db.APIKey) []map[string]any {
	out := make([]map[string]any, len(keys))
	for i, k := range keys {
		label := ""
		if k.Label.Valid {
			label = k.Label.String
		}
		item := map[string]any{
			"id":         k.ID,
			"label":      label,
			"created_at": k.CreatedAt,
		}
		if k.LastUsedAt.Valid {
			item["last_used_at"] = k.LastUsedAt.Time
		} else {
			item["last_used_at"] = nil
		}
		out[i] = item
	}
	return out
}

// HandleGenerateAPIKey creates a key (POST /api/keys). Session auth is
// MANDATORY: key generation must happen from a logged-in dashboard, never
// through another API key. The response carries the plaintext key EXACTLY
// ONCE; afterwards the key can never be viewed again, which is the standard
// for API key generation.
func (h *Handler) HandleGenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		http.Error(w, "Login required", http.StatusUnauthorized)
		return
	}

	var req struct {
		Label string `json:"label"`
	}
	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	}
	req.Label = strings.TrimSpace(req.Label)
	if len(req.Label) > 100 {
		http.Error(w, "Label max 100 chars", http.StatusBadRequest)
		return
	}

	key, err := generateAPIKey()
	if err != nil {
		h.Logger.Printf("GenerateAPIKey failed: %v", err)
		http.Error(w, "Failed to generate key", http.StatusInternalServerError)
		return
	}
	id, err := h.Store.StoreAPIKey(*creatorID, hashAPIKey(key), req.Label)
	if err != nil {
		h.Logger.Printf("StoreAPIKey failed: %v", err)
		http.Error(w, "Failed to store key", http.StatusInternalServerError)
		return
	}

	// The plaintext key appears ONLY in this response; every later response
	// (GET /api/keys) returns metadata only: id, label, timestamps.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id":         id,
		"label":      req.Label,
		"created_at": time.Now(),
		"key":        key,
	})
}

// HandleListAPIKeys returns the caller's keys (GET /api/keys). Session auth
// required. The plaintext key is NEVER sent: only id, label and timestamps.
func (h *Handler) HandleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		http.Error(w, "Login required", http.StatusUnauthorized)
		return
	}

	keys, err := h.Store.ListAPIKeys(*creatorID)
	if err != nil {
		h.Logger.Printf("ListAPIKeys failed: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if keys == nil {
		keys = []db.APIKey{}
	}
	json.NewEncoder(w).Encode(keysJSON(keys))
}

// HandleDeleteAPIKey removes one key (DELETE /api/keys/{id}). Session auth
// required, scoped by WHERE creator_id: another creator's key or a foreign
// id yields 404 rather than 403, the same anti-leak principle used by
// UpdateLink/Reorder.
func (h *Handler) HandleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		http.Error(w, "Login required", http.StatusUnauthorized)
		return
	}
	keyID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid key id", http.StatusBadRequest)
		return
	}

	if err := h.Store.DeleteAPIKey(*creatorID, keyID); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Unknown API key or not yours", http.StatusNotFound)
			return
		}
		h.Logger.Printf("DeleteAPIKey failed (id=%d): %v", keyID, err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// creatorFromAPIKey authenticates a public API request: Bearer token →
// SHA-256 hash → api_keys lookup → creatorID. Errors surface as 401 or 500
// and are handled by the caller. As with passwords, the hash exists only for
// storage: matching re-hashes the input instead of decrypting the stored
// value (SHA-256 is one-way). The last_used_at update is fire-and-forget:
// the request has already succeeded at this point, so a failed timestamp
// write is only logged and must never fail the request.
func (h *Handler) creatorFromAPIKey(r *http.Request) (*int64, error) {
	raw, ok := bearerKeyFromRequest(r)
	if !ok || !strings.HasPrefix(raw, "jjk_") {
		return nil, sql.ErrNoRows
	}
	keyHash := hashAPIKey(raw)
	k, err := h.Store.GetAPIKeyByHash(keyHash)
	if err != nil {
		return nil, err
	}
	if err := h.Store.TouchAPIKeyLastUsed(k.ID); err != nil {
		h.Logger.Printf("Warning: TouchAPIKeyLastUsed failed: %v", err)
	}
	id := k.CreatorID
	return &id, nil
}

// HandleV1Shorten authenticates with an API key rather than a session
// cookie: cookies require a browser (login plus cookie handling), while
// creators automating through curl, scripts or integrations hold a key
// instead. Two auth paths therefore yield the same creator context (the
// creator id): the session cookie for the dashboard and the Bearer API key
// for the public API: and doShorten neither knows nor needs to know which
// path supplied the id: polymorphic identity sources, one business
// implementation. The rate limit is kept separate from browser/login traffic:
// 100 req/minute PER KEY (versus 5 failures/minute for login), keyed by the
// key's hash so every key gets its own budget and a dead key is throttled
// under its own hash without affecting other keys.
func (h *Handler) HandleV1Shorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	creatorID, err := h.creatorFromAPIKey(r)
	if err != nil {
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	// Rate limit: 100 req/minute per key; Allow + Record together consume a
	// single unit of quota.
	if h.APILimiter != nil {
		raw, _ := bearerKeyFromRequest(r)
		keyHash := hashAPIKey(raw)
		if !h.APILimiter.Allow(keyHash) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		h.APILimiter.Record(keyHash)
	}

	h.doShorten(creatorID, w, r)
}
