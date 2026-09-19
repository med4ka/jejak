package handler

import (
	"encoding/json"
	"net/http"

	"jejak/internal/db"
)

// LEARN:
//
//	Kenapa: Halaman publik kreator (/api/u/{username}) memakai denormalized
//	urls.click_count — 1 query WHERE creator_id (lihat db.ListLinksByCreator),
//	BUKAN 1 query list + N query count (N+1). Keputusan ini disengaja (PRD Fase 9):
//	counter sudah di-maintain di path redirect, jadi baca profil tidak perlu
//	agregat click_events yang berat.
//	Trade-off: Angka bisa sedikit basi vs click_events real-time (replica lag +
//	async worker), tapi halaman profil butuh cepat, bukan audit-grade.
//	Alternatif: JOIN agregat COUNT(click_events) GROUP BY — tepat tapi mahal
//	dan mengalahkan tujuan denormalisasi Fase 2.
func (h *Handler) HandleCreatorLinks(username string, w http.ResponseWriter, r *http.Request) {
	creator, err := h.Store.GetCreatorByUsername(username)
	if err != nil {
		http.Error(w, "Creator not found", http.StatusNotFound)
		return
	}

	links, err := h.Store.ListLinksByCreator(creator.ID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if links == nil {
		links = []db.Link{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(creatorProfileJSON(creator, links))
}
