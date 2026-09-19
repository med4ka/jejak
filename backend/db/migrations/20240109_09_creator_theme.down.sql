-- ROLLBACK MANUAL ONLY (reversibel).
-- Hapus pilihan tema; semua akun kembali ke 'classic' (perilaku lama).

ALTER TABLE creators DROP COLUMN IF EXISTS theme;