// Package migrate menjalankan file migrasi SQL (goose-style, marker
// "-- +goose Up"/"-- +goose Down") secara otomatis saat server boot,
// terhadap database PRIMARY saja.
//
// LEARN:
//
//	Kenapa ada auto-migrate: Sejak Fase 1 migrasi HANYA bisa jalan manual
//	(psql/goose). Dampak nyata — riwayat bug berulang yang nyaris tidak
//	terdeteksi:
//	  * DB `jejak` lokal tertinggal migrasi 12 & 13 (kolom is_active,
//	    unique_click_count, dsb. belum ada). Konsekuensinya TIDAK muncul di
//	    INSERT/shorten (yang cuma kolom lama) tapi di SELECT/redirect:
//	      - GET /r/{code} SELECT menyebut is_active → ERROR column does not
//	        exist → semua link yang baru dibuat malah 404.
//	      - GET /api/profile (ListLinksByCreatorPrimary) SELECT menyebut
//	        is_active → error yang sama → 500 "Database error".
//	    Ini jebakan asimtomatik: aplikasi "jalan" (register/login/shorten
//	    ok) tapi fitur inti (redirect hidup + dashboard) mati — persis
//	    yang nyaris lolos smoke test kemarin.
//	Prinsip auto-migrate:
//	  * Hanya PRIMARY (DATABASE_URL) yang dimigrate otomatis. REPLICA
//	    TIDAK IKUT — sinkronisasi replica tetap manual, sesuai prinsip
//	    project (lihat README: replica di-sync lewat migrasi terpisah).
//	    Auto-migrate ke replica bisa bikin replica melenceng dari
//	    konfigurasi manual yang dikelola operator.
//	  * Aman dijalankan ulang (idempoten): setiap versi dicatat di tabel
//	    schema_migrations, jadi boot berikutnya me-skip yang sudah applied.
//	  * Toleran "already exists": file migrasi kami sebagian TIDAK
//	    idempoten (mis. migrasi 13 pakai ADD COLUMN tanpa IF NOT EXISTS
//	    SEDANGKAN 01-12 semuanya IF NOT EXISTS/IF EXISTS). Kalau DB sudah
//	    punya kolom (mis. sudah pernah dimigrate manual ke 13 lalu kita
//	    aktifkan auto-migrate), statement ADD COLUMN akan gagal
//	    duplicate_column — kita anggap itu "sudah ada" dan lanjut, karena
//	    tujuannya cuma memastikan kolom ADA, bukan mengulang DELETE yang
//	    berbahaya. Trade-off: ini berarti migrasi NON-idempoten baru ke
//	    depan dihimbau memakai IF NOT EXISTS supaya auto-migrate tetap
//	    stabil; dokumentasikan itu di README.
//	Alternatif yang dipertimbangkan: tarik dependency goose/pressly.
//	  - Batal: project ini sengaja dependency-free di pilihan non-sepele
//	    (env loader ditulis tangan, lihat internal/env; embed di sini juga
//	    ditulis tangan). Satu dependency eksternal untuk "jalankan N file
//	    SQL + catat versi" tidak sebanding dengan ~80 baris sendiri yang
//	    eksplisit dan bisa dibaca.
//	Alternatif yang dipertimbangkan (2): baca file dari path disk saat boot.
//	  - Batal: rapuh terhadap working-directory (jebakan yang sama dengan
//	    .env — dan pernah bikin load test tidak konsisten). File migrasi
//	    di-embed (db/migrations/embed.go) sehingga binary mandiri.
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

// Run memastikan semua migrasi tersedia di DB primary. idempoten; aman
// dipanggil tiap boot. logger dipakai untuk pesan — panggil hanya untuk
// primary (TIDAK untuk replica).
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
				// Toleransi "sudah ada" supaya DB yang sudah dimigrate manual
				// ke versi non-idempoten (mis. 13) tidak gagal boot. lihat
				// LEARN di atas.
				if !isAlreadyExists(err) {
					return fmt.Errorf("migrate %s: %w\nSQL: %s", name, err, stmt)
				}
				logger.Printf("[migrate] %s: kolom/tabel sudah ada — dilewati (%s)", name, stmt)
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

// upStatements mengekstrak statement SQL dari bagian "Up" sebuah file
// migrasi (antara marker "-- +goose Up" dan "-- +goose Down" atau akhir
// file), membuang baris komentar dan marker.
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

// splitStatements memecah blok SQL per-statement pada ";", dengan mengabaikan
// ; yang ada di dalam string literal '...' dan di dalam baris komentar.
func splitStatements(s string) []string {
	var out []string
	var cur strings.Builder
	var inStr, inComment bool
	for i := 0; i < len(s); i++ {
		r := s[i]
		switch {
		case inComment:
			// Abaikan seluruh isi baris komentar "..." sampai newline.
			// Mulai dari "--" dan berakhir di '\n'. Kalimat Indonesia di
			// komentar (mis. "kalau device tidak match...") tidak boleh
			// ikut dieksekusi sebagai SQL.
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
			// "--" memulai komentar baris: lewati sampai akhir baris,
			// TIDAK ikut ditulis ke statement.
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

// isAlreadyExists mengenali error duplicate (kolom/tabel/index/database)
// Postgres agar non-idempoten migration tetap aman dijalankan ulang.
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
