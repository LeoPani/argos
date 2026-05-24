// CSV export helpers — client-side only.
//
// We use ; (semicolon) as separator: Excel-PT-BR opens it natively
// without prompting for delimiter. Each row is enclosed when needed,
// double-quotes inside fields are doubled per RFC 4180.

/** Convert a value to a safely-quoted CSV cell. */
function csvCell(v: unknown): string {
  if (v === null || v === undefined) return "";
  const s = String(v);
  if (s.includes(";") || s.includes('"') || s.includes("\n") || s.includes("\r")) {
    return `"${s.replace(/"/g, '""')}"`;
  }
  return s;
}

/** Build a CSV string from rows. The first row defines the columns. */
export function toCSV<T extends Record<string, unknown>>(
  rows: T[],
  columns?: { key: keyof T; label: string }[],
): string {
  if (rows.length === 0) return "";

  const cols = columns ?? (Object.keys(rows[0]) as (keyof T)[]).map(k => ({ key: k, label: String(k) }));

  const header = cols.map(c => csvCell(c.label)).join(";");
  const body   = rows.map(r => cols.map(c => csvCell(r[c.key])).join(";")).join("\n");

  // Excel reads UTF-8 better with a BOM
  return "﻿" + header + "\n" + body;
}

/** Trigger a file download from a CSV string. */
export function downloadCSV(csv: string, filename: string) {
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
  const url  = URL.createObjectURL(blob);
  const a    = document.createElement("a");
  a.href     = url;
  a.download = filename.endsWith(".csv") ? filename : filename + ".csv";
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

/** Format date for CSV — ISO or empty. */
export function csvDate(d: string | null | undefined): string {
  if (!d) return "";
  try {
    return new Date(d).toISOString().slice(0, 10);
  } catch {
    return d;
  }
}
