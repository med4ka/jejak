package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"jejak/internal/middleware"
	"jejak/internal/db"
)

// LEARN:
//
//	Kenapa: PUT profil memakai semantik full-replace (seluruh display_name +
//	bio + avatar + socials dikirim tiap save), bukan PATCH per-field. Untuk
//	form kecil ini lebih sederhana: tidak ada merge-setengah-jalan, validasi
//	satu tempat, dan respons bisa echo input tervalidasi TANPA baca ulang DB
//	(baca-ulang dari replica akan basi karena update baru masuk primary —
//	jebakan read-your-write yang sama seperti smoke test clicks kemarin).
//	Batasan validasi (nama 1-100, bio ≤500, ≤10 sosial, URL http(s)) adalah
//	pilihan MVP yang eksplisit, bukan aturan produk final.
//	Trade-off: Client harus selalu kirim lengkap (lupa 1 field = ke-reset);
//	username immutable (ganti username = ganti URL publik + cek unik ulang).
//	Alternatif: PATCH JSON-merge per field — fleksibel tapi butuh logika
//	"field absen vs kosong" yang gampang salah.
func (h *Handler) HandleGetMyProfile(w http.ResponseWriter, r *http.Request) {
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		http.Error(w, "Login required", http.StatusUnauthorized)
		return
	}

	// Baca dari PRIMARY (bukan replica): ini data milik user sendiri, jadi
	// user harus langsung lihat perubahannya sendiri tanpa menunggu
	// sinkronisasi replica manual (read-your-own-writes). Halaman publik
	// /api/u/{username} TETAP baca replica (HandleCreatorLinks).
	creator, err := h.Store.GetCreatorByIDPrimary(*creatorID)
	if err != nil {
		http.Error(w, "Creator not found", http.StatusNotFound)
		return
	}

	links, err := h.Store.ListLinksByCreatorPrimary(creator.ID)
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

func (h *Handler) HandleUpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		http.Error(w, "Login required", http.StatusUnauthorized)
		return
	}

	var req struct {
		DisplayName string          `json:"display_name"`
		Bio         string          `json:"bio"`
		AvatarURL   string          `json:"avatar_url"`
		Socials     []db.SocialLink `json:"socials"`
		Theme       string          `json:"theme"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.Bio = strings.TrimSpace(req.Bio)
	req.AvatarURL = strings.TrimSpace(req.AvatarURL)
	if req.DisplayName == "" || len(req.DisplayName) > 100 {
		http.Error(w, "display_name required (1-100 chars)", http.StatusBadRequest)
		return
	}
	if len(req.Bio) > 500 {
		http.Error(w, "bio max 500 chars", http.StatusBadRequest)
		return
	}
	if !validHTTPURL(req.AvatarURL) {
		http.Error(w, "avatar_url must be empty or http(s) URL", http.StatusBadRequest)
		return
	}
	// Tema dari preset tertutup (classic|darkroom|coral|glass) — bukan
	// free-form color. Set terbuka ke sana saat bereksperimen preset baru;
	// theme "glass" sengaja EXPONENT (bisa dihapus lagi tanpa migrasi —
	// migrasi — teks lancar di-read validTheme, jadi GET tidak pernah bocor
	// nilai tak dikenal ke frontend).
	// Data kotor (mis. kolom DB yang di-set manual) di-coerce di layer payload
	// (validTheme), jadi GET tidak pernah bocor nilai tak dikenal ke frontend.
	if !validThemePreset(req.Theme) {
		http.Error(w, "theme must be one of: classic, darkroom, coral, glass", http.StatusBadRequest)
		return
	}
	if len(req.Socials) > 10 {
		http.Error(w, "max 10 social links", http.StatusBadRequest)
		return
	}
	for i := range req.Socials {
		req.Socials[i].Platform = strings.TrimSpace(req.Socials[i].Platform)
		req.Socials[i].URL = strings.TrimSpace(req.Socials[i].URL)
		if req.Socials[i].Platform == "" || len(req.Socials[i].Platform) > 30 {
			http.Error(w, "each social needs platform (1-30 chars)", http.StatusBadRequest)
			return
		}
		if !validHTTPURL(req.Socials[i].URL) || req.Socials[i].URL == "" {
			http.Error(w, "each social needs valid http(s) url", http.StatusBadRequest)
			return
		}
	}
	if req.Socials == nil {
		req.Socials = []db.SocialLink{}
	}
	socialsJSON, err := json.Marshal(req.Socials)
	if err != nil {
		http.Error(w, "Invalid socials", http.StatusBadRequest)
		return
	}

	if err := h.Store.UpdateCreatorProfile(*creatorID, req.DisplayName, req.Bio, req.AvatarURL, string(socialsJSON), req.Theme); err != nil {
		h.Logger.Printf("UpdateCreatorProfile failed: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Echo input tervalidasi (bukan baca ulang — lihat LEARN di atas).
	// Baca username via PRIMARY: data milik sendiri, konsisten dgn GET.
	creator, err := h.Store.GetCreatorByIDPrimary(*creatorID)
	username := ""
	if err == nil {
		username = creator.Username
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"username":     username,
		"display_name": req.DisplayName,
		"bio":          req.Bio,
		"avatar_url":   nullableString(req.AvatarURL),
		"theme":        req.Theme,
		"socials":      req.Socials,
	})
}

// LEARN: Kenapa magic bytes, bukan cuma ekstensi/Content-Type?
// ----------
// Klien BISA berbohong soal Content-Type (header HTTP) dan nama file.
// Contoh: upload file PHP berbahaya tapi beri nama "photo.jpg" + kirim
// Content-Type: image/jpeg. Kalau server hanya cek ekstensi atau header,
// file berbahaya itu lolos dan tersimpan di disk. Magic bytes (file
// signature) adalah urutan byte PERTAMA yang ditulis oleh software pembuat
// format gambar — JPEG selalu diawali FF D8 FF, PNG selalu 89 50 4E 47
// dst. Urutan ini tidak bisa dipalsukan tanpa merusak file gambar itu
// sendiri. Validasi ini dilakukan 100% server-side; klien tidak punya
// kontrol sedikitpun atas pengecekan ini.
// ----------
// HandleUploadAvatar menerima upload gambar profil (POST /api/profile/avatar,
// multipart/form-data, field "avatar"). Validasi: magic bytes (jpg/png/webp),
// max 2MB. Simpan ke uploads/avatars/{id}.{ext}, update avatar_url di DB,
// return JSON {"avatar_url": "..."}.
func (h *Handler) HandleUploadAvatar(w http.ResponseWriter, r *http.Request) {
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		http.Error(w, "Login required", http.StatusUnauthorized)
		return
	}

	// Batas 3MB (2MB file + headroom untuk multipart overhead).
	if err := r.ParseMultipartForm(3 << 20); err != nil {
		http.Error(w, "File too large (max 2MB)", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		http.Error(w, "Missing avatar file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Baca 12 byte pertama untuk magic bytes check.
	buf := make([]byte, 12)
	n, _ := io.ReadFull(file, buf)
	if n < 4 {
		http.Error(w, "File too small to be an image", http.StatusBadRequest)
		return
	}

	var ext string
	switch {
	case bytes.HasPrefix(buf, []byte{0xFF, 0xD8, 0xFF}):
		ext = "jpg"
	case bytes.HasPrefix(buf, []byte{0x89, 0x50, 0x4E, 0x47}):
		ext = "png"
	case n >= 12 && string(buf[0:4]) == "RIFF" && string(buf[8:12]) == "WEBP":
		ext = "webp"
	default:
		http.Error(w, "Only JPG, PNG, and WebP images are accepted", http.StatusBadRequest)
		return
	}

	// Ukuran 2MB — header.Size di-set oleh multipart parser (bukan klien),
	// tapi tetap validasi berdasarkan isi aktual untuk jaga-jaga.
	if header.Size > 2*1024*1024 {
		http.Error(w, "File too large (max 2MB)", http.StatusBadRequest)
		return
	}

	// Rewind — kita sudah konsumsi 12 byte untuk magic check.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Path: /uploads/avatars/{creator_id}.{ext}
	avatarPath := fmt.Sprintf("/uploads/avatars/%d.%s", *creatorID, ext)
	diskPath := filepath.Join("uploads", "avatars", fmt.Sprintf("%d.%s", *creatorID, ext))

	if err := os.MkdirAll(filepath.Dir(diskPath), 0755); err != nil {
		h.Logger.Printf("MkdirAll: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	dst, err := os.Create(diskPath)
	if err != nil {
		h.Logger.Printf("Create avatar file: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		h.Logger.Printf("Copy avatar: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Read current profile lalu merge avatar_url — tidak bisa pakai
	// UpdateCreatorProfile langsung karena itu full-replace. Baca dari PRIMARY
	// supaya merge tidak menulis ulang nilai basi dari replica (e.g. socials)
	// ke primary saat user mengunggah avatar (read-your-own-writes).
	creator, err := h.Store.GetCreatorByIDPrimary(*creatorID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	bio := ""
	if creator.Bio.Valid {
		bio = creator.Bio.String
	}
	socials := creator.Socials.String
	if socials == "" {
		socials = "[]"
	}

	if err := h.Store.UpdateCreatorProfile(*creatorID, creator.DisplayName, bio, avatarPath, socials, creator.Theme); err != nil {
		h.Logger.Printf("UpdateCreatorProfile avatar: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"avatar_url": avatarPath,
	})
}

// validThemePreset reports whether the value is one of the four closed
// preset themes (classic|darkroom|coral|glass). Empty string is
// INVALID on write — a missing theme in an old client would silently reset
// another theme to classic; the frontend always sends the full preset list.
// Preset glass adalah EKSPERIMEN (preset 4) — diputuskan nanti mau
// dipertahankan permanen atau dibuang; hapus dari sini kalau dibuang.
func validThemePreset(t string) bool {
	switch t {
	case "classic", "darkroom", "coral", "glass":
		return true
	}
	return false
}

// validTheme coerces any non-preset value to "classic" for output. Read path
// (bukan write) — payload JSON tidak boleh membawa nilai theme yang tidak
// dikenal frontend. Legacy "night" (nama lama preset darkroom) di-mapping ke
// "darkroom" supaya baris yang ditulis sebelum rebranding tidak jatuh ke
// classic.
func validTheme(t string) string {
	if t == "night" {
		return "darkroom"
	}
	if validThemePreset(t) {
		return t
	}
	return "classic"
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
