package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"

	"jejak/internal/auth"
	"jejak/internal/cache"
	"jejak/internal/db"
	"jejak/internal/env"
	"jejak/internal/handler"
	"jejak/internal/middleware"
	"jejak/internal/migrate"
	"jejak/internal/ratelimit"
)

// LEARN:
//
//	Kenapa: APP_MODE memilih arsitektur saat runtime tanpa ubah kode: baseline =
//	1 instance langsung ke 1 DB tanpa cache/queue (mirip Fase 0, untuk load test
//	pembanding); full = SingleStore primary+replica + cache + queue async;
//	shard = sharded store ala Fase 5. Semua DSN/URL dibaca dari env.
//	Trade-off: Branching mode di main menambah if/else, tapi tiap mode tetap
//	memakai Handler/Store yang sama sehingga perbandingan load test adil
//	(yang berubah hanya infra yang dinyalakan, bukan logika bisnis).
//	Alternatif: Binary terpisah per fase, tapi duplikasi kode dan drift config.
func main() {
	// .env (kalau ada) dimuat dulu; env OS yang sudah di-set selalu menang.
	env.LoadDotEnv()

	mode := strings.TrimSpace(env.Get("APP_MODE", "full"))
	port := strings.TrimSpace(env.Get("PORT", "8080"))
	dbURL := env.Get("DATABASE_URL", "postgres://jejak:password@localhost:5432/jejak?sslmode=disable")
	replicaURL := os.Getenv("DATABASE_REPLICA_URL") // kosong = tanpa replica
	redisURL := os.Getenv("REDIS_URL")              // kosong = tanpa cache/queue

	// Auto-migrate PRIMARY sebelum store dibuka (lihat LEARN di
	// internal/migrate — ini menutup bug berulang "migrasi lupa dijalankan
	// manual" yang pernah bikin redirect 404 & dashboard 500 diam-diam).
	// PRINSIP: Hanya primary (DATABASE_URL). Replica TIDAK di-migrate
	// otomatis — sinkronisasi replica tetap manual (lihat LEARN di migrate.go).
	mgDB, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatal("Failed to open connection for migrate:", err)
	}
	if err := migrate.Run(mgDB, log.Default()); err != nil {
		log.Fatal("Auto-migrate gagal:", err)
	}
	if err := mgDB.Close(); err != nil {
		log.Printf("Warning: close migrate connection: %v", err)
	}

	var store db.ShardStore
	switch mode {
	case "baseline":
		// Fase 0: 1 database, tanpa sharding/replica, tanpa cache/queue.
		s, err := db.NewSingleStore(dbURL, "")
		if err != nil {
			log.Fatal("Failed to create store:", err)
		}
		store = s
	case "shard":
		// Fase 5: sharding via SHARD_DSNS (comma-separated). Infra shard
		// khusus belum ada di compose; default localhost untuk eksperimen lokal.
		raw := env.Get("SHARD_DSNS", "postgres://jejak:password@localhost:5432/jejak,postgres://jejak:password@localhost:5433/jejak")
		var dsns []string
		for _, d := range strings.Split(raw, ",") {
			if d = strings.TrimSpace(d); d != "" {
				dsns = append(dsns, d)
			}
		}
		if len(dsns) == 0 {
			log.Fatal("SHARD_DSNS is empty")
		}
		s, err := db.NewShardStoreFromDSNs(dsns)
		if err != nil {
			log.Fatal("Failed to create sharded store:", err)
		}
		store = s
	default: // "full"
		// Full stack: tulis ke primary, baca dari replica (kalau ada).
		s, err := db.NewSingleStore(dbURL, replicaURL)
		if err != nil {
			log.Fatal("Failed to create store:", err)
		}
		store = s
	}

	// Redis hanya dinyalakan di luar baseline dan kalau REDIS_URL diisi.
	// 1 client dipakai bersama untuk queue (handler.Redis) dan cache.
	var rdb *redis.Client
	var c cache.Cache
	async := false
	if mode != "baseline" && redisURL != "" {
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Fatal("Invalid REDIS_URL:", err)
		}
		rdb = redis.NewClient(opt)
		c = cache.NewCacheFromClient(rdb)
		async = true
	}

	h := handler.NewHandler(store, rdb, c, async)

	// Fase 9 auth: Redis-backed kalau Redis ada (dishare semua instance di
	// belakang LB + selamat dari restart), in-memory hanya untuk baseline
	// tanpa Redis (lihat LEARN di auth package).
	if rdb != nil {
		h.Auth = auth.NewRedisStore(rdb)
	} else {
		h.Auth = auth.NewMemoryStore()
	}

	// Fase 7: throttle login 5 gagal/menit per IP+username (in-memory,
	// jalan di semua mode; lihat LEARN di ratelimit package).
	h.LoginLimiter = ratelimit.NewLimiter()
	// Public API throttle: 100 req/menit per API key (in-memory, cukup
	// untuk single-instance dev; multi-instance butuh Redis INCR per key).
	h.APILimiter = ratelimit.NewLimiterWithMax(100)

	// authH wraps a session-required handler (Fase 9): anonymous requests get
	// 401 BEFORE the handler runs; the handler reads the creator id from the
	// request context via middleware.CreatorID (see middleware package).
	authH := func(fn http.HandlerFunc) http.Handler {
		return middleware.RequireAuth(h.Auth)(fn)
	}

	mux := http.NewServeMux()

	// Optional auth: anonymous can shorten, logged-in auto-owns the link
	// (creator id resolved from context, nil for anonymous).
	mux.Handle("POST /api/shorten", middleware.OptionalAuth(h.Auth)(http.HandlerFunc(h.HandleShorten)))

	mux.HandleFunc("POST /api/register", func(w http.ResponseWriter, r *http.Request) {
		h.HandleRegister(w, r)
	})

	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		h.HandleLogin(w, r)
	})

	mux.HandleFunc("POST /api/logout", func(w http.ResponseWriter, r *http.Request) {
		h.HandleLogout(w, r)
	})

	mux.HandleFunc("GET /api/u/", func(w http.ResponseWriter, r *http.Request) {
		username := strings.TrimPrefix(r.URL.Path, "/api/u/")
		if username == "" || strings.Contains(username, "/") {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		h.HandleCreatorLinks(username, w, r)
	})

	// Auth WAJIB + scope per-creator: versi Fase 8 yang menampilkan SEMUA link
	// SEMUA user ke publik sudah di-deprecate (kebocoran privasi — lihat
	// frontend/app/links/page.jsx). Dashboard read link lewat GET /api/profile
	// (ListLinksByCreatorPrimary), sehingga endpoint list terpisah ini tetap
	// berguna tapi hanya untuk link milik user sendiri.
	mux.Handle("GET /api/links", authH(h.HandleListLinks))

	mux.Handle("PUT /api/links/reorder", authH(h.HandleReorderLinks))

	mux.Handle("POST /api/links/claim", authH(h.HandleClaimLinks))

	mux.Handle("GET /api/keys", authH(h.HandleListAPIKeys))
	mux.Handle("POST /api/keys", authH(h.HandleGenerateAPIKey))
	mux.Handle("DELETE /api/keys/{id}", authH(h.HandleDeleteAPIKey))
	// Public API (session INDEPENDENT — auth via Bearer API key + rate limit
	// inline inside HandleV1Shorten: limiter jalan SETELAH key valid, jadi
	// key mati tidak ikut menguras kuota bucket).
	mux.HandleFunc("POST /api/v1/shorten", func(w http.ResponseWriter, r *http.Request) {
		h.HandleV1Shorten(w, r)
	})

	// Pattern literal "../reorder" menang atas "{short_code}" (precedence
	// ServeMux Go 1.22: literal > wildcard), jadi reorder tetap aman.
	mux.Handle("PUT /api/links/{short_code}", authH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.HandleUpdateLink(r.PathValue("short_code"), w, r)
	})))

	mux.Handle("GET /api/analytics/clicks-by-day", authH(h.HandleClicksByDay))

	mux.Handle("GET /api/profile", authH(h.HandleGetMyProfile))

	mux.Handle("PUT /api/profile", authH(h.HandleUpdateMyProfile))

	mux.Handle("POST /api/profile/avatar", authH(h.HandleUploadAvatar))

	// Static file server untuk avatar upload: /uploads/avatars/{id}.{ext}.
	// LEARN:
	//   Kenapa: Root FileServer = http.Dir("uploads") BUKAN http.Dir(".")
	//   + StripPrefix("/"). Sebelumnya root-nya seluruh working directory
	//   (http.Dir(".")) dengan strip cuma "/", jadi satu-satunya hal yang
	//   membatasi akses hanyalah pelayanan path di bawah /uploads/ — rapuh:
	//   kalau response cleaning ServeMux salah satu, bisa bocorkan .env,
	//   source, dan data repo lain. Dengan root "uploads" + StripPrefix
	//   "/uploads/" yang benar, hanya subtree uploads/ yang bisa di-serve.
	//   Path di DB ("/uploads/avatars/1.jpg") tetap sama — ini daerah yang
	//   paling aman dikonfirmasi: pattern mux menerima prefixnya, strip
	//   memotongnya, FileServer melayani relatif dari direktori uploads.
	//   Trade-off: Absolute path di luar uploads (mis. /uploads/../.env) kini
	//   di-clean ServeMux ke "/.env" (301 + 404) dan tidak menyentuh file.
	//   Alternatif: Cloud storage + signed URL (S3/GCS) — benar untuk skala,
	//   overkill untuk belajar lokal.
	os.MkdirAll("uploads/avatars", 0755)
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	mux.HandleFunc("GET /r/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if len(path) <= 3 {
			// GET /r/ tanpa short code: bukan link. Sebelumnya dijawab 200
			// dengan body kosong (sering dipukul crawler/probe GET "/r/").
			// Dipastikan 404 supaya konsisten dengan unknown code.
			http.NotFound(w, r)
			return
		}
		shortCode := path[3:] // Remove /r/
		if suffix := "/clicks"; len(shortCode) > len(suffix) && shortCode[len(shortCode)-len(suffix):] == suffix {
			shortCode = shortCode[:len(shortCode)-len(suffix)]
			h.HandleGetClickCount(shortCode, w, r)
		} else {
			h.HandleRedirect(shortCode, w, r)
		}
	})

	log.Printf("Server starting on :%s (mode=%s)", port, mode)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
