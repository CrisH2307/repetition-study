# Deploy (P1-DEPLOY)

Shape (D32): Vercel serves the web app. The API runs as the Docker image from `Dockerfile` on a container host you have not picked yet. Vercel forwards `/v1/*` and `/healthz` to the API, so the sign-in cookie stays on one site.

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
4. In `vercel.json`, replace `API_ORIGIN_PLACEHOLDER` (twice) with the API host name, e.g. `api-xyz.example.com`, then commit via a PR. No env vars are needed on Vercel.
5. Deploy. Open the Vercel URL. `/healthz` should say ok.

## B. API container

Any host that runs a Docker image and gives HTTPS. Pick a **Canadian region** (D17). Database: managed Postgres in the same region.

Set these on the host (names only, you type the values):
`DATABASE_URL`, `COOKIE_SECURE` (leave unset; it must stay true), `APP_URL` (the Vercel site address), `STUDENT_MODEL`, `CHECKER_MODEL`, `OPENROUTER_API_KEY`, `MONTHLY_CAP_USD`.
`PORT` is 8080 by default.

Start command (applies new tables, then starts the API; same as `compose.yaml`):

```
sh -c "migrate && exec api"
```

Check: `https://<api-host>/healthz` answers ok.

## C. Daily backup and one restore test

1. Schedule `deploy/backup.sh` once a day (host's scheduler, or turn on the managed database's daily backup if it has one). Store copies in a Canadian region.
2. Restore test, once, before real students: create an empty scratch database, then
   `RESTORE_URL=<scratch url> deploy/restore-test.sh backup-<date>.dump`.
3. Pass: no errors, and the number is not 0. On failure copy the output to Cris and stop.
4. Delete the scratch database after.

## D. First invite

On any computer with the repo and the live `DATABASE_URL` and `APP_URL` set:

```
go run ./cmd/invite -name "Dino"
```

It prints one link ending in `/invite#...`. Send it to the student. Each link is one use.

## E. Final check (step 5)

Sign in with the invite link, open a study chat, and watch the AI student's replies appear as they stream. If the page stays blank until the end, or sign-in does not stick, copy what you saw to Cris and stop.
