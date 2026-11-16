# P1-DEPLOY log
- vercel.json: build web/, output web/dist, rewrites /v1 and /healthz to API, SPA fallback last.
- deploy/: backup.sh, restore-test.sh (scripts only, not run).
- docs/DEPLOY.md: plain steps. Vercel docs read 2026-11-16: external rewrite = reverse proxy confirmed; cookies, SSE, timeouts, region, filesystem-vs-fallback order NOT stated (unverified).
- Not run: any deploy, vite build with the real API origin.
- fly.toml added (Fly yyz, min 1 machine, release_command migrate). Fly docs read 2026-11-16; MPG backup details and SSE timeout unverified.
