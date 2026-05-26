"""
Argos — Fase 1: Anotação automática do dataset UFOP via Claude (LLM-as-annotator).

Para cada uma das oportunidades UFOP no banco, Claude classifica:
  - is_patentable: bool (tem potencial real de virar patente?)
  - ipc_category:  0-7 (A..H) ou null
  - confidence:    0.0-1.0
  - rationale:     justificativa textual curta

Validação metodológica:
  Honovich, O., Scialom, T., Levy, O., & Schick, T. (2022).
    "Unnatural instructions: Tuning language models with (almost) no human labor."
  Wang, Y., et al. (2022). "Self-Instruct: Aligning language models with
    self-generated instructions."
  Estudos mostram concordância >85% entre LLM-as-annotator e expert humano
  para tarefas de classificação textual em domínios técnicos.

Output: ai-service/training/data/annotations.jsonl (uma linha por trabalho)

Suporta dois providers (detectados automaticamente):
  - ANTHROPIC_API_KEY  → Claude Sonnet 4.6   (pago, ~US$1 pras 261)
  - GROQ_API_KEY       → Llama 3.3 70B       (free 14400 req/dia)

Uso:
    # Opção A — Groq (free):
    export GROQ_API_KEY=gsk_...
    pip install groq psycopg2-binary tqdm

    # Opção B — Claude (pago):
    export ANTHROPIC_API_KEY=sk-ant-...
    pip install anthropic psycopg2-binary tqdm

    cd ai-service
    python training/01_annotate.py             # all
    python training/01_annotate.py --limit 50  # primeiros 50 (teste)
    python training/01_annotate.py --resume    # continua de onde parou
"""

import argparse
import json
import os
import re
import sys
import time
from pathlib import Path

try:
    import psycopg2
    from tqdm import tqdm
except ImportError as e:
    sys.exit(f"Faltam dependências: pip install psycopg2-binary tqdm  ({e})")

# ─── Config ─────────────────────────────────────────────────────────────────

OUTPUT_PATH   = Path(__file__).parent / "data" / "annotations.jsonl"
DB_URL        = os.environ.get("DATABASE_URL",
    "postgres://argos:argos_dev@localhost:5432/argos")
ANTHROPIC_KEY = os.environ.get("ANTHROPIC_API_KEY")
GROQ_KEY      = os.environ.get("GROQ_API_KEY")
SLEEP_S       = 1.0   # respeitar rate limit (1s entre requests)
MAX_RETRIES   = 6     # tentativas por item em caso de rate-limit

# ─── Prompt template ────────────────────────────────────────────────────────

SYSTEM_PROMPT = """Você é um especialista em Propriedade Intelectual (PI) brasileiro
trabalhando com o NIT-UFOP (Núcleo de Inovação Tecnológica da Universidade
Federal de Ouro Preto). Sua tarefa é avaliar trabalhos acadêmicos quanto ao
potencial de gerarem patentes industriais defensáveis junto ao INPI sob a
Lei n. 9.279/1996.

Para cada trabalho, retorne APENAS um JSON válido (sem markdown, sem texto extra)
com:
  - "is_patentable": true/false — tem aspecto técnico patenteável (Art. 8 LPI)?
  - "ipc_category": 0-7 ou null, onde:
       0=A (Necessidades humanas - saúde, farmácia, alimentos)
       1=B (Operações/transportes - processos industriais)
       2=C (Química e metalurgia)
       3=D (Têxteis e papel)
       4=E (Construção civil)
       5=F (Engenharia mecânica)
       6=G (Física / TI / sensores)
       7=H (Eletricidade e eletrônica)
  - "confidence": 0.0 a 1.0 — sua confiança na classificação
  - "rationale": frase curta (max 200 chars) em PT-BR

Critérios para is_patentable (Art. 8 LPI):
  ✓ Novidade técnica (não é estado da arte)
  ✓ Atividade inventiva (não óbvio)
  ✓ Aplicação industrial (uso prático)

NÃO é patenteável (Art. 10 LPI):
  ✗ Descobertas, teorias científicas, métodos matemáticos
  ✗ Concepções puramente abstratas
  ✗ Esquemas jurídicos, comerciais, contábeis
  ✗ Apresentações de informação (textos puros)
  ✗ Programas de computador PER SE (mas pode ser parte de processo)
"""

USER_TEMPLATE = """Analise o trabalho UFOP:

TÍTULO: {title}

DEPARTAMENTO: {department}

RESUMO (abstract):
{abstract}

Responda APENAS o JSON conforme instruído."""

# ─── Pipeline ───────────────────────────────────────────────────────────────

def load_opportunities(limit: int | None, skip_ids: set[int]) -> list[dict]:
    """Carrega oportunidades do banco — só REAIS UFOP, ordenadas por score desc."""
    conn = psycopg2.connect(DB_URL)
    cur = conn.cursor()
    q = """
        SELECT id, title, COALESCE(abstract, ''), department, opportunity_level
        FROM ufop_opportunities
        WHERE external_id LIKE 'oai:repositorio.ufop.br%'
        ORDER BY pi_score DESC
    """
    if limit:
        q += f" LIMIT {limit}"
    cur.execute(q)
    rows = [
        {"id": r[0], "title": r[1], "abstract": r[2],
         "department": r[3], "level": r[4]}
        for r in cur.fetchall() if r[0] not in skip_ids
    ]
    cur.close(); conn.close()
    return rows

def build_user_prompt(opp: dict) -> str:
    return USER_TEMPLATE.format(
        title=opp["title"][:300],
        department=opp.get("department", "UFOP"),
        abstract=opp.get("abstract", "")[:3000],
    )

def clean_json_response(text: str) -> str:
    """Remove code fences que LLMs gostam de adicionar mesmo quando pedimos JSON puro."""
    text = text.strip()
    if text.startswith("```"):
        text = text.strip("`").split("\n", 1)[1].rsplit("```", 1)[0]
    if text.startswith("json"):
        text = text[4:].strip()
    return text

def annotate_with_anthropic(client, opp: dict, model: str) -> dict:
    msg = client.messages.create(
        model=model, max_tokens=512, system=SYSTEM_PROMPT,
        messages=[{"role": "user", "content": build_user_prompt(opp)}],
    )
    return _finalize(clean_json_response(msg.content[0].text), opp)

def _parse_retry_after(err_msg: str) -> float:
    """Extrai segundos do 'Please try again in Xm Ys.' da mensagem de rate-limit."""
    m = re.search(r"try again in\s+(?:(\d+)m)?(?:(\d+(?:\.\d+)?)s)?", str(err_msg), re.I)
    if not m:
        return 60.0
    minutes = float(m.group(1) or 0)
    seconds = float(m.group(2) or 0)
    return minutes * 60 + seconds + 2  # +2s de margem

def annotate_with_groq(client, opp: dict, model: str) -> dict:
    for attempt in range(MAX_RETRIES):
        try:
            msg = client.chat.completions.create(
                model=model, max_tokens=512, temperature=0.0,
                response_format={"type": "json_object"},
                messages=[
                    {"role": "system", "content": SYSTEM_PROMPT},
                    {"role": "user",   "content": build_user_prompt(opp)},
                ],
            )
            return _finalize(clean_json_response(msg.choices[0].message.content), opp)
        except Exception as e:
            err_str = str(e)
            if "429" in err_str or "rate_limit" in err_str.lower():
                wait = _parse_retry_after(err_str)
                if attempt < MAX_RETRIES - 1:
                    tqdm.write(f"  [rate-limit] aguardando {wait:.0f}s (tentativa {attempt+1}/{MAX_RETRIES})")
                    time.sleep(wait)
                    continue
            raise  # re-lança erros não-rate-limit ou última tentativa
    raise RuntimeError(f"Rate-limit persistiu após {MAX_RETRIES} tentativas")

def _finalize(raw_json: str, opp: dict) -> dict:
    parsed = json.loads(raw_json)
    parsed["opportunity_id"] = opp["id"]
    parsed["title"] = opp["title"]
    parsed["department"] = opp.get("department", "")
    parsed["heuristic_level"] = opp["level"]
    return parsed

def pick_provider(force: str | None) -> tuple[str, str, object, callable]:
    """Decide qual provider usar e retorna (name, model, client, annotate_fn)."""
    want = (force or "auto").lower()

    if want in ("auto", "groq") and GROQ_KEY:
        try:
            from groq import Groq
        except ImportError:
            sys.exit("GROQ_API_KEY setado mas groq não instalado: pip install groq")
        # llama-3.1-8b-instant: 500k TPD (vs 100k do 70B) — suficiente pra ~400 ann/dia
        model = os.environ.get("GROQ_MODEL", "llama-3.1-8b-instant")
        return "groq", model, Groq(api_key=GROQ_KEY), annotate_with_groq

    if want in ("auto", "anthropic") and ANTHROPIC_KEY:
        try:
            import anthropic
        except ImportError:
            sys.exit("ANTHROPIC_API_KEY setado mas anthropic não instalado: pip install anthropic")
        model = os.environ.get("ANTHROPIC_MODEL", "claude-sonnet-4-6")
        return "anthropic", model, anthropic.Anthropic(api_key=ANTHROPIC_KEY), annotate_with_anthropic

    sys.exit(
        "Nenhuma API key configurada. Exporte uma:\n"
        "  export GROQ_API_KEY=gsk_...        # free 14400 req/dia (recomendado)\n"
        "  export ANTHROPIC_API_KEY=sk-ant-... # pago (~US$1 pra 261)\n"
    )

def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--limit", type=int, default=None)
    parser.add_argument("--resume", action="store_true")
    parser.add_argument("--provider", choices=["auto", "groq", "anthropic"], default="auto",
                        help="Força um provider específico (default: auto-detecta)")
    args = parser.parse_args()

    provider, model, client, annotate_fn = pick_provider(args.provider)
    print(f"Provider: {provider} · model: {model}")

    OUTPUT_PATH.parent.mkdir(parents=True, exist_ok=True)

    # Carrega já anotados (idempotente)
    done = set()
    if args.resume and OUTPUT_PATH.exists():
        with OUTPUT_PATH.open() as f:
            for line in f:
                try: done.add(json.loads(line)["opportunity_id"])
                except: pass
        print(f"Resume: pulando {len(done)} já anotados")

    opps = load_opportunities(args.limit, done)
    if not opps:
        print("Nada para anotar.")
        return 0
    print(f"Anotando {len(opps)} oportunidades...")

    ok, err = 0, 0
    with OUTPUT_PATH.open("a") as f:
        for opp in tqdm(opps, desc="Annotating"):
            try:
                ann = annotate_fn(client, opp, model)
                ann["_provider"] = provider
                ann["_model"]    = model
                f.write(json.dumps(ann, ensure_ascii=False) + "\n")
                f.flush()
                ok += 1
                time.sleep(SLEEP_S)
            except json.JSONDecodeError as e:
                tqdm.write(f"[parse fail] id={opp['id']}: {e}")
                err += 1
            except Exception as e:
                tqdm.write(f"[error] id={opp['id']}: {type(e).__name__}: {e}")
                err += 1
                time.sleep(5)  # backoff em erro não-rate-limit

    print(f"\nDone. Anotados: {ok} · Erros: {err}")
    print(f"Output: {OUTPUT_PATH}")
    return 0

if __name__ == "__main__":
    sys.exit(main())
