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

harvest-ufop: ## Harvest publicações REAIS UFOP (Direito + Eng Minas) via OAI-PMH
	go run ./cmd/harvest-ufop all 50

harvest-direito: ## Só Direito (DEDIR + PPG Direito) - até 100 records
	go run ./cmd/harvest-ufop direito 100

harvest-minas: ## Só Engenharia de Minas - até 100 records
	go run ./cmd/harvest-ufop minas 100

reanalyze-ufop: ## Reclassifica todas as opportunities com a versão atual do classifier
	go run ./cmd/reanalyze-ufop

# --- AI service (Python) ---
# Código vive em ../ai-service/. Usa o venv ~/argos-ai (criado uma vez
# com `python -m venv ~/argos-ai` e `pip install -r ../ai-service/requirements.txt`).

AI_DIR ?= ../ai-service
AI_VENV ?= $(HOME)/argos-ai

ai-up: ## Sobe argos_classifier.py (Fase 5 — SBERT + TF-IDF servidos)
	cd $(AI_DIR) && $(AI_VENV)/bin/uvicorn argos_classifier:app --host 0.0.0.0 --port 8000

ai-up-legacy: ## Sobe api_argos.py legacy (BERTimbau original — só pra fallback)
	cd $(AI_DIR) && $(AI_VENV)/bin/uvicorn api_argos:app --host 0.0.0.0 --port 8000

ai-train: ## Roda pipeline ML completo (Fases 3 + 4)
	cd $(AI_DIR) && $(AI_VENV)/bin/python training/03_train_baseline.py
	cd $(AI_DIR) && $(AI_VENV)/bin/python training/04_train_sentence_transformers.py

ai-explore: ## Análise exploratória do dataset (Fase 2)
	cd $(AI_DIR) && $(AI_VENV)/bin/python training/02_explore.py

ai-annotate: ## Anotação LLM (Fase 1) — precisa GROQ_API_KEY ou ANTHROPIC_API_KEY
	cd $(AI_DIR) && $(AI_VENV)/bin/python training/01_annotate.py

# --- Quality ---

test: ## Run all tests
	go test -race -v ./...

vet: ## Run go vet
	go vet ./...

ingest-ufop-patents: ## Pesca patentes UFOP REAIS do Google Patents
	go run ./cmd/ingest-ufop-patents 30

fmt: ## Format all code
	gofmt -w .

clean: ## Remove build artifacts
	rm -rf bin/

.PHONY: help setup db-up db-down db-logs db-shell build build-api build-worker run-api run-worker seed test vet fmt clean harvest-ufop harvest-direito harvest-minas reanalyze-ufop ai-up ai-up-legacy ai-train ai-explore ai-annotate ingest-ufop-patents
