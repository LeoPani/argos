"use client";

import Link from "next/link";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useStats, usePortfolio, useINPITimeline } from "@/lib/hooks";
import { formatDate, formatBRL, IPC_COLORS } from "@/lib/utils";
import { Skeleton, SkeletonKPI } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { AnalysisModeBadge } from "@/components/ui/analysis-mode-badge";
import { DataStateBanner } from "@/components/ui/data-state-banner";
import {
  PieChart, Pie, Cell, AreaChart, Area,
  XAxis, YAxis, Tooltip, ResponsiveContainer, BarChart, Bar,
} from "recharts";
import {
  FileText, Tag, AlertTriangle, Cpu, GraduationCap, Activity,
  RefreshCw, Newspaper, ShieldCheck, Rss,
} from "lucide-react";
import type { ActivityItem } from "@/lib/types";

// ─── kind → icon/color for activity feed ──────────────────────────────────────

const kindInfo: Record<ActivityItem["kind"], { label: string; color: string; Icon: typeof FileText }> = {
  patent:    { label: "Patente",   color: "#6366f1", Icon: FileText },
  trademark: { label: "Marca",     color: "#f59e0b", Icon: Tag      },
  dispute:   { label: "Disputa",   color: "#ef4444", Icon: AlertTriangle },
  ufop:      { label: "UFOP",      color: "#a855f7", Icon: GraduationCap },
};

const statusColors: Record<string, string> = {
  classified: "#34d399", pending: "#fbbf24", failed: "#ef4444",
  granted: "#34d399",    filed: "#fbbf24",  published: "#a855f7",
  denied: "#ef4444",     expired: "#64748b", open: "#fbbf24",
  resolved: "#34d399",   urgent: "#ef4444", new: "#6366f1",
  reviewed: "#34d399",   converted: "#a855f7", dismissed: "#64748b",
};

export default function DashboardPage() {
  const { data: stats, error: statsErr, isLoading: statsLoading, mutate: refreshStats } = useStats();
  const { data: portfolio } = usePortfolio();
  const { data: inpiTimeline } = useINPITimeline(30);

  const isLive  = !!stats;
  const loading = statsLoading && !stats && !statsErr;

  const c = stats?.counts;

  const ipcSlices = (stats?.ipc_distribution ?? []).map((s, i) => ({
    name:  `${s.letter} — ${s.name}`,
    value: s.pct,
    count: s.count,
    color: IPC_COLORS[i % IPC_COLORS.length],
  }));

  const tmStatuses = (stats?.trademark_statuses ?? []).map(s => ({
    name:  tmStatusLabel(s.status),
    value: s.pct,
    color: statusColors[s.status] ?? "#64748b",
  }));

  const timeline = portfolio?.cost_timeline ?? [];
  const aiAcc = c && c.patents > 0 ? Math.round((c.patents_classified / c.patents) * 100) : null;

  return (
    <div className="p-8 space-y-6">
      <DataStateBanner />

      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">BI &amp; Analytics</h1>
          <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
            Portfólio UFOP · OAI-PMH + Google Patents + INPI RPI
          </p>
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          <AnalysisModeBadge />
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

      {/* KPI grid — row 1: patents/trademarks/disputes/AI */}
      <div className="grid grid-cols-4 gap-4">
        {loading ? (
          <><SkeletonKPI /><SkeletonKPI /><SkeletonKPI /><SkeletonKPI /></>
        ) : (
          <>
            <KPI Icon={FileText}      label="Patentes UFOP"
              value={c ? c.patents.toLocaleString("pt-BR") : "—"}
              sub={c ? `${c.patents_classified} classificadas` : "carregando..."}
              color="#6366f1"
              delta={c?.patents_week} />
            <KPI Icon={Tag}           label="Marcas"
              value={c ? c.trademarks.toLocaleString("pt-BR") : "—"}
              sub={c ? `${c.trademarks_active} ativas` : "carregando..."}
              color="#8b5cf6"
              delta={c?.trademarks_week} />
            <KPI Icon={AlertTriangle} label="Disputas abertas"
              value={c ? c.disputes_open.toString() : "—"}
              sub={c ? `${c.disputes} totais` : "carregando..."}
              color="#f59e0b"
              delta={c?.disputes_week} />
            <KPI Icon={Cpu}           label="IA Acurácia"
              value={aiAcc !== null ? `${aiAcc}%` : "—"}
              sub={isLive ? "BERT + TF-IDF ativo" : "offline"}
              color="#34d399" />
          </>
        )}
      </div>

      {/* KPI grid — row 2: INPI + UFOP + timestamps */}
      <div className="grid grid-cols-4 gap-4">
        {loading ? (
          <><SkeletonKPI /><SkeletonKPI /><SkeletonKPI /><SkeletonKPI /></>
        ) : (
          <>
            <KPI Icon={Newspaper}    label="Despachos INPI"
              value={c ? c.inpi_publications.toLocaleString("pt-BR") : "—"}
              sub={c && c.latest_rpi > 0 ? `Última RPI: ${c.latest_rpi}` : "sem dados"}
              color="#06b6d4" />
            <KPI Icon={GraduationCap} label="Oport. UFOP"
              value={c ? c.ufop_opportunities.toLocaleString("pt-BR") : "—"}
              sub={c ? `${c.ufop_high} de alto potencial` : "carregando..."}
              color="#a855f7"
              delta={c?.ufop_week} />
            <KPI Icon={ShieldCheck}  label="Reg. Anterioridade"
              value={c ? c.ip_timestamps.toString() : "—"}
              sub="prova de existência SHA-256"
              color="#34d399" />
            <Card style={{ borderColor: "#06b6d440" }}>
              <div className="flex items-start gap-3">
                <div className="p-2 rounded-lg" style={{ background: "#06b6d420" }}>
                  <Rss size={16} className="text-cyan-400" />
                </div>
                <div>
                  <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>Portfólio</p>
                  <div className="space-y-0.5">
                    <p className="text-xs">
                      <span className="text-white font-semibold">{formatBRL(portfolio?.cost_summary.monthly ?? 0)}</span>
                      <span style={{ color: "var(--text-muted)" }}> /mês</span>
                    </p>
                    <p className="text-xs">
                      <span className="text-white font-semibold">{formatBRL(portfolio?.cost_summary.annual ?? 0)}</span>
                      <span style={{ color: "var(--text-muted)" }}> /ano</span>
                    </p>
                  </div>
                </div>
              </div>
            </Card>
          </>
        )}
      </div>

      {/* Charts row */}
      <div className="grid grid-cols-3 gap-4">
        {/* IPC Distribution */}
        <Card className="col-span-1">
          <CardHeader>
            <CardTitle>Patentes por categoria IPC</CardTitle>
          </CardHeader>
          {ipcSlices.length === 0 ? (
            <div className="py-8 text-center">
              <p className="text-xs" style={{ color: "var(--text-muted)" }}>
                {loading ? "Carregando..." : "Sem patentes classificadas ainda."}
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {ipcSlices.map(({ name, value, count, color }) => (
                <div key={name}>
                  <div className="flex justify-between text-xs mb-1">
                    <span style={{ color: "var(--text-muted)" }} className="truncate max-w-[140px]">{name}</span>
                    <span className="text-white font-medium">
                      {value}%
                      <span style={{ color: "var(--text-muted)" }}> ({count})</span>
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

        {/* INPI Timeline (primary) — or trademark pie if timeline unavailable */}
        <Card className="col-span-1">
          <CardHeader>
            <CardTitle>
              {(inpiTimeline?.points?.length ?? 0) > 0
                ? "INPI — Despachos por RPI"
                : tmStatuses.length > 0 ? "Marcas por status" : "INPI — Despachos RPI"}
            </CardTitle>
          </CardHeader>
          {(inpiTimeline?.points?.length ?? 0) > 0 ? (
            /* INPI timeline AreaChart */
            <ResponsiveContainer width="100%" height={130}>
              <AreaChart data={inpiTimeline!.points}>
                <defs>
                  <linearGradient id="inpiGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%"  stopColor="#06b6d4" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#06b6d4" stopOpacity={0}   />
                  </linearGradient>
                  <linearGradient id="ufopGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%"  stopColor="#a855f7" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#a855f7" stopOpacity={0}   />
                  </linearGradient>
                </defs>
                <XAxis dataKey="rpi" tick={{ fontSize: 9 }}
                  tickFormatter={(v: number) => `${v}`}
                  interval={Math.floor((inpiTimeline!.points.length - 1) / 4)} />
                <YAxis tick={{ fontSize: 9 }}
                  tickFormatter={(v: number) => v >= 1000 ? `${(v/1000).toFixed(1)}k` : String(v)} />
                <Tooltip
                  formatter={(v, name) => [
                    Number(v).toLocaleString("pt-BR"),
                    name === "total" ? "Total" : "UFOP"
                  ]}
                  labelFormatter={(l) => `RPI ${l}`}
                />
                <Area type="monotone" dataKey="total" stroke="#06b6d4" fill="url(#inpiGrad)" strokeWidth={2} name="total" />
                <Area type="monotone" dataKey="ufop"  stroke="#a855f7" fill="url(#ufopGrad)" strokeWidth={1.5} name="ufop" />
              </AreaChart>
            </ResponsiveContainer>
          ) : tmStatuses.length > 0 ? (
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
          ) : (
            /* Fallback: static INPI stats */
            <div className="space-y-3 py-2">
              <div className="text-center">
                <p className="text-3xl font-bold text-cyan-400">
                  {c ? c.inpi_publications.toLocaleString("pt-BR") : "—"}
                </p>
                <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>despachos indexados</p>
              </div>
              <div className="grid grid-cols-2 gap-2 text-center">
                <div className="rounded-lg p-2" style={{ background: "var(--surface-2)" }}>
                  <p className="text-lg font-bold text-white">{c?.latest_rpi ?? "—"}</p>
                  <p className="text-xs" style={{ color: "var(--text-muted)" }}>Última RPI</p>
                </div>
                <div className="rounded-lg p-2" style={{ background: "var(--surface-2)" }}>
                  <p className="text-lg font-bold text-white">{c ? c.ufop_opportunities.toLocaleString("pt-BR") : "—"}</p>
                  <p className="text-xs" style={{ color: "var(--text-muted)" }}>Oport. UFOP</p>
                </div>
              </div>
            </div>
          )}
        </Card>

        {/* Cost timeline or UFOP distribution */}
        <Card className="col-span-1">
          <CardHeader>
            <CardTitle>
              {timeline.length > 0 ? "Custo anual projetado" : "UFOP — Nível de potencial"}
            </CardTitle>
          </CardHeader>
          {timeline.length > 0 ? (
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
          ) : c ? (
            <ResponsiveContainer width="100%" height={130}>
              <BarChart data={[
                { label: "Alto", value: c.ufop_high, fill: "#34d399" },
                { label: "Total patentáveis", value: c.ufop_opportunities, fill: "#6366f1" },
              ]}>
                <XAxis dataKey="label" tick={{ fontSize: 10 }} />
                <YAxis tick={{ fontSize: 10 }} tickFormatter={v => v >= 1000 ? `${(v/1000).toFixed(1)}k` : v.toString()} />
                <Tooltip formatter={(v) => [Number(v).toLocaleString("pt-BR"), "Registros"]} />
                <Bar dataKey="value" radius={[4, 4, 0, 0]}>
                  {[{ fill: "#34d399" }, { fill: "#6366f1" }].map((e, i) => (
                    <Cell key={i} fill={e.fill} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-32 flex items-center justify-center">
              <p className="text-xs" style={{ color: "var(--text-muted)" }}>Carregando...</p>
            </div>
          )}
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

        {loading ? (
          <div className="space-y-2">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="flex items-center gap-3 py-2 px-3">
                <Skeleton className="h-4 w-4 rounded-full" />
                <Skeleton className="h-4 w-16" />
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-4 flex-1" />
                <Skeleton className="h-4 w-20" />
              </div>
            ))}
          </div>
        ) : (stats?.recent_activity?.length ?? 0) > 0 ? (
          <div className="space-y-1">
            {stats!.recent_activity.map(item => {
              const info = kindInfo[item.kind];
              const Icon = info.Icon;
              const href =
                item.kind === "patent"    ? `/patents/${item.id}` :
                item.kind === "trademark" ? `/trademarks/${item.id}` :
                item.kind === "dispute"   ? "/arbitragem" :
                item.kind === "ufop"      ? "/ufop" : "/dashboard";
              return (
                <Link key={`${item.kind}-${item.id}`} href={href}>
                  <div className="flex items-center gap-3 py-2 px-3 rounded-lg hover:bg-white/5 cursor-pointer transition-colors"
                    style={{ borderLeft: `2px solid ${info.color}` }}>
                    <Icon size={14} style={{ color: info.color }} />
                    <Badge variant="muted">{info.label}</Badge>
                    <span className="font-mono text-xs text-indigo-400 shrink-0 max-w-[160px] truncate">{item.reference}</span>
                    <span className="text-sm text-white flex-1 truncate">{item.title}</span>
                    <Badge variant="muted">{item.status}</Badge>
                    <span className="text-xs shrink-0" style={{ color: "var(--text-muted)" }}>
                      {formatDate(item.created_at)}
                    </span>
                  </div>
                </Link>
              );
            })}
          </div>
        ) : (
          <EmptyState
            title="Nenhuma atividade recente"
            description="Cadastre uma patente, marca ou disputa para começar a ver o feed unificado."
            size="sm"
          />
        )}
      </Card>
    </div>
  );
}

// ─── small components ─────────────────────────────────────────────────────────

function KPI({ Icon, label, value, sub, color, delta }: {
  Icon: typeof FileText; label: string; value: string; sub: string; color: string;
  delta?: number;
}) {
  return (
    <Card>
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>{label}</p>
          <div className="flex items-baseline gap-2">
            <p className="text-2xl font-bold text-white">{value}</p>
            {delta !== undefined && delta > 0 && (
              <span className="text-xs font-semibold px-1.5 py-0.5 rounded-full"
                style={{ background: "#34d39920", color: "#34d399" }}>
                +{delta} semana
              </span>
            )}
          </div>
          <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>{sub}</p>
        </div>
        <div className="p-2 rounded-lg" style={{ background: color + "20" }}>
          <Icon size={18} style={{ color }} />
        </div>
      </div>
    </Card>
  );
}

function tmStatusLabel(s: string): string {
  const labels: Record<string, string> = {
    granted: "Registrada", filed: "Depositada", published: "Em oposição",
    denied: "Indeferida", archived: "Arquivada", expired: "Extinta",
  };
  return labels[s] ?? s;
}
