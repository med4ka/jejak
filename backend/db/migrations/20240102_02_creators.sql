-- +goose Up
-- Fase 9: creators table + optional urls.creator_id relation.
-- creator_id is NULLABLE so anonymous short links (Fase 0-8) stay valid.
-- Run against BOTH the primary and the replica (both need the same schema).

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
