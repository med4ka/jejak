# DESIGN.md — Jejak Design System

## 0. IDENTITAS: "INSTANT PRINT, BUKAN SAAS DASHBOARD"

Kesalahan paling umum di tool "link-in-bio" bikinan sendiri (atau bikinan AI): niru Linktree generic — gradient ungu-biru, card putih melayang dengan shadow lembut, font Inter/Poppins. Kesalahan generasi BARU (2026): lari ke arah lain yang sama-sama jadi cliche — background krem, font serif elegan, aksen hijau sage. Keduanya sama-sama "rata-rata statistik", bukan keputusan desain.

Jejak berangkat dari sudut pandang berbeda: dunia kreator itu identik dengan **foto instan** (Polaroid/Instax) — bingkai putih tebal, warna hasil cetak yang solid & jenuh (bukan pastel lembut), tulisan spidol di margin bawah foto, momen yang "baru dicetak" dan langsung dibagikan. Jejak mengambil bahasa visual itu — 1 link = 1 "cetakan" yang kreator bagikan ke publik.

**DILARANG KERAS:**
- Background gelap sebagai default (ini keputusan desain baru — Jejak versi terang, bukan panggung gelap)
- Gradient ungu-ke-biru sebagai identitas (klise lama)
- Background krem + font serif + aksen hijau sage (klise BARU 2026 — "anti-slop" yang jadi generic lagi)
- Font Inter atau Poppins sebagai typeface utama
- Card dengan shadow lembut generic — pakai border tebal, bukan shadow
- Emoji sebagai pengganti ikon asli

## 1. Palet Warna

### Base ("Instant Print", default & utama)
| Token | Hex | Fungsi |
|---|---|---|
| `print-white` | `#FAFAF7` | Background utama — putih bersih, bukan krem hangat |
| `paper-grey` | `#EDEBE4` | Background sekunder/section alternatif, bukan card utama |
| `ink` | `#1C1A12` | Teks utama + border tebal (pengganti shadow) |
| `muted` | `#8A8578` | Teks sekunder |

### Aksen (warna hasil cetak — solid & jenuh, bukan pastel)
| Token | Hex | Fungsi |
|---|---|---|
| `flash-yellow` | `#FFD23F` | Aksen UTAMA — CTA, highlight, tombol aksi utama. Warna "flash foto instan" |
| `flash-coral` | `#FF5C3D` | Aksen SEKUNDER — HANYA untuk badge "trending"/status/CTA sekunder, dipakai langka. Warna "merah-oranye lampu flash", pengganti instax-blue (versi lama) |

**Catatan eksplisit:** border `ink` 2px dipakai MENGGANTIKAN shadow di semua card/tombol penting — ini yang bikin identitas "instant print" kerasa (bingkai foto tebal), bukan card melayang generic.

## 2. Tipografi

| Peran | Font | Alasan |
|---|---|---|
| Heading/Display | **Space Grotesk** (bold/700) | Karakter modern-teknikal, BUKAN Poppins/Montserrat, BUKAN serif elegan (klise baru) |
| Body/UI | **Work Sans** | Netral, humanist, BUKAN Inter |
| Angka (click count, follower count, stats) | **JetBrains Mono** | Tabular figures, presisi visual |
| Caption kecil (§3, dipakai SANGAT terbatas) | **Caveat** atau **Permanent Marker** | Meniru tulisan spidol di margin foto instan — HANYA untuk 1-2 elemen kecil, bukan teks panjang |

## 3. Motif "Instant Frame" (Signature Element, Dipakai Terbatas)

Border putih tebal (`print-white` dengan `ink` 2px) mengelilingi card link individual, dengan sedikit ruang kosong di bagian bawah card (mirip margin bawah foto Polaroid) berisi caption kecil pakai font tulisan-tangan (§2) — HANYA di card link dan di header halaman profil kreator. **Hanya 2 tempat ini** — kalau dipakai di semua elemen, kehilangan makna sebagai signature.

## 4. Navigasi: Capsule Floating Bar

Top bar TIDAK menempel ke tepi atas viewport — melayang dengan jarak (`margin-top` ~16-24px), bentuk pill sepenuhnya, lebar tidak penuh (max-width, centered horizontal).

**Spesifikasi (update warna, struktur tetap sama):**
- Background: `print-white` solid (bukan transparan/blur gelap)
- Border: `ink` 2px solid tegas (pengganti shadow — konsisten sama prinsip §1)
- Shadow: TIDAK PAKAI shadow sama sekali — border tebal sudah cukup memberi "angkat" visual
- Posisi: `position: fixed`, `top: 20px`, horizontal center, `max-width` sekitar 640-720px
- Saat scroll: border sedikit menebal (2px → 2.5px) dan padding mengecil ~20% — transisi halus 200ms
- Isi: logo/nama kiri, CTA (`flash-yellow` solid button, border `ink`) kanan

## 5. Layout & Komponen

- **Halaman profil publik kreator**: mobile-first, 1 kolom, card link disusun vertikal dengan sedikit rotasi acak halus (±1-2 derajat per card, seperti foto ditempel manual) — efek "scrapbook", bukan grid kaku sempurna. Background `print-white`.
- **Dashboard analytics**: background `paper-grey` (beda dari halaman publik biar kerasa "mode kerja"), angka pakai `JetBrains Mono`, chart klik pakai `flash-yellow` sebagai warna garis utama.
- **Badge status** ("trending"): pill kecil, background `flash-coral`, teks `print-white`, HANYA untuk 1 link paling top performing.

## 6. Animasi (Framer Motion)

Riset "AI slop" 2026 secara eksplisit nyebut **"bounce animation yang gak ada yang minta"** sebagai salah satu ciri paling gampang ketauan generic. Prinsip animasi Jejak:

- **Easing tegas (ease-out), bukan spring dengan overshoot besar** — gerakan harus terasa "klik" instan (seperti kamera Polaroid nge-print), bukan mantul-mantul lucu
- **Durasi pendek**: 150-250ms untuk micro-interaction (hover, tap), maksimal 400ms untuk entrance animation
- **Entrance halaman profil**: card link muncul dengan stagger (jeda antar card ~50-80ms), animasi `opacity` + sedikit `translateY` (8-12px) — meniru foto yang "keluar" dari kamera, BUKAN slide-in dari samping atau zoom dramatis
- **Tap/klik tombol**: scale-down halus (0.97-0.98), bukan bounce
- **JANGAN**: animasi hover yang gak ada fungsinya (card muter 3D random, ikon goyang sendiri tanpa interaksi user), parallax berlebihan, confetti/particle effect default

## 7. Nada Konten (Tone)

Energik & to-the-point — audiensnya kreator, bukan analis finance/operasional gudang. Tetap hindari korporat-generic ("Unlock your potential!", "Elevate your brand!") yang justru kedengaran AI-generated.

## 8. Aksesibilitas
- Kontras `ink` di atas `print-white` dan `paper-grey` — dicek rasio minimal 4.5:1 (kombinasi ini harusnya jauh di atas minimum, hitam pekat di atas putih bersih)
- `flash-yellow` sebagai warna TEKS tidak boleh dipakai di atas `print-white` (kontras kurang) — hanya untuk background elemen besar/CTA dengan teks `ink` di atasnya
- Chart analytics tidak boleh HANYA mengandalkan warna — sertakan label angka langsung