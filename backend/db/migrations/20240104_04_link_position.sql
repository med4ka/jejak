-- +goose Up
-- Reorder link kreator: kolom position (int, default 0). Baris lama otomatis
-- 0 semua sehingga urutan lama (id DESC) tidak berubah sampai user me-reorder.
-- Dijalankan di primary DAN replica.

ALTER TABLE urls ADD COLUMN IF NOT EXISTS position INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_urls_creator_position ON urls(creator_id, position);
