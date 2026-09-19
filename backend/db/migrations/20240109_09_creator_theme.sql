-- +goose Up
-- Fase 10: preset tema halaman publik kreator (/u/[username]).
-- Preset TERBATAS: classic (print-white + flash-yellow, default sekarang),
-- night (ink bg + print-white text), coral (print-white + flash-coral).
-- Hanya 3 nilai — validasi ada di handler; kolom tetap varchar biasa karena
-- ini enum kecil yang KONSUMENNYA frontend (bukan query/join/agg).
-- Default 'classic' = perilaku lama, semua akun existing otomatis classic.

ALTER TABLE creators ADD COLUMN IF NOT EXISTS theme VARCHAR(20) NOT NULL DEFAULT 'classic';