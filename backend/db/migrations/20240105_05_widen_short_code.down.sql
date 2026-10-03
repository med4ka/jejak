-- MANUAL ROLLBACK ONLY (reversible).
-- Do not run it through the normal migration flow: narrowing back to
-- VARCHAR(10) FAILS when any row is longer than 10 chars (check first!) and
-- breaks custom slugs.
--   psql -U postgres -h localhost -d jejak -f backend/db/migrations/20240105_05_widen_short_code.down.sql

ALTER TABLE click_events ALTER COLUMN short_code TYPE VARCHAR(10);
ALTER TABLE urls ALTER COLUMN short_code TYPE VARCHAR(10);
