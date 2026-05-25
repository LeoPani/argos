"use client";

// /tt-contract/new?from_ufop={id}
// Tela de geração assistida de contrato TT a partir de uma oportunidade
// UFOP. Mostra a tecnologia original, termos comerciais sugeridos,
// patentes UFOP relacionadas na área e o rascunho do contrato em Markdown.

import { Suspense } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SkeletonKPI, SkeletonList } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { useToast } from "@/components/ui/toast";
import { useTTTemplateFromUFOP } from "@/lib/hooks";
import { formatBRL } from "@/lib/utils";
import {
  ArrowLeft, FileSignature, Copy, Download, AlertCircle,
  Coins, Percent, Calendar, Building2, Sparkles, ExternalLink,
  Layers, BookOpen,
} from "lucide-react";

export default function NewTTContractPage() {
  return (
    <Suspense fallback={<div className="p-8"><SkeletonKPI /></div>}>
      <TTContractContent />
    </Suspense>
  );
}

function TTContractContent() {
  const searchParams = useSearchParams();
  const fromUFOPRaw = searchParams.get("from_ufop");
  const fromUFOP = fromUFOPRaw ? Number(fromUFOPRaw) : null;

  const toast = useToast();
  const { data: tpl, isLoading, error } = useTTTemplateFromUFOP(fromUFOP);

  if (!fromUFOP) {
    return (
      <div className="p-8 max-w-4xl">
        <Card>
          <EmptyState
            icon={AlertCircle}
            title="Origem não especificada"
            description="Acesse esta tela a partir de uma oportunidade no UFOP Intelligence."
            action={{
              label: "Ir para UFOP Intelligence",
              onClick: () => { window.location.href = "/ufop"; },
              icon: ArrowLeft,
            }}
          />
        </Card>
      </div>
    );
  }

  if (isLoading) {
    return <div className="p-8 max-w-4xl space-y-4"><SkeletonKPI /><SkeletonList count={2} /></div>;
  }

  if (error || !tpl) {
    return (
      <div className="p-8 max-w-4xl">
        <Card>
          <EmptyState
            icon={AlertCircle}
            title="Oportunidade não encontrada"
            description={`ID ${fromUFOP} não existe no banco.`}
          />
        </Card>
      </div>
    );
  }

  function copyMarkdown() {
    if (!tpl) return;
    navigator.clipboard.writeText(tpl.contract_markdown);
    toast.success("Contrato copiado", "Markdown completo na área de transferência.");
  }

  function downloadMarkdown() {
    if (!tpl) return;
    const blob = new Blob([tpl.contract_markdown], { type: "text/markdown;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${tpl.suggested_contract_number}.md`;
    document.body.appendChild(a); a.click(); document.body.removeChild(a);
    URL.revokeObjectURL(url);
    toast.success("Download iniciado", `${tpl.suggested_contract_number}.md`);
  }

  return (
    <div className="p-8 max-w-5xl space-y-6 fade-in">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-sm">
        <Link href="/ufop" className="hover:text-white transition-colors"
          style={{ color: "var(--text-muted)" }}>
          <ArrowLeft size={14} className="inline mr-1" />
          UFOP Intelligence
        </Link>
        <span style={{ color: "var(--text-muted)" }}>/</span>
        <span className="text-white">Gerar Contrato TT</span>
      </div>

      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white flex items-center gap-2">
          <FileSignature size={22} className="text-emerald-400" />
          Contrato TT adaptado
        </h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Rascunho gerado a partir da oportunidade UFOP, com termos sugeridos pela
          Lei 10.973/2004 + benchmark FORTEC 2023.
        </p>
      </div>

      {/* Tecnologia base */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BookOpen size={14} className="text-indigo-400" />
            Tecnologia base
          </CardTitle>
          <Badge variant="info">{tpl.suggested_contract_number}</Badge>
        </CardHeader>

        <p className="text-base font-semibold text-white leading-snug">{tpl.title}</p>
        <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>
          {tpl.department}
          {tpl.authors.length > 0 && ` · ${tpl.authors.join("; ")}`}
        </p>

        <p className="text-sm mt-3 leading-relaxed" style={{ color: "var(--text-muted)" }}>
          {tpl.abstract}
        </p>

        <div className="flex gap-2 mt-3 flex-wrap">
          {tpl.ipc_letter && <Badge variant="muted">IPC: {tpl.ipc_suggestion}</Badge>}
          {tpl.source_url && (
            <a href={tpl.source_url} target="_blank" rel="noopener noreferrer">
              <Button variant="ghost" size="sm">
                <ExternalLink size={11} /> Fonte no repositório UFOP
              </Button>
            </a>
          )}
        </div>
      </Card>

      {/* Termos sugeridos */}
      <Card style={{ borderColor: "#34d39930" }}>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Sparkles size={14} className="text-emerald-400" />
            Termos comerciais sugeridos
          </CardTitle>
          <Badge variant="muted">{tpl.methodology}</Badge>
        </CardHeader>

        <div className="grid grid-cols-4 gap-3">
          <Term icon={Percent}    label="Royalty"          value={`${tpl.suggested_royalty_pct.toFixed(1)}%`}      highlight />
          <Term icon={Coins}      label="Upfront"          value={formatBRL(tpl.suggested_upfront_brl)}            />
          <Term icon={Coins}      label="Floor anual"      value={formatBRL(tpl.suggested_floor_brl)}              />
          <Term icon={Building2}  label="Tipo de licença"  value={tpl.suggested_license_kind}                       />
          <Term icon={Calendar}   label="Vigência"         value={`${tpl.suggested_duration_years} anos`}          />
          <Term icon={Building2}  label="Território"       value={tpl.suggested_territory}                          />
          <Term icon={Sparkles}   label="Inventores (Lei)" value={`${tpl.suggested_inventor_share_pct}%`}           />
        </div>

        <div className="mt-4 pt-3" style={{ borderTop: "1px solid var(--border)" }}>
          <p className="text-xs font-semibold text-white mb-2">Justificativa acadêmica</p>
          <ul className="space-y-1">
            {tpl.rationale.map((r, i) => (
              <li key={i} className="text-xs leading-relaxed" style={{ color: "var(--text-muted)" }}>
                · {r}
              </li>
            ))}
          </ul>
        </div>
      </Card>

      {/* Patentes UFOP relacionadas */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Layers size={14} className="text-purple-400" />
            Patentes UFOP relacionadas (mesma área IPC)
            <Badge variant="muted">{tpl.related_patents.length}</Badge>
          </CardTitle>
        </CardHeader>
        {tpl.related_patents.length === 0 ? (
          <p className="text-xs" style={{ color: "var(--text-muted)" }}>
            Nenhuma patente UFOP registrada nesta área ainda.
            Rode <code className="text-indigo-400">make ingest-inpi-ufop</code> para popular.
          </p>
        ) : (
          <div className="space-y-2">
            {tpl.related_patents.map(p => (
              <Link key={p.id} href={`/patents/${p.id}`}>
                <div className="flex items-center gap-3 p-2 rounded hover:bg-white/5 transition-colors cursor-pointer"
                  style={{ background: "var(--surface-2)" }}>
                  <span className="font-mono text-xs text-indigo-400 shrink-0">{p.application_number}</span>
                  <span className="text-sm text-white flex-1 truncate">{p.title}</span>
                  <Badge variant={p.status === "classified" ? "success" : "warning"}>{p.status}</Badge>
                </div>
              </Link>
            ))}
          </div>
        )}
      </Card>

      {/* Contrato Markdown */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileSignature size={14} className="text-amber-400" />
            Rascunho do contrato (Markdown)
          </CardTitle>
          <div className="flex gap-2">
            <Button variant="ghost" size="sm" onClick={copyMarkdown}>
              <Copy size={11} /> Copiar
            </Button>
            <Button size="sm" onClick={downloadMarkdown}>
              <Download size={11} /> Baixar .md
            </Button>
          </div>
        </CardHeader>

        <pre className="p-4 rounded text-xs overflow-x-auto whitespace-pre-wrap leading-relaxed max-h-[600px] overflow-y-auto"
          style={{ background: "var(--surface-2)", color: "var(--text-muted)" }}>
          {tpl.contract_markdown}
        </pre>
      </Card>

      <div className="text-xs text-center p-3 rounded"
        style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text-muted)" }}>
        <AlertCircle size={11} className="inline mr-1 text-amber-400" />
        <strong className="text-white">Rascunho automatizado</strong> — revisão obrigatória
        pelo NIT-UFOP, Procuradoria UFOP e Resolução CUNI vigente antes da assinatura.
      </div>
    </div>
  );
}

function Term({ icon: Icon, label, value, highlight }: {
  icon: typeof Coins; label: string; value: string; highlight?: boolean;
}) {
  return (
    <div className="p-2.5 rounded-lg" style={{ background: "var(--surface-2)" }}>
      <div className="flex items-center gap-1 text-xs mb-1" style={{ color: "var(--text-muted)" }}>
        <Icon size={11} className={highlight ? "text-amber-400" : "text-slate-500"} />
        {label}
      </div>
      <p className={`text-sm ${highlight ? "text-amber-300 font-semibold" : "text-white"}`}>{value}</p>
    </div>
  );
}
