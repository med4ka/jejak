-- +goose Up
-- Fase 10: preset themes for the creator's public page (/u/[username]).
-- LIMITED presets: classic (print-white + flash-yellow, current default),
-- night (ink bg + print-white text), coral (print-white + flash-coral).
-- Only 3 values - validation lives in the handler; the column stays a plain
-- varchar because this is a small enum CONSUMED by the frontend (never
-- queried, joined, or aggregated).
-- Default 'classic' = old behavior; every existing account becomes classic
-- automatically.

ALTER TABLE creators ADD COLUMN IF NOT EXISTS theme VARCHAR(20) NOT NULL DEFAULT 'classic';