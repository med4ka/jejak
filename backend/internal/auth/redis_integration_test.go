//go:build integration

package auth

import (
	"context"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"

	"jejak/internal/env"
)

// openRedis connects to the local Redis (REDIS_URL, or localhost default —
// the same instance the server uses in full mode). Gated behind the
// `integration` build tag:
//
//	go test -tags integration ./internal/auth/
func openRedis(t *testing.T) *redis.Client {
	t.Helper()
	// Same .env semantics as the server (REDIS_URL from the repo root).
	env.LoadDotEnv()
	raw := os.Getenv("REDIS_URL")
	if raw == "" {
		raw = "redis://localhost:6379/0"
	}
	opt, err := redis.ParseURL(raw)
	if err != nil {
		t.Fatalf("REDIS_URL: %v", err)
	}
	c := redis.NewClient(opt)
	if err := c.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("redis ping: %v (is Redis running on :6379?)", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

// TestIntegrationRedisSessionLifecycle: real Redis sessions — mint, read,
// revoke, and the negative cases (unknown/tampered token) that keep a
// stolen or guessed value from ever resolving to an account.
func TestIntegrationRedisSessionLifecycle(t *testing.T) {
	c := openRedis(t)
	st := NewRedisStore(c)

	token, err := st.Create(77)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if token == "" {
		t.Fatal("empty token")
	}
	id, ok := st.Get(token)
	if !ok || id != 77 {
		t.Fatalf("Get = %d,%v want 77,true", id, ok)
	}

	// A tampered token must miss: one flipped character invalidates the
	// whole key (tokens are exact-match keys, never parsed).
	tampered := token[:len(token)-1] + flipHex(token[len(token)-1])
	if tampered == token {
		tampered = token[:len(token)-1] + "0"
	}
	if _, ok := st.Get(tampered); ok {
		t.Fatal("tampered token resolved to a session")
	}
	if _, ok := st.Get("not-a-token"); ok {
		t.Fatal("garbage token resolved to a session")
	}

	st.Delete(token)
	if _, ok := st.Get(token); ok {
		t.Fatal("deleted token still valid")
	}
}

// TestIntegrationRedisDeleteAllForUser: logout-all revokes every session of
// the creator EXCEPT the current one (the keep token stays usable), and
// other creators' sessions are untouched.
func TestIntegrationRedisDeleteAllForUser(t *testing.T) {
	c := openRedis(t)
	st := NewRedisStore(c)

	keep, _ := st.Create(88)
	otherDev, _ := st.Create(88)
	bystander, _ := st.Create(99)

	removed, err := st.DeleteAllForUser(88, keep)
	if err != nil {
		t.Fatalf("DeleteAllForUser: %v", err)
	}
	if removed < 1 {
		t.Fatalf("removed = %d, want >= 1 (the other device)", removed)
	}
	if _, ok := st.Get(keep); !ok {
		t.Fatal("keep token was revoked")
	}
	if _, ok := st.Get(otherDev); ok {
		t.Fatal("other device survived logout-all")
	}
	if _, ok := st.Get(bystander); !ok {
		t.Fatal("bystander session was collateral damage")
	}

	// Cleanup so the fixed keys do not accumulate across runs.
	st.Delete(keep)
	st.Delete(bystander)
}

// flipHex returns a different hex character (token tampering helper).
func flipHex(b byte) string {
	if b == '0' {
		return "1"
	}
	return "0"
}
