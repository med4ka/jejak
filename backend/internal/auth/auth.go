// Package auth implements simple session-based auth for Phase 9.
// Cookie carries an opaque session token; server holds token -> creator mapping.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// CookieName is the session cookie. HttpOnly so JS can't read the token.
const CookieName = "jejak_session"

// sessionKeyPrefix namespaces session keys in Redis (shared DB 0 with cache
// and queue: prefix avoids collision with short-code cache entries).
const sessionKeyPrefix = "session:"

// SessionTTL bounds how long a login lasts without re-login. Same value
// drives both Redis key expiry and cookie MaxAge so they lapse together.
const SessionTTL = 7 * 24 * time.Hour

var validUsername = regexp.MustCompile(`^[a-zA-Z0-9_]{3,30}$`)

// ValidUsername enforces SCHEMA.md: alphanumeric + underscore, at most 30
// chars (minimum 3 so it cannot collide with the random 6-char short code).
func ValidUsername(u string) bool {
	return validUsername.MatchString(u)
}

// HashPassword hashes with bcrypt (salt built-in, cost default). Returns the
// bcrypt error when the password exceeds bcrypt's 72-byte limit or hashing
// fails; on error the returned hash is "".
func HashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// CheckPassword compares plaintext against a stored bcrypt hash.
// Returns false (not the bcrypt error) so callers map it to 401 uniformly.
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// Store maps session tokens to creator ids. Two backends: Redis (shared,
// production path) and in-memory (baseline without Redis). Handler depends
// only on this interface so backends are swappable without touching routes.
type Store interface {
	// Create mints a new session token bound to creatorID (SessionTTL
	// expiry). Returns an error when the token cannot be generated or
	// stored; then no session exists and login/registration must fail.
	Create(creatorID int64) (string, error)
	// Get resolves a token to its creator. Missing, expired, or unreadable
	// sessions all return false - never an error path.
	Get(token string) (int64, bool)
	// Delete revokes one session (logout); a missing token is not an error.
	Delete(token string)
	// DeleteAllForUser revokes ALL sessions belonging to creatorID (logout-all,
	// account deletion): keepToken is excluded (""): password change uses this
	// to kick OTHER devices without logging out the device in use (its cookie
	// token is passed as keepToken). Returns the number of sessions removed.
	// No error: a session that does not exist is not a failure.
	DeleteAllForUser(creatorID int64, keepToken string) (int, error)
}

// RedisStore is a session store backed by Redis, shared by all API instances.
// Rationale: Redis-backed rather than in-memory: EVERY API instance behind
// the load balancer reads and writes the SAME sessions, so a login on
// instance A is still recognized by instance B (round-robin no longer logs
// users out) and sessions survive server restarts. Revocation stays instant
// (DEL on the key): the advantage over JWT that is deliberately kept. Native
// Redis TTL (EX 7 days) gives automatic expiry with no sweeper, and the cookie
// MaxAge is matched so both lapse together.
// Trade-off: Redis becomes a hard dependency for auth: if Redis is down,
// login, register (auto-login), logout, AND every session read fail as well.
// Each authenticated request adds 1 network RTT to Redis, and session data
// crosses the network (acceptable within one's own DC; do not expose Redis to
// the public). Alternative: stateless JWT (no store, trivial scaling) but
// revocation is not instant: it requires a blocklist, which is state again.
type RedisStore struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisStore creates a session store on top of an existing Redis client
// (shared with cache + queue: one connection pool, keys separated by prefix).
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client, ctx: context.Background()}
}

// sessionKey builds the Redis key for a token. Tokens are 64 hex chars from
// crypto/rand, so no escaping needed (no spaces/colons inside).
func sessionKey(token string) string {
	return sessionKeyPrefix + token
}

// mintToken generates a 256-bit random hex token (unpredictable session id).
func mintToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Create mints a random token bound to creatorID with SessionTTL expiry.
// Returns the rand error when the OS entropy source fails, or the Redis
// error when SET fails; in both cases no session exists and the caller's
// login/registration request must fail.
func (s *RedisStore) Create(creatorID int64) (string, error) {
	token, err := mintToken()
	if err != nil {
		return "", err
	}
	if err := s.client.Set(s.ctx, sessionKey(token), creatorID, SessionTTL).Err(); err != nil {
		return "", err
	}
	return token, nil
}

// Get returns the creator bound to token, or false when missing/expired
// (Redis auto-deletes expired keys; redis.Nil maps to "no session").
// On success the TTL slides: EXPIRE resets to SessionTTL from NOW (no re-create,
// value untouched). Best-effort: EXPIRE failure doesn't fail auth.
// Rationale for sliding rather than fixed expiry: a fixed 7 days from login
// would still kick out a user who is active every day, exactly on day 7 (a
// poor experience with no security benefit). Sliding recomputes 7 days from
// the LAST authenticated request: while the user stays active the session
// lives on, and only a user truly gone for a full 7 days must log in again.
// That is why consumer products (FB/IG etc.) use sliding: it reduces re-login
// friction without extending the window of an abandoned session (a stolen
// token that is never used still dies on its last TTL).
// Trade-off: every successful Get costs 1 extra EXPIRE write (GET+EXPIRE,
// not GET alone), and the exposure window of a stolen token stretches while
// an attacker uses it: mitigation is instant revocation (Delete), not
// expiry. Alternative: periodic refresh (e.g. only when remaining TTL < 1 day)
// to save writes: a valid optimization if authenticated traffic is high.
func (s *RedisStore) Get(token string) (int64, bool) {
	if token == "" {
		return 0, false
	}
	val, err := s.client.Get(s.ctx, sessionKey(token)).Result()
	if err != nil {
		return 0, false
	}
	id, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, false
	}
	s.client.Expire(s.ctx, sessionKey(token), SessionTTL)
	return id, true
}

// Delete revokes a session (logout). Missing token is not an error.
func (s *RedisStore) Delete(token string) {
	if token == "" {
		return
	}
	s.client.Del(s.ctx, sessionKey(token))
}

// DeleteAllForUser revokes all of creatorID's sessions except keepToken (""
// = no exception). Redis has no "sessions per user" index (the value stores
// creatorID as a string and the key is the token), so scanning session:* and
// GETting each key is the only route without a new schema. SCAN (not KEYS) so
// Redis is never blocked on a large keyspace; one pass in batches of 128 keys
// because sessions per user are few (1 per logged-in device, 7-day TTL), so a
// long scan is not a risk.
// Returns the SCAN error when the scan fails (nothing deleted yet), or the
// DEL error when the bulk delete fails (sessions found before that point may
// already be gone); a key that expires between SCAN and GET is skipped, not
// an error. 0 with a nil error means there was nothing to revoke.
func (s *RedisStore) DeleteAllForUser(creatorID int64, keepToken string) (int, error) {
	want := strconv.FormatInt(creatorID, 10)
	keep := ""
	if keepToken != "" {
		keep = sessionKey(keepToken)
	}
	var toDelete []string
	var cursor uint64
	for {
		keys, next, err := s.client.Scan(s.ctx, cursor, sessionKeyPrefix+"*", 128).Result()
		if err != nil {
			return 0, err
		}
		for _, k := range keys {
			if k == keep {
				continue
			}
			val, err := s.client.Get(s.ctx, k).Result()
			if err != nil {
				continue // key expired between SCAN and GET: not a failure
			}
			if val == want {
				toDelete = append(toDelete, k)
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	if len(toDelete) == 0 {
		return 0, nil
	}
	if err := s.client.Del(s.ctx, toDelete...).Err(); err != nil {
		return 0, err
	}
	return len(toDelete), nil
}

// Session is one in-memory login: which creator, until when.
type Session struct {
	CreatorID int64
	ExpiresAt time.Time
}

// MemoryStore is the in-memory fallback for baseline mode (no Redis, single
// instance only). Same methods as RedisStore so handler code is identical.
// Do NOT use it behind a load balancer: every instance has its own map.
type MemoryStore struct {
	mu       sync.Mutex
	sessions map[string]Session
}

// NewMemoryStore creates an empty in-memory session store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{sessions: make(map[string]Session)}
}

// Create mints a random token bound to creatorID with SessionTTL expiry.
// Returns the rand error when the OS entropy source fails; the in-memory
// map itself never fails.
func (s *MemoryStore) Create(creatorID int64) (string, error) {
	token, err := mintToken()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.sessions[token] = Session{CreatorID: creatorID, ExpiresAt: time.Now().Add(SessionTTL)}
	s.mu.Unlock()
	return token, nil
}

// Get returns the creator bound to token, or false. Expired sessions are
// deleted lazily on lookup (no background sweeper in MVP).
func (s *MemoryStore) Get(token string) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[token]
	if !ok {
		return 0, false
	}
	if time.Now().After(sess.ExpiresAt) {
		delete(s.sessions, token)
		return 0, false
	}
	// Sliding here as well, so baseline behavior is identical to Redis.
	s.sessions[token] = Session{CreatorID: sess.CreatorID, ExpiresAt: time.Now().Add(SessionTTL)}
	return sess.CreatorID, true
}

// Delete revokes a session (logout). Missing token is not an error.
func (s *MemoryStore) Delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// DeleteAllForUser revokes all of creatorID's sessions except keepToken:
// behavior identical to RedisStore (see the rationale there); only the scan
// mechanism differs (local map iteration). It never returns an error (the
// map cannot fail); the count is the number of sessions actually removed.
func (s *MemoryStore) DeleteAllForUser(creatorID int64, keepToken string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := 0
	for tok, sess := range s.sessions {
		if sess.CreatorID != creatorID || tok == keepToken {
			continue
		}
		delete(s.sessions, tok)
		removed++
	}
	return removed, nil
}

// SecureCookies controls the Secure attribute on session cookies (G124):
// Secure keeps the cookie off plain-HTTP responses, so it cannot leak through
// a sniffed request. It stays false by default because local development runs
// on plain http (a Secure cookie would never be sent back and login would
// appear broken); production must set COOKIE_SECURE=true behind TLS. Wired
// once at startup from main, not per request.
var SecureCookies bool

// SetCookie writes the session cookie: HttpOnly, Path=/, SameSite=Lax, and
// Secure when SecureCookies is enabled (COOKIE_SECURE=true in production).
func SetCookie(w http.ResponseWriter, token string) {
	// #nosec G124 -- the Secure attribute is set from SecureCookies
	// (COOKIE_SECURE env, see the var's doc): it cannot be a literal true
	// because plain-http localhost development would then never receive the
	// cookie back; production must set COOKIE_SECURE=true behind TLS.
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		// Secure comes from SecureCookies (COOKIE_SECURE env): it cannot be
		// a literal true because plain-http localhost development would then
		// never receive the cookie back; production must set
		// COOKIE_SECURE=true behind TLS.
		Secure:   SecureCookies, // #nosec G124 -- gated by COOKIE_SECURE (see above)
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((SessionTTL).Seconds()),
	})
}

// ClearCookie removes the session cookie client-side (server also deletes);
// it mirrors SetCookie's attributes so the browser actually matches and
// drops the original cookie.
func ClearCookie(w http.ResponseWriter) {
	// #nosec G124 -- mirrors SetCookie: Secure comes from SecureCookies
	// (COOKIE_SECURE env) so the clearing cookie matches the issued one.
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		// Mirrors SetCookie's Secure so the clearing cookie matches the one
		// that was issued (COOKIE_SECURE env).
		Secure:   SecureCookies, // #nosec G124 -- gated by COOKIE_SECURE (see SetCookie)
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// TokenFromRequest extracts the session token, or "" when absent.
func TokenFromRequest(r *http.Request) string {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return c.Value
}
