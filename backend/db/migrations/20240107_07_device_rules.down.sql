-- MANUAL ROLLBACK ONLY (reversible).
-- Do not run it through the normal migration flow: DROP COLUMN permanently
-- deletes the device_rules data. Run it only when the risk is understood:
--   psql -U postgres -h localhost -d jejak -f backend/db/migrations/20240107_07_device_rules.down.sql

ALTER TABLE urls DROP COLUMN IF EXISTS device_rules;