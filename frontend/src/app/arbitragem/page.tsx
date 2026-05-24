"use client";

import { useState } from "react";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { mockDisputes } from "@/lib/mock-data";
import { formatDate } from "@/lib/utils";
import type { Dispute, DisputeStatus } from "@/lib/types";
import { Scale, Plus, Link2, FileText, Clock, AlertTriangle } from "lucide-react";

function statusInfo(s: DisputeStatus): { label: string; variant: "warning" | "danger" | "info" | "muted" | "success" } {
  const map: Record<DisputeStatus, { label: string; variant: "warning" | "danger" | "info" | "muted" | "success" }> = {
    open: { label: "Aberta", variant: "info" },
    in_analysis: { label: "Em análise", variant: "warning" },
    mediation: { label: "Mediação", variant: "info" },
    resolved: { label: "Resolvida", variant: "success" },
    urgent: { label: "⚠ URGENTE", variant: "danger" },
  };
  return map[s];
}

const milestones = [
  { label: "Abertura da disputa", date: "15/04/2026", done: true },
  { label: "Notificação das partes", date: "18/04/2026", done: true },
  { label: "Análise de documentos", date: "05/05/2026", done: true },
  { label: "Sessão de mediação", date: "30/05/2026", done: false },
  { label: "Decisão final", date: "30/06/2026", done: false },
];

export default function ArbitragemPage() {
  const [selected, setSelected] = useState<Dispute | null>(null);

  return (
    <div className="p-8 space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white flex items-center gap-2">
            <Scale size={22} />
            Arbitragem de PI
          </h1>
          <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
            Gestão de disputas, provas com carimbo blockchain e mediação
          </p>
        </div>
        <Button size="sm">
          <Plus size={14} />
          Nova Disputa
        </Button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-3 gap-4">
        {[
          { icon: Scale, label: "Disputas ativas", value: "3", color: "#6366f1" },
          { icon: AlertTriangle, label: "Urgentes", value: "1", color: "#ef4444" },
          { icon: Clock, label: "Prazo médio restante", value: "49 dias", color: "#f59e0b" },
        ].map(({ icon: Icon, label, value, color }) => (
          <Card key={label}>
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg" style={{ background: color + "20" }}>
                <Icon size={16} style={{ color }} />
              </div>
              <div>
                <p className="text-xs" style={{ color: "var(--text-muted)" }}>{label}</p>
                <p className="text-xl font-bold text-white">{value}</p>
              </div>
            </div>
          </Card>
        ))}
      </div>

      <div className="grid grid-cols-2 gap-6">
        {/* Dispute list */}
        <div className="space-y-3">
          <h2 className="text-sm font-semibold text-white">Disputas ativas</h2>
          {mockDisputes.map(d => {
            const { label, variant } = statusInfo(d.status);
            const isSelected = selected?.id === d.id;
            return (
              <button key={d.id} onClick={() => setSelected(d)} className="w-full text-left">
                <Card
                  style={{
                    borderColor: isSelected ? "var(--accent)" : d.status === "urgent" ? "#ef444440" : "var(--border)",
                    cursor: "pointer",
                    transition: "border-color 0.2s",
                  }}
                >
                  <div className="flex items-start justify-between gap-2 mb-2">
                    <div>
                      <p className="text-xs font-mono text-indigo-400">{d.number}</p>
                      <p className="text-sm font-semibold text-white mt-0.5 leading-snug">{d.title}</p>
                    </div>
                    <Badge variant={variant}>{label}</Badge>
                  </div>
                  <p className="text-xs" style={{ color: "var(--text-muted)" }}>
                    {d.plaintiff} vs {d.defendant}
                  </p>
                  <div className="flex items-center gap-3 mt-2">
                    <span className="text-xs" style={{ color: "var(--text-muted)" }}>
                      Aberta: {formatDate(d.opened_at)}
                    </span>
                    <Badge variant={d.deadline_days <= 20 ? "danger" : d.deadline_days <= 60 ? "warning" : "muted"}>
                      {d.deadline_days}d restantes
                    </Badge>
                    {d.blockchain_hash && (
                      <span className="flex items-center gap-1 text-xs text-indigo-400">
                        <Link2 size={11} /> Blockchain
                      </span>
                    )}
                  </div>
                </Card>
              </button>
            );
          })}
        </div>

        {/* Detail / Timeline */}
        {selected ? (
          <div className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>{selected.number} — Timeline</CardTitle>
                <Button variant="secondary" size="sm">
                  <FileText size={12} />
                  Gerar relatório
                </Button>
              </CardHeader>

              {/* Timeline visual */}
              <div className="relative pl-6 space-y-4">
                {milestones.map((m, i) => (
                  <div key={i} className="relative">
                    <div className={`absolute -left-6 w-3 h-3 rounded-full border-2 top-0.5 ${m.done ? "bg-indigo-500 border-indigo-500" : "border-slate-600 bg-transparent"}`} />
                    {i < milestones.length - 1 && (
                      <div className="absolute -left-5 top-3.5 w-px h-full" style={{ background: m.done ? "#6366f1" : "var(--border)" }} />
                    )}
                    <div className="ml-2">
                      <p className={`text-sm font-medium ${m.done ? "text-white" : ""}`} style={{ color: m.done ? "white" : "var(--text-muted)" }}>
                        {m.label}
                      </p>
                      <p className="text-xs" style={{ color: "var(--text-muted)" }}>{m.date}</p>
                    </div>
                  </div>
                ))}
              </div>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Provas & documentos</CardTitle>
                <Button variant="secondary" size="sm">
                  <Plus size={12} />
                  Adicionar prova
                </Button>
              </CardHeader>
              <div className="space-y-2">
                {[
                  { label: "Contrato_Original.pdf", hash: "0x4f3a..." },
                  { label: "Email_Notificacao.pdf", hash: "0x9b2c..." },
                ].map(doc => (
                  <div key={doc.label} className="flex items-center justify-between p-2.5 rounded-lg"
                    style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
                    <div className="flex items-center gap-2">
                      <FileText size={13} className="text-slate-400" />
                      <span className="text-sm text-white">{doc.label}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-mono text-indigo-400">{doc.hash}</span>
                      <Badge variant="success">✓ Blockchain</Badge>
                    </div>
                  </div>
                ))}
              </div>
            </Card>
          </div>
        ) : (
          <div className="flex items-center justify-center h-64 rounded-xl"
            style={{ border: "1px dashed var(--border)" }}>
            <p className="text-sm" style={{ color: "var(--text-muted)" }}>
              Selecione uma disputa para ver os detalhes
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
