-- +goose Up
-- Fase 9: creator profile customization.
-- avatar_url = image URL (varchar, nullable) - NOT a file upload, kept simple.
-- socials = JSON array [{platform, url}] (JSONB, default []) - deliberately
-- NOT a separate table: too small to justify its own relation (see the
-- SocialLink comment in db.go for that decision). Run on primary AND replica.

ALTER TABLE creators ADD COLUMN IF NOT EXISTS avatar_url VARCHAR NULL;
ALTER TABLE creators ADD COLUMN IF NOT EXISTS socials JSONB NULL DEFAULT '[]';
