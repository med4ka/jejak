-- +goose Up
-- Initial schema: urls (short_code, original_url, expires_at, click_count)
-- and click_events (per-click log with referrer and country), plus an index
-- on each short_code column.
-- SQL in section 'Up' is applied to the database

CREATE TABLE IF NOT EXISTS urls (
    id SERIAL PRIMARY KEY,
    short_code VARCHAR(10) UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NULL,
    click_count BIGINT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS click_events (
    id SERIAL PRIMARY KEY,
    short_code VARCHAR(10) NOT NULL,
    clicked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    referrer VARCHAR NULL,
    country VARCHAR(2) NULL
);

CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);
CREATE INDEX IF NOT EXISTS idx_clicks_short_code ON click_events(short_code);