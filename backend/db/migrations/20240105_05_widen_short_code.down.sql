-- ROLLBACK MANUAL ONLY (reversibel).
-- Jangan dijalankan via alur migrasi normal: menyempitkan ke VARCHAR(10)
-- GAGAL kalau sudah ada baris >10 char (cek dulu!), dan memutus custom slug.
--   psql -U postgres -h localhost -d jejak -f backend/db/migrations/20240105_05_widen_short_code.down.sql

ALTER TABLE click_events ALTER COLUMN short_code TYPE VARCHAR(10);
ALTER TABLE urls ALTER COLUMN short_code TYPE VARCHAR(10);
