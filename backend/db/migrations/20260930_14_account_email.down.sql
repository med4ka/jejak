-- +goose Down
-- Removes the email column and its partial unique index (migration 14).
DROP INDEX IF EXISTS creators_email_key;
ALTER TABLE creators DROP COLUMN IF EXISTS email;
