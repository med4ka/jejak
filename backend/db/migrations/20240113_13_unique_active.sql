-- +goose Up
-- Fase 13 (improvement pass #2): honest analytics + link lifecycle.
-- 1. unique_click_count: per-link daily-unique counter (dedup by IP+UA, 24h
--    window). click_count stays = total human clicks; unique_click_count =
--    clicks from distinct visitors. This mirrors bit.ly's "total clicks vs
--    unique clicks" without adding a visitor table: the is_unique flag is
--    flushed from the handler through Redis SETNX.
-- 2. is_active: hard-disable a link without deleting it (like bit.ly's
--    "deactivate"): redirect => 410 Gone, the public page hides it, and the
--    dashboard can switch it back on. Permanent deletion still exists (the
--    new DELETE endpoint).
-- 3. click_events.is_unique + referrer_domain: click metadata that was
--    previously thrown away - analytics can break down referrer/source later.
-- 4. click_events.clicked_at is now filled EXPLICITLY by the worker (it used
--    to be DEFAULT CURRENT_TIMESTAMP = PROCESS time, not the real click time,
--    which shifted the 30-day chart whenever a backlog built up). The column
--    already existed; this migration simply starts using it.
-- The compound index (short_code, clicked_at) speeds up ClicksByDay, which
-- always filters to 30 days and joins per short_code.

ALTER TABLE urls ADD COLUMN unique_click_count BIGINT NOT NULL DEFAULT 0;
ALTER TABLE urls ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE click_events ADD COLUMN is_unique BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE click_events ADD COLUMN referrer_domain VARCHAR(255) NULL;

CREATE INDEX IF NOT EXISTS idx_clicks_short_code_clicked_at ON click_events(short_code, clicked_at);