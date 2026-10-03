-- +goose Down
-- Drops the three analytics dimension columns added by migration 16.
ALTER TABLE click_events DROP COLUMN IF EXISTS referrer_type;
ALTER TABLE click_events DROP COLUMN IF EXISTS device_type;
ALTER TABLE click_events DROP COLUMN IF EXISTS user_agent;
