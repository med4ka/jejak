-- +goose Up
-- Fase 12: drop columns the application never writes or reads.
-- Context: urls.expires_at was declared in Fase 0 (initial schema) but no
-- handler or store ever filled it (there is no expiry feature).
-- click_events.country is also dead - geo-IP is out of PRD scope, and a
-- column that stays NULL forever only adds schema noise. Removing them makes
-- the schema honest: an existing column is a column in use. Use .down.sql to
-- roll back.
-- Run against BOTH the primary and the replica (both need the same schema).

ALTER TABLE urls DROP COLUMN IF EXISTS expires_at;
ALTER TABLE click_events DROP COLUMN IF EXISTS country;
