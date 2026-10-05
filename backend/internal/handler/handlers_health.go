package handler

// Link health monitor API (migration 17, 2026-10-04): the manual "check
// now" trigger and the in-app notification feed for the navbar bell. All
// endpoints are session-auth (authH in main) and creator-scoped: another
// user's short code answers 404, never 403, so the existence of someone
// else's code is not disclosed (same rule as HandleUpdateLink).

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"jejak/internal/apierror"
	"jejak/internal/middleware"
	"jejak/internal/ratelimit"
)

// HandleCheckHealth runs ONE live health check for an owned link
// (POST /api/links/{short_code}/check-health). The ownership check runs
// BEFORE the outbound request: a foreign code must not make this server
// probe an arbitrary URL on the attacker's behalf (SSRF-shaped abuse), and
// the caller sees 404 without a network round trip.
//
// Rate limit: 10/minute per IP (HealthTriggerLimiter) - each attempt makes
// a live request (5s worst case), so it needs its own budget next to the
// worker's 50/min batch (documented deviation: the two budgets are not
// globally coordinated).
//
// Response: {"short_code","health_status","last_health_check"} - the same
// shape the dashboard list already carries, so the frontend can patch its
// local state without refetching.
func (h *Handler) HandleCheckHealth(shortCode string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierror.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		apierror.WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Login required")
		return
	}
	rlKey := "health\x00" + ratelimit.ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For"))
	if h.HealthTriggerLimiter != nil {
		if !h.HealthTriggerLimiter.Allow(rlKey) {
			w.Header().Set("Retry-After", "60")
			apierror.WriteError(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many requests, try again later")
			return
		}
	}
	if h.Checker == nil {
		apierror.WriteError(w, http.StatusServiceUnavailable, "HEALTH_UNAVAILABLE", "Health checker not available")
		return
	}

	// Ownership first (see SSRF note above).
	link, err := h.Store.GetLink(shortCode)
	if err != nil {
		if err == sql.ErrNoRows {
			apierror.WriteError(w, http.StatusNotFound, "LINK_NOT_FOUND", "Unknown short code or not yours")
			return
		}
		h.Logger.Printf("GetLink failed for %q: %v", shortCode, err)
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
		return
	}
	if link.CreatorID == nil || *link.CreatorID != *creatorID {
		apierror.WriteError(w, http.StatusNotFound, "LINK_NOT_FOUND", "Unknown short code or not yours")
		return
	}

	// Quota is only spent on a real attempt (after auth + ownership), like
	// the shorten limiter records after Allow.
	if h.HealthTriggerLimiter != nil {
		h.HealthTriggerLimiter.Record(rlKey)
	}

	status, err := h.Checker.CheckOne(r.Context(), shortCode)
	if err != nil {
		if err == sql.ErrNoRows {
			apierror.WriteError(w, http.StatusNotFound, "LINK_NOT_FOUND", "Unknown short code or not yours")
			return
		}
		h.Logger.Printf("CheckOne failed for %q: %v", shortCode, err)
		apierror.WriteError(w, http.StatusInternalServerError, "HEALTH_CHECK_FAILED", "Failed to run health check")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"short_code":        shortCode,
		"health_status":     status,
		"last_health_check": time.Now().UTC().Format(time.RFC3339),
	})
}

// HandleListNotifications returns the creator's newest 50 notifications
// plus the unread count for the bell badge (GET /api/notifications).
// Empty feed = {"notifications":[],"unread":0}, not an error.
func (h *Handler) HandleListNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apierror.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		apierror.WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Login required")
		return
	}
	items, err := h.Store.ListNotifications(*creatorID, 50)
	if err != nil {
		h.Logger.Printf("ListNotifications failed: %v", err)
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
		return
	}
	unread, err := h.Store.CountUnreadNotifications(*creatorID)
	if err != nil {
		h.Logger.Printf("CountUnreadNotifications failed: %v", err)
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"notifications": items,
		"unread":        unread,
	})
}

// HandleMarkNotificationRead marks ONE notification read
// (PUT /api/notifications/{id}/read). Owner-scoped in the store: a foreign
// or unknown id -> 404 (no id probing). Idempotent: marking an already-read
// notification again succeeds (still 1 affected row).
func (h *Handler) HandleMarkNotificationRead(idParam string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		apierror.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		apierror.WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Login required")
		return
	}
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		apierror.WriteError(w, http.StatusBadRequest, "NOTIFICATION_INVALID_ID", "Invalid notification id")
		return
	}
	if err := h.Store.MarkNotificationRead(*creatorID, id); err != nil {
		if err == sql.ErrNoRows {
			apierror.WriteError(w, http.StatusNotFound, "NOTIFICATION_NOT_FOUND", "Notification not found")
			return
		}
		h.Logger.Printf("MarkNotificationRead failed: %v", err)
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}
