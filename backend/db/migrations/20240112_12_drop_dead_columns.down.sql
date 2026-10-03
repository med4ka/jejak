-- MANUAL ROLLBACK ONLY (reversible).
-- Restores the spare columns (declared back in Fase 0, never used).

ALTER TABLE urls ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP NULL;
ALTER TABLE click_events ADD COLUMN IF NOT EXISTS country VARCHAR(2) NULL;
