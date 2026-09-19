-- +goose Up
-- Custom slug membolehkan s.d. 30 karakter (validasi aplikasi), tapi kolom
-- short_code masih VARCHAR(10) dari skema awal -> INSERT slug >10 char gagal
-- "value too long". Lebarkan ke VARCHAR(30) di KEDUA tabel yang menyimpan
-- short_code (urls DAN click_events — redirect logging kena tembok yang sama).
-- Widening varchar tidak rewrite table, aman jalan online.
-- Dijalankan di primary DAN replica.

ALTER TABLE urls ALTER COLUMN short_code TYPE VARCHAR(30);
ALTER TABLE click_events ALTER COLUMN short_code TYPE VARCHAR(30);
