-- +goose Up
-- Account settings (2026-09-30): email column for Section A "Change Email".
-- Nullable so older rows without an email remain valid; the UNIQUE PARTIAL
-- index covers only filled emails - the intent "email already used = conflict"
-- is explicit and does not rely on NULL != NULL behavior (Postgres indeed
-- never treats two NULLs as equal, but the partial index makes the
-- filled-email conflict a database guarantee instead of an application
-- check-then-insert).
ALTER TABLE creators ADD COLUMN IF NOT EXISTS email VARCHAR(255) NULL;

CREATE UNIQUE INDEX IF NOT EXISTS creators_email_key ON creators (email) WHERE email IS NOT NULL;
