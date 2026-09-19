-- ROLLBACK MANUAL ONLY (reversibel).
-- Kembalikan key rebranding ke nama lama (night). Baris yang baru ditulis
-- sebagai darkroom ikut diceploskan — tidak ada cara membedakan asal rebrand.

UPDATE creators SET theme = 'night' WHERE theme = 'darkroom';