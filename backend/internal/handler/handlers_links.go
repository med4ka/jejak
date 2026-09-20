package handler

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"jejak/internal/db"
	"jejak/internal/middleware"
	"jejak/internal/ratelimit"
	"jejak/internal/shortener"
)

// LEARN:
//
//	Kenapa: Fungsi ini menangani POST /api/shorten untuk membuat URL pendek.
//	Konsep design: Write path - data baru ditulis ke sharded DB. Click counting
//	tidak dilakukan di sini agar redirect cepat (best-effort). Fokus hanya pada
//	create URL dan populate cache.
//	Trade-off: click_count diHandleShorten tidak diincrement, hanya didatabase
//	create saja. Artinya click_count di awal selalu 0 sampai worker proses
//	event dari queue. Jika butuh click count real-time, arsitektur berbeda.
//	Alternatif: Increment langsung di HandleShorten, tapi akan memperlambat
//	redirect response yang "seharusnya" cepat.
func (h *Handler) HandleShorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	h.doShorten(middleware.CreatorID(r), w, r)
}

// doShorten adalah core yang dipakai bersama oleh POST /api/shorten (auth
// session cookie) dan POST /api/v1/shorten (auth API key). Kepemilikan link
// datang melalui parameter creatorID: nil = link anonim, non-nil = milik
// kreator itu (id dari cookie / dari api_keys — sumber auth yang berbeda,
// hasil akhir sama: CreateURL(creator_id)).
func (h *Handler) doShorten(creatorID *int64, w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL  string   `json:"url"`
		Slug string   `json:"slug"`
		Tags []string `json:"tags"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// LEARN:
	//   Kenapa: original_url di-validate http(s) SEBELUM disimpan. Tanpa ini
	//   doShorten menerima string apa pun — termasuk "javascript:", "data:",
	//   "//evil.com" (scheme-relative), atau teks kosong — yang disimpan apa
	//   adanya dan ditulis ke header Location saat redirect. Browser modern
	//   memblokir javascript: di Location untuk navigasi top-level, tapi short
	//   link tetap menjadi gadget open-redirect/XSS yang tidak perlu. Reuse
	//   validRemoteURL (bukan validHTTPURL): field ini WAJIB menunjuk internet,
	//   jadi input kosong dan path lokal "/uploads/*" tidak boleh lolos.
	//   Trade-off: Memang truncate use-case "short link ke halaman internal",
	//   tapi itu bukan target produk. Device_rules + socials sudah memakai
	//   validHTTPURL yang sama sejak sebelum perbaikan ini.
	if !validRemoteURL(req.URL) {
		http.Error(w, "url must be a valid http(s) URL", http.StatusBadRequest)
		return
	}

	// LEARN:
	//   Kenapa: Slug custom opsional — kalau diisi dipakai sebagai short_code,
	//   kalau kosong fallback ke random seperti dulu. Urutan cek: format dulu
	//   (murah, tanpa I/O), reserved kedua (tanpa I/O), baru cek DB. Cek-then-
	//   insert punya race (2 request slug sama bersamaan lolos cek) — backstop-
	//   nya UNIQUE constraint: error 23505 dipetakan ke 409, BUKAN 500 generik,
	//   supaya client tahu artinya "keduluan", bukan "server rusak".
	//   Trade-off: 1 query SELECT ekstra per custom slug (random path tidak kena).
	//   Alternatif: Langsung INSERT dan hanya andalkan constraint (hemat 1 query,
	//   tapi pesan error reserved-word tidak bisa dibedakan dari tabrakan).
	shortCode := strings.TrimSpace(req.Slug)
	if shortCode == "" {
		var err error
		shortCode, err = shortener.GenerateShortCode(6)
		if err != nil {
			http.Error(w, "Failed to generate short code", http.StatusInternalServerError)
			return
		}
	} else {
		if !shortener.ValidSlug(shortCode) {
			http.Error(w, "Invalid slug (3-30 chars, letters/digits/_/-)", http.StatusBadRequest)
			return
		}
		if shortener.IsReserved(shortCode) {
			http.Error(w, "Slug is reserved", http.StatusBadRequest)
			return
		}
		if _, err := h.Store.GetURL(shortCode); err == nil {
			http.Error(w, "Slug already taken", http.StatusConflict)
			return
		} else if err != sql.ErrNoRows {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	// Normalize tags: trim, lowercase (biar "YouTube" == "youtube"), buang
	// yang kosong/duplikat. Client (form home) sudah pre-parse koma jadi
	// array, tapi server validasi ulang — jangan percaya input client.
	// Batas: maks 5 tag, tiap tag 1-20 char. Gagal validasi -> 400 jelas.
	tags, ok := normalizeTags(req.Tags)
	if !ok {
		http.Error(w, "Invalid tags (max 5 tags, 1-20 chars each)", http.StatusBadRequest)
		return
	}
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		http.Error(w, "Invalid tags", http.StatusBadRequest)
		return
	}

	// Store in DB. Logged-in creator auto-owns the link; anonymous (nil)
	// keeps Fase 0-8 behavior (creator_id NULL stays valid).
	if err := h.Store.CreateURL(shortCode, req.URL, creatorID, string(tagsJSON)); err != nil {
		// Log error ASLI ke terminal server: tanpa ini, semua kegagalan tulis
		// hanya terlihat sebagai "Failed to store URL" generik di client dan
		// akar masalahnya (constraint apa, kolom apa) tidak terlacak.
		// Pelajaran dari bug varchar(10) vs slug 30-char.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// Lost the check-then-insert race: someone took it first.
			h.Logger.Printf("CreateURL race lost for %q: %v", shortCode, err)
			http.Error(w, "Slug already taken", http.StatusConflict)
			return
		}
		h.Logger.Printf("CreateURL failed for %q: %v", shortCode, err)
		http.Error(w, "Failed to store URL", http.StatusInternalServerError)
		return
	}

	// Populate cache (write-through on create). Nil-safe: baseline skips this.
	// Stores the JSON redirect payload (device_rules kosong saat create) supaya
	// format cache konsisten dengan Smart Link di path redirect.
	if h.Cache != nil {
		if b, err := json.Marshal(redirectTarget{URL: req.URL}); err == nil {
			if err := h.Cache.Set(shortCode, string(b), 300); err != nil {
				h.Logger.Printf("Warning: failed to populate cache: %v", err)
			}
		}
	}

	// LEARN:
	//   Kenapa: Short URL dibentuk dari request yang masuk (r.Host + scheme),
	//   bukan hardcode host/port. Server yang sama bisa jalan di port beda
	//   (8081/8082), di belakang LB, atau di domain non-lokal — respons selalu
	//   menunjuk ke alamat yang benar-benar bisa dihubungi client.
	//   Trade-off: Percaya Host header dari client (bisa dipalsukan); untuk
	//   produk perlu whitelist domain. X-Forwarded-Proto dihormati supaya
	//   di belakang proxy TLS tidak keliru jadi http.
	//   Alternatif: Config BASE_URL statis per environment, tapi 1 binary
	//   tidak bisa dipakai ulang di port berbeda tanpa ubah config.
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
		scheme = p
	}
	shortURL := scheme + "://" + r.Host + "/r/" + shortCode
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

// LEARN:
//
//	Kenapa deteksi device pakai MATCHING KEYWORD SIMPLE, bukan library
//	User-Agent parsing yang berat (ps, ua-parser): kebutuhan kita cuma 2 case
//	— iOS (iPhone/iPad) vs Android. Dua substring sudah memisahkan ~semua
//	device nyata untuk use case link-in-bio; UA string alias 'Mobile' sering
//	muncul di tablet/desktop, jadi 'iPhone'/'iPad'/'Android' lebih tepat
//	daripada sekadar 'Mobile'.
//	Trade-off: Tidak presisi untuk edge cases (iPadOS desktop-mode UA masih
//	menyebut 'Macintosh' + 'Mobile', Android versi lawas tanpa kata 'Android'
//	di UA WebView, bot/crawler yang menyamar). Konsekuensinya: sebagian kecil
//	request fallback ke original_url — acceptable, karena itu persis perilaku
//	default yang sudah ada (backward compatible, tidak ada link yang rusak).
//	Library UA parser penuh = akurasi >99% tapi +dependensi berat +CPU per
//	redirect (2-3x lipat untuk 2 keyword ini) +rentan ke decoder quirk.
//	aturan: kalau rule device terisi dan non-kosong, pakai itu; selain itu
//	original_url seperti biasa.
func detectDevice(ua string) string {
	if strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad") {
		return "ios"
	}
	if strings.Contains(ua, "Android") {
		return "android"
	}
	return ""
}

// redirectTarget is what the cache stores for a short code: the default URL
// plus the Smart Link device rules. JSON (not a bare string) so the redirect
// path can route per device WITHOUT an extra DB read on a cache hit.
// Disabled di-omitempty (absent = aktif): server ini TIDAK pernah menulis
// target disabled (link yang di-disable short-circuit ke 410 SEBELUM cache),
// tapi field ini ada supaya entri cache ber-nilai disabled yang ditulis
// writer lain / tool eksternal tetap dihormati (tidak redirect) — dan juga
// forward-compat untuk is_active di payload cache. Backward-compat: entri
// lama tanpa field ini otomatis = aktif.
type redirectTarget struct {
	URL         string            `json:"url"`
	DeviceRules map[string]string `json:"device_rules,omitempty"`
	Disabled    bool              `json:"disabled,omitempty"`
}

// parseTarget reads a cache value back into a redirectTarget. Old cache
// entries written before Smart Link stored a bare URL — those parse-fail and
// the caller falls back to treating the string as the URL (backward compat).
func parseTarget(s string) (redirectTarget, bool) {
	var t redirectTarget
	if err := json.Unmarshal([]byte(s), &t); err != nil || t.URL == "" {
		return redirectTarget{}, false
	}
	return t, true
}

// pickURL applies device rules: a matched key with a non-empty value wins;
// otherwise (no rules, no match, empty value) fall back to the default URL.
func (t redirectTarget) pickURL(device string) string {
	if v := strings.TrimSpace(t.DeviceRules[device]); v != "" {
		return v
	}
	return t.URL
}

// LEARN:
//
//	Kenapa: Fungsi ini menangani GET /r/{shortCode} untuk redirect.
//	Konsep design: Read path + async decoupling. Redirect seharusnya tidak boleh
//	menunggu logging selesai (fire-and-forget), karena user menunggu redirect,
//	bukan menunggu proses logging selesika. Ini adalah prinsip fundamental Fase 6:
//	"redirect gak boleh nunggu logging selesai"
//	Trade-off: Click count bisa sedikit delayed (belum langsung di DB) karena diproses
//	oleh worker di latar belakang. Risiko: kalau worker crash, click count bisa hilang
//	atau tidak sepenuhnya catat, tapi user experience redirect tetap cepat.
//	Alternatif: Sync logging di mana redirect menunggu DB update, tapi akan merusak
//	performa redirect dan mengalahkan tujuan Fase 6 (decoupling read/write path).
//	Smart Link: target URL dipilih per device (detectDevice) setelah membaca
//	device_rules — cache menyimpan {url, device_rules} sehingga cache hit tetap
//	bisa route per device tanpa query DB tambahan.
func (h *Handler) HandleRedirect(shortCode string, w http.ResponseWriter, r *http.Request) {
	device := detectDevice(r.UserAgent())

	// Cache-aside: cek cache dulu (nil-safe, baseline langsung ke DB).
	if h.Cache != nil {
		if cached, found := h.Cache.Get(shortCode); found {
			if target, ok := parseTarget(cached); ok {
				// Entri cache yang menandai link disabled TIDAK boleh redirect.
				// Evict supaya request berikutnya baca ulang DB — link yang
				// di-hidupkan kembali langsung aktif, tidak nyangkut TTL 300s.
				if target.Disabled {
					h.Cache.Delete(shortCode)
					http.Error(w, "Link disabled", http.StatusGone)
					return
				}
				h.logClickAsync(shortCode, r)
				http.Redirect(w, r, target.pickURL(device), http.StatusFound)
				return
			}
			// Entry lama (URL polos, sebelum Smart Link): tetap jalan.
			h.logClickAsync(shortCode, r)
			http.Redirect(w, r, cached, http.StatusFound)
			return
		}
	}

	// Read from sharded store (original_url + device_rules untuk routing).
	link, err := h.Store.GetLink(shortCode)
	if err != nil {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	// Fase 13 lifecycle: link yang di-disable (is_active=false) tidak boleh
	// redirect — jawab 410 Gone supaya link "hilang" dari publik tapi tetap
	// terlihat di dashboard pemilik (yang bisa menyalakannya lagi). Dicek
	// SEBELUM menulis cache: link disabled tidak pernah masuk cache sebagai
	// target aktif, jadi tallied click tidak bisa masuk lewat jalur cache.
	if !link.IsActive {
		http.Error(w, "Link disabled", http.StatusGone)
		return
	}

	target := redirectTarget{URL: link.OriginalURL, DeviceRules: link.DeviceRules}

	// Populate cache for future requests (nil-safe).
	if h.Cache != nil {
		if b, err := json.Marshal(target); err == nil {
			if err := h.Cache.Set(shortCode, string(b), 300); err != nil {
				h.Logger.Printf("Warning: failed to populate cache: %v", err)
			}
		}
	}

	// Async: fire-and-forget click logging to queue
	// The redirect response is not blocked by this operation
	h.logClickAsync(shortCode, r)

	http.Redirect(w, r, target.pickURL(device), http.StatusFound)
}

// LEARN:
//
//	Kenapa: Fungsi ini push click event ke Redis List queue secara async (fire-and-forget).
//	Konsep design: Async queue (Redis List) - memisahkan write path dari read path.
//	Prinsip utama: redirect response harus segera selesai, logging ditangani secara
//	background. Menggunakan LPush ke queue "click_events", worker akan LPop dan
//	proses (update DB) secara berkala.
//	Trade-off: Ada risiko data click hilang jika worker crash sebelum memproses,
//	tapi total request tidak terblock. Data mungkin sedikit delayed (eventual consistency).
//	Alternatif: Bisa pakai Redis Stream dengan consumer group untuk guarantee lebih kuat,
//	tapi lebih kompleks bagi fase pengajaran ini.
func (h *Handler) logClickAsync(shortCode string, r *http.Request) {
	// Baseline mode (Async=false atau Redis=nil): tulis sinkron langsung ke DB,
	// dengan CLICK EVENT lengkap (referrer_domain + is_unique + clicked_at)
	// supaya jalur sinkron menghasilkan data identik dengan worker (satu jalur
	// persist, dua mode penulisan). Ini Fase 14: unique click + referrer domain
	// breakdown HARUS terisi juga di baseline, bukan cuma di mode async.
	if !h.Async || h.Redis == nil {
		ev := h.buildClickEvent(shortCode, r) // necesita deteksi unique + fingerprint
		if err := h.Store.LogClick(ev); err != nil {
			h.Logger.Printf("Warning: failed to log click: %v", err)
		}
		return
	}

	// Create click event message — field diisi LENGKAP supaya worker tidak
	// perlu menebak/ulang-memparsing (fire-and-forget yang tetap akurat).
	event := map[string]string{
		"short_code":      shortCode,
		"referrer":        r.Referer(),
		"referrer_domain": referrerDomain(r.Referer()),
		"is_unique":       strconv.FormatBool(h.isUniqueClick(shortCode, r)),
		"clicked_at":      time.Now().UTC().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		h.Logger.Printf("Warning: failed to marshal click event: %v", err)
		return
	}

	// Push to Redis list (queue) - use Redis client directly
	// LPush signature: LPush(ctx context.Context, key string, values ...interface{}) *SliceCmd
	if err := h.Redis.LPush(context.Background(), "click_events", jsonData).Err(); err != nil {
		h.Logger.Printf("Warning: failed to push click event to queue: %v", err)
		// Don't return error - this is async, redirect should not be affected
	}
}

// referrerDomain extracts the bare host (WITHOUT scheme + path) from a
// Referer header. Alasan: breakdown analitik "per-domain" harus punya satu
// bentuk konsisten — "https://google.com/", "google.com", "http://google.com"
// semuanya harus masuk bucket yang sama, bukan 3 bucket terpisah.
// Trade-off: Non-URL referrer (misal "android-app://x") dikembalikan apa
// adanya via callfresh parse; corrupt dianggap kosong (referrer_domain NULL
// di DB, tidak menyumbang breakdown).
func referrerDomain(ref string) string {
	if strings.TrimSpace(ref) == "" {
		return ""
	}
	u, err := url.Parse(ref)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Host
}

// clickFingerprint produces a stable-per-visitor hash so we can tell "adakah
// orang yang sama ini sudah klik link sama dalam 24 jam?" — dasar is_unique.
// Fingerprint = client IP + User-Agent (dua sinyal yang paling membedakan
// visitor di level HTTP yang kita punya). Trade-off: hash berarti kita tidak
// menyimpan IP mentah di mana pun (privasi), tapi dua orang di NAT/UA yang
// identik bisa salah dianggap "satu orang". Bit.ly punya masalah yang sama.
func (h *Handler) clickFingerprint(r *http.Request) string {
	sum := sha256.Sum256([]byte(ratelimit.ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For")) + "|" + r.UserAgent()))
	return hex.EncodeToString(sum[:])
}

// isUniqueClick says whether THIS visitor (per fingerprint) is new to THIS
// shortCode within the last 24h (bit.ly-style unique window).
//
//	Mode Redis set (produksi/async): SETNX atomik key = unique.
//
//
//	Mode Redis nil (baseline): pakai map in-memory (uniqueSeen) yang dijaga
//	Handler. Ini mode Fase 14 baseline: unik tetap terhitung akurat dalam
//	proses single-worker (bukan cuma di async). Trade-off: state hidup di
//	memory proses ini saja — tidak ter-distribusi antar instance; bila proses
//	restart, window unik "hangat kembali". Pendekatan ini cocok untuk hvandelt
//	single-process baseline; produksi memakai Redis SETNX yang shared +
//	persisten. Lihat juga LEARN di logClickAsync & comment di handler.go.
func (h *Handler) isUniqueClick(shortCode string, r *http.Request) bool {
	key := "click:unique:" + shortCode + ":" + h.clickFingerprint(r)
	if h.Redis != nil {
		// SETNX returns true only when the key did NOT exist (i.e. first time
		// THIS fingerprint shows up). TTL 24h = the dedup window; after that
		// the same visitor counts as a NEW unique click (exactly bit.ly).
		n, err := h.Redis.SetNX(context.Background(), key, "1", 24*time.Hour).Result()
		if err != nil {
			h.Logger.Printf("Warning: unique-click check failed, treating as unique: %v", err)
			return true
		}
		return n
	}

	// Baseline (Redis == nil): in-memory 24h window synced by uniqueMu.
	h.uniqueMu.Lock()
	defer h.uniqueMu.Unlock()
	if h.uniqueSeen == nil {
		h.uniqueSeen = make(map[string]int64)
	}
	now := time.Now().Unix()
	last, seen := h.uniqueSeen[key]
	if seen && now-last < int64(24*time.Hour/time.Second) {
		return false // same visitor within window → not a new unique click
	}
	h.uniqueSeen[key] = now
	return true
}

// buildClickEvent assembles the FULL ClickEvent so LogClick (baseline sync)
// and the async worker both persist the SAME complete metadata: referrer +
// referrer_domain (breakdown analitik), is_unique (window 24h), dan
// clicked_at = waktu klik ASLI (bukan waktu proses). Satu jalur persist,
// dua mode penulisan — lihat LEARN logClickAsync.
func (h *Handler) buildClickEvent(shortCode string, r *http.Request) db.ClickEvent {
	ev := db.ClickEvent{
		ShortCode:      shortCode,
		Referrer:       r.Referer(),
		ReferrerDomain: referrerDomain(r.Referer()),
		IsUnique:       h.isUniqueClick(shortCode, r),
		ClickedAt:      time.Now().UTC(),
	}
	return ev
}

// LEARN:
//
//	Kenapa: Fungsi ini membaca click count dari DB sharded untuk analytics.
//	Konsep design: Read path dari sharded DB. Karena click logging async via queue,
//	click count di DB mungkin belum terupdate sepenuhnya (eventual consistency).
//	Fungsi ini query langsung dari tabel urls click_count counter yang diupdate
//	oleh worker. Pengguna harus sadar bahwa angka bisa sedikit "basah" (not yet
//	updated) tergantung waktu proses worker.
//	Trade-off: Response analytics langsung dari DB, tidak butuh menunggu worker.
//	Tapi angka bisa sedikit salah (under-counted) jika worker belum sempat proses.
//	Alternatif: Bisa pakai separate counter di Redis yang diincrement sync saat
//	logClickAsync, tapi dua source of truth (DB + Redis) harus disinkronkan, kompleks.
func (h *Handler) HandleGetClickCount(shortCode string, w http.ResponseWriter, r *http.Request) {
	// Get shard DB for this shortCode
	shardDB := h.Store.GetShard(shortCode)
	if shardDB == nil {
		http.Error(w, "Database not available", http.StatusInternalServerError)
		return
	}

	var clickCount int64
	err := shardDB.QueryRow("SELECT click_count FROM urls WHERE short_code = $1", shortCode).Scan(&clickCount)
	if err == sql.ErrNoRows {
		w.Write([]byte(`{"click_count": 0}`))
		return
	}
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"click_count": clickCount})
}

// HandleListLinks serves GET /api/links for the loggged-in creator: returns
// ONLY the caller's own links (same data as GET /api/profile's links array).
// Auth wajib (main.go wraps it in RequireAuth) — versi lama yang menampilkan
// semua link SEMUA user dibuang karena membocorkan short-code + URL orang
// lain ke publik begitu saja (lihat komentar main.go & links/page.jsx).
// Baca via PRIMARY (read-your-own-writes, konsisten dengan /api/profile):
// link yang baru dibuat oleh user harus langsung terlihat di list sendiri.
func (h *Handler) HandleListLinks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		http.Error(w, "Login required", http.StatusUnauthorized)
		return
	}

	links, err := h.Store.ListLinksByCreatorPrimary(*creatorID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if links == nil {
		links = []db.Link{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(links)
}
