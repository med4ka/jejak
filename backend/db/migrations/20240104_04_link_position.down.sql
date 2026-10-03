-- MANUAL ROLLBACK ONLY (reversible).
-- Do not run it through the normal migration flow: DROP COLUMN permanently
-- deletes the ordering data. Run it only when the risk is understood:
--   psql -U postgres -h localhost -d jejak -f backend/db/migrations/20240104_04_link_position.down.sql

DROP INDEX IF EXISTS idx_urls_creator_position;
ALTER TABLE urls DROP COLUMN IF EXISTS position;
