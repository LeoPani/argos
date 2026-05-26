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
import type {
  Dispute, DisputeStatus, DisputeKind, SubjectKind,
  DisputeSubject, Patent, PIComparisonResult,
} from "@/lib/types";
import {
  Scale, Plus, FileText, Clock, AlertTriangle,
  RefreshCw, CheckCircle2, X, ArrowRight,
  Sparkles, Trophy, Trash2, Tag as TagIcon,
  GitCompare, AlertCircle, CheckCircle, HelpCircle,
  Zap, Loader2,
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

// ─── tabs ─────────────────────────────────────────────────────────────────────

type Tab = "compare" | "disputes";

// ─── main page ────────────────────────────────────────────────────────────────

export default function ArbitragemPage() {
  const [tab, setTab] = useState<Tab>("compare");

  return (
    <div className="p-8 space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white flex items-center gap-2">
          <Scale size={22} />
          Arbitragem de PI
        </h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Comparação direta de patentes via IA e gestão de disputas formais
        </p>
      </div>

      {/* Tab bar */}
      <div className="flex gap-1 p-1 rounded-xl w-fit"
        style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
        {([
          { id: "compare" as Tab, icon: GitCompare, label: "Comparação Direta" },
          { id: "disputes" as Tab, icon: Scale, label: "Disputas Formais" },
        ]).map(t => (
          <button key={t.id} onClick={() => setTab(t.id)}
            className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all"
            style={{
              background: tab === t.id ? "var(--accent)" : "transparent",
              color: tab === t.id ? "white" : "var(--text-muted)",
            }}>
            <t.icon size={14} />
            {t.label}
          </button>
        ))}
      </div>

      {tab === "compare" ? <CompareTab /> : <DisputesTab />}
    </div>
  );
}

// ─── Comparação Direta ────────────────────────────────────────────────────────

function CompareTab() {
  const { data: patentsData } = usePatents({ limit: "200" });
  const patents: Patent[] = patentsData?.items ?? [];

  const [idA, setIdA] = useState<number | "">("");
  const [idB, setIdB] = useState<number | "">("");
  const [loading, setLoading] = useState(false);
  const [error, setError]     = useState<string | null>(null);
  const [result, setResult]   = useState<PIComparisonResult | null>(null);

  async function run() {
    if (!idA || !idB) return;
    setLoading(true); setError(null); setResult(null);
    try {
      const r = await api.disputes.compare(Number(idA), Number(idB));
      setResult(r);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Erro na comparação");
    } finally { setLoading(false); }
  }

  const pA = patents.find(p => p.id === Number(idA));
  const pB = patents.find(p => p.id === Number(idB));

  return (
    <div className="space-y-6">
      {/* Explainer */}
      <div className="rounded-xl p-4 grid grid-cols-3 gap-4 text-center text-sm"
        style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
        {[
          { icon: <GitCompare size={16} className="text-indigo-400" />, t: "1. Selecione", d: "Escolha dois documentos de PI do banco de dados" },
          { icon: <Zap size={16} className="text-amber-400" />, t: "2. Groq LLM", d: "IA analisa títulos, abstracts, IPC e datas de depósito" },
          { icon: <Scale size={16} className="text-emerald-400" />, t: "3. Veredito", d: "Recebe: score de similaridade, áreas de conflito e recomendação" },
        ].map(s => (
          <div key={s.t} className="space-y-1.5">
            <div className="flex justify-center">{s.icon}</div>
            <p className="font-semibold text-white">{s.t}</p>
            <p className="text-xs" style={{ color: "var(--text-muted)" }}>{s.d}</p>
          </div>
        ))}
      </div>

      {/* Selectors */}
      <Card>
        <h2 className="text-sm font-semibold text-white mb-4 flex items-center gap-2">
          <GitCompare size={14} className="text-indigo-400" />
          Selecionar patentes para comparar
        </h2>
        <div className="grid grid-cols-2 gap-4 mb-4">
          <div>
            <label className="text-xs mb-1 block" style={{ color: "var(--text-muted)" }}>
              Patente A
            </label>
            <select
              className="w-full px-3 py-2 rounded-lg text-sm text-white outline-none focus:ring-1 focus:ring-indigo-500"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}
              value={idA}
              onChange={e => setIdA(e.target.value ? Number(e.target.value) : "")}>
              <option value="">-- Selecione --</option>
              {patents.map(p => (
                <option key={p.id} value={p.id}>
                  #{p.id} · {p.application_number} — {p.title.slice(0, 60)}{p.title.length > 60 ? "…" : ""}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-xs mb-1 block" style={{ color: "var(--text-muted)" }}>
              Patente B
            </label>
            <select
              className="w-full px-3 py-2 rounded-lg text-sm text-white outline-none focus:ring-1 focus:ring-indigo-500"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}
              value={idB}
              onChange={e => setIdB(e.target.value ? Number(e.target.value) : "")}>
              <option value="">-- Selecione --</option>
              {patents.filter(p => p.id !== Number(idA)).map(p => (
                <option key={p.id} value={p.id}>
                  #{p.id} · {p.application_number} — {p.title.slice(0, 60)}{p.title.length > 60 ? "…" : ""}
                </option>
              ))}
            </select>
          </div>
        </div>

        {error && (
          <div className="mb-4 flex items-center gap-2 p-3 rounded-lg text-sm text-red-400"
            style={{ background: "#2a1010", border: "1px solid #ef444440" }}>
            <AlertTriangle size={14} /> {error}
          </div>
        )}

        <Button
          onClick={run}
          disabled={!idA || !idB || loading || idA === idB}
          className="w-full">
          {loading
            ? <><Loader2 size={14} className="animate-spin" /> Analisando via Groq LLM…</>
            : <><Sparkles size={14} /> Comparar com IA</>}
        </Button>

        {idA === idB && idA !== "" && (
          <p className="text-xs mt-2 text-amber-400 text-center">Selecione patentes diferentes</p>
        )}
      </Card>

      {/* Side-by-side mini preview */}
      {pA && pB && !result && !loading && (
        <div className="grid grid-cols-2 gap-4">
          <PatentPreview patent={pA} label="A" />
          <PatentPreview patent={pB} label="B" />
        </div>
      )}

      {/* Result */}
      {result && <ComparisonResult result={result} />}
    </div>
  );
}

// ─── Patent preview card ──────────────────────────────────────────────────────

function PatentPreview({ patent, label }: { patent: Patent; label: string }) {
  return (
    <Card style={{ borderColor: label === "A" ? "#6366f140" : "#f59e0b40" }}>
      <div className="flex items-center gap-2 mb-2">
        <span className="text-xs font-mono font-bold px-2 py-0.5 rounded"
          style={{ background: label === "A" ? "#6366f140" : "#f59e0b40", color: label === "A" ? "#818cf8" : "#fbbf24" }}>
          PI {label}
        </span>
        <span className="text-xs font-mono" style={{ color: "var(--text-muted)" }}>
          {patent.application_number}
        </span>
      </div>
      <p className="text-sm font-semibold text-white leading-snug mb-2">{patent.title}</p>
      <p className="text-xs leading-relaxed line-clamp-3" style={{ color: "var(--text-muted)" }}>
        {patent.abstract || "Sem abstract disponível"}
      </p>
      <div className="flex items-center gap-3 mt-3 text-xs" style={{ color: "var(--text-muted)" }}>
        {patent.filing_date && <span>📅 {formatDate(patent.filing_date)}</span>}
        {patent.ipc_category !== null && <span>IPC: {["A","B","C","D","E","F","G","H"][patent.ipc_category] ?? patent.ipc_category}</span>}
      </div>
    </Card>
  );
}

// ─── Comparison result ────────────────────────────────────────────────────────

const recInfo: Record<string, { label: string; color: string; icon: typeof AlertCircle }> = {
  possivel_infracao: { label: "Possível infração",  color: "#ef4444", icon: AlertCircle   },
  sem_conflito:      { label: "Sem conflito",        color: "#34d399", icon: CheckCircle   },
  inconclusivo:      { label: "Inconclusivo",        color: "#f59e0b", icon: HelpCircle    },
};

function ComparisonResult({ result }: { result: PIComparisonResult }) {
  const rec = recInfo[result.recommendation] ?? recInfo["inconclusivo"];
  const RecIcon = rec.icon;
  const pctScore = Math.round(result.similarity_score * 100);

  return (
    <div className="space-y-4">
      {/* Verdict banner */}
      <div className="rounded-xl p-5"
        style={{ background: rec.color + "15", border: `2px solid ${rec.color}40` }}>
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-3">
            <RecIcon size={24} style={{ color: rec.color }} />
            <div>
              <p className="text-xs" style={{ color: "var(--text-muted)" }}>Veredito da IA</p>
              <p className="text-xl font-bold" style={{ color: rec.color }}>{rec.label}</p>
            </div>
          </div>
          <div className="text-right">
            <p className="text-xs" style={{ color: "var(--text-muted)" }}>Similaridade técnica</p>
            <p className="text-3xl font-bold text-white">{pctScore}%</p>
          </div>
        </div>

        {/* Score bar */}
        <div className="h-2 rounded-full overflow-hidden" style={{ background: "var(--surface-2)" }}>
          <div className="h-full rounded-full transition-all"
            style={{
              width: `${pctScore}%`,
              background: pctScore >= 65 ? "#ef4444" : pctScore >= 35 ? "#f59e0b" : "#34d399",
            }} />
        </div>

        <p className="text-sm mt-3 leading-relaxed" style={{ color: "var(--text-muted)" }}>
          {result.narrative}
        </p>

        <div className="flex items-center gap-3 mt-3 text-xs" style={{ color: "var(--text-muted)" }}>
          <span>Método: <span className="text-white">{result.method === "llm_groq" ? "🤖 Groq LLM" : "📐 Heurística local"}</span></span>
          <span>·</span>
          <span>Prioridade: <span style={{ color: "#fbbf24" }}>PI {result.priority_winner === "equal" ? "A=B" : result.priority_winner} depositou primeiro</span></span>
        </div>
      </div>

      {/* Side-by-side analysis */}
      <div className="grid grid-cols-2 gap-4">
        <PatentPreview patent={result.patent_a} label="A" />
        <PatentPreview patent={result.patent_b} label="B" />
      </div>

      {/* Conflict + Diff */}
      <div className="grid grid-cols-2 gap-4">
        {result.conflict_areas.length > 0 && (
          <Card style={{ borderColor: "#ef444430" }}>
            <h3 className="text-xs font-semibold text-red-400 mb-2 flex items-center gap-1">
              <AlertCircle size={11} /> Áreas de conflito
            </h3>
            <ul className="space-y-1">
              {result.conflict_areas.map((c, i) => (
                <li key={i} className="text-xs text-red-300">⚠ {c}</li>
              ))}
            </ul>
          </Card>
        )}
        {result.differentiating_claims.length > 0 && (
          <Card style={{ borderColor: "#34d39930" }}>
            <h3 className="text-xs font-semibold text-emerald-400 mb-2 flex items-center gap-1">
              <CheckCircle size={11} /> Fatores diferenciadores
            </h3>
            <ul className="space-y-1">
              {result.differentiating_claims.map((d, i) => (
                <li key={i} className="text-xs text-emerald-300">✓ {d}</li>
              ))}
            </ul>
          </Card>
        )}
      </div>

      {/* Per-patent strengths */}
      <div className="grid grid-cols-2 gap-4">
        {[
          { label: "PI A", strengths: result.patent_a_strengths, color: "#6366f1" },
          { label: "PI B", strengths: result.patent_b_strengths, color: "#f59e0b" },
        ].map(({ label, strengths, color }) => (
          <Card key={label}>
            <h3 className="text-xs font-semibold mb-2" style={{ color }}>
              Pontos fortes — {label}
            </h3>
            {strengths.length > 0 ? (
              <ul className="space-y-1">
                {strengths.map((s, i) => (
                  <li key={i} className="text-xs" style={{ color: "var(--text-muted)" }}>+ {s}</li>
                ))}
              </ul>
            ) : (
              <p className="text-xs" style={{ color: "var(--text-muted)" }}>Sem pontos fortes destacados</p>
            )}
          </Card>
        ))}
      </div>
    </div>
  );
}

// ─── DisputesTab (dispute management, existing logic) ─────────────────────────

function DisputesTab() {
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
    <div className="space-y-6">
      {/* Controls */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          {isLive ? (
            <span className="text-xs text-emerald-400 flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse inline-block" />
              {data!.pagination.total} disputas no banco
            </span>
          ) : (
            <span className="text-xs text-amber-400">backend offline</span>
          )}
        </div>
        <div className="flex items-center gap-2">
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
              const { label } = statusInfo(s);
              return (
                <Button key={s} size="sm" variant="secondary"
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

// ─── Subjects panel ────────────────────────────────────────────────────────────

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

// ─── Verdict panel ─────────────────────────────────────────────────────────────

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
        </p>
      ) : (
        <div className="space-y-3">
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

          <div>
            <p className="text-xs font-semibold text-white mb-1">Fatores considerados</p>
            <ul className="space-y-0.5">
              {verdict.reasoning.factors.map((f, i) => (
                <li key={i} className="text-xs" style={{ color: "var(--text-muted)" }}>· {f}</li>
              ))}
            </ul>
          </div>

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
                        {s.pro.map((p, i) => <p key={i} className="text-xs text-emerald-400">+ {p}</p>)}
                      </div>
                    )}
                    {s.con.length > 0 && (
                      <div className="space-y-0.5">
                        {s.con.map((c, i) => <p key={i} className="text-xs text-red-400">− {c}</p>)}
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
