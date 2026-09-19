#!/bin/sh
# Entrypoint for postgres_replica. On first start (empty PGDATA) it clones the
# primary via pg_basebackup -R (writes standby.signal + primary_conninfo), then
# starts postgres as a hot standby. Invoked as: sh replica-entrypoint.sh
# (sh form avoids needing the exec bit on Windows checkouts).
set -eu
PGDATA_DIR="${PGDATA:-/var/lib/postgresql/data}"
if [ ! -s "$PGDATA_DIR/PG_VERSION" ]; then
	echo "[replica] waiting for primary ($PRIMARY_HOST)..."
	until pg_isready -h "$PRIMARY_HOST" -U "$POSTGRES_USER"; do sleep 2; done
	echo "[replica] running pg_basebackup from primary..."
	pg_basebackup -h "$PRIMARY_HOST" -U replicator -D "$PGDATA_DIR" -Fp -Xs -P -R
fi
exec docker-entrypoint.sh "$@"
