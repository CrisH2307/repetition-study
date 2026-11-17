# Deploy (P1-DEPLOY)

Shape (D32): Vercel serves the web app. The API runs as the Docker image from `Dockerfile` on Fly.io (Toronto). Vercel forwards `/v1/*` and `/healthz` to the API, so the sign-in cookie stays on one site.

Nothing here costs money until you click a paid button. Never paste secrets into git.

## What is and is not verified

Checked in Vercel's docs (2026-11-16): a rewrite to an outside URL keeps the browser address and acts as a reverse proxy.
**Not stated in the docs, so unverified:** whether cookies (`Set-Cookie`) and POST bodies pass through, whether Server-Sent Events (live streams) are streamed or buffered, any proxy time limit, Canadian region for the proxy, and whether the catch-all rule leaves real files alone.
Plain risk: if live streams do not work through Vercel, the study chat will look stuck. Test step 5 below. If it fails, tell Cris; the fallback is to serve the web app from the API image itself (it already can, with `WEB_DIR`) and drop Vercel.
Project must be new (created after 2026-04-06) or caching of proxied replies may apply; the API should send `Cache-Control: no-store` (unverified that it does).

## A. Vercel project

1. Vercel, Add New Project, import `CrisH2307/repetition-study`.
2. Root Directory: leave as the repo root. Framework Preset: Other. `vercel.json` sets build and output.
3. Production branch: `main`. (Vercel previews qa if you want.)
4. In `vercel.json` the API address is `repetition-study.fly.dev`; change it only if you rename the Fly app.
5. Deploy. Open the Vercel URL. `/healthz` should say ok.

## B. API on Fly.io (small, intro phase)

Files: `fly.toml` (app `repetition-study`, region `yyz`, port 8080, `/healthz` check, release command `migrate`). It is set to the smallest machine (256 MB) that sleeps when idle, so the first request after a pause is slow. If PDF reading runs out of memory, raise `memory` to `512mb` in `fly.toml`.

Verified in Fly docs (2026-11-16): `yyz` is Toronto; the release command runs `migrate` in a temporary machine with the new image and secrets, and a failure stops the deploy.
**Cost, unverified:** Fly's current free allowance and whether it asks for a card. Check Fly's pricing page before `fly deploy`. Do NOT run `fly mpg create`: Fly Managed Postgres is paid (about $38/month).

Database: use a free Postgres from another provider (for example Supabase or Neon; free limits, Canada region and pausing rules are unverified, check their pricing pages). Pick a Canada region if offered (D17). If none is, tell Cris before real student data goes in.

1. Create the free database. Copy its connection string (keep it private).
2. Install `flyctl` and sign in yourself (`fly auth login`).
3. Secrets (you type the values, never paste into git or chat):
   `fly secrets set DATABASE_URL=... APP_URL=https://<your-vercel-address> OPENROUTER_API_KEY=... STUDENT_MODEL=... CHECKER_MODEL=... MONTHLY_CAP_USD=... -a repetition-study`
   `COOKIE_SECURE` is already `true` in `fly.toml`.
4. `fly deploy -a repetition-study`. The tables are created by the release command before the new version starts.
5. Check: `https://repetition-study.fly.dev/healthz` answers ok.

## C. Backups and one restore test

1. Check whether your free database provider takes automatic backups and for how long: ______ (unverified; free plans often keep few or none, so step 2 matters).
2. Your own copy as well: run `deploy/backup.sh` once a day with `DATABASE_URL` set (needs `pg_dump`; use the provider's connection string). Keep the file somewhere in Canada.
3. Restore test, once, before real students: make an empty scratch database (use a local Postgres, or a second free database), then
   `RESTORE_URL=<scratch url> deploy/restore-test.sh backup-<date>.dump`.
4. Pass: no errors, and the number is not 0. On failure copy the output to Cris and stop.

## D. First invite

On any computer with the repo and the live `DATABASE_URL` and `APP_URL` set:

```
go run ./cmd/invite -name "Dino"
```

It prints one link ending in `/invite#...`. Send it to the student. Each link is one use.

## E. Final check (step 5)

Sign in with the invite link, open a study chat, and watch the AI student's replies appear as they stream. If the page stays blank until the end, or sign-in does not stick, copy what you saw to Cris and stop.
