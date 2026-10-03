// Package ratelimit implements a tiny fixed-window login throttle (Phase 7).
// In-memory MVP: no external dependency, works in every APP_MODE.
package ratelimit

import (
	"strings"
	"sync"
	"time"
)

// MaxAttempts is the login throttle budget: this many failed attempts are
// allowed per Window, per IP+username combination (POST /api/login).
const MaxAttempts = 5

// Window is the fixed-window length of the login throttle; together with
// MaxAttempts it defines the 5 failures/min limit.
const Window = time.Minute

type bucket struct {
	count int
	start time.Time
}

// Limiter is a fixed-window in-memory throttle, one bucket per key.
// Rationale: the key is IP+username, not IP alone: keying on IP only would
// let one attacker brute-forcing a single account from a shared campus or
// office IP lock out EVERY other user on that IP (collateral damage). Keying
// on IP+username limits the damage to the (attacker, target account) pair, so
// other users on the same IP logging into their own accounts are unaffected.
// The username is lowercased so capitalization variants do not open new
// buckets. Trade-off: in-memory per instance: behind 2 instances the
// effective limit becomes 2× (5+5) because buckets are not shared, and a
// restart resets all buckets. The map grows per unique combination (spam
// cardinality = memory) → there is an opportunistic sweep every 256 records.
// Production: Redis (INCR+EXPIRE, atomic, shared). Alternative: token bucket
// or sliding-window-log (smoother, more code).
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	sweeps  uint64
	max     int
}

// NewLimiter creates an empty in-memory limiter with MaxAttempts (5) per minute
// (default for login throttle, Phase 7).
func NewLimiter() *Limiter {
	return NewLimiterWithMax(MaxAttempts)
}

// NewLimiterWithMax creates a limiter with a custom per-minute cap. Use 100
// for the public API key rate limit (100 req/min per key, looser than the
// login throttle of 5 failures/min).
func NewLimiterWithMax(max int) *Limiter {
	return &Limiter{buckets: make(map[string]*bucket), max: max}
}

// Key builds the throttle bucket: client IP + attempted username.
func Key(ip, username string) string {
	return strings.TrimSpace(ip) + "\x00" + strings.ToLower(strings.TrimSpace(username))
}

// Allow reports whether another attempt may proceed (does not record).
func (l *Limiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[key]
	if !ok {
		return true
	}
	if now.Sub(b.start) >= Window {
		delete(l.buckets, key)
		return true
	}
	return b.count < l.max
}

// Record registers one FAILED attempt. Successes must call Reset instead.
func (l *Limiter) Record(key string) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[key]
	if !ok || now.Sub(b.start) >= Window {
		l.buckets[key] = &bucket{count: 1, start: now}
	} else {
		b.count++
	}
	l.sweeps++
	if l.sweeps%256 == 0 {
		l.sweepLocked(now)
	}
}

// Reset clears the bucket (successful login forgives past failures so the
// legitimate owner is never punished for their own typos).
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	delete(l.buckets, key)
	l.mu.Unlock()
}

// sweepLocked drops expired buckets. Caller must hold l.mu.
func (l *Limiter) sweepLocked(now time.Time) {
	for k, b := range l.buckets {
		if now.Sub(b.start) >= Window {
			delete(l.buckets, k)
		}
	}
}

// ClientIP returns the peer address without port. X-Forwarded-For is
// client-controlled: any caller can spoof it, which would defeat rate limits
// keyed on IP (an attacker rotates the header to get a fresh bucket). The
// first XFF hop is therefore ignored by default; SetTrustProxy(true) enables
// it for deployments behind a real reverse proxy (nginx/Traefik/Cloudflare)
// that overwrites the header on ingress. Default (false) is the safe choice:
// direct exposure cannot be spoofed, and a misconfigured trust lets an
// attacker mint unlimited buckets.
func ClientIP(remoteAddr, forwardedFor string) string {
	if trustProxy && forwardedFor != "" {
		if i := strings.Index(forwardedFor, ","); i >= 0 {
			return strings.TrimSpace(forwardedFor[:i])
		}
		return strings.TrimSpace(forwardedFor)
	}
	if i := strings.LastIndex(remoteAddr, ":"); i >= 0 {
		return remoteAddr[:i]
	}
	return remoteAddr
}

// SetTrustProxy toggles whether ClientIP honors X-Forwarded-For. Wire it once
// at startup from TRUST_PROXY (default false); it is not concurrency-safe to
// flip per request.
func SetTrustProxy(trust bool) {
	trustProxy = trust
}

// TrustProxy reports whether X-Forwarded-For is currently honored (exported
// for tests and diagnostics).
func TrustProxy() bool {
	return trustProxy
}

// trustProxy is the package-level switch behind ClientIP; false until the
// server explicitly opts in via SetTrustProxy.
var trustProxy bool
