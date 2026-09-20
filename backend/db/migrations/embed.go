// Package migrations membundel semua file migrasi SQL ke dalam binary lewat
// go:embed, sehingga auto-migrate di cmd/server tidak bergantung pada working
// directory saat server dijalankan (aman dipanggil dari cmd/* maupun saat
// binary dipindah). Folder ini hanya berisi *.sql + embed.go sebagai
// "classpath"-nya — semacam data embedded ala goose (-- +goose Up marker).
//
// LEARN:
//   Kenapa: Server membaca file SQL saat boot untuk auto-migrate (lihat
//   internal/migrate). Kalau file tidak di-embed melainkan dibaca dari path
//   relatif ke cwd (mis. "db/migrations"), server akan diam-diam gagal (atau
//   salah fallback) ketika dijalankan dari direktori lain — persis jebakan
//   working-directory yang pernah bikin load test tidak konsisten. Embed
//   menghilangkan seluruh kelas bug itu: file migrasi ikut binary, tidak ada
//   lookup path runtime. Trade-off: ganti file migrasi = build ulang binary
//   (wajar, karena migrasi adalah bagian dari rilis), bukan hot-swap.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS

// Names mengembalikan daftar nama file migrasi (termasuk .down.sql; pemanggil
// yang menyaring sendiri, hanya punya filesystem tidak boleh berasumsi isi).
func Names() ([]string, error) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}
