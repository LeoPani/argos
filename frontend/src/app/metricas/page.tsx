"use client";

// /metricas — BI dashboard de inteligência de PI com métricas
// academicamente validadas (AUTM, HJT 2001, Etzkowitz 2000, L-S 2004).

import Link from "next/link";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SkeletonKPI } from "@/components/ui/skeleton";
import { MetricTooltip } from "@/components/ui/metric-tooltip";
import { useToast } from "@/components/ui/toast";
import { useMetrics, useDepartments, useKnowledgeStock, useRoyaltyForecast } from "@/lib/hooks";
import { api } from "@/lib/api";
import { formatBRL } from "@/lib/utils";
import {
  BarChart, Bar, ResponsiveContainer, XAxis, YAxis, Tooltip,
  RadialBarChart, RadialBar, PolarAngleAxis,
  AreaChart, Area,
} from "recharts";
import {
  GraduationCap, TrendingUp, Network, Award,
  RefreshCw, BookOpen, Database, ArrowRight, AlertCircle,
  Building, Layers, Coins, Sparkles,
} from "lucide-react";
import { useState } from "react";

export default function MetricsPage() {
  const { data, isLoading, mutate } = useMetrics("UFOP");
  const { data: depts } = useDepartments();
  const { data: stock } = useKnowledgeStock("UFOP");
  const { data: forecast } = useRoyaltyForecast(10);
  const [enriching, setEnriching] = useState(false);
  const toast = useToast();

  async function enrichAll() {
    setEnriching(true);
    try {
      const r = await api.metrics.enrichAll(50);
      toast.success(
        `Enrichment concluído (${r.source})`,
        `${r.processed} patentes UFOP processadas · média ${r.avg_fwd_citations.toFixed(1)} forward citations`,
      );
      mutate();
    } catch (e) {
      toast.error("Falha no enrichment", e instanceof Error ? e.message : "Erro");
    } finally { setEnriching(false); }
  }

  if (isLoading || !data) {
    return (
      <div className="p-8 space-y-6">
        <h1 className="text-2xl font-bold text-white">Métricas Acadêmicas</h1>
        <div className="grid grid-cols-4 gap-4">
          <SkeletonKPI /><SkeletonKPI /><SkeletonKPI /><SkeletonKPI />
        </div>
      </div>
    );
  }

  const hs = data.health_score;
  const ttf = data.tt_funnel;
  const div = data.ipc_diversity;
  const th = data.triple_helix;

  // Score gauge data
  const gaugeData = [{ name: "score", value: hs.composite_score, fill: scoreColor(hs.composite_score) }];

  // Funnel as bars
  const funnelData = [
    { stage: "Disclosures",       value: ttf.disclosures },
    { stage: "Filed",             value: ttf.patents_filed },
    { stage: "Granted",           value: ttf.patents_granted },
    { stage: "Active Contracts",  value: ttf.active_contracts },
  ];

  // Triple Helix as radial composite
  const total = th.u_count + th.i_count + th.g_count || 1;
  const thData = [
    { name: "Universidade", value: (th.u_count / total) * 100, fill: "#6366f1" },
    { name: "Indústria",    value: (th.i_count / total) * 100, fill: "#f59e0b" },
    { name: "Governo",      value: (th.g_count / total) * 100, fill: "#34d399" },
  ];

  return (
    <div className="p-8 space-y-6 fade-in">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white flex items-center gap-2">
            <GraduationCap size={22} />
            Métricas Acadêmicas de PI
          </h1>
          <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
            Indicadores AUTM · Hall-Jaffe-Trajtenberg · Etzkowitz · Lanjouw-Schankerman
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Link href="/smart-filing">
            <Button variant="secondary" size="sm">
              <Sparkles size={13} /> Smart Filing
            </Button>
          </Link>
          <Link href="/metodologia">
            <Button variant="ghost" size="sm">
              <BookOpen size={13} /> Metodologia
            </Button>
          </Link>
          <Button variant="ghost" size="sm" onClick={() => mutate()}>
            <RefreshCw size={13} /> Atualizar
          </Button>
          <Button size="sm" onClick={enrichAll} disabled={enriching}>
            {enriching
              ? <><div className="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin" /> Lens.org…</>
              : <><Database size={13} /> Enriquecer via Lens</>}
          </Button>
        </div>
      </div>

      {/* AUTM Health Score — large gauge */}
      <Card style={{ borderColor: scoreColor(hs.composite_score) + "40" }}>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Award size={15} className="text-amber-400" />
            AUTM Health Score
            <MetricTooltip metricID="autm_health_score" />
            <Badge variant="muted">{hs.methodology}</Badge>
          </CardTitle>
          <span className="text-xs" style={{ color: "var(--text-muted)" }}>
            Escopo: {hs.scope} · {hs.patents} patentes · {hs.inventors} inventores
          </span>
        </CardHeader>

        <div className="grid grid-cols-6 gap-4 items-center">
          {/* Gauge */}
          <div className="col-span-2 relative">
            <ResponsiveContainer width="100%" height={180}>
              <RadialBarChart innerRadius="60%" outerRadius="100%" data={gaugeData} startAngle={180} endAngle={0}>
                <PolarAngleAxis type="number" domain={[0, 100]} tick={false} />
                <RadialBar background dataKey="value" cornerRadius={10} />
              </RadialBarChart>
            </ResponsiveContainer>
            <div className="absolute inset-0 flex items-center justify-center pointer-events-none mt-6">
              <div className="text-center">
                <p className="text-4xl font-bold text-white">{hs.composite_score.toFixed(1)}</p>
                <p className="text-xs" style={{ color: "var(--text-muted)" }}>de 100</p>
              </div>
            </div>
          </div>

          {/* P1-P5 indicators */}
          <div className="col-span-4 grid grid-cols-5 gap-3">
            <Indicator label="P1 Disclosures/inv"   value={hs.p1_disclosures.toFixed(2)}      tip="autm_health_score" />
            <Indicator label="P2 Grant rate"         value={`${(hs.p2_grant_rate*100).toFixed(1)}%`}  tip="autm_health_score" />
            <Indicator label="P3 License intensity"  value={`${(hs.p3_license_rate*100).toFixed(1)}%`} tip="autm_health_score"
              benchmark={`benchmark FORTEC: ${(hs.benchmarks.autm_median_license_intensity*100).toFixed(0)}%`} />
            <Indicator label="P4 Revenue / asset"    value={formatBRL(hs.p4_revenue_per_asset)} tip="autm_health_score" />
            <Indicator label="P5 Time to grant"      value={`${hs.p5_time_to_grant}d`}          tip="autm_health_score" />
          </div>
        </div>
      </Card>

      {/* TT Conversion Funnel */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TrendingUp size={14} className="text-indigo-400" />
            TT Conversion Funnel
            <MetricTooltip metricID="tt_funnel" />
          </CardTitle>
          <span className="text-xs" style={{ color: "var(--text-muted)" }}>
            Revenue ativo: <span className="text-amber-400 font-semibold">{formatBRL(ttf.total_revenue_brl)}</span>
          </span>
        </CardHeader>

        <ResponsiveContainer width="100%" height={180}>
          <BarChart data={funnelData} layout="vertical">
            <XAxis type="number" tick={{ fontSize: 11 }} />
            <YAxis dataKey="stage" type="category" tick={{ fontSize: 11 }} width={120} />
            <Tooltip cursor={{ fill: "rgba(99,102,241,0.1)" }} />
            <Bar dataKey="value" fill="#6366f1" radius={[0, 4, 4, 0]} />
          </BarChart>
        </ResponsiveContainer>

        <div className="grid grid-cols-3 gap-2 text-xs mt-2">
          <RateBox label="Disclosure → Filed"  rate={ttf.rate_disclosure_to_file} />
          <RateBox label="Filed → Granted"     rate={ttf.rate_file_to_grant} />
          <RateBox label="Granted → Contract"  rate={ttf.rate_grant_to_contract} />
        </div>
      </Card>

      {/* HJT Diversity + Triple Helix side by side */}
      <div className="grid grid-cols-2 gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Network size={14} className="text-purple-400" />
              HJT IPC Diversity (light)
              <MetricTooltip metricID="hjt_diversity" />
            </CardTitle>
          </CardHeader>
          <div className="flex items-baseline gap-2 mb-2">
            <p className="text-4xl font-bold text-white">{div.diversity_index.toFixed(3)}</p>
            <p className="text-sm" style={{ color: "var(--text-muted)" }}>/ 0.875 max</p>
          </div>
          <p className="text-xs" style={{ color: "var(--text-muted)" }}>
            {div.ipc_categories_present} de 8 categorias IPC presentes ·
            Especialização: <span className="text-white">{div.specialization_index.toFixed(3)}</span>
          </p>
          <p className="text-xs mt-2" style={{ color: "var(--text-muted)" }}>
            D = 1 − Σⱼ (sⱼ)² · adaptado de Hall-Jaffe-Trajtenberg (2001)
          </p>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Network size={14} className="text-emerald-400" />
              Triple Helix Score
              <MetricTooltip metricID="triple_helix" />
            </CardTitle>
            <Badge variant={th.helix_score >= 60 ? "success" : "warning"}>
              {th.helix_score.toFixed(1)} / 100
            </Badge>
          </CardHeader>

          <ResponsiveContainer width="100%" height={130}>
            <RadialBarChart innerRadius="35%" outerRadius="100%" data={thData} startAngle={90} endAngle={-270}>
              <PolarAngleAxis type="number" domain={[0, 100]} tick={false} />
              <RadialBar background dataKey="value" cornerRadius={6} />
            </RadialBarChart>
          </ResponsiveContainer>

          <div className="grid grid-cols-3 gap-2 text-xs mt-2">
            <div className="text-center">
              <div className="w-2 h-2 rounded-full mx-auto" style={{ background: "#6366f1" }} />
              <p className="text-white font-semibold">{th.u_count}</p>
              <p style={{ color: "var(--text-muted)" }}>Univ.</p>
            </div>
            <div className="text-center">
              <div className="w-2 h-2 rounded-full mx-auto" style={{ background: "#f59e0b" }} />
              <p className="text-white font-semibold">{th.i_count}</p>
              <p style={{ color: "var(--text-muted)" }}>Indúst.</p>
            </div>
            <div className="text-center">
              <div className="w-2 h-2 rounded-full mx-auto" style={{ background: "#34d399" }} />
              <p className="text-white font-semibold">{th.g_count}</p>
              <p style={{ color: "var(--text-muted)" }}>Gov.</p>
            </div>
          </div>
        </Card>
      </div>

      {/* Knowledge Stock (Griliches 1990) */}
      {stock && stock.series.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Layers size={14} className="text-cyan-400" />
              Knowledge Stock (Capital de R&amp;D)
              <MetricTooltip metricID="knowledge_stock" />
            </CardTitle>
            <span className="text-xs" style={{ color: "var(--text-muted)" }}>
              Inventário perpétuo · δ = {(stock.depreciation_rate * 100).toFixed(0)}% · {stock.methodology}
            </span>
          </CardHeader>

          <ResponsiveContainer width="100%" height={180}>
            <AreaChart data={stock.series}>
              <defs>
                <linearGradient id="kstock" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#06b6d4" stopOpacity={0.4} />
                  <stop offset="95%" stopColor="#06b6d4" stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis dataKey="year" tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} />
              <Tooltip
                formatter={(v, name) =>
                  name === "knowledge_stock"
                    ? [Number(v).toFixed(2), "Stock"]
                    : [String(v), "Novas patentes"]
                } />
              <Area type="monotone" dataKey="knowledge_stock" stroke="#06b6d4" fill="url(#kstock)" strokeWidth={2} />
              <Area type="step"     dataKey="new_patents"     stroke="#6366f1" fill="none" strokeWidth={1.5} strokeDasharray="3 3" />
            </AreaChart>
          </ResponsiveContainer>

          <p className="text-xs mt-2" style={{ color: "var(--text-muted)" }}>
            <span className="text-white">Stock atual:</span>{" "}
            {stock.series[stock.series.length - 1]?.knowledge_stock.toFixed(2)} unidades de R&amp;D acumulado
            {" · "}
            <span className="text-cyan-400">━ stock</span>
            {" "}<span style={{ color: "#6366f1" }}>--- novas/ano</span>
          </p>
        </Card>
      )}

      {/* Department health breakdown */}
      {depts && depts.departments.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Building size={14} className="text-emerald-400" />
              Health Score por departamento UFOP
              <MetricTooltip metricID="autm_health_score" />
            </CardTitle>
          </CardHeader>

          <div className="space-y-2">
            {depts.departments.map(d => (
              <div key={d.department} className="grid grid-cols-12 items-center gap-3 p-2 rounded"
                style={{ background: "var(--surface-2)" }}>
                <span className="col-span-3 text-sm text-white truncate">{d.department}</span>
                <div className="col-span-5 h-2.5 rounded-full" style={{ background: "var(--border)" }}>
                  <div className="h-full rounded-full transition-all"
                    style={{
                      width: `${d.composite_score}%`,
                      background: d.composite_score >= 70 ? "#34d399" :
                                  d.composite_score >= 40 ? "#fbbf24" : "#f87171",
                    }} />
                </div>
                <span className="col-span-1 text-sm text-white font-semibold">{d.composite_score.toFixed(1)}</span>
                <span className="col-span-3 text-xs text-right" style={{ color: "var(--text-muted)" }}>
                  {d.patents} pat · {(d.license_rate*100).toFixed(0)}% lic · {formatBRL(d.revenue_per_asset_brl)}
                </span>
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Royalty Forecast (Pakes 1986) */}
      {forecast && forecast.years.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Coins size={14} className="text-amber-400" />
              Royalty Forecast — 10 anos
              <MetricTooltip metricID="royalty_forecast" />
            </CardTitle>
            <span className="text-xs" style={{ color: "var(--text-muted)" }}>
              NPV @ {(forecast.discount_rate * 100).toFixed(0)}%: <span className="text-amber-300 font-semibold">{formatBRL(forecast.total_npv_brl)}</span>
              {" · "}
              Total: {formatBRL(forecast.total_projected_brl)}
            </span>
          </CardHeader>

          <ResponsiveContainer width="100%" height={180}>
            <AreaChart data={forecast.years}>
              <defs>
                <linearGradient id="rev" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#fbbf24" stopOpacity={0.4} />
                  <stop offset="95%" stopColor="#fbbf24" stopOpacity={0} />
                </linearGradient>
                <linearGradient id="npv" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#34d399" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#34d399" stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis dataKey="year" tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} tickFormatter={v => `R$${(v/1000).toFixed(0)}k`} />
              <Tooltip
                formatter={(v, name) => [
                  `R$ ${Number(v).toLocaleString("pt-BR", { maximumFractionDigits: 0 })}`,
                  name === "expected_royalty_brl" ? "Receita anual" :
                  name === "expected_npv_brl"     ? "NPV anual"     : String(name),
                ]}
              />
              <Area type="monotone" dataKey="expected_royalty_brl" stroke="#fbbf24" fill="url(#rev)" strokeWidth={2} />
              <Area type="monotone" dataKey="expected_npv_brl"     stroke="#34d399" fill="url(#npv)" strokeWidth={1.5} strokeDasharray="3 3" />
            </AreaChart>
          </ResponsiveContainer>

          <p className="text-xs mt-2" style={{ color: "var(--text-muted)" }}>
            <span className="text-amber-400">━ receita</span>
            {" "}<span className="text-emerald-400">--- NPV descontado</span>
            {" · "}
            Premissa: {forecast.growth_assumption}
          </p>
        </Card>
      )}

      {/* Top inventors */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Award size={14} className="text-amber-400" />
            Top inventores (Hirsch-adapted)
            <MetricTooltip metricID="inventor_h" />
          </CardTitle>
        </CardHeader>

        {data.top_inventors.length === 0 ? (
          <p className="text-sm text-center py-4" style={{ color: "var(--text-muted)" }}>
            Sem dados de inventor.
          </p>
        ) : (
          <div className="space-y-1.5">
            {data.top_inventors.map((inv, i) => (
              <Link key={inv.name + i} href={`/inventors/${encodeURIComponent(inv.name)}`}>
                <div className="flex items-center gap-3 p-2 rounded transition-colors cursor-pointer hover:bg-white/5"
                  style={{ background: i === 0 ? "#fbbf2410" : "transparent" }}>
                  <span className="text-xs font-mono w-6 text-slate-500">#{i+1}</span>
                  <span className="text-sm font-medium text-white flex-1 truncate">{inv.name}</span>
                  <Badge variant="muted">{inv.total_patents} patentes</Badge>
                  <Badge variant="info">h ≈ {inv.h_index_proxy}</Badge>
                  <Badge variant="muted">{inv.ipc_breadth} cat IPC</Badge>
                </div>
              </Link>
            ))}
          </div>
        )}
      </Card>

      {/* Footnote — mock disclaimer */}
      <div className="text-xs text-center p-3 rounded"
        style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text-muted)" }}>
        <AlertCircle size={12} className="inline mr-1 text-amber-400" />
        Sem token Lens.org configurado: dados de citações são mock determinístico (calibrados via NBER 2001).
        Configure <span className="font-mono">LENS_API_TOKEN</span> para dados reais.
        <Link href="/metodologia" className="ml-1 text-indigo-400 hover:text-indigo-300">
          Ler metodologia <ArrowRight size={9} className="inline" />
        </Link>
      </div>
    </div>
  );
}

// ─── helpers ─────────────────────────────────────────────────────────────────

function scoreColor(s: number): string {
  if (s >= 70) return "#34d399";
  if (s >= 50) return "#fbbf24";
  if (s >= 30) return "#f59e0b";
  return "#ef4444";
}

function Indicator({ label, value, tip, benchmark }: {
  label: string; value: string; tip: string; benchmark?: string;
}) {
  return (
    <div className="p-2.5 rounded-lg" style={{ background: "var(--surface-2)" }}>
      <p className="text-xs flex items-center gap-1" style={{ color: "var(--text-muted)" }}>
        {label} <MetricTooltip metricID={tip} />
      </p>
      <p className="text-lg font-semibold text-white mt-0.5">{value}</p>
      {benchmark && (
        <p className="text-xs mt-0.5" style={{ color: "var(--text-muted)" }}>{benchmark}</p>
      )}
    </div>
  );
}

function RateBox({ label, rate }: { label: string; rate: number }) {
  const color = rate >= 0.7 ? "#34d399" : rate >= 0.4 ? "#fbbf24" : "#f87171";
  return (
    <div className="p-2 rounded text-center" style={{ background: "var(--surface-2)" }}>
      <p style={{ color: "var(--text-muted)" }}>{label}</p>
      <p className="text-base font-bold" style={{ color }}>{(rate*100).toFixed(1)}%</p>
    </div>
  );
}
