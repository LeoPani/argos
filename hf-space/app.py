"""
Argos IP Classifier — Gradio Space
Classifica patenteabilidade e categoria IPC de invenções brasileiras.

Modelos baixados automaticamente do HF Hub na primeira execução.
"""

import os
import re
import math
import string
from pathlib import Path
from functools import lru_cache
from typing import Optional

import gradio as gr
import numpy as np
import joblib
from huggingface_hub import hf_hub_download

# ─── Config ───────────────────────────────────────────────────────────────────

HF_REPO = os.environ.get("MODEL_REPO", "LeoPani/argos-ip-classifier")

IPC_LETTERS = list("ABCDEFGH")
IPC_NAMES = [
    "A — Necessidades Humanas",
    "B — Operações e Transportes",
    "C — Química e Metalurgia",
    "D — Têxteis e Papel",
    "E — Construção Civil",
    "F — Engenharia Mecânica",
    "G — Física / Tecnologia da Informação",
    "H — Eletricidade",
]

ART10_EXCLUSIONS = [
    (["software", "programa de computador", "aplicativo"], "Art. 10, V: programas de computador per se são excluídos. Descreva o efeito técnico do hardware."),
    (["método de negócio", "modelo de negócio", "método comercial"], "Art. 10, III: métodos comerciais são excluídos. Foque no processo técnico."),
    (["método matemático", "algoritmo puro", "cálculo matemático"], "Art. 10, I: métodos matemáticos puros são excluídos. Patentear apenas a aplicação técnica."),
    (["método de ensino", "método pedagógico", "regras de jogo"], "Art. 10, IV: regras e métodos pedagógicos são excluídos."),
]

# ─── Model loading ────────────────────────────────────────────────────────────

@lru_cache(maxsize=1)
def load_models():
    """Download and load TF-IDF + RF models from HF Hub."""
    models = {}
    files = {
        "vectorizer":     "tfidf_vectorizer.pkl",
        "ipc_clf":        "rf_ipc_classifier.pkl",
        "patent_clf":     "rf_patentability.pkl",
    }
    for key, filename in files.items():
        try:
            path = hf_hub_download(repo_id=HF_REPO, filename=filename, repo_type="model")
            models[key] = joblib.load(path)
        except Exception as e:
            print(f"⚠ Could not load {filename}: {e}")
            models[key] = None
    return models


# ─── Art. 10 LPI detection ────────────────────────────────────────────────────

def detect_art10(text: str) -> list[str]:
    text_lower = text.lower()
    alerts = []
    for keywords, desc in ART10_EXCLUSIONS:
        if any(kw in text_lower for kw in keywords):
            alerts.append(desc)
    return alerts


# ─── Prediction ───────────────────────────────────────────────────────────────

def classify(title: str, abstract: str) -> tuple[str, str, str, str]:
    """
    Returns (patenteability_html, ipc_html, art10_html, methodology_html)
    """
    title    = (title    or "").strip()
    abstract = (abstract or "").strip()

    if not title:
        return (
            "<p style='color:#f87171'>⚠ Informe o título da invenção.</p>",
            "—", "—", "—"
        )

    text = (title + ". " + abstract).strip()

    models = load_models()
    vec    = models.get("vectorizer")
    ipc_clf   = models.get("ipc_clf")
    pat_clf   = models.get("patent_clf")

    ipc_result  = "Modelo não disponível"
    pat_result  = "Modelo não disponível"
    pat_pct     = None
    ipc_cat     = None

    if vec is not None:
        X = vec.transform([text])

        if ipc_clf is not None:
            ipc_cat = int(ipc_clf.predict(X)[0])
            ipc_name = IPC_NAMES[ipc_cat] if 0 <= ipc_cat < 8 else f"Categoria {ipc_cat}"
            if hasattr(ipc_clf, "predict_proba"):
                prob = float(ipc_clf.predict_proba(X)[0][ipc_cat])
                ipc_result = f"{ipc_name}  ({prob*100:.0f}% confiança)"
            else:
                ipc_result = ipc_name

        if pat_clf is not None:
            pred = int(pat_clf.predict(X)[0])
            if hasattr(pat_clf, "predict_proba"):
                proba = pat_clf.predict_proba(X)[0]
                pat_pct = float(proba[1]) * 100 if len(proba) > 1 else float(proba[0]) * 100
            else:
                pat_pct = 80.0 if pred == 1 else 20.0
            pat_result = "Patenteável" if pred == 1 else "Baixo potencial"

    # Art. 10 check
    art10 = detect_art10(text)

    # ── Build HTML outputs ──────────────────────────────────────────────────

    # Patenteability card
    if pat_pct is not None:
        color    = "#34d399" if pat_pct >= 60 else ("#f59e0b" if pat_pct >= 35 else "#f87171")
        emoji    = "✅" if pat_pct >= 60 else ("⚠️" if pat_pct >= 35 else "❌")
        pat_html = f"""
<div style="padding:16px;border-radius:10px;border:1px solid {color}40;background:{color}10">
  <p style="font-size:1.4rem;font-weight:700;color:{color};margin:0">{emoji} {pat_pct:.0f}%</p>
  <p style="color:#e2e8f0;margin:4px 0 0">{pat_result}</p>
  <div style="margin-top:8px;height:6px;border-radius:3px;background:#1e293b">
    <div style="width:{pat_pct:.0f}%;height:100%;border-radius:3px;background:{color};transition:width 0.4s"></div>
  </div>
</div>"""
    else:
        pat_html = f"<p>{pat_result}</p>"

    # IPC card
    if ipc_cat is not None and 0 <= ipc_cat < 8:
        ipc_html = f"""
<div style="padding:12px;border-radius:10px;background:#6366f110;border:1px solid #6366f140">
  <p style="font-size:2rem;font-weight:800;color:#818cf8;margin:0">{IPC_LETTERS[ipc_cat]}</p>
  <p style="color:#e2e8f0;margin:4px 0 0">{IPC_NAMES[ipc_cat]}</p>
</div>"""
    else:
        ipc_html = f"<p>{ipc_result}</p>"

    # Art. 10 card
    if art10:
        art10_items = "".join(f"<li style='margin:4px 0'>{a}</li>" for a in art10)
        art10_html = f"""
<div style="padding:12px;border-radius:10px;background:#f59e0b10;border:1px solid #f59e0b40">
  <p style="color:#f59e0b;font-weight:600;margin:0 0 8px">⚠ Possíveis exclusões (Art. 10 LPI)</p>
  <ul style="color:#fcd34d;padding-left:16px;margin:0">{art10_items}</ul>
  <p style="color:#6b7280;font-size:0.8rem;margin:8px 0 0">Consulte um agente de PI antes do depósito.</p>
</div>"""
    else:
        art10_html = """
<div style="padding:12px;border-radius:10px;background:#34d39910;border:1px solid #34d39940">
  <p style="color:#34d399;margin:0">✓ Nenhuma exclusão do Art. 10 LPI detectada</p>
</div>"""

    # Methodology note
    method_html = """
<details>
<summary style="color:#6366f1;cursor:pointer;font-size:0.85rem">Metodologia</summary>
<div style="padding:8px 0;color:#64748b;font-size:0.8rem;line-height:1.6">
<strong>Patenteabilidade:</strong> TF-IDF + Random Forest (Breiman 2001) — F1 ~0.81 em CV estratificado.<br>
<strong>IPC:</strong> TF-IDF + Random Forest — F1 ~0.98 em 8 classes (treinado em corpus INPI+UFOP).<br>
<strong>Ground truth:</strong> 770 amostras UFOP anotadas por Groq llama-3.3-70b (LLM-as-annotator, Honovich et al. 2022).<br>
<strong>Validação:</strong> Cohen's κ = 0.286 (razoável, Landis & Koch 1977) vs heurística Go.<br>
<strong>Corpus:</strong> NIT-UFOP · 14.4k despachos INPI (RPIs 2884–2890) + portfolio UFOP.
</div>
</details>"""

    return pat_html, ipc_html, art10_html, method_html


# ─── Gradio interface ─────────────────────────────────────────────────────────

EXAMPLES = [
    [
        "Sistema de purificação de água por geração de ozônio via descarga elétrica",
        "A presente invenção descreve um sistema portátil de tratamento de água utilizando geração de ozônio por descarga elétrica de barreira dielétrica, com eficiência de eliminação de 99,7% de E. coli."
    ],
    [
        "Método de machine learning para predição de vida útil de rolamentos",
        "Propõe-se um método baseado em redes neurais LSTM para diagnóstico preditivo de falhas em rolamentos industriais através de análise de vibração e temperatura em tempo real."
    ],
    [
        "Software de gestão de contratos de propriedade intelectual",
        "Sistema de software para gerenciamento automatizado de contratos de licenciamento de patentes com módulo de notificação de vencimentos."
    ],
    [
        "Absorvedor dinâmico de vibrações passivo para fresamento CNC",
        "Desenvolvimento e caracterização de um absorvedor dinâmico de vibrações (DVA) passivo de dupla massa, sintonizado para a frequência natural dominante do processo de fresamento de acabamento em aço inoxidável."
    ],
]

with gr.Blocks(
    title="Argos IP Classifier",
    theme=gr.themes.Base(
        primary_hue="indigo",
        secondary_hue="purple",
    ),
    css="""
    .gradio-container { max-width: 900px !important; }
    footer { display: none !important; }
    .eye-header { text-align: center; padding: 24px 0 8px; }
    """,
) as demo:

    gr.HTML("""
    <div class="eye-header">
      <svg width="80" height="54" viewBox="0 0 120 80" fill="none" xmlns="http://www.w3.org/2000/svg">
        <defs>
          <radialGradient id="irisG" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stop-color="#6366f1" stop-opacity="0.9"/>
            <stop offset="100%" stop-color="#6366f1" stop-opacity="0.1"/>
          </radialGradient>
        </defs>
        <path d="M8 40 C20 15 40 6 60 6 C80 6 100 15 112 40 C100 65 80 74 60 74 C40 74 20 65 8 40Z"
          stroke="#6366f1" stroke-width="1.5" fill="#6366f108"/>
        <circle cx="60" cy="40" r="22" fill="url(#irisG)"/>
        <circle cx="60" cy="40" r="22" stroke="#6366f1" stroke-width="1" fill="none" opacity="0.8"/>
        <circle cx="60" cy="40" r="11" fill="#07070e"/>
        <circle cx="60" cy="40" r="3" fill="#6366f1"/>
        <circle cx="65" cy="34" r="3.5" fill="white" opacity="0.18"/>
      </svg>
      <h1 style="font-size:1.6rem;font-weight:800;letter-spacing:0.3em;color:#e2e8f0;margin:8px 0 4px">ARGOS</h1>
      <p style="color:#475569;font-size:0.75rem;letter-spacing:0.1em;text-transform:uppercase">
        IP Intelligence · NIT-UFOP · Classificador de Patentes
      </p>
    </div>
    """)

    with gr.Row():
        with gr.Column(scale=1):
            title_in = gr.Textbox(
                label="Título da Invenção",
                placeholder="Ex: Sistema de purificação por ozônio para efluentes industriais",
                lines=2,
            )
            abstract_in = gr.Textbox(
                label="Resumo / Abstract (opcional)",
                placeholder="Descreva a invenção em 3-5 frases técnicas...",
                lines=6,
            )
            btn = gr.Button("🔍 Analisar", variant="primary", size="lg")

        with gr.Column(scale=1):
            pat_out    = gr.HTML(label="Patenteabilidade")
            ipc_out    = gr.HTML(label="Categoria IPC")
            art10_out  = gr.HTML(label="Alertas Art. 10 LPI")
            method_out = gr.HTML()

    btn.click(
        fn=classify,
        inputs=[title_in, abstract_in],
        outputs=[pat_out, ipc_out, art10_out, method_out],
    )

    gr.Examples(
        examples=EXAMPLES,
        inputs=[title_in, abstract_in],
        label="Exemplos",
    )

    gr.HTML("""
    <div style="text-align:center;padding:16px;color:#1e293b;font-size:0.7rem;letter-spacing:0.08em">
      UFOP · NIT · Argos IP Intelligence · Uso acadêmico
    </div>
    """)

if __name__ == "__main__":
    demo.launch()
