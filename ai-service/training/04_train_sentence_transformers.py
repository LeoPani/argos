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

Vantagens vs baseline TF-IDF:
  - Captura semântica (sinônimos, paráfrases)
  - Funciona melhor em datasets pequenos
  - Resultado é vetor 384d reutilizável (busca semântica de prior art interno)

Requisitos: pip install sentence-transformers scikit-learn
Tempo: ~2-5 min em CPU pra ~260 trabalhos.
"""

import json
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
    import sys
    sys.exit(f"pip install sentence-transformers scikit-learn  ({e})")

DATA   = Path(__file__).parent / "data" / "annotations.jsonl"
MODELS = Path(__file__).parent / "models"

MODEL_NAME = "paraphrase-multilingual-MiniLM-L12-v2"  # 50+ idiomas, suporta PT-BR

LETTERS = ["A","B","C","D","E","F","G","H"]

def main():
    if not DATA.exists():
        print(f"Falta {DATA}")
        return

    rows = [json.loads(l) for l in DATA.read_text().splitlines() if l]
    print(f"Loaded {len(rows)} annotations")

    print(f"\nCarregando modelo {MODEL_NAME}...")
    model = SentenceTransformer(MODEL_NAME)
    print(f"Modelo carregado. Dimensão: {model.get_sentence_embedding_dimension()}")

    # Concatena título + abstract pra embedding
    texts = [f"{r.get('title','')}. {r.get('rationale','')}" for r in rows]
    print(f"\nGerando embeddings de {len(texts)} textos...")
    X = model.encode(texts, show_progress_bar=True, convert_to_numpy=True)
    print(f"Embedding matrix: {X.shape}")

    # ── Target 1: patenteabilidade ──
    y_pat = np.array([1 if r.get("is_patentable") else 0 for r in rows])

    print("\n━━━ MODELO 1 (Sentence-BERT + LogReg): PATENTEABILIDADE ━━━")
    if len(set(y_pat)) < 2:
        print(f"  ⚠ Apenas 1 classe — pulando")
    else:
        X_tr, X_te, y_tr, y_te = train_test_split(
            X, y_pat, test_size=0.2, random_state=42, stratify=y_pat)

        clf = LogisticRegression(class_weight="balanced", max_iter=1000)
        clf.fit(X_tr, y_tr)
        pred = clf.predict(X_te)

        print(classification_report(y_te, pred,
              target_names=["não-patenteável", "patenteável"], digits=3))
        cv = cross_val_score(clf, X, y_pat, cv=5, scoring="f1")
        print(f"  5-fold CV F1: {cv.mean():.3f} ± {cv.std():.3f}")

        MODELS.mkdir(parents=True, exist_ok=True)
        joblib.dump(clf, MODELS / "sbert_logreg_patentability.pkl")
        # Salva também os embeddings (úteis pra busca semântica)
        np.save(MODELS / "embeddings.npy", X)
        ids = np.array([r["opportunity_id"] for r in rows])
        np.save(MODELS / "embeddings_ids.npy", ids)

    # ── Target 2: IPC ──
    y_ipc = np.array([
        r.get("ipc_category") if r.get("ipc_category") is not None else -1
        for r in rows
    ])

    print("\n━━━ MODELO 2 (Sentence-BERT + RandomForest): IPC ━━━")
    mask = y_ipc >= 0
    X_clean = X[mask]
    y_clean = y_ipc[mask]
    if len(set(y_clean)) < 2:
        print(f"  ⚠ Apenas 1 classe IPC — pulando")
    else:
        X_tr, X_te, y_tr, y_te = train_test_split(
            X_clean, y_clean, test_size=0.2, random_state=42)

        clf_ipc = RandomForestClassifier(
            n_estimators=400, max_depth=30,
            class_weight="balanced", random_state=42, n_jobs=-1)
        clf_ipc.fit(X_tr, y_tr)
        pred = clf_ipc.predict(X_te)

        labels = sorted(set(y_te) | set(pred))
        names  = [LETTERS[i] for i in labels]
        print(classification_report(y_te, pred, target_names=names, digits=3,
              labels=labels, zero_division=0))

        cm = confusion_matrix(y_te, pred, labels=labels)
        print("\nConfusion matrix:")
        print("      " + " ".join(f"{n:>4}" for n in names))
        for i, n in enumerate(names):
            print(f"  {n:>3} " + " ".join(f"{cm[i][j]:>4}" for j in range(len(labels))))

        joblib.dump(clf_ipc, MODELS / "sbert_rf_ipc.pkl")

    # Salva metadata
    meta = {
        "model": MODEL_NAME,
        "embedding_dim": int(X.shape[1]),
        "n_examples": int(X.shape[0]),
        "ref": "Reimers & Gurevych (2019) EMNLP",
    }
    (MODELS / "metadata.json").write_text(json.dumps(meta, indent=2))
    print(f"\nModels saved to {MODELS}")

if __name__ == "__main__":
    main()
