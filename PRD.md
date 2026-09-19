# PRD.md — Jejak

## 0. Apa Ini

URL shortener + click analytics, direposisi sebagai **link-in-bio untuk kreator/influencer** (mirip Linktree, tapi backend-nya dibangun sendiri) — 1 kreator punya 1 halaman profil publik berisi banyak short-link, masing-masing dengan analytics klik sendiri-sendiri.

Produk ini **sengaja dimulai simpel** (Fase 0-8, single-link shortener) — value awalnya bukan di fitur, tapi di tiap fase pembangunan yang memaksa praktik 1 konsep system design. Fase 9 (profil kreator) adalah lapisan produk yang dibangun DI ATAS fondasi itu, bukan pengganti — semua konsep system design dari Fase 0-8 tetap dipertahankan & dipakai fitur baru ini (lihat DESIGN.md untuk identitas visual).

## 1. Fase Pembangunan (urutan wajib, jangan diloncat)

| Fase | Fitur | Konsep System Design yang Dipraktikkan |
|---|---|---|
| **0** | Baseline naif: 1 server, shorten + redirect, tanpa optimasi apapun | Ngerasain masalahnya dulu sebelum mikirin solusinya. Sengaja dibiarkan jadi SPOF |
| **1** | Load testing baseline Fase 0 (pakai tool sederhana, mis. `hey` atau `k6`) | Mengukur baseline SEBELUM optimasi — kebiasaan penting yang sering diskip pemula |
| **2** | Tambah caching di jalur redirect | Cache-aside vs write-through, TTL, cache invalidation |
| **3** | Multiple instance API + load balancer (lokal, mis. Nginx atau HAProxy) | Horizontal scaling, load balancing algorithm (round robin vs least connection) |
| **4** | Read replica database | Redundancy, read/write splitting, replication lag |
| **5** | Sharding tabel `urls` (partisi berdasarkan hash short-code) | Sharding — pembagian data, bukan penyalinan (beda dari redundancy) |
| **6** | Async click logging (queue, bukan langsung tulis ke DB pas redirect) | Decoupling read-path dari write-path, async processing, kenapa redirect gak boleh nunggu logging selesai |
| **7** | Rate limiting di endpoint shorten | Melindungi sistem dari abuse — praktik langsung, bukan cuma teori |
| **8 (opsional)** | Dashboard analytics (Next.js) baca dari read replica | Menyatukan semua fase jadi 1 alur baca yang koheren |
| **9** | Profil kreator: 1 akun kreator → banyak short-link, halaman publik `/​u/{username}` menampilkan semua link-nya (lihat SCHEMA.md untuk model data baru) | Data modeling (relasi 1-ke-banyak), auth sederhana (kepemilikan link), N+1 query awareness saat load semua link + click count sekaligus |

**Prinsip fase:** satu fase = satu concept, dan the developer WAJIB bisa jelasin sendiri kenapa fase itu dibutuhkan sebelum masuk ke implementasi (lihat SYSTEM.md §5).

## 2. Primer Istilah System Design

> Dokumen hidup — setiap istilah baru yang muncul selama pembangunan project ini WAJIB ditambahkan ke sini dengan penjelasan singkat (SYSTEM.md §1 poin 4).

| Istilah | Penjelasan Singkat |
|---|---|
| **SPOF (Single Point of Failure)** | Komponen tunggal yang kalau mati, seluruh sistem ikut mati. Fase 0 di project ini SENGAJA dibuat SPOF supaya kerasa masalahnya |
| **Horizontal Scaling (Scale Out)** | Nambah JUMLAH mesin/instance untuk nambah kapasitas (bukan nambah spek 1 mesin) |
| **Vertical Scaling (Scale Up)** | Nambah kapasitas 1 mesin yang sama (RAM/CPU lebih besar) |
| **Load Balancer** | Komponen yang membagi trafik masuk ke banyak instance server, supaya beban merata & tidak ada 1 titik kebanjiran request |
| **Redundancy** | Menyalin (duplikasi) komponen/data supaya ada cadangan kalau salah satu gagal |
| **Sharding** | Membagi data ke beberapa tempat berbeda (bukan menyalin) — supaya 1 database tidak menanggung semua beban |
| **Cache-aside** | Pola caching: aplikasi cek cache dulu, kalau miss baru query DB lalu isi cache. Paling umum dipakai |
| **Cache invalidation** | Proses menghapus/update cache saat data aslinya berubah — sering disebut salah satu hal tersulit di computer science |
| **Read Replica** | Salinan database yang hanya untuk dibaca (read-only), mengurangi beban baca dari database utama (primary) |
| **Replication Lag** | Jeda waktu antara data ditulis di primary dan muncul di replica — kenapa replica gak selalu 100% up-to-date |
| **Rate Limiting** | Membatasi jumlah request yang boleh dilakukan 1 client dalam periode waktu tertentu, melindungi sistem dari abuse/overload |
| **N+1 Query** | Bug performa klasik: query 1x buat ambil list (misal 20 link), lalu query TERPISAH sebanyak 20x lagi buat ambil data terkait tiap item (misal click count per link) — total 21 query padahal bisa 1-2 query pakai JOIN/batch. Relevan banget di Fase 9 pas load halaman profil kreator |

## 3. Non-Goals

- Bukan produk komersial — jangan optimasi buat "scale ke jutaan user beneran", cukup simulasikan konsepnya di skala kecil (load testing lokal)
- Autentikasi TETAP sederhana meski Fase 9 butuh kepemilikan link (1 kreator = 1 akun) — cukup buat nentuin "link ini punya siapa", BUKAN sistem auth kompleks (gak perlu role granular, OAuth multi-provider, dst)
- Tidak perlu deployment ke cloud beneran di fase awal — semua fase 0-9 bisa disimulasikan lokal
