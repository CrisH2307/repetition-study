# Dev defaults match compose.yaml. .env (gitignored) may override them.
DEFAULT_DB := postgres://repetition:repetition@localhost:5432/repetition?sslmode=disable
LOAD_ENV := set -a; [ ! -f .env ] || . ./.env; set +a; export DATABASE_URL=$${DATABASE_URL:-$(DEFAULT_DB)};

.PHONY: up migrate test run seed export-disputes

up:
	docker compose up -d --wait

migrate:
	@$(LOAD_ENV) go run ./cmd/migrate

test:
	@$(LOAD_ENV) go test ./...

# Dev serves plain http on localhost, so the cookie may not be Secure here. Production leaves COOKIE_SECURE unset.
run:
	@$(LOAD_ENV) COOKIE_SECURE=false go run ./cmd/api

seed:
	@$(LOAD_ENV) go run ./cmd/seed

# Every "I think this check is wrong" note as CSV, for Dino.
export-disputes:
	@$(LOAD_ENV) go run ./cmd/export -disputes
