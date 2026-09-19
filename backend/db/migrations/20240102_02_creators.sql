-- +goose Up
-- Fase 9: tabel creators + relasi opsional urls.creator_id.
-- creator_id NULLABLE supaya short-link anonim (Fase 0-8) tetap valid.
-- Dijalankan di database primary DAN replica (keduanya butuh skema sama).

CREATE TABLE IF NOT EXISTS creators (
    id SERIAL PRIMARY KEY,
    username VARCHAR(30) UNIQUE NOT NULL,
    display_name VARCHAR NOT NULL,
    bio TEXT NULL,
    password_hash VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE urls ADD COLUMN IF NOT EXISTS creator_id INTEGER NULL REFERENCES creators(id);

CREATE INDEX IF NOT EXISTS idx_urls_creator_id ON urls(creator_id);
