-- +goose Up
-- Fase 13 (improvement pass #2): analytics jujur + lifecycle link.
-- 1. unique_click_count: counter harian-unik per link (dedup by IP+UA, window 24h).
--    click_count tetap = total klik manusia; unique_click_count = klik dari visitor
--    berbeda. Ini meniru "total clicks vs unique clicks" punya bit.ly tanpa
--    menambah tabel visitor: flag is_unique diflush dari handler via Redis SETNX.
-- 2. is_active: hard-disable link tanpa menghapus (mirip bit.ly "deactivate"):
--    redirect => 410 Gone, halaman publik menyembunyikannya, dashboard bisa
--    menyalakan lagi. Hapus permanen tetap ada (DELETE endpoint baru).
-- 3. click_events.is_unique + referrer_domain: menyimpan metadata klik yang
--    sebelumnya langsung dibuang — analitik bisa nanti breakdown referrer/source.
-- 4. klik_events.clicked_at sekarang diisi EKSPLISIT dari worker (sebelumnya
--    DEFAULT CURRENT_TIMESTAMP = waktu PROSES, bukan waktu klik sebenarnya;
--    menggeser chart 30-hari saat backlog). Kolom sudah ada, cukup dipakai header.
-- Index compound (short_code, clicked_at) mempercepat query ClicksByDay
-- yang selalu filter 30 hari + join per short_code.

ALTER TABLE urls ADD COLUMN unique_click_count BIGINT NOT NULL DEFAULT 0;
ALTER TABLE urls ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE click_events ADD COLUMN is_unique BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE click_events ADD COLUMN referrer_domain VARCHAR(255) NULL;

CREATE INDEX IF NOT EXISTS idx_clicks_short_code_clicked_at ON click_events(short_code, clicked_at);