package auth

import (
	"testing"
	"time"
)

func TestHashPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("s3cret-pass!")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "s3cret-pass!" {
		t.Fatal("HashPassword returned plaintext: password must never be stored as-is")
	}
	if !CheckPassword(hash, "s3cret-pass!") {
		t.Fatal("CheckPassword(hash, correct) = false, want true")
	}
	if CheckPassword(hash, "wrong-pass") {
		t.Fatal("CheckPassword(hash, wrong) = true, want false")
	}
}

func TestCheckPasswordMalformedHash(t *testing.T) {
	// Corrupt hash must return false, never panic.
	if CheckPassword("not-a-bcrypt-hash", "anything") {
		t.Fatal("CheckPassword on corrupt hash = true, want false")
	}
}

func TestCheckPasswordEmpty(t *testing.T) {
	hash, err := HashPassword("some-password")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if CheckPassword(hash, "") {
		t.Fatal("CheckPassword(hash, empty) = true, want false")
	}
}

// TestMemoryStoreSessionLifecycle covers create -> get -> delete in order.
func TestMemoryStoreSessionLifecycle(t *testing.T) {
	s := NewMemoryStore()

	token, err := s.Create(42)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if token == "" {
		t.Fatal("Create returned empty token")
	}
	if len(token) != 64 {
		t.Fatalf("token length = %d, want 64 (32-byte hex)", len(token))
	}

	// Get returns the same creator bound at create time.
	id, ok := s.Get(token)
	if !ok {
		t.Fatal("Get(token) = not found, want found")
	}
	if id != 42 {
		t.Fatalf("Get(token) = %d, want 42", id)
	}

	// Delete revokes the session; repeat delete is a no-op (not an error).
	s.Delete(token)
	if _, ok := s.Get(token); ok {
		t.Fatal("Get after Delete = found, want not found")
	}
	s.Delete(token)
}

func TestMemoryStoreGetUnknownToken(t *testing.T) {
	s := NewMemoryStore()
	if _, ok := s.Get("ghost-token"); ok {
		t.Fatal("Get(unknown) = found, want not found")
	}
}

func TestMemoryStoreExpiredSession(t *testing.T) {
	// Inject a session already past its expiry directly (bypasses Create so
	// we don't have to wait SessionTTL). Get must treat it as logged out and
	// lazily purge the entry.
	s := NewMemoryStore()
	s.mu.Lock()
	s.sessions["expired"] = Session{CreatorID: 7, ExpiresAt: time.Now().Add(-time.Minute)}
	s.mu.Unlock()

	if _, ok := s.Get("expired"); ok {
		t.Fatal("Get(expired) = found, want not found")
	}
	// Lazy purge: entry must be gone from the map now.
	s.mu.Lock()
	_, stillThere := s.sessions["expired"]
	s.mu.Unlock()
	if stillThere {
		t.Fatal("expired session not purged after Get")
	}
}

func TestMemoryStoreSlidingExpiryExtends(t *testing.T) {
	s := NewMemoryStore()
	token, _ := s.Create(1)

	// Force the stored expiry to ~30s from now (still valid, but far below
	// the 7-day SessionTTL). A Get inside the window must succeed AND slide
	// the expiry back out to a fresh 7 days: that is the sliding behavior.
	s.mu.Lock()
	old := s.sessions[token]
	s.sessions[token] = Session{CreatorID: old.CreatorID, ExpiresAt: time.Now().Add(30 * time.Second)}
	s.mu.Unlock()

	if _, ok := s.Get(token); !ok {
		t.Fatal("Get near-expiry session = not found, want found (still within window)")
	}
	s.mu.Lock()
	slid := s.sessions[token].ExpiresAt
	s.mu.Unlock()
	if !slid.After(time.Now().Add(24 * time.Hour)) {
		t.Fatalf("expiry not slid to full TTL: ExpiresAt=%v, want ~7 days out", slid)
	}
}

func TestMemoryStoreUniqueTokens(t *testing.T) {
	s := NewMemoryStore()
	a, _ := s.Create(1)
	b, _ := s.Create(1)
	if a == b {
		t.Fatal("two Create calls returned the same token")
	}
}

func TestValidUsername(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"alice", true},
		{"alice_99", true},
		{"user123", true},
		{"abc", true},        // min 3
		{"a", false},         // too short
		{"ab", false},        // too short
		{"", false},          // empty
		{"alice bob", false}, // space
		{"alice-bob", false}, // hyphen not allowed
		{"user.name", false}, // dot not allowed
	}
	for _, tt := range tests {
		if got := ValidUsername(tt.name); got != tt.want {
			t.Errorf("ValidUsername(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
func TestMemoryStoreDeleteAllForUser(t *testing.T) {
	s := NewMemoryStore()
	a, _ := s.Create(1)
	b, _ := s.Create(1)
	c, _ := s.Create(2)

	removed, err := s.DeleteAllForUser(1, "")
	if err != nil {
		t.Fatalf("DeleteAllForUser returned error: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	for _, tok := range []string{a, b} {
		if _, ok := s.Get(tok); ok {
			t.Fatal("sesi creator 1 masih hidup setelah DeleteAllForUser")
		}
	}
	if _, ok := s.Get(c); !ok {
		t.Fatal("sesi creator 2 ikut ter-revoke: harusnya hanya creator 1")
	}
}

func TestMemoryStoreDeleteAllForUserKeepToken(t *testing.T) {
	s := NewMemoryStore()
	keep, _ := s.Create(1)
	other, _ := s.Create(1)

	removed, err := s.DeleteAllForUser(1, keep)
	if err != nil {
		t.Fatalf("DeleteAllForUser returned error: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1 (hanya sesi non-keep)", removed)
	}
	if _, ok := s.Get(keep); !ok {
		t.Fatal("sesi peminta (keepToken) ikut ter-revoke")
	}
	if _, ok := s.Get(other); ok {
		t.Fatal("sesi device lain masih hidup")
	}
}
