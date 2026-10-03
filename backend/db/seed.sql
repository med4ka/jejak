-- Minimal seed for manual testing + a demo of manual sync to the replica.
-- Run it on the primary database (jejak) AND the replica (jejak_replica) so
-- both hold the same rows. This is the "manual sync" that replaces real
-- Postgres streaming replication (see ARCHITECTURE.md §6).
-- Idempotent: safe to run repeatedly (ON CONFLICT DO NOTHING).

INSERT INTO urls (short_code, original_url)
VALUES ('abc123', 'https://example.com')
ON CONFLICT (short_code) DO NOTHING;
