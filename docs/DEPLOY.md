# Deploy the image

The host is not chosen yet (D32). The image runs anywhere Docker runs.

## Build and run on your computer

1. Copy `.env.example` to `.env` and fill in the model ids and one API key.
2. Run `docker compose --profile prod up --build`.
3. Open http://localhost:8080. You should see the app.
4. Open http://localhost:8080/healthz. It should show `{"status":"ok"}`.
5. If something fails, copy the output to Cris and stop.

In `.env`, set `DATABASE_URL` for the db service with your own password (it must use host `db`, port 5432).

The `app` service runs `migrate`, then `api`, against the `db` service.

## Environment variables (names only)

- `DATABASE_URL` (required)
- `PORT` (default 8080)
- `WEB_DIR` (set to `/srv/web` in the image; unset means API only)
- `STUDENT_MODEL`, `CHECKER_MODEL`
- `GEMINI_API_KEY`, `ANTHROPIC_API_KEY`, `OPENROUTER_API_KEY` (the one your model ids need)

`.env` and `content/private/` are never copied into the image.
