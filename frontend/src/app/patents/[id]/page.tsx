"use client";

import { use } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SkeletonKPI, SkeletonList } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { usePatent, useTTContracts, useDisputes, useMaintenance, useCitationNetwork } from "@/lib/hooks";
import { CitationNetworkViz } from "@/components/ui/citation-network";
import { formatDate, formatBRL, ipcLabel } from "@/lib/utils";
import type { Dispute } from "@/lib/types";
import { MetricTooltip } from "@/components/ui/metric-tooltip";
import {
  ArrowLeft, FileText, User, Calendar, Hash,
  BookOpen, Briefcase, Scale, Layers, AlertCircle,
  Lightbulb, TrendingDown, RefreshCw,
} from "lucide-react";

export default function PatentDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id: idStr } = use(params);
  const id = Number(idStr);
  const router = useRouter();

  const { data: patent, error, isLoading } = usePatent(id);
  const { data: contractsData } = useTTContracts({ patent_id: String(id), limit: "20" });
  const { data: disputesData }  = useDisputes({ limit: "200" });
  const { data: maintenance }   = useMaintenance(id);
  const { data: network }       = useCitationNetwork(id);

  // Disputes that mention this patent (best-effort: matches kind=patent_infringement and we'd ideally hit /patents/{id}/disputes — for now use title/summary heuristic)
  const relatedDisputes: Dispute[] = (disputesData?.items ?? []).filter(
    d => d.title.includes(patent?.application_number ?? "@@") ||
         d.summary.includes(patent?.application_number ?? "@@")
  );

  const contracts = contractsData?.items ?? [];

  if (isLoading) {
    return (
      <div className="p-8 space-y-6">
        <Breadcrumb backTo="/portfolio" current="Carregando…" />
        <SkeletonKPI />
        <SkeletonList count={2} />
      </div>
    );
  }

  if (error || !patent) {
    return (
      <div className="p-8 space-y-6">
        <Breadcrumb backTo="/portfolio" current="Não encontrado" />
        <Card>
          <EmptyState
            icon={AlertCircle}
            title="Patente não encontrada"
            description={`Nenhuma patente com id ${id} no banco.`}
            action={{ label: "Voltar ao portfolio", onClick: () => router.push("/portfolio"), icon: ArrowLeft }}
          />
        </Card>
      </div>
    );
  }

  return (
    <div className="p-8 space-y-6 fade-in">
      <Breadcrumb backTo="/portfolio" current={patent.application_number} />

      {/* Header */}
      <Card>
        <div className="flex items-start justify-between gap-3 mb-3">
          <div className="flex-1">
            <div className="flex items-center gap-2 mb-1 flex-wrap">
              <p className="font-mono text-sm text-indigo-400">{patent.application_number}</p>
              <Badge variant={
                patent.status === "classified" ? "success" :
                patent.status === "failed" ? "danger" : "warning"
              }>
                {patent.status}
              </Badge>
              {patent.ipc_category !== null && (
                <Badge variant="default">IPC {ipcLabel(patent.ipc_category)}</Badge>
              )}
              {patent.ipc_code && <Badge variant="muted">{patent.ipc_code}</Badge>}
            </div>
            <h1 className="text-2xl font-bold text-white leading-tight">{patent.title}</h1>
          </div>
        </div>

        <div className="grid grid-cols-4 gap-3 mt-4 text-sm">
          <Field icon={User}      label="Titular"            value={patent.applicant} />
          <Field icon={Calendar}  label="Data de depósito"   value={formatDate(patent.filing_date)} />
          <Field icon={Calendar}  label="Publicação"         value={formatDate(patent.publication_date)} />
          <Field icon={Hash}      label="Revista (RPI)"      value={patent.rpi_issue || "—"} />
        </div>

        {patent.inventors.length > 0 && (
          <div className="mt-3 pt-3" style={{ borderTop: "1px solid var(--border)" }}>
            <p className="text-xs mb-1.5" style={{ color: "var(--text-muted)" }}>Inventores</p>
            <div className="flex gap-1.5 flex-wrap">
              {patent.inventors.map(inv => (
                <Link key={inv} href={`/inventors/${encodeURIComponent(inv)}`}>
                  <Badge variant="info">
                    <User size={10} /> {inv}
                  </Badge>
                </Link>
              ))}
            </div>
          </div>
        )}
      </Card>

      {/* Maintenance recommendation (Schankerman-Pakes 1986) */}
      {maintenance && <MaintenanceCard m={maintenance} /> }

      {/* Citation Network (Narin 1994) */}
      {network && network.stats.node_count > 1 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Layers size={14} className="text-purple-400" />
              Citation Network
              <Badge variant="muted">{network.stats.node_count} nós</Badge>
            </CardTitle>
          </CardHeader>
          <CitationNetworkViz network={network} />
        </Card>
      )}

      {/* Abstract */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BookOpen size={14} className="text-indigo-400" />
            Resumo
          </CardTitle>
        </CardHeader>
        {patent.abstract ? (
          <p className="text-sm leading-relaxed" style={{ color: "var(--text-muted)" }}>
            {patent.abstract}
          </p>
        ) : (
          <p className="text-xs italic" style={{ color: "var(--text-muted)" }}>
            Resumo não disponível.
          </p>
        )}
      </Card>

      {/* Related TT contracts */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Briefcase size={14} className="text-emerald-400" />
            Contratos TT vinculados ({contracts.length})
          </CardTitle>
          <Link href={`/pool`}>
            <Button variant="ghost" size="sm">Ver todos</Button>
          </Link>
        </CardHeader>
        {contracts.length === 0 ? (
          <EmptyState
            icon={Briefcase}
            title="Nenhum contrato vinculado"
            description="Esta patente ainda não foi licenciada via contrato TT."
            size="sm"
          />
        ) : (
          <div className="space-y-2">
            {contracts.map(c => (
              <div key={c.id} className="p-2.5 rounded-lg flex items-center justify-between"
                style={{ background: "var(--surface-2)" }}>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-xs text-indigo-400">{c.contract_number}</span>
                    <Badge variant={c.status === "active" ? "success" : "muted"}>{c.status}</Badge>
                    <Badge variant="muted">{c.license_kind}</Badge>
                  </div>
                  <p className="text-sm text-white truncate">{c.licensee}</p>
                </div>
                <div className="text-right shrink-0 ml-3">
                  <p className="text-sm font-semibold text-amber-300">{c.royalty_rate}%</p>
                  <p className="text-xs" style={{ color: "var(--text-muted)" }}>
                    + {formatBRL(c.upfront_fee)} upfront
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      {/* Related disputes */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Scale size={14} className="text-red-400" />
            Disputas relacionadas ({relatedDisputes.length})
          </CardTitle>
        </CardHeader>
        {relatedDisputes.length === 0 ? (
          <EmptyState
            icon={Scale}
            title="Sem disputas registradas"
            description="Esta patente não aparece em nenhum caso de arbitragem."
            size="sm"
          />
        ) : (
          <div className="space-y-2">
            {relatedDisputes.map(d => (
              <Link key={d.id} href="/arbitragem">
                <div className="p-2.5 rounded-lg cursor-pointer hover:bg-white/5 transition-colors"
                  style={{ background: "var(--surface-2)" }}>
                  <div className="flex items-center gap-2 mb-0.5">
                    <span className="font-mono text-xs text-indigo-400">{d.case_number}</span>
                    <Badge variant="muted">{d.status}</Badge>
                  </div>
                  <p className="text-sm text-white">{d.title}</p>
                </div>
              </Link>
            ))}
          </div>
        )}
      </Card>

      {/* Metadata */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Layers size={13} className="text-slate-500" />
            <span className="text-xs" style={{ color: "var(--text-muted)" }}>Metadados</span>
          </CardTitle>
        </CardHeader>
        <div className="grid grid-cols-2 gap-2 text-xs" style={{ color: "var(--text-muted)" }}>
          <span>ID interno: <span className="text-white font-mono">#{patent.id}</span></span>
          <span>Categoria IPC: <span className="text-white">{patent.ipc_category ?? "—"}</span></span>
          <span>Criado em: <span className="text-white">{formatDate(patent.created_at)}</span></span>
          <span>Atualizado: <span className="text-white">{formatDate(patent.updated_at)}</span></span>
        </div>
      </Card>
    </div>
  );
}

// ─── helpers ─────────────────────────────────────────────────────────────────

function Breadcrumb({ backTo, current }: { backTo: string; current: string }) {
  return (
    <div className="flex items-center gap-2 text-sm">
      <Link href={backTo} className="hover:text-white transition-colors"
        style={{ color: "var(--text-muted)" }}>
        <ArrowLeft size={14} className="inline mr-1" />
        Portfolio
      </Link>
      <span style={{ color: "var(--text-muted)" }}>/</span>
      <span className="text-white font-mono text-xs">{current}</span>
    </div>
  );
}

function MaintenanceCard({ m }: { m: import("@/lib/types").MaintenanceRecommendation }) {
  const recMeta = {
    keep:    { icon: RefreshCw,    label: "Manter (Keep)",      color: "#34d399" },
    license: { icon: Lightbulb,    label: "Buscar licenciado",   color: "#fbbf24" },
    abandon: { icon: TrendingDown, label: "Abandonar",           color: "#ef4444" },
  }[m.recommendation];
  const Icon = recMeta.icon;

  return (
    <Card style={{ borderColor: recMeta.color + "40" }}>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Icon size={14} style={{ color: recMeta.color }} />
          Recomendação de manutenção
          <MetricTooltip metricID="maintenance_decision" />
        </CardTitle>
        <Badge variant="muted">{m.confidence}% confiança</Badge>
      </CardHeader>

      <div className="p-3 rounded-lg mb-3"
        style={{ background: recMeta.color + "15", border: `1px solid ${recMeta.color}40` }}>
        <div className="flex items-center gap-3">
          <Icon size={18} style={{ color: recMeta.color }} />
          <div>
            <p className="text-base font-semibold" style={{ color: recMeta.color }}>
              {recMeta.label}
            </p>
            <p className="text-xs" style={{ color: "var(--text-muted)" }}>
              Idade: {m.age_years}a · Restante: {m.remaining_years}a
            </p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-4 gap-3 mb-3 text-sm">
        <Metric label="Próxima anuidade"   value={formatBRL(m.next_annuity_brl)} />
        <Metric label="Custo restante"     value={formatBRL(m.total_remaining_cost_brl)} />
        <Metric label="Receita até hoje"   value={formatBRL(m.revenue_so_far_brl)} highlight />
        <Metric label="NPV esperado"       value={formatBRL(m.expected_npv_brl)} highlight />
      </div>

      <div>
        <p className="text-xs font-medium text-white mb-1">Justificativa</p>
        <ul className="space-y-1">
          {m.reasoning.map((r, i) => (
            <li key={i} className="text-xs leading-relaxed" style={{ color: "var(--text-muted)" }}>
              · {r}
            </li>
          ))}
        </ul>
      </div>
    </Card>
  );
}

function Metric({ label, value, highlight }: { label: string; value: string; highlight?: boolean }) {
  return (
    <div className="p-2 rounded" style={{ background: "var(--surface-2)" }}>
      <p className="text-xs" style={{ color: "var(--text-muted)" }}>{label}</p>
      <p className={`text-sm ${highlight ? "text-amber-300 font-semibold" : "text-white"}`}>{value}</p>
    </div>
  );
}

function Field({ icon: Icon, label, value }: { icon: typeof FileText; label: string; value: string }) {
  return (
    <div className="flex items-start gap-2">
      <Icon size={13} className="text-slate-500 mt-0.5 shrink-0" />
      <div className="min-w-0">
        <p className="text-xs" style={{ color: "var(--text-muted)" }}>{label}</p>
        <p className="text-sm text-white truncate">{value}</p>
      </div>
    </div>
  );
}
