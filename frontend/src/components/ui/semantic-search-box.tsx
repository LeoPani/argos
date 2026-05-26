"use client";

// SemanticSearchBox — busca semântica local via TF-IDF + cosine.
// Não usa Claude/BERT/Lens. Documentado em METHODOLOGY.md.

import { useState } from "react";
import { Search, Loader2, Sparkles } from "lucide-react";
import { api } from "@/lib/api";
import type { SemanticHit } from "@/lib/types";

export function SemanticSearchBox({ placeholder }: { placeholder?: string }) {
  const [q, setQ] = useState("");
  const [loading, setLoading] = useState(false);
  const [hits, setHits] = useState<SemanticHit[] | null>(null);
  const [docCount, setDocCount] = useState(0);

  async function run() {
    const query = q.trim();
    if (!query) return;
    setLoading(true);
    try {
      const r = await api.search.semantic(query, 10);
      setHits(r.hits);
      setDocCount(r.doc_count);
    } catch {
      setHits([]);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2 px-3 py-2 rounded-lg"
        style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
        <Sparkles size={13} className="text-indigo-400" />
        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Enter") run(); }}
          placeholder={placeholder ?? "Busca semântica (TF-IDF) — ex: beneficiamento de minério, contrato de licenciamento…"}
          className="flex-1 bg-transparent text-sm text-white placeholder:text-slate-500 outline-none"
        />
        <button
          onClick={run}
          disabled={loading || !q.trim()}
          className="text-xs px-2 py-1 rounded flex items-center gap-1 transition"
          style={{
            background: "#6366f120",
            border: "1px solid #6366f140",
            color: "#a5b4fc",
            opacity: loading || !q.trim() ? 0.5 : 1,
          }}
        >
          {loading ? <Loader2 size={11} className="animate-spin" /> : <Search size={11} />}
          buscar
        </button>
      </div>

      {hits && (
        <div className="rounded-lg p-3 space-y-2"
          style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
          <div className="flex items-center justify-between text-xs">
            <span style={{ color: "var(--text-muted)" }}>
              {hits.length} resultados · {docCount} docs indexados · método: TF-IDF + cosine
            </span>
            <button onClick={() => { setHits(null); setQ(""); }}
              className="text-slate-500 hover:text-white">
              limpar
            </button>
          </div>
          {hits.length === 0 ? (
            <p className="text-xs italic" style={{ color: "var(--text-muted)" }}>
              Nenhum documento semelhante encontrado.
            </p>
          ) : (
            <ul className="space-y-1.5">
              {hits.map((h) => (
                <li key={`${h.kind}-${h.id}`}
                  className="rounded px-2 py-1.5 text-sm hover:bg-white/5 transition"
                  style={{ background: "var(--surface-2)" }}>
                  <div className="flex items-start justify-between gap-3">
                    <span className="text-white truncate">{h.title || "(sem título)"}</span>
                    <span className="text-xs whitespace-nowrap"
                      style={{ color: scoreColor(h.score) }}>
                      {(h.score * 100).toFixed(1)}%
                    </span>
                  </div>
                  <div className="text-xs flex items-center gap-2">
                    <span className="px-1.5 py-0.5 rounded"
                      style={{ background: kindBg(h.kind), color: kindColor(h.kind) }}>
                      {h.kind === "ufop_opp" ? "UFOP" : "Patente"}
                    </span>
                    <span className="truncate" style={{ color: "var(--text-muted)" }}>
                      {h.snippet || "—"}
                    </span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}

function scoreColor(s: number): string {
  if (s >= 0.4) return "#34d399";
  if (s >= 0.2) return "#fbbf24";
  return "#94a3b8";
}
function kindColor(kind: string): string {
  return kind === "ufop_opp" ? "#a855f7" : "#6366f1";
}
function kindBg(kind: string): string {
  return kind === "ufop_opp" ? "#a855f720" : "#6366f120";
}
