.DEFAULT_GOAL := help

# --- Help ---

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# --- Setup ---

setup: ## Pull Go deps and create .env from example
	@test -f .env || cp .env.example .env
	go mod tidy

# --- Database ---

db-up: ## Start PostgreSQL
	docker compose up -d postgres

db-down: ## Stop PostgreSQL
	docker compose down

db-logs: ## Tail PostgreSQL logs
	docker compose logs -f postgres

db-shell: ## Open psql shell
	docker exec -it argos_postgres psql -U argos -d argos

# --- Build ---

build: build-api build-worker ## Build all binaries

build-api: ## Build API server
	@mkdir -p bin
	go build -o bin/argos-api ./cmd/api

build-worker: ## Build worker
	@mkdir -p bin
	go build -o bin/argos-worker ./cmd/worker

# --- Run ---

run-api: ## Run API server
	go run ./cmd/api

run-worker: ## Run worker
	go run ./cmd/worker

seed: ## Populate database with realistic demo data (~150 records)
	go run ./cmd/seed

# --- Quality ---

test: ## Run all tests
	go test -race -v ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format all code
	gofmt -w .

clean: ## Remove build artifacts
	rm -rf bin/

.PHONY: help setup db-up db-down db-logs db-shell build build-api build-worker run-api run-worker seed test vet fmt clean
