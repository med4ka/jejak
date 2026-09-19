-- +goose Up
-- Fase 10 (rebranding): preset "night" diganti konsep jadi "darkroom"
-- (kamar gelap cuci foto, aksen safelight coral). Rewrite data legacy supaya
-- API/master key-nya selalu "darkroom"; handler juga mem- mapping read "night"
-- -> "darkroom" sebagai safety, tapi data di sini dibersihkan permanen.

UPDATE creators SET theme = 'darkroom' WHERE theme = 'night';