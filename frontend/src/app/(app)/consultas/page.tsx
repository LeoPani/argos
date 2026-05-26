"use client";

import { useState } from "react";
import Link from "next/link";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { RiskScale } from "@/components/ui/risk-scale";
import { formatDate } from "@/lib/utils";
import { api } from "@/lib/api";
import type { SearchResult, SemanticSearchResponse } from "@/lib/types";
import {
  Search, Shield, ArrowRight, Zap,
  FileText, GraduationCap, Newspaper,
} from "lucide-react";

type SearchType = "patent" | "trademark" | "both";

export default function ConsultasPage() {
  const [query, setQuery]   = useState("");
  const [type, setType]     = useState<SearchType>("patent");
  const [result, setResult] = useState<SearchResult | null>(null);
  const [semantic, setSemantic] = useState<SemanticSearchResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError]   = useState<string | null>(null);

  async function handleSearch() {
    if (!query.trim()) return;
    setLoading(true);
    setResult(null);
    setSemantic(null);
    setError(null);

    try {
      // Run prior-art proxy + semantic search in parallel.
      const [priorArtRes, semanticRes] = await Promise.allSettled([
        fetch(`/api/prior-art?q=${encodeURIComponent(query)}&kind=${type}`).then(r =>
          r.ok ? r.json() : Promise.reject(`HTTP ${r.status}`)
        ),
        api.search.semantic(query, 12),
      ]);

      if (priorArtRes.status === "fulfilled") {
        const data = priorArtRes.value;
        const hits = (data.Hits ?? data.hits ?? []).map((h: {
          Number?: string; number?: string; Title?: string; title?: string;
          Owner?: string; owner?: string; FilingDate?: string; filing_date?: string;
          SimilarityPct?: number; similarity_pct?: number;
        }) => ({
          number:         h.Number ?? h.number ?? "",
          title:          h.Title  ?? h.title  ?? "",
          owner:          h.Owner  ?? h.owner  ?? "",
          filing_date:    h.FilingDate ?? h.filing_date ?? "",
          similarity_pct: h.SimilarityPct ?? h.similarity_pct ?? 0,
        }));
        const score = data.RiskScore ?? data.risk_score ?? 0;
        setResult({
          query, type,
          risk_score: score,
          risk_label: score <= 3 ? "Baixo" : score <= 6 ? "Médio" : score <= 8 ? "Alto" : "Muito Alto",
          conflicts: hits,
        });
      } else {
        setError(`Prior-art: ${priorArtRes.reason}`);
      }

      if (semanticRes.status === "fulfilled" && semanticRes.value.hits?.length > 0) {
        setSemantic(semanticRes.value);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Erro de rede.");
    } finally {
      setLoading(false);
    }
  }

  function handleKey(e: React.KeyboardEvent) {
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) handleSearch();
  }

  return (
    <div className="p-8 space-y-6 max-w-5xl mx-auto">
      <div>
        <h1 className="text-2xl font-bold text-white">Consulta de Anterioridade</h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Busca em patentes/marcas/publicações locais por similaridade Jaccard + busca semântica TF-IDF.
          Não substitui busca oficial no INPI/Espacenet.
        </p>
      </div>

      {/* Como funciona */}
      <div className="grid grid-cols-3 gap-3 text-sm">
        {[
          { icon: <Search size={14} className="text-indigo-400" />, label: "Jaccard bigrams", desc: "Similaridade textual no portfólio local (patentes + marcas)" },
          { icon: <Zap size={14} className="text-amber-400" />, label: "Semântica TF-IDF", desc: "Busca vetorial cosine em abstracts — captura sinônimos" },
          { icon: <Shield size={14} className="text-emerald-400" />, label: "Score de risco", desc: "0–10 baseado no número e força dos conflitos encontrados" },
        ].map(s => (
          <div key={s.label} className="flex items-start gap-2 p-3 rounded-lg"
            style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
            {s.icon}
            <div>
              <p className="font-medium text-white text-xs">{s.label}</p>
              <p className="text-xs mt-0.5" style={{ color: "var(--text-muted)" }}>{s.desc}</p>
            </div>
          </div>
        ))}
      </div>

      {/* Search form */}
      <Card>
        <div className="space-y-4">
          <div>
            <label className="text-xs font-medium mb-2 block" style={{ color: "var(--text-muted)" }}>
              Descreva sua marca ou ideia de invenção
            </label>
            <textarea
              rows={4}
              value={query}
              onChange={e => setQuery(e.target.value)}
              onKeyDown={handleKey}
              placeholder="ex: Software de classificação automática de patentes usando inteligência artificial para análise de documentos do INPI..."
              className="w-full rounded-lg px-4 py-3 text-sm text-white placeholder-slate-600 resize-none outline-none focus:ring-1 focus:ring-indigo-500 transition-all"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}
            />
            <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>
              Dica: use ⌘+Enter para buscar
            </p>
          </div>

          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <span className="text-xs font-medium" style={{ color: "var(--text-muted)" }}>Tipo:</span>
              {(["patent", "trademark", "both"] as SearchType[]).map(t => (
                <label key={t} className="flex items-center gap-1.5 cursor-pointer">
                  <input type="radio" name="type" value={t} checked={type === t}
                    onChange={() => setType(t)} className="accent-indigo-500" />
                  <span className="text-sm" style={{ color: type === t ? "white" : "var(--text-muted)" }}>
                    {t === "patent" ? "Patente" : t === "trademark" ? "Marca" : "Ambos"}
                  </span>
                </label>
              ))}
            </div>
            <Button onClick={handleSearch} disabled={loading || !query.trim()}>
              {loading ? (
                <><div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" /> Consultando…</>
              ) : (
                <><Search size={14} /> Consultar Anterioridade</>
              )}
            </Button>
          </div>
        </div>
      </Card>

      {/* Error */}
      {error && (
        <Card style={{ borderColor: "#ef444440" }}>
          <p className="text-sm text-red-300">⚠ {error}</p>
        </Card>
      )}

      {/* Results */}
      {result && (
        <div className="space-y-4">
          {/* Risk Score */}
          <Card>
            <CardHeader>
              <CardTitle>Resultado — {result.conflicts.length} anterioridade(s) encontrada(s)</CardTitle>
              <Badge variant={result.risk_score > 7 ? "danger" : result.risk_score > 4 ? "warning" : "success"}>
                {result.risk_label}
              </Badge>
            </CardHeader>
            <RiskScale score={result.risk_score} />
          </Card>

          {/* Conflicts table */}
          {result.conflicts.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <FileText size={14} className="text-indigo-400" />
                  Anterioridades conflitantes
                </CardTitle>
              </CardHeader>
              <div className="space-y-2">
                {result.conflicts.map(c => (
                  <div key={c.number}
                    className="flex items-center gap-4 p-3 rounded-lg"
                    style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="text-xs font-mono text-indigo-400">{c.number}</span>
                        <Badge variant={c.similarity_pct > 70 ? "danger" : c.similarity_pct > 50 ? "warning" : "muted"}>
                          {c.similarity_pct}% similar
                        </Badge>
                      </div>
                      <p className="text-sm text-white font-medium">{c.title}</p>
                      <p className="text-xs mt-0.5" style={{ color: "var(--text-muted)" }}>
                        {c.owner}
                        {c.filing_date ? ` · Depósito: ${formatDate(c.filing_date)}` : ""}
                      </p>
                    </div>
                    <div className="w-24 shrink-0">
                      <RiskScale score={c.similarity_pct / 10} showLabel={false} size="sm" />
                    </div>
                  </div>
                ))}
              </div>
            </Card>
          )}

          {result.conflicts.length === 0 && (
            <Card style={{ borderColor: "#34d39930" }}>
              <div className="flex items-center gap-3 py-2">
                <Shield size={20} className="text-emerald-400 shrink-0" />
                <div>
                  <p className="text-sm font-semibold text-emerald-400">Nenhuma anterioridade direta encontrada</p>
                  <p className="text-xs mt-0.5" style={{ color: "var(--text-muted)" }}>
                    O portfólio local não contém registros similares. Recomenda-se verificar também no INPI/Espacenet.
                  </p>
                </div>
              </div>
            </Card>
          )}

          {/* CTA: go to Anterioridade page */}
          <div className="flex items-center justify-between p-4 rounded-xl"
            style={{ background: "var(--surface-2)", border: "1px solid #6366f140" }}>
            <div className="flex items-center gap-3">
              <Shield size={18} className="text-indigo-400 shrink-0" />
              <div>
                <p className="text-sm font-semibold text-white">Registrar anterioridade desta invenção</p>
                <p className="text-xs mt-0.5" style={{ color: "var(--text-muted)" }}>
                  Gera prova criptográfica SHA-256 com timestamp UTC imutável
                </p>
              </div>
            </div>
            <Link href="/anterioridade">
              <Button size="sm">
                <Shield size={12} /> Registrar <ArrowRight size={12} />
              </Button>
            </Link>
          </div>
        </div>
      )}

      {/* Semantic search results */}
      {semantic && (semantic.hits ?? []).length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Zap size={14} className="text-amber-400" />
              Busca semântica — {semantic.hits.length} documentos relacionados
              <span className="text-xs font-normal ml-1" style={{ color: "var(--text-muted)" }}>
                ({semantic.method})
              </span>
            </CardTitle>
          </CardHeader>
          <div className="space-y-2">
            {semantic.hits.map((r, i) => {
              const isUFOP = r.kind === "ufop_opp";
              const isINPI = r.kind === "inpi";
              const Icon = isUFOP ? GraduationCap : isINPI ? Newspaper : FileText;
              const color = isUFOP ? "#a855f7" : isINPI ? "#06b6d4" : "#6366f1";
              const kindLabel = isUFOP ? "UFOP" : isINPI ? "INPI RPI" : "Patente";
              const href = r.url || (isUFOP ? `/ufop` : isINPI ? `/consultas` : `/patents/${r.id}`);
              return (
                <a key={i} href={href} target={r.url ? "_blank" : undefined} rel="noreferrer">
                  <div className="flex items-start gap-3 p-3 rounded-lg hover:bg-white/5 transition-colors"
                    style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
                    <div className="p-1.5 rounded shrink-0" style={{ background: color + "20", marginTop: 2 }}>
                      <Icon size={12} style={{ color }} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-0.5">
                        <Badge variant="muted">{kindLabel}</Badge>
                        {!isINPI && <span className="text-xs font-mono text-indigo-400">#{r.id}</span>}
                        <span className="ml-auto text-xs font-medium shrink-0"
                          style={{ color: r.score > 0.6 ? "#34d399" : r.score > 0.4 ? "#fbbf24" : "var(--text-muted)" }}>
                          {(r.score * 100).toFixed(0)}% match
                        </span>
                      </div>
                      <p className="text-sm text-white truncate">{r.title}</p>
                      {r.snippet && (
                        <p className="text-xs mt-0.5 line-clamp-2" style={{ color: "var(--text-muted)" }}>
                          {r.snippet}
                        </p>
                      )}
                    </div>
                  </div>
                </a>
              );
            })}
          </div>
          <p className="text-xs mt-3 pt-3" style={{ borderTop: "1px solid var(--border)", color: "var(--text-muted)" }}>
            {semantic.doc_count.toLocaleString("pt-BR")} documentos indexados · índice criado: {new Date(semantic.built_at).toLocaleTimeString("pt-BR")}
          </p>
        </Card>
      )}
    </div>
  );
}
