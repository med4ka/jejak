-- ROLLBACK MANUAL ONLY (reversibel).
-- Jangan dijalankan via alur migrasi normal: DROP COLUMN menghapus data
-- device_rules permanen. Jalankan hanya kalau sadar risikonya:
--   psql -U postgres -h localhost -d jejak -f backend/db/migrations/20240107_07_device_rules.down.sql

ALTER TABLE urls DROP COLUMN IF EXISTS device_rules;