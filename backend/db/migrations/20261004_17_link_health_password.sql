-- +goose Up
-- Link health monitor + password protection (2026-10-04):
-- 1) Health columns: the worker checks the destination with HEAD every 6
--    hours and stores unknown/healthy/broken/timeout. fallback_url is where
--    visitors are sent (via a 1s interstitial) while the destination is
--    broken; health_notified_at marks that the "link is broken" in-app
--    notification was already created, so a link that stays broken for days
--    does not spam the dashboard (it resets when the link heals).
-- 2) password_hash: bcrypt hash of the optional per-link password. When set,
--    /r/{code} shows a form first and only redirects after a correct
--    password (cookie jejak_link_access_{code}, 1 hour).
--    SHA-256 of this hash is what goes into the cookie, never the hash
--    itself (bcrypt hashes contain "/" which is not a valid cookie-octet).
-- 3) notifications: in-app notification feed for the navbar bell (email
--    delivery does not exist in this codebase yet - see PROGRESS TODO).
-- TIMESTAMP columns follow the created_at/UTC convention used everywhere.
ALTER TABLE urls ADD COLUMN IF NOT EXISTS health_status VARCHAR(16) NOT NULL DEFAULT 'unknown';
ALTER TABLE urls ADD COLUMN IF NOT EXISTS last_health_check TIMESTAMP NULL;
ALTER TABLE urls ADD COLUMN IF NOT EXISTS fallback_url TEXT NOT NULL DEFAULT '';
ALTER TABLE urls ADD COLUMN IF NOT EXISTS health_notified_at TIMESTAMP NULL;
ALTER TABLE urls ADD COLUMN IF NOT EXISTS password_hash TEXT NOT NULL DEFAULT '';

-- Partial index: the worker only ever looks for links that have not been
-- checked in the current window (last_health_check IS NULL first, then the
-- oldest check). The second partial index backs the "broken links" badge
-- scan on the dashboard.
CREATE INDEX IF NOT EXISTS idx_urls_health_due ON urls(last_health_check NULLS FIRST)
    WHERE is_active AND health_status <> '';
CREATE INDEX IF NOT EXISTS idx_urls_health_broken ON urls(short_code)
    WHERE health_status = 'broken';

-- In-app notifications (navbar bell). Linked to creators on the primary DB,
-- same pattern as the creators table itself (shard 0 in shard mode).
CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    creator_id BIGINT NOT NULL REFERENCES creators(id) ON DELETE CASCADE,
    type VARCHAR(32) NOT NULL,
    short_code VARCHAR(64) NOT NULL DEFAULT '',
    message TEXT NOT NULL,
    read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc')
);
CREATE INDEX IF NOT EXISTS idx_notifications_creator ON notifications(creator_id, read, created_at DESC);
