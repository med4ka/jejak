-- ROLLBACK MANUAL ONLY
-- +goose Down
DROP TABLE IF EXISTS api_keys;
DROP INDEX IF EXISTS idx_api_keys_creator;