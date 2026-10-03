-- +goose Down
-- Drops the Fase 13 additions: unique_click_count and is_active on urls,
-- is_unique and referrer_domain on click_events, and the compound
-- (short_code, clicked_at) index.
ALTER TABLE urls DROP COLUMN IF EXISTS unique_click_count;
ALTER TABLE urls DROP COLUMN IF EXISTS is_active;

ALTER TABLE click_events DROP COLUMN IF EXISTS is_unique;
ALTER TABLE click_events DROP COLUMN IF EXISTS referrer_domain;

DROP INDEX IF EXISTS idx_clicks_short_code_clicked_at;