-- +goose Up
-- Simple per-link tags/categories: JSONB array of strings, default [].
-- NOT a separate table (see the Link comment in db.go): small and never
-- queried across creators. Run on primary AND replica.

ALTER TABLE urls ADD COLUMN IF NOT EXISTS tags JSONB NULL DEFAULT '[]';
