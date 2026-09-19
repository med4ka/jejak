package shortener

import (
	"crypto/md5"
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"
)

var validSlug = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,30}$`)

// ReservedSlugs adalah path level-1 yang dipakai aplikasi sendiri — slug
// custom tidak boleh sama persis dengan salah satunya, kalau tidak link
// publik seperti /r/api atau /u/login tidak akan pernah bisa diakses.
// Daftar eksplisit supaya mudah ditambah (satu baris per kata).
// LEARN:
//   Kenapa: Namespace short-code dibagi dengan routing aplikasi (/r, /u, /api,
//   halaman statis). Tanpa daftar ini, user bisa mengklaim "login" dan
//   men-deface route penting — ini open-redirect-adjacent footgun klasik.
//   Trade-off: Exact-match saja (bukan prefix): "apiku" tetap boleh, hanya
//   "api" persis yang ditolak. Daftar statis = restart untuk menambah kata.
//   Alternatif: Namespace terpisah (mis. semua link di /s/{code}) sehingga
//   tidak ada tabrakan sama sekali — tapi URL jadi lebih panjang.
var ReservedSlugs = []string{
	"api", "u", "r",
	"login", "register", "dashboard", "links",
	"admin", "static", "favicon.ico",
}

// IsReserved reports whether slug collides with app routes (case-insensitive).
func IsReserved(slug string) bool {
	lowered := strings.ToLower(slug)
	for _, r := range ReservedSlugs {
		if lowered == r {
			return true
		}
	}
	return false
}

// ValidSlug enforces 3-30 chars of [a-zA-Z0-9_-] (same alphabet as the
// random generator output, so custom and random codes share one namespace).
func ValidSlug(slug string) bool {
	return validSlug.MatchString(slug)
}

// LEARN:
//   Kenapa: Fungsi ini menghasilkan kode pendek acak (short code) yang unik untuk setiap URL yang dimasukkan. Konsep design yang terkait: collision handling - mengapa kita butuh cara pastikan short code unik dan menghindari konflik di database.
//   Trade-off: Randomness memungkinkan ruang nama yang besar (62^6 kombinasi untuk kode 6 karakter) tetapi ada risiko kolisi yang sangat kecil. Jika terlalu terstruktur, bisa membuat pola yang ditebak.
//   Alternatif: Bisa pakai UUID penuh, tetapi panjangnya terlalu panjang untuk URL. Bisa juga pakai increment ID lalu encode base62, tapi membutuhkan central counter.
func GenerateShortCode(length int) (string, error) {
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(62))
		if err != nil {
			return "", err
		}
		b[i] = letters[n.Int64()]
	}
	return string(b), nil
}

// LEARN:
//   Kenapa: Menentukan shard (lokasi fisik) di mana short_code akan disimpan berdasarkan hash MD5. Konsep design yang terkait: sharding - membagi data ke beberapa database berbeda supaya 1 database tidak menanggung semua beban. Fungsi ini memetakan short_code ke index shard (0, 1, dst).
//   Trade-off: Hash deterministik memungkinkan routing yang konsisten (same short_code -> same shard setiap kali), tetapi memerlukan logika penanganan di application layer untuk tahu shard mana yang harus dibaca/tulis. Jika jumlah shard berubah, perlu rebalancing.
//   Alternatif: Bisa pakai consistent hashing (ketimbang modulo sederhana) yang lebih fleksibel saat menambah shard, tetapi lebih kompleks diimplementasikan.
func ShardKey(shortCode string, numShards int) int {
	hash := md5.Sum([]byte(shortCode))
	hashInt := int64(hash[0])<<24 | int64(hash[1])<<16 | int64(hash[2])<<8 | int64(hash[3])
	return int(hashInt) % numShards
}

// LEARN:
//   Kenapa: Menentukan nama shard (seperti urls_shard_a, urls_shard_b) berdasarkan index shard. Konsep design yang terkait: sharding nomenclature - cara nama-tabel dibagi agar mudah diidentifikasi dan di-maintenance. Memisahkan data fisik ke shard_a/shard_b rather than semua di satu tabel urls.
//   Trade-off: Nama yang deskriptif memudahkan debugging dan monitoring (lihat beban di shard_a berapa), tetapi menambah complexity di code untuk menentukan nama shard berdasarkan index. Jika ingin menambah shard ke-3, perlu mengubah logic penamaan.
//   Alternatif: Bisa pakai nomor urut (shard_0, shard_1, shard_2) atau nama berdasarkan region, tetapi kurangi readability untuk kasus sederhana ini.
func ShardName(shortCode string, numShards int) string {
	idx := ShardKey(shortCode, numShards)
	return "urls_shard_" + string(rune('a' + idx))
}