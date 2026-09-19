# ARCHITECTURE.md — Jejak

## 1. Prinsip Kunci

Arsitektur project ini BERUBAH secara sengaja tiap fase (lihat PRD.md §1) — bukan didesain final di awal seperti project produk biasa. Ini bagian dari poin belajarnya: ngerasain kenapa tiap komponen ditambahkan.

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
| API | Go (`net/http` atau `chi` router — ringan, gak perlu framework berat) | Konsisten sama stack the developer di project-project lain |
| Database | PostgreSQL | Familiar, cukup buat simulasi replica & sharding di skala kecil |
| Cache | Redis | Standar industri buat cache-aside, gampang diukur hit/miss-nya |
| Load Balancer | Nginx (mode paling sederhana dulu: round robin) | Ringan, konfigurasinya kelihatan jelas (bagus buat belajar, dibanding cloud LB yang "ajaib" dari luar) |
| Queue (Fase 6) | Redis List/Streams (BUKAN Kafka/RabbitMQ di awal) | Jangan over-engineer — cukup buat ngerasain konsep async decoupling, gak perlu infra berat |
| Dashboard (Fase 8, opsional) | Next.js (App Router) | Konsisten sama stack the developer |
| Orkestrasi lokal | Docker Compose | Biar bisa jalanin multi-instance API + LB + DB + Redis di 1 laptop tanpa cloud |

## 3. Struktur Folder (Target)

```
jejak/
├── backend/                      # semua kode Go + aset backend
│   ├── cmd/
│   │   ├── server/main.go        # entrypoint API
│   │   ├── worker/main.go        # [Fase 6] proses async click logging
│   │   ├── proxy/main.go         # [Fase 3] load balancer round-robin
│   │   └── loadtest/main.go      # [Fase 1] harness baseline & perbandingan
│   ├── internal/
│   │   ├── handler/              # HTTP handlers (shorten, redirect, dst.)
│   │   ├── shortener/            # logic generate short-code
│   │   ├── cache/                # [Fase 2] cache-aside logic
│   │   ├── ratelimit/            # [Fase 7]
│   │   ├── middleware/           # [Fase 9] auth/session
│   │   ├── auth/                 # [Fase 9] session store
│   │   ├── env/                  # pembaca .env
│   │   └── db/                   # query layer (SingleStore + shardStore)
│   ├── db/
│   │   └── migrations/           # SQL migration per fase (lihat SCHEMA.md)
│   ├── go.mod
│   ├── Dockerfile
│   └── Dockerfile.worker
│
├── frontend/                     # [Fase 8] Next.js dashboard
│
├── infra/
│   ├── docker-compose.yml        # API instances + Nginx LB + Postgres + Redis
│   └── nginx.conf                # [Fase 3]
│
└── PRD.md, ARCHITECTURE.md, SCHEMA.md, RULES.md
```

## 4. Catatan Per Fase (diisi the developer sendiri sebelum implementasi)

> Bagian ini SENGAJA kosong. Sebelum mulai tiap fase, the developer isi di sini: pendekatan yang dipilih + alasannya. AI TIDAK BOLEH mengisi bagian ini (SYSTEM.md §1 poin 1).

- **Fase 2 (Caching):** _(isi sendiri sebelum mulai — strategi apa, TTL berapa, kenapa)_
- **Fase 3 (Load Balancer):** _(isi sendiri — algoritma apa, kenapa)_
- **Fase 4 (Read Replica):** _(isi sendiri — bagaimana app tahu harus baca dari mana)_
- **Fase 5 (Sharding):** _(isi sendiri — strategi hash/partisi apa, kenapa)_
- **Fase 6 (Async Queue):** _(isi sendiri — apa yang terjadi kalau worker down, data hilang atau ditunda?)_

## 5. Mode Native (pengembangan saat ini)

Docker tidak bisa dipakai di environment ini (virtualisasi BIOS mati),
jadi development jalan **native**: PostgreSQL 17 lokal, Redis lokal/Upstash,
2 instance API via `go run` di port beda, worker via `go run ./cmd/worker`
(semua perintah Go dijalankan dari folder `backend/`).
Caranya ada di README.md. `infra/docker-compose.yml` **tidak dihapus** —
biarkan sebagai referensi untuk dipelajari nanti (topologi service yang
diharapkan saat Docker tersedia).

## 6. Simplifikasi Read Replica (Fase 4)

Replikasi Postgres yang sesungguhnya (streaming replication primary →
standby + replication lag) belum dipakai. Sebagai gantinya: **database kedua
(`jejak_replica`) di server PostgreSQL yang sama**, dibuat dengan
`createdb` + migrasi yang sama (lihat README.md §3). Aplikasi sudah
melakukan read/write splitting beneran (tulis → primary, baca → replica
kalau `DATABASE_REPLICA_URL` diisi), tapi **sinkronisasi datanya manual**
(seed/INSERT yang sama di kedua DB). Konsekuensinya: kalau baris baru
belum disalin ke replica, baca dari replica akan 404/stale — itu justru
demo murah dari konsep replication lag yang dipelajari di Fase 4.
