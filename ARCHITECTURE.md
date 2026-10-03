# ARCHITECTURE.md: Jejak

## 1. Prinsip Kunci

Arsitektur project ini BERUBAH secara sengaja tiap fase (lihat PRD.md §1): bukan didesain final di awal seperti project produk biasa. Ini bagian dari poin belajarnya: ngerasain kenapa tiap komponen ditambahkan.

```
Fase 0:  [Client] → [1 Server API+DB]

Fase 2:  [Client] → [1 Server API] → [Cache] → [DB]

Fase 3:  [Client] → [Load Balancer] → [API instance 1..N] → [Cache] → [DB]

Fase 4:  [Client] → [LB] → [API instance 1..N] → [Cache] → [DB Primary + Read Replica]

Fase 5:  [Client] → [LB] → [API instance 1..N] → [Cache] → [DB Shard A / Shard B]

Fase 6:  ... + [Queue] → [Worker: tulis click_events async]
```

## 2. Tech Stack

| Layer | Teknologi | Alasan |
|---|---|---|
| API | Go 1.26 + `net/http` standar (ServeMux pola `METHOD /path`; tanpa router eksternal) | Pola rute didukung stdlib sejak Go 1.22: nol dependency routing |
| Database | PostgreSQL | Familiar, cukup buat simulasi replica & sharding di skala kecil |
| Cache | Redis | Standar industri buat cache-aside, gampang diukur hit/miss-nya |
| Load Balancer | Proxy round-robin custom (`backend/cmd/proxy`, baca `BACKENDS`, default port 8090) | Algoritma kelihatan langsung di kode; `infra/nginx.conf` disiapkan sebagai alternatif kalau Docker tersedia |
| Queue (Fase 6) | Redis List/Streams (BUKAN Kafka/RabbitMQ di awal) | Jangan over-engineer: cukup buat ngerasain konsep async decoupling, gak perlu infra berat |
| Dashboard (Fase 8, opsional) | Next.js 14 (App Router) + Tailwind CSS | Konsisten sama stack the developer |
| Orkestrasi lokal | Docker Compose | Referensi topologi (API multi-instance + LB + DB + Redis); development berjalan native, lihat §5 |

## 3. Struktur Folder (Aktual)

```
jejak/
├── backend/                      # semua kode Go + aset backend
│   ├── cmd/
│   │   ├── server/main.go        # entrypoint API (registrasi semua rute)
│   │   ├── worker/main.go        # [Fase 6] proses async click logging
│   │   ├── proxy/main.go         # [Fase 3] load balancer round-robin
│   │   ├── loadtest/main.go      # [Fase 1] harness baseline & perbandingan
│   │   ├── smoketest/main.go     # 12 langkah end-to-end, exit code 0/1
│   │   └── migrate-replica/      # sinkronkan schema database replica
│   ├── internal/
│   │   ├── handler/              # HTTP handlers (shorten, redirect, analytics, ...)
│   │   ├── shortener/            # logic generate short-code
│   │   ├── cache/                # [Fase 2] cache-aside logic
│   │   ├── ratelimit/            # [Fase 7]
│   │   ├── middleware/           # [Fase 9] auth/session + rate limit
│   │   ├── auth/                 # [Fase 9] session store
│   │   ├── env/                  # pembaca .env
│   │   ├── migrate/              # runner migrasi (dipakai server & migrate-replica)
│   │   └── db/                   # query layer: primary / replica / shard
│   ├── db/
│   │   ├── migrations/           # 14 migrasi aktif (nomor 01-16; 10-11 dilewati;
│   │   │                         #   total 26 file SQL termasuk .down) -> SCHEMA.md
│   │   └── seed.sql              # baris uji (short code abc123)
│   ├── go.mod
│   ├── Dockerfile
│   └── Dockerfile.worker
│
├── frontend/                     # [Fase 8] Next.js dashboard
│   ├── app/                      # routes: /, /dashboard, /u/[username], /r, /login, ...
│   ├── components/               # komponen React (termasuk LanguageSwitcher)
│   ├── lib/                      # i18n.js (3 locale), themes.js (11 tema)
│   └── messages/                 # id.json, en.json, de.json
│
├── infra/
│   ├── docker-compose.yml        # API instances + Nginx LB + Postgres + Redis
│   └── nginx.conf                # [Fase 3]
│
└── README.md, ARCHITECTURE.md, SCHEMA.md, PRD.md, RULES.md, DESIGN.md, PROGRESS.md
```

## 4. Catatan Per Fase (diisi the developer sendiri sebelum implementasi)

> Bagian ini SENGAJA kosong. Sebelum mulai tiap fase, the developer isi di sini: pendekatan yang dipilih + alasannya. AI TIDAK BOLEH mengisi bagian ini (SYSTEM.md §1 poin 1).

- **Fase 2 (Caching):** _(isi sendiri sebelum mulai: strategi apa, TTL berapa, kenapa)_
- **Fase 3 (Load Balancer):** _(isi sendiri: algoritma apa, kenapa)_
- **Fase 4 (Read Replica):** _(isi sendiri: bagaimana app tahu harus baca dari mana)_
- **Fase 5 (Sharding):** _(isi sendiri: strategi hash/partisi apa, kenapa)_
- **Fase 6 (Async Queue):** _(isi sendiri: apa yang terjadi kalau worker down, data hilang atau ditunda?)_

## 5. Mode Native (pengembangan saat ini)

Docker tidak bisa dipakai di environment ini (virtualisasi BIOS mati),
jadi development jalan **native**: PostgreSQL 17 lokal, Redis lokal/Upstash,
2 instance API via `go run` di port beda, worker via `go run ./cmd/worker`
(semua perintah Go dijalankan dari folder `backend/`).
Caranya ada di README.md. `infra/docker-compose.yml` **tidak dihapus**:
biarkan sebagai referensi untuk dipelajari nanti (topologi service yang
diharapkan saat Docker tersedia).

## 6. Simplifikasi Read Replica (Fase 4)

Replikasi Postgres yang sesungguhnya (streaming replication primary →
standby + replication lag) belum dipakai. Sebagai gantinya: **database kedua
(`jejak_replica`) di server PostgreSQL yang sama**, dibuat dengan
`createdb` + schema yang disamakan lewat `go run ./cmd/migrate-replica`
(runner migrasi `backend/internal/migrate`, idempoten: lihat README.md §3).
Aplikasi
melakukan read/write splitting beneran: tulis → primary; **baca redirect
(`GetURL`) → replica** kalau `DATABASE_REPLICA_URL` diisi, sedangkan
**login + halaman profil publik (`/api/u/...`) → primary** (read-your-own-writes:
setelan seperti tema harus langsung terlihat setelah disimpan: bukan stale
di replica). **Schema** disinkronkan lewat `migrate-replica` (wajib:
schema replica tertinggal membuat semua redirect mode full 404, karena
query baca gagal di replica), sedangkan **data** tetap disinkronkan
**manual** (seed/INSERT yang sama di kedua DB). Konsekuensinya: kalau baris
baru belum disalin ke replica, redirect di mode full akan 404/stale: itu
justru demo murah dari konsep replication lag yang dipelajari di Fase 4.

## 7. Endpoint API

Registrasi lengkap di `backend/cmd/server/main.go` (rute pakai pola
`METHOD /path` bawaan `net/http`). Ringkasan:

| Kelompok | Endpoint |
|---|---|
| Auth | `POST /api/register`, `POST /api/login`, `POST /api/logout` |
| Akun | `PUT /api/account/email`, `PUT /api/account/password`, `POST /api/account/logout-all`, `DELETE /api/account` |
| Profil | `GET /api/profile`, `PUT /api/profile`, `POST /api/profile/avatar`, `GET /api/u/{username}` (publik) |
| Link | `POST /api/shorten`, `GET /api/links`, `PUT /api/links/{short_code}`, `PUT /api/links/reorder`, `POST /api/links/claim`, `POST /api/links/bulk`, `GET /api/links/qr-bulk.zip` |
| API key | `GET /api/keys`, `POST /api/keys`, `DELETE /api/keys/{id}`, `POST /api/v1/shorten` (publik, 100 req/menit per key) |
| Analitik | `GET /api/analytics/clicks-by-day`, `GET /api/analytics/breakdown` (dimension: `device` atau `referrer`), `GET /api/analytics/timeseries`, `GET /api/analytics/export.csv`, `GET /api/analytics/summary` |
| Redirect | `GET /r/{code}` (302; lewat rewrite Next.js atau langsung ke Go) |
| Lainnya | `GET /uploads/*` (avatar, static file) |

Rate limit in-memory per-IP ada di auth/shorten/bulk/export (lihat
`backend/cmd/server/main.go` bagian limiter).

## 8. Mode Runtime (`APP_MODE`)

| Mode | Komponen | Perilaku |
|---|---|---|
| `baseline` | 1 instance API, 1 database, tanpa Redis | Click logging sinkron, baca selalu primary |
| `full` (default) | Primary + replica + Redis + worker | Tulis ke primary; redirect baca replica (kalau `DATABASE_REPLICA_URL` diisi); event klik antre di list Redis `click_events` lalu di-flush worker |
| `shard` | `SHARD_DSNS` (satu DSN per shard) | Store ter-shard (Fase 5) |

Mode dipilih lewat env `APP_MODE` tanpa mengubah kode; default dan cabang
perilaku ada di `backend/cmd/server/main.go`. Frontend tidak peduli mode:
semua request lewat rewrite ke `GO_API_URL` (default `http://localhost:8081`).
