# SCHEMA.md — Jejak

## Skema Database (PostgreSQL)

### `urls`
Tabel inti — mapping short-code ke URL asli.

| Kolom | Tipe | Catatan |
|---|---|---|
| id | UUID/serial | PK |
| short_code | varchar(30), unique | Kode pendek (mis. `abc123`; custom slug s.d. 30 char — kolom awalnya varchar(10), dilebarkan saat fitur custom slug karena validasi sudah 30) |
| original_url | text | URL tujuan |
| created_at | timestamp | |
| click_count | bigint, default 0 | Denormalized counter (dibahas trade-off-nya di Fase 2 — konsisten vs cepat) |

> **Fase 12:** kolom `expires_at` dihapus — dideklarasikan sejak Fase 0 tetapi tak pernah ditulis/dibaca (tidak ada fitur expiry di PRD). Lihat `backend/db/migrations/20240112_12_drop_dead_columns.sql`.

> **[Fase 5 — Sharding]:** Saat masuk fase sharding, tabel ini akan dipecah jadi `urls_shard_a` / `urls_shard_b` (atau sejenisnya) berdasarkan hash `short_code`. Strategi hash-nya WAJIB diisi the developer sendiri di ARCHITECTURE.md §4 sebelum migration dibuat — jangan diputuskan oleh AI.

### `click_events`
Log tiap klik — dipakai untuk analytics. Ditulis ASYNC mulai Fase 6 (lihat PRD.md §1), sebelum itu boleh sinkron dulu di Fase 0-5 (biar bisa ngerasain masalah performanya dulu sebelum diperbaiki).

| Kolom | Tipe | Catatan |
|---|---|---|
| id | UUID/serial | PK |
| short_code | varchar(30) | FK logis ke `urls.short_code` (ikut dilebarkan — redirect logging kena tembok varchar(10) yang sama) |
| clicked_at | timestamp | |
| referrer | varchar, nullable | Domain pengirim traffic, kalau ada |

> **Fase 12:** kolom `country` dihapus — geo-IP bukan scope PRD; kolom NULL selamanya hanya menambah noise. Lihat `backend/db/migrations/20240112_12_drop_dead_columns.sql`.

## Fase 9 — Profil Kreator

### `creators`
Akun kreator — pemilik dari sekumpulan short-link yang ditampilkan di 1 halaman publik.

| Kolom | Tipe | Catatan |
|---|---|---|
| id | UUID/serial | PK |
| username | varchar(30), unique | Dipakai di URL publik `/u/{username}` — validasi format (alfanumerik + underscore) di level aplikasi |
| display_name | varchar | Nama yang tampil di halaman profil |
| bio | text, nullable | Deskripsi singkat di halaman profil |
| password_hash | varchar | Auth sederhana (SYSTEM.md/PRD.md §3 — bukan sistem role kompleks) |
| created_at | timestamp | |
| avatar_url | varchar, nullable | URL gambar avatar — BUKAN file upload, tetap simpel |
| socials | JSONB, nullable, default `[]` | Array `[{platform, url}]` — sengaja JSON column, BUKAN tabel terpisah (terlalu kecil untuk relasi sendiri; lihat LEARN di db.go) |

### `urls` (perubahan dari skema awal)
Tambah 1 kolom baru:

| Kolom | Tipe | Catatan |
|---|---|---|
| creator_id | FK ke `creators`, **nullable** | Nullable supaya short-link "anonim" (dari Fase 0-8, tanpa akun) tetap valid — link berpemilik cuma yang dibuat lewat halaman profil kreator |

> **Catatan N+1 (PRD.md §2):** Query buat load halaman profil (`SELECT * FROM urls WHERE creator_id = ?`) itu 1 query. Tapi kalau nampilin click count per link dengan query TERPISAH per baris hasil, itu jadi N+1. The developer WAJIB rancang sendiri solusinya sebelum minta AI implementasi (opsi: JOIN ke agregat, atau pakai `urls.click_count` yang udah didenormalisasi dari skema awal — cek dulu opsi mana yang lebih make sense).

## Prinsip Umum

- `click_events` itu **append-only**, tidak pernah di-update — cocok buat latihan pola write-heavy yang butuh async processing (Fase 6)
- `urls.click_count` sengaja didenormalisasi sebagai counter cepat — ini contoh trade-off consistency vs speed yang WAJIB the developer pahami sendiri sebelum implementasi (kapan counter ini boleh "sedikit basi", kapan tidak)
- Index wajib di `urls.short_code` (jalur baca paling sering — redirect) dan `click_events.short_code` (buat query analytics per link)
