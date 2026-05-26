"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { formatDate } from "@/lib/utils";
import type { UFOPOpportunity, OpportunityLevel, UFOPStatus } from "@/lib/types";
import { useUFOPOpportunities } from "@/lib/hooks";
import { api } from "@/lib/api";
import { SkeletonKPI, SkeletonList } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { AnalysisModeBadge } from "@/components/ui/analysis-mode-badge";
import { SemanticSearchBox } from "@/components/ui/semantic-search-box";
import {
  GraduationCap, Flame, Minus, TrendingDown,
  ExternalLink, RefreshCw, CheckCircle2, XCircle,
  FileSignature, Filter,
} from "lucide-react";

// ─── helpers ─────────────────────────────────────────────────────────────────

function levelInfo(l: OpportunityLevel) {
  const map: Record<OpportunityLevel, { label: string; variant: "danger" | "warning" | "muted"; icon: React.ReactNode }> = {
    high:   { label: "Alta oportunidade",  variant: "danger",  icon: <Flame size={12} /> },
    medium: { label: "Oportunidade média", variant: "warning", icon: <Minus size={12} /> },
    low:    { label: "Baixa prioridade",   variant: "muted",   icon: <TrendingDown size={12} /> },
  };
  return map[l];
}

function sourceLabel(s: UFOPOpportunity["source"]): string {
  return ({ oai: "RI-UFOP", portal: "Portal UFOP", lens: "Lens.org" } as Record<string, string>)[s] ?? s;
}

// Áreas conhecidas para filtro (são os departments mapeados do setSpec).
const KNOWN_AREAS = [
  { key: "Engenharia de Minas",      match: "Minas"   },
  { key: "Engenharia Mineral",       match: "Mineral" },
  { key: "Escola de Minas",          match: "Escola de Minas" },
  { key: "Geologia",                 match: "Geologia" },
  { key: "Direito",                  match: "Direito" },
];

// ─── main component ──────────────────────────────────────────────────────────

const PAGE_SIZE = 50;

export default function UFOPPage() {
  const [levelFilter, setLevelFilter] = useState<OpportunityLevel | "all">("all");
  const [areaFilter, setAreaFilter]   = useState<string>("all");
  const [patentableOnly, setPatentableOnly] = useState<boolean>(true);
  const [page, setPage] = useState(0);

  // Reseta página quando filtros mudam
  useMemo(() => { setPage(0); }, [levelFilter, areaFilter, patentableOnly]);

  // Filtros agora vão pro servidor — paginação real
  const params: Record<string, string> = {
    limit:  String(PAGE_SIZE),
    offset: String(page * PAGE_SIZE),
  };
  if (patentableOnly)         params.patentable_only = "true";
  if (levelFilter !== "all")  params.level           = levelFilter;
  if (areaFilter !== "all")   params.department      = areaFilter;

  const { data, error, isLoading, mutate } = useUFOPOpportunities(params);

  // KPIs globais — só patenteáveis ativos (sem filtros adicionais)
  const { data: globalStats } = useUFOPOpportunities({ limit: "1", patentable_only: "true" });
  const { data: rejectedStats } = useUFOPOpportunities({ limit: "1" });

  const isLive  = !error && !!data;
  const loading = isLoading && !data && !error;
  const opportunities: UFOPOpportunity[] = data?.items ?? [];
  const total      = data?.pagination?.total ?? 0;
  const pageCount  = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const patentableCount    = globalStats?.pagination?.total ?? 0;
  const allCount           = rejectedStats?.pagination?.total ?? 0;
  const nonPatentableCount = Math.max(0, allCount - patentableCount);
  // KPIs adicionais só fazem sentido sobre o que veio agora (página corrente)
  const highCount   = opportunities.filter(o => o.opportunity_level === "high").length;
  const newCount    = opportunities.filter(o => o.status === "new").length;
  const converted   = opportunities.filter(o => o.status === "converted").length;

  async function handleStatus(id: number, status: UFOPStatus) {
    try {
      await api.ufop.updateStatus(id, status);
      mutate();
    } catch { /* silent */ }
  }

  return (
    <div className="p-8 space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white flex items-center gap-2">
            <GraduationCap size={22} />
            UFOP Intelligence
          </h1>
          <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
            Publicações reais do repositório UFOP analisadas para potencial de PI.
          </p>
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          <AnalysisModeBadge />
          {isLive ? (
            <span className="text-xs text-emerald-400 flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse inline-block" />
              {total} no filtro · {allCount} no total
            </span>
          ) : (
            <span className="text-xs text-amber-400">backend offline</span>
          )}
          <Button variant="ghost" size="sm" onClick={() => mutate()}>
            <RefreshCw size={13} /> Atualizar
          </Button>
        </div>
      </div>

      {/* KPIs */}
      <div className="grid grid-cols-4 gap-4">
        {loading ? (
          <><SkeletonKPI /><SkeletonKPI /><SkeletonKPI /><SkeletonKPI /></>
        ) : (
          [
            { label: "Patenteáveis (Art. 8)",   value: patentableCount.toString(), hint: `${allCount} total` },
            { label: "Nesta página · alta",     value: highCount.toString(), hint: `${total} no filtro` },
            { label: "Excluídas (Art. 10 LPI)", value: nonPatentableCount.toString(), hint: "Direito, Letras, etc" },
            { label: "Convertidas em consulta", value: converted.toString(), hint: `${newCount} aguardando revisão` },
          ].map(({ label, value, hint }) => (
            <Card key={label}>
              <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>{label}</p>
              <p className="text-2xl font-bold text-white">{value}</p>
              {hint && <p className="text-[10px] mt-0.5" style={{ color: "var(--text-muted)" }}>{hint}</p>}
            </Card>
          ))
        )}
      </div>

      {/* Busca semântica local */}
      <SemanticSearchBox placeholder="Buscar oportunidades UFOP por similaridade (ex: beneficiamento de minério, contrato de transferência)…" />

      {/* Filtros */}
      <Card>
        <div className="space-y-3">
          {/* Filtro is_patentable (Art. 10 LPI) */}
          <div className="flex items-center gap-2 flex-wrap">
            <Filter size={11} className="text-slate-500" />
            <span className="text-xs" style={{ color: "var(--text-muted)" }}>Patenteabilidade:</span>
            <button onClick={() => setPatentableOnly(true)}
              className="px-2.5 py-1 rounded-full text-xs transition-colors"
              style={{
                background: patentableOnly ? "#34d39920" : "var(--surface-2)",
                color: patentableOnly ? "#34d399" : "var(--text-muted)",
                border: `1px solid ${patentableOnly ? "#34d39960" : "var(--border)"}`,
              }}>
              ✓ Só patenteáveis (Art. 8)
            </button>
            <button onClick={() => setPatentableOnly(false)}
              className="px-2.5 py-1 rounded-full text-xs transition-colors"
              style={{
                background: !patentableOnly ? "var(--accent)" : "var(--surface-2)",
                color: !patentableOnly ? "white" : "var(--text-muted)",
                border: "1px solid var(--border)",
              }}>
              Mostrar todos (inclui Art. 10)
            </button>
          </div>

          {/* Filtro por nível */}
          <div className="flex items-center gap-2 flex-wrap">
            <Filter size={11} className="text-slate-500" />
            <span className="text-xs" style={{ color: "var(--text-muted)" }}>Nível:</span>
            {(["all", "high", "medium", "low"] as const).map(l => (
              <button key={l} onClick={() => setLevelFilter(l)}
                className="px-2.5 py-1 rounded-full text-xs transition-colors"
                style={{
                  background: levelFilter === l ? "var(--accent)" : "var(--surface-2)",
                  color: levelFilter === l ? "white" : "var(--text-muted)",
                  border: "1px solid var(--border)",
                }}>
                {l === "all" ? "Todos" : l === "high" ? "Alta" : l === "medium" ? "Média" : "Baixa"}
              </button>
            ))}
          </div>

          {/* Filtro por área */}
          <div className="flex items-center gap-2 flex-wrap">
            <Filter size={11} className="text-slate-500" />
            <span className="text-xs" style={{ color: "var(--text-muted)" }}>Área:</span>
            <button onClick={() => setAreaFilter("all")}
              className="px-2.5 py-1 rounded-full text-xs transition-colors"
              style={{
                background: areaFilter === "all" ? "var(--accent)" : "var(--surface-2)",
                color: areaFilter === "all" ? "white" : "var(--text-muted)",
                border: "1px solid var(--border)",
              }}>
              Todas
            </button>
            {KNOWN_AREAS.map(a => {
              const active = areaFilter === a.match;
              return (
                <button key={a.key} onClick={() => setAreaFilter(active ? "all" : a.match)}
                  className="px-2.5 py-1 rounded-full text-xs transition-colors"
                  style={{
                    background: active ? "#a855f7" : "var(--surface-2)",
                    color: active ? "white" : "var(--text-muted)",
                    border: `1px solid ${active ? "#a855f7" : "var(--border)"}`,
                  }}>
                  {a.key}
                </button>
              );
            })}
          </div>
        </div>
      </Card>

      {/* Lista de oportunidades — 100% width (sem sidebar de notícias mock) */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-semibold text-white">
            Oportunidades de PI detectadas
            <span className="ml-2 text-xs font-normal" style={{ color: "var(--text-muted)" }}>
              · página {page + 1} de {pageCount} · {total} no filtro
            </span>
          </h2>
          <PaginationControls page={page} pageCount={pageCount} onChange={setPage} />
        </div>

        {loading && <SkeletonList count={3} />}

        {!loading && opportunities.length === 0 && (
          <Card>
            <EmptyState
              icon={GraduationCap}
              title="Nenhuma oportunidade no filtro atual"
              description="Tente outro nível ou área."
              size="sm"
            />
          </Card>
        )}

        <div className="space-y-3">
          {opportunities.map(opp => (
            <OpportunityCard key={opp.id} opp={opp} onStatus={handleStatus} />
          ))}
        </div>

        {!loading && opportunities.length > 0 && (
          <div className="flex items-center justify-between mt-4 pt-3"
            style={{ borderTop: "1px solid var(--border)" }}>
            <span className="text-xs" style={{ color: "var(--text-muted)" }}>
              Mostrando {page * PAGE_SIZE + 1}–{Math.min((page + 1) * PAGE_SIZE, total)} de {total}
            </span>
            <PaginationControls page={page} pageCount={pageCount} onChange={setPage} />
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Paginação ──────────────────────────────────────────────────────────────

function PaginationControls({
  page, pageCount, onChange,
}: { page: number; pageCount: number; onChange: (p: number) => void }) {
  if (pageCount <= 1) return null;
  return (
    <div className="flex items-center gap-1">
      <Button variant="ghost" size="sm" onClick={() => onChange(0)} disabled={page === 0}>
        «
      </Button>
      <Button variant="ghost" size="sm" onClick={() => onChange(page - 1)} disabled={page === 0}>
        ‹ Anterior
      </Button>
      <span className="text-xs px-2" style={{ color: "var(--text-muted)" }}>
        {page + 1} / {pageCount}
      </span>
      <Button variant="ghost" size="sm" onClick={() => onChange(page + 1)} disabled={page >= pageCount - 1}>
        Próximo ›
      </Button>
      <Button variant="ghost" size="sm" onClick={() => onChange(pageCount - 1)} disabled={page >= pageCount - 1}>
        »
      </Button>
    </div>
  );
}

// ─── Card de oportunidade (clicável → leva pra contrato TT) ─────────────────

function OpportunityCard({ opp, onStatus }: {
  opp: UFOPOpportunity;
  onStatus: (id: number, status: UFOPStatus) => void;
}) {
  const [open, setOpen] = useState(false);
  const { label, variant, icon } = levelInfo(opp.opportunity_level);
  const firstAuthor = opp.authors?.[0] ?? "—";
  const nonPatentable = opp.is_patentable === false;
  const borderColor =
    nonPatentable                      ? "#6b728030" :
    opp.opportunity_level === "high"   ? "#ef444430" :
    opp.opportunity_level === "medium" ? "#f59e0b30" : "var(--border)";

  return (
    <Card style={{ borderColor, opacity: nonPatentable ? 0.78 : 1 }}>
      {/* Top row — título + badges + status */}
      <div className="flex items-start justify-between gap-3 mb-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1.5 flex-wrap">
            {nonPatentable ? (
              <span className="text-xs px-2 py-0.5 rounded-full"
                style={{ background: "#6b728020", color: "#94a3b8", border: "1px solid #6b728060" }}>
                ⚖ Não-patenteável (Art. 10 LPI)
              </span>
            ) : (
              <Badge variant={variant}>{icon} {label}</Badge>
            )}
            <Badge variant="muted">{sourceLabel(opp.source)}</Badge>
            {opp.department && <Badge variant="info">{opp.department}</Badge>}
            <Badge variant="muted">IPC: {opp.ipc_suggestion}</Badge>
            {!nonPatentable && (
              <span className="text-xs font-semibold"
                style={{ color: opp.pi_score >= 5.5 ? "#34d399" : opp.pi_score >= 3 ? "#fbbf24" : "var(--text-muted)" }}>
                PI Score {opp.pi_score.toFixed(1)}
              </span>
            )}
            {opp.classifier_version && (
              <span className="text-[10px] px-1.5 py-0.5 rounded font-mono"
                style={{ background: "var(--surface-2)", color: "var(--text-muted)" }}
                title={opp.confidence ? `Confiança: ${(opp.confidence * 100).toFixed(0)}%` : undefined}>
                {opp.classifier_version}
              </span>
            )}
          </div>
          <p className="text-base font-semibold text-white leading-snug">{opp.title}</p>
          <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>
            {firstAuthor}
            {opp.published_at ? ` · ${formatDate(opp.published_at)}` : ""}
          </p>
          {opp.rationale && (
            <p className="text-xs mt-1.5 italic" style={{ color: nonPatentable ? "#94a3b8" : "var(--text-muted)" }}>
              → {opp.rationale}
            </p>
          )}
        </div>
        <StatusChip status={opp.status} />
      </div>

      {/* Expandido — abstract + AI analysis */}
      {open && (
        <div className="space-y-2 mb-3">
          {opp.abstract && (
            <p className="text-sm leading-relaxed" style={{ color: "var(--text-muted)" }}>
              {opp.abstract}
            </p>
          )}
          {opp.ai_analysis && opp.ai_analysis !== opp.rationale && (
            <div className="rounded-lg p-3 text-sm"
              style={{ background: "var(--surface-2)", color: "#a5b4fc" }}>
              🤖 {opp.ai_analysis}
            </div>
          )}
        </div>
      )}

      {/* Ações — agora destacando o fluxo de contrato TT */}
      <div className="flex gap-2 flex-wrap">
        <Button variant="ghost" size="sm" onClick={() => setOpen(o => !o)}>
          {open ? "Recolher" : "Ver resumo + análise IA"}
        </Button>

        {/* Contrato TT só faz sentido pra patenteáveis */}
        {!nonPatentable && (
          <Link href={`/tt-contract/new?from_ufop=${opp.id}`}>
            <Button size="sm">
              <FileSignature size={12} />
              Gerar contrato TT
            </Button>
          </Link>
        )}

        {opp.url && (
          <a href={opp.url} target="_blank" rel="noopener noreferrer">
            <Button variant="ghost" size="sm">
              <ExternalLink size={12} />
              Fonte no repositório
            </Button>
          </a>
        )}

        {opp.status === "new" && (
          <>
            <Button variant="ghost" size="sm"
              onClick={() => onStatus(opp.id, "reviewed")}
              style={{ color: "#34d399" }}>
              <CheckCircle2 size={12} /> Revisar
            </Button>
            <Button variant="ghost" size="sm"
              onClick={() => onStatus(opp.id, "dismissed")}
              style={{ color: "#f87171" }}>
              <XCircle size={12} /> Descartar
            </Button>
          </>
        )}
      </div>
    </Card>
  );
}

function StatusChip({ status }: { status: UFOPStatus }) {
  const map: Record<UFOPStatus, { label: string; color: string }> = {
    new:       { label: "Novo",       color: "#6366f1" },
    reviewed:  { label: "Revisado",   color: "#22c55e" },
    converted: { label: "Convertido", color: "#a78bfa" },
    dismissed: { label: "Descartado", color: "#6b7280" },
  };
  const { label, color } = map[status] ?? map.new;
  return (
    <span className="text-xs px-2 py-0.5 rounded-full font-medium shrink-0"
      style={{ background: color + "22", color }}>
      {label}
    </span>
  );
}
