-- ROLLBACK MANUAL ONLY (reversibel).
-- Jangan dijalankan via alur migrasi normal: DROP COLUMN menghapus data
-- tag permanen. Jalankan hanya kalau sadar risikonya:
--   psql -U postgres -h localhost -d jejak -f backend/db/migrations/20240106_06_link_tags.down.sql

ALTER TABLE urls DROP COLUMN IF EXISTS tags;
