# Jejak — run manual native (tanpa Docker)

Docker tidak dipakai di environment ini, jadi semua service jalan native.
`infra/docker-compose.yml` tetap ada sebagai **referensi** (lihat ARCHITECTURE.md §5).

Kode Go ada di `backend/` (server, worker, proxy, loadtest, migrasi DB);
dashboard Next.js ada di `frontend/`.

## 1. Prasyarat

- Go 1.25+ (`go version`)
- PostgreSQL 17 native, service jalan. Binary ada di
  `C:\Program Files\PostgreSQL\17\bin` (belum masuk PATH):
  ```powershell
  & "C:\Program Files\PostgreSQL\17\bin\psql.exe" --version
  ```
- Redis: lokal yang listen di `localhost:6379`, atau akun Upstash
  (pakai URL `rediss://...` di `REDIS_URL`). Cek cepat:
  ```powershell
  $c = New-Object Net.Sockets.TcpClient
  $c.BeginConnect("localhost", 6379, $null, $null).AsyncWaitHandle.WaitOne(1500)
  ```

## 2. Config (.env)

```powershell
Copy-Item .env.example .env
notepad .env
```

Isi yang penting (default `.env.example` sudah mengarah ke localhost):

| Key | Baseline | Full |
|---|---|---|
| `APP_MODE` | `baseline` | `full` |
| `DATABASE_URL` | `.../jejak?...` | sama (primary) |
| `DATABASE_REPLICA_URL` | kosongkan | `.../jejak_replica?...` |
| `REDIS_URL` | kosongkan | `redis://localhost:6379` (atau Upstash) |

Kosong = fitur mati: tanpa replica → baca fallback ke primary;
tanpa Redis → tanpa cache dan click logging jadi sinkron.

## 3. Database: primary + simulasi replica

User/password PostgreSQL mengikuti instalasi lokal masing-masing
(ganti `jejak` di bawah bila beda). `psql`/`createdb` pakai path penuh:

```powershell
$pg = "C:\Program Files\PostgreSQL\17\bin"
& "$pg\createdb.exe" -U postgres -h localhost jejak
& "$pg\createdb.exe" -U postgres -h localhost jejak_replica
```

Jalankan migrasi yang **sama** di kedua database:

```powershell
& "$pg\psql.exe" -U postgres -h localhost -d jejak -f backend/db/migrations/20240101_01_initial_schema.sql
& "$pg\psql.exe" -U postgres -h localhost -d jejak_replica -f backend/db/migrations/20240101_01_initial_schema.sql
```

Seed baris uji di kedua database (ini "sync manual" pengganti replikasi asli):

```powershell
& "$pg\psql.exe" -U postgres -h localhost -d jejak -f backend/db/seed.sql
& "$pg\psql.exe" -U postgres -h localhost -d jejak_replica -f backend/db/seed.sql
```

> Setiap URL baru yang dibuat via API hanya tertulis di **primary**.
> Untuk keperluan belajar Fase 4, salin barisnya manual ke replica
> (contoh: `INSERT INTO urls ...` yang sama), atau seed ulang.
> Ini simplifikasi yang disengaja — replikasi Postgres asli
> (streaming replication + lag) didokumentasikan di ARCHITECTURE.md §6.

## 4. Jalankan API (2 instance, port beda)

Tiap perintah di terminal **terpisah**, dari folder `backend/`:

```powershell
# Instance 1
$env:APP_MODE = "full"; $env:PORT = "8081"; go run ./cmd/server

# Instance 2
$env:APP_MODE = "full"; $env:PORT = "8082"; go run ./cmd/server
```

Variasi baseline (1 instance, tanpa cache/replica/queue):

```powershell
$env:APP_MODE = "baseline"; $env:PORT = "8081"; go run ./cmd/server
```

## 5. Jalankan worker (mode full saja)

```powershell
go run ./cmd/worker
```

Worker mengambil event dari Redis list `click_events`. Tanpa worker,
queue menumpuk tapi redirect tetap jalan (fire-and-forget).

## 6. Smoke test manual

```powershell
# Buat short URL (jalur tulis → primary)
curl.exe -X POST localhost:8081/api/shorten -H "Content-Type: application/json" -d "{\"url\":\"https://example.com\"}"

# Redirect seed abc123 (jalur baca → replica di mode full)
curl.exe -I localhost:8081/r/abc123

# Cek click count
curl.exe localhost:8081/r/abc123/clicks
```

Catatan: `abc123` dari seed harus juga ada di replica (lihat §3),
kalau tidak, redirect di mode full akan 404 — itu justru demo
"replication lag"/data belum sync yang sedang dipelajari di Fase 4.
