-- +goose Up
-- Fase 10: fitur "link unggulan" — kreator bisa menandai 1 link sebagai
-- featured. Kolom boolean; default FALSE = perilaku lama tanpa perlu backfill.
-- Uniqueness (maks 1 per creator) dijamin di APPLICATION LAYER via transaksi
-- (DB-level partial unique index over-engineering untuk data 1 user).

ALTER TABLE urls ADD COLUMN IF NOT EXISTS is_featured BOOLEAN NOT NULL DEFAULT FALSE;
