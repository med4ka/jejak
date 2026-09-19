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

	"jejak/internal/middleware"
	"jejak/internal/db"
)

// LEARN:
//
//	Kenapa API key di-hash (SHA-256) sebelum disimpan — prinsip yang SAMA
//	dengan password. Kalau database bocor (dump, backup bocor, SQLi), isi
//	api_keys hanyalah deretan hash; attacker tidak langsung bisa memakai key
//	apa pun. Kecepatan hash dipilih beda dari bcrypt karena: (1) key kita
//	random 128-bit ("jjk_" + 32 hex) — brute-force offline mustahil secara
//	komputasi meski pakai SHA-256, tidak seperti password user yang berentropy
//	rendah; (2) hash API key dicek di SETIAP request public API, bcrypt = +100ms
//	latensi per panggilan, SHA-256 = mikrodetik. Trade-off-nya jelas: fungsi
//	lambat tidak perlu untuk materi berentropy tinggi yang diautentikasi sering.
func hashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// generateAPIKey memproduksi key baru. Format "jjk_" + 32 hex (16 byte / 128
// bit dari crypto/rand — sumber CSPRNG, jangan pakai math/rand untuk materi
// sekret). Plaintext hanya ada di memori fungsi pembuatnya lalu dikirim ke
// response SEKALI; yang masuk database hanyalah hash-nya.
func generateAPIKey() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return "jjk_" + hex.EncodeToString(buf[:]), nil
}

// bearerKeyFromRequest mengekstrak "Authorization: Bearer <key>".
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

// HandleGenerateAPIKey creates a key (POST /api/keys). Session auth WAJIB —
// generate key harus lewat dashboard yang login, bukan lewat API key lain.
// Response memuat plaintext key SEKALI SAJA; sesudah ini key tidak pernah
// bisa dilihat lagi (standar API key generation).
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

	// Plaintext key — SENDANG response ini yang memuatnya. Response kedua dan
	// seterusnya (GET /api/keys) hanya memuat hash (label/timestamps).
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
// required. Plaintext key TIDAK pernah dikirim — hanya id/label/timestamps.
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
// required, scoped WHERE creator_id (key kepunyaan orang lain / id asing ->
// 404, bukan 403, sama prinsip anti-leak seperti UpdateLink/Reorder).
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

// creatorFromAPIKey mengautentikasi request public API via Bearer token →
// SHA-256 hash → lookup api_keys → kembalikan creatorID. Error = 401 atau
// 500 (dihandle oleh caller). LEARN: Sama seperti password — hash hanya
// untuk penyimpanan; pencocokan dilakukan dengan meng-hash input, bukan
// mendekrip hash (karena SHA-256 one-way). Update last_used_at dilakukan
// fire-and-forget: request SUDAH sukses pada titik ini, gagal update
// timestamp hanya log (tidak boleh gagalkan request). Lihat LEARN di atas.
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

// LEARN:
//
//	Kenapa HandlerV1Shorten memakai API key, BUKAN session cookie: cookie
//	memerlukan browser (login + cookie dikelola browser). Kreator yang
//	otomatisasi (curl, script, integrasi) tidak punya session browser — mereka
//	punya key. Jadi, muncul 2 jalur auth yang menghasilkan "creator context"
//	yang sama (id kreator): session cookie (dashboard) dan Bearer API key
//	(public API). doShorten tidak tahu (dan tidak perlu tahu) dari jalur mana
//	id datang — polimorfisme sumber identitas, satu implementasi bisnis.
//	Rate limit endpoint ini dipisah dari browser/login: 100 req/menit PER KEY
//	(dibanding 5 gagal/menit untuk login). Bucket di-key oleh hash key, jadi
//	tiap key punya jatah sendiri; key mati tetap ikut di-throttle sesuai
//	hash-nya (tidak merugikan key lain).
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

	// Rate limit: 100 req/menit per key. Allow + Record = konsumsi 1 kuota.
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
