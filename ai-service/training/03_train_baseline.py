"""
Argos — Fase 3: Treino baseline (TF-IDF + RandomForest).

Modelo simples, reprodutível, defensável academicamente.
Reporta precision/recall/F1/accuracy + confusion matrix + feature importance.

Validação metodológica:
  - TF-IDF: Salton & Buckley (1988) "Term-weighting approaches in automatic
    text retrieval" — abordagem clássica, citada em milhares de papers
  - RandomForest: Breiman (2001) "Random forests" — Machine Learning, 45(1)
  - Cross-validation: Kohavi (1995) "A study of cross-validation and bootstrap
    for accuracy estimation and model selection"

Por que baseline:
  - Sem GPU, treina em < 1 minuto
  - Interpretável (feature importance = palavras mais discriminativas)
  - Ground truth: anotações LLM (Claude/Groq) quando disponíveis;
    senão a heurística v2 do Postgres (com aviso explícito)

Fontes de ground truth (auto-detect, em ordem de preferência):
  1. ai-service/training/data/annotations.jsonl  — anotações LLM (Fase 1)
  2. Postgres ufop_opportunities                 — heurística v2 (fallback)

Importante: treinar contra heurística NÃO é validação acadêmica completa.
O modelo aprenderá a IMITAR a heurística (acima de 95% accuracy esperado),
não a "verdade". Use só pra validar o pipeline E2E. Pra defesa de banca,
PRECISA da Fase 1 com LLM (oracle externo, validado por Honovich 2022).
"""

import argparse
import json
import os
import sys
from pathlib import Path

try:
    import numpy as np
    from sklearn.ensemble import RandomForestClassifier
    from sklearn.feature_extraction.text import TfidfVectorizer
    from sklearn.metrics import classification_report, confusion_matrix
    from sklearn.model_selection import train_test_split, cross_val_score
    import joblib
except ImportError as e:
    sys.exit(f"pip install scikit-learn joblib numpy  ({e})")

try:
    import psycopg2
except ImportError:
    psycopg2 = None  # opcional — só precisa se usar fallback Postgres

DATA_DIR  = Path(__file__).parent / "data"
MODELS    = Path(__file__).parent / "models"
OUT       = Path(__file__).parent / "outputs"
ANNOT     = DATA_DIR / "annotations.jsonl"
DB_URL    = os.environ.get("DATABASE_URL",
    "postgres://argos:argos_dev@localhost:5432/argos")

LETTERS = ["A", "B", "C", "D", "E", "F", "G", "H"]


# ─── Loaders ────────────────────────────────────────────────────────────────

def load_llm_annotations() -> tuple[list[dict], str]:
    """Output da Fase 1 (01_annotate.py)."""
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
    # Source info
    src = "annotations.jsonl"
    if rows and rows[0].get("_provider"):
        src += f" (provider={rows[0].get('_provider')}, model={rows[0].get('_model', '?')})"
    return rows, src


def load_postgres_as_ground_truth() -> tuple[list[dict], str]:
    """Heurística v2 do Postgres. Formato compatível com annotations.jsonl."""
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


# ─── Training ───────────────────────────────────────────────────────────────

def train_patentability(X, y, vec, models_dir: Path, report_lines: list[str]) -> dict | None:
    if len(set(y)) < 2:
        report_lines.append(f"  ⚠ Apenas 1 classe — pulando patenteabilidade ({set(y)})")
        return None

    X_tr, X_te, y_tr, y_te = train_test_split(
        X, y, test_size=0.2, random_state=42, stratify=y)

    clf = RandomForestClassifier(
        n_estimators=200, max_depth=20,
        class_weight="balanced", random_state=42, n_jobs=-1)
    clf.fit(X_tr, y_tr)
    pred = clf.predict(X_te)

    cls_report = classification_report(
        y_te, pred,
        target_names=["não-patenteável", "patenteável"], digits=3,
        output_dict=True, zero_division=0)
    report_lines.append("Patenteabilidade (binário) — classification_report:")
    report_lines.append(classification_report(
        y_te, pred,
        target_names=["não-patenteável", "patenteável"], digits=3,
        zero_division=0))

    cv = cross_val_score(clf, X, y, cv=5, scoring="f1")
    report_lines.append(f"  5-fold CV F1: {cv.mean():.3f} ± {cv.std():.3f}")

    # Top features
    importances = clf.feature_importances_
    feature_names = vec.get_feature_names_out()
    top = np.argsort(importances)[-20:][::-1]
    report_lines.append("\nTop features discriminativas:")
    for i in top:
        report_lines.append(f"  {importances[i]:.4f}  {feature_names[i]}")

    joblib.dump(clf, models_dir / "rf_patentability.pkl")
    return {
        "macro_f1": cls_report["macro avg"]["f1-score"],
        "accuracy": cls_report["accuracy"],
        "cv_f1_mean": float(cv.mean()),
        "cv_f1_std": float(cv.std()),
        "top_features": [(feature_names[i], float(importances[i])) for i in top[:10]],
    }


def train_ipc(X, y, vec, models_dir: Path, report_lines: list[str]) -> dict | None:
    mask = y >= 0
    X_ipc = X[mask]
    y_ipc = y[mask]
    if X_ipc.shape[0] < 50:
        report_lines.append(f"  ⚠ Apenas {X_ipc.shape[0]} amostras IPC — pulando")
        return None
    if len(set(y_ipc)) < 2:
        report_lines.append(f"  ⚠ Apenas 1 classe IPC — pulando ({set(y_ipc)})")
        return None

    X_tr, X_te, y_tr, y_te = train_test_split(
        X_ipc, y_ipc, test_size=0.2, random_state=42)

    clf = RandomForestClassifier(
        n_estimators=300, max_depth=25,
        class_weight="balanced", random_state=42, n_jobs=-1)
    clf.fit(X_tr, y_tr)
    pred = clf.predict(X_te)

    labels = sorted(set(y_te) | set(pred))
    names  = [LETTERS[i] for i in labels]
    report_lines.append("IPC multiclass — classification_report:")
    report_lines.append(classification_report(
        y_te, pred, target_names=names, digits=3,
        labels=labels, zero_division=0))

    cm = confusion_matrix(y_te, pred, labels=labels)
    report_lines.append("\nConfusion matrix (rows=true, cols=pred):")
    header = "      " + " ".join(f"{n:>4}" for n in names)
    report_lines.append(header)
    for i, name in enumerate(names):
        row = " ".join(f"{cm[i][j]:>4}" for j in range(len(labels)))
        report_lines.append(f"  {name:>3} {row}")

    joblib.dump(clf, models_dir / "rf_ipc_classifier.pkl")
    return {
        "n_samples": int(X_ipc.shape[0]),
        "labels_present": names,
        "confusion_matrix": cm.tolist(),
    }


# ─── Main ───────────────────────────────────────────────────────────────────

def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", choices=["auto", "llm", "postgres"], default="auto",
                        help="Fonte de ground truth (auto detecta a melhor)")
    parser.add_argument("--min-samples", type=int, default=100,
                        help="Mínimo de amostras pra treinar (default: 100)")
    args = parser.parse_args()

    MODELS.mkdir(parents=True, exist_ok=True)
    OUT.mkdir(parents=True, exist_ok=True)
    report = []

    # ── Load data ───────────────────────────────────────────────────────────
    rows, source_desc = [], "—"
    if args.source in ("auto", "llm"):
        rows, source_desc = load_llm_annotations()
    if (not rows) and args.source in ("auto", "postgres"):
        rows, source_desc = load_postgres_as_ground_truth()

    if not rows:
        print(f"Nenhuma fonte disponível ({source_desc}). Saindo.")
        return 1

    if len(rows) < args.min_samples:
        print(f"⚠ Apenas {len(rows)} amostras — abaixo do mínimo {args.min_samples}")
        return 1

    using_heuristic = "postgres" in source_desc.lower() or "heuristic" in source_desc.lower()
    report.append("=" * 70)
    report.append("ARGOS — Fase 3: Treino baseline (TF-IDF + RandomForest)")
    report.append("=" * 70)
    report.append("")
    report.append(f"Source: {source_desc}")
    report.append(f"Samples: {len(rows)}")
    if using_heuristic:
        report.append("")
        report.append("⚠ ATENÇÃO: ground truth é a heurística v2, NÃO um LLM/specialist.")
        report.append("   Modelo vai aprender a IMITAR a heurística — accuracy alta")
        report.append("   é esperada e NÃO indica validação científica.")
        report.append("   Pra defesa de banca, rode Fase 1 com GROQ_API_KEY e re-treine.")
    report.append("")

    # ── Features ────────────────────────────────────────────────────────────
    texts = [
        f"{(r.get('title') or '')}. {(r.get('abstract') or '')[:1500]}. "
        f"DEP: {(r.get('department') or '')}"
        for r in rows
    ]
    y_pat = np.array([1 if r.get("is_patentable") else 0 for r in rows])
    y_ipc = np.array([
        r.get("ipc_category") if r.get("ipc_category") is not None else -1
        for r in rows
    ])

    # Vectorize
    vec = TfidfVectorizer(
        max_features=5000,
        ngram_range=(1, 2),
        min_df=3,
        max_df=0.9,
        stop_words=None,  # PT-BR — manter
        sublinear_tf=True,
    )
    X = vec.fit_transform(texts)
    report.append(f"Feature matrix: {X.shape}")
    report.append(f"Vocabulary size: {len(vec.vocabulary_)}")
    report.append("")

    # ── Train both targets ─────────────────────────────────────────────────
    report.append("━━━ Modelo 1: PATENTEABILIDADE (binário) ━━━")
    pat_metrics = train_patentability(X, y_pat, vec, MODELS, report)
    report.append("")

    report.append("━━━ Modelo 2: IPC (multiclass A..H) ━━━")
    ipc_metrics = train_ipc(X, y_ipc, vec, MODELS, report)
    report.append("")

    # Salva vectorizer + metadata
    joblib.dump(vec, MODELS / "tfidf_vectorizer.pkl")
    metadata = {
        "source": source_desc,
        "n_samples": len(rows),
        "using_heuristic_groundtruth": using_heuristic,
        "patentability_metrics": pat_metrics,
        "ipc_metrics": ipc_metrics,
        "warning": ("Modelo treinado contra heurística — re-treine após Fase 1 LLM"
                    if using_heuristic else None),
    }
    (MODELS / "metadata.json").write_text(json.dumps(metadata, indent=2, ensure_ascii=False))

    # Print + save report
    text_report = "\n".join(report)
    print(text_report)
    (OUT / "train_baseline_report.txt").write_text(text_report)
    print(f"\n→ Models in: {MODELS}")
    print(f"→ Report  in: {OUT}/train_baseline_report.txt")
    return 0


if __name__ == "__main__":
    sys.exit(main())
