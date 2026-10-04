package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"jejak/internal/auth"
	"jejak/internal/cache"
	"jejak/internal/db"
	"jejak/internal/env"
	"jejak/internal/handler"
	"jejak/internal/health"
	"jejak/internal/middleware"
	"jejak/internal/migrate"
	"jejak/internal/ratelimit"
)

// APP_MODE selects the architecture at runtime without code changes: baseline
// = one instance straight to one database, no cache/queue (Phase 0 reference
// for comparative load tests); full = SingleStore primary+replica with cache
// and async queue; shard = sharded store as in Phase 5. All DSNs/URLs are read
// from the environment.
// Trade-off: mode branching adds if/else in main, but every mode uses the same
// Handler/Store so load-test results stay comparable (only the infrastructure
// that is enabled changes, not the business logic). Separate binaries per
// phase were rejected: duplicated code and configuration drift.
func main() {
	// .env (if present) is loaded first; OS environment variables that are
	// already set always win.
	env.LoadDotEnv()

	mode := strings.TrimSpace(env.Get("APP_MODE", "full"))
	port := strings.TrimSpace(env.Get("PORT", "8080"))
	dbURL := env.Get("DATABASE_URL", "postgres://jejak:password@localhost:5432/jejak?sslmode=disable")
	replicaURL := os.Getenv("DATABASE_REPLICA_URL") // empty = no replica
	redisURL := os.Getenv("REDIS_URL")              // empty = no cache/queue

	// TRUST_PROXY: X-Forwarded-For is client-spoofable, so ClientIP ignores
	// it by default (an attacker rotating the header would defeat every
	// IP-keyed rate limit). Set TRUST_PROXY=true ONLY behind a real reverse
	// proxy that overwrites the header on ingress (nginx/Traefik/Cloudflare).
	// COOKIE_SECURE: marks the session cookie Secure so it never travels over
	// plain HTTP; required behind TLS in production, must stay false on the
	// plain-http localhost dev setup (the browser would drop the cookie and
	// login would silently fail).
	ratelimit.SetTrustProxy(strings.EqualFold(strings.TrimSpace(env.Get("TRUST_PROXY", "false")), "true"))
	auth.SecureCookies = strings.EqualFold(strings.TrimSpace(env.Get("COOKIE_SECURE", "false")), "true")

	// Auto-migrate the PRIMARY before the store is opened (see the rationale
	// in internal/migrate: this closes the recurring "migration left unrun"
	// bug that once caused silent redirect 404s and dashboard 500s).
	// Principle: only the primary (DATABASE_URL) is migrated automatically.
	// The replica is NEVER migrated here: replica synchronization stays
	// manual (see the rationale in migrate.go).
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
		// Phase 0: one database, no sharding/replica, no cache/queue.
		s, err := db.NewSingleStore(dbURL, "")
		if err != nil {
			log.Fatal("Failed to create store:", err)
		}
		store = s
	case "shard":
		// Phase 5: sharding via SHARD_DSNS (comma-separated). Dedicated shard
		// infrastructure is not yet defined in compose; the localhost defaults
		// are for local experiments.
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
		// Full stack: write to the primary, read from the replica (if any).
		s, err := db.NewSingleStore(dbURL, replicaURL)
		if err != nil {
			log.Fatal("Failed to create store:", err)
		}
		store = s
	}

	// Redis is enabled outside baseline mode and only when REDIS_URL is set.
	// One client is shared by the queue (handler.Redis) and the cache.
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

	// Phase 9 auth: Redis-backed when Redis is present (shared by every
	// instance behind the load balancer and surviving restarts); in-memory
	// only for baseline mode without Redis (see the rationale in the auth
	// package).
	if rdb != nil {
		h.Auth = auth.NewRedisStore(rdb)
	} else {
		h.Auth = auth.NewMemoryStore()
	}

	// Phase 7: login throttle of 5 failures/minute per IP+username
	// (in-memory, active in every mode; see the rationale in the ratelimit
	// package).
	h.LoginLimiter = ratelimit.NewLimiter()
	// Account settings: 5 requests/minute per IP (bucket separate from login).
	h.AccountLimiter = ratelimit.NewLimiter()
	// Bulk import (link management): 3 requests/minute per creator: a
	// separate bucket; bulk import is heavy and rare and must not drain the
	// quota of other endpoints.
	h.BulkLimiter = ratelimit.NewLimiterWithMax(3)
	// Public API throttle: 100 requests/minute per API key (in-memory,
	// sufficient for single-instance development; a multi-instance deployment
	// needs Redis INCR per key).
	h.APILimiter = ratelimit.NewLimiterWithMax(100)
	// Analytics CSV export: 5 requests/minute per creator: slower than
	// ordinary dashboard reads because an export pulls raw rows (clicks mode
	// can touch 10,000 rows per request). A separate bucket keeps report
	// downloads from draining the quota of other endpoints.
	h.ExportLimiter = ratelimit.NewLimiterWithMax(5)
	// Registration: 10 attempts/minute per IP: bcrypt on every attempt makes
	// an unthrottled register endpoint a CPU-burning and spam-account vector
	// (see Handler.RegisterLimiter).
	h.RegisterLimiter = ratelimit.NewLimiterWithMax(10)
	// Anonymous shorten: 30 requests/minute per IP, capping spam-link
	// generation; the API-key path is not affected (its own per-key bucket).
	h.ShortenLimiter = ratelimit.NewLimiterWithMax(30)
	// Password verify (migration 17): 10 attempts/minute per IP+code -
	// bcrypt runs on every wrong guess, so the form needs a CPU budget;
	// successes reset the bucket (see Handler.VerifyLimiter).
	h.VerifyLimiter = ratelimit.NewLimiterWithMax(10)
	// Manual health check (migration 17): 10 checks/minute per IP: each one
	// makes a live outbound request (5s worst case) with its own budget
	// next to the worker batch (see Handler.HealthTriggerLimiter).
	h.HealthTriggerLimiter = ratelimit.NewLimiterWithMax(10)

	// Health checker for the manual "check now" trigger: the worker runs the
	// same Checker from cmd/worker/health.go (shared code, one classification
	// rule). Cache lets a status flip evict the redirect payload immediately.
	h.Checker = &health.Checker{Store: store, Cache: c, Logger: log.Default()}

	// authH wraps a session-required handler (Phase 9): anonymous requests get
	// 401 BEFORE the handler runs; the handler reads the creator id from the
	// request context via middleware.CreatorID (see middleware package).
	authH := func(fn http.HandlerFunc) http.Handler {
		return middleware.RequireAuth(h.Auth)(fn)
	}

	// Account settings (2026-09-30): rate limit of 5 requests/minute per IP on
	// ALL /api/account/* endpoints: a limiter SEPARATE from login (see
	// Handler.AccountLimiter). OptionalAuth (not authH) because these
	// endpoints also accept a Bearer API key: the resolver lives inside the
	// handler (accountCreatorID) and still returns 401 to anonymous callers.
	acctRL := middleware.RateLimit(h.AccountLimiter, func(r *http.Request) string {
		return "acct:" + ratelimit.ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For"))
	})
	acctH := func(fn http.HandlerFunc) http.Handler {
		return acctRL(middleware.OptionalAuth(h.Auth)(fn))
	}

	mux := http.NewServeMux()

	// Account settings: change email, change password, log out all devices,
	// delete account (Danger Zone: "HAPUS" confirmation plus password
	// enforced in the handler).
	mux.Handle("PUT /api/account/email", acctH(h.HandleUpdateEmail))
	mux.Handle("PUT /api/account/password", acctH(h.HandleUpdatePassword))
	mux.Handle("POST /api/account/logout-all", acctH(h.HandleLogoutAll))
	mux.Handle("DELETE /api/account", acctH(h.HandleDeleteAccount))

	// Optional auth: anonymous can shorten, logged-in auto-owns the link
	// (creator id resolved from context, nil for anonymous).
	mux.Handle("POST /api/shorten", middleware.OptionalAuth(h.Auth)(http.HandlerFunc(h.HandleShorten)))

	// Deep link tools (2026-10-04). Generate = pure URL lookup, no auth and
	// no rate limit (single parse, zero side effects); WhatsApp = optional
	// auth so a logged-in user's auto-shortened link is owned by them, with
	// the shorten path sharing the /api/shorten rate bucket.
	mux.HandleFunc("POST /api/deeplink/generate", h.HandleDeepLinkGenerate)
	mux.Handle("POST /api/tools/whatsapp-link", middleware.OptionalAuth(h.Auth)(http.HandlerFunc(h.HandleWhatsAppLink)))

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

	// Mandatory auth + per-creator scope: the Phase 8 version that exposed ALL
	// links of ALL users publicly has been deprecated (privacy leak: see
	// frontend/app/links/page.jsx). The dashboard reads links through
	// GET /api/profile (ListLinksByCreatorPrimary), so this separate list
	// endpoint remains useful, but only for the requesting user's own links.
	mux.Handle("GET /api/links", authH(h.HandleListLinks))

	mux.Handle("PUT /api/links/reorder", authH(h.HandleReorderLinks))

	mux.Handle("POST /api/links/claim", authH(h.HandleClaimLinks))

	// Link management bulk (2026-09-30): import 10-100 URLs + download ZIP QR.
	// RequireAuth (authH) is OUTSIDE RateLimit: a sessionless request gets 401
	// BEFORE consuming quota: random callers cannot burn another creator's
	// bucket. Key per creator (not IP): import is a user action performed
	// behind a shared NAT, so per-IP keys would compete for the same
	// allowance. 3/minute.
	bulkRL := middleware.RateLimit(h.BulkLimiter, func(r *http.Request) string {
		if id := middleware.CreatorID(r); id != nil {
			return fmt.Sprintf("bulk:%d", *id)
		}
		return "bulk:" + ratelimit.ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For"))
	})
	mux.Handle("POST /api/links/bulk", authH(func(w http.ResponseWriter, r *http.Request) {
		// RequireAuth (authH) already ran first, so CreatorID is populated and
		// the 401 is issued before quota is spent; rate limiting happens at
		// this step.
		bulkRL(http.HandlerFunc(h.HandleBulkShorten)).ServeHTTP(w, r)
	}))
	// QR bulk: no rate limit (PNG generation is cheap, capped at 200 files in
	// the handler). The literal ".zip" path must never become the wildcard
	// {short_code} (it would be swallowed by the detail route). The frontend
	// calls /api/links/qr-bulk without an extension through the Next proxy
	// (app/api/links/qr-bulk/route.js), which forwards to this .zip path.
	mux.Handle("GET /api/links/qr-bulk.zip", authH(h.HandleBulkQR))

	mux.Handle("GET /api/keys", authH(h.HandleListAPIKeys))
	mux.Handle("POST /api/keys", authH(h.HandleGenerateAPIKey))
	mux.Handle("DELETE /api/keys/{id}", authH(h.HandleDeleteAPIKey))
	// Public API (session INDEPENDENT: auth via Bearer API key, with rate
	// limiting inline inside HandleV1Shorten: the limiter runs AFTER the key
	// is valid, so a dead key cannot drain the bucket).
	mux.HandleFunc("POST /api/v1/shorten", func(w http.ResponseWriter, r *http.Request) {
		h.HandleV1Shorten(w, r)
	})

	// The literal pattern "../reorder" wins over "{short_code}" (Go 1.22
	// ServeMux precedence: literal > wildcard), so reorder stays safe.
	mux.Handle("PUT /api/links/{short_code}", authH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.HandleUpdateLink(r.PathValue("short_code"), w, r)
	})))

	// Manual health check (migration 17): authH first (401 without a
	// session), ownership + rate limit inside the handler - the ownership
	// check must precede the outbound probe (SSRF-shaped abuse), so it
	// cannot live in middleware.
	mux.Handle("POST /api/links/{short_code}/check-health", authH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.HandleCheckHealth(r.PathValue("short_code"), w, r)
	})))

	// In-app notifications (migration 17): navbar bell feed + read marker.
	mux.Handle("GET /api/notifications", authH(h.HandleListNotifications))
	mux.Handle("PUT /api/notifications/{id}/read", authH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.HandleMarkNotificationRead(r.PathValue("id"), w, r)
	})))

	mux.Handle("GET /api/analytics/clicks-by-day", authH(h.HandleClicksByDay))

	// Analytics depth (2026-09-30): device/referrer breakdown plus a bounded
	// timeseries. authH outside (anonymous → 401 before any query runs); no
	// dedicated rate limit: both are bounded aggregates (GROUP BY, at most
	// 90 days) invoked once per tab/range change.
	mux.Handle("GET /api/analytics/breakdown", authH(h.HandleBreakdown))
	mux.Handle("GET /api/analytics/timeseries", authH(h.HandleTimeseries))

	// CSV export: authH OUTSIDE the rate limit (the bulk pattern: a
	// sessionless request gets 401 without draining quota), then exportRL
	// (5/minute per creator). Go registers the literal ".csv" path; the Next
	// proxy calls /api/analytics/export without the extension (see
	// frontend/app/api/analytics/export/route.js).
	exportRL := middleware.RateLimit(h.ExportLimiter, func(r *http.Request) string {
		if id := middleware.CreatorID(r); id != nil {
			return fmt.Sprintf("export:%d", *id)
		}
		return "export:" + ratelimit.ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For"))
	})
	mux.Handle("GET /api/analytics/export.csv", authH(func(w http.ResponseWriter, r *http.Request) {
		exportRL(http.HandlerFunc(h.HandleExportCSV)).ServeHTTP(w, r)
	}))

	// Summary: session auth (preferred) or API key: hence OptionalAuth, then
	// the handler falls back to creatorFromAPIKey when the context is empty.
	mux.Handle("GET /api/analytics/summary", middleware.OptionalAuth(h.Auth)(http.HandlerFunc(h.HandleAnalyticsSummary)))

	mux.Handle("GET /api/profile", authH(h.HandleGetMyProfile))

	mux.Handle("PUT /api/profile", authH(h.HandleUpdateMyProfile))

	mux.Handle("POST /api/profile/avatar", authH(h.HandleUploadAvatar))

	// Static file server for avatar uploads: /uploads/avatars/{id}-{ms}.{ext}
	// (versioned per upload, see newAvatarPaths). The FileServer root is http.Dir("uploads") + StripPrefix("/uploads/"),
	// NOT http.Dir(".") + StripPrefix("/"): the previous setup rooted the
	// entire working directory and depended only on responses staying under
	// /uploads/, which was fragile: a single ServeMux path-cleaning mistake
	// could expose .env, source code, and other repository data. With root
	// "uploads" and the correct "/uploads/" strip, only the uploads/ subtree
	// can be served; database paths ("/uploads/avatars/1-<ms>.jpg") are the
	// same shape the mux pattern accepts, which is the safest area to
	// confirm: the mux pattern accepts the
	// prefix, the strip removes it, and the FileServer serves paths relative
	// to the uploads directory. Trade-off: absolute paths outside uploads
	// (e.g. /uploads/../.env) are now cleaned by ServeMux to "/.env"
	// (301 + 404) and never touch a file. Alternative: cloud storage with
	// signed URLs (S3/GCS): correct at scale, overkill for local learning.
	// 0750 (G301): owner + web-server group write access to the avatar store
	// is enough; group/other get no write. The error is checked (G104): a
	// failed mkdir would otherwise surface later as opaque 500s per upload.
	if err := os.MkdirAll("uploads/avatars", 0750); err != nil {
		log.Fatal("Failed to create uploads/avatars:", err)
	}
	mux.Handle("GET /uploads/", cacheRevalidate(http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads")))))

	// Password verify (migration 17): POST target of the /r/{code} form.
	// No session auth - the link itself is the resource - and the handler
	// rate-limits per IP+code before running bcrypt. The literal "/verify"
	// segment outranks nothing here (GET /r/ is a different method).
	mux.Handle("POST /r/{short_code}/verify", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.HandlePasswordVerify(r.PathValue("short_code"), w, r)
	}))

	mux.HandleFunc("GET /r/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if len(path) <= 3 {
			// GET /r/ without a short code is not a link. It previously
			// answered 200 with an empty body (frequently hit by crawlers and
			// probes issuing GET "/r/"). It now always returns 404 so it is
			// consistent with an unknown code.
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

	// Server timeouts (G114): ListenAndServe's zero-value timeouts let a slow
	// or malicious client hold a connection (and its goroutine) open
	// indefinitely: ReadHeaderTimeout caps the request-header read,
	// Read/WriteTimeout bound the body phases, IdleTimeout keeps drained
	// keep-alive connections from accumulating. Values are generous enough
	// for dashboard JSON and avatar uploads.
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("Server starting on :%s (mode=%s)", port, mode)
	log.Fatal(srv.ListenAndServe())
}

// cacheRevalidate sets Cache-Control: no-cache on uploaded files (avatars).
// Without any Cache-Control the browser falls back to heuristic freshness
// (RFC 7234 4.2.2: 10% of the age implied by Last-Modified) and may keep
// showing a replaced image from its cache without ever asking the server.
// "no-cache" still allows storing the bytes, but every reuse must first
// revalidate via If-Modified-Since; http.FileServer answers 304 when the
// file is unchanged, so repeat views stay cheap while a replaced or new
// file is always fetched.
func cacheRevalidate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}
