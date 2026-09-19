# PROGRESS.md — Jejak System Design Learning

Catatan progres per fase yang telah diselesaikan.

## Refleksi Fase 1 (Load Testing Baseline)
Fase 1 mengimplementasikan load testing baseline menggunakan `hey` untuk mengukur requests/second dan latency sebelum optimasi. Konsep utamanya adalah membuat "bukti" performa awal sebagai acuan perbandingan setiap fase selanjutnya (Fase 2 caching, Fase 3 LB, dst). Tanpa baseline, tidak bisa menentukan apakah optimasi meningkatkan performa atau malah memburuk akibat overhead baru.

## Refleksi Fase 2 (Cache-Aside)
Fase 2 mengimplementasikan cache-aside pattern menggunakan Redis dengan TTL 5 menit. Konsep utamanya adalah menyederhanakan jalur baca: cek cache dulu, kalau miss baru query DB. Trade-off utama adalah read latency menjadi 2x saat cache miss (cache -> DB query), namun write tetap cepat karena tidak harus update cache. Implementasi ini mengurangi beban DB pada lalu lintas read tinggi, meski perlu mengelola cache expiration dan invalidation.

## Hasil Smoke Test Fase 0–1 (native, 2026-09-08)

Setup: 1 instance (`PORT=8081`, `APP_MODE=full`), `REDIS_URL` kosong
(→ tanpa cache, click logging sinkron), PostgreSQL trust mode,
seed `abc123` di kedua database.

- `POST /api/shorten` → **201**, short URL `http://localhost:8080/r/iuKA2R`.
  **Bug ditemukan:** port respons hardcode `8080` (`handler.go:98`) padahal
  server jalan di 8081. Diperbaiki dengan membentuk short URL dari request
  masuk (`r.Host` + scheme, hormat `X-Forwarded-Proto`); `go build` exit 0,
  server di-restart. Shorten berikutnya me-return port yang benar.
- `GET /r/iuKA2R` → **302, `Location: https://google.com`** ✅. Sempat
  butuh sync manual baris `iuKA2R` ke replica dulu (tulisan hanya ke primary)
  — tanpa itu redirect 404, yang juga bagian dari demo lag di bawah.
- `GET /r/iuKA2R/clicks` → **`{"click_count":0}`** — sesuai prediksi, bukan bug.
  Verifikasi `psql`: primary `iuKA2R.click_count=2`, replica `=0`. Increment
  ditulis sinkron ke primary, tapi endpoint clicks membaca dari replica yang
  belum di-sync: demo langsung replication lag / read-your-write inconsistency
  ala simulasi replica Fase 4 (lihat ARCHITECTURE.md §6).

Kesimpulan: jalur tulis, baca, dan counting Fase 0 terbukti benar end-to-end;
infra native (2 DB + env) siap menjadi arena load testing Fase 1.

## Hasil Load Test Baseline — Fase 1 (native, 2026-09-08)

Perintah (parameter ini WAJIB sama persis di semua fase berikutnya):
`go run ./cmd/loadtest -url http://localhost:8081/r/abc123 -n 200 -c 10`
Kondisi: 1 instance, `APP_MODE=baseline`, tanpa cache/replica/queue.

```
RESULT url=http://localhost:8081/r/abc123 n=200 c=10
RESULT wall=383ms rps=521.88 ok=200 errors=0
RESULT latency avg=15.715ms p50=2.634ms p95=93.865ms
RESULT status=map[302:200]
```

Interpretasi the developer: 0 error dan 302 semua = sistem sehat secara
fungsional. p95 93.9ms belum bisa dinilai cepat/lambat karena belum ada
target — diterima sebagai baseline apa adanya. Tindak lanjut: tentukan NFR
(target latency) yang jelas sebelum menilai hasil fase-fase berikutnya.

## Hasil Load Test Full Stack (cache + replica + queue, 2026-09-08)

Perintah identik dengan baseline:
`go run ./cmd/loadtest -url http://localhost:8081/r/abc123 -n 200 -c 10`
Kondisi: 1 instance, `APP_MODE=full`, Redis native hidup (cache-aside +
queue async aktif), baca dari replica, worker jalan.

```
RESULT url=http://localhost:8081/r/abc123 n=200 c=10
RESULT wall=278ms rps=719.48 ok=200 errors=0
RESULT latency avg=11.281ms p50=1.996ms p95=81.947ms
RESULT status=map[302:200]
```

Interpretasi the developer: full stack lebih cepat di semua metrik — RPS
naik 521.88 → 719.48 (+38%), p95 turun 93.9ms → 81.9ms. Optimasi kebukti
membantu, bukan cuma asumsi.

NFR yang ditetapkan: **p95 < 100ms** untuk endpoint redirect (zona "terasa
instan" menurut skala persepsi Google; juga di bawah bar internal
microservice <100ms). Verdict: **full memenuhi target** (81.9 < 100).
Catatan jujur: baseline (93.9) sebenarnya juga di bawah 100, tapi marginnya
tipis — full memberi headroom yang jelas.

## Hasil Smoke Test Backend Fase 9 (native, 2026-09-08)

Server: 1 instance `:8081` (kode Fase 9, restart oleh the developer).
Alur uji: register → shorten sebagai owner → halaman publik kreator.

- `POST /api/register` `{"username":"ghifari",...}` → **201**
  `{"id":1,"username":"ghifari"}` + cookie `jejak_session` ✅.
  Catatan: percobaan pertama 400 karena JSON inline rusak oleh quoting
  shell agen — diulangi via file (`-d @register.json`) lalu lolos.
- `POST /api/shorten` (dengan cookie) → **201**
  `http://localhost:8081/r/s5qEag` ✅ — port respons sudah benar
  (bug hardcode 8080 dari smoke test sebelumnya terbukti sembuh).
  Verifikasi DB primary: `s5qEag.creator_id=1 → ghifari` ✅.
- `GET /api/u/ghifari` → pertama **404** (baca dari replica yang belum
  punya baris creator/link — perilaku simulasi replica yang
  terdokumentasi, bukan bug). Setelah sync manual kedua baris ke replica
  (+ perbaiki sequence): **200** dengan `links:[{s5qEag, click_count:0}]`,
  array TIDAK kosong ✅.

Kesimpulan: relasi creator_id nyambung end-to-end (session → owner →
single-query list dengan click_count denormalized, tanpa N+1). Backend
Fase 9 siap; frontend redesign (DESIGN.md) menunggu giliran.

## Bugfix: custom slug >10 char 500 (2026-09-08)

Gejala: `POST /api/shorten` slug `barang_kesukaan` (16 char) → 500
"Failed to store URL" berulang. Akar masalah: validasi `ValidSlug`
mengizinkan s.d. 30 char, tapi kolom `short_code` masih `VARCHAR(10)`
— INSERT gagal `value too long`, dan handler tidak me-log error asli
sehingga terminal server buta. Bukan regresi fix cookie analytics
(jalur kode tidak bersinggungan; terbukti via repro psql langsung).
Fix: migrasi `ALTER COLUMN short_code TYPE VARCHAR(30)` di `urls` +
`click_events` (keduanya, kedua DB) + `Logger.Printf` saat `CreateURL`
gagal + SCHEMA.md. Retest: shorten → **201**
`http://localhost:8081/r/barang_kesukaan`, sync manual ke replica,
redirect → **302** ke URL asli ✅.