#!/bin/sh
# Daily database backup. Run from the host's scheduler. Needs DATABASE_URL and pg_dump.
# Writes backup-YYYY-MM-DD.dump in the current folder. Keep copies in a Canadian region (D17).
set -eu
out="backup-$(date +%F).dump"
pg_dump --format=custom --no-owner "$DATABASE_URL" > "$out"
echo "wrote $out"
