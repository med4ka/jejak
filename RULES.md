# RULES.md — Jejak

> Hidup di root repo, dibaca otomatis oleh coding agent (OpenCode). Konvensi teknis spesifik project ini — persona/gaya komunikasi & aturan paling penting ada di SYSTEM.md §1, baca itu dulu.

## 1. Kontrak "AI Cuma Implementasi" (paling penting di project ini)

Sebelum AI menulis SATU baris kode untuk fitur baru, checklist ini WAJIB terpenuhi:

1. ☐ The developer sudah cerita rancangannya sendiri (pendekatan + alasan) — bukan AI yang nanya "mau pendekatan apa" lalu AI juga yang jawab sendiri
2. ☐ Rancangan itu SPESIFIK — bukan "pakai cache" doang, tapi "cache-aside, TTL 5 menit, alasan: data URL jarang berubah setelah dibuat"
3. ☐ Kalau ada bagian yang the developer belum putuskan tapi dibutuhkan kode (misal: apa yang terjadi kalau cache miss dan DB juga down?), AI WAJIB tanya balik, BUKAN isi dengan asumsi "best practice"

Kalau the developer minta kode tanpa checklist ini terpenuhi, respons AI yang benar adalah: **"Ceritain dulu rancangan kamu — pendekatan apa dan kenapa, baru aku bantu implementasi."** Bukan langsung ngoding "yang penting jalan".

## 2. Konvensi Kode Go

- Error handling eksplisit, jangan `_ = err` kecuali benar-benar sengaja dan dikasih komentar kenapa
- Package per domain (`handler`, `shortener`, `cache`, `db`) — jangan taruh semua logic di `main.go`
- Docstring/komentar di fungsi non-obvious menjelaskan APA dan KENAPA, bukan textbook
- Test minimal untuk: logic generate short-code (collision handling), dan handler utama (shorten, redirect)

## 2a. Komentar `// LEARN:` (belajar cepat sambil jalan project)

Setiap fungsi yang mengandung 1 konsep system design baru (lihat PRD.md §1, kolom "Konsep") WAJIB dikasih blok komentar ini oleh AI, format persis:

```go
// LEARN:
//   Kenapa: <1 kalimat, apa yang fungsi ini coba selesaikan>
//   Trade-off: <1-2 kalimat, apa yang dikorbankan/risiko>
//   Alternatif: <1 kalimat, cara lain yang mungkin dipakai>
```

Aturan pakai (biar tetap belajar, bukan sekadar dibaca lewat):
- Taruh HANYA di fungsi yang punya nilai konsep baru — jangan di semua fungsi (getter/setter sepele gak perlu), biar gak jadi noise
- Bahasa Indonesia (beda dari komentar biasa yang Bahasa Inggris) — sengaja beda biar keliatan jelas ini "catatan belajar", bukan dokumentasi production
- **Gate sebelum lanjut fase:** the developer WAJIB baca semua `// LEARN:` di fase yang baru selesai, lalu tulis ulang pemahamannya pakai kata sendiri di PROGRESS.md bagian Refleksi (boleh singkat, 2-3 kalimat per konsep) — SEBELUM minta AI lanjut ke fase berikutnya
- Kalau project ini nanti mau dipajang di portofolio/CV, tag `// LEARN:` gampang di-strip semua (cari-replace) tanpa ganggu kode aslinya

## 3. Keputusan yang Sudah Diambil (jangan diusulkan ulang oleh AI)

- **Fase dikerjakan berurutan** (PRD.md §1) — AI tidak boleh menyarankan loncat fase "biar lebih efisien"
- **Load test dulu sebelum optimasi** (Fase 1) — jangan optimasi berdasarkan tebakan, semua keputusan optimasi harus based on angka baseline
- **Docker Compose untuk simulasi lokal**, bukan deploy ke cloud beneran di fase awal (ARCHITECTURE.md §2)
- **Queue pakai Redis, bukan Kafka/RabbitMQ** — sengaja ringan, jangan diusulkan upgrade infra kecuali the developer eksplisit minta di fase lanjutan

## 4. Checkpoint & Sesi Kerja

- Satu sesi kerja = satu fase/satu fitur (SYSTEM.md §1 poin 5) — jangan gabung Fase 3 dan Fase 4 dalam satu sesi meskipun teknis bisa
- Setelah 1 fase selesai, AI mengingatkan the developer untuk isi bagian refleksi (lihat PRD.md — apa yang dipelajari, apa yang ternyata salah rancang) sebelum lanjut fase berikutnya
- Update primer istilah di PRD.md §2 setiap kali istilah baru muncul (SYSTEM.md §1 poin 4)

## 5. Yang Perlu Direview Manual oleh the developer (prioritas tinggi)

- SEMUA keputusan arsitektur per fase (lihat ARCHITECTURE.md §4) — ini bukan opsional, ini inti dari project
- Hasil load test tiap fase — pastikan benar-benar dibandingkan dengan baseline Fase 0/1, bukan cuma "kelihatannya lebih cepat"
- Trade-off consistency vs speed di `urls.click_count` (SCHEMA.md) — the developer harus bisa jelasin sendiri kenapa itu acceptable atau tidak untuk use case ini