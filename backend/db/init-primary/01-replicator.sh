#!/bin/sh
# Runs ONCE on primary's first init (mounted to /docker-entrypoint-initdb.d).
# Creates the replication role used by postgres_replica's pg_basebackup.
set -e
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
	CREATE ROLE replicator WITH REPLICATION LOGIN PASSWORD 'password';
EOSQL
