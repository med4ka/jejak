package handler

import (
	"encoding/json"
	"net/http"

	"jejak/internal/auth"
	"jejak/internal/ratelimit"
)

// HandleRegister creates the account and logs the creator in within a single
// call, so link creation can start immediately after signup. Uniqueness of
// the username is enforced by the database UNIQUE constraint rather than a
// check-then-insert sequence (no race), and the password is never stored in
// plaintext (bcrypt via the auth package). Trade-off: auto-login means every
// registration also creates a session (one extra in-memory write), and the
// 201 response returns the profile without the hash: the hash must never be
// echoed. Email verification / OTP was rejected as overkill for an MVP
// creator account.
func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Auth == nil {
		http.Error(w, "Auth not configured", http.StatusInternalServerError)
		return
	}

	// Account-spam / CPU throttle: 10 registrations/minute per IP. Unlike
	// login, EVERY attempt is recorded (no credential separates a "failed"
	// from a "succeeded" attempt at this point): a flood of duplicate
	// usernames must drain the bucket just like valid signups, otherwise an
	// attacker could submit endless taken usernames for free. Checked before
	// decode+bcrypt so an attacker cannot burn CPU.
	rlKey := "reg\x00" + ratelimit.ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For"))
	if h.RegisterLimiter != nil {
		if !h.RegisterLimiter.Allow(rlKey) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too many registration attempts, try again later", http.StatusTooManyRequests)
			return
		}
		h.RegisterLimiter.Record(rlKey)
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

	// Brute-force throttle: 5 failures/minute per IP+username (rationale in
	// the ratelimit package). Checked BEFORE bcrypt so an attacker cannot
	// burn CPU.
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
// Both halves matter: a cookie without a server entry is dead anyway, while
// an entry left behind without clearing the cookie makes the browser keep
// sending a dead token: harmless but noisy in the logs.
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
