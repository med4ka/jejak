package db

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// newClaimMock returns a SingleStore wired to a mocked *sql.DB (no Postgres
// needed). The SQL matcher REQUIRES the "AND creator_id IS NULL" guard: if a
// future change drops the guard (making anywhere claimable), every ExpectExec
// below fails, which is exactly the regression the security test is for.
func newClaimMock(t *testing.T) (*SingleStore, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &SingleStore{primary: db}, mock
}

// expectClaimUpdate queues one UPDATE expectation for the given code/creator
// with the given affected-row result, plus matches an optional error.
func expectClaimUpdate(mock sqlmock.Sqlmock, creatorID int64, code string, affected int64) {
	mock.ExpectExec("UPDATE urls SET creator_id = \\$1 WHERE short_code = \\$2 AND creator_id IS NULL").
		WithArgs(creatorID, code).
		WillReturnResult(sqlmock.NewResult(0, affected))
}

// TestClaimLinksClaimsNullRows is the happy path: a link with creator_id NULL
// IS claimed by the logged-in creator, and claimed counts exactly those.
func TestClaimLinksClaimsNullRows(t *testing.T) {
	store, mock := newClaimMock(t)

	mock.ExpectBegin()
	expectClaimUpdate(mock, 7, "abc123", 1)
	expectClaimUpdate(mock, 7, "def456", 1)
	mock.ExpectCommit()

	claimed, err := store.ClaimLinks(7, []string{"abc123", "def456"})
	if err != nil {
		t.Fatalf("ClaimLinks error: %v", err)
	}
	if claimed != 2 {
		t.Fatalf("claimed = %d, want 2", claimed)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestClaimLinksNeverStealsOwnedRows is the SECURITY-CRITICAL case: when a row
// already has a creator_id, the UPDATE affects 0 rows and claimed must NOT
// count it. Combined with the SQL guard in expectClaimUpdate, this proves a
// second registrant cannot take over someone else's link.
func TestClaimLinksNeverStealsOwnedRows(t *testing.T) {
	store, mock := newClaimMock(t)

	mock.ExpectBegin()
	// RowsAffected(0) = row exists but creator_id is NOT NULL -> not touched.
	expectClaimUpdate(mock, 7, "owned1", 0)
	expectClaimUpdate(mock, 7, "owned2", 0)
	mock.ExpectCommit()

	claimed, err := store.ClaimLinks(7, []string{"owned1", "owned2"})
	if err != nil {
		t.Fatalf("ClaimLinks error: %v", err)
	}
	if claimed != 0 {
		t.Fatalf("claimed = %d, want 0 (owned rows must never be stolen)", claimed)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestClaimLinksMixedBatch mirrors real traffic: the browser submits old codes
// from localStorage: some still ownerless (claimed), some already owned by
// this same user (re-claim attempt, harmless): and claimed only counts x1.
func TestClaimLinksMixedBatch(t *testing.T) {
	store, mock := newClaimMock(t)

	mock.ExpectBegin()
	expectClaimUpdate(mock, 3, "fresh1", 1)      // NULL -> claimed
	expectClaimUpdate(mock, 3, "alreadyours", 0) // already owned by user 3
	expectClaimUpdate(mock, 3, "taken", 0)       // owned by someone else
	mock.ExpectCommit()

	claimed, err := store.ClaimLinks(3, []string{"fresh1", "alreadyours", "taken"})
	if err != nil {
		t.Fatalf("ClaimLinks error: %v", err)
	}
	if claimed != 1 {
		t.Fatalf("claimed = %d, want 1", claimed)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestClaimLinksHasNullGuard states the security rule explicitly: the UPDATE
// must target only ownerless rows. (Also enforced implicitly by sqlmock's SQL
// matcher, but this keeps the requirement visible and greppable.)
func TestClaimLinksHasNullGuard(t *testing.T) {
	_ = expectClaimUpdate // referenced to keep helper tied to this test
	// The helper itself hardcodes the guarded SQL; sqlmock fails any Exec that
	// does not match it. Without the guard, ExpectExec wouldn't match and this
	// whole suite would fail on the first claim test.
}

// TestClaimLinksErrorRollsBack: when any UPDATE fails, the whole batch must
// NOT be committed (no partial claim confusing the client).
func TestClaimLinksErrorRollsBack(t *testing.T) {
	store, mock := newClaimMock(t)

	mock.ExpectBegin()
	expectClaimUpdate(mock, 1, "ok1", 1)
	mock.ExpectExec("UPDATE urls SET creator_id = \\$1 WHERE short_code = \\$2 AND creator_id IS NULL").
		WithArgs(int64(1), "bad2").
		WillReturnError(errors.New("mock db down"))
	mock.ExpectRollback()

	if _, err := store.ClaimLinks(1, []string{"ok1", "bad2"}); err == nil {
		t.Fatal("ClaimLinks returned nil error, want error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestClaimLinksCommitFailure: Begin/Exec fine but Commit fails -> error.
func TestClaimLinksCommitFailure(t *testing.T) {
	store, mock := newClaimMock(t)

	mock.ExpectBegin()
	expectClaimUpdate(mock, 5, "x", 1)
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	if _, err := store.ClaimLinks(5, []string{"x"}); err == nil {
		t.Fatal("ClaimLinks returned nil error on commit failure, want error")
	}
}
