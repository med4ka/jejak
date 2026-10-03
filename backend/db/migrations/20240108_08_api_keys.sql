-- +goose Up

-- API keys for the public API (POST /api/v1/shorten).
-- Key point: the REAL key is never stored - only key_hash (SHA-256) is
-- stored. If the database leaks, the real key cannot be used by anyone else
-- (the same principle as passwords, see the note in api/internal/auth).
-- label = optional friendly name shown in the dashboard; last_used_at is
-- updated whenever the key is used so users know which key is active.
CREATE TABLE IF NOT EXISTS api_keys (
    id          SERIAL PRIMARY KEY,
    creator_id  INTEGER NOT NULL REFERENCES creators(id) ON DELETE CASCADE,
    key_hash    VARCHAR(64) NOT NULL UNIQUE,
    label       VARCHAR(100) NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ NULL
);

-- The "list a creator's keys" and "find a key by hash" queries run on every
-- public API request - the UNIQUE hash already provides an index, but the
-- per-creator index is used when deleting a key or validating ownership.
CREATE INDEX IF NOT EXISTS idx_api_keys_creator ON api_keys (creator_id);