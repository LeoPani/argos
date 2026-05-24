"use client";

import { useState } from "react";
import Link from "next/link";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton, SkeletonKPI, SkeletonTable } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { mockAIOpportunities, mockCostTimeline, mockPortfolioAssets } from "@/lib/mock-data";
import { formatBRL, formatDate, daysUntil, cn } from "@/lib/utils";
import type { PortfolioAsset, AIOpportunity, CostPoint } from "@/lib/types";
import { usePortfolio } from "@/lib/hooks";
import {
  AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer,
} from "recharts";
import {
  TrendingUp, AlertCircle, Search, Plus, Link2,
  FileText, Tag, Cpu, Package, Lightbulb, ShieldAlert,
  RefreshCw, Download,
} from "lucide-react";
import { toCSV, downloadCSV, csvDate } from "@/lib/csv";
import { useToast } from "@/components/ui/toast";

type Tab = "own" | "third";

// ─── icon / badge maps ────────────────────────────────────────────────────────

const assetTypeIcon: Record<string, React.ReactNode> = {
  PI: <FileText size={14} className="text-indigo-400" />,
  MU: <Package size={14} className="text-purple-400" />,
  TM: <Tag size={14} className="text-orange-400" />,
  DP: <Cpu size={14} className="text-blue-400" />,
};

const assetTypeBadge: Record<string, "default" | "info" | "warning" | "muted"> = {
  PI: "default", MU: "info", TM: "warning", DP: "muted",
};

function statusBadge(status: PortfolioAsset["status"]) {
  const map = {
    active:    { variant: "success"  as const, label: "Ativa" },
    pending:   { variant: "warning"  as const, label: "Pendente" },
    expired:   { variant: "muted"    as const, label: "Expirada" },
    opposition:{ variant: "danger"   as const, label: "Oposição" },
  };
  const { variant, label } = map[status] ?? map.pending;
  return <Badge variant={variant}>{label}</Badge>;
}

function DeadlinePill({ date }: { date: string | null }) {
  const days = daysUntil(date);
  if (days === null) return <span style={{ color: "var(--text-muted)" }}>—</span>;
  if (days < 0)   return <Badge variant="muted">Vencida</Badge>;
  if (days <= 30) return <Badge variant="danger">⚠ {days}d</Badge>;
  if (days <= 90) return <Badge variant="warning">{days}d</Badge>;
  return <Badge variant="success">{formatDate(date)}</Badge>;
}

// ─── main page ────────────────────────────────────────────────────────────────

export default function PortfolioPage() {
  const [tab, setTab]                 = useState<Tab>("own");
  const [searchQuery, setSearchQuery] = useState("");
  const [thirdPartyQuery, setThirdPartyQuery] = useState("");
  const [loading, setLoading]         = useState(false);
  const [thirdPartyResult, setThirdPartyResult] = useState(false);

  const { data, error, isLoading, mutate } = usePortfolio();
  const toast = useToast();

  const isLive  = !error && !!data;
  const portfolioLoading = isLoading && !data && !error;

  function handleExport() {
    if (!data || data.assets.length === 0) {
      toast.warning("Nada para exportar", "O portfolio está vazio.");
      return;
    }
    const csv = toCSV(
      data.assets.map(a => ({
        ...a,
        filing_date:  csvDate(a.filing_date),
        expiry_date:  csvDate(a.expiry_date),
        next_fee_date: csvDate(a.next_fee_date),
      })),
      [
        { key: "type",         label: "Tipo" },
        { key: "number",       label: "Número" },
        { key: "title",        label: "Título" },
        { key: "owner",        label: "Titular" },
        { key: "status",       label: "Status" },
        { key: "ipc_code",     label: "IPC" },
        { key: "filing_date",  label: "Depósito" },
        { key: "expiry_date",  label: "Expiração" },
        { key: "next_fee_date",label: "Próxima taxa" },
        { key: "cost_monthly", label: "Custo mensal (R$)" },
        { key: "cost_annual",  label: "Custo anual (R$)" },
        { key: "cost_total",   label: "Custo total (R$)" },
      ],
    );
    downloadCSV(csv, `argos-portfolio-${new Date().toISOString().slice(0, 10)}.csv`);
    toast.success("Portfolio exportado", `${data.assets.length} ativos baixados como CSV.`);
  }

  const assets: PortfolioAsset[]       = isLive ? data!.assets          : mockPortfolioAssets;
  const timeline: CostPoint[]          = isLive ? data!.cost_timeline    : mockCostTimeline;
  const aiOpps: AIOpportunity[]        = isLive ? data!.ai_opportunities : mockAIOpportunities;
  const summary = isLive
    ? data!.cost_summary
    : {
        monthly: mockPortfolioAssets.reduce((s, a) => s + a.cost_monthly, 0),
        annual:  mockPortfolioAssets.reduce((s, a) => s + a.cost_annual, 0),
        total:   mockPortfolioAssets.reduce((s, a) => s + a.cost_total, 0),
      };

  const filtered = assets.filter(
    a => !searchQuery || a.title.toLowerCase().includes(searchQuery.toLowerCase())
  );

  async function estimateThirdParty() {
    if (!thirdPartyQuery.trim()) return;
    setLoading(true);
    await new Promise(r => setTimeout(r, 1600));
    setThirdPartyResult(true);
    setLoading(false);
  }

  return (
    <div className="p-8 space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Portfolio de PI</h1>
          <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
            Gestão de ativos, prazos, custos e oportunidades detectadas por IA
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
            <RefreshCw size={13} /> Atualizar
          </Button>
          <Button variant="ghost" size="sm" onClick={handleExport}>
            <Download size={13} /> Exportar CSV
          </Button>
          <Button size="sm">
            <Plus size={14} /> Adicionar ativo
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 p-1 rounded-lg w-fit" style={{ background: "var(--surface)" }}>
        {(["own", "third"] as Tab[]).map(t => (
          <button key={t} onClick={() => setTab(t)}
            className={cn("px-4 py-2 rounded-md text-sm transition-all", tab === t ? "text-white font-medium" : "hover:text-white")}
            style={{
              background: tab === t ? "var(--accent)" : "transparent",
              color: tab === t ? "white" : "var(--text-muted)",
            }}>
            {t === "own" ? "Meu Portfolio" : "Estimar Terceiro"}
          </button>
        ))}
      </div>

      {tab === "own" ? (
        <>
          {/* KPIs */}
          <div className="grid grid-cols-4 gap-4">
            {portfolioLoading ? (
              <><SkeletonKPI /><SkeletonKPI /><SkeletonKPI /><SkeletonKPI /></>
            ) : (
              <>
                <Card>
                  <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>Total de ativos</p>
                  <p className="text-2xl font-bold text-white">{assets.length}</p>
                  <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>
                    {assets.filter(a => a.type === "PI" || a.type === "MU").length} patentes ·{" "}
                    {assets.filter(a => a.type === "TM").length} marcas
                  </p>
                </Card>
                {[
                  { label: "Custo Mensal",          value: summary.monthly, sub: "estimado" },
                  { label: "Custo Anual",           value: summary.annual,  sub: "estimado" },
                  { label: "Custo Total (período)", value: summary.total,   sub: "até vencimento" },
                ].map(({ label, value, sub }) => (
                  <Card key={label}>
                    <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>{label}</p>
                    <p className="text-2xl font-bold text-white">{formatBRL(value)}</p>
                    <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>{sub}</p>
                  </Card>
                ))}
              </>
            )}
          </div>

          {/* Cost timeline */}
          <Card>
            <CardHeader>
              <CardTitle>Projeção de custos — anuidades INPI (5 anos)</CardTitle>
              <div className="flex gap-3 text-xs">
                <span className="flex items-center gap-1">
                  <span className="w-2 h-2 rounded-full bg-indigo-400" />
                  Anuidades patentes + renovação marcas
                </span>
              </div>
            </CardHeader>
            <ResponsiveContainer width="100%" height={160}>
              <AreaChart data={timeline}>
                <defs>
                  <linearGradient id="pg" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%"  stopColor="#6366f1" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#6366f1" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <XAxis dataKey="year" tick={{ fontSize: 11 }} />
                <YAxis tick={{ fontSize: 11 }} tickFormatter={v => `R$${(v / 1000).toFixed(0)}k`} />
                <Tooltip formatter={(v) => [`R$ ${Number(v).toLocaleString("pt-BR")}`, "Custo"]} />
                <Area type="monotone" dataKey="value" stroke="#6366f1" fill="url(#pg)" strokeWidth={2} />
              </AreaChart>
            </ResponsiveContainer>
          </Card>

          {/* Assets table */}
          <Card>
            <CardHeader>
              <CardTitle>Ativos ({filtered.length})</CardTitle>
              <div className="relative">
                <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-500" />
                <input
                  value={searchQuery}
                  onChange={e => setSearchQuery(e.target.value)}
                  placeholder="Buscar ativo..."
                  className="pl-8 pr-3 py-1.5 rounded-lg text-xs outline-none"
                  style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }}
                />
              </div>
            </CardHeader>

            {portfolioLoading ? (
              <SkeletonTable rows={5} cols={8} />
            ) : assets.length === 0 ? (
              <EmptyState
                icon={AlertCircle}
                title="Nenhum ativo no portfolio ainda"
                description="Cadastre patentes via POST /api/v1/patents ou rode `make seed` para popular o banco com dados de demonstração."
              />
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr style={{ borderBottom: "1px solid var(--border)" }}>
                      {["Tipo", "Número / Título", "Status", "Vencimento", "Próx. taxa", "Mensal", "Anual", "Total", ""].map(h => (
                        <th key={h} className="text-left pb-2 pr-3 text-xs font-medium" style={{ color: "var(--text-muted)" }}>{h}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {filtered.map(asset => {
                      // Extract numeric id from "pat-{N}" or "tm-{N}" prefix
                      const numericID = Number(asset.id.split("-")[1]);
                      const detailURL =
                        asset.type === "PI" || asset.type === "MU"
                          ? `/patents/${numericID}`
                          : asset.type === "TM"
                            ? `/trademarks/${numericID}`
                            : null;
                      const TitleCell = detailURL
                        ? ({ children }: { children: React.ReactNode }) =>
                            <Link href={detailURL} className="hover:underline">{children}</Link>
                        : ({ children }: { children: React.ReactNode }) =>
                            <span>{children}</span>;
                      return (
                      <tr key={asset.id} style={{ borderBottom: "1px solid var(--border)" }} className="hover:bg-white/5">
                        <td className="py-3 pr-3">
                          <div className="flex items-center gap-1.5">
                            {assetTypeIcon[asset.type]}
                            <Badge variant={assetTypeBadge[asset.type]}>{asset.type}</Badge>
                          </div>
                        </td>
                        <td className="py-3 pr-3">
                          <TitleCell>
                            <p className="text-white font-medium text-xs max-w-[180px] truncate">{asset.title}</p>
                            <p className="font-mono text-xs text-indigo-400">{asset.number}</p>
                          </TitleCell>
                          {asset.ipc_code && (
                            <p className="text-xs mt-0.5" style={{ color: "var(--text-muted)" }}>IPC: {asset.ipc_code}</p>
                          )}
                        </td>
                        <td className="py-3 pr-3">{statusBadge(asset.status)}</td>
                        <td className="py-3 pr-3 text-xs" style={{ color: "var(--text-muted)" }}>
                          {formatDate(asset.expiry_date || null)}
                        </td>
                        <td className="py-3 pr-3"><DeadlinePill date={asset.next_fee_date} /></td>
                        <td className="py-3 pr-3 text-xs text-white">{formatBRL(asset.cost_monthly)}</td>
                        <td className="py-3 pr-3 text-xs text-white">{formatBRL(asset.cost_annual)}</td>
                        <td className="py-3 pr-3 text-xs text-white">{formatBRL(asset.cost_total)}</td>
                        <td className="py-3">
                          {asset.blockchain_hash && (
                            <span title={asset.blockchain_hash}>
                              <Link2 size={12} className="text-indigo-400" />
                            </span>
                          )}
                        </td>
                      </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </Card>

          {/* AI Opportunities */}
          <div>
            <h2 className="text-base font-semibold text-white mb-3 flex items-center gap-2">
              <Lightbulb size={16} className="text-amber-400" />
              Oportunidades sugeridas pela IA
              {isLive && aiOpps.length > 0 && (
                <Badge variant="warning">{aiOpps.length} nova{aiOpps.length > 1 ? "s" : ""}</Badge>
              )}
            </h2>
            {aiOpps.length === 0 ? (
              <Card>
                <p className="text-sm text-center py-4" style={{ color: "var(--text-muted)" }}>
                  Nenhuma oportunidade de alta prioridade detectada ainda.
                  Execute o harvest UFOP para popular.
                </p>
              </Card>
            ) : (
              <div className="grid grid-cols-1 gap-3">
                {aiOpps.map(opp => (
                  <OpportunityCard key={opp.id} opp={opp} />
                ))}
              </div>
            )}
          </div>
        </>
      ) : (
        <ThirdPartyEstimator
          query={thirdPartyQuery}
          setQuery={setThirdPartyQuery}
          loading={loading}
          onSearch={estimateThirdParty}
          hasResult={thirdPartyResult}
        />
      )}
    </div>
  );
}

// ─── sub-components ───────────────────────────────────────────────────────────

function OpportunityCard({ opp }: { opp: AIOpportunity }) {
  return (
    <Card style={{ borderColor: opp.type === "opportunity" ? "#6366f130" : "#ef444430" }}>
      <div className="flex items-start gap-3">
        <div className={cn("p-2 rounded-lg mt-0.5 shrink-0",
          opp.type === "opportunity" ? "bg-amber-500/20" : "bg-red-500/20")}>
          {opp.type === "opportunity"
            ? <TrendingUp  size={15} className="text-amber-400" />
            : <ShieldAlert size={15} className="text-red-400" />}
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1 flex-wrap">
            <p className="text-sm font-semibold text-white">{opp.title}</p>
            <Badge variant="muted">{opp.confidence}% confiança</Badge>
            {opp.ipc_class && <Badge variant="info">IPC: {opp.ipc_class}</Badge>}
          </div>
          <p className="text-xs leading-relaxed" style={{ color: "var(--text-muted)" }}>{opp.description}</p>
          {opp.estimated_cost && (
            <p className="text-xs mt-1 text-indigo-400">
              Custo estimado de depósito: {formatBRL(opp.estimated_cost)}
            </p>
          )}
        </div>
        <Button variant="secondary" size="sm">{opp.action_label}</Button>
      </div>
    </Card>
  );
}

function ThirdPartyEstimator({
  query, setQuery, loading, onSearch, hasResult,
}: {
  query: string; setQuery: (v: string) => void;
  loading: boolean; onSearch: () => void; hasResult: boolean;
}) {
  const distribution = [
    { area: "Exploração de petróleo",    pct: 38, count: 322 },
    { area: "Refino e processos",        pct: 27, count: 229 },
    { area: "Energias renováveis",       pct: 18, count: 153, growing: true },
    { area: "Meio ambiente",             pct: 17, count: 143 },
  ];

  return (
    <div className="space-y-4">
      <Card>
        <div className="space-y-4">
          <label className="text-xs font-medium block" style={{ color: "var(--text-muted)" }}>
            Titular / Empresa a estimar
          </label>
          <div className="flex gap-3">
            <input
              value={query}
              onChange={e => setQuery(e.target.value)}
              placeholder="ex: Petrobras S.A., UFOP, Embraer…"
              className="flex-1 px-4 py-2.5 rounded-lg text-sm outline-none"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }}
            />
            <Button onClick={onSearch} disabled={loading || !query.trim()}>
              {loading
                ? <><div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />Buscando…</>
                : <><Search size={14} />Estimar Portfolio</>}
            </Button>
          </div>
          <p className="text-xs" style={{ color: "var(--text-muted)" }}>
            Fontes: INPI · Lens.org · INPADOC
          </p>
        </div>
      </Card>

      {hasResult && (
        <>
          <div className="grid grid-cols-3 gap-4">
            {[
              { label: "Custo Mensal Estimado",  value: "R$ 42.300",  sub: "847 ativos encontrados" },
              { label: "Custo Anual Estimado",   value: "R$ 507.600", sub: "anuidades + taxas INPI" },
              { label: "Custo Total (período)",  value: "R$ 6,1M",    sub: "até vencimento estimado" },
            ].map(({ label, value, sub }) => (
              <Card key={label}>
                <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>{label}</p>
                <p className="text-2xl font-bold text-white">{value}</p>
                <p className="text-xs mt-1 text-amber-400">{sub}</p>
              </Card>
            ))}
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Distribuição por área — {query}</CardTitle>
            </CardHeader>
            <div className="space-y-3">
              {distribution.map(a => (
                <div key={a.area}>
                  <div className="flex justify-between text-xs mb-1.5">
                    <span className="flex items-center gap-1.5" style={{ color: "var(--text-muted)" }}>
                      {a.area}
                      {a.growing && <Badge variant="success">⚡ crescendo</Badge>}
                    </span>
                    <span className="text-white font-medium">{a.count} ativos · {a.pct}%</span>
                  </div>
                  <div className="h-2 rounded-full" style={{ background: "var(--border)" }}>
                    <div className="h-full rounded-full bg-indigo-500" style={{ width: `${a.pct}%` }} />
                  </div>
                </div>
              ))}
            </div>
            <div className="flex gap-2 mt-4">
              <Button variant="secondary" size="sm">Salvar como referência</Button>
              <Button variant="secondary" size="sm">Comparar com meu portfolio</Button>
            </div>
          </Card>
        </>
      )}
    </div>
  );
}
