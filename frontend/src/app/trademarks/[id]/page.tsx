"use client";

import { use } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SkeletonKPI, SkeletonList } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { useTrademark, useDisputes } from "@/lib/hooks";
import { formatDate } from "@/lib/utils";
import type { Dispute, TrademarkStatus, TrademarkKind } from "@/lib/types";
import {
  ArrowLeft, Tag, User, Calendar, Hash, Scale,
  AlertCircle, Layers, Building2,
} from "lucide-react";

const statusMeta: Record<TrademarkStatus, { label: string; variant: "info" | "warning" | "success" | "muted" | "danger" }> = {
  filed:     { label: "Depositada",      variant: "warning" },
  published: { label: "Em publicação",   variant: "info"    },
  granted:   { label: "Registrada",      variant: "success" },
  denied:    { label: "Indeferida",      variant: "danger"  },
  archived:  { label: "Arquivada",       variant: "muted"   },
  expired:   { label: "Extinta",         variant: "muted"   },
};

const kindLabel: Record<TrademarkKind, string> = {
  nominative:        "Nominativa",
  figurative:        "Figurativa",
  mixed:             "Mista",
  three_dimensional: "Tridimensional",
};

export default function TrademarkDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id: idStr } = use(params);
  const id = Number(idStr);
  const router = useRouter();

  const { data: trademark, error, isLoading } = useTrademark(id);
  const { data: disputesData } = useDisputes({ limit: "200" });

  const relatedDisputes: Dispute[] = (disputesData?.items ?? []).filter(
    d => trademark && (
      d.title.includes(trademark.name) || d.summary.includes(trademark.name)
    )
  );

  if (isLoading) {
    return (
      <div className="p-8 space-y-6">
        <Breadcrumb backTo="/portfolio" current="Carregando…" />
        <SkeletonKPI />
        <SkeletonList count={2} />
      </div>
    );
  }

  if (error || !trademark) {
    return (
      <div className="p-8 space-y-6">
        <Breadcrumb backTo="/portfolio" current="Não encontrado" />
        <Card>
          <EmptyState
            icon={AlertCircle}
            title="Marca não encontrada"
            description={`Nenhuma marca com id ${id} no banco.`}
            action={{ label: "Voltar ao portfolio", onClick: () => router.push("/portfolio"), icon: ArrowLeft }}
          />
        </Card>
      </div>
    );
  }

  const meta = statusMeta[trademark.status] ?? statusMeta.filed;
  const expiresAt = trademark.granted_date
    ? addYears(trademark.granted_date, 10)
    : trademark.filing_date
      ? addYears(trademark.filing_date, 10)
      : null;

  return (
    <div className="p-8 space-y-6 fade-in">
      <Breadcrumb backTo="/portfolio" current={trademark.process_number} />

      {/* Header */}
      <Card>
        <div className="flex items-start gap-4 mb-3">
          <div className="p-3 rounded-xl shrink-0"
            style={{ background: "#f59e0b20", border: "1px solid #f59e0b40" }}>
            <Tag size={20} className="text-orange-400" />
          </div>
          <div className="flex-1">
            <div className="flex items-center gap-2 mb-1 flex-wrap">
              <p className="font-mono text-sm text-indigo-400">{trademark.process_number}</p>
              <Badge variant={meta.variant}>{meta.label}</Badge>
              <Badge variant="muted">{kindLabel[trademark.kind]}</Badge>
              {trademark.rpi_issue && <Badge variant="muted">RPI {trademark.rpi_issue}</Badge>}
            </div>
            <h1 className="text-2xl font-bold text-white leading-tight">{trademark.name}</h1>
            <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>
              Normalizada: <span className="font-mono">{trademark.normalized_name}</span>
            </p>
          </div>
        </div>

        <div className="grid grid-cols-4 gap-3 mt-4 text-sm">
          <Field icon={Building2} label="Titular"        value={trademark.owner} />
          <Field icon={Calendar}  label="Depósito"       value={formatDate(trademark.filing_date)} />
          <Field icon={Calendar}  label="Publicação"     value={formatDate(trademark.publication_date)} />
          <Field icon={Calendar}  label="Concessão"      value={formatDate(trademark.granted_date)} />
        </div>

        {trademark.nice_classes.length > 0 && (
          <div className="mt-3 pt-3" style={{ borderTop: "1px solid var(--border)" }}>
            <p className="text-xs mb-1.5" style={{ color: "var(--text-muted)" }}>
              Classes Nice ({trademark.nice_classes.length})
            </p>
            <div className="flex gap-1.5 flex-wrap">
              {trademark.nice_classes.map(c => (
                <Badge key={c} variant="info">Classe {c}</Badge>
              ))}
            </div>
          </div>
        )}

        {expiresAt && (
          <div className="mt-3 pt-3" style={{ borderTop: "1px solid var(--border)" }}>
            <p className="text-xs" style={{ color: "var(--text-muted)" }}>
              Próxima renovação (10 anos a partir da concessão): <span className="text-white">{formatDate(expiresAt)}</span>
            </p>
          </div>
        )}
      </Card>

      {/* Related disputes */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Scale size={14} className="text-red-400" />
            Disputas envolvendo esta marca ({relatedDisputes.length})
          </CardTitle>
          <Link href="/arbitragem">
            <Button variant="ghost" size="sm">Ir para Arbitragem</Button>
          </Link>
        </CardHeader>
        {relatedDisputes.length === 0 ? (
          <EmptyState
            icon={Scale}
            title="Sem disputas registradas"
            description="Esta marca não aparece em nenhum caso de arbitragem."
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
          <span>ID interno: <span className="text-white font-mono">#{trademark.id}</span></span>
          <span>Status INPI: <span className="text-white">{trademark.status}</span></span>
          <span>Criado em: <span className="text-white">{formatDate(trademark.created_at)}</span></span>
          <span>Atualizado: <span className="text-white">{formatDate(trademark.updated_at)}</span></span>
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

function Field({ icon: Icon, label, value }: { icon: typeof Tag; label: string; value: string }) {
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

function addYears(date: string, years: number): string {
  const d = new Date(date);
  d.setFullYear(d.getFullYear() + years);
  return d.toISOString();
}
