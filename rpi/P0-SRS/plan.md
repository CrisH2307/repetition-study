# P0-SRS plan
1. `ApplyTeachBackSession`: set session id on ctx, look up existing teach_back for the session inside the locked tx, else write with session_id.
2. `cmd/migrate`: `pg_advisory_lock` on its connection before touching schema_migrations.
3. srs tests: throwaway database per test (CREATE DATABASE, run migrations, DROP ... WITH (FORCE)).
Risks: the dev DB must allow CREATEDB for the test user (it does locally).
