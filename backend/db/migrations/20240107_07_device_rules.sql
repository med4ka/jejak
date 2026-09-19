-- +goose Up
-- Smart Link (Fase fitur): 1 short-link bisa arahkan ke URL berbeda per
-- device. Format JSONB: {"ios": "https://...", "android": "https://..."}.
-- Kunci opsional; kalau device tidak match kunci manapun (atau rules kosong),
-- fallback ke original_url. Default '{}' supaya baris lama langsung kompatibel
-- (backward compatible, redirect lama tetap jalan tanpa perubahan).
-- Dijalankan di primary DAN replica.

ALTER TABLE urls ADD COLUMN IF NOT EXISTS device_rules JSONB NULL DEFAULT '{}';