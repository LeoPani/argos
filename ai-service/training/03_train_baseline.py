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
  - Ground truth: anotações Claude (proxy de specialist)
"""

import json
from pathlib import Path

try:
    import numpy as np
    from sklearn.ensemble import RandomForestClassifier
    from sklearn.feature_extraction.text import TfidfVectorizer
    from sklearn.metrics import classification_report, confusion_matrix
    from sklearn.model_selection import train_test_split, cross_val_score
    import joblib
except ImportError as e:
    import sys
    sys.exit(f"pip install scikit-learn joblib  ({e})")

DATA   = Path(__file__).parent / "data" / "annotations.jsonl"
MODELS = Path(__file__).parent / "models"
OUT    = Path(__file__).parent / "outputs"

LETTERS = ["A","B","C","D","E","F","G","H"]

def main():
    if not DATA.exists():
        print(f"Falta {DATA} — rode 01_annotate.py")
        return

    rows = [json.loads(l) for l in DATA.read_text().splitlines() if l]
    print(f"Loaded {len(rows)} annotations\n")

    # ── Features ────────────────────────────────────────────────────────
    texts = [f"{r.get('title','')}. {r.get('rationale','')}" for r in rows]

    # ── Target 1: patenteabilidade (binário) ────────────────────────────
    y_pat = np.array([1 if r.get("is_patentable") else 0 for r in rows])

    # ── Target 2: IPC (multiclass) ──────────────────────────────────────
    y_ipc = np.array([
        r.get("ipc_category") if r.get("ipc_category") is not None else -1
        for r in rows
    ])

    # Vectorize (mantém 3000 features máximas)
    vec = TfidfVectorizer(
        max_features=3000,
        ngram_range=(1, 2),
        min_df=2,
        stop_words=None,  # PT-BR — manter
    )
    X = vec.fit_transform(texts)
    print(f"Feature matrix: {X.shape}\n")

    # ━━━ MODELO 1: patenteabilidade ━━━
    print("━━━ Modelo 1: PATENTEABILIDADE (binário) ━━━")
    if len(set(y_pat)) < 2:
        print(f"  ⚠ Apenas 1 classe nas labels — pulando ({set(y_pat)})")
    else:
        X_tr, X_te, y_tr, y_te = train_test_split(
            X, y_pat, test_size=0.2, random_state=42, stratify=y_pat)

        clf = RandomForestClassifier(
            n_estimators=200, max_depth=20,
            class_weight="balanced", random_state=42, n_jobs=-1)
        clf.fit(X_tr, y_tr)
        pred = clf.predict(X_te)

        print(classification_report(y_te, pred,
              target_names=["não-patenteável", "patenteável"], digits=3))

        # CV
        cv = cross_val_score(clf, X, y_pat, cv=5, scoring="f1")
        print(f"  5-fold CV F1: {cv.mean():.3f} ± {cv.std():.3f}\n")

        # Top features (palavras + discriminativas)
        importances = clf.feature_importances_
        feature_names = vec.get_feature_names_out()
        top = np.argsort(importances)[-15:][::-1]
        print("Top features (TF-IDF * importance):")
        for i in top:
            print(f"  {importances[i]:.4f}  {feature_names[i]}")
        print()

        # Salvar
        MODELS.mkdir(parents=True, exist_ok=True)
        joblib.dump(clf, MODELS / "rf_patentability.pkl")
        joblib.dump(vec, MODELS / "tfidf_vectorizer.pkl")

    # ━━━ MODELO 2: IPC ━━━
    print("\n━━━ Modelo 2: IPC (multiclass A..H) ━━━")
    mask = y_ipc >= 0
    X_ipc = X[mask]
    y_ipc_clean = y_ipc[mask]
    if len(X_ipc.toarray()) < 20:
        print("  ⚠ Dataset pequeno demais")
        return
    if len(set(y_ipc_clean)) < 2:
        print(f"  ⚠ Apenas 1 classe IPC — pulando ({set(y_ipc_clean)})")
    else:
        X_tr, X_te, y_tr, y_te = train_test_split(
            X_ipc, y_ipc_clean, test_size=0.2, random_state=42)

        clf_ipc = RandomForestClassifier(
            n_estimators=300, max_depth=25,
            class_weight="balanced", random_state=42, n_jobs=-1)
        clf_ipc.fit(X_tr, y_tr)
        pred = clf_ipc.predict(X_te)

        labels = sorted(set(y_te) | set(pred))
        names  = [LETTERS[i] for i in labels]
        print(classification_report(y_te, pred, target_names=names, digits=3,
              labels=labels, zero_division=0))

        # Confusion matrix
        cm = confusion_matrix(y_te, pred, labels=labels)
        print("Confusion matrix (rows=true, cols=pred):")
        header = "      " + " ".join(f"{n:>4}" for n in names)
        print(header)
        for i, name in enumerate(names):
            row = " ".join(f"{cm[i][j]:>4}" for j in range(len(labels)))
            print(f"  {name:>3} {row}")

        joblib.dump(clf_ipc, MODELS / "rf_ipc_classifier.pkl")

    print("\nModels saved to", MODELS)

if __name__ == "__main__":
    main()
