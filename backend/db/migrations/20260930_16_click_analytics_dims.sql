-- +goose Up
-- Analytics depth (2026-09-30): click dimensions for dashboard breakdowns.
-- 1. user_agent: the raw UA is truncated to 512 CHARACTERS (Postgres
--    VARCHAR(512) counts characters, not bytes - the handler truncates at a
--    rune boundary so no multi-byte rune is split during encoding).
-- 2. device_type: classified at WRITE time (not at read time) so it can be
--    queried directly without a full regex per request. Values: bot|tablet|
--    mobile|desktop|unknown. Old rows are backfilled with 'unknown' - honest
--    reclassification of history is impossible because the raw UA was not
--    always stored (the user_agent column itself only arrives with this
--    migration).
-- 3. referrer_type: referrer bucket computed at write time:
--    direct|search|social|chat|other. Unlike referrer_domain (raw host,
--    migration 13), this column is ready for a breakdown card with no
--    per-row host normalization.
-- All columns are NULLABLE so old rows stay valid and old INSERTs keep
-- working.

ALTER TABLE click_events ADD COLUMN IF NOT EXISTS user_agent VARCHAR(512) NULL;
ALTER TABLE click_events ADD COLUMN IF NOT EXISTS device_type VARCHAR(32) NULL;
ALTER TABLE click_events ADD COLUMN IF NOT EXISTS referrer_type VARCHAR(32) NULL;

-- Light backfill: old device_type is unknown (honestly 'unknown');
-- referrer_type can be guessed from referrer_domain, available since
-- migration 13. An empty referrer = 'direct', known hosts go to their bucket,
-- the rest to 'other' - the same classification used in the Go code
-- (classifyReferrer).
UPDATE click_events
   SET device_type = 'unknown'
 WHERE device_type IS NULL;

UPDATE click_events
   SET referrer_type = CASE
         WHEN referrer_domain IS NULL OR referrer_domain = '' THEN 'direct'
         WHEN referrer_domain IN ('www.google.com', 'google.com', 'www.bing.com', 'bing.com', 'duckduckgo.com', 'search.yahoo.com') THEN 'search'
         WHEN referrer_domain IN ('l.facebook.com', 'www.facebook.com', 'fb.me', 't.co', 'x.com', 'www.instagram.com') THEN 'social'
         ELSE 'other'
       END
 WHERE referrer_type IS NULL;
