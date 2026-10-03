-- MANUAL ROLLBACK ONLY (reversible).
-- Removes the theme choice; every account returns to 'classic' (old behavior).

ALTER TABLE creators DROP COLUMN IF EXISTS theme;