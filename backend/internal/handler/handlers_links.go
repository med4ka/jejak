package handler

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

// HandleShorten handles POST /api/shorten to create a short URL. Write-path
// design: new data lands in the sharded DB, and click counting is deliberately
// kept out of here so redirects stay fast (best-effort): this handler only
// creates the URL and populates the cache. Trade-off: HandleShorten never
// increments click_count, so it reads 0 until a worker consumes the queue
// event; real-time counts would require a different architecture.
// Incrementing here directly was rejected because it would slow down the
// redirect response, which must stay fast.
func (h *Handler) HandleShorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// 30 shorten/minute per IP: caps anonymous spam-link generation (the API
	// key path never reaches this handler: it has its own per-key bucket).
	// Every attempt consumes the budget, including invalid bodies, so a
	// flood of malformed requests cannot probe validation for free. Record
	// fires only after Allow so the bucket is not inflated past max.
	rlKey := "shorten\x00" + ratelimit.ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For"))
	if h.ShortenLimiter != nil {
		if !h.ShortenLimiter.Allow(rlKey) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too many requests, try again later", http.StatusTooManyRequests)
			return
		}
		h.ShortenLimiter.Record(rlKey)
	}
	h.doShorten(middleware.CreatorID(r), w, r)
}

// doShorten is the shared core behind POST /api/shorten (session-cookie auth)
// and POST /api/v1/shorten (API-key auth). Link ownership arrives through the
// creatorID parameter: nil = anonymous link, non-nil = owned by that creator
// (id from the cookie / from api_keys: different auth sources, identical
// outcome: CreateURL(creator_id)).
func (h *Handler) doShorten(creatorID *int64, w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL       string   `json:"url"`
		Slug      string   `json:"slug"`
		Tags      []string `json:"tags"`
		ExpiresAt string   `json:"expires_at"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// original_url is validated as http(s) BEFORE it is stored. Without this,
	// doShorten would accept any string: "javascript:", "data:", "//evil.com"
	// (scheme-relative) or empty text: stored as-is and emitted in the
	// Location header on redirect; modern browsers block javascript: there,
	// but the short link would remain an avoidable open-redirect/XSS gadget.
	// validRemoteURL is reused (not validHTTPURL) because this field MUST
	// point at the Internet: empty input and local "/uploads/*" paths must not
	// pass. Trade-off: short links to internal pages are ruled out, which is
	// not a product target; device_rules and socials have always used
	// validHTTPURL.
	if !validRemoteURL(req.URL) {
		http.Error(w, "url must be a valid http(s) URL", http.StatusBadRequest)
		return
	}

	// The custom slug is optional: when set it becomes the short_code, when
	// empty the code falls back to random as before. Check order: format first
	// (cheap, no I/O), reserved second (no I/O), then the DB. Check-then-insert
	// races (two requests with the same slug can both pass the check), so the
	// UNIQUE constraint is the backstop: error 23505 maps to 409 rather than a
	// generic 500, telling the client it lost the race instead of that the
	// server broke. Trade-off: one extra SELECT per custom slug (the random
	// path pays nothing). Inserting straight away and relying only on the
	// constraint was rejected because the reserved-word error could no longer
	// be distinguished from a collision.
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

	// Normalize tags: trim, lowercase (so "YouTube" == "youtube"), drop empty
	// and duplicate entries. The client (home form) already pre-parses commas
	// into an array, but the server re-validates: client input is never
	// trusted. Limits: max 5 tags, 1-20 chars each. Failure -> clear 400.
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

	// Expiry (link management, 2026-09-30): optional, RFC3339: the frontend
	// sends toISOString() output (always UTC, e.g. 2026-10-01T09:00:00Z).
	// Two error tiers: malformed format = 400 (like a malformed URL or slug);
	// a valid value already past or <= 1 hour from now = 422 Unprocessable
	// Entity plus a field error: the request is WELL-FORMED but its
	// semantics cannot be satisfied ("expired at creation" makes no sense;
	// allow at least 1 hour so a human can review). UTC discipline: Parse →
	// .UTC(), comparisons against time.Now().UTC(), and values reaching the DB
	// through nullableTime() never carry a local zone (the TIMESTAMP column is
	// zone-less, i.e. absolute UTC).
	var expiresAt *time.Time
	if s := strings.TrimSpace(req.ExpiresAt); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			writeFieldError(w, http.StatusBadRequest, "expires_at", "expires_at must be RFC3339 UTC (contoh: 2026-10-01T09:00:00Z)")
			return
		}
		u := t.UTC()
		if !u.After(time.Now().UTC().Add(time.Hour)) {
			writeFieldError(w, http.StatusUnprocessableEntity, "expires_at", "expires_at must be more than 1 hour in the future")
			return
		}
		expiresAt = &u
	}

	// Store in DB. Logged-in creator auto-owns the link; anonymous (nil)
	// keeps Phase 0-8 behavior (creator_id NULL stays valid).
	if err := h.Store.CreateURL(shortCode, req.URL, creatorID, string(tagsJSON), expiresAt); err != nil {
		// Log the REAL error to the server terminal: without it, every write
		// failure shows clients only the generic "Failed to store URL" and the
		// root cause (which constraint, which column) cannot be traced.
		// Lesson from the varchar(10) vs 30-char slug bug.
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
	// Stores the JSON redirect payload (device_rules empty at create) so the
	// cache format matches Smart Link on the redirect path. The TTL is clamped
	// to the remaining expiry: a cache entry must not outlive expires_at (the
	// redirect also reads expiry from the cached payload: see HandleRedirect).
	if h.Cache != nil {
		if b, err := json.Marshal(redirectTarget{URL: req.URL, ExpiresAt: expiresAt}); err == nil {
			if err := h.Cache.Set(shortCode, string(b), redirectTTL(expiresAt)); err != nil {
				h.Logger.Printf("Warning: failed to populate cache: %v", err)
			}
		}
	}

	// The short URL is built from the incoming request (r.Host + scheme)
	// rather than a hardcoded host/port: the same server may run on another
	// port (8081/8082), behind a load balancer, or on a non-local domain:
	// the response always points at an address the client can really reach.
	// Trade-off: the client-supplied Host header is trusted (it can be
	// spoofed); production needs a domain whitelist. X-Forwarded-Proto is
	// honored so a TLS proxy does not wrongly yield http. Alternative: a
	// static BASE_URL config per environment, but one binary could not then
	// be reused on other ports without a config change.
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Only an EXACT "https" upgrades the scheme: an arbitrary
	// X-Forwarded-Proto value must never reach the body (it would place
	// attacker-controlled bytes inside the 201 response, a reflected-
	// injection gadget that Go's content sniffer could even re-label as
	// HTML). Any other value simply keeps the real scheme; a TLS proxy
	// always sends exactly "https".
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	shortURL := scheme + "://" + r.Host + "/r/" + shortCode
	w.WriteHeader(http.StatusCreated)
	// #nosec G705 -- the body is a literal http/https scheme + r.Host (Go's
	// server rejects hosts containing HTML metacharacters or control bytes:
	// httpguts.ValidHostHeader) + an allowlisted short code: nothing
	// scriptable can reach the response, and net/http sniffs it as
	// text/plain. Host spoofing can only poison the caller's OWN response
	// (documented trade-off above: production needs a domain whitelist).
	w.Write([]byte(shortURL))
}

// detectDevice uses SIMPLE KEYWORD MATCHING rather than a heavy User-Agent
// parsing library (ps, ua-parser): only two cases are needed: iOS
// (iPhone/iPad) vs Android, and two substrings already separate ~all real
// devices for the link-in-bio use case; 'Mobile' alone is unreliable because
// it also appears in tablet/desktop UAs, so 'iPhone'/'iPad'/'Android' is the
// better signal. Trade-off: edge cases are imprecise (iPadOS desktop-mode UAs
// still report 'Macintosh' + 'Mobile', old Android versions omit 'Android' in
// WebView UAs, bots disguise themselves), so a small share of requests fall
// back to original_url: acceptable because that is exactly the pre-existing
// default (backward compatible, no link breaks). A full UA parser would give
// >99% accuracy but adds a heavy dependency, 2-3x CPU per redirect for these
// two keywords, and decoder-quirk risk. Rule: a non-empty device rule wins;
// otherwise original_url as usual.
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
// Disabled is omitempty (absent = active): this server never writes a
// disabled target (a disabled link short-circuits to 410 BEFORE the cache),
// but the field exists so that a cache entry with disabled set by another
// writer or an external tool is still honored (no redirect): and it keeps
// is_active forward compatible in the cache payload. Backward compatible:
// old entries lacking the field are automatically active.
type redirectTarget struct {
	URL         string            `json:"url"`
	DeviceRules map[string]string `json:"device_rules,omitempty"`
	Disabled    bool              `json:"disabled,omitempty"`
	// ExpiresAt is omitempty (absent = no expiry): old cache entries (written
	// before link expiry existed) remain valid = active forever. When present,
	// the redirect uses it to detect 410 WITHOUT an extra DB query.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// redirectTTL computes the redirect cache TTL in seconds: 300 by default, but
// it must NOT outlive expires_at (remaining time <= 0 is clamped to 1 second:
// that case should already have been handled by the 410 before the cache
// write; this is defensive).
func redirectTTL(expiresAt *time.Time) int64 {
	const def int64 = 300
	if expiresAt == nil {
		return def
	}
	remain := int64(time.Until(expiresAt.UTC()).Seconds())
	if remain < 1 {
		return 1
	}
	if remain < def {
		return remain
	}
	return def
}

// writeFieldError sends a field-specific JSON error (any status): used by
// expires_at validation, which MUST name the offending field rather than emit
// a generic message (the frontend displays per-field messages in the form).
func writeFieldError(w http.ResponseWriter, status int, field, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg, "field": field})
}

// writeGoneHTML answers 410 Gone with a minimal HTML page (inline styles, no
// external assets): the expiry specification: not an http.Error plain-text
// body like "Link disabled", but a document worth loading in a browser when a
// user clicks a link that has expired.
func writeGoneHTML(w http.ResponseWriter, title, detail string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusGone)
	fmt.Fprintf(w, `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<style>
  body { margin:0; min-height:100vh; display:flex; align-items:center; justify-content:center;
         background:#FAFAF7; color:#1C1A12; font-family:ui-monospace,SFMono-Regular,Menlo,monospace; }
  main { text-align:center; padding:24px; }
  h1 { font-size:2rem; margin:0 0 8px; }
  h1 span { color:#FF5C3D; }
  p { margin:0; font-size:0.95rem; opacity:0.75; }
</style>
</head>
<body>
<main>
  <h1><span>410</span> %s</h1>
  <p>%s</p>
</main>
</body>
</html>
`, title, title, detail)
}

// parseTarget reads a cache value back into a redirectTarget. Old cache
// entries written before Smart Link stored a bare URL: those parse-fail and
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

// HandleRedirect handles GET /r/{shortCode}. Design: read path plus
// asynchronous decoupling: the redirect never waits for logging
// (fire-and-forget), because the user waits for the redirect itself; this is
// the core Phase 6 principle ("the redirect must not wait for logging to
// finish"). Trade-off: the click count can lag slightly because a worker
// processes it in the background, and a worker crash may lose clicks, but
// redirect latency stays unaffected; synchronous logging was rejected because
// it would hurt redirect performance and defeat Phase 6 (decoupling read/write
// path). Smart Link: the target URL is chosen per device (detectDevice) from
// device_rules: the cache stores {url, device_rules} so a cache hit still
// routes per device without another DB query.
func (h *Handler) HandleRedirect(shortCode string, w http.ResponseWriter, r *http.Request) {
	device := detectDevice(r.UserAgent())

	// Cache-aside: check the cache first (nil-safe; baseline goes straight to the DB).
	if h.Cache != nil {
		if cached, found := h.Cache.Get(shortCode); found {
			if target, ok := parseTarget(cached); ok {
				// A cache entry marking the link disabled must NOT redirect.
				// Evict it so the next request re-reads the DB: a re-enabled
				// link takes effect immediately instead of waiting out the 300s TTL.
				if target.Disabled {
					h.Cache.Delete(shortCode)
					http.Error(w, "Link disabled", http.StatusGone)
					return
				}
				// Cached expiry: once it has passed, 410 + evict (the next
				// request reads the DB and finds the same condition without
				// spending cache CPU). The cache TTL was already clamped to
				// expires_at when written (redirectTTL), so this path only
				// triggers for requests arriving right around the deadline.
				if target.ExpiresAt != nil && time.Now().UTC().After(target.ExpiresAt.UTC()) {
					h.Cache.Delete(shortCode)
					writeGoneHTML(w, "Link kedaluwarsa", "Link ini sudah melewati waktu aktifnya.")
					return
				}
				h.logClickAsync(shortCode, r)
				http.Redirect(w, r, target.pickURL(device), http.StatusFound)
				return
			}
			// Old entry (bare URL, pre-Smart Link): still works.
			h.logClickAsync(shortCode, r)
			http.Redirect(w, r, cached, http.StatusFound)
			return
		}
	}

	// Read from the sharded store (original_url + device_rules for routing).
	link, err := h.Store.GetLink(shortCode)
	if err != nil {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	// Phase 13 lifecycle: a disabled link (is_active=false) must not redirect:
	// answer 410 Gone so the link "disappears" from the public side while
	// remaining visible in the owner's dashboard (where it can be re-enabled).
	// Checked BEFORE writing the cache: a disabled link never enters the cache
	// as an active target, so no click can be tallied through the cache path.
	if !link.IsActive {
		http.Error(w, "Link disabled", http.StatusGone)
		return
	}

	// Link management (migration 15): checked BEFORE writing the cache and
	// BEFORE logClick: an expired link must not enter the cache (others
	// would read it later) and must not tally clicks. It precedes the other
	// checks because it is a TIME condition (not a state the owner can fix:
	// expiry passed = permanent). 410 HTML, not an http.Error plain-text body.
	if link.ExpiresAt != nil && time.Now().UTC().After(link.ExpiresAt.UTC()) {
		writeGoneHTML(w, "Link kedaluwarsa", "Link ini sudah melewati waktu aktifnya.")
		return
	}

	target := redirectTarget{URL: link.OriginalURL, DeviceRules: link.DeviceRules, ExpiresAt: link.ExpiresAt}

	// Populate cache for future requests (nil-safe). The TTL is clamped to the
	// remaining expiry so a cache entry never outlives the point where the
	// 410 applies.
	if h.Cache != nil {
		if b, err := json.Marshal(target); err == nil {
			if err := h.Cache.Set(shortCode, string(b), redirectTTL(link.ExpiresAt)); err != nil {
				h.Logger.Printf("Warning: failed to populate cache: %v", err)
			}
		}
	}

	// Async: fire-and-forget click logging to queue
	// The redirect response is not blocked by this operation
	h.logClickAsync(shortCode, r)

	http.Redirect(w, r, target.pickURL(device), http.StatusFound)
}

// logClickAsync pushes the click event onto a Redis List queue asynchronously
// (fire-and-forget). Design: an async queue (Redis List) separates the write
// path from the read path: the redirect response must finish immediately
// while logging happens in the background (LPush onto "click_events"; the
// worker LPops and updates the DB periodically). Trade-off: clicks may be lost
// if the worker crashes before processing, but no request is blocked and data
// is at worst slightly delayed (eventual consistency). Redis Streams with
// consumer groups would offer stronger guarantees but were rejected as too
// complex for this phase.
func (h *Handler) logClickAsync(shortCode string, r *http.Request) {
	// Baseline mode (Async=false or Redis=nil): write synchronously straight to
	// the DB with the COMPLETE click event (referrer_domain + is_unique +
	// clicked_at) so the synchronous path persists data identical to the worker
	// (one persistence path, two write modes). Phase 14: unique clicks and the
	// referrer-domain breakdown MUST also be populated in baseline mode, not
	// only when async.
	if !h.Async || h.Redis == nil {
		ev := h.buildClickEvent(shortCode, r) // performs unique detection + fingerprinting
		if err := h.Store.LogClick(ev); err != nil {
			h.Logger.Printf("Warning: failed to log click: %v", err)
		}
		return
	}

	// Create click event message: fields are filled in FULL so the worker
	// never has to guess or re-parse (fire-and-forget that stays accurate).
	// The analytics dimensions (user_agent plus its classification) are
	// already computed HERE (deviation A: classify at write time): the worker
	// only forwards them and needs no device/referrer rules at all.
	device := classifyDevice(r.UserAgent())
	refHost := referrerDomain(r.Referer())
	event := map[string]string{
		"short_code":      shortCode,
		"referrer":        r.Referer(),
		"referrer_domain": refHost,
		"referrer_type":   classifyReferrer(refHost),
		"user_agent":      truncateUA(r.UserAgent()),
		"device_type":     device,
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

// referrerDomain extracts the bare host (WITHOUT scheme or path) from a
// Referer header. Rationale: the per-domain analytics breakdown needs one
// consistent shape: "https://google.com/", "google.com" and
// "http://google.com" must land in the same bucket instead of three separate
// ones. Trade-off: a non-URL referrer (e.g. "android-app://x") or an
// unparseable one yields "" (referrer_domain NULL in the DB, contributing
// nothing to the breakdown).
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

// clickFingerprint produces a stable per-visitor hash so the system can tell
// "has this same person already clicked this link within 24 hours?": the
// basis of is_unique. Fingerprint = client IP + User-Agent (the two signals
// that best separate visitors at the HTTP level available here). Trade-off:
// hashing means no raw IP is stored anywhere (privacy), but two people behind
// the same NAT with identical UAs can be mistaken for one person: bit.ly has
// the same limitation.
func (h *Handler) clickFingerprint(r *http.Request) string {
	sum := sha256.Sum256([]byte(ratelimit.ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For")) + "|" + r.UserAgent()))
	return hex.EncodeToString(sum[:])
}

// isUniqueClick reports whether THIS visitor (per fingerprint) is new to THIS
// shortCode within the last 24h (a bit.ly-style unique window).
//
//	Redis configured (production/async): atomic SETNX on the key = unique.
//
//	Redis nil (baseline): the in-memory map (uniqueSeen) guarded by the
//	Handler. This is the Phase 14 baseline mode: uniques stay accurate inside
//	the single-worker process, not only in async. Trade-off: the state lives
//	in this process: it is not distributed across instances: and a restart
//	"warms the window back up", which is acceptable for the single-process
//	baseline; production uses shared, persistent Redis SETNX instead. See
//	also the notes on logClickAsync and in handler.go.
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

// truncateUA cuts the User-Agent at 512 CHARACTERS: the limit of the
// click_events.user_agent column (VARCHAR(512)). Postgres counts characters,
// not bytes, so the cut must happen at a rune boundary; cutting at a byte
// boundary could split a multi-byte rune and fail on encoding. A normal
// browser UA is < 200 chars, so the len <= 512 fast path almost always holds.
func truncateUA(s string) string {
	if len(s) <= 512 {
		return s
	}
	r := []rune(s)
	if len(r) <= 512 {
		return s
	}
	return string(r[:512])
}

// buildClickEvent assembles the FULL ClickEvent so LogClick (baseline sync)
// and the async worker both persist the SAME complete metadata: referrer +
// referrer_domain (analytics breakdown), is_unique (24h window), and
// clicked_at = the ACTUAL click time (not the processing time). One
// persistence path, two write modes: see logClickAsync. Analytics depth
// (2026-09-30): the raw UA + device_type + referrer_type are computed HERE as
// well: at the point where the request is still complete: so the stored log
// is ready to GROUP BY without re-parsing (deviation A).
func (h *Handler) buildClickEvent(shortCode string, r *http.Request) db.ClickEvent {
	refHost := referrerDomain(r.Referer())
	ev := db.ClickEvent{
		ShortCode:      shortCode,
		Referrer:       r.Referer(),
		ReferrerDomain: refHost,
		ReferrerType:   classifyReferrer(refHost),
		UserAgent:      truncateUA(r.UserAgent()),
		DeviceType:     classifyDevice(r.UserAgent()),
		IsUnique:       h.isUniqueClick(shortCode, r),
		ClickedAt:      time.Now().UTC(),
	}
	return ev
}

// HandleGetClickCount reads the click count from the sharded DB for
// analytics. Design: a read path straight from the sharded DB: because click
// logging is asynchronous via the queue, the count in the DB may not be fully
// up to date (eventual consistency); this queries urls.click_count, the
// counter maintained by the worker, so users should be aware numbers can be
// slightly "wet" (not yet updated) depending on worker timing. Trade-off: the
// response comes from the DB without waiting for the worker, but may
// under-count until the event has been processed. A separate Redis counter
// incremented synchronously from logClickAsync was rejected: two sources of
// truth (DB + Redis) would then have to be kept in sync.
func (h *Handler) HandleGetClickCount(shortCode string, w http.ResponseWriter, r *http.Request) {
	// Resolve the shard owning this shortCode before querying.
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

// HandleListLinks serves GET /api/links for the logged-in creator: it returns
// ONLY the caller's own links (the same data as GET /api/profile's links
// array). Auth is mandatory (main.go wraps it in RequireAuth): the older
// version that listed every user's links was removed because it leaked other
// people's short codes and URLs to the public (see comments in main.go and
// links/page.jsx). Reads go via PRIMARY (read-your-own-writes, consistent with
// /api/profile): a link the user just created must appear in their own list.
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
