# Argos — Roadmap & Justificativas para Banca

Este documento mapeia **o que foi implementado**, **o que ficou para depois** e
**por que cada escolha foi feita**. Serve como base para defesa acadêmica do
projeto.

---

## ✅ Já implementado e em produção

### Núcleo de PI — completo

| Feature | Status | Validação acadêmica |
|---|---|---|
| Cadastro/consulta de patentes (INPI) | ✅ | Lei 9.279/1996 |
| Cadastro de marcas (Nice classification) | ✅ | Acordo de Nice 1957 |
| Sistema de arbitragem com **IA heurística** | ✅ | Score por similaridade fonética PT-BR + Nice overlap |
| **Contratos TT** com 14 campos NIT-UFOP | ✅ | Lei 10.973/2004 (Marco Legal) |
| **Patent pools** com rateio de share | ✅ | Padrão SEP/FRAND |
| **Watchlists/alertas** com auto-dispute | ✅ | Padrão CompuMark/Markify |
| Chat de PI assistido por **Claude** | ✅ | Threads persistidas |
| **UFOP Intelligence** (OAI-PMH + portal scraper) | ✅ | Dublin Core padrão |

### Indicadores acadêmicos — 10 com peer-review

| Indicador | Paper seminal | Citações |
|---|---|---|
| **AUTM Health Score** | AUTM Licensing Survey FY2022 + FORTEC 2023 | Standard US/BR |
| **TT Conversion Funnel** | AUTM Survey methodology | Standard US |
| **HJT IPC Diversity** | Hall, Jaffe & Trajtenberg (2001) NBER WP 8498 | 5000+ |
| **Triple Helix Score** | Etzkowitz & Leydesdorff (2000) *Research Policy* | 4000+ |
| **Inventor h-index proxy** | Hirsch (2005) *PNAS* + Wong-Pang (2011) | 8000+ |
| **PCI (Patent Composite Index)** | Lanjouw & Schankerman (2004) *Economic Journal* | 1200+ |
| **Maintenance Decision** | Schankerman-Pakes (1986) + Pakes (1986) *Econometrica* | 2000+ |
| **Knowledge Stock** | Griliches (1990) *JEL* — perpetual inventory | 6000+ |
| **Royalty Forecast** | Pakes (1986) *Econometrica* — patent as option | 2000+ |
| **Smart Filing patentability** | Lerner-Seru (2017) *RFS* + Bessen (2008) | 800+ |

### Frontend — 18 rotas + componentes

- 8 abas principais conectadas ao backend real
- Páginas de detalhe (`/patents/[id]`, `/trademarks/[id]`, `/inventors/[name]`)
- `/marketplace` público (sem auth)
- `/calendario` NIT auto-populado
- `/smart-filing` wizard
- `/metodologia` (defesa acadêmica)
- ⌘K search global federado
- Toast notifications, CSV export, dark theme
- Visualização **citation network** (Narin 1994) em SVG nativo

### Dados reais (em produção)

- **Repositório UFOP via OAI-PMH** (DSpace 7) — código + execução validados
  - Sets identificados: DEDIR, DEMIN, PPG Direito, Escola de Minas
  - 60 publicações reais Direito + Eng. Minas testadas
- **Portal de notícias UFOP** — scraper rodando, 6/21 matches PI keywords
- **Postgres com 12 migrations** + idempotência via ON CONFLICT
- **Stack docker-compose completo** (postgres + api + migrate + frontend)

---

## ⏳ Em construção / parcial (marcado no UI com tarja)

### 🟡 Lens.org integração real
**Status atual:** cliente dual-mode (real se `LENS_API_TOKEN`, senão mock determinístico calibrado via NBER 2001).

**Por que falta:**
- Acesso acadêmico ao Lens Patent API requer **cadastro institucional via UFOP** (~1-3 dias de aprovação).
- Free tier: 5.000 calls/dia (suficiente).

**Quando ativar:**
- Cadastro em https://www.lens.org/lens/user/subscriptions com email @ufop.edu.br.
- Setar `LENS_API_TOKEN` no env → automaticamente sai do modo mock.

**Impacto científico:**
- Habilita PCI Lanjouw-Schankerman **completo** (citations reais)
- HJT Originality/Generality com base citation real (não apenas IPC portfolio)

---

### 🟡 BERT FastAPI em produção
**Status:** classifier rodando local (uvicorn + BERTimbau fine-tuned, 8 classes IPC).

**Por que falta:**
- Modelo `argos_model/` é pesado (~440MB) — não viável para Vercel/Fly free tier.
- Hugging Face Spaces é a opção indicada (Docker, gratuito).

**Mitigação atual:**
- Sistema tolerante a BERT offline — `Smart Filing` continua rodando com heurística pura, patentes ficam `status: failed` (corrigível via re-classify endpoint).

---

### 🟡 INPI bulk ingestion
**Status:** worker scaffolded em `worker/inpi_patents/` (downloader + parser + pipeline).

**Por que adiado:**
- RPI semanal do INPI: ZIP ~50MB, schema BRPI XML complexo, sujeito a quebras.
- Requer BERT FastAPI rodando para classificação automática.
- Tempo de processamento: ~30 min para 1000 patentes.

**Quando rodar:**
- Quando BERT estiver em Hugging Face Spaces.
- 1 RPI traz ~1000-3000 patentes brasileiras reais.

---

## 📚 Documentado para fases futuras

### Phase 4 — Blockchain timestamping
**Por que NÃO foi feito:**
- Decisão explícita (Leonardo) de **adiar blockchain real** para o MVP.
- Custo de transação em Polygon (~$0.01/tx) é trivial mas requer infra adicional.
- Não agrega à validação acadêmica nesta fase.

**Quando faria sentido:**
- Após validação por orientador, se o projeto for adotado pela UFOP de fato.
- Hash de provas de disputas + timestamp on-chain serviria como evidência forense.

**Estado atual:** UI tem placeholders (`blockchain_hash` opcional em ativos), nada conectado.

---

### Phase 5 — Lattes integration
**Por que NÃO foi feito:**
- **LGPD**: dados de currículo são pessoais, requerem consentimento explícito.
- Plataforma Lattes exige autenticação (sem API pública).
- Scraping é frágil e violaria ToS.

**Alternativa adotada:**
- Página `/inventors/[name]` agrega dados a partir de patentes UFOP públicas
  (Repositório institucional já tem essa metadata).

---

### Phase 6 — Web of Science / Scopus
**Por que NÃO foi feito:**
- Assinaturas institucionais pagas, sem free tier público.
- UFOP tem acesso via CAPES, mas APIs são acessíveis apenas via VPN institucional.

**Quando faria sentido:**
- Acesso autenticado via terminal UFOP para citações acadêmicas de teses → patente.

---

### Phase 7 — Worker INPI proativo (cron)
**Por que NÃO foi feito agora:**
- Requer infraestrutura: cron + retry + monitoramento.
- Operacional, não acadêmico.

**Quando faria sentido:**
- Em produção real, agendamento semanal coincidindo com publicação RPI.

---

## 🤔 Decisões científicas defensáveis

| Escolha | Por quê |
|---|---|
| **Heurística** em vez de Claude para arbitragem | Determinismo (defensável); custo zero; passível de auditoria. Claude entra como upgrade Phase 2. |
| **Mock Lens** em vez de assinatura paga | Permite demonstração imediata; cliente já é dual-mode plug-and-play. |
| **Métricas adaptadas** (HJT-light, h-index proxy) quando dados reais ausentes | Mantém o rigor da fórmula original mas adapta inputs ao que temos. Sempre documentado na UI. |
| **OAI-PMH** vs scraping da busca INPI | OAI é padrão internacional (Open Archives Initiative 2001), legal e estável. |
| **Postgres** monolítico vs microservices | Estado pequeno (< 100k registros estimado), JSONB cobre flexibilidade. Premature optimization desnecessária. |
| **Next.js App Router** vs SSR custom | Padrão de fato em 2026; Vercel/Netlify zero config. |
| **Go stdlib** vs frameworks (gin/echo) | `net/http` 1.22+ tem pattern matching nativo. Menos dependências = menos risco de supply chain. |

---

## 📊 Métricas do projeto

```
Backend Go:     ~12.000 linhas (50+ endpoints REST)
Frontend Next:  ~10.000 linhas (18 rotas)
Migrations:     12 SQL files (idempotentes)
Papers citados: 10 com peer-review consolidado
Domínios:       16 entidades (Patent, Trademark, Dispute, Contract,
                 Pool, Watchlist, ChatThread, Inventor, etc)
```

---

## 🎓 Recomendação para slideshow

Estrutura sugerida para apresentação ao orientador:

1. **Problema** — NIT-UFOP não tem ferramenta de inteligência de PI
2. **Estado da arte** — AUTM, Etzkowitz, HJT, Lens.org como referências
3. **Solução proposta** — Argos: 10 indicadores peer-reviewed + automação
4. **Implementação** — Stack Go + Postgres + Next.js + BERT
5. **Resultados** — Métricas computadas com dados reais UFOP
6. **Limitações** — BERT em produção, Lens.org acesso, blockchain (esta lista)
7. **Trabalhos futuros** — fases 4-7 acima

---

## 🚀 Próximos passos de execução

Em ordem de impacto:

1. **Cadastrar UFOP no Lens.org acadêmico** — desbloqueia PCI completo
2. **Subir BERT em Hugging Face Spaces** — desbloqueia Smart Filing 100%
3. **Rodar INPI bulk ingestion** (1 RPI) — popula sistema com 1000+ patentes reais
4. **Deploy em produção** (Neon + Fly.io + Vercel) — DEPLOY.md já tem o passo-a-passo
5. **Apresentação ao orientador** com slideshow baseado nesta estrutura
