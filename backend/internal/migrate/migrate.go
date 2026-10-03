// Package migrate runs the SQL migration files (goose-style, markers
// "-- +goose Up"/"-- +goose Down") automatically at server boot, against the
// PRIMARY database only.
//
// Auto-migrate exists because since Fase 1 migrations could only be applied
// manually (psql/goose), which left a trail of recurring bugs that were almost
// undetectable:
//   - The local `jejak` database lagged behind migrations 12 and 13 (columns
//     such as is_active and unique_click_count were missing). The consequence
//     did not surface in INSERT/shorten (old columns only) but in
//     SELECT/redirect:
//   - GET /r/{code} selects is_active -> ERROR column does not exist ->
//     every newly created link answers 404.
//   - GET /api/profile (ListLinksByCreatorPrimary) selects is_active ->
//     the same error -> 500 "Database error".
//     This is an asymptomatic trap: the application "works" (register/login/
//     shorten OK) while the core features (live redirect + dashboard) are
//     dead - exactly what a smoke test nearly missed.
//     Principles of auto-migrate:
//   - Only the PRIMARY (DATABASE_URL) is migrated automatically. The REPLICA
//     IS NOT - replica synchronization stays manual, per the project rule
//     (see README: the replica is synced through separate migrations).
//     Auto-migrating the replica could pull it away from the operator's
//     manual configuration.
//   - Safe to re-run (idempotent): every version is recorded in the
//     schema_migrations table, so the next boot skips applied versions.
//   - Tolerates "already exists": some of our migration files are not
//     idempotent (migration 13 uses ADD COLUMN without IF NOT EXISTS whereas
//     01-12 all use IF NOT EXISTS/IF EXISTS). If the database already has the
//     column (e.g. it was migrated manually to 13 before auto-migrate was
//     enabled), the ADD COLUMN statement fails with duplicate_column - that
//     is treated as "already present" and skipped, because the only goal is
//     to ensure the column EXISTS, never to repeat a dangerous DELETE.
//     Trade-off: new non-idempotent migrations are expected to use IF NOT
//     EXISTS so that auto-migrate stays stable; that is documented in README.
//     Alternative considered: pulling in the goose/pressly dependency.
//   - Rejected: the project deliberately avoids external dependencies for
//     non-trivial choices (the env loader is hand-written, see internal/env;
//     the embed here is hand-written too). One external dependency to "run N
//     SQL files and record versions" is not worth ~80 explicit, readable
//     lines of our own.
//     Alternative considered (2): reading the migration files from a disk path
//     at boot.
//   - Rejected: fragile with respect to the working directory (the same trap
//     as .env - which once made load tests inconsistent). The migration files
//     are embedded (db/migrations/embed.go) so the binary is self-contained.
package migrate

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"jejak/db/migrations"
)

// Run makes sure every migration is present in the primary DB. It is
// idempotent and safe to call on each boot. logger carries the messages - call
// it for the PRIMARY only (NEVER for the replica).
// Returns a wrapped error (fmt.Errorf with %w, prefixed "migrate: ..." or
// "migrate <file>: ...") when the schema_migrations bootstrap, the version
// read, the embedded file listing, statement parsing, or a migration
// execution fails; cmd/server treats that as fatal and refuses to start.
// Already-applied versions and tolerated duplicate_column retries (see the
// package comment) are skips, not errors.
func Run(db *sql.DB, logger *log.Logger) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`); err != nil {
		return fmt.Errorf("migrate: create schema_migrations: %w", err)
	}

	applied := map[string]bool{}
	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("migrate: read versions: %w", err)
	}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return fmt.Errorf("migrate: scan version: %w", err)
		}
		applied[v] = true
	}
	rows.Close()

	names, err := migrations.Names()
	if err != nil {
		return fmt.Errorf("migrate: list embedded files: %w", err)
	}
	sort.Strings(names)

	var appliedCount int
	for _, name := range names {
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		version := strings.TrimSuffix(name, ".sql")
		if applied[version] {
			continue
		}

		stmts, err := upStatements(name)
		if err != nil {
			return fmt.Errorf("migrate %s: %w", name, err)
		}
		executed := 0
		for _, stmt := range stmts {
			if _, err := db.Exec(stmt); err != nil {
				// Tolerate "already exists" so a database that was migrated
				// manually to a non-idempotent version (e.g. 13) still boots;
				// see the package comment above.
				if !isAlreadyExists(err) {
					return fmt.Errorf("migrate %s: %w\nSQL: %s", name, err, stmt)
				}
				logger.Printf("[migrate] %s: kolom/tabel sudah ada: dilewati (%s)", name, stmt)
				continue
			}
			executed++
		}
		if _, err := db.Exec(`INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			return fmt.Errorf("migrate %s: record version: %w", name, err)
		}
		appliedCount++
		logger.Printf("[migrate] %s: diterapkan (%d statement berhasil)", name, executed)
	}

	logger.Printf("[migrate] selesai: %d migrasi baru diterapkan, %d sudah ada", appliedCount, len(applied))
	return nil
}

// upStatements extracts the SQL statements from the "Up" section of a
// migration file (between the "-- +goose Up" and "-- +goose Down" markers, or
// the end of the file), dropping comment lines and markers.
func upStatements(name string) ([]string, error) {
	data, err := migrations.FS.ReadFile(name)
	if err != nil {
		return nil, err
	}
	body := upBody(string(data))
	if body == "" {
		return nil, nil
	}
	var stmts []string
	for _, stmt := range splitStatements(body) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		stmts = append(stmts, stmt)
	}
	return stmts, nil
}

var upMarker = regexp.MustCompile(`(?m)^--\s*\+goose Up\s*$`)
var downMarker = regexp.MustCompile(`(?m)^--\s*\+goose Down\s*$`)

func upBody(content string) string {
	up := upMarker.FindStringIndex(content)
	if up == nil {
		return ""
	}
	body := content[up[1]:]
	if d := downMarker.FindStringIndex(body); d != nil {
		body = body[:d[0]]
	}
	return body
}

// splitStatements splits an SQL block into statements on ";", ignoring
// semicolons that appear inside '...' string literals and inside comment lines.
func splitStatements(s string) []string {
	var out []string
	var cur strings.Builder
	var inStr, inComment bool
	for i := 0; i < len(s); i++ {
		r := s[i]
		switch {
		case inComment:
			// Ignore the entire comment line up to the newline: it starts at
			// "--" and ends at '\n'. Comment prose (e.g. "if the device does
			// not match...") must never be executed as SQL.
			if r == '\n' {
				inComment = false
				cur.WriteByte(r)
			}
		case inStr:
			cur.WriteByte(r)
			if r == '\'' {
				inStr = false
			}
		case r == '-' && i+1 < len(s) && s[i+1] == '-':
			// "--" starts a line comment: skip to the end of the line and
			// never write it into the statement.
			inComment = true
			i++
		case r == '\'':
			inStr = true
			cur.WriteByte(r)
		case r == ';':
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// isAlreadyExists reports whether err is a Postgres duplicate error
// (column/table/object/index/database), so non-idempotent migrations stay
// safe to re-run.
func isAlreadyExists(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	switch pgErr.Code {
	case "42701", // duplicate_column
		"42P07", // duplicate_table
		"42P11", // duplicate_object
		"42710", // duplicate_object (schema/group/role)
		"42P04", // duplicate_database
		"42723": // duplicate_function
		return true
	}
	return false
}
