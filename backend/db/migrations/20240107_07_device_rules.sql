-- +goose Up
-- Smart Link (feature phase): one short link can route to a different URL per
-- device. JSONB format: {"ios": "https://...", "android": "https://..."}.
-- Keys are optional; when the device matches no key (or the rules are empty),
-- the fallback is original_url. The default '{}' keeps existing rows
-- immediately compatible (backward compatible, old redirects keep working
-- unchanged). Run on primary AND replica.

ALTER TABLE urls ADD COLUMN IF NOT EXISTS device_rules JSONB NULL DEFAULT '{}';