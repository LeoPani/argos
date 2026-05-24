# Argos — Deploy em produção

Guia completo para subir Argos em ambiente de produção usando free-tier
de serviços confiáveis. Custo total estimado: **R$ 0 / mês** para começar.

## Arquitetura sugerida

```
┌────────────────────────────────────────────────────────┐
│  Vercel              │  Fly.io (GRU)    │  Neon         │
│  ────────            │  ────────────    │  ──────       │
│  argos.vercel.app    │  argos-api.fly   │  argos-pg     │
│  (Next.js frontend)  │  (Go API)        │  (Postgres)   │
│                      │  + Worker        │               │
└────────────────────────────────────────────────────────┘
        ↑                       ↑
        │                       │
   Claude API              FastAPI / BERT
   (Anthropic)             (opcional — pode rodar local)
```

| Componente | Provider | Free tier | Custo @ scale |
|---|---|---|---|
| Frontend Next.js | **Vercel** | 100 GB bandwidth | $20/mo (Pro) |
| API Go | **Fly.io** | 3 shared-CPU machines, 256MB | ~$2/mo per machine |
| Postgres | **Neon** | 0.5 GB storage, 1 project | $19/mo (Launch) |
| BERT FastAPI | **Hugging Face Spaces** | Spaces free tier | $0 |
| Claude API | **Anthropic** | Pay-per-token | ~$0.003 / 1K input |

---

## 1. Postgres no Neon (5 min)

1. Crie conta em https://neon.tech (grátis, sem cartão)
2. **Create project** → nome: `argos-pg` → região: `AWS São Paulo (sa-east-1)`
3. Copie a connection string (algo como `postgres://user:pass@ep-xxx.sa-east-1.aws.neon.tech/argos`)
4. Conecte localmente para aplicar migrations:
   ```bash
   psql 'postgres://user:pass@...neon.tech/argos' \
     -f backend/migrations/0001_create_patents.up.sql
   # repita para 0002..0011
   ```
   Ou use o utilitário do projeto (TODO: o `cmd/migrate` roda tudo automaticamente):
   ```bash
   DATABASE_URL='postgres://...' go run ./cmd/migrate
   ```

5. (Opcional) Popular com dados de demo:
   ```bash
   DATABASE_URL='postgres://...' go run ./cmd/seed
   ```

---

## 2. API Go no Fly.io (10 min)

1. Instale Flyctl: `curl -L https://fly.io/install.sh | sh`
2. `flyctl auth login`
3. No diretório `backend/`:
   ```bash
   flyctl launch --no-deploy
   #   → app name: argos-api
   #   → region:   gru (São Paulo)
   #   → Postgres: skip (usaremos Neon)
   ```
4. Configurar secrets:
   ```bash
   flyctl secrets set \
     DATABASE_URL='postgres://user:pass@...neon.tech/argos?sslmode=require' \
     ANTHROPIC_API_KEY='sk-ant-...' \
     AI_BERT_URL='https://seu-bert.hf.space'   # ou deixar fora, AI tolera ausência
   ```
5. Deploy:
   ```bash
   flyctl deploy
   ```
6. Confira:
   ```bash
   curl https://argos-api.fly.dev/health
   # → {"status":"ok","time":"..."}
   ```

**Custo:** com auto-stop habilitado (já no `fly.toml`), a máquina hiberna quando
ociosa e cold-starts em ~1s no primeiro request. Para um demo isso é R$ 0.

---

## 3. Frontend Next.js no Vercel (3 min)

1. Push do repositório para GitHub (se ainda não estiver).
2. https://vercel.com → **Import Git Repository** → seu repositório
3. **Root Directory**: `frontend/` ← **importante**
4. **Environment Variables**:
   - `NEXT_PUBLIC_API_URL` → `https://argos-api.fly.dev`
   - `ANTHROPIC_API_KEY` → `sk-ant-...` (para o chat funcionar)
5. **Deploy** — leva ~1 min. URL: `https://argos.vercel.app`

---

## 4. BERT FastAPI (opcional) no Hugging Face Spaces

A IA tolera o BERT offline (patentes ficam com `status: failed` — corrigíveis depois).
Se quiser ter classificação automática:

1. https://huggingface.co/spaces → **Create new Space** → SDK: **Docker**
2. Visibility: Public
3. Faça upload do conteúdo de `ai-service/` (incluindo Dockerfile)
4. URL gerada: `https://USER-bert-argos.hf.space` — use no `AI_BERT_URL` da API.

---

## 5. CORS — atualizar lista de origens permitidas

No backend, `middleware.go` permite por padrão:
- `localhost:3000`, `localhost:3001`, `127.0.0.1:3000`
- `*.vercel.app`

Se seu domínio do Vercel for `argos-app.vercel.app`, está OK.
Se usar **domínio custom** (ex: `argos.com.br`), adicione em `internal/transport/httpapi/middleware.go`:

```go
"https://argos.com.br",
"https://www.argos.com.br",
```

E faça redeploy do API.

---

## 6. GitHub Actions (CI/CD automático)

Crie `.github/workflows/deploy.yml`:

```yaml
name: Deploy
on:
  push:
    branches: [main]

jobs:
  deploy-api:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: superfly/flyctl-actions/setup-flyctl@master
      - run: flyctl deploy --remote-only
        working-directory: backend/
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}

  # Frontend deploy is automatic via Vercel GitHub integration
```

Adicione `FLY_API_TOKEN` em **Settings → Secrets → Actions** (token em `flyctl auth token`).

---

## 7. Custom domain (opcional)

### Frontend (Vercel)
1. Project Settings → Domains → Add → `argos.com.br`
2. Configure no DNS do seu provider: CNAME → `cname.vercel-dns.com`

### API (Fly.io)
1. `flyctl certs create api.argos.com.br`
2. DNS: CNAME `api` → `argos-api.fly.dev`
3. Aguarde propagação + cert Let's Encrypt automático

Depois atualize no Vercel: `NEXT_PUBLIC_API_URL=https://api.argos.com.br`

---

## 8. Monitoramento

### Logs em tempo real
- API:        `flyctl logs -a argos-api`
- Frontend:   Vercel dashboard → Logs
- Postgres:   Neon dashboard → Monitoring

### Health checks
- API:        https://argos-api.fly.dev/health
- Frontend:   https://argos.vercel.app/dashboard

### Métricas
- API requests: Fly.io dashboard → Metrics
- Postgres queries: Neon dashboard → Compute usage

---

## 9. Backup automático

### Postgres (Neon)
- **Free tier**: 7 dias de PITR (Point-In-Time Recovery) automático
- Para snapshot manual: Dashboard → Branches → Create branch from current

### Aplicação
- Código está no GitHub (origem da verdade)
- Sem state local — tudo no Postgres

---

## 10. Troubleshooting

### "CORS error" no browser
- Verifique `NEXT_PUBLIC_API_URL` no Vercel
- Confirme que o domínio do frontend está em `middleware.go`

### API retorna 500
- `flyctl logs -a argos-api` → procure por "ERROR"
- Comum: `DATABASE_URL` mal configurada → confira `flyctl secrets list`

### Cold start lento (>5s)
- Aumente `min_machines_running` para 1 em `fly.toml` (~$2/mo a mais)

### "ANTHROPIC_API_KEY não configurada"
- Vercel: Settings → Environment Variables → Production → Edit
- Re-deploy para aplicar

---

## Checklist final

- [ ] Postgres no Neon com 11 migrations aplicadas
- [ ] Seed rodado (101 patentes, 30 marcas, etc)
- [ ] API no Fly.io com `DATABASE_URL` configurado
- [ ] Vercel apontando para a API correta
- [ ] CORS atualizado para domínio do Vercel
- [ ] `/health` retorna 200
- [ ] `/dashboard` carrega dados ao vivo
- [ ] Chat com Claude funcionando (se key configurada)

---

## Como **eu** rodo localmente vs. produção

| Componente | Local | Produção |
|---|---|---|
| Postgres | Docker `argos_postgres` | Neon |
| API Go | `make run-api` | Fly.io |
| Worker | `make run-worker` | Fly.io machine separada (opcional) |
| Frontend | `npm run dev` | Vercel |
| BERT | Local uvicorn | Hugging Face Spaces (opcional) |

**Diferença na config:** apenas as 3 secrets (`DATABASE_URL`, `ANTHROPIC_API_KEY`, `AI_BERT_URL`). Tudo mais é igual.
