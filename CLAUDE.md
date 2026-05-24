# Argos — Competitive IP Intelligence Platform

## What This Project Is

Argos is a competitive intelligence platform for patents, trademarks, and academic
publications. The architecture is multi-phase. Phase 1 (current) ingests Brazilian
patents from INPI, classifies them via a fine-tuned BERTimbau model, and serves them
via REST API.

## Tech Stack

- **Backend:** Go (stdlib net/http, lib/pq, log/slog)
- **AI service:** Python 3.14 + FastAPI + transformers (BERTimbau classifier)
- **Database:** PostgreSQL 16 (running in Docker, container `argos_postgres`)
- **Architecture:** Hexagonal / Clean Architecture (Domain → Repo → Service → Transport)
- **AI integration:** Hybrid pattern with `ai.AIService` interface, currently routes to
  BERT classifier; LLM adapter stubbed for future generation tasks (prior-art reports,
  dispute summaries)

## Repo Layout

## Current Status

### Phase 1 — 95% complete
- [x] Domain layer with 6 entities + sentinel errors
- [x] Postgres repository (Patent only so far)
- [x] Service layer with PatentService.Ingest (validate → classify → persist)
- [x] HTTP transport with logging + recovery middleware
- [x] Migration 0001 applied (patents table)
- [x] Docker Postgres running with pg_trgm for full-text search
- [x] FastAPI service classifying via fine-tuned BERTimbau (8 categories: 0..7)
- [ ] **Last bug:** POST /api/v1/patents failing with "null value in column inventors"
      Fix applied in patent_repo.go but API binary may be outdated. Need to:
      1. Confirm fix in source (lines 56, 65)
      2. Restart API to recompile
      3. Test end-to-end

### Future phases (designed, not built)
- Phase 2: Lens.org integration
- Phase 3: Trademark CRUD + prior-art search
- Phase 4: Blockchain timestamping (Polygon)
- Phase 5: Internal arbitration/dispute system
- Phase 6: Web of Science integration

## How to Run Everything

```bash
# Terminal A: Start Postgres
cd ~/projetos/argos/backend && make db-up

# Terminal B: Start FastAPI (LEAVE RUNNING)
cd ~/projetos/argos/ai-service
source ~/argos-ai/bin/activate
uvicorn api_argos:app

# Terminal C: Start Go API (LEAVE RUNNING)
cd ~/projetos/argos/backend
make run-api

# Terminal D: Test/Development
curl -X POST http://localhost:8080/api/v1/patents ...
```

## Code Conventions

- **Errors:** Sentinel errors (domain.ErrNotFound, etc.) in repo; wrapped in service;
  translated to HTTP status in handler
- **Logging:** Structured slog with JSON output
- **Package naming:** No underscores. internal/transport/http → internal/transport/httpapi
  (to avoid stdlib conflict)
- **Migrations:** Custom runner. Pattern: NNNN_name.up.sql / NNNN_name.down.sql
- **Commits:** Conventional Commits (feat, fix, refactor)

## Decisions Made

- **Postgres driver:** lib/pq (stable, v1.12.3)
- **Router:** stdlib ServeMux (Go 1.22+ pattern matching)
- **AI architecture:** ai.AIService interface with Composite routing to Classifier (BERT)
  and Generator (LLM, stubbed)
- **BERT contract:** POST /classify {"text": "..."} → {"text_received": "...",
  "predicted_category_id": 0..7}
- **Service tolerates AI failure:** If BERT is down, patent saved as status: "failed"

## User Context

- **Name:** Leonardo Paniago (@LeoPani)
- **Environment:** macOS Apple Silicon, Docker Desktop, Python 3.14
- **Skill:** Beginner-to-intermediate Go developer
- **Language:** Portuguese (Brazil) for conversation; English for code/commits
- **Repo:** https://github.com/LeoPani/argos

## Common Gotchas

- Docker must be running before `make db-up`
- uvicorn must run from `ai-service/` folder to find `argos_model/`
- Never use `--reload` on uvicorn
- Postgres dev password: `argos_dev`
