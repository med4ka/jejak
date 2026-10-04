-- +goose Down
-- Removes the health/password columns (migration 17) and the notifications
-- feed. Dropping password_hash silently removes the gate from any link that
-- had one - that is the intended rollback behaviour.
DROP INDEX IF EXISTS idx_notifications_creator;
DROP TABLE IF EXISTS notifications;
DROP INDEX IF EXISTS idx_urls_health_broken;
DROP INDEX IF EXISTS idx_urls_health_due;
ALTER TABLE urls DROP COLUMN IF EXISTS password_hash;
ALTER TABLE urls DROP COLUMN IF EXISTS health_notified_at;
ALTER TABLE urls DROP COLUMN IF EXISTS fallback_url;
ALTER TABLE urls DROP COLUMN IF EXISTS last_health_check;
ALTER TABLE urls DROP COLUMN IF EXISTS health_status;
