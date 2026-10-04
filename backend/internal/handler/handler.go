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
	"time"

	"github.com/redis/go-redis/v9"

	"jejak/internal/auth"
	"jejak/internal/cache"
	"jejak/internal/db"
	"jejak/internal/health"
	"jejak/internal/ratelimit"
)

// Handler groups the sharded store, Redis client (click queue), and logger for
// all request handling. Rationale: keeping storage, asynchronous processing,
// and logging separate lets each dependency be swapped independently (e.g.,
// Redis replaced by another queue) at the cost of a longer NewHandler
// signature; a functional-options pattern was rejected as less explicit here.
// Mode notes: Cache=nil disables cache-aside (baseline); Redis=nil or
// Async=false writes clicks synchronously to the DB (baseline).
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
	// LoginLimiter throttles POST /api/login (Phase 7). Set by main; nil
	// means no throttling (defensive fallback, keeps unit-test setup simple).
	LoginLimiter *ratelimit.Limiter
	// RegisterLimiter throttles POST /api/register: 10 attempts/minute per
	// IP (bucket keyed by IP alone: usernames are unique, so an attacker
	// rotates them anyway). Registration runs bcrypt, so an unthrottled
	// endpoint is a CPU-burning and spam-account vector. Every attempt counts
	// (success included): unlike login there is no credential to verify, so
	// "Record" acts as a straight budget. Set by main; nil = no throttling
	// (tests).
	RegisterLimiter *ratelimit.Limiter
	// ShortenLimiter throttles POST /api/shorten: 30 requests/minute per IP,
	// a bucket separate from login/register so legitimate link creation is
	// never blocked by a failed login. Anonymous shorten is the spam vector
	// this caps; the API-key path has its own per-key APILimiter. Set by
	// main; nil = no throttling (tests).
	ShortenLimiter *ratelimit.Limiter
	// AccountLimiter throttles the 4 account-settings endpoints (change
	// email/password, logout-all, delete account): 5 requests/minute per IP:
	// a bucket SEPARATE from LoginLimiter so a password-change attempt cannot
	// consume the login quota (and vice versa). Set by main; nil = no
	// throttling (tests).
	AccountLimiter *ratelimit.Limiter
	// BulkLimiter throttles POST /api/links/bulk (bulk URL import): 3
	// requests/minute per creator: separate bucket; a bulk import is heavy
	// but rare and must not drain the quota of other endpoints. Set by main;
	// nil = no throttling (tests).
	BulkLimiter *ratelimit.Limiter
	// APILimiter throttles POST /api/v1/shorten per API key (100 req/min
	// per key: looser than the login rate limit). Set by main; nil =
	// no throttling (tests, baseline mode).
	APILimiter *ratelimit.Limiter
	// ExportLimiter throttles GET /api/analytics/export.csv (analytics depth):
	// 5 requests/minute per creator: an export pulls a large row set from the
	// DB, so it must be slower than an ordinary dashboard read. Bucket is per
	// creator (not IP), like BulkLimiter. Set by main; nil = no throttling (tests).
	ExportLimiter *ratelimit.Limiter
	// VerifyLimiter throttles POST /r/{code}/verify: 10 attempts/minute per
	// IP+code - bcrypt runs on every attempt, so an unthrottled form would
	// be a CPU-burning brute-force vector. Failed attempts Record; a success
	// Reset forgives the bucket (same pattern as LoginLimiter: the
	// legitimate owner is never punished for typos). Set by main; nil = no
	// throttling (tests).
	VerifyLimiter *ratelimit.Limiter
	// HealthTriggerLimiter throttles POST /api/links/{code}/check-health:
	// 10 manual checks/minute per IP - every attempt makes a live outbound
	// request to the destination, so it needs its own budget (separate from
	// ShortenLimiter so checking never blocks link creation). Set by main;
	// nil = no throttling (tests).
	HealthTriggerLimiter *ratelimit.Limiter
	// Checker runs health checks (live HEAD request + persistence +
	// notification + cache eviction). Set by main after NewHandler; nil
	// makes the manual trigger answer 503 instead of making an outbound
	// request without its dependency wired (defensive).
	Checker *health.Checker

	// uniqueMu guards uniqueSeen: the in-memory dedup fallback (Phase 14)
	// used when Redis == nil (single-process baseline): isUniqueClick writes
	// here with a 24h window so unique clicks stay accurate in baseline mode
	// as well, not only when async (see the notes on logClickAsync). Trade-off:
	// the state lives in this process only, not across instances; production
	// relies on shared, persistent Redis SETNX instead (see isUniqueClick).
	uniqueMu   sync.Mutex
	uniqueSeen map[string]int64
}

// NewHandler builds a Handler from externally supplied dependencies
// (dependency injection): callers must construct them first, which trades a
// longer call site for isolated tests and swappable implementations: a
// missing dependency fails at compile time rather than at runtime. Function
// options or package defaults were rejected as less transparent.
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

// parseSocials decodes the JSONB document; corrupt or empty input becomes []
// (fail-soft: a single corrupt row must not take down the whole profile page).
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

// validHTTPURL allows empty (optional field), an http(s) URL with a host, or
// a local "/uploads/*" path (server-side uploaded avatar).
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

// validRemoteURL requires a URL shaped the way url.Parse accepts: http(s)
// scheme plus a host. Empty values and "/uploads/*" are rejected because this
// guards fields that MUST point at an Internet resource (original_url, Smart
// Link device_rules), never a local asset.
func validRemoteURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// maxJSONBodyBytes caps JSON request bodies: a body over the limit fails to
// decode (http.MaxBytesReader), which prevents a huge request from exhausting
// decoder RAM. 1MB is far above normal payloads (shorten/profile/login).
const maxJSONBodyBytes = 1 << 20

// decodeJSON decodes a JSON body under a hard size cap. Rationale: the default
// json.Decoder reads without limit, so a 500MB POST would make it allocate
// that much memory; http.MaxBytesReader bounds the read and the resulting
// "request body too large" failure surfaces as 400 here (not 413, so clients
// do not read it as a server crash). Trade-off: 400 for an over-limit body is
// an imperfect status, but it keeps a single error path across every handler;
// a global body-limiting middleware was rejected because it cannot return
// consistent per-endpoint JSON errors.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return errors.New("invalid request body")
	}
	return nil
}

// botTokens marks crawler/preview user agents so the "visitor" numbers in
// breakdowns are not polluted by Googlebot, Slackbot, and similar. Tokens are
// chosen strictly: only ones almost impossible to see in a human browser UA.
// ("bot" already matches googlebot/bingbot/twitterbot/discordbot/robot;
// "preview" matches facebookexternalhit-style link previews.)
// Trade-off: the phone brand "CUBOT" is filtered too (it contains "bot"):
// that device volume is so small that a special-case rule would only make
// classification harder without improving any meaningful number.
var botTokens = []string{
	"bot", "crawler", "spider", "slurp", "preview", "monitor",
	"curl/", "wget/", "python-requests", "go-http-client", "facebookexternalhit", "embedly",
}

// classifyDevice buckets a User-Agent for analytics:
// bot → tablet → mobile → desktop → unknown.
//
//	ORDER MATTERS (deviation B): tablet is checked BEFORE mobile because a
//	standard iPad UA ("(iPad; ...)" + "Mobile/15E148") and an Android tablet
//	both contain "Mobile"/"Android": if mobile were tested first, every
//	tablet would remain classified as mobile and the "Tablet" card would stay
//	0 forever. Tablet candidates: an explicit tablet token
//	(ipad/tablet/kindle/silk/playbook) or Android WITHOUT the word "mobile"
//	(stock Android on tablets does not say "Mobile", phones always do).
//	Trade-off: iPadOS desktop mode (reporting "Macintosh; Intel Mac OS X")
//	still counts as mobile because its UA also contains "Mobile": the UA
//	deliberately disguises itself, and a full parsing library would not fix
//	this without far more elaborate heuristics.
func classifyDevice(ua string) string {
	u := strings.ToLower(strings.TrimSpace(ua))
	if u == "" {
		return "unknown"
	}
	for _, tok := range botTokens {
		if strings.Contains(u, tok) {
			return "bot"
		}
	}
	// Tablet BEFORE mobile: see the note above.
	if strings.Contains(u, "ipad") || strings.Contains(u, "tablet") ||
		strings.Contains(u, "kindle") || strings.Contains(u, "silk") ||
		strings.Contains(u, "playbook") {
		return "tablet"
	}
	if strings.Contains(u, "android") && !strings.Contains(u, "mobile") {
		return "tablet"
	}
	if strings.Contains(u, "mobi") || strings.Contains(u, "iphone") ||
		strings.Contains(u, "ipod") || strings.Contains(u, "windows phone") {
		return "mobile"
	}
	if strings.Contains(u, "windows nt") || strings.Contains(u, "macintosh") ||
		strings.Contains(u, "x11") || strings.Contains(u, "linux") {
		return "desktop"
	}
	return "unknown"
}

// matchHost compares a host against a single domain name: exact match,
// subdomain ("www.x.com", "l.facebook.com"), or registrable domain ("x.com").
// Without these conditions "x.com" would also match the "matrix.com"
// substring in other referrers: raw substring matching is unsafe for hosts.
func matchHost(host, name string) bool {
	return host == name || strings.HasPrefix(host, name+".") || strings.HasSuffix(host, "."+name)
}

// emailHosts is checked BEFORE the search engines: "mail.google.com" ends in
// ".google.com" and would be collected as search if the order were reversed.
// Email counts as "other" (not chat: chat here means a real-time messenger).
var emailHosts = []string{"mail.google.com", "mail.yahoo.com", "mail.aol.com", "outlook.live.com", "outlook.office.com", "outlook.office365.com"}

// classifyReferrer maps the referrer host to an analytics bucket:
// direct (no referrer) | search | chat | social | other.
// It runs AT WRITE TIME (deviation A): the referrer_type column is filled
// once and then only needs GROUP BY; referrer_domain (raw host, migration 13)
// is still stored for the detailed view.
// Deviation C: wa.me/chat.whatsapp.com/t.me/m.me = "chat" (sent directly
// through a messenger), fb.me = "social" (Facebook's shortening route).
func classifyReferrer(host string) string {
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "" {
		return "direct"
	}
	h = strings.TrimPrefix(h, "www.")
	for _, m := range emailHosts {
		if matchHost(h, m) {
			return "other"
		}
	}
	switch {
	case matchHost(h, "google.com") || matchHost(h, "google.co.id") ||
		matchHost(h, "bing.com") || matchHost(h, "duckduckgo.com") ||
		matchHost(h, "yahoo.com") || matchHost(h, "yandex.com") ||
		matchHost(h, "baidu.com") || matchHost(h, "ecosia.org") ||
		matchHost(h, "startpage.com") || matchHost(h, "search.brave.com") ||
		matchHost(h, "qwant.com") || matchHost(h, "mojeek.com"):
		return "search"
	case matchHost(h, "wa.me") || matchHost(h, "chat.whatsapp.com") ||
		matchHost(h, "web.whatsapp.com") || matchHost(h, "line.me") ||
		matchHost(h, "t.me") || matchHost(h, "telegram.me") ||
		matchHost(h, "m.me") || matchHost(h, "fb-messenger.com"):
		return "chat"
	case matchHost(h, "fb.me") || matchHost(h, "facebook.com") ||
		matchHost(h, "instagram.com") || matchHost(h, "twitter.com") ||
		matchHost(h, "x.com") || matchHost(h, "t.co") ||
		matchHost(h, "tiktok.com") || matchHost(h, "linkedin.com") ||
		matchHost(h, "reddit.com") || matchHost(h, "youtube.com") ||
		matchHost(h, "youtu.be") || matchHost(h, "pinterest.com") ||
		matchHost(h, "threads.net") || matchHost(h, "bsky.app") ||
		matchHost(h, "snapchat.com") || matchHost(h, "discord.com"):
		return "social"
	default:
		return "other"
	}
}

// RangeParams is the result of parseRange: from is inclusive (start of the
// day, UTC), to is exclusive (start of the day AFTER the last day): a span
// of exactly `days` calendar days ending today.
type RangeParams struct {
	From time.Time
	To   time.Time
	Days int
}

// parseRange turns the ?range= query into a UTC date range. Whitespace and ""
// default to 30 days; any other value ("7", "1y", "all", "bogus") is rejected
// (ok=false → 400 in the handler). The 7/30/90 restriction is deliberate: this
// endpoint scans click_events JOIN urls, and the longer the window the more
// expensive it becomes: the options exposed in the UI are the ones safe to
// run repeatedly.
func parseRange(q string) (RangeParams, bool) {
	n := 30
	switch strings.TrimSpace(q) {
	case "", "30d":
		n = 30
	case "7d":
		n = 7
	case "90d":
		n = 90
	default:
		return RangeParams{}, false
	}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return RangeParams{
		From: today.AddDate(0, 0, -(n - 1)),
		To:   today.AddDate(0, 0, 1),
		Days: n,
	}, true
}
