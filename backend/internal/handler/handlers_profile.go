package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"jejak/internal/apierror"
	"jejak/internal/db"
	"jejak/internal/middleware"
)

// HandleGetMyProfile returns the logged-in creator's profile plus link list
// (GET /api/profile, session auth). 401 without a session, 404 when the
// account no longer exists, 500 on database errors. The reads go to the
// primary on purpose (read-your-own-writes, see below), not the replica.
func (h *Handler) HandleGetMyProfile(w http.ResponseWriter, r *http.Request) {
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		apierror.WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Login required")
		return
	}

	// Read from PRIMARY, not the replica: this is the caller's own data, so
	// changes must become visible immediately without waiting for manual
	// replica sync (read-your-own-writes). The public page
	// /api/u/{username} still reads from the replica (HandleCreatorLinks).
	creator, err := h.Store.GetCreatorByIDPrimary(*creatorID)
	if err != nil {
		apierror.WriteError(w, http.StatusNotFound, "PROFILE_NOT_FOUND", "Creator not found")
		return
	}

	links, err := h.Store.ListLinksByCreatorPrimary(creator.ID)
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

// HandleUpdateMyProfile replaces the entire profile (display_name, bio,
// avatar, socials, theme) on every save rather than patching single fields.
// For a form this small full-replace is simpler: no half-way merge, a single
// validation site, and the response can echo the validated input without a
// database re-read: a re-read would hit the replica and return stale data
// (the read-your-write trap). Validation limits (name 1-100, bio <= 500,
// <= 10 socials, http(s) URLs) are explicit MVP choices, not final product
// rules. Trade-off: the client must always send every field (a forgotten
// field is reset) and the username stays immutable, since changing it would
// change the public URL and force a new uniqueness check. A per-field JSON
// merge PATCH was rejected because "absent versus empty" handling is easy
// to get wrong.
func (h *Handler) HandleUpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		apierror.WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Login required")
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
		apierror.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
		return
	}

	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.Bio = strings.TrimSpace(req.Bio)
	req.AvatarURL = strings.TrimSpace(req.AvatarURL)
	if req.DisplayName == "" || len(req.DisplayName) > 100 {
		apierror.WriteError(w, http.StatusBadRequest, "PROFILE_DISPLAY_NAME_REQUIRED", "display_name required (1-100 chars)")
		return
	}
	if len(req.Bio) > 500 {
		apierror.WriteError(w, http.StatusBadRequest, "PROFILE_BIO_TOO_LONG", "bio max 500 chars")
		return
	}
	if !validHTTPURL(req.AvatarURL) {
		apierror.WriteError(w, http.StatusBadRequest, "PROFILE_AVATAR_URL_INVALID", "avatar_url must be empty or http(s) URL")
		return
	}
	// Themes come from a closed preset set
	// (classic|darkroom|coral|glass|risoPrint|peach|lavender|matcha|sakura|
	// ocean|sunset), never a free-form color; extend that set when
	// experimenting with new presets. The "glass" preset is deliberately
	// EXPERIMENTAL so it can be dropped again without a migration. Dirty
	// data (e.g. a manually updated DB column) is coerced at the payload
	// layer (validTheme), so GET never leaks an unknown value to the frontend.
	if !validThemePreset(req.Theme) {
		apierror.WriteError(w, http.StatusBadRequest, "PROFILE_INVALID_THEME", "theme must be one of: classic, darkroom, coral, glass, risoPrint, peach, lavender, matcha, sakura, ocean, sunset")
		return
	}
	if len(req.Socials) > 10 {
		apierror.WriteError(w, http.StatusBadRequest, "PROFILE_SOCIALS_LIMIT", "max 10 social links")
		return
	}
	for i := range req.Socials {
		req.Socials[i].Platform = strings.TrimSpace(req.Socials[i].Platform)
		req.Socials[i].URL = strings.TrimSpace(req.Socials[i].URL)
		if req.Socials[i].Platform == "" || len(req.Socials[i].Platform) > 30 {
			apierror.WriteError(w, http.StatusBadRequest, "PROFILE_SOCIAL_PLATFORM_REQUIRED", "each social needs platform (1-30 chars)")
			return
		}
		if !validHTTPURL(req.Socials[i].URL) || req.Socials[i].URL == "" {
			apierror.WriteError(w, http.StatusBadRequest, "PROFILE_SOCIAL_URL_INVALID", "each social needs valid http(s) url")
			return
		}
	}
	if req.Socials == nil {
		req.Socials = []db.SocialLink{}
	}
	socialsJSON, err := json.Marshal(req.Socials)
	if err != nil {
		apierror.WriteError(w, http.StatusBadRequest, "PROFILE_SOCIALS_INVALID", "Invalid socials")
		return
	}

	if err := h.Store.UpdateCreatorProfile(*creatorID, req.DisplayName, req.Bio, req.AvatarURL, string(socialsJSON), req.Theme); err != nil {
		h.Logger.Printf("UpdateCreatorProfile failed: %v", err)
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
		return
	}

	// Echo the validated input instead of re-reading it: a re-read could come
	// back stale from the replica. The username is read via PRIMARY (own
	// data, consistent with GET).
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

// HandleUploadAvatar serves POST /api/profile/avatar: it validates and stores
// the profile image from the multipart "avatar" field. Detection uses magic
// bytes rather than the file extension or Content-Type because both are
// client-supplied and falsifiable: a malicious PHP file named "photo.jpg"
// and labeled image/jpeg would pass either check and be written to disk.
// Magic bytes are the first sequence emitted by the format's own writer
// (JPEG starts FF D8 FF, PNG starts 89 50 4E 47, WebP carries
// RIFF....WEBP) and cannot be forged without corrupting the image itself;
// the check runs entirely server-side, so the client has no influence over it.
func (h *Handler) HandleUploadAvatar(w http.ResponseWriter, r *http.Request) {
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		apierror.WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Login required")
		return
	}

	// 3 MB limit: the 2 MB file cap plus headroom for multipart overhead.
	// MaxBytesReader bounds the WHOLE body before the parser buffers it:
	// ParseMultipartForm's argument only decides when parts spill to disk,
	// it does NOT cap the request, so without this a multi-GB upload would
	// fill the temp directory (G120). An over-limit body surfaces as 413
	// here (image endpoint: the oversized object IS the body, unlike the
	// JSON handlers where decodeJSON deliberately maps it to 400).
	r.Body = http.MaxBytesReader(w, r.Body, 3<<20)
	// #nosec G120 -- the request body is already capped at 3 MB by
	// MaxBytesReader above, so ParseMultipartForm cannot be fed an
	// unbounded stream (the rule's concern); its argument only bounds
	// in-memory buffering before parts spill to disk.
	if err := r.ParseMultipartForm(3 << 20); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			apierror.WriteError(w, http.StatusRequestEntityTooLarge, "AVATAR_TOO_LARGE", "File too large (max 2MB)")
			return
		}
		apierror.WriteError(w, http.StatusBadRequest, "AVATAR_TOO_LARGE", "File too large (max 2MB)")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		apierror.WriteError(w, http.StatusBadRequest, "AVATAR_MISSING", "Missing avatar file")
		return
	}
	defer file.Close()

	// 12 bytes cover every signature checked below, including the WebP
	// markers at offsets 0 and 8.
	buf := make([]byte, 12)
	n, _ := io.ReadFull(file, buf)
	if n < 4 {
		apierror.WriteError(w, http.StatusBadRequest, "AVATAR_TOO_SMALL", "File too small to be an image")
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
		apierror.WriteError(w, http.StatusBadRequest, "AVATAR_BAD_TYPE", "Only JPG, PNG, and WebP images are accepted")
		return
	}

	// 2 MB cap: header.Size is set by the multipart parser rather than by
	// the client, but the actual contents are still checked as a precaution.
	if header.Size > 2*1024*1024 {
		apierror.WriteError(w, http.StatusBadRequest, "AVATAR_TOO_LARGE", "File too large (max 2MB)")
		return
	}

	// Rewind: 12 bytes were already consumed by the magic-byte check.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		apierror.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal error")
		return
	}

	avatarPath, diskPath := newAvatarPaths(*creatorID, ext)

	// #nosec G703,G304 -- diskPath is built from the session's int64
	// creatorID (formatted %d, never a raw string) plus an extension chosen
	// from the magic-byte allowlist: no user-controlled path segment reaches
	// either call, so the taint that flags path traversal cannot be
	// exercised. 0750: the owner (and web server group) write; group/other
	// get no write access to the avatar store (G301).
	if err := os.MkdirAll(filepath.Dir(diskPath), 0750); err != nil {
		h.Logger.Printf("MkdirAll: %v", err)
		apierror.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal error")
		return
	}

	// #nosec G703,G304 -- same derivation as MkdirAll above (int64
	// creatorID + allowlisted extension), not a user-supplied path.
	dst, err := os.Create(diskPath)
	if err != nil {
		h.Logger.Printf("Create avatar file: %v", err)
		apierror.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal error")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		h.Logger.Printf("Copy avatar: %v", err)
		apierror.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal error")
		return
	}

	// Read the current profile and merge in avatar_url: UpdateCreatorProfile
	// cannot be used directly because it performs a full replace. The read
	// goes to PRIMARY so the merge does not write stale replica values (e.g.
	// socials) back to the primary row (read-your-own-writes).
	creator, err := h.Store.GetCreatorByIDPrimary(*creatorID)
	if err != nil {
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
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
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
		return
	}

	// Best effort: the row now points at the new file, so every other file
	// with this creator's id prefix (the previous versioned name and the
	// legacy stable "{id}.{ext}" name) is unreferenced. Failures are
	// ignored: an orphan file costs disk space, a failed delete must not
	// fail an otherwise successful upload.
	pruneOldAvatars(*creatorID, filepath.Base(diskPath))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"avatar_url": avatarPath,
	})
}

// validThemePreset reports whether the value is one of the eleven closed
// preset themes (classic|darkroom|coral|glass|risoPrint|peach|lavender|
// matcha|sakura|ocean|sunset). Empty is INVALID on write: a client that
// omits the theme would silently reset an existing theme to classic, so the
// frontend always sends one of the presets. The glass preset is an
// EXPERIMENT (preset 4): decide later whether to keep or drop it; remove it
// here if it is dropped. Preset risoPrint (preset 5, risograph 2026-09-29),
// presets 6-9 (peach/lavender/matcha/sakura, playful 2026-09-30) and
// presets 10-11 (ocean/sunset, 2026-10-01) follow the same pattern: add or
// remove the value here plus THEMES in lib/themes.js.
func validThemePreset(t string) bool {
	switch t {
	case "classic", "darkroom", "coral", "glass", "risoPrint",
		"peach", "lavender", "matcha", "sakura", "ocean", "sunset":
		return true
	}
	return false
}

// validTheme coerces any non-preset value to "classic" for output. It runs
// on the read path, never on write: the JSON payload must not carry a theme
// the frontend does not know. Legacy "night" (the old name of the darkroom
// preset) maps to "darkroom" so rows written before the rebranding do not
// fall back to classic.
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

// newAvatarPaths derives the public URL and the disk path for an upload:
// "/uploads/avatars/{id}-{unixmillis}.{ext}". The version suffix is the fix
// for the stale-avatar bug: a stable "{id}.{ext}" name keeps the URL identical
// across re-uploads, so a browser that fetched the image earlier serves the
// OLD bytes from its HTTP cache (the static handler sends no Cache-Control,
// which means heuristic freshness from Last-Modified) and the new photo never
// appears after a refresh. A fresh URL forces a cache miss everywhere the
// avatar is rendered (dashboard, navbar, public page, OG image) without
// query-string hacks in the database.
func newAvatarPaths(creatorID int64, ext string) (publicURL, diskPath string) {
	name := fmt.Sprintf("%d-%d.%s", creatorID, time.Now().UnixMilli(), ext)
	return "/uploads/avatars/" + name, filepath.Join("uploads", "avatars", name)
}

// pruneOldAvatars removes this creator's earlier avatar files after a
// successful upload: versioned names ("{id}-*") and the legacy stable name
// ("{id}.{ext}"). Only files of the same id are touched, never another
// creator's. Errors are deliberately ignored (best effort, see caller).
func pruneOldAvatars(creatorID int64, keep string) {
	entries, err := os.ReadDir("uploads/avatars")
	if err != nil {
		return
	}
	versioned := fmt.Sprintf("%d-", creatorID)
	legacy := fmt.Sprintf("%d.", creatorID)
	for _, e := range entries {
		n := e.Name()
		if n == keep || e.IsDir() {
			continue
		}
		if strings.HasPrefix(n, versioned) || strings.HasPrefix(n, legacy) {
			_ = os.Remove(filepath.Join("uploads", "avatars", n))
		}
	}
}
