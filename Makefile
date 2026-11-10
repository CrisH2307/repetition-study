# Dev defaults match compose.yaml. .env (gitignored) may override them.
DEFAULT_DB := postgres://repetition:repetition@localhost:5432/repetition?sslmode=disable
LOAD_ENV := set -a; [ ! -f .env ] || . ./.env; set +a; export DATABASE_URL=$${DATABASE_URL:-$(DEFAULT_DB)};

.PHONY: up migrate test run seed

up:
	docker compose up -d --wait

migrate:
	@$(LOAD_ENV) go run ./cmd/migrate

test:
	@$(LOAD_ENV) go test ./...

run:
	@$(LOAD_ENV) go run ./cmd/api

seed:
	@$(LOAD_ENV) docker compose exec -T db sh -c 'psql -v ON_ERROR_STOP=1 -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"' < scripts/seed.sql
