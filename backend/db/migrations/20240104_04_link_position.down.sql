-- ROLLBACK MANUAL ONLY (reversibel).
-- Jangan dijalankan via alur migrasi normal: DROP COLUMN menghapus data
-- urutan permanen. Jalankan hanya kalau sadar risikonya:
--   psql -U postgres -h localhost -d jejak -f backend/db/migrations/20240104_04_link_position.down.sql

DROP INDEX IF EXISTS idx_urls_creator_position;
ALTER TABLE urls DROP COLUMN IF EXISTS position;
