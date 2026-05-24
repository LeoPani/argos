"use client";

import { useState } from "react";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SkeletonKPI, SkeletonList } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { useDisputes, useDisputeSubjects, useDisputeVerdict, usePatents } from "@/lib/hooks";
import { api } from "@/lib/api";
import { formatDate } from "@/lib/utils";
import type { Dispute, DisputeStatus, DisputeKind, SubjectKind, DisputeSubject } from "@/lib/types";
import {
  Scale, Plus, FileText, Clock, AlertTriangle,
  RefreshCw, CheckCircle2, X, ArrowRight,
  Sparkles, Trophy, Trash2, Tag as TagIcon,
} from "lucide-react";

// ─── status / kind labels ─────────────────────────────────────────────────────

function statusInfo(s: DisputeStatus): {
  label: string;
  variant: "warning" | "danger" | "info" | "muted" | "success";
} {
  const map: Record<DisputeStatus, { label: string; variant: "warning" | "danger" | "info" | "muted" | "success" }> = {
    open:           { label: "Aberta",            variant: "info"    },
    in_review:      { label: "Em análise",         variant: "warning" },
    awaiting_info:  { label: "Aguardando info",    variant: "warning" },
    resolved:       { label: "Resolvida",          variant: "success" },
    withdrawn:      { label: "Retirada",           variant: "muted"   },
    escalated:      { label: "⚠ Escalada",         variant: "danger"  },
  };
  return map[s] ?? { label: s, variant: "muted" };
}

const kindLabel: Record<DisputeKind, string> = {
  trademark_infringement: "Infração de marca",
  patent_infringement:    "Infração de patente",
  authorship:             "Autoria",
  licensing:              "Licenciamento",
  other:                  "Outro",
};

const activeStatuses: DisputeStatus[] = ["open", "in_review", "awaiting_info", "escalated"];

// ─── main page ────────────────────────────────────────────────────────────────

export default function ArbitragemPage() {
  const { data, error, isLoading, mutate } = useDisputes({ limit: "50" });
  const [selectedID, setSelectedID] = useState<number | null>(null);
  const [showForm, setShowForm]     = useState(false);

  const isLive   = !error && !!data;
  const loading  = isLoading && !data && !error;
  const items: Dispute[] = data?.items ?? [];

  const selected = items.find(d => d.id === selectedID) ?? null;
  const active   = items.filter(d => activeStatuses.includes(d.status));
  const escalated = items.filter(d => d.status === "escalated").length;

  return (
    <div className="p-8 space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white flex items-center gap-2">
            <Scale size={22} />
            Arbitragem de PI
          </h1>
          <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
            Gestão de disputas, provas e mediação interna
          </p>
        </div>
        <div className="flex items-center gap-2">
          {isLive ? (
            <span className="text-xs text-emerald-400 flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse inline-block" />
              {data!.pagination.total} disputas no banco
            </span>
          ) : (
            <span className="text-xs text-amber-400">backend offline</span>
          )}
          <Button variant="ghost" size="sm" onClick={() => mutate()}>
            <RefreshCw size={13} /> Atualizar
          </Button>
          <Button size="sm" onClick={() => setShowForm(s => !s)}>
            {showForm ? <X size={14} /> : <Plus size={14} />}
            {showForm ? "Cancelar" : "Nova disputa"}
          </Button>
        </div>
      </div>

      {/* Open dispute form */}
      {showForm && (
        <OpenDisputeForm onCreated={() => { setShowForm(false); mutate(); }} />
      )}

      {/* KPIs */}
      <div className="grid grid-cols-3 gap-4">
        {loading ? (
          <><SkeletonKPI /><SkeletonKPI /><SkeletonKPI /></>
        ) : (
          <>
            <KPICard icon={Scale}          label="Disputas ativas"   value={active.length.toString()}                  color="#6366f1" />
            <KPICard icon={AlertTriangle}  label="Escaladas"          value={escalated.toString()}                       color="#ef4444" />
            <KPICard icon={Clock}          label="Resolvidas (total)" value={items.filter(d => d.status === "resolved").length.toString()} color="#34d399" />
          </>
        )}
      </div>

      <div className="grid grid-cols-2 gap-6">
        {/* Dispute list */}
        <div className="space-y-3">
          <h2 className="text-sm font-semibold text-white">
            Disputas
            <span className="ml-2 text-xs font-normal" style={{ color: "var(--text-muted)" }}>
              ({items.length})
            </span>
          </h2>

          {loading && <SkeletonList count={4} />}

          {!loading && items.length === 0 && (
            <Card>
              <EmptyState
                icon={Scale}
                title="Nenhuma disputa registrada"
                description="Use o botão 'Nova disputa' para abrir um caso de arbitragem."
                size="sm"
              />
            </Card>
          )}

          {!loading && items.map(d => {
            const { label, variant } = statusInfo(d.status);
            const isSelected = selected?.id === d.id;
            return (
              <button key={d.id} onClick={() => setSelectedID(d.id)} className="w-full text-left">
                <Card
                  style={{
                    borderColor: isSelected
                      ? "var(--accent)"
                      : d.status === "escalated"
                        ? "#ef444440"
                        : "var(--border)",
                    cursor: "pointer",
                    transition: "border-color 0.2s",
                  }}
                >
                  <div className="flex items-start justify-between gap-2 mb-2">
                    <div className="flex-1">
                      <p className="text-xs font-mono text-indigo-400">{d.case_number}</p>
                      <p className="text-sm font-semibold text-white mt-0.5 leading-snug">{d.title}</p>
                    </div>
                    <Badge variant={variant}>{label}</Badge>
                  </div>
                  <p className="text-xs leading-relaxed line-clamp-2" style={{ color: "var(--text-muted)" }}>
                    {d.summary}
                  </p>
                  <div className="flex items-center gap-3 mt-2">
                    <Badge variant="muted">{kindLabel[d.kind]}</Badge>
                    <span className="text-xs" style={{ color: "var(--text-muted)" }}>
                      Aberta: {formatDate(d.opened_at)}
                    </span>
                  </div>
                </Card>
              </button>
            );
          })}
        </div>

        {/* Detail panel */}
        {selected ? (
          <DisputeDetail dispute={selected} onChanged={() => mutate()} />
        ) : (
          <div className="flex items-center justify-center h-64 rounded-xl"
            style={{ border: "1px dashed var(--border)" }}>
            <p className="text-sm" style={{ color: "var(--text-muted)" }}>
              Selecione uma disputa para ver detalhes
            </p>
          </div>
        )}
      </div>
    </div>
  );
}

// ─── KPI card ─────────────────────────────────────────────────────────────────

function KPICard({ icon: Icon, label, value, color }: {
  icon: typeof Scale; label: string; value: string; color: string;
}) {
  return (
    <Card>
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
  );
}

// ─── Detail panel with status transition ──────────────────────────────────────

function DisputeDetail({ dispute, onChanged }: { dispute: Dispute; onChanged: () => void }) {
  const [busy, setBusy]       = useState(false);
  const [error, setError]     = useState<string | null>(null);

  // Next-state options based on current status
  const transitions: Record<DisputeStatus, DisputeStatus[]> = {
    open:          ["in_review", "withdrawn"],
    in_review:     ["awaiting_info", "resolved", "escalated"],
    awaiting_info: ["in_review", "resolved"],
    resolved:      [],
    withdrawn:     [],
    escalated:     ["in_review", "resolved"],
  };
  const nextOptions = transitions[dispute.status] ?? [];

  async function changeStatus(s: DisputeStatus) {
    setBusy(true); setError(null);
    try {
      await api.disputes.updateStatus(dispute.id, s);
      onChanged();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Erro desconhecido");
    } finally { setBusy(false); }
  }

  const { label, variant } = statusInfo(dispute.status);

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>{dispute.case_number}</CardTitle>
          <Badge variant={variant}>{label}</Badge>
        </CardHeader>

        <p className="text-sm font-semibold text-white mb-2">{dispute.title}</p>
        <p className="text-sm leading-relaxed mb-3" style={{ color: "var(--text-muted)" }}>
          {dispute.summary}
        </p>

        <div className="grid grid-cols-2 gap-3 text-xs">
          <Field label="Tipo"        value={kindLabel[dispute.kind]} />
          <Field label="ID"          value={`#${dispute.id}`} />
          <Field label="Aberta em"   value={formatDate(dispute.opened_at)} />
          <Field label="Atualizada"  value={formatDate(dispute.updated_at)} />
          {dispute.resolved_at && (
            <Field label="Resolvida em" value={formatDate(dispute.resolved_at)} />
          )}
        </div>
      </Card>

      {/* Status transitions */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ArrowRight size={14} className="text-indigo-400" />
            Próxima etapa
          </CardTitle>
        </CardHeader>

        {nextOptions.length === 0 ? (
          <p className="text-sm py-2" style={{ color: "var(--text-muted)" }}>
            <CheckCircle2 size={12} className="inline mr-1 text-emerald-400" />
            Disputa em estado final — nenhuma transição disponível.
          </p>
        ) : (
          <div className="flex gap-2 flex-wrap">
            {nextOptions.map(s => {
              const { label, variant } = statusInfo(s);
              return (
                <Button key={s} size="sm"
                  variant={variant === "danger" ? "secondary" : "secondary"}
                  disabled={busy}
                  onClick={() => changeStatus(s)}>
                  {busy ? "…" : `→ ${label}`}
                </Button>
              );
            })}
          </div>
        )}

        {error && (
          <p className="text-xs mt-2" style={{ color: "#f87171" }}>{error}</p>
        )}
      </Card>

      <SubjectsPanel disputeID={dispute.id} kind={dispute.kind} />
      <VerdictPanel disputeID={dispute.id} />
    </div>
  );
}

// ─── Subjects panel — N marcas/patentes em comparação ─────────────────────────

function SubjectsPanel({ disputeID, kind }: { disputeID: number; kind: DisputeKind }) {
  const { data: subjData, mutate } = useDisputeSubjects(disputeID);
  const subjects = subjData?.items ?? [];

  const subjectKind: SubjectKind =
    kind === "trademark_infringement" ? "trademark" :
    kind === "patent_infringement"    ? "patent"    :
    kind === "authorship"             ? "inventor"  : "other";

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <TagIcon size={14} className="text-indigo-400" />
          Partes em comparação ({subjects.length})
        </CardTitle>
      </CardHeader>

      {subjects.length === 0 && (
        <p className="text-xs py-2" style={{ color: "var(--text-muted)" }}>
          Adicione pelo menos 2 itens para a IA comparar.
        </p>
      )}

      <div className="space-y-2 mb-3">
        {subjects.map(s => <SubjectRow key={s.id} subject={s} onRemoved={() => mutate()} />)}
      </div>

      <AddSubjectForm disputeID={disputeID} defaultKind={subjectKind} onAdded={() => mutate()} />
    </Card>
  );
}

function SubjectRow({ subject, onRemoved }: { subject: DisputeSubject; onRemoved: () => void }) {
  const [busy, setBusy] = useState(false);
  async function remove() {
    setBusy(true);
    try { await api.disputes.removeSubject(subject.id); onRemoved(); }
    finally { setBusy(false); }
  }
  return (
    <div className="flex items-center justify-between p-2.5 rounded-lg"
      style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
      <div className="flex items-center gap-2">
        <Badge variant="muted">{subject.kind}</Badge>
        <span className="text-sm text-white">{subject.label}</span>
        {subject.ref_id && (
          <span className="font-mono text-xs text-indigo-400">#{subject.ref_id}</span>
        )}
      </div>
      <Button variant="ghost" size="sm" onClick={remove} disabled={busy} style={{ color: "#f87171" }}>
        <Trash2 size={11} />
      </Button>
    </div>
  );
}

function AddSubjectForm({ disputeID, defaultKind, onAdded }: {
  disputeID: number; defaultKind: SubjectKind; onAdded: () => void;
}) {
  const [kind, setKind]   = useState<SubjectKind>(defaultKind);
  const [refID, setRefID] = useState("");
  const [label, setLabel] = useState("");
  const [busy, setBusy]   = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!label.trim()) return;
    setBusy(true); setError(null);
    try {
      await api.disputes.addSubject(disputeID, {
        kind,
        ref_id: refID ? Number(refID) : undefined,
        label: label.trim(),
      });
      setRefID(""); setLabel("");
      onAdded();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Erro");
    } finally { setBusy(false); }
  }

  return (
    <form onSubmit={submit} className="flex gap-2 items-end"
      style={{ paddingTop: "12px", borderTop: "1px solid var(--border)" }}>
      <div className="w-32">
        <label className="text-xs" style={{ color: "var(--text-muted)" }}>Tipo</label>
        <select value={kind} onChange={e => setKind(e.target.value as SubjectKind)}
          className="w-full px-2 py-1.5 rounded text-xs outline-none"
          style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }}>
          <option value="trademark">Marca</option>
          <option value="patent">Patente</option>
          <option value="inventor">Inventor</option>
          <option value="other">Outro</option>
        </select>
      </div>
      <div className="w-24">
        <label className="text-xs" style={{ color: "var(--text-muted)" }}>ID (opc.)</label>
        <input value={refID} onChange={e => setRefID(e.target.value)} placeholder="#id"
          className="w-full px-2 py-1.5 rounded text-xs outline-none font-mono"
          style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }} />
      </div>
      <div className="flex-1">
        <label className="text-xs" style={{ color: "var(--text-muted)" }}>Nome/Label</label>
        <input value={label} onChange={e => setLabel(e.target.value)}
          placeholder="ex: ARGOS INTELLIGENCE"
          className="w-full px-2 py-1.5 rounded text-xs outline-none"
          style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }} />
      </div>
      <Button type="submit" size="sm" disabled={busy || !label.trim()}>
        <Plus size={11} /> Adicionar
      </Button>
      {error && <span className="text-xs" style={{ color: "#f87171" }}>{error}</span>}
    </form>
  );
}

// ─── Verdict panel — IA generates report ──────────────────────────────────────

function VerdictPanel({ disputeID }: { disputeID: number }) {
  const { data, mutate } = useDisputeVerdict(disputeID);
  const verdict = data?.verdict ?? null;
  const [busy, setBusy]   = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function analyze() {
    setBusy(true); setError(null);
    try { await api.disputes.analyze(disputeID); mutate(); }
    catch (e: unknown) { setError(e instanceof Error ? e.message : "Erro"); }
    finally { setBusy(false); }
  }

  return (
    <Card style={{ borderColor: verdict ? "#a855f730" : "var(--border)" }}>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Sparkles size={14} className="text-purple-400" />
          Análise da IA
          {verdict && <Badge variant="muted">{verdict.method}</Badge>}
        </CardTitle>
        <Button size="sm" onClick={analyze} disabled={busy}>
          {busy
            ? <><div className="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin" /> Analisando…</>
            : <><Sparkles size={11} /> {verdict ? "Re-analisar" : "Analisar agora"}</>}
        </Button>
      </CardHeader>

      {error && <p className="text-xs mb-2" style={{ color: "#f87171" }}>{error}</p>}

      {!verdict ? (
        <p className="text-xs py-2" style={{ color: "var(--text-muted)" }}>
          Clique em "Analisar" para gerar veredito baseado nas marcas/patentes adicionadas.
          O modelo compara prioridade temporal, distintividade, classes Nice e status no INPI.
        </p>
      ) : (
        <div className="space-y-3">
          {/* Summary */}
          <div className="rounded-lg p-3"
            style={{ background: "#a855f720", border: "1px solid #a855f740" }}>
            <div className="flex items-start gap-2">
              <Trophy size={14} className="text-amber-400 mt-0.5 shrink-0" />
              <div className="flex-1">
                <p className="text-sm text-white">{verdict.summary}</p>
                <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>
                  Confiança: <span className="text-white font-semibold">{verdict.confidence}%</span>
                  {" · "}
                  Gerado: {formatDate(verdict.created_at)}
                </p>
              </div>
            </div>
          </div>

          {/* Factors considered */}
          <div>
            <p className="text-xs font-semibold text-white mb-1">Fatores considerados</p>
            <ul className="space-y-0.5">
              {verdict.reasoning.factors.map((f, i) => (
                <li key={i} className="text-xs" style={{ color: "var(--text-muted)" }}>
                  · {f}
                </li>
              ))}
            </ul>
          </div>

          {/* Per-subject scores */}
          <div>
            <p className="text-xs font-semibold text-white mb-2">Análise por candidato</p>
            <div className="space-y-2">
              {verdict.reasoning.subjects.map(s => {
                const isWinner = verdict.winner_subject_id === s.subject_id;
                return (
                  <div key={s.subject_id} className="rounded-lg p-2.5"
                    style={{
                      background: "var(--surface-2)",
                      border: `1px solid ${isWinner ? "#fbbf24" : "var(--border)"}`,
                    }}>
                    <div className="flex items-center justify-between mb-1.5">
                      <div className="flex items-center gap-2">
                        {isWinner && <Trophy size={11} className="text-amber-400" />}
                        <span className="text-sm font-semibold text-white">{s.label}</span>
                      </div>
                      <span className="text-sm font-mono font-bold"
                        style={{ color: s.score >= 70 ? "#34d399" : s.score >= 50 ? "#fbbf24" : "#f87171" }}>
                        {s.score.toFixed(1)}/100
                      </span>
                    </div>
                    {s.pro.length > 0 && (
                      <div className="space-y-0.5 mb-1">
                        {s.pro.map((p, i) => (
                          <p key={i} className="text-xs text-emerald-400">+ {p}</p>
                        ))}
                      </div>
                    )}
                    {s.con.length > 0 && (
                      <div className="space-y-0.5">
                        {s.con.map((c, i) => (
                          <p key={i} className="text-xs text-red-400">− {c}</p>
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      )}
    </Card>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs mb-0.5" style={{ color: "var(--text-muted)" }}>{label}</p>
      <p className="text-sm text-white">{value}</p>
    </div>
  );
}

// ─── Open Dispute form ────────────────────────────────────────────────────────

function OpenDisputeForm({ onCreated }: { onCreated: () => void }) {
  const [caseNumber, setCaseNumber] = useState("");
  const [title, setTitle]           = useState("");
  const [summary, setSummary]       = useState("");
  const [kind, setKind]             = useState<DisputeKind>("trademark_infringement");
  const [busy, setBusy]             = useState(false);
  const [error, setError]           = useState<string | null>(null);

  async function handle(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true); setError(null);
    try {
      await api.disputes.open({
        case_number: caseNumber || autoCaseNumber(),
        title, summary, kind,
      });
      setCaseNumber(""); setTitle(""); setSummary(""); setKind("trademark_infringement");
      onCreated();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Erro ao criar disputa");
    } finally { setBusy(false); }
  }

  return (
    <Card style={{ borderColor: "var(--accent)" }}>
      <CardHeader>
        <CardTitle>Abrir nova disputa</CardTitle>
      </CardHeader>
      <form onSubmit={handle} className="space-y-3">
        <div className="grid grid-cols-2 gap-3">
          <FormField label="Número do caso (vazio = automático)">
            <input value={caseNumber} onChange={e => setCaseNumber(e.target.value)}
              placeholder={autoCaseNumber()}
              className="input" />
          </FormField>
          <FormField label="Tipo">
            <select value={kind} onChange={e => setKind(e.target.value as DisputeKind)}
              className="input">
              {Object.entries(kindLabel).map(([k, v]) => (
                <option key={k} value={k}>{v}</option>
              ))}
            </select>
          </FormField>
        </div>
        <FormField label="Título *">
          <input value={title} onChange={e => setTitle(e.target.value)} required
            placeholder="ex: Conflito de marca: ARGOS vs ARGUS"
            className="input" />
        </FormField>
        <FormField label="Resumo">
          <textarea value={summary} onChange={e => setSummary(e.target.value)}
            placeholder="Contexto, partes, alegações iniciais..."
            rows={3} className="input resize-y" />
        </FormField>

        {error && <p className="text-xs" style={{ color: "#f87171" }}>{error}</p>}

        <Button type="submit" size="sm" disabled={busy || !title.trim()}>
          {busy
            ? <><div className="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin" /> Abrindo…</>
            : <><Plus size={14} /> Abrir disputa</>}
        </Button>
      </form>

      {/* shared input style */}
      <style jsx>{`
        .input {
          width: 100%;
          padding: 0.5rem 0.75rem;
          background: var(--surface-2);
          border: 1px solid var(--border);
          border-radius: 0.5rem;
          color: white;
          font-size: 0.875rem;
          outline: none;
        }
        .input:focus { border-color: var(--accent); }
      `}</style>
    </Card>
  );
}

function FormField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="text-xs mb-1 block" style={{ color: "var(--text-muted)" }}>{label}</label>
      {children}
    </div>
  );
}

function autoCaseNumber(): string {
  const yr = new Date().getFullYear();
  const rnd = Math.floor(Math.random() * 900 + 100);
  return `ARB-${yr}-${rnd}`;
}
