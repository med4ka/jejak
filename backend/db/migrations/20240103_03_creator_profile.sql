-- +goose Up
-- Fase 9: kustomisasi profil kreator.
-- avatar_url = URL gambar (varchar, nullable) — BUKAN file upload, tetap simpel.
-- socials = array JSON [{platform, url}] (JSONB, default []) — sengaja BUKAN
-- tabel terpisah: terlalu kecil untuk butuh relasi sendiri (lihat LEARN di
-- db.go soal keputusan ini). Dijalankan di primary DAN replica.

ALTER TABLE creators ADD COLUMN IF NOT EXISTS avatar_url VARCHAR NULL;
ALTER TABLE creators ADD COLUMN IF NOT EXISTS socials JSONB NULL DEFAULT '[]';
