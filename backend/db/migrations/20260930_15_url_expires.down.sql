-- +goose Down
-- Removes the expires_at column and its partial index (migration 15).
DROP INDEX IF EXISTS idx_urls_expires_at;
ALTER TABLE urls DROP COLUMN IF EXISTS expires_at;
