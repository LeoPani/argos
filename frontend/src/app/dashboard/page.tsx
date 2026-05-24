"use client";

import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { mockIpcDistribution, mockCostTimeline } from "@/lib/mock-data";
import { useStats, usePortfolio } from "@/lib/hooks";
import { formatDate, formatBRL, IPC_COLORS } from "@/lib/utils";
import {
  PieChart, Pie, Cell, AreaChart, Area,
  XAxis, YAxis, Tooltip, ResponsiveContainer,
} from "recharts";
import {
  FileText, Tag, AlertTriangle, Cpu, GraduationCap, Activity,
  RefreshCw,
} from "lucide-react";
import type { ActivityItem } from "@/lib/types";

// ─── kind → icon/color for activity feed ──────────────────────────────────────

const kindInfo: Record<ActivityItem["kind"], { label: string; color: string; Icon: typeof FileText }> = {
  patent:    { label: "Patente",   color: "#6366f1", Icon: FileText },
  trademark: { label: "Marca",     color: "#f59e0b", Icon: Tag      },
  dispute:   { label: "Disputa",   color: "#ef4444", Icon: AlertTriangle },
  ufop:      { label: "UFOP",      color: "#a855f7", Icon: GraduationCap },
};

// status → pie color
const statusColors: Record<string, string> = {
  classified: "#34d399",
  pending:    "#fbbf24",
  failed:     "#ef4444",
  granted:    "#34d399",
  filed:      "#fbbf24",
  published:  "#a855f7",
  denied:     "#ef4444",
  expired:    "#64748b",
  open:       "#fbbf24",
  resolved:   "#34d399",
  urgent:     "#ef4444",
  new:        "#6366f1",
  reviewed:   "#34d399",
  converted:  "#a855f7",
  dismissed:  "#64748b",
};

export default function DashboardPage() {
  const { data: stats,     mutate: refreshStats } = useStats();
  const { data: portfolio }                      = usePortfolio();

  const isLive = !!stats;

  // Use real stats when present, otherwise fall back to plausible mock numbers.
  const c = stats?.counts ?? {
    patents: 1204, patents_classified: 1170,
    trademarks: 893, trademarks_active: 730,
    disputes: 18, disputes_open: 6,
    ufop_opportunities: 23, ufop_high: 7,
  };

  const ipcSlices = stats?.ipc_distribution.length
    ? stats.ipc_distribution.map((s, i) => ({
        name:  `${s.letter} — ${s.name}`,
        value: s.pct,
        count: s.count,
        color: IPC_COLORS[i % IPC_COLORS.length],
      }))
    : mockIpcDistribution.map((d, i) => ({
        ...d, count: 0, color: IPC_COLORS[i],
      }));

  const tmStatuses = stats?.trademark_statuses.length
    ? stats.trademark_statuses.map(s => ({
        name: tmStatusLabel(s.status), value: s.pct,
        color: statusColors[s.status] ?? "#64748b",
      }))
    : [
        { name: "Ativas",    value: 72, color: "#34d399" },
        { name: "Oposição",  value: 18, color: "#f59e0b" },
        { name: "Extintas",  value: 10, color: "#64748b" },
      ];

  const timeline = portfolio?.cost_timeline.length
    ? portfolio.cost_timeline
    : mockCostTimeline;

  const aiAcc = c.patents > 0 ? Math.round((c.patents_classified / c.patents) * 100) : 97;

  return (
    <div className="p-8 space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">BI &amp; Analytics</h1>
          <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
            Visão geral do sistema — dados INPI + classificação IA
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
          <Button variant="ghost" size="sm" onClick={() => refreshStats()}>
            <RefreshCw size={13} /> Atualizar
          </Button>
        </div>
      </div>

      {/* KPI grid (4 cards) */}
      <div className="grid grid-cols-4 gap-4">
        <KPI Icon={FileText}      label="Patentes"          value={c.patents.toLocaleString("pt-BR")}     sub={`${c.patents_classified} classificadas`}    color="#6366f1" />
        <KPI Icon={Tag}           label="Marcas"            value={c.trademarks.toLocaleString("pt-BR")}  sub={`${c.trademarks_active} ativas`}            color="#8b5cf6" />
        <KPI Icon={AlertTriangle} label="Disputas abertas"  value={c.disputes_open.toString()}            sub={`${c.disputes} totais`}                     color="#f59e0b" />
        <KPI Icon={Cpu}           label="IA Acurácia"       value={`${aiAcc}%`}                            sub={isLive ? "BERT ativo" : "estimativa"}        color="#34d399" />
      </div>

      {/* Second row: UFOP highlight */}
      <div className="grid grid-cols-4 gap-4">
        <Card style={{ borderColor: "#a855f730" }}>
          <div className="flex items-start gap-3">
            <div className="p-2 rounded-lg" style={{ background: "#a855f720" }}>
              <GraduationCap size={18} className="text-purple-400" />
            </div>
            <div className="flex-1">
              <p className="text-xs" style={{ color: "var(--text-muted)" }}>Oportunidades UFOP</p>
              <p className="text-2xl font-bold text-white">{c.ufop_opportunities}</p>
              <p className="text-xs mt-0.5">
                <span className="text-red-400 font-semibold">{c.ufop_high}</span>
                <span style={{ color: "var(--text-muted)" }}> de alto potencial</span>
              </p>
            </div>
          </div>
        </Card>

        <Card className="col-span-3">
          <CardHeader>
            <CardTitle>Resumo do portfolio</CardTitle>
          </CardHeader>
          <div className="grid grid-cols-3 gap-6">
            <Mini label="Custo mensal" value={formatBRL(portfolio?.cost_summary.monthly ?? 0)} />
            <Mini label="Custo anual"  value={formatBRL(portfolio?.cost_summary.annual  ?? 0)} />
            <Mini label="Custo total"  value={formatBRL(portfolio?.cost_summary.total   ?? 0)} />
          </div>
        </Card>
      </div>

      {/* Charts row */}
      <div className="grid grid-cols-3 gap-4">
        {/* IPC Distribution */}
        <Card className="col-span-1">
          <CardHeader>
            <CardTitle>Patentes por categoria IPC</CardTitle>
          </CardHeader>
          {ipcSlices.length === 0 ? (
            <p className="text-xs text-center py-6" style={{ color: "var(--text-muted)" }}>
              Sem patentes classificadas ainda.
            </p>
          ) : (
            <div className="space-y-2">
              {ipcSlices.map(({ name, value, count, color }) => (
                <div key={name}>
                  <div className="flex justify-between text-xs mb-1">
                    <span style={{ color: "var(--text-muted)" }} className="truncate max-w-[140px]">{name}</span>
                    <span className="text-white font-medium">
                      {value}%
                      {count > 0 && <span style={{ color: "var(--text-muted)" }}> ({count})</span>}
                    </span>
                  </div>
                  <div className="h-1.5 rounded-full" style={{ background: "var(--border)" }}>
                    <div className="h-full rounded-full" style={{ width: `${value}%`, background: color }} />
                  </div>
                </div>
              ))}
            </div>
          )}
        </Card>

        {/* Trademark statuses */}
        <Card className="col-span-1">
          <CardHeader>
            <CardTitle>Marcas por status</CardTitle>
          </CardHeader>
          {tmStatuses.length === 0 ? (
            <p className="text-xs text-center py-6" style={{ color: "var(--text-muted)" }}>
              Sem marcas cadastradas.
            </p>
          ) : (
            <div className="flex items-center gap-4">
              <ResponsiveContainer width={120} height={120}>
                <PieChart>
                  <Pie data={tmStatuses} cx="50%" cy="50%" innerRadius={35} outerRadius={55} dataKey="value" strokeWidth={0}>
                    {tmStatuses.map((d, i) => <Cell key={i} fill={d.color} />)}
                  </Pie>
                </PieChart>
              </ResponsiveContainer>
              <div className="space-y-2">
                {tmStatuses.map(d => (
                  <div key={d.name} className="flex items-center gap-2 text-xs">
                    <div className="w-2 h-2 rounded-full" style={{ background: d.color }} />
                    <span style={{ color: "var(--text-muted)" }}>{d.name}</span>
                    <span className="text-white font-medium ml-auto">{d.value}%</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </Card>

        {/* Cost timeline */}
        <Card className="col-span-1">
          <CardHeader>
            <CardTitle>Custo anual projetado</CardTitle>
          </CardHeader>
          <ResponsiveContainer width="100%" height={130}>
            <AreaChart data={timeline}>
              <defs>
                <linearGradient id="costGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%"  stopColor="#6366f1" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#6366f1" stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis dataKey="year" tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} tickFormatter={v => `R$${(v / 1000).toFixed(0)}k`} />
              <Tooltip formatter={(v) => [`R$ ${Number(v).toLocaleString("pt-BR")}`, "Custo"]} />
              <Area type="monotone" dataKey="value" stroke="#6366f1" fill="url(#costGrad)" strokeWidth={2} />
            </AreaChart>
          </ResponsiveContainer>
        </Card>
      </div>

      {/* Recent activity feed */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Activity size={15} className="text-indigo-400" />
            Atividade recente
          </CardTitle>
          <span className="text-xs" style={{ color: "var(--text-muted)" }}>
            {isLive ? "✓ feed unificado ao vivo" : "modo offline"}
          </span>
        </CardHeader>

        {stats?.recent_activity?.length ? (
          <div className="space-y-2">
            {stats.recent_activity.map(item => {
              const info = kindInfo[item.kind];
              const Icon = info.Icon;
              return (
                <div key={`${item.kind}-${item.id}`}
                  className="flex items-center gap-3 py-2 px-3 rounded-lg hover:bg-white/5 transition-colors"
                  style={{ borderLeft: `2px solid ${info.color}` }}>
                  <Icon size={14} style={{ color: info.color }} />
                  <Badge variant="muted">{info.label}</Badge>
                  <span className="font-mono text-xs text-indigo-400 shrink-0">{item.reference}</span>
                  <span className="text-sm text-white flex-1 truncate">{item.title}</span>
                  <Badge variant="muted">{item.status}</Badge>
                  <span className="text-xs shrink-0" style={{ color: "var(--text-muted)" }}>
                    {formatDate(item.created_at)}
                  </span>
                </div>
              );
            })}
          </div>
        ) : (
          <p className="text-sm text-center py-6" style={{ color: "var(--text-muted)" }}>
            Nenhuma atividade recente. Cadastre uma patente ou marca para começar.
          </p>
        )}
      </Card>
    </div>
  );
}

// ─── small components ─────────────────────────────────────────────────────────

function KPI({ Icon, label, value, sub, color }: {
  Icon: typeof FileText; label: string; value: string; sub: string; color: string;
}) {
  return (
    <Card>
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>{label}</p>
          <p className="text-2xl font-bold text-white">{value}</p>
          <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>{sub}</p>
        </div>
        <div className="p-2 rounded-lg" style={{ background: color + "20" }}>
          <Icon size={18} style={{ color }} />
        </div>
      </div>
    </Card>
  );
}

function Mini({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>{label}</p>
      <p className="text-lg font-semibold text-white">{value}</p>
    </div>
  );
}

function tmStatusLabel(s: string): string {
  const labels: Record<string, string> = {
    granted: "Registrada", filed: "Depositada", published: "Em oposição",
    denied: "Indeferida", archived: "Arquivada", expired: "Extinta",
  };
  return labels[s] ?? s;
}
