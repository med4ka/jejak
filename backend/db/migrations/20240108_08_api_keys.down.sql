-- MANUAL ROLLBACK ONLY: drops the api_keys table and its per-creator index.
-- +goose Down
DROP TABLE IF EXISTS api_keys;
DROP INDEX IF EXISTS idx_api_keys_creator;