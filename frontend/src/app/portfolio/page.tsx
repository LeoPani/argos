"use client";

import { useState } from "react";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { mockPortfolioAssets, mockAIOpportunities, mockCostTimeline } from "@/lib/mock-data";
import { formatBRL, formatDate, daysUntil, cn } from "@/lib/utils";
import type { PortfolioAsset, AIOpportunity } from "@/lib/types";
import {
  AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer,
} from "recharts";
import {
  TrendingUp, AlertCircle, Search, Plus, Link2,
  FileText, Tag, Cpu, Package, Lightbulb, ShieldAlert
} from "lucide-react";

type Tab = "own" | "third";

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
  const map = { active: "success", pending: "warning", expired: "muted", opposition: "danger" } as const;
  const labels = { active: "Ativa", pending: "Pendente", expired: "Expirada", opposition: "Oposição" };
  return <Badge variant={map[status]}>{labels[status]}</Badge>;
}

function DeadlinePill({ date }: { date: string | null }) {
  const days = daysUntil(date);
  if (days === null) return <span style={{ color: "var(--text-muted)" }}>—</span>;
  if (days < 0) return <Badge variant="muted">Vencida</Badge>;
  if (days <= 30) return <Badge variant="danger">⚠ {days}d</Badge>;
  if (days <= 90) return <Badge variant="warning">{days}d</Badge>;
  return <Badge variant="success">{formatDate(date)}</Badge>;
}

const totalCosts = mockPortfolioAssets.reduce(
  (acc, a) => ({ monthly: acc.monthly + a.cost_monthly, annual: acc.annual + a.cost_annual, total: acc.total + a.cost_total }),
  { monthly: 0, annual: 0, total: 0 }
);

export default function PortfolioPage() {
  const [tab, setTab] = useState<Tab>("own");
  const [searchQuery, setSearchQuery] = useState("");
  const [thirdPartyQuery, setThirdPartyQuery] = useState("");
  const [loading, setLoading] = useState(false);
  const [thirdPartyResult, setThirdPartyResult] = useState(false);

  async function estimateThirdParty() {
    if (!thirdPartyQuery.trim()) return;
    setLoading(true);
    await new Promise(r => setTimeout(r, 1600));
    setThirdPartyResult(true);
    setLoading(false);
  }

  return (
    <div className="p-8 space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Portfolio de PI</h1>
          <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
            Gestão de ativos, prazos, custos e oportunidades detectadas por IA
          </p>
        </div>
        <Button size="sm">
          <Plus size={14} />
          Adicionar ativo
        </Button>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 p-1 rounded-lg w-fit" style={{ background: "var(--surface)" }}>
        {(["own", "third"] as Tab[]).map(t => (
          <button key={t} onClick={() => setTab(t)}
            className={cn("px-4 py-2 rounded-md text-sm transition-all", tab === t ? "text-white font-medium" : "hover:text-white")}
            style={{ background: tab === t ? "var(--accent)" : "transparent", color: tab === t ? "white" : "var(--text-muted)" }}>
            {t === "own" ? "Meu Portfolio" : "Estimar Terceiro"}
          </button>
        ))}
      </div>

      {tab === "own" ? (
        <>
          {/* Cost Summary */}
          <div className="grid grid-cols-3 gap-4">
            {[
              { label: "Custo Mensal", value: totalCosts.monthly, sub: "estimado" },
              { label: "Custo Anual", value: totalCosts.annual, sub: "estimado" },
              { label: "Custo Total (período)", value: totalCosts.total, sub: "até vencimento" },
            ].map(({ label, value, sub }) => (
              <Card key={label}>
                <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>{label}</p>
                <p className="text-2xl font-bold text-white">{formatBRL(value)}</p>
                <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>{sub}</p>
              </Card>
            ))}
          </div>

          {/* Cost timeline */}
          <Card>
            <CardHeader>
              <CardTitle>Projeção de custos (5 anos)</CardTitle>
              <div className="flex gap-3 text-xs">
                <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-indigo-400" />Anuidades patentes</span>
                <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-purple-400" />Renovação marcas</span>
              </div>
            </CardHeader>
            <ResponsiveContainer width="100%" height={160}>
              <AreaChart data={mockCostTimeline}>
                <defs>
                  <linearGradient id="pg" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#6366f1" stopOpacity={0.3} />
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

          {/* Assets Table */}
          <Card>
            <CardHeader>
              <CardTitle>Ativos ({mockPortfolioAssets.length})</CardTitle>
              <div className="flex items-center gap-2">
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
              </div>
            </CardHeader>
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
                  {mockPortfolioAssets
                    .filter(a => !searchQuery || a.title.toLowerCase().includes(searchQuery.toLowerCase()))
                    .map(asset => (
                      <tr key={asset.id} style={{ borderBottom: "1px solid var(--border)" }} className="hover:bg-white/5">
                        <td className="py-3 pr-3">
                          <div className="flex items-center gap-1.5">
                            {assetTypeIcon[asset.type]}
                            <Badge variant={assetTypeBadge[asset.type]}>{asset.type}</Badge>
                          </div>
                        </td>
                        <td className="py-3 pr-3">
                          <p className="text-white font-medium text-xs max-w-[180px] truncate">{asset.title}</p>
                          <p className="font-mono text-xs text-indigo-400">{asset.number}</p>
                        </td>
                        <td className="py-3 pr-3">{statusBadge(asset.status)}</td>
                        <td className="py-3 pr-3 text-xs" style={{ color: "var(--text-muted)" }}>{formatDate(asset.expiry_date)}</td>
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
                    ))}
                </tbody>
              </table>
            </div>
          </Card>

          {/* AI Opportunities */}
          <div>
            <h2 className="text-base font-semibold text-white mb-3 flex items-center gap-2">
              <Lightbulb size={16} className="text-amber-400" />
              Oportunidades sugeridas pela IA
            </h2>
            <div className="grid grid-cols-1 gap-3">
              {mockAIOpportunities.map(opp => (
                <OpportunityCard key={opp.id} opp={opp} />
              ))}
            </div>
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

function OpportunityCard({ opp }: { opp: AIOpportunity }) {
  return (
    <Card style={{ borderColor: opp.type === "opportunity" ? "#6366f130" : "#ef444430" }}>
      <div className="flex items-start gap-3">
        <div className={cn("p-2 rounded-lg mt-0.5 shrink-0",
          opp.type === "opportunity" ? "bg-amber-500/20" : "bg-red-500/20")}>
          {opp.type === "opportunity"
            ? <TrendingUp size={15} className="text-amber-400" />
            : <ShieldAlert size={15} className="text-red-400" />}
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            <p className="text-sm font-semibold text-white">{opp.title}</p>
            <Badge variant="muted">{opp.confidence}% confiança</Badge>
            {opp.ipc_class && <Badge variant="info">IPC: {opp.ipc_class}</Badge>}
          </div>
          <p className="text-xs leading-relaxed" style={{ color: "var(--text-muted)" }}>{opp.description}</p>
          {opp.estimated_cost && (
            <p className="text-xs mt-1 text-indigo-400">
              Custo estimado do depósito: {formatBRL(opp.estimated_cost)}
            </p>
          )}
        </div>
        <Button variant="secondary" size="sm">{opp.action_label}</Button>
      </div>
    </Card>
  );
}

function ThirdPartyEstimator({ query, setQuery, loading, onSearch, hasResult }: {
  query: string;
  setQuery: (v: string) => void;
  loading: boolean;
  onSearch: () => void;
  hasResult: boolean;
}) {
  const thirdPartyAssets = [
    { area: "Exploração de petróleo", pct: 38, count: 322 },
    { area: "Refino e processos", pct: 27, count: 229 },
    { area: "Energias renováveis", pct: 18, count: 153, growing: true },
    { area: "Meio ambiente", pct: 17, count: 143 },
  ];

  return (
    <div className="space-y-4">
      <Card>
        <div className="space-y-4">
          <div>
            <label className="text-xs font-medium mb-2 block" style={{ color: "var(--text-muted)" }}>
              Titular / Empresa a estimar
            </label>
            <div className="flex gap-3">
              <input
                value={query}
                onChange={e => setQuery(e.target.value)}
                placeholder="ex: Petrobras S.A., UFOP, Embraer..."
                className="flex-1 px-4 py-2.5 rounded-lg text-sm outline-none"
                style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }}
              />
              <Button onClick={onSearch} disabled={loading || !query.trim()}>
                {loading
                  ? <><div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />Buscando...</>
                  : <><Search size={14} />Estimar Portfolio</>}
              </Button>
            </div>
            <p className="text-xs mt-2" style={{ color: "var(--text-muted)" }}>
              Fontes: INPI · Lens.org · INPADOC
            </p>
          </div>
        </div>
      </Card>

      {hasResult && (
        <>
          <div className="grid grid-cols-3 gap-4">
            <Card>
              <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>Custo Mensal Estimado</p>
              <p className="text-2xl font-bold text-white">R$ 42.300</p>
              <p className="text-xs mt-1 text-amber-400">847 ativos encontrados</p>
            </Card>
            <Card>
              <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>Custo Anual Estimado</p>
              <p className="text-2xl font-bold text-white">R$ 507.600</p>
              <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>anuidades + taxas INPI</p>
            </Card>
            <Card>
              <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>Custo Total (período)</p>
              <p className="text-2xl font-bold text-white">R$ 6,1M</p>
              <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>até vencimento estimado</p>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Distribuição por área tecnológica — {query}</CardTitle>
            </CardHeader>
            <div className="space-y-3">
              {thirdPartyAssets.map(a => (
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
