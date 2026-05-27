"""
Deploy Argos IP Classifier para Hugging Face.

Cria dois repos:
  1. LeoPani/argos-ip-classifier  (Model — pkl files)
  2. LeoPani/argos-ip-space       (Space — Gradio app)

Uso:
    # 1. Login (uma vez)
    huggingface-cli login

    # 2. Deploy
    cd hf-space
    python deploy.py

    # Ou em modo dry-run (só verifica arquivos, não faz upload):
    python deploy.py --dry-run

    # Forçar recriação mesmo se repo existir:
    python deploy.py --force
"""

import argparse
import os
import sys
from pathlib import Path

try:
    from huggingface_hub import (
        HfApi,
        create_repo,
        upload_file,
        upload_folder,
        whoami,
    )
except ImportError:
    print("❌ huggingface_hub não instalado. Execute: pip install huggingface_hub")
    sys.exit(1)


# ─── Config ───────────────────────────────────────────────────────────────────

SPACE_DIR   = Path(__file__).parent
MODELS_DIR  = SPACE_DIR.parent / "ai-service" / "training" / "models"

MODEL_FILES = [
    "tfidf_vectorizer.pkl",
    "rf_ipc_classifier.pkl",
    "rf_patentability.pkl",
]

SPACE_FILES = [
    "app.py",
    "requirements.txt",
    "README.md",
]


# ─── Helpers ──────────────────────────────────────────────────────────────────

def check_login(api: HfApi) -> str:
    """Return username or exit if not authenticated."""
    try:
        user = whoami()
        username = user["name"]
        print(f"✅ Autenticado como: {username}")
        return username
    except Exception:
        print("❌ Não autenticado. Execute: huggingface-cli login")
        sys.exit(1)


def check_models(models_dir: Path) -> list[Path]:
    """Verify all model files exist."""
    missing = []
    found   = []
    for fname in MODEL_FILES:
        path = models_dir / fname
        if path.exists():
            size_mb = path.stat().st_size / 1e6
            print(f"   ✓ {fname:<35} ({size_mb:.1f} MB)")
            found.append(path)
        else:
            print(f"   ✗ {fname} — NÃO ENCONTRADO")
            missing.append(fname)
    if missing:
        print(f"\n⚠  Modelos ausentes: {missing}")
        print("   Execute: cd ai-service && python training/03_train_baseline.py")
        sys.exit(1)
    return found


def create_or_get_repo(api: HfApi, repo_id: str, repo_type: str, force: bool) -> str:
    """Create repo if it doesn't exist, return URL."""
    try:
        url = create_repo(repo_id=repo_id, repo_type=repo_type, exist_ok=True)
        print(f"   {'Criado' if not force else 'Atualizado'}: {url}")
        return str(url)
    except Exception as e:
        print(f"   ⚠ {e}")
        return f"https://huggingface.co/{repo_id}"


# ─── Main deploy ──────────────────────────────────────────────────────────────

def deploy(dry_run: bool = False, force: bool = False):
    api      = HfApi()
    username = check_login(api)

    model_repo_id = f"{username}/argos-ip-classifier"
    space_repo_id = f"{username}/argos-ip-space"

    print(f"\n📦 Model repo: {model_repo_id}")
    print(f"🚀 Space repo: {space_repo_id}")

    # ── 1. Check model files ──
    print(f"\n🔍 Verificando modelos em: {MODELS_DIR}")
    model_paths = check_models(MODELS_DIR)

    if dry_run:
        print("\n✅ Dry-run completo — todos os arquivos encontrados.")
        print(f"\nPara fazer o deploy real, execute:\n  python deploy.py")
        return

    # ── 2. Create repos ──
    print("\n🏗  Criando repositórios…")
    create_or_get_repo(api, model_repo_id, "model", force)
    # Update app.py MODEL_REPO env before deploying Space
    app_content = (SPACE_DIR / "app.py").read_text()
    app_content = app_content.replace(
        '"LeoPani/argos-ip-classifier"',
        f'"{model_repo_id}"'
    )
    (SPACE_DIR / "app.py").write_text(app_content)

    create_or_get_repo(api, space_repo_id, "space", force)

    # ── 3. Upload models ──
    print(f"\n⬆  Enviando modelos para {model_repo_id}…")
    for path in model_paths:
        size_mb = path.stat().st_size / 1e6
        print(f"   Enviando {path.name} ({size_mb:.1f} MB)…", end="", flush=True)
        try:
            api.upload_file(
                path_or_fileobj=str(path),
                path_in_repo=path.name,
                repo_id=model_repo_id,
                repo_type="model",
            )
            print(" ✓")
        except Exception as e:
            print(f" ✗ {e}")

    # Model card
    model_card = f"""---
language:
- pt
license: other
tags:
- patents
- classification
- portuguese
- scikit-learn
---

# Argos IP Classifier — Models

TF-IDF + Random Forest models for patent classification (NIT-UFOP).

- `tfidf_vectorizer.pkl` — TF-IDF vectorizer (pt-BR corpus)
- `rf_ipc_classifier.pkl` — IPC category classifier (8 classes, F1 ~0.98)
- `rf_patentability.pkl` — Patentability binary classifier (F1 ~0.81)

Trained on 770 UFOP thesis/dissertation samples annotated via Groq llama-3.3-70b.
Cohen's κ = 0.286 vs heuristic baseline (Landis & Koch "fair").
"""
    api.upload_file(
        path_or_fileobj=model_card.encode(),
        path_in_repo="README.md",
        repo_id=model_repo_id,
        repo_type="model",
    )
    print("   Uploaded README.md ✓")

    # ── 4. Upload Space files ──
    print(f"\n⬆  Enviando Space para {space_repo_id}…")
    for fname in SPACE_FILES:
        fpath = SPACE_DIR / fname
        if not fpath.exists():
            print(f"   ✗ {fname} não encontrado, pulando")
            continue
        print(f"   Enviando {fname}…", end="", flush=True)
        try:
            api.upload_file(
                path_or_fileobj=str(fpath),
                path_in_repo=fname,
                repo_id=space_repo_id,
                repo_type="space",
            )
            print(" ✓")
        except Exception as e:
            print(f" ✗ {e}")

    # ── 5. Summary ──
    print(f"""
╔══════════════════════════════════════════════════════════╗
║  Deploy concluído! 🎉                                   ║
╠══════════════════════════════════════════════════════════╣
║  Modelos: https://huggingface.co/{model_repo_id:<24}║
║  Space:   https://huggingface.co/spaces/{space_repo_id:<19}║
╚══════════════════════════════════════════════════════════╝

O Space pode levar 2-3 minutos para construir.
""")


# ─── Entry point ──────────────────────────────────────────────────────────────

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--dry-run", action="store_true", help="Verificar arquivos sem upload")
    parser.add_argument("--force",   action="store_true", help="Recriar repos mesmo se existirem")
    args = parser.parse_args()
    deploy(dry_run=args.dry_run, force=args.force)
