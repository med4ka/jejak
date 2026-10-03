-- +goose Up
-- Custom slugs allow up to 30 characters (application validation), but the
-- short_code column was still VARCHAR(10) from the initial schema, so
-- INSERTing a slug longer than 10 chars failed with "value too long". Widen
-- to VARCHAR(30) in BOTH tables that store short_code (urls AND click_events
-- - redirect logging hits the same wall). Widening a varchar does not rewrite
-- the table, so this is safe to run online.
-- Run on primary AND replica.

ALTER TABLE urls ALTER COLUMN short_code TYPE VARCHAR(30);
ALTER TABLE click_events ALTER COLUMN short_code TYPE VARCHAR(30);
