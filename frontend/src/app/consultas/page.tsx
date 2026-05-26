"use client";

import { useState } from "react";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { RiskScale } from "@/components/ui/risk-scale";
import { formatDate } from "@/lib/utils";
import type { SearchResult } from "@/lib/types";
import { Search, Link2, Shield, Save } from "lucide-react";

type SearchType = "patent" | "trademark" | "both";

export default function ConsultasPage() {
  const [query, setQuery] = useState("");
  const [type, setType] = useState<SearchType>("patent");
  const [result, setResult] = useState<SearchResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSearch() {
    if (!query.trim()) return;
    setLoading(true);
    setResult(null);
    setError(null);
    try {
      const res = await fetch(
        `/api/prior-art?q=${encodeURIComponent(query)}&kind=${type}`
      );
      if (!res.ok) {
        setError(`Backend retornou ${res.status}. Verifique se o argos-api está rodando.`);
        return;
      }
      const data = await res.json();
      const hits = (data.Hits ?? data.hits ?? []).map((h: { Number?: string; number?: string; Title?: string; title?: string; Owner?: string; owner?: string; FilingDate?: string; filing_date?: string; SimilarityPct?: number; similarity_pct?: number }) => ({
        number: h.Number ?? h.number ?? "",
        title: h.Title ?? h.title ?? "",
        owner: h.Owner ?? h.owner ?? "",
        filing_date: h.FilingDate ?? h.filing_date ?? "",
        similarity_pct: h.SimilarityPct ?? h.similarity_pct ?? 0,
      }));
      const score = data.RiskScore ?? data.risk_score ?? 0;
      setResult({
        query, type,
        risk_score: score,
        risk_label: score <= 3 ? "Baixo" : score <= 6 ? "Médio" : score <= 8 ? "Alto" : "Muito Alto",
        conflicts: hits,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Erro de rede.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="p-8 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Consulta de Anterioridade</h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Busca em patentes/marcas/publicações <strong>locais</strong> (portfolio Argos)
          por similaridade Jaccard de bigrams. Não substitui busca oficial no INPI/Espacenet.
        </p>
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
              placeholder="ex: Software de classificação automática de patentes usando inteligência artificial para análise de documentos do INPI..."
              className="w-full rounded-lg px-4 py-3 text-sm text-white placeholder-slate-600 resize-none outline-none focus:ring-1 focus:ring-indigo-500 transition-all"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}
            />
          </div>

          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <span className="text-xs font-medium" style={{ color: "var(--text-muted)" }}>Tipo:</span>
              {(["patent", "trademark", "both"] as SearchType[]).map(t => (
                <label key={t} className="flex items-center gap-1.5 cursor-pointer">
                  <input
                    type="radio"
                    name="type"
                    value={t}
                    checked={type === t}
                    onChange={() => setType(t)}
                    className="accent-indigo-500"
                  />
                  <span className="text-sm" style={{ color: type === t ? "white" : "var(--text-muted)" }}>
                    {t === "patent" ? "Patente" : t === "trademark" ? "Marca" : "Ambos"}
                  </span>
                </label>
              ))}
            </div>
            <Button onClick={handleSearch} disabled={loading || !query.trim()}>
              {loading ? (
                <>
                  <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                  Consultando IA...
                </>
              ) : (
                <>
                  <Search size={14} />
                  Consultar Anterioridade
                </>
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
              <CardTitle>Resultado — {result.conflicts.length} anterioridades encontradas</CardTitle>
              <Badge variant={result.risk_score > 7 ? "danger" : result.risk_score > 4 ? "warning" : "success"}>
                {result.risk_label}
              </Badge>
            </CardHeader>
            <RiskScale score={result.risk_score} />
          </Card>

          {/* Conflicts table */}
          <Card>
            <CardHeader>
              <CardTitle>Anterioridades conflitantes</CardTitle>
            </CardHeader>
            <div className="space-y-3">
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
                      {c.owner} · Depósito: {formatDate(c.filing_date)}
                    </p>
                  </div>
                  <div className="w-24">
                    <RiskScale score={c.similarity_pct / 10} showLabel={false} size="sm" />
                  </div>
                </div>
              ))}
            </div>
          </Card>

          {/* Blockchain CTA */}
          <Card style={{ borderColor: "#6366f1", borderWidth: "1px" }}>
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-indigo-500/20">
                  <Shield size={18} className="text-indigo-400" />
                </div>
                <div>
                  <p className="text-sm font-semibold text-white">
                    Registrar timestamp desta consulta na blockchain
                  </p>
                  <p className="text-xs mt-0.5" style={{ color: "var(--text-muted)" }}>
                    Polygon Mainnet · Custo estimado: ~R$ 0,10 · Prova imutável de data de consulta
                  </p>
                </div>
              </div>
              <div className="flex gap-2 shrink-0">
                <Button variant="secondary" size="sm">
                  <Save size={12} />
                  Salvar no portfolio
                </Button>
                <Button size="sm">
                  <Link2 size={12} />
                  Registrar na blockchain
                </Button>
              </div>
            </div>
          </Card>
        </div>
      )}
    </div>
  );
}
