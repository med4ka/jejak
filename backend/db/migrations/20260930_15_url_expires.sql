-- +goose Up
-- Link expiry (2026-09-30): expires_at column for the "Active until" feature -
-- the redirect answers 410 Gone after the deadline and the dashboard shows an
-- active/scheduled/expired status. TIMESTAMP without time zone follows the
-- created_at convention; EVERY application write uses time.Now().UTC() or a
-- UTC value from the client (toISOString), so the column always reads back as
-- UTC. This column existed in migration 01 and was dropped by migration 12
-- (drop_dead_columns) while it was still unused - it is reinstated here with
-- contextual comments.
ALTER TABLE urls ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP NULL;

-- Partial index: only rows that have an expiry (no expiry = NULL, which needs
-- no index). A future "what has already expired?" query can still use this
-- index; the redirect itself keeps reading per short_code.
CREATE INDEX IF NOT EXISTS idx_urls_expires_at ON urls(expires_at) WHERE expires_at IS NOT NULL;
