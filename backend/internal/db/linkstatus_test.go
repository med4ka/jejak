package db

import (
	"testing"
	"time"
)

// TestLinkStatus pins the agreed status mapping (2026-09-30):
// NULL -> "active", future -> "scheduled", past -> "expired". The comparison
// is absolute (time.After), so the origin zone of expiresAt (UTC from the
// database, WIB from a converted client value) must not change the outcome.
func TestLinkStatus(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	future := now.Add(48 * time.Hour)
	past := now.Add(-time.Minute)

	if got := LinkStatus(nil, now); got != "active" {
		t.Errorf("LinkStatus(nil) = %q, want active", got)
	}
	if got := LinkStatus(&future, now); got != "scheduled" {
		t.Errorf("LinkStatus(future) = %q, want scheduled", got)
	}
	if got := LinkStatus(&past, now); got != "expired" {
		t.Errorf("LinkStatus(past) = %q, want expired", got)
	}

	// Exact boundary: now.After(expires) is false when both instants are
	// equal, so the link is still "scheduled" (the same second counts as not
	// yet past - expiry requires now to be strictly after expires).
	same := now
	if got := LinkStatus(&same, now); got != "scheduled" {
		t.Errorf("LinkStatus(same instant) = %q, want scheduled", got)
	}

	// An expiresAt in a different zone (WIB) is still compared absolutely.
	wib := time.FixedZone("WIB", 7*3600)
	futureLocal := now.Add(48 * time.Hour).In(wib)
	if got := LinkStatus(&futureLocal, now); got != "scheduled" {
		t.Errorf("LinkStatus(future WIB) = %q, want scheduled", got)
	}
}
