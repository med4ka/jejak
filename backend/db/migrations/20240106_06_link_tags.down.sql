-- MANUAL ROLLBACK ONLY (reversible).
-- Do not run it through the normal migration flow: DROP COLUMN permanently
-- deletes the tag data. Run it only when the risk is understood:
--   psql -U postgres -h localhost -d jejak -f backend/db/migrations/20240106_06_link_tags.down.sql

ALTER TABLE urls DROP COLUMN IF EXISTS tags;
