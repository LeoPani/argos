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

Uso:
    export ANTHROPIC_API_KEY=sk-ant-...
    cd ai-service
    pip install anthropic psycopg2-binary tqdm
    python training/01_annotate.py             # all
    python training/01_annotate.py --limit 50  # primeiros 50 (teste)
    python training/01_annotate.py --resume    # continua de onde parou
"""

import argparse
import json
import os
import sys
import time
from pathlib import Path

try:
    import anthropic
    import psycopg2
    from tqdm import tqdm
except ImportError as e:
    sys.exit(f"Faltam dependências: pip install anthropic psycopg2-binary tqdm  ({e})")

# ─── Config ─────────────────────────────────────────────────────────────────

OUTPUT_PATH = Path(__file__).parent / "data" / "annotations.jsonl"
DB_URL      = os.environ.get("DATABASE_URL",
    "postgres://argos:argos_dev@localhost:5432/argos")
ANTHROPIC_KEY = os.environ.get("ANTHROPIC_API_KEY")
MODEL       = "claude-sonnet-4-6"
SLEEP_S     = 0.5  # respeitar rate limit

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

def annotate_one(client: anthropic.Anthropic, opp: dict) -> dict:
    """Envia ao Claude e parseia o JSON de volta."""
    msg = client.messages.create(
        model=MODEL,
        max_tokens=512,
        system=SYSTEM_PROMPT,
        messages=[{
            "role": "user",
            "content": USER_TEMPLATE.format(
                title=opp["title"][:300],
                department=opp.get("department", "UFOP"),
                abstract=opp.get("abstract", "")[:3000],
            ),
        }],
    )
    text = msg.content[0].text.strip()
    # Limpa code fence se Claude botou
    if text.startswith("```"):
        text = text.strip("`").split("\n", 1)[1].rsplit("```", 1)[0]
    if text.startswith("json"):
        text = text[4:].strip()
    parsed = json.loads(text)
    parsed["opportunity_id"] = opp["id"]
    parsed["title"] = opp["title"]
    parsed["department"] = opp.get("department", "")
    parsed["heuristic_level"] = opp["level"]
    return parsed

def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--limit", type=int, default=None)
    parser.add_argument("--resume", action="store_true")
    args = parser.parse_args()

    if not ANTHROPIC_KEY:
        print("ERRO: ANTHROPIC_API_KEY não está no ambiente.\n"
              "Setar com: export ANTHROPIC_API_KEY=sk-ant-...")
        return 1

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
    print(f"Anotando {len(opps)} oportunidades via {MODEL}...")

    client = anthropic.Anthropic(api_key=ANTHROPIC_KEY)
    ok, err = 0, 0
    with OUTPUT_PATH.open("a") as f:
        for opp in tqdm(opps, desc="Annotating"):
            try:
                ann = annotate_one(client, opp)
                f.write(json.dumps(ann, ensure_ascii=False) + "\n")
                f.flush()
                ok += 1
                time.sleep(SLEEP_S)
            except json.JSONDecodeError as e:
                print(f"\n[parse fail] id={opp['id']}: {e}")
                err += 1
            except Exception as e:
                print(f"\n[error] id={opp['id']}: {type(e).__name__}: {e}")
                err += 1
                time.sleep(2)  # backoff em erro

    print(f"\nDone. Anotados: {ok} · Erros: {err}")
    print(f"Output: {OUTPUT_PATH}")
    return 0

if __name__ == "__main__":
    sys.exit(main())
