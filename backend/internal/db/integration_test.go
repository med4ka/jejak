//go:build integration

package db

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"jejak/internal/env"
)

// openTestStore connects to the real PostgreSQL instance (DATABASE_URL, or
// the localhost default the server uses). The integration suite is
// gated behind the `integration` build tag:
//
//	go test -tags integration ./internal/db/
//
// The schema is created by the server's auto-migrate (migrate.Run), so the
// database must already exist — this suite never applies migrations itself
// (playbook rule: schema changes are human-reviewed).
func openTestStore(t *testing.T) *SingleStore {
	t.Helper()
	// Same .env semantics as the server: local config wins over defaults
	// (the dev database uses user=postgres without a password).
	env.LoadDotEnv()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://jejak:password@localhost:5432/jejak?sslmode=disable"
	}
	s, err := NewSingleStore(dsn, "")
	if err != nil {
		t.Fatalf("NewSingleStore: %v (is PostgreSQL running on :5432?)", err)
	}
	return s
}

// uniqCode returns a short_code unique per test run so re-runs against a
// live database never collide with rows from previous runs (the UNIQUE
// constraint would otherwise turn a rerun into a false failure).
func uniqCode(prefix string) string {
	return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano()%1e12)
}

func uniqUser(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano()%1e12)
}

// TestIntegrationCRUDRoundTrip drives the full lifecycle against real
// Postgres: create a creator, store a link owned by them, read it back
// through every read path, mutate it, then delete it (and prove the delete
// actually removed the row).
func TestIntegrationCRUDRoundTrip(t *testing.T) {
	s := openTestStore(t)

	id, err := s.CreateCreator(uniqUser("itgcrud"), "Integration", "", "x-hash")
	if err != nil {
		t.Fatalf("CreateCreator: %v", err)
	}
	creator := &id

	code := uniqCode("itg")
	exp := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)
	if err := s.CreateURL(code, "https://example.com/crud", creator, `["alpha","beta"]`, &exp); err != nil {
		t.Fatalf("CreateURL: %v", err)
	}

	// Read path 1: raw destination (redirect path).
	got, err := s.GetURL(code)
	if err != nil {
		t.Fatalf("GetURL: %v", err)
	}
	if got != "https://example.com/crud" {
		t.Fatalf("GetURL = %q, want the stored URL", got)
	}

	// Read path 2: dashboard list, scoped to the owner.
	links, err := s.ListLinksByCreator(id)
	if err != nil {
		t.Fatalf("ListLinksByCreator: %v", err)
	}
	var row *Link
	for i := range links {
		if links[i].ShortCode == code {
			row = &links[i]
		}
	}
	if row == nil {
		t.Fatalf("created link missing from owner list (%d rows)", len(links))
	}
	if row.OriginalURL != "https://example.com/crud" || row.ClickCount != 0 {
		t.Fatalf("row = %+v", row)
	}
	if len(row.Tags) != 2 || row.Tags[0] != "alpha" {
		t.Fatalf("tags = %v, want [alpha beta]", row.Tags)
	}
	if row.Status != "scheduled" {
		t.Fatalf("status = %q, want scheduled (future expiry set)", row.Status)
	}
	if row.ExpiresAt == nil || !row.ExpiresAt.UTC().After(time.Now().UTC()) {
		t.Fatalf("ExpiresAt = %v, want future UTC", row.ExpiresAt)
	}

	// Read path 3: single-row fetch (edit/redirect screen). GetLink is a
	// projected read: code, URL, rules, active, expiry — tags and Status
	// belong to the list path (scanLinks), asserted below instead.
	l, err := s.GetLink(code)
	if err != nil {
		t.Fatalf("GetLink: %v", err)
	}
	if l.DeviceRules != nil {
		t.Fatalf("empty device_rules must read as the nil zero value, got %v", l.DeviceRules)
	}
	if l.ExpiresAt == nil || !l.ExpiresAt.UTC().After(time.Now().UTC()) {
		t.Fatalf("GetLink ExpiresAt = %v, want future UTC", l.ExpiresAt)
	}

	// Update: tags change sticks (verified through the LIST path, the one
	// that materializes tags), device rules through GetLink.
	if err := s.UpdateLink(id, code, `{"ios":"https://ios.example.com"}`, `["gamma"]`); err != nil {
		t.Fatalf("UpdateLink: %v", err)
	}
	l2, err := s.GetLink(code)
	if err != nil {
		t.Fatalf("GetLink after update: %v", err)
	}
	if len(l2.DeviceRules) != 1 || l2.DeviceRules["ios"] != "https://ios.example.com" {
		t.Fatalf("device rules not persisted: %v", l2.DeviceRules)
	}
	links2, err := s.ListLinksByCreator(id)
	if err != nil {
		t.Fatalf("ListLinksByCreator after update: %v", err)
	}
	for _, row2 := range links2 {
		if row2.ShortCode != code {
			continue
		}
		if len(row2.Tags) != 1 || row2.Tags[0] != "gamma" {
			t.Fatalf("tags after update = %v, want [gamma]", row2.Tags)
		}
	}

	// Delete: the row must vanish from every read path.
	if err := s.DeleteLink(id, code); err != nil {
		t.Fatalf("DeleteLink: %v", err)
	}
	if _, err := s.GetURL(code); err != sql.ErrNoRows {
		t.Fatalf("GetURL after delete: err = %v, want sql.ErrNoRows", err)
	}
	if _, err := s.GetLink(code); err != sql.ErrNoRows {
		t.Fatalf("GetLink after delete: err = %v, want sql.ErrNoRows", err)
	}
}

// TestIntegrationUniqueUsername: the duplicate-username path relies on the
// DB UNIQUE constraint (no check-then-insert race), and the driver's
// 23505 error must surface to the caller (the handler maps it to 409).
func TestIntegrationUniqueUsername(t *testing.T) {
	s := openTestStore(t)

	name := uniqUser("itguniq")
	if _, err := s.CreateCreator(name, "First", "", "h"); err != nil {
		t.Fatalf("first CreateCreator: %v", err)
	}
	if _, err := s.CreateCreator(name, "Second", "", "h"); err == nil {
		t.Fatal("duplicate username succeeded, want unique-violation error")
	}
}

// TestIntegrationNoRowsContract: every miss is sql.ErrNoRows — the exact
// value handlers match to answer 404 instead of 500.
func TestIntegrationNoRowsContract(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.GetURL(uniqCode("none")); err != sql.ErrNoRows {
		t.Errorf("GetURL unknown: err = %v, want sql.ErrNoRows", err)
	}
	if _, err := s.GetLink(uniqCode("none")); err != sql.ErrNoRows {
		t.Errorf("GetLink unknown: err = %v, want sql.ErrNoRows", err)
	}
	if _, err := s.GetCreatorByID(999999999); err != sql.ErrNoRows {
		t.Errorf("GetCreatorByID unknown: err = %v, want sql.ErrNoRows", err)
	}
}

// TestIntegrationExpiredLinkStored: an expiry written in the past survives
// the round trip and reads back with Status "expired" (the redirect handler
// relies on this to answer 410 instead of redirecting).
func TestIntegrationExpiredLinkStored(t *testing.T) {
	s := openTestStore(t)

	id, err := s.CreateCreator(uniqUser("itgexp"), "E", "", "h")
	if err != nil {
		t.Fatalf("CreateCreator: %v", err)
	}
	code := uniqCode("itgx")
	past := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)
	if err := s.CreateURL(code, "https://example.com/gone", &id, "[]", &past); err != nil {
		t.Fatalf("CreateURL: %v", err)
	}
	// GetLink proves the expiry TIMESTAMP round-trips; the LIST path
	// (scanLinks) is the one that derives Status, so assert it there.
	l, err := s.GetLink(code)
	if err != nil {
		t.Fatalf("GetLink: %v", err)
	}
	if l.ExpiresAt == nil {
		t.Fatal("ExpiresAt lost in round trip")
	}
	if !l.ExpiresAt.UTC().Before(time.Now().UTC()) {
		t.Fatalf("ExpiresAt = %v, want past", l.ExpiresAt)
	}
	links, err := s.ListLinksByCreator(id)
	if err != nil {
		t.Fatalf("ListLinksByCreator: %v", err)
	}
	found := false
	for _, row := range links {
		if row.ShortCode == code {
			found = true
			if row.Status != "expired" {
				t.Fatalf("status = %q, want expired", row.Status)
			}
		}
	}
	if !found {
		t.Fatal("expired link missing from owner list")
	}
}

// TestIntegrationOwnerScoping: mutation queries carry creator_id in their
// WHERE clause — creator B touching creator A's link affects zero rows and
// surfaces sql.ErrNoRows (the object-level authorization guarantee: knowing
// a short code is not enough to edit or delete it).
func TestIntegrationOwnerScoping(t *testing.T) {
	s := openTestStore(t)

	a, err := s.CreateCreator(uniqUser("itgowna"), "A", "", "h")
	if err != nil {
		t.Fatalf("CreateCreator A: %v", err)
	}
	b, err := s.CreateCreator(uniqUser("itgownb"), "B", "", "h")
	if err != nil {
		t.Fatalf("CreateCreator B: %v", err)
	}

	code := uniqCode("itgo")
	if err := s.CreateURL(code, "https://example.com/owned", &a, "[]", nil); err != nil {
		t.Fatalf("CreateURL: %v", err)
	}

	if err := s.UpdateLink(b, code, `{"ios":"https://evil.example"}`, `["hijack"]`); err != sql.ErrNoRows {
		t.Errorf("foreign UpdateLink: err = %v, want sql.ErrNoRows", err)
	}
	if err := s.DeleteLink(b, code); err != sql.ErrNoRows {
		t.Errorf("foreign DeleteLink: err = %v, want sql.ErrNoRows", err)
	}
	// The link must be untouched by the failed foreign writes.
	l, err := s.GetLink(code)
	if err != nil {
		t.Fatalf("GetLink: %v", err)
	}
	if len(l.Tags) != 0 || len(l.DeviceRules) != 0 {
		t.Fatalf("foreign mutation changed the row: %+v", l)
	}
	// The real owner can still delete.
	if err := s.DeleteLink(a, code); err != nil {
		t.Fatalf("owner DeleteLink: %v", err)
	}
}

// TestIntegrationBulkConflictAndFlags drives the remaining write surface
// against real Postgres: bulk import (per-row ON CONFLICT skips, order
// preserved), the featured/expiry/expiry-clear toggles, click counting,
// API-key storage (hash-at-rest round trip), and the email-unique coercion
// to ErrEmailTaken.
func TestIntegrationBulkConflictAndFlags(t *testing.T) {
	s := openTestStore(t)

	id, err := s.CreateCreator(uniqUser("itgbulk"), "Bulk", "", "h")
	if err != nil {
		t.Fatalf("CreateCreator: %v", err)
	}

	// Bulk: three rows, the middle one conflicts with a pre-existing code.
	taken := uniqCode("itgt")
	if err := s.CreateURL(taken, "https://example.com/pre", &id, "[]", nil); err != nil {
		t.Fatalf("seed CreateURL: %v", err)
	}
	c1, c2, c3 := uniqCode("itb1"), uniqCode("itb2"), uniqCode("itb3")
	codes, err := s.CreateURLsBatch(&id, []BulkURL{
		{ShortCode: c1, OriginalURL: "https://example.com/1"},
		{ShortCode: taken, OriginalURL: "https://example.com/conflict"},
		{ShortCode: c3, OriginalURL: "https://example.com/3"},
	})
	if err != nil {
		t.Fatalf("CreateURLsBatch: %v", err)
	}
	if codes[0] != c1 || codes[2] != c3 {
		t.Fatalf("codes = %v, want [%s <empty> %s]", codes, c1, c3)
	}
	if codes[1] != "" {
		t.Fatalf("conflict slot = %q, want empty (skipped row)", codes[1])
	}
	// The conflicting row must NOT have been overwritten.
	if got, _ := s.GetURL(taken); got != "https://example.com/pre" {
		t.Fatalf("conflict overwrote the existing row: %q", got)
	}
	if got, _ := s.GetURL(c1); got != "https://example.com/1" {
		t.Fatalf("bulk row 1 missing: %q", got)
	}
	_ = c2 // reserved: keeps the numbering readable if the batch grows

	// Featured toggle: owner flips it on (and a second link gets
	// unfeatured by the radio behavior), then off again.
	if err := s.SetFeaturedLink(id, c1, true); err != nil {
		t.Fatalf("SetFeaturedLink on: %v", err)
	}
	if err := s.SetFeaturedLink(id, c3, true); err != nil {
		t.Fatalf("SetFeaturedLink second: %v", err)
	}
	links, err := s.ListLinksByCreator(id)
	if err != nil {
		t.Fatalf("ListLinksByCreator: %v", err)
	}
	featured := 0
	for _, l := range links {
		if l.IsFeatured {
			featured++
		}
	}
	if featured != 1 {
		t.Fatalf("featured count = %d, want exactly 1 (radio semantics)", featured)
	}
	if err := s.SetFeaturedLink(id, c3, false); err != nil {
		t.Fatalf("SetFeaturedLink off: %v", err)
	}

	// Expiry set + clear round trip.
	exp := time.Now().UTC().Add(6 * time.Hour).Truncate(time.Second)
	if err := s.SetLinkExpiry(id, c1, &exp); err != nil {
		t.Fatalf("SetLinkExpiry: %v", err)
	}
	if l, err := s.GetLink(c1); err != nil || l.ExpiresAt == nil {
		t.Fatalf("expiry not persisted: l=%+v err=%v", l, err)
	}
	if err := s.SetLinkExpiry(id, c1, nil); err != nil {
		t.Fatalf("SetLinkExpiry clear: %v", err)
	}
	if l, err := s.GetLink(c1); err != nil || l.ExpiresAt != nil {
		t.Fatalf("expiry not cleared: l=%+v err=%v", l, err)
	}

	// Click counter: unknown code is a no-op success, known code increments.
	if err := s.IncrementClickCount(uniqCode("none")); err != nil {
		t.Fatalf("IncrementClickCount unknown: %v", err)
	}
	if err := s.IncrementClickCount(c1); err != nil {
		t.Fatalf("IncrementClickCount: %v", err)
	}

	// API keys: the stored value is the HASH (never the raw key), listing
	// is owner-scoped, and delete removes exactly that key.
	keyHash := "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"
	keyID, err := s.StoreAPIKey(id, keyHash, "ci-key")
	if err != nil {
		t.Fatalf("StoreAPIKey: %v", err)
	}
	keys, err := s.ListAPIKeys(id)
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	foundKey := false
	for _, k := range keys {
		if k.ID == keyID && k.Label.String == "ci-key" {
			foundKey = true
		}
	}
	if !foundKey {
		t.Fatalf("stored key missing from list (id=%d, %d keys)", keyID, len(keys))
	}
	if err := s.DeleteAPIKey(id, keyID); err != nil {
		t.Fatalf("DeleteAPIKey: %v", err)
	}
	if err := s.DeleteAPIKey(id, keyID); err != sql.ErrNoRows {
		t.Fatalf("second DeleteAPIKey: err = %v, want sql.ErrNoRows", err)
	}

	// Email uniqueness coerces the driver's 23505 to ErrEmailTaken (the
	// exact error the handler matches for the 409 response).
	other, err := s.CreateCreator(uniqUser("itgmail2"), "Other", "", "h")
	if err != nil {
		t.Fatalf("CreateCreator other: %v", err)
	}
	if err := s.UpdateCreatorEmail(id, "itg-taken@example.com"); err != nil {
		t.Fatalf("UpdateCreatorEmail first: %v", err)
	}
	if err := s.UpdateCreatorEmail(other, "itg-taken@example.com"); err != ErrEmailTaken {
		t.Fatalf("duplicate email: err = %v, want ErrEmailTaken", err)
	}
}
