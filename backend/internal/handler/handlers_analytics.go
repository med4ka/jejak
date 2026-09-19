package handler

import (
	"encoding/json"
	"net/http"

	"jejak/internal/middleware"
)

// HandleClicksByDay serves GET /api/analytics/clicks-by-day for the logged-in
// creator: 30-day window [{date, count}] with zeros filled (see db LEARN).
// Auth required (a creator sees only their own numbers); reads go through
// the store's read path (replica when enabled, same staleness caveats).
func (h *Handler) HandleClicksByDay(w http.ResponseWriter, r *http.Request) {
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		http.Error(w, "Login required", http.StatusUnauthorized)
		return
	}

	days, err := h.Store.ClicksByDay(*creatorID)
	if err != nil {
		h.Logger.Printf("ClicksByDay failed: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(days)
}
