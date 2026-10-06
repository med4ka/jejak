# Jejak: Progress Log

Catatan setiap perubahan besar di project. Format: YYYY-MM-DD: [Judul Task].  
Append-only log. Jangan hapus entry lama.

**Cara pakai file ini:**

- Entry terbaru selalu di paling atas; urutan di bawahnya mundur ke belakang.
- Tiap entry punya baris **Status** (Done / Sebagian / TODO) tepat di bawah judulnya.
- Untuk kondisi project terkini, baca entry paling atas; entry lama dipertahankan sebagai riwayat.

---

## 2026-10-05: Untrack .md + cleanup node_modules
**Status:** Done (entry ini lokal saja — PROGRESS.md sudah di-untrack, tidak ikut push)
- **Untrack 10 file .md** (`git rm --cached`, file disk utuh diverifikasi byte-size): ARCHITECTURE, DESIGN, PRD, PROGRESS, RULES, SCHEMA, SECURITY_CHECKLIST, docs/deploy, frontend/AGENTS, frontend/CLAUDE. Ter-track tinggal `README.md` ✓. Push `986c910` ke `origin/main` (github.com/med4ka/jejak) sukses.
- **.gitignore** digabung (aturan lama dipertahankan): tambah `*.md` + `!README.md` + `!LICENSE.md`, `node_modules/` + `**/node_modules/`, `.next/out/dist/build/frontend/.next/frontend/out/.turbo`, `backend/bin/`, `.env.local/.env.*.local`, `uploads/`, `npm-debug.log*/dev.log/start.log`, `.vscode/.idea/*.swp/*.swo`, `coverage/.nyc_output`.
- **node_modules:** hanya `frontend/node_modules` yang ada (aktif, dipertahankan); tidak ada duplikat/leftover di root/backend. `npm prune` (hapus 1 flag `"peer"` basi di lockfile) + `npm outdated` info: framer-motion 13.2→14, lucide-react 0.469→1.52 (major, JANGAN update tanpa test), postcss patch.
- **Ukuran:** `.git` 126.8 → 126.9 MB (untrack tak mengecilkan history — wajar, tanpa rewrite). File ter-track terbesar: `glass-bg.jpg` 510KB (asset sah); sisanya source. Tak ada binary/secret/dump.
- **Gates:** `npm run build` exit 0, `go build ./...` OK, smoke **12/12**.

## 2026-10-05: Fix landing text wrap + konsistenkan language pill drawer
**Status:** Done (tanpa screenshot, sesuai instruksi — verifikasi manual oleh user)
- **Sub-headline hero (Option A):** `landing.hero.subheadline` dipecah jadi 2 kalimat, titik ganti titik-dua yang menggantung (render tetap 1 `<p> max-w-xl`, tanpa ubah layout): ID "…bisa dibagikan. Dilengkapi smart redirect per device dan analytics lengkap.", EN "…shareable page. Featuring smart per-device redirect and full analytics.", DE "…teilbaren Seite. Mit intelligenter Weiterleitung pro Gerät und umfassenden Analysen." — hanya dipakai `app/page.jsx`.
- **Pill konsisten:** drawer logged-in kini pill terisolasi blok sendiri (divider my-6 atas + bawah, sama persis logged-out): Link Baru → ── → Dashboard → Notifikasi → ━━ → [pill] → ━━ → Keluar. Dropdown in-flow tak berubah (mendorong konten, anti-overlap).
- Build exit 0, smoke **12/12**.

## 2026-10-05: Drawer polish (language pill + remove Profil) + title simplification
**Status:** Done
- **Bahasa kembali ke pill + dropdown** (`DrawerLanguage` di `NavbarClient.jsx`): pill `border-2 rounded-full px-3 py-1.5` (icon 14px + short + chevron), menu in-flow `rounded-12px border-2 shadow keras` (ID/EN/DE + check aktif kuning). In-flow dipilih ganti absolute+mb-16: menu asli ~136px > 64px sehingga absolute tetap overlap; in-flow mendorong konten (terbukti Masuk 371→507, Keluar 327→463) dan aman di 320px.
- **Hapus Profil** dari drawer logged-in (redundan: tab Profil ada di dashboard); tak ada link `?tab=profil` di repo. Urutan: + Link Baru → Dashboard → Notifikasi+badge → Bahasa → Keluar.
- **Title:** `metadata.title` → `"Jejak"` (3 dict; og:title/twitter ikut — preview WA = "Jejak"), `dashboard/page.jsx` → `"Dashboard"`, `/u/[username]` tetap `"{display} (@{user}): Jejak"` (terverifikasi di RSC payload).
- **Dua bug nyata diperbaiki saat verifikasi:** (1) `SPRING` tak di-import di NavbarClient → klik pill crash se-halaman (`ReferenceError`, lolos build karena Turbopack tak cek identifier); (2) `DrawerLanguage`/`DrawerRow` nested di dalam komponen → remount tiap render reset state `open` (pindah ke module scope). Keduanya hanya terlihat di runtime CDP, bukan build.
- **Verifikasi CDP** (`jejak-drawer-shots/`, 375px via metrics-override karena min-width headless 500px): logged-in (tanpa Profil), logged-out, dropdown-open (menu penuh, Keluar terdorong), 320px (dropdown in-view). Label EN/DE tanpa bocor ID. Build exit 0, smoke **12/12**.

## 2026-10-05: Redesign mobile drawer (logged-in + logged-out)
**Status:** Done
- **Logged-in:** CTA `+ Link Baru` full-width paling atas (mb-4) → divider → item icon 20px+label (Dashboard/LayoutDashboard, Profil/User, Notifikasi/Bell+badge coral kanan, Bahasa/Languages inline) → divider → Keluar coral (LogOut). Row: `gap-3 px-4 py-3 rounded-lg hover:bg-ink/5`, teks ikut tema (fallback ink), tanpa emoji.
- **Notifikasi pindah ke modal terpisah** (`NotificationsModal.jsx`, z-70, max-w-sm, scroll 60vh, aggregate → `/dashboard?filter=broken`): drawer selalu pendek, tak ada panel inline. Feed logic diekstrak ke `lib/useNotifications.js` (dipakai Bell desktop + modal); `NotificationsBell.jsx` disederhanakan jadi icon-only (varian drawer dihapus).
- **Logged-out:** Fitur accordion + Demo/FAQ → divider (my-6) → Bahasa inline expand (ID/EN/DE native + check aktif kuning, bukan dropdown absolut) → divider → Masuk ghost + Daftar Gratis primary full-width stack gap-3.
- **Bahasa:** `LANGUAGE_OPTIONS`/`LANGUAGE_ARIA` di-export dari `LanguageSwitcher.jsx`; key baru `nav.drawer.languageLabel` (Bahasa/Language/Sprache). Header tetap brand+X (tanpa judul panjang). Drawer `overflow-y-auto`; breakpoint `< lg` (768px iPad = drawer).
- **Verifikasi CDP** (`%TEMP%\opencode\cdp-drawer.ps1`, `%TEMP%\opencode\jejak-drawer-shots\`, 5 shot visual-ok): logged-in, logged-in+notif (modal terpisah, drawer tertutup), logged-out, logged-out+lang-expand, 320px (scrollH=clientH=700, CTA 81-129 terlihat). Label EN ("Log out") + DE ("Abmelden") tanpa bocor ID. Catatan harness: headless Chrome min-width 500px → viewport 375/320 via `Emulation.setDeviceMetricsOverride`; selector hamburger via class `lg:hidden` (aria-label DE non-ASCII rusak di PS 5.1).
- **Gates:** `npm run build` exit 0, smoke **12/12**.

## 2026-10-05: Screenshot password form 3 bahasa
**Status:** Done
- Bukti visual Task 1 (form `/r/HUYEjW` via headless Chrome `--accept-lang`, profile terpisah per locale — 1 profile bersama bikin proses hang berebut lock): `%TEMP%\opencode\jejak-pw-shots\pw-form-{id,en,de}.png` (1280×900, ketiganya dicek visual: ID "Link ini dilindungi password / Buka link", EN "This link is password-protected / Open link", DE "Dieser Link ist passwortgeschützt / Link öffnen", styling neo-brutal utuh).

## 2026-10-05: i18n backend (password form + error code)
**Status:** Done
- **Task 1 — password form (server-rendered) diterjemahkan:** paket baru `backend/internal/i18n/` (`DetectLocale` parse Accept-Language + q-weight, `Translate` dot-path + fallback id→key, locale `locales/{id,en,de}.json` via go:embed — hanya 7 key yg dirender server; go:embed tak bisa keluar modul jadi bukan seluruh dict). `writePasswordForm(w, r, code, errKind)` render `<html lang>` + title/h1/p/label/button + error `wrong`/`rate_limited` per locale; `passwordErrorFromQuery` kini kembalikan kind. Test: `TestPasswordFormLocales` + `TestParityWithFrontend` (cocokkan 7 key vs `frontend/messages/*.json` — drift gagalkan build).
- **Task 2 — error code JSON:** paket baru `backend/internal/apierror/` (`WriteError` → `{"code","message"}`, `WriteFieldError` → +`"field"`). **~200 `http.Error` plaintext di 12 file handler + middleware + `main.go:/api/u/` dimigrasi ke 81 code** (katalog di `apierror.go`: METHOD_NOT_ALLOWED/AUTH_REQUIRED/INVALID_REQUEST_BODY/DATABASE_ERROR/…/LINK_NOT_FOUND/…/WHATSAPP_PHONE_INVALID/…). Disengaja tetap plaintext (browser-facing, tanpa konsumen JSON): `GET /r/{code}` "Link disabled"/"URL not found" (410/404).
- **Frontend:** `lib/errors.js` `translateError(err, t, fallback)` (code→`errors.CODE`, fallback message→fallback→`errors.UNKNOWN`), `lib/goError.js` untuk 2 proxy wrapping (`shorten`, `links` GET → `{error, code}`; proxy lain passthrough mentah). **~18 call-site** dimigrasi (AuthModal/ShortenForm/EditLinkModal×2/BulkImport/Dashboard×6/Analytics/ClicksChart/Ringkasan×2/WhatsApp×2/PengaturanTab). Dict: **82 key SCREAMING per locale** (81 code + UNKNOWN) + `errors.analytics.trendLoad/summaryLoad`, 0 drift antar-locale (cek node).
- **Verifikasi:** curl form `/r/HUYEjW` id/en/de ✓ (3 bahasa render benar); curl error → `{"code":"LINK_INVALID_URL"…}`, `AUTH_REQUIRED`, `AUTH_INVALID_CREDENTIALS` ✓; rantai proxy `:3000/api/shorten` → `{"error":"…","code":"LINK_INVALID_URL"}` ✓; node test helper **15/15**; grep ID-hardcode di handler = hanya fallback `message` + halaman HTML statis + log (bukan UI API). `gofmt` bersih, `go vet` OK, `go test ./...` semua ok (test baru: i18n 3, apierror 2, locales, middleware envelope), `gosec` **0** (1 G705 XSS-taint `locale` → di-escape eksplisit), `npm run build` **exit 0** (35/35 static), smoke **12/12**.
- **TODO terbuka:** halaman HTML statis lain (gone/expired/health interstitial+503, `<html lang="id">`) + teks notifikasi tersimpan (`health.go` bell message) belum i18n — butuh threading `r`/locale ke redirect path.

## 2026-10-05: Verify build + setup GitHub remote
**Status:** Sebagian (semua build/test hijau + commit lokal bersih; `remote`+`push` menunggu user — user push sendiri setelah kirim URL)
- **Fix crash blocking:** i18n NotificationsBell memanggil `t("notifications…")` padahal prop `t` dari NavbarClient = theme object → `TypeError: t is not a function`, `GET /` + `GET /dashboard` 500. Fix: `useTranslation()` internal sebagai `tr` untuk strings, prop `t` tetap theme tokens (`NotificationsBell.jsx:19-27`). Commit `04f40a2`.
- **E2E bell lengkap (CDP `%TEMP%\opencode\cdp-bell-click.ps1`, shots 8/9/10):** badge `6` (aria `Notifikasi (6 belum dibaca)`), klik aggregate → `NAV=/dashboard?filter=broken`, chip `Hanya link rusak ✕` muncul, **7 baris broken tampil, 0 healthy** (`HEALTHY-ROW-VISIBLE=False`, `BROKEN-ROWS=7`).
- **Clean build frontend:** dev server dimatikan (`:3000` owner di-kill), `rm .next`, `npm run build` **exit 0, 0 error, 41 routes** (35 static-generated).
- **Backend:** `gofmt` bersih, `go build ./...` + `go vet ./...` OK, `go test ./... -count=1` semua `ok`, smoke `go run ./cmd/smoketest` (BASE_URL=8082) **12/12 passed**.
- **Git:** `git remote -v` kosong; `user.name=Medaka356`; `.gitignore` OK (`.env`, `*.exe`, `*.log`, `/bin/`, `uploads/`); `git ls-files` sensitif: hanya `backend/uploads/avatars/.gitkeep` — **`.env` TIDAK ke-track, aman, tidak ada yang perlu di-rotate**. `gh` CLI tidak tersedia → repo dibuat manual via web sesuai instruksi.
- **TODO (user):** `git remote add origin https://github.com/USERNAME/jejak.git` lalu `git push -u origin master`; verifikasi di GitHub tidak ada `.env`/secret.

## 2026-10-05: i18n fitur baru (health + password + deeplink + notifikasi + WhatsApp tools)
**Status:** Done
- Extract semua string literal Indonesia dari komponen fitur baru ke `messages/{id,en,de}.json` (3 bahasa penuh,+ wire `t()`/`tr()`).
- **File disentuh:** `WhatsAppTool.jsx`, `tools/whatsapp/page.jsx`, `ShortenForm.jsx`, `EditLinkModal.jsx`, `NotificationsBell.jsx`, `DashboardClient.jsx` + 3 dictionary.
- **Key baru (top-level `tools/link/notifications/deeplink`):** `tools.whatsapp.*` (builder: form/preview/result/bio/metadata), `link.health.*` (label sehat/rusak/timeout/belum dicek, badge, check button/result, deeplink aktif/mati, fallback, dashboard badge broken/timeout), `link.password.*` (shorten field, edit modal existing/new/remove, error length/invalid/protocol, dashboard badge "Dilindungi password"), `notifications.*` (title/reload/loading/empty/unread aria/count drawer), `deeplink.badge.detected`, `dashboard.links.filter.brokenOnly` + `dashboard.emptyStates.noBroken`.
- `tools/whatsapp/page.jsx`: `metadata` statis → **`generateMetadata`** (via `getServerTranslation`),mengikuti cookie locale seperti layout.
- `WhatsAppTool`/`EditLinkModal`/`DashboardClient(healthBadge)`: tambah `useTranslation`; module-level `healthLabel`/`healthBadge` terima `t` sebagai argumen.
- **Verifikasi:** `node JSON.parse` 3 dictionary OK; `npm run build` **exit 0** (`.next` 710 file fresh clean build;; scan 6 file target: **0 literal UI Indonesia tersisa** (sisa match cuma komentar/orang enum `"timeout"`/`"broken"` status constants).
- Catatan: entry lama deviasi "(5) Teks UI hardcode" telah usang untuk fitur-fitur di atas — teks kini i18n,+ deviasi "(7) Badge Broken/Timeout literal" ikut teri18n. Sisa TODO (line 22): error Go API translate + metadata per-file (selain tools/whatsapp) tetap terbuka.

## 2026-10-05: Commit + fix notifikasi flooding
**Status:** Done
- Merged branch upgrade/next16-tailwind4 (already merged via fast-forward)
- Commit: 0a8f113 fix: health check notification flood cap 5+1 per creator per batch
- Fix detail: health checker RunBatch now caps individual link_broken notifications at 5 per creator per batch, excess aggregated into single health_aggregate notification with message "Dan X link lainnya bermasalah. Cek dashboard untuk detail."
- Frontend: NotificationsBell.jsx routes health_aggregate clicks to /dashboard?filter=broken; DashboardClient.jsx supports filter query param and shows "Hanya link rusak ✕" chip; badge shows "9+" for counts >9
- Build: go build ./... ok
- Smoke: health unit tests pass
- Sisa TODO: i18n fitur baru (error Go API translate, metadata per-file)

## 2026-10-04: Link health monitor + password protection
**Status:** Done (2 fitur; **migration 17** auto-applied; build **41 routes hijau** (2×), smoke **12/12**, `gofmt`/`go vet`/`go test ./...` semua ok, `gosec` **0** (excl G104), E2E penuh lewat **:8082 dan proxy :3000**, worker batch live **50 link/1m8s**, 7 screenshot terverifikasi)

**Migration `20261004_17_link_health_password.sql` (+ `.down.sql`, 9 statement):** kolom `urls` — `health_status` (DEFAULT 'unknown'), `last_health_check`, `fallback_url`, `health_notified_at`, `password_hash` — + index parsial `idx_urls_health_due` (`WHERE is_active AND health_status <> ''`), `idx_urls_health_broken`; tabel **`notifications`** (creator FK CASCADE, `read`, `created_at DESC` index partisi per creator). Konvensi: `-- +goose Down` hanya di `.down.sql` terpisah; file `.down` ikut terekam sbg migrasi **0-statement** (identik 13 pasang sebelumnya).

**Health monitor (`backend/internal/health/health.go` + test):**
- `CheckURL`: **HEAD** (UA `Jejak-HealthCheck/1.0`, timeout 5s, maks 3 redirect → `ErrUseLastResponse`), 405/501 → retry **GET sekali**; 2xx/3xx `healthy`, 4xx/5xx `broken`, transport error → `timeout`. Body di-drain 4096.
- `Checker.CheckOne`: **policy notifikasi** — `broken && health_notified_at==NULL && CreatorID!=nil` → 1 notif in-app (`link_broken`, pesan + tujuan dipotong 80 char) + marker disimpan; `healthy` → marker di-reset (breakage berikutnya notif lagi); **persist SEBELUM insert notif** (at-most-once); anonim → marker tetap nil (notif menyusul setelah ClaimLinks). Cache redirect **di-evict hanya saat status BERUBAH**.
- `RunBatch`: ambil ≤50 link `ORDER BY last_health_check NULLS FIRST, id`, spacing **1.2s** (budget 50 req/menit), baris gagal di-skip.
- **Worker** `cmd/worker/health.go`: boot **langsung run** + ticker **6 jam**, log `health batch: N link dicek dalam Xs` (bukti live: `50 link dicek dalam 1m8.796s`); lock = worker lock Redis yang sama (single instance).
- **Trigger manual** `POST /api/links/{short_code}/check-health` (auth → **ownership dulu, 404 utk kode orang (anti-SSRF abuse)** → limiter **10/menit/IP** (Record setelah ownership) → `CheckOne` → JSON `{short_code, health_status, last_health_check}`).
- **Gate di redirect** (`handlers_links.go`, urutan: **password → health → logClick → redirect**; kedua gate sebelum click = hit gagal **bukan** click): `broken` saja yg digate — + `fallback_url` → **interstitial 200** (banner "Link ini dialihkan…", meta refresh 1s, link manual); tanpa fallback → **503** + `Retry-After: 21600`. `timeout`/`healthy`/`unknown` tetap 302 normal. Payload cache bertambah `password_hash/health_status(broken only)/fallback_url` (omitempty).

**Password protection (`handlers_password.go` + test):**
- Create/edit: field `password` 4–72 char (400 field error) → **bcrypt cost 10** (`auth.HashPassword`); PUT `nil`=keep, `""`=hapus, value=ganti (tanpa auto-clear; validasi URL fallback via `validRemoteURL`).
- **Gate**: form 200 neo-brutal (tombol "Buka link", `autocomplete=off`) sebelum semua gate lain; gagal → PRG **303 `/r/{code}?e=1`** + pesan inline (tanpa cookie, tanpa click); sukses → cookie **`jejak_link_access_{code}` = hex SHA-256 dari bcrypt hash** (bukan hash-nya: bcrypt base64 mengandung `/` = bukan cookie-octet), `HttpOnly`, `Path=/r/`, `Max-Age=3600`, `SameSite=Lax`, `Secure` = `COOKIE_SECURE || TLS` (`#nosec G124` utk gosec); ganti password → digest berubah → cookie lama otomatis mati.
- **Limiter verify 10/menit/IP+code**: gagal `Record`, sukses `Reset` (typo tak menghukum); lewat limit → 429 + form "Terlalu banyak percobaan…".
- Handler baru: `HandlePasswordVerify`, `VerifyLimiterReset`, `serveHealthGate`, `writePasswordForm/writeHealthFallbackHTML/writeHealthProblemHTML`, `accessDigest/passwordVerified/passwordErrorFromQuery`; route `POST /r/{short_code}/verify` (public).

**Notifications:** `GET /api/notifications` → `{notifications:[50 terbaru], unread}` (owner), `PUT /api/notifications/{id}/read` (owner-scoped, foreign → 404) — di primary/shard0 (catatan di kode: creators selalu primary). Frontend **`NotificationsBell.jsx`**: badge hitung (9+) fetch saat mount, dropdown lazy-load + "Muat ulang", mark-read optimistik (rollback bila gagal), tampil di navbar desktop (sebelum LanguageSwitcher) + mobile drawer (varian `drawer`).

**Frontend lain:** `ShortenForm` — input password opsional (`minLength=4 maxLength=72`, di-drop dari state setelah sukses); `EditLinkModal` — blok **"Kesehatan: {label}" + tombol "Cek sekarang"** (POST check-health, tanpa `onSaved` agar draft tak ter-reset), kontrol password existing ("Kosongkan untuk tetap, isi untuk ganti" + "Hapus password"/"Batal", kirim hanya bila berubah), field **Fallback URL** + helper "Kalau link utama rusak, user akan diarahkan ke URL ini."; `DashboardClient` — **`healthBadge`** (broken=coral "Broken", timeout=kuning "Timeout", healthy/unknown = **tanpa badge** [noise policy]) + **🔒** utk `has_password`, mengalir inline setelah expiryBadge.
**Proxy baru** `app/api/notifications/route.js` (GET), `app/api/notifications/[id]/read/route.js` (PUT), `app/api/links/[short_code]/check-health/route.js` (POST) — semua teruskan cookie; **FIX** `app/r/[code]/route.js`: (a) **query string diteruskan** (`?e=1` sempat dibuang → pesan "Password salah" hilang di :3000), (b) **5xx HTML** (halaman 503 health dari backend) diteruskan apa adanya + `Retry-After` (plain-text 5xx tetap fallback ramah); **BARU** `app/r/[code]/verify/route.js` (POST — tanpa ini unlock password 404 di origin Next).

**Verifikasi E2E:** lewat **:8082** — form 200 → wrong 303 `?e=1` (inline error) → right 303 + Set-Cookie digest (64 hex, HttpOnly, Path=/r/) → 302 destination ✓; PUT `password:""` → 302 tanpa cookie ✓; broken+fallback → interstitial (meta refresh + banner) ✓; fallback dibersihkan → **503 + Retry-After: 21600** ✓; check-health `localhost:9` → `timeout` (redirect normal ✓), self-404 → `broken` ✓; notifikasi `{unread:1}` → mark-read → `0` ✓; rate limit 11× → `200×10, 429` ✓. Lewat **:3000** — form, `?e=1` error, verify 303 + cookie, interstitial, 503 passthrough semuanya ✓. Link gate: `ssppqC` (broken) klik = **0** (tak dihitung), `yhtWJi` (timeout) = 1 ✓. **Worker live**: lock + boot batch 50 link + klik queue ter-drain ✓.

**Screenshot** (`%TEMP%\opencode\jejak-health-shots\`, dimensi diverifikasi): `1-password-form` (1280×900), `2-fallback-interstitial` (1280×900, budget 700ms — meta refresh 1s membuat chrome hang bila budget lebih), `3-503-page`, `4-dashboard-badges` (tab Link Saya: Broken/Timeout/🔒/tanpa-badge utk healthy + bell "58"), `5-bell-open` (feed + timestamp), `6-edit-modal` (Kesehatan: rusak + Cek sekarang + Password baru + Fallback URL terisi), `7-landing-shorten` (1280×1500, field password di hero). Shot login via **CDP** (`--remote-debugging-port` + `Runtime.evaluate`: login fetch → klik tab/bell/Edit).

**Deviasi / TODO:** (1) **Tidak ada infra email/SMS** → notifikasi **in-app saja** (keputusan: sesuai scope; email = TODO bila infra ada). (2) Budget rate manual (10/menit/IP) + worker (50/jam) **tak koordinasi global** — deviasi terdokumentasi, cukup utk deployment single-instance. (3) Batch pertama worker memicu **~58 notifikasi** utk link uji lama (semua first-breakage, kebijakan by design; dashboard jadi demo bagus utk bell). (4) PUT password/fallback **tak evict cache** → jendela 300s status lama (diterima; health-check evict saat status berubah). (5) Teks UI baru **hardcode Bahasa Indonesia** (`messages/*.json` tak disentuh, sesuai batasan task). (6) Smoke test butuh `BASE_URL=http://localhost:8082` (default 8081 = XAMPP). (7) Judul browser/kartu health badge memakai teks literal "Broken"/"Timeout" (bukan i18n). (8) `redis-cli` tak tersedia di mesin (verifikasi queue via perilaku worker + click_count).

## 2026-10-04: Deep link e-commerce + WhatsApp builder
**Status:** Done (2 fitur; **6 platform e-commerce**; build **34 routes hijau**, smoke **12/12**, unit 27/27 test case, E2E interstitial/mobile/desktop/analytics terverifikasi)

**Endpoint baru:**
- `POST /api/deeplink/generate` — `{url}` → `{original_url, platform, deep_link, web_fallback, note}`; non-e-commerce → **404**; `javascript:`/invalid → 400. Tanpa auth + tanpa rate limit (pure parse, nol side effect).
- `POST /api/tools/whatsapp-link` — `{phone, message?, shorten?}` → `{wa_link, short_url?}`; normalisasi nomor (`0812…`/`+62812…`/`62812…`/`812…` → `62812…`, tolak <10/>15 digit & non-digit), pesan maks 500 rune (400 field error), `shorten` default **true**; path shorten memakai **bucket rate limit yang sama** dengan `/api/shorten` (30/menit/IP: tak bisa dilipatgandakan) dan `OptionalAuth` → link milik user login.

**Platform e-commerce didukung (deep link `internal/deeplink`):**
| Platform | Domain | App scheme |
|---|---|---|
| Shopee | shopee.co.id + sg/my/th/vn/com.tw | `shopeeid://product/{shopid}/{itemid}` (modern `/product/a/b` + legacy `-i.a.b`); path lain → `shopeeid://` (app root) |
| Tokopedia | tokopedia.com | `tokopedia://product/…` + root fallback |
| TikTok Shop | tiktok.com | **web-only** (m.tiktok.com sudah universal-link auto-buka app; scheme root justru membuka app HOME = downgrade) |
| Lazada | lazada.co.id / lazada.com | `lazada://` (app root) |
| Blibli | blibli.com | `blibli://` (app root) |
| Bukalapak | bukalapak.com | `bukalapak://` (app root) |

**Redirect `GET /r/{code}` (handlers_links.go):** `respondRedirect` — **mobile** (iPhone/iPad/Android) **+** non-bot **+** scheme non-kosong → **interstitial 200** dgn script app-first: hidden **iframe** coba buka scheme (nav top-frame ke scheme tak terdaftar = Chrome `ERR_UNKNOWN_URL_SCHEME` → fallback tak jalan; iframe gagal = diam), **`visibilitychange`** batal fallback saat app kebuka, **1.5s** → `location.replace(webFallback)`, tombol manual "Buka di web", `noindex` + `json.Marshal` (escape `<`→`\u003c`) utk anti-`</script>`; selain itu **302 biasa**. Desktop/bot/link biasa tak berubah. Click **tetap tercatat sebelum cabang** (verified: `click_events` = mobile+desktop+bot rows, `click_count` naik).

**File baru:** `backend/internal/deeplink/deeplink.go` (+`deeplink_test.go`, 19 deteksi case: suffix-safe host — `shopee.co.id.evil.com` **ditolak**, vs spek `strings.Contains` yg bocor), `backend/internal/handler/handlers_tools.go` (+`handlers_tools_test.go`: NormalizePhone 15 case, WhatsApp 5 test, generate 3 test), `frontend/lib/deeplink.js` (mirror deteksi utk badge; 12/12 node test), `frontend/app/components/WhatsAppTool.jsx`, `frontend/app/tools/whatsapp/page.jsx` (+prefill `?phone=&message=`), `frontend/app/api/tools/whatsapp-link/route.js`, `frontend/app/api/deeplink/generate/route.js` (proxy pola `/api/shorten`).

**File diubah:** `backend/cmd/server/main.go:196` (register 2 route), `backend/internal/handler/handlers_links.go` (import deeplink/html; `respondRedirect` + `writeDeepLinkHTML`; **3 call site** `http.Redirect` → `respondRedirect`), `frontend/app/components/ShortenForm.jsx` (badge 🛍️ "{Platform} terdeteksi — link akan buka app di HP" via `detectEcommerce`, muncul saat URL cocok), `frontend/app/components/EditLinkModal.jsx` (status read-only "Deep link: aktif/mati" — **toggle per-link ditunda**: butuh kolom DB baru = migration, sesuai kebijakan).

**Verifikasi:** E2E: interstitial iPhone → 200 HTML (scheme+fallback+web URL benar), desktop → 302, link biasa mobile → 302, bot mobile → 302 — lewat **:8082 dan proxy :3000** ✓; whatsapp: `08123456789`→`wa.me/628123456789`, pesan `Halo%2C%20saya…` (`%20` bukan `+`), `shorten:false` → tanpa `short_url`, 400 phone/message ✓; generate: 4 platform OK + non-ecom 404 ✓. `go build/vet` 0, gofmt bersih, **gosec 0** (exclude G104), `go test ./...` all ok, `npm run build` **34 routes** (+3), deteksi frontend 12/12, dev.log bersih.

**Screenshot:** `C:\Users\akbar\AppData\Local\Temp\opencode\jejak-wa-shots\1-whatsapp-builder-desktop.png` (+`2-whatsapp-builder-mobile-375.png`, `3-contact-375-REFERENSI-LAMA.png` — potongan kanan shot 375 = artifact Chrome headless min-window-width, **identik utk halaman lama /contact**: bukan regresi).

**TODO / catatan:** (1) Tidak ada platform yg butuh **API key** — semua deep link = public app URL scheme. (2) Tombol "Ubah ke link biasa" (ShortenForm) & toggle aktif/mati (EditLinkModal) ditunda: perlu kolom `deep_link_enabled` baru → migration file + flag review dulu. (3) Teks UI baru hardcode Bahasa Indonesia (file `messages/*.json` tak disentuh, sesuai batasan task). (4) Format legacy product Tokopedia dipakai pola `-i.{a}.{b}` (sama Shopee); kalau meleset → fallback web tetap aman.

## 2026-10-04: Auto-detect brand icon di link
**Status:** Done (file baru `frontend/lib/brands.js` — **60 brand / 35 ikon lucide**; kartu link di /u/[username] kini punya icon box 32/40px dgn warna brand; build **31 routes hijau**, smoke **12/12**, deteksi 14/14 test case, 11 tema lolos loop verifikasi, 4 screenshot bukti)

**File dibuat/diubah:**
- **`frontend/lib/brands.js` (baru)** — registry brand + renderer helpers:
  - `BRANDS` (**60 domain**): e-commerce ID (Shopee×5 domain, Tokopedia, Lazada, Bukalapak, Blibli, Amazon×2, Etsy), social (IG, TikTok, Twitter/X, Facebook, Threads, LinkedIn, Pinterest, Reddit, Snapchat, Dribbble, Behance), video/music (YouTube×2, Vimeo, Spotify, SoundCloud, Twitch, Netflix), messaging (WhatsApp×2, Telegram×2, LINE, Discord×2, Slack, Zoom), dev (GitHub, GitLab, StackOverflow, DEV, Figma, Steam×2), blog/docs (Medium, Substack, Notion, Google Docs/Drive, Dropbox, Google, Apple, Quora, Canva), link-in-bio (Linktree, Beacons, Bio.link). 20 brand ditambahkan di luar daftar awal prompt (Shopee SG/MY/TH, Blibli, Amazon, Etsy, Dribbble, Behance, Netflix, Slack, Zoom, DEV, Figma, Steam, Dropbox, Google, Apple, Quora, Canva, dll).
  - `detectBrand(url)`: exact → root 2-label (`mail.google.com`→`google.com`) → fallback `{name:"Link", color:"#1C1A12", icon:"link"}`; URL rusak → fallback.
  - `luminance/isBrightColor/isDarkColor/rgba` + `ICON_MAP` (35 component; diverifikasi ada di lucide-react ^0.469 — termasuk `AtSign` utk Threads yang terlewat di spek).
- **`frontend/app/components/ProfileLinks.jsx`** — kartu link kini flex row **gap-3**: icon box (`w-8 h-8 md:w-10 md:w-10`, `rounded-[8px]`, border) + kolom konten (URL row + slug row **spacing lama dipertahankan**: pt-4/pb-3, truncate/badge/count identik). Rendering warna 3 kasus (inline style):
  1. **normal**: `bg = brand 20%`, `border = brand 55%`, ikon = brand color;
  2. **terang** (lum>0.65, mis Snapchat #FFFC00): bg **solid** + ikon gelap `#1C1A12`;
  3. **gelap** (lum<0.10, mis X/GitHub #000000): bg solid + ikon **putih** + rim `rgba(255,255,255,.25)` → box tetap terpisah di darkroom;
  - **fallback tak dikenal**: token `ts.accent` tema (bg solid konsisten dgn badge Unggulan — token hex accent tak tersedia utk 4 tema lama, jadi `/20` mustahil tanpa menyentuh themes.js yang dilarang).
  - Hover icon box: `whileHover scale 1.05` framer spring (stiffness 400, damping 17); `aria-hidden`; svg `h-4 w-4 md:h-5 md:w-5`.

**Verifikasi:**
- **Deteksi**: 14/14 test case node (exact/subdomain/root/fallback/invalid URL) ✓; semua `icon` terpetakan (validasi runtime: 0 missing).
- **Render server**: `GET /u/next16test8681` → 8/8 icon box + svg; inline style benar: Shopee `rgba(238,77,45,.2)/#EE4D2D`, IG `#E1306C`, YouTube `#FF0000`, WhatsApp `#25D366`; GitHub (gelap) → `#181717` solid + `#FFFFFF`; unknown-domain → `bg-flash-yellow text-ink` (accent classic) ✓.
- **11 tema**: loop PUT theme → GET → assert (fallback accent berubah per tema + tint brand konstan) = **11/11 OK**; theme user dikembalikan ke classic.
- **Screenshot** (Chrome headless, `%TEMP%\opencode\jejak-brand-shots\`): `1-desktop-classic-brand.png`, `2-mobile-375-classic.png`, `3-desktop-darkroom.png` (case gelap: GitHub solid+putih, fallback accent oranye), `4-desktop-sunset.png`. Visual diperiksa: layout rapi, kontras OK. Catatan: potongan tepi kanan pada shot mobile **identik dgn layout lama** (dibuktikan via stash → screenshot `5-mobile-LAYOUT-LAMA.png`) = artifact environment, **bukan regresi**.
- **Build**: `npm run build` **0 (31 routes)** ✓ · **Smoke 12/12 (exit 0)** ✓ · dev.log **0 error** ✓ · i18n/backend/11 tema tak tersentuh.

**Screenshot path** (belum di-commit, binary): `C:\Users\akbar\AppData\Local\Temp\opencode\jejak-brand-shots\{1-desktop-classic-brand,2-mobile-375-classic,3-desktop-darkroom,4-desktop-sunset}.png`.

## 2026-10-04: Fix avatar upload persist bug (stale HTTP cache)
**Status:** Done (root cause = **H5 cache statis `/uploads`**, bukan backend/DB — API terbukti persist sejak awal; fix = URL avatar versioned per upload + `Cache-Control: no-cache` + error 413/400 teri18n; build 31 routes hijau, smoke **12/12**, gosec 0 excl G104, vet/gofmt/test semua bersih)

**Diagnosis (5 hipotesis, H5 menang):**
- Repro E2E API penuh (login → POST avatar → PUT profile → GET → sesi baru): **semua 200 dan persist** → H1 (form kirim avatar lama), H3 (backend tak update DB), H4 (replica stale) **gugur** — `HandleUploadAvatar` update DB via `UpdateCreatorProfile` (`handlers_profile.go:288`) dan GET profile baca **PRIMARY** (`handlers_profile.go:33`).
- **Root cause H5**: nama file **deterministik** `/uploads/avatars/{id}.{ext}` — re-upload menimpa file dengan URL **identik**; response `GET /uploads` hanya punya `Last-Modified` **tanpa `Cache-Control`** → browser pakai **heuristic freshness** (RFC 7234 §4.2.2: 10% umur file) → menampilkan **gambar lama dari cache** tanpa revalidate, baik sesudah save maupun refresh. Segala API 200 + preview blob berubah, tapi `<img src>` lama → "avatar balik ke yang lama".

**Fix:**
- **`backend/internal/handler/handlers_profile.go`** — `newAvatarPaths` (`:357`, dipakai `:240`): nama kini **`{id}-{unixms}.{ext}`** → setiap upload dapat URL BARU → cache miss pasti di semua titik render (dashboard, navbar, halaman publik, OG image) tanpa query-string di DB. `pruneOldAvatars` (`:366`, dipanggil `:299` sesudah update DB sukses): hapus best-effort file versi lama + nama legacy `{id}.{ext}` milik creator yang sama (orphan tak merusak apa pun).
- **`backend/cmd/server/main.go:324`** — wrapper `cacheRevalidate` (`:371`) set `Cache-Control: no-cache` pada `/uploads/`: setiap reuse wajib revalidate `If-Modified-Since` (FileServer jawab 304 bila tak berubah) — jaring pengaman bila ada URL yang dipakai ulang.
- **`frontend/app/dashboard/DashboardClient.jsx:615`** — upload error di-map **sebelum** parse JSON: **413 → `avatarTooLarge`**, **400 "Only JPG…" → `avatarBadFormat`** (key baru di `messages/{id,en,de}.json:480`), selain itu teks upstream; guard `if (!upData.avatar_url) throw` mencegah PUT terkirim tanpa field (yang akan **mengosongkan** avatar — dibuktikan saat E2E).
- State/touchpoint lain sudah benar dari awal: `setAvatarUrl/Preview/File` + broadcast `jejak:profile-updated` utk navbar (`DashboardClient.jsx:650-674`), toast `savedNotice`.

**Verifikasi (semua via :3000 proxy → :8082):**
- Upload → URL versi baru → PUT 200 → **GET refresh & sesi baru persist** ✓; re-upload **file identik** tetap menghasilkan URL baru ✓; file lama terprune, hanya 1 file/creator di disk ✓; DB `creators.avatar_url` = path versi baru ✓.
- `GET /uploads/...` → **`cache-control: no-cache`** + `content-type: image/png` ✓ (langsung & via rewrite Next).
- Edge: body 4MB → **413** · file `.txt` → **400** "Only JPG, PNG, and WebP" · tanpa field → 400 · tanpa session → 401 · avatar DB tak terganggu oleh request gagal ✓.
- `go build` 0 · `go vet` bersih · `gofmt -l` kosong · `gosec -exclude=G104` **0** · `go test ./...` **all ok** (termasuk subtest baru "re-upload gets a fresh URL and prunes the old file") · `npm run build` **0 (31 routes)** · smoke **12/12 (exit 0)** · halaman `/`, `/dashboard`, `/u/medaka_` = 200, dev.log tanpa error.

**Catatan:** file avatar di DB bisa berupa nama **legacy** (`/uploads/avatars/3.jpg`) — tetap dilayani FileServer; otomatis terganti saat user berikutnya upload. Masih TODO lama: avatar → S3/R2 (filesystem ephemeral saat deploy).

## 2026-10-04: Deploy prep (docs + checklist)
**Status:** Done (`docs/deploy.md` baru — env checklist Vercel/Railway, secret audit, pre-deploy checklist terisi status aktual, urutan deploy, rollback plan; `.env.example` diperbarui; **deploy belum di-execute**)

**File dibuat/diubah:**
- **`docs/deploy.md` (baru)** — 5 task dalam 1 dokumen:
  - **Env checklist**: backend Railway (`PORT`, `APP_MODE=full`, `DATABASE_URL` dgn `sslmode=require`, `DATABASE_REPLICA_URL` kosong, `REDIS_URL`, `COOKIE_SECURE=true`, `TRUST_PROXY=true`) + **service `cmd/worker` wajib** utk APP_MODE=full (tanpa worker click logging menumpuk di queue); frontend Vercel (`GO_API_URL=https://api.jejak.app` server-side only, `SITE_URL=https://jejak.app`). Koreksi penting terhadap prompt: `GO_API_URL` TIDAK dipakai backend Go; `NEXT_PUBLIC_BASE_URL` tidak ada di codebase (yang dipakai `SITE_URL`); `BACKENDS`/`SHARD_DSNS` tidak perlu (tanpa proxy LB).
  - **Secret audit**: project **tidak punya signing secret** — session = token `crypto/rand` 32-byte di Redis, API key = SHA-256 dari `crypto/rand` (salt tidak ada by design, indexed lookup). Secret produksi = password Postgres/Redis dari Railway (auto-generate). Template `openssl rand -hex 32` dicatat tanpa nilai real.
  - **Checklist pre-deploy terisi status aktual** (di-run ulang 2026-10-04): `go build` 0 ✓, `go test` all ok ✓, `npm run build` 0 (31 routes) ✓, smoke **12/12** (310ms) ✓, SECURITY_CHECKLIST pass, npm audit **0** ✓. Belum: backup Postgres (plan-dependent), domain/DNS, SSL, monitoring.
  - **Deploy order**: backend dulu (server + worker + Postgres + Redis + domain `api.jejak.app`) → verify (`curl /api/u/<user>` + smoketest ke prod) → frontend Vercel (root `frontend/`, env 2 var) → custom domain → verify UX penuh. Health check: **tidak ada endpoint `/health`** (di luar scope "jangan ubah code") → pakai `GET /api/u/<username>` atau smoketest penuh.
  - **Rollback plan**: Railway Deployments rollback, Vercel promote versi lama, Postgres restore, hash git (`acf53b8` pre-upgrade / `33387d4` post-upgrade), env fix via redeploy.
- **`.env.example`**: tambah `COOKIE_SECURE=false` + `TRUST_PROXY=false` (dengan komentar kapan boleh true) — sebelumnya tak terdokumentasi.

**Temuan penting utk deploy:**
- **`cmd/worker` = service wajib kedua** di Railway bila `APP_MODE=full` + Redis (queue async; tanpa worker analytics kosong).
- **Avatar `uploads/` = filesystem ephemeral** Railway → hilang saat redeploy → TODO object storage (S3/R2) sebelum launch publik.
- Auto-migrate on boot (`migrate.Run`) → tanpa release phase manual; idempotent.
- Rate limit register/shorten hardcoded (10 & 30 per menit per IP) — bukan env.

**TODO follow-up (sebelum launch publik, list lengkap di docs/deploy.md):** beli domain + DNS · backup Postgres terjadwal (plan free umumnya tanpa auto-backup — verifikasi di dashboard) · Sentry/uptime monitor · avatar → S3/R2 · opsional `GET /health` (butuh ubah code) · `golangci-lint` di CI.

## 2026-10-04: Upgrade Next.js 14→16 + Tailwind 3→4
**Status:** Done (branch `upgrade/next16-tailwind4`; build hijau 31 routes, smoketest Go 12/12, i18n ID/EN/DE terverifikasi, auth flow via Next API 201/200/200, dev console 0 error, npm audit **0 vulnerabilities**; rollback point commit `acf53b87aa6431781eefdd9b5b3d2f4d64e453e6`)

**Versi:**
| Paket | Sebelum | Sesudah |
|---|---|---|
| next | 14.2.35 (critical RCE advisories) | **16.3.8** (Turbopack default) |
| react / react-dom | 18.3.1 | **19.2** (React canary via Next 16) |
| tailwindcss | 3.4.19 | **4.3.3** (`@tailwindcss/postcss` 4.3.3, autoprefixer dihapus — sudah dibundel) |
| npm audit | 6 (1 critical next + 5 high rantai tailwind 3) | **0 vulnerabilities** |

**Codemod:** `npx @next/codemod@canary next-async-request-api .` — **6 file ok, 0 errors**: `app/layout.jsx` (RootLayout + generateMetadata → `await cookies()`), `lib/serverI18n.js` (diberi error marker → fix manual async), `app/u/[username]/page.jsx` (page + generateMetadata → `await props.params`), `app/r/[code]/route.js` (GET/HEAD), `app/api/keys/[id]/route.js`, `app/api/links/[short_code]/route.js` (→ `await props.params`).

**Breaking changes Next 15/16 yang di-handle:**
- **Async Request APIs** (sync access dihapus total di 16): semua `cookies()` → `await cookies()`; semua `params` page/route handler/opengraph-image → `await`. Fix manual di luar codemod: `lib/serverI18n.js` → `export async function getServerTranslation()` + 4 pemanggil (`privacy`, `terms`, `contact` page + `not-found.jsx`) → komponen async + `await`; `app/u/[username]/opengraph-image.jsx` (terlewat codemod) → `await props.params`.
- **Turbopack default** (`next build`/`dev`): `next.config.js` ditambah `turbopack: { root: __dirname }` (menghilangkan warning lockfile di luar repo; tanpa webpack custom → aman).
- React 19: tidak ada pemakaian `useFormState`/`defaultProps`/`ReactDOM.render` (terverifikasi grep → 0).

**Breaking changes Tailwind 3→4 yang di-handle:**
- `globals.css`: `@tailwind base/components/utilities` → `@import "tailwindcss"` + **`@config "../tailwind.config.js"`** (jalur yang disarankan guide utk project custom: config content `./lib` + theme.extend Instant Print tetap jadi single source of truth → risiko visual minimal). Preflight compat: `cursor: pointer` utk `<button>` (v4 default-nya `cursor: default`).
- `postcss.config.js`: plugin `tailwindcss`+`autoprefixer` → **`@tailwindcss/postcss`**.
- Rename kelas sesuai scale v4 (v3 look dipertahankan): **17×** `focus:outline-none` → `focus:outline-hidden` (11 file), **4×** `rounded-sm` → `rounded-xs` (DashboardClient), **2×** `backdrop-blur-sm` → `backdrop-blur-xs` (ConfirmModal, ShareModal). Tidak ada: `@apply`, `bg-gradient-to-*`, `shadow-sm`, bare `ring`, `bg-opacity-*` (0 match — tak perlu diubah).
- Default border color v4 (`currentColor`): aman — semua `border-2` di project selalu punya color token (`border-ink` / arbitrary `border-[#hex]` dari `lib/themes.js`); bare `border` = 0 match.

**File diubah (26 + 2 generated):** codemod 6 (di atas) · fix manual: `opengraph-image.jsx`, `privacy/page.jsx`, `terms/page.jsx`, `contact/page.jsx`, `not-found.jsx` · rename kelas: `page.jsx` (landing), `dashboard/{DashboardClient,PengaturanTab,AnalyticsTab}.jsx`, `components/{AuthModal,BulkImportModal,ConfirmModal,EditLinkModal,NavbarClient,ShareModal,ShortenForm}.jsx` · konfigurasi: `package.json` (+lock), `postcss.config.js`, `globals.css`, `next.config.js` · generated Next 16: `AGENTS.md`, `CLAUDE.md` (agent rules — Next menulis ulang saat `next dev`, di-commit agar tree clean).

**Verifikasi:**
- `npm run build` hijau — Next.js 16.3.8 (Turbopack), 31 routes, 31/31 static pages.
- Halaman via dev: `/` 200 (62.8 KB), `/app` `/dashboard` `/terms` `/privacy` `/contact` 200, `/u/{ghifari,medaka,medaka_,web0b07b6}` 200 (profil render + title benar), `/r/itb3290279571800` **302 → https://example.com/3**, custom 404 → status 404.
- i18n cookie `NEXT_LOCALE`: landing `<title>` id/en/de berbeda ✓; `/terms` h1: "Syarat & Ketentuan" / "Terms & Conditions" / "Nutzungs- & Geschäftsbedingungen" ✓.
- Auth via Next API: `/api/register` 201 (perlu field `display_name`), `/api/login` 200 + cookie, `/api/profile` (auth) 200.
- **Go smoketest `BASE_URL=localhost:8082` → 12/12 passed** (292ms).
- dev console: **0 error/warning** (log scan semua request 200/302/404/400-validasi).
- **No-visual-change evidence** (tanpa browser): CSS v3 (backup `.next`) vs v4 — **34 vs 34 rules `data-profile-theme` (identik)** + semua token utilities kunci (`bg-print-white`, `text-ink`, `font-display`, `border-ink`, arbitrary `shadow-[4px...]`, dll) ada di keduanya; 11 tema rules lengkap di CSS output.
- `npm audit` → **0 vulnerabilities** (target keamanan upgrade tercapai; critical next@14 hilang + rantai tailwind 3 ikut terhapus).

**Catatan operasional:** dev backend Go = `PORT=8082` (8080/8081 dipakai proses lain); Next dev harus dijalankan dengan `GO_API_URL=http://localhost:8082` (tanpa `.env` frontend, default-nya 8081 = XAMPP).

**TODO:**
- Cek visual manual di browser (landing/dashboard/theme picker 11 tema) — bukti otomatis CSS identik sudah kuat, tapi screenshot tak memungkinkan dari CLI.
- Tailwind 4: pertimbangkan migrasi `tailwind.config.js` → `@theme` CSS-native di task terpisah (kini lewat `@config`, didukung resmi tapi deprecated path jangka panjang).
- Tailwind 4 butuh browser modern (Chrome 111+/Safari 16.4+/Firefox 128+) — sesuaikan target browser bila ada user lama.

## 2026-10-03: Security audit + comprehensive testing
**Status:** Done (audit 13 butir §10.2 + §10.3 dengan bukti file:baris, 9 fix keamanan diterapkan, 8 test keamanan baru, 4 test coverage utilitas, 8 skenario integration hijau, race hijau, smoke 12/12; laporan lengkap di `SECURITY_CHECKLIST.md`)

**Fix keamanan (semua diverifikasi ulang dengan tool):**
- **Server timeouts (G114)**: `http.Server` dengan ReadHeader/Read/Write/Idle timeout di `cmd/server/main.go` dan `cmd/proxy/main.go` (sebelumnya `ListenAndServe` polos — koneksi lambat menahan goroutine selamanya).
- **X-Forwarded-For tidak dipercaya lagi**: `ratelimit.ClientIP` mengabaikan XFF secara default; opt-in `TRUST_PROXY=true` (`SetTrustProxy`) hanya utk di balik reverse proxy sungguhan. Sebelumnya header yang bisa dipalsukan mengalahkan semua rate limit per-IP.
- **Rate limit baru**: register 10/menit per IP (`RegisterLimiter`, cek sebelum bcrypt) dan shorten 30/menit per IP (`ShortenLimiter`) — sebelumnya keduanya tanpa throttle (vektor spam-akun & spam-link; bcrypt per percobaan = vektor CPU).
- **Cookie `Secure`**: env `COOKIE_SECURE` → `auth.SecureCookies` di SetCookie/ClearCookie (default false utk dev http; wajib true di balik TLS).
- **Avatar body cap (G120)**: `http.MaxBytesReader` 3 MB sebelum `ParseMultipartForm` → oversize kini **413** (sebelumnya parser tanpa batas bisa mengisi disk temp); direktori upload 0755 → 0750 (G301).
- **Kolom SQL dinamis di-allowlist**: `breakdownQuery` menolak semua kecuali `ce.device_type`/`ce.referrer_type` sebelum concat (G202).
- **X-Forwarded-Proto tidak lagi dipercaya buta**: hanya nilai literal `https` yang meng-upgrade scheme — nilai header tak pernah masuk body respons 201 (menutup gadget injeksi reflected, G705).
- **Log-injection di proxy**: path di-log dengan `%q` (G706).
- **`toolchain go1.26.6`** di go.mod → `govulncheck` turun dari 20 vuln stdlib menjadi **0**.
- **npm `overrides.postcss: ^8.5.28`** → audit 7 → 6 (postcss fixed; sisa butuh major Next/Tailwind, dicatat sebagai WARNING).

**Tool verification (setelah fix):** `go vet` bersih · `gofmt -l` kosong · `gosec -exclude=G104` **0 issues** (73 → 0; 10 `#nosec` dengan alasan tertulis) · `govulncheck` **0** · `npm audit` 6 tersisa (major breaking) · build Go + `npm run build` sukses.

**Test baru:**
- `internal/handler/security_test.go` (8 test): scheme berbahaya (`javascript:`/`file:`/`data:`/`vbscript:`) → 400, slug traversal → 400, JSON oversize → 400 (kontrak by-design), JSON rusak → 400, register ke-11 → 429, shorten ke-31 → 429, create ganda slug → 201/409 tepat 1 baris (idempotent conflict), 0 header CORS.
- Coverage utilitas: `ratelimit_test.go` (97.4%, termasuk bukti XFF default diabaikan), `env_test.go` (92.5%), `cookie_test.go` di auth (Set/Clear/TokenFromRequest), `shard_test.go` di shortener (94.7%).
- Integration (`//go:build integration`, Postgres + Redis asli): 6 skenario DB (CRUD round trip, unique 23505, kontrak ErrNoRows, expiry "expired" terbaca, owner-scoping/IDOR, bulk ON CONFLICT + featured + expiry + API key + ErrEmailTaken) + 2 skenario Redis session (token tamper → miss, logout-all keep-token). **Semua hijau.**
- Test avatar diperbarui: oversize ekspektasi 400 → **413** (kontrak baru yang disengaja).

**Coverage unit:** handler 69.7% · middleware 100% · ratelimit 97.4% · shortener 94.7% · env 92.5% · auth 46.2% (87.1% dgn integration) · db 1.9% (18.1% dgn integration) · total 35.0%. **Race: `go test -race ./...` hijau** (MinGW GCC 16.2 dipasang via winget karena mesin tanpa gcc). **Smoke: 12/12** terhadap server build baru.

**File diubah/ditambah:** `SECURITY_CHECKLIST.md` (baru), `cmd/server/main.go`, `cmd/proxy/main.go`, `cmd/migrate-replica/main.go`, `cmd/smoketest/main.go`, `internal/ratelimit/ratelimit.go` (+test baru), `internal/auth/auth.go` (+`cookie_test.go`, +`redis_integration_test.go`), `internal/handler/{handler.go,handlers_auth.go,handlers_links.go,handlers_profile.go}` (+`security_test.go`), `internal/db/db.go` (+`integration_test.go`), `internal/shortener/shortener.go` (+`shard_test.go`), `internal/env/env_test.go` (baru), `go.mod` (toolchain), `frontend/package.json` (overrides), `handlers_profile_test.go` (413).

**WARNING/TODO terbuka** (rinci di `SECURITY_CHECKLIST.md`): upgrade Next 14→16 (critical advisories) & Tailwind 3→4 (rantai braces) = breaking + review manusia; Host header trust utk shortURL → whitelist domain; limiter in-memory → Redis utk multi-instance; `COOKIE_SECURE`/`TRUST_PROXY` wajib `true` di produksi; pasang `golangci-lint`; coverage `internal/db` masih 18.1% vs target 70%.

## 2026-10-03: Code comment polish: godoc + error contracts
**Status:** Done (86 edit komentar di 9 file Go; 0 perubahan behavior, semua test hijau)

**Apa yang dikerjakan (ikhtisar, playbook `SKILLS/skills-go-backend.md`):**
- **13 gap godoc** exported identifier diperbaiki (`proxy.ServeHTTP`, `smoketest.Error`, `migrations.FS`, `ratelimit.MaxAttempts/Window`, 8 method `db.go`) supaya godoc selalu diawali nama identifier.
- **4 artefak sisa sweep dash** (komentar `//` yang diawali karakter dash) dikonversi jadi kalimat `"..."` natural (`auth.go`, `handlers_analytics.go` x2, `frontend/app/r/[code]/route.js`).
- **68 edit kontrak error** menambahkan kapan error terjadi + side effect di godoc: `ErrNoRows` (unknown/not-owned -> 404), `ErrConnDone` (shard hilang), `ErrEmailTaken`, rollback `defer tx.Rollback`, "0-row UPDATE bukan error", "missing shard di-skip (angka mengecil, bukan gagal)", "empty slice/map bukan error", best-effort `TouchAPIKeyLastUsed`, constructor (DSN invalid -> error; replica unreachable -> fallback + warning; tidak ping di constructor). Rumah kontrak utama: interface `ShardStore`, `Store` (auth), `Cache` (cache) - method tanpa doc kini berkontrak di sana.
- **1 hyperbole** di code diganti netral: `worker/main.go` "robust message ack" -> "explicit per-message ack".

**Verifikasi (semua dijalankan setelah edit terakhir):**
- Checker kustom `godoccheck` (godoc name-prefix + error contract): **0 problems**.
- `gofmt -l .` = kosong; `go build ./...` + `go vet ./...` = bersih; `go test ./...` = 6 paket **ok** (auth, db, handler, middleware, shortener, + no-test-file).
- `npm run build` frontend = sukses (21 routes).
- Scan em/en-dash di semua code (go/js/jsx/css/sql) = 0 file; hyperbole di code = 0; emoji dekoratif di code = 0 (simbol `★`/`✓`/`✕` yang ada adalah konten UI, dipertahankan).
- Komentar frontend/Go yang tersisa sudah WHY-rich (scan pola WHAT generik hanya menemukan baris lanjutan komentar multi-baris); `page.jsx:152` merujuk "TODO" yang memang masih tercatat di PROGRESS.md (referensi valid).

**File diubah:** `backend/internal/db/db.go` (kontrak terbanyak), `internal/auth/auth.go`, `internal/cache/cache.go`, `internal/migrate/migrate.go`, `internal/shortener/shortener.go`, `db/migrations/embed.go`, `cmd/worker/main.go` (+ file godoc-fix dari sesi sebelumnya: `cmd/proxy/main.go`, `cmd/smoketest/main.go`, `internal/ratelimit/ratelimit.go`, `internal/handler/handlers_profile.go`, `handlers_analytics.go`, `frontend/app/r/[code]/route.js`).

## 2026-10-03: Documentation polish: README + ARCHITECTURE
**Status:** Done (4 dokumen: README, ARCHITECTURE, SCHEMA, header PROGRESS ini; kode tidak disentuh)

**Task 1: README.md (tulis ulang penuh, 150 baris, bahasa Inggris sesuai struktur spesifikasi):**
- Struktur: intro 1 paragraf fakta, Features, Tech Stack (tabel Layer/Tech/Why), Getting Started (Prerequisites, Setup, Running 3 terminals), Development (Project Structure, Smoke Test, i18n, Themes), Deployment (placeholder `docs/deploy.md`), License (fakta: belum ada file LICENSE).
- Semua klaim diverifikasi dari kode langsung: Go 1.26 (go.mod), Node 18.17+ (syarat Next 14), Next.js 14 App Router, 14 migrasi, 11 tema, 3 locale, `APP_MODE` default `full`, `PORT` default 8080, `GO_API_URL` default 8081, rate limit API key 100 req/menit.
- Catatan port: 8081 bisa terisi proses lain (mis. XAMPP di mesin ini), jalur alternatif (PORT lain + `GO_API_URL`) didokumentasikan.

**Task 2: ARCHITECTURE.md (edit in place, tidak rewrite):**
- Tech stack diperbaiki ke fakta kode: baris API (`net/http` stdlib ServeMux, bukan chi), baris LB (`backend/cmd/proxy` round-robin, `infra/nginx.conf` sebagai alternatif, bukan Nginx sebagai utama), Dashboard (Next.js 14 + Tailwind), Orkestrasi (dokumen referensi, jalan native).
- Struktur folder: judul "Target" jadi "Aktual"; tambah `smoketest`, `migrate-replica`, `seed.sql`, subfolder frontend; hapus klaim keliru "SingleStore".
- Bagian baru: §7 Endpoint API (ringkasan 29 rute per kelompok) dan §8 Mode Runtime (`baseline` / `full` / `shard`).

**Task 3: SCHEMA.md (edit in place):**
- Header: catatan 14 migrasi aktif (penomoran 01-16, nomor 10-11 dilewati, 26 file termasuk `.down`) + tanggal verifikasi `information_schema`.
- `urls`: 8 kolom sebelumnya tidak terdokumentasi ditambahkan (`position`, `tags`, `device_rules`, `is_featured`, `unique_click_count`, `is_active`, `expires_at`, `creator_id`); koreksi catatan `expires_at` (dihapus migrasi 12, ditambahkan kembali migrasi 15).
- `click_events`: tambah `is_unique`, `referrer_domain`, `user_agent`, `device_type`, `referrer_type` (migrasi 16); koreksi catatan `country`.
- `creators`: tambah `theme` (migrasi 09) dan `email` (migrasi 14). Bagian baru: tabel `api_keys` (migrasi 08).

**Task 4: PROGRESS.md:**
- Entry lama tidak disentuh; tambah section "Cara pakai file ini" di header; deskripsi format tetap akurat.

**Verifikasi:**
- Grep em-dash dan en-dash di semua `*.md` = 0 file (termasuk file tersembunyi).
- Grep hyperbole (revolutionary, robust, seamless, powerful, dst.) di README + ARCHITECTURE + SCHEMA = 0.
- README = 150 baris (batas < 250). Perintah Setup/Running/Smoke Test = perintah yang selama ini terbukti jalan di mesin ini (`go run ./cmd/migrate-replica` dijalankan ulang saat verifikasi: exit 0, 0 migrasi baru).
- File code tidak diubah (hanya `*.md`): build tidak terpengaruh.

**File diubah:** README.md (tulis ulang), ARCHITECTURE.md, SCHEMA.md, PROGRESS.md (header + entry ini).

## 2026-10-03: UI polish: avatar, scrollbar, CTA, em-dash cleanup
**Status:** ✅ Done (4 dari 4 task; Firefox tak terinstall di mesin ini: verifikasi via jalur CSS + dokumentasi)

**Task 1: label avatar tab Profil (`frontend/app/dashboard/DashboardClient.jsx`):**
- Reproduksi CDP 375px sebelum fix: label `Avatar` sudah di atas lingkaran (overlap TIDAK terjadi di build saat ini), lingkaran 80px dengan border solid theme.
- Sesuai spesifikasi dirombak strukturnya: `<p>` jadi `<label htmlFor="avatar">` (di luar lingkaran), hidden input diberi `id="avatar"`.
- Lingkaran: `h-24 w-24` (96x96); state kosong = `border-2 border-dashed border-ink` (ink #1C1A12); state ada foto = border solid theme tetap.
- Placeholder dalam lingkaran = inisial huruf (bukan teks "Avatar"), overlay "Ganti foto" tetap on hover.
- Terverifikasi di 375px dan 1440px: tak overlap, 96x96, dashed 2px ink, `label[for="avatar"]` cocok.

**Task 2: scrollbar landing (`frontend/app/globals.css`):**
- **Bug akar ketemu dan dibuktikan**: rule global `* { scrollbar-width: thin; scrollbar-color: ... }` MEMATIKAN pseudo `::-webkit-scrollbar` di Chromium (properti standar menimpa pseudo), jadi pill 12px tidak pernah aktif dan yang tampil hanya bar tipis standar. Inilah sebab laporan "scrollbar tidak custom".
- Fix: properti standar dibungkus `@supports not selector(::-webkit-scrollbar)` sehingga HANYA Firefox yang menerimanya; Chromium/Safari kembali pakai pill webkit.
- Diperkuat: lebar desktop 14px via `@media (min-width: 768px)` (bawah 768px tetap 12px), `--scrollbar-thumb` ink 70% (dari 60%), hover tetap ink full.
- Terukur via CDP di 1440px: reserved scrollbar `14`, `scrollbarWidth: auto` (webkit aktif), var thumb `rgba(28,26,18,.7)`; pill terlihat di screenshot strip kanan.
- Firefox: `scrollbar-width: thin` + warna variabel yang sama; bentuk pill + lebar piksel presisi TIDAK BISA di Firefox (limitasi browser, expected). Dijelaskan di komentar globals.css. Mesin ini tidak punya Firefox: screenshot Firefox tak bisa dibuat, silakan cek di browser Anda.

**Task 3: CTA "Tertarik?" (`frontend/app/page.jsx`):**
- Bug nyata: `<p class="mt-2 max-w-xl text-base">` TANPA `mx-auto` sehingga box 576px menempel kiri di container 768px (teks bergeser ~96px ke kiri, line break tak seimbang).
- Fix: `mt-3 max-w-md mx-auto leading-relaxed` (box 448px ter-center, line-height lega). Copy tidak diubah.
- Terverifikasi CDP: 375/768/1440 x id/en/de = box <=448, center delta <=2px, teks bebas dash; screenshot `cta-375-id.png`, `final-cta-id-desktop.png`, `cta-1440-en.png`, `final-cta-de-desktop.png`.

**Task 4: hapus em-dash/en-dash seluruh repo:**
- Skrip sweep (`%TEMP%\opencode\emdash-sweep.js`), aturan: dash berspasi (U+2014) jadi `: ` (penjelasan), setelah `[.?!]` jadi spasi, sebelum tanda baca jadi tandanya saja, dash di ujung baris jadi `:`, sisanya `: `; en-dash (U+2013, rentang angka) jadi `-`.
- **1316 em-dash + 12 en-dash dihapus dari 103 file** (101 file dari sweep utama, lalu `.gitignore` 2 dan `.env.example` 3 dibersihkan terpisah). Terbesar: PROGRESS.md 506, frontend/lib/themes.js 84, frontend/app/page.jsx 45, messages id/en/de 40 masing-masing, backend handlers_links.go 36, DashboardClient 34. Kumpulan per-file tercetak di log skrip.
- Contoh hasil: title metadata `Jejak: Satu link, semua konten kamu` (3 locale), judul halaman `Kontak: Jejak` dll, `1-2 hari kerja`. JSON tervalidasi `JSON.parse` saat sweep; 0 kandidat di regex/logic code.
- Verifikasi: pencarian char U+2014 dan U+2013 (dengan `--hidden`, tanpa kecuali) = 0 file di seluruh repo, termasuk file tersembunyi root.

**Verifikasi keseluruhan:**
| cek | hasil |
|---|---|
| `npm run build` | hijau |
| `go build ./...` + `go vet` | hijau |
| smoke test (BASE_URL :8082) | **12/12 passed** |
| CDP polish-verify (scrollbar, CTA 3x3, avatar 375+1440, em-dash visible, lang+key-leak) | semua lulus |
| regresi verify-wiring + verify-3b (snapshot locale=id) | lulus (normalisasi delta: title metadata, label switcher, sweep em-dash) |
| grep em/en-dash | 0 |
| server | Next :3000 (GO_API_URL :8082), backend :8082, worker jalan |

**Files changed:** `frontend/app/dashboard/DashboardClient.jsx`, `frontend/app/globals.css`, `frontend/app/page.jsx`, `frontend/messages/{id,en,de}.json` + 97 file lain (komentar/string, lihat sweep), `PROGRESS.md` (entry ini). Catatan: format judul entry kini `YYYY-MM-DD: Judul` (konsekuensi sweep sendiri, sesuai garis header log).

---

## 2026-10-03: i18n Task 3B: sisa dashboard + language switcher + metadata
**Status:** ✅ Done (Poin D: pesan error Go API: di-SKIP jadi TODO, sesuai izin spesifikasi)

**Task 1: wiring sisa dashboard (Poin A/B/C):**
| poin | hasil |
|---|---|
| **A** ConfirmModal props di `PengaturanTab` | 2 call site (`confirmLogout.*`, `confirmDelete.title/description/confirmLabel/typeToken/passwordLabel`) → `t()`; **semua key sudah ada** (ekstraksi Task 2), 0 key baru |
| **B** `label="Salin"` di 3 lokasi | `DashboardClient` ×2 → `t("common.copy")`, `BulkImportModal` → `t("common.copy")`, `QrModal` → `t("forms.qr.copyUrlLabel")`; `CopyButton` sendiri sudah pakai `common.copy/copied/failed` |
| **C** label isi dashboard | 2 agent paralel: **DashboardClient 110 + PengaturanTab 42** dan **9 file** (RingkasanTab 12, AnalyticsTab 25, ClicksChart 17, ChartTooltip 13, Onboarding 4, ProfileLinks 5, links/page 1, BulkImport 23, Qr 4) = **256 `t()` baru**; termasuk `toast.*`, `errors.*`, month axis ×12, `{action}`/`{url}` split |
| **D** pesan error Go API | **SKIP (TODO)**: opsi C butuh `code` di **163 call site** backend (`http.Error`/`writeFieldError`); alternatif ringan tercatat di laporan (reverse-lookup value id→key di frontend) |

Total frontend kini **418 `t()`/`tr()` calls · 0 key missing** di id/en/de (script `%TEMP%\opencode\i18n\check-keys.js`). Shadowing `t` dibersihkan (loop tab → `tabItem`, `themeStyles` → `ts`/`ts`, `await res.text()` → `raw`).

**Task 2: `LanguageSwitcher.jsx` (baru) + integrasi `NavbarClient` (+6 baris):**
- Pill `border-2 border-ink bg-white rounded-full` + ikon lucide `Languages` 16px + kode aktif `ID|EN|DE` (tanpa emoji bendera).
- Dropdown `AnimatePresence`+`SPRING`: panel `bg-white border-2 border-ink shadow-[4px_4px_0_#1C1A12] rounded-[12px] p-2`; item native-name (Bahasa Indonesia / English / Deutsch), aktif `bg-flash-yellow font-bold`; klik → `setLocale()` → reload.
- A11y: `aria-haspopup="menu"`, `aria-expanded`, `aria-label` map lokal, Escape close, klik-luar close (listener dibersihkan saat tutup).
- Pasang di 3 posisi: desktop logged-in (sebelum avatar), desktop logged-out (sebelum Masuk/Daftar), mobile drawer (di atas tombol).

**Task 3: metadata per locale:**
- `lib/serverI18n.js` + `loadMessages(locale)`; `app/layout.jsx` static `metadata` → **`generateMetadata()`** (cookie → `loadMessages` → `messages.metadata.title/description` + `openGraph` + `metadataBase`).
- Key `metadata.title`/`metadata.description` ditambahkan ke id/en/de → **427 leaves/file, parity 0 mismatch**.

**Verifikasi (semua lulus):**
| cek | hasil |
|---|---|
| `npm run build` | hijau, semua route |
| HTTP: title/og:title/description per locale | id/en/de cocok `messages.metadata.*` |
| key-leak teks visible (landing/terms/privacy/contact/404/dashboard/links × 3 locale) | **0** |
| teks visible locale=id vs snapshot sebelum | identik (2 delta DIAWASKAN dinormalisasi: title metadata + label switcher) |
| **CDP Chrome headless** | switcher: 3 opsi ✓ highlight aktif ✓ Escape ✓ klik-luar ✓ klik "English" → cookie+reload+`lang=en`+label EN+headline EN ✓ |
| **CDP dashboard login** (`i18nsnap7`) | id `Ringkasan/Pengaturan` ✓ en `Summary/Settings` ✓ de `Übersicht/Einstellungen` ✓, ID-label hilang di en/de ✓ |
| regression `verify-wiring.js` (Task 3A) | lulus |

**Infra (insiden lingkungan):** `:8081` diambil **Apache XAMPP lama** (`C:\Users\akbar\Downloads\xampp-win32-1.7.2`, start 10/03 18:42; service `Apache2.2` = Stopped) sehingga Go backend tak bisa bind → backend dijalankan di **`:8082`** (`APP_MODE=full PORT=8082` + `DATABASE_REPLICA_URL`/`REDIS_URL`), Next distart dengan **`GO_API_URL=http://localhost:8082`**. Postgres/Redis/worker normal. Ingin kembali ke konvensi 8081 → matikan httpd XAMPP itu dulu.

**TODO / sisa diketahui:**
1. **Poin D**: `code` di response error Go + `translateError()` frontend (atau reverse-lookup id→key).
2. Metadata halaman per-file (terms/privacy/contact/404 `export const metadata`) masih Indonesia: spesifikasi hanya meminta layout.
3. `fmtDate()` masih `toLocaleString("id-ID")` (bukan literal terjemahan; saran: pakai `{locale}`).
4. Payload `confirmation: "HAPUS"` dikirim literal (kontrak backend); pesan error backend tetap Indonesia; 2 string dead-code (`Gagal memuat tren/ringkasan`) tak punya key.

**Files changed:** `frontend/app/components/LanguageSwitcher.jsx` (baru), `NavbarClient.jsx` (+6), `dashboard/DashboardClient.jsx`, `PengaturanTab.jsx`, `RingkasanTab.jsx`, `AnalyticsTab.jsx`, `components/{ClicksChart,ChartTooltip,OnboardingChecklist,ProfileLinks,BulkImportModal,QrModal}.jsx`, `app/links/page.jsx`, `lib/serverI18n.js` (+`loadMessages`), `app/layout.jsx` (generateMetadata), `messages/{id,en,de}.json` (+`metadata.*`), `PROGRESS.md` (entry ini).

---

## 2026-10-03: i18n Task 3A: wiring `t()` (landing, auth, halaman umum)
**Status:** ✅ Done (dashboard sengaja TIDAK disentuh: task berikutnya)

**Cakupan:** 13 file sesuai spesifikasi: `app/page.jsx` (landing+footer inline; tidak ada `Footer.jsx` terpisah), `app/terms|privacy|contact/page.jsx`, `app/not-found.jsx`, `components/NavbarClient.jsx`, `AuthModal.jsx`, `ShareModal.jsx`, `ConfirmModal.jsx`, `CopyButton.jsx`, `ShortenForm.jsx`, `SubmitButton.jsx`, `Navbar.jsx` (dicek: 0 string, tak diubah).

**Pola wiring:**
- Client components (11 file, sudah `"use client"`): `useTranslation()` dari `lib/I18nProvider` (relative import: repo tidak punya alias `@/`).
- Server pages yang export `metadata` (terms/privacy/contact/404): tetap server, pakai helper baru **`frontend/lib/serverI18n.js`** → `getServerTranslation()` (baca cookie via `next/headers`, load `messages/<locale>.json`, lookup identik sisi client).
- Refactor: logika lookup/interpolation dipindah ke **`frontend/lib/translate.js`** (murni, dipakai bersama client provider & server helper): `I18nProvider.jsx` di-rewrite pakai `translate()`.
- Edge cases DOM: `landing.hero.headline` (`\n` → split + `<br/>`), `footer.copyright` (split token `{heart}` → reinsert `<Heart/>`), `contact.paragraph2` (split token `{email}` → reinsert `<a href="mailto:">`), `forms.confirm.typingPrompt` (split `{requireTyping}` → reinsert `<span class="font-mono">`). NavbarClient memakai alias `tr` karena `t` sudah dipakai `themeStyles(pageTheme)`.

**Jumlah:** **162 `t()`/`tr()` calls** (page 59 · NavbarClient 35 · AuthModal 17 · ShortenForm 15 · ShareModal 9 · terms/privacy/contact 5+5+5 · 404 4 · ConfirmModal 4 · CopyButton 3 · SubmitButton 1 · Navbar 0): **0 key missing** di `id/en/de` (script `%TEMP%\opencode\i18n\check-keys.js`), **0 sisa literal Indonesia** di 13 file target.

**Verifikasi (`%TEMP%\opencode\i18n\verify-wiring.js`, server prod :3000 build 18:39):**
| cek | hasil |
|---|---|
| 5 halaman × 3 locale: status + `<html lang>` | ✅ (404→404, sisanya 200) |
| key-leak di teks visible (`landing.*` dll muncul mentah) | **0** |
| headline EN/DE muncul, headline ID hilang dari halaman EN/DE | ✅ |
| judul terms/404/contact EN+DE+ID | ✅ (perlu decode `&amp;`) |
| **teks visible locale=id == snapshot SEBELUM wiring** | **identik** (home 2113 · terms 529 · privacy 516 · contact 419 · 404 174 char) |
| `/dashboard` (pakai NavbarClient) | 200, `lang="de"` ✅ |
| `npm run build` | exit 0 |

**Sisa / Task 3B (belum):** string di call-site dashboard (`ConfirmModal` props `title/description/confirmLabel/requireTyping` di `PengaturanTab`, `CopyButton label="Salin"` di `DashboardClient`/`BulkImportModal`/`QrModal`), label tab/isi dashboard, `export const metadata` (title/description semua halaman), pesan error dari Go API, data demo landing (`nama-kamu`).

**Files changed:** `frontend/lib/translate.js` (baru), `frontend/lib/serverI18n.js` (baru), `frontend/lib/I18nProvider.jsx` (refactor pakai translate), 12 file target (wiring), `PROGRESS.md` (entry ini).

---

## 2026-10-01: i18n 3 bahasa (ID default, EN, DE): infra + dictionary
**Status:** ✅ Done (Task 1 + Task 2 spesifikasi; wiring komponen + language switcher belum: menunggu task berikutnya)

**Task 1: infrastruktur:**
- `frontend/messages/{id,en,de}.json`: 3 dictionary (ID = default).
- `frontend/lib/i18n.js`: `LOCALES ['id','en','de']`, `DEFAULT_LOCALE 'id'`, `normalizeLocale()` (case-insensitive + strip region `en-US`→`en`), `getLocale(cookieValue?)` (server: terima nilai cookie dari `cookies()`; client: baca `document.cookie`; fallback default), `setLocale(locale)` (tulis cookie `NEXT_LOCALE` max-age 1 tahun + `location.reload()`).
- `frontend/lib/I18nProvider.jsx`: client context; `I18nProvider({locale, messages})`; hook `useTranslation()` → `{ t, locale }`; `t(key)` dot-path lookup (`dashboard.tabs.ringkasan`) + fallback key itu sendiri bila tak ditemukan; dukungan opsional `t(key, {var})` untuk placeholder `{name}`.
- `frontend/app/layout.jsx`: baca `cookies().get("NEXT_LOCALE")` via `next/headers`, import statis 3 JSON (bundled, tanpa fetch runtime), wrap `<I18nProvider>` di root, `<html lang={locale}>`.

**Task 2: ekstraksi + terjemahan:** 4 subagent paralel (A: nav+landing+footer 87 · B: dashboard+onboarding 174 · C: auth+errors+toast+404/terms/privacy/contact 79 · D: forms+common 85) → **425 strings** hierarchical di `id.json`; lalu 2 subagent translate → `en.json` + `de.json` (percobaan DE pertama gagal menulis file, retry sukses). `common.*` memakai key Inggris (`save`, `cancel`, `copy`, ...) sesuai contoh spesifikasi; `dashboard.tabs.ringkasan/linkSaya` tetap key Indonesia sesuai contoh.

**Verifikasi:**
| cek | hasil |
|---|---|
| parity key id/en/de | **425/425, 0 missing, 0 extra** (urutan key identik) |
| parity placeholder `{var}` | **0 mismatch** di en & de |
| `\n` / spasi / `&` / segmen kalimat | 0 beda (EN & DE) |
| `npm run build` | hijau (Compiled successfully) |
| runtime `curl -H "Cookie: NEXT_LOCALE=..."` | tanpa cookie/`fr` → `lang="id"`, `en`→`en`, `de`→`de`, `en-US`→`en`, `ID`→`id` |
| halaman (dengan cookie `de`) | `/` `/app` `/dashboard` `/terms` `/u/themay76318` 200 · `/nonexistent` 404 |
| prerender | hanya `/icon` static; `/` dynamic (root layout baca cookie) |

**Catatan/deviasi:**
1. React **18.3.1** (bukan 19 seperti konteks task): tidak mempengaruhi API yang dipakai.
2. Cookie `NEXT_LOCALE` ternyata **belum punya pemakai sama sekali** di codebase (grep kosong): semua dibangun dari nol.
3. Dictionary ikut ter-serialize ke payload RSC tiap halaman (~24-26 KB per locale): konsekuensi pendekatan client-side dictionary.
4. Di luar cakupan: `export const metadata` (title/description masih Indonesia), pesan error dari Go API (backend), halaman publik `/u/[username]`, `next dev` port logic.
5. Belum ada komponen yang memanggil `t()`: UI masih menampilkan string hardcoded; pergantian bahasa saat ini baru terlihat di `<html lang>`.

**Files changed:** `frontend/lib/i18n.js` (baru), `frontend/lib/I18nProvider.jsx` (baru), `frontend/messages/{id,en,de}.json` (baru), `frontend/app/layout.jsx` (edit), `PROGRESS.md` (entry ini).

---

## 2026-10-01: Komentar codebase → Inggris (dokumentasi)
**Status:** ✅ Done

**Cakupan:** semua komentar di `backend/**` (`.go`, `.sql` migrasi + `seed.sql`) dan `frontend/**` (`.jsx`, `.js`, `.css`) diterjemahkan ke bahasa Inggris formal kualitas dokumentasi: fokus WHY/rationale, hapus marker `// LEARN:`, hapus komentar redundant/obvious, godoc-style untuk exported Go. **Logika kode tidak diubah**; dokumen (`README`, `PROGRESS`, `RULES`, `ARCHITECTURE`) dan string runtime (pesan error/warning) tetap Indonesia: di luar definisi "komentar".

**Pelaksanaan:** 8 batch subagent paralel (GO-A db+migrasi, GO-B handler links, GO-C analytics/profile/auth/account/apikeys/public, GO-D cmd+middleware+lib, FRONT-A lib+scripts, FRONT-B dashboard+pages, FRONT-C 18 komponen, FRONT-D globals.css+route.js API) + 6 file sisa dikerjakan manual: `frontend/next.config.js`, `frontend/tailwind.config.js`, `frontend/app/u/[username]/opengraph-image.jsx`, `backend/db/seed.sql`, 1 trailing comment `ClicksChart.jsx`, dan `gofmt -w` doc-comment `migrate.go`.

**Metrik:**
| cek | sebelum | sesudah |
|---|---|---|
| baris komentar Indonesia (Go/frontend) | **1.254** di 90 file | **0** (scan penuh: line-start + trailing, dictionary luas) |
| marker `// LEARN:` | banyak (GO-B 9, GO-C 6, dst.) | **0** |
| `gofmt -l` vs baseline | 19 file | **15 file** (4 file justru bersih; 0 regresi: `migrate.go` sempat kotor lalu diperbaiki) |

**Verifikasi (semua hijau):** `go vet ./...` 0 · `go test ./... -count=1` semua `ok` · `go build ./...` OK · `gofmt` set ≤ baseline · `npm run build` Next 14 sukses (semua route) · **smoke test 12/12 passed exit 0** (`SMOKE_STRICT=1`) · halaman `200`: `/`, `/app`, `/dashboard`, `/u/themay76318`, `opengraph-image` + `GET /api/u/themay76318`.

**Catatan:** seluruh pekerjaan task sebelumnya masih uncommitted (HEAD `f9c6c17`), sehingga `git diff` bercampur dengan task lama: jaminan "hanya baris komentar berubah" diambil dari self-check `git diff` per batch + verifikasi perilaku di atas. Fakta teknis dalam komentar dipertahankan (hex, rasio WCAG, TTL, limit, tanggal keputusan `2026-09-30`, pointer `README`/`PROGRESS`/`ARCHITECTURE.md §`).

**Files changed:** ~96 file komentar (daftar lengkap = 8 batch di atas + 6 manual); `PROGRESS.md`: entry ini.

---

## 2026-10-01: Tambah 2 tema: Ocean + Sunset
**Status:** ✅ Done

**Token utama (persis spesifikasi task; font TIDAK diubah: Space Grotesk + Work Sans + JetBrains Mono):**
| tema | paper | ink | accent | accentSecondary | muted | border / shadow / radius | accent bar | navHover |
|---|---|---|---|---|---|---|---|---|
| **ocean** | `#E8F2F7` | `#0F2A3D` | `#4A90B8` | `#7BB3D1` | `rgba(15,42,61,.68)` | 2px ink / keras `5px 5px 0 ink` / 12px | KIRI 4px accent | `#2A6A8A` |
| **sunset** | `#FFF3E0` | `#3D1F0F` | `#F5A623` | `#E8634A` | `rgba(61,31,15,.68)` | 2px ink / keras `4px 4px 0 ink` / 14px | KIRI 4px accent | `#A35F00` (deviasi) |

**WCAG audit (rasio kontras dihitung manual, rumus WCAG sRGB):**
| pasangan | ocean | sunset | syarat |
|---|---|---|---|
| teks utama ink @ paper | **13.02** | **13.66** | ≥7 ✅ |
| muted 0.68 @ paper | **4.97** | **5.11** | ≥4.5 ✅ |
| navHover @ paper | **5.24** `#2A6A8A` | **4.57** `#A35F00` | ≥4.5 ✅ |
| teks di atas aksen (★/CTA/avatar) | **4.78** `#0A1F2E` di `#4A90B8` | **7.39** ink `#3D1F0F` di `#F5A623` | ≥4.5 ✅ |
| TRENDING: ink @ accentSecondary | **6.49** di `#7BB3D1` | **4.51** di `#E8634A` | ≥4.5 ✅ |

**Deviasi spesifikasi (aturan audit: gelapkan TEKS, bg/border tetap spek):**
1. **sunset navHover `#C77800` = 3.13:1 GAGAL** untuk teks → dipakai **`#A35F00` = 4.57:1**. Dibuktikan di browser: `CSS.forcePseudoState(hover)` pada link nav → `rgb(163, 95, 0)` (idle `rgb(61,31,15)`).
2. **ocean teks-di-atas-aksen `#0F2A3D` @ `#4A90B8` = 4.21:1** (badge 11px butuh 4.5) → token `avatar`/`accent`/`cta` pakai ink lebih gelap **`#0A1F2E` = 4.78:1**; bg aksen tetap `#4A90B8`. Paksa-hover nav ocean → `rgb(42,106,138)` = `#2A6A8A` ✓.

**Files changed:**
- `frontend/lib/themes.js`: `THEMES` → **11 key** (`+ ocean, sunset`) + **2 preset penuh**: semua token chrome (page/border/borderW/borderStrong/radius/heading/headingFont/card/caption/avatar/accent/badge/text/textMuted/placeholder/featuredClass/chip/shadow/nav/navHover/cta/ghost/logoDot/panel/panelHover/panelRule/drawer/navCircle) + token opsional `swatch` (3 strip ink/aksen/paper), `url`, `barPrimary`/`barSecondary` (kiri 4px, kedua parity sama). Radius `rounded-xl`=12px (ocean), `rounded-[14px]` (sunset, di luar scale → arbitrary value). 9 tema existing **tidak disentuh** (hanya append).
- `frontend/app/globals.css`: 2 rule body baru (`background-color` paper + `color` ink tema) + **2 blok scrollbar pola selector kembar** `html:has(body[...]), body[...]`: ocean thumb `#4a90b8`/hover `#2a6a8a`, sunset `#f5a623`/hover `#c77800`; komentar daftar tema jadi 11.
- `backend/internal/handler/handlers_profile.go`: allowlist `validThemePreset` + 2 key (11 preset), pesan 400 diperluas, komentar preset 10-11; **sekalian perbaiki urutan import** (`jejak/internal/db` sebelum `middleware`) yang bikin `gofmt -l` kotor sejak sebelumnya (pre-existing, diverifikasi via `git show HEAD:`).
- `backend/internal/handler/handlers_profile_test.go`: loop preset 6 → **8** (`+ ocean, sunset`, assert echo theme), komentar "closed set = 11 preset".
- `PROGRESS.md`: entry ini.
- **TIDAK diubah:** 9 tema existing, logic shortener/analytics/redirect, Navbar/AuthModal/dashboard layout.

**Verifikasi (Chrome headless CDP: skrip `%TEMP%\opencode\verify-themes.mjs`; PUT tema ATOMIK `PUT ocean → ukur → PUT sunset → ukur`, akun uji `themay76318` + 2 link, dikembalikan ke `classic` di akhir):**
| # | cek | hasil |
|---|---|---|
| 1 | `PUT theme=ocean` → 200 echo `ocean`; render `/u/<user>` | attr `body[data-profile-theme]=ocean`, bg `rgb(232,242,247)`, teks `rgb(15,42,61)`, accent bar `4px rgb(74,144,184)` (2/2 kartu) ✅ |
| 2 | `PUT theme=sunset` → 200 echo `sunset`; render | attr `sunset`, bg `rgb(255,243,224)`, teks `rgb(61,31,15)`, bar `4px rgb(245,166,35)` ✅ |
| 3 | scrollbar ikut tema | `--scrollbar-thumb` di `:root` **`#4a90b8`/`#f5a623`**, `::-webkit-scrollbar-thumb` `rgb(74,144,184)`/`rgb(245,166,35)`, `scrollbar-color` Firefox ikut; **sampling piksel**: 648 px `#4A90B8` (ocean) & 648 px `#F5A623` (sunset) di 16px tepi kanan screenshot `*-scroll.png` (viewport dipendekkan biar halaman scrollable) ✅ |
| 4 | Navbar halaman publik ikut tema | `<header>` bar = `rgba(232,242,247,.85)` / `rgba(255,243,224,.85)`, border-b **2px** `rgb(15,42,61)` / `rgb(61,31,15)`; link hover paksa → `#2A6A8A` / `#A35F00` ✅ |
| 5 | Navbar `/`, `/dashboard`, `/app` TIDAK berubah | attr `(tanpa)`, bg `rgb(250,250,247)`, header `rgba(249,249,246,.8)` + border `rgb(28,26,18)` (classic) di ketiganya ✅ |
| 6 | Mobile 375 + 320 | `scrollWidth==clientWidth` (375/320), `overflow:false`, `maxRight==lebar` utk kedua tema ✅ |
| 7 | Theme picker | tab **Profil** → **11 chip** (Classic…Ocean, Sunset): desktop 1440 = 3 baris `chipsRight 987 ≤ 1430`; mobile 375 = 6 baris (2/rapat) `chipsRight 248 ≤ 375`; keduanya `overflow:false`; chip `aria-pressed` aktif = tema tersimpan (`Tema Sunset`); swatch ocean biru & sunset coklat-oranye tampil ✅ |
| 8 | Allowlist | `PUT theme=neon` → **400**; `night` tetap 400; `go test ./...` **ok** (loop 8 preset + echo) ✅ |
| 9 | Build | `gofmt -l` bersih (copy LF), `go vet ./...` **0**, `go test ./...` **ok**, `npm run build` **hijau** (11 route) ✅ |

**Screenshot** (`%TEMP%\opencode\shots2/`, Chrome headless CDP: ukuran diverifikasi): `ocean-375.png` (375×800), `ocean-1440.png`, `sunset-375.png`, `sunset-1440.png`, `ocean-scroll.png`/`sunset-scroll.png` (1440×420, bukti scrollbar), `chip-picker-11-1440.png` (1440×900), `chip-picker-11-375.png` (375×800), `landing-1440.png` (kontrol classic). **Eyeball:** kertas biru muda + tinta laut (ocean) dan krem + tinta coklat (sunset) konsisten di header/avatar/bar/shadow; kontras nyaman, kemiringan kartu ±2°/±0.6° tetap seperti Task 8.
**Cek manual:** hover scrollbar (thumb `#4A90B8`/`#F5A623` → hover `#2A6A8A`/`#C77800`) di browser asli: screenshot headless hanya memperlihatkan keadaan idle; hover link nav desktop (sudah dipaksa via CDP, belum dicoba mouse asli); kemiringan kartu di HP asli.

---

## 2026-09-30: Smoke test end-to-end 12 langkah (Go)
**Status:** ✅ Done
**Files changed:**
- `backend/cmd/smoketest/main.go`: **BARU**: smoke test HTTP asli (bukan `httptest`) yang menjalankan 12 langkah kritis berurutan: register → login → `GET /api/profile` → `POST /api/shorten` → `GET /r/{code}` (302) → `analytics/summary` (`total_links>=1`) → `analytics/breakdown?range=7d` → `analytics/export.csv` (`text/csv`) → `PUT /api/profile` (bio) → `PUT /api/account/email` (echo match) → `DELETE /api/account` (`confirmation=HAPUS`+password) → `GET /api/profile` **401**. Config via env `BASE_URL` (default `http://localhost:8081`) + `SMOKE_STRICT`; identitas random per run (`smoketest_{unixms}`, password & email `smoketest-{ts}@test.local`); checklist per langkah dengan durasi; **gagal → cetak request/body/status/response**; ringkasan `12/12 passed` atau `10/12 passed, 2 failed`; **exit 0/1** (siap CI); timeout 5 dtk/request (total run ratusan milidetik); **cleanup otomatis** akun uji bila test gagal di tengah (retries DELETE kalau langkah 11 sendiri yang gagal).
- `README.md`: §6: cara pakai `go run ./cmd/smoketest` (+ `BASE_URL`, `SMOKE_STRICT`) + catatan mode full/replica.
- `PROGRESS.md`: entry ini.
- **TIDAK diubah:** handler/backend logic, route, frontend, 9 tema.
**Why:** Task: script yang bisa dijalankan sebelum deploy untuk memastikan semua jalur kritis hidup; kalau lolos, produk sehat.
**Penyesuaian spesifikasi ↔ API nyata (4 temuan, semua diuji):**
1. **Register tidak menerima email**: `POST /api/register` hanya `{username, display_name, bio, password}` (kolom `creators.email` nullable, diisi via settings). Jadi: username/email/password dibuat random, **match email di-assert di langkah 10** lewat echo `PUT /api/account/email` → `{ok:true, email:"…"}`, dan langkah 3 memasang assert `email` siap pakai kalau `/api/profile` kelak mengeksposnya (saat ini `creatorProfileJSON` memang tidak membawa `email`). Field `email` tetap ikut dikirim saat register (`decodeJSON` tidak strict) supaya otomatis terpakai kalau handler mulai menerimanya.
2. **`POST /api/shorten` membalas 201 plain-text** `http://host/r/{code}` (bukan JSON) → `short_code` di-parse dari path `/r/…`.
3. **`breakdown` wajib `kind`** (`device|referrer`; `?range=7d` saja = 400) → langkah 7 memakai `?kind=device&range=7d` dan meng-assert echo `kind`+`range`.
4. **Mode full + replikasi manual** (README §3): `analytics/summary` membaca **replica** → `total_links=0` untuk link yang baru dibuat (terbukti: primary 76 vs replica 13 baris `urls`). Fallback: cek ke **primary** via `GET /api/profile` (`ListLinksByCreatorPrimary`): kalau link ada di sana, langkah **lolos dengan catatan** (`replica belum sinkron`), bukan gagal; `SMOKE_STRICT=1` mematikan fallback sehingga assert persis spesifikasi (`FAIL`).
5. Redirect tidak di-follow (`CheckRedirect → ErrUseLastResponse`): 302 adalah hasil uji, kalau di-follow yang diuji malah example.com. Cookie jar diganti **baru** sebelum login supaya assert `jejak_session` benar-benar milik login; kalau login gagal, jar lama (register) dipasang kembali agar cleanup tetap jalan.
**Verifikasi (server `:8081`, `go run ./cmd/server`):**
| skenario | hasil | exit |
|---|---|---|
| run normal | **12/12 passed** (271 ms; langkah 1 73ms, login 61ms, sisanya 0-2ms) + 1 catatan replica | **0** |
| `SMOKE_STRICT=1` | 11/12 (langkah 6 FAIL: dump `request/status/resp {"total_links":0,…}/alasan`) | **1** |
| `BASE_URL=http://localhost:9999` (mati) | 0/12 dalam 15 ms, tiap langkah cetak alasan transport | **1** |
| sisa sampah | `select count(*) from creators where username like 'smoketest_%'` → **0** (cleanup jalan) |
| suite backend | `go vet ./...` **0 error**, `go test ./...` **ok** (auth/db/handler/middleware/shortener) |
| build | `gofmt -l` bersih, `go build -o bin/smoketest.exe ./cmd/smoketest` sukses

---

## 2026-09-30: Polish: scrollbar custom + mobile fix + dev env
**Status:** ✅ Done
**Files changed:**
- `frontend/app/globals.css`: **(T1) Sistem custom scrollbar, CSS murni tanpa library**: `:root{--scrollbar-thumb: rgba(28,26,18,.6); --scrollbar-thumb-hover:#1C1A12}` (idle 60% → full saat hover, untuk landing/dashboard), `::-webkit-scrollbar{width/height:12px}` + track `transparent` + thumb `var(--scrollbar-thumb)` `border-radius:999px` `border:3px solid transparent` + `background-clip:content-box` (terlihat 6px, area hover 12px) + `:hover → --scrollbar-thumb-hover`; Firefox `*{scrollbar-width:thin; scrollbar-color:var(--scrollbar-thumb) transparent}` (rule global `*` supaya elemen scroll dalam-modal ikut). **9 varian tema** ditulis **dua selector** `html:has(body[data-profile-theme=…]), body[data-profile-theme=…]`: kembar `html:has()` WAJIB: atribut tema ada di `<body>`, tapi scrollbar dokumen (WebKit & `scrollbar-color`) baca variabel dari `<html>`; tanpa itu halaman publik tetap warna `:root`. Nilai = token aksen tema: classic/coral `#1C1A12`, darkroom `#FF6B35`, glass `rgba(28,26,18,.5)`, risoPrint `#1F3A5F`, peach `#FF9B7B`, lavender `#A88BEB`, matcha `#7BA05B`, sakura `#F5A4B8` (+ varian hover sedikit lebih terang/pekat). **(T2)** rule `.tilt-wrap` + `@media (max-width:640px){rotate(calc(var(--tilt)*0.3))}`.
- `frontend/app/components/ProfileLinks.jsx`: **(T2)** tiap `motion.a` dibungkus `<div className="tilt-wrap" style={{"--tilt":"…deg"}}>`; **rotate dihapus dari `animate` framer-motion** (dipindah ke CSS: supaya kemiringan bisa diperkecil per breakpoint; entrance/hover/tap framer tetap jalan, transform tidak tabrakan). Row kartu `gap-3 px-4` → **`gap-2 px-3 … sm:gap-3 sm:px-4`** (caption juga `px-3 … sm:px-4`), grup badge ditambah **`whitespace-nowrap`** (di samping `shrink-0` yang sudah ada). Layout desktop ≥640px identik dengan sebelumnya.
- `frontend/package.json`: **(T3)** scripts: `dev` → `next dev -p 3000`, `dev:alt` → `next dev -p 3001`, `start` → `next start -p 3000`, `build`, `clean` → `rimraf .next`; devDependency baru **`rimraf@^6`**.
- `frontend/next.config.js`: **(T3)** `warnIfServerAlreadyRunning(3000)`: kalau `NODE_ENV=production` **dan argv memuat `build`** (probe net 500ms), cetak peringatan "port 3000 sudah terisi → kemungkinan `next dev`, matikan dulu; dev & start share `.next`". Cek sengaja **hanya saat `next build`**: saat `next start` port sudah bisa diikat oleh prosesnya sendiri sebelum config dimuat → probe jadi **false positive** (terbukti: peringatan selalu muncul walau port kosong; dibuktikan juga `next start -p 3001` dengan :3000 kosong → TANPA peringatan). Bentrok di `start` sudah dilaporkan Next sendiri sebagai `EADDRINUSE`.
- `README.md`: **(T3)** §7 "Frontend (Next.js): dev vs prod": `npm install`, `npm run dev` (:3000), `npm run dev:alt` (:3001), `npm run build && npm run start`, `npm run clean`, plus kotak PENTING "jangan jalankan dev + start bersamaan" + penyebab/solusi.
- `PROGRESS.md`: entry ini.
- **Dihapus:** `frontend/public/__debug-overflow.html` (alat ukur sementara: ukur lewat iframe; digantikan pengukuran CDP langsung).
- **TIDAK diubah:** warna/font/layout existing, Navbar, AuthModal, 9 preset tema (`lib/themes.js`), logic redirect/analytics/shortener, kode Go.
**Why:** Task: (1) custom scrollbar per tema, (2) fix "mobile overflow" kartu link di 375px, (3) setup dev environment + dokumentasi.
**Temuan kritis (T2): "overflow 375px" itu PALSU: artefak harness, bukan bug layout:**
1. `--window-size=375` di Chrome **desktop** di-clamp ke **lebar minimum ~504px** (diukur: minta 320/375/414 → `window.innerWidth=504`; minta 600 → 582; minta 1440 → 1422). `--screenshot` tetap menulis gambar selebar permintaan → **layout 504 yang dipotong jadi 375** → sudut kartu + badge tembus tepi gambar. Itulah "overflow" pada screenshot Task 7.
2. Bukti pengukuran sesungguhnya (CDP `Emulation.setDeviceMetricsOverride`, viewport 320/375/414 asli) **sebelum** perubahan: `scrollWidth == clientWidth` di ketiga lebar → **tidak ada overflow horizontal sama sekali**. (Alat: `measure-overflow.mjs` di `%TEMP%\opencode\`: Node 24 + `WebSocket` global, speak CDP langsung.)
3. Tetap diterapkan hardening sesuai task (semua terukur): tilt **±2° → ±0.6°** di ≤640px (desktop tetap ±2°), padding row mobile `px-3`, gap mobile `gap-2`, badge `whitespace-nowrap`. Insurance `overflow-x: clip` sudah ada sejak awal di `html,body` (`globals.css`): clip di `main`/section sengaja TIDAK ditambah karena akan memotong hard-shadow 4px kartu.
**Verifikasi T1: warna thumb (CDP `getComputedStyle`, idle):**
| konteks | `--scrollbar-thumb` di `<html>` | catatan |
|---|---|---|
| landing `/` | `rgba(28,26,18,.6)` | idle 60%, hover `#1c1a12` |
| 9 tema di `/u/medaka` | classic/coral `#1c1a12`, darkroom `#ff6b35`, glass `rgba(28,26,18,.5)`, risoPrint `#1f3a5f`, peach `#ff9b7b`, lavender `#a88beb`, matcha `#7ba05b`, sakura `#f5a4b8` | **9/9 sesuai** token; pseudo WebKit ikut (`::-webkit-scrollbar-thumb.backgroundColor` benar) |
Piksel screenshot (Chrome headless **tanpa** `--hide-scrollbars`, `%TEMP%\opencode\shots\`): `sb-light.png` (1440×900) kolom x=W-5 y=60…200 = **`117,116,110`** = ink 60% di atas print-white (hitungan persis `0.6·28+0.4·250`), y≥300 = track putih; `sb-darkroom.png` (1440×300, viewport dipaksa pendek supaya benar-benar scroll) = **606 piksel oranye `255,107,53` = `#FF6B35`** di strip scrollbar (saat viewport 500/900 halaman tidak scroll → memang tanpa scrollbar, itu sebabnya sampel pertama gelap semua).
**Verifikasi T2: geometri mobile (CDP, viewport asli):**
| lebar | overflow | scroll==client | badge kanan (butuh ≤ lebar) | tilt |
|---|---|---|---|---|
| 320 | **false** | ya | 290/320 | **−0.6°** |
| 375 | **false** | ya | 345/375 | **−0.6°** |
| 414 | **false** | ya | 384/414 | **−0.6°** |
| 1440 (desktop) | false | ya | 1022/1440 | **−2°** (sesuai desain) |
9 tema di 375 **dan** 1440: `overflow=false` semua, badge muat, tilt konsisten (−0.6 / −2), thumb per tema benar; `creators.theme` medaka **dikembalikan ke `classic`**.
**Verifikasi T3: dev environment:**
- `npm run build` **hijau 32/32**; `npm run clean` menghapus `.next` (dicek `Test-Path=false`); `npm run dev` **Ready :3000**; `npm run dev:alt` **Ready :3001** (status 200, :3000 kosong): keduanya dijalankan berurutan dengan server lain berhenti, lalu `.next` dibersihkan + build ulang.
- Peringatan `next.config.js` teruji dua arah: `next build` saat :3000 terisi → **PERINGATAN tercetak** + build tetap sukses; `next start` (port bebas) → **tanpa** peringatan (false positive sudah dibenahi).
- Smoke prod final (`npm run start`, :3000): `/` **200**, `/u/medaka` **200**, `/r/abc123` **302** `location: https://example.com` (proxy utuh), `/r/tidakada` **404** (halaman fallback), log start **tanpa** peringatan.
- Catatan harness: `Start-Process cmd /c npm …` **wajib `-WorkingDirectory frontend`**: tanpa itu npm naik ke `package.json` di `C:\Users\akbar\` (`start: node src/app.js`) dan crash `MODULE_NOT_FOUND`.
**Perlu dicek manual (tidak bisa di-headless):** (1) warna thumb saat **di-hover** di browser asli (rule `::-webkit-scrollbar-thumb:hover` + `--scrollbar-thumb-hover` per tema sudah ada & nilainya terverifikasi, tapi `Input.dispatchMouseEvent` tidak memicu hover scrollbar di headless); (2) kemiringan kartu ±0.6° di HP sungguhan.

---

## 2026-09-30: Redirect diagnose + fallback page + verify 9 tema
**Status:** ✅ Done
**Files changed:**
- `frontend/app/r/[code]/route.js`: **halaman fallback Jejak-style** untuk error dari backend Jejak sendiri. 302 tetap **diteruskan apa adanya** (tanpa pernah cek URL tujuan: URL mati di luar kendali Jejak). Status lain: **404** → "Link tidak ditemukan.", **410** → "Link sudah kadaluarsa.", **≥500** → "Sedang ada masalah. Coba lagi."; `fetch` gagal (backend mati) → halaman 5xx dengan **status 502**. `FALLBACK`/`ICONS` (SVG inline `searchX`/`clock`/`alert`) + `fallbackPage()` + `htmlResponse()`; token **inline style** (`#FAFAF7` print-white, `#1C1A12` ink, `#FFD23F` flash-yellow, teks `rgba(28,26,18,.75)` ≈7.4:1) karena HTML mentah tidak memuat bundle CSS Next → kelas Tailwind tidak berlaku. Card: border 2px ink, shadow keras `4px 4px 0`, radius 16px, pill CTA kuning "Balik ke Beranda" → `/`, `<title>` ikut pesan error. Status 400 (kode tidak valid) tetap plain text. Bug build awal: key objek `5xx` harus di-quotes (`"5xx"`): "Identifier cannot follow number".
- `PROGRESS.md`: entry ini.
- **TIDAK diubah:** 9 tema, `ProfileLinks.jsx`, logic shortener/analytics, Navbar, AuthModal, kode Go.
**Why:** Task: (1) diagnosa "URL not found" di `/r/web_belajar`, (2) ganti error teks polos jadi halaman Jejak-style, (3) verifikasi visual 9 tema.
**Temuan diagnosa "URL not found" (SPESIFIK: bukan bug data, bukan bug proxy):**
1. **Data bersih:** `SELECT` → `web_belajar` = `https://roadmap.sh/system-design?fl=1` (37 char, `is_active=t`, `expires_at` NULL, tanpa whitespace/newline/encoding aneh), `web_favorit` = `https://11.shinigami.asia/`; keduanya **ada di primary DAN di replica**; tujuan hidup (`roadmap.sh` HEAD **200**, `11.shinigami.asia` **403** = hidup tapi nolak bot).
2. **Proxy benar:** `curl -I` identik Location dari backend (`:8081` → 302 `https://roadmap.sh/system-design?fl=1`) maupun via proxy (`:3001` → 302, Location persis sama) → tidak ada bug parsing `Location`.
3. **"URL not found" = teks 404 Go** (`http.Error(w, "URL not found", 404)` di `HandleRedirect`): terjadi saat backend jalan **mode `full` dengan schema replica tertinggal** (kolom `expires_at` belum ada → query baca gagal → semua kode 404). Itu bug yang sudah diperbaiki pada task sebelumnya (sync schema replica). Bukti tambahan: mode `full` yang sudah di-sync kini mengembalikan **302** untuk kode yang sama. **Kesimpulan: bukan URL mati, bukan bug proxy.**
4. **Temuan lain (yang bikin500 di :3000):** `next dev` (:3001) dan `next start` (:3000) **berbagi folder `.next` yang sama** → dev menulis ulang artefak route `/r` jadi *dev-compiled* (eval-source-map) → prod `next start` gagal eksekusi route → **500 Internal Server Error** khusus `/r/*` (halaman lain tetap 200). Fix: matikan dev, `npm run build`, start ulang prod. **Aturan: jangan jalankan dev + prod bersamaan.**
**Verifikasi halaman fallback (semua di `:3000` prod, lalu dikembalikan normal):**
| kasus | cara picu | hasil |
|---|---|---|
| 302 | `/r/web_belajar` | **302** `Location: https://roadmap.sh/...` (diteruskan apa adanya) |
| 404 | `/r/tidak_ada` | **404** + halaman "Link tidak ditemukan." + CTA |
| 410 | insert `exp_uji` `expires_at = now()-1 day` | **410** + "Link sudah kadaluarsa." + CTA (row test di-DELETE sesudahnya) |
| backend mati | stop :8081 → `/r/abc123` | **502** + "Sedang ada masalah. Coba lagi." + CTA |
| backend 5xx | server palsu :9999 balas 500 + start prod `GO_API_URL=http://localhost:9999` | **500** + halaman 5xx yang sama |
  Setelah test: fake server :9999 dihentikan, prod di-start ulang **tanpa** `GO_API_URL` override, backend :8081 baseline dijalankan lagi → `/r/abc123` **302**.
**Verifikasi 9 tema (medaka, tema di-set via `UPDATE creators SET theme=…`, prod :3000):**
| tema | paper (piksel vs token) | border | accent bar (piksel di kartu) | shadow | radius | verdict |
|---|---|---|---|---|---|---|
| classic | `F8F8F5` ≈ `#FAFAF7` | `1C1A12` 2px | **tanpa bar** (memang tak punya `barPrimary`) | `4px_4px_0px_#1C1A12` | xl/2xl | MATCH |
| darkroom | `131313` ≈ `#121212` | `FF6B35` 2px (kiri+kanan) | tanpa bar | `4px #FF6B35` | xl | MATCH |
| coral | `F3EFE6` ≈ `#F5F1E8` | `1C1A12` | tanpa bar | `4px #1C1A12` | xl | MATCH |
| glass | `F1F1F1` (translucent) | putih 1px (`white/[0.28]`, piksel `FEFEFE`) | tanpa bar | `inset_0_1px_0…, 0_12px_40px…` | 2xl | MATCH |
| risoPrint | `F0ECE2` ≈ `#F4F0E6` | `1F3A5F` | **kiri 4px `FF5C39`** (x400-403) | `4px #1F3A5F` | lg | MATCH |
| peach | `FDEFE4` ≈ `#FFF1E6` | `2B1F1A` | **kiri 4px `FF9B7B`** | `4px #2B1F1A` | xl | MATCH |
| lavender | `F3EEFD` ≈ `#F5F0FF` | `2A2140` | **kiri 4px `A88BEB`** | **6px soft** `rgba(42,33,64,.15)` | 2xl | MATCH |
| matcha | `EDF3EA` ≈ `#EFF5EC` | `1F2E1A` **1.5px** | **kiri 3px `7BA05B`** (x400-402) | **3px** `#1F2E1A` | **`rounded-[10px]`** | MATCH |
| sakura | `FDF3F6` ≈ `#FFF5F8` | `2B1F26` | **KANAN 4px `F5A4B8`** (x1036-1038; kiri tetap ink) | **tinted** `4px rgba(245,164,184,.5)` | 2xl | MATCH |
- **9/9 MATCH**, tidak ada yang perlu tweak. Angka kontras = audit WCAG task sebelumnya (semua ≥4.5:1; muted matcha pakai `/[0.72]` = 5.48). DOM per tema dicek via `--dump-dom`: `data-profile-theme` benar + kelas bar/shadow/border/radius persis token (bar hanya ada di risoPrint + 4 tema baru, sesuai `ProfileLinks.jsx:90`: `classic/darkroom/coral/glass` memang tidak punya `barPrimary`).
- **Picker 9 chip:** chunk build `.next/static/chunks/957-*.js` memuat ke-9 label (`Classic…Sakura`) ✓.
- **Screenshot:** `%TEMP%\opencode\shots\{theme}-375.png` + `{theme}-1440.png` untuk ke-9 tema (Chrome headless `--force-prefers-reduced-motion`), plus crop `card-*.png`. Pakai reduced-motion karena screenshot default menangkap animasi entrance (kartu masih miring).
- **2 observasi (BUKAN bug tema, tidak diubah):** (a) kartu link sengaja miring ±1-2° (`l.tilt` deterministik, `ProfileLinks.jsx:78-80`): desain stiker; (b) **di mobile 375px kartu link melewati tepi kanan viewport** (border kanan + badge TRENDING terpotong, `overflow-x: clip` menutupinya): mempengaruhi **semua** tema, jadi layout issue pre-existing di luar scope task ini.
**Build status:** `npm run build` **hijau** (setelah fix quotes `"5xx"`); backend tidak diubah: `go build ./...` + `go vet ./...` + `go test ./...` tetap **ok**.
**Runtime:** prod `next start` :3000, backend baseline :8081, 1 worker, `next dev` :3001 **dimatikan** (jangan dinyalakan bersamaan dengan prod).

## 2026-09-30: Backend hygiene: replica sync + worker guard
**Status:** ✅ Done
**Files changed:**
- `backend/cmd/migrate-replica/main.go`: **BARU.** Runner migrasi khusus replica/primary: flag `-target=replica|primary` (default `replica`), DSN dari `DATABASE_REPLICA_URL`/`DATABASE_URL` (override `-url`), `env.LoadDotEnv()`, `sql.Open("pgx")` + `Ping()`, lalu **memakai runner yang sama** `migrate.Run(db, log.Default())` (`backend/internal/migrate`, idempoten, toleran `42710/42P07/42701`), `maskDSN()` (regex `(?i)(://[^:/@]+:)[^@]+@` → `***@`) supaya password tidak bocor ke log, LEARN panjang soal bug asimptomatik 404 + alternatif `pg_dump -s | psql` yang ditolak (hanya fallback darurat: tidak membawa `schema_migrations`, gampang beda urutan kolom).
- `backend/cmd/worker/main.go`: **guard single-instance via Redis**: konstanta `workerLockKey="jejak:worker:lock:click_events"`, `workerLockTTL=30s`, `workerHeartbeat=10s`; script Lua `renewWorkerLock` (GET==id → PEXPIRE, selain itu 0) & `releaseWorkerLock` (GET==id → DEL); `workerInstanceID()` = `hostname:pid`; utama: `SetNX` (timeout 5s): kalau sudah ada → log `Worker sudah jalan, keluar.` + `os.Exit(1)`; `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)`; goroutine heartbeat tiap 10s (err → warning + lanjut; `n==0` → `Worker lock diambil proses lain: keluar.` + exit 1); saat shutdown release lock (timeout 3s). Import ditambah `fmt`/`os/signal`/`syscall`.
- `README.md` §3: langkah migrasi replica diganti `go run ./cmd/migrate-replica` (dulu: `psql -f ...01_initial_schema.sql` ke dua DB = **sumber drift**), cara verifikasi `\d urls`, peringatan "schema replica tertinggal = semua redirect mode full 404"; §5: dokumentasi lock worker (satu instance, exit sendiri kalau kembar, kunci dilepas saat SIGINT/SIGTERM, worker keluar kalau Redis mati).
- `ARCHITECTURE.md` §6: bedakan **schema sync** (`go run ./cmd/migrate-replica`, wajib) vs **data sync** (manual seed/INSERT, tetap disengaja).
- `.env.example`: komentar `DATABASE_REPLICA_URL`: wajib jalankan `migrate-replica` setiap menambah migrasi.
- `PROGRESS.md`: entry ini.
- **TIDAK diubah:** logic redirect (`HandleRedirect`), kode server/migrasi utama (`cmd/server/main.go` tetap hanya auto-migrate **primary**), frontend, 9 tema, warna/font/layout.
**Why:** Task: mode `full` rusak (semua link redirect 404), 2 worker kembar, dan `DATABASE_REPLICA_URL` belum terdokumentasi cara sync schema-nya.
**LEARN (root cause 404 mode full):** replica `jejak_replica` **tidak pernah di-migrate**: README §3 lama menyuruh apply **hanya migrasi 01** ke replica → kolom `expires_at` (migrasi 15) tidak ada. Mode full baca redirect dari replica (`SingleStore.readDB()` → `GetLink` SELECT dengan `expires_at`) → query gagal → `HandleRedirect` balas `URL not found` (404) untuk **semua** kode, termasuk yang ada. **Direproduksi & dibuktikan:** binary mode full → 404 identik; `/api/u/medaka` tetap 200 (profil publik baca primary) & SQL `GetLink` manual di primary sukses → kode redirect/`.env`/Redis/DB primary **bukan** penyebab; `information_schema.columns` replica kala itu **tidak punya `expires_at`**. **Kenapa baseline selamat:** `cmd/server/main.go:61-67` case `baseline` memanggil `NewSingleStore(dbURL, "")` → replica dipakai **hanya** di mode full. Data sync replica tetap manual (design; baris baru hanya di primary): itu demo "replication lag", bukan bug.
**Cara sync replica (sekarang):** dari `backend/`, `go run ./cmd/migrate-replica` (target default = replica; `-target=primary` bila perlu). Jalankan **setiap kali menambah file `backend/db/migrations/*.sql`**. Rerun kedua = `0 migrasi baru diterapkan, 26 sudah ada` (idempoten). Fallback darurat kalau runner gagal: `pg_dump -s` per DB lalu diff.
**Verifikasi (semua dijalankan & tercatat):**
1. **Schema:** `migrate-replica` ke `jejak_replica` → 26 migrasi diterapkan (toleran duplikat dijalankan sebagai "sudah ada"), 0 error; `\d urls` replica kini punya `expires_at` + `idx_urls_expires_at`; **diff `information_schema.columns` primary vs replica = 0 selisih** (40 kolom vs 40), **index 16 vs 16 identik**, `schema_migrations` **26 vs 26**.
2. **Mode full (Task 3):** kill semua backend → start `bin/api-8081.exe` `APP_MODE=full` (log → `api-full.err.log`) → `/r/yt_favorit` **302** (YouTube), `/r/abc123` **302**, `/r/web_favorit` **302**: dibaca dari replica, bukan cache (log `Cache miss`); **0 baris error query** di log; kode yang **belum disalin data** ke replica (`wwGWI0`, `LL9EAT`) → **404 = gap DATA, sesuai desain** (dokumentasi README/ARCHITECTURE); `POST /api/shorten` **201** (`TsE9nH`); analytics summary/breakdown/timeseries **200** (`data_per: primary`: lag read replica tidak dipenuhi → fallback); `GET /api/u/medaka` **200**; klik `/r/yt_favorit` ×2 → worker menulis **2 baris** `click_events` di primary.
3. **Worker guard (Task 2):** kill 2 worker lama (25508/25356+3580) → start 1× `bin/worker.exe` → log `Worker lock diambil: LAPTOP-RODMKGDI:5688 (TTL 30s, heartbeat tiap 10s)`; instance kedua → log `Worker sudah jalan, keluar.` + **exit sendiri** (alive=False), hanya 1 worker tersisa; **heartbeat terbukti**: RESP `TTL jejak:worker:lock:click_events` = `26 → 21 → 25 → 20` (lonjakan 21→25 = perpanjangan; tanpa heartbeat akan terus turun); `GET` = `LAPTOP-RODMKGDI:5688`.
4. **Build/test:** `go build ./...` + `go vet ./...` + `go test ./...` **semua ok** (16 paket; `cmd/migrate-replica`/`cmd/worker` tanpa test file).
5. **Runtime dikembalikan:** API **`APP_MODE=baseline`** (sesuai `.env`, PID 15012 :8081): `/r/abc123|yt_favorit|wwGWI0|TsE9nH` semua **302**; worker 5688 (bin baru, lock aktif); frontend `next start` :3000 **200**. `next dev` :3001 tetap mati (jangan dijalankan bersamaan).
**Perlu dicek manual:** (1) bila ingin pakai mode full lagi, set `APP_MODE=full` di `.env`: kode yang **belum ada di replica** tetap 404 (sync data manual: `INSERT` baris sama / jalankan `seed.sql` di replica); (2) lock worker butuh Redis: worker **tidak mau start** kalau Redis mati; (3) pastikan hanya 1 worker yang dijalankan saat deploy.

## 2026-09-30: Tambah 4 tema: Peach, Lavender, Matcha, Sakura
**Status:** ✅ Done  
**Files changed:**
- `frontend/lib/themes.js`: **`THEMES` → 9 key** (`classic, darkroom, coral, glass, risoPrint, peach, lavender, matcha, sakura`) + **4 preset baru** dengan SEMUA token chrome (page/border/borderW/borderStrong/radius/heading/card/caption/avatar/accent/badge/text/textMuted/placeholder/chip/shadow/nav/navHover/cta/ghost/logoDot/panel/panelHover/panelRule/drawer/navCircle) + token opsional `swatch`/`url`/`barPrimary`/`barSecondary`. **Token baru per preset (persis spesifikasi):**
  - **peach**: paper `#FFF1E6`, ink `#2B1F1A`, accent `#FF9B7B`, accent2 `#FFC8A8`, muted `rgba(43,31,26,.65)` → `text-[#2B1F1A]/65`; border `border-2` ink; shadow keras `4px_4px_0px_#2B1F1A`; radius `rounded-xl` (12px); accent bar **kiri** `border-l-4 border-l-[#FF9B7B]`.
  - **lavender**: paper `#F5F0FF`, ink `#2A2140`, accent `#A88BEB`, accent2 `#C9B6F0`, muted `rgba(42,33,64,.65)`; border 2px; shadow **SOFT** `6px_6px_0px_rgba(42,33,64,0.15)`; radius `rounded-2xl` (16px); bar kiri 4px `#A88BEB`.
  - **matcha**: paper `#EFF5EC`, ink `#1F2E1A`, accent `#7BA05B`, accent2 `#A8C68A`; border **`border-[1.5px]`** (lebih tipis); shadow halus `3px_3px_0px_#1F2E1A`; radius **`rounded-[10px]`**; bar kiri **`border-l-[3px]`** `#7BA05B`; muted **`/[0.72]`** (deviasi: 0.65 = 4.44:1 gagal AA).
  - **sakura**: paper `#FFF5F8`, ink `#2B1F26`, accent `#F5A4B8`, accent2 `#FAD0DA`; border 2px; shadow **TINTED** `4px_4px_0px_rgba(245,164,184,0.5)`; radius `rounded-2xl` (16px); accent bar **KANAN** `border-r-4 border-r-[#F5A4B8]` (satu-satunya dari 9 tema).
  - `swatch` semua = `[ink, accent, paper]` (3 strip utk picker, diminta task). `url` = muted (AA). `barSecondary` = nilai sama dgn `barPrimary` (paritas indeks kartu → bar seragam; kalau tidak, template string jadi "undefined").
  - Komentar header file diperbarui (daftar key tertutup, palet 4 tema, definisi token opsional kini dipakai risoPrint + 4 tema baru).
- `frontend/app/globals.css`: **4 rule body baru** `body[data-profile-theme="peach|lavender|matcha|sakura"]` = `background-color` (paper) + `color` (ink tema): pola sama dgn darkroom/coral/risoPrint (teks TANPA token ikut ink tema, bukan `#1C1A12`); komentar daftar tema diperbarui. Tanpa ini panggung jatuh ke `bg-print-white`.
- `backend/internal/handler/handlers_profile.go`: **allowlist `validThemePreset` + 4 key** (`case "classic","darkroom","coral","glass","risoPrint","peach","lavender","matcha","sakura"`), pesan error 400 diperluas, komentar disesuaikan. `validTheme` (read path) otomatis ikut: payload GET tidak membocorkan nilai tak dikenal.
- `backend/internal/handler/handlers_profile_test.go`: loop preset diperluas `{"glass","risoPrint","peach","lavender","matcha","sakura"}` + assert **echo theme** per preset (dulu hanya cek status); komentar "closed set" diperbarui. `night`/`neon` tetap 400.
- `PROGRESS.md`: entry ini.
- **TIDAK diubah:** 5 tema existing (identik byte), `ThemeBackdrop.jsx` (dicek: murni grain global, **tanpa logic per tema** → tidak ada yang perlu ditambah), `ProfileLinks.jsx`/`NavbarClient.jsx` (memang membaca token → 4 tema baru otomatis), warna/font/layout dashboard, analytics, shortener.
**Why:** Task: 4 tema baru dgn karakter playful/cute (Peach hangat, Lavender dreamy, Matcha zen/earthy, Sakura kawaii) yang hidup di halaman publik `/u/[username]` + navbar di sana, dengan picker 9 chip, allowlist backend, dan lolos WCAG AA.
**Notes:** **WCAG audit (kontras dihitung, ratio = (L1+.05)/(L2+.05)):**
| tema | teks utama ink/paper | muted (caption/url/placeholder) | ink di atas accent (★/avatar/CTA) | ink di atas accent2 (TRENDING 11px) | navHover di paper |
|---|---|---|---|---|---|
| peach | **14.45** | **4.80** (`/65`) | **7.78** | **10.72** | **5.25** (`#A8481F`) |
| lavender | **13.52** | **4.64** (`/65`) | **5.44** | **8.22** | **6.91** (`#5B3FA8`) |
| matcha | **12.94** | **5.48** (`/[0.72]`; `/65` = **4.44 GAGAL**) | **4.80** | **7.59** | **6.43** (`#43602C`) |
| sakura | **14.83** | **4.85** (`/65`) | **8.20** | **11.38** | **5.46** (`#A44561`) |
  Semua ≥4.5:1 → **lolos AA utk teks normal**. Dua penyesuaian di luar teks spesifikasi (aturan task: "shade lebih gelap utk teks"): (1) **muted matcha 0.65 → 0.72** (4.44 → 5.48); (2) **navHover pakai shade TUA tiap tema**: accent terang (`#FF9B7B`/`#A88BEB`/`#7BA05B`/`#F5A4B8`) cuma ~1.7-2.5:1 di paper → dipakai utk bg/border/bar saja, persis pola coral `#C43A20`/riso `#C43A20`. Angka klik count tidak pakai token → mewarisi ink (≥12.9:1).
**Verifikasi (semua via `curl` + Chrome headless, tanpa intervensi manual):**
1. **Save → render:** register `themeday9674` (cookie), `PUT /api/profile` ke-4 tema → **200 + echo tema benar**; `GET /api/profile` (proxy :3000 **dan** langsung :8081) balik `"theme":"lavender"` dst; **`GET /u/<user>` per tema** memuat kelas token di HTML SSR: peach 5/6, lavender 5/6, matcha 6/7, sakura 5/6 (**yang tidak ada = hanya `hover:text-*` navbar → kelas client-side, sudah diverifikasi terpisah**), **accent bar `border-l-*`/`border-r-*` hadir di SEMUA tema** ✓ (gap = sakura bar kanan).
2. **Navbar publik ikut tema:** Chrome `--dump-dom` `/u/themeday9674` (tema peach) → `data-profile-theme="peach"` di `<body>` + kelas chrome `hover:text-[#A8481F]`, `bg-[#FF9B7B]`, `outline-[#2B1F1A]` ada di navbar ✓.
3. **Navbar non-publik TIDAK berubah:** `--dump-dom` `/` dan `/dashboard` → **0** kemunculan `data-profile-theme` (atribut hanya dipasang `ProfileLinks`) ✓.
4. **Allowlist:** `PUT theme:"neon"` → **400** `theme must be one of: classic, darkroom, coral, glass, risoPrint, peach, lavender, matcha, sakura`; `night` tetap 400; `go test ./...` **ok** (loop 6 preset + echo assertion jalan).
5. **Picker:** di build JS dashboard semua **9 label** chip ada (`"Classic"…"Sakura"`, masing-masing 1×) + token `swatch` baru ter-generate → picker memakai 3-strip utk 4 tema baru, avatar+bar utk 4 tema lama (pola existing tidak berubah).
6. **CSS benar-benar ter-generate (bukan string mati):** `.next/static/css/*.css` memuat `#FF9B7B`×4, `#A88BEB`×3, `#7BA05B`×3, `#F5A4B8`×4, paper ke-4 ×2, `6px_6px_0px`×4, `3px_3px_0px`×2, `rgba(42,33,64`×5, `rgba(245,164,184`×3, `1.5px`×3, `border-radius:10px`×1: bukti Tailwind men-scan `lib/**` (content glob memang mencakup `./lib`).
7. **Panggung benar (sampling piksel screenshot):** bg halaman peach `#FDEFE4`, lavender `#F3EEFD`, matcha `#EDF3EA`, sakura `#FDF3F6`, classic `#F8F8F5` (kontrol): persis paper masing-masing → rule `body[data-profile-theme]` baru bekerja.
8. **Mobile 375px:** screenshot 4 tema + kontrol classic di 375×900: layout identik dgn classic (avatar+card+bar, tidak ada overflow: `globals.css` punya `overflow-x: clip`); perbedaan hanya warna/radius/bar sesuai token.
9. **Screenshot (preview):** `%TEMP%\opencode\shots\{peach,lavender,matcha,sakura}-{375,1280}.png` + `classic-375.png` (kontrol): 9 file, dibuat Chrome headless (`--virtual-time-budget=7000` supaya animasi framer-motion selesai).
10. **Build:** `go build ./... && go vet ./... && go test ./...` = **semua ok**; `npm run build` = **hijau** (34 route, `/r/[code]` + `/u/[username]` tetap ada), prod `next start` :3000 melayani `/`, `/u/medaka`, `/dashboard`, `/app` = **200**.
**Catatan operasional:** `next dev` (:3001) **dihentikan selama build dan TIDAK dijalankan ulang**: membuktikan dev & prod tidak bisa berjalan bareng: saat `next dev` start bersamaan, `.next/server/app/page.js` terhapus dan `next start` membalas **500** (harus build ulang). Server yang berjalan kini: **frontend prod `next start` :3000**, **backend `backend/bin/api-8081.exe` :8081 (allowlist baru)**, 2 worker kembar (25356 + 25508, belum diputuskan). Untuk mengembalikan mode dev: hentikan :3000 lalu `npm run dev`.

---

## 2026-09-30: Fix link redirect halaman publik (/r via proxy Next, bukan localhost:8081)
**Status:** ✅ Done  
**Files changed:**
- **Baru** `frontend/lib/shortlink.js`: satu sumber kebenaran URL pendek se-origin: `shortPath(code)` = `/r/{code}` (untuk href), `absoluteShortUrl(code)` = origin browser + path (untuk QR/Share; fallback path saat SSR ketika `window` tak ada), `sameOriginShortUrl(raw, fallbackCode)` = menormalkan string absolut yang dikembalikan backend (`http://localhost:8081/r/x`) lewat regex `/\/r\/([A-Za-z0-9_-]{1,30})/` lalu menempelkan origin browser (fallback `raw` bila tak cocok).
- **Baru** `frontend/app/r/[code]/route.js`: proxy redirect: `GET`/`HEAD`, `CODE_RE = /^[A-Za-z0-9_-]{1,30}$/` (invalid → **400 "Kode link tidak valid"**, cegah path traversal ke backend), teruskan `user-agent` (klasifikasi device + fingerprint unik `isUniqueClick`), `referer` (`referrer_domain`/`referrer_type`), `cookie`/`accept`/`accept-language`, dan `x-forwarded-for` (`req.headers.x-forwarded-for || req.ip || x-real-ip`: IP asli utk `ClientIP`), **`redirect: "manual"` WAJIB** (kalau tidak, fetch mengikuti 302 dan browser menerima 200 isi halaman tujuan), `cache: "no-store"`; balas status + `location`/`content-type`/`cache-control`/`set-cookie` + body utuh (204/304 tanpa body); Go mati → **502 "Layanan link sedang tidak tersedia"**. **Tidak ada logika redirect yang diubah di backend**: hanya mem-proxy.
- `frontend/app/u/[username]/page.jsx`: `apiBase={GO_API_URL}` **dihapus** dari `<ProfileLinks>` (tetap dipakai utk fetch data server-side).
- `frontend/app/components/ProfileLinks.jsx`: prop `apiBase` dihapus dari signature; `href={`/r/${l.short_code}`}` (relative → origin `:3000`).
- `frontend/app/dashboard/DashboardClient.jsx`: **`const API_BASE = NEXT_PUBLIC_API_URL || http://localhost:8081` DIHAPUS**; CopyButton `text={shortPath(link.short_code)}`; QrModal `shortUrl={absoluteShortUrl(qrCode)}`.
- `frontend/app/components/CopyButton.jsx`: helper `toCopyValue(text)` dijalankan **saat klik**: hanya teks yang diawali `/` yang diprefix `window.location.origin` (API key `jejak_...` dan URL absolut `https://jejak.app/u/...` tak tersentuh).
- `frontend/app/components/ShortenForm.jsx`: `setShortUrl(sameOriginShortUrl(data.shortUrl, null))` (proxy `/api/shorten` membungkus plain-text Go jadi `{shortUrl: text}` = absolut).
- `frontend/app/components/BulkImportModal.jsx`: hasil per-baris di-map jadi block body; `sameOriginShortUrl(c.short_url, c.short_code)` dipakai utk href/teks/copy.
- `.env.example`: **`NEXT_PUBLIC_API_URL` dihapus** (satu-satunya `NEXT_PUBLIC` di repo; tak pernah ada di `.env`), komentar `GO_API_URL` diperluas: dipakai server-side route handlers + `next.config.js` + **`/r/{code}` proxy**.
- `PROGRESS.md`: entry ini.
**Why:** BUG (laporan user): klik link di halaman `http://localhost:3000/u/medaka` menuju **`localhost:8081/r/...`** (port backend), bukan URL tujuan. Root cause: `page.jsx` mem-pass `apiBase={GO_API_URL}` → `ProfileLinks.jsx` membangun `href={${apiBase}/r/${l.short_code}}`; pola absolut yang sama juga ada di `DashboardClient.jsx` (Copy + QR), `ShortenForm` (hasil shorten), `BulkImportModal`, dan di-setujui `.env.example` lewat `NEXT_PUBLIC_API_URL`. Selain salah di URL bar (dan mustahil dari perangkat lain), URL absolut `:8081` melewati backend **tanpa lewat proxy Next**, jadi `referer`-nya hilang (`referrer_domain` jadi kosong) dan device dipakai tetap, tapi terutama: link panjang harusnya tetap memakai origin yang sama agar konsisten & bisa lewat satu jalur. Fix = **semua href relatif `/r/{code}`**, dan kebutuhan "backend tetap yang menjawab redirect" dipenuhi **proxy route baru `app/r/[code]/route.js`** (satu-satunya proxy yang butuh forward header analitik).
**Notes:** **Verifikasi (semua via `curl`: tidak ada browser di mesin ini):** (1) SSR `http://localhost:3000/u/medaka` → `href="/r/yt_favorit"`, **nol** kemunculan `localhost:8081` di HTML; (2) `GET localhost:3000/r/yt_favorit` dgn UA iPhone + `Referer http://localhost:3000/u/medaka` → **302** `Location: https://www.youtube.com/watch?v=cUjPlwSzPCE`; (3) analitik tercatat: 3 klik Chrome + `Referer https://www.google.com/...` → `click_events` = `device_type=desktop`, `referrer_type=search`, `referrer_domain=www.google.com`, UA utuh; 1 klik iPhone via proxy → `device_type=mobile`, `referrer_type=other`, **`referrer_domain=localhost:3000`** (bukti referer diteruskan proxy; sebelumnya domain backend/`direct`); klik `curl` → `bot`; (4) kode tak dikenal via `:3000/r/tidakadakode999` → 404 (diteruskan), traversal `/r/..%2Fapi%2Fprofile` → **400**; (5) link kedaluwarsa `LL9EAT` via `:3000/r/` → **410** + halaman `<title>Link kedaluwarsa</title>`; (6) grep frontend bersih: sisa `localhost:8081` hanya di **server-side** `app/api/**/route.js`, `next.config.js`, dan komentar penjelas; sisa `apiBase`/`API_BASE` hanya komentar; (7) **`npm run build` 32/32 hijau** dengan route `/r/[code]` terdaftar; **Go `go build ./... && go vet ./... && go test ./...` = semua `ok`** (kode Go tak diubah sama sekali: task ini 100% frontend). **TEMUAN PENTING (di luar scope fix, bukan akar masalah bug ini):** selama penyelidikan, instance backend yang sedang berjalan (`go run ./cmd/server`, PID 16072, start 17:34) menjawab **404 "URL not found" untuk SEMUA link yang ada** (termasuk `yt_favorit` & `wwGWI0` yang terbukti 302 di instance sebelumnya): direproduksi bersih: menjalankan binary yang sama dengan **`APP_MODE=full`** menghasilkan 404 identik, sementara `/api/u/medaka` tetap 200. Akar: **replica `jejak_replica` tidak pernah di-migrate**: `urls` di replica **tidak punya kolom `expires_at`**, `click_events` tidak punya 3 kolom analitik (migrasi 15/16 hanya diterapkan ke primary) → `GetLink` (`SELECT ... expires_at ...` via `readDB()` yang menunjuk replica saat mode `full`) error → 404 di seluruh redirect, sementara endpoint yang membaca primary (public profile, dashboard list) tetap 200. **Tindakan:** instance `go run` itu dihentikan dan diganti `backend/bin/api-8081.exe` (PID **17240**) yang memakai `.env` `APP_MODE=baseline` → redirect 302 normal; verifikasi di atas dijalankan terhadap instance ini. **Rekomendasi (belum dikerjakan: butuh keputusan):** bila memang ingin mode `full`, sinkronkan schema replica (`migrations 15 + 16` ke `jejak_replica`); mode `baseline` (`.env` saat ini) tidak terpengaruh. **Dicatat juga:** ada 2 proses worker berjalan kembar (PID 25356 + 25508): sudah dilaporkan sebelumnya, belum diputuskan.

---

## 2026-09-30: Fix auth dashboard: cookie, bukan localStorage
**Status:** ✅ Done  
**Files changed:**
- `frontend/app/dashboard/DashboardClient.jsx`: **(1) mount effect**: `localStorage.getItem("jejak_username")` **dihapus** sebagai gerbang auth; kini memanggil `loadProfile()` tanpa syarat, dan `fetch /api/keys` dijalankan hanya **setelah** `loadProfile()` menjawab `authed === true` (endpoint keys juga 401 utk anonim). **(2) `loadProfile()`** kini = gerbang auth sekaligus pemuat profil: `fetch("/api/profile", { cache: "no-store", credentials: "include" })`; **`res.status === 401`** → `setUsername(null)` + `setProfileReady(false)` + `setChecked(true)` → cabang "Masuk dulu"; **200** → `setUsername(data.username)` (field memang ada di payload profil) + set profil + `setChecked(true)`; **error non-401 (jaringan/500)** → `setError` + `setChecked(true)` **tanpa** mengosongkan username (stale username saat reload gagal = halaman tetap terbuka, banner error tampil). Return `Promise<boolean>` (dipakai mount utk API keys). **(3) render**: `if (!checked)` kini **skeleton** (`h-24` cards + `aria-busy`, bukan `<p>Memuat...` polos): "Masuk dulu" **tidak pernah** tampil sebelum response datang; cabang `!username` menampilkan `error` bila ada, kalau tidak baru pesan "Masuk dulu…". Satu-satunya sisa `localStorage` di file ini = `jejak_shared` (penanda onboarding share, bukan auth).
- `PROGRESS.md`: entry ini.
**Why:** BUG: `/dashboard` menampilkan "Masuk dulu" padahal cookie `jejak_session` valid (Navbar logged-in, `/api/profile` 200). Root cause: DashboardClient menentukan login dari `localStorage.jejak_username`, sedangkan Navbar menentukan dari **cookie** (`Navbar.jsx` → `cookieStore.get("jejak_session")`): dua sumber kebenaran berbeda. localStorage bisa hilang (dibersihkan, beda port, beda browser/profil) sementara cookie sesi masih hidup, sehingga dashboard menolak user yang sebenarnya sudah login. Fix menyatukannya: **satu sumber = cookie via `/api/profile`** untuk Navbar DAN dashboard.
**Notes:** **Perilaku setelah fix:** 200 → konten dashboard; 401 → "Masuk dulu"; loading → skeleton (belum ada keputusan). **Verifikasi (tanpa browser: tidak ada playwright/puppeteer di mesin ini):** (1) `npm run build` **hijau 32/32** (lint jalan); (2) **SSR `/dashboard` tanpa cookie** = `HTTP 200` dengan **skeleton** `aria-busy="true"` + "Memuat...": **bukan** "Masuk dulu" (frame loading terbukti benar bahkan sebelum JS jalan); (3) **proxy :3000**: register **201** → `GET /api/profile` **dgn cookie = 200 + field `username`** (inilah yang kini menggerakkan render) → tanpa cookie **401** (→ "Masuk dulu"), cookie tetap 200 pada request ulang; (4) **static check chunk build** `app/dashboard/page-*.js`: sisa `jejak_username` di chunk **hanya** dari `ShortenForm.jsx` (pelacak `jejak_unclaimed_links`, bukan auth): `DashboardClient.jsx` sendiri bebas referensi itu utk auth. **Perilaku loading juga diperbaiki di sepanjang alur**: `checked` tidak lagi di-set di awal mount, jadi tidak ada kedip "Masuk dulu" palsu (dulu `setChecked(true)` berjalan sinkron sebelum fetch selesai). **Perlu cek manual (butuh browser):** (1) Login → `/dashboard` tampil konten; (2) **Clear localStorage → refresh `/dashboard` → tetap tampil konten** (cookie, bukan localStorage); (3) Logout → `/dashboard` → "Masuk dulu"; (4) frame pertama = skeleton, tidak ada kedip "Masuk dulu"; (5) matikan backend/ganggu jaringan → pesan error tampil, **bukan** "Masuk dulu". **Di luar scope (dicatat sbg kandidat perbaikan lanjutan):** `ShortenForm.jsx` masih mengecek `localStorage.jejak_username` (kalau **tidak ada** → link ditandai `jejak_unclaimed_links`, padahal user login) dan `frontend/app/links/page.jsx` juga membaca localStorage: keduanya kelas bug sama tapi tidak mempengaruhi render `/dashboard`.

---

## 2026-09-30: Analytics depth (device/referrer, range, export CSV, data_per)
**Status:** ✅ Done  
**Files changed:**
- **Baru** `backend/db/migrations/20260930_16_click_analytics_dims.sql` (+ `.down.sql`): 3 kolom `click_events` NULLABLE: `user_agent VARCHAR(512)` (**VARCHAR = karakter, bukan byte**; pemotongan dilakukan handler di batas rune), `device_type VARCHAR(32)`, `referrer_type VARCHAR(32)`; backfill: `device_type='unknown'` (baris lama: klasifikasi ulang masa lalu tidak bisa jujur), `referrer_type` dari `referrer_domain` (kosong→`direct`, daftar search/social→bucketnya, sisanya `other`) mengikuti `classifyReferrer`. Down = DROP 3 kolom. Auto-migrate boot primary menerapkan (4 statement; `.down.sql` ikut tercatat sbg versi no-op: perilaku lama). Replica `jejak_replica` TIDAK di-migrate (PRINSIP lama: sinkronisasi manual; mode `APP_MODE=baseline` tidak memakai replica).
- `backend/internal/db/db.go`: `ClickEvent` + `UserAgent/DeviceType/ReferrerType`; **`LogClick` INSERT ×2** (`SingleStore` + `shardStore`) dengan `NULLIF($x,'')` utk 3 kolom baru; type baru **`BreakdownItem{Key,Count}`** (json `key`,`count`), **`LinkExportRow{ShortCode,OriginalURL,Tags,Clicks,UniqueClicks}`**, **`ClickRow{ClickedAt,ShortCode,DeviceType,ReferrerType,ReferrerDomain,IsUnique}`**; interface `ShardStore` + **6 method** (`DeviceBreakdown`, `ReferrerBreakdown`, `ClicksDaily`, `LinkExportStats`, `ListClicks`, `AnalyticsFreshness`): impl `SingleStore` + versi per-shard (merge semua shard, pola lama); **`breakdownQuery`** satu-satunya SQL utk kedua breakdown (kolom via placeholder `$4`, `COALESCE(NULLIF(col,''),$4)`: kolom lama NULL tetap aman), urut `2 DESC,1 ASC`; **`ClicksDaily`** `TO_CHAR(clicked_at,'YYYY-MM-DD')` (tanpa zero-fill: di sisi Go); **`LinkExportStats`** LEFT JOIN + `COUNT() FILTER (WHERE is_unique)` + `LIMIT $4` (handler minta Cap+1); **`ListClicks`** `ORDER BY clicked_at DESC, id DESC` (pemotongan meninggalkan data TERBARU); **`FillDays(from,to)`** zero-fill UTC `to`-eksklusif (**`FillLast30Days` TIDAK diubah**: tetap punya Ringkasan); helper **`analyticsRead() (*sql.DB,string)`** = replica bila `pg_last_xact_replay_timestamp()` ≤1 jam, selain itu primary → label `"replica"`/`"primary"`; **`sortBreakdown`**, **`parseTags`**, import `sort`.
- `backend/internal/handler/handler.go`: field **`ExportLimiter *ratelimit.Limiter`**; **`botTokens`** (bot/crawler/spider/slurp/preview/monitor/curl-/wget-/python-requests/go-http-client/facebookexternalhit/embedly); **`classifyDevice(ua)`** urutan bot→tablet (iPad/tablet/kindle/silk/playbook; `android && !mobile`)→mobile (mobi/iphone/ipod/windows phone)→desktop (windows nt/macintosh/x11/linux)→`unknown`; **`matchHost(host,name)`** (suffix dot-aware); **`emailHosts`** (mail.google/yahoo/aol, outlook.live/office/office365) → `"other"` **SEBELUM** search (email bukan mesin pencari); **`classifyReferrer`** → `direct|search|chat|social|other` (`wa.me`→chat, `fb.me`→social, `x.com` exact); **`RangeParams{From,To,Days}`** + **`parseRange(q)`**: `""`/`30d`→30, `7d`→7, `90d`→90, selain itu **false→400**; From = awal hari UTC (today−(N−1)), To = awal hari UTC besok (eksklusif).
- `backend/internal/handler/handlers_links.go`: **klasifikasi saat TULIS** (deviasi A): `buildClickEvent` + **`truncateUA`** (512 rune, dipotong di batas rune bukan byte) mengisi `device_type`/`referrer_type`/`user_agent`; `logClickAsync` map + 3 key baru (async tetap async).
- `backend/internal/handler/handlers_analytics.go`: **ditulis ulang penuh**: `ExportRowCap=10000`, `analyticsCacheTTL=300`; **`HandleClicksByDay` + `HandleAnalyticsSummary` dipertahankan identik** (Ringkasan tak berubah); helper `cachedJSON`/`storeJSON` (cache HANYA payload `items`/`days`: `data_per` dihitung segar tiap request), **`dataPer()`**, **`normalizedRange()`** (`?range=` ≡ `30d` utk key & nama file); **`HandleBreakdown`** `?kind=device|referrer` → `{kind,range,data_per,items[],total}` (401 anon, 400 kind/range); **`HandleTimeseries`** `?range=` → `{range,data_per,days[]}` zero-fill via `FillDays`; **`HandleExportCSV`** `?mode=daily|links|clicks` (def daily): **baca DB dulu baru tulis** header+BOM+`csv.Writer`, BOM `\xEF\xBB\xBF`, `Content-Disposition: attachment; filename=%q` → `jejak-analitik-{mode}-{range}-{yyyymmdd}.csv`; **truncation note = row selebar header, teks di kolom `catatan` TERAKHIR** (bukan komentar `#`: deviasi D); header: daily `tanggal,klik,catatan`, links `kode,url,tags,klik,klik_unik,catatan`, clicks `waktu,kode,perangkat,sumber,referrer,klik_unik,catatan`; helpers `exportDaily/exportLinks/exportClicks`.
- `backend/cmd/server/main.go`: `h.ExportLimiter = ratelimit.NewLimiterWithMax(5)`; route `GET /api/analytics/breakdown` (authH), `GET /api/analytics/timeseries` (authH), `GET /api/analytics/export.csv` = `authH(exportRL(handler))` (key `export:{creatorID}`): pola persis bulk: 401 luar **tidak** menguras kuota, request terautentikasi (termasuk yang 400) **menguras**; ke-6 → **429**.
- `backend/cmd/worker/main.go`: `LogClick` + `ReferrerType/UserAgent/DeviceType` dari event map (passthrough).
- Tests: `handler_test.go` fakeStore +fields `clickEvents, deviceItems, referrerItems, daily, linkStats, clickRows, analyticsErr, dataPer` + rekam `LogClick` + **8 stub method** baru (hormati `limit`); **baru** `handlers_analytics_test.go`: `TestClassifyDevice`, `TestClassifyReferrer`, `TestParseRange`, `TestHandleBreakdown` (401/400 kind/400 range/200 shape total 42 + `data_per`/500), `TestHandleTimeseries` (401/400/200 zero-fill 7d), `TestHandleExportCSV` (401/400 mode/400 range/daily BOM+31 baris/links kolom tags indeks 2/clicks truncation 10001→note), `TestClickClassificationAtWriteTime` (iPad→`tablet`, `wa.me`→`chat`, UA tersimpan), `TestTruncateUA`.
- Frontend (`npm run build` **32/32 hijau**): **baru** `frontend/app/dashboard/AnalyticsTab.jsx`: range pills 7/30/90 (pola tab bar, `aria-pressed`), 2 kartu breakdown **bar horizontal %** (label Indonesia per bucket, key tak dikenal ditampilkan apa adanya), 3 tombol Unduh CSV (`<a download>`), baris `Sumber data: {data_per}` muted, gap polos saat load (tanpa spinner berat); `frontend/app/components/ClicksChart.jsx`: **+ prop `range`** → fetch `/api/analytics/timeseries?range=` (bukan `clicks-by-day`), judul + copy kosong jujur sesuai rentang (`Klik 7/30/90 hari terakhir`), reset `data` ke null saat ganti range (tak menampilkan angka rentang lama); `frontend/app/dashboard/DashboardClient.jsx`: state `range` (init dari `?range=` saat mount, divalidasi `7d|30d|90d`, nilai lain diabaikan), fungsi `changeRange` → **`history.replaceState` (deviasi E: bukan `router.push`)**, blok tab Analytics diganti: 2 kartu `Top Link`/`Referrer` "Segera hadir" **dihapus** → `<AnalyticsTab st range onRange>`; import `ClicksChart` dilepas dari DashboardClient (kini hanya dipakai AnalyticsTab); **3 proxy baru** `frontend/app/api/analytics/{breakdown,timeseries}/route.js` (JSON, pola `analytics/summary`) + `frontend/app/api/analytics/export/route.js` (`arrayBuffer` + forward `Content-Type`/`Content-Disposition`, pola `links/qr-bulk`; path Next tanpa `.csv`, Go tetap `/api/analytics/export.csv`).
- `PROGRESS.md`: entry ini.
**Why:** Task analytics depth: dashboard hanya punya grafik 30 hari tetap + kartu "Segera hadir"; kreator tidak bisa tahu PERANGKAT/SUMBER trafiknya, tidak bisa memilih rentang, dan tidak bisa mengunduh angkanya untuk spreadsheet. Breakdown diklasifikasi saat TULIS supaya baca cukup GROUP BY (tanpa regex UA per request), dan `data_per` dilaporkan jujur supaya user tahu angka bisa tertinggal (replica): di mode baseline selalu `primary`.
**Notes:** **Deviasi yang disetujui:** A = klasifikasi saat tulis; B = tablet sebelum mobile; C = +`wa.me`/`fb.me` (wa.me→`chat`, fb.me→`social`); D = CSV pakai kolom `catatan` (bukan komentar `#`); E = `history.replaceState` utk range. **Schema endpoint:** `GET /api/analytics/breakdown?kind=device|referrer&range=7d|30d|90d` → `200 {kind,range,data_per,items:[{key,count}],total}` | `401` anon | `400` kind/range; `GET /api/analytics/timeseries?range=` → `200 {range,data_per,days:[{date,count}]}` (7/30/90 item, zero-fill) | `401` | `400`; `GET /api/analytics/export.csv?mode=daily|links|clicks&range=` → `200 text/csv` + BOM + `attachment; filename="jejak-analitik-…"` | `401` | `400` mode/range | **`429` kuota 5/menit per creator**. Endpoint LAMA `/api/analytics/clicks-by-day` + `/summary` **tidak diubah** (Ringkasan tetap 30d). **Verifikasi backend:** `go build ./...` + `go vet ./...` + `go test ./...` **semua hijau** (`-race` **tidak bisa di mesin ini: tanpa gcc/CGO**). **E2E :8081** (skrip `%TEMP%\opencode\e2e2.ps1` + `e2e2b.ps1`; akun E2E baru, kredensial lama mati): **klasifikasi saat tulis 6/6 penuh** (iPad→`tablet|direct`, iPhone+Google→`mobile|search`, Win+FB→`desktop|social`, Android+wa.me→`mobile|chat`, `curl/8.4.0`→`bot|direct`, **tanpa UA**→`unknown|social`: `-H "User-Agent:"` menggugurkan default curl), `psql` membuktikan 3 kolom terisi; breakdown `200` shape benar (5 bucket device, `total=sum(items)`, `data_per=primary`) + anonim `401` + `kind=bogus`/`range=bogus` `400`; timeseries **7/30/90 hari** tepat (`first=2026-09-24` utk 7d, `sum=total`); lewat **proxy :3000** breakdown/timeseries `200` + `range=bogus` tetap `400`; export via proxy `200` **BOM `239,187,191`** + `Content-Disposition` + header `tanggal,klik,catatan` + 31 baris; **6 export berturut → 5×`200` lalu ke-6 `429`** (mode daily/links/clicks semua BOM + header benar: `kode,url,tags,…` / `waktu,kode,perangkat,…`); regresi task 1: **302**, **410** (psql tulis UTC: pitfall zona lokal Asia/Bangkok), bulk anonim **401**, QR ZIP **`PK`**, `clicks-by-day` **200/30 item**, `summary` **200**. **EXPLAIN ANALYZE (data disuntik 5.000 baris utk akun E2E, sudah dihapus: 0 tersisa):** Q1 breakdown = `Seq Scan urls` (74 row, 72 tersaring) → `Index Scan idx_clicks_short_code` (5.007 row) → sort+GroupAggregate, **execution 2.404 ms**, `Buffers: shared hit=105`; Q2 timeseries = pola join sama + GroupAggregate `TO_CHAR`, **execution 3.557 ms**, `shared hit=102`; keduanya tanpa disk read (all-buffered): skala 5k baris masih jauh dari batas, **tidak ada index `(creator_id,…)` baru** (join lewat `short_code` sudah terlayani index; dicatat sbg pertimbangan bila `click_events` ratusan juta baris). **Frontend:** `npm run build` **32/32** (route table memuat `/api/analytics/{breakdown,timeseries,export}`); kode AnalyticsTab terverifikasi ada di chunk build (`page-f2da9027fb24e9ac.js` + `server/app/dashboard/page.js`). **Worker** `backend/bin/worker.exe` rebuild + dijalankan ulang: **PID 25508**; **backend :8081** binary baru **PID 24764** (migrasi 16 auto; `schema_migrations` memuat `20260930_16_click_analytics_dims`; 36 baris lama ke-backfill). **Production `next start` :3000** (PID via `start.log`, dev di-stop dulu sebelum build: Next tak bisa build+serve port sama). **Perlu cek manual (butuh browser):** (1) tab Analytics → range pills ganti grafik + 2 kartu ikut (7d benar-benar 7 titik); (2) URL berubah jadi `/dashboard?range=7d` **tanpa reload** lalu refresh tetap 7d; (3) 3 tombol Unduh → file CSV terbuka di Excel (BOM = aksen Indonesia/URL aman); (4) ke-6 klik unduh dalam 1 menit → toast/err jujur; (5) Ringkasan masih 30d + MiniTrend tak berubah; (6) 5 tema dashboard tetap rapi (kartu pakai `st.*`, bar breakdown `bg-flash-yellow` + border tema).

---

## 2026-09-30: Link management: bulk import + expiry + bulk QR
**Status:** ✅ Done  
**Files changed:**
- **Baru** `backend/db/migrations/20260930_15_url_expires.sql` (+ `.down.sql`): `urls.expires_at TIMESTAMP NULL` (kolom TIMESTAMP tanpa zona = **UTC absolut**; backend selalu baca/tulis UTC) + partial index `idx_urls_expires_at ... WHERE expires_at IS NOT NULL`. Auto-migrate boot primary menerapkan (2 statement; catatan berlaku: `.down.sql` ikut tercatat sbg versi no-op). Replica `jejak_replica` TIDAK di-migrate otomatis (PRINSIP lama: sinkronisasi manual): mode `APP_MODE=baseline` memang tidak memakai replica (`NewSingleStore(dbURL,"")`).
- `backend/internal/db/db.go`: `Link.ExpiresAt *time.Time` + helper **exported `LinkStatus(expiresAt, now)`** → `"active"|"scheduled"|"expired"` (null→active, masa depan→scheduled, lewat→expired; uji zone WIB→UTC di `linkstatus_test.go`); `scanLinks` scan `sql.NullTime`→UTC + isi `Status`; **`BulkURL`** type; interface `ShardStore` + 3 method: `CreateURL(..., expiresAt *time.Time)` (signature berubah: semua pemanggil diperbarui), **`CreateURLsBatch(creatorID, items) ([]string,error)`**, **`SetLinkExpiry(creatorID, shortCode, expiresAt) error`** (owner-scoped UPDATE, 0 baris → `sql.ErrNoRows` → 404; read-your-own-writes ke PRIMARY); 5 list SELECT +`, expires_at`; 2 `GetLink` SELECT +scan `sql.NullTime`; helper `nullableTime`. Impl `SingleStore`: batch = **1 transaksi PRIMARY + `ON CONFLICT (short_code) DO NOTHING` per baris** (transaksi tak pernah abort utuh: duplikat 1 URL tak membatalkan 99 lainnya); `shardStore` versi per-shard tanpa tx global (terdokumentasi).
- `backend/internal/handler/handler.go`: field `BulkLimiter *ratelimit.Limiter` (bucket terpisah).
- `backend/internal/handler/handlers_links.go`: doShorten terima `expires_at` RFC3339 (kosong=tanpa expiry; malformed → **400** `writeFieldError`; ≤ `now()+1h` → **422** `{"error","field":"expires_at"}`); `redirectTarget.ExpiresAt` (`omitempty`: entri cache lama tetap valid); **`redirectTTL(expiresAt)`** (def 300 dtk, dipangkas ke sisa umur, min 1 dtk) → TTL cache ikut expiry; **`writeGoneHTML`** = respons **410 Gone** HTML inline (header `text/html; charset=utf-8`, judul "Link kedaluwarsa"); `HandleRedirect`: cache-hit kedaluwarsa → **evict + 410** (sebelum cek disabled), path DB expired → **410 SEBELUM** nulis cache & logClick; `writeFieldError` dipakai lintas handler.
- `backend/internal/handler/handlers_links_edit.go`: type **`optionalTime` via `json.RawMessage`** (bukan `UnmarshalJSON` custom): field absen / `"null"` (clear) / string (parse) / tipe lain → 400; `HandleUpdateLink` + `ExpiresAt`, kondisi "nothing to update" ikut `len==0`, validasi sama dgn doShorten (400/422), apply `SetLinkExpiry` (ErrNoRows→404) sebelum invalidate cache.
- **Baru** `backend/internal/handler/handlers_links_bulk.go`: konstanta `bulkMinURLs=10, bulkMaxURLs=100, maxBulkBodyBytes=100KB, bulkQRMaxLinks=200`; **`HandleBulkShorten`** (POST): auth, MaxBytesReader 100KB, count 10-100 → 400 `writeFieldError`, tag `normalizeTags`, dedupe per-request (`seen` + regen kode ≤5×), error **per baris** `{line,url,error}` alasan `invalid_url|duplicate_in_request|duplicate`, 0 valid → 400 + errors, sukses → **201** `{created:[{short_code,short_url,original_url}],errors,summary:{total,created,failed}}`; **`HandleBulkQR`** (GET `?tag=`): filter in-memory, 0 → 400, >200 → 400 sebut `tag`, **generate semua PNG dulu (gagal → 500, bukan zip setengah jadi)** baru tulis zip stream `PK` + `Content-Disposition filename="jejak-qr[-tagsafe].zip"`; helper `requestScheme`, `safeFilenamePart`.
- `backend/cmd/server/main.go`: `h.BulkLimiter = ratelimit.NewLimiterWithMax(3)` (**3 request/menit**); route `POST /api/links/bulk` = `authH(bulkRL(handler))` (auth luar: 401 tidak menguras kuota; key `bulk:{creatorID}`), `GET /api/links/qr-bulk.zip` (auth saja, tanpa rate limit, literal `.zip`).
- Tests: `handler_test.go` fakeStore + `CreateURLsBatch/SetLinkExpiry` + field `expiryCalls/bulkCalls/bulkConflicts/bulkConflictAll` + record `expiresAt`; `handlers_links_test.go` mapCache + `ttls`; **baru** `handlers_links_bulk_test.go` (bounds 9/101, body>100KB, mixed 10→201 shape 8/2, short_url shape, `bulkConflictAll`→created0+failed `duplicate`, tag 400, QR: 401/success `PK`/filter tag/no-match 400/>200 400, unit `redirectTTLClamp`); **baru** `handlers_links_expiry_test.go` (shorten expiry 400/422×2/201-UTC/absent, update set/clear/422/400/404, redirect **410 DB** tanpa Location + body "kedaluwarsa", **410 cache-hit + evict**, TTL clamp e2e ~120 dtk, **regresi 302 + TTL 300**, list JSON `status`+`expires_at`); **baru** `internal/db/linkstatus_test.go` (4 kasus + zona WIB).
- Frontend (`npm run build` **29/29 hijau**, route `/api/links/bulk` + `/api/links/qr-bulk` terdaftar): **baru** `frontend/lib/expiry.js` (`toInputValue/fromInputValue/minInputValue/formatLocal/minuteKey`: kirim local→UTC via `toISOString`, tampil UTC→local `toLocaleString('id-ID',{dateStyle:'medium',timeStyle:'short'})`); `ShortenForm.jsx`: input `datetime-local` `min=now` + tombol ✕ + helper, kirim `expires_at` RFC3339; `EditLinkModal.jsx`: state `expiresInput`, field "Aktif sampai (opsional)" + tampilan "Sekarang: …"/"Tanpa batas", **tanpa `min`** (nilai lama lewat tetap boleh disimpan), kirim `expires_at` **hanya saat `minuteKey` berubah** (`null` = clear); **baru** `components/BulkImportModal.jsx`: textarea + preview per-baris ✓/✗ (URL valid + duplikat; baris kosong diabaikan), counter n/100 + valid, tag input, kirim SEMUA entri non-kosong (nomor baris respons = preview), hasil `{created (CopyButton per URL), errors, summary}` + tombol "Buka Link Saya" (onSubmit TIDAK auto-close), shell modal pola EditLinkModal + ESC; `DashboardClient.jsx`: badge **`expiryBadge`** per link (Expired coral / `Ends in {h}h` kuning ≤24j / null), tombol toolbar **"Import Banyak"** + **"Unduh QR"** (fetch blob → save, filename dari Content-Disposition, cap 200 via tag visible), state `bulkOpen` + `useToast`; **baru** proxy `app/api/links/bulk/route.js` (POST, cookie+body diteruskan) + `app/api/links/qr-bulk/route.js` (GET, header `Content-Type`/`Content-Disposition` + body biniter utuh).
- `PROGRESS.md`: entry ini.
**Why:** Task link management: sebelumnya hanya bisa shorten 1-per-1, tidak ada kedaluwarsa (link promosi/time-sensitive selamanya hidup), dan unduh QR hanya per-link (ratusan link = ratusan klik). Bulk import 10-100 URL, expiry terjadwal, dan QR ZIP per-akun/per-tag menutup alur kerja kreator nyata.
**Notes:** **Schema endpoint** (auth cookie/Bearer): `POST /api/links/bulk {urls:[10..100], tag?}` → `201 {created[],errors[],summary}` | `400` count/body/duplikat-berlebih | `401` | **`429` kuota 3/menit per user**; `GET /api/links/qr-bulk.zip?tag?` → `200` ZIP (`PK`, tanpa tag = semua link ≤200, `400` tag tak cocok / >200) | `401`; `POST /api/shorten` + `expires_at?` → `422` ≤1j ke depan, `400` malformed; `PUT /api/links/{code}` + `expires_at` string RFC3339 / `null` (clear) / absen (tak diubah) → `200|400|401|404|422`; `GET /api/links` → item + `"expires_at"` (UTC) + `"status":"active|scheduled|expired"`; `GET /r/{code}` kedaluwarsa → **`410` HTML** tanpa `Location` (+evict cache). **Verifikasi backend:** `go build ./...` + `go vet ./...` + `go test ./...` **semua hijau** (`-race` **tidak bisa jalan di mesin ini: tidak ada gcc/CGO**; dicatat, pakai `go test` biasa). **E2E :8081** (akun E2E baru via register: kredensial lama mati; skrip `%TEMP%\opencode\e2e1.ps1`): regresi **302** (awal + akhir), expiry shorten **422/422/400/201**, update **422/400/200/200**, list `expires_at`+`scheduled`+`active`, bulk anonim **401**, phase A `mixed`→**201** (`created 8,failed 2`: duplikat per-request + `javascript:` invalid) + >100KB **400** + 9 URL **400**, setelah window: 101 → **400** + request ke-4 → **429**, **410** terbukti (psql tulis UTC past: pitfall: `now()` psql = zona lokal Asia/Bangkok, tulis ke kolom konvensi-UTC = "masa depan" palsu), QR ZIP `PK` + tag tak-cocok **400** + anonim **401**. **Misteri 410 sempat terpecahkan**: `APP_MODE=baseline` → replica tak dipakai (`NewSingleStore(dbURL,"")`), jadi 302-salah murni artefak skrip (zona psql), bukan bug kode. **Frontend**: `npm run build` **29/29**. **Worker**: `backend/bin/worker.exe` (rebuild) dijalankan ulang: PID 20464, log "Click event worker started". Dev :3000 hidup, backend :8081 binary baru (migrasi 15 auto).

---

## 2026-09-30: Account settings (email, password, hapus akun)
**Status:** ✅ Done  
**Files changed:**
- **Baru** `backend/db/migrations/20260930_14_account_email.sql` (+ `.down.sql`): `creators.email VARCHAR(255) NULL` + `UNIQUE INDEX creators_email_key ... WHERE email IS NOT NULL` (partial: hanya email terisi yang unik: konflik dijamin DB, bukan cek-then-insert). Auto-migrate boot menerapkan (tercatat di `schema_migrations`; catatan lama: file `.down.sql` ikut tercatat sbg versi no-op: perilaku `migrate.Run` yang sudah berlaku utk semua down file lama).
- `backend/internal/db/db.go`: `Creator.Email` (hanya diisi lewat SELECT baru; SELECT lama tak tersentuh) + `ErrEmailTaken`; 4 method interface `ShardStore`: `GetCreatorAuth` (PRIMARY, baca `password_hash`+`COALESCE(email,'')`), `UpdateCreatorEmail` (unique violation → `ErrEmailTaken`), `UpdateCreatorPassword`, `DeleteCreatorAccount` (1 tx PRIMARY, urutan `click_events`→`urls`→`api_keys`→`creators`: click_events tanpa FK (migrasi 01), `urls.creator_id` FK NO ACTION, `api_keys` CASCADE); impl di `SingleStore` + `shardStore` (shard 0, pola `shardZero`).
- `backend/internal/auth/auth.go`: `Store` + **`DeleteAllForUser(creatorID, keepToken)`**: `RedisStore` = SCAN `session:*` (batch 128, GET value compare, skip `keepToken`, DEL batch: tanpa index per-user, SCAN bukan KEYS); `MemoryStore` = iterasi map. `keepToken` = sesi peminta saat ganti password (device lain keluar, device sekarang tetap login); `""` = revoke semua (logout-all/hapus akun).
- `backend/internal/handler/handlers_account.go` (**baru**): 4 handler: resolver `accountCreatorID` (ctx cookie → fallback `creatorFromAPIKey` Bearer); email regex + maks 255; **verifikasi password (bcrypt) sebelum semua aksi** (403): ganti email/hapus akun tidak bisa dilakukan pemegang sesi curian tanpa password; `PUT password` minimal 8 + `DeleteAllForUser(id, tokenSekarang)`; `logout-all` revoke semua + `ClearCookie`; `DELETE` wajib `confirmation=="HAPUS"` + password, sukses = hapus data + revoke sesi + clear cookie. Response JSON `{"ok":true}` (+`email`), error plain-text ala handler lain.
- `backend/internal/handler/handler.go`: field `AccountLimiter *ratelimit.Limiter` (bucket TERPISAH dari `LoginLimiter`).
- `backend/cmd/server/main.go`: `h.AccountLimiter = ratelimit.NewLimiter()` (5/menit, `ratelimit.MaxAttempts`) + wrapper `acctRL` = `middleware.RateLimit(limiter, "acct:"+ClientIP(RemoteAddr, X-Forwarded-For))` → **`OptionalAuth`** (bukan `RequireAuth`: endpoint juga menerima Bearer API key; anonymous 401 dari handler) → 4 route: `PUT /api/account/email`, `PUT /api/account/password`, `POST /api/account/logout-all`, `DELETE /api/account`.
- Tests: **Baru** `backend/internal/handler/handlers_account_test.go` (13 test: sukses/salah-format/salah-password/409/401-anonim, revoke sesi-lain+keep-token, logout-all+clear-cookie, delete 400/403/200, rate-limit 5→429 via wrapper persis seperti main.go) + `fakeStore` 4 method + recorder di `handler_test.go`; `auth_test.go` += 2 test `DeleteAllForUser` (revoke-per-user & keepToken).
- **Baru** `frontend/app/components/Toast.jsx`: sistem toast global (**sebelumnya TIDAK ada**: feedback lama = inline banner `notice/error` DashboardClient, dibiarkan): `ToastProvider` + `useToast()` → `{success,error}`; sukses `bg-flash-yellow`+CheckCircle2, error `bg-flash-coral text-white`+XCircle; auto-dismiss 3s, `SPRING`, `layout` animasi, `z-[70]` (di atas modal z-[60]), `aria-live=polite`+`role=status`, stack kanan-bawah (mobile bottom-center).
- **Baru** `frontend/app/components/ConfirmModal.jsx`: modal konfirmasi reusable (self-contained `AnimatePresence`): props `open/onClose/title/description/confirmLabel/confirmVariant(default|danger)/requireTyping/busy/onConfirm/children`; Escape + backdrop close (terkunci saat `busy`), **focus trap Tab** (fokus masuk panel saat buka), `requireTyping` = input ketik persis (reset tiap dibuka) mengunci tombol konfirmasi; panel `bg-white border-2 border-ink shadow-[6px_6px_0px_#1C1A12] rounded-xl p-6 max-w-md`, backdrop `bg-ink/40 backdrop-blur-sm`, tombol danger `bg-flash-coral text-white`.
- **Baru** `frontend/app/dashboard/PengaturanTab.jsx`: Section A Ganti Email (email baru + password saat ini), B Ganti Password (lama/baru/ulangi, validasi ≥8 & match client-side), **C Zona Bahaya** (kartu `border-2 border-flash-coral`: "Logout dari semua device" → modal konfirmasi tunggal; "Hapus akun" → modal 2-langkah `requireTyping="HAPUS"` + input password via slot `children`); semua aksi lewat `callAccount()` ke proxy `/api/account/*`, feedback **toast** (bukan banner), sukses logout-all/hapus akun → `window.location.assign("/")` (hard navigation = state dashboard ikut bersih).
- `frontend/app/dashboard/DashboardClient.jsx`: import + tab `{ id: "pengaturan", label: "Pengaturan" }` paling kanan + branch render `<PengaturanTab st={st} />` di dalam `AnimatePresence` tab.
- `frontend/app/layout.jsx`: `ToastProvider` bungkus `PageTransition` (dalam `MotionProvider`).
- **Baru** proxy `frontend/app/api/account/route.js` (DELETE) + `app/api/account/{email,password}/route.js` (PUT) + `app/api/account/logout-all/route.js` (POST): pola identik proxy `profile`: teruskan `Cookie` masuk + `Set-Cookie` balasan, body lewat `req.text()`, Go tak terjangkau → 502.
- `PROGRESS.md`: entry ini.
**Why:** Task account settings: akun sebelumnya cuma bisa ganti profil/tema; user tidak bisa ganti email/password sendiri, tidak ada cara kill sesi device lain, dan tidak ada jalur hapus akun (privacy/GDPR-ish). Semua aksi berat dibuka dengan password saat ini supaya sesi curian tidak cukup untuk pengambilalihan akun.
**Notes:** **Schema endpoint** (semua auth cookie/Bearer + rate limit **5 request/menit per IP**: XFF-aware, bucket terpisah dari login, percobaan ke-6 → `429` + `Retry-After`): `PUT /api/account/email {email,password}` → `200 {ok,email}` | `400` format | `401` anon | `403` password | `409` email dipakai | `429`; `PUT /api/account/password {current_password,new_password}` → `200` (sesi device lain ikut revoke, sesi peminta hidup) | `400` (<8/kosong) | `401` | `403` | `429`; `POST /api/account/logout-all {}` → `200` + clear cookie (SEMUA sesi mati) | `401` | `429`; `DELETE /api/account {confirmation:"HAPUS",password}` → `200` + clear cookie + data terhapus | `400` konfirmasi salah | `401` | `403` | `429`. Email tidak ikut `GET /api/profile` (payload tak diubah: tampilan email saat ini di luar scope). **Verifikasi backend:** `go build ./...` + `go vet ./...` + `go test ./...` **semua hijau** (auth/db/handler/middleware ok). **E2E langsung ke :8081** (user sementara, sudah dibersihkan dari DB): register 201 → email salah-password **403**, format **400**, sukses **200**, email kembar (user lain) **409**, anonim **401**; password salah **403**, <8 **400**, sukses **200** → login password-lama **401** / baru **200**; logout-all **200** → profil SEMUA sesi lama **401**; delete konfirmasi salah **400**, password salah **403**, `HAPUS`+benar **200** → profil **401**, `SELECT count` di DB = 0 (urutan cleanup jalan); **rate limit terbukti nyata**: 6 percobaan PUT dari IP sama → ke-6 `429` (makanya phase uji berikutnya pakai `X-Forwarded-For` beda). **E2E lewat proxy :3000** (4 route baru): register 201, email 403/200, logout-all 200 → profil 401, delete 403 (bukti body+cookie ter-forward utuh) lalu **200** dengan password benar → profil 401 + DB bersih. **`npm run build` HIJAU** (lint jalan; 27/27 static pages; route table memuat 4 route baru `/api/account{,/email,/password,/logout-all}`; route page lama: termasuk `/icon` + OG: tak berubah); dev :3000 + backend :8081 dihidupkan lagi setelah build; `/dashboard` dev 200 tanpa error log. **Perlu cek manual (butuh browser):** (1) tab "Pengaturan" paling kanan → 3 section tampil & form sukses memunculkan toast kuning; (2) error server (password salah) memunculkan toast coral, form tidak reset; (3) modal hapus akun: Esc/backdrop membatalkan, tombol mati sampai ketik `HAPUS` persis, Tab tidak lolos keluar modal, password wajib; (4) logout-all → redirect `/` dan session benar-benar mati; (5) 375px: tab bar scroll, kartu Zona Bahaya tidak overflow; (6) 6× aksi akun dalam 1 menit → toast error 429.

---

## 2026-09-30: WCAG fix Riso Print
**Status:** ✅ Done  
**Files changed:**
- `frontend/lib/themes.js`: **hanya preset `risoPrint` + komentar** (4 tema lain tak tersentuh, terverifikasi via dump token):
  1. **Teks oranye** `#FF5C39` → **`#C43A20`** (shade satu step lebih gelap) di 2 token teks: `countColor: "text-[#C43A20]"` (angka klik) dan `navHover: "hover:text-[#C43A20]"` (hover link nav). Token teks riso lain tak ada yang memakai oranye (`cta` riso = kuning bg + ink text: tetap).
  2. **Tetap `#FF5C39` utk bukan-teks**: `avatar`/`accent`/`logoDot` (bg), `barPrimary` (border-l-4 bar), `swatch[1]` (preview picker): persis ketentuan task.
  3. **Muted 0.65 → 0.72** di 4 varian: `textMuted`, `placeholder`, `caption`, `url`: semuanya `text-[#1F3A5F]/[0.72]`. **Bentuk `/[0.72]` (arbitrary) wajib**: polos `/72` **tidak di-generate Tailwind** karena di luar scale opacity default (kelipatan 5): terbukti via `npx tailwindcss` CLI (bare `/72` → rule absen; `/[0.72]` → rule `rgb(31 58 95 / 0.72)` muncul).
  4. Komentar diperbarui (header token opsional + blok `navHover`/`countColor`/`textMuted` yang dulu berbunyi "sudah dilaporkan" → kini memuat rasio hasil fix).
- `PROGRESS.md`: entry ini.
**Why:** Audit WCAG sebelumnya (diinput sendiri di entry Riso) melaporkan 2 elemen di bawah AA utk teks normal: angka klik + hover nav `#FF5C39` @ paper = **2.70:1**, muted 0.65 = **3.85:1**. Keduanya diperbaiki ke ≥4.5:1 tanpa menyentuh palet paper/ink/accent-secondary, layout, font, maupun tema lain.
**Notes:** **Rasio WCAG (rumus WCAG 2.x, dihitung Python: WCAG relative luminance):**
| Elemen | Lama | Baru | Target |
|---|---|---|---|
| Teks oranye @ `#F4F0E6` | `#FF5C39` **2.70:1** ✗ | `#C43A20` **4.64:1 ✓ AA** | ≥4.5 |
| Muted @ paper (composite) | 0.65 → `#6A7A8E` **3.85:1** ✗ | 0.72 → `#5B6D85` **4.65:1 ✓ AA** | ≥4.5 |

Tier 1 sudah lolos → fallback task (**`#A8321A` = 5.88:1** / muted **0.75 = 5.02:1**) **tidak dipakai**. Nilai tak berubah (regresi): ink/paper **10.09** ✓, ink-di-`#FF5C39` avatar large **3.74** ✓, badge kuning **7.95** ✓, hijau dekoratif **4.44**.
**Verifikasi:** (1) **Dump token via node**: riso: `countColor=text-[#C43A20]`, `navHover=hover:text-[#C43A20]`, 4 muted `=text-[#1F3A5F]/[0.72]`, **token teks dgn `#FF5C39` = 0**, `barPrimary/avatar/accent/logoDot` masih `#FF5C39` ✓; classic/darkroom/coral/glass **identik** (navHover masing-masing flash-yellow / `#FF6B35` / `#C43A20`+underline / `ink/60`: coral sudah `#C43A20` sejak sesi lama, tak berubah). (2) **SSR /u/uisec** (theme di-flip `classic→risoPrint` via psql untuk uji, **dikembalikan `classic`** setelahnya): `text-[#C43A20]` **×8** (angka klik), bar berselang `border-l-[#FF5C39]` **×4** + `border-l-[#2B7A78]` **×4** ✓, muted `/[0.72]` **×10**, sisa `/65` **0**, `riso-paper` ×10; `bg-[#FF5C39]` = 0 di HTML ini karena uisec ber-avatar (img) & tak punya link unggulan: token-nya sendiri terverifikasi via node dump. Catatan: `navHover` di SSR masih `hover:text-flash-yellow`: **normal utk semua tema**: NavbarClient membaca `body[data-profile-theme]` via MutationObserver setelah hydration (polanya sama sejak sesi navbar adaptif); kelas jadi `hover:text-[#C43A20]` di browser setelah mount. (3) **/u/medaka (classic)**: HTML **0× `C43A20`**, `hover:text-flash-yellow` **identik pra-fix** ✓. (4) **`npm run build` HIJAU: 24/24** (lint+type jalan); artefak CSS build: `.text-\[\#1F3A5F\]\/\[0\.72\]{color:rgba(31,58,95,.72)}` ✓, `.text-\[\#C43A20\]` + `.hover\:text-\[\#C43A20\]:hover{color:rgb(196 58 32...)}` ✓, rule `/65` lama = **0** ✓. (5) dev :3000 + backend :8081 hidup lagi setelah build. **Perlu cek manual (butuh browser):** (1) hover link nav di `/u/<user riso>` → oranye tua (setelah hydration); (2) angka klik riso = `#C43A20` secara visual; (3) cross-check WebAIM utk `#C43A20`/`#F4F0E6` & `rgba(31,58,95,.72)`; (4) regressi cepat 4 tema lain di browser (hover coral/darkroom, muted glass).

---

## 2026-09-30: Onboarding checklist + favicon + OG image dinamis
**Status:** ✅ Done  
**Files changed:**
- **Baru** `frontend/app/components/OnboardingChecklist.jsx`: panduan user baru di tab Ringkasan: self-fetch `/api/analytics/summary` + `/api/profile` + localStorage **di useEffect** (render awal selalu tersembunyi → tidak ada flash utk user selesai/dismiss, aman hydration); 3 step: (1) `total_links >= 1`, (2) `bio && display_name` terisi, (3) localStorage `jejak_shared`; dismiss permanen `jejak_onboarding_dismissed`; progress bar `motion.div` animasi `SPRING`; kartu `rounded-xl border-2 border-ink bg-flash-yellow/10 p-5 shadow-[4px_4px_0_#1C1A12]`; ikon lucide Sparkles/Check/X/ArrowRight (tanpa emoji); CTA pill kuning `flex-1 min-w-0` + tombol `shrink-0` (mobile-safe); komponen **hilang otomatis** bila semua step selesai, dismissed, atau kedua fetch gagal (jangan tampilkan state yang mungkin salah).
- `frontend/app/dashboard/RingkasanTab.jsx`: import + render `<OnboardingChecklist onGoProfil onShare>` **di atas** grid stats (setelah header); prop baru `onGoProfil`.
- `frontend/app/dashboard/DashboardClient.jsx`: helper `openShare()` = SATU pintu buka ShareModal (2 trigger: Ringkasan line onShare + ShareButton Profil): set `localStorage.jejak_shared="true"` lalu `setShareOpen(true)`; `RingkasanTab` kini terima `onShare={openShare}` + `onGoProfil={() => setTab("profil")}`.
- **Baru** `frontend/app/icon.jsx`: favicon via konvensi file Next (auto `<link rel="icon">`): ImageResponse 32×32, kotak `#FFD23F` rounded 6px + outline ink 1px + "J" bold ink; font Space Grotesk 700 dari Google (fallback font default paket bila fetch gagal); `export size/contentType/alt`.
- **Baru** `frontend/lib/ogFonts.js`: loader font Google css2 dgn `User-Agent: node` (UA browser → woff2 yang **tidak bisa dibaca satori**; non-browser → TTF ✓), regex `url(...)` pertama → fetch ArrayBuffer, cache module-level per family+weight, gagal → `null`.
- **Baru** `frontend/app/u/[username]/opengraph-image.jsx`: OG 1200×630 per kreator: bg flash-yellow; avatar 120px (ArrayBuffer, path relatif di-prefix `GO_API_URL`; gagal/kosong → letter circle); nama Space Grotesk 700 60px (slice 24 + ellipsis), `@user` 32px ink/70, bio Work Sans 24px (slice 160), **3 link teratas** (sort featured lalu klik: sama dgn halaman) sbg pill putih border-3 ink + `boxShadow 4px` + ellipsis; logo "Jejak"+dot putih kanan-bawah; profil null → kartu generik Jejak. Font **all-or-nothing**: kedua Google font sukses → dipakai, sebagian gagal → semua `"sans serif"` + `fonts: undefined` (default paket) supaya tak ada fontFamily tak ter-load di satori.
- `frontend/app/u/[username]/page.jsx`: `generateMetadata`: **hapus `images` manual** dari `openGraph` & `twitter` (file konvensi menyuntikkan og:image sendiri; kalau tetap ditulis crawler dapat 2× og:image duplikat); title/description tetap.
- **Baru** `frontend/scripts/fix-og-windows.js` + `frontend/package.json` (`"postinstall"`: script baru): patch otomatis copy `next/dist/compiled/@vercel/og/index.node.js` (detail di Notes).
- `PROGRESS.md`: entry ini.
**Why:** 3 quick wins: user baru tidak tahu mulai dari mana (butuh checklist onboarding 3 langkah sesuai spec); tab browser pakai icon Next default (belum ada `app/icon`); link dibagikan ke WhatsApp/social tidak punya pratinjau (cuma `og-default.png` statis generik: tiap kreator harusnya dapat kartu sendiri).
**Notes:** **BUG Windows next/og (vercel/next.js#74385): semua route ImageResponse 500 di mesin ini:** copy `next/dist/compiled/@vercel/og/index.node.js` membaca font+wasm dgn `fileURLToPath(join(import.meta.url, "../x"))` → `path.join` win32 merusak file-URL jadi `.\file:\C:\...` → `ERR_INVALID_URL` (reproduksi murni di luar Next juga gagal: **bukan** karena spasi path). Import `@vercel/og` TIDAK bisa menipu webpack (alias `"@vercel/og$": "next/dist/server/og/image-response"` di `create-compiler-aliases.js:81`); paket upstream `@vercel/og@1.0.3` sempat dicoba (font-nya benar `new URL`) tapi punya bug lain (`hb.wasm` dibuka relatif cwd) → **dilepas lagi**. **Solusi final:** patch 3 ekspresi di copy compiled → `fileURLToPath(new URL("./x", import.meta.url))`: penting: `join(url,"../x")` meng-pop NAMA FILE (saudara index.node.js) sedangkan `new URL("../x",base)` naik dari direktori → konversinya ke `"./x"` (ekuivalen 1:1; salah arah sempat bikin ENOENT satu level ke atas). Script idempoten (deteksi pola lama HASIL patch salah juga), auto-jalan tiap `npm install` via postinstall. **Verifikasi dev :3000 (+backend :8081):** `GET /icon` → **200 image/png 653B, 32×32** ✓; `GET /u/medaka/opengraph-image` → **200 image/png 30.109B, 1200×630** ✓; `GET /u/uisec/opengraph-image` (avatar asli) → **200 45.134B** ✓; user tak ada → **200 22.190B** (kartu generik) ✓; HTML `/u/medaka`: `<link rel="icon" href="/icon?…">` ✓ + **tepat 1 `og:image`** (`/u/medaka/opengraph-image?…`) + `og:image:width|height|type|alt` + `twitter:image` + `twitter:card=summary_large_image` ✓ (duplikat manual hilang); visual PNG dibaca langsung: layout sesuai spec (avatar letter, nama SG 60, pill ellipsis, logo). **`next build` HIJAU: 24/24** (22 sebelumnya + `/icon` + `/u/[username]/opengraph-image`; lint+type jalan): `/icon` ikut **prerender static saat build** (bukti satori jalan di build), OG = dynamic; **prod test** `next start -p 3005`: og 200 byte-identik dgn dev, icon 200 ✓; `/dashboard` 200 (checklist = client-side & tersembunyi di SSR by design → memang tak ada di HTML awal). **Perlu cek manual (butuh browser):** (1) checklist muncul utk user anyar; step tercentang saat buat link (1), isi nama+bio (2: CTA "Lengkapi" lompat tab Profil), klik "Bagikan" (3: `jejak_shared`); (2) progress bar melompat + kartu hilang permanen setelah semua selesai / X dismiss (localStorage); (3) "Buat Link" → `/app`; (4) favicon tab baru; (5) preview share WhatsApp utk URL publik `/u/<user>`. `public/og-default.png` tak dirujuk lagi (konvensi file), dibiarkan.

---

## 2026-09-30: Tema baru: Riso Print
**Status:** ✅ Done  
**Files changed:**
- `frontend/lib/themes.js`: `THEMES += "risoPrint"` + objek preset `risoPrint` (label "Riso Print"): palet paper `#F4F0E6`, ink biru-tinta `#1F3A5F` (arbitrary class, bukan ink config `#1C1A12`), aksen `#FF5C39`/`#2B7A78`, kuning `#FFD23F` dipakai sedikit (CTA+badge); `border-2 border-[#1F3A5F]`, `shadow-[4px_4px_0px_#1F3A5F]`, `radius: rounded-lg` (8px), `radiusLarge: rounded-xl`; nav = `bg-[#F4F0E6] border-b-2 border-[#1F3A5F]`, `navHover: hover:text-[#FF5C39]`, cta kuning+ink (7.95:1), ghost/panel/drawer/navCircle/logoDot/chip ikut krem+tinta; kartu `card: "riso-paper text-[#1F3A5F]"`. **Token OPSIONAL baru** (preset lain tak punya → perilaku lama identik): `swatch` (3 hex utk preview picker), `slug` (JetBrains Mono ink), `url` (ink/65), `countColor` (aksen), `barPrimary`/`barSecondary` (accent bar kiri berselang oranye/hijau per index kartu). Komentar header file dokumentasikan token opsional + palet riso.
- `frontend/app/globals.css`: `body[data-profile-theme="risoPrint"]`: `background-color #F4F0E6` + `background-image` noise SVG data-URI (`feTurbulence`, rect `opacity='0.03'`), `background-attachment: fixed`, `color: #1F3A5F` (teks tanpa token: mis. bio: ikut tinta riso); override `body[...] .font-caption { font-family: var(--font-mono) }` (**spec: JANGAN font handwritten**: Caveat dipakai tema lain, riso pakai JetBrains Mono; spesifisitas selector menang tanpa `!important`); class `.riso-paper` = kertas krem + grain 0.03 sendiri utk kartu (ThemeBackdrop grain global ada di BELAKANG kartu opak).
- `frontend/app/components/ProfileLinks.jsx`: pakai token opsional: accent bar kiri (`t.barPrimary` parity genap/ganjil, kosong utk tema lain), baris slug `t.slug ? t.slug : fallback font-caption+lama`, `${t.url || ""}` di baris URL, `${t.countColor || ""}` di angka klik. 4 tema existing → string identik sebelumnya.
- `frontend/app/dashboard/DashboardClient.jsx`: preview picker: `st.swatch ? 3 strip (inline style, aman purge) : preview lama`: opsi "Riso Print" muncul otomatis dari `THEMES` (save via PUT `/api/profile` existing, `THEMES.includes` load-back ✓).
- `backend/internal/handler/handlers_profile.go`: allowlist preset `validThemePreset` += `"risoPrint"` (closed set 5), pesan error 400 & komentar disesuaikan; `validTheme` (read coercion) otomatis meneruskan. **DB tak perlu migrasi**: `theme varchar(20) default 'classic'`, tanpa CHECK, "risoPrint" muat.
- `backend/internal/handler/handlers_profile_test.go`: loop preset eksperimen `["glass"]` → `["glass", "risoPrint"]`.
- `PROGRESS.md`: entry ini.
**Why:** Task "tema baru Riso Print": estetika risograph (kertas krem, tinta biru/oranye/hijau tumpang-tindih, hard shadow tetap dalam bahasa Instant Print).
**Notes:** **Hasil WCAG audit (kontras dihitung, WCAG 2.1):** ink `#1F3A5F` di paper `#F4F0E6` = **10.09:1 ✅ AA** (syarat "harus lulus"); tinta di kuning CTA/badge = **7.95:1 ✅**; huruf avatar (24px bold = large text) tinta-di-`#FF5C39` = **3.74:1 ✅** (lolos syarat large 3:1; kombinasi lain di atas oranye semua gagal); hijau `#2B7A78` di paper = 4.44:1: hanya dipakai sbg accent bar/dinding (dekoratif ≥3:1 ✅), bukan teks. **Temuan di bawah AA utk teks normal (sesuai palet spec, dilaporkan):** (1) `accent-primary #FF5C39` di paper = **2.70:1**: dipakai persis per spec utk angka klik + hover link nav (navHover eksplisit "accent-primary"); rekomendasi lanjutan: angka klik pakai ink atau shade lebih gelap. (2) muted `rgba(31,58,95,.65)` (efektif `#6A7A8E`) = **3.85:1**: teks sekunder/URL/caption (spec palette fixed 0.65); utk 4.5:1 perlu opasitas ≥0.72. **Verifikasi:** `go build ./...` OK + `go test ./...` semua `ok` (1× transient "Access is denied" fork/exec → retry OK); `npm run build` **22/22 hijau** (19 template + 3 stub task sebelumnya; lint jalan). **Save→render:** PUT `/api/profile {"theme":"risoPrint"}` = **200** (allowlist baru) → `GET /api/u/ujiris0` = `risoPrint` (coercion lolos) → SSR `/u/ujiris0` via dev: `riso-paper`×4, `border-[#1F3A5F]`×6, `shadow-[4px_4px_0px_#1F3A5F]`×2, **bar berselang `border-l-[#FF5C39]`×1 + `border-l-[#2B7A78]`×1** ✓, angka `text-[#FF5C39]`×2, slug `font-mono …text-[#1F3A5F]`×2 & `font-caption` hanya di `@username` ✓ (di-override mono via CSS: rule terverifikasi di CSS dev: body rule cream+noise+fixed+color, `.riso-paper`, override `.font-caption`); avatar `bg-[#FF5C39]` ✓; chunk client (`layout.js`/`page.js`) memuat `risoPrint` + token nav krem-tinta + hex swatch ✓. **Regressi:** `/u/medaka` (classic) = `riso-paper`0/`border-l-4`0/`bg-print-white`7/`font-caption`3 **identik pra-perubahan**; HTML landing = `data-profile-theme` **0** (navbar non-publik aman: attribute hanya dipasang ProfileLinks di /u, body rule keyed ke attribute). User uji `ujiris0` + 2 link dihapus dari DB. **Perlu dicek manual (butuh browser):** (1) buka `/u/<user riso>`: **navbar menyala krem+border tinta setelah hydrate** (SSR awal classic lalu swap via MutationObserver: token sudah di bundle), hover link jadi oranye; (2) mobile 375px: kartu 8px, bar kiri 4px, pill nav; (3) picker dashboard: chip "Riso Print" dgn 3 strip `#1F3A5F/#FF5C39/#2B7A78`, klik → save → buka halaman publik; (4) rasa grain (body 3% + ThemeBackdrop 4% menumpuk di area paper: halus, cek apakah cukup "kasar riso" atau perlu disetel).

---

## 2026-09-29: Quick wins: 404 custom + empty state + footer
**Status:** ✅ Done  
**Files changed:**
- `frontend/app/page.jsx`: audit footer: **A (dibiarkan, 3):** Shorten→`/app`, Link-in-bio→`/app`, QR Code→`/#demo` (id `demo` ada). **B (direpoint + stub dibuat, 3):** Kontak `/#kontak`→`/contact`, Kebijakan Privasi `/#privasi`→`/privacy`, Syarat & Ketentuan `/#syarat`→`/terms`. **C (dihapus, 3):** Tentang `/#about`, Blog `/#blog`, Cookie `/#cookie`: anchor id-nya tidak ada di landing (link mati), tidak masuk daftar halaman MVP, tidak ada rencana konten dekat; Kontak/Privasi/Syarat justru kebalikannya → dibuatkan halaman.
- **Baru** `frontend/app/not-found.jsx`: 404 custom Instant Print: SearchX 64px ink, headline "Halaman ini tidak ada jejaknya.", sub, CTA pill kuning `shadow-[4px_4px_0px_#1C1A12]` → `/`, ghost "Cek link kamu" → `/app`; tombol `w-full` di mobile, row di sm+; animasi fade+y-8 dari PageTransition global layout (tak diubah). Berlaku juga untuk `notFound()` (mis. `/u/user-hilang`).
- **Baru** `frontend/app/terms/page.jsx`, `frontend/app/privacy/page.jsx`, `frontend/app/contact/page.jsx`: stub minimal: heading Space Grotesk (`font-display`) + 2-3 paragraf placeholder Work Sans + tombol "Kembali ke Beranda" (ghost); kartu `rounded-xl border-2 border-ink bg-print-white p-6 shadow-[4px_4px_0px_#1C1A12]`; `pt-24` (navbar h-16); metadata title per halaman.
- `frontend/app/components/ProfileLinks.jsx`: empty state ganti `<p>` polos: `<div>` kartu `${t.radius} ${t.borderW} ${t.border} ${t.card} ${t.shadow}` + ikon lucide `Inbox` 48px (`t.text`), headline "Belum ada link di sini." (`font-display`, `t.text`), sub "Kreator ini belum menambahkan link. Cek lagi nanti." (`t.textMuted`); center, tanpa CTA (halaman publik); `ThemeBackdrop` ikut dipasang; `data-profile-theme` tetap terpasang via useEffect yang ada.
- `PROGRESS.md`: entry ini.
**Why:** Footer punya 6 link mati (anchor `#about/#blog/#kontak/#privasi/#syarat/#cookie` tidak ada: id landing cuma `fitur/demo/faq`); 404 masih bawaan Next (generik, di luar design system); profil tanpa link tampil `<p>` telanjang tanpa card/padanan tema.
**Notes:** **Build `npm run build` HIJAU: 22/22** (19 lama + `/contact` `/privacy` `/terms`; lint ikut jalan). **Verifikasi via dev :3000 + backend :8081:** `/halaman-tidak-ada` → **404** + isi custom ✓ (`lucide-search-x ... h-16 w-16 text-ink`, pill `bg-flash-yellow`+shadow keras, kedua CTA); `/u/ujicosong` setelah user uji dihapus → 404 + isi custom yang sama ✓ (jalur `notFound()`); `/terms` `/privacy` `/contact` → 200 ✓, `/app` 200 ✓; HTML landing: `href="/contact"|"/privacy"|"/terms"` masing-masing 1 ✓, enam link mati lama = 0 ✓, `/#demo` tetap 2 (navbar+footer) ✓. **Empty state 4 tema** (user uji `ujicosong`, link=[]): kartu per tema terverifikasi: classic `bg-print-white + shadow ink` ✓, darkroom `bg-[#121212] + border/shadow #FF6B35` ✓, coral (kartu memang identik classic per spesifikasi; pembeda `bg-flash-coral` avatar + page cream) ✓, glass `border-white/[0.28] border-t-white/[0.55] bg-white/70 backdrop-blur-[20px] backdrop-saturate-[180%] + shadow inset` ✓; headline+ikon ada di semua ✓; user uji dihapus dari DB (`DELETE creators id=22`). **Catatan:** `PUT /api/profile` wajib bawa `display_name` (full-replace): body PS→curl wajib lewat file (PS 5.1 merusak quoting JSON). **Perlu dicek manual (butuh browser):** (1) 375px: 404 (ikon/headline/tombol menumpuk full-width, tidak overflow) & empty state; (2) klik seluruh footer di landing → semua mendarat tanpa 404; (3) hover/tap tombol 404 (translate halus): reduced-motion hanya via MotionProvider global.

---

## 2026-09-29: Resep tema Glass → Apple Liquid Glass (WWDC25)
**Status:** ✅ Done  
**Files changed:**
- `frontend/lib/themes.js`: preset **`glass`** (satu-satunya file token) + komentar palet & field `shadow` di header file disesuaikan:
  1. **Backdrop material seragam** → `backdrop-blur-[20px] backdrop-saturate-[180%]` di SEMUA surface kaca: `card` (sudah ada), `avatar`, `accent`, `nav`, `cta`, `ghost`, `panel`, `drawer` (sebelumnya ada yang `blur-xl` tanpa saturate): resep CSS `backdrop-filter: blur(20px) saturate(180%)`.
  2. **Rim light**: `border` token + semua tepi `border-white/80` (`cta`, `ghost`, `panel`, `drawer`, `navCircle`): `border-white/[0.28] border-t-white/[0.55]` = `border: 1px solid rgba(255,255,255,0.28); border-top-color: rgba(255,255,255,0.55)` (dipasangkan `borderW: border-[1px]` yang tak berubah).
  3. **Kontras latar**: `card` `bg-white/50` → **`bg-white/70`**; `ghost` ikut `/50 → /70` (hover `/70 → /90` supaya feedback hover tetap beda). Token lain sudah `/70`.
  4. **Kedalaman**: `shadow`: dari `shadow-[0_8px_32px_rgba(0,0,0,0.08)]` (nyaris tak terlihat) → **`shadow-[inset_0_1px_0_rgba(255,255,255,0.35),0_12px_40px_rgba(0,0,0,0.35)]`** (highlight inset tepi atas + drop shadow panjang).
  - Tak diubah: `page` (paper gradient di globals.css), `chip` (swatch picker), warna teks (`text-ink` dll.), radius/heading, preset lain.
- `PROGRESS.md`: entry ini.
**Why:** Tema Glass terlihat washout (teks di atas kaca putih/50 kurang kontras), tepi seragam putih/80 (tanpa arah cahaya → kaca terasa datar), material tidak seragam (beberapa surface blur tanpa saturate → warna di belakang kaca kusam), dan shadow 0.08 hampir tak ada → tidak ada kedalaman. Resep baru mengikuti material Liquid Glass: saturate menghidupkan kilau warna, bg/70 menaikkan kontras teks, tepi lebih terang di atas (rim light), dan inset+drop shadow memberi "angkat" kartu.
**Notes:** **Verifikasi:** `npm run build` **19/19** ✓ (lint ikut jalan). **CSS output** (`.next/static/css/8ed0536c….css`): `--tw-shadow:inset 0 1px 0 hsla(0,0%,100%,.35),0 12px 40px rgba(0,0,0,.35)` ✓, `.border-white/[0.28]{border-color:…,.28}` + `.border-t-white/[0.55]{border-top-color:…,.55}` ✓, `.bg-white/70{background-color:hsla(0,0%,100%,.7)}` ✓, rantai `backdrop-filter` memuat `blur(20px)` + `saturate(180%)` ✓, aturan `.bg-white/50` TIDAK ada lagi ✓. **SSR `/u/medaka_` (akun glass asli)** via dev :3000: halaman 16,2 KB: header = `border-white/[0.28] border-t-white/[0.55] bg-white/70 backdrop-blur-[20px] backdrop-saturate-[180%]` ✓; hitungan class di HTML: `bg-white/70`×6, rim-light ×8, `shadow-[inset_0_1px_0`×2 (link rows), **`bg-white/50`×0 & `border-white/80`×0** (nilai lama bersih) ✓. **Catatan lingkungan:** dev :3000 di-restart (urutan aman: build saat dev mati → start dev); backend :8081 sempat mati (`go run` anak shell tool kena terminasi) → kini dijalankan dari **binary** `backend/bin/api-8081.exe` (mode full) yang tahan shell mati; `medaka_`=glass ✓. **Perlu dicek manual (butuh browser):** (1) buka `/u/medaka_`: kaca terasa "hidup" (warna gradient di belakang kartu lebih saturan), teks lebih tegas, **tepi atas kartu/pill lebih terang dari tepi bawah (rim light)**, kartu link "terangkat" (drop shadow panjang + kilau inset); (2) navbar di halaman glass: bar frosted dgn material blur+saturate sama, panel dropdown & tombol ikut rim light; (3) picker dashboard (preview chip glass) tidak berubah; (4) tema classic/darkroom/coral tidak berubah.

---

## 2026-09-29: Fix bug card link sejajar + tema persist
**Status:** ✅ Done  
**Files changed:**
- `backend/internal/db/db.go`: **BUG 2**: `GetCreatorByUsername` + `ListLinksByCreator` (SingleStore) dialihkan dari `readDB()` (**replica**) ke **`s.primary`** (read-your-own-writes: login + `GET /api/u/{username}` wajib melihat data terbaru); komentar di `ShardStore` interface + `GetCreatorByIDPrimary` disesuaikan (endpoint publik tak lagi lewat replica).
- `backend/internal/handler/handlers_public.go`: komentar LEARN: trade-off denormalisasi tak lagi menyebut "replica lag" (baca kini primary).
- `ARCHITECTURE.md`: §6 dipertajam: read/write splitting kini **redirect (`GetURL`) → replica** (demo lag Fase 4 tetap, ikut README §6) sedangkan **login + profil publik → primary**.
- `frontend/app/dashboard/DashboardClient.jsx`: **BUG 1**: `SortableLinkRow`: inline `style` dnd-kit hanya dipasang saat dnd aktif (`dndActive = isDragging || transform != null`; idle = tanpa style → semua kartu punya state awal identik, inline tak bisa menimpa CSS hover); class `hover:-translate-y-1 hover:shadow-[6px_6px_0px_#1C1A12] hover:opacity-90` diganti utility **`link-lift`**; komentar header diperbarui (sistem transform tetap TUNGGAL: dnd inline saat drag, CSS saat idle).
- `frontend/app/globals.css`: utility baru **`.link-lift`** dibungkus **`@media (hover: hover) and (pointer: fine)`** (translateY -4px + shadow 6px + opacity .9; transisi 150ms dari class baris).
- `PROGRESS.md`: entry ini.
**Why (akar masalah per bug):**
- **BUG 1 (kartu pertama lebih tinggi / tidak sejajar):** hover-lift dari task detail Apple #4 dipasang via `hover:*` Tailwind di tiap baris. Di **layar sentuh `:hover menempel` setelah tap** (kartu yang tersentuh: biasanya kartu pertama: terangkat permanen), dan di desktop kursor dapat mengendap di posisi kartu pertama setelah klik tab → kartu itu terlihat lebih tinggi dari baris lain. Plus: inline `style` dnd-kit (`transform`+`transition`) dipasang **terus-menerus termasuk saat idle**: berpotensi bertabrakan dengan transform CSS (inline menang). Diverifikasi list kartu TIDAK punya motion/stagger apa pun (motion di dashboard hanya wrapper tab y4, overlay avatar, chevron, expand API docs) jadi sumber geser = transform hover/dnd. Solusi: lift dipindah ke satu utility CSS yang hanya aktif di pointer halus (touch dapat lift = bug; hover tetap utuh di desktop), inline transform hanya saat dnd aktif.
- **BUG 2 (simpan tema di dashboard, `/u/*` tetap Darkroom):** **read/write splitting tanpa sinkron otomatis**. Tulis (`UpdateCreatorProfile`) → PRIMARY; baca publik (`HandleCreatorLinks` → `GetCreatorByUsername` + `ListLinksByCreator`) → `readDB()` = **replica** saat `APP_MODE=full` + `DATABASE_REPLICA_URL` terisi (server yang berjalan di :8081 memakai resep README §4 `$env:APP_MODE="full"` yang MENIMPA `.env` yang isinya `baseline`: `.env` sendiri tidak pernah diubah, mtime 20 Sep). Replica = database kedua `jejak_replica` dengan sinkron **MANUAL** (`db/sync-replica.ps1`, README §3, ARCHITECTURE §6) → setelah simpan tema, `/u/*` menyajikan **tema LAMA selamanya**, sedangkan dashboard (baca primary) menampilkan tema BARU. **Bukti di data:** primary `medaka_`=**glass** vs replica `medaka_`=**darkroom**; `uisec` primary=classic vs replica=darkroom; 3 user tak ada di replica. **Bukti runtime:** kode LAMA (server mode full di :8099) → `GET /api/u/medaka_`=`darkroom`; kode BARU → `glass` ✓; server :8081 di-restart dgn kode baru → `medaka_`=glass ✓; login user baru tadinya GAGAL di :8081 (baris belum disalin ke replica: gejala kelas sama). **Bukan penyebab:** Next fetch (`cache:"no-store"` ✓), Redis (path profil tidak pakai cache; baseline pun mati), body attr (`useEffect [theme]` ✓), PUT (200 + echo tema ✓), `themeStyles` fallback (bukan masalah).
**Notes:** **Verifikasi:** `go build ./...` ✓; `go test ./...` semua paket **OK**; `npm run build` **19/19** ✓ (lint jalan di dalam build). **Round-trip 4 tema** (classic→darkroom→coral→glass) lewat stack nyata: `POST /api/login` + `PUT /api/profile` via proxy Next :3000 → 200 + echo ✓, `GET /api/profile` (persist picker) ✓, `GET /api/u` (server :8081 mode full) selalu segar ✓, render `/u/ujitema929` (glass) = class `border-white/80 bg-white/50 backdrop-blur-[20px]` ✓: akun uji + link tes dihapus dari DB setelah selesai. **BUG 1 statis:** CSS output memuat `@media (hover: hover) and (pointer: fine)` + `.link-lift:hover` ✓; chunk client `app/dashboard/page.js` (dev) berisi `link-lift`×3, `hover:-translate-y-1`=**0**, `dndActive`×2 ✓; `GET /dashboard` 200 tanpa error compile (konten kartu dirender client-side saat hydration: SSR hanya shell loading, jadi cek baris dilakukan via chunk). **Status lingkungan:** server :8081 **di-restart dengan kode fix** (mode full, sama seperti runtime user: log `%TEMP%\opencode\api8081.err.log`); dev :3000 **direstart** (tersedak saat `npm run build` menimpa `.next`); preview :3210 (task lama) dimatikan. **Perlu dicek manual (butuh browser):** (1) dashboard tab **Link Saya**: kartu sejajar saat idle; hover desktop hanya mengangkat kartu yang dituju (turun lagi saat keluar); **tap di mobile 375px TIDAK meninggalkan kartu terangkat**; drag-reorder tetap mulus dan kartu kembali sejajar setelah drop; (2) tab **Profil** → simpan 4 tema satu per satu → buka `/u/medaka` & `/u/medaka_` → tema **langsung berubah** (buka via tab baru + refresh); picker tema tetap menampilkan pilihan terakhir setelah reload; (3) Network: `PUT /api/profile` 200.

---

## 2026-09-29: Landing motion wow (7 primitives)
**Status:** ✅ Done  
**Files changed:**
- `frontend/lib/animations.js`: varian shared baru: **`revealContainer`** (stagger **80ms**) + **`revealItem`** (`y:30 → 0` + **SPRING_SOFT**) utk scroll-reveal kartu fitur (primitive #5). Varian lain tak diubah.
- `frontend/app/page.jsx`: **7 primitive** (semua transform+opacity, tanpa spotlight/marquee/particle):
  1. **Headline reveal per kata**: H1 dipecah jadi `<motion.span className="inline-block">` per kata (copy TIDAK diubah: struktur teks identik, `{" "}` menjaga spasi asli): `{opacity:0, y:20, filter:blur(4px)}` → target, **stagger 60ms + SPRING_SOFT** via `heroWords` (`delayChildren: 0.05`); lalu sub (y8) → ShortenForm → social proof chip dengan **delay eksplisit** `HERO = {badge:0, words:0.05, sub:0.55, form:0.8, proof:1.0}` → total ~1.2s (parent stagger diganti stage-delay karena stagger 60ms antar-sibling tidak menunggu 5 kata selesai)
  2. **Mockup card float + tilt**: pembungkus jadi 2 lapis: luar entrance (y12, EASE 0.25: tak berubah), dalam `animate={{y:[0,-4,0,4,0]}}` loop 4s easeInOut (**float idle ±4px**) + `style={{rotateX, rotateY, transformPerspective:600}}` dari `useMotionValue(-0.5..0.5) → useTransform(±3deg) → useSpring` (cfg SPRING): **tilt reset halus ke 0 saat mouse keluar**, handler `onMouseMove/onMouseLeave` **hanya kartu ini** (bukan halaman), tanpa cursor-spotlight; rotate (style motion value) & y (`animate`) properti transform berbeda → tidak berebut state
  3. **Bar progres scroll**: `<motion.div>` fixed `top-0 left-0 z-[60] h-[2px] w-full bg-flash-yellow`, `style={{scaleX: scrollYProgress, transformOrigin:"left"}}` (`useScroll()`), `aria-hidden`: transform-only, di atas navbar z-50
  4. **Social proof counter**: diverifikasi **sudah sesuai**: "Gratis · Instan · Jujur" (angka BUKAN data real → tanpa counter, sesuai honest-copy gate) pakai `staggerContainer` 60ms `whileInView once:true` → **tanpa perubahan**
  5. **Feature cards stagger**: container `staggerContainer/y12/EASE/60ms` → **`revealContainer` + `cardItem`** (y30, SPRING_SOFT, 80ms, `viewport once:true` tetap)
  6. **FAQ accordion**: diverifikasi **sudah** `transition={SPRING}` (panel) + `faqChevron` SPRING dari task sebelumnya → **tanpa perubahan**
  7. **CTA akhir**: section → `motion.section` `initial {opacity:0,y:8} / whileInView {opacity:1,y:0}` + `SPRING_SOFT` (reduce → `duration:0.15`), `viewport once:true`; tombol `MotionLink` dapat `animate={ctaPulse}` (`useAnimationControls`): **pulse SEKALI** `scale 1 → 1.04 → 1` via **dua spring SPRING berurutan** +800ms setelah `onViewportEnter` (guard `useRef pulsedRef` + timer cleanup; **bukan loop**; keyframes+spring tidak bisa digabung di framer → dua tahap)
  - **Reduced-motion gate**: `useReducedMotion()` → variant hero/kartu/CTA jatuh ke `fade150` (opacity 0→1 `duration:0.15`, **tanpa y/blur**: blur TIDAK ikut dibuang `reducedMotion="user"` karena bukan properti transform), tilt handler tidak dipasang, pulse dilewati; MotionProvider global tetap membuang semua transform. Progress bar scroll-linked tetap jalan (mengikuti scroll user, bukan animasi otonom: didokumentasikan)
- `PROGRESS.md`: entry ini
**Why:** Task "landing motion wow" minta 7 primitive terukur (reveal kata-per-kata, mockup hidup, progres scroll, stagger angka, reveal kartu, akordeon halus, pulse CTA) dengan disiplin anti-AI-slop: transform+opacity saja, spring tanpa overshoot, tanpa spotlight/marquee/confetti/scroll-jacking, dan setiap gerak spatial punya jalur collapse ke opacity saat reduced-motion.
**Notes:** **Deviasi keputusan (didokumentasi):** (a) **copy headline TIDAK diubah**: task mengutip "Satu link, semua konten kamu." sedangkan H1 asli (honest-copy task) = "Satu halaman, / semua link kamu." → reveal dipasang ke teks yang ada (JANGAN DIUBAH); (b) repo ini **framer-motion 13 + React 18** (bukan motion/react + React 19: ikut konvensi repo; `motion.create(Link)` sudah dari task lalu); (c) pulse = dua spring urutan (naik lalu turun) karena framer tak mengizinkan keyframes+spring, hasil visual tetap "scale 1→1.04→1 spring"; (d) stage hero tetap pakai y8 (bukan opacity murni) di mode normal: "fade in + y-8" sesuai spesifikasi. **Build:** `npm run build` **19/19** ✓ (landing `/` 5.45→10.3 kB, First Load 158→164 kB: +4.9 kB utk useScroll/useSpring/useAnimationControls). **Static/SSR (production `next start` port 3210):** HTML 200: bar `<div aria-hidden … z-[60] h-[2px] bg-flash-yellow style="transform-origin:left;transform:scaleX(0)">` ✓; H1 = 5 span per kata dgn spasi asli ✓; teks visible **2005 karakter: identik 100% dgn sebelum task** (copy tak berubah) ✓; CTA section ada ✓; `opacity:0` SSR hanya 1 (entrance mockup: konsisten dgn perilaku framer sebelumnya); CSS output: `z-[60]` ✓ `bg-flash-yellow` ✓; chunk landing berisi `blur(4px)`, `scaleX`, `repeat`, `1.04` ✓; `overflow-x: clip` globals tetap ✓. **Catatan lingkungan:** server dev port 3000 (proses lama) mati/tersedak karena `.next` dihapus saat build → dimatikan; preview production berjalan di **localhost:3210** utk verifikasi manual (atau jalankan ulang `npm run dev`). **Perlu dicek manual (butuh browser):** (1) `/`: headline muncul kata-per-kata lalu sub→form→chip, total ~1.2s; (2) mockup: float ±4px loop 4s + tilt ±3deg ikut kursor & reset saat keluar; (3) scroll → bar kuning 2px di atas navbar terisi kiri→kanan; (4) section fitur → 4 kartu reveal y30 stagger 80ms, scroll naik-turun TIDAK re-animate; (5) CTA → section muncul lalu tombol pulse 1.04 sekali (+800ms), tidak loop; (6) FAQ → expand spring halus; (7) DevTools reduced-motion:reduce → headline/kartu/CTA jadi fade polos, tilt mati, float diam, pulse dilewati; (8) Performance record 5s → 60fps (motion hanya transform/opacity, bar scroll-linked); (9) 320/375/414/768 → tanpa scroll horizontal (span inline-block identik teks asli + `overflow-x:clip`), tanpa layout shift; (10) build 19/19 ✓.

---

## 2026-09-29: Tooltip fix + 6 detail Apple
**Status:** ✅ Done  
**Files changed:**
- **Task 1: tooltip chart tak lagi tertutup kursor:**
  - `frontend/app/components/ClicksChart.jsx` + `frontend/app/dashboard/RingkasanTab.jsx`: kedua instance `<Tooltip>` (chart Analytics + MiniTrend Ringkasan) + `position={{y:-70}} offset={16} allowEscapeViewBox={{x:false,y:true}} wrapperStyle={{zIndex:100,pointerEvents:"none"}}` + komentar penjelas. **Semantik Recharts:** nilai `position.y` dipakai **absolut** sbg translateY (translate.js: `if (position && isNumber(position[key])) return position[key]`): BUKAN relatif titik data → kartu terpaku 70px di atas chart container (di area padding atas, di luar jalur kursor); `position.x` tak diisi → `offset 16` tetap mengikuti kursor + boundary flip otomatis di tepi; `allowEscapeViewBox.y` inert selama `position.y` diset (dipertahankan sesuai spec); wrapper Recharts `position:relative` tanpa overflow → tak terpotong; `pointerEvents:'none'` sudah default `TooltipBoundingBox` (diduplikat sesuai spec)
- **Task 2: 6 detail Apple:**
  - `frontend/lib/animations.js`: **`SPRING`** = `{type:"spring",stiffness:300,damping:30,mass:0.8}` (ζ≈0.97 ≈ kritis → settle halus TANPA overshoot) + **`SPRING_SOFT`** = `180/24` diekspor; `faqChevron` → SPRING; `DEFAULT_TRANSITION` tetap ada (dipakai NavbarClient)
  - `frontend/lib/PageTransition.jsx`: **baru**, file `app/components/PageTransition.jsx` **dihapus** (dead code: tak pernah di-import): `<AnimatePresence mode="wait" initial={false}>` enter `opacity 0→1 + y 8→0` 250ms EASE, tanpa exit → **transisi per halaman** (detail #6). `transform:none` setelah settle (buildTransform default) → anak fixed aman; navbar sendiri **di luar** wrapper
  - `frontend/lib/MotionProvider.jsx`: **baru**: wrapper client `<MotionConfig reducedMotion="user">` (framer-motion v13 tak punya directive `use client` → context server tidak sampai ke client children) → seluruh animasi durasi/spring ikut collapse jadi opacity saat `prefers-reduced-motion`
  - `frontend/app/layout.jsx`: `<MotionProvider><Navbar/><PageTransition>{children}</PageTransition></MotionProvider>`
  - **#1 spring utk UI state:** `ShareModal.jsx` (ShareButton+backdrop+panel), `QrModal.jsx`, `EditLinkModal.jsx` → `transition={SPRING}`; `SubmitButton.jsx`, ShareButton, tombol CTA landing → `whileHover`/`whileTap` spring (**transition didalam gesture object** supaya tidak menimpa transisi variant entrance: detail #4 pattern); FAQ panel + chevron landing → SPRING. **Sesuai JANGAN DIUBAH, ditinggalkan sbg durasi:** `NavbarClient.jsx` (3), `AuthModal.jsx`, `ProfileLinks.jsx` (halaman bertema) 
  - **#2 transisi angka:** `RingkasanTab.jsx`: `AnimatedNumber` baru (`animate(mv,value,{type:"spring",duration:0.8,bounce:0})`: durasi+bounce didukung: damping ratio = 1-bounce = 1 → kritis ±800ms, angka tak "kelewatan"; value `0` → `mv.set(0)` tanpa anim); `formatStat` ≥1000 → `"1.2K"` (round dulu; node-test: 999→999, 999.6→1.0K, 1234→1.2K, -1500→-1.5K: 1.200.000→"1200.0K" = pola K persis spec)
  - **#3 micro-feedback salin:** `CopyButton.jsx`: ikon lucide `Copy`→`Check` (sukses) / `X` (gagal), teks Salin/Tersalin/Gagal, **bump** spring scale `1.05→1` saat sukses (`bump` state + `setTimeout 200ms`; keyframes+spring tak bisa digabung → toggle), bg tetap inline style (keputusan task lama: inline meng-override class bg manapun)
  - **#4 hover lift shadow 4→6px:** `RingkasanTab` StatCard + kartu fitur landing (motion, `y:-4` + `boxShadow:"6px 6px 0 #1C1A12"` + SPRING_SOFT + base `${st.shadow}` 4px); **SortableLinkRow (DashboardClient) pakai utility Tailwind** `hover:-translate-y-1 hover:shadow-[6px_6px_0px_#1C1A12]` + base `${st.shadow}`: **BUKAN framer**: dnd-kit memegang `style.transform` baris via inline style (transform cuma boleh satu pemilik; framer+drag → nilai `drag-y` vs `hover-y` bertabrakan di motion value yang sama). CSS tak punya spring → transisi 150ms (nilai target tetap -4px/6px), selama drag inline transform mengalahkan hover CSS
  - **#5 skeleton shimmer:** `frontend/app/components/Skeleton.jsx` **baru** + rule `.skeleton` (`globals.css`): base `rgba(28,26,18,.08)` (= ink/8), band `transparent→rgba(255,255,255,.7)→transparent` bergerak 1.5s linear (`@keyframes shimmer`, `:after` translateX -100%→100%), `border-radius:8px`: **interpretasi spec**: highlight = band putih transparan (sesuai deskripsi gradien) di atas dasar ink/8, bukan rgba ink; `prefers-reduced-motion` → `animation:none`. Dipakai 3 titik loading RingkasanTab (angka `h-8 w-16`, MiniTrend `h-[120px]`, top-5 `h-14`)
  - **#6 transisi + micro:** tab dashboard → `initial y:4` / `animate y:0` / exit opacity, 200ms EASE; chevron API docs → SPRING; height panel API docs (0.25 easeInOut) → SPRING; overlay avatar hover → SPRING
  - `frontend/app/page.jsx`: kartu fitur +base shadow + whileHover lift; CTA "Daftar Gratis" → `const MotionLink = motion.create(Link)` (v13: `motion(Component)` deprecated) dgn gesture spring; FAQ panel → SPRING
- `PROGRESS.md`: entry ini
**Why:** Task 1: spesifikasi fix: tooltip mengikuti kursor tanpa offset-y → kartu menempel persis di titik/garis `cursor` (yang dibuat tebal di task sebelumnya) sehingga garis menutupi isi tooltip; minta kartu ~70px di atas. Task 2: enam detail gaya Apple (spring alami utk state/gesture, angka menghitung naik, feedback salin berwujud, hover mengangkat kartu dgn shadow membesar, transisi antar halaman, skeleton menyala halus) belum ada: motion UI masih berupa duration-tween bawaan.
**Notes:** **Build:** `npm run build` **19/19** ✓ (backend tak disentuh). **CSS output:** `.skeleton` (base/`:after`/reduced-motion) + `@keyframes shimmer` ✓, `hover:-translate-y-1` ×2 ✓, `hover:shadow-[6px_...]` ✓, `font-style:italic` **0** ✓. **Chunks:** `allowEscapeViewBox` (chart chunks) ✓, `Tersalin` + `1.05` ✓, `reducedMotion` di `layout-*.js` (MotionProvider) ✓. **SSR (server dev port 3000: `next start` lokal tak jalan spawn di sesi ini, pakai server yang sudah hidup):** `/` 200: CTA `href="/app"` ✓, `shadow-[4px_4px_0px_#1C1A12]` **×5** (4 kartu fitur + CTA) ✓, `faq-panel` ×4 ✓, italic 0 ✓, copy FAQ/demo/footer utuh; `/dashboard` 200 shell + "Memuat" (markup skeleton muncul setelah fetch client: tak terlihat di SSR). **Dokumentasi keputusan:** (a) Skeleton highlight = band putih-transparan di atas ink/8 (dua frasa spec "ink/8 base" + "highlight bergerak" dibaca sebagai lapisan gradien putih; alternatif rgba ink mentah dgn `mix-blend` tak memakai blend di spec); (b) animasi `duration`+`bounce` spring valid: tipe `Spring` di motion-dom (`bounce?: number`, "overridden jika stiffness/damping/mass diset": AnimatedNumber hanya set durasi+bounce): SPRING konstan pakai stiffness/damping/mass tanpa durasi, jadi kedua jalur tidak bercampur; (c) StatCard/feature card animasi box-shadow dari nilai class (computed) → inline saat hover, revert by framer. **Perlu dicek manual (butuh browser):** (1) hover chart Ringkasan & Analytics di 3 tepi: kartu 70px di atas container, tak kepotong/tak tertutup kursor, flip `offset 16` di kanan; (2) buka `/dashboard`: 4 angka menghitung dari 0 ±800ms (tanpa overshoot), `0` tampil langsung; (3) hover StatCard + kartu fitur + baris link (lift -4px, shadow 4→6px), SortableLinkRow tetap bisa di-drag (inline transform menang); (4) salin short URL → ikon Check + "Tersalin" + scale 1.05 bounce, gagal → X merah; (5) ganti tab dashboard (fade+y4 200ms), buka/tutup API docs (spring height), hover avatar (overlay spring); (6) reload dashboard (ganti jaringan) → skeleton shimmer; (7) `prefers-reduced-motion` → semua anim collapse jadi opacity + skeleton diam + `scroll-behavior:auto`; (8) navigasi antar halaman → enter halus tanpa "hit"; (9) Navbar/AuthModal/ProfileLinks/Tema /u/ tetap identik (tak disentuh: JANGAN DIUBAH).

---

## 2026-09-29: Landing polish + chart tooltip
**Status:** ✅ Done  
**Files changed:**
- `frontend/app/components/ChartTooltip.jsx`: **komponen baru**: konten tooltip kustom Recharts utk semua chart dashboard (Ringkasan + Analytics). Styling Instant Print: kartu `bg-white border-2 border-ink shadow-[4px_4px_0px_#1C1A12] rounded-lg p-2`; baris tanggal = font body (Work Sans), baris angka = `font-mono` bold. + `formatChartDate("2026-09-17")` → `"17 Sep 2026"` (tanpa koma; fallback ke field `date` datum aktif saat `label` undefined: MiniTrend tanpa `<XAxis>`)
- `frontend/app/components/ClicksChart.jsx`: import `Tooltip` + ChartTooltip; `<Tooltip content={<ChartTooltip/>} cursor={{stroke:"#1C1A12",strokeOpacity:0.35,strokeWidth:1.5}} isAnimationActive={false}/>` + `activeDot` (r4, stroke ink, fill = warna garis). **Catatan:** di LineChart, Recharts menggambar `cursor` sbg Curve vertikal → **garis ReferenceLine saat hover terpenuhi oleh `cursor`** tanpa state/index terpisah (nice-to-have opsional tidak perlu komponen tambahan)
- `frontend/app/dashboard/RingkasanTab.jsx`: `MiniTrend` dapat wiring identik (import Tooltip + ChartTooltip + activeDot)
- `frontend/app/globals.css`: `html, body { overflow-x: clip }` (CLIP, bukan hidden: hidden bikin body jadi scroll container & bisa menangkap scroll elemen fixed) + `@media (prefers-reduced-motion: reduce) { html { scroll-behavior: auto } }`
- `frontend/app/components/CopyButton.jsx`: **fix bug ukuran**: className SELALU dipertahankan, feedback state (bg `#FFD23F`/`#FF5C3D` = token flash-yellow/coral) dipasang via **inline style** (inline meng-override class bg manapun tanpa bergantung urutan CSS); dulu string feedback menggantikan className penuh → tombol mendadak menyusut ke `py-0.5` di tengah layout. Komentar token desain ditambahkan
- `frontend/app/components/ShortenForm.jsx`: input utama `min-w-0` (boleh menyusut < lebar intrinsik → cegah overflow 320px); 3 input dapat `aria-label` + placeholder deskriptif (`https://contoh.com/link-kamu`, slug, tag); tombol **"Shorten" → "Persingkat"** (verb+outcome, label pendek disengaja: "Persingkat link" + input 173px melebihi 288px di 320px) `py-2.5 → py-3` (44px); loading `"Memproses..."` → `"Membuat link..."`; error fallback `"Gagal membuat short URL"` → `"Gagal membuat link: periksa URL lalu coba lagi."` (server tetap kirim pesan spesifik utk kasus validasi); CopyButton hasil `py-0.5 → py-3.5` (46px ≥44)
- `frontend/app/page.jsx`: **(A) honest copy**: stats `500+/1.2M+/300+` → `Gratis/Instan/Jujur` (label produk verifiable) + grid `grid-cols-1 sm:grid-cols-3` (kata 4-6 huruf di text-3xl tak lagi menyumpal 3 kolom 320px); social-proof hero avatars "300+ kreator" → chip ikon Link2 + "link baru jadi tanpa daftar" (diverifikasi: main.go: anonim boleh shorten); marquee caption "Dipercaya oleh kreator & bisnis" → "Satu halaman untuk link dari semua platform kamu"; **section testimoni ("Kata mereka": persona fiktif Raka/Dinda) DIHAPUS** (+ import `Quote`); blok aksen "500+ link … setiap bulannya" → **"Gratis."** + "Dashboard, custom slug, QR, dan analytics: tanpa kartu di tahap ini." + tag "untuk kreator indie". **(B) URL accuracy**: `jejak.app/nama-kamu` → `jejak.app/r/nama-kamu` (features[0] + FAQ[2]: `/r/` = redirect API, satu-satunya short route); mockup `jejak.app/@kreator` → `jejak.app/u/kreator` (+ teks CopyButton); demo `display = jejak.app/r/{slug}` + prefix input `jejak.app/r/` + **hapus slash nyasar** `/{display}` → `{display}`; baris link mockup `/{code}` → `r/{code}`; footer "QR Code" `/#qr` → `/#demo` (anchor asli). **(C) motion**: marquee infinite 28s **dihapus** → pill statis `flex flex-wrap` (dekoratif murni, wrap di layar sempit); mockup entrance `duration 0.4 → 0.25`; seluruh landing dibungkus `<MotionConfig reducedMotion="user">`. **(D) mockup honest**: chip **"contoh"** di header kartu (data klik 284/210/98/156 tak terbaca sbg metrik asli); hapus import tak terpakai (`ArrowRight`, `Smartphone`, `fadeUp`); demo input `min-w-0`
- `PROGRESS.md`: entry ini
**Why:** Task 1: chart Ringkasan/Analytics tidak punya hover tooltip: user tak bisa baca nilai klik per tanggal (hanya lihat bentuk garis). Task 2: audit halaman landing menemukan (a) klaim kuantitatif tanpa sumber (500+/1.2M+/300+/testimoni fiktif), (b) URL di copy menampilkan route yang tidak ada (`@kreator`, `nama-kamu` tanpa `/r/`, `/#qr`), (c) body bisa scroll horizontal di 320px, (d) motion dekoratif (marquee infinite) tanpa guard reduced-motion global, (e) tombol/input < 44px, placeholder "https://..." tak deskriptif, label tombol campur EN/ID.
**Notes:** **Verifikasi (tanpa browser: static + SSR):** `npm run build` **19/19** ✓, `go build ./...` ✓; landing di-`next start` (port 3111) lalu HTML di-cek: **20/20 pass**: "Persingkat"/"Membuat link..."/placeholder baru ada; "500+ link", "1.2M+", "300+ kreator", "Dipercaya oleh", "Kata mereka", "Raka"/"Dinda", `jejak.app/@`, `flash-green` = **0 match**; `jejak.app/r/nama-kamu`, `jejak.app/u/kreator` (SSR `<!-- -->` antar text node), prefix demo `jejak.app/r/`, footer `href="/#demo"`, chip "contoh" ada. CSS output: `overflow-x:clip` ✓, `prefers-reduced-motion` ✓, `after:-inset-x-4/-inset-y-4/content-['']` (hit-slop ±16px → ~51px utk CopyButton mockup) ✓, `font-style:italic` = 0 ✓; chunk `page-*.js` berisi `reducedMotion` (MotionConfig) ✓; `formatChartDate` diuji node: `17 Sep 2026`, `1 Des 2026`, `""`, passthrough. **Audit per disiplin:** honest copy ✅ (semua angka karangan dihapus/diganti janji verifiable) · italic heading ✅ 0 · mobile ✅ (clip + stack stats + min-w-0 + 44px target; PR 283px < 288px available di 320px) · motion ✅ (marquee gone, 0.25s, reducedMotion="user" + scroll auto) · re-drawn chrome ✅ (tanpa browser/phone frame, kartu preview + label "contoh") · micro-copy ✅ (verb+outcome, error = salah+aksi, placeholder deskriptif). **Perlu dicek manual (butuh browser):** (1) hover chart Analytics & Ringkasan: kartu tooltip + garis vertikal + tanggal "17 Sep 2026"; (2) 320/375/414/768: tidak ada scroll horizontal di `/`; (3) shorten 1 langkah → hasil CopyButton "Tersalin" tanpa loncat ukuran; (4) hover/salin CopyButton mockup (hit-slop); (5) `prefers-reduced-motion` → tanpa anim. **Sisa TODO:** link footer mati (`/#about`, `/#blog`, `/#kontak`, `/#privasi`, `/#syarat`, `/#cookie`: semua halaman/anchor tidak ada); section testimoni hanya boleh dikembalikan dgn testimoni asli + izin; target sentuh dashboard (tab bar scroller `overflow-x-auto`, CopyButton list 16px, input 40px) di luar scope landing; footer link utama ("Shorten"/"Link-in-bio") mengarah `/app` generik.

---

## 2026-09-29: Refine 4 tema halaman publik + navbar adaptif (scope tema: hanya /u/[username])
**Status:** ✅ Done  
**Files changed:**
- `frontend/lib/themes.js`: (1) komentar SCOPE baru: tema hanya berlaku di `/u/[username]` + navbar di halaman itu; dashboard, /app, landing wajib Instant Print + LEARN (chrome navbar = token di file ini, bukan ternary hardcode); (2) **refine preset**: **Darkroom**: teks `print-white` → `#F5F5F5`, aksen/border card/shadow `#000000`/putih → **`#FF6B35`** (spec: border 2px oranye + shadow `4px 4px 0 #FF6B35`), chip border oranye; **Coral**: cream `#F5F1E8` + aksen `#FF5C3D` (teks ink) sudah sesuai, ditambah ghost hover tint coral; **Glass**: redesign Apple-style penuh: `radius` `rounded-xl`→`rounded-2xl`, `radiusLarge`→`rounded-3xl`, `border` `ink/10`→`white/80` 1px, `card` `white/60 blur-2xl`→`white/50 backdrop-blur-[20px] backdrop-saturate-[180%]`, `shadow` `""`→`shadow-[0_8px_32px_rgba(0,0,0,0.08)]`, avatar/accent `white/55`→`white/70` (= `rgba(255,255,255,0.7)` spec), badge `white/55`→`bg-ink text-print-white` (16.7:1), placeholder→`#8E8E93` (abu iOS); (3) **10 token CHROME baru × 4 preset**: `nav` (bar bg+border-b), `navHover`, `cta`, `ghost`, `logoDot`, `panel` (dropdown), `panelHover`, `panelRule`, `drawer`, `navCircle` (avatar akun + tombol tutup drawer)
- `frontend/app/globals.css`: rule glass: `background-color:#ECECEC` → **`linear-gradient(135deg,#F0F0F0 0%,#FFFFFF 50%,#F5F5F7 100%)` + `background-attachment:fixed`** (paper gradient jadi satu sumber kebenaran di body, seperti tema solid lain); komentar header skop tema diperbarui (hanya ProfileLinks yang boleh pasang attribute)
- `frontend/app/components/ThemeBackdrop.jsx`: cabang radial-gradient glass **dihapus** (gradient pindah ke globals.css); prop `theme` dihapus: komponen jadi murni grain 4% untuk semua tema; 4 pemanggil diperbarui
- `frontend/app/components/ProfileLinks.jsx`: `<ThemeBackdrop theme={theme}/>` → `<ThemeBackdrop/>`; komentar: satu-satunya setter `body[data-profile-theme]`
- `frontend/app/components/NavbarClient.jsx`: **semua hardcode warna chrome diganti token tema**: `barCls` ternary 4 cabang → `t.nav`; hover link `hover:text-flash-yellow` → `t.navHover`; `ghostBtnCls`/`primaryBtnCls` (asLight/lightBtn branches) → `t.ghost`/`t.cta`; logo dot ternary coral → `t.logoDot`; avatar akun → `t.navCircle`; panel dropdown `asLight?...` → `${t.radiusLarge} ${t.panel}`; divider → `t.panelRule`; item dropdown/drawer → `${t.text} ${t.panelHover}`; drawer panel → `t.drawer`; tombol tutup drawer → `t.navCircle`; variabel `asLight`/`lightBtn` dihapus
- `frontend/app/dashboard/DashboardClient.jsx`: **efek `document.body.dataset.profileTheme` DIHAPUS** (dashboard tak lagi ikut tema kreator); `st = themeStyles(theme)` → `themeStyles("classic")` (state `theme` dipertahankan hanya utk picker preset + payload PUT /api/profile); `<ThemeBackdrop theme={theme}/>` → `<ThemeBackdrop/>`; `<ClicksChart theme={theme} st={st}/>` → `<ClicksChart st={st}/>`; komentar diperbarui
- `frontend/app/app/page.jsx`: fetch `GET /api/profile` + state theme + body-attr effect **dihapus** (efek lama juga stale: deps `[]` tapi men-set attribute tiap render pertama); selalu `themeStyles("classic")`; komentar diganti skop baru
- `frontend/app/page.jsx`: `<ThemeBackdrop theme="classic"/>` → `<ThemeBackdrop/>` (visual tak berubah)
- `PROGRESS.md`: entry ini
**Why:** Spesifikasi refine: (1) tema hanya untuk halaman publik + navbar di sana: dashboard dan /app harus selalu Instant Print (saat ini keduanya masih memasang attribute → body dashboard bisa jadi gelap & navbar /dashboard ikut tema padahal halaman terang); (2) Darkroom harus oranye-#FF6B35 (border+shadow ikut aksen), Glass harus redesign Apple-style penuh (radius 16, blur+saturate, soft shadow, gradient paper); (3) navbar memakai ternary hardcode warna per-tema: pindah ke token supaya menambah tema = menambah 1 objek preset.
**Notes:** **Hasil per tema (static audit: tidak ada browser utk screenshot):**
| Tema | paper | teks | aksen / border / shadow | navbar | Status |
|---|---|---|---|---|---|
| Classic | #FAFAF7 (tak berubah) | ink | kuning, border ink, shadow keras: TAK BERUBAH | `bg-print-white/80 blur + border-b-2 ink`, hover kuning: **1:1 dgn sebelumnya** | ✅ identik (verifikasi: navbar di /, /dashboard, /app) |
| Darkroom | #121212 | `#F5F5F5` (17.2:1) | **#FF6B35** avatar/accent + border card + shadow `4px_4px_0 #FF6B35` (FIX: dulu border putih + shadow #000000) | bar `#121212` + `border-b-2 #FF6B35`, link hover oranye | ✅ match spec |
| Coral | cream #F5F1E8 | ink | `#FF5C3D` avatar/accent (teks ink), badge `#FF6B1A`, shadow ink | bar `cream/85 + border-b-2 ink`, link hover **`#C43A20`** | ✅ match spec |
| Glass | `linear-gradient(135°,#F0F0F0,#FFF 50%,#F5F5F7)` fixed (globals.css) | ink | card `white/50 + blur-20 + saturate-180`, border `1px white/80`, radius 16px, shadow `0 8px 32px rgba(0,0,0,.08)`, accent `white/70`, badge `ink/putih` | bar `white/60 blur` + `border-b rgba(0,0,0,.06)`, tombol soft shadow | ✅ match spec |
**WCAG AA (dihitung, ratio WCAG 2.x):** Darkroom: `#F5F5F5`/`#121212` **17.18** ✓, muted/70 **8.74** ✓, placeholder/50 **5.01** ✓, teks `#FF6B35` **6.61** ✓, ink-di-`#FF6B35` **6.14** ✓, ink-di-coral badge **5.68** ✓. Coral: hover `#C43A20`/cream **4.69** ✓: **deviasi terdokumentasi**: `#FF5C3D` murni cuma **2.72** ✗ (dan `#E04A2E` 3.59 ✗) → dipakai utk **bg/border saja**, teks/hover pakai `#C43A20`. Glass: teks ink di card **16.83** ✓, ink/70 **6.29** ✓, nav-hover ink/60 **4.55** ✓, badge **16.65** ✓; placeholder `#8E8E93` **2.86-3.26** ⚠ (di bawah AA: warna wajib spesifikasi, khusus placeholder non-esensial). **Pre-existing, TIDAK diperbaiki (keputusan terpisah):** muted `#8A8578` di print-white **3.52** / cream **3.26** (kandidat `#6E6A5E` ≈5.2:1); hover nav classic `flash-yellow` di print-white **1.38** (konflik DESIGN.md §8 klasik); badge classic putih-di-coral **2.93**.
**Scope enforcement:** grep `profileTheme` = 3 hasil: setter HANYA `ProfileLinks.jsx` (2), reader `NavbarClient.jsx` (1); DashboardClient & /app bersih. **Verifikasi build:** `npm run build` **19/19** ✓; `go build ./...` ✓ (backend tak disentuh). CSS output dicek: kelas baru ada (`#FF6B35`, `#C43A20`, `#8E8E93`, `backdrop-saturate-[180%]`, `backdrop-blur-[20px]`, `linear-gradient(135deg,...)`, `background-attachment:fixed`, `hover:bg-[#FF6B35]/15`, `bg-black/[4%]`, `0_8px_32px`) dan nilai lama hilang (`ECECEC`, radial `120% 90%` = 0 match). **Perlu dicek manual (butuh browser):** (1) `/u/{username}` × 4 tema: bar navbar, border-b, hover link, CTA/Masuk, avatar+dropdown, drawer mobile; (2) navbar klasik di `/`, `/dashboard`, `/app` identik dgn sebelumnya; (3) akun darkroom/glass → dashboard & /app tetap terang (klasik); (4) glass: gradient diam saat scroll, card blur+saturate terlihat, shadow lembut; (5) darkroom: border & shadow oranye di card link; (6) coral: hover nav gelap; (7) AuthModal terbuka dari navbar bertema; (8) 375px drawer per tema.

---

## 2026-09-29: Polish: emoji → icon, fix 4 tema profil, Bagikan Halaman + QR
**Status:** ✅ Done  
**Files changed:**
- **Task 1: emoji → lucide icon:**
  - `frontend/app/dashboard/RingkasanTab.jsx`: greeting `"Halo, {nama} 👋"` → `"Halo, {nama}"` (emoji dihapus; sub-teks cukup)
  - `frontend/app/page.jsx`: footer `"dicetak dengan 💛 di Indonesia"` → ikon `<Heart size={14}>` lucide (fill+text flash-coral, aria-hidden); import Heart ditambahkan
  - Scan non-ASCII seluruh `frontend/**/*.{js,jsx,css}`: sisa glyph = `★/☆` (U+2605/2606: dingbat teks, **bukan emoji**; token desain "★ Unggulan" di themes.js) + typographic `: →©×≡`: tidak diubah
- **Task 2: verifikasi + fix 4 tema:**
  - `frontend/lib/themes.js`: field baru `shadow` per preset (classic/coral `shadow-[4px_4px_0px_#1C1A12]` = "shadow keras"; darkroom `shadow-[4px_4px_0px_#000000]` = "shadow gelap"; glass `""`); **Darkroom**: accent/avatar `bg-flash-coral text-print-white` → `bg-flash-orange text-ink` (spec aksen ORANYE + WCAG: putih-di-oranye ≈2.9:1 ✗ → ink ≈6.1:1 ✓); badge → `bg-flash-coral text-ink` (putih-di-coral ≈3.1:1 ✗ → ink ≈5.7:1 ✓); **Coral**: accent/avatar `bg-flash-orange` → `bg-flash-coral text-ink` (spec aksen **#FF5C3D**: sebelumnya kebalik dgn Darkroom); `page` stale `bg-print-white` → `bg-[#F5F1E8]` (sync dgn rule globals.css); badge tetap `bg-flash-orange text-ink` (pasangan beda-warna, ≈6.1:1 ✓)
  - `frontend/app/components/ProfileLinks.jsx`: card link kini pakai `${t.shadow}` (spec Classic "shadow keras" / Darkroom "shadow gelap" sebelumnya tidak ada); komentar body-rule diperbaiki (coral kini punya rule cream)
  - `frontend/app/dashboard/DashboardClient.jsx`: tombol **"Lihat Profil"** di tab Profil: pill `border-2 border-ink bg-white` + ikon ExternalLink + `target="_blank" rel="noopener noreferrer"`, sejajar dgn "Halaman publikmu"; komentar theme-body diperbaiki
  - Preview picker vs real: **MATCH**: chip = swatch solid per preset (bg/border/label + bar `st.avatar`), mengikuti accent baru di kedua sisi; glass chip tetap solid putih (sudah by-design anti "putih-di-atas-putih")
- **Task 3: Bagikan Halaman + QR:**
  - `frontend/app/components/ShareModal.jsx`: **komponen baru**: export `ShareButton` (pill `border-2 border-ink bg-flash-yellow shadow-[4px_4px_0px_#1C1A12] px-5 py-2.5`, ikon Share2, hover `y:-2` + shadow membesar, tap `scale:0.97`, EASE array) + default `ShareModal` (backdrop `bg-ink/40 backdrop-blur-sm`, panel `bg-white border-2 border-ink shadow-[6px_6px_0px_#1C1A12] rounded-xl p-6 max-w-md`, fade/scale 0.95→1 200ms EASE; Escape + klik backdrop; `role="dialog"` + `aria-modal` + `aria-labelledby` + focus trap Tab/Shift+Tab); isi: (1) input readonly URL + tombol Salin via `useCopyToClipboard` → "Tersalin"+Check 2s, (2) `QRCodeCanvas` 200px `fgColor #1C1A12 / bgColor #FAFAF7` + "Unduh QR" PNG, (3) "Lihat Profil" → tab baru
  - `frontend/app/dashboard/RingkasanTab.jsx`: header jadi flex: greeting kiri + `ShareButton` kanan (prop baru `onShare`)
  - `frontend/app/dashboard/DashboardClient.jsx`: state `shareOpen`; trigger di Ringkasan (`onShare`) + tab Profil (sebelah Lihat Profil); `ShareModal` di-mount di AnimatePresence modal yang sudah ada
  - URL share: `window.location.origin + "/u/" + username` (dev/prod otomatis)
- `PROGRESS.md`: entry baru
**Why:** (1) emoji 👋/💛 tidak konsisten dgn design system icon SVG lucide (DESIGN.md §0: "Emoji sebagai pengganti ikon asli" DILARANG); (2) aksen Coral↔Darkroom kebalik dari spec, Coral `page` stale, card link tanpa shadow, teks badge darkroom gagal WCAG AA, tombol "Lihat Profil" belum ada; (3) user belum bisa membagikan halaman publik (copy/QR/preview) dari dashboard.
**Notes:** **Hasil verifikasi 4 tema (static audit: stack tidak jalan utk screenshot):**
| Tema | bg | teks | aksen | card link | Status |
|---|---|---|---|---|---|
| Classic | print-white ✓ | ink ✓ | flash-yellow ✓ | border-2 ink + shadow keras ✓ (FIX: shadow ditambah) | ✅ match |
| Darkroom | #121212 ✓ | print-white ✓ (≈18.7:1) | **ORANYE flash-orange** ✓ (FIX: dari coral) | border terang + shadow gelap ✓ (FIX: shadow ditambah) | ✅ match after fix |
| Coral | cream #F5F1E8 ✓ | ink ✓ | **coral #FF5C3D** ✓ (FIX: dari orange) | border-2 ink + shadow ✓ | ✅ match after fix |
| Glass | frosted gradient + grain ✓ | ink ✓ | frosted white/55 ✓ | `bg-white/60 backdrop-blur-2xl`: blur **aktif** (class ter-generate ✓, menye blur grain/gradient di belakang card) ✅ | ✅ match |
WCAG AA: badge/aksen darkroom fixed (ink text); caption darkroom /70 ≈9:1 ✓. Verifikasi emoji ulang: 0 emoji tersisa (★/☆ = dingbat teks, bukan emoji). Build: `go build ./...` ✓, `npm run build` **19/19** ✓ (CSS payload berisi `shadow-[4px_4px_0px_#1C1A12]`, `#000000`, `bg-[#F5F1E8]`, `bg-flash-orange`, `backdrop-blur-2xl` ✓). **Perlu dicek manual (butuh browser):** (1) `/dashboard` greeting tanpa emoji; (2) footer landing Heart icon; (3) 4 tema `/u/{username}` + bandingkan dgn chip picker (768/1440px); (4) tombol Lihat Profil → tab baru; (5) Bagikan Halaman → modal smooth, Salin→"Tersalin" 2s, QR 200px terunduh PNG, Escape/backdrop close, Tab terkunci di modal; (6) mobile 375px modal full-width minus padding.

---

## 2026-09-22: Backend GET /api/analytics/summary + sambung stats card Ringkasan
**Status:** ✅ Done  
**Files changed:**
- `backend/internal/db/db.go`: `AnalyticsSummary` struct (total_links/total_clicks/unique_clicks_30d/growth_pct) + `growthPct(last, prev)` helper (prev==0 → 0 atau 100); `ShardStore.AnalyticsSummary` di interface; `SingleStore.AnalyticsSummary` (2 aggregate query: COUNT FILTER + SUM link, lalu 3 window FILTER dari click_events JOIN urls 60 hari); `shardStore.AnalyticsSummary` (merge per-shard seperti ClicksByDay) + LEARN block
- `backend/internal/handler/handlers_analytics.go`: `HandleAnalyticsSummary`: session `middleware.CreatorID` didahulukan, fallback `creatorFromAPIKey` (Bearer) → 401 bila keduanya kosong; 401/500/JSON encode + LEARN dual-auth
- `backend/cmd/server/main.go`: route `GET /api/analytics/summary` pakai `OptionalAuth` (bukan RequireAuth) supaya API key bisa masuk; session tetap di-resolve ke context
- `backend/internal/handler/handler_test.go`: field `summary`/`summaryErr` + method stub `AnalyticsSummary` pada fakeStore
- `backend/internal/handler/handlers_analytics_test.go`: test baru: 401 tanpa auth, 200 + payload shape, 500 saat store error
- `frontend/app/api/analytics/summary/route.js`: proxy Next → Go, forward cookie (sama dengan clicks-by-day)
- `frontend/app/dashboard/RingkasanTab.jsx`: fetch `/api/analytics/summary` saat mount; 4 stats card kini pakai `summary.total_links/total_clicks/unique_clicks_30d/growth_pct` (fallback 0 saat loading/error); `loading = !profileReady || summary === null`
- `PROGRESS.md`: entry baru
**Why:** Stats card Ringkasan masih hard-code 0 untuk Klik Unik 30d & Growth (dan total dihitung client-side): perlu 1 endpoint agregat yang menghitung unique window 30 hari + growth 30 vs 30 sebelumnya di DB.
**Notes:** **Response JSON:** `{total_links, total_clicks, unique_clicks_30d, growth_pct}`. **Auth:** session cookie (dashboard) ATAU `Authorization: Bearer jjk_...`: route `OptionalAuth` + handler fallback API key. **Scope:** `creator_id` match di query WHERE (scoping di store). **Growth:** `(clicks[last30] - clicks[prev30]) / prev30 * 100` dibulatkan; prev==0 → 0 (tanpa baseline). **Unik 30d:** COUNT `click_events.is_unique` window 29 hari (bukan counter all-time). Verifikasi: `go build ./...` + `go test ./...` OK; `npm run build` 19/19 (route baru `/api/analytics/summary`). **Perlu dicek manual:** (1) login → 4 kartu terisi (bukan skeleton stuck); (2) growth +/- warna hijau/coral; (3) tanpa login → 401 (console); (4) build tetap hijau.

---

## 2026-09-22: Isi tab Ringkasan (greeting, stats, mini chart, top 5)
**Status:** ✅ Done  
**Files changed:**
- `frontend/app/dashboard/RingkasanTab.jsx`: komponen baru: greeting "Halo, {nama} 👋" (+ sub-teks); grid 4 stats card (1/2/4 col, gap-3/4): Total Link Aktif, Total Klik, Klik Unik (30 hari), Growth %: ikon lucide (Link/MousePointerClick/Users/TrendingUp|Down) box 40×40 rounded-8px bg flash-yellow/20, angka JetBrains Mono counter 800ms `EASE`, growth hijau `#22C55E` / flash-coral + tanda +/−; mini chart "Tren 30 Hari Terakhir" (garis polos h-120px, data `/api/analytics/clicks-by-day`, tanpa duplikasi chart penuh); "Link Terpopuler" top-5 by `click_count` (rank /kode / URL / klik)
- `frontend/app/dashboard/DashboardClient.jsx`: import + render `RingkasanTab`; state `profileReady` (true setelah GET /api/profile sukses) untuk skeleton vs data kosong
- `PROGRESS.md`: entry baru
**Why:** Tab Ringkasan masih placeholder dari prompt sebelumnya: perlu konten agregat (sapaan, KPI, tren mini, link terpopuler) tanpa endpoint backend baru.
**Notes:** **Data client-side:** link aktif & total klik dihitung dari `links` (state profile); tren dari endpoint existing `/api/analytics/clicks-by-day`; **Klik Unik (30 hari) = 0** dan **Growth % = 0** karena API belum punya unique-per-hari / window 60 hari (JANGAN bikin endpoint: angka naik saat data tersedia). Chart penuh TETAP hanya di tab Analytics (mini = visual beda, data sama). Skeleton saat `!profileReady`. **Perlu dicek manual** di 3 viewport (375/768/1440px): (1) greeting pakai nama display / fallback "Halo 👋"; (2) 4 kartu counter animasi 0→nilai; (3) grid 1→2→4 kolom; (4) mini chart render + empty state; (5) top-5 urut klik; (6) tab Analytics masih chart penuh; (7) build 18/18.

---

## 2026-09-22: Restructure dashboard tabs + fix spacing
**Status:** ✅ Done  
**Files changed:**
- `frontend/app/dashboard/DashboardClient.jsx`: tab default `ringkasan`; tab bar 5 item (Ringkasan | Link Saya | Analytics | Profil | API Keys) pill `border-2 border-ink rounded-full bg-white p-1 overflow-x-auto`, aktif `bg-flash-yellow font-bold`, non-aktif `text-ink/60 hover:text-ink`, transisi 150ms ease-out; wrapper `space-y-6 md:space-y-8`; tab bar `mb-6 md:mb-8`; konten tab `space-y-4`; AnimatePresence `mode="wait"` fade 150ms `EASE`; tab Ringkasan = placeholder + 4 skeleton kartu abu-abu; tab Analytics = ClicksChart + placeholder Top Link/Referrer "Segera hadir"; chart dihapus dari Link Saya; section heading `text-lg font-bold mb-4`; card padding `p-5 md:p-6`; list link/key `gap-4`
- `frontend/app/components/ClicksChart.jsx`: card `p-5 md:p-6`; heading `mb-4`; margin-top dinonaktifkan (diatur parent `space-y-4`)
- `frontend/app/components/ShortenForm.jsx`: hapus `mt-6` form & `mt-3` error/hasil (spacing diatur parent; landing tetap `mt-8` wrapper, `/app` dikasih wrapper `mt-6`)
- `frontend/app/app/page.jsx`: bungkus ShortenForm dengan `div.mt-6` (kompensasi hapus mt-6 internal)
- `PROGRESS.md`: entry baru
**Why:** Dashboard nempel tanpa napas (form/chart/list rapat); tab lama cuma 3 (Link Saya | Profil | API Keys) tanpa ringkasan/agregat; chart 30 hari campur di tab links.
**Notes:** **Tab Ringkasan masih placeholder** (teks + skeleton): stats card/chart/aktivitas diisi di task berikutnya. Tab Analytics: chart dipindah + Top Link/Referrer "Segera hadir" (JANGAN implement dulu). Profil & API Keys isi tidak diubah (hanya spacing). Logic shortener/drag-sort/edit/hapus/QR/salin, AuthModal, Navbar, warna/font/adaptive theme tidak diubah. Refresh selalu balik ke Ringkasan (state, bukan URL). **Perlu dicek manual** di 3 viewport (375/768/1440px): (1) `/dashboard` default Ringkasan + skeleton; (2) Link Saya → form + list TANPA chart; (3) Analytics → chart + 2 placeholder; (4) Profil/API Keys isinya sama; (5) spacing lega antar section/card; (6) 375px tab bar 1 baris scrollable horizontal, tidak wrap; (7) transisi tab fade 150ms; (8) build 18/18.

---

## 2026-09-22: Polish detail: focus ring, loading state, copy feedback
**Status:** ✅ Done  
**Files changed:**
- `frontend/app/components/SubmitButton.jsx`: komponen baru: `motion.button` type=submit, spinner SVG 16x16 `animate-spin`, tampil minimal 300ms (anti-flicker), `disabled={disabled || isLoading}`, `aria-busy`, `disabled:cursor-not-allowed disabled:opacity-70`
- `frontend/lib/useCopyToClipboard.js`: hook baru: return `{ copied, copy }`, status `null | "success" | "failed"`, reset 2000ms, `navigator.clipboard` + textarea `execCommand` fallback
- `frontend/app/components/CopyButton.jsx`: rewrite: pakai `useCopyToClipboard`, feedback inline "Tersalin" (lucide `Check`, bg flash-yellow) / "Gagal" (bg flash-coral), `aria-live="polite"`, toast bawah dihapus
- `frontend/app/components/ShortenForm.jsx`: focus ring `focus:border-flash-yellow focus:ring-2 focus:ring-flash-yellow/30`; SubmitButton loading "Memproses..."; CopyButton di result box
- `frontend/app/components/AuthModal.jsx`: focus ring; SubmitButton loading "Mendaftar..."/"Masuk..."
- `frontend/app/components/EditLinkModal.jsx`: focus ring; SubmitButton loading "Menyimpan..."
- `frontend/app/dashboard/DashboardClient.jsx`: focus ring input + tag filter select; SubmitButton ×2 ("Menyimpan...", "Membuat...")
- `frontend/app/page.jsx`: focus ring demo slug input; CopyButton di mockup `jejak.app/@username`; import CopyButton
- `PROGRESS.md`: entry baru
**Why:** Aksesibilitas & UX polish: input tanpa focus ring (keyboard nav kurang jelas), submit bisa double-click tanpa indikasi loading, copy feedback cuma toast lewat tanpa status gagal.
**Notes:** **Focus ring** (`focus:border-flash-yellow focus:ring-2 focus:ring-flash-yellow/30 focus:outline-none transition-all duration-150`) di 6 lokasi: ShortenForm, AuthModal, EditLinkModal, DashboardClient (inputThemed + select), page.jsx demo slug. **SubmitButton** wiring: ShortenForm "Memproses...", AuthModal conditional, EditLinkModal "Menyimpan...", DashboardClient "Simpan Profil"/"Buat Key Baru". **CopyButton** dipakai di DashboardClient ×2, ShortenForm result, QrModal, page.jsx mockup. **Perlu dicek manual** di 3 viewport (375/768/1440px): (1) Tab input → ring flash-yellow terlihat; (2) Shorten → spinner + "Memproses..." + no double-submit; (3) Salin → "Tersalin" 2s di /dashboard, /app, landing; (4) build 18/18.

---

## 2026-09-22: Rombak navbar logged-out (dropdown Fitur)
**Status:** ✅ Done  
**Files changed:**
- `frontend/app/components/NavbarClient.jsx`: "Fitur" jadi dropdown trigger (desktop) + accordion (drawer mobile); `FEATURE_ITEMS` (4 item: Custom Slug/Smart Link/QR Code/Analytics: ikon lucide SVG, semua anchor `/#fitur`); `PUBLIC_LINKS` tinggal Demo | FAQ; state/refs/effects untuk `fiturOpen` & `fiturAccOpen`
- `PROGRESS.md`: entry baru
**Why:** Navbar logged-out terlalu flat (Fitur | Demo | FAQ | Masuk | Daftar): tidak mengkomunikasikan value produk. Referensi: bit.ly pakai dropdown kategori ("Platform ▾", "Solutions ▾").
**Notes:** Build hijau `npm run build` 18/18. **Dropdown desktop:** klik trigger → panel fade + slide turun 8px (EASE array), klik luar / Escape / klik item → tutup, chevron rotate 180° saat open, `aria-expanded`/`aria-haspopup`/`aria-controls`, keyboard Tab/Enter/Escape; panel `bg-white border-2 border-ink shadow-[4px_4px_0px_#1C1A12]` radius 12px p-2; item hover `bg-flash-yellow/20` radius 8px p-10px, ikon SVG dalam kotak flash-yellow + label + deskripsi 1 baris (truncate). **Accordion mobile:** "Fitur" di drawer jadi collapsible (height auto, DEFAULT_TRANSITION), Demo/FAQ/Masuk/Daftar tetap langsung terlihat. **TIDAK diubah:** logged-in menu (Dashboard | + Link Baru | Avatar), adaptive theme `body[data-profile-theme]` (bar chrome: panel dropdown putih solid per spesifikasi, konten di-atas bar bukan background navbar), AuthModal, logo JEJAK, dropdown avatar. **Perlu dicek manual** di 3 viewport (375/768/1440px): (1) `/` tanpa login → navbar `Fitur ▾ | Demo | FAQ | Masuk | Daftar Gratis →`; (2) klik Fitur ▾ → 4 item muncul, klik "Custom Slug" → tutup + scroll `#fitur`, klik luar/Escape → tutup; (3) Masuk → AuthModal login, Daftar Gratis → AuthModal register; (4) mobile 375px → hamburger → drawer slide-in, Fitur accordion expand/isi 4 item, item lain langsung terlihat; (5) login → navbar berubah ke logged-in, tidak terpengaruh perubahan ini.

---

## 2026-09-22: Sederhanakan navbar logged-in
**Status:** ✅ Done  
**Files changed:**
- `frontend/app/components/NavbarClient.jsx`: array `AUTH_LINKS` tinggal `Dashboard` saja; "Link Saya" & "Analytics" dihapus dari desktop nav sekaligus mobile drawer (satu sumber array untuk keduanya)
- `PROGRESS.md`: entry baru
**Why:** Ketiga menu logged-in (Dashboard/Link Saya/Analytics) mengarah ke `/dashboard` yang sama: tidak ada fungsi unik, user bingung memilih menu yang identik.
**Notes:** Hapus "Link Saya" & "Analytics" dari navbar; **rencana:** keduanya dipindah ke dalam dashboard sebagai tab/section di task terpisah (JANGAN dikerjakan di task ini). Logged-out (Fitur/Demo/FAQ/Masuk/Daftar), styling, warna, adaptive theme, dan hamburger drawer TIDAK diubah: isi drawer logged-in tinggal Dashboard, divider, + Link Baru, Profil, Keluar. Build hijau 18/18. **Perlu dicek manual:** login → navbar hanya Dashboard | + Link Baru | Avatar; Dashboard → /dashboard; + Link Baru → /app; avatar dropdown Profil & Keluar; mobile 375px drawer sesuai isi di atas.

---

## 2026-09-22: Navbar auth-aware + scroll-margin
**Status:** ✅ Done  
**Files changed:**
- `frontend/app/components/Navbar.jsx`: refactor jadi Server Component: baca cookie `jejak_session` via `cookies()` dari `next/headers`, cek keberadaan cookie (MVP), pass `isLoggedIn` sebagai prop ke client
- `frontend/app/components/NavbarClient.jsx`: Client Component baru ("use client"): menu logged-out (Fitur/Demo/FAQ + Masuk/Daftar via AuthModal) vs logged-in (Dashboard/Link Saya/Analytics + Link Baru + avatar dropdown Profil/Keluar), hamburger drawer slide-in dari kanan (framer-motion, `EASE` dari lib/animations.js: array, bukan string), body scroll-lock, claim flow + toast dipindah utuh dari Navbar lama
- `frontend/app/page.jsx`: Task 2 SKIP: `#fitur`/`#demo`/`#faq` sudah punya `scroll-mt-24 md:scroll-mt-28` dari entry sebelumnya (verifikasi baris 309/341/389)
- `frontend/app/app/page.jsx`: hero `pt-14` → `pt-20`: navbar baru full-width `h-16` (64px) menutupi konten `/app` yang cuma kasih 56px (pill lama terpusat, lolos horizontal di kiri; bar penuh tidak)
- `PROGRESS.md`: entry baru
**Why:** Navbar hardcoded menu logged-in (state dari `localStorage.jejak_username`): pengunjung publik lihat `+ Link Baru`/`Dashboard`/avatar di landing, membingungkan dan duplikat ShortenForm; butuh auth-aware dari cookie via server + nav jual-fitur ala Bitly/TinyURL untuk logged-out.
**Notes:** Build hijau `npm run build` 18/18 (semua route `ƒ` dynamic: efek samping `cookies()` di root layout, expected). **Mapping route (adaptasi, tanpa route baru):** Masuk → AuthModal mode login; Daftar Gratis → AuthModal mode register; Dashboard/Link Saya/Analytics → `/dashboard` (tab belum di-drive URL); Link Baru → `/app`; Avatar dropdown → Profil (`/dashboard`) + Keluar; anchor `/#fitur|/#demo|/#faq` prefix `/` dari halaman mana pun. **Keputusan:** adaptasi tema via `body[data-profile-theme]` DIPERTAHANKAN (bar chrome ikut tema: classic/coral/darkroom/glass: default Instant Print `bg-print-white/80 backdrop-blur-xl border-b-2 border-ink` hanya saat tanpa attribute); `router.refresh()` dipakai setelah login sukses & logout untuk force re-render Navbar server. **Perlu dicek manual** di 3 viewport (375/768/1440px): logged-out landing (Fitur/Demo/FAQ/Masuk/Daftar), AuthModal terbuka dari desktop & drawer mobile, login → navbar berubah + tetap ikut tema /dashboard, avatar dropdown, `/u/[username]` ikut tema profil, logout → balik logged-out, smooth scroll anchor dari /dashboard ke `/#fitur`.

---

## 2026-09-22: Fix overlap navbar + setup PROGRESS.md
**Status:** ✅ Done  
**Files changed:**
- `frontend/app/page.jsx`: hero `pt-14 lg:pt-20` → `pt-24 md:pt-28 lg:pt-32` + `z-10`; section anchor (`#fitur`, `#demo`, `#faq`) dikasih `id` + `scroll-mt-24 md:scroll-mt-28`
- `frontend/app/globals.css`: tambah `html { scroll-behavior: smooth }`
- `PROGRESS.md`: file baru, progress log (entry pertama & kedua)
- `RULES.md`: tambah section konvensi update PROGRESS.md
**Why:** Navbar pill `fixed top-5 z-50` (tepi bawah ±72px) menutupi badge hero mobile (`pt-14`=56px < 72px): gap cuma ±16px; anchor link tanpa `scroll-mt` juga nembak konten di bawah navbar; butuh log perubahan yang append-only.
**Notes:** Build hijau 18/18 (`npm run build`). Tinggi navbar TIDAK diubah (sesuai constraint). **Perlu dicek manual** di 3 viewport (375/768/1440px): badge hero tidak ketutup, anchor #fitur/#demo/#faq smooth scroll.

---

## 2026-09-21: Rombak total landing page (9 section mobile-first)
**Status:** ✅ Done  
**Files changed:**
- `frontend/app/page.jsx` (576 baris): 9 section: Navbar, Hero, Social Proof, Fitur, Demo, Testimonial, FAQ, CTA, Footer. Hero pakai `ShortenForm` FUNGSIONAL (POST `/api/shorten` via proxy)
- `frontend/lib/animations.js` (53 baris): shared framer-motion variants: `fadeUp`, `staggerContainer`, `staggerItem`, `faqChevron`, `EASE`, `DEFAULT_TRANSITION`
**Why:** Landing page lama kurang optimal untuk konversi & tidak mobile-first; butuh satu sumber animasi (framer-motion) ikut design system "Instant Print".
**Notes:** Build hijau 18/18, `/` = 6.93 kB JS. Easing `EASE` di lib/animations.js sempat error runtime `"Invalid easing type 'cubic-bezier(...)'"` → sudah diperbaiki jadi array `[0.22, 1, 0.36, 1]` (Motion hanya terima nama easing / array 4 angka / fungsi: JANGAN string CSS). Landing selalu pakai tema `classic` (`themeStyles("classic")`), konten dummy statis.
