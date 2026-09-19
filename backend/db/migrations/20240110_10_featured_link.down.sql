-- ROLLBACK MANUAL ONLY (reversibel).
-- Hapus fitur link unggulan; semua featured flag hilang.

ALTER TABLE urls DROP COLUMN IF EXISTS is_featured;
