-- +goose Up

-- API Keys untuk public API (POST /api/v1/shorten).
-- Kunci penting: key ASLI tidak pernah disimpan — hanya key_hash (SHA-256) yang
-- disimpan. Kalau database bocor, key asli tidak langsung kepakai orang lain
-- (prinsip yang sama seperti password, lihat LEARN di api/internal/auth).
-- label = nama ramah (opsional) yang ditampilkan di dashboard; last_used_at
-- di-update tiap kali key dipakai supaya user tahu key mana yang aktif.
CREATE TABLE IF NOT EXISTS api_keys (
    id          SERIAL PRIMARY KEY,
    creator_id  INTEGER NOT NULL REFERENCES creators(id) ON DELETE CASCADE,
    key_hash    VARCHAR(64) NOT NULL UNIQUE,
    label       VARCHAR(100) NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ NULL
);

-- Query "list key milik kreator" dan "cari key by hash" dipakai di tiap
-- request ke public API — hash UNIQUE sudah menyediakan index, tapi index
-- per creator dipakai saat menghapus/memvalidasi kepemilikan.
CREATE INDEX IF NOT EXISTS idx_api_keys_creator ON api_keys (creator_id);