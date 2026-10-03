-- MANUAL ROLLBACK ONLY (reversible).
-- Do not run it through the normal migration flow: DROP COLUMN permanently
-- deletes the avatar and social data. Run it only when the risk is understood:
--   psql -U postgres -h localhost -d jejak -f backend/db/migrations/20240103_03_creator_profile.down.sql

ALTER TABLE creators DROP COLUMN IF EXISTS socials;
ALTER TABLE creators DROP COLUMN IF EXISTS avatar_url;
