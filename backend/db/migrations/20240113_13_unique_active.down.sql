-- +goose Down
ALTER TABLE urls DROP COLUMN IF EXISTS unique_click_count;
ALTER TABLE urls DROP COLUMN IF EXISTS is_active;

ALTER TABLE click_events DROP COLUMN IF EXISTS is_unique;
ALTER TABLE click_events DROP COLUMN IF EXISTS referrer_domain;

DROP INDEX IF EXISTS idx_clicks_short_code_clicked_at;