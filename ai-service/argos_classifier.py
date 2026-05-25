"""
Argos — FastAPI que serve os modelos treinados (Fase 5).

Endpoints:
  POST /classify           — usa modelo Sentence-BERT treinado
  POST /classify-baseline  — usa baseline TF-IDF
  GET  /health             — status
  GET  /model-info         — metadata do modelo

Compatível com a interface antiga (api_argos.py) pra Go consumir
sem mudanças.

Rodar:
    cd ai-service
    source ~/argos-ai/bin/activate  (ou venv própria)
    uvicorn argos_classifier:app --host 0.0.0.0 --port 8000
"""

import json
from pathlib import Path
from typing import Optional

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

try:
    import joblib
    import numpy as np
    from sentence_transformers import SentenceTransformer
except ImportError as e:
    raise SystemExit(f"pip install -r requirements.txt  ({e})")

MODELS_DIR = Path(__file__).parent / "training" / "models"

app = FastAPI(
    title="Argos Classifier",
    description="ML-based IPC classification + patentability prediction",
    version="2.0",
)

# ─── State ──────────────────────────────────────────────────────────────────

_sbert_model = None
_clf_pat = None     # patentability classifier
_clf_ipc = None     # IPC classifier
_tfidf = None
_rf_pat = None
_rf_ipc = None
_metadata = {}

def load_models():
    """Lazy load — não carrega tudo no startup."""
    global _sbert_model, _clf_pat, _clf_ipc, _tfidf, _rf_pat, _rf_ipc, _metadata

    meta_file = MODELS_DIR / "metadata.json"
    if meta_file.exists():
        _metadata = json.loads(meta_file.read_text())

    sbert_path  = MODELS_DIR / "sbert_logreg_patentability.pkl"
    sbert_ipc   = MODELS_DIR / "sbert_rf_ipc.pkl"
    tfidf_path  = MODELS_DIR / "tfidf_vectorizer.pkl"
    rf_pat_path = MODELS_DIR / "rf_patentability.pkl"
    rf_ipc_path = MODELS_DIR / "rf_ipc_classifier.pkl"

    if sbert_path.exists():
        _clf_pat = joblib.load(sbert_path)
    if sbert_ipc.exists():
        _clf_ipc = joblib.load(sbert_ipc)
    if tfidf_path.exists():
        _tfidf = joblib.load(tfidf_path)
    if rf_pat_path.exists():
        _rf_pat = joblib.load(rf_pat_path)
    if rf_ipc_path.exists():
        _rf_ipc = joblib.load(rf_ipc_path)

    if _clf_pat is not None or _clf_ipc is not None:
        # SBERT só se algum modelo SBERT estiver carregado
        model_name = _metadata.get("model", "paraphrase-multilingual-MiniLM-L12-v2")
        print(f"Loading {model_name}...")
        _sbert_model = SentenceTransformer(model_name)
        print("SBERT loaded.")

load_models()

# ─── Schemas ────────────────────────────────────────────────────────────────

class ClassifyRequest(BaseModel):
    text: str

class ClassifyResponse(BaseModel):
    text_received: str
    predicted_category_id: int          # 0..7 (IPC A..H)
    patentable: Optional[bool] = None
    patentable_confidence: Optional[float] = None
    ipc_confidence: Optional[float] = None
    method: str

# ─── Endpoints ──────────────────────────────────────────────────────────────

@app.get("/health")
def health():
    return {
        "status": "ok",
        "has_sbert": _sbert_model is not None,
        "has_patentability": _clf_pat is not None,
        "has_ipc": _clf_ipc is not None,
        "has_tfidf_baseline": _tfidf is not None,
        "metadata": _metadata,
    }

@app.get("/model-info")
def model_info():
    return _metadata or {"warning": "no model loaded yet — run training scripts"}

@app.post("/classify", response_model=ClassifyResponse)
def classify(req: ClassifyRequest):
    """Classifica via Sentence-BERT (preferido) ou baseline TF-IDF."""
    if not req.text.strip():
        raise HTTPException(400, "Empty text")

    # Tenta SBERT primeiro
    if _sbert_model is not None and _clf_ipc is not None:
        emb = _sbert_model.encode([req.text], convert_to_numpy=True)
        ipc_id = int(_clf_ipc.predict(emb)[0])
        ipc_conf = float(np.max(_clf_ipc.predict_proba(emb)))

        pat, pat_conf = None, None
        if _clf_pat is not None:
            pat = bool(_clf_pat.predict(emb)[0])
            pat_conf = float(np.max(_clf_pat.predict_proba(emb)))

        return ClassifyResponse(
            text_received=req.text[:200],
            predicted_category_id=ipc_id,
            patentable=pat,
            patentable_confidence=pat_conf,
            ipc_confidence=ipc_conf,
            method="sentence_bert_v1",
        )

    # Fallback TF-IDF
    if _tfidf is not None and _rf_ipc is not None:
        X = _tfidf.transform([req.text])
        ipc_id = int(_rf_ipc.predict(X)[0])
        ipc_conf = float(np.max(_rf_ipc.predict_proba(X)))
        pat, pat_conf = None, None
        if _rf_pat is not None:
            pat = bool(_rf_pat.predict(X)[0])
            pat_conf = float(np.max(_rf_pat.predict_proba(X)))
        return ClassifyResponse(
            text_received=req.text[:200],
            predicted_category_id=ipc_id,
            patentable=pat,
            patentable_confidence=pat_conf,
            ipc_confidence=ipc_conf,
            method="tfidf_random_forest_v1",
        )

    raise HTTPException(503, "No model trained yet. Run training scripts first.")

@app.post("/classify-baseline", response_model=ClassifyResponse)
def classify_baseline(req: ClassifyRequest):
    """Força TF-IDF (pra comparar com SBERT)."""
    if _tfidf is None or _rf_ipc is None:
        raise HTTPException(503, "Baseline not trained")

    X = _tfidf.transform([req.text])
    ipc_id = int(_rf_ipc.predict(X)[0])
    ipc_conf = float(np.max(_rf_ipc.predict_proba(X)))
    pat, pat_conf = None, None
    if _rf_pat is not None:
        pat = bool(_rf_pat.predict(X)[0])
        pat_conf = float(np.max(_rf_pat.predict_proba(X)))
    return ClassifyResponse(
        text_received=req.text[:200],
        predicted_category_id=ipc_id,
        patentable=pat,
        patentable_confidence=pat_conf,
        ipc_confidence=ipc_conf,
        method="tfidf_random_forest_v1",
    )
