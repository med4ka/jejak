package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"jejak/internal/shortener"
)

// Link is one row of the dashboard link list (Fase 8 reads from the replica).
// Position drives creator-page order (reorder feature); 0 means never
// reordered. Tags live in a single JSONB column instead of a tags table: they
// are small (max 5 per link), always read together with their link, and never
// queried across creators, so a separate table would only add joins,
// migrations, and extra CRUD without benefit. Trade-off: no per-tag UNIQUE
// constraint in the database - validation (max 5 tags, <=20 chars, lowercase)
// runs in the application layer, and a future "top tags global" query would be
// an expensive full scan; migrate to a real tags table only when that use case
// appears.
type Link struct {
	ShortCode        string            `json:"short_code"`
	OriginalURL      string            `json:"original_url"`
	ClickCount       int64             `json:"click_count"`
	UniqueClickCount int64             `json:"unique_click_count"`
	Position         int               `json:"position"`
	IsFeatured       bool              `json:"is_featured"`
	IsActive         bool              `json:"is_active"`
	Tags             []string          `json:"tags"`
	DeviceRules      map[string]string `json:"device_rules,omitempty"`
	// ExpiresAt (migration 15) is when the link stops redirecting (NULL means
	// no limit). It is always UTC: writes use time.Now().UTC() or a UTC value
	// from the client, and pgx reads a TIMESTAMP without time zone as UTC.
	ExpiresAt *time.Time `json:"expires_at"`
	// Status is derived from ExpiresAt when the row is READ (active = no
	// expiry, scheduled = still in the future, expired = already past). It is
	// computed in scanLinks so every list reader (dashboard, public profile,
	// API) agrees on the value without recomputing the rule per handler.
	Status string `json:"status"`
	// Health monitor (migration 17): unknown|healthy|broken|timeout, stamped
	// by the worker every 6h or by the manual "check now" trigger. The badge
	// only shows broken (red) and timeout (yellow); healthy/unknown stay
	// silent, so a quiet dashboard is a healthy dashboard.
	HealthStatus string `json:"health_status"`
	// LastHealthCheck is the moment of the last worker/manual check (NULL =
	// never checked). Always UTC - written from time.Now().UTC().
	LastHealthCheck *time.Time `json:"last_health_check,omitempty"`
	// FallbackURL, when set, is where visitors are sent (via a 1s
	// interstitial) while the destination is broken. Empty string = no
	// fallback (the redirect then answers 503 instead).
	FallbackURL string `json:"fallback_url,omitempty"`
	// HasPassword is derived from password_hash <> '' at scan time. The hash
	// itself is never serialized (PasswordHash, json:"-") - only GetLink
	// reads it, for the redirect gate and cookie verification.
	HasPassword bool `json:"has_password"`
	// PasswordHash is the bcrypt hash of the optional per-link password. It
	// must never leave the backend (json:"-"); the redirect compares the
	// cookie (SHA-256 of this value) against it.
	PasswordHash string `json:"-"`
	// CreatorID is only filled by GetLink (the owner check for the manual
	// health trigger and the notification recipient). List queries are
	// already scoped by creator, so they do not select it (json:"-").
	CreatorID *int64 `json:"-"`
	// HealthNotifiedAt marks that the "link is broken" notification was
	// already created, so a link that stays broken does not spam the bell
	// (internal bookkeeping, json:"-").
	HealthNotifiedAt *time.Time `json:"-"`
}

// BulkURL is one input row of CreateURLsBatch, already validated by the
// handler: an http(s) URL, a code that is valid and unique within the batch,
// and tags JSON ready to write.
type BulkURL struct {
	ShortCode   string
	OriginalURL string
	TagsJSON    string
	ExpiresAt   *time.Time // always UTC; a bulk import without expiry passes nil
}

// ClickEvent is one persisted click (Fase 13). LogClick previously accepted
// only (short_code, referrer) and took the timestamp from DEFAULT
// CURRENT_TIMESTAMP at worker INSERT time, so clicked_at recorded the PROCESS
// time rather than the moment the user clicked. Handlers and workers now send
// the full event: an explicit clicked_at (the real click time), is_unique (a
// different visitor within a 24h window, bit.ly-style), and referrer_domain
// (the referrer host without its path - the basis for analytics breakdowns).
type ClickEvent struct {
	ShortCode      string
	Referrer       string
	ReferrerDomain string
	IsUnique       bool
	ClickedAt      time.Time
	// UserAgent, DeviceType, ReferrerType (migration 16) are analytics
	// dimensions classified at WRITE time rather than at read time, so a
	// dashboard breakdown is a plain GROUP BY - no regex per request and no
	// shipping whole rows to the application. The raw UA is truncated to 512
	// characters (VARCHAR(512) counts characters, not bytes) so a crawler with
	// a very long UA cannot trigger a truncation error.
	UserAgent    string
	DeviceType   string // bot|tablet|mobile|desktop|unknown (classifyDevice)
	ReferrerType string // direct|search|social|chat|other (classifyReferrer)
}

// BreakdownItem is one bucket of a device/referrer breakdown: the label plus
// how many clicks fell into it inside the requested window. Keys are already
// normalized at write time, so handlers can pass them straight to JSON.
type BreakdownItem struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

// LinkExportRow is one row of the "links" CSV export: the link itself plus
// its click/unique counts INSIDE the export window (not all-time counters -
// exported numbers must match the range the user selected).
type LinkExportRow struct {
	ShortCode    string
	OriginalURL  string
	Tags         []string
	Clicks       int64
	UniqueClicks int64
}

// ClickRow is one raw click event for the "clicks" CSV export (the largest
// mode - capped by the handler through limit; see ExportRowCap).
type ClickRow struct {
	ClickedAt      time.Time
	ShortCode      string
	DeviceType     string
	ReferrerType   string
	ReferrerDomain string
	IsUnique       bool
}

// parseRules decodes the device_rules JSONB document. Corrupt/empty/'{}' -> nil
// (nil map lookup returns "", so handlers treat missing key = fallback).
func parseRules(raw sql.NullString) map[string]string {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" || strings.TrimSpace(raw.String) == "{}" {
		return nil
	}
	var m map[string]string
	if json.Unmarshal([]byte(raw.String), &m) == nil && m != nil {
		return m
	}
	return nil
}

// LinkStatus maps expires_at to a list status: NULL -> "active", still ahead
// -> "scheduled", already past -> "expired" (rule agreed on 2026-09-30). The
// comparison uses time.After on absolute instants, so the zone of origin (UTC
// in the database, local in tests) cannot change the result.
func LinkStatus(expiresAt *time.Time, now time.Time) string {
	if expiresAt == nil {
		return "active"
	}
	if now.After(*expiresAt) {
		return "expired"
	}
	return "scheduled"
}

// scanLinks reads link rows (SELECT must include is_featured AND
// COALESCE(tags,'[]') AS tags AND COALESCE(device_rules,'{}') AS device_rules
// AND unique_click_count AND is_active AND expires_at AND health_status AND
// last_health_check AND fallback_url AND (password_hash <> ”) AS has_password)
// - the four migration 17 columns come last, in that order. Corrupt JSON
// fails soft to empty: one broken row must never take the whole list down.
func scanLinks(rows *sql.Rows) ([]Link, error) {
	var out []Link
	now := time.Now().UTC()
	for rows.Next() {
		var l Link
		var tags sql.NullString
		var rules sql.NullString
		var expires sql.NullTime
		var lastCheck sql.NullTime
		if err := rows.Scan(&l.ShortCode, &l.OriginalURL, &l.ClickCount, &l.UniqueClickCount, &l.Position, &l.IsFeatured, &l.IsActive, &tags, &rules, &expires, &l.HealthStatus, &lastCheck, &l.FallbackURL, &l.HasPassword); err != nil {
			rows.Close()
			return nil, err
		}
		l.Tags = []string{}
		if tags.Valid && strings.TrimSpace(tags.String) != "" {
			var parsed []string
			if err := json.Unmarshal([]byte(tags.String), &parsed); err == nil && parsed != nil {
				l.Tags = parsed
			}
		}
		l.DeviceRules = parseRules(rules)
		if expires.Valid {
			u := expires.Time.UTC()
			l.ExpiresAt = &u
		}
		if lastCheck.Valid {
			u := lastCheck.Time.UTC()
			l.LastHealthCheck = &u
		}
		l.Status = LinkStatus(l.ExpiresAt, now)
		out = append(out, l)
	}
	rows.Close()
	return out, nil
}

// ShardStore is the store interface over sharded databases: every operation
// (create, get, increment) routes to the shard selected by hashing short_code
// so that one database does not absorb the entire write load. Queries grow
// more complex because of that routing; the mapping stays deterministic (a
// short_code always lands on the same shard), but adding a third shard
// requires rebalancing existing data - without it, rows remain on the wrong
// shard. Consistent hashing would make shard growth cheaper but is
// considerably more complex to implement.
type ShardStore interface {
	// GetURL reads the destination of a short code (public redirect path).
	// Returns sql.ErrNoRows when the code is unknown; sharded mode adds
	// sql.ErrConnDone when the code's shard is unavailable. A replica read
	// may be briefly stale.
	GetURL(shortCode string) (string, error)
	// CreateURL stores one link; expiresAt nil means no expiry (always UTC -
	// the caller has already validated and converted it, see doShorten).
	// passwordHash is the bcrypt hash of the optional per-link password ( ""
	// = no gate) - hashed by the caller through auth.HashPassword, so the
	// row is gate-ready in the same INSERT (no create-then-update race).
	CreateURL(shortCode, originalURL string, creatorID *int64, tagsJSON string, expiresAt *time.Time, passwordHash string) error
	// CreateURLsBatch inserts a batch in ONE transaction (bulk import of
	// 10-100 URLs). Unique conflicts (short_code already taken) are skipped
	// per row through ON CONFLICT DO NOTHING - not by rolling back the whole
	// batch - and reported as an empty code at the same index, so the handler
	// can surface a per-row error. Result order matches items order.
	CreateURLsBatch(creatorID *int64, items []BulkURL) ([]string, error)
	// IncrementClickCount bumps urls.click_count by one (legacy counter
	// path). An unknown code is not an error (0-row UPDATE); sharded mode
	// returns sql.ErrConnDone when the shard is unavailable.
	IncrementClickCount(shortCode string) error
	// GetShard returns the *sql.DB serving shortCode (read DB on the single
	// store, the hashed shard when sharded), or nil when unavailable.
	GetShard(shortCode string) *sql.DB
	// ListLinks returns every link of every account. Sharded mode skips
	// missing shards; no links anywhere means an empty result, not an error.
	ListLinks() ([]Link, error)
	// CreateCreator inserts a creator and returns the new id. A duplicate
	// username surfaces as the database's unique-violation error; sharded
	// mode writes shard 0 (see its NOTE) and returns sql.ErrConnDone when
	// that shard is unavailable.
	CreateCreator(username, displayName, bio, passwordHash string) (int64, error)
	// GetCreatorByUsername reads from the PRIMARY: login and the public
	// profile page must see the freshest data (settings such as the theme must
	// not be stale on the replica).
	GetCreatorByUsername(username string) (Creator, error)
	// GetCreatorByID reads one creator by id. Returns sql.ErrNoRows when the
	// id is unknown (or not yet visible on a lagging replica); sharded mode
	// adds sql.ErrConnDone when shard 0 is unavailable.
	GetCreatorByID(id int64) (Creator, error)
	// GetCreatorByIDPrimary reads straight from the PRIMARY - for the user's
	// own data (dashboard) so reads immediately see their own writes; see the
	// SingleStore implementation below.
	GetCreatorByIDPrimary(id int64) (Creator, error)
	// GetCreatorAuth reads from the PRIMARY only for account settings - it
	// needs password_hash (to verify the password before changing email or
	// password, or deleting the account) and email (shown as the current
	// email). The hash is NEVER released to JSON; it is only used by
	// auth.CheckPassword in the handler.
	GetCreatorAuth(id int64) (Creator, error)
	// UpdateCreatorEmail writes the new email to the PRIMARY. It returns
	// ErrEmailTaken when another creator already uses it (UNIQUE partial index
	// creators_email_key - enforced by the database, not a racy
	// check-then-insert).
	UpdateCreatorEmail(id int64, email string) error
	// UpdateCreatorPassword writes the NEW password_hash (the caller has
	// already hashed it with bcrypt).
	UpdateCreatorPassword(id int64, passwordHash string) error
	// DeleteCreatorAccount removes the account and all of its data in one
	// PRIMARY transaction (click_events -> urls -> api_keys -> creators; the
	// order matters: urls.creator_id has a foreign key without CASCADE).
	DeleteCreatorAccount(id int64) error
	// UpdateCreatorProfile updates the owner's display fields (username is
	// immutable - it is the public URL). A missing id affects 0 rows and
	// still returns nil; sharded mode errors only when shard 0 is
	// unavailable.
	UpdateCreatorProfile(id int64, displayName, bio, avatarURL, socialsJSON, theme string) error
	// ListLinksByCreator reads from the PRIMARY and filters is_active (public
	// profile: the owner's edits must be visible immediately, disabled links
	// stay hidden).
	ListLinksByCreator(creatorID int64) ([]Link, error)
	// ListLinksByCreatorPrimary reads the link list straight from the PRIMARY
	// (the same read-your-own-writes set as GetCreatorByIDPrimary).
	ListLinksByCreatorPrimary(creatorID int64) ([]Link, error)
	// ReorderLinks rewrites positions from order in one transaction.
	// Returns sql.ErrNoRows when a code is unknown or not owned (404 in the
	// handler); sharded mode has no cross-shard transaction (documented
	// limitation) and adds sql.ErrConnDone for a missing shard.
	ReorderLinks(creatorID int64, order []string) error
	// LogClick writes one click atomically: event row plus counters, or
	// neither. Failures roll back both sides; delivery stays at-most-once
	// (see SingleStore.LogClick for the trade-off).
	LogClick(e ClickEvent) error
	// ClaimLinks attaches anonymous links to creatorID (IS NULL guard -
	// see SingleStore.ClaimLinks). Returns the number actually claimed;
	// unknown or already-owned codes are not errors, they only lower the
	// count. Sharded mode returns sql.ErrConnDone for a missing shard.
	ClaimLinks(creatorID int64, codes []string) (int64, error)
	// ClicksByDay returns the zero-filled 30-day series for the dashboard
	// chart. Query errors abort; a creator with no clicks gets all zeros,
	// not an error.
	ClicksByDay(creatorID int64) ([]DayCount, error)
	// AnalyticsSummary computes the 4 dashboard summary numbers (active links,
	// total clicks, uniques over 30 days, growth %) in the database, not the
	// client.
	AnalyticsSummary(creatorID int64) (AnalyticsSummary, error)
	// DeviceBreakdown / ReferrerBreakdown (analytics depth): clicks per
	// device_type / referrer_type bucket in the range [from, to) of one
	// creator. The columns are already classified at write time (see
	// ClickEvent), so the query only needs GROUP BY plus COALESCE for rows
	// that predate classification.
	DeviceBreakdown(creatorID int64, from, to time.Time) ([]BreakdownItem, error)
	ReferrerBreakdown(creatorID int64, from, to time.Time) ([]BreakdownItem, error)
	// ClicksDaily: clicks per day (YYYY-MM-DD) in the range - used by the
	// 7/30/90-day chart range and the daily CSV export mode. It does not
	// zero-fill (FillDays does that in the handler); only days with clicks are
	// returned.
	ClicksDaily(creatorID int64, from, to time.Time) (map[string]int64, error)
	// LinkExportStats: all of the creator's links plus click/unique counts
	// within the range. limit > 0 caps the rows (the CSV export requests
	// limit+1 to detect truncation without reading the whole table).
	LinkExportStats(creatorID int64, from, to time.Time, limit int) ([]LinkExportRow, error)
	// ListClicks: the most recent raw clicks in the range, capped by limit -
	// input for the clicks-mode CSV export (raw, one row per event).
	ListClicks(creatorID int64, from, to time.Time, limit int) ([]ClickRow, error)
	// AnalyticsFreshness: the analytics read source ("primary" / "replica")
	// for the data_per field of the response - so clients know whether the
	// numbers may be stale.
	AnalyticsFreshness() (string, error)
	// GetLink reads the full redirect row (device_rules, is_active,
	// expires_at) for Smart Link routing and 410 answers. Returns
	// sql.ErrNoRows when the code is unknown; sharded mode adds
	// sql.ErrConnDone for a missing shard.
	GetLink(shortCode string) (Link, error)
	// UpdateLink owner-scoped full replace of device_rules + tags (no PATCH
	// merge). Returns sql.ErrNoRows when unknown or not owned (404);
	// sharded mode adds sql.ErrConnDone for a missing shard.
	UpdateLink(creatorID int64, shortCode, deviceRulesJSON, tagsJSON string) error
	// SetFeaturedLink keeps ONE featured link per creator (a radio).
	// Featuring an unknown or not-owned code returns sql.ErrNoRows (404);
	// featured=false is idempotent. Sharded mode has no cross-shard
	// transaction (documented) and adds sql.ErrConnDone for a missing shard.
	SetFeaturedLink(creatorID int64, shortCode string, featured bool) error
	// SetLinkActive toggles is_active (Fase 13: disabling a link answers 410
	// on redirect and hides it from the public page; the dashboard still lists
	// it so it can be switched on again).
	SetLinkActive(creatorID int64, shortCode string, active bool) error
	// SetLinkExpiry writes or clears expires_at (migration 15) for a link
	// owned by the creator; nil removes the expiry (the link stays active
	// forever). RowsAffected 0 -> sql.ErrNoRows (not owned / unknown code ->
	// 404 in the handler).
	SetLinkExpiry(creatorID int64, shortCode string, expiresAt *time.Time) error
	// SetLinkPassword writes or clears the bcrypt hash (migration 17): "" is
	// the documented "remove the gate" value. Owner-scoped like the other
	// link setters -> sql.ErrNoRows for unknown/foreign codes (404).
	SetLinkPassword(creatorID int64, shortCode, passwordHash string) error
	// SetLinkFallback writes or clears fallback_url ("" = clear) for an
	// owned link - where visitors are sent (1s interstitial) while the
	// destination is broken. sql.ErrNoRows when not owned / unknown.
	SetLinkFallback(creatorID int64, shortCode, fallbackURL string) error
	// UpdateLinkHealth stamps health_status + last_health_check plus the
	// notification bookkeeping (health_notified_at; zero time = NULL so a
	// healed link can notify again on the next breakage). NOT creator-scoped
	// - the checker runs system-wide. Unknown code = 0-row UPDATE, not an
	// error (the row may vanish between the due-list and the check).
	UpdateLinkHealth(shortCode, status string, checkedAt, notifiedAt time.Time) error
	// ListLinksHealthDue picks up to limit ACTIVE links that are least
	// recently checked (never-checked first) - input for the worker's 6h
	// batch. Read from the PRIMARY so manual API checks count as done work.
	ListLinksHealthDue(limit int) ([]Link, error)
	// CreateNotification inserts one in-app notification (navbar bell) and
	// returns its id. PRIMARY only (creators pattern: shard 0 when sharded).
	CreateNotification(creatorID int64, typ, shortCode, message string) (int64, error)
	// ListNotifications returns the newest limit notifications of one
	// creator (bell dropdown). No notifications = empty slice, not an error.
	ListNotifications(creatorID int64, limit int) ([]Notification, error)
	// CountUnreadNotifications drives the red badge on the bell.
	CountUnreadNotifications(creatorID int64) (int64, error)
	// MarkNotificationRead is owner-scoped: unknown or foreign id ->
	// sql.ErrNoRows (404 without leaking ids).
	MarkNotificationRead(creatorID, id int64) error
	// DeleteLink removes an OWNED link permanently together with its
	// click_events history.
	DeleteLink(creatorID int64, shortCode string) error
	// StoreAPIKey inserts a key hash and returns the new id (empty label is
	// stored as NULL). The plaintext key is never stored. Sharded mode keeps
	// keys on shard 0: sql.ErrConnDone when it is unavailable.
	StoreAPIKey(creatorID int64, keyHash, label string) (int64, error)
	// ListAPIKeys lists the creator's keys, newest first, without the hash.
	// No keys means an empty result, not an error.
	ListAPIKeys(creatorID int64) ([]APIKey, error)
	// DeleteAPIKey removes one key scoped to the owner. Returns
	// sql.ErrNoRows for an unknown or foreign id (404 without leaking ids).
	DeleteAPIKey(creatorID int64, keyID int64) error
	// GetAPIKeyByHash is the auth lookup for /api/v1. Returns sql.ErrNoRows
	// for an unknown or revoked key (the middleware then rejects the
	// request); the read may be served by a replica.
	GetAPIKeyByHash(keyHash string) (APIKey, error)
	// TouchAPIKeyLastUsed stamps last_used_at after successful auth.
	// Best-effort: callers log the error and never fail the request - the
	// request already succeeded by then.
	TouchAPIKeyLastUsed(keyID int64) error
}

// Creator is one row of Fase 9's creators table. Bio is sql.NullString
// because the column is nullable (NULL must scan, empty string must not
// be confused with "no bio" at the DB layer; handler converts for JSON).
// Same for AvatarURL; Socials holds the raw JSONB document.
type Creator struct {
	ID           int64
	Username     string
	DisplayName  string
	Bio          sql.NullString
	AvatarURL    sql.NullString
	Socials      sql.NullString
	Theme        string
	PasswordHash string
	// Email is filled in through account settings (migration 14); "" means it
	// was never set. Only GetCreatorAuth reads it - other SELECTs do not touch
	// the column, so existing read paths are unchanged.
	Email string
}

// ErrEmailTaken is returned by UpdateCreatorEmail when another creator already
// uses the email (a UNIQUE violation is coerced into this sentinel so handlers
// can map it to 409 without logging the raw driver error string).
var ErrEmailTaken = errors.New("email already taken")

// APIKey is one row of the api_keys table. key_hash is NEVER exposed to
// handlers/JSON - only id/label/created_at/last_used_at leave the DB layer
// (plaintext only ever exists in the response the moment a key is generated).
// LastUsedAt is NULL until the key is first used (spec: nullable). No JSON
// tags: sql.Null* doesn't marshal to the shape frontends expect, so the
// handler shapes the payload (same as Creator/creatorProfileJSON).
type APIKey struct {
	ID         int64
	CreatorID  int64
	Label      sql.NullString
	CreatedAt  time.Time
	LastUsedAt sql.NullTime
}

// Notification is one row of the notifications table (migration 17, navbar
// bell). Typed as a short string ("link_broken") so new kinds (expiry soon,
// weekly summary) need no schema change. Message is the Indonesian sentence
// shown to the user - built by the caller, not the DB.
type Notification struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	ShortCode string    `json:"short_code"`
	Message   string    `json:"message"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

// SocialLink is one entry of the profile's social links, stored as a single
// JSONB array [{platform,url}] on the creator row instead of a socials table:
// the data is small (<=10 items), always read together with the profile, and
// never queried on its own, so a table would only add joins and migrations.
// Trade-off: no per-platform FK/UNIQUE in the database - shape validation
// (max items, valid URLs) moves to the application layer, and cross-creator
// queries such as "all creators linking Instagram" become expensive, but no
// such use case exists. JSONB rather than TEXT keeps the document validated
// as JSON and, if it is ever needed, queryable and indexable.
type SocialLink struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

// CreateCreator inserts a creator on the PRIMARY (writes always go primary).
// Returns the new id. Caller hashes the password first (see auth package).
// A duplicate username comes back as the driver's unique-violation error
// (HandleRegister coerces "duplicate key" to 409, same as
// UpdateCreatorEmail).
func (s *SingleStore) CreateCreator(username, displayName, bio, passwordHash string) (int64, error) {
	var id int64
	err := s.primary.QueryRow(
		"INSERT INTO creators (username, display_name, bio, password_hash) VALUES ($1, $2, NULLIF($3,''), $4) RETURNING id",
		username, displayName, bio, passwordHash,
	).Scan(&id)
	return id, err
}

// GetCreatorByUsername reads one creator from the PRIMARY (read-your-own-writes).
// It serves login and the public page /api/u/{username}: credentials and the
// user's settings (theme, bio, avatar) MUST be visible as soon as they are
// saved. Reading through the manually synchronized replica (ARCHITECTURE.md
// section 6) caused the "saved glass, /u still darkroom" bug - replica
// synchronization only runs when it is triggered manually. The replication-lag
// lesson still lives on the redirect path (GetURL -> readDB).
// Returns sql.ErrNoRows when no creator has that username.
func (s *SingleStore) GetCreatorByUsername(username string) (Creator, error) {
	var c Creator
	err := s.primary.QueryRow(
		"SELECT id, username, display_name, bio, avatar_url, socials, theme, password_hash FROM creators WHERE username = $1",
		username,
	).Scan(&c.ID, &c.Username, &c.DisplayName, &c.Bio, &c.AvatarURL, &c.Socials, &c.Theme, &c.PasswordHash)
	return c, err
}

// GetCreatorByID reads one creator by id via readDB (replica when enabled).
// Returns sql.ErrNoRows when the id is unknown or not yet visible on a
// lagging replica.
func (s *SingleStore) GetCreatorByID(id int64) (Creator, error) {
	var c Creator
	err := s.readDB().QueryRow(
		"SELECT id, username, display_name, bio, avatar_url, socials, theme, password_hash FROM creators WHERE id = $1",
		id,
	).Scan(&c.ID, &c.Username, &c.DisplayName, &c.Bio, &c.AvatarURL, &c.Socials, &c.Theme, &c.PasswordHash)
	return c, err
}

// UpdateCreatorProfile writes display_name, bio, avatar_url, socials, theme to the
// PRIMARY by session owner id. Username is immutable (it is the public URL).
// A missing id affects 0 rows and still returns nil: the session already
// proved the account exists, so this is not an error path.
func (s *SingleStore) UpdateCreatorProfile(id int64, displayName, bio, avatarURL, socialsJSON, theme string) error {
	_, err := s.primary.Exec(
		"UPDATE creators SET display_name = $1, bio = NULLIF($2,''), avatar_url = NULLIF($3,''), socials = $4, theme = $5 WHERE id = $6",
		displayName, bio, avatarURL, socialsJSON, theme, id,
	)
	return err
}

// GetCreatorByIDPrimary reads one creator by id from the PRIMARY. Only for the
// user's own data (GET /api/profile and similar): users must see their own
// changes without waiting for manual replica synchronization
// (read-your-own-writes). Public endpoints (/api/u) also read the PRIMARY
// since the stale-theme bug was fixed - see GetCreatorByUsername; the only
// remaining replica read path is the redirect (GetURL), where Fase 4
// demonstrates replication lag.
// Returns sql.ErrNoRows when the account no longer exists.
func (s *SingleStore) GetCreatorByIDPrimary(id int64) (Creator, error) {
	var c Creator
	err := s.primary.QueryRow(
		"SELECT id, username, display_name, bio, avatar_url, socials, theme, password_hash FROM creators WHERE id = $1",
		id,
	).Scan(&c.ID, &c.Username, &c.DisplayName, &c.Bio, &c.AvatarURL, &c.Socials, &c.Theme, &c.PasswordHash)
	return c, err
}

// GetCreatorAuth reads password_hash + email from the PRIMARY (account
// settings - read-your-own-writes, same as GetCreatorByIDPrimary).
// Returns sql.ErrNoRows when the account no longer exists.
func (s *SingleStore) GetCreatorAuth(id int64) (Creator, error) {
	var c Creator
	err := s.primary.QueryRow(
		"SELECT id, username, display_name, bio, avatar_url, socials, theme, password_hash, COALESCE(email,'') FROM creators WHERE id = $1",
		id,
	).Scan(&c.ID, &c.Username, &c.DisplayName, &c.Bio, &c.AvatarURL, &c.Socials, &c.Theme, &c.PasswordHash, &c.Email)
	return c, err
}

// UpdateCreatorEmail writes the new email to the PRIMARY. A unique violation
// (23505, "duplicate key") is coerced to ErrEmailTaken - a driver-agnostic
// string match, the same pattern as HandleRegister (both pgx and pq carry that
// text).
func (s *SingleStore) UpdateCreatorEmail(id int64, email string) error {
	_, err := s.primary.Exec("UPDATE creators SET email = $1 WHERE id = $2", email, id)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return ErrEmailTaken
	}
	return err
}

// UpdateCreatorPassword writes the new password_hash to the PRIMARY (the
// caller has already hashed it with auth.HashPassword).
// Returns database errors as-is: password_hash carries no uniqueness
// constraint, so unlike UpdateCreatorEmail there is no sentinel to translate.
func (s *SingleStore) UpdateCreatorPassword(id int64, passwordHash string) error {
	_, err := s.primary.Exec("UPDATE creators SET password_hash = $1 WHERE id = $2", passwordHash, id)
	return err
}

// DeleteCreatorAccount removes the creator together with its data in one
// transaction. click_events has no foreign key (migration 01) - it is deleted
// first so no rows are orphaned once urls are gone; urls.creator_id is FK NO
// ACTION and must be deleted before creators; api_keys already has ON DELETE
// CASCADE but is deleted explicitly so the cleanup order stays readable (and
// each account has few rows anyway).
// Returns sql.ErrNoRows when the account did not exist; any earlier error
// rolls the whole deletion back (defer tx.Rollback runs before the error
// escapes).
func (s *SingleStore) DeleteCreatorAccount(id int64) error {
	tx, err := s.primary.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		"DELETE FROM click_events WHERE short_code IN (SELECT short_code FROM urls WHERE creator_id = $1)",
		id,
	); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM urls WHERE creator_id = $1", id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM api_keys WHERE creator_id = $1", id); err != nil {
		return err
	}
	res, err := tx.Exec("DELETE FROM creators WHERE id = $1", id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

// ListLinksByCreator returns the ACTIVE links of one creator in ONE query from
// the PRIMARY (read-your-own-writes: the owner's edit/reorder/toggle appears on
// /u/{username} immediately instead of lagging on the manually synchronized
// replica), with denormalized click_count + unique_click_count included - this
// is the deliberate N+1 avoidance (PRD.md section 2): one WHERE creator_id
// query instead of 1 + N per-row count queries. Disabled links (is_active =
// false) are not shown to the public, as on bit.ly.
// A creator without active links returns an empty slice, not an error; query
// and scan errors are returned unchanged.
func (s *SingleStore) ListLinksByCreator(creatorID int64) ([]Link, error) {
	rows, err := s.primary.Query(
		"SELECT short_code, original_url, click_count, unique_click_count, position, is_featured, is_active, COALESCE(tags,'[]'), COALESCE(device_rules,'{}'), expires_at, health_status, last_health_check, fallback_url, (password_hash <> '') AS has_password FROM urls WHERE creator_id = $1 AND is_active = TRUE ORDER BY position ASC, id DESC",
		creatorID,
	)
	if err != nil {
		return nil, err
	}
	return scanLinks(rows)
}

// ListLinksByCreatorPrimary returns ALL links (active and disabled) of one
// creator directly from the PRIMARY (read-your-own-writes for the dashboard;
// see GetCreatorByIDPrimary). The dashboard must list disabled links so their
// owner can switch them back on.
// Same error behavior as ListLinksByCreator: empty slice, not an error.
func (s *SingleStore) ListLinksByCreatorPrimary(creatorID int64) ([]Link, error) {
	rows, err := s.primary.Query(
		"SELECT short_code, original_url, click_count, unique_click_count, position, is_featured, is_active, COALESCE(tags,'[]'), COALESCE(device_rules,'{}'), expires_at, health_status, last_health_check, fallback_url, (password_hash <> '') AS has_password FROM urls WHERE creator_id = $1 ORDER BY position ASC, id DESC",
		creatorID,
	)
	if err != nil {
		return nil, err
	}
	return scanLinks(rows)
}

// shardStore implements ShardStore using sharded databases
type shardStore struct {
	shards    map[int]*sql.DB
	numShards int
}

// NewShardStore creates a new sharded store with the given number of shards
func NewShardStore(numShards int) *shardStore {
	ss := &shardStore{
		numShards: numShards,
		shards:    make(map[int]*sql.DB),
	}

	for i := 0; i < numShards; i++ {
		dsn := "postgres://jejak:password@postgres_shard" + string(rune('a'+i%26)) + ":5432/jejak?sslmode=disable"
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			log.Printf("Warning: could not connect to shard %d: %v", i, err)
			continue
		}
		ss.shards[i] = db
	}

	return ss
}

// CreateURL implements ShardStore - routes to the correct shard based on short_code hash.
// creatorID nil = anonymous link (Fase 0-8 stay valid); non-nil = owned by creator.
// Returns sql.ErrConnDone when the target shard is unavailable; a duplicate
// short_code surfaces as the driver's unique-violation error.
func (s *shardStore) CreateURL(shortCode, originalURL string, creatorID *int64, tagsJSON string, expiresAt *time.Time, passwordHash string) error {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return sql.ErrConnDone // shard not available
	}
	if tagsJSON == "" {
		tagsJSON = "[]"
	}
	_, err := db.Exec("INSERT INTO urls (short_code, original_url, creator_id, tags, expires_at, password_hash) VALUES ($1, $2, $3, $4, $5, $6)", shortCode, originalURL, nullableInt(creatorID), tagsJSON, nullableTime(expiresAt), passwordHash)
	return err
}

// CreateURLsBatch routes each item to its shard. Sharded mode has no
// cross-shard transaction (documented the same way as ClaimLinks and
// ReorderLinks: the official baseline is SingleStore); per-row conflicts are
// still skipped with ON CONFLICT.
// Returns sql.ErrConnDone when a shard is missing; a driver error aborts the
// batch (rows already inserted stay - there is no cross-shard rollback); a
// code that is already taken leaves "" at its index instead of failing.
func (s *shardStore) CreateURLsBatch(creatorID *int64, items []BulkURL) ([]string, error) {
	codes := make([]string, len(items))
	for i, it := range items {
		shardIdx := shortener.ShardKey(it.ShortCode, s.numShards)
		db, ok := s.shards[shardIdx]
		if !ok {
			return nil, sql.ErrConnDone
		}
		tagsJSON := it.TagsJSON
		if tagsJSON == "" {
			tagsJSON = "[]"
		}
		res, err := db.Exec(
			"INSERT INTO urls (short_code, original_url, creator_id, tags, expires_at) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (short_code) DO NOTHING",
			it.ShortCode, it.OriginalURL, nullableInt(creatorID), tagsJSON, nullableTime(it.ExpiresAt),
		)
		if err != nil {
			return nil, err
		}
		if n, _ := res.RowsAffected(); n == 1 {
			codes[i] = it.ShortCode
		}
	}
	return codes, nil
}

// SetLinkExpiry routes to the short_code's shard (the WHERE creator_id scope
// leaves rows not owned by the caller untouched; see SingleStore.SetLinkExpiry).
// Returns sql.ErrConnDone when the shard is missing, sql.ErrNoRows when the
// code is unknown or not owned.
func (s *shardStore) SetLinkExpiry(creatorID int64, shortCode string, expiresAt *time.Time) error {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return sql.ErrConnDone
	}
	res, err := db.Exec(
		"UPDATE urls SET expires_at = $3 WHERE short_code = $1 AND creator_id = $2",
		shortCode, creatorID, nullableTime(expiresAt),
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// SetLinkPassword routes to the short_code's shard (owner-scoped like
// SetLinkExpiry). Returns sql.ErrConnDone when the shard is missing,
// sql.ErrNoRows when the code is unknown or not owned.
func (s *shardStore) SetLinkPassword(creatorID int64, shortCode, passwordHash string) error {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return sql.ErrConnDone
	}
	res, err := db.Exec(
		"UPDATE urls SET password_hash = $3 WHERE short_code = $1 AND creator_id = $2",
		shortCode, creatorID, passwordHash,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// SetLinkFallback routes to the short_code's shard; same contract as
// SetLinkPassword ("" clears the fallback).
func (s *shardStore) SetLinkFallback(creatorID int64, shortCode, fallbackURL string) error {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return sql.ErrConnDone
	}
	res, err := db.Exec(
		"UPDATE urls SET fallback_url = $3 WHERE short_code = $1 AND creator_id = $2",
		shortCode, creatorID, fallbackURL,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdateLinkHealth routes to the short_code's shard (see SingleStore for the
// notification-bookkeeping contract). Unknown code = 0-row UPDATE, not an
// error. Returns sql.ErrConnDone when the shard is missing.
func (s *shardStore) UpdateLinkHealth(shortCode, status string, checkedAt, notifiedAt time.Time) error {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return sql.ErrConnDone
	}
	_, err := db.Exec(
		"UPDATE urls SET health_status = $2, last_health_check = $3, health_notified_at = $4 WHERE short_code = $1",
		shortCode, status, checkedAt, nullableTimeZero(notifiedAt),
	)
	return err
}

// ListLinksHealthDue queries every shard (same shape as ListLinksByCreator),
// merges by last check (never-checked first, oldest next) and trims to
// limit - a cross-shard approximation of the SingleStore ORDER BY. Missing
// shards are skipped; no due links anywhere = empty slice, not an error.
func (s *shardStore) ListLinksHealthDue(limit int) ([]Link, error) {
	var out []Link
	for i := 0; i < s.numShards; i++ {
		db, ok := s.shards[i]
		if !ok {
			continue
		}
		rows, err := db.Query(
			"SELECT short_code, original_url, click_count, unique_click_count, position, is_featured, is_active, COALESCE(tags,'[]'), COALESCE(device_rules,'{}'), expires_at, health_status, last_health_check, fallback_url, (password_hash <> '') AS has_password FROM urls WHERE is_active = TRUE ORDER BY last_health_check NULLS FIRST, id LIMIT $1",
			limit,
		)
		if err != nil {
			return nil, err
		}
		part, err := scanLinks(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].LastHealthCheck, out[j].LastHealthCheck
		if a == nil {
			return b != nil
		}
		if b == nil {
			return false
		}
		return a.Before(*b)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// CreateNotification writes to shard 0 (see the NOTE: creators and their
// notifications have no sharding design; Fase 9 runs on SingleStore).
func (s *shardStore) CreateNotification(creatorID int64, typ, shortCode, message string) (int64, error) {
	db, ok := s.shardZero()
	if !ok {
		return 0, sql.ErrConnDone
	}
	var id int64
	err := db.QueryRow(
		"INSERT INTO notifications (creator_id, type, short_code, message) VALUES ($1, $2, $3, $4) RETURNING id",
		creatorID, typ, shortCode, message,
	).Scan(&id)
	return id, err
}

// ListNotifications reads the creator's feed from shard 0 (see NOTE).
func (s *shardStore) ListNotifications(creatorID int64, limit int) ([]Notification, error) {
	db, ok := s.shardZero()
	if !ok {
		return nil, sql.ErrConnDone
	}
	rows, err := db.Query(
		"SELECT id, type, short_code, message, read, created_at FROM notifications WHERE creator_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2",
		creatorID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.ShortCode, &n.Message, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		n.CreatedAt = n.CreatedAt.UTC()
		out = append(out, n)
	}
	return out, rows.Err()
}

// CountUnreadNotifications reads from shard 0 (see NOTE).
func (s *shardStore) CountUnreadNotifications(creatorID int64) (int64, error) {
	db, ok := s.shardZero()
	if !ok {
		return 0, sql.ErrConnDone
	}
	var n int64
	err := db.QueryRow(
		"SELECT COUNT(*) FROM notifications WHERE creator_id = $1 AND read = FALSE",
		creatorID,
	).Scan(&n)
	return n, err
}

// MarkNotificationRead is owner-scoped on shard 0 (see NOTE); unknown or
// foreign id -> sql.ErrNoRows.
func (s *shardStore) MarkNotificationRead(creatorID, id int64) error {
	db, ok := s.shardZero()
	if !ok {
		return sql.ErrConnDone
	}
	res, err := db.Exec(
		"UPDATE notifications SET read = TRUE WHERE id = $1 AND creator_id = $2",
		id, creatorID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// creator reads/writes go to shard 0. Documented simplification: Fase 9 is meant
// to run on SingleStore (baseline/full); shard mode keeps working for urls.
func (s *shardStore) shardZero() (*sql.DB, bool) {
	db, ok := s.shards[0]
	return db, ok
}

// CreateCreator inserts a creator on shard 0 (see NOTE above). Returns
// sql.ErrConnDone when shard 0 is unavailable; a duplicate username comes
// back as the driver's unique-violation error.
func (s *shardStore) CreateCreator(username, displayName, bio, passwordHash string) (int64, error) {
	db, ok := s.shardZero()
	if !ok {
		return 0, sql.ErrConnDone
	}
	var id int64
	err := db.QueryRow(
		"INSERT INTO creators (username, display_name, bio, password_hash) VALUES ($1, $2, NULLIF($3,''), $4) RETURNING id",
		username, displayName, bio, passwordHash,
	).Scan(&id)
	return id, err
}

// GetCreatorByUsername reads from shard 0 (see NOTE above). Returns
// sql.ErrNoRows when unknown, sql.ErrConnDone when shard 0 is unavailable.
func (s *shardStore) GetCreatorByUsername(username string) (Creator, error) {
	var c Creator
	db, ok := s.shardZero()
	if !ok {
		return c, sql.ErrConnDone
	}
	err := db.QueryRow(
		"SELECT id, username, display_name, bio, avatar_url, socials, theme, password_hash FROM creators WHERE username = $1",
		username,
	).Scan(&c.ID, &c.Username, &c.DisplayName, &c.Bio, &c.AvatarURL, &c.Socials, &c.Theme, &c.PasswordHash)
	return c, err
}

// GetCreatorByID reads one creator by id from shard 0 (see NOTE above).
// Returns sql.ErrNoRows when unknown, sql.ErrConnDone when shard 0 is
// unavailable.
func (s *shardStore) GetCreatorByID(id int64) (Creator, error) {
	var c Creator
	db, ok := s.shardZero()
	if !ok {
		return c, sql.ErrConnDone
	}
	err := db.QueryRow(
		"SELECT id, username, display_name, bio, avatar_url, socials, theme, password_hash FROM creators WHERE id = $1",
		id,
	).Scan(&c.ID, &c.Username, &c.DisplayName, &c.Bio, &c.AvatarURL, &c.Socials, &c.Theme, &c.PasswordHash)
	return c, err
}

// UpdateCreatorProfile writes to shard 0 (see NOTE above). Returns
// sql.ErrConnDone when shard 0 is unavailable; a missing id affects 0 rows
// and still returns nil (same as SingleStore.UpdateCreatorProfile).
func (s *shardStore) UpdateCreatorProfile(id int64, displayName, bio, avatarURL, socialsJSON, theme string) error {
	db, ok := s.shardZero()
	if !ok {
		return sql.ErrConnDone
	}
	_, err := db.Exec(
		"UPDATE creators SET display_name = $1, bio = NULLIF($2,''), avatar_url = NULLIF($3,''), socials = $4, theme = $5 WHERE id = $6",
		displayName, bio, avatarURL, socialsJSON, theme, id,
	)
	return err
}

// GetCreatorByIDPrimary reads from shard 0 - sharded mode has no separate
// primary/replica, so the result is identical to GetCreatorByID, including
// its error contract (sql.ErrNoRows / sql.ErrConnDone).
func (s *shardStore) GetCreatorByIDPrimary(id int64) (Creator, error) {
	return s.GetCreatorByID(id)
}

// GetCreatorAuth reads password_hash + email from shard 0 (see NOTE above).
// Returns sql.ErrNoRows when the account is gone, sql.ErrConnDone when
// shard 0 is unavailable.
func (s *shardStore) GetCreatorAuth(id int64) (Creator, error) {
	var c Creator
	db, ok := s.shardZero()
	if !ok {
		return c, sql.ErrConnDone
	}
	err := db.QueryRow(
		"SELECT id, username, display_name, bio, avatar_url, socials, theme, password_hash, COALESCE(email,'') FROM creators WHERE id = $1",
		id,
	).Scan(&c.ID, &c.Username, &c.DisplayName, &c.Bio, &c.AvatarURL, &c.Socials, &c.Theme, &c.PasswordHash, &c.Email)
	return c, err
}

// UpdateCreatorEmail writes to shard 0 (see NOTE above). Unique violations
// are coerced to ErrEmailTaken, same as SingleStore.UpdateCreatorEmail.
func (s *shardStore) UpdateCreatorEmail(id int64, email string) error {
	db, ok := s.shardZero()
	if !ok {
		return sql.ErrConnDone
	}
	_, err := db.Exec("UPDATE creators SET email = $1 WHERE id = $2", email, id)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return ErrEmailTaken
	}
	return err
}

// UpdateCreatorPassword writes to shard 0 (see NOTE above). Returns
// sql.ErrConnDone when shard 0 is unavailable, otherwise the driver error
// as-is (no uniqueness constraint to translate).
func (s *shardStore) UpdateCreatorPassword(id int64, passwordHash string) error {
	db, ok := s.shardZero()
	if !ok {
		return sql.ErrConnDone
	}
	_, err := db.Exec("UPDATE creators SET password_hash = $1 WHERE id = $2", passwordHash, id)
	return err
}

// DeleteCreatorAccount: all creator data lives on shard 0 (creators are never
// sharded - see the NOTE at shardZero), so a single transaction on shard 0
// suffices; only urls are spread across shards and every one of them is
// deleted too (same order as SingleStore.DeleteCreatorAccount).
// Returns sql.ErrConnDone when shard 0 is unavailable, sql.ErrNoRows when
// the account did not exist; any earlier error rolls the transaction back.
func (s *shardStore) DeleteCreatorAccount(id int64) error {
	db, ok := s.shardZero()
	if !ok {
		return sql.ErrConnDone
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		"DELETE FROM click_events WHERE short_code IN (SELECT short_code FROM urls WHERE creator_id = $1)",
		id,
	); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM urls WHERE creator_id = $1", id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM api_keys WHERE creator_id = $1", id); err != nil {
		return err
	}
	res, err := tx.Exec("DELETE FROM creators WHERE id = $1", id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

// ListLinksByCreator queries every shard with the same single-query shape
// (denormalized click_count, no N+1) and merges. Only the public scope is
// used (is_active = TRUE): disabled link tiles stay invisible to visitors.
// Cross-shard ordering is by shard index, not global time - acceptable at
// learning scale.
// A missing shard is skipped (its links simply do not appear) instead of
// failing the whole read; query or scan errors abort the merge and are
// returned as-is. No links anywhere = empty slice, not an error.
func (s *shardStore) ListLinksByCreator(creatorID int64) ([]Link, error) {
	var out []Link
	for i := 0; i < s.numShards; i++ {
		db, ok := s.shards[i]
		if !ok {
			continue
		}
		rows, err := db.Query(
			"SELECT short_code, original_url, click_count, unique_click_count, position, is_featured, is_active, COALESCE(tags,'[]'), COALESCE(device_rules,'{}'), expires_at, health_status, last_health_check, fallback_url, (password_hash <> '') AS has_password FROM urls WHERE creator_id = $1 AND is_active = TRUE ORDER BY position ASC, id DESC",
			creatorID,
		)
		if err != nil {
			return nil, err
		}
		part, err := scanLinks(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	return out, nil
}

// ListLinksByCreatorPrimary - sharded mode is the same as ListLinksByCreator
// (shardStore has no separate replica), including its error contract.
func (s *shardStore) ListLinksByCreatorPrimary(creatorID int64) ([]Link, error) {
	return s.ListLinksByCreator(creatorID)
}

// ReorderLinks sets position per short_code in caller order. No cross-shard
// transaction exists (shards are separate connections), so each shard updates
// independently - documented limitation; Fase 9 runs on SingleStore anyway.
// Returns sql.ErrConnDone when a target shard is missing, sql.ErrNoRows
// when a code is unknown or not owned by creatorID (404 in the handler);
// an empty order is a successful no-op.
func (s *shardStore) ReorderLinks(creatorID int64, order []string) error {
	for pos, code := range order {
		shardIdx := shortener.ShardKey(code, s.numShards)
		db, ok := s.shards[shardIdx]
		if !ok {
			return sql.ErrConnDone
		}
		res, err := db.Exec("UPDATE urls SET position = $1 WHERE short_code = $2 AND creator_id = $3", pos, code, creatorID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return sql.ErrNoRows
		}
	}
	return nil
}

// LogClick routes the event write to the short_code's shard (log and counters
// in one transaction, same semantics as SingleStore - see that
// implementation). Unique clicks (Fase 13): when IsUnique both counters rise
// (total and unique); repeat visits only increase click_count.
// Returns sql.ErrConnDone when the short_code's shard is missing; any error
// inside the transaction rolls back the event row and both counters together
// (defer tx.Rollback).
func (s *shardStore) LogClick(e ClickEvent) error {
	shardIdx := shortener.ShardKey(e.ShortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return sql.ErrConnDone
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		"INSERT INTO click_events (short_code, referrer, referrer_domain, is_unique, clicked_at, user_agent, device_type, referrer_type) VALUES ($1, NULLIF($2,''), NULLIF($3,''), $4, $5, NULLIF($6,''), NULLIF($7,''), NULLIF($8,''))",
		e.ShortCode, e.Referrer, e.ReferrerDomain, e.IsUnique, clickTime(e.ClickedAt),
		e.UserAgent, e.DeviceType, e.ReferrerType,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		"UPDATE urls SET click_count = click_count + 1, unique_click_count = unique_click_count + $1 WHERE short_code = $2",
		e.IsUniqueBool(), e.ShortCode,
	); err != nil {
		return err
	}
	return tx.Commit()
}

// IsUniqueBool converts the boolean to 0/1 for the SQL addition.
func (e ClickEvent) IsUniqueBool() int {
	if e.IsUnique {
		return 1
	}
	return 0
}

// clickTime returns the explicit click timestamp, or now when zero (defensive).
func clickTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now()
	}
	return t
}

// ClicksByDay merges per-day counts from all shards, then zero-fills the
// 30-day window (same helper as SingleStore - one fill logic everywhere).
// Missing shards are skipped (their clicks drop out of the merge instead of
// failing the page); query/scan errors are returned as-is.
func (s *shardStore) ClicksByDay(creatorID int64) ([]DayCount, error) {
	merged := make(map[string]int64)
	for i := 0; i < s.numShards; i++ {
		db, ok := s.shards[i]
		if !ok {
			continue
		}
		rows, err := db.Query(
			`SELECT TO_CHAR(ce.clicked_at, 'YYYY-MM-DD') AS day, COUNT(*) AS count
			 FROM click_events ce JOIN urls u ON u.short_code = ce.short_code
			 WHERE u.creator_id = $1 AND ce.clicked_at >= CURRENT_DATE - INTERVAL '29 days'
			 GROUP BY day`,
			creatorID,
		)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var day string
			var n int64
			if err := rows.Scan(&day, &n); err != nil {
				rows.Close()
				return nil, err
			}
			merged[day] += n
		}
		rows.Close()
	}
	return FillLast30Days(merged, time.Now()), nil
}

// AnalyticsSummary merges 4 dashboard stats across shards (same SUM pattern
// as ClicksByDay's multi-shard merge - one row per shard, add into totals).
// A shard query error aborts with the zero-value summary; missing shards are
// skipped, so a down shard silently lowers the numbers (same trade-off as
// the other merged reads).
func (s *shardStore) AnalyticsSummary(creatorID int64) (AnalyticsSummary, error) {
	var out AnalyticsSummary
	var last30, prev30 int64
	for i := 0; i < s.numShards; i++ {
		db, ok := s.shards[i]
		if !ok {
			continue
		}
		var links, clicks, unique int64
		if err := db.QueryRow(
			`SELECT COUNT(*) FILTER (WHERE is_active), COALESCE(SUM(click_count), 0)
			 FROM urls WHERE creator_id = $1`,
			creatorID,
		).Scan(&links, &clicks); err != nil {
			return AnalyticsSummary{}, err
		}
		out.TotalLinks += links
		out.TotalClicks += clicks

		var shLast, shPrev int64
		if err := db.QueryRow(
			`SELECT
				COUNT(*) FILTER (WHERE ce.is_unique AND ce.clicked_at >= CURRENT_DATE - INTERVAL '29 days'),
				COUNT(*) FILTER (WHERE ce.clicked_at >= CURRENT_DATE - INTERVAL '29 days'),
				COUNT(*) FILTER (WHERE ce.clicked_at >= CURRENT_DATE - INTERVAL '59 days'
					AND ce.clicked_at < CURRENT_DATE - INTERVAL '29 days')
			 FROM click_events ce
			 JOIN urls u ON u.short_code = ce.short_code
			 WHERE u.creator_id = $1`,
			creatorID,
		).Scan(&unique, &shLast, &shPrev); err != nil {
			return AnalyticsSummary{}, err
		}
		out.UniqueClicks30d += unique
		last30 += shLast
		prev30 += shPrev
	}
	out.GrowthPct = growthPct(last30, prev30)
	return out, nil
}

// sortBreakdown orders analytics buckets by descending count, then ascending
// label (deterministic order, so "unknown" does not jump between refreshes).
// Used by the cross-shard merge; the SingleStore path is already ordered by SQL.
func sortBreakdown(items []BreakdownItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Key < items[j].Key
	})
}

// DeviceBreakdown implements ShardStore by running one grouped query per shard
// and merging the buckets in Go (a single query cannot span shards; counts are
// summed, not picked - the same merge pattern as ClicksByDay/AnalyticsSummary).
// Missing shards are skipped; query/scan errors abort with a nil slice.
func (s *shardStore) DeviceBreakdown(creatorID int64, from, to time.Time) ([]BreakdownItem, error) {
	return s.breakdownAcrossShards(creatorID, from, to, "device")
}

// ReferrerBreakdown implements ShardStore for the referrer dimension: the
// merge works exactly like DeviceBreakdown (breakdownAcrossShards with kind
// "referrer"), including its error contract.
func (s *shardStore) ReferrerBreakdown(creatorID int64, from, to time.Time) ([]BreakdownItem, error) {
	return s.breakdownAcrossShards(creatorID, from, to, "referrer")
}

func (s *shardStore) breakdownAcrossShards(creatorID int64, from, to time.Time, kind string) ([]BreakdownItem, error) {
	column, nullBucket := "ce.device_type", "unknown"
	if kind == "referrer" {
		column, nullBucket = "ce.referrer_type", "other"
	}
	query := `SELECT COALESCE(NULLIF(` + column + `, ''), $4) AS bucket, COUNT(*) AS n
		 FROM click_events ce JOIN urls u ON u.short_code = ce.short_code
		 WHERE u.creator_id = $1 AND ce.clicked_at >= $2 AND ce.clicked_at < $3
		 GROUP BY 1`
	merged := map[string]int64{}
	for i := 0; i < s.numShards; i++ {
		shard, ok := s.shards[i]
		if !ok {
			continue
		}
		rows, err := shard.Query(query, creatorID, from, to, nullBucket)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var key string
			var n int64
			if err := rows.Scan(&key, &n); err != nil {
				rows.Close()
				return nil, err
			}
			merged[key] += n
		}
		rows.Close()
	}
	out := make([]BreakdownItem, 0, len(merged))
	for k, n := range merged {
		out = append(out, BreakdownItem{Key: k, Count: n})
	}
	sortBreakdown(out)
	return out, nil
}

// ClicksDaily merges per-day counts from every shard (values are summed, not
// picked) - like ClicksByDay but for an arbitrary range.
// Missing shards are skipped; query/scan errors abort with a nil map. No
// clicks in range means an empty map (no zero-fill here, see FillDays).
func (s *shardStore) ClicksDaily(creatorID int64, from, to time.Time) (map[string]int64, error) {
	merged := make(map[string]int64)
	for i := 0; i < s.numShards; i++ {
		shard, ok := s.shards[i]
		if !ok {
			continue
		}
		rows, err := shard.Query(
			`SELECT TO_CHAR(ce.clicked_at, 'YYYY-MM-DD') AS day, COUNT(*) AS n
			 FROM click_events ce JOIN urls u ON u.short_code = ce.short_code
			 WHERE u.creator_id = $1 AND ce.clicked_at >= $2 AND ce.clicked_at < $3
			 GROUP BY day`,
			creatorID, from, to,
		)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var day string
			var n int64
			if err := rows.Scan(&day, &n); err != nil {
				rows.Close()
				return nil, err
			}
			merged[day] += n
		}
		rows.Close()
	}
	return merged, nil
}

// LinkExportStats merges rows across shards per short_code (a code lives in
// exactly 1 shard, so merging is effectively concatenation; the limit is
// applied after merging so the truncation stays deterministic).
// Missing shards are skipped; query/scan errors abort with a nil slice; no
// matching links means an empty slice (the handler then reports zero rows).
func (s *shardStore) LinkExportStats(creatorID int64, from, to time.Time, limit int) ([]LinkExportRow, error) {
	var out []LinkExportRow
	for i := 0; i < s.numShards; i++ {
		shard, ok := s.shards[i]
		if !ok {
			continue
		}
		query := `SELECT u.short_code, u.original_url, COALESCE(u.tags,'[]'),
		       COUNT(ce.id) AS clicks, COUNT(ce.id) FILTER (WHERE ce.is_unique) AS uniques
			 FROM urls u
			 LEFT JOIN click_events ce ON ce.short_code = u.short_code
			    AND ce.clicked_at >= $2 AND ce.clicked_at < $3
			 WHERE u.creator_id = $1
			 GROUP BY u.id`
		rows, err := shard.Query(query, creatorID, from, to)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var r LinkExportRow
			var tags sql.NullString
			if err := rows.Scan(&r.ShortCode, &r.OriginalURL, &tags, &r.Clicks, &r.UniqueClicks); err != nil {
				rows.Close()
				return nil, err
			}
			r.Tags = parseTags(tags)
			out = append(out, r)
		}
		rows.Close()
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Clicks != out[j].Clicks {
			return out[i].Clicks > out[j].Clicks
		}
		return out[i].ShortCode < out[j].ShortCode
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ListClicks merges raw clicks across shards, sorts them descending by time,
// then trims to the limit (same logic as SingleStore).
// Missing shards are skipped; query/scan errors abort with a nil slice; no
// clicks in range means an empty result, not an error.
func (s *shardStore) ListClicks(creatorID int64, from, to time.Time, limit int) ([]ClickRow, error) {
	var out []ClickRow
	for i := 0; i < s.numShards; i++ {
		shard, ok := s.shards[i]
		if !ok {
			continue
		}
		rows, err := shard.Query(
			`SELECT ce.clicked_at, ce.short_code,
		       COALESCE(NULLIF(ce.device_type,''), 'unknown'),
		       COALESCE(NULLIF(ce.referrer_type,''), 'other'),
		       COALESCE(ce.referrer_domain, ''), ce.is_unique
			 FROM click_events ce JOIN urls u ON u.short_code = ce.short_code
			 WHERE u.creator_id = $1 AND ce.clicked_at >= $2 AND ce.clicked_at < $3`,
			creatorID, from, to,
		)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var c ClickRow
			if err := rows.Scan(&c.ClickedAt, &c.ShortCode, &c.DeviceType, &c.ReferrerType, &c.ReferrerDomain, &c.IsUnique); err != nil {
				rows.Close()
				return nil, err
			}
			out = append(out, c)
		}
		rows.Close()
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].ClickedAt.Equal(out[j].ClickedAt) {
			return out[i].ClickedAt.After(out[j].ClickedAt)
		}
		return out[i].ShortCode < out[j].ShortCode
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// AnalyticsFreshness: every shard IS the PRIMARY - there is no replication
// path to report (see SingleStore.analyticsRead for the dual-DB version).
// It never returns an error.
func (s *shardStore) AnalyticsFreshness() (string, error) { return "primary", nil }

// ClaimLinks routes each code to its shard (same IS NULL guard per row;
// no cross-shard transaction - documented limitation, see ReorderLinks).
// Returns sql.ErrConnDone when a target shard is missing (the count so far
// is discarded); unknown or already-owned codes are simply not counted and
// are never an error - the handler reports {"claimed": n}.
func (s *shardStore) ClaimLinks(creatorID int64, codes []string) (int64, error) {
	var claimed int64
	for _, code := range codes {
		shardIdx := shortener.ShardKey(code, s.numShards)
		db, ok := s.shards[shardIdx]
		if !ok {
			return 0, sql.ErrConnDone
		}
		res, err := db.Exec("UPDATE urls SET creator_id = $1 WHERE short_code = $2 AND creator_id IS NULL", creatorID, code)
		if err != nil {
			return 0, err
		}
		if n, _ := res.RowsAffected(); n == 1 {
			claimed++
		}
	}
	return claimed, nil
}

// UpdateLink implements ShardStore - routes to the shard, owner-scoped.
// Returns sql.ErrConnDone when the shard is missing, sql.ErrNoRows when the
// code is unknown or not owned by creatorID (the handler maps that to 404).
// Empty JSON inputs are normalized to "{}"/"[]" before writing.
func (s *shardStore) UpdateLink(creatorID int64, shortCode, deviceRulesJSON, tagsJSON string) error {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return sql.ErrConnDone
	}
	if strings.TrimSpace(deviceRulesJSON) == "" {
		deviceRulesJSON = "{}"
	}
	if strings.TrimSpace(tagsJSON) == "" {
		tagsJSON = "[]"
	}
	res, err := db.Exec(
		"UPDATE urls SET device_rules = $1, tags = $2 WHERE short_code = $3 AND creator_id = $4",
		deviceRulesJSON, tagsJSON, shortCode, creatorID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// SetFeaturedLink implements ShardStore. The semantics match SingleStore (one
// radio per creator), BUT there is no cross-shard transaction (shards are
// separate connections): "unfeature all" iterates shard by shard before the
// target is set - if the process dies in between, two links may remain
// featured. Documented limitation; Fase 9 runs on the transactional
// SingleStore (see its comment).
// Returns sql.ErrConnDone when the target shard is missing; an error from
// the "unfeature all" sweep aborts before the target link is set; featuring
// a code that is unknown or not owned yields sql.ErrNoRows.
func (s *shardStore) SetFeaturedLink(creatorID int64, shortCode string, featured bool) error {
	if featured {
		for i := 0; i < s.numShards; i++ {
			db, ok := s.shards[i]
			if !ok {
				continue
			}
			if _, err := db.Exec("UPDATE urls SET is_featured = FALSE WHERE creator_id = $1", creatorID); err != nil {
				return err
			}
		}
	}
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return sql.ErrConnDone
	}
	res, err := db.Exec(
		"UPDATE urls SET is_featured = $1 WHERE short_code = $2 AND creator_id = $3",
		featured, shortCode, creatorID,
	)
	if err != nil {
		return err
	}
	if featured {
		if n, _ := res.RowsAffected(); n != 1 {
			return sql.ErrNoRows
		}
	}
	return nil
}

// shardFor returns the connection of the link's shard (shared helper of the
// lifecycle methods DeleteLink / SetLinkActive / UpdateLink / CreateURL) and
// removes the repetitive "compute shard, look up db, bail if missing"
// preamble. Trade-off: it hides WHERE the shard lives, but every caller below
// is indifferent - they only need the correct db for short routing.
func (s *shardStore) shardFor(shortCode string) (*sql.DB, error) {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return nil, sql.ErrConnDone
	}
	return db, nil
}

// DeleteLink implements ShardStore - permanent delete, owner-scoped. Mirrors
// SingleStore.DeleteLink: owner-scoped WHERE plus related-row cleanup.
// Trade-off (serial here): deletes happen sequentially across shards - if the
// process dies midway, some click_events might be orphaned. Acceptable: a hard
// delete is deliberately rare and the FK is ON DELETE CASCADE (migration 01).
// Returns sql.ErrConnDone when the shard is missing, sql.ErrNoRows when the
// code is unknown or not owned (404 in the handler); a transaction error
// rolls the delete back (defer tx.Rollback).
func (s *shardStore) DeleteLink(creatorID int64, shortCode string) error {
	db, err := s.shardFor(shortCode)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(
		"DELETE FROM urls WHERE short_code = $1 AND creator_id = $2",
		shortCode, creatorID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

// SetLinkActive implements ShardStore - owner-scoped broadcast of is_active
// (Fase 13 lifecycle). Trade-off: no transaction across shards - if the two
// rows fight feature state mid-write, both may briefly be active; scoped to
// the single short_code so only one shard is touched per call.
// Returns sql.ErrConnDone when the shard is missing, sql.ErrNoRows when the
// code is unknown or not owned (404 in the handler).
func (s *shardStore) SetLinkActive(creatorID int64, shortCode string, active bool) error {
	db, err := s.shardFor(shortCode)
	if err != nil {
		return err
	}
	res, err := db.Exec(
		"UPDATE urls SET is_active = $1 WHERE short_code = $2 AND creator_id = $3",
		active, shortCode, creatorID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// StoreAPIKey inserts an API key row for creatorID and returns the new key id
// (an empty label is stored as NULL). API key rows live on shard 0 together
// with creators (see NOTE at shardZero): keys are per-creator metadata, not
// per-URL data, so they follow the same no-sharding simplification as Fase
// 9's creators table.
// Returns sql.ErrConnDone when shard 0 is unavailable; otherwise the driver
// error from INSERT ... RETURNING as-is.
func (s *shardStore) StoreAPIKey(creatorID int64, keyHash, label string) (int64, error) {
	db, ok := s.shardZero()
	if !ok {
		return 0, sql.ErrConnDone
	}
	var id int64
	err := db.QueryRow(
		"INSERT INTO api_keys (creator_id, key_hash, label) VALUES ($1, $2, NULLIF($3,'')) RETURNING id",
		creatorID, keyHash, label,
	).Scan(&id)
	return id, err
}

// ListAPIKeys returns the creator's API keys, newest first, from shard 0
// (see StoreAPIKey for why keys live there). The secret itself is never
// stored: rows only carry the SHA-256 hash, label, and timestamps, and the
// hash is not even selected here because the dashboard does not need it.
// Returns sql.ErrConnDone when shard 0 is unavailable; query/scan errors are
// returned as-is; a creator with no keys gets an empty result, not an error.
func (s *shardStore) ListAPIKeys(creatorID int64) ([]APIKey, error) {
	db, ok := s.shardZero()
	if !ok {
		return nil, sql.ErrConnDone
	}
	rows, err := db.Query(
		"SELECT id, creator_id, label, created_at, last_used_at FROM api_keys WHERE creator_id = $1 ORDER BY created_at DESC",
		creatorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.CreatorID, &k.Label, &k.CreatedAt, &k.LastUsedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// DeleteAPIKey removes the key with the given id, scoped to creatorID so one
// account can never delete another account's key. Returns sql.ErrNoRows when
// the key does not exist or is not owned by creatorID; the handler maps that
// to 404 without revealing which of the two cases occurred.
func (s *shardStore) DeleteAPIKey(creatorID int64, keyID int64) error {
	db, ok := s.shardZero()
	if !ok {
		return sql.ErrConnDone
	}
	res, err := db.Exec("DELETE FROM api_keys WHERE id = $1 AND creator_id = $2", keyID, creatorID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// GetAPIKeyByHash loads the API key whose stored SHA-256 hash equals keyHash;
// the plaintext key is never stored, so this lookup is how the public API
// identifies its caller. Returns sql.ErrNoRows when no key matches (unknown
// or deleted key).
func (s *shardStore) GetAPIKeyByHash(keyHash string) (APIKey, error) {
	db, ok := s.shardZero()
	if !ok {
		return APIKey{}, sql.ErrConnDone
	}
	var k APIKey
	err := db.QueryRow(
		"SELECT id, creator_id, label, created_at, last_used_at FROM api_keys WHERE key_hash = $1",
		keyHash,
	).Scan(&k.ID, &k.CreatorID, &k.Label, &k.CreatedAt, &k.LastUsedAt)
	return k, err
}

// TouchAPIKeyLastUsed stamps last_used_at after a successful public-API call
// so the dashboard can show when each key was last used. It is best-effort:
// callers log the error and continue, because the API response itself has
// already succeeded by then (handlers_apikeys.go).
func (s *shardStore) TouchAPIKeyLastUsed(keyID int64) error {
	db, ok := s.shardZero()
	if !ok {
		return sql.ErrConnDone
	}
	_, err := db.Exec("UPDATE api_keys SET last_used_at = NOW() WHERE id = $1", keyID)
	return err
}

// nullableInt converts *int64 to driver value: nil -> NULL.
func nullableInt(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

// nullableTime converts *time.Time to driver value: nil -> NULL. The value is
// passed through unchanged - the caller MUST already be in UTC (the TIMESTAMP
// without time zone column stores whatever it receives; see the expires_at
// note on Link).
func nullableTime(v *time.Time) any {
	if v == nil {
		return nil
	}
	return *v
}

// nullableTimeZero is nullableTime for a non-pointer value: the zero time
// means "clear the column" (health_notified_at reset when a link heals).
// The caller MUST already be in UTC.
func nullableTimeZero(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

// GetURL implements ShardStore - routes to the correct shard based on short_code hash.
// Returns sql.ErrConnDone when the shard is unavailable, sql.ErrNoRows when
// the code is unknown.
func (s *shardStore) GetURL(shortCode string) (string, error) {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return "", sql.ErrConnDone
	}
	var url string
	err := db.QueryRow("SELECT original_url FROM urls WHERE short_code = $1", shortCode).Scan(&url)
	if err != nil {
		return "", err
	}
	return url, nil
}

// GetLink implements ShardStore - routes to the shard and returns the row
// needed for Smart Link device routing (original_url + device_rules) plus
// lifecycle state is_active (redirect -> 410 when disabled, Fase 13).
// Returns sql.ErrConnDone when the shard is unavailable, sql.ErrNoRows when
// the code is unknown.
func (s *shardStore) GetLink(shortCode string) (Link, error) {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return Link{}, sql.ErrConnDone
	}
	var l Link
	var rules sql.NullString
	var expires sql.NullTime
	var lastCheck sql.NullTime
	var notified sql.NullTime
	var creatorID sql.NullInt64
	err := db.QueryRow(
		"SELECT short_code, original_url, COALESCE(device_rules,'{}'), is_active, expires_at, creator_id, health_status, last_health_check, fallback_url, health_notified_at, password_hash FROM urls WHERE short_code = $1",
		shortCode,
	).Scan(&l.ShortCode, &l.OriginalURL, &rules, &l.IsActive, &expires, &creatorID, &l.HealthStatus, &lastCheck, &l.FallbackURL, &notified, &l.PasswordHash)
	if err != nil {
		return Link{}, err
	}
	l.DeviceRules = parseRules(rules)
	if expires.Valid {
		u := expires.Time.UTC()
		l.ExpiresAt = &u
	}
	if lastCheck.Valid {
		u := lastCheck.Time.UTC()
		l.LastHealthCheck = &u
	}
	if notified.Valid {
		u := notified.Time.UTC()
		l.HealthNotifiedAt = &u
	}
	if creatorID.Valid {
		id := creatorID.Int64
		l.CreatorID = &id
	}
	l.HasPassword = l.PasswordHash != ""
	return l, nil
}

// IncrementClickCount implements ShardStore - routes to the correct shard.
// Returns sql.ErrConnDone when the shard is unavailable; an unknown code is
// still a success (0-row UPDATE, no error).
func (s *shardStore) IncrementClickCount(shortCode string) error {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return sql.ErrConnDone
	}
	_, err := db.Exec("UPDATE urls SET click_count = click_count + 1 WHERE short_code = $1", shortCode)
	return err
}

// GetShard returns the underlying DB for a specific short_code
func (s *shardStore) GetShard(shortCode string) *sql.DB {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return nil
	}
	return db
}

// ListLinks returns all links across shards (merged in shard index order).
// Missing shards are skipped; query/scan errors abort with a nil slice; no
// links anywhere means an empty result, not an error.
func (s *shardStore) ListLinks() ([]Link, error) {
	var out []Link
	for i := 0; i < s.numShards; i++ {
		db, ok := s.shards[i]
		if !ok {
			continue
		}
		rows, err := db.Query("SELECT short_code, original_url, click_count, unique_click_count, position, is_featured, is_active, COALESCE(tags,'[]'), COALESCE(device_rules,'{}'), expires_at, health_status, last_health_check, fallback_url, (password_hash <> '') AS has_password FROM urls ORDER BY id DESC")
		if err != nil {
			return nil, err
		}
		part, err := scanLinks(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	return out, nil
}

// DB returns the first available shard's connection (for backward compatibility)
func (s *shardStore) DB() *sql.DB {
	for _, db := range s.shards {
		return db
	}
	return nil
}

// NewShardStoreFromDSNs creates a sharded store from explicit DSNs (one per
// shard). DSNs come from env (SHARD_DSNS, comma-separated), never hardcoded
// hosts, so the shard list can change without touching code (adding a shard =
// edit the env and restart) and every environment (local/compose) can use its
// own hosts. Trade-off: the comma-separated env format is fragile (a stray
// space or typo silently drops a shard) and the DSN count defines numShards,
// so a wrong config misroutes every write. Service discovery (Consul/etcd) was
// considered and rejected as overkill for local learning.
// Returns the driver's open error for a malformed DSN; connections are NOT
// pinged here, so an unreachable shard only surfaces later as
// sql.ErrConnDone on first use.
func NewShardStoreFromDSNs(dsns []string) (*shardStore, error) {
	ss := &shardStore{
		numShards: len(dsns),
		shards:    make(map[int]*sql.DB),
	}
	for i, dsn := range dsns {
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			return nil, err
		}
		ss.shards[i] = db
	}
	return ss, nil
}

// SingleStore is the one-database store for baseline mode (Fase 0): no
// sharding, no mandatory replica. Writes always go to the primary; reads go to
// the replica ONLY when a replica DSN is configured and reachable (Fase 4
// read/write splitting). It implements ShardStore so handlers need not know
// which backend is in use. Trade-off: if the replica is down at startup, reads
// fall back to the primary with a warning (availability first; reads stay
// consistent with the primary). The startup ping means the API crash-loops
// while the primary is not ready - handled through the restart policy. A
// per-query health check plus circuit breaker was rejected as too complex for
// a baseline.
type SingleStore struct {
	primary    *sql.DB
	replica    *sql.DB
	useReplica bool
}

// NewSingleStore opens the primary DB and, optionally, a read replica.
// Empty replicaDSN = reads fall back to primary (pure Fase 0 baseline).
// Returns the driver's open error for the primary or for a malformed replica
// DSN (the primary is closed first when the replica open fails); an
// unreachable replica is not an error - it logs a warning and reads fall back
// to the primary. The primary is not pinged here.
func NewSingleStore(primaryDSN, replicaDSN string) (*SingleStore, error) {
	primary, err := sql.Open("pgx", primaryDSN)
	if err != nil {
		return nil, err
	}
	s := &SingleStore{primary: primary}
	if replicaDSN != "" {
		rep, err := sql.Open("pgx", replicaDSN)
		if err != nil {
			primary.Close()
			return nil, err
		}
		if err := rep.Ping(); err != nil {
			log.Printf("Warning: replica unreachable, reads fall back to primary: %v", err)
			rep.Close()
		} else {
			s.replica = rep
			s.useReplica = true
		}
	}
	return s, nil
}

// readDB returns the replica when enabled, otherwise the primary.
func (s *SingleStore) readDB() *sql.DB {
	if s.useReplica && s.replica != nil {
		return s.replica
	}
	return s.primary
}

// CreateURL writes to the primary. Returns the driver error as-is; a
// duplicate short_code surfaces as a unique-constraint violation.
func (s *SingleStore) CreateURL(shortCode, originalURL string, creatorID *int64, tagsJSON string, expiresAt *time.Time, passwordHash string) error {
	if tagsJSON == "" {
		tagsJSON = "[]"
	}
	_, err := s.primary.Exec("INSERT INTO urls (short_code, original_url, creator_id, tags, expires_at, password_hash) VALUES ($1, $2, $3, $4, $5, $6)", shortCode, originalURL, nullableInt(creatorID), tagsJSON, nullableTime(expiresAt), passwordHash)
	return err
}

// CreateURLsBatch runs a bulk import (10-100 URLs) in ONE PRIMARY transaction:
// every row commits together. Unique short_code conflicts are skipped PER ROW
// through ON CONFLICT DO NOTHING - the transaction never aborts (no savepoint
// required), so one duplicate URL cannot cancel the other 99. The result
// mirrors items order: the stored short code, or "" when conflicted (the
// caller reports it as a per-row error).
func (s *SingleStore) CreateURLsBatch(creatorID *int64, items []BulkURL) ([]string, error) {
	codes := make([]string, len(items))
	if len(items) == 0 {
		return codes, nil
	}
	tx, err := s.primary.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	for i, it := range items {
		tagsJSON := it.TagsJSON
		if tagsJSON == "" {
			tagsJSON = "[]"
		}
		res, err := tx.Exec(
			"INSERT INTO urls (short_code, original_url, creator_id, tags, expires_at) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (short_code) DO NOTHING",
			it.ShortCode, it.OriginalURL, nullableInt(creatorID), tagsJSON, nullableTime(it.ExpiresAt),
		)
		if err != nil {
			return nil, err
		}
		if n, _ := res.RowsAffected(); n == 1 {
			codes[i] = it.ShortCode
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return codes, nil
}

// SetLinkExpiry writes or clears expires_at on the PRIMARY (read-your-own-
// writes: the expired / ends-in badge on the dashboard changes right after
// saving). Scoped by WHERE creator_id - someone else's link yields
// sql.ErrNoRows (404, not 403).
func (s *SingleStore) SetLinkExpiry(creatorID int64, shortCode string, expiresAt *time.Time) error {
	res, err := s.primary.Exec(
		"UPDATE urls SET expires_at = $3 WHERE short_code = $1 AND creator_id = $2",
		shortCode, creatorID, nullableTime(expiresAt),
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// SetLinkPassword writes or clears password_hash on the PRIMARY
// (read-your-own-writes: the lock badge appears right after saving).
// Owner-scoped -> sql.ErrNoRows for unknown/not-owned codes (404).
func (s *SingleStore) SetLinkPassword(creatorID int64, shortCode, passwordHash string) error {
	res, err := s.primary.Exec(
		"UPDATE urls SET password_hash = $3 WHERE short_code = $1 AND creator_id = $2",
		shortCode, creatorID, passwordHash,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// SetLinkFallback writes or clears fallback_url ("" = clear) for an owned
// link on the PRIMARY. Same owner-scoped contract as SetLinkPassword.
func (s *SingleStore) SetLinkFallback(creatorID int64, shortCode, fallbackURL string) error {
	res, err := s.primary.Exec(
		"UPDATE urls SET fallback_url = $3 WHERE short_code = $1 AND creator_id = $2",
		shortCode, creatorID, fallbackURL,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdateLinkHealth stamps the worker/manual check on the PRIMARY. Not
// owner-scoped (the checker runs system-wide). A zero notifiedAt clears the
// notification marker (healed -> the next breakage notifies again); a
// non-zero value keeps/sets it so a link that stays broken never spams the
// bell. Unknown code = 0-row UPDATE, not an error: the row may have been
// deleted between the due-list and the check.
func (s *SingleStore) UpdateLinkHealth(shortCode, status string, checkedAt, notifiedAt time.Time) error {
	_, err := s.primary.Exec(
		"UPDATE urls SET health_status = $2, last_health_check = $3, health_notified_at = $4 WHERE short_code = $1",
		shortCode, status, checkedAt, nullableTimeZero(notifiedAt),
	)
	return err
}

// ListLinksHealthDue picks the least-recently-checked ACTIVE links from the
// PRIMARY (NULLs first = never checked) so the worker spreads its 6h window
// fairly: no link hogs the batch while another never gets verified.
func (s *SingleStore) ListLinksHealthDue(limit int) ([]Link, error) {
	rows, err := s.primary.Query(
		"SELECT short_code, original_url, click_count, unique_click_count, position, is_featured, is_active, COALESCE(tags,'[]'), COALESCE(device_rules,'{}'), expires_at, health_status, last_health_check, fallback_url, (password_hash <> '') AS has_password FROM urls WHERE is_active = TRUE ORDER BY last_health_check NULLS FIRST, id LIMIT $1",
		limit,
	)
	if err != nil {
		return nil, err
	}
	return scanLinks(rows)
}

// CreateNotification inserts an in-app notification on the PRIMARY.
func (s *SingleStore) CreateNotification(creatorID int64, typ, shortCode, message string) (int64, error) {
	var id int64
	err := s.primary.QueryRow(
		"INSERT INTO notifications (creator_id, type, short_code, message) VALUES ($1, $2, $3, $4) RETURNING id",
		creatorID, typ, shortCode, message,
	).Scan(&id)
	return id, err
}

// ListNotifications returns the newest limit rows for one creator from the
// PRIMARY (the bell must show a fresh feed right after a break is detected).
// Empty result (no rows) is returned as an empty slice, not an error.
func (s *SingleStore) ListNotifications(creatorID int64, limit int) ([]Notification, error) {
	rows, err := s.primary.Query(
		"SELECT id, type, short_code, message, read, created_at FROM notifications WHERE creator_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2",
		creatorID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.ShortCode, &n.Message, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		n.CreatedAt = n.CreatedAt.UTC()
		out = append(out, n)
	}
	return out, rows.Err()
}

// CountUnreadNotifications drives the bell badge (PRIMARY, read-your-own-
// writes: marking read updates the count immediately).
func (s *SingleStore) CountUnreadNotifications(creatorID int64) (int64, error) {
	var n int64
	err := s.primary.QueryRow(
		"SELECT COUNT(*) FROM notifications WHERE creator_id = $1 AND read = FALSE",
		creatorID,
	).Scan(&n)
	return n, err
}

// MarkNotificationRead owner-scoped: unknown or foreign id -> sql.ErrNoRows
// (404 without leaking which ids exist).
func (s *SingleStore) MarkNotificationRead(creatorID, id int64) error {
	res, err := s.primary.Exec(
		"UPDATE notifications SET read = TRUE WHERE id = $1 AND creator_id = $2",
		id, creatorID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// GetURL reads from the replica when enabled, otherwise the primary.
// Returns sql.ErrNoRows when the code is unknown (404 in the redirect
// handler).
func (s *SingleStore) GetURL(shortCode string) (string, error) {
	var url string
	err := s.readDB().QueryRow("SELECT original_url FROM urls WHERE short_code = $1", shortCode).Scan(&url)
	if err != nil {
		return "", err
	}
	return url, nil
}

// GetLink reads the full redirect row (original_url + device_rules +
// is_active + expires_at + migration 17: creator_id, health_status,
// fallback_url, health_notified_at, password_hash) - needed by the Smart
// Link redirect path to route by device, to answer 410 for disabled links,
// to gate the password form, and by the manual health trigger for the owner
// check. Same read path as GetURL, including its error contract
// (sql.ErrNoRows when the code is unknown). The password_hash selected here
// NEVER reaches JSON (Link.PasswordHash is json:"-").
func (s *SingleStore) GetLink(shortCode string) (Link, error) {
	var l Link
	var rules sql.NullString
	var expires sql.NullTime
	var lastCheck sql.NullTime
	var notified sql.NullTime
	var creatorID sql.NullInt64
	err := s.readDB().QueryRow(
		"SELECT short_code, original_url, COALESCE(device_rules,'{}'), is_active, expires_at, creator_id, health_status, last_health_check, fallback_url, health_notified_at, password_hash FROM urls WHERE short_code = $1",
		shortCode,
	).Scan(&l.ShortCode, &l.OriginalURL, &rules, &l.IsActive, &expires, &creatorID, &l.HealthStatus, &lastCheck, &l.FallbackURL, &notified, &l.PasswordHash)
	if err != nil {
		return Link{}, err
	}
	l.DeviceRules = parseRules(rules)
	if expires.Valid {
		u := expires.Time.UTC()
		l.ExpiresAt = &u
	}
	if lastCheck.Valid {
		u := lastCheck.Time.UTC()
		l.LastHealthCheck = &u
	}
	if notified.Valid {
		u := notified.Time.UTC()
		l.HealthNotifiedAt = &u
	}
	if creatorID.Valid {
		id := creatorID.Int64
		l.CreatorID = &id
	}
	l.HasPassword = l.PasswordHash != ""
	return l, nil
}

// IncrementClickCount writes to the primary. Never fails for an unknown code
// (a 0-row UPDATE is still success); only driver/connection errors are
// returned.
func (s *SingleStore) IncrementClickCount(shortCode string) error {
	_, err := s.primary.Exec("UPDATE urls SET click_count = click_count + 1 WHERE short_code = $1", shortCode)
	return err
}

// GetShard returns the DB used for reads (single connection, no sharding).
func (s *SingleStore) GetShard(shortCode string) *sql.DB {
	return s.readDB()
}

// ListLinks returns all links, read from the replica when enabled
// (Fase 8 dashboard reads from the read replica). A query error returns
// nil; no rows yields an empty slice, not an error.
func (s *SingleStore) ListLinks() ([]Link, error) {
	rows, err := s.readDB().Query("SELECT short_code, original_url, click_count, unique_click_count, position, is_featured, is_active, COALESCE(tags,'[]'), COALESCE(device_rules,'{}'), expires_at, health_status, last_health_check, fallback_url, (password_hash <> '') AS has_password FROM urls ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	return scanLinks(rows)
}

// DB returns the primary connection.
func (s *SingleStore) DB() *sql.DB {
	return s.primary
}

// ClaimLinks attaches previously anonymous links to the logged-in creator. The
// write MUST stay conditional (WHERE creator_id IS NULL) rather than a blind
// update: the server cannot know which browser created which anonymous link
// (stateless HTTP, no identity before login), so the candidate list comes from
// the creating browser's localStorage. An unconditional update would let ANY
// user take over someone else's link merely by guessing its short code; the IS
// NULL condition is the security boundary - owned rows stay untouched and a
// double claim is harmless (0 affected). The whole batch runs in one
// transaction (all-or-nothing) so a partial claim cannot confuse the client
// ("3 of 5 landed, which ones?"). Claiming every NULL link for whoever logs in
// first would let the first registrant sweep up EVERYONE's anonymous links and
// must never be implemented.
// Returns the begin/exec/commit error as-is (any failure rolls the batch back
// via defer tx.Rollback); unknown or already-owned codes never fail - they
// only lower the returned count.
func (s *SingleStore) ClaimLinks(creatorID int64, codes []string) (int64, error) {
	tx, err := s.primary.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var claimed int64
	for _, code := range codes {
		res, err := tx.Exec("UPDATE urls SET creator_id = $1 WHERE short_code = $2 AND creator_id IS NULL", creatorID, code)
		if err != nil {
			return 0, err
		}
		if n, _ := res.RowsAffected(); n == 1 {
			claimed++
		}
	}
	return claimed, tx.Commit()
}

// UpdateLink overwrites device_rules + tags for one link, scoped to the owner
// (WHERE creator_id). Rows scoped elsewhere / unknown -> sql.ErrNoRows -> 404.
// Both values are full-replace (matches the profile endpoint philosophy: the
// client sends the complete new state, no PATCH merge ambiguity).
func (s *SingleStore) UpdateLink(creatorID int64, shortCode, deviceRulesJSON, tagsJSON string) error {
	if strings.TrimSpace(deviceRulesJSON) == "" {
		deviceRulesJSON = "{}"
	}
	if strings.TrimSpace(tagsJSON) == "" {
		tagsJSON = "[]"
	}
	res, err := s.primary.Exec(
		"UPDATE urls SET device_rules = $1, tags = $2 WHERE short_code = $3 AND creator_id = $4",
		deviceRulesJSON, tagsJSON, shortCode, creatorID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// SetFeaturedLink makes `shortCode` the creator's single featured link (a
// radio, not a checkbox): featured=true unfeatures EVERY link of that account
// first, then promotes the target - two writes in ONE transaction so an
// observer never sees 0 or 2 featured links (atomic, like ClaimLinks). The
// target statement is scoped WITHIN creator_id: someone else's link or an
// empty code -> sql.ErrNoRows -> 404. featured=false is idempotent (unfeature
// the target; affected 0 = already not featured = success, not an error).
// shardStore imitates this semantics without a cross-shard transaction (see
// its comment).
func (s *SingleStore) SetFeaturedLink(creatorID int64, shortCode string, featured bool) error {
	tx, err := s.primary.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if featured {
		if _, err := tx.Exec("UPDATE urls SET is_featured = FALSE WHERE creator_id = $1", creatorID); err != nil {
			return err
		}
	}
	res, err := tx.Exec(
		"UPDATE urls SET is_featured = $1 WHERE short_code = $2 AND creator_id = $3",
		featured, shortCode, creatorID,
	)
	if err != nil {
		return err
	}
	if featured {
		if n, _ := res.RowsAffected(); n != 1 {
			return sql.ErrNoRows
		}
	}
	return tx.Commit()
}

// DeleteLink implements ShardStore on the single non-sharded store - the same
// owner-scoped hard-delete semantics as shardStore.DeleteLink, in ONE
// transaction: click_events first (FK to urls), then urls. The worker uses
// SingleStore as a ShardStore (cmd/worker), so the delete lifecycle has to
// work here too.
// Returns sql.ErrNoRows when the code is unknown or not owned (404 in the
// handler); any earlier error rolls the transaction back (defer tx.Rollback).
func (s *SingleStore) DeleteLink(creatorID int64, shortCode string) error {
	tx, err := s.primary.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM click_events WHERE short_code = $1", shortCode); err != nil {
		return err
	}
	res, err := tx.Exec(
		"DELETE FROM urls WHERE short_code = $1 AND creator_id = $2",
		shortCode, creatorID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

// SetLinkActive implements ShardStore on the single store - owner-scoped
// toggle of is_active (Fase 14 active/inactive lifecycle). Mirrors
// shardStore.SetLinkActive; the single store needs no shard routing.
// Returns sql.ErrNoRows when the code is unknown or not owned (404 in the
// handler); driver errors are returned as-is.
func (s *SingleStore) SetLinkActive(creatorID int64, shortCode string, active bool) error {
	res, err := s.primary.Exec(
		"UPDATE urls SET is_active = $1 WHERE short_code = $2 AND creator_id = $3",
		active, shortCode, creatorID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// StoreAPIKey keeps only the hash, so the real key never touches the
// database: if the database leaks, the key cannot be used directly. The
// "jjk_" + 32 hex (128 bit) format makes brute-forcing SHA-256 computationally
// impossible (unlike user passwords, which need bcrypt/argon2 because their
// entropy is low - an API key is fully random, so a fast hash suffices, and
// that speed is mandatory because the hash is checked on EVERY request).
// Trade-off: SHA-256 is deliberately not a slow function (unlike bcrypt) -
// a random 128-bit key can never be brute-forced through an offline hash,
// while API keys are authenticated per request (bcrypt per request = +100ms
// latency per call).
// Returns the driver error from INSERT ... RETURNING as-is (an empty label
// is stored as NULL, so it never fails validation here).
func (s *SingleStore) StoreAPIKey(creatorID int64, keyHash, label string) (int64, error) {
	var id int64
	err := s.primary.QueryRow(
		"INSERT INTO api_keys (creator_id, key_hash, label) VALUES ($1, $2, NULLIF($3,'')) RETURNING id",
		creatorID, keyHash, label,
	).Scan(&id)
	return id, err
}

// ListAPIKeys returns the caller's keys. Writes (creator_id scope) read from
// primary - keys are low-volume and must be instantly consistent for the
// list/delete flow, so replica staleness is not worth it here.
// Query/scan errors are returned as-is; a creator with no keys gets an empty
// result, not an error.
func (s *SingleStore) ListAPIKeys(creatorID int64) ([]APIKey, error) {
	rows, err := s.primary.Query(
		"SELECT id, creator_id, label, created_at, last_used_at FROM api_keys WHERE creator_id = $1 ORDER BY created_at DESC",
		creatorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.CreatorID, &k.Label, &k.CreatedAt, &k.LastUsedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// DeleteAPIKey removes one key, scoped to the owner. Unknown id or another
// creator's key -> sql.ErrNoRows (handler maps this to 404, no key-id leak).
func (s *SingleStore) DeleteAPIKey(creatorID int64, keyID int64) error {
	res, err := s.primary.Exec("DELETE FROM api_keys WHERE id = $1 AND creator_id = $2", keyID, creatorID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// GetAPIKeyByHash looks a hash up for the API-key auth middleware. Read path
// (replica when enabled) because every /api/v1 request hits this.
// Returns sql.ErrNoRows when no key matches (unknown, revoked, or served by
// a lagging replica right after creation); the middleware then rejects the
// request.
func (s *SingleStore) GetAPIKeyByHash(keyHash string) (APIKey, error) {
	var k APIKey
	err := s.readDB().QueryRow(
		"SELECT id, creator_id, label, created_at, last_used_at FROM api_keys WHERE key_hash = $1",
		keyHash,
	).Scan(&k.ID, &k.CreatorID, &k.Label, &k.CreatedAt, &k.LastUsedAt)
	return k, err
}

// TouchAPIKeyLastUsed stamps last_used_at on a key (run after successful API
// auth). Fire-and-forget-friendly: error is logged by the caller, never
// fails the request - the request already succeeded by then.
func (s *SingleStore) TouchAPIKeyLastUsed(keyID int64) error {
	_, err := s.primary.Exec("UPDATE api_keys SET last_used_at = NOW() WHERE id = $1", keyID)
	return err
}

// DayCount is one chart point: clicks on a calendar day (YYYY-MM-DD).
type DayCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// AnalyticsSummary is the payload of GET /api/analytics/summary - the 4 stat
// numbers of the Ringkasan card. GrowthPct compares the clicks of the last 30
// days against the previous 30 (rounded, %); UniqueClicks30d is a COUNT of
// click_events rows with is_unique inside the 30-day window (not the all-time
// unique_click_count counter).
type AnalyticsSummary struct {
	TotalLinks      int64 `json:"total_links"`
	TotalClicks     int64 `json:"total_clicks"`
	UniqueClicks30d int64 `json:"unique_clicks_30d"`
	GrowthPct       int   `json:"growth_pct"`
}

// growthPct computes (last-prev)/prev*100, rounded. When prev==0 there is no
// baseline: 0 if last is also 0, 100 when there are clicks (growth from zero).
func growthPct(last, prev int64) int {
	if prev == 0 {
		if last > 0 {
			return 100
		}
		return 0
	}
	return int(math.Round(float64(last-prev) / float64(prev) * 100))
}

// dailyCounts aggregates per day inside the database instead of pulling every
// click_events row into the application - the same principle as avoiding N+1,
// in summary form: when only 30 numbers are needed, transferring thousands of
// raw rows over the network only to discard 99% of them in process memory is
// wasteful (databases are built for aggregation through index + GROUP BY,
// applications are not). Trade-off: business logic is split - GROUP BY in SQL,
// zero-filling of empty days in Go (see FillLast30Days) - because SQL without
// generate_series would drop days without clicks and leave holes in the chart.
// Alternatives: generate_series in SQL (one complete query), or fetching raw
// rows and aggregating in Go (bandwidth-hungry, rejected).
func (s *SingleStore) dailyCounts(creatorID int64) (map[string]int64, error) {
	rows, err := s.readDB().Query(
		`SELECT TO_CHAR(ce.clicked_at, 'YYYY-MM-DD') AS day, COUNT(*) AS count
		 FROM click_events ce JOIN urls u ON u.short_code = ce.short_code
		 WHERE u.creator_id = $1 AND ce.clicked_at >= CURRENT_DATE - INTERVAL '29 days'
		 GROUP BY day`,
		creatorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int64)
	for rows.Next() {
		var day string
		var n int64
		if err := rows.Scan(&day, &n); err != nil {
			return nil, err
		}
		out[day] = n
	}
	return out, nil
}

// ClicksByDay returns a complete 30-day window (today included) with zeros
// for days without clicks, read from the replica when enabled.
// dailyCounts errors are returned unchanged; no clicks at all means every
// day is 0, not an error.
func (s *SingleStore) ClicksByDay(creatorID int64) ([]DayCount, error) {
	counts, err := s.dailyCounts(creatorID)
	if err != nil {
		return nil, err
	}
	return FillLast30Days(counts, time.Now()), nil
}

// AnalyticsSummary computes the 4 dashboard numbers in 2 aggregate queries
// (COUNT/SUM FILTER plus a 60-day window JOIN) instead of loading every link
// and event into the application. Trade-off: growth needs a 60-day window over
// click_events (ClicksByDay covers only 30 days) - a heavier query than the
// chart, though still O(days) through the (short_code, clicked_at) index;
// prev==0 yields growth 0 or 100 (no baseline). A materialized view or a daily
// precompute was rejected as overkill for an MVP.
// A query error aborts with the zero-value summary; an account with no links
// or no clicks yields zeros (growth 0 or 100 when there is no baseline).
func (s *SingleStore) AnalyticsSummary(creatorID int64) (AnalyticsSummary, error) {
	var out AnalyticsSummary
	err := s.readDB().QueryRow(
		`SELECT COUNT(*) FILTER (WHERE is_active), COALESCE(SUM(click_count), 0)
		 FROM urls WHERE creator_id = $1`,
		creatorID,
	).Scan(&out.TotalLinks, &out.TotalClicks)
	if err != nil {
		return AnalyticsSummary{}, err
	}

	var last30, prev30 int64
	err = s.readDB().QueryRow(
		`SELECT
			COUNT(*) FILTER (WHERE ce.is_unique AND ce.clicked_at >= CURRENT_DATE - INTERVAL '29 days'),
			COUNT(*) FILTER (WHERE ce.clicked_at >= CURRENT_DATE - INTERVAL '29 days'),
			COUNT(*) FILTER (WHERE ce.clicked_at >= CURRENT_DATE - INTERVAL '59 days'
				AND ce.clicked_at < CURRENT_DATE - INTERVAL '29 days')
		 FROM click_events ce
		 JOIN urls u ON u.short_code = ce.short_code
		 WHERE u.creator_id = $1`,
		creatorID,
	).Scan(&out.UniqueClicks30d, &last30, &prev30)
	if err != nil {
		return AnalyticsSummary{}, err
	}
	out.GrowthPct = growthPct(last30, prev30)
	return out, nil
}

// analyticsRead picks the REPLICA only while its lag stays within reason
// (< 1 hour), instead of blindly selecting the replica like the other read
// paths. The dashboard shows cumulative numbers that nobody chases within
// seconds (unlike read-your-own-writes on profile/links), so a few seconds of
// staleness are unnoticeable - but a replica stuck for hours would make "the
// last 30 days" look stale for no visible reason. The 1-hour threshold was
// chosen because both queries run occasionally rather than per redirect, so
// one extra query (pg_last_xact_replay_timestamp) costs nothing next to the
// large GROUP BY that follows. Trade-off: baseline mode (no replica) always
// stays PRIMARY - the data_per label follows that reality, not a hope. A
// standby that has never received WAL returns NULL and is treated as PRIMARY
// (fail-safe). Alternatives: always primary (honest but wasteful on a
// read-heavy path) or always replica (fast but silently stale).
func (s *SingleStore) analyticsRead() (*sql.DB, string) {
	if !s.useReplica || s.replica == nil {
		return s.primary, "primary"
	}
	var last sql.NullTime
	if err := s.replica.QueryRow("SELECT pg_last_xact_replay_timestamp()").Scan(&last); err != nil || !last.Valid {
		return s.primary, "primary"
	}
	if time.Since(last.Time) > time.Hour {
		return s.primary, "primary"
	}
	return s.replica, "replica"
}

// AnalyticsFreshness implements ShardStore - it reports the analytics read
// source for the data_per field of the dashboard response (see
// analyticsRead). It never returns an error.
func (s *SingleStore) AnalyticsFreshness() (string, error) {
	_, dataPer := s.analyticsRead()
	return dataPer, nil
}

// breakdownQuery is the single query used for BOTH breakdowns (device and
// referrer); the column is selected through placeholder $4 so the two SQL
// copies cannot drift apart. COALESCE covers rows that predate classification
// (migration 16 backfills, but NULL still maps to a sensible bucket).
func (s *SingleStore) breakdownQuery(creatorID int64, from, to time.Time, column, nullBucket string) ([]BreakdownItem, error) {
	// Identifier allowlist (SQL-injection guard): a column name cannot be a
	// bound parameter, so the two supported identifiers are hard-coded and
	// every other value is rejected BEFORE the query is assembled. Callers
	// pass literals today, but the signature accepts any string: the allowlist
	// makes that safety a property of this function, not of its call sites.
	switch column {
	case "ce.device_type", "ce.referrer_type":
	default:
		return nil, fmt.Errorf("breakdownQuery: unsupported column %q", column)
	}
	// #nosec G202 -- `column` is restricted to the two literals above; it is
	// concatenated only after the allowlist check, never bound from input.
	query := `SELECT COALESCE(NULLIF(` + column + `, ''), $4) AS bucket, COUNT(*) AS n
		 FROM click_events ce JOIN urls u ON u.short_code = ce.short_code
		 WHERE u.creator_id = $1 AND ce.clicked_at >= $2 AND ce.clicked_at < $3
		 GROUP BY 1 ORDER BY 2 DESC, 1 ASC`
	db, _ := s.analyticsRead()
	rows, err := db.Query(query, creatorID, from, to, nullBucket)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BreakdownItem{}
	for rows.Next() {
		var it BreakdownItem
		if err := rows.Scan(&it.Key, &it.Count); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// DeviceBreakdown implements ShardStore: one grouped query on the single
// database (breakdownQuery maps the column and the null bucket).
// Query/scan errors are returned as-is; no matching rows means an empty
// slice (the chart renders nothing).
func (s *SingleStore) DeviceBreakdown(creatorID int64, from, to time.Time) ([]BreakdownItem, error) {
	return s.breakdownQuery(creatorID, from, to, "ce.device_type", "unknown")
}

// ReferrerBreakdown implements ShardStore for the referrer dimension: same
// query shape as DeviceBreakdown, different column and null bucket, same
// error contract (errors as-is, empty slice when nothing matches).
func (s *SingleStore) ReferrerBreakdown(creatorID int64, from, to time.Time) ([]BreakdownItem, error) {
	return s.breakdownQuery(creatorID, from, to, "ce.referrer_type", "other")
}

// ClicksDaily counts clicks per day in the range (no zero-fill - see
// FillDays). Reads use the analytics path (fresh replica or primary).
// Query/scan/rows errors are returned as-is; no clicks in range means an
// empty map, not an error.
func (s *SingleStore) ClicksDaily(creatorID int64, from, to time.Time) (map[string]int64, error) {
	db, _ := s.analyticsRead()
	rows, err := db.Query(
		`SELECT TO_CHAR(ce.clicked_at, 'YYYY-MM-DD') AS day, COUNT(*) AS n
		 FROM click_events ce JOIN urls u ON u.short_code = ce.short_code
		 WHERE u.creator_id = $1 AND ce.clicked_at >= $2 AND ce.clicked_at < $3
		 GROUP BY day`,
		creatorID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int64)
	for rows.Next() {
		var day string
		var n int64
		if err := rows.Scan(&day, &n); err != nil {
			return nil, err
		}
		out[day] = n
	}
	return out, rows.Err()
}

// parseTags splits the tags JSONB column (["a","b"]) into []string. Corrupt
// or empty -> [] (fail soft, consistent with scanLinks).
func parseTags(raw sql.NullString) []string {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return []string{}
	}
	var parsed []string
	if err := json.Unmarshal([]byte(raw.String), &parsed); err != nil || parsed == nil {
		return []string{}
	}
	return parsed
}

// LinkExportStats returns every link of the creator together with its click
// and unique counts INSIDE the range (LEFT JOIN: links without clicks still
// appear, with 0 - an honest export for new links). limit > 0 trims the
// result (the handler requests limit+1 to learn whether rows were left out).
// Query/scan errors are returned as-is; a creator with no links (or none in
// range) gets an empty slice, not an error.
func (s *SingleStore) LinkExportStats(creatorID int64, from, to time.Time, limit int) ([]LinkExportRow, error) {
	query := `SELECT u.short_code, u.original_url, COALESCE(u.tags,'[]'),
	       COUNT(ce.id) AS clicks, COUNT(ce.id) FILTER (WHERE ce.is_unique) AS uniques
		 FROM urls u
		 LEFT JOIN click_events ce ON ce.short_code = u.short_code
		    AND ce.clicked_at >= $2 AND ce.clicked_at < $3
		 WHERE u.creator_id = $1
		 GROUP BY u.id
		 ORDER BY clicks DESC, u.short_code ASC`
	args := []any{creatorID, from, to}
	if limit > 0 {
		query += " LIMIT $4"
		args = append(args, limit)
	}
	db, _ := s.analyticsRead()
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LinkExportRow{}
	for rows.Next() {
		var r LinkExportRow
		var tags sql.NullString
		if err := rows.Scan(&r.ShortCode, &r.OriginalURL, &tags, &r.Clicks, &r.UniqueClicks); err != nil {
			return nil, err
		}
		r.Tags = parseTags(tags)
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListClicks returns the most recent raw clicks in the range (capped by
// limit) - input for the clicks-mode CSV export. Ordering by descending time
// puts the newest rows first, so truncation leaves the freshest data instead
// of the oldest ever recorded.
// Query/scan/rows errors are returned as-is; no clicks in range means an
// empty slice, not an error.
func (s *SingleStore) ListClicks(creatorID int64, from, to time.Time, limit int) ([]ClickRow, error) {
	query := `SELECT ce.clicked_at, ce.short_code,
	       COALESCE(NULLIF(ce.device_type,''), 'unknown'),
	       COALESCE(NULLIF(ce.referrer_type,''), 'other'),
	       COALESCE(ce.referrer_domain, ''), ce.is_unique
		 FROM click_events ce JOIN urls u ON u.short_code = ce.short_code
		 WHERE u.creator_id = $1 AND ce.clicked_at >= $2 AND ce.clicked_at < $3
		 ORDER BY ce.clicked_at DESC, ce.id DESC`
	args := []any{creatorID, from, to}
	if limit > 0 {
		query += " LIMIT $4"
		args = append(args, limit)
	}
	db, _ := s.analyticsRead()
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ClickRow{}
	for rows.Next() {
		var c ClickRow
		if err := rows.Scan(&c.ClickedAt, &c.ShortCode, &c.DeviceType, &c.ReferrerType, &c.ReferrerDomain, &c.IsUnique); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// FillLast30Days merges per-day counts into a complete window ending today.
// Missing days become 0 so charts never have gaps.
func FillLast30Days(counts map[string]int64, today time.Time) []DayCount {
	out := make([]DayCount, 0, 30)
	for i := 29; i >= 0; i-- {
		d := today.AddDate(0, 0, -i).Format("2006-01-02")
		out = append(out, DayCount{Date: d, Count: counts[d]})
	}
	return out
}

// FillDays is the ranged version of FillLast30Days: it fills EVERY day from
// from to to (both UTC dates, to exclusive), including days without clicks.
// Used by the 7/30/90-day chart range so the X axis has no gaps - the
// zero-fill logic was moved here from SQL (generate_series), following the
// trade-off documented in the dailyCounts comment. Dates are formatted with
// the location of from/to themselves - callers pass time.UTC, so the buckets
// match TO_CHAR(clicked_at) in the database (a TIMESTAMP without time zone
// column always written in UTC by the application).
func FillDays(counts map[string]int64, from, to time.Time) []DayCount {
	out := make([]DayCount, 0, 32)
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		out = append(out, DayCount{Date: key, Count: counts[key]})
	}
	return out
}

// LogClick writes one click as 2 rows in ONE transaction: INSERT into
// click_events (the append-only log, with the is_unique flag +
// referrer_domain) plus UPDATE of the click_count counter (and
// unique_click_count when is_unique). Both must succeed together - if the
// counter rises while the log is missing (or vice versa), analytics and the
// counter would contradict each other. Trade-off: one transaction per event
// means 2 serial writes per click (heavier than pure fire-and-forget);
// batching N events per transaction would be faster but delays data
// visibility. Delivery stays at-most-once: a failure midway loses the event
// with a warning log (no retry, no dead-letter queue). Alternatives rejected:
// two operations without a transaction (skew risk), or deriving the counter
// from click_events aggregates at read time (expensive, against Fase 2).
func (s *SingleStore) LogClick(e ClickEvent) error {
	tx, err := s.primary.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		"INSERT INTO click_events (short_code, referrer, referrer_domain, is_unique, clicked_at, user_agent, device_type, referrer_type) VALUES ($1, NULLIF($2,''), NULLIF($3,''), $4, $5, NULLIF($6,''), NULLIF($7,''), NULLIF($8,''))",
		e.ShortCode, e.Referrer, e.ReferrerDomain, e.IsUnique, clickTime(e.ClickedAt),
		e.UserAgent, e.DeviceType, e.ReferrerType,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		"UPDATE urls SET click_count = click_count + 1, unique_click_count = unique_click_count + $1 WHERE short_code = $2",
		e.IsUniqueBool(), e.ShortCode,
	); err != nil {
		return err
	}
	return tx.Commit()
}

// ReorderLinks writes the new order in ONE transaction: every position UPDATE
// commits together, or all of them roll back if one fails. Without a
// transaction (N separate queries), a request dying midway leaves a
// half-applied order - undetected data corruption. Every row is
// ownership-checked (WHERE creator_id) so a user cannot scramble someone
// else's links. Trade-off: the transaction holds row locks during the update
// (milliseconds for a few dozen rows, harmless at this scale); a huge array
// would hold locks for a long time - hence the handler caps the order length.
// Alternatives: N UPDATEs without a transaction (simple but not atomic), or
// fractional indexing (insert without rewriting everything - fewer writes,
// but complex).
func (s *SingleStore) ReorderLinks(creatorID int64, order []string) error {
	tx, err := s.primary.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for pos, code := range order {
		res, err := tx.Exec("UPDATE urls SET position = $1 WHERE short_code = $2 AND creator_id = $3", pos, code, creatorID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return sql.ErrNoRows
		}
	}
	return tx.Commit()
}
