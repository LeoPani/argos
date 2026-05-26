"use client";

// /inventors/[name] — perfil produtivo do inventor.
// h-index proxy (Hirsch 2005 / Wong-Pang 2011), portfolio,
// IPC distribution, co-inventores e estimativa de royalty
// devido por Lei 10.973/2004.

import { use } from "react";
import Link from "next/link";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SkeletonKPI, SkeletonList } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { MetricTooltip } from "@/components/ui/metric-tooltip";
import { useInventorProfile } from "@/lib/hooks";
import { formatBRL, ipcLabel } from "@/lib/utils";
import {
  ArrowLeft, User, Award, BookOpen, Network,
  Coins, AlertCircle, Calendar,
} from "lucide-react";

const ipcColors: Record<string, string> = {
  A: "#ef4444", B: "#f59e0b", C: "#34d399", D: "#06b6d4",
  E: "#3b82f6", F: "#8b5cf6", G: "#a855f7", H: "#ec4899",
};

export default function InventorPage({ params }: { params: Promise<{ name: string }> }) {
  const { name: rawName } = use(params);
  const name = decodeURIComponent(rawName);
  const { data: profile, error, isLoading } = useInventorProfile(name);

  if (isLoading) {
    return (
      <div className="p-8 space-y-6">
        <Breadcrumb backTo="/metricas" current="Carregando…" />
        <SkeletonKPI />
        <SkeletonList count={3} />
      </div>
    );
  }

  if (error || !profile) {
    return (
      <div className="p-8 space-y-6">
        <Breadcrumb backTo="/metricas" current="Não encontrado" />
        <Card>
          <EmptyState
            icon={AlertCircle}
            title="Inventor não encontrado"
            description={`Nenhuma patente registrada para "${name}".`}
          />
        </Card>
      </div>
    );
  }

  const ipcEntries = Object.entries(profile.ipc_distribution).sort((a, b) => b[1] - a[1]);
  const total = ipcEntries.reduce((s, [, c]) => s + c, 0);

  return (
    <div className="p-8 space-y-6 max-w-5xl fade-in">
      <Breadcrumb backTo="/metricas" current={profile.name} />

      {/* Header */}
      <Card>
        <div className="flex items-start gap-4">
          <div className="p-3 rounded-xl shrink-0"
            style={{ background: "#6366f120", border: "1px solid #6366f140" }}>
            <User size={22} className="text-indigo-400" />
          </div>
          <div className="flex-1">
            <h1 className="text-2xl font-bold text-white">{profile.name}</h1>
            {profile.filing_year_span && (
              <p className="text-xs mt-1 flex items-center gap-1" style={{ color: "var(--text-muted)" }}>
                <Calendar size={11} />
                Atividade: {profile.filing_year_span}
              </p>
            )}
          </div>
        </div>

        {/* KPIs */}
        <div className="grid grid-cols-5 gap-3 mt-4">
          <KPI label="Total de patentes"    value={profile.total_patents.toString()}     icon={BookOpen} color="#6366f1" />
          <KPI label="Concedidas"           value={profile.granted_patents.toString()}    icon={Award}    color="#34d399" />
          <KPI label="h-index proxy"        value={`h ≈ ${profile.h_index_proxy}`}        icon={Award}    color="#fbbf24" tip="inventor_h" />
          <KPI label="Amplitude IPC"        value={`${profile.ipc_breadth} cat`}          icon={Network}  color="#a855f7" />
          <KPI label="Royalty estimado"     value={formatBRL(profile.estimated_royalty_brl)} icon={Coins} color="#f59e0b" tip="inventor_profile" />
        </div>
      </Card>

      {/* IPC distribution + coinventors */}
      <div className="grid grid-cols-2 gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Network size={14} className="text-purple-400" />
              Distribuição IPC
            </CardTitle>
          </CardHeader>
          {ipcEntries.length === 0 ? (
            <p className="text-xs" style={{ color: "var(--text-muted)" }}>Sem dados de IPC.</p>
          ) : (
            <div className="space-y-2">
              {ipcEntries.map(([letter, count]) => {
                const pct = total > 0 ? (count / total) * 100 : 0;
                return (
                  <div key={letter}>
                    <div className="flex justify-between text-xs mb-1">
                      <span className="text-white font-medium">{letter} — Classe IPC</span>
                      <span style={{ color: "var(--text-muted)" }}>{count} ({pct.toFixed(0)}%)</span>
                    </div>
                    <div className="h-2 rounded-full" style={{ background: "var(--border)" }}>
                      <div className="h-full rounded-full" style={{
                        width: `${pct}%`, background: ipcColors[letter] ?? "#6366f1",
                      }} />
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Network size={14} className="text-emerald-400" />
              Top co-inventores
            </CardTitle>
          </CardHeader>
          {profile.coinventors.length === 0 ? (
            <p className="text-xs py-2" style={{ color: "var(--text-muted)" }}>
              Inventor solo — sem co-autoria registrada.
            </p>
          ) : (
            <div className="space-y-1.5">
              {profile.coinventors.map((co, i) => (
                <Link key={co.name} href={`/inventors/${encodeURIComponent(co.name)}`}>
                  <div className="flex items-center justify-between p-2 rounded transition-colors cursor-pointer hover:bg-white/5"
                    style={{ background: "var(--surface-2)" }}>
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-mono text-slate-500">#{i+1}</span>
                      <User size={11} className="text-indigo-400" />
                      <span className="text-sm text-white">{co.name}</span>
                    </div>
                    <Badge variant="muted">{co.co_patent_count}× juntos</Badge>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </Card>
      </div>

      {/* Patents list */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BookOpen size={14} className="text-indigo-400" />
            Portfolio ({profile.patents.length})
          </CardTitle>
        </CardHeader>
        <div className="space-y-1">
          {profile.patents.map(p => (
            <Link key={p.id} href={`/patents/${p.id}`}>
              <div className="flex items-center gap-3 p-2 rounded hover:bg-white/5 transition-colors cursor-pointer">
                <Badge variant={p.status === "classified" ? "success" : "warning"}>
                  {p.status}
                </Badge>
                <span className="font-mono text-xs text-indigo-400 shrink-0">{p.application_number}</span>
                <span className="text-sm text-white flex-1 truncate">{p.title}</span>
                {p.ipc_category >= 0 && (
                  <Badge variant="muted">{ipcLabel(p.ipc_category)}</Badge>
                )}
                {p.filing_year > 0 && (
                  <span className="text-xs shrink-0" style={{ color: "var(--text-muted)" }}>
                    {p.filing_year}
                  </span>
                )}
              </div>
            </Link>
          ))}
        </div>
      </Card>

      {/* Methodology footnote */}
      <div className="text-xs text-center p-3 rounded"
        style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text-muted)" }}>
        Royalty estimado pela Lei n. 10.973/2004 (Marco Legal C&amp;T):
        inventor_share_pct do contrato TT × valor recebido pela UFOP.
        <Link href="/metodologia#inventor_profile" className="ml-1 text-indigo-400 hover:text-indigo-300">
          Ver metodologia
        </Link>
      </div>
    </div>
  );
}

// ─── helpers ──────────────────────────────────────────────────────────────────

function Breadcrumb({ backTo, current }: { backTo: string; current: string }) {
  return (
    <div className="flex items-center gap-2 text-sm">
      <Link href={backTo} className="hover:text-white transition-colors"
        style={{ color: "var(--text-muted)" }}>
        <ArrowLeft size={14} className="inline mr-1" />
        Métricas
      </Link>
      <span style={{ color: "var(--text-muted)" }}>/</span>
      <span className="text-white">{current}</span>
    </div>
  );
}

function KPI({ label, value, icon: Icon, color, tip }: {
  label: string; value: string; icon: typeof User; color: string; tip?: string;
}) {
  return (
    <div className="rounded-lg p-3" style={{ background: "var(--surface-2)" }}>
      <div className="flex items-center gap-1.5 text-xs mb-1" style={{ color: "var(--text-muted)" }}>
        <Icon size={11} style={{ color }} />
        {label}
        {tip && <MetricTooltip metricID={tip} />}
      </div>
      <p className="text-lg font-bold text-white">{value}</p>
    </div>
  );
}
