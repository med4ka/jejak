// Package ratelimit implements a tiny fixed-window login throttle (Fase 7).
// In-memory MVP: no external dependency, works in every APP_MODE.
package ratelimit

import (
	"strings"
	"sync"
	"time"
)

// Limits for POST /api/login: 5 gagal per 1 menit per kombinasi IP+username.
const (
	MaxAttempts = 5
	Window      = time.Minute
)

type bucket struct {
	count int
	start time.Time
}

// LEARN:
//
//	Kenapa key-nya IP+username, bukan cuma IP: kalau kunci cuma IP, 1 penyerang
//	yang brute-force 1 akun dari IP kampus/kantor ikut mengunci SEMUA user lain
//	di IP yang sama (collateral damage). Kunci IP+username membatasi dampak
//	hanya ke pasangan (penyerang, akun target) — user lain di IP sama yang login
//	ke akunnya sendiri tidak terganggu. Username di-lowercase supaya variasi
//	kapitalisasi tidak membuka bucket baru.
//	Trade-off: In-memory per instance — di belakang 2 instance, limit efektif
//	jadi 2× (5+5) karena bucket tidak dishare; restart me-reset semua bucket.
//	Map tumbuh per kombinasi unik (spam cardinality = memori) → ada sweep
//	oportunistik tiap 256 record. Produksi: Redis (INCR+EXPIRE, atomik, shared).
//	Alternatif: Token bucket / sliding-window-log (lebih halus, lebih kode).
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	sweeps  uint64
	max     int
}

// NewLimiter creates an empty in-memory limiter with MaxAttempts (5) per minute
// (default for login throttle, Fase 7).
func NewLimiter() *Limiter {
	return NewLimiterWithMax(MaxAttempts)
}

// NewLimiterWithMax creates a limiter with a custom per-minute cap. Use 100
// for the public API key rate limit (100 req/menit per key, lebih longgar
// dari rate limit login yang 5 gagal/menit).
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

// ClientIP prefers X-Forwarded-For (first hop) behind a proxy/LB, else the
// direct peer address without port.
func ClientIP(remoteAddr, forwardedFor string) string {
	if forwardedFor != "" {
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
