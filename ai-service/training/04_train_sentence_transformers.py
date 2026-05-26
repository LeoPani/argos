"""
Argos — Fase 4: Modelo avançado via Sentence Transformers PT-BR.

Usa paraphrase-multilingual-MiniLM-L12-v2 (Reimers & Gurevych 2019, 2020)
para gerar embeddings de 384d de cada trabalho, depois LogReg / RandomForest
em cima dos embeddings.

Validação metodológica:
  Reimers, N., & Gurevych, I. (2019). "Sentence-BERT: Sentence embeddings
    using Siamese BERT-networks." EMNLP-IJCNLP 2019.
  Reimers, N., & Gurevych, I. (2020). "Making Monolingual Sentence Embeddings
    Multilingual using Knowledge Distillation." EMNLP 2020. — origem do
    paraphrase-multilingual-MiniLM, treinado em 50+ idiomas incluindo PT.

Vantagens vs baseline TF-IDF (Fase 3):
  - Captura semântica (sinônimos, paráfrases)
  - Funciona melhor em datasets pequenos
  - Resultado é vetor 384d reutilizável (busca semântica de prior art interno)

Fontes de ground truth (auto-detect, em ordem de preferência):
  1. ai-service/training/data/annotations.jsonl  — anotações LLM (Fase 1)
  2. Postgres ufop_opportunities                 — heurística v2 (fallback)

Requisitos: pip install sentence-transformers scikit-learn psycopg2-binary
Tempo: ~3-8 min em CPU pra 3000 trabalhos (download do modelo na 1ª vez).

Cache de embeddings: por padrão lê/grava `models/embeddings_cache.npz` —
recalcular embeddings é o passo lento. Use --no-cache pra forçar regerar.
"""

import argparse
import json
import os
import sys
from pathlib import Path

try:
    import numpy as np
    from sentence_transformers import SentenceTransformer
    from sklearn.ensemble import RandomForestClassifier
    from sklearn.linear_model import LogisticRegression
    from sklearn.metrics import classification_report, confusion_matrix
    from sklearn.model_selection import train_test_split, cross_val_score
    import joblib
except ImportError as e:
    sys.exit(f"pip install sentence-transformers scikit-learn  ({e})")

try:
    import psycopg2
except ImportError:
    psycopg2 = None  # opcional — só pra fallback Postgres

DATA_DIR = Path(__file__).parent / "data"
MODELS   = Path(__file__).parent / "models"
OUT      = Path(__file__).parent / "outputs"
ANNOT    = DATA_DIR / "annotations.jsonl"
DB_URL   = os.environ.get("DATABASE_URL",
    "postgres://argos:argos_dev@localhost:5432/argos")

MODEL_NAME = "paraphrase-multilingual-MiniLM-L12-v2"  # 50+ idiomas, suporta PT-BR
LETTERS    = ["A", "B", "C", "D", "E", "F", "G", "H"]


# ─── Loaders (mesma lógica da Fase 3) ──────────────────────────────────────

def load_llm_annotations() -> tuple[list[dict], str]:
    if not ANNOT.exists():
        return [], "—"
    rows = []
    for line in ANNOT.read_text().splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            rows.append(json.loads(line))
        except json.JSONDecodeError:
            continue
    src = "annotations.jsonl"
    if rows and rows[0].get("_provider"):
        src += f" (provider={rows[0].get('_provider')}, model={rows[0].get('_model', '?')})"
    return rows, src


def load_postgres_as_ground_truth() -> tuple[list[dict], str]:
    if psycopg2 is None:
        return [], "psycopg2 não instalado"
    try:
        conn = psycopg2.connect(DB_URL)
    except Exception as e:
        return [], f"conexão falhou: {e}"
    cur = conn.cursor()
    cur.execute("""
        SELECT id, title, COALESCE(abstract, ''), department,
               is_patentable, ipc_category, confidence,
               COALESCE(rationale, ''), classifier_version
        FROM ufop_opportunities
        WHERE is_patentable IS NOT NULL
    """)
    rows = []
    for r in cur.fetchall():
        rows.append({
            "opportunity_id":   r[0],
            "title":            r[1],
            "abstract":         r[2],
            "department":       r[3],
            "is_patentable":    bool(r[4]),
            "ipc_category":     r[5],
            "confidence":       r[6] or 0.5,
            "rationale":        r[7],
            "_provider":        "postgres",
            "_model":           r[8] or "heuristic-v2",
        })
    cur.close()
    conn.close()
    return rows, f"Postgres ({len(rows)} rows, classifier=heuristic-v2)"


# ─── Embeddings (cache aware) ──────────────────────────────────────────────

def build_texts(rows: list[dict]) -> list[str]:
    return [
        f"{(r.get('title') or '')}. {(r.get('abstract') or '')[:1500]}. "
        f"DEP: {(r.get('department') or '')}"
        for r in rows
    ]


def get_or_compute_embeddings(texts: list[str], ids: list[int],
                              cache_path: Path, use_cache: bool) -> np.ndarray:
    if use_cache and cache_path.exists():
        try:
            cached = np.load(cache_path, allow_pickle=False)
            cached_ids = cached["ids"].tolist()
            if cached_ids == ids and cached["X"].shape[0] == len(texts):
                print(f"  ✓ cache hit ({cache_path.name}) — {cached['X'].shape}")
                return cached["X"]
            print(f"  ⚠ cache stale (ids/shape mismatch) — regenerando")
        except Exception as e:
            print(f"  ⚠ cache leitura falhou ({e}) — regenerando")

    print(f"  Carregando modelo {MODEL_NAME}…")
    model = SentenceTransformer(MODEL_NAME)
    print(f"  Modelo carregado · dim={model.get_sentence_embedding_dimension()}")

    print(f"  Gerando embeddings de {len(texts)} textos…")
    X = model.encode(texts, show_progress_bar=True, convert_to_numpy=True,
                     batch_size=32)
    np.savez_compressed(cache_path, X=X, ids=np.array(ids))
    print(f"  ✓ embeddings cached em {cache_path.name}")
    return X


# ─── Trainers ──────────────────────────────────────────────────────────────

def train_patentability(X, y, report: list[str]) -> dict | None:
    if len(set(y)) < 2:
        report.append(f"  ⚠ Apenas 1 classe — pulando ({set(y)})")
        return None

    X_tr, X_te, y_tr, y_te = train_test_split(
        X, y, test_size=0.2, random_state=42, stratify=y)

    clf = LogisticRegression(class_weight="balanced", max_iter=2000, n_jobs=-1)
    clf.fit(X_tr, y_tr)
    pred = clf.predict(X_te)
    proba = clf.predict_proba(X_te)[:, 1]

    report.append("Patenteabilidade — SBERT + LogReg:")
    report.append(classification_report(
        y_te, pred,
        target_names=["não-patenteável", "patenteável"], digits=3,
        zero_division=0))

    cv = cross_val_score(clf, X, y, cv=5, scoring="f1")
    report.append(f"  5-fold CV F1: {cv.mean():.3f} ± {cv.std():.3f}")

    cls_report = classification_report(
        y_te, pred,
        target_names=["não-patenteável", "patenteável"], digits=3,
        output_dict=True, zero_division=0)

    # Salvar
    joblib.dump(clf, MODELS / "sbert_logreg_patentability.pkl")

    return {
        "macro_f1": cls_report["macro avg"]["f1-score"],
        "accuracy": cls_report["accuracy"],
        "cv_f1_mean": float(cv.mean()),
        "cv_f1_std": float(cv.std()),
        "n_test": len(y_te),
        "mean_predicted_proba": float(proba.mean()),
    }


def train_ipc(X, y, report: list[str]) -> dict | None:
    mask = y >= 0
    X_ipc = X[mask]
    y_ipc = y[mask]
    if X_ipc.shape[0] < 50:
        report.append(f"  ⚠ Apenas {X_ipc.shape[0]} amostras IPC — pulando")
        return None
    if len(set(y_ipc)) < 2:
        report.append(f"  ⚠ Apenas 1 classe IPC — pulando ({set(y_ipc)})")
        return None

    X_tr, X_te, y_tr, y_te = train_test_split(
        X_ipc, y_ipc, test_size=0.2, random_state=42)

    clf = RandomForestClassifier(
        n_estimators=400, max_depth=30,
        class_weight="balanced", random_state=42, n_jobs=-1)
    clf.fit(X_tr, y_tr)
    pred = clf.predict(X_te)

    labels = sorted(set(y_te) | set(pred))
    names  = [LETTERS[i] for i in labels]
    report.append("IPC multiclass — SBERT + RandomForest:")
    report.append(classification_report(
        y_te, pred, target_names=names, digits=3,
        labels=labels, zero_division=0))

    cm = confusion_matrix(y_te, pred, labels=labels)
    report.append("\nConfusion matrix (rows=true, cols=pred):")
    header = "      " + " ".join(f"{n:>4}" for n in names)
    report.append(header)
    for i, name in enumerate(names):
        row = " ".join(f"{cm[i][j]:>4}" for j in range(len(labels)))
        report.append(f"  {name:>3} {row}")

    joblib.dump(clf, MODELS / "sbert_rf_ipc.pkl")
    return {
        "n_samples": int(X_ipc.shape[0]),
        "labels_present": names,
        "confusion_matrix": cm.tolist(),
    }


# ─── Main ──────────────────────────────────────────────────────────────────

def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", choices=["auto", "llm", "postgres"], default="auto")
    parser.add_argument("--no-cache", action="store_true",
                        help="Força regerar embeddings (ignora cache)")
    parser.add_argument("--min-samples", type=int, default=100)
    args = parser.parse_args()

    MODELS.mkdir(parents=True, exist_ok=True)
    OUT.mkdir(parents=True, exist_ok=True)
    report = []

    # Load
    rows, source_desc = [], "—"
    if args.source in ("auto", "llm"):
        rows, source_desc = load_llm_annotations()
    if (not rows) and args.source in ("auto", "postgres"):
        rows, source_desc = load_postgres_as_ground_truth()

    if not rows:
        print(f"Nenhuma fonte disponível ({source_desc}).")
        return 1
    if len(rows) < args.min_samples:
        print(f"⚠ Apenas {len(rows)} amostras — abaixo do mínimo {args.min_samples}")
        return 1

    using_heuristic = "postgres" in source_desc.lower() or "heuristic" in source_desc.lower()
    report.append("=" * 70)
    report.append("ARGOS — Fase 4: Sentence-BERT supervised")
    report.append("=" * 70)
    report.append("")
    report.append(f"Encoder:  {MODEL_NAME}")
    report.append(f"Source:   {source_desc}")
    report.append(f"Samples:  {len(rows)}")
    if using_heuristic:
        report.append("")
        report.append("⚠ Ground truth = heurística v2 (não LLM). Mesma ressalva da Fase 3:")
        report.append("  o modelo aprenderá a imitar a heurística. Re-treine após Fase 1.")
    report.append("")

    # Embeddings
    texts = build_texts(rows)
    ids   = [r["opportunity_id"] for r in rows]
    cache_path = MODELS / "embeddings_cache.npz"
    report.append("━━━ Embeddings ━━━")
    X = get_or_compute_embeddings(texts, ids, cache_path, use_cache=not args.no_cache)
    report.append(f"  shape: {X.shape}")
    report.append("")

    # Models
    y_pat = np.array([1 if r.get("is_patentable") else 0 for r in rows])
    y_ipc = np.array([
        r.get("ipc_category") if r.get("ipc_category") is not None else -1
        for r in rows
    ])

    report.append("━━━ Modelo 1: SBERT + LogReg (patenteabilidade) ━━━")
    pat_metrics = train_patentability(X, y_pat, report)
    report.append("")

    report.append("━━━ Modelo 2: SBERT + RandomForest (IPC) ━━━")
    ipc_metrics = train_ipc(X, y_ipc, report)
    report.append("")

    # Comparação rápida com Fase 3 se artefatos existirem
    rf_meta = MODELS / "metadata.json"
    if rf_meta.exists():
        try:
            base = json.loads(rf_meta.read_text())
            base_pat = base.get("patentability_metrics") or {}
            if pat_metrics and base_pat:
                delta_f1 = pat_metrics["macro_f1"] - base_pat.get("macro_f1", 0)
                report.append("━━━ Comparação Fase 4 (SBERT) vs Fase 3 (TF-IDF) ━━━")
                report.append(f"  Macro F1: {base_pat.get('macro_f1', 0):.3f} → {pat_metrics['macro_f1']:.3f} ({delta_f1:+.3f})")
                report.append(f"  CV F1:    {base_pat.get('cv_f1_mean', 0):.3f} → {pat_metrics['cv_f1_mean']:.3f}")
                report.append("")
        except Exception:
            pass

    # Salva metadata (sobrescreve metadata.json com info SBERT também)
    meta = {
        "encoder": MODEL_NAME,
        "embedding_dim": int(X.shape[1]),
        "n_samples": int(X.shape[0]),
        "source": source_desc,
        "using_heuristic_groundtruth": using_heuristic,
        "patentability_sbert": pat_metrics,
        "ipc_sbert": ipc_metrics,
        "ref": "Reimers & Gurevych (2020) EMNLP",
    }
    (MODELS / "sbert_metadata.json").write_text(
        json.dumps(meta, indent=2, ensure_ascii=False))

    text_report = "\n".join(report)
    print(text_report)
    (OUT / "train_sbert_report.txt").write_text(text_report)
    print(f"\n→ Models in: {MODELS}")
    print(f"→ Report  in: {OUT}/train_sbert_report.txt")
    return 0


if __name__ == "__main__":
    sys.exit(main())
