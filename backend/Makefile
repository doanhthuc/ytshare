.PHONY: help dev up down reset migrate migrate-new test build lint fmt fmt-check tidy

# Use bash for process substitution (`<(...)`) in fmt-check.
SHELL       := /bin/bash

GO          ?= go
BIN         := bin/server
COMPOSE_DEV := docker compose -f ../docker-compose.dev.yml

help: ## Show available commands.
	@echo "First-time setup:"
	@echo "  make up && make migrate         # bring up Postgres+Redis, apply schema"
	@echo ""
	@echo "Daily workflow:"
	@echo "  make dev            Start Postgres+Redis and run the API (no auto-migrate)"
	@echo "  make up             Start Postgres+Redis only (background)"
	@echo "  make down           Stop containers (keep data)"
	@echo "  make reset          Stop containers and DELETE all DB data"
	@echo ""
	@echo "Schema migrations:"
	@echo "  make migrate              Apply pending migrations"
	@echo "  make migrate-new name=X   Scaffold a new *.up.sql / *.down.sql pair"
	@echo ""
	@echo "  Advanced ops use the CLI directly:"
	@echo "    go run ./cmd/migrate down              # rollback last"
	@echo "    go run ./cmd/migrate status            # current version"
	@echo "    go run ./cmd/migrate force <version>   # recovery only"
	@echo ""
	@echo "Quality:"
	@echo "  make test           Run all tests (unit + integration, race detector)"
	@echo "  make build          Build server + migrate binaries"
	@echo "  make lint           golangci-lint"
	@echo "  make fmt            gofumpt + goimports"
	@echo "  make tidy           go mod tidy"

# --- Daily workflow ---------------------------------------------------------

dev: up ## Start infra and run the API in the foreground (no auto-migrate).
	$(GO) run ./cmd/server

up: ## Start Postgres + Redis, wait for healthchecks.
	$(COMPOSE_DEV) up -d --wait

down: ## Stop the dev containers (keep data volumes).
	$(COMPOSE_DEV) down

reset: ## Stop containers and DELETE the data volumes (fresh DB).
	$(COMPOSE_DEV) down -v

# --- Migrations -------------------------------------------------------------
# The server never auto-applies migrations. Run `make migrate` explicitly
# the first time and after every schema change. For rollback / status /
# force, call the CLI directly:
#   go run ./cmd/migrate <down|status|force N>.

migrate: ## Apply all pending migrations.
	$(GO) run ./cmd/migrate up

migrate-new: ## Scaffold a new migration pair: make migrate-new name=add_votes
	@if [ -z "$(name)" ]; then echo "usage: make migrate-new name=<short_name>"; exit 1; fi
	$(GO) run ./cmd/migrate create $(name)

# --- Quality ----------------------------------------------------------------

test: ## Run all tests (unit + integration, race detector on).
	$(GO) test -race -count=1 ./...

build: ## Build the production binaries (server + migrate CLI).
	mkdir -p bin
	$(GO) build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/server
	$(GO) build -trimpath -ldflags="-s -w" -o bin/migrate ./cmd/migrate

lint: ## Run golangci-lint.
	golangci-lint run

fmt: ## Format with gofumpt + goimports (local-prefix aware).
	gofumpt -l -w .
	goimports -local backend -l -w .

fmt-check: ## Verify formatting (used in CI).
	@diff -u <(echo -n) <(gofumpt -l . && goimports -local backend -l .) \
		|| (echo "Run 'make fmt' to format the code"; exit 1)

tidy: ## Sync go.mod / go.sum.
	$(GO) mod tidy
