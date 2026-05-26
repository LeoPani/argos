"""
Argos — Fase 2: Análise exploratória do dataset UFOP.

Funciona em três modos (auto-detect):

  A) Só Postgres (heurística v2 disponível):
     Lê 4212 ufop_opportunities, gera stats descritivas:
       - Distribuição is_patentable, opportunity_level, IPC
       - Top departments com HIGH
       - Top trabalhos com PI score alto
       - Distribuição de confidence

  B) Só LLM (annotations.jsonl):
     Análise como anteriormente (script v1).

  C) Ambos disponíveis:
     Modo completo — compara heurística vs LLM, calcula:
       - Agreement / Cohen's kappa
       - Matriz de confusão
       - Casos onde divergem

Output (sempre em ai-service/training/outputs/):
  - explore_report.txt          (relatório texto)
  - distribuicoes.json          (counts pra plotagem futura)
  - amostras_high.json          (top 20 HIGH segundo cada método)
  - divergencias.jsonl          (casos onde LLM≠heurística, p/ inspeção manual)

Uso:
    cd ai-service
    pip install psycopg2-binary
    python training/02_explore.py
"""

import json
import os
import sys
from collections import Counter
from pathlib import Path

try:
    import psycopg2
except ImportError as e:
    sys.exit(f"Faltam dependências: pip install psycopg2-binary  ({e})")

# ─── Config ─────────────────────────────────────────────────────────────────

DATA_DIR    = Path(__file__).parent / "data"
OUTPUT_DIR  = Path(__file__).parent / "outputs"
ANNOT_PATH  = DATA_DIR / "annotations.jsonl"
DB_URL      = os.environ.get("DATABASE_URL",
    "postgres://argos:argos_dev@localhost:5432/argos")

IPC_LETTERS = ["A", "B", "C", "D", "E", "F", "G", "H"]
IPC_NAMES = [
    "Necessidades humanas", "Operações/transportes", "Química/metalurgia",
    "Têxteis/papel", "Construção", "Mec. industrial",
    "Física/TI", "Eletricidade",
]


# ─── Loaders ────────────────────────────────────────────────────────────────

def load_postgres() -> list[dict]:
    """Lê ufop_opportunities do banco. Retorna [] se falhar."""
    try:
        conn = psycopg2.connect(DB_URL)
    except Exception as e:
        print(f"[postgres unavailable] {e}")
        return []
    cur = conn.cursor()
    cur.execute("""
        SELECT id, external_id, title, department, opportunity_level,
               ipc_category, pi_score, is_patentable, classifier_version,
               confidence, COALESCE(rationale, ''), COALESCE(abstract, '')
        FROM ufop_opportunities
        ORDER BY pi_score DESC
    """)
    cols = ["id", "external_id", "title", "department", "level",
            "ipc_category", "pi_score", "is_patentable",
            "classifier_version", "confidence", "rationale", "abstract"]
    rows = [dict(zip(cols, r)) for r in cur.fetchall()]
    cur.close()
    conn.close()
    return rows


def load_annotations() -> list[dict]:
    """Lê annotations.jsonl (output da Fase 1). Retorna [] se não existir."""
    if not ANNOT_PATH.exists():
        return []
    out = []
    for line in ANNOT_PATH.read_text().splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            out.append(json.loads(line))
        except json.JSONDecodeError:
            continue
    return out


# ─── Análise A: Postgres / heurística ──────────────────────────────────────

def analyze_postgres(rows: list[dict]) -> dict:
    n = len(rows)
    if n == 0:
        return {"error": "empty"}

    pat   = Counter(r["is_patentable"] for r in rows)
    level = Counter(r["level"] for r in rows)
    ipc   = Counter(r["ipc_category"] for r in rows)
    clf   = Counter(r["classifier_version"] or "—" for r in rows)
    dept  = Counter(r["department"] or "—" for r in rows)

    confs = [r["confidence"] for r in rows if r["confidence"] is not None]
    scores = [r["pi_score"] for r in rows]

    # Top departments com HIGH patenteável
    top_high_dept = Counter()
    for r in rows:
        if r["level"] == "high" and r["is_patentable"]:
            top_high_dept[r["department"] or "—"] += 1

    return {
        "n": n,
        "patentable": {
            "true":  pat.get(True, 0),
            "false": pat.get(False, 0),
            "null":  pat.get(None, 0),
        },
        "level": dict(level),
        "ipc": {IPC_LETTERS[k] if k is not None and 0 <= k < 8 else "?": v
                for k, v in ipc.items()},
        "classifier": dict(clf),
        "top_departments": dict(dept.most_common(15)),
        "top_high_departments": dict(top_high_dept.most_common(10)),
        "confidence": {
            "avg": sum(confs) / len(confs) if confs else 0.0,
            "high_count": sum(1 for c in confs if c >= 0.8),
            "low_count":  sum(1 for c in confs if c < 0.5),
        },
        "pi_score": {
            "avg": sum(scores) / len(scores) if scores else 0.0,
            "max": max(scores) if scores else 0,
            "min": min(scores) if scores else 0,
        },
    }


def top_samples(rows: list[dict], k: int = 20) -> list[dict]:
    """Top K patenteáveis com maior pi_score."""
    cands = [r for r in rows if r["is_patentable"]]
    cands.sort(key=lambda r: (r["pi_score"], r.get("confidence") or 0), reverse=True)
    return [
        {"id": r["id"], "title": r["title"][:200],
         "department": r["department"], "ipc": r["ipc_category"],
         "level": r["level"], "pi_score": r["pi_score"],
         "rationale": r["rationale"][:300]}
        for r in cands[:k]
    ]


def rejected_samples(rows: list[dict], k: int = 20) -> list[dict]:
    """Top K rejeitadas — pra auditar se a regra Art. 10 LPI está pegando bem."""
    cands = [r for r in rows if r["is_patentable"] is False]
    cands.sort(key=lambda r: r["pi_score"], reverse=True)
    return [
        {"id": r["id"], "title": r["title"][:200],
         "department": r["department"], "rationale": r["rationale"][:300]}
        for r in cands[:k]
    ]


# ─── Análise C: Comparação heurística vs LLM ────────────────────────────────

def compare_heuristic_vs_llm(pg_rows: list[dict], annotations: list[dict]) -> dict:
    """Calcula concordância + matriz confusão entre os dois métodos."""
    ann_by_id = {a["opportunity_id"]: a for a in annotations}
    paired = [(r, ann_by_id[r["id"]]) for r in pg_rows if r["id"] in ann_by_id]

    if not paired:
        return {"paired": 0}

    matrix = Counter()
    divergences = []
    for r, a in paired:
        heur = bool(r["is_patentable"])
        llm  = bool(a.get("is_patentable"))
        matrix[(heur, llm)] += 1
        if heur != llm:
            divergences.append({
                "id":         r["id"],
                "title":      r["title"][:150],
                "department": r["department"],
                "heuristic":  {"is_patentable": heur, "rationale": r["rationale"][:200]},
                "llm":        {"is_patentable": llm,
                              "rationale": a.get("rationale", "")[:200],
                              "confidence": a.get("confidence")},
            })

    tp = matrix.get((True, True), 0)
    fp = matrix.get((True, False), 0)   # heur disse sim, LLM disse não
    fn = matrix.get((False, True), 0)   # heur disse não, LLM disse sim
    tn = matrix.get((False, False), 0)
    total = tp + fp + fn + tn
    accuracy = (tp + tn) / total if total else 0.0

    # Cohen's kappa
    po = accuracy
    pe_yes = ((tp + fp) / total) * ((tp + fn) / total)
    pe_no  = ((fn + tn) / total) * ((fp + tn) / total)
    pe = pe_yes + pe_no
    kappa = (po - pe) / (1 - pe) if pe < 1 else 0.0

    return {
        "paired": total,
        "confusion": {
            "true_patentable_both":     tp,
            "false_patentable_both":    tn,
            "heur_yes_llm_no":          fp,
            "heur_no_llm_yes":          fn,
        },
        "accuracy": round(accuracy, 3),
        "cohens_kappa": round(kappa, 3),
        "divergences": divergences,
    }


# ─── Report writer ──────────────────────────────────────────────────────────

def write_report(pg_stats: dict, annotations: list[dict], comparison: dict | None,
                 top: list[dict], rejected: list[dict]) -> str:
    out = []
    out.append("=" * 70)
    out.append("ARGOS — Análise exploratória do dataset UFOP (Fase 2)")
    out.append("=" * 70)
    out.append("")

    out.append(f"## Dataset")
    out.append(f"- Total ufop_opportunities (Postgres): {pg_stats['n']}")
    out.append(f"- Anotações LLM (annotations.jsonl):   {len(annotations)}")
    out.append("")

    # Patenteabilidade
    p = pg_stats["patentable"]
    n = pg_stats["n"]
    out.append("## Patenteabilidade (heurística v2)")
    out.append(f"- Patenteáveis (Art. 8):     {p['true']:4d} ({p['true']/n*100:.1f}%)")
    out.append(f"- Rejeitadas (Art. 10 LPI):  {p['false']:4d} ({p['false']/n*100:.1f}%)")
    out.append(f"- Sem decisão (null):        {p['null']:4d} ({p['null']/n*100:.1f}%)")
    out.append("")

    # Level
    lvl = pg_stats["level"]
    out.append("## Distribuição por nível de oportunidade")
    for k in ("high", "medium", "low"):
        v = lvl.get(k, 0)
        out.append(f"- {k:6s}: {v:4d} ({v/n*100:.1f}%)")
    out.append("")

    # IPC
    out.append("## Distribuição IPC (heurística v2)")
    for letter in IPC_LETTERS:
        v = pg_stats["ipc"].get(letter, 0)
        idx = IPC_LETTERS.index(letter)
        out.append(f"- {letter} ({IPC_NAMES[idx]:<24s}): {v:4d} ({v/n*100:.1f}%)")
    out.append("")

    # Top dept HIGH
    out.append("## Departamentos com mais oportunidades HIGH patenteáveis")
    for dept, count in pg_stats["top_high_departments"].items():
        out.append(f"- {count:3d} · {dept}")
    out.append("")

    # Classifier mix
    out.append("## Classifier mix")
    for clf, count in pg_stats["classifier"].items():
        out.append(f"- {count:4d} · {clf}")
    out.append("")

    # Confiança
    c = pg_stats["confidence"]
    out.append("## Confiança da classificação")
    out.append(f"- Média:       {c['avg']:.2f}")
    out.append(f"- Alta (≥0.8): {c['high_count']}")
    out.append(f"- Baixa (<0.5):{c['low_count']}")
    out.append("")

    # PI Score
    s = pg_stats["pi_score"]
    out.append("## PI Score")
    out.append(f"- Média: {s['avg']:.2f} · Min: {s['min']:.1f} · Max: {s['max']:.1f}")
    out.append("")

    # Comparação
    if comparison is not None and comparison.get("paired", 0) > 0:
        out.append("## Heurística vs LLM (Cohen's kappa)")
        cm = comparison["confusion"]
        out.append(f"- Pareados: {comparison['paired']}")
        out.append(f"- Concordam patenteável:   {cm['true_patentable_both']}")
        out.append(f"- Concordam rejeitado:     {cm['false_patentable_both']}")
        out.append(f"- Heur YES, LLM NO:        {cm['heur_yes_llm_no']}")
        out.append(f"- Heur NO,  LLM YES:       {cm['heur_no_llm_yes']}")
        out.append(f"- Accuracy:       {comparison['accuracy']:.3f}")
        out.append(f"- Cohen's kappa:  {comparison['cohens_kappa']:.3f}")
        out.append("  (>0.81 = quase perfeito; 0.61-0.80 = substancial; 0.41-0.60 = moderado; <0.40 = ruim)")
        out.append("")
    else:
        out.append("## Heurística vs LLM")
        out.append("  Nenhum dado de LLM disponível. Rode 01_annotate.py com GROQ_API_KEY.")
        out.append("")

    out.append("## Top 5 oportunidades (preview — todos em amostras_high.json)")
    for r in top[:5]:
        out.append(f"- [{r['pi_score']:.1f}] {r['title']}")
        out.append(f"     {r['department']}")
    out.append("")

    out.append("## Top 5 rejeitadas (preview — auditar regra Art. 10)")
    for r in rejected[:5]:
        out.append(f"- {r['title']}")
        out.append(f"     {r['department']} · {r['rationale'][:80]}")
    out.append("")

    return "\n".join(out)


# ─── Main ───────────────────────────────────────────────────────────────────

def main():
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

    pg_rows = load_postgres()
    annots  = load_annotations()

    if not pg_rows and not annots:
        print("Nenhum dataset disponível. Inicia o banco (make db-up) ou roda 01_annotate.py.")
        return 1

    pg_stats = analyze_postgres(pg_rows) if pg_rows else {"n": 0, "patentable": {"true":0,"false":0,"null":0},
                                                          "level": {}, "ipc": {}, "classifier": {},
                                                          "top_departments": {}, "top_high_departments": {},
                                                          "confidence": {"avg":0,"high_count":0,"low_count":0},
                                                          "pi_score": {"avg":0,"max":0,"min":0}}
    comparison = compare_heuristic_vs_llm(pg_rows, annots) if (pg_rows and annots) else None

    top      = top_samples(pg_rows, k=20) if pg_rows else []
    rejected = rejected_samples(pg_rows, k=20) if pg_rows else []

    # Console + arquivo
    report = write_report(pg_stats, annots, comparison, top, rejected)
    print(report)
    (OUTPUT_DIR / "explore_report.txt").write_text(report)

    # JSON outputs
    (OUTPUT_DIR / "distribuicoes.json").write_text(json.dumps({
        "pg_stats": pg_stats,
        "n_annotations": len(annots),
        "comparison": comparison,
    }, indent=2, ensure_ascii=False))

    (OUTPUT_DIR / "amostras_high.json").write_text(
        json.dumps(top, indent=2, ensure_ascii=False))

    if comparison and comparison.get("divergences"):
        with (OUTPUT_DIR / "divergencias.jsonl").open("w") as f:
            for d in comparison["divergences"]:
                f.write(json.dumps(d, ensure_ascii=False) + "\n")
        print(f"\n→ {len(comparison['divergences'])} divergências em outputs/divergencias.jsonl")

    print(f"\n→ Outputs em: {OUTPUT_DIR}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
