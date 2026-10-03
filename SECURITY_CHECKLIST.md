# SECURITY_CHECKLIST.md — Jejak

Status audit keamanan menyeluruh: **2026-10-03** (playbook `SKILLS/skills-go-backend.md` §10, Testing §12).
Setiap klaim memuat bukti `file:baris` terhadap kode pada tanggal tersebut.

Legenda: **OK** = aman sejak awal · **FIXED** = perbaikan pada audit ini · **WARNING** = diterima dengan alasan + TODO · **TODO** = perlu tindak lanjut.

---

## §10.2 Aturan keamanan (13 butir)

| # | Item | Status | Bukti | Keterangan |
|---|------|--------|-------|------------|
| 1 | SQL injection | **OK** | seluruh query pakai placeholder `$1..$n` (`internal/db/db.go`); kolom dinamis `breakdownQuery` kini melewati allowlist 2 literal sebelum dipotong ke query — `db.go:2194-2206` (FIXED: sebelumnya concat tanpa guard) | 0 `fmt.Sprintf` jadi SQL; concat identifier hanya setelah `switch column` menolak nilai lain |
| 2 | Command injection | **OK** | 0 match `exec.Command`/`os/exec` di seluruh `backend/` | Tidak ada permukaan eksekusi proses |
| 3 | XSS | **OK** | React auto-escape, 0 `dangerouslySetInnerHTML`/`innerHTML`/`document.write` di frontend; body 201 shorten = literal `http(s)://` + host tervalidasi server + kode allowlist (`handlers_links.go:226-231`); `X-Forwarded-Proto` hanya nilai `https` literal yang dihormati — nilai bebas tak pernah masuk body (`handlers_links.go:204-216`, FIXED); semua error = `http.Error` (text/plain) | gosec G705 dijawab `#nosec` + justifikasi validasi | 
| 4 | Path traversal | **OK** | nama file avatar = `%d` (int64 sesi) + ekstensi hasil magic-byte allowlist jpg/png/webp (`handlers_profile.go:240-253`); static server = root `uploads` + `StripPrefix` (`cmd/server/main.go:296-306`); slug regex `^[a-zA-Z0-9_-]{3,30}$` menolak `../` (`handlers_links.go:92-94`); 0 `os.Open` dari input selain `.env` internal (`env.go:21`, `#nosec G304` + alasan) | direktori 0750 (FIXED, G301) |
| 5 | SSRF / open redirect | **OK** | `validRemoteURL` mewajibkan `http/https` + host non-kosong → `javascript:`/`file:`/`data:` = 400 (`handler.go:147-153`, teruji `security_test.go`); tujuan redirect diambil dari DB yang tervalidasi saat create (`handlers_links.go:335+`); smoketest = tool dev lokal (by design, `#nosec G704` + alasan, `smoketest/main.go:159-172`) | X-Forwarded-Proto tidak lagi dipercaya buta (FIXED) |
| 6 | Timing attack | **OK** | password: `bcrypt.CompareHashAndPassword` (`auth.go:52`); API key: SHA-256 → lookup indeks DB (`handlers_apikeys.go:27`, `:198-213`) — pencocokan lewat index hash, bukan perbandingan string | 0 `subtle.ConstantTimeCompare` diperlukan |
| 7 | Random lemah | **OK** | token sesi: `crypto/rand` 256-bit (`auth.go:108-115`); kode pendek: `crypto/rand` (`shortener.go`); MD5 **hanya** hash routing shard, bukan kontrol keamanan — `#nosec G401,G501` + alasan (`shortener.go:78-86`) | 0 `math/rand` |
| 8 | Penyimpanan password | **OK** | bcrypt cost default 10 (`auth.go:34-38`); 0 match log berisi password | — |
| 9 | Enkripsi data rahasia | **OK** | password = bcrypt (one-way); API key disimpan sebagai hash SHA-256 (`handlers_apikeys.go:27`); token sesi = random opaque | — |
| 10 | Secrets di repo | **OK** | `.env` di-gitignore, `.env.example` tracked (placeholder dev) — `.gitignore:3-5`; scan hardcoded = hanya fixture test (`jjk_deadbeef…`); DSN di-log sudah `maskDSN` (`migrate-replica/main.go:73`) | — |
| 11 | DoS / resource exhaustion | **FIXED** | `http.Server` kini punya timeout ReadHeader/Read/Write/Idle — `cmd/server/main.go:339-351`, `cmd/proxy/main.go:80-90` (sebelumnya `ListenAndServe` polos, G114); body JSON dibatasi 1 MB `MaxBytesReader` di `decodeJSON` (`handler.go:169`); body avatar dibatasi 3 MB → **413** (`handlers_profile.go:174-186`, FIXED: sebelumnya `ParseMultipartForm` tanpa cap = G120, permintaan multi-GB mengisi disk temp); rate limit: login 5/min per IP+user (`handlers_auth.go:99`), **register 10/min per IP BARU** (`handlers_auth.go:30-45`, `main.go:150-154`), **shorten 30/min per IP BARU** (`handlers_links.go:33-53`, `main.go:155-157`), account 5/min, bulk 3/min, export 5/min, API key 100/min per key | WARNING: limiter in-memory per-instance (2 instans efektif 2×) — trade-off terdokumentasi `ratelimit.go:31-36`; production = Redis INCR |
| 12 | Kebocoran info lewat log/error | **OK** | 0 match log berisi `password`/`token`/`email`/`secret`/`Bearer`; error 500 = pesan generik ("Database error", "Internal error"); `decodeJSON` mengembalikan "Invalid request body" generik (`handler.go:171`); login 401 identik utk user tak dikenal & password salah (anti-enumerasi, `handlers_auth.go:107-113`) | — |
| 13 | Kepercayaan berlebih pada client | **FIXED** | `X-Forwarded-For` **diabaikan secara default** — `ratelimit.go:112-135` (`SetTrustProxy`, env `TRUST_PROXY=false`); sebelumnya header yang bisa dipalsukan mengalahkan semua rate limit per-IP; `X-Forwarded-Proto` kini hanya nilai literal `https` (`handlers_links.go:204-216`); object-level authz = semua mutasi membawa `WHERE creator_id` (teruji integrasi `TestIntegrationOwnerScoping`); Host dipercaya utk membangun shortURL = trade-off terdokumentasi `handlers_links.go:199-203` | WARNING: untuk production gunakan whitelist domain / `BASE_URL` statis (TODO) |

## §10.3 Autentikasi & sesi

| Poin | Status | Bukti |
|------|--------|-------|
| Cookie `HttpOnly` + `SameSite=Lax` + `Path=/` | **OK** | `auth.go:308-325`, teruji `cookie_test.go` |
| Cookie `Secure` | **FIXED** | env `COOKIE_SECURE` → `auth.SecureCookies` (`auth.go:298-306`, `cmd/server/main.go:47-53`); default `false` karena dev plain-http (cookie Secure tak akan kembali); wajib `true` di balik TLS. `#nosec G124` + alasan |
| Token di `localStorage` | **OK** | 0 match; sesi hanya cookie + server-side store |
| CSRF | **OK** | `SameSite=Lax` + semua endpoint tulis ber-JSON (`decodeJSON` menolak form body) + tanpa header CORS (teruji `TestNoCORSHeaders`) |
| Rate limit brute force | **OK** | login 5 gagal/menit per IP+username, cek sebelum bcrypt (`handlers_auth.go:96-116`); register & shorten ikut (lihat butir 11) |
| Re-auth password utk aksi kritikal | **OK** | ganti email/password & hapus akun mewajibkan password saat ini (`handlers_account.go:72-85`) |
| Logout & logout-all | **OK** | hapus sesi server-side + clear cookie (`handlers_auth.go:133-149`); logout-all = `DeleteAllForUser` keep-token (teruji integrasi Redis) |
| Anti-enumerasi login | **OK** | 401 identik utk username tak ada & password salah (`handlers_auth.go:107-113`) |
| API key | **OK** | hash SHA-256 at rest, rate limit per-key 100/menit setelah key valid (`handlers_apikeys.go:222-225`) |
| Object-level authz (IDOR) | **OK** | owner-scoped SQL di semua mutasi; teruji `TestIntegrationOwnerScoping` (update/delete oleh pemilik lain → `ErrNoRows`) |

---

## §10.4 Hasil tools verifikasi

| Tool | Hasil | Catatan |
|------|-------|---------|
| `go vet ./...` | **bersih** | — |
| `gofmt -l .` | **bersih** | — |
| `gosec -exclude=G104 ./...` | **0 issues** | 73 → 0. 10 `#nosec` dengan alasan tertulis di kode (G703/G304 path, G120 cap, G124 cookie env, G501 MD5 routing, G706 log `%q`, G115 modulo-bounded, G705 body tervalidasi, G202 allowlist, G704 tool dev). `G104` (error `rows.Close()`/`body.Close()` dibuang) dikecualikan: konvensi close best-effort, tak memengaruhi keamanan/konsistensi |
| `govulncheck ./...` | **0 vulnerabilities** | stdlib Go → `toolchain go1.26.6` di `go.mod` (sebelumnya 20 vuln @1.25/1.26.0, 16 @1.26.1 — semua fixed) |
| `npm audit` | **6 tersisa (5 high, 1 critical)** — WARNING | `postcss` di-fix via `overrides: 8.5.28` (7 → 6). Sisa **butuh major breaking**: `next@14.2.35` (critical, advisories fixed ≥15.5.x) dan rantai `braces` ≤3.0.3 via `tailwindcss@3` (fix = tailwind 4). `npm audit fix --force` = install next@16 → DITOLAK tanpa task terpisah + review |
| `golangci-lint` | **tidak tersedia** | tidak terpasang di mesin; pengganti = `go vet` + `gosec` (keduanya bersih). TODO: pasang & jalankan sekali di CI |

## §10.5 Correctness pitfalls

- `defer` di loop: **bersih** (semua `defer rows.Close()` di scope fungsi).
- Nil map: **bersih** (semua map di-init `make`/`map[k]v{}`; `var map` hanya target `json.Unmarshal`).
- Nil slice → JSON `null`: **bersih** (0 `var x []db.T` di `db.go`; respons memakai `[]T{}`).
- Bare type assertion: **0 match**.
- Konversi integer: 2 temuan gosec G115 (proxy modulo, shard rune) — keduanya dibuktikan ter-bounded + `#nosec` + alasan.

---

## Test keamanan (unit, package `internal/handler` + `middleware`)

| Test | File | Membuktikan |
|------|------|-------------|
| `TestShortenDangerousSchemes400` | `security_test.go` | `javascript:`/`data:`/`file:`/`vbscript:` → 400, tak tersimpan |
| `TestShortenTraversalSlug400` | `security_test.go` | slug `../etc/passwd` → 400 |
| `TestDecodeJSONOversize400` | `security_test.go` | body JSON >1 MB → 400 (kontrak by-design) |
| `TestDecodeJSONMalformed400` | `security_test.go` | JSON rusak → 400, bukan 500 |
| `TestHandleRegisterRateLimit` | `security_test.go` | percobaan ke-11 → 429 + `Retry-After` |
| `TestHandleShortenRateLimit` | `security_test.go` | percobaan ke-31 → 429 |
| `TestShortenSameSlugIdempotentConflict` | `security_test.go` | create ganda slug sama → 201 lalu 409, tepat 1 baris |
| `TestNoCORSHeaders` | `security_test.go` | 0 header `Access-Control-*` pada respons API |
| `TestHandleLoginRateLimit` (existing) | `handlers_auth_test.go` | gagal ke-6 → 429 |
| `TestHandleUploadAvatar` (diperbarui) | `handlers_profile_test.go` | oversize → **413**, file tak tertulis |
| Auth bypass (anon/store-nil/token palsu) | `middleware_test.go` | 401 di semua jalur |
| Rate limit API key & account (existing) | `handlers_apikeys_test.go`, `handlers_account_test.go` | 429 per key/per IP |

Integration (`go test -tags integration ./...`, Postgres :5432 + Redis :6379, **8 skenario hijau**):
`TestIntegrationCRUDRoundTrip`, `TestIntegrationUniqueUsername` (23505), `TestIntegrationNoRowsContract`,
`TestIntegrationExpiredLinkStored`, `TestIntegrationOwnerScoping` (IDOR), `TestIntegrationBulkConflictAndFlags`,
`TestIntegrationRedisSessionLifecycle` (token tamper → miss), `TestIntegrationRedisDeleteAllForUser`.

## Coverage (2026-10-03, `go test -cover`)

| Package | Unit | +integration | Target |
|---------|------|--------------|--------|
| handler | 69.7% | — | ≥60 ✅ |
| middleware | 100% | — | ≥80 ✅ |
| ratelimit | 97.4% | — | ≥80 ✅ |
| shortener | 94.7% | — | ≥80 ✅ |
| env | 92.5% | — | ≥80 ✅ |
| auth | 46.2% | **87.1%** | ≥80 ✅ (dgn integration) |
| db | 1.9% | **18.1%** | ≥70 ❌ — TODO: unit test query sharding/replica/analytics |
| **total** | 35.0% | — | — |

Race detector: **`go test -race ./...` hijau** (MinGW GCC 16.2 via winget; `GOENV=off` + `CGO_ENABLED=1`).

## Verifikasi akhir (semua hijau)

`go build ./...` · `go vet ./...` · `gofmt -l` kosong · `go test ./...` (7 paket) · `go test -race ./...` ·
`go test -tags integration ./...` · `gosec` 0 (excl. G104) · `govulncheck` 0 · `npm run build` (21 routes) ·
smoketest **12/12** terhadap server build baru (`BASE_URL=http://localhost:8082`).

## TODO / WARNING terbuka

1. **CRITICAL-WARNING:** `next@14.2.35` — 1 advisories critical (RCE Image Optimization AVIF, Windows) + DoS/cache-poisoning; fix ≥ `15.5.24`. Upgrade major Next 14→16 = breaking (App Router, config) → task terpisah + review manusia.
2. **WARNING:** rantai `braces` ≤3.0.3 via `tailwindcss@3` (high, stack-exhaustion DoS saat build) → fix = Tailwind 4 (breaking).
3. **WARNING:** Host header dipercaya utk membangun shortURL (`handlers_links.go:199-203`) → production perlu whitelist domain / `BASE_URL` statis.
4. **WARNING:** rate limiter in-memory per-instance → multi-instance butuh Redis (`ratelimit.go:31-36`).
5. **WARNING:** `COOKIE_SECURE=true` & `TRUST_PROXY=true` **wajib** diset saat deploy di balik TLS + reverse proxy (default keduanya `false`, aman utk dev).
6. **TODO:** pasang `golangci-lint` dan jalankan di CI.
7. **TODO:** unit test `internal/db` (sharding, replica failover, analytics) — coverage 18.1% jauh dari target 70%.
8. **TODO:** pengujian browser (Playwright) — belum ada infra, di-skip sesuai cakupan task.
