-- ROLLBACK MANUAL ONLY (reversibel).
-- Jangan dijalankan via alur migrasi normal: DROP COLUMN menghapus data
-- avatar & sosial secara permanen. Jalankan hanya kalau sadar risikonya:
--   psql -U postgres -h localhost -d jejak -f backend/db/migrations/20240103_03_creator_profile.down.sql

ALTER TABLE creators DROP COLUMN IF EXISTS socials;
ALTER TABLE creators DROP COLUMN IF EXISTS avatar_url;
