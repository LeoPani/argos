"use client";

import { useState } from "react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { mockUFOPNews } from "@/lib/mock-data";
import { formatDate } from "@/lib/utils";
import type { UFOPOpportunity, OpportunityLevel, UFOPStatus } from "@/lib/types";
import { useUFOPOpportunities } from "@/lib/hooks";
import { api } from "@/lib/api";
import { SkeletonKPI, SkeletonList } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import {
  GraduationCap, Flame, Minus, TrendingDown,
  ExternalLink, Search, RefreshCw, CheckCircle2,
  XCircle, ArrowRightCircle,
} from "lucide-react";

// ─── helpers ─────────────────────────────────────────────────────────────────

function levelInfo(l: OpportunityLevel): {
  label: string;
  variant: "danger" | "warning" | "muted";
  icon: React.ReactNode;
} {
  const map: Record<OpportunityLevel, { label: string; variant: "danger" | "warning" | "muted"; icon: React.ReactNode }> = {
    high:   { label: "Alta oportunidade",  variant: "danger",  icon: <Flame size={12} /> },
    medium: { label: "Oportunidade média", variant: "warning", icon: <Minus size={12} /> },
    low:    { label: "Baixa prioridade",   variant: "muted",   icon: <TrendingDown size={12} /> },
  };
  return map[l];
}

function sourceLabel(s: UFOPOpportunity["source"]): string {
  const labels: Record<string, string> = {
    oai:    "RI-UFOP",
    portal: "Portal UFOP",
    lens:   "Lens.org",
  };
  return labels[s] ?? s;
}

// Mock opportunities shown when backend is offline
const MOCK_OPPORTUNITIES: UFOPOpportunity[] = [
  {
    id: 1,
    source: "oai",
    external_id: "oai:repositorio.ufop.br:1/demo-1",
    title: "Processo de biorremediação de solos contaminados com metais pesados via consórcio microbiano",
    authors: ["Prof. Dr. Carlos Henrique Silva", "Dra. Ana Costa"],
    department: "Departamento de Química — UFOP",
    abstract:
      "Método biológico para tratamento de solos com metais pesados oriundos de atividades de mineração, "
      + "utilizando consórcio de microrganismos selecionados. Apresenta eficiência superior a 85% na remoção de chumbo e arsênio.",
    url: "https://repositorio.ufop.br/handle/1/demo-1",
    published_at: "2026-03-15T00:00:00Z",
    ipc_suggestion: "C — Química e Metalurgia",
    ipc_category: 2,
    opportunity_level: "high",
    similarity_pct: 28,
    pi_score: 8.2,
    ai_analysis:
      "A publicação apresenta alto potencial de patenteabilidade na categoria IPC C — Química e Metalurgia. "
      + "Foram identificados 7 indicadores de PI no título e resumo. Recomenda-se iniciar imediatamente "
      + "uma consulta de anterioridade e avaliar o depósito de pedido de patente junto ao INPI.",
    status: "new",
    publication_id: null,
    created_at: "2026-05-23T00:00:00Z",
    updated_at: "2026-05-23T00:00:00Z",
  },
  {
    id: 2,
    source: "oai",
    external_id: "oai:repositorio.ufop.br:1/demo-2",
    title: "Sistema de controle inteligente para distribuição de energia em microrredes rurais",
    authors: ["Eng. Roberto Rocha", "Profa. Dra. Mariana Lima"],
    department: "Departamento de Engenharia Elétrica — UFOP",
    abstract:
      "Sistema embarcado de controle adaptativo para otimização de fluxo de energia em microrredes isoladas. "
      + "Algoritmo proprietário reduz perdas em até 30% e melhora resiliência a falhas.",
    url: "https://repositorio.ufop.br/handle/1/demo-2",
    published_at: "2026-01-22T00:00:00Z",
    ipc_suggestion: "H — Eletricidade e Eletrônica",
    ipc_category: 7,
    opportunity_level: "high",
    similarity_pct: 35,
    pi_score: 7.6,
    ai_analysis:
      "A publicação apresenta alto potencial de patenteabilidade na categoria IPC H — Eletricidade e Eletrônica. "
      + "Foram identificados 6 indicadores de PI. Recomenda-se iniciar imediatamente uma consulta de anterioridade.",
    status: "new",
    publication_id: null,
    created_at: "2026-05-23T00:00:00Z",
    updated_at: "2026-05-23T00:00:00Z",
  },
  {
    id: 3,
    source: "portal",
    external_id: "https://www.ufop.br/noticias/demo-3",
    title: "Pesquisadores da UFOP desenvolvem método inovador de síntese de nanomateriais para aplicações biomédicas",
    authors: [],
    department: "Instituto de Ciências Exatas e Biológicas",
    abstract:
      "Equipe do ICEB desenvolveu técnica de síntese de nanopartículas de óxido de ferro com alto grau de pureza "
      + "e biocompatibilidade, abrindo caminho para aplicações em diagnóstico por imagem e terapia direcionada.",
    url: "https://www.ufop.br/noticias/demo-3",
    published_at: "2026-04-10T00:00:00Z",
    ipc_suggestion: "C — Química e Metalurgia",
    ipc_category: 2,
    opportunity_level: "medium",
    similarity_pct: 45,
    pi_score: 5.1,
    ai_analysis:
      "A publicação apresenta potencial moderado de patenteabilidade. "
      + "Recomenda-se análise mais aprofundada de novidade antes de decidir pelo depósito.",
    status: "reviewed",
    publication_id: null,
    created_at: "2026-05-23T00:00:00Z",
    updated_at: "2026-05-23T00:00:00Z",
  },
];

// ─── main component ───────────────────────────────────────────────────────────

export default function UFOPPage() {
  const [expanded, setExpanded] = useState<number | null>(null);
  const [filter, setFilter] = useState<OpportunityLevel | "all">("all");

  const { data, error, isLoading, mutate } = useUFOPOpportunities(
    filter !== "all" ? { level: filter } : undefined
  );

  const isLive  = !error && !!data;
  const loading = isLoading && !data && !error;
  const opportunities: UFOPOpportunity[] = isLive
    ? data!.items
    : MOCK_OPPORTUNITIES.filter(o => filter === "all" || o.opportunity_level === filter);
  const total = isLive ? data!.pagination.total : MOCK_OPPORTUNITIES.length;

  const highCount  = opportunities.filter(o => o.opportunity_level === "high").length;
  const newCount   = opportunities.filter(o => o.status === "new").length;
  const converted  = opportunities.filter(o => o.status === "converted").length;

  async function handleStatus(id: number, status: UFOPStatus) {
    try {
      await api.ufop.updateStatus(id, status);
      mutate();
    } catch {
      // silent — still works on mock
    }
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
            Monitoramento de publicações e oportunidades de PI da Universidade Federal de Ouro Preto
          </p>
        </div>
        <div className="flex items-center gap-2">
          {isLive ? (
            <span className="text-xs text-emerald-400 flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse inline-block" />
              dados ao vivo
            </span>
          ) : (
            <span className="text-xs text-amber-400">modo offline</span>
          )}
          <Button variant="ghost" size="sm" onClick={() => mutate()}>
            <RefreshCw size={13} />
            Atualizar
          </Button>
        </div>
      </div>

      {/* Source status chips */}
      <div className="flex gap-3 flex-wrap">
        {[
          { label: "Portal UFOP", key: "portal" },
          { label: "RI-UFOP (OAI)", key: "oai" },
          { label: "Lens.org/UFOP", key: "lens" },
        ].map(s => (
          <div key={s.label} className="flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs"
            style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
            <div className={`w-1.5 h-1.5 rounded-full ${isLive ? "bg-emerald-400" : "bg-amber-400"}`} />
            <span className="text-white">{s.label}</span>
          </div>
        ))}
      </div>

      {/* KPIs */}
      <div className="grid grid-cols-4 gap-4">
        {[
          { label: "Total de oportunidades", value: total.toString() },
          { label: "Alta prioridade", value: highCount.toString() },
          { label: "Aguardando revisão", value: newCount.toString() },
          { label: "Convertidas em consulta", value: converted.toString() },
        ].map(({ label, value }) => (
          <Card key={label}>
            <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>{label}</p>
            <p className="text-2xl font-bold text-white">{value}</p>
          </Card>
        ))}
      </div>

      {/* Level filter */}
      <div className="flex gap-2">
        {(["all", "high", "medium", "low"] as const).map(l => (
          <button key={l}
            onClick={() => setFilter(l)}
            className="px-3 py-1.5 rounded-lg text-xs transition-colors"
            style={{
              background: filter === l ? "var(--accent)" : "var(--surface)",
              color: filter === l ? "#fff" : "var(--text-muted)",
              border: "1px solid var(--border)",
            }}>
            {l === "all" ? "Todos" : l === "high" ? "Alta prioridade" : l === "medium" ? "Média" : "Baixa"}
          </button>
        ))}
      </div>

      <div className="grid grid-cols-3 gap-6">
        {/* Opportunities list */}
        <div className="col-span-2 space-y-4">
          <h2 className="text-sm font-semibold text-white">
            Oportunidades de PI detectadas pela IA
            <span className="ml-2 text-xs font-normal" style={{ color: "var(--text-muted)" }}>
              ({opportunities.length} resultados)
            </span>
          </h2>

          {loading && <SkeletonList count={3} />}

          {!loading && opportunities.length === 0 && (
            <Card>
              <EmptyState
                icon={GraduationCap}
                title={filter === "all"
                  ? "Nenhuma oportunidade ainda"
                  : `Nenhuma oportunidade ${filter === "high" ? "alta" : filter === "medium" ? "média" : "baixa"}`}
                description={filter === "all"
                  ? "Rode o harvest UFOP (ScrapePortal + HarvestOAI) ou use `make seed` para popular."
                  : "Tente outro nível ou volte para 'Todos'."}
                size="sm"
              />
            </Card>
          )}

          {opportunities.map(opp => {
            const { label, variant, icon } = levelInfo(opp.opportunity_level);
            const isOpen = expanded === opp.id;
            const firstAuthor = opp.authors?.[0] ?? "—";

            return (
              <Card key={opp.id}
                style={{ borderColor: opp.opportunity_level === "high" ? "#ef444430" : opp.opportunity_level === "medium" ? "#f59e0b30" : "var(--border)" }}>

                {/* Top row */}
                <div className="flex items-start justify-between gap-3 mb-3">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1.5 flex-wrap">
                      <Badge variant={variant}>{icon} {label}</Badge>
                      <Badge variant="muted">{sourceLabel(opp.source)}</Badge>
                      <Badge variant="info">IPC: {opp.ipc_suggestion}</Badge>
                      <Badge variant="muted">{opp.similarity_pct}% similar</Badge>
                      <span className="text-xs font-semibold"
                        style={{ color: opp.pi_score >= 6 ? "#34d399" : opp.pi_score >= 3.5 ? "#fbbf24" : "var(--text-muted)" }}>
                        PI Score {opp.pi_score.toFixed(1)}
                      </span>
                    </div>
                    <p className="text-base font-semibold text-white leading-snug">{opp.title}</p>
                    <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>
                      {firstAuthor}
                      {opp.department ? ` · ${opp.department}` : ""}
                      {opp.published_at ? ` · ${formatDate(opp.published_at)}` : ""}
                    </p>
                  </div>

                  {/* Status badge */}
                  <StatusChip status={opp.status} />
                </div>

                {/* Expanded: abstract + AI analysis */}
                {isOpen && (
                  <div className="space-y-2 mb-3">
                    {opp.abstract && (
                      <p className="text-sm leading-relaxed" style={{ color: "var(--text-muted)" }}>
                        {opp.abstract}
                      </p>
                    )}
                    {opp.ai_analysis && (
                      <div className="rounded-lg p-3 text-sm"
                        style={{ background: "var(--surface-2)", color: "#a5b4fc" }}>
                        🤖 {opp.ai_analysis}
                      </div>
                    )}
                  </div>
                )}

                {/* Actions */}
                <div className="flex gap-2 flex-wrap">
                  <Button variant="ghost" size="sm" onClick={() => setExpanded(isOpen ? null : opp.id)}>
                    {isOpen ? "Recolher" : "Ver resumo + análise IA"}
                  </Button>
                  <Button variant="secondary" size="sm">
                    <Search size={12} />
                    Consulta de anterioridade
                  </Button>
                  {opp.url && (
                    <a href={opp.url} target="_blank" rel="noopener noreferrer">
                      <Button variant="ghost" size="sm">
                        <ExternalLink size={12} />
                        Fonte
                      </Button>
                    </a>
                  )}
                  {opp.status === "new" && (
                    <>
                      <Button variant="ghost" size="sm"
                        onClick={() => handleStatus(opp.id, "reviewed")}
                        style={{ color: "#34d399" }}>
                        <CheckCircle2 size={12} />
                        Revisar
                      </Button>
                      <Button variant="ghost" size="sm"
                        onClick={() => handleStatus(opp.id, "dismissed")}
                        style={{ color: "#f87171" }}>
                        <XCircle size={12} />
                        Descartar
                      </Button>
                    </>
                  )}
                  {opp.status === "reviewed" && (
                    <Button variant="ghost" size="sm"
                      onClick={() => handleStatus(opp.id, "converted")}
                      style={{ color: "#818cf8" }}>
                      <ArrowRightCircle size={12} />
                      Converter em consulta
                    </Button>
                  )}
                </div>
              </Card>
            );
          })}
        </div>

        {/* News sidebar */}
        <div className="space-y-4">
          <h2 className="text-sm font-semibold text-white">Notícias UFOP com keywords de PI</h2>
          <div className="space-y-3">
            {mockUFOPNews.map(news => (
              <Card key={news.id} className="p-4">
                <p className="text-sm font-medium text-white leading-snug mb-2">{news.title}</p>
                <p className="text-xs mb-2" style={{ color: "var(--text-muted)" }}>{formatDate(news.date)}</p>
                <div className="flex flex-wrap gap-1 mb-2">
                  {news.pi_keywords.map(kw => (
                    <span key={kw} className="text-xs px-1.5 py-0.5 rounded"
                      style={{ background: "var(--surface-2)", color: "var(--text-muted)" }}>
                      #{kw}
                    </span>
                  ))}
                </div>
                <a href={news.url} className="text-xs text-indigo-400 flex items-center gap-1 hover:text-indigo-300">
                  <ExternalLink size={11} />
                  Ler na íntegra
                </a>
              </Card>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

// ─── StatusChip ───────────────────────────────────────────────────────────────

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
