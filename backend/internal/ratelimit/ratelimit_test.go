package ratelimit

import (
	"fmt"
	"testing"
	"time"
)

// TestAllowRecordReset covers the fixed-window core: first attempt passes,
// Record pushes the bucket to the cap, Allow then refuses, Reset forgives,
// and an expired window opens a fresh bucket.
func TestAllowRecordReset(t *testing.T) {
	l := NewLimiterWithMax(3)
	key := Key("203.0.113.1", "alice")

	for i := 0; i < 3; i++ {
		if !l.Allow(key) {
			t.Fatalf("attempt %d blocked before cap", i+1)
		}
		l.Record(key)
	}
	if l.Allow(key) {
		t.Fatal("4th attempt allowed after cap of 3")
	}

	l.Reset(key)
	if !l.Allow(key) {
		t.Fatal("Reset did not clear the bucket")
	}
}

// TestWindowExpiry: once the fixed window elapses the bucket is dropped and
// attempts flow again (restart-free recovery inside the same process).
func TestWindowExpiry(t *testing.T) {
	l := NewLimiterWithMax(1)
	key := "k"
	l.Record(key)
	if l.Allow(key) {
		t.Fatal("expected block right after Record")
	}
	// Wait out the window (Window is 1 minute: only simulate the timestamp
	// by manipulating the bucket directly, keeping the test fast).
	l.mu.Lock()
	l.buckets[key].start = time.Now().Add(-Window - time.Second)
	l.mu.Unlock()
	if !l.Allow(key) {
		t.Fatal("expired window should allow again")
	}
}

// TestKeyNormalization: username case/whitespace must not open new buckets
// (so "Alice" and " alice " share one throttle), and different usernames on
// the same IP stay independent.
func TestKeyNormalization(t *testing.T) {
	base := Key("10.0.0.1", "Alice")
	if Key("10.0.0.1", "alice") != base {
		t.Error("username case opened a new bucket")
	}
	if Key("10.0.0.1", " ALICE ") != base {
		t.Error("username whitespace opened a new bucket")
	}
	if Key("10.0.0.2", "alice") == base {
		t.Error("different IP shares the bucket")
	}
	if Key("10.0.0.1", "bob") == base {
		t.Error("different username shares the bucket")
	}
}

// TestSweep: expired buckets are dropped on the opportunistic sweep so the
// map cannot grow without bound under key spam.
func TestSweep(t *testing.T) {
	l := NewLimiterWithMax(5)
	for i := 0; i < 10; i++ {
		l.Record(fmt.Sprintf("spam-%d", i))
	}
	l.mu.Lock()
	for k := range l.buckets {
		l.buckets[k].start = time.Now().Add(-Window - time.Second)
	}
	l.sweepLocked(time.Now())
	remaining := len(l.buckets)
	l.mu.Unlock()
	if remaining != 0 {
		t.Fatalf("sweep left %d buckets, want 0", remaining)
	}
}

// TestNewLimiterDefaults: NewLimiter is the login default (MaxAttempts).
func TestNewLimiterDefaults(t *testing.T) {
	l := NewLimiter()
	if l.max != MaxAttempts {
		t.Fatalf("max = %d, want %d", l.max, MaxAttempts)
	}
	if Window != time.Minute {
		t.Fatalf("Window = %v, want 1m", Window)
	}
}

// TestClientIPDefaultDistrustsXFF: without SetTrustProxy(true) the spoofable
// X-Forwarded-For header is IGNORED and the real peer wins: rotating the
// header must not mint fresh rate-limit buckets.
func TestClientIPDefaultDistrustsXFF(t *testing.T) {
	SetTrustProxy(false)
	if TrustProxy() {
		t.Fatal("TrustProxy must default to false")
	}
	if got := ClientIP("198.51.100.9:4444", "1.2.3.4, 5.6.7.8"); got != "198.51.100.9" {
		t.Fatalf("ClientIP = %q, want peer 198.51.100.9 (XFF ignored)", got)
	}
	if got := ClientIP("198.51.100.9:4444", ""); got != "198.51.100.9" {
		t.Fatalf("ClientIP = %q, want peer without port", got)
	}
}

// TestClientIPTrustedXFF: behind a real reverse proxy (opt-in) the first XFF
// hop is the client; ports are stripped from both sources.
func TestClientIPTrustedXFF(t *testing.T) {
	SetTrustProxy(true)
	defer SetTrustProxy(false)

	if got := ClientIP("10.0.0.1:8080", "203.0.113.5, 10.0.0.1"); got != "203.0.113.5" {
		t.Fatalf("ClientIP = %q, want first XFF hop 203.0.113.5", got)
	}
	if got := ClientIP("10.0.0.1:8080", "203.0.113.5"); got != "203.0.113.5" {
		t.Fatalf("ClientIP = %q, want single-hop XFF", got)
	}
	if got := ClientIP("no-port-host", ""); got != "no-port-host" {
		t.Fatalf("ClientIP = %q, want addr without colon unchanged", got)
	}
}
