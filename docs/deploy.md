# Jejak — Deploy Prep: Vercel (frontend) + Railway (backend, Postgres, Redis)

Dokumen ini checklist **persiapan** deploy (bukan execute). Terakhir diverifikasi: **2026-10-04** — semua status `[x]` di bawah sudah dijalankan lokal pada commit branch `upgrade/next16-tailwind4`.

## Arsitektur produksi

```
Browser ──► https://jejak.app        (Vercel, Next.js 16 — SSR + proxy /api, /r, /uploads)
                    │ GO_API_URL (server-side only)
                    ▼
           https://api.jejak.app     (Railway, Go cmd/server + cmd/worker)
                    ├── Railway Postgres  (DATABASE_URL, sslmode=require)
                    └── Railway Redis     (REDIS_URL — session, cache, queue)
```

- Domain utama: `jejak.app` (Vercel) · API: `api.jejak.app` (Railway).
- Backend auto-migrate saat boot (`migrate.Run` di `cmd/server/main.go`) — tidak ada langkah migration manual, idempotent (tracking di `schema_migrations`, "sudah ada: dilewati").
- Route Next yang memproxy ke Go: `/api/*`, `/uploads/*`, `/r/{code}`.

---

## TASK 1 — Environment variables checklist

### Backend (Railway) — service 1: `cmd/server`

| Var | Value produksi | Catatan (sumber: code) |
|---|---|---|
| `PORT` | `8080` | Railway umumnya inject sendiri; server baca `env.Get("PORT","8080")` (`cmd/server/main.go:39`). Expose port = sama. |
| `APP_MODE` | `full` | primary + cache + queue async (`main.go:38`). |
| `DATABASE_URL` | `postgres://USER:PASS@HOST:5432/jejak?sslmode=require` | **`sslmode=require` wajib** (Railway Postgres pakai TLS; nilai lokal `disable` hanya utk dev). |
| `DATABASE_REPLICA_URL` | *(kosongkan)* | Tidak ada replica di Railway — code fallback baca ke primary (`main.go:41`). |
| `REDIS_URL` | `redis://default:PASS@HOST:6379` | Dipakai session login, cache, dan queue click. **Wajib** untuk `APP_MODE=full`. |
| `COOKIE_SECURE` | `true` | Set cookie `Secure` di belakang HTTPS (fix security audit). |
| `TRUST_PROXY` | `true` | Railway = reverse proxy → XFF dipercaya utk rate limit per-IP (fix security audit). |

**TIDAK dipakai / jangan di-set di Railway:**

- `GO_API_URL` — hanya dibaca **frontend Next** (next.config.js), bukan backend Go.
- `BACKENDS`, `SHARD_DSNS` — hanya utk binary load-balancer `cmd/proxy` / mode shard; deploy Railway cukup `cmd/server` + `cmd/worker` (1 instance, tanpa LB).
- `NEXT_PUBLIC_BASE_URL` — **tidak ada di codebase**; origin publik dibaca dari `SITE_URL` (frontend).

### Backend (Railway) — service 2: `cmd/worker` (WAJIB utk `APP_MODE=full`)

| Var | Value | Catatan |
|---|---|---|
| `PORT` | *(opsional, worker tak listen HTTP)* | Worker hanya consume queue. |
| `DATABASE_URL` | sama dengan server | **WAJIB** (lihat catatan di `.env.example`: worker gagal tanpa ini). |
| `REDIS_URL` | sama dengan server | Queue click logging async. |

> Tanpa worker: redirect tetap jalan, tetapi click logging (analytics) menumpuk di queue dan tidak pernah diproses → analytics kosong. Pastikan service worker aktif.

### Frontend (Vercel)

| Var | Value | Catatan (sumber: code) |
|---|---|---|
| `GO_API_URL` | `https://api.jejak.app` | Dibaca **hanya di server** (route handler Next) — tidak masuk bundle browser. Set **setelah** backend Railway dapat domain. |
| `SITE_URL` | `https://jejak.app` | `metadataBase` + SEO/OG absolute URL (`app/layout.jsx`). |

Build command Vercel: default (`npm run build` / auto-detect Next). Root directory: `frontend/`.

### Rate limit & konfigurasi lain (bukan env — hardcoded by design)

- Register: **10/menit per IP** · Shorten: **30/menit per IP** (`cmd/server/main.go` wiring `RegisterLimiter`/`ShortenLimiter`). Bukan env — mengubahnya = ubah code (di luar scope prep ini).
- `SMOKE_STRICT=1` (opsional) — smoketest jadi strict mode.

---

## TASK 2 — Secret generation

**Fakta dari code (audit): project ini TIDAK punya signing secret:**

- **Session**: token acak 32-byte `crypto/rand` disimpan di Redis (`internal/auth`) — tidak ditandatangani, tidak ada `SESSION_SECRET`/JWT secret untuk di-generate.
- **API key**: 32-byte `crypto/rand`, disimpan sebagai **SHA-256** hash (`handlers_apikeys.go:27,37`) — hash tanpa salt **by design** (deterministic utk indexed lookup; entropi key sudah 256-bit).
- Kredensial yang jadi "secret" produksi = **password Postgres & Redis** → Railway meng-generate otomatis saat provision service; salin ke `DATABASE_URL`/`REDIS_URL`. Password DB lokal/dev tidak boleh dipakai di produksi.

Kalau suatu saat butuh secret signing (mis. integrasi baru), template:

```bash
# openssl rand -hex 32   → 64 karakter hex, tempel sebagai nilai env, JANGAN commit
```

> Template saja — **jangan taruh nilai real di file ini atau di repo** (aturan repo: `.env` selalu gitignored).

---

## TASK 3 — Checklist pre-deploy

### Code (status aktual — sudah dijalankan 2026-10-04)

- [x] Build Go hijau: `go build ./...` → exit 0
- [x] Test Go hijau: `go test ./...` → semua paket ok (unit) + `go test -race` hijau (sesi audit) + `-tags integration` hijau (Postgres/Redis lokal)
- [x] Build Next.js hijau: `npm run build` → exit 0, 31 routes
- [x] Smoke test 12/12: `BASE_URL=http://localhost:8082 go run ./cmd/smoketest` → **12/12 passed** (310ms)
- [x] Security checklist pass → lihat `SECURITY_CHECKLIST.md` (gosec 0, govulncheck 0, npm audit **0**)
- [x] Semua TODO critical resolved/documented → `SECURITY_CHECKLIST.md` bagian TODO (major Next/Tailwind kini sudah di-upgrade; sisa: `golangci-lint` belum terpasang, coverage `internal/db` unit <70% — tidak blocking deploy)
- [x] npm audit 0 vulnerabilities (setelah upgrade Next 16 + Tailwind 4)

### Database

- [x] Migration terbaru ada di `backend/db/migrations/` (26 file, termasuk `20260930_14..16`)
- [x] Migration idempotent — bisa di-rerun (tracking `schema_migrations` + "sudah ada: dilewati")
- [x] Auto-migrate on boot → Railway tak perlu release phase khusus
- [ ] Backup strategy — **verifikasi di Railway dashboard**: auto-backup mengikuti plan (plan kecil/free umumnya TIDAK otomatis). Fallback wajib: `pg_dump` terjadwal ke storage luar. **TBD sebelum launch.**

### Environment

- [x] Semua env var terdokumentasi di dokumen ini (nama var diverifikasi terhadap code)
- [x] `.env.example` match produksi tanpa value — **diperbarui**: `TRUST_PROXY`, `COOKIE_SECURE` ditambahkan (lihat diff file ini)
- [x] Secret produksi: tidak ada signing secret yang perlu di-generate (Task 2); pakai password DB/Redis dari Railway (bukan dev)
- [ ] `COOKIE_SECURE=true` di Railway → **set saat provision** (belum — deploy belum execute)
- [ ] `TRUST_PROXY=true` di Railway → **set saat provision**

### Domain

- [ ] Domain dibeli (`jejak.app` atau ganti sesuai keputusan)
- [ ] DNS: `jejak.app` → Vercel (CNAME/A), `api.jejak.app` → Railway (CNAME)
- [ ] SSL: otomatis di Vercel & Railway (Let's Encrypt) — verify setelah custom domain terpasang

### Monitoring

- [ ] Error tracking: minimal — Railway/Vercel log stream dulu; **Sentry (JS + Go) = recommended TODO** sebelum launch publik
- [ ] Uptime monitoring: UptimeRobot (atau sejenis) → probe `https://jejak.app` + `https://api.jejak.app/api/u/<username>` (lihat catatan health di bawah)
- [ ] Log aggregation: Railway persistent logs + Vercel function logs (export bila perlu)

### Known limitations (dokumentasikan, non-blocking)

- **Avatar `uploads/` = filesystem ephemeral** di Railway → hilang saat redeploy. Mitigasi sebelum launch: object storage (S3/R2) — TODO terpisah, jangan blocker pertama deploy.
- **Tidak ada endpoint `/health`** di backend (di luar scope "jangan ubah code"). Health check praktis: `GET /api/u/<username>` (public) harus balas 200/404, atau jalankan smoketest penuh. (Opsional nanti: tambah `GET /health` = 1 baris route.)

---

## TASK 4 — Deploy order

### 1. Backend ke Railway

1. Push branch (merge `upgrade/next16-tailwind4` → `master` dulu bila sudah review).
2. Railway → *New Project* → connect repo → service **server** dengan root `backend/`, start command `go run ./cmd/server` (atau build binary `go build -o server ./cmd/server && ./server`).
3. Tambah Postgres & Redis di project yang sama → Railway mengisi `DATABASE_URL` / `REDIS_URL` → samakan `sslmode=require`.
4. Set env: `APP_MODE=full`, `COOKIE_SECURE=true`, `TRUST_PROXY=true`.
5. Service kedua: **worker**, root `backend/`, start `go run ./cmd/worker`, env `DATABASE_URL` + `REDIS_URL`.
6. Generate domain `api.jejak.app` (Settings → Domains).
7. Cek sehat:

```bash
# respons apa pun (200/404) = server hidup & routing jalan:
curl -s -o /dev/null -w "%{http_code}\n" https://api.jejak.app/api/u/ghifari
# verifikasi penuh (12 endpoint, register+login+shorten+redirect+analytics):
BASE_URL=https://api.jejak.app go run ./cmd/smoketest
```

> Jangan lupa: smoketest kena rate limit register 10/menit — tunggu bila baru saja test manual.

### 2. Frontend ke Vercel

1. Vercel → *Import Project* → repo → **Root Directory: `frontend/`** (framework Next.js otomatis terdeteksi).
2. Set env: `GO_API_URL=https://api.jejak.app`, `SITE_URL=https://jejak.app`.
3. Deploy (build Next 16 otomatis, Turbopack).
4. Cek: `curl -s -o /dev/null -w "%{http_code}\n" https://jejak.app` → **200**.

> Urutan penting: **backend dulu** (GO_API_URL harus sudah hidup agar route `/api/*` Next tidak 502).

### 3. Custom domain

1. DNS: tambah record sesuai instruksi Vercel (`jejak.app`) dan Railway (`api.jejak.app`).
2. Vercel → Project → Domains → add `jejak.app`.
3. Railway → service server → Domains → add `api.jejak.app`.
4. Tunggu propagasi + cert otomatis terbit.

### 4. Verify produksi

1. Buka `https://jejak.app` → landing render, console 0 error.
2. Register → login → shorten → copy link → redirect jalan → buka analytics (pastikan worker aktif: angka click naik).
3. Ganti bahasa ID/EN/DE (cookie `NEXT_LOCALE`) ✓
4. Smoke test penuh ke produksi:

```bash
BASE_URL=https://api.jejak.app go run ./cmd/smoketest
```

---

## TASK 5 — Rollback plan

| Skenario | Aksi |
|---|---|
| Backend gagal (crash/5xx) | Railway → service → **Deployments → Rollback** ke versi sebelumnya (otomatis pakai image lama; env tidak berubah). |
| Frontend gagal | Vercel → **Deployments →** pilih versi sehat → **Promote to Production** (instant, tanpa rebuild). |
| Database corrupt / data loss | Railway → Postgres → **Backups → Restore** ke point-in-time (ikut plan; kalau plan tanpa backup → mitigasi `pg_dump` terjadwal, lihat checklist). |
| Kedua-duanya perlu mundur jauh | Git: `git checkout master && git reset --hard <hash-sehat>` lalu redeploy (hash upgrade Next16: `acf53b8` = pre-upgrade, `33387d4` = post-upgrade siap). |
| Env salah (mis. lupa `COOKIE_SECURE`/`TRUST_PROXY`) | Cukup ubah env di Railway/Vercel → redeploy otomatis (rollback code tidak menyelesaikan masalah env). |

---

## Follow-up TODO (sebelum launch publik, tidak blocking deploy pertama)

1. [ ] Beli domain + DNS + verify SSL (checklist Domain).
2. [ ] Backup Postgres terjadwal (atau upgrade plan yang auto-backup).
3. [ ] Sentry (Next + Go) atau minimal alert error log.
4. [ ] UptimeRobot probe 2 URL.
5. [ ] Avatar uploads → object storage (S3/R2) sebelum data user penting.
6. [ ] Opsional: route `GET /health` di Go (1 baris) utk probe standar.
7. [ ] Opsional: `golangci-lint` masuk CI (catatan di SECURITY_CHECKLIST.md).
