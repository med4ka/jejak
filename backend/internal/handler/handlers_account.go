package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"jejak/internal/auth"
	"jejak/internal/db"
	"jejak/internal/middleware"
)

// validEmail uses a simple pattern (one @, a dotted domain, no spaces):
// enough for an MVP without a library; actual mail verification stays the
// job of the confirmation-link flow (out of scope for account settings).
var validEmail = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// accountCreatorID resolves the identity for account settings endpoints:
// the session cookie (dashboard path) first, then a Bearer API key
// (automation path: same pattern as HandleAnalyticsSummary). The route uses
// OptionalAuth, so anonymous requests reach this helper and nil means 401.
func (h *Handler) accountCreatorID(r *http.Request) *int64 {
	if id := middleware.CreatorID(r); id != nil {
		return id
	}
	if id, err := h.creatorFromAPIKey(r); err == nil {
		return id
	}
	return nil
}

// decodeAccountBody is a thin wrapper over decodeJSON so that all four
// account endpoints decode request bodies identically.
func (h *Handler) decodeAccountBody(w http.ResponseWriter, r *http.Request, dst any) error {
	return decodeJSON(w, r, dst)
}

// HandleUpdateEmail requires the current password because a session cookie
// can be stolen (old XSS, a borrowed device): a holder of the session
// without the password must not be able to take over the account through an
// email change, since future recovery/reset would be directed to the
// attacker's address. Proving the password separates the true account owner
// from a mere session holder. Trade-off: slight friction (the password must
// be remembered), but an identity-changing action warrants the cost. An OTP
// to the old address was rejected as overkill without a mailer in the MVP.
func (h *Handler) HandleUpdateEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	creatorID := h.accountCreatorID(r)
	if creatorID == nil {
		http.Error(w, "Login required", http.StatusUnauthorized)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := h.decodeAccountBody(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	if !validEmail.MatchString(req.Email) || len(req.Email) > 255 {
		http.Error(w, "Email tidak valid", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, "Password saat ini wajib diisi", http.StatusBadRequest)
		return
	}

	creator, err := h.Store.GetCreatorAuth(*creatorID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusUnauthorized)
		return
	}
	if !auth.CheckPassword(creator.PasswordHash, req.Password) {
		http.Error(w, "Password salah", http.StatusForbidden)
		return
	}

	// An email equal to the current one is a successful no-op: it avoids a
	// self-conflict on the unique index against the row itself.
	if creator.Email != req.Email {
		if err := h.Store.UpdateCreatorEmail(*creatorID, req.Email); err != nil {
			if errors.Is(err, db.ErrEmailTaken) {
				http.Error(w, "Email sudah dipakai akun lain", http.StatusConflict)
				return
			}
			h.Logger.Printf("UpdateCreatorEmail failed: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "email": req.Email})
}

// HandleUpdatePassword changes the password: verify the current one, hash
// the new one (bcrypt), then write it. Sessions on OTHER devices are revoked
// (keepToken is the current session) so a device that lost access in a
// breach is logged out as well, while the current device stays on the
// dashboard.
func (h *Handler) HandleUpdatePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	creatorID := h.accountCreatorID(r)
	if creatorID == nil {
		http.Error(w, "Login required", http.StatusUnauthorized)
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := h.decodeAccountBody(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		http.Error(w, "Password lama dan baru wajib diisi", http.StatusBadRequest)
		return
	}
	if len(req.NewPassword) < 8 {
		http.Error(w, "Password baru minimal 8 karakter", http.StatusBadRequest)
		return
	}

	creator, err := h.Store.GetCreatorAuth(*creatorID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusUnauthorized)
		return
	}
	if !auth.CheckPassword(creator.PasswordHash, req.CurrentPassword) {
		http.Error(w, "Password salah", http.StatusForbidden)
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		h.Logger.Printf("HashPassword failed: %v", err)
		http.Error(w, "Failed to secure password", http.StatusInternalServerError)
		return
	}
	if err := h.Store.UpdateCreatorPassword(*creatorID, hash); err != nil {
		h.Logger.Printf("UpdateCreatorPassword failed: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Revoke other-device sessions (keep = the requesting session). On the
	// Bearer API key path there is no cookie token, so keepToken is "" and
	// every session is revoked: acceptable because the caller is not a
	// logged-in browser.
	if h.Auth != nil {
		if _, err := h.Auth.DeleteAllForUser(*creatorID, auth.TokenFromRequest(r)); err != nil {
			// A failed session revocation must NOT fail the request: the password
			// has already been changed correctly in the database, so log only.
			h.Logger.Printf("Warning: DeleteAllForUser after password change: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// HandleLogoutAll revokes EVERY session of the account (including this one)
// and clears the cookie; it backs the "Logout from All Devices" button in
// the Danger Zone. Unlike POST /api/logout (this session only), it is a
// security action for when another device is suspected of still being logged
// in.
func (h *Handler) HandleLogoutAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	creatorID := h.accountCreatorID(r)
	if creatorID == nil {
		http.Error(w, "Login required", http.StatusUnauthorized)
		return
	}
	if h.Auth != nil {
		if _, err := h.Auth.DeleteAllForUser(*creatorID, ""); err != nil {
			h.Logger.Printf("DeleteAllForUser failed: %v", err)
			http.Error(w, "Failed to revoke sessions", http.StatusInternalServerError)
			return
		}
	}
	auth.ClearCookie(w)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// HandleDeleteAccount deletes the account PERMANENTLY (Danger Zone, two
// steps): the body must carry the literal confirmation "HAPUS" plus the
// correct password. The two distinct keys (typing plus proof of ownership)
// prevent an accidental single-click deletion while still being safe against
// a stolen session. Data is purged in order (click_events → urls → api_keys
// → creators), then every session is revoked and the cookie is cleared.
func (h *Handler) HandleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	creatorID := h.accountCreatorID(r)
	if creatorID == nil {
		http.Error(w, "Login required", http.StatusUnauthorized)
		return
	}

	var req struct {
		Confirmation string `json:"confirmation"`
		Password     string `json:"password"`
	}
	if err := h.decodeAccountBody(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Confirmation != "HAPUS" {
		http.Error(w, `Konfirmasi harus tepat "HAPUS"`, http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, "Password wajib diisi", http.StatusBadRequest)
		return
	}

	creator, err := h.Store.GetCreatorAuth(*creatorID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusUnauthorized)
		return
	}
	if !auth.CheckPassword(creator.PasswordHash, req.Password) {
		http.Error(w, "Password salah", http.StatusForbidden)
		return
	}

	if err := h.Store.DeleteCreatorAccount(*creatorID); err != nil {
		h.Logger.Printf("DeleteCreatorAccount failed (id=%d): %v", *creatorID, err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if h.Auth != nil {
		if _, err := h.Auth.DeleteAllForUser(*creatorID, ""); err != nil {
			h.Logger.Printf("Warning: DeleteAllForUser after account delete: %v", err)
		}
	}
	auth.ClearCookie(w)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}
