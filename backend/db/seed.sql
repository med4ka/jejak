-- Seed minimal untuk testing manual + demo sync manual ke replica.
-- Jalankan di database primary (jejak) DAN di database replica (jejak_replica)
-- supaya keduanya punya baris yang sama. Ini adalah "sync manual" yang
-- menggantikan streaming replication Postgres asli (lihat ARCHITECTURE.md §6).
-- Idempotent: aman dijalankan berulang (ON CONFLICT DO NOTHING).

INSERT INTO urls (short_code, original_url)
VALUES ('abc123', 'https://example.com')
ON CONFLICT (short_code) DO NOTHING;
