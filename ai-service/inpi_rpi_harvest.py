"""
Argos — Coleta das últimas N Revistas da Propriedade Industrial (RPI) do INPI.

A RPI é o boletim oficial semanal do INPI publicado em revistas.inpi.gov.br.
Cada edição é um PDF grande (~30-80MB) com milhares de despachos sobre
patentes, marcas e desenhos industriais.

Este script:
  1. Descobre as últimas N RPIs disponíveis (parsing do índice HTML)
  2. Baixa os PDFs (skip se já em ai-service/training/data/rpi/)
  3. Extrai despachos via pdfplumber + regex
  4. Insere em `inpi_publications` no Postgres
  5. Marca is_ufop=TRUE quando applicant contém "UFOP" ou "Ouro Preto"

Uso:
    cd ai-service
    pip install pdfplumber requests psycopg2-binary tqdm
    python inpi_rpi_harvest.py --count 10
    python inpi_rpi_harvest.py --count 3 --skip-download  # só re-parsear PDFs já baixados

Nota: o INPI não tem API oficial. Esta é a melhor alternativa free e
auditável. Lens.org seria mais limpo mas exige token.
"""

import argparse
import os
import re
import sys
from datetime import datetime
from pathlib import Path
from urllib.parse import urljoin

try:
    import pdfplumber
    import psycopg2
    import requests
    from tqdm import tqdm
except ImportError as e:
    sys.exit(f"Faltam dependências: pip install pdfplumber requests psycopg2-binary tqdm  ({e})")

DATA_DIR    = Path(__file__).parent / "training" / "data" / "rpi"
DB_URL      = os.environ.get("DATABASE_URL",
    "postgres://argos:argos_dev@localhost:5432/argos")
INDEX_URL   = "https://revistas.inpi.gov.br/rpi/"
USER_AGENT  = "argos-research/1.0 (UFOP academic project)"

# Padrões observados na RPI (Manual de Despachos do INPI):
RE_PROCESS  = re.compile(r"\b(BR\s*\d{2}\s*\d{4}\s*\d{6}[-\s]?\d|\d{6,9})\b")
RE_DESPACHO = re.compile(r"^\(?(\d{1,2}\.\d{1,2}(?:\.\d{1,2})?)\)?")
RE_IPC      = re.compile(r"\b([A-H]\d{2}[A-Z]\s?\d+/\d{2,4})\b")
RE_NICE     = re.compile(r"NCL\s*\(?\d+\)?\s*[:\s]+([\d,\s]+)")
RE_UFOP     = re.compile(r"\b(ufop|federal\s+de\s+ouro\s+preto|universidade\s+federal\s+ouro\s+preto)\b", re.I)


def list_recent_rpis(count: int) -> list[dict]:
    """Faz scraping leve do índice HTML pra descobrir as últimas N RPIs."""
    resp = requests.get(INDEX_URL, headers={"User-Agent": USER_AGENT}, timeout=30)
    resp.raise_for_status()
    html = resp.text

    # O índice atual lista RPIs como links contendo /rpi/{NUM}/ ou similar.
    # Padrão regex flexível: extrai número + URL relativa.
    rpis = []
    for m in re.finditer(r'href="([^"]*?(?:rpi[/_-]?(\d{3,5})[^"]*?))"', html, re.I):
        url, num = urljoin(INDEX_URL, m.group(1)), int(m.group(2))
        if not any(r["number"] == num for r in rpis):
            rpis.append({"number": num, "index_url": url})

    rpis.sort(key=lambda r: r["number"], reverse=True)
    return rpis[:count]


def find_pdf_in_index(index_url: str) -> str | None:
    """Dado o HTML de uma RPI, extrai a URL do PDF principal."""
    try:
        r = requests.get(index_url, headers={"User-Agent": USER_AGENT}, timeout=30)
        r.raise_for_status()
    except Exception:
        return None

    # Procura .pdf na página da RPI
    m = re.search(r'href="([^"]+\.pdf)"', r.text, re.I)
    if m:
        return urljoin(index_url, m.group(1))
    return None


def download_pdf(url: str, dest: Path) -> bool:
    if dest.exists() and dest.stat().st_size > 0:
        return True
    dest.parent.mkdir(parents=True, exist_ok=True)
    try:
        with requests.get(url, headers={"User-Agent": USER_AGENT}, stream=True, timeout=120) as r:
            r.raise_for_status()
            total = int(r.headers.get("content-length", 0))
            with dest.open("wb") as f, tqdm(
                total=total, unit="B", unit_scale=True, desc=dest.name, leave=False,
            ) as bar:
                for chunk in r.iter_content(chunk_size=64 * 1024):
                    f.write(chunk)
                    bar.update(len(chunk))
        return True
    except Exception as e:
        print(f"[download fail] {url}: {e}")
        if dest.exists():
            dest.unlink()
        return False


def parse_pdf(pdf_path: Path, rpi_number: int) -> list[dict]:
    """Extrai despachos do PDF. Heurística por seções e blocos."""
    despachos = []
    section = "unknown"

    try:
        pdf = pdfplumber.open(pdf_path)
    except Exception as e:
        print(f"[pdf open fail] {pdf_path}: {e}")
        return []

    try:
        for page in pdf.pages:
            text = page.extract_text() or ""
            up   = text.upper()
            if "PATENTES" in up[:300]:
                section = "patentes"
            elif "MARCAS" in up[:300]:
                section = "marcas"
            elif "DESENHO INDUSTRIAL" in up[:300] or "DESENHOS INDUSTRIAIS" in up[:300]:
                section = "des_ind"

            # Cada bloco geralmente começa com código de despacho + nº processo
            blocks = re.split(r"\n(?=\(?\d{1,2}\.\d{1,2}(?:\.\d{1,2})?\)?\s)", text)
            for block in blocks:
                block = block.strip()
                if len(block) < 20:
                    continue
                m_proc = RE_PROCESS.search(block)
                m_desp = RE_DESPACHO.search(block)
                if not m_proc or not m_desp:
                    continue

                ipcs    = RE_IPC.findall(block)
                niceM   = RE_NICE.search(block)
                niceCls = []
                if niceM:
                    niceCls = [int(x) for x in re.findall(r"\d+", niceM.group(1))][:30]

                applicant = extract_applicant(block)
                despachos.append({
                    "rpi_number":     rpi_number,
                    "rpi_section":    section,
                    "process_number": m_proc.group(0).replace(" ", ""),
                    "despacho_code":  m_desp.group(1),
                    "title":          extract_title(block),
                    "applicant":      applicant,
                    "ipc_codes":      ipcs,
                    "nice_class":     niceCls,
                    "raw_text":       block[:4000],
                    "is_ufop":        bool(applicant and RE_UFOP.search(applicant or "")),
                })
    finally:
        pdf.close()
    return despachos


def extract_applicant(block: str) -> str | None:
    """Tenta extrair depositante. RPI usa formato '(71) Depositante:' ou 'Titular:'."""
    for label in (r"\(71\)", r"\(73\)", r"Depositante", r"Titular"):
        m = re.search(rf"{label}\s*[:\-]?\s*([^\n;()]+)", block, re.I)
        if m:
            return m.group(1).strip()[:300]
    return None


def extract_title(block: str) -> str | None:
    """Título da invenção/marca — após '(54)' ou 'Título:'."""
    for label in (r"\(54\)", r"Título", r"Titulo"):
        m = re.search(rf"{label}\s*[:\-]?\s*([^\n]+)", block, re.I)
        if m:
            return m.group(1).strip()[:500]
    return None


def insert_batch(conn, records: list[dict]) -> int:
    if not records:
        return 0
    cur = conn.cursor()
    inserted = 0
    for r in records:
        try:
            cur.execute("""
                INSERT INTO inpi_publications
                  (rpi_number, rpi_section, process_number, despacho_code,
                   title, applicant, ipc_codes, nice_class, raw_text, is_ufop)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (rpi_number, process_number, despacho_code) DO NOTHING
            """, (
                r["rpi_number"], r["rpi_section"], r["process_number"], r["despacho_code"],
                r["title"], r["applicant"], r["ipc_codes"], r["nice_class"],
                r["raw_text"], r["is_ufop"],
            ))
            if cur.rowcount > 0:
                inserted += 1
        except Exception as e:
            print(f"[insert fail] {r['process_number']}: {e}")
    conn.commit()
    cur.close()
    return inserted


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--count", type=int, default=10, help="Número de RPIs (default: 10)")
    parser.add_argument("--skip-download", action="store_true",
                        help="Não baixa, só re-parseia PDFs em data/rpi/")
    args = parser.parse_args()

    DATA_DIR.mkdir(parents=True, exist_ok=True)

    if args.skip_download:
        pdfs = sorted(DATA_DIR.glob("rpi-*.pdf"))[-args.count:]
        print(f"Reparsing {len(pdfs)} PDFs locais (skip-download)")
    else:
        print(f"Buscando últimas {args.count} RPIs em {INDEX_URL}")
        rpis = list_recent_rpis(args.count)
        if not rpis:
            print("⚠ Não consegui identificar nenhuma RPI no índice. INPI pode ter "
                  "mudado o layout — verifique manualmente em "
                  f"{INDEX_URL}")
            return 1

        print(f"Achei: {[r['number'] for r in rpis]}")
        pdfs = []
        for r in tqdm(rpis, desc="Downloading"):
            pdf_url = find_pdf_in_index(r["index_url"])
            if not pdf_url:
                print(f"[no pdf] RPI {r['number']}")
                continue
            dest = DATA_DIR / f"rpi-{r['number']}.pdf"
            if download_pdf(pdf_url, dest):
                pdfs.append(dest)

    if not pdfs:
        print("Nenhum PDF para processar.")
        return 0

    conn = psycopg2.connect(DB_URL)
    grand_total = 0
    for pdf in pdfs:
        num_match = re.search(r"rpi-(\d+)", pdf.name)
        if not num_match:
            continue
        rpi_num = int(num_match.group(1))
        print(f"\n=== RPI {rpi_num} — extraindo {pdf.name} ({pdf.stat().st_size / 1e6:.1f}MB) ===")
        records = parse_pdf(pdf, rpi_num)
        ufop_count = sum(1 for r in records if r["is_ufop"])
        n = insert_batch(conn, records)
        grand_total += n
        print(f"   → {len(records)} despachos extraídos · {n} inseridos · {ufop_count} UFOP")

    conn.close()
    print(f"\nTotal inserido: {grand_total} despachos")
    print(f"Para conferir UFOP: SELECT * FROM inpi_publications WHERE is_ufop;")
    return 0


if __name__ == "__main__":
    sys.exit(main())
