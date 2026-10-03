// Command migrate-replica runs ALL SQL migrations (exactly the same set the
// server auto-migrate uses) against the REPLICA: not the primary.
//
// Rationale: auto-migrate in cmd/server deliberately targets ONLY the primary
// (DATABASE_URL): see the rationale in internal/migrate: the replica may be
// managed manually by an operator and must not be silently altered at every
// boot. As a result, schema migrations that were never applied to the replica
// accumulate silently, and the bug is ASYMPTOMATIC: the application keeps
// looking "alive" (register, login, shorten: all through the primary) until a
// READ path touches a column that exists only on the primary. Real case
// (2026-09-30): migration 15 (urls.expires_at) was applied to the primary only
// → in APP_MODE=full, GetLink SELECTs expires_at → error on the replica →
// every redirect /r/{code} answered 404 even though the link row existed.
// Synchronizing the replica schema is the root fix.
// Trade-off: running twice (primary at boot, replica manually) is easier to
// forget than a single command; therefore usage is recorded in README §3 and
// this command is made as simple as possible: one command, idempotent, no
// required arguments, safe to repeat at any time (each version is recorded in
// the replica's own schema_migrations).
// Alternatives rejected: auto-migrating the replica at boot in full mode:
// it conflicts with the original principle (the operator manages the replica)
// and adds a boot failure point; pg_dump -s primary | psql replica: rejected
// for the main path because it also copies state (including indexes that may
// deliberately differ) and creates untracked drift; retained only as an
// emergency fallback in README.
package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"regexp"
	"strings"

	"jejak/internal/env"
	"jejak/internal/migrate"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// main runs the migration against the replica (default) or the primary.
//
//	-target=replica   DSN taken from DATABASE_REPLICA_URL (default)
//	-target=primary   DSN taken from DATABASE_URL (to re-verify the primary)
//	-url=...          overrides any DSN (for ad-hoc databases / other containers)
//
// No other flags are intentional: this command must stay as short as possible
// so it is easy to remember ("go run ./cmd/migrate-replica") and less likely
// to hit the wrong target.
func main() {
	env.LoadDotEnv()

	target := flag.String("target", "replica", "replica | primary")
	urlOverride := flag.String("url", "", "override DSN (kalau kosong: DATABASE_REPLICA_URL / DATABASE_URL)")
	flag.Parse()

	key := "DATABASE_REPLICA_URL"
	if strings.TrimSpace(*target) == "primary" {
		key = "DATABASE_URL"
	} else if strings.TrimSpace(*target) != "replica" {
		log.Fatalf("-target harus replica atau primary, bukan %q", *target)
	}

	dsn := strings.TrimSpace(*urlOverride)
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv(key))
	}
	if dsn == "" {
		log.Fatalf("%s kosong: isi dulu di .env (lihat .env.example) atau pakai -url=...", key)
	}

	// #nosec G706 -- *target is a validated flag (only "primary"/"replica"
	// reach this line, enforced above) and the DSN is masked before logging:
	// this is a local CLI for the operator, not request data.
	log.Printf("[migrate-replica] target=%s dsn=%s", *target, maskDSN(dsn))

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("buka koneksi: %v", err)
	}
	defer db.Close()

	// Ping first: connection errors (database missing, wrong password) must be
	// clear and immediate rather than the misleading "migrate: create
	// schema_migrations".
	if err := db.Ping(); err != nil {
		log.Fatalf("koneksi ke %s gagal: %v", key, err)
	}

	if err := migrate.Run(db, log.Default()); err != nil {
		log.Fatalf("migrasi gagal: %v", err)
	}
	log.Printf("[migrate-replica] selesai: jalankan verifikasi, mis. psql -d <db> -c \"\\d urls\"")
}

// maskDSN hides the password in logs (credentials must never leak into the
// terminal/log: the same pattern used in the project's documentation).
var userInfoRe = regexp.MustCompile(`(?i)(://[^:/@]+:)[^@]+@`)

func maskDSN(dsn string) string {
	return userInfoRe.ReplaceAllString(dsn, "$1***@")
}
