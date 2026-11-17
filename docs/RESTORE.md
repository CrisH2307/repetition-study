# Restore drill (P3-LOAD, D53)

Do this once before real students, then every month. It proves a backup can really be loaded. It never touches the live database.

Needs: `pg_dump`, `pg_restore` and `psql` on your laptop. Their version must be the same as or newer than the database's (an older `pg_dump` stops with "server version mismatch"; this happened on the test laptop). Check with `pg_dump --version`, and an empty scratch database. A local one is free:

```bash
docker compose up -d --wait
docker exec repetition-study-db-1 psql -U repetition -d postgres -c "CREATE DATABASE restore_drill"
```

## Steps

1. Take the backup. Use the live address Fly gave you for `DATABASE_URL` (see `docs/DEPLOY.md`, section C):

```bash
DATABASE_URL="<live database address>" deploy/backup.sh
```

   It prints `wrote backup-<date>.dump`. Write down the time it took.

2. Count what is in the live database, so you can compare. Read only:

```bash
psql "<live database address>" -c "SELECT (SELECT count(*) FROM users) AS users, (SELECT count(*) FROM schema_migrations) AS migrations"
```

3. Restore into the scratch database:

```bash
RESTORE_URL="postgres://repetition:repetition@localhost:5432/restore_drill?sslmode=disable" deploy/restore-test.sh backup-<date>.dump
```

4. Count again in the scratch database with the same query, using the scratch address.

5. Clean up: delete the scratch database and the dump file if it holds real student data.

```bash
docker exec repetition-study-db-1 psql -U repetition -d postgres -c "DROP DATABASE restore_drill"
```

## Checked so far

The steps were tried once on a laptop on 2026-11-16 and stopped at step 1 on the version mismatch above, so **the full drill has not been run end to end** (unverified). Run it yourself.

## What pass looks like

- No error lines in step 3.
- `migrations` is not 0, and the two counts in steps 2 and 4 match (the live one may be a little higher if students used the app between steps).
- Total time from step 1 to step 4 is written below. The G3 aim is under 30 minutes (unverified target, Cris may change it).

## If it fails

Copy everything on your screen to Cris and stop. Do not retry against the live database.

## Record

| Date | Backup size | Backup time | Restore time | Counts match | Done by |
|------|-------------|-------------|--------------|--------------|---------|
|      |             |             |              |              |         |

Unverified: how long Fly Managed Postgres keeps its own automatic backups, and whether it offers a point-in-time restore. Ask Fly and write the answer here: ______
