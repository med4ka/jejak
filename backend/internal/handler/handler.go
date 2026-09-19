package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"

	"jejak/internal/auth"
	"jejak/internal/cache"
	"jejak/internal/ratelimit"
	"jejak/internal/db"
)

// LEARN:
//
//	Kenapa: Struct ini menggabungkan store (sharded DB), Redis client (queue),
//	dan logger untuk operasi handler lengkap. Konsep design: memisahkan concerns
//	(storage, async processing, logging) agar setiap komponen bisa dikembang-
//	secara terpisah tanpa mengganggu yang lainnya.
//	Trade-off: Menambah jumlah parameter di NewHandler, tapi memungkinkan pengujian
//	unit yang terisolasi dan penggantian implementasi (misal: swap Redis dengan Kafka)
//	Alternatif: Bisa pakai struct options pattern, tapi LEARN lebih eksplisit tentang
//	dependency-nya untuk pembelajaran ini.
//	Catatan mode: Cache=nil berarti cache-aside mati (baseline); Redis=nil atau
//	Async=false berarti click logging sinkron langsung ke DB (baseline).
type Handler struct {
	Store  db.ShardStore
	Redis  *redis.Client
	Cache  cache.Cache
	Async  bool
	Logger *log.Logger
	// Auth is set by main after NewHandler (keeps constructor stable).
	// Nil means auth endpoints are disabled (fail safe: middleware.RequireAuth
	// rejects everyone, matching the old authedCreator fail-fast behavior).
	Auth auth.Store
	// LoginLimiter throttles POST /api/login (Fase 7). Set by main; nil
	// means no throttling (defensive fallback, keeps unit-test setup simple).
	LoginLimiter *ratelimit.Limiter
	// APILimiter throttles POST /api/v1/shorten per API key (100 req/menit
	// per key — lebih longgar dari login rate limit). Set by main; nil =
	// no throttling (tests, baseline mode).
	APILimiter *ratelimit.Limiter

	// uniqueMu guards uniqueSeen — in-memory dedup fallback (Fase 14) yang
	// dipakai ketika Redis == nil (baseline single-process). isUniqueClick
	// menulis di sini dgn window 24h supaya klik unik TETAP terisi akurat di
	// baseline — bukan cuma di mode async (lihat LEARN di logClickAsync).
	// Trade-off: State hidup di memory proses ini saja (tidak shared antar
	// instance). Cocok untuk baseline single-worker; produksi memakai Redis
	// SETNX yang shared + persisten (lihat isUniqueClick).
	uniqueMu   sync.Mutex
	uniqueSeen map[string]int64
}

// LEARN:
//
//	Kenapa: Fungsi ini membuat handler baru dengan dependensi yang dibutuhkan.
//	Konsep design: Dependency Injection - memasukkan dependensi dari luar
//	rather than membuat di dalam fungsi, memudahkan pengujian dan penggantian.
//	Trade-off: Caller harus instantiate dependensi terlebih dahulu, tapi kode jadi
//	lebih bersih dan testable. Jika lupa inject satu dependensi, compile error akan
//	mengejutkan pembaca kode.
//	Alternatif: Bisa pakai func options atau default values, tapi kurang transparan.
func NewHandler(s db.ShardStore, r *redis.Client, c cache.Cache, async bool) *Handler {
	return &Handler{
		Store:  s,
		Redis:  r,
		Cache:  c,
		Async:  async,
		Logger: log.Default(),
	}
}

// creatorProfileJSON shapes the public profile payload. Password hash is
// NEVER included. Nullable columns become "" / null / [] (never SQL artifacts).
// Theme falls back to "classic" for rows written before the theme column
// existed (DB default covers new rows; this covers in-progress deployments).
func creatorProfileJSON(c db.Creator, links []db.Link) map[string]any {
	bio := ""
	if c.Bio.Valid {
		bio = c.Bio.String
	}
	var avatar any
	if c.AvatarURL.Valid && c.AvatarURL.String != "" {
		avatar = c.AvatarURL.String
	}
	socials := parseSocials(c.Socials)
	return map[string]any{
		"username":     c.Username,
		"display_name": c.DisplayName,
		"bio":          bio,
		"avatar_url":   avatar,
		"theme":        validTheme(c.Theme),
		"socials":      socials,
		"links":        links,
	}
}

// parseSocials decodes the JSONB document; corrupt/empty becomes [] (fail
// soft: 1 baris rusak tidak boleh merobohkan seluruh halaman profil).
func parseSocials(raw sql.NullString) []db.SocialLink {
	out := []db.SocialLink{}
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return out
	}
	var parsed []db.SocialLink
	if err := json.Unmarshal([]byte(raw.String), &parsed); err != nil {
		return out
	}
	if parsed == nil {
		return out
	}
	return parsed
}

// validHTTPURL allows empty (field opsional), http(s) URL with host, atau
// path lokal "/uploads/*" (avatar hasil upload server-side).
func validHTTPURL(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	if strings.HasPrefix(s, "/uploads/") {
		return true
	}
	return validRemoteURL(s)
}

// validRemoteURL membutuhkan URL yang memenuhi syarat MIRIP url.Parse:
// scheme http(s) + host terisi. Tidak mengizinkan empty atau "/uploads/*"
// — untuk field yang WAJIB mengarah ke resource INTERNET (original_url,
// device_rules Smart Link), bukan aset lokal.
func validRemoteURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// maxJSONBodyBytes is the cap for JSON request bodies. Body lewat batas akan
// gagal di-decode (http.MaxBytesReader) — mencegah request raksasa menghabiskan
// RAM decoder. Nilai 1MB jauh di atas payload normal (shorten/profil/login).
const maxJSONBodyBytes = 1 << 20

// decodeJSON decodes a JSON body with a hard size cap. LEARN:
//
//	Kenapa: json.Decoder membaca seluruh body tanpa batas default. DoS sederhana:
//	kirim POST 500MB JSON -> decoder alokasi memori sebanyak itu. http.MaxBytesReader
//	membatasi baca; decode error "request body too large" -> 400 di sini (bukan 413,
//	biar client tidak menganggap server crash).
//	Trade-off: Client yang sah tidak pernah dekat batas ini; error 400 untuk over-limit
//	agak "bohong" (seharusnya 413) tapi menyederhanakan satu jalur error di semua
//	handler. Alternatif: middleware pembatas body global, tapi middleware tidak bisa
//	mengembalikan error JSON yang konsisten per-endpoint.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return errors.New("invalid request body")
	}
	return nil
}
