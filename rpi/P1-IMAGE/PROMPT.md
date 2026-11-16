Unit P1-IMAGE, wave 1. Branch `chore/docker-image`. Model: Sonnet 5.5, effort low. Labels: `Issue`, `phase-1`, `wave-1`.
Files: `Dockerfile`, `.dockerignore`, `compose.yaml`, `cmd/api/`, `docs/DEPLOY.md`, `rpi/P1-IMAGE/`.
Read first: D32 in `docs/DECISIONS.md`, `cmd/api/main.go`, `Makefile`, `compose.yaml`.

1. Multi-stage `Dockerfile`: build `web/` (npm ci, npm run build), build the Go binaries (api, migrate, seed), final small image with `poppler-utils` (pdftotext, pdftoppm), non-root user.
2. `cmd/api` serves the built web app from a directory given by `WEB_DIR` (default unset means API only, as today), with SPA fallback to index.html. API routes keep priority.
3. `compose.yaml`: add an `app` service under a `prod` profile that runs migrate then api against `db`. The dev `db` service stays as it is.
4. `docs/DEPLOY.md`: plain steps to build and run the image locally, the environment variables it reads (names only), and that the host is not chosen yet.
Never copy `.env` or `content/private/` into the image (`.dockerignore`).
Done when: `docker build .` works, `docker compose --profile prod up` serves the page at localhost and `/healthz` is ok, Go tests green. Push, PR into qa with the labels above, "Closes #<issue>". Never merge. Report under 200 words.
