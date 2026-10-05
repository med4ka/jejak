package handler

import (
	"encoding/json"
	"net/http"

	"jejak/internal/apierror"
	"jejak/internal/db"
)

// HandleCreatorLinks serves the public creator page (/api/u/{username}) from
// the denormalized urls.click_count: a single WHERE creator_id query (see
// db.ListLinksByCreator), NOT one list query plus N count queries (N+1). The
// choice is deliberate (PRD Phase 9): the counter is maintained on the
// redirect path, so reading a profile needs no heavy click_events
// aggregation. Trade-off: the figure can lag real-time click_events
// (denormalization plus an async worker), but a profile page must be fast,
// not audit-grade. A COUNT(click_events) GROUP BY join would be exact but
// expensive, and would defeat the Phase 2 denormalization.
func (h *Handler) HandleCreatorLinks(username string, w http.ResponseWriter, r *http.Request) {
	creator, err := h.Store.GetCreatorByUsername(username)
	if err != nil {
		apierror.WriteError(w, http.StatusNotFound, "PROFILE_NOT_FOUND", "Creator not found")
		return
	}

	links, err := h.Store.ListLinksByCreator(creator.ID)
	if err != nil {
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
		return
	}
	if links == nil {
		links = []db.Link{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(creatorProfileJSON(creator, links))
}
