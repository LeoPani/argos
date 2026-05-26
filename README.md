# Argos — IP Intelligence Platform

> Plataforma de inteligência competitiva para Propriedade Intelectual, desenvolvida para o NIT-UFOP.  
> Classifica, monitora e analisa patentes, marcas e publicações acadêmicas com IA.

---

## Visão Geral

O Argos integra três frentes de trabalho em PI universitária:

| Módulo | O que faz |
|--------|-----------|
| **INPI Monitor** | Ingere RPIs semanais do INPI, classifica despachos por categoria IPC via BERTimbau fine-tuned |
| **UFOP Intelligence** | Coleta teses/dissertações do repositório OAI-PMH da UFOP e avalia potencial de patenteabilidade |
| **Smart Filing** | Assistente de depósito — pontua novidade, gera reivindicações independentes via LLM (Groq llama-3.3-70b) com alertas Art. 10 LPI |
| **Prior Art Search** | Busca semântica TF-IDF cosine (Salton & Buckley 1988) sobre corpus de 14 k+ despachos INPI + portfolio |
| **Arbitragem** | Comparação de PIs com score heurístico ou LLM Groq; vereditos estruturados |
| **TT Marketplace** | Licenciamento de tecnologias UFOP — contratos de transferência, pipeline de negociação |
| **IP Timestamps** | Registro de anterioridade com cadeia de hashes SHA-256 (prova de existência sem blockchain) |
| **Chat de PI** | Chat contextualizado com estado real do portfolio injetado no system prompt |

---

## Arquitetura

```
┌─────────────────────────────────────────────────────────────┐
│  Next.js 16 (App Router)  ·  frontend/:3000                │
│  Auth: httpOnly cookie + proxy.ts                          │
└────────────────────┬────────────────────────────────────────┘
                     │ REST
┌────────────────────▼────────────────────────────────────────┐
│  Go API (stdlib net/http)  ·  backend/:8080                │
│  Hexagonal: Domain → Repository → Service → Transport      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────┐  │
│  │ Patents  │  │Trademarks│  │  UFOP    │  │   Stats   │  │
│  │ Service  │  │ Service  │  │ Service  │  │  Service  │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └─────┬─────┘  │
│       └─────────────┴─────────────┴──────────────┘        │
│                         │ SQL                              │
└─────────────────────────┼──────────────────────────────────┘
                          │
┌─────────────────────────▼──────────────────────────────────┐
│  PostgreSQL 16  ·  Docker  ·  pg_trgm                      │
└────────────────────────────────────────────────────────────┘
                          │
┌─────────────────────────▼──────────────────────────────────┐
│  Python FastAPI  ·  ai-service/:8000                       │
│  BERTimbau fine-tuned · SBERT multilingual · TF-IDF+RF     │
└────────────────────────────────────────────────────────────┘
```

**Padrão de AI:** Interface `ai.AIService` com Composite routing — BERT para classificação IPC, Groq (llama-3.3-70b) para geração de texto (claims, relatórios, comparações). O serviço degrada graciosamente se qualquer componente estiver offline.

---

## Stack

| Camada | Tecnologia |
|--------|-----------|
| Frontend | Next.js 16, TypeScript, Tailwind CSS, Recharts, Lucide |
| Backend | Go 1.22+, stdlib `net/http`, `lib/pq`, `log/slog` |
| AI service | Python 3.14, FastAPI, Transformers (BERTimbau), Sentence-BERT |
| LLM | Groq API — `llama-3.3-70b-versatile` |
| Banco | PostgreSQL 16 com `pg_trgm` (full-text search) |
| Infra | Docker Compose, Makefile |

---

## Como Rodar

### Pré-requisitos

- Docker Desktop rodando
- Go 1.22+
- Python 3.10+ com venv `~/argos-ai`
- Node.js 20+
- `GROQ_API_KEY` no ambiente (opcional — degrada graciosamente)

### 1. Banco de dados

```bash
cd backend
make db-up          # sobe o Postgres via Docker
make migrate        # roda as migrations (0001..N)
```

### 2. AI Service (FastAPI + BERTimbau)

```bash
cd ai-service
source ~/argos-ai/bin/activate
uvicorn argos_classifier:app --host 0.0.0.0 --port 8000
```

### 3. Go API

```bash
cd backend
export GROQ_API_KEY=<sua-chave>   # opcional
make run-api                       # compila e serve em :8080
```

### 4. Frontend

```bash
cd frontend
cp .env.local.example .env.local  # ou edite .env.local
npm install
npm run dev                        # serve em :3000
```

Acesse `http://localhost:3000` — será redirecionado para `/login`.  
Use a chave em `frontend/.env.local` → `ARGOS_ACCESS_KEY`.

### Variáveis de ambiente (`frontend/.env.local`)

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
GROQ_API_KEY=gsk_...
ANTHROPIC_API_KEY=sk-ant-...   # opcional (fallback para chat)
ARGOS_ACCESS_KEY=ARG-XXXX-...  # chave de acesso da plataforma
```

---

## Harvest de Dados

### INPI — RPIs semanais

```bash
cd ai-service
python inpi_rpi_harvest.py --count 5   # últimas 5 RPIs
```

### UFOP — Repositório OAI-PMH

```bash
cd backend
make harvest-ufop   # coleta e analisa teses/dissertações
```

---

## Testes

```bash
# Go
cd backend
go test ./...

# Python
cd ai-service
pytest tests/ -v
```

---

## Estrutura do Repositório

```
argos/
├── backend/                  # Go API (Hexagonal Architecture)
│   ├── cmd/api/              # Entry point
│   ├── internal/
│   │   ├── ai/               # Interfaces + adapters (BERT, LLM, Groq)
│   │   ├── domain/           # Entidades + erros sentinela
│   │   ├── repository/       # Interfaces de persistência
│   │   │   └── postgres/     # Implementações SQL
│   │   ├── service/          # Lógica de negócio + testes
│   │   └── transport/httpapi/ # Handlers HTTP + middleware
│   └── migrations/           # SQL migrations versionadas
│
├── ai-service/               # Python FastAPI
│   ├── argos_classifier.py   # API principal (BERT + SBERT + TF-IDF)
│   ├── training/             # Pipeline de treinamento (4 fases)
│   ├── inpi_rpi_harvest.py   # Coleta de RPIs do INPI
│   └── tests/                # pytest
│
├── frontend/                 # Next.js 16
│   └── src/
│       ├── app/
│       │   ├── (app)/        # Rotas protegidas (com sidebar)
│       │   ├── login/        # Página de login (pública)
│       │   └── api/          # API Routes (chat, auth)
│       ├── components/       # UI reutilizável
│       └── lib/              # hooks, tipos, api client
│
├── METHODOLOGY.md            # Metodologia científica com referências
└── docker-compose.yml
```

---

## Metodologia

Documentada em [`METHODOLOGY.md`](METHODOLOGY.md). Principais referências:

- **Salton & Buckley (1988)** — TF-IDF cosine similarity (busca semântica)
- **Trajtenberg, Henderson & Jaffe (1997)** — Patentes universitárias
- **Lerner & Seru (2017)** — Métricas de qualidade de patentes
- **Honovich et al. (2022)** — LLM-as-annotator (ground truth UFOP)
- **Triple Helix / AUTM** — Métricas de TT (Technology Transfer)

---

## Contexto Acadêmico

Desenvolvido como plataforma de pesquisa para o **NIT-UFOP** (Núcleo de Inovação Tecnológica da Universidade Federal de Ouro Preto).

Objetivos:
- Automatizar a análise de patenteabilidade de pesquisas UFOP
- Monitorar prior art brasileiro via INPI em tempo real  
- Gerar métricas de TT comparáveis com benchmarks AUTM/FORTEC
- Demonstrar aplicação de LLMs em domínio jurídico especializado (PI brasileira)

---

## Licença

Uso interno UFOP/NIT. Todos os direitos reservados.

---

*Argos — "O gigante de muitos olhos" (mitologia grega). Um olho para cada aspecto da propriedade intelectual.*
