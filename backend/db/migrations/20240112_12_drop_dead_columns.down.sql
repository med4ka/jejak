-- ROLLBACK MANUAL ONLY (reversibel).
-- Kembalikan kolom cadangan (dulu dideklarasikan Fase 0, tak pernah dipakai).

ALTER TABLE urls ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP NULL;
ALTER TABLE click_events ADD COLUMN IF NOT EXISTS country VARCHAR(2) NULL;
