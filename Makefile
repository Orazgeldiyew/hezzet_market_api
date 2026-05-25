# Hezzet Market backend — common dev/ops tasks.
# Run `make help` for the list.

.DEFAULT_GOAL := help

BINARY      := hezzet_market_backend
PKG         := ./...
MIG_DIR     := migrations
AUDIT_SCRIPT := dev/production_audit.sh

# Pull DB connection info from .env (falls back to localhost defaults). When
# you have a different .env layout, override on the command line:
#     make psql DB_DSN=postgres://...
include .env
export
DB_DSN ?= postgres://$(DB_USERNAME):$(DB_PASSWORD)@$(DB_HOST):5432/$(DB_DATABASE)?sslmode=disable
PORT ?= 8080

.PHONY: help run dev build start stop restart \
        psql db-shell migrate migrate-up migrate-down migrate-list \
        seed test test-v lint vet fmt tidy swagger \
        audit smoke clean

help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ─── Run / build ────────────────────────────────────────────────────────────

run: ## Run backend (foreground)
	go run main.go

dev: run ## Alias for `run`

build: ## Compile binary
	go build -o $(BINARY) main.go

start: ## Start backend in background (logs → /tmp/backend.log)
	@if fuser -s $(PORT)/tcp 2>/dev/null; then echo "Port $(PORT) already in use"; exit 1; fi
	@nohup go run main.go > /tmp/backend.log 2>&1 &
	@sleep 4
	@curl -sf http://localhost:$(PORT)/api/auth/login -X POST -H "Content-Type: application/json" -d '{}' >/dev/null && echo "✅ backend up on :$(PORT) (logs: /tmp/backend.log)" || (echo "❌ start failed; tail /tmp/backend.log"; tail -20 /tmp/backend.log; exit 1)

stop: ## Stop background backend on $(PORT)
	@fuser -k $(PORT)/tcp 2>/dev/null && echo "✅ stopped" || echo "nothing running on :$(PORT)"

restart: stop start ## Restart backend in background

# ─── Database ───────────────────────────────────────────────────────────────

psql: ## Open psql shell against the local DB
	PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -U $(DB_USERNAME) -d $(DB_DATABASE)

db-shell: psql ## Alias for `psql`

migrate: migrate-up ## Apply all unapplied migrations (alias for migrate-up)

migrate-up: ## Apply all .up.sql migrations not yet applied (idempotent via IF NOT EXISTS in files)
	@for f in $$(ls $(MIG_DIR)/*.up.sql | sort); do \
	  echo "▶ $$f"; \
	  PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -U $(DB_USERNAME) -d $(DB_DATABASE) -v ON_ERROR_STOP=1 -f $$f >/dev/null || exit 1; \
	done
	@echo "✅ migrations applied"

migrate-down: ## Roll back the LAST migration (uses paired .down.sql)
	@latest=$$(ls $(MIG_DIR)/*.down.sql | sort | tail -1); \
	echo "▼ $$latest"; \
	PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -U $(DB_USERNAME) -d $(DB_DATABASE) -v ON_ERROR_STOP=1 -f $$latest

migrate-list: ## List migrations in order
	@ls $(MIG_DIR)/*.up.sql | sort | sed 's|.*/||' | cat -n

seed: ## Run seed scripts (cmd/seed)
	go run cmd/seed/main.go

# ─── Quality ────────────────────────────────────────────────────────────────

test: ## Run all Go tests
	go test $(PKG)

test-v: ## Run tests with verbose output
	go test -v $(PKG)

vet: ## go vet
	go vet $(PKG)

fmt: ## go fmt
	go fmt $(PKG)

tidy: ## go mod tidy
	go mod tidy

lint: vet ## Run linters (currently just `go vet`; install golangci-lint for more)
	@if command -v golangci-lint >/dev/null 2>&1; then \
	  golangci-lint run; \
	else \
	  echo "ℹ️  golangci-lint not installed — only `go vet` ran"; \
	fi

swagger: ## Regenerate Swagger docs
	swag init -g main.go -o docs

# ─── Audit / smoke ──────────────────────────────────────────────────────────

audit: ## Run full production-readiness audit (68 checks)
	@if ! curl -sf http://localhost:$(PORT)/api/auth/login -X POST -H "Content-Type: application/json" -d '{}' >/dev/null; then \
	  echo "⚠️  backend not running on :$(PORT) — start it first ('make start')"; \
	  exit 1; \
	fi
	bash $(AUDIT_SCRIPT)

smoke: audit ## Alias for `audit`

# ─── Cleanup ────────────────────────────────────────────────────────────────

clean: ## Remove built binary
	rm -f $(BINARY)
