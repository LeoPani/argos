"""
Argos — FastAPI que serve os modelos treinados (Fase 5).

Carrega os 2 modelos pré-treinados (Fase 3 TF-IDF + Fase 4 SBERT) e expõe
classificação em runtime para o backend Go consumir.

Endpoints:
  GET  /health         — status detalhado + honestidade sobre ground truth
  GET  /model-info     — metadata bruta dos modelos
  POST /classify       — modelo preferido (SBERT se disponível, senão TF-IDF)
                          Compatível com cliente Go: {"text": "..."} → {"predicted_category_id"}
  POST /classify-baseline — força TF-IDF
  POST /classify-sbert    — força SBERT
  POST /compare        — roda os DOIS e devolve diff (útil pra inspeção)

Schema do request:
  Mínimo: {"text": "..."}
  Completo: {"text": "...", "title": "...", "abstract": "...", "department": "..."}
  Se title+abstract vierem, ignora "text" e usa concat (mesmo formato do training).

Rodar:
    cd ai-service
    source ~/argos-ai/bin/activate
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

IPC_LETTERS = ["A", "B", "C", "D", "E", "F", "G", "H"]
IPC_NAMES = [
    "Necessidades humanas",
    "Operações/transportes",
    "Química/metalurgia",
    "Têxteis/papel",
    "Construção",
    "Mec. industrial",
    "Física/TI",
    "Eletricidade",
]

app = FastAPI(
    title="Argos Classifier",
    description="Sirva os modelos treinados (TF-IDF + Sentence-BERT) para o backend Go.",
    version="3.0",
)

# ─── State ──────────────────────────────────────────────────────────────────

_sbert_model = None
_clf_sbert_pat = None
_clf_sbert_ipc = None
_tfidf_vec = None
_clf_rf_pat = None
_clf_rf_ipc = None
_meta_tfidf: dict = {}
_meta_sbert: dict = {}


def load_models():
    global _sbert_model, _clf_sbert_pat, _clf_sbert_ipc
    global _tfidf_vec, _clf_rf_pat, _clf_rf_ipc
    global _meta_tfidf, _meta_sbert

    # Fase 3 metadata
    m_tfidf = MODELS_DIR / "metadata.json"
    if m_tfidf.exists():
        try:
            _meta_tfidf = json.loads(m_tfidf.read_text())
        except Exception:
            _meta_tfidf = {}

    # Fase 4 metadata (separado pra não sobrescrever)
    m_sbert = MODELS_DIR / "sbert_metadata.json"
    if m_sbert.exists():
        try:
            _meta_sbert = json.loads(m_sbert.read_text())
        except Exception:
            _meta_sbert = {}

    # Pickles
    paths = {
        "_clf_sbert_pat": MODELS_DIR / "sbert_logreg_patentability.pkl",
        "_clf_sbert_ipc": MODELS_DIR / "sbert_rf_ipc.pkl",
        "_tfidf_vec":     MODELS_DIR / "tfidf_vectorizer.pkl",
        "_clf_rf_pat":    MODELS_DIR / "rf_patentability.pkl",
        "_clf_rf_ipc":    MODELS_DIR / "rf_ipc_classifier.pkl",
    }
    g = globals()
    for var, p in paths.items():
        if p.exists():
            try:
                g[var] = joblib.load(p)
                print(f"  ✓ loaded {p.name}")
            except Exception as e:
                print(f"  ⚠ failed {p.name}: {e}")

    # SBERT encoder (compartilhado entre os 2 modelos SBERT)
    if _clf_sbert_pat is not None or _clf_sbert_ipc is not None:
        encoder_name = _meta_sbert.get("encoder", "paraphrase-multilingual-MiniLM-L12-v2")
        print(f"  Loading SBERT encoder: {encoder_name}…")
        g["_sbert_model"] = SentenceTransformer(encoder_name)
        print(f"  ✓ encoder ready · dim={g['_sbert_model'].get_sentence_embedding_dimension()}")


print("Loading Argos classifier models from", MODELS_DIR)
load_models()


# ─── Schemas ────────────────────────────────────────────────────────────────

class ClassifyRequest(BaseModel):
    text: Optional[str] = None
    title: Optional[str] = None
    abstract: Optional[str] = None
    department: Optional[str] = None


class ClassifyResponse(BaseModel):
    text_received: str
    predicted_category_id: int          # 0..7 (compatibilidade com cliente Go)
    ipc_letter: str
    ipc_name: str
    patentable: Optional[bool] = None
    patentable_confidence: Optional[float] = None
    ipc_confidence: Optional[float] = None
    method: str
    rationale: str


class CompareResponse(BaseModel):
    text_received: str
    sbert: Optional[ClassifyResponse] = None
    tfidf: Optional[ClassifyResponse] = None
    agreement: dict   # ipc_agree, patentable_agree


# ─── Helpers ────────────────────────────────────────────────────────────────

def build_text(req: ClassifyRequest) -> str:
    """Espelha o build_texts() do training pra não ter drift de feature."""
    if req.title or req.abstract or req.department:
        title    = (req.title or "").strip()
        abstract = (req.abstract or "")[:1500].strip()
        dept     = (req.department or "").strip()
        return f"{title}. {abstract}. DEP: {dept}"
    return (req.text or "").strip()


def predict_sbert(text: str) -> Optional[ClassifyResponse]:
    if _sbert_model is None or _clf_sbert_ipc is None:
        return None
    emb = _sbert_model.encode([text], convert_to_numpy=True)
    ipc_id = int(_clf_sbert_ipc.predict(emb)[0])
    ipc_conf = float(np.max(_clf_sbert_ipc.predict_proba(emb)))
    pat, pat_conf = None, None
    if _clf_sbert_pat is not None:
        pat = bool(_clf_sbert_pat.predict(emb)[0])
        pat_conf = float(np.max(_clf_sbert_pat.predict_proba(emb)))
    return ClassifyResponse(
        text_received=text[:200],
        predicted_category_id=ipc_id,
        ipc_letter=safe_letter(ipc_id),
        ipc_name=safe_name(ipc_id),
        patentable=pat,
        patentable_confidence=pat_conf,
        ipc_confidence=ipc_conf,
        method="sbert_logreg_v1",
        rationale=build_rationale("sbert", ipc_id, pat, pat_conf, ipc_conf),
    )


def predict_tfidf(text: str) -> Optional[ClassifyResponse]:
    if _tfidf_vec is None or _clf_rf_ipc is None:
        return None
    X = _tfidf_vec.transform([text])
    ipc_id = int(_clf_rf_ipc.predict(X)[0])
    ipc_conf = float(np.max(_clf_rf_ipc.predict_proba(X)))
    pat, pat_conf = None, None
    if _clf_rf_pat is not None:
        pat = bool(_clf_rf_pat.predict(X)[0])
        pat_conf = float(np.max(_clf_rf_pat.predict_proba(X)))
    return ClassifyResponse(
        text_received=text[:200],
        predicted_category_id=ipc_id,
        ipc_letter=safe_letter(ipc_id),
        ipc_name=safe_name(ipc_id),
        patentable=pat,
        patentable_confidence=pat_conf,
        ipc_confidence=ipc_conf,
        method="tfidf_random_forest_v1",
        rationale=build_rationale("tfidf", ipc_id, pat, pat_conf, ipc_conf),
    )


def safe_letter(i: int) -> str:
    return IPC_LETTERS[i] if 0 <= i < 8 else "?"


def safe_name(i: int) -> str:
    return IPC_NAMES[i] if 0 <= i < 8 else "Unknown"


def build_rationale(method: str, ipc_id: int, pat: Optional[bool],
                    pat_conf: Optional[float], ipc_conf: float) -> str:
    parts = []
    parts.append(f"Classificador {method}")
    parts.append(f"IPC: {safe_letter(ipc_id)} ({safe_name(ipc_id)}) — confiança {ipc_conf:.0%}")
    if pat is not None and pat_conf is not None:
        parts.append(
            f"Patenteabilidade: {'sim' if pat else 'NÃO'} (confiança {pat_conf:.0%})"
        )
    return " · ".join(parts)


def is_trained_on_heuristic() -> bool:
    if _meta_sbert.get("using_heuristic_groundtruth"):
        return True
    if _meta_tfidf.get("using_heuristic_groundtruth"):
        return True
    return False


# ─── Endpoints ──────────────────────────────────────────────────────────────

@app.get("/health")
def health():
    """Status detalhado. O backend Go consulta isso pra exibir o badge correto."""
    return {
        "status": "ok",
        "has_sbert":          _sbert_model is not None,
        "has_sbert_pat":      _clf_sbert_pat is not None,
        "has_sbert_ipc":      _clf_sbert_ipc is not None,
        "has_tfidf_baseline": _tfidf_vec is not None and _clf_rf_ipc is not None,
        "has_rf_pat":         _clf_rf_pat is not None,
        "trained_on_heuristic": is_trained_on_heuristic(),
        "warning": (
            "Modelos treinados contra heurística v2 — para defesa acadêmica "
            "completa, rode Fase 1 com LLM e re-treine Fases 3/4."
            if is_trained_on_heuristic() else None
        ),
        "metadata_tfidf": _meta_tfidf,
        "metadata_sbert": _meta_sbert,
    }


@app.get("/model-info")
def model_info():
    return {
        "tfidf": _meta_tfidf or {"warning": "no Fase 3 metadata"},
        "sbert": _meta_sbert or {"warning": "no Fase 4 metadata"},
    }


@app.post("/classify", response_model=ClassifyResponse)
def classify(req: ClassifyRequest):
    """Modelo preferido — SBERT > TF-IDF como fallback."""
    text = build_text(req)
    if not text:
        raise HTTPException(400, "Empty text/title")

    out = predict_sbert(text) or predict_tfidf(text)
    if out is None:
        raise HTTPException(503, "No model trained yet. Run training/ scripts.")
    return out


@app.post("/classify-baseline", response_model=ClassifyResponse)
def classify_baseline(req: ClassifyRequest):
    """Força TF-IDF (Fase 3)."""
    text = build_text(req)
    if not text:
        raise HTTPException(400, "Empty text/title")
    out = predict_tfidf(text)
    if out is None:
        raise HTTPException(503, "TF-IDF baseline not trained")
    return out


@app.post("/classify-sbert", response_model=ClassifyResponse)
def classify_sbert(req: ClassifyRequest):
    """Força SBERT (Fase 4)."""
    text = build_text(req)
    if not text:
        raise HTTPException(400, "Empty text/title")
    out = predict_sbert(text)
    if out is None:
        raise HTTPException(503, "SBERT model not trained")
    return out


@app.post("/compare", response_model=CompareResponse)
def compare(req: ClassifyRequest):
    """Roda os 2 modelos e mostra agreement. Útil pra inspeção / dashboard."""
    text = build_text(req)
    if not text:
        raise HTTPException(400, "Empty text/title")

    sbert = predict_sbert(text)
    tfidf = predict_tfidf(text)
    agreement = {
        "ipc_agree": (sbert and tfidf and
                      sbert.predicted_category_id == tfidf.predicted_category_id),
        "patentable_agree": (sbert and tfidf and
                             sbert.patentable == tfidf.patentable),
    }
    return CompareResponse(
        text_received=text[:200],
        sbert=sbert,
        tfidf=tfidf,
        agreement=agreement,
    )
