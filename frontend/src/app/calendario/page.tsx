"use client";

// /calendario — calendário NIT-UFOP auto-populado com:
//   - Anuidades INPI por patente (table MPE 2024)
//   - Renovações decenais de marcas
//   - Milestones de contratos TT (de tt_contracts.milestones JSONB)
//   - Prazos arbitrais (opened_at + 90d default)
//
// Visualização: agenda cronológica (lista) + cards de KPI por kind.

import { useMemo, useState } from "react";
import Link from "next/link";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SkeletonList } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { useCalendar } from "@/lib/hooks";
import { formatBRL } from "@/lib/utils";
import type { CalendarEvent } from "@/lib/types";
import {
  Calendar, AlertCircle, Coins, RefreshCw, Scale,
  FileText, Briefcase, Filter,
} from "lucide-react";

const kindMeta: Record<string, { label: string; color: string; icon: typeof Calendar }> = {
  annuity:   { label: "Anuidade INPI",  color: "#fbbf24", icon: Coins },
  renewal:   { label: "Renovação marca", color: "#f59e0b", icon: RefreshCw },
  milestone: { label: "Milestone TT",   color: "#34d399", icon: Briefcase },
  dispute:   { label: "Prazo arbitral", color: "#ef4444", icon: Scale },
  filing:    { label: "Depósito",       color: "#6366f1", icon: FileText },
};

const priorityMeta: Record<string, { label: string; color: string }> = {
  critical: { label: "🔴 Crítico", color: "#ef4444" },
  high:     { label: "🟠 Alto",    color: "#f59e0b" },
  medium:   { label: "🟡 Médio",   color: "#fbbf24" },
  low:      { label: "🟢 Baixo",   color: "#34d399" },
};

export default function CalendarioPage() {
  const { data, isLoading } = useCalendar();
  const [kindFilter, setKindFilter] = useState<string>("all");
  const [priorityFilter, setPriorityFilter] = useState<string>("all");

  const events = useMemo<CalendarEvent[]>(() => {
    const all = data?.events ?? [];
    return all.filter(e =>
      (kindFilter === "all" || e.kind === kindFilter) &&
      (priorityFilter === "all" || e.priority === priorityFilter)
    );
  }, [data, kindFilter, priorityFilter]);

  const totalAmount = events
    .filter(e => e.kind === "annuity" || e.kind === "renewal")
    .reduce((s, e) => s + (e.amount_brl ?? 0), 0);

  const criticalCount = events.filter(e => e.priority === "critical").length;

  // Group by month for display
  const byMonth = useMemo(() => {
    const m: Record<string, CalendarEvent[]> = {};
    for (const e of events) {
      const key = e.date.substring(0, 7); // "2026-06"
      if (!m[key]) m[key] = [];
      m[key].push(e);
    }
    return m;
  }, [events]);

  return (
    <div className="p-8 space-y-6 max-w-6xl fade-in">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white flex items-center gap-2">
          <Calendar size={22} />
          Calendário NIT-UFOP
        </h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Deadlines auto-gerados de anuidades INPI, renovações de marca,
          milestones de contratos TT e prazos arbitrais.
        </p>
      </div>

      {/* KPIs */}
      <div className="grid grid-cols-4 gap-4">
        <KPI label="Total de eventos"  value={(data?.count ?? 0).toString()}  sub="próximos 365 dias" color="#6366f1" />
        <KPI label="Críticos"          value={criticalCount.toString()}        sub="≤ 30 dias"        color="#ef4444" />
        <KPI label="Anuidades INPI"    value={(data?.by_kind.annuity ?? 0).toString()} sub="vencendo no período" color="#fbbf24" />
        <KPI label="Compromisso $$$"   value={formatBRL(totalAmount)}          sub="custo agregado"   color="#34d399" />
      </div>

      {/* Filters */}
      <Card>
        <div className="flex items-center gap-3 flex-wrap">
          <div className="flex items-center gap-2">
            <Filter size={12} className="text-slate-500" />
            <span className="text-xs" style={{ color: "var(--text-muted)" }}>Tipo:</span>
          </div>
          <FilterChip active={kindFilter === "all"}     label="Todos"               onClick={() => setKindFilter("all")} />
          {Object.entries(kindMeta).map(([k, m]) => {
            const count = data?.by_kind[k] ?? 0;
            return (
              <FilterChip key={k}
                active={kindFilter === k}
                label={`${m.label} (${count})`}
                color={m.color}
                onClick={() => setKindFilter(k)} />
            );
          })}
          <div className="flex items-center gap-2 ml-4 pl-4" style={{ borderLeft: "1px solid var(--border)" }}>
            <span className="text-xs" style={{ color: "var(--text-muted)" }}>Prioridade:</span>
            <FilterChip active={priorityFilter === "all"}      label="Todas"   onClick={() => setPriorityFilter("all")} />
            <FilterChip active={priorityFilter === "critical"} label="Crítico" color="#ef4444" onClick={() => setPriorityFilter("critical")} />
            <FilterChip active={priorityFilter === "high"}     label="Alto"    color="#f59e0b" onClick={() => setPriorityFilter("high")} />
          </div>
        </div>
      </Card>

      {/* Events list */}
      {isLoading && <SkeletonList count={4} />}

      {!isLoading && events.length === 0 && (
        <Card>
          <EmptyState
            icon={Calendar}
            title="Nenhum evento no filtro atual"
            description="Tente outro filtro ou aguarde — o calendário se popula automaticamente."
          />
        </Card>
      )}

      {Object.entries(byMonth).map(([monthKey, monthEvents]) => (
        <div key={monthKey}>
          <h2 className="text-sm font-semibold text-white mb-2 flex items-center gap-2 sticky top-0 z-10 py-1"
            style={{ background: "var(--bg)" }}>
            <Calendar size={12} className="text-indigo-400" />
            {formatMonth(monthKey)}
            <Badge variant="muted">{monthEvents.length}</Badge>
          </h2>
          <div className="space-y-2">
            {monthEvents.map(e => <EventCard key={e.id} event={e} />)}
          </div>
        </div>
      ))}

      {/* Footer */}
      <div className="text-xs text-center p-3 rounded mt-8"
        style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text-muted)" }}>
        Eventos gerados automaticamente a partir do banco. Anuidades calculadas
        pela tabela INPI MPE 2024 (R$ 310 anos 3-6, R$ 620 anos 7-10, etc).
        Prazos arbitrais padrão de 90 dias (configurável por disputa).
      </div>
    </div>
  );
}

// ─── sub-components ─────────────────────────────────────────────────────────

function KPI({ label, value, sub, color }: { label: string; value: string; sub: string; color: string }) {
  return (
    <Card>
      <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>{label}</p>
      <p className="text-2xl font-bold text-white">{value}</p>
      <p className="text-xs mt-1" style={{ color }}>{sub}</p>
    </Card>
  );
}

function FilterChip({ active, label, color, onClick }: {
  active: boolean; label: string; color?: string; onClick: () => void;
}) {
  return (
    <button onClick={onClick}
      className="px-2.5 py-1 rounded-full text-xs transition-all"
      style={{
        background: active ? (color ?? "var(--accent)") : "var(--surface-2)",
        color: active ? "white" : "var(--text-muted)",
        border: `1px solid ${active ? (color ?? "var(--accent)") : "var(--border)"}`,
      }}>
      {label}
    </button>
  );
}

function EventCard({ event }: { event: CalendarEvent }) {
  const meta = kindMeta[event.kind] ?? kindMeta.filing;
  const Icon = meta.icon;
  const prio = priorityMeta[event.priority] ?? priorityMeta.medium;
  const dayMonth = new Date(event.date).toLocaleDateString("pt-BR", {
    day: "2-digit", month: "short",
  });
  const isCritical = event.priority === "critical";

  const content = (
    <Card style={{ borderColor: isCritical ? "#ef4444" + "40" : "var(--border)" }}>
      <div className="flex items-center gap-3">
        {/* Date column */}
        <div className="shrink-0 text-center w-12">
          <p className="text-xs uppercase font-mono" style={{ color: "var(--text-muted)" }}>
            {dayMonth.split(" ")[1].replace(".", "")}
          </p>
          <p className="text-xl font-bold text-white">{dayMonth.split(" ")[0]}</p>
        </div>

        <div className="shrink-0 p-2 rounded-lg"
          style={{ background: meta.color + "20", border: `1px solid ${meta.color}40` }}>
          <Icon size={14} style={{ color: meta.color }} />
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-0.5 flex-wrap">
            <Badge variant="muted">{meta.label}</Badge>
            <span className="text-xs" style={{ color: prio.color }}>{prio.label}</span>
            {event.entity_ref && (
              <span className="font-mono text-xs text-indigo-400">{event.entity_ref}</span>
            )}
          </div>
          <p className="text-sm text-white truncate">{event.title}</p>
          {event.description && (
            <p className="text-xs mt-0.5 truncate" style={{ color: "var(--text-muted)" }}>
              {event.description}
            </p>
          )}
        </div>

        {event.amount_brl != null && event.amount_brl > 0 && (
          <div className="text-right shrink-0">
            <p className="text-sm font-semibold text-amber-300">{formatBRL(event.amount_brl)}</p>
          </div>
        )}
      </div>
    </Card>
  );

  if (event.url) {
    return <Link href={event.url} className="block hover:scale-[1.005] transition-transform">{content}</Link>;
  }
  return content;
}

function formatMonth(yyyyMm: string): string {
  const [y, m] = yyyyMm.split("-");
  const d = new Date(Number(y), Number(m) - 1, 1);
  return d.toLocaleDateString("pt-BR", { month: "long", year: "numeric" });
}
