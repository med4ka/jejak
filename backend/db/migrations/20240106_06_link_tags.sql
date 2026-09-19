-- +goose Up
-- Tag/kategori sederhana per link: JSONB array of strings, default [].
-- BUKAN tabel terpisah (lihat LEARN di db.go): kecil, tidak pernah di-query
-- lintas creator. Dijalankan di primary DAN replica.

ALTER TABLE urls ADD COLUMN IF NOT EXISTS tags JSONB NULL DEFAULT '[]';
