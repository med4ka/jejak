-- +goose Up
-- Fase 12: buang kolom yang tidak pernah ditulis/dibaca aplikasi.
-- Konteks: urls.expires_at dideklarasikan sejak Fase 0 (schema awal) tetapi
-- tidak ada satupun handler/store yang mengisinya (tidak ada fitur expiry).
-- click_events.country juga mati — geo-IP bukan scope PRD, dan column NULL
-- selamanya hanya menambah noise pada schema. Menghapusnya membuat skema
-- jujur: kolom yang ada = kolom yang dipakai. Gunakan .down.sql untuk rollback.
-- Dijalankan di database primary DAN replica (keduanya butuh skema sama).

ALTER TABLE urls DROP COLUMN IF EXISTS expires_at;
ALTER TABLE click_events DROP COLUMN IF EXISTS country;
