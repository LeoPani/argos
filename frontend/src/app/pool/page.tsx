"use client";

import { useState } from "react";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { mockPoolPatents, mockContracts } from "@/lib/mock-data";
import { formatDate, formatBRL } from "@/lib/utils";
import type { TTContract } from "@/lib/types";
import { Landmark, Plus, Download, Link2, CheckCircle, Circle, Clock } from "lucide-react";
import { cn } from "@/lib/utils";

type Tab = "pool" | "contracts";

function licenseLabel(t: string): string {
  return { exclusive: "Exclusiva", "non-exclusive": "Não-exclusiva", "sub-licensable": "Sub-licenciável" }[t] ?? t;
}

function royaltyStatus(s: string) {
  const map = {
    received: { label: "Recebido", variant: "success" as const, icon: <CheckCircle size={11} /> },
    pending: { label: "Aguardando", variant: "warning" as const, icon: <Clock size={11} /> },
    upcoming: { label: "A vencer", variant: "muted" as const, icon: <Circle size={11} /> },
  };
  return map[s as keyof typeof map] ?? map.upcoming;
}

export default function PoolPage() {
  const [tab, setTab] = useState<Tab>("pool");
  const [selectedContract, setSelectedContract] = useState<TTContract | null>(null);

  return (
    <div className="p-8 space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white flex items-center gap-2">
            <Landmark size={22} />
            Pool de Patentes & Transferência de Tecnologia
          </h1>
          <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
            Patentes UFOP disponíveis para licenciamento e contratos de TT ativos
          </p>
        </div>
        <Button size="sm">
          <Plus size={14} />
          Nova oferta
        </Button>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 p-1 rounded-lg w-fit" style={{ background: "var(--surface)" }}>
        {(["pool", "contracts"] as Tab[]).map(t => (
          <button key={t} onClick={() => setTab(t)}
            className="px-4 py-2 rounded-md text-sm transition-all"
            style={{ background: tab === t ? "var(--accent)" : "transparent", color: tab === t ? "white" : "var(--text-muted)" }}>
            {t === "pool" ? "Pool de Patentes" : "Contratos de TT"}
          </button>
        ))}
      </div>

      {tab === "pool" ? (
        <>
          {/* Summary */}
          <div className="grid grid-cols-3 gap-4">
            {[
              { label: "Patentes disponíveis", value: "23" },
              { label: "Em negociação", value: "7" },
              { label: "Contratos ativos", value: "4" },
            ].map(({ label, value }) => (
              <Card key={label}>
                <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>{label}</p>
                <p className="text-2xl font-bold text-white">{value}</p>
              </Card>
            ))}
          </div>

          {/* Patent pool list */}
          <div className="space-y-4">
            {mockPoolPatents.map(patent => (
              <Card key={patent.id}>
                <div className="flex items-start gap-4">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1.5 flex-wrap">
                      <span className="text-xs font-mono text-indigo-400">{patent.number}</span>
                      <Badge variant="info">IPC: {patent.ipc_code}</Badge>
                      <Badge variant={patent.license_type === "exclusive" ? "warning" : "default"}>
                        {licenseLabel(patent.license_type)}
                      </Badge>
                    </div>
                    <p className="text-base font-semibold text-white mb-1">{patent.title}</p>
                    <p className="text-xs" style={{ color: "var(--text-muted)" }}>
                      Dep.: {patent.department} · Royalty sugerido: {patent.royalty_suggestion}
                    </p>
                    <div className="mt-2 px-3 py-2 rounded-lg text-xs"
                      style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
                      🤖 <span style={{ color: "var(--text-muted)" }}>{patent.ai_match}</span>
                    </div>
                  </div>
                  <div className="flex flex-col gap-2 shrink-0">
                    <Button size="sm">Manifestar interesse</Button>
                    {patent.prospectus_url && (
                      <Button variant="secondary" size="sm">
                        <Download size={12} />
                        Prospecto
                      </Button>
                    )}
                  </div>
                </div>
              </Card>
            ))}
          </div>
        </>
      ) : (
        <div className="grid grid-cols-2 gap-6">
          {/* Contract list */}
          <div className="space-y-3">
            <h2 className="text-sm font-semibold text-white">Contratos ativos</h2>
            {mockContracts.map(c => (
              <button key={c.id} onClick={() => setSelectedContract(c)} className="w-full text-left">
                <Card style={{ borderColor: selectedContract?.id === c.id ? "var(--accent)" : "var(--border)", cursor: "pointer" }}>
                  <div className="flex items-start justify-between gap-2 mb-2">
                    <div>
                      <p className="text-xs font-mono text-indigo-400">{c.number}</p>
                      <p className="text-sm font-semibold text-white mt-0.5 leading-snug">{c.patent_title}</p>
                    </div>
                    <Badge variant="success">Ativo</Badge>
                  </div>
                  <p className="text-xs" style={{ color: "var(--text-muted)" }}>
                    {c.licensor} → {c.licensee}
                  </p>
                  <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>
                    Royalty: {c.royalty_rate}% · Vence: {formatDate(c.expiry_at)}
                  </p>
                  {c.blockchain_hash && (
                    <div className="flex items-center gap-1 mt-1.5 text-xs text-indigo-400">
                      <Link2 size={11} />
                      Hash: {c.blockchain_hash}
                    </div>
                  )}
                </Card>
              </button>
            ))}
          </div>

          {/* Contract detail */}
          {selectedContract ? (
            <div className="space-y-4">
              {/* Royalties */}
              <Card>
                <CardHeader>
                  <CardTitle>Royalties — {selectedContract.number}</CardTitle>
                  <Button variant="secondary" size="sm">Gerar relatório</Button>
                </CardHeader>
                <table className="w-full text-sm">
                  <thead>
                    <tr style={{ borderBottom: "1px solid var(--border)" }}>
                      {["Período", "Previsto", "Realizado", "Status"].map(h => (
                        <th key={h} className="text-left pb-2 pr-3 text-xs font-medium" style={{ color: "var(--text-muted)" }}>{h}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {selectedContract.royalties.map(r => {
                      const { label, variant, icon } = royaltyStatus(r.status);
                      return (
                        <tr key={r.period} style={{ borderBottom: "1px solid var(--border)" }}>
                          <td className="py-2.5 pr-3 text-xs text-white">{r.period}</td>
                          <td className="py-2.5 pr-3 text-xs text-white">{formatBRL(r.expected)}</td>
                          <td className="py-2.5 pr-3 text-xs" style={{ color: "var(--text-muted)" }}>
                            {r.received !== null ? formatBRL(r.received) : "—"}
                          </td>
                          <td className="py-2.5">
                            <Badge variant={variant}>{icon} {label}</Badge>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </Card>

              {/* Milestones */}
              <Card>
                <CardHeader>
                  <CardTitle>Marcos contratuais</CardTitle>
                </CardHeader>
                <div className="space-y-3">
                  {selectedContract.milestones.map((m, i) => (
                    <div key={i} className="flex items-center gap-3">
                      {m.done
                        ? <CheckCircle size={16} className="text-emerald-400 shrink-0" />
                        : <Circle size={16} className="shrink-0" style={{ color: "var(--border)" }} />}
                      <div className="flex-1">
                        <p className={cn("text-sm", m.done ? "text-white" : "")}
                          style={{ color: m.done ? "white" : "var(--text-muted)" }}>
                          {m.label}
                        </p>
                        <p className="text-xs" style={{ color: "var(--text-muted)" }}>
                          Prazo: {formatDate(m.due_date)}
                        </p>
                      </div>
                      {!m.done && (
                        <Badge variant="warning">Pendente</Badge>
                      )}
                    </div>
                  ))}
                </div>
              </Card>
            </div>
          ) : (
            <div className="flex items-center justify-center rounded-xl"
              style={{ border: "1px dashed var(--border)", minHeight: 200 }}>
              <p className="text-sm" style={{ color: "var(--text-muted)" }}>
                Selecione um contrato para ver os detalhes
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
