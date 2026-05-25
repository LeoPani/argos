"""
Argos — Fase 2: Análise exploratória do dataset anotado.

Stats descritivas pra entender:
  - Distribuição de classes (patenteável/não)
  - Distribuição de IPC
  - Viés por departamento
  - Concordância anotação vs heurística atual
  - Confiança média
"""

import json
from pathlib import Path
from collections import Counter

DATA = Path(__file__).parent / "data" / "annotations.jsonl"

def main():
    if not DATA.exists():
        print(f"Não existe ainda: {DATA}\nRode 01_annotate.py primeiro.")
        return

    rows = [json.loads(l) for l in DATA.read_text().splitlines() if l]
    n = len(rows)
    if n == 0:
        print("Dataset vazio.")
        return

    print(f"━━━ Dataset anotado: {n} oportunidades ━━━\n")

    # Distribuição patenteável
    pat = Counter(r.get("is_patentable") for r in rows)
    print("Patenteável (Claude):")
    print(f"  True:  {pat[True]:4d}  ({pat[True]/n*100:.1f}%)")
    print(f"  False: {pat[False]:4d}  ({pat[False]/n*100:.1f}%)")
    print()

    # Distribuição IPC
    ipc = Counter(r.get("ipc_category") for r in rows)
    print("IPC (Claude):")
    letters = ["A","B","C","D","E","F","G","H"]
    for cat in sorted(ipc.keys(), key=lambda x: (x is None, x)):
        label = "—" if cat is None else letters[cat]
        print(f"  {label}: {ipc[cat]:4d}  ({ipc[cat]/n*100:.1f}%)")
    print()

    # Confiança
    confs = [r.get("confidence", 0) for r in rows if r.get("confidence") is not None]
    if confs:
        avg = sum(confs)/len(confs)
        print(f"Confiança média (Claude): {avg:.2f}")
        high_conf = sum(1 for c in confs if c >= 0.8)
        print(f"  >= 0.80 (alta): {high_conf} ({high_conf/n*100:.1f}%)")
        print()

    # Concordância vs heurística atual
    print("Concordância Claude vs heurística atual:")
    matrix = Counter()
    for r in rows:
        claude_pat = bool(r.get("is_patentable"))
        heur_level = r.get("heuristic_level", "low")
        heur_pat = heur_level in ("high", "medium")
        matrix[(claude_pat, heur_pat)] += 1

    tp = matrix.get((True, True), 0)
    fp = matrix.get((False, True), 0)
    fn = matrix.get((True, False), 0)
    tn = matrix.get((False, False), 0)
    total = tp + fp + fn + tn

    if total > 0:
        agree = tp + tn
        print(f"  Concordam:     {agree:4d} ({agree/total*100:.1f}%)")
        print(f"  Heurística FP: {fp:4d} (heur disse sim, Claude disse não)")
        print(f"  Heurística FN: {fn:4d} (heur disse não, Claude disse sim)")
        if tp + fp > 0:
            prec = tp / (tp + fp)
            print(f"  Heurística precision: {prec:.2f}")
        if tp + fn > 0:
            rec = tp / (tp + fn)
            print(f"  Heurística recall:    {rec:.2f}")

    print()
    print("Top 5 com alta confiança patenteável:")
    high_pat = [r for r in rows if r.get("is_patentable") and (r.get("confidence", 0) >= 0.8)]
    high_pat.sort(key=lambda r: r.get("confidence", 0), reverse=True)
    for r in high_pat[:5]:
        print(f"  [{r['confidence']:.2f}] {r['title'][:60]}")
        print(f"         → {r.get('rationale', '')[:100]}")

if __name__ == "__main__":
    main()
