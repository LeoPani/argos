# Argos — Metodologia de IA (defesa acadêmica)

Este documento explicita **o que é IA de verdade vs. heurística** no projeto,
e descreve o pipeline supervisionado treinado contra dataset UFOP real.

---

## 🚨 Honestidade científica primeiro

### O que ANTES era "IA" no nome mas não no fato

O analisador inicial (`analyzer.go` v1) era **inteligência heurística**:

```
1. Score = count(keywords no título) × peso + count(keywords no abstract) × peso
2. Threshold-based classification (HIGH ≥ 5.5, MEDIUM ≥ 3.0, LOW < 3.0)
3. "ai_analysis" gerado via template Mad Libs (string formatting)
```

**Não é IA no sentido moderno.** É um sistema de regras válido como **baseline**,
mas insuficiente como contribuição científica.

### O que AGORA é IA de verdade

O pipeline `ai-service/training/` implementa:

1. **Anotação automática** via Claude (LLM-as-annotator validado por Honovich 2022)
2. **Modelo supervisionado** treinado nessas anotações
3. **Embeddings semânticos** via Sentence-BERT multilingual
4. **Métricas reprodutíveis** (precision, recall, F1, confusion matrix)
5. **Comparação baseline vs avançado** (TF-IDF vs SBERT)

---

## 📚 Fundamentação teórica

### LLM-as-annotator (substitui anotação manual)

**Problema:** o NIT-UFOP não tem dataset rotulado de "patenteável vs não" pra treinar.

**Solução:** usar um LLM forte (Claude) como **oracle**, anotando trabalhos
acadêmicos automaticamente.

**Papers validando essa abordagem:**

1. **Honovich, O., Scialom, T., Levy, O., & Schick, T. (2022)**.
   *Unnatural Instructions: Tuning Language Models with (Almost) No Human Labor.*
   arXiv:2212.09689. — Mostra que LLMs podem gerar dados de instrução com
   qualidade próxima à humana.

2. **Wang, Y., et al. (2022)**.
   *Self-Instruct: Aligning Language Models with Self-Generated Instructions.*
   ACL 2023. — Validou geração automática de anotações com 85%+ de concordância
   com anotadores humanos especialistas.

3. **He, X., et al. (2024)**. *AnnoLLM: Making Large Language Models to Be Better
   Crowdsourced Annotators.* NAACL 2024. — Confirma o uso de LLMs como anotadores
   em domínios técnicos.

**Custo:** ~$1.30 por dataset completo (261 trabalhos × ~$0.005).
**Vantagem:** reprodutível, escalável, sem viés de anotador humano cansado.

### Sentence-BERT (substitui Jaccard de bigrams)

**Problema:** comparação textual por Jaccard/trigramas perde semântica.
"Aprendizado de máquina" e "redes neurais" são quase sinônimos mas Jaccard daria 0.

**Solução:** embeddings de sentenças (vetores 384d) — palavras similares no
sentido ficam próximas no espaço vetorial.

**Papers:**

1. **Reimers, N., & Gurevych, I. (2019)**.
   *Sentence-BERT: Sentence Embeddings using Siamese BERT-Networks.*
   EMNLP-IJCNLP 2019. — Origem do Sentence-BERT, citado 10000+ vezes.

2. **Reimers, N., & Gurevych, I. (2020)**.
   *Making Monolingual Sentence Embeddings Multilingual using Knowledge
   Distillation.* EMNLP 2020. — Origem específica do
   `paraphrase-multilingual-MiniLM-L12-v2` que usamos, treinado em 50+
   idiomas incluindo português.

**Modelo escolhido:**
`sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2`
- 384 dimensões
- Suporta PT-BR
- ~120MB
- Inferência em CPU: ~50ms por trabalho

### Modelos supervisionados

**Baseline (TF-IDF + Random Forest):**
- Salton & Buckley (1988). *Term-weighting approaches in automatic text
  retrieval.* IPM, 24(5). — Origem TF-IDF.
- Breiman, L. (2001). *Random Forests.* Machine Learning, 45(1).

**Avançado (SBERT + LogReg/RF):**
- Reimers & Gurevych (acima).
- Pedregosa et al. (2011). *Scikit-learn: Machine Learning in Python.* JMLR.

### Avaliação

- Kohavi, R. (1995). *A Study of Cross-Validation and Bootstrap for Accuracy
  Estimation and Model Selection.* IJCAI. — Origem do k-fold cross-validation.
- Métricas reportadas: precision, recall, F1, accuracy, confusion matrix.
- Split: 80/20 train/test estratificado.
- Cross-validation: 5-fold no dataset completo.

---

## 🔬 Pipeline completo

### Fase 1 — Anotação automática (`01_annotate.py`)

```python
INPUT:  261 oportunidades UFOP do banco (título + abstract + departamento)
ORACLE: Claude Sonnet 4.6 com prompt jurídico estruturado
OUTPUT: annotations.jsonl, uma linha por trabalho:
  {
    "opportunity_id": 42,
    "title": "...",
    "is_patentable": true,
    "ipc_category": 2,  // 0..7 = A..H
    "confidence": 0.85,
    "rationale": "Processo hidrometalúrgico com aplicação industrial..."
  }
```

**Critérios do prompt (Art. 8 LPI):**
- Novidade técnica
- Atividade inventiva (não-óbvio)
- Aplicação industrial

**Exclusões (Art. 10 LPI):**
- Teorias científicas, descobertas
- Esquemas jurídicos, comerciais
- Programas de computador per se
- Concepções abstratas

### Fase 2 — Análise exploratória (`02_explore.py`)

- Distribuição de classes (patenteável/não, por IPC)
- Confiança média do anotador
- **Concordância entre Claude e heurística atual** — métrica crítica
  - Se concordância for alta: heurística é boa baseline
  - Se baixa: justifica o investimento em modelo treinado

### Fase 3 — Baseline TF-IDF + Random Forest (`03_train_baseline.py`)

- TF-IDF: 3000 features, n-gramas 1-2, min_df=2
- RandomForest: 200 árvores, max_depth=20, class_weight=balanced
- Validação: 80/20 split + 5-fold CV
- Saída: `models/rf_patentability.pkl`, `rf_ipc_classifier.pkl`, `tfidf_vectorizer.pkl`

### Fase 4 — Sentence-BERT (`04_train_sentence_transformers.py`)

- Embedding: paraphrase-multilingual-MiniLM-L12-v2 (384d)
- Patentability: LogReg + class_weight=balanced
- IPC: RandomForest + 400 árvores
- Salva embeddings.npy pra busca semântica posterior

### Fase 5 — Servir + integrar (`argos_classifier.py`)

FastAPI substitui `api_argos.py`:
- `POST /classify` — usa SBERT (preferido)
- `POST /classify-baseline` — força TF-IDF
- Retorna IPC + patentability + confidence

Go `analyzer.go` continua chamando `/classify` — mudança é
**transparente pro frontend**.

---

## 📊 Métricas que serão reportadas

| Métrica | O que mede | Reportada em |
|---|---|---|
| **Accuracy** | Pct global de acerto | Slideshow "Resultados" |
| **Precision** | Dos previstos patenteáveis, qts realmente são | Defesa anti-falso-positivo |
| **Recall** | Dos patenteáveis reais, qts achamos | Defesa cobertura |
| **F1-score** | Média harmônica | Métrica principal |
| **5-fold CV F1** | Robustez | Mostra que não é overfit |
| **Confusion matrix** | Erros por classe IPC | Mostra onde modelo confunde |
| **Feature importance** | Palavras + discriminativas | Interpretabilidade |

---

## ⚠️ Limitações explícitas

| Limitação | Mitigação atual | Trabalho futuro |
|---|---|---|
| **Ground truth via LLM** (não humano expert) | Honovich 2022 valida; manter confidence ≥ 0.7 | Eventualmente NIT-UFOP anotar amostra ouro |
| **Dataset pequeno** (~260 trabalhos) | CV de 5 folds; data augmentation via paraphrasing | Expandir pra outros departamentos UFOP |
| **Modelo PT-BR multilingual** (não dedicated) | MiniLM teve bom resultado em benchmarks PT | Fine-tune BERTimbau específico |
| **Sem ground truth de "virou patente?"** | Proxy: patentes UFOP reais via Google Patents | Cruzar com base do INPI quando viável |
| **Domain shift** (treinou Direito+Minas, testar Química) | Avaliação out-of-domain explícita no relatório | Coleta multidisciplinar |

---

## 🎯 Para o slide do orientador

**Antes:**
> "Sistema analisa publicações UFOP e identifica oportunidades de PI."

**Depois (defensável):**
> "Pipeline supervisionado classifica patenteabilidade e IPC de trabalhos
> UFOP usando Sentence-BERT multilingual (Reimers & Gurevych, 2020)
> treinado em 261 trabalhos anotados via Claude como oracle
> (LLM-as-annotator, Honovich et al., 2022). Métricas via 5-fold CV
> com F1-score X.XX para patenteabilidade e accuracy Y.YY para IPC.
> Modelo serve inferência em < 50ms via FastAPI."

---

## 🚀 Como rodar (pra reproduzir)

```bash
cd ai-service
source ~/argos-ai/bin/activate  # ou venv própria
pip install -r requirements.txt

export ANTHROPIC_API_KEY=sk-ant-...
export DATABASE_URL=postgres://argos:argos_dev@localhost:5432/argos

# Fase 1: anota dataset (custo ~$1.30, ~30min)
python training/01_annotate.py

# Fase 2: explora
python training/02_explore.py

# Fase 3: baseline TF-IDF
python training/03_train_baseline.py

# Fase 4: SBERT avançado
python training/04_train_sentence_transformers.py

# Fase 5: servir
uvicorn argos_classifier:app --host 0.0.0.0 --port 8000

# Smoke test
curl -X POST http://localhost:8000/classify \
  -H "Content-Type: application/json" \
  -d '{"text":"Método hidrometalúrgico para extração de lítio em pegmatitos"}'
```

---

## 📦 Reproducibilidade

- Random seed fixo (42) em todos os scripts
- Versão exata dos pacotes em `requirements.txt`
- Modelos serializados em `training/models/` (gitignored, regenerável)
- Dataset anotado em `training/data/annotations.jsonl`
- Cada commit referencia este documento
