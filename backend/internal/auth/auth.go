// Package auth implements simple session-based auth for Fase 9.
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
// and queue — prefix avoids collision with short-code cache entries).
const sessionKeyPrefix = "session:"

// SessionTTL bounds how long a login lasts without re-login. Same value
// drives both Redis key expiry and cookie MaxAge so they lapse together.
const SessionTTL = 7 * 24 * time.Hour

var validUsername = regexp.MustCompile(`^[a-zA-Z0-9_]{3,30}$`)

// ValidUsername enforces SCHEMA.md: alfanumerik + underscore, maks 30 char
// (min 3 char supaya tidak bentrok dengan short-code 6 char yang acak).
func ValidUsername(u string) bool {
	return validUsername.MatchString(u)
}

// HashPassword hashes with bcrypt (salt built-in, cost default).
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
	Create(creatorID int64) (string, error)
	Get(token string) (int64, bool)
	Delete(token string)
}

// LEARN:
//   Kenapa: Session store Redis-backed (bukan in-memory): SEMUA instance API
//   di belakang load balancer membaca/menulis sesi yang SAMA, jadi login di
//   instance A tetap dikenal instance B (round-robin tidak lagi me-logout
//   user), dan sesi selamat dari restart server. Revoke tetap instan (DEL key
//   langsung) — keunggulan session-based atas JWT yang dipertahankan. TTL
//   native Redis (EX 7 hari) = expiry otomatis tanpa sweeper + cookie MaxAge
//   disamakan supaya keduanya lapse bersama.
//   Trade-off: Redis jadi dependensi keras untuk auth — Redis mati = login,
//   register (auto-login), logout, DAN setiap baca sesi ikut gagal. Tiap
//   request ber-auth tambah 1 RTT network ke Redis. Data sesi melintasi
//   network (di DC sendiri, acceptable; jangan expose Redis ke publik).
//   Alternatif: JWT stateless (tanpa store, scaling trivial) tapi revoke
//   tidak instan — butuh blocklist yang ujung-ujungnya state lagi.
type RedisStore struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisStore creates a session store on top of an existing Redis client
// (shared with cache + queue — 1 pool koneksi, key dipisah via prefix).
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

// LEARN:
//   Kenapa sliding-expiry, bukan fixed-expiry: fixed 7 hari dari login berarti
//   user yang aktif tiap hari tetap ditendang tepat hari ke-7 (pengalaman
//   buruk yang tidak ada hubungannya dengan keamanan). Sliding menghitung
//   ulang 7 hari dari request authed TERAKHIR — selama user aktif, sesi hidup
//   terus; hanya user yang benar-benar hilang 7 hari penuh yang login ulang.
//   Itu sebabnya produk consumer (FB/IG/dst) pakai sliding: mengurangi friksi
//   login ulang tanpa memperpanjang jendela sesi yang sudah ditinggalkan
//   (token curian yang tidak dipakai tetap mati sesuai TTL terakhir).
//   Trade-off: Tiap Get sukses = 1 write EXPIRE ekstra (GET+EXPIRE, bukan GET
//   saja). Jendela eksposur token curian ikut memanjang selama dipakai penyerang
//   — mitigasinya revoke instan (Delete), bukan expiry.
//   Alternatif: Refresh periodik (mis. hanya kalau sisa TTL < 1 hari) untuk
//   hemat write — optimasi yang valid kalau traffic auth tinggi.
// Get returns the creator bound to token, or false when missing/expired
// (Redis auto-deletes expired keys; redis.Nil maps to "no session").
// On success the TTL slides: EXPIRE resets to SessionTTL from NOW (no re-create,
// value untouched). Best-effort: EXPIRE failure doesn't fail auth.
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

// Session is one in-memory login: which creator, until when.
type Session struct {
	CreatorID int64
	ExpiresAt time.Time
}

// MemoryStore is the in-memory fallback for baseline mode (no Redis, single
// instance only). Same methods as RedisStore so handler code is identical.
// JANGAN dipakai di belakang load balancer: tiap instance punya map sendiri.
type MemoryStore struct {
	mu       sync.Mutex
	sessions map[string]Session
}

// NewMemoryStore creates an empty in-memory session store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{sessions: make(map[string]Session)}
}

// Create mints a random token bound to creatorID with SessionTTL expiry.
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
	// Sliding juga di sini supaya perilaku baseline identik dengan Redis.
	s.sessions[token] = Session{CreatorID: sess.CreatorID, ExpiresAt: time.Now().Add(SessionTTL)}
	return sess.CreatorID, true
}

// Delete revokes a session (logout). Missing token is not an error.
func (s *MemoryStore) Delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// SetCookie writes the session cookie: HttpOnly, Path=/, SameSite=Lax.
// Secure flag sengaja tidak di-set karena dev lokal jalan di http.
func SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((SessionTTL).Seconds()),
	})
}

// ClearCookie removes the session cookie client-side (server also deletes).
func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
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
