-- +goose Up
-- Creator link reorder: position column (int, default 0). Existing rows are
-- all 0, so the old order (id DESC) does not change until the user reorders.
-- Run on primary AND replica.

ALTER TABLE urls ADD COLUMN IF NOT EXISTS position INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_urls_creator_position ON urls(creator_id, position);
