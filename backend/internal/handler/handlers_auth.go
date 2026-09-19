package handler

import (
	"encoding/json"
	"net/http"

	"jejak/internal/auth"
	"jejak/internal/ratelimit"
)

// LEARN:
//
//	Kenapa: Register + auto-login dalam 1 call (UX: langsung bisa bikin link).
//	Username unik dijamin UNIQUE constraint DB (bukan cek-then-insert race);
//	password tidak pernah disimpan plaintext (bcrypt via auth package).
//	Trade-off: Auto-login berarti register selalu membuat session (1 write memori
//	ekstra); respons 201 mengembalikan profil TANPA hash (jangan pernah echo hash).
//	Alternatif: Email verification / OTP, tapi overkill untuk akun kreator MVP.
func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Auth == nil {
		http.Error(w, "Auth not configured", http.StatusInternalServerError)
		return
	}

	var req struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Bio         string `json:"bio"`
		Password    string `json:"password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if !auth.ValidUsername(req.Username) {
		http.Error(w, "Invalid username (3-30 chars, letters/digits/underscore)", http.StatusBadRequest)
		return
	}
	if req.DisplayName == "" || len(req.Password) < 8 {
		http.Error(w, "display_name required, password min 8 chars", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to secure password", http.StatusInternalServerError)
		return
	}
	id, err := h.Store.CreateCreator(req.Username, req.DisplayName, req.Bio, hash)
	if err != nil {
		// UNIQUE violation -> username taken. String match is driver-specific
		// but pgx/pq both surface "duplicate key" for 23505; keep generic 409.
		h.Logger.Printf("CreateCreator failed: %v", err)
		http.Error(w, "Username already taken", http.StatusConflict)
		return
	}

	token, err := h.Auth.Create(id)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}
	auth.SetCookie(w, token)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id, "username": req.Username})
}

// HandleLogin verifies credentials and mints a session cookie.
// Unknown user and wrong password return the SAME 401 (no user enumeration).
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Auth == nil {
		http.Error(w, "Auth not configured", http.StatusInternalServerError)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Throttle brute force: 5 gagal/menit per IP+username (lihat LEARN di
	// ratelimit package). Cek SEBELUM bcrypt supaya CPU tidak dibakar penyerang.
	rlKey := ratelimit.Key(ratelimit.ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For")), req.Username)
	if h.LoginLimiter != nil && !h.LoginLimiter.Allow(rlKey) {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "Too many login attempts, try again later", http.StatusTooManyRequests)
		return
	}

	creator, err := h.Store.GetCreatorByUsername(req.Username)
	if err != nil || !auth.CheckPassword(creator.PasswordHash, req.Password) {
		if h.LoginLimiter != nil {
			h.LoginLimiter.Record(rlKey)
		}
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}
	if h.LoginLimiter != nil {
		h.LoginLimiter.Reset(rlKey)
	}

	token, err := h.Auth.Create(creator.ID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}
	auth.SetCookie(w, token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"id": creator.ID, "username": creator.Username})
}

// HandleLogout revokes the server-side session AND clears the client cookie.
// Both halves matter: cookie tanpa server entry = mati; entry tanpa clear
// cookie = browser ngirim token mati terus (harmless tapi berisik di log).
func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Auth == nil {
		http.Error(w, "Auth not configured", http.StatusInternalServerError)
		return
	}

	if token := auth.TokenFromRequest(r); token != "" {
		h.Auth.Delete(token)
	}
	auth.ClearCookie(w)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true}`))
}
