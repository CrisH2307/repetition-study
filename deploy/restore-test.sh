#!/bin/sh
# Restore test: loads a backup into a THROWAWAY database and counts rows.
# Usage: RESTORE_URL=postgres://.../scratch deploy/restore-test.sh backup-2026-11-16.dump
# RESTORE_URL must point at an empty scratch database, never the live one.
set -eu
: "${RESTORE_URL:?set RESTORE_URL to an empty scratch database}"
pg_restore --no-owner --exit-on-error -d "$RESTORE_URL" "$1"
psql "$RESTORE_URL" -c "SELECT count(*) AS migrations_applied FROM schema_migrations"
echo "PASS if the number above is not 0 and there were no errors"
